//  Copyright (c) 2013, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.
//
// Copyright (c) 2011 The LevelDB Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file. See the AUTHORS file for names of contributors.

#include "db/flush_job.h"

#ifndef __STDC_FORMAT_MACROS
#define __STDC_FORMAT_MACROS
#endif

#include <inttypes.h>

#include <algorithm>
#include <vector>

#include "db/builder.h"
#include "db/db_iter.h"
#include "db/dbformat.h"
#include "db/event_helpers.h"
#include "db/filename.h"
#include "db/log_reader.h"
#include "db/log_writer.h"
#include "db/memtable.h"
#include "db/memtable_list.h"
#include "db/merge_context.h"
#include "db/version_set.h"
#include "port/likely.h"
#include "port/port.h"
#include "rocksdb/db.h"
#include "rocksdb/env.h"
#include "rocksdb/statistics.h"
#include "rocksdb/status.h"
#include "rocksdb/table.h"
#include "table/block.h"
#include "table/block_based_table_factory.h"
#include "table/merger.h"
#include "table/table_builder.h"
#include "table/two_level_iterator.h"
#include "util/coding.h"
#include "util/event_logger.h"
#include "util/file_util.h"
#include "util/iostats_context_imp.h"
#include "util/log_buffer.h"
#include "util/logging.h"
#include "util/mutexlock.h"
#include "util/perf_context_imp.h"
#include "util/stop_watch.h"
#include "util/sync_point.h"
#include "util/thread_status_util.h"

namespace rocksdb {

FlushJob::FlushJob(const std::string& dbname, ColumnFamilyData* cfd,
                   const DBOptions& db_options,
                   const MutableCFOptions& mutable_cf_options,
                   const EnvOptions& env_options, VersionSet* versions,
                   InstrumentedMutex* db_mutex,
                   std::atomic<bool>* shutting_down,
                   std::vector<SequenceNumber> existing_snapshots,
                   JobContext* job_context, LogBuffer* log_buffer,
                   Directory* db_directory, Directory* output_file_directory,
                   CompressionType output_compression, Statistics* stats,
                   EventLogger* event_logger)
    : dbname_(dbname),
      cfd_(cfd),
      db_options_(db_options),
      mutable_cf_options_(mutable_cf_options),
      env_options_(env_options),
      versions_(versions),
      db_mutex_(db_mutex),
      shutting_down_(shutting_down),
      existing_snapshots_(std::move(existing_snapshots)),
      job_context_(job_context),
      log_buffer_(log_buffer),
      db_directory_(db_directory),
      output_file_directory_(output_file_directory),
      output_compression_(output_compression),
      stats_(stats),
      event_logger_(event_logger) {
  // Update the thread status to indicate flush.
  ReportStartedFlush();
  TEST_SYNC_POINT("FlushJob::FlushJob()");
}

FlushJob::~FlushJob() {
  ThreadStatusUtil::ResetThreadStatus();
}

void FlushJob::ReportStartedFlush() {
  ThreadStatusUtil::SetColumnFamily(cfd_);
  ThreadStatusUtil::SetThreadOperation(ThreadStatus::OP_FLUSH);
  ThreadStatusUtil::SetThreadOperationProperty(
      ThreadStatus::COMPACTION_JOB_ID,
      job_context_->job_id);
  IOSTATS_RESET(bytes_written);
}

void FlushJob::ReportFlushInputSize(const autovector<MemTable*>& mems) {
  uint64_t input_size = 0;
  for (auto* mem : mems) {
    input_size += mem->ApproximateMemoryUsage();
  }
  ThreadStatusUtil::IncreaseThreadOperationProperty(
      ThreadStatus::FLUSH_BYTES_MEMTABLES,
      input_size);
}

void FlushJob::RecordFlushIOStats() {
  ThreadStatusUtil::SetThreadOperationProperty(
      ThreadStatus::FLUSH_BYTES_WRITTEN, IOSTATS(bytes_written));
}

Status FlushJob::Run(FileMetaData* file_meta) {
  AutoThreadOperationStageUpdater stage_run(
      ThreadStatus::STAGE_FLUSH_RUN);
  // Save the contents of the earliest memtable as a new Table
  FileMetaData meta;
  autovector<MemTable*> mems;
  cfd_->imm()->PickMemtablesToFlush(&mems);
  if (mems.empty()) {
    LogToBuffer(log_buffer_, "[%s] Nothing in memtable to flush",
                cfd_->GetName().c_str());
    return Status::OK();
  }

  ReportFlushInputSize(mems);

  // entries mems are (implicitly) sorted in ascending order by their created
  // time. We will use the first memtable's `edit` to keep the meta info for
  // this flush.
  MemTable* m = mems[0];
  VersionEdit* edit = m->GetEdits();
  edit->SetPrevLogNumber(0);
  // SetLogNumber(log_num) indicates logs with number smaller than log_num
  // will no longer be picked up for recovery.
  edit->SetLogNumber(mems.back()->GetNextLogNumber());
  edit->SetColumnFamily(cfd_->GetID());

  // This will release and re-acquire the mutex.
  Status s = WriteLevel0Table(mems, edit, &meta);

  if (s.ok() &&
      (shutting_down_->load(std::memory_order_acquire) || cfd_->IsDropped())) {
    s = Status::ShutdownInProgress(
        "Database shutdown or Column family drop during flush");
  }

  if (!s.ok()) {
    cfd_->imm()->RollbackMemtableFlush(mems, meta.fd.GetNumber());
  } else {
    TEST_SYNC_POINT("FlushJob::InstallResults");
    // Replace immutable memtable with the generated Table
    s = cfd_->imm()->InstallMemtableFlushResults(
        cfd_, mutable_cf_options_, mems, versions_, db_mutex_,
        meta.fd.GetNumber(), &job_context_->memtables_to_free, db_directory_,
        log_buffer_);
  }

  if (s.ok() && file_meta != nullptr) {
    *file_meta = meta;
  }
  RecordFlushIOStats();

  auto stream = event_logger_->LogToBuffer(log_buffer_);
  stream << "job" << job_context_->job_id << "event"
         << "flush_finished";
  stream << "lsm_state";
  stream.StartArray();
  auto vstorage = cfd_->current()->storage_info();
  for (int level = 0; level < vstorage->num_levels(); ++level) {
    stream << vstorage->NumLevelFiles(level);
  }
  stream.EndArray();

  return s;
}

Status FlushJob::WriteLevel0Table(const autovector<MemTable*>& mems,
                                  VersionEdit* edit, FileMetaData* meta) {
  AutoThreadOperationStageUpdater stage_updater(
      ThreadStatus::STAGE_FLUSH_WRITE_L0);
  db_mutex_->AssertHeld();
  const uint64_t start_micros = db_options_.env->NowMicros();
  // path 0 for level 0 file.
  meta->fd = FileDescriptor(versions_->NewFileNumber(), 0, 0);

  Version* base = cfd_->current();
  base->Ref();  // it is likely that we do not need this reference
  Status s;
  {
    db_mutex_->Unlock();
    if (log_buffer_) {
      log_buffer_->FlushBufferToLog();
    }
    std::vector<Iterator*> memtables;
    ReadOptions ro;
    ro.total_order_seek = true;
    Arena arena;
    uint64_t total_num_entries = 0, total_num_deletes = 0;
    size_t total_memory_usage = 0;
    for (MemTable* m : mems) {
      Log(InfoLogLevel::INFO_LEVEL, db_options_.info_log,
          "[%s] [JOB %d] Flushing memtable with next log file: %" PRIu64 "\n",
          cfd_->GetName().c_str(), job_context_->job_id, m->GetNextLogNumber());
      memtables.push_back(m->NewIterator(ro, &arena));
      total_num_entries += m->num_entries();
      total_num_deletes += m->num_deletes();
      total_memory_usage += m->ApproximateMemoryUsage();
    }

    event_logger_->Log() << "job" << job_context_->job_id << "event"
                         << "flush_started"
                         << "num_memtables" << mems.size() << "num_entries"
                         << total_num_entries << "num_deletes"
                         << total_num_deletes << "memory_usage"
                         << total_memory_usage;

    TableFileCreationInfo info;
    {
      ScopedArenaIterator iter(
          NewMergingIterator(&cfd_->internal_comparator(), &memtables[0],
                             static_cast<int>(memtables.size()), &arena));
      Log(InfoLogLevel::INFO_LEVEL, db_options_.info_log,
          "[%s] [JOB %d] Level-0 flush table #%" PRIu64 ": started",
          cfd_->GetName().c_str(), job_context_->job_id, meta->fd.GetNumber());

      TEST_SYNC_POINT_CALLBACK("FlushJob::WriteLevel0Table:output_compression",
                               &output_compression_);
      s = BuildTable(
          dbname_, db_options_.env, *cfd_->ioptions(), env_options_,
          cfd_->table_cache(), iter.get(), meta, cfd_->internal_comparator(),
          cfd_->int_tbl_prop_collector_factories(), existing_snapshots_,
          output_compression_, cfd_->ioptions()->compression_opts,
          mutable_cf_options_.paranoid_file_checks, cfd_->internal_stats(),
          Env::IO_HIGH, &info.table_properties);
      LogFlush(db_options_.info_log);
    }
    Log(InfoLogLevel::INFO_LEVEL, db_options_.info_log,
        "[%s] [JOB %d] Level-0 flush table #%" PRIu64 ": %" PRIu64
        " bytes %s"
        "%s",
        cfd_->GetName().c_str(), job_context_->job_id, meta->fd.GetNumber(),
        meta->fd.GetFileSize(), s.ToString().c_str(),
        meta->marked_for_compaction ? " (needs compaction)" : "");

    // output to event logger
    if (s.ok()) {
      info.db_name = dbname_;
      info.cf_name = cfd_->GetName();
      info.file_path = TableFileName(db_options_.db_paths,
                                     meta->fd.GetNumber(),
                                     meta->fd.GetPathId());
      info.file_size = meta->fd.GetFileSize();
      info.job_id = job_context_->job_id;
      EventHelpers::LogAndNotifyTableFileCreation(
          event_logger_, db_options_.listeners,
          meta->fd, info);
      TEST_SYNC_POINT("FlushJob::LogAndNotifyTableFileCreation()");
    }

    if (!db_options_.disableDataSync && output_file_directory_ != nullptr) {
      output_file_directory_->Fsync();
    }
    db_mutex_->Lock();
  }
  base->Unref();

  // re-acquire the most current version
  base = cfd_->current();

  // Note that if file_size is zero, the file has been deleted and
  // should not be added to the manifest.
  if (s.ok() && meta->fd.GetFileSize() > 0) {
    // if we have more than 1 background thread, then we cannot
    // insert files directly into higher levels because some other
    // threads could be concurrently producing compacted files for
    // that key range.
    // Add file to L0
    edit->AddFile(0 /* level */, meta->fd.GetNumber(), meta->fd.GetPathId(),
                  meta->fd.GetFileSize(), meta->smallest, meta->largest,
                  meta->smallest_seqno, meta->largest_seqno,
                  meta->marked_for_compaction);
  }

  InternalStats::CompactionStats stats(1);
  stats.micros = db_options_.env->NowMicros() - start_micros;
  stats.bytes_written = meta->fd.GetFileSize();
  cfd_->internal_stats()->AddCompactionStats(0 /* level */, stats);
  cfd_->internal_stats()->AddCFStats(InternalStats::BYTES_FLUSHED,
                                     meta->fd.GetFileSize());
  RecordTick(stats_, COMPACT_WRITE_BYTES, meta->fd.GetFileSize());
  return s;
}

}  // namespace rocksdb
