//  Copyright (c) 2013, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.
//
// Copyright (c) 2011 The LevelDB Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file. See the AUTHORS file for names of contributors.

#include "db/db_impl.h"

#ifndef __STDC_FORMAT_MACROS
#define __STDC_FORMAT_MACROS
#endif

#include <inttypes.h>
#include <stdint.h>

#include <algorithm>
#include <climits>
#include <cstdio>
#include <set>
#include <stdexcept>
#include <string>
#include <unordered_map>
#include <unordered_set>
#include <utility>
#include <vector>

#include "db/builder.h"
#include "db/compaction_job.h"
#include "db/db_iter.h"
#include "db/dbformat.h"
#include "db/event_helpers.h"
#include "db/filename.h"
#include "db/flush_job.h"
#include "db/forward_iterator.h"
#include "db/job_context.h"
#include "db/log_reader.h"
#include "db/log_writer.h"
#include "db/managed_iterator.h"
#include "db/memtable.h"
#include "db/memtable_list.h"
#include "db/merge_context.h"
#include "db/merge_helper.h"
#include "db/table_cache.h"
#include "db/table_properties_collector.h"
#include "db/transaction_log_impl.h"
#include "db/version_set.h"
#include "db/write_batch_internal.h"
#include "db/write_callback.h"
#include "db/writebuffer.h"
#include "port/likely.h"
#include "port/port.h"
#include "rocksdb/cache.h"
#include "rocksdb/compaction_filter.h"
#include "rocksdb/db.h"
#include "rocksdb/delete_scheduler.h"
#include "rocksdb/env.h"
#include "rocksdb/merge_operator.h"
#include "rocksdb/statistics.h"
#include "rocksdb/status.h"
#include "rocksdb/table.h"
#include "rocksdb/version.h"
#include "table/block.h"
#include "table/block_based_table_factory.h"
#include "table/merger.h"
#include "table/table_builder.h"
#include "table/two_level_iterator.h"
#include "util/auto_roll_logger.h"
#include "util/autovector.h"
#include "util/build_version.h"
#include "util/coding.h"
#include "util/compression.h"
#include "util/crc32c.h"
#include "util/db_info_dumper.h"
#include "util/file_reader_writer.h"
#include "util/file_util.h"
#include "util/hash_linklist_rep.h"
#include "util/hash_skiplist_rep.h"
#include "util/iostats_context_imp.h"
#include "util/log_buffer.h"
#include "util/logging.h"
#include "util/mutexlock.h"
#include "util/perf_context_imp.h"
#include "util/stop_watch.h"
#include "util/string_util.h"
#include "util/sync_point.h"
#include "util/thread_status_updater.h"
#include "util/thread_status_util.h"
#include "util/xfunc.h"

namespace rocksdb {

const std::string kDefaultColumnFamilyName("default");

void DumpRocksDBBuildVersion(Logger * log);

struct DBImpl::WriteContext {
  autovector<SuperVersion*> superversions_to_free_;
  autovector<MemTable*> memtables_to_free_;

  ~WriteContext() {
    for (auto& sv : superversions_to_free_) {
      delete sv;
    }
    for (auto& m : memtables_to_free_) {
      delete m;
    }
  }
};

Options SanitizeOptions(const std::string& dbname,
                        const InternalKeyComparator* icmp,
                        const Options& src) {
  auto db_options = SanitizeOptions(dbname, DBOptions(src));
  auto cf_options = SanitizeOptions(db_options, icmp, ColumnFamilyOptions(src));
  return Options(db_options, cf_options);
}

DBOptions SanitizeOptions(const std::string& dbname, const DBOptions& src) {
  DBOptions result = src;

  // result.max_open_files means an "infinite" open files.
  if (result.max_open_files != -1) {
    int max_max_open_files = port::GetMaxOpenFiles();
    if (max_max_open_files == -1) {
      max_max_open_files = 1000000;
    }
    ClipToRange(&result.max_open_files, 20, max_max_open_files);
  }

  if (result.info_log == nullptr) {
    Status s = CreateLoggerFromOptions(dbname, result.db_log_dir, src.env,
                                       result, &result.info_log);
    if (!s.ok()) {
      // No place suitable for logging
      result.info_log = nullptr;
    }
  }
  result.env->IncBackgroundThreadsIfNeeded(src.max_background_compactions,
                                           Env::Priority::LOW);
  result.env->IncBackgroundThreadsIfNeeded(src.max_background_flushes,
                                           Env::Priority::HIGH);

  if (result.rate_limiter.get() != nullptr) {
    if (result.bytes_per_sync == 0) {
      result.bytes_per_sync = 1024 * 1024;
    }
  }

  if (result.wal_dir.empty()) {
    // Use dbname as default
    result.wal_dir = dbname;
  }
  if (result.wal_dir.back() == '/') {
    result.wal_dir = result.wal_dir.substr(0, result.wal_dir.size() - 1);
  }

  if (result.db_paths.size() == 0) {
    result.db_paths.emplace_back(dbname, std::numeric_limits<uint64_t>::max());
  }

  if (result.compaction_readahead_size > 0) {
    result.new_table_reader_for_compaction_inputs = true;
  }

  return result;
}

namespace {

Status SanitizeOptionsByTable(
    const DBOptions& db_opts,
    const std::vector<ColumnFamilyDescriptor>& column_families) {
  Status s;
  for (auto cf : column_families) {
    s = cf.options.table_factory->SanitizeOptions(db_opts, cf.options);
    if (!s.ok()) {
      return s;
    }
  }
  return Status::OK();
}

CompressionType GetCompressionFlush(const ImmutableCFOptions& ioptions) {
  // Compressing memtable flushes might not help unless the sequential load
  // optimization is used for leveled compaction. Otherwise the CPU and
  // latency overhead is not offset by saving much space.

  bool can_compress;

  if (ioptions.compaction_style == kCompactionStyleUniversal) {
    can_compress =
        (ioptions.compaction_options_universal.compression_size_percent < 0);
  } else {
    // For leveled compress when min_level_to_compress == 0.
    can_compress = ioptions.compression_per_level.empty() ||
                   ioptions.compression_per_level[0] != kNoCompression;
  }

  if (can_compress) {
    return ioptions.compression;
  } else {
    return kNoCompression;
  }
}

void DumpSupportInfo(Logger* logger) {
  Log(InfoLogLevel::INFO_LEVEL, logger, "Compression algorithms supported:");
  Log(InfoLogLevel::INFO_LEVEL, logger, "\tSnappy supported: %d",
      Snappy_Supported());
  Log(InfoLogLevel::INFO_LEVEL, logger, "\tZlib supported: %d",
      Zlib_Supported());
  Log(InfoLogLevel::INFO_LEVEL, logger, "\tBzip supported: %d",
      BZip2_Supported());
  Log(InfoLogLevel::INFO_LEVEL, logger, "\tLZ4 supported: %d", LZ4_Supported());
  Log(InfoLogLevel::INFO_LEVEL, logger, "Fast CRC32 supported: %d",
      crc32c::IsFastCrc32Supported());
}

}  // namespace

DBImpl::DBImpl(const DBOptions& options, const std::string& dbname)
    : env_(options.env),
      dbname_(dbname),
      db_options_(SanitizeOptions(dbname, options)),
      stats_(db_options_.statistics.get()),
      db_lock_(nullptr),
      mutex_(stats_, env_, DB_MUTEX_WAIT_MICROS, options.use_adaptive_mutex),
      shutting_down_(false),
      bg_cv_(&mutex_),
      logfile_number_(0),
      log_dir_synced_(false),
      log_empty_(true),
      default_cf_handle_(nullptr),
      log_sync_cv_(&mutex_),
      total_log_size_(0),
      max_total_in_memory_state_(0),
      is_snapshot_supported_(true),
      write_buffer_(options.db_write_buffer_size),
      write_controller_(options.delayed_write_rate),
      last_batch_group_size_(0),
      unscheduled_flushes_(0),
      unscheduled_compactions_(0),
      bg_compaction_scheduled_(0),
      bg_manual_only_(0),
      bg_flush_scheduled_(0),
      manual_compaction_(nullptr),
      disable_delete_obsolete_files_(0),
      delete_obsolete_files_next_run_(
          options.env->NowMicros() +
          db_options_.delete_obsolete_files_period_micros),
      last_stats_dump_time_microsec_(0),
      next_job_id_(1),
      flush_on_destroy_(false),
      env_options_(db_options_),
#ifndef ROCKSDB_LITE
      wal_manager_(db_options_, env_options_),
#endif  // ROCKSDB_LITE
      event_logger_(db_options_.info_log.get()),
      bg_work_gate_closed_(false),
      refitting_level_(false),
      opened_successfully_(false) {
  env_->GetAbsolutePath(dbname, &db_absolute_path_);

  // Reserve ten files or so for other uses and give the rest to TableCache.
  // Give a large number for setting of "infinite" open files.
  const int table_cache_size = (db_options_.max_open_files == -1) ?
        4194304 : db_options_.max_open_files - 10;
  table_cache_ =
      NewLRUCache(table_cache_size, db_options_.table_cache_numshardbits);

  versions_.reset(new VersionSet(dbname_, &db_options_, env_options_,
                                 table_cache_.get(), &write_buffer_,
                                 &write_controller_));
  column_family_memtables_.reset(new ColumnFamilyMemTablesImpl(
      versions_->GetColumnFamilySet(), &flush_scheduler_));

  DumpRocksDBBuildVersion(db_options_.info_log.get());
  DumpDBFileSummary(db_options_, dbname_);
  db_options_.Dump(db_options_.info_log.get());
  DumpSupportInfo(db_options_.info_log.get());

  LogFlush(db_options_.info_log);
}

// Will lock the mutex_,  will wait for completion if wait is true
void DBImpl::CancelAllBackgroundWork(bool wait) {
  InstrumentedMutexLock l(&mutex_);
  shutting_down_.store(true, std::memory_order_release);
  bg_cv_.SignalAll();
  if (!wait) {
    return;
  }
  // Wait for background work to finish
  while (bg_compaction_scheduled_ || bg_flush_scheduled_) {
    bg_cv_.Wait();
  }
}

DBImpl::~DBImpl() {
  mutex_.Lock();

  if (!shutting_down_.load(std::memory_order_acquire) && flush_on_destroy_) {
    for (auto cfd : *versions_->GetColumnFamilySet()) {
      if (!cfd->IsDropped() && !cfd->mem()->IsEmpty()) {
        cfd->Ref();
        mutex_.Unlock();
        FlushMemTable(cfd, FlushOptions());
        mutex_.Lock();
        cfd->Unref();
      }
    }
    versions_->GetColumnFamilySet()->FreeDeadColumnFamilies();
  }
  mutex_.Unlock();
  // CancelAllBackgroundWork called with false means we just set the shutdown
  // marker. After this we do a variant of the waiting and unschedule work
  // (to consider: moving all the waiting into CancelAllBackgroundWork(true))
  CancelAllBackgroundWork(false);
  int compactions_unscheduled = env_->UnSchedule(this, Env::Priority::LOW);
  int flushes_unscheduled = env_->UnSchedule(this, Env::Priority::HIGH);
  mutex_.Lock();
  bg_compaction_scheduled_ -= compactions_unscheduled;
  bg_flush_scheduled_ -= flushes_unscheduled;

  // Wait for background work to finish
  while (bg_compaction_scheduled_ || bg_flush_scheduled_) {
    bg_cv_.Wait();
  }
  EraseThreadStatusDbInfo();
  flush_scheduler_.Clear();

  while (!flush_queue_.empty()) {
    auto cfd = PopFirstFromFlushQueue();
    if (cfd->Unref()) {
      delete cfd;
    }
  }
  while (!compaction_queue_.empty()) {
    auto cfd = PopFirstFromCompactionQueue();
    if (cfd->Unref()) {
      delete cfd;
    }
  }

  if (default_cf_handle_ != nullptr) {
    // we need to delete handle outside of lock because it does its own locking
    mutex_.Unlock();
    delete default_cf_handle_;
    mutex_.Lock();
  }

  // Clean up obsolete files due to SuperVersion release.
  // (1) Need to delete to obsolete files before closing because RepairDB()
  // scans all existing files in the file system and builds manifest file.
  // Keeping obsolete files confuses the repair process.
  // (2) Need to check if we Open()/Recover() the DB successfully before
  // deleting because if VersionSet recover fails (may be due to corrupted
  // manifest file), it is not able to identify live files correctly. As a
  // result, all "live" files can get deleted by accident. However, corrupted
  // manifest is recoverable by RepairDB().
  if (opened_successfully_) {
    JobContext job_context(next_job_id_.fetch_add(1));
    FindObsoleteFiles(&job_context, true);

    mutex_.Unlock();
    // manifest number starting from 2
    job_context.manifest_file_number = 1;
    if (job_context.HaveSomethingToDelete()) {
      PurgeObsoleteFiles(job_context);
    }
    job_context.Clean();
    mutex_.Lock();
  }

  for (auto l : logs_to_free_) {
    delete l;
  }
  for (auto& log : logs_) {
    log.ClearWriter();
  }
  logs_.clear();

  // versions need to be destroyed before table_cache since it can hold
  // references to table_cache.
  versions_.reset();
  mutex_.Unlock();
  if (db_lock_ != nullptr) {
    env_->UnlockFile(db_lock_);
  }

  LogFlush(db_options_.info_log);
}

Status DBImpl::NewDB() {
  VersionEdit new_db;
  new_db.SetLogNumber(0);
  new_db.SetNextFile(2);
  new_db.SetLastSequence(0);

  Status s;

  Log(InfoLogLevel::INFO_LEVEL,
      db_options_.info_log, "Creating manifest 1 \n");
  const std::string manifest = DescriptorFileName(dbname_, 1);
  {
    unique_ptr<WritableFile> file;
    EnvOptions env_options = env_->OptimizeForManifestWrite(env_options_);
    s = env_->NewWritableFile(manifest, &file, env_options);
    if (!s.ok()) {
      return s;
    }
    file->SetPreallocationBlockSize(db_options_.manifest_preallocation_size);
    unique_ptr<WritableFileWriter> file_writer(
        new WritableFileWriter(std::move(file), env_options));
    log::Writer log(std::move(file_writer));
    std::string record;
    new_db.EncodeTo(&record);
    s = log.AddRecord(record);
    if (s.ok()) {
      s = SyncManifest(env_, &db_options_, log.file());
    }
  }
  if (s.ok()) {
    // Make "CURRENT" file that points to the new manifest file.
    s = SetCurrentFile(env_, dbname_, 1, directories_.GetDbDir());
  } else {
    env_->DeleteFile(manifest);
  }
  return s;
}

void DBImpl::MaybeIgnoreError(Status* s) const {
  if (s->ok() || db_options_.paranoid_checks) {
    // No change needed
  } else {
    Log(InfoLogLevel::WARN_LEVEL,
        db_options_.info_log, "Ignoring error %s", s->ToString().c_str());
    *s = Status::OK();
  }
}

const Status DBImpl::CreateArchivalDirectory() {
  if (db_options_.WAL_ttl_seconds > 0 || db_options_.WAL_size_limit_MB > 0) {
    std::string archivalPath = ArchivalDirectory(db_options_.wal_dir);
    return env_->CreateDirIfMissing(archivalPath);
  }
  return Status::OK();
}

void DBImpl::PrintStatistics() {
  auto dbstats = db_options_.statistics.get();
  if (dbstats) {
    Log(InfoLogLevel::WARN_LEVEL, db_options_.info_log,
        "STATISTICS:\n %s",
        dbstats->ToString().c_str());
  }
}

void DBImpl::MaybeDumpStats() {
  if (db_options_.stats_dump_period_sec == 0) return;

  const uint64_t now_micros = env_->NowMicros();

  if (last_stats_dump_time_microsec_ +
      db_options_.stats_dump_period_sec * 1000000
      <= now_micros) {
    // Multiple threads could race in here simultaneously.
    // However, the last one will update last_stats_dump_time_microsec_
    // atomically. We could see more than one dump during one dump
    // period in rare cases.
    last_stats_dump_time_microsec_ = now_micros;

#ifndef ROCKSDB_LITE
    bool tmp1 = false;
    bool tmp2 = false;
    DBPropertyType cf_property_type =
        GetPropertyType(DB::Properties::kCFStats, &tmp1, &tmp2);
    DBPropertyType db_property_type =
        GetPropertyType(DB::Properties::kDBStats, &tmp1, &tmp2);
    std::string stats;
    {
      InstrumentedMutexLock l(&mutex_);
      for (auto cfd : *versions_->GetColumnFamilySet()) {
        cfd->internal_stats()->GetStringProperty(cf_property_type,
                                                 DB::Properties::kCFStats,
                                                 &stats);
      }
      default_cf_internal_stats_->GetStringProperty(db_property_type,
                                                    DB::Properties::kDBStats,
                                                    &stats);
    }
    Log(InfoLogLevel::WARN_LEVEL,
        db_options_.info_log, "------- DUMPING STATS -------");
    Log(InfoLogLevel::WARN_LEVEL,
        db_options_.info_log, "%s", stats.c_str());
#endif  // !ROCKSDB_LITE

    PrintStatistics();
  }
}

// * Returns the list of live files in 'sst_live'
// If it's doing full scan:
// * Returns the list of all files in the filesystem in
// 'full_scan_candidate_files'.
// Otherwise, gets obsolete files from VersionSet.
// no_full_scan = true -- never do the full scan using GetChildren()
// force = false -- don't force the full scan, except every
//  db_options_.delete_obsolete_files_period_micros
// force = true -- force the full scan
void DBImpl::FindObsoleteFiles(JobContext* job_context, bool force,
                               bool no_full_scan) {
  mutex_.AssertHeld();

  // if deletion is disabled, do nothing
  if (disable_delete_obsolete_files_ > 0) {
    return;
  }

  bool doing_the_full_scan = false;

  // logic for figurint out if we're doing the full scan
  if (no_full_scan) {
    doing_the_full_scan = false;
  } else if (force || db_options_.delete_obsolete_files_period_micros == 0) {
    doing_the_full_scan = true;
  } else {
    const uint64_t now_micros = env_->NowMicros();
    if (delete_obsolete_files_next_run_ < now_micros) {
      doing_the_full_scan = true;
      delete_obsolete_files_next_run_ =
          now_micros + db_options_.delete_obsolete_files_period_micros;
    }
  }

  // don't delete files that might be currently written to from compaction
  // threads
  // Since job_context->min_pending_output is set, until file scan finishes,
  // mutex_ cannot be released. Otherwise, we might see no min_pending_output
  // here but later find newer generated unfinalized files while scannint.
  if (!pending_outputs_.empty()) {
    job_context->min_pending_output = *pending_outputs_.begin();
  } else {
    // delete all of them
    job_context->min_pending_output = std::numeric_limits<uint64_t>::max();
  }

  // Get obsolete files.  This function will also update the list of
  // pending files in VersionSet().
  versions_->GetObsoleteFiles(&job_context->sst_delete_files,
                              job_context->min_pending_output);

  // store the current filenum, lognum, etc
  job_context->manifest_file_number = versions_->manifest_file_number();
  job_context->pending_manifest_file_number =
      versions_->pending_manifest_file_number();
  job_context->log_number = versions_->MinLogNumber();
  job_context->prev_log_number = versions_->prev_log_number();

  versions_->AddLiveFiles(&job_context->sst_live);
  if (doing_the_full_scan) {
    for (uint32_t path_id = 0; path_id < db_options_.db_paths.size();
         path_id++) {
      // set of all files in the directory. We'll exclude files that are still
      // alive in the subsequent processings.
      std::vector<std::string> files;
      env_->GetChildren(db_options_.db_paths[path_id].path,
                        &files);  // Ignore errors
      for (std::string file : files) {
        // TODO(icanadi) clean up this mess to avoid having one-off "/" prefixes
        job_context->full_scan_candidate_files.emplace_back("/" + file,
                                                            path_id);
      }
    }

    //Add log files in wal_dir
    if (db_options_.wal_dir != dbname_) {
      std::vector<std::string> log_files;
      env_->GetChildren(db_options_.wal_dir, &log_files);  // Ignore errors
      for (std::string log_file : log_files) {
        job_context->full_scan_candidate_files.emplace_back(log_file, 0);
      }
    }
    // Add info log files in db_log_dir
    if (!db_options_.db_log_dir.empty() && db_options_.db_log_dir != dbname_) {
      std::vector<std::string> info_log_files;
      // Ignore errors
      env_->GetChildren(db_options_.db_log_dir, &info_log_files);
      for (std::string log_file : info_log_files) {
        job_context->full_scan_candidate_files.emplace_back(log_file, 0);
      }
    }
  }

  if (!alive_log_files_.empty()) {
    uint64_t min_log_number = versions_->MinLogNumber();
    // find newly obsoleted log files
    while (alive_log_files_.begin()->number < min_log_number) {
      auto& earliest = *alive_log_files_.begin();
      job_context->log_delete_files.push_back(earliest.number);
      total_log_size_ -= earliest.size;
      alive_log_files_.pop_front();
      // Current log should always stay alive since it can't have
      // number < MinLogNumber().
      assert(alive_log_files_.size());
    }
    while (!logs_.empty() && logs_.front().number < min_log_number) {
      auto& log = logs_.front();
      if (log.getting_synced) {
        log_sync_cv_.Wait();
        // logs_ could have changed while we were waiting.
        continue;
      }
      logs_to_free_.push_back(log.ReleaseWriter());
      logs_.pop_front();
    }
    // Current log cannot be obsolete.
    assert(!logs_.empty());
  }

  // We're just cleaning up for DB::Write().
  assert(job_context->logs_to_free.empty());
  job_context->logs_to_free = logs_to_free_;
  logs_to_free_.clear();
}

namespace {
bool CompareCandidateFile(const JobContext::CandidateFileInfo& first,
                          const JobContext::CandidateFileInfo& second) {
  if (first.file_name > second.file_name) {
    return true;
  } else if (first.file_name < second.file_name) {
    return false;
  } else {
    return (first.path_id > second.path_id);
  }
}
};  // namespace

// Diffs the files listed in filenames and those that do not
// belong to live files are posibly removed. Also, removes all the
// files in sst_delete_files and log_delete_files.
// It is not necessary to hold the mutex when invoking this method.
void DBImpl::PurgeObsoleteFiles(const JobContext& state) {
  // we'd better have sth to delete
  assert(state.HaveSomethingToDelete());

  // this checks if FindObsoleteFiles() was run before. If not, don't do
  // PurgeObsoleteFiles(). If FindObsoleteFiles() was run, we need to also
  // run PurgeObsoleteFiles(), even if disable_delete_obsolete_files_ is true
  if (state.manifest_file_number == 0) {
    return;
  }

  // Now, convert live list to an unordered map, WITHOUT mutex held;
  // set is slow.
  std::unordered_map<uint64_t, const FileDescriptor*> sst_live_map;
  for (const FileDescriptor& fd : state.sst_live) {
    sst_live_map[fd.GetNumber()] = &fd;
  }

  auto candidate_files = state.full_scan_candidate_files;
  candidate_files.reserve(candidate_files.size() +
                          state.sst_delete_files.size() +
                          state.log_delete_files.size());
  // We may ignore the dbname when generating the file names.
  const char* kDumbDbName = "";
  for (auto file : state.sst_delete_files) {
    candidate_files.emplace_back(
        MakeTableFileName(kDumbDbName, file->fd.GetNumber()),
        file->fd.GetPathId());
    delete file;
  }

  for (auto file_num : state.log_delete_files) {
    if (file_num > 0) {
      candidate_files.emplace_back(LogFileName(kDumbDbName, file_num).substr(1),
                                   0);
    }
  }

  // dedup state.candidate_files so we don't try to delete the same
  // file twice
  sort(candidate_files.begin(), candidate_files.end(), CompareCandidateFile);
  candidate_files.erase(unique(candidate_files.begin(), candidate_files.end()),
                        candidate_files.end());

  std::vector<std::string> old_info_log_files;
  InfoLogPrefix info_log_prefix(!db_options_.db_log_dir.empty(), dbname_);
  for (const auto& candidate_file : candidate_files) {
    std::string to_delete = candidate_file.file_name;
    uint32_t path_id = candidate_file.path_id;
    uint64_t number;
    FileType type;
    // Ignore file if we cannot recognize it.
    if (!ParseFileName(to_delete, &number, info_log_prefix.prefix, &type)) {
      continue;
    }

    bool keep = true;
    switch (type) {
      case kLogFile:
        keep = ((number >= state.log_number) ||
                (number == state.prev_log_number));
        break;
      case kDescriptorFile:
        // Keep my manifest file, and any newer incarnations'
        // (can happen during manifest roll)
        keep = (number >= state.manifest_file_number);
        break;
      case kTableFile:
        // If the second condition is not there, this makes
        // DontDeletePendingOutputs fail
        keep = (sst_live_map.find(number) != sst_live_map.end()) ||
               number >= state.min_pending_output;
        break;
      case kTempFile:
        // Any temp files that are currently being written to must
        // be recorded in pending_outputs_, which is inserted into "live".
        // Also, SetCurrentFile creates a temp file when writing out new
        // manifest, which is equal to state.pending_manifest_file_number. We
        // should not delete that file
        keep = (sst_live_map.find(number) != sst_live_map.end()) ||
               (number == state.pending_manifest_file_number);
        break;
      case kInfoLogFile:
        keep = true;
        if (number != 0) {
          old_info_log_files.push_back(to_delete);
        }
        break;
      case kCurrentFile:
      case kDBLockFile:
      case kIdentityFile:
      case kMetaDatabase:
        keep = true;
        break;
    }

    if (keep) {
      continue;
    }

    std::string fname;
    if (type == kTableFile) {
      // evict from cache
      TableCache::Evict(table_cache_.get(), number);
      fname = TableFileName(db_options_.db_paths, number, path_id);
    } else {
      fname = ((type == kLogFile) ?
          db_options_.wal_dir : dbname_) + "/" + to_delete;
    }

#ifndef ROCKSDB_LITE
    if (type == kLogFile && (db_options_.WAL_ttl_seconds > 0 ||
                              db_options_.WAL_size_limit_MB > 0)) {
      wal_manager_.ArchiveWALFile(fname, number);
      continue;
    }
#endif  // !ROCKSDB_LITE
    Status file_deletion_status;
    if (type == kTableFile && path_id == 0) {
      file_deletion_status = DeleteOrMoveToTrash(&db_options_, fname);
    } else {
      file_deletion_status = env_->DeleteFile(fname);
    }
    if (file_deletion_status.ok()) {
      Log(InfoLogLevel::DEBUG_LEVEL, db_options_.info_log,
          "[JOB %d] Delete %s type=%d #%" PRIu64 " -- %s\n", state.job_id,
          fname.c_str(), type, number,
          file_deletion_status.ToString().c_str());
    } else {
      Log(InfoLogLevel::ERROR_LEVEL, db_options_.info_log,
          "[JOB %d] Failed to delete %s type=%d #%" PRIu64 " -- %s\n",
          state.job_id, fname.c_str(), type, number,
          file_deletion_status.ToString().c_str());
    }
    if (type == kTableFile) {
      EventHelpers::LogAndNotifyTableFileDeletion(
          &event_logger_, state.job_id, number, fname,
          file_deletion_status, GetName(),
          db_options_.listeners);
    }
  }

  // Delete old info log files.
  size_t old_info_log_file_count = old_info_log_files.size();
  if (old_info_log_file_count >= db_options_.keep_log_file_num) {
    std::sort(old_info_log_files.begin(), old_info_log_files.end());
    size_t end = old_info_log_file_count - db_options_.keep_log_file_num;
    for (unsigned int i = 0; i <= end; i++) {
      std::string& to_delete = old_info_log_files.at(i);
      std::string full_path_to_delete = (db_options_.db_log_dir.empty() ?
           dbname_ : db_options_.db_log_dir) + "/" + to_delete;
      Log(InfoLogLevel::INFO_LEVEL, db_options_.info_log,
          "[JOB %d] Delete info log file %s\n", state.job_id,
          full_path_to_delete.c_str());
      Status s = env_->DeleteFile(full_path_to_delete);
      if (!s.ok()) {
        Log(InfoLogLevel::ERROR_LEVEL, db_options_.info_log,
            "[JOB %d] Delete info log file %s FAILED -- %s\n", state.job_id,
            to_delete.c_str(), s.ToString().c_str());
      }
    }
  }
#ifndef ROCKSDB_LITE
  wal_manager_.PurgeObsoleteWALFiles();
#endif  // ROCKSDB_LITE
  LogFlush(db_options_.info_log);
}

void DBImpl::DeleteObsoleteFiles() {
  mutex_.AssertHeld();
  JobContext job_context(next_job_id_.fetch_add(1));
  FindObsoleteFiles(&job_context, true);

  mutex_.Unlock();
  if (job_context.HaveSomethingToDelete()) {
    PurgeObsoleteFiles(job_context);
  }
  job_context.Clean();
  mutex_.Lock();
}

Status DBImpl::Directories::CreateAndNewDirectory(
    Env* env, const std::string& dirname,
    std::unique_ptr<Directory>* directory) const {
  // We call CreateDirIfMissing() as the directory may already exist (if we
  // are reopening a DB), when this happens we don't want creating the
  // directory to cause an error. However, we need to check if creating the
  // directory fails or else we may get an obscure message about the lock
  // file not existing. One real-world example of this occurring is if
  // env->CreateDirIfMissing() doesn't create intermediate directories, e.g.
  // when dbname_ is "dir/db" but when "dir" doesn't exist.
  Status s = env->CreateDirIfMissing(dirname);
  if (!s.ok()) {
    return s;
  }
  return env->NewDirectory(dirname, directory);
}

Status DBImpl::Directories::SetDirectories(
    Env* env, const std::string& dbname, const std::string& wal_dir,
    const std::vector<DbPath>& data_paths) {
  Status s = CreateAndNewDirectory(env, dbname, &db_dir_);
  if (!s.ok()) {
    return s;
  }
  if (!wal_dir.empty() && dbname != wal_dir) {
    s = CreateAndNewDirectory(env, wal_dir, &wal_dir_);
    if (!s.ok()) {
      return s;
    }
  }

  data_dirs_.clear();
  for (auto& p : data_paths) {
    const std::string db_path = p.path;
    if (db_path == dbname) {
      data_dirs_.emplace_back(nullptr);
    } else {
      std::unique_ptr<Directory> path_directory;
      s = CreateAndNewDirectory(env, db_path, &path_directory);
      if (!s.ok()) {
        return s;
      }
      data_dirs_.emplace_back(path_directory.release());
    }
  }
  assert(data_dirs_.size() == data_paths.size());
  return Status::OK();
}

Directory* DBImpl::Directories::GetDataDir(size_t path_id) {
  assert(path_id < data_dirs_.size());
  Directory* ret_dir = data_dirs_[path_id].get();
  if (ret_dir == nullptr) {
    // Should use db_dir_
    return db_dir_.get();
  }
  return ret_dir;
}

Status DBImpl::Recover(
    const std::vector<ColumnFamilyDescriptor>& column_families, bool read_only,
    bool error_if_log_file_exist) {
  mutex_.AssertHeld();

  bool is_new_db = false;
  assert(db_lock_ == nullptr);
  if (!read_only) {
    Status s = directories_.SetDirectories(env_, dbname_, db_options_.wal_dir,
                                           db_options_.db_paths);
    if (!s.ok()) {
      return s;
    }

    s = env_->LockFile(LockFileName(dbname_), &db_lock_);
    if (!s.ok()) {
      return s;
    }

    s = env_->FileExists(CurrentFileName(dbname_));
    if (s.IsNotFound()) {
      if (db_options_.create_if_missing) {
        s = NewDB();
        is_new_db = true;
        if (!s.ok()) {
          return s;
        }
      } else {
        return Status::InvalidArgument(
            dbname_, "does not exist (create_if_missing is false)");
      }
    } else if (s.ok()) {
      if (db_options_.error_if_exists) {
        return Status::InvalidArgument(
            dbname_, "exists (error_if_exists is true)");
      }
    } else {
      // Unexpected error reading file
      assert(s.IsIOError());
      return s;
    }
    // Check for the IDENTITY file and create it if not there
    s = env_->FileExists(IdentityFileName(dbname_));
    if (s.IsNotFound()) {
      s = SetIdentityFile(env_, dbname_);
      if (!s.ok()) {
        return s;
      }
    } else if (!s.ok()) {
      assert(s.IsIOError());
      return s;
    }
  }

  Status s = versions_->Recover(column_families, read_only);
  if (db_options_.paranoid_checks && s.ok()) {
    s = CheckConsistency();
  }
  if (s.ok()) {
    SequenceNumber max_sequence(kMaxSequenceNumber);
    default_cf_handle_ = new ColumnFamilyHandleImpl(
        versions_->GetColumnFamilySet()->GetDefault(), this, &mutex_);
    default_cf_internal_stats_ = default_cf_handle_->cfd()->internal_stats();
    single_column_family_mode_ =
        versions_->GetColumnFamilySet()->NumberOfColumnFamilies() == 1;

    // Recover from all newer log files than the ones named in the
    // descriptor (new log files may have been added by the previous
    // incarnation without registering them in the descriptor).
    //
    // Note that prev_log_number() is no longer used, but we pay
    // attention to it in case we are recovering a database
    // produced by an older version of rocksdb.
    const uint64_t min_log = versions_->MinLogNumber();
    const uint64_t prev_log = versions_->prev_log_number();
    std::vector<std::string> filenames;
    s = env_->GetChildren(db_options_.wal_dir, &filenames);
    if (!s.ok()) {
      return s;
    }

    std::vector<uint64_t> logs;
    for (size_t i = 0; i < filenames.size(); i++) {
      uint64_t number;
      FileType type;
      if (ParseFileName(filenames[i], &number, &type) && type == kLogFile) {
        if (is_new_db) {
          return Status::Corruption(
              "While creating a new Db, wal_dir contains "
              "existing log file: ",
              filenames[i]);
        } else if ((number >= min_log) || (number == prev_log)) {
          logs.push_back(number);
        }
      }
    }

    if (logs.size() > 0 && error_if_log_file_exist) {
      return Status::Corruption(""
          "The db was opened in readonly mode with error_if_log_file_exist"
          "flag but a log file already exists");
    }

    if (!logs.empty()) {
      // Recover in the order in which the logs were generated
      std::sort(logs.begin(), logs.end());
      s = RecoverLogFiles(logs, &max_sequence, read_only);
      if (!s.ok()) {
        // Clear memtables if recovery failed
        for (auto cfd : *versions_->GetColumnFamilySet()) {
          cfd->CreateNewMemtable(*cfd->GetLatestMutableCFOptions(),
                                 kMaxSequenceNumber);
        }
      }
    }
    SetTickerCount(stats_, SEQUENCE_NUMBER, versions_->LastSequence());
  }

  // Initial value
  max_total_in_memory_state_ = 0;
  for (auto cfd : *versions_->GetColumnFamilySet()) {
    auto* mutable_cf_options = cfd->GetLatestMutableCFOptions();
    max_total_in_memory_state_ += mutable_cf_options->write_buffer_size *
                                  mutable_cf_options->max_write_buffer_number;
  }

  return s;
}

// REQUIRES: log_numbers are sorted in ascending order
Status DBImpl::RecoverLogFiles(const std::vector<uint64_t>& log_numbers,
                               SequenceNumber* max_sequence, bool read_only) {
  struct LogReporter : public log::Reader::Reporter {
    Env* env;
    Logger* info_log;
    const char* fname;
    Status* status;  // nullptr if db_options_.paranoid_checks==false
    virtual void Corruption(size_t bytes, const Status& s) override {
      Log(InfoLogLevel::WARN_LEVEL,
          info_log, "%s%s: dropping %d bytes; %s",
          (this->status == nullptr ? "(ignoring error) " : ""),
          fname, static_cast<int>(bytes), s.ToString().c_str());
      if (this->status != nullptr && this->status->ok()) {
        *this->status = s;
      }
    }
  };

  mutex_.AssertHeld();
  Status status;
  std::unordered_map<int, VersionEdit> version_edits;
  // no need to refcount because iteration is under mutex
  for (auto cfd : *versions_->GetColumnFamilySet()) {
    VersionEdit edit;
    edit.SetColumnFamily(cfd->GetID());
    version_edits.insert({cfd->GetID(), edit});
  }
  int job_id = next_job_id_.fetch_add(1);
  {
    auto stream = event_logger_.Log();
    stream << "job" << job_id << "event"
           << "recovery_started";
    stream << "log_files";
    stream.StartArray();
    for (auto log_number : log_numbers) {
      stream << log_number;
    }
    stream.EndArray();
  }

  bool continue_replay_log = true;
  for (auto log_number : log_numbers) {
    // The previous incarnation may not have written any MANIFEST
    // records after allocating this log number.  So we manually
    // update the file number allocation counter in VersionSet.
    versions_->MarkFileNumberUsedDuringRecovery(log_number);
    // Open the log file
    std::string fname = LogFileName(db_options_.wal_dir, log_number);
    unique_ptr<SequentialFileReader> file_reader;
    {
      unique_ptr<SequentialFile> file;
      status = env_->NewSequentialFile(fname, &file, env_options_);
      if (!status.ok()) {
        MaybeIgnoreError(&status);
        if (!status.ok()) {
          return status;
        } else {
          // Fail with one log file, but that's ok.
          // Try next one.
          continue;
        }
      }
      file_reader.reset(new SequentialFileReader(std::move(file)));
    }

    // Create the log reader.
    LogReporter reporter;
    reporter.env = env_;
    reporter.info_log = db_options_.info_log.get();
    reporter.fname = fname.c_str();
    if (!db_options_.paranoid_checks ||
        db_options_.wal_recovery_mode ==
            WALRecoveryMode::kSkipAnyCorruptedRecords) {
      reporter.status = nullptr;
    } else {
      reporter.status = &status;
    }
    // We intentially make log::Reader do checksumming even if
    // paranoid_checks==false so that corruptions cause entire commits
    // to be skipped instead of propagating bad information (like overly
    // large sequence numbers).
    log::Reader reader(std::move(file_reader), &reporter, true /*checksum*/,
                       0 /*initial_offset*/);
    Log(InfoLogLevel::INFO_LEVEL, db_options_.info_log,
        "Recovering log #%" PRIu64 " mode %d skip-recovery %d", log_number,
        db_options_.wal_recovery_mode, !continue_replay_log);

    // Determine if we should tolerate incomplete records at the tail end of the
    // log
    bool report_eof_inconsistency;
    if (db_options_.wal_recovery_mode ==
        WALRecoveryMode::kAbsoluteConsistency) {
      // in clean shutdown we don't expect any error in the log files
      report_eof_inconsistency = true;
    } else {
      // for other modes ignore only incomplete records in the last log file
      // which is presumably due to write in progress during restart
      report_eof_inconsistency = false;

      // TODO krad: Evaluate if we need to move to a more strict mode where we
      // restrict the inconsistency to only the last log
    }

    // Read all the records and add to a memtable
    std::string scratch;
    Slice record;
    WriteBatch batch;

    if (!continue_replay_log) {
      uint64_t bytes;
      if (env_->GetFileSize(fname, &bytes).ok()) {
        auto info_log = db_options_.info_log.get();
        Log(InfoLogLevel::WARN_LEVEL, info_log, "%s: dropping %d bytes",
            fname.c_str(), static_cast<int>(bytes));
      }
    }

    while (continue_replay_log &&
           reader.ReadRecord(&record, &scratch, report_eof_inconsistency) &&
           status.ok()) {
      if (record.size() < 12) {
        reporter.Corruption(record.size(),
                            Status::Corruption("log record too small"));
        continue;
      }
      WriteBatchInternal::SetContents(&batch, record);

      // If column family was not found, it might mean that the WAL write
      // batch references to the column family that was dropped after the
      // insert. We don't want to fail the whole write batch in that case --
      // we just ignore the update.
      // That's why we set ignore missing column families to true
      status = WriteBatchInternal::InsertInto(
          &batch, column_family_memtables_.get(), true, log_number);

      MaybeIgnoreError(&status);
      if (!status.ok()) {
        // We are treating this as a failure while reading since we read valid
        // blocks that do not form coherent data
        reporter.Corruption(record.size(), status);
        continue;
      }

      const SequenceNumber last_seq = WriteBatchInternal::Sequence(&batch) +
                                      WriteBatchInternal::Count(&batch) - 1;
      if ((*max_sequence == kMaxSequenceNumber) || (last_seq > *max_sequence)) {
        *max_sequence = last_seq;
      }

      if (!read_only) {
        // we can do this because this is called before client has access to the
        // DB and there is only a single thread operating on DB
        ColumnFamilyData* cfd;

        while ((cfd = flush_scheduler_.GetNextColumnFamily()) != nullptr) {
          cfd->Unref();
          // If this asserts, it means that InsertInto failed in
          // filtering updates to already-flushed column families
          assert(cfd->GetLogNumber() <= log_number);
          auto iter = version_edits.find(cfd->GetID());
          assert(iter != version_edits.end());
          VersionEdit* edit = &iter->second;
          status = WriteLevel0TableForRecovery(job_id, cfd, cfd->mem(), edit);
          if (!status.ok()) {
            // Reflect errors immediately so that conditions like full
            // file-systems cause the DB::Open() to fail.
            return status;
          }

          cfd->CreateNewMemtable(*cfd->GetLatestMutableCFOptions(),
                                 *max_sequence);
        }
      }
    }

    if (!status.ok()) {
      if (db_options_.wal_recovery_mode ==
             WALRecoveryMode::kSkipAnyCorruptedRecords) {
        // We should ignore all errors unconditionally
        status = Status::OK();
      } else if (db_options_.wal_recovery_mode ==
                 WALRecoveryMode::kPointInTimeRecovery) {
        // We should ignore the error but not continue replaying
        status = Status::OK();
        continue_replay_log = false;

        Log(InfoLogLevel::INFO_LEVEL, db_options_.info_log,
            "Point in time recovered to log #%" PRIu64 " seq #%" PRIu64,
            log_number, *max_sequence);
      } else {
        assert(db_options_.wal_recovery_mode ==
                  WALRecoveryMode::kTolerateCorruptedTailRecords
               || db_options_.wal_recovery_mode ==
                  WALRecoveryMode::kAbsoluteConsistency);
        return status;
      }
    }

    flush_scheduler_.Clear();
    if ((*max_sequence != kMaxSequenceNumber) &&
        (versions_->LastSequence() < *max_sequence)) {
      versions_->SetLastSequence(*max_sequence);
    }
  }

  if (!read_only) {
    // no need to refcount since client still doesn't have access
    // to the DB and can not drop column families while we iterate
    auto max_log_number = log_numbers.back();
    for (auto cfd : *versions_->GetColumnFamilySet()) {
      auto iter = version_edits.find(cfd->GetID());
      assert(iter != version_edits.end());
      VersionEdit* edit = &iter->second;

      if (cfd->GetLogNumber() > max_log_number) {
        // Column family cfd has already flushed the data
        // from all logs. Memtable has to be empty because
        // we filter the updates based on log_number
        // (in WriteBatch::InsertInto)
        assert(cfd->mem()->GetFirstSequenceNumber() == 0);
        assert(edit->NumEntries() == 0);
        continue;
      }

      // flush the final memtable (if non-empty)
      if (cfd->mem()->GetFirstSequenceNumber() != 0) {
        status = WriteLevel0TableForRecovery(job_id, cfd, cfd->mem(), edit);
        if (!status.ok()) {
          // Recovery failed
          break;
        }

        cfd->CreateNewMemtable(*cfd->GetLatestMutableCFOptions(),
                               *max_sequence);
      }

      // write MANIFEST with update
      // writing log_number in the manifest means that any log file
      // with number strongly less than (log_number + 1) is already
      // recovered and should be ignored on next reincarnation.
      // Since we already recovered max_log_number, we want all logs
      // with numbers `<= max_log_number` (includes this one) to be ignored
      edit->SetLogNumber(max_log_number + 1);
      // we must mark the next log number as used, even though it's
      // not actually used. that is because VersionSet assumes
      // VersionSet::next_file_number_ always to be strictly greater than any
      // log number
      versions_->MarkFileNumberUsedDuringRecovery(max_log_number + 1);
      status = versions_->LogAndApply(
          cfd, *cfd->GetLatestMutableCFOptions(), edit, &mutex_);
      if (!status.ok()) {
        // Recovery failed
        break;
      }
    }
  }

  event_logger_.Log() << "job" << job_id << "event"
                      << "recovery_finished";

  return status;
}

Status DBImpl::WriteLevel0TableForRecovery(int job_id, ColumnFamilyData* cfd,
                                           MemTable* mem, VersionEdit* edit) {
  mutex_.AssertHeld();
  const uint64_t start_micros = env_->NowMicros();
  FileMetaData meta;
  meta.fd = FileDescriptor(versions_->NewFileNumber(), 0, 0);
  auto pending_outputs_inserted_elem =
      CaptureCurrentFileNumberInPendingOutputs();
  ReadOptions ro;
  ro.total_order_seek = true;
  Arena arena;
  Status s;
  TableProperties table_properties;
  {
    ScopedArenaIterator iter(mem->NewIterator(ro, &arena));
    Log(InfoLogLevel::DEBUG_LEVEL, db_options_.info_log,
        "[%s] [WriteLevel0TableForRecovery]"
        " Level-0 table #%" PRIu64 ": started",
        cfd->GetName().c_str(), meta.fd.GetNumber());

    bool paranoid_file_checks =
        cfd->GetLatestMutableCFOptions()->paranoid_file_checks;
    {
      mutex_.Unlock();
      TableFileCreationInfo info;
      s = BuildTable(
          dbname_, env_, *cfd->ioptions(), env_options_, cfd->table_cache(),
          iter.get(), &meta, cfd->internal_comparator(),
          cfd->int_tbl_prop_collector_factories(), snapshots_.GetAll(),
          GetCompressionFlush(*cfd->ioptions()),
          cfd->ioptions()->compression_opts, paranoid_file_checks,
          cfd->internal_stats(), Env::IO_HIGH, &info.table_properties);
      LogFlush(db_options_.info_log);
      Log(InfoLogLevel::DEBUG_LEVEL, db_options_.info_log,
          "[%s] [WriteLevel0TableForRecovery]"
          " Level-0 table #%" PRIu64 ": %" PRIu64 " bytes %s",
          cfd->GetName().c_str(), meta.fd.GetNumber(), meta.fd.GetFileSize(),
          s.ToString().c_str());

      // output to event logger
      if (s.ok()) {
        info.db_name = dbname_;
        info.cf_name = cfd->GetName();
        info.file_path = TableFileName(db_options_.db_paths,
                                       meta.fd.GetNumber(),
                                       meta.fd.GetPathId());
        info.file_size = meta.fd.GetFileSize();
        info.job_id = job_id;
        EventHelpers::LogAndNotifyTableFileCreation(
            &event_logger_, db_options_.listeners, meta.fd, info);
      }
      mutex_.Lock();
    }
  }
  ReleaseFileNumberFromPendingOutputs(pending_outputs_inserted_elem);

  // Note that if file_size is zero, the file has been deleted and
  // should not be added to the manifest.
  int level = 0;
  if (s.ok() && meta.fd.GetFileSize() > 0) {
    edit->AddFile(level, meta.fd.GetNumber(), meta.fd.GetPathId(),
                  meta.fd.GetFileSize(), meta.smallest, meta.largest,
                  meta.smallest_seqno, meta.largest_seqno,
                  meta.marked_for_compaction);
  }

  InternalStats::CompactionStats stats(1);
  stats.micros = env_->NowMicros() - start_micros;
  stats.bytes_written = meta.fd.GetFileSize();
  stats.num_output_files = 1;
  cfd->internal_stats()->AddCompactionStats(level, stats);
  cfd->internal_stats()->AddCFStats(
      InternalStats::BYTES_FLUSHED, meta.fd.GetFileSize());
  RecordTick(stats_, COMPACT_WRITE_BYTES, meta.fd.GetFileSize());
  return s;
}

Status DBImpl::FlushMemTableToOutputFile(
    ColumnFamilyData* cfd, const MutableCFOptions& mutable_cf_options,
    bool* made_progress, JobContext* job_context, LogBuffer* log_buffer) {
  mutex_.AssertHeld();
  assert(cfd->imm()->NumNotFlushed() != 0);
  assert(cfd->imm()->IsFlushPending());

  FlushJob flush_job(dbname_, cfd, db_options_, mutable_cf_options,
                     env_options_, versions_.get(), &mutex_, &shutting_down_,
                     snapshots_.GetAll(), job_context, log_buffer,
                     directories_.GetDbDir(), directories_.GetDataDir(0U),
                     GetCompressionFlush(*cfd->ioptions()), stats_,
                     &event_logger_);

  FileMetaData file_meta;

  // Within flush_job.Run, rocksdb may call event listener to notify
  // file creation and deletion.
  //
  // Note that flush_job.Run will unlock and lock the db_mutex,
  // and EventListener callback will be called when the db_mutex
  // is unlocked by the current thread.
  Status s = flush_job.Run(&file_meta);

  if (s.ok()) {
    InstallSuperVersionAndScheduleWorkWrapper(cfd, job_context,
                                              mutable_cf_options);
    if (made_progress) {
      *made_progress = 1;
    }
    VersionStorageInfo::LevelSummaryStorage tmp;
    LogToBuffer(log_buffer, "[%s] Level summary: %s\n", cfd->GetName().c_str(),
                cfd->current()->storage_info()->LevelSummary(&tmp));
  }

  if (!s.ok() && !s.IsShutdownInProgress() && db_options_.paranoid_checks &&
      bg_error_.ok()) {
    // if a bad error happened (not ShutdownInProgress) and paranoid_checks is
    // true, mark DB read-only
    bg_error_ = s;
  }
  RecordFlushIOStats();
#ifndef ROCKSDB_LITE
  if (s.ok()) {
    // may temporarily unlock and lock the mutex.
    NotifyOnFlushCompleted(cfd, &file_meta, mutable_cf_options,
                           job_context->job_id);
  }
#endif  // ROCKSDB_LITE
  return s;
}

void DBImpl::NotifyOnFlushCompleted(
    ColumnFamilyData* cfd, FileMetaData* file_meta,
    const MutableCFOptions& mutable_cf_options, int job_id) {
#ifndef ROCKSDB_LITE
  if (db_options_.listeners.size() == 0U) {
    return;
  }
  mutex_.AssertHeld();
  if (shutting_down_.load(std::memory_order_acquire)) {
    return;
  }
  bool triggered_writes_slowdown =
      (cfd->current()->storage_info()->NumLevelFiles(0) >=
       mutable_cf_options.level0_slowdown_writes_trigger);
  bool triggered_writes_stop =
      (cfd->current()->storage_info()->NumLevelFiles(0) >=
       mutable_cf_options.level0_stop_writes_trigger);
  // release lock while notifying events
  mutex_.Unlock();
  {
    FlushJobInfo info;
    info.cf_name = cfd->GetName();
    // TODO(yhchiang): make db_paths dynamic in case flush does not
    //                 go to L0 in the future.
    info.file_path = MakeTableFileName(db_options_.db_paths[0].path,
                                       file_meta->fd.GetNumber());
    info.thread_id = env_->GetThreadID();
    info.job_id = job_id;
    info.triggered_writes_slowdown = triggered_writes_slowdown;
    info.triggered_writes_stop = triggered_writes_stop;
    info.smallest_seqno = file_meta->smallest_seqno;
    info.largest_seqno = file_meta->largest_seqno;
    for (auto listener : db_options_.listeners) {
      listener->OnFlushCompleted(this, info);
    }
  }
  mutex_.Lock();
  // no need to signal bg_cv_ as it will be signaled at the end of the
  // flush process.
#endif  // ROCKSDB_LITE
}

Status DBImpl::CompactRange(const CompactRangeOptions& options,
                            ColumnFamilyHandle* column_family,
                            const Slice* begin, const Slice* end) {
  if (options.target_path_id >= db_options_.db_paths.size()) {
    return Status::InvalidArgument("Invalid target path ID");
  }

  auto cfh = reinterpret_cast<ColumnFamilyHandleImpl*>(column_family);
  auto cfd = cfh->cfd();

  Status s = FlushMemTable(cfd, FlushOptions());
  if (!s.ok()) {
    LogFlush(db_options_.info_log);
    return s;
  }

  int max_level_with_files = 0;
  {
    InstrumentedMutexLock l(&mutex_);
    Version* base = cfd->current();
    for (int level = 1; level < base->storage_info()->num_non_empty_levels();
         level++) {
      if (base->storage_info()->OverlapInLevel(level, begin, end)) {
        max_level_with_files = level;
      }
    }
  }

  int final_output_level = 0;
  if (cfd->ioptions()->compaction_style == kCompactionStyleUniversal &&
      cfd->NumberLevels() > 1) {
    // Always compact all files together.
    s = RunManualCompaction(cfd, ColumnFamilyData::kCompactAllLevels,
                            cfd->NumberLevels() - 1, options.target_path_id,
                            begin, end);
    final_output_level = cfd->NumberLevels() - 1;
  } else {
    for (int level = 0; level <= max_level_with_files; level++) {
      int output_level;
      // in case the compaction is universal or if we're compacting the
      // bottom-most level, the output level will be the same as input one.
      // level 0 can never be the bottommost level (i.e. if all files are in
      // level 0, we will compact to level 1)
      if (cfd->ioptions()->compaction_style == kCompactionStyleUniversal ||
          cfd->ioptions()->compaction_style == kCompactionStyleFIFO) {
        output_level = level;
      } else if (level == max_level_with_files && level > 0) {
        if (options.bottommost_level_compaction ==
            BottommostLevelCompaction::kSkip) {
          // Skip bottommost level compaction
          continue;
        } else if (options.bottommost_level_compaction ==
                       BottommostLevelCompaction::kIfHaveCompactionFilter &&
                   cfd->ioptions()->compaction_filter == nullptr &&
                   cfd->ioptions()->compaction_filter_factory == nullptr) {
          // Skip bottommost level compaction since we don't have a compaction
          // filter
          continue;
        }
        output_level = level;
      } else {
        output_level = level + 1;
        if (cfd->ioptions()->compaction_style == kCompactionStyleLevel &&
            cfd->ioptions()->level_compaction_dynamic_level_bytes &&
            level == 0) {
          output_level = ColumnFamilyData::kCompactToBaseLevel;
        }
      }
      s = RunManualCompaction(cfd, level, output_level, options.target_path_id,
                              begin, end);
      if (!s.ok()) {
        break;
      }
      if (output_level == ColumnFamilyData::kCompactToBaseLevel) {
        final_output_level = cfd->NumberLevels() - 1;
      } else if (output_level > final_output_level) {
        final_output_level = output_level;
      }
      TEST_SYNC_POINT("DBImpl::RunManualCompaction()::1");
      TEST_SYNC_POINT("DBImpl::RunManualCompaction()::2");
    }
  }
  if (!s.ok()) {
    LogFlush(db_options_.info_log);
    return s;
  }

  if (options.change_level) {
    s = ReFitLevel(cfd, final_output_level, options.target_level);
  }
  LogFlush(db_options_.info_log);

  {
    InstrumentedMutexLock l(&mutex_);
    // an automatic compaction that has been scheduled might have been
    // preempted by the manual compactions. Need to schedule it back.
    MaybeScheduleFlushOrCompaction();
  }

  return s;
}

Status DBImpl::CompactFiles(
    const CompactionOptions& compact_options,
    ColumnFamilyHandle* column_family,
    const std::vector<std::string>& input_file_names,
    const int output_level, const int output_path_id) {
#ifdef ROCKSDB_LITE
    // not supported in lite version
  return Status::NotSupported("Not supported in ROCKSDB LITE");
#else
  if (column_family == nullptr) {
    return Status::InvalidArgument("ColumnFamilyHandle must be non-null.");
  }

  auto cfd = reinterpret_cast<ColumnFamilyHandleImpl*>(column_family)->cfd();
  assert(cfd);

  Status s;
  JobContext job_context(0, true);
  LogBuffer log_buffer(InfoLogLevel::INFO_LEVEL,
                       db_options_.info_log.get());

  // Perform CompactFiles
  SuperVersion* sv = GetAndRefSuperVersion(cfd);
  {
    InstrumentedMutexLock l(&mutex_);

    s = CompactFilesImpl(compact_options, cfd, sv->current,
                         input_file_names, output_level,
                         output_path_id, &job_context, &log_buffer);
  }
  ReturnAndCleanupSuperVersion(cfd, sv);

  // Find and delete obsolete files
  {
    InstrumentedMutexLock l(&mutex_);
    // If !s.ok(), this means that Compaction failed. In that case, we want
    // to delete all obsolete files we might have created and we force
    // FindObsoleteFiles(). This is because job_context does not
    // catch all created files if compaction failed.
    FindObsoleteFiles(&job_context, !s.ok());
  }  // release the mutex

  // delete unnecessary files if any, this is done outside the mutex
  if (job_context.HaveSomethingToDelete() || !log_buffer.IsEmpty()) {
    // Have to flush the info logs before bg_compaction_scheduled_--
    // because if bg_flush_scheduled_ becomes 0 and the lock is
    // released, the deconstructor of DB can kick in and destroy all the
    // states of DB so info_log might not be available after that point.
    // It also applies to access other states that DB owns.
    log_buffer.FlushBufferToLog();
    if (job_context.HaveSomethingToDelete()) {
      // no mutex is locked here.  No need to Unlock() and Lock() here.
      PurgeObsoleteFiles(job_context);
    }
    job_context.Clean();
  }

  return s;
#endif  // ROCKSDB_LITE
}

#ifndef ROCKSDB_LITE
Status DBImpl::CompactFilesImpl(
    const CompactionOptions& compact_options, ColumnFamilyData* cfd,
    Version* version, const std::vector<std::string>& input_file_names,
    const int output_level, int output_path_id, JobContext* job_context,
    LogBuffer* log_buffer) {
  mutex_.AssertHeld();

  if (shutting_down_.load(std::memory_order_acquire)) {
    return Status::ShutdownInProgress();
  }

  std::unordered_set<uint64_t> input_set;
  for (auto file_name : input_file_names) {
    input_set.insert(TableFileNameToNumber(file_name));
  }

  ColumnFamilyMetaData cf_meta;
  // TODO(yhchiang): can directly use version here if none of the
  // following functions call is pluggable to external developers.
  version->GetColumnFamilyMetaData(&cf_meta);

  if (output_path_id < 0) {
    if (db_options_.db_paths.size() == 1U) {
      output_path_id = 0;
    } else {
      return Status::NotSupported(
          "Automatic output path selection is not "
          "yet supported in CompactFiles()");
    }
  }

  Status s = cfd->compaction_picker()->SanitizeCompactionInputFiles(
      &input_set, cf_meta, output_level);
  if (!s.ok()) {
    return s;
  }

  std::vector<CompactionInputFiles> input_files;
  s = cfd->compaction_picker()->GetCompactionInputsFromFileNumbers(
      &input_files, &input_set, version->storage_info(), compact_options);
  if (!s.ok()) {
    return s;
  }

  for (auto inputs : input_files) {
    if (cfd->compaction_picker()->FilesInCompaction(inputs.files)) {
      return Status::Aborted(
          "Some of the necessary compaction input "
          "files are already being compacted");
    }
  }

  // At this point, CompactFiles will be run.
  bg_compaction_scheduled_++;

  unique_ptr<Compaction> c;
  assert(cfd->compaction_picker());
  c.reset(cfd->compaction_picker()->FormCompaction(
      compact_options, input_files, output_level, version->storage_info(),
      *cfd->GetLatestMutableCFOptions(), output_path_id));
  assert(c);
  c->SetInputVersion(version);
  // deletion compaction currently not allowed in CompactFiles.
  assert(!c->deletion_compaction());

  assert(is_snapshot_supported_ || snapshots_.empty());
  CompactionJob compaction_job(
      job_context->job_id, c.get(), db_options_, env_options_, versions_.get(),
      &shutting_down_, log_buffer, directories_.GetDbDir(),
      directories_.GetDataDir(c->output_path_id()), stats_, snapshots_.GetAll(),
      table_cache_, &event_logger_,
      c->mutable_cf_options()->paranoid_file_checks,
      c->mutable_cf_options()->compaction_measure_io_stats, dbname_,
      nullptr);  // Here we pass a nullptr for CompactionJobStats because
                 // CompactFiles does not trigger OnCompactionCompleted(),
                 // which is the only place where CompactionJobStats is
                 // returned.  The idea of not triggering OnCompationCompleted()
                 // is that CompactFiles runs in the caller thread, so the user
                 // should always know when it completes.  As a result, it makes
                 // less sense to notify the users something they should already
                 // know.
                 //
                 // In the future, if we would like to add CompactionJobStats
                 // support for CompactFiles, we should have CompactFiles API
                 // pass a pointer of CompactionJobStats as the out-value
                 // instead of using EventListener.
  compaction_job.Prepare();

  mutex_.Unlock();
  compaction_job.Run();
  mutex_.Lock();

  Status status = compaction_job.Install(*c->mutable_cf_options(), &mutex_);
  if (status.ok()) {
    InstallSuperVersionAndScheduleWorkWrapper(
        c->column_family_data(), job_context, *c->mutable_cf_options());
  }
  c->ReleaseCompactionFiles(s);
  c.reset();

  if (status.ok()) {
    // Done
  } else if (status.IsShutdownInProgress()) {
    // Ignore compaction errors found during shutting down
  } else {
    Log(InfoLogLevel::WARN_LEVEL, db_options_.info_log,
        "[%s] [JOB %d] Compaction error: %s",
        c->column_family_data()->GetName().c_str(), job_context->job_id,
        status.ToString().c_str());
    if (db_options_.paranoid_checks && bg_error_.ok()) {
      bg_error_ = status;
    }
  }

  bg_compaction_scheduled_--;
  if (bg_compaction_scheduled_ == 0) {
    bg_cv_.SignalAll();
  }

  return status;
}
#endif  // ROCKSDB_LITE

void DBImpl::NotifyOnCompactionCompleted(
    ColumnFamilyData* cfd, Compaction *c, const Status &st,
    const CompactionJobStats& compaction_job_stats,
    const int job_id) {
#ifndef ROCKSDB_LITE
  if (db_options_.listeners.size() == 0U) {
    return;
  }
  mutex_.AssertHeld();
  if (shutting_down_.load(std::memory_order_acquire)) {
    return;
  }
  // release lock while notifying events
  mutex_.Unlock();
  {
    CompactionJobInfo info;
    info.cf_name = cfd->GetName();
    info.status = st;
    info.thread_id = env_->GetThreadID();
    info.job_id = job_id;
    info.base_input_level = c->start_level();
    info.output_level = c->output_level();
    info.stats = compaction_job_stats;
    for (size_t i = 0; i < c->num_input_levels(); ++i) {
      for (const auto fmd : *c->inputs(i)) {
        info.input_files.push_back(
            TableFileName(db_options_.db_paths,
                          fmd->fd.GetNumber(),
                          fmd->fd.GetPathId()));
      }
    }
    for (const auto newf : c->edit()->GetNewFiles()) {
      info.output_files.push_back(
          TableFileName(db_options_.db_paths,
                        newf.second.fd.GetNumber(),
                        newf.second.fd.GetPathId()));
    }
    for (auto listener : db_options_.listeners) {
      listener->OnCompactionCompleted(this, info);
    }
  }
  mutex_.Lock();
  // no need to signal bg_cv_ as it will be signaled at the end of the
  // flush process.
#endif  // ROCKSDB_LITE
}

Status DBImpl::SetOptions(ColumnFamilyHandle* column_family,
    const std::unordered_map<std::string, std::string>& options_map) {
#ifdef ROCKSDB_LITE
  return Status::NotSupported("Not supported in ROCKSDB LITE");
#else
  auto* cfd = reinterpret_cast<ColumnFamilyHandleImpl*>(column_family)->cfd();
  if (options_map.empty()) {
    Log(InfoLogLevel::WARN_LEVEL,
        db_options_.info_log, "SetOptions() on column family [%s], empty input",
        cfd->GetName().c_str());
    return Status::InvalidArgument("empty input");
  }

  MutableCFOptions new_options;
  Status s;
  {
    InstrumentedMutexLock l(&mutex_);
    s = cfd->SetOptions(options_map);
    if (s.ok()) {
      new_options = *cfd->GetLatestMutableCFOptions();
    }
  }

  Log(InfoLogLevel::INFO_LEVEL, db_options_.info_log,
      "SetOptions() on column family [%s], inputs:",
      cfd->GetName().c_str());
  for (const auto& o : options_map) {
    Log(InfoLogLevel::INFO_LEVEL, db_options_.info_log,
        "%s: %s\n", o.first.c_str(), o.second.c_str());
  }
  if (s.ok()) {
    Log(InfoLogLevel::INFO_LEVEL,
        db_options_.info_log, "[%s] SetOptions succeeded",
        cfd->GetName().c_str());
    new_options.Dump(db_options_.info_log.get());
  } else {
    Log(InfoLogLevel::WARN_LEVEL, db_options_.info_log,
        "[%s] SetOptions failed", cfd->GetName().c_str());
  }
  LogFlush(db_options_.info_log);
  return s;
#endif  // ROCKSDB_LITE
}

// return the same level if it cannot be moved
int DBImpl::FindMinimumEmptyLevelFitting(ColumnFamilyData* cfd,
    const MutableCFOptions& mutable_cf_options, int level) {
  mutex_.AssertHeld();
  const auto* vstorage = cfd->current()->storage_info();
  int minimum_level = level;
  for (int i = level - 1; i > 0; --i) {
    // stop if level i is not empty
    if (vstorage->NumLevelFiles(i) > 0) break;
    // stop if level i is too small (cannot fit the level files)
    if (vstorage->MaxBytesForLevel(i) < vstorage->NumLevelBytes(level)) {
      break;
    }

    minimum_level = i;
  }
  return minimum_level;
}

Status DBImpl::ReFitLevel(ColumnFamilyData* cfd, int level, int target_level) {
  assert(level < cfd->NumberLevels());
  if (target_level >= cfd->NumberLevels()) {
    return Status::InvalidArgument("Target level exceeds number of levels");
  }

  SuperVersion* superversion_to_free = nullptr;
  SuperVersion* new_superversion = new SuperVersion();

  InstrumentedMutexLock guard_lock(&mutex_);

  // only allow one thread refitting
  if (refitting_level_) {
    Log(InfoLogLevel::INFO_LEVEL, db_options_.info_log,
        "[ReFitLevel] another thread is refitting");
    delete new_superversion;
    return Status::NotSupported("another thread is refitting");
  }
  refitting_level_ = true;

  // wait for all background threads to stop
  bg_work_gate_closed_ = true;
  while (bg_compaction_scheduled_ > 0 || bg_flush_scheduled_) {
    Log(InfoLogLevel::INFO_LEVEL, db_options_.info_log,
        "[RefitLevel] waiting for background threads to stop: %d %d",
        bg_compaction_scheduled_, bg_flush_scheduled_);
    bg_cv_.Wait();
  }

  const MutableCFOptions mutable_cf_options =
    *cfd->GetLatestMutableCFOptions();
  // move to a smaller level
  int to_level = target_level;
  if (target_level < 0) {
    to_level = FindMinimumEmptyLevelFitting(cfd, mutable_cf_options, level);
  }

  Status status;
  auto* vstorage = cfd->current()->storage_info();
  if (to_level > level) {
    if (level == 0) {
      delete new_superversion;
      return Status::NotSupported(
          "Cannot change from level 0 to other levels.");
    }
    // Check levels are empty for a trivial move
    for (int l = level + 1; l <= to_level; l++) {
      if (vstorage->NumLevelFiles(l) > 0) {
        delete new_superversion;
        return Status::NotSupported(
            "Levels between source and target are not empty for a move.");
      }
    }
  }
  if (to_level != level) {
    Log(InfoLogLevel::DEBUG_LEVEL, db_options_.info_log,
        "[%s] Before refitting:\n%s",
        cfd->GetName().c_str(), cfd->current()->DebugString().data());

    VersionEdit edit;
    edit.SetColumnFamily(cfd->GetID());
    for (const auto& f : vstorage->LevelFiles(level)) {
      edit.DeleteFile(level, f->fd.GetNumber());
      edit.AddFile(to_level, f->fd.GetNumber(), f->fd.GetPathId(),
                   f->fd.GetFileSize(), f->smallest, f->largest,
                   f->smallest_seqno, f->largest_seqno,
                   f->marked_for_compaction);
    }
    Log(InfoLogLevel::DEBUG_LEVEL, db_options_.info_log,
        "[%s] Apply version edit:\n%s",
        cfd->GetName().c_str(), edit.DebugString().data());

    status = versions_->LogAndApply(cfd, mutable_cf_options, &edit, &mutex_,
                                    directories_.GetDbDir());
    superversion_to_free = InstallSuperVersionAndScheduleWork(
        cfd, new_superversion, mutable_cf_options);
    new_superversion = nullptr;

    Log(InfoLogLevel::DEBUG_LEVEL, db_options_.info_log,
        "[%s] LogAndApply: %s\n", cfd->GetName().c_str(),
        status.ToString().data());

    if (status.ok()) {
      Log(InfoLogLevel::DEBUG_LEVEL, db_options_.info_log,
          "[%s] After refitting:\n%s",
          cfd->GetName().c_str(), cfd->current()->DebugString().data());
    }
  }

  refitting_level_ = false;
  bg_work_gate_closed_ = false;

  delete superversion_to_free;
  delete new_superversion;
  return status;
}

int DBImpl::NumberLevels(ColumnFamilyHandle* column_family) {
  auto cfh = reinterpret_cast<ColumnFamilyHandleImpl*>(column_family);
  return cfh->cfd()->NumberLevels();
}

int DBImpl::MaxMemCompactionLevel(ColumnFamilyHandle* column_family) {
  return 0;
}

int DBImpl::Level0StopWriteTrigger(ColumnFamilyHandle* column_family) {
  auto cfh = reinterpret_cast<ColumnFamilyHandleImpl*>(column_family);
  InstrumentedMutexLock l(&mutex_);
  return cfh->cfd()->GetSuperVersion()->
      mutable_cf_options.level0_stop_writes_trigger;
}

Status DBImpl::Flush(const FlushOptions& flush_options,
                     ColumnFamilyHandle* column_family) {
  auto cfh = reinterpret_cast<ColumnFamilyHandleImpl*>(column_family);
  return FlushMemTable(cfh->cfd(), flush_options);
}

Status DBImpl::SyncWAL() {
  autovector<log::Writer*, 1> logs_to_sync;
  bool need_log_dir_sync;
  uint64_t current_log_number;

  {
    InstrumentedMutexLock l(&mutex_);
    assert(!logs_.empty());

    // This SyncWAL() call only cares about logs up to this number.
    current_log_number = logfile_number_;

    while (logs_.front().number <= current_log_number &&
           logs_.front().getting_synced) {
      log_sync_cv_.Wait();
    }
    // First check that logs are safe to sync in background.
    for (auto it = logs_.begin();
         it != logs_.end() && it->number <= current_log_number; ++it) {
      if (!it->writer->file()->writable_file()->IsSyncThreadSafe()) {
        return Status::NotSupported(
          "SyncWAL() is not supported for this implementation of WAL file",
          db_options_.allow_mmap_writes
            ? "try setting Options::allow_mmap_writes to false"
            : Slice());
      }
    }
    for (auto it = logs_.begin();
         it != logs_.end() && it->number <= current_log_number; ++it) {
      auto& log = *it;
      assert(!log.getting_synced);
      log.getting_synced = true;
      logs_to_sync.push_back(log.writer);
    }

    need_log_dir_sync = !log_dir_synced_;
  }

  Status status;
  for (log::Writer* log : logs_to_sync) {
    status = log->file()->SyncWithoutFlush(db_options_.use_fsync);
    if (!status.ok()) {
      break;
    }
  }
  if (status.ok() && need_log_dir_sync) {
    status = directories_.GetWalDir()->Fsync();
  }

  {
    InstrumentedMutexLock l(&mutex_);
    MarkLogsSynced(current_log_number, need_log_dir_sync, status);
  }

  return status;
}

void DBImpl::MarkLogsSynced(
    uint64_t up_to, bool synced_dir, const Status& status) {
  mutex_.AssertHeld();
  if (synced_dir &&
      logfile_number_ == up_to &&
      status.ok()) {
    log_dir_synced_ = true;
  }
  for (auto it = logs_.begin(); it != logs_.end() && it->number <= up_to;) {
    auto& log = *it;
    assert(log.getting_synced);
    if (status.ok() && logs_.size() > 1) {
      logs_to_free_.push_back(log.ReleaseWriter());
      it = logs_.erase(it);
    } else {
      log.getting_synced = false;
      ++it;
    }
  }
  assert(logs_.empty() || (logs_.size() == 1 && !logs_[0].getting_synced));
  log_sync_cv_.SignalAll();
}

SequenceNumber DBImpl::GetLatestSequenceNumber() const {
  return versions_->LastSequence();
}

Status DBImpl::RunManualCompaction(ColumnFamilyData* cfd, int input_level,
                                   int output_level, uint32_t output_path_id,
                                   const Slice* begin, const Slice* end,
                                   bool disallow_trivial_move) {
  assert(input_level == ColumnFamilyData::kCompactAllLevels ||
         input_level >= 0);

  InternalKey begin_storage, end_storage;

  ManualCompaction manual;
  manual.cfd = cfd;
  manual.input_level = input_level;
  manual.output_level = output_level;
  manual.output_path_id = output_path_id;
  manual.done = false;
  manual.in_progress = false;
  manual.disallow_trivial_move = disallow_trivial_move;
  // For universal compaction, we enforce every manual compaction to compact
  // all files.
  if (begin == nullptr ||
      cfd->ioptions()->compaction_style == kCompactionStyleUniversal ||
      cfd->ioptions()->compaction_style == kCompactionStyleFIFO) {
    manual.begin = nullptr;
  } else {
    begin_storage.SetMaxPossibleForUserKey(*begin);
    manual.begin = &begin_storage;
  }
  if (end == nullptr ||
      cfd->ioptions()->compaction_style == kCompactionStyleUniversal ||
      cfd->ioptions()->compaction_style == kCompactionStyleFIFO) {
    manual.end = nullptr;
  } else {
    end_storage.SetMinPossibleForUserKey(*end);
    manual.end = &end_storage;
  }

  InstrumentedMutexLock l(&mutex_);

  // When a manual compaction arrives, temporarily disable scheduling of
  // non-manual compactions and wait until the number of scheduled compaction
  // jobs drops to zero. This is needed to ensure that this manual compaction
  // can compact any range of keys/files.
  //
  // bg_manual_only_ is non-zero when at least one thread is inside
  // RunManualCompaction(), i.e. during that time no other compaction will
  // get scheduled (see MaybeScheduleFlushOrCompaction).
  //
  // Note that the following loop doesn't stop more that one thread calling
  // RunManualCompaction() from getting to the second while loop below.
  // However, only one of them will actually schedule compaction, while
  // others will wait on a condition variable until it completes.

  ++bg_manual_only_;
  while (bg_compaction_scheduled_ > 0) {
    Log(InfoLogLevel::INFO_LEVEL, db_options_.info_log,
        "[%s] Manual compaction waiting for all other scheduled background "
        "compactions to finish",
        cfd->GetName().c_str());
    bg_cv_.Wait();
  }

  Log(InfoLogLevel::INFO_LEVEL, db_options_.info_log,
      "[%s] Manual compaction starting",
      cfd->GetName().c_str());

  // We don't check bg_error_ here, because if we get the error in compaction,
  // the compaction will set manual.status to bg_error_ and set manual.done to
  // true.
  while (!manual.done) {
    assert(bg_manual_only_ > 0);
    if (manual_compaction_ != nullptr) {
      // Running either this or some other manual compaction
      bg_cv_.Wait();
    } else {
      manual_compaction_ = &manual;
      bg_compaction_scheduled_++;
      env_->Schedule(&DBImpl::BGWorkCompaction, this, Env::Priority::LOW, this);
    }
  }

  assert(!manual.in_progress);
  assert(bg_manual_only_ > 0);
  --bg_manual_only_;
  return manual.status;
}

Status DBImpl::FlushMemTable(ColumnFamilyData* cfd,
                             const FlushOptions& flush_options) {
  Status s;
  {
    WriteContext context;
    InstrumentedMutexLock guard_lock(&mutex_);

    if (cfd->imm()->NumNotFlushed() == 0 && cfd->mem()->IsEmpty()) {
      // Nothing to flush
      return Status::OK();
    }

    WriteThread::Writer w;
    write_thread_.EnterUnbatched(&w, &mutex_);

    // SwitchMemtable() will release and reacquire mutex
    // during execution
    s = SwitchMemtable(cfd, &context);
    write_thread_.ExitUnbatched(&w);

    cfd->imm()->FlushRequested();

    // schedule flush
    SchedulePendingFlush(cfd);
    MaybeScheduleFlushOrCompaction();
  }

  if (s.ok() && flush_options.wait) {
    // Wait until the compaction completes
    s = WaitForFlushMemTable(cfd);
  }
  return s;
}

Status DBImpl::WaitForFlushMemTable(ColumnFamilyData* cfd) {
  Status s;
  // Wait until the compaction completes
  InstrumentedMutexLock l(&mutex_);
  while (cfd->imm()->NumNotFlushed() > 0 && bg_error_.ok()) {
    if (shutting_down_.load(std::memory_order_acquire)) {
      return Status::ShutdownInProgress();
    }
    bg_cv_.Wait();
  }
  if (!bg_error_.ok()) {
    s = bg_error_;
  }
  return s;
}

void DBImpl::MaybeScheduleFlushOrCompaction() {
  mutex_.AssertHeld();
  if (!opened_successfully_) {
    // Compaction may introduce data race to DB open
    return;
  }
  if (bg_work_gate_closed_) {
    // gate closed for background work
    return;
  } else if (shutting_down_.load(std::memory_order_acquire)) {
    // DB is being deleted; no more background compactions
    return;
  }

  while (unscheduled_flushes_ > 0 &&
         bg_flush_scheduled_ < db_options_.max_background_flushes) {
    unscheduled_flushes_--;
    bg_flush_scheduled_++;
    env_->Schedule(&DBImpl::BGWorkFlush, this, Env::Priority::HIGH, this);
  }

  // special case -- if max_background_flushes == 0, then schedule flush on a
  // compaction thread
  if (db_options_.max_background_flushes == 0) {
    while (unscheduled_flushes_ > 0 &&
           bg_flush_scheduled_ + bg_compaction_scheduled_ <
               db_options_.max_background_compactions) {
      unscheduled_flushes_--;
      bg_flush_scheduled_++;
      env_->Schedule(&DBImpl::BGWorkFlush, this, Env::Priority::LOW, this);
    }
  }

  if (bg_manual_only_) {
    // only manual compactions are allowed to run. don't schedule automatic
    // compactions
    return;
  }

  while (bg_compaction_scheduled_ < db_options_.max_background_compactions &&
         unscheduled_compactions_ > 0) {
    bg_compaction_scheduled_++;
    unscheduled_compactions_--;
    env_->Schedule(&DBImpl::BGWorkCompaction, this, Env::Priority::LOW, this);
  }
}

void DBImpl::AddToCompactionQueue(ColumnFamilyData* cfd) {
  assert(!cfd->pending_compaction());
  cfd->Ref();
  compaction_queue_.push_back(cfd);
  cfd->set_pending_compaction(true);
}

ColumnFamilyData* DBImpl::PopFirstFromCompactionQueue() {
  assert(!compaction_queue_.empty());
  auto cfd = *compaction_queue_.begin();
  compaction_queue_.pop_front();
  assert(cfd->pending_compaction());
  cfd->set_pending_compaction(false);
  return cfd;
}

void DBImpl::AddToFlushQueue(ColumnFamilyData* cfd) {
  assert(!cfd->pending_flush());
  cfd->Ref();
  flush_queue_.push_back(cfd);
  cfd->set_pending_flush(true);
}

ColumnFamilyData* DBImpl::PopFirstFromFlushQueue() {
  assert(!flush_queue_.empty());
  auto cfd = *flush_queue_.begin();
  flush_queue_.pop_front();
  assert(cfd->pending_flush());
  cfd->set_pending_flush(false);
  return cfd;
}

void DBImpl::SchedulePendingFlush(ColumnFamilyData* cfd) {
  if (!cfd->pending_flush() && cfd->imm()->IsFlushPending()) {
    AddToFlushQueue(cfd);
    ++unscheduled_flushes_;
  }
}

void DBImpl::SchedulePendingCompaction(ColumnFamilyData* cfd) {
  if (!cfd->pending_compaction() && cfd->NeedsCompaction()) {
    AddToCompactionQueue(cfd);
    ++unscheduled_compactions_;
  }
}

void DBImpl::RecordFlushIOStats() {
  RecordTick(stats_, FLUSH_WRITE_BYTES, IOSTATS(bytes_written));
  IOSTATS_RESET(bytes_written);
}

void DBImpl::BGWorkFlush(void* db) {
  IOSTATS_SET_THREAD_POOL_ID(Env::Priority::HIGH);
  TEST_SYNC_POINT("DBImpl::BGWorkFlush");
  reinterpret_cast<DBImpl*>(db)->BackgroundCallFlush();
  TEST_SYNC_POINT("DBImpl::BGWorkFlush:done");
}

void DBImpl::BGWorkCompaction(void* db) {
  IOSTATS_SET_THREAD_POOL_ID(Env::Priority::LOW);
  TEST_SYNC_POINT("DBImpl::BGWorkCompaction");
  reinterpret_cast<DBImpl*>(db)->BackgroundCallCompaction();
}

Status DBImpl::BackgroundFlush(bool* made_progress, JobContext* job_context,
                               LogBuffer* log_buffer) {
  mutex_.AssertHeld();

  Status status = bg_error_;
  if (status.ok() && shutting_down_.load(std::memory_order_acquire)) {
    status = Status::ShutdownInProgress();
  }

  if (!status.ok()) {
    return status;
  }

  ColumnFamilyData* cfd = nullptr;
  while (!flush_queue_.empty()) {
    // This cfd is already referenced
    auto first_cfd = PopFirstFromFlushQueue();

    if (first_cfd->IsDropped() || !first_cfd->imm()->IsFlushPending()) {
      // can't flush this CF, try next one
      if (first_cfd->Unref()) {
        delete first_cfd;
      }
      continue;
    }

    // found a flush!
    cfd = first_cfd;
    break;
  }

  if (cfd != nullptr) {
    const MutableCFOptions mutable_cf_options =
        *cfd->GetLatestMutableCFOptions();
    LogToBuffer(
        log_buffer,
        "Calling FlushMemTableToOutputFile with column "
        "family [%s], flush slots available %d, compaction slots available %d",
        cfd->GetName().c_str(),
        db_options_.max_background_flushes - bg_flush_scheduled_,
        db_options_.max_background_compactions - bg_compaction_scheduled_);
    status = FlushMemTableToOutputFile(cfd, mutable_cf_options, made_progress,
                                       job_context, log_buffer);
    if (cfd->Unref()) {
      delete cfd;
    }
  }
  return status;
}

void DBImpl::BackgroundCallFlush() {
  bool made_progress = false;
  JobContext job_context(next_job_id_.fetch_add(1), true);
  assert(bg_flush_scheduled_);

  LogBuffer log_buffer(InfoLogLevel::INFO_LEVEL, db_options_.info_log.get());
  {
    InstrumentedMutexLock l(&mutex_);

    auto pending_outputs_inserted_elem =
        CaptureCurrentFileNumberInPendingOutputs();

    Status s = BackgroundFlush(&made_progress, &job_context, &log_buffer);
    if (!s.ok() && !s.IsShutdownInProgress()) {
      // Wait a little bit before retrying background flush in
      // case this is an environmental problem and we do not want to
      // chew up resources for failed flushes for the duration of
      // the problem.
      uint64_t error_cnt =
        default_cf_internal_stats_->BumpAndGetBackgroundErrorCount();
      bg_cv_.SignalAll();  // In case a waiter can proceed despite the error
      mutex_.Unlock();
      Log(InfoLogLevel::ERROR_LEVEL, db_options_.info_log,
          "Waiting after background flush error: %s"
          "Accumulated background error counts: %" PRIu64,
          s.ToString().c_str(), error_cnt);
      log_buffer.FlushBufferToLog();
      LogFlush(db_options_.info_log);
      env_->SleepForMicroseconds(1000000);
      mutex_.Lock();
    }

    ReleaseFileNumberFromPendingOutputs(pending_outputs_inserted_elem);

    // If flush failed, we want to delete all temporary files that we might have
    // created. Thus, we force full scan in FindObsoleteFiles()
    FindObsoleteFiles(&job_context, !s.ok() && !s.IsShutdownInProgress());
    // delete unnecessary files if any, this is done outside the mutex
    if (job_context.HaveSomethingToDelete() || !log_buffer.IsEmpty()) {
      mutex_.Unlock();
      // Have to flush the info logs before bg_flush_scheduled_--
      // because if bg_flush_scheduled_ becomes 0 and the lock is
      // released, the deconstructor of DB can kick in and destroy all the
      // states of DB so info_log might not be available after that point.
      // It also applies to access other states that DB owns.
      log_buffer.FlushBufferToLog();
      if (job_context.HaveSomethingToDelete()) {
        PurgeObsoleteFiles(job_context);
      }
      job_context.Clean();
      mutex_.Lock();
    }

    bg_flush_scheduled_--;
    // See if there's more work to be done
    MaybeScheduleFlushOrCompaction();
    RecordFlushIOStats();
    bg_cv_.SignalAll();
    // IMPORTANT: there should be no code after calling SignalAll. This call may
    // signal the DB destructor that it's OK to proceed with destruction. In
    // that case, all DB variables will be dealloacated and referencing them
    // will cause trouble.
  }
}

void DBImpl::BackgroundCallCompaction() {
  bool made_progress = false;
  JobContext job_context(next_job_id_.fetch_add(1), true);

  MaybeDumpStats();
  LogBuffer log_buffer(InfoLogLevel::INFO_LEVEL, db_options_.info_log.get());
  {
    InstrumentedMutexLock l(&mutex_);

    auto pending_outputs_inserted_elem =
        CaptureCurrentFileNumberInPendingOutputs();

    assert(bg_compaction_scheduled_);
    Status s = BackgroundCompaction(&made_progress, &job_context, &log_buffer);
    if (!s.ok() && !s.IsShutdownInProgress()) {
      // Wait a little bit before retrying background compaction in
      // case this is an environmental problem and we do not want to
      // chew up resources for failed compactions for the duration of
      // the problem.
      uint64_t error_cnt =
          default_cf_internal_stats_->BumpAndGetBackgroundErrorCount();
      bg_cv_.SignalAll();  // In case a waiter can proceed despite the error
      mutex_.Unlock();
      log_buffer.FlushBufferToLog();
      Log(InfoLogLevel::ERROR_LEVEL, db_options_.info_log,
          "Waiting after background compaction error: %s, "
          "Accumulated background error counts: %" PRIu64,
          s.ToString().c_str(), error_cnt);
      LogFlush(db_options_.info_log);
      env_->SleepForMicroseconds(1000000);
      mutex_.Lock();
    }

    ReleaseFileNumberFromPendingOutputs(pending_outputs_inserted_elem);

    // If compaction failed, we want to delete all temporary files that we might
    // have created (they might not be all recorded in job_context in case of a
    // failure). Thus, we force full scan in FindObsoleteFiles()
    FindObsoleteFiles(&job_context, !s.ok() && !s.IsShutdownInProgress());

    // delete unnecessary files if any, this is done outside the mutex
    if (job_context.HaveSomethingToDelete() || !log_buffer.IsEmpty()) {
      mutex_.Unlock();
      // Have to flush the info logs before bg_compaction_scheduled_--
      // because if bg_flush_scheduled_ becomes 0 and the lock is
      // released, the deconstructor of DB can kick in and destroy all the
      // states of DB so info_log might not be available after that point.
      // It also applies to access other states that DB owns.
      log_buffer.FlushBufferToLog();
      if (job_context.HaveSomethingToDelete()) {
        PurgeObsoleteFiles(job_context);
      }
      job_context.Clean();
      mutex_.Lock();
    }

    bg_compaction_scheduled_--;

    versions_->GetColumnFamilySet()->FreeDeadColumnFamilies();

    // See if there's more work to be done
    MaybeScheduleFlushOrCompaction();
    if (made_progress || bg_compaction_scheduled_ == 0 || bg_manual_only_ > 0) {
      // signal if
      // * made_progress -- need to wakeup DelayWrite
      // * bg_compaction_scheduled_ == 0 -- need to wakeup ~DBImpl
      // * bg_manual_only_ > 0 -- need to wakeup RunManualCompaction
      // If none of this is true, there is no need to signal since nobody is
      // waiting for it
      bg_cv_.SignalAll();
    }
    // IMPORTANT: there should be no code after calling SignalAll. This call may
    // signal the DB destructor that it's OK to proceed with destruction. In
    // that case, all DB variables will be dealloacated and referencing them
    // will cause trouble.
  }
}

Status DBImpl::BackgroundCompaction(bool* made_progress,
                                    JobContext* job_context,
                                    LogBuffer* log_buffer) {
  *made_progress = false;
  mutex_.AssertHeld();

  bool is_manual = (manual_compaction_ != nullptr) &&
                   (manual_compaction_->in_progress == false);
  bool trivial_move_disallowed = is_manual &&
                                 manual_compaction_->disallow_trivial_move;

  CompactionJobStats compaction_job_stats;
  Status status = bg_error_;
  if (status.ok() && shutting_down_.load(std::memory_order_acquire)) {
    status = Status::ShutdownInProgress();
  }

  if (!status.ok()) {
    if (is_manual) {
      manual_compaction_->status = status;
      manual_compaction_->done = true;
      manual_compaction_->in_progress = false;
      manual_compaction_ = nullptr;
    }
    return status;
  }

  if (is_manual) {
    // another thread cannot pick up the same work
    manual_compaction_->in_progress = true;
  } else if (manual_compaction_ != nullptr) {
    // there should be no automatic compactions running when manual compaction
    // is running
    return Status::OK();
  }

  unique_ptr<Compaction> c;
  InternalKey manual_end_storage;
  InternalKey* manual_end = &manual_end_storage;
  if (is_manual) {
    ManualCompaction* m = manual_compaction_;
    assert(m->in_progress);
    c.reset(m->cfd->CompactRange(
          *m->cfd->GetLatestMutableCFOptions(), m->input_level, m->output_level,
          m->output_path_id, m->begin, m->end, &manual_end));
    if (!c) {
      m->done = true;
      LogToBuffer(log_buffer,
                  "[%s] Manual compaction from level-%d from %s .. "
                  "%s; nothing to do\n",
                  m->cfd->GetName().c_str(), m->input_level,
                  (m->begin ? m->begin->DebugString().c_str() : "(begin)"),
                  (m->end ? m->end->DebugString().c_str() : "(end)"));
    } else {
      LogToBuffer(log_buffer,
                  "[%s] Manual compaction from level-%d to level-%d from %s .. "
                  "%s; will stop at %s\n",
                  m->cfd->GetName().c_str(), m->input_level, c->output_level(),
                  (m->begin ? m->begin->DebugString().c_str() : "(begin)"),
                  (m->end ? m->end->DebugString().c_str() : "(end)"),
                  ((m->done || manual_end == nullptr)
                       ? "(end)"
                       : manual_end->DebugString().c_str()));
    }
  } else if (!compaction_queue_.empty()) {
    // cfd is referenced here
    auto cfd = PopFirstFromCompactionQueue();
    // We unreference here because the following code will take a Ref() on
    // this cfd if it is going to use it (Compaction class holds a
    // reference).
    // This will all happen under a mutex so we don't have to be afraid of
    // somebody else deleting it.
    if (cfd->Unref()) {
      delete cfd;
      // This was the last reference of the column family, so no need to
      // compact.
      return Status::OK();
    }

    // Pick up latest mutable CF Options and use it throughout the
    // compaction job
    // Compaction makes a copy of the latest MutableCFOptions. It should be used
    // throughout the compaction procedure to make sure consistency. It will
    // eventually be installed into SuperVersion
    auto* mutable_cf_options = cfd->GetLatestMutableCFOptions();
    if (!mutable_cf_options->disable_auto_compactions && !cfd->IsDropped()) {
      // NOTE: try to avoid unnecessary copy of MutableCFOptions if
      // compaction is not necessary. Need to make sure mutex is held
      // until we make a copy in the following code
      c.reset(cfd->PickCompaction(*mutable_cf_options, log_buffer));
      if (c != nullptr) {
        // update statistics
        MeasureTime(stats_, NUM_FILES_IN_SINGLE_COMPACTION,
                    c->inputs(0)->size());
        // There are three things that can change compaction score:
        // 1) When flush or compaction finish. This case is covered by
        // InstallSuperVersionAndScheduleWork
        // 2) When MutableCFOptions changes. This case is also covered by
        // InstallSuperVersionAndScheduleWork, because this is when the new
        // options take effect.
        // 3) When we Pick a new compaction, we "remove" those files being
        // compacted from the calculation, which then influences compaction
        // score. Here we check if we need the new compaction even without the
        // files that are currently being compacted. If we need another
        // compaction, we might be able to execute it in parallel, so we add it
        // to the queue and schedule a new thread.
        if (cfd->NeedsCompaction()) {
          // Yes, we need more compactions!
          AddToCompactionQueue(cfd);
          ++unscheduled_compactions_;
          MaybeScheduleFlushOrCompaction();
        }
      }
    }
  }

  if (!c) {
    // Nothing to do
    LogToBuffer(log_buffer, "Compaction nothing to do");
  } else if (c->deletion_compaction()) {
    // TODO(icanadi) Do we want to honor snapshots here? i.e. not delete old
    // file if there is alive snapshot pointing to it
    assert(c->num_input_files(1) == 0);
    assert(c->level() == 0);
    assert(c->column_family_data()->ioptions()->compaction_style ==
           kCompactionStyleFIFO);

    compaction_job_stats.num_input_files = c->num_input_files(0);

    for (const auto& f : *c->inputs(0)) {
      c->edit()->DeleteFile(c->level(), f->fd.GetNumber());
    }
    status = versions_->LogAndApply(c->column_family_data(),
                                    *c->mutable_cf_options(), c->edit(),
                                    &mutex_, directories_.GetDbDir());
    InstallSuperVersionAndScheduleWorkWrapper(
        c->column_family_data(), job_context, *c->mutable_cf_options());
    LogToBuffer(log_buffer, "[%s] Deleted %d files\n",
                c->column_family_data()->GetName().c_str(),
                c->num_input_files(0));
    *made_progress = true;
  } else if (!trivial_move_disallowed && c->IsTrivialMove()) {
    TEST_SYNC_POINT("DBImpl::BackgroundCompaction:TrivialMove");
    // Instrument for event update
    // TODO(yhchiang): add op details for showing trivial-move.
    ThreadStatusUtil::SetColumnFamily(c->column_family_data());
    ThreadStatusUtil::SetThreadOperation(ThreadStatus::OP_COMPACTION);

    compaction_job_stats.num_input_files = c->num_input_files(0);

    // Move files to next level
    int32_t moved_files = 0;
    int64_t moved_bytes = 0;
    for (unsigned int l = 0; l < c->num_input_levels(); l++) {
      if (c->level(l) == c->output_level()) {
        continue;
      }
      for (size_t i = 0; i < c->num_input_files(l); i++) {
        FileMetaData* f = c->input(l, i);
        c->edit()->DeleteFile(c->level(l), f->fd.GetNumber());
        c->edit()->AddFile(c->output_level(), f->fd.GetNumber(),
                           f->fd.GetPathId(), f->fd.GetFileSize(), f->smallest,
                           f->largest, f->smallest_seqno, f->largest_seqno,
                           f->marked_for_compaction);

        LogToBuffer(log_buffer,
                    "[%s] Moving #%" PRIu64 " to level-%d %" PRIu64 " bytes\n",
                    c->column_family_data()->GetName().c_str(),
                    f->fd.GetNumber(), c->output_level(), f->fd.GetFileSize());
        ++moved_files;
        moved_bytes += f->fd.GetFileSize();
      }
    }

    status = versions_->LogAndApply(c->column_family_data(),
                                    *c->mutable_cf_options(), c->edit(),
                                    &mutex_, directories_.GetDbDir());
    // Use latest MutableCFOptions
    InstallSuperVersionAndScheduleWorkWrapper(
        c->column_family_data(), job_context, *c->mutable_cf_options());

    VersionStorageInfo::LevelSummaryStorage tmp;
    c->column_family_data()->internal_stats()->IncBytesMoved(c->output_level(),
                                                             moved_bytes);
    {
      event_logger_.LogToBuffer(log_buffer)
          << "job" << job_context->job_id << "event"
          << "trivial_move"
          << "destination_level" << c->output_level() << "files" << moved_files
          << "total_files_size" << moved_bytes;
    }
    LogToBuffer(
        log_buffer,
        "[%s] Moved #%d files to level-%d %" PRIu64 " bytes %s: %s\n",
        c->column_family_data()->GetName().c_str(), moved_files,
        c->output_level(), moved_bytes, status.ToString().c_str(),
        c->column_family_data()->current()->storage_info()->LevelSummary(&tmp));
    *made_progress = true;

    // Clear Instrument
    ThreadStatusUtil::ResetThreadStatus();
  } else {
    int output_level  __attribute__((unused)) = c->output_level();
    TEST_SYNC_POINT_CALLBACK("DBImpl::BackgroundCompaction:NonTrivial",
                             &output_level);
    assert(is_snapshot_supported_ || snapshots_.empty());
    CompactionJob compaction_job(
        job_context->job_id, c.get(), db_options_, env_options_,
        versions_.get(), &shutting_down_, log_buffer, directories_.GetDbDir(),
        directories_.GetDataDir(c->output_path_id()), stats_,
        snapshots_.GetAll(), table_cache_, &event_logger_,
        c->mutable_cf_options()->paranoid_file_checks,
        c->mutable_cf_options()->compaction_measure_io_stats, dbname_,
        &compaction_job_stats);
    compaction_job.Prepare();

    mutex_.Unlock();
    compaction_job.Run();
    TEST_SYNC_POINT("DBImpl::BackgroundCompaction:NonTrivial:AfterRun");
    mutex_.Lock();

    status = compaction_job.Install(*c->mutable_cf_options(), &mutex_);
    if (status.ok()) {
      InstallSuperVersionAndScheduleWorkWrapper(
          c->column_family_data(), job_context, *c->mutable_cf_options());
    }
    *made_progress = true;
  }
  if (c != nullptr) {
    NotifyOnCompactionCompleted(
        c->column_family_data(), c.get(), status,
        compaction_job_stats, job_context->job_id);
    c->ReleaseCompactionFiles(status);
    *made_progress = true;
  }
  // this will unref its input_version and column_family_data
  c.reset();

  if (status.ok()) {
    // Done
  } else if (status.IsShutdownInProgress()) {
    // Ignore compaction errors found during shutting down
  } else {
    Log(InfoLogLevel::WARN_LEVEL, db_options_.info_log, "Compaction error: %s",
        status.ToString().c_str());
    if (db_options_.paranoid_checks && bg_error_.ok()) {
      bg_error_ = status;
    }
  }

  if (is_manual) {
    ManualCompaction* m = manual_compaction_;
    if (!status.ok()) {
      m->status = status;
      m->done = true;
    }
    // For universal compaction:
    //   Because universal compaction always happens at level 0, so one
    //   compaction will pick up all overlapped files. No files will be
    //   filtered out due to size limit and left for a successive compaction.
    //   So we can safely conclude the current compaction.
    //
    //   Also note that, if we don't stop here, then the current compaction
    //   writes a new file back to level 0, which will be used in successive
    //   compaction. Hence the manual compaction will never finish.
    //
    // Stop the compaction if manual_end points to nullptr -- this means
    // that we compacted the whole range. manual_end should always point
    // to nullptr in case of universal compaction
    if (manual_end == nullptr) {
      m->done = true;
    }
    if (!m->done) {
      // We only compacted part of the requested range.  Update *m
      // to the range that is left to be compacted.
      // Universal and FIFO compactions should always compact the whole range
      assert(m->cfd->ioptions()->compaction_style !=
                 kCompactionStyleUniversal ||
             m->cfd->ioptions()->num_levels > 1);
      assert(m->cfd->ioptions()->compaction_style != kCompactionStyleFIFO);
      m->tmp_storage = *manual_end;
      m->begin = &m->tmp_storage;
    }
    m->in_progress = false; // not being processed anymore
    manual_compaction_ = nullptr;
  }
  return status;
}

namespace {
struct IterState {
  IterState(DBImpl* _db, InstrumentedMutex* _mu, SuperVersion* _super_version)
      : db(_db), mu(_mu), super_version(_super_version) {}

  DBImpl* db;
  InstrumentedMutex* mu;
  SuperVersion* super_version;
};

static void CleanupIteratorState(void* arg1, void* arg2) {
  IterState* state = reinterpret_cast<IterState*>(arg1);

  if (state->super_version->Unref()) {
    // Job id == 0 means that this is not our background process, but rather
    // user thread
    JobContext job_context(0);

    state->mu->Lock();
    state->super_version->Cleanup();
    state->db->FindObsoleteFiles(&job_context, false, true);
    state->mu->Unlock();

    delete state->super_version;
    if (job_context.HaveSomethingToDelete()) {
      state->db->PurgeObsoleteFiles(job_context);
    }
    job_context.Clean();
  }

  delete state;
}
}  // namespace

Iterator* DBImpl::NewInternalIterator(const ReadOptions& read_options,
                                      ColumnFamilyData* cfd,
                                      SuperVersion* super_version,
                                      Arena* arena) {
  Iterator* internal_iter;
  assert(arena != nullptr);
  // Need to create internal iterator from the arena.
  MergeIteratorBuilder merge_iter_builder(&cfd->internal_comparator(), arena);
  // Collect iterator for mutable mem
  merge_iter_builder.AddIterator(
      super_version->mem->NewIterator(read_options, arena));
  // Collect all needed child iterators for immutable memtables
  super_version->imm->AddIterators(read_options, &merge_iter_builder);
  // Collect iterators for files in L0 - Ln
  super_version->current->AddIterators(read_options, env_options_,
                                       &merge_iter_builder);
  internal_iter = merge_iter_builder.Finish();
  IterState* cleanup = new IterState(this, &mutex_, super_version);
  internal_iter->RegisterCleanup(CleanupIteratorState, cleanup, nullptr);

  return internal_iter;
}

ColumnFamilyHandle* DBImpl::DefaultColumnFamily() const {
  return default_cf_handle_;
}

Status DBImpl::Get(const ReadOptions& read_options,
                   ColumnFamilyHandle* column_family, const Slice& key,
                   std::string* value) {
  return GetImpl(read_options, column_family, key, value);
}

// JobContext gets created and destructed outside of the lock --
// we
// use this convinently to:
// * malloc one SuperVersion() outside of the lock -- new_superversion
// * delete SuperVersion()s outside of the lock -- superversions_to_free
//
// However, if InstallSuperVersionAndScheduleWork() gets called twice with the
// same job_context, we can't reuse the SuperVersion() that got
// malloced because
// first call already used it. In that rare case, we take a hit and create a
// new SuperVersion() inside of the mutex. We do similar thing
// for superversion_to_free
void DBImpl::InstallSuperVersionAndScheduleWorkWrapper(
    ColumnFamilyData* cfd, JobContext* job_context,
    const MutableCFOptions& mutable_cf_options) {
  mutex_.AssertHeld();
  SuperVersion* old_superversion = InstallSuperVersionAndScheduleWork(
      cfd, job_context->new_superversion, mutable_cf_options);
  job_context->new_superversion = nullptr;
  job_context->superversions_to_free.push_back(old_superversion);
}

SuperVersion* DBImpl::InstallSuperVersionAndScheduleWork(
    ColumnFamilyData* cfd, SuperVersion* new_sv,
    const MutableCFOptions& mutable_cf_options) {
  mutex_.AssertHeld();

  // Update max_total_in_memory_state_
  size_t old_memtable_size = 0;
  auto* old_sv = cfd->GetSuperVersion();
  if (old_sv) {
    old_memtable_size = old_sv->mutable_cf_options.write_buffer_size *
                        old_sv->mutable_cf_options.max_write_buffer_number;
  }

  auto* old = cfd->InstallSuperVersion(
      new_sv ? new_sv : new SuperVersion(), &mutex_, mutable_cf_options);

  // Whenever we install new SuperVersion, we might need to issue new flushes or
  // compactions.
  SchedulePendingFlush(cfd);
  SchedulePendingCompaction(cfd);
  MaybeScheduleFlushOrCompaction();

  // Update max_total_in_memory_state_
  max_total_in_memory_state_ =
      max_total_in_memory_state_ - old_memtable_size +
      mutable_cf_options.write_buffer_size *
      mutable_cf_options.max_write_buffer_number;
  return old;
}

Status DBImpl::GetImpl(const ReadOptions& read_options,
                       ColumnFamilyHandle* column_family, const Slice& key,
                       std::string* value, bool* value_found) {
  StopWatch sw(env_, stats_, DB_GET);
  PERF_TIMER_GUARD(get_snapshot_time);

  auto cfh = reinterpret_cast<ColumnFamilyHandleImpl*>(column_family);
  auto cfd = cfh->cfd();

  SequenceNumber snapshot;
  if (read_options.snapshot != nullptr) {
    snapshot = reinterpret_cast<const SnapshotImpl*>(
        read_options.snapshot)->number_;
  } else {
    snapshot = versions_->LastSequence();
  }
  // Acquire SuperVersion
  SuperVersion* sv = GetAndRefSuperVersion(cfd);
  // Prepare to store a list of merge operations if merge occurs.
  MergeContext merge_context;

  Status s;
  // First look in the memtable, then in the immutable memtable (if any).
  // s is both in/out. When in, s could either be OK or MergeInProgress.
  // merge_operands will contain the sequence of merges in the latter case.
  LookupKey lkey(key, snapshot);
  PERF_TIMER_STOP(get_snapshot_time);

  if (sv->mem->Get(lkey, value, &s, &merge_context)) {
    // Done
    RecordTick(stats_, MEMTABLE_HIT);
  } else if (sv->imm->Get(lkey, value, &s, &merge_context)) {
    // Done
    RecordTick(stats_, MEMTABLE_HIT);
  } else {
    PERF_TIMER_GUARD(get_from_output_files_time);
    sv->current->Get(read_options, lkey, value, &s, &merge_context,
                     value_found);
    RecordTick(stats_, MEMTABLE_MISS);
  }

  {
    PERF_TIMER_GUARD(get_post_process_time);

    ReturnAndCleanupSuperVersion(cfd, sv);

    RecordTick(stats_, NUMBER_KEYS_READ);
    RecordTick(stats_, BYTES_READ, value->size());
  }
  return s;
}

std::vector<Status> DBImpl::MultiGet(
    const ReadOptions& read_options,
    const std::vector<ColumnFamilyHandle*>& column_family,
    const std::vector<Slice>& keys, std::vector<std::string>* values) {

  StopWatch sw(env_, stats_, DB_MULTIGET);
  PERF_TIMER_GUARD(get_snapshot_time);

  SequenceNumber snapshot;

  struct MultiGetColumnFamilyData {
    ColumnFamilyData* cfd;
    SuperVersion* super_version;
  };
  std::unordered_map<uint32_t, MultiGetColumnFamilyData*> multiget_cf_data;
  // fill up and allocate outside of mutex
  for (auto cf : column_family) {
    auto cfh = reinterpret_cast<ColumnFamilyHandleImpl*>(cf);
    auto cfd = cfh->cfd();
    if (multiget_cf_data.find(cfd->GetID()) == multiget_cf_data.end()) {
      auto mgcfd = new MultiGetColumnFamilyData();
      mgcfd->cfd = cfd;
      multiget_cf_data.insert({cfd->GetID(), mgcfd});
    }
  }

  mutex_.Lock();
  if (read_options.snapshot != nullptr) {
    snapshot = reinterpret_cast<const SnapshotImpl*>(
        read_options.snapshot)->number_;
  } else {
    snapshot = versions_->LastSequence();
  }
  for (auto mgd_iter : multiget_cf_data) {
    mgd_iter.second->super_version =
        mgd_iter.second->cfd->GetSuperVersion()->Ref();
  }
  mutex_.Unlock();

  // Contain a list of merge operations if merge occurs.
  MergeContext merge_context;

  // Note: this always resizes the values array
  size_t num_keys = keys.size();
  std::vector<Status> stat_list(num_keys);
  values->resize(num_keys);

  // Keep track of bytes that we read for statistics-recording later
  uint64_t bytes_read = 0;
  PERF_TIMER_STOP(get_snapshot_time);

  // For each of the given keys, apply the entire "get" process as follows:
  // First look in the memtable, then in the immutable memtable (if any).
  // s is both in/out. When in, s could either be OK or MergeInProgress.
  // merge_operands will contain the sequence of merges in the latter case.
  for (size_t i = 0; i < num_keys; ++i) {
    merge_context.Clear();
    Status& s = stat_list[i];
    std::string* value = &(*values)[i];

    LookupKey lkey(keys[i], snapshot);
    auto cfh = reinterpret_cast<ColumnFamilyHandleImpl*>(column_family[i]);
    auto mgd_iter = multiget_cf_data.find(cfh->cfd()->GetID());
    assert(mgd_iter != multiget_cf_data.end());
    auto mgd = mgd_iter->second;
    auto super_version = mgd->super_version;
    if (super_version->mem->Get(lkey, value, &s, &merge_context)) {
      // Done
    } else if (super_version->imm->Get(lkey, value, &s, &merge_context)) {
      // Done
    } else {
      PERF_TIMER_GUARD(get_from_output_files_time);
      super_version->current->Get(read_options, lkey, value, &s,
                                  &merge_context);
    }

    if (s.ok()) {
      bytes_read += value->size();
    }
  }

  // Post processing (decrement reference counts and record statistics)
  PERF_TIMER_GUARD(get_post_process_time);
  autovector<SuperVersion*> superversions_to_delete;

  // TODO(icanadi) do we need lock here or just around Cleanup()?
  mutex_.Lock();
  for (auto mgd_iter : multiget_cf_data) {
    auto mgd = mgd_iter.second;
    if (mgd->super_version->Unref()) {
      mgd->super_version->Cleanup();
      superversions_to_delete.push_back(mgd->super_version);
    }
  }
  mutex_.Unlock();

  for (auto td : superversions_to_delete) {
    delete td;
  }
  for (auto mgd : multiget_cf_data) {
    delete mgd.second;
  }

  RecordTick(stats_, NUMBER_MULTIGET_CALLS);
  RecordTick(stats_, NUMBER_MULTIGET_KEYS_READ, num_keys);
  RecordTick(stats_, NUMBER_MULTIGET_BYTES_READ, bytes_read);
  PERF_TIMER_STOP(get_post_process_time);

  return stat_list;
}

Status DBImpl::CreateColumnFamily(const ColumnFamilyOptions& cf_options,
                                  const std::string& column_family_name,
                                  ColumnFamilyHandle** handle) {
  Status s;
  *handle = nullptr;

  s = CheckCompressionSupported(cf_options);
  if (!s.ok()) {
    return s;
  }

  {
    InstrumentedMutexLock l(&mutex_);

    if (versions_->GetColumnFamilySet()->GetColumnFamily(column_family_name) !=
        nullptr) {
      return Status::InvalidArgument("Column family already exists");
    }
    VersionEdit edit;
    edit.AddColumnFamily(column_family_name);
    uint32_t new_id = versions_->GetColumnFamilySet()->GetNextColumnFamilyID();
    edit.SetColumnFamily(new_id);
    edit.SetLogNumber(logfile_number_);
    edit.SetComparatorName(cf_options.comparator->Name());

    // LogAndApply will both write the creation in MANIFEST and create
    // ColumnFamilyData object
    Options opt(db_options_, cf_options);
    {  // write thread
      WriteThread::Writer w;
      write_thread_.EnterUnbatched(&w, &mutex_);
      // LogAndApply will both write the creation in MANIFEST and create
      // ColumnFamilyData object
      s = versions_->LogAndApply(
          nullptr, MutableCFOptions(opt, ImmutableCFOptions(opt)), &edit,
          &mutex_, directories_.GetDbDir(), false, &cf_options);
      write_thread_.ExitUnbatched(&w);
    }
    if (s.ok()) {
      single_column_family_mode_ = false;
      auto* cfd =
          versions_->GetColumnFamilySet()->GetColumnFamily(column_family_name);
      assert(cfd != nullptr);
      delete InstallSuperVersionAndScheduleWork(
          cfd, nullptr, *cfd->GetLatestMutableCFOptions());

      if (!cfd->mem()->IsSnapshotSupported()) {
        is_snapshot_supported_ = false;
      }

      *handle = new ColumnFamilyHandleImpl(cfd, this, &mutex_);
      Log(InfoLogLevel::INFO_LEVEL, db_options_.info_log,
          "Created column family [%s] (ID %u)",
          column_family_name.c_str(), (unsigned)cfd->GetID());
    } else {
      Log(InfoLogLevel::ERROR_LEVEL, db_options_.info_log,
          "Creating column family [%s] FAILED -- %s",
          column_family_name.c_str(), s.ToString().c_str());
    }
  }  // InstrumentedMutexLock l(&mutex_)

  // this is outside the mutex
  if (s.ok()) {
    NewThreadStatusCfInfo(
        reinterpret_cast<ColumnFamilyHandleImpl*>(*handle)->cfd());
  }
  return s;
}

Status DBImpl::DropColumnFamily(ColumnFamilyHandle* column_family) {
  auto cfh = reinterpret_cast<ColumnFamilyHandleImpl*>(column_family);
  auto cfd = cfh->cfd();
  if (cfd->GetID() == 0) {
    return Status::InvalidArgument("Can't drop default column family");
  }

  bool cf_support_snapshot = cfd->mem()->IsSnapshotSupported();

  VersionEdit edit;
  edit.DropColumnFamily();
  edit.SetColumnFamily(cfd->GetID());

  Status s;
  {
    InstrumentedMutexLock l(&mutex_);
    if (cfd->IsDropped()) {
      s = Status::InvalidArgument("Column family already dropped!\n");
    }
    if (s.ok()) {
      // we drop column family from a single write thread
      WriteThread::Writer w;
      write_thread_.EnterUnbatched(&w, &mutex_);
      s = versions_->LogAndApply(cfd, *cfd->GetLatestMutableCFOptions(),
                                 &edit, &mutex_);
      write_thread_.ExitUnbatched(&w);
    }

    if (!cf_support_snapshot) {
      // Dropped Column Family doesn't support snapshot. Need to recalculate
      // is_snapshot_supported_.
      bool new_is_snapshot_supported = true;
      for (auto c : *versions_->GetColumnFamilySet()) {
        if (!c->IsDropped() && !c->mem()->IsSnapshotSupported()) {
          new_is_snapshot_supported = false;
          break;
        }
      }
      is_snapshot_supported_ = new_is_snapshot_supported;
    }
  }

  if (s.ok()) {
    // Note that here we erase the associated cf_info of the to-be-dropped
    // cfd before its ref-count goes to zero to avoid having to erase cf_info
    // later inside db_mutex.
    EraseThreadStatusCfInfo(cfd);
    assert(cfd->IsDropped());
    auto* mutable_cf_options = cfd->GetLatestMutableCFOptions();
    max_total_in_memory_state_ -= mutable_cf_options->write_buffer_size *
                                  mutable_cf_options->max_write_buffer_number;
    Log(InfoLogLevel::INFO_LEVEL, db_options_.info_log,
        "Dropped column family with id %u\n",
        cfd->GetID());
  } else {
    Log(InfoLogLevel::ERROR_LEVEL, db_options_.info_log,
        "Dropping column family with id %u FAILED -- %s\n",
        cfd->GetID(), s.ToString().c_str());
  }

  return s;
}

bool DBImpl::KeyMayExist(const ReadOptions& read_options,
                         ColumnFamilyHandle* column_family, const Slice& key,
                         std::string* value, bool* value_found) {
  if (value_found != nullptr) {
    // falsify later if key-may-exist but can't fetch value
    *value_found = true;
  }
  ReadOptions roptions = read_options;
  roptions.read_tier = kBlockCacheTier; // read from block cache only
  auto s = GetImpl(roptions, column_family, key, value, value_found);

  // If block_cache is enabled and the index block of the table didn't
  // not present in block_cache, the return value will be Status::Incomplete.
  // In this case, key may still exist in the table.
  return s.ok() || s.IsIncomplete();
}

Iterator* DBImpl::NewIterator(const ReadOptions& read_options,
                              ColumnFamilyHandle* column_family) {
  auto cfh = reinterpret_cast<ColumnFamilyHandleImpl*>(column_family);
  auto cfd = cfh->cfd();

  XFUNC_TEST("", "managed_new", managed_new1, xf_manage_new,
             reinterpret_cast<DBImpl*>(this),
             const_cast<ReadOptions*>(&read_options), is_snapshot_supported_);
  if (read_options.managed) {
#ifdef ROCKSDB_LITE
    // not supported in lite version
    return NewErrorIterator(Status::InvalidArgument(
        "Managed Iterators not supported in RocksDBLite."));
#else
    if ((read_options.tailing) || (read_options.snapshot != nullptr) ||
        (is_snapshot_supported_)) {
      return new ManagedIterator(this, read_options, cfd);
    }
    // Managed iter not supported
    return NewErrorIterator(Status::InvalidArgument(
        "Managed Iterators not supported without snapshots."));
#endif
  } else if (read_options.tailing) {
#ifdef ROCKSDB_LITE
    // not supported in lite version
    return nullptr;
#else
    SuperVersion* sv = cfd->GetReferencedSuperVersion(&mutex_);
    auto iter = new ForwardIterator(this, read_options, cfd, sv);
    return NewDBIterator(env_, *cfd->ioptions(), cfd->user_comparator(), iter,
        kMaxSequenceNumber,
        sv->mutable_cf_options.max_sequential_skip_in_iterations,
        read_options.iterate_upper_bound);
#endif
  } else {
    SequenceNumber latest_snapshot = versions_->LastSequence();
    SuperVersion* sv = cfd->GetReferencedSuperVersion(&mutex_);

    auto snapshot =
        read_options.snapshot != nullptr
            ? reinterpret_cast<const SnapshotImpl*>(
                read_options.snapshot)->number_
            : latest_snapshot;

    // Try to generate a DB iterator tree in continuous memory area to be
    // cache friendly. Here is an example of result:
    // +-------------------------------+
    // |                               |
    // | ArenaWrappedDBIter            |
    // |  +                            |
    // |  +---> Inner Iterator   ------------+
    // |  |                            |     |
    // |  |    +-- -- -- -- -- -- -- --+     |
    // |  +--- | Arena                 |     |
    // |       |                       |     |
    // |          Allocated Memory:    |     |
    // |       |   +-------------------+     |
    // |       |   | DBIter            | <---+
    // |           |  +                |
    // |       |   |  +-> iter_  ------------+
    // |       |   |                   |     |
    // |       |   +-------------------+     |
    // |       |   | MergingIterator   | <---+
    // |           |  +                |
    // |       |   |  +->child iter1  ------------+
    // |       |   |  |                |          |
    // |           |  +->child iter2  ----------+ |
    // |       |   |  |                |        | |
    // |       |   |  +->child iter3  --------+ | |
    // |           |                   |      | | |
    // |       |   +-------------------+      | | |
    // |       |   | Iterator1         | <--------+
    // |       |   +-------------------+      | |
    // |       |   | Iterator2         | <------+
    // |       |   +-------------------+      |
    // |       |   | Iterator3         | <----+
    // |       |   +-------------------+
    // |       |                       |
    // +-------+-----------------------+
    //
    // ArenaWrappedDBIter inlines an arena area where all the iterators in
    // the iterator tree are allocated in the order of being accessed when
    // querying.
    // Laying out the iterators in the order of being accessed makes it more
    // likely that any iterator pointer is close to the iterator it points to so
    // that they are likely to be in the same cache line and/or page.
    ArenaWrappedDBIter* db_iter = NewArenaWrappedDbIterator(
        env_, *cfd->ioptions(), cfd->user_comparator(),
        snapshot, sv->mutable_cf_options.max_sequential_skip_in_iterations,
        read_options.iterate_upper_bound);

    Iterator* internal_iter =
        NewInternalIterator(read_options, cfd, sv, db_iter->GetArena());
    db_iter->SetIterUnderDBIter(internal_iter);

    return db_iter;
  }
  // To stop compiler from complaining
  return nullptr;
}

Status DBImpl::NewIterators(
    const ReadOptions& read_options,
    const std::vector<ColumnFamilyHandle*>& column_families,
    std::vector<Iterator*>* iterators) {
  iterators->clear();
  iterators->reserve(column_families.size());
  XFUNC_TEST("", "managed_new", managed_new1, xf_manage_new,
             reinterpret_cast<DBImpl*>(this),
             const_cast<ReadOptions*>(&read_options), is_snapshot_supported_);
  if (read_options.managed) {
#ifdef ROCKSDB_LITE
    return Status::InvalidArgument(
        "Managed interator not supported in RocksDB lite");
#else
    if ((!read_options.tailing) && (read_options.snapshot == nullptr) &&
        (!is_snapshot_supported_)) {
      return Status::InvalidArgument(
          "Managed interator not supported without snapshots");
    }
    for (auto cfh : column_families) {
      auto cfd = reinterpret_cast<ColumnFamilyHandleImpl*>(cfh)->cfd();
      auto iter = new ManagedIterator(this, read_options, cfd);
      iterators->push_back(iter);
    }
#endif
  } else if (read_options.tailing) {
#ifdef ROCKSDB_LITE
    return Status::InvalidArgument(
        "Tailing interator not supported in RocksDB lite");
#else
    for (auto cfh : column_families) {
      auto cfd = reinterpret_cast<ColumnFamilyHandleImpl*>(cfh)->cfd();
      SuperVersion* sv = cfd->GetReferencedSuperVersion(&mutex_);
      auto iter = new ForwardIterator(this, read_options, cfd, sv);
      iterators->push_back(
          NewDBIterator(env_, *cfd->ioptions(), cfd->user_comparator(), iter,
              kMaxSequenceNumber,
              sv->mutable_cf_options.max_sequential_skip_in_iterations));
    }
#endif
  } else {
    SequenceNumber latest_snapshot = versions_->LastSequence();

    for (size_t i = 0; i < column_families.size(); ++i) {
      auto* cfd = reinterpret_cast<ColumnFamilyHandleImpl*>(
          column_families[i])->cfd();
      SuperVersion* sv = cfd->GetReferencedSuperVersion(&mutex_);

      auto snapshot =
          read_options.snapshot != nullptr
              ? reinterpret_cast<const SnapshotImpl*>(
                  read_options.snapshot)->number_
              : latest_snapshot;

      ArenaWrappedDBIter* db_iter = NewArenaWrappedDbIterator(
          env_, *cfd->ioptions(), cfd->user_comparator(), snapshot,
          sv->mutable_cf_options.max_sequential_skip_in_iterations);
      Iterator* internal_iter = NewInternalIterator(
          read_options, cfd, sv, db_iter->GetArena());
      db_iter->SetIterUnderDBIter(internal_iter);
      iterators->push_back(db_iter);
    }
  }

  return Status::OK();
}

const Snapshot* DBImpl::GetSnapshot() {
  int64_t unix_time = 0;
  env_->GetCurrentTime(&unix_time);  // Ignore error
  SnapshotImpl* s = new SnapshotImpl;

  InstrumentedMutexLock l(&mutex_);
  // returns null if the underlying memtable does not support snapshot.
  if (!is_snapshot_supported_) {
    delete s;
    return nullptr;
  }
  return snapshots_.New(s, versions_->LastSequence(), unix_time);
}

void DBImpl::ReleaseSnapshot(const Snapshot* s) {
  const SnapshotImpl* casted_s = reinterpret_cast<const SnapshotImpl*>(s);
  {
    InstrumentedMutexLock l(&mutex_);
    snapshots_.Delete(casted_s);
  }
  delete casted_s;
}

// Convenience methods
Status DBImpl::Put(const WriteOptions& o, ColumnFamilyHandle* column_family,
                   const Slice& key, const Slice& val) {
  return DB::Put(o, column_family, key, val);
}

Status DBImpl::Merge(const WriteOptions& o, ColumnFamilyHandle* column_family,
                     const Slice& key, const Slice& val) {
  auto cfh = reinterpret_cast<ColumnFamilyHandleImpl*>(column_family);
  if (!cfh->cfd()->ioptions()->merge_operator) {
    return Status::NotSupported("Provide a merge_operator when opening DB");
  } else {
    return DB::Merge(o, column_family, key, val);
  }
}

Status DBImpl::Delete(const WriteOptions& write_options,
                      ColumnFamilyHandle* column_family, const Slice& key) {
  return DB::Delete(write_options, column_family, key);
}

Status DBImpl::Write(const WriteOptions& write_options, WriteBatch* my_batch) {
  return WriteImpl(write_options, my_batch, nullptr);
}

#ifndef ROCKSDB_LITE
Status DBImpl::WriteWithCallback(const WriteOptions& write_options,
                                 WriteBatch* my_batch,
                                 WriteCallback* callback) {
  return WriteImpl(write_options, my_batch, callback);
}
#endif  // ROCKSDB_LITE

Status DBImpl::WriteImpl(const WriteOptions& write_options,
                         WriteBatch* my_batch, WriteCallback* callback) {
  if (my_batch == nullptr) {
    return Status::Corruption("Batch is nullptr!");
  }
  if (write_options.timeout_hint_us != 0) {
    return Status::InvalidArgument("timeout_hint_us is deprecated");
  }

  Status status;
  bool callback_failed = false;

  bool xfunc_attempted_write = false;
  XFUNC_TEST("transaction", "transaction_xftest_write_impl",
             xf_transaction_write1, xf_transaction_write, write_options,
             db_options_, my_batch, callback, this, &status,
             &xfunc_attempted_write);
  if (xfunc_attempted_write) {
    // Test already did the write
    return status;
  }

  PERF_TIMER_GUARD(write_pre_and_post_process_time);
  WriteThread::Writer w;
  w.batch = my_batch;
  w.sync = write_options.sync;
  w.disableWAL = write_options.disableWAL;
  w.in_batch_group = false;
  w.done = false;
  w.has_callback = (callback != nullptr) ? true : false;

  if (!write_options.disableWAL) {
    RecordTick(stats_, WRITE_WITH_WAL);
  }

  StopWatch write_sw(env_, db_options_.statistics.get(), DB_WRITE);

  write_thread_.JoinBatchGroup(&w);
  if (w.done) {
    // write was done by someone else, no need to grab mutex
    RecordTick(stats_, WRITE_DONE_BY_OTHER);
    return w.status;
  }
  // else we are the leader of the write batch group

  WriteContext context;
  mutex_.Lock();

  if (!write_options.disableWAL) {
    default_cf_internal_stats_->AddDBStats(InternalStats::WRITE_WITH_WAL, 1);
  }

  RecordTick(stats_, WRITE_DONE_BY_SELF);
  default_cf_internal_stats_->AddDBStats(InternalStats::WRITE_DONE_BY_SELF, 1);

  // Once reaches this point, the current writer "w" will try to do its write
  // job.  It may also pick up some of the remaining writers in the "writers_"
  // when it finds suitable, and finish them in the same write batch.
  // This is how a write job could be done by the other writer.
  assert(!single_column_family_mode_ ||
         versions_->GetColumnFamilySet()->NumberOfColumnFamilies() == 1);

  uint64_t max_total_wal_size = (db_options_.max_total_wal_size == 0)
                                    ? 4 * max_total_in_memory_state_
                                    : db_options_.max_total_wal_size;
  if (UNLIKELY(!single_column_family_mode_) &&
      alive_log_files_.begin()->getting_flushed == false &&
      total_log_size_ > max_total_wal_size) {
    uint64_t flush_column_family_if_log_file = alive_log_files_.begin()->number;
    alive_log_files_.begin()->getting_flushed = true;
    Log(InfoLogLevel::INFO_LEVEL, db_options_.info_log,
        "Flushing all column families with data in WAL number %" PRIu64
        ". Total log size is %" PRIu64 " while max_total_wal_size is %" PRIu64,
        flush_column_family_if_log_file, total_log_size_, max_total_wal_size);
    // no need to refcount because drop is happening in write thread, so can't
    // happen while we're in the write thread
    for (auto cfd : *versions_->GetColumnFamilySet()) {
      if (cfd->IsDropped()) {
        continue;
      }
      if (cfd->GetLogNumber() <= flush_column_family_if_log_file) {
        status = SwitchMemtable(cfd, &context);
        if (!status.ok()) {
          break;
        }
        cfd->imm()->FlushRequested();
        SchedulePendingFlush(cfd);
      }
    }
    MaybeScheduleFlushOrCompaction();
  } else if (UNLIKELY(write_buffer_.ShouldFlush())) {
    Log(InfoLogLevel::INFO_LEVEL, db_options_.info_log,
        "Flushing all column families. Write buffer is using %" PRIu64
        " bytes out of a total of %" PRIu64 ".",
        write_buffer_.memory_usage(), write_buffer_.buffer_size());
    // no need to refcount because drop is happening in write thread, so can't
    // happen while we're in the write thread
    for (auto cfd : *versions_->GetColumnFamilySet()) {
      if (cfd->IsDropped()) {
        continue;
      }
      if (!cfd->mem()->IsEmpty()) {
        status = SwitchMemtable(cfd, &context);
        if (!status.ok()) {
          break;
        }
        cfd->imm()->FlushRequested();
        SchedulePendingFlush(cfd);
      }
    }
    MaybeScheduleFlushOrCompaction();
  }

  if (UNLIKELY(status.ok() && !bg_error_.ok())) {
    status = bg_error_;
  }

  if (UNLIKELY(status.ok() && !flush_scheduler_.Empty())) {
    status = ScheduleFlushes(&context);
  }

  if (UNLIKELY(status.ok()) &&
      (write_controller_.IsStopped() || write_controller_.NeedsDelay())) {
    PERF_TIMER_STOP(write_pre_and_post_process_time);
    PERF_TIMER_GUARD(write_delay_time);
    // We don't know size of curent batch so that we always use the size
    // for previous one. It might create a fairness issue that expiration
    // might happen for smaller writes but larger writes can go through.
    // Can optimize it if it is an issue.
    status = DelayWrite(last_batch_group_size_);
    PERF_TIMER_START(write_pre_and_post_process_time);
  }

  uint64_t last_sequence = versions_->LastSequence();
  WriteThread::Writer* last_writer = &w;
  autovector<WriteBatch*> write_batch_group;
  bool need_log_sync = !write_options.disableWAL && write_options.sync;
  bool need_log_dir_sync = need_log_sync && !log_dir_synced_;

  if (status.ok()) {
    last_batch_group_size_ = write_thread_.EnterAsBatchGroupLeader(
        &w, &last_writer, &write_batch_group);

    if (need_log_sync) {
      while (logs_.front().getting_synced) {
        log_sync_cv_.Wait();
      }
      for (auto& log : logs_) {
        assert(!log.getting_synced);
        log.getting_synced = true;
      }
    }

    // Add to log and apply to memtable.  We can release the lock
    // during this phase since &w is currently responsible for logging
    // and protects against concurrent loggers and concurrent writes
    // into memtables

    mutex_.Unlock();

    if (callback != nullptr) {
      // If this write has a validation callback, check to see if this write
      // is able to be written.  Must be called on the write thread.
      status = callback->Callback(this);
      callback_failed = true;
    }
  } else {
    mutex_.Unlock();
  }

  // At this point the mutex is unlocked

  if (status.ok()) {
      WriteBatch* updates = nullptr;
      if (write_batch_group.size() == 1) {
        updates = write_batch_group[0];
      } else {
        updates = &tmp_batch_;
        for (size_t i = 0; i < write_batch_group.size(); ++i) {
          WriteBatchInternal::Append(updates, write_batch_group[i]);
        }
      }

      const SequenceNumber current_sequence = last_sequence + 1;
      WriteBatchInternal::SetSequence(updates, current_sequence);
      int my_batch_count = WriteBatchInternal::Count(updates);
      last_sequence += my_batch_count;
      const uint64_t batch_size = WriteBatchInternal::ByteSize(updates);
      // Record statistics
      RecordTick(stats_, NUMBER_KEYS_WRITTEN, my_batch_count);
      RecordTick(stats_, BYTES_WRITTEN, batch_size);
      if (write_options.disableWAL) {
        flush_on_destroy_ = true;
      }
      PERF_TIMER_STOP(write_pre_and_post_process_time);

      uint64_t log_size = 0;
      if (!write_options.disableWAL) {
        PERF_TIMER_GUARD(write_wal_time);
        Slice log_entry = WriteBatchInternal::Contents(updates);
        status = logs_.back().writer->AddRecord(log_entry);
        total_log_size_ += log_entry.size();
        alive_log_files_.back().AddSize(log_entry.size());
        log_empty_ = false;
        log_size = log_entry.size();
        RecordTick(stats_, WAL_FILE_BYTES, log_size);
        if (status.ok() && need_log_sync) {
          RecordTick(stats_, WAL_FILE_SYNCED);
          StopWatch sw(env_, stats_, WAL_FILE_SYNC_MICROS);
          // It's safe to access logs_ with unlocked mutex_ here because:
          //  - we've set getting_synced=true for all logs,
          //    so other threads won't pop from logs_ while we're here,
          //  - only writer thread can push to logs_, and we're in
          //    writer thread, so no one will push to logs_,
          //  - as long as other threads don't modify it, it's safe to read
          //    from std::deque from multiple threads concurrently.
          for (auto& log : logs_) {
            status = log.writer->file()->Sync(db_options_.use_fsync);
            if (!status.ok()) {
              break;
            }
          }
          if (status.ok() && need_log_dir_sync) {
            // We only sync WAL directory the first time WAL syncing is
            // requested, so that in case users never turn on WAL sync,
            // we can avoid the disk I/O in the write code path.
            status = directories_.GetWalDir()->Fsync();
          }
        }
      }
      if (status.ok()) {
        PERF_TIMER_GUARD(write_memtable_time);

        status = WriteBatchInternal::InsertInto(
            updates, column_family_memtables_.get(),
            write_options.ignore_missing_column_families, 0, this, false);
        // A non-OK status here indicates iteration failure (either in-memory
        // writebatch corruption (very bad), or the client specified invalid
        // column family).  This will later on trigger bg_error_.
        //
        // Note that existing logic was not sound. Any partial failure writing
        // into the memtable would result in a state that some write ops might
        // have succeeded in memtable but Status reports error for all writes.

        SetTickerCount(stats_, SEQUENCE_NUMBER, last_sequence);
      }
      PERF_TIMER_START(write_pre_and_post_process_time);
      if (updates == &tmp_batch_) {
        tmp_batch_.Clear();
      }
      mutex_.Lock();

      // internal stats
      default_cf_internal_stats_->AddDBStats(
          InternalStats::BYTES_WRITTEN, batch_size);
      default_cf_internal_stats_->AddDBStats(InternalStats::NUMBER_KEYS_WRITTEN,
                                             my_batch_count);
      if (!write_options.disableWAL) {
        default_cf_internal_stats_->AddDBStats(
            InternalStats::WAL_FILE_SYNCED, 1);
        default_cf_internal_stats_->AddDBStats(
            InternalStats::WAL_FILE_BYTES, log_size);
      }
      if (status.ok()) {
        versions_->SetLastSequence(last_sequence);
      }
  } else {
    // Operation failed.  Make sure sure mutex is held for cleanup code below.
    mutex_.Lock();
  }

  if (db_options_.paranoid_checks && !status.ok() && !callback_failed &&
      !status.IsBusy() && bg_error_.ok()) {
    bg_error_ = status; // stop compaction & fail any further writes
  }

  mutex_.AssertHeld();

  if (need_log_sync) {
    MarkLogsSynced(logfile_number_, need_log_dir_sync, status);
  }

  uint64_t writes_for_other = write_batch_group.size() - 1;
  if (writes_for_other > 0) {
    default_cf_internal_stats_->AddDBStats(InternalStats::WRITE_DONE_BY_OTHER,
                                           writes_for_other);
    if (!write_options.disableWAL) {
      default_cf_internal_stats_->AddDBStats(InternalStats::WRITE_WITH_WAL,
                                             writes_for_other);
    }
  }

  mutex_.Unlock();

  write_thread_.ExitAsBatchGroupLeader(&w, last_writer, status);

  return status;
}

// REQUIRES: mutex_ is held
// REQUIRES: this thread is currently at the front of the writer queue
Status DBImpl::DelayWrite(uint64_t num_bytes) {
  uint64_t time_delayed = 0;
  bool delayed = false;
  {
    StopWatch sw(env_, stats_, WRITE_STALL, &time_delayed);
    auto delay = write_controller_.GetDelay(env_, num_bytes);
    if (delay > 0) {
      mutex_.Unlock();
      delayed = true;
      TEST_SYNC_POINT("DBImpl::DelayWrite:Sleep");
      // hopefully we don't have to sleep more than 2 billion microseconds
      env_->SleepForMicroseconds(static_cast<int>(delay));
      mutex_.Lock();
    }

    while (bg_error_.ok() && write_controller_.IsStopped()) {
      delayed = true;
      TEST_SYNC_POINT("DBImpl::DelayWrite:Wait");
      bg_cv_.Wait();
    }
  }
  if (delayed) {
    default_cf_internal_stats_->AddDBStats(InternalStats::WRITE_STALL_MICROS,
                                           time_delayed);
    RecordTick(stats_, STALL_MICROS, time_delayed);
  }

  return bg_error_;
}

Status DBImpl::ScheduleFlushes(WriteContext* context) {
  ColumnFamilyData* cfd;
  while ((cfd = flush_scheduler_.GetNextColumnFamily()) != nullptr) {
    auto status = SwitchMemtable(cfd, context);
    if (cfd->Unref()) {
      delete cfd;
    }
    if (!status.ok()) {
      return status;
    }
  }
  return Status::OK();
}

// REQUIRES: mutex_ is held
// REQUIRES: this thread is currently at the front of the writer queue
Status DBImpl::SwitchMemtable(ColumnFamilyData* cfd, WriteContext* context) {
  mutex_.AssertHeld();
  unique_ptr<WritableFile> lfile;
  log::Writer* new_log = nullptr;
  MemTable* new_mem = nullptr;

  // Attempt to switch to a new memtable and trigger flush of old.
  // Do this without holding the dbmutex lock.
  assert(versions_->prev_log_number() == 0);
  bool creating_new_log = !log_empty_;
  uint64_t new_log_number =
      creating_new_log ? versions_->NewFileNumber() : logfile_number_;
  SuperVersion* new_superversion = nullptr;
  const MutableCFOptions mutable_cf_options = *cfd->GetLatestMutableCFOptions();
  mutex_.Unlock();
  Status s;
  {
    if (creating_new_log) {
      EnvOptions opt_env_opt =
          env_->OptimizeForLogWrite(env_options_, db_options_);
      s = env_->NewWritableFile(
          LogFileName(db_options_.wal_dir, new_log_number), &lfile,
          opt_env_opt);
      if (s.ok()) {
        // Our final size should be less than write_buffer_size
        // (compression, etc) but err on the side of caution.
        lfile->SetPreallocationBlockSize(
            1.1 * mutable_cf_options.write_buffer_size);
        unique_ptr<WritableFileWriter> file_writer(
            new WritableFileWriter(std::move(lfile), opt_env_opt));
        new_log = new log::Writer(std::move(file_writer));
      }
    }

    if (s.ok()) {
      SequenceNumber seq = versions_->LastSequence();
      new_mem = cfd->ConstructNewMemtable(mutable_cf_options, seq);
      new_superversion = new SuperVersion();
    }
  }
  Log(InfoLogLevel::DEBUG_LEVEL, db_options_.info_log,
      "[%s] New memtable created with log file: #%" PRIu64 "\n",
      cfd->GetName().c_str(), new_log_number);
  mutex_.Lock();
  if (!s.ok()) {
    // how do we fail if we're not creating new log?
    assert(creating_new_log);
    assert(!new_mem);
    assert(!new_log);
    return s;
  }
  if (creating_new_log) {
    logfile_number_ = new_log_number;
    assert(new_log != nullptr);
    log_empty_ = true;
    log_dir_synced_ = false;
    logs_.emplace_back(logfile_number_, new_log);
    alive_log_files_.push_back(LogFileNumberSize(logfile_number_));
    for (auto loop_cfd : *versions_->GetColumnFamilySet()) {
      // all this is just optimization to delete logs that
      // are no longer needed -- if CF is empty, that means it
      // doesn't need that particular log to stay alive, so we just
      // advance the log number. no need to persist this in the manifest
      if (loop_cfd->mem()->GetFirstSequenceNumber() == 0 &&
          loop_cfd->imm()->NumNotFlushed() == 0) {
        loop_cfd->SetLogNumber(logfile_number_);
      }
    }
  }
  cfd->mem()->SetNextLogNumber(logfile_number_);
  cfd->imm()->Add(cfd->mem(), &context->memtables_to_free_);
  new_mem->Ref();
  cfd->SetMemtable(new_mem);
  context->superversions_to_free_.push_back(InstallSuperVersionAndScheduleWork(
      cfd, new_superversion, mutable_cf_options));
  return s;
}

#ifndef ROCKSDB_LITE
Status DBImpl::GetPropertiesOfAllTables(ColumnFamilyHandle* column_family,
                                        TablePropertiesCollection* props) {
  auto cfh = reinterpret_cast<ColumnFamilyHandleImpl*>(column_family);
  auto cfd = cfh->cfd();

  // Increment the ref count
  mutex_.Lock();
  auto version = cfd->current();
  version->Ref();
  mutex_.Unlock();

  auto s = version->GetPropertiesOfAllTables(props);

  // Decrement the ref count
  mutex_.Lock();
  version->Unref();
  mutex_.Unlock();

  return s;
}
#endif  // ROCKSDB_LITE

const std::string& DBImpl::GetName() const {
  return dbname_;
}

Env* DBImpl::GetEnv() const {
  return env_;
}

const Options& DBImpl::GetOptions(ColumnFamilyHandle* column_family) const {
  auto cfh = reinterpret_cast<ColumnFamilyHandleImpl*>(column_family);
  return *cfh->cfd()->options();
}

const DBOptions& DBImpl::GetDBOptions() const { return db_options_; }

bool DBImpl::GetProperty(ColumnFamilyHandle* column_family,
                         const Slice& property, std::string* value) {
  bool is_int_property = false;
  bool need_out_of_mutex = false;
  DBPropertyType property_type =
      GetPropertyType(property, &is_int_property, &need_out_of_mutex);

  value->clear();
  if (is_int_property) {
    uint64_t int_value;
    bool ret_value = GetIntPropertyInternal(column_family, property_type,
                                            need_out_of_mutex, &int_value);
    if (ret_value) {
      *value = ToString(int_value);
    }
    return ret_value;
  } else {
    auto cfh = reinterpret_cast<ColumnFamilyHandleImpl*>(column_family);
    auto cfd = cfh->cfd();
    InstrumentedMutexLock l(&mutex_);
    return cfd->internal_stats()->GetStringProperty(property_type, property,
                                                    value);
  }
}

bool DBImpl::GetIntProperty(ColumnFamilyHandle* column_family,
                            const Slice& property, uint64_t* value) {
  bool is_int_property = false;
  bool need_out_of_mutex = false;
  DBPropertyType property_type =
      GetPropertyType(property, &is_int_property, &need_out_of_mutex);
  if (!is_int_property) {
    return false;
  }
  return GetIntPropertyInternal(column_family, property_type, need_out_of_mutex,
                                value);
}

bool DBImpl::GetIntPropertyInternal(ColumnFamilyHandle* column_family,
                                    DBPropertyType property_type,
                                    bool need_out_of_mutex, uint64_t* value) {
  auto cfh = reinterpret_cast<ColumnFamilyHandleImpl*>(column_family);
  auto cfd = cfh->cfd();

  if (!need_out_of_mutex) {
    InstrumentedMutexLock l(&mutex_);
    return cfd->internal_stats()->GetIntProperty(property_type, value, this);
  } else {
    SuperVersion* sv = GetAndRefSuperVersion(cfd);

    bool ret = cfd->internal_stats()->GetIntPropertyOutOfMutex(
        property_type, sv->current, value);

    ReturnAndCleanupSuperVersion(cfd, sv);

    return ret;
  }
}

SuperVersion* DBImpl::GetAndRefSuperVersion(ColumnFamilyData* cfd) {
  // TODO(ljin): consider using GetReferencedSuperVersion() directly
  return cfd->GetThreadLocalSuperVersion(&mutex_);
}

// REQUIRED: this function should only be called on the write thread or if the
// mutex is held.
SuperVersion* DBImpl::GetAndRefSuperVersion(uint32_t column_family_id) {
  auto column_family_set = versions_->GetColumnFamilySet();
  auto cfd = column_family_set->GetColumnFamily(column_family_id);
  if (!cfd) {
    return nullptr;
  }

  return GetAndRefSuperVersion(cfd);
}

// REQUIRED:  mutex is NOT held
SuperVersion* DBImpl::GetAndRefSuperVersionUnlocked(uint32_t column_family_id) {
  ColumnFamilyData* cfd;
  {
    InstrumentedMutexLock l(&mutex_);
    auto column_family_set = versions_->GetColumnFamilySet();
    cfd = column_family_set->GetColumnFamily(column_family_id);
  }

  if (!cfd) {
    return nullptr;
  }

  return GetAndRefSuperVersion(cfd);
}

void DBImpl::ReturnAndCleanupSuperVersion(ColumnFamilyData* cfd,
                                          SuperVersion* sv) {
  bool unref_sv = !cfd->ReturnThreadLocalSuperVersion(sv);

  if (unref_sv) {
    // Release SuperVersion
    if (sv->Unref()) {
      {
        InstrumentedMutexLock l(&mutex_);
        sv->Cleanup();
      }
      delete sv;
      RecordTick(stats_, NUMBER_SUPERVERSION_CLEANUPS);
    }
    RecordTick(stats_, NUMBER_SUPERVERSION_RELEASES);
  }
}

// REQUIRED: this function should only be called on the write thread.
void DBImpl::ReturnAndCleanupSuperVersion(uint32_t column_family_id,
                                          SuperVersion* sv) {
  auto column_family_set = versions_->GetColumnFamilySet();
  auto cfd = column_family_set->GetColumnFamily(column_family_id);

  // If SuperVersion is held, and we successfully fetched a cfd using
  // GetAndRefSuperVersion(), it must still exist.
  assert(cfd != nullptr);
  ReturnAndCleanupSuperVersion(cfd, sv);
}

// REQUIRED: Mutex should NOT be held.
void DBImpl::ReturnAndCleanupSuperVersionUnlocked(uint32_t column_family_id,
                                                  SuperVersion* sv) {
  ColumnFamilyData* cfd;
  {
    InstrumentedMutexLock l(&mutex_);
    auto column_family_set = versions_->GetColumnFamilySet();
    cfd = column_family_set->GetColumnFamily(column_family_id);
  }

  // If SuperVersion is held, and we successfully fetched a cfd using
  // GetAndRefSuperVersion(), it must still exist.
  assert(cfd != nullptr);
  ReturnAndCleanupSuperVersion(cfd, sv);
}

// REQUIRED: this function should only be called on the write thread or if the
// mutex is held.
ColumnFamilyHandle* DBImpl::GetColumnFamilyHandle(uint32_t column_family_id) {
  ColumnFamilyMemTables* cf_memtables = column_family_memtables_.get();

  if (!cf_memtables->Seek(column_family_id)) {
    return nullptr;
  }

  return cf_memtables->GetColumnFamilyHandle();
}

// REQUIRED: mutex is NOT held.
ColumnFamilyHandle* DBImpl::GetColumnFamilyHandleUnlocked(
    uint32_t column_family_id) {
  ColumnFamilyMemTables* cf_memtables = column_family_memtables_.get();

  InstrumentedMutexLock l(&mutex_);

  if (!cf_memtables->Seek(column_family_id)) {
    return nullptr;
  }

  return cf_memtables->GetColumnFamilyHandle();
}

void DBImpl::GetApproximateSizes(ColumnFamilyHandle* column_family,
                                 const Range* range, int n, uint64_t* sizes,
                                 bool include_memtable) {
  Version* v;
  auto cfh = reinterpret_cast<ColumnFamilyHandleImpl*>(column_family);
  auto cfd = cfh->cfd();
  SuperVersion* sv = GetAndRefSuperVersion(cfd);
  v = sv->current;

  for (int i = 0; i < n; i++) {
    // Convert user_key into a corresponding internal key.
    InternalKey k1(range[i].start, kMaxSequenceNumber, kValueTypeForSeek);
    InternalKey k2(range[i].limit, kMaxSequenceNumber, kValueTypeForSeek);
    sizes[i] = versions_->ApproximateSize(v, k1.Encode(), k2.Encode());
    if (include_memtable) {
      sizes[i] += sv->mem->ApproximateSize(k1.Encode(), k2.Encode());
      sizes[i] += sv->imm->ApproximateSize(k1.Encode(), k2.Encode());
    }
  }

  ReturnAndCleanupSuperVersion(cfd, sv);
}

std::list<uint64_t>::iterator
DBImpl::CaptureCurrentFileNumberInPendingOutputs() {
  // We need to remember the iterator of our insert, because after the
  // background job is done, we need to remove that element from
  // pending_outputs_.
  pending_outputs_.push_back(versions_->current_next_file_number());
  auto pending_outputs_inserted_elem = pending_outputs_.end();
  --pending_outputs_inserted_elem;
  return pending_outputs_inserted_elem;
}

void DBImpl::ReleaseFileNumberFromPendingOutputs(
    std::list<uint64_t>::iterator v) {
  pending_outputs_.erase(v);
}

#ifndef ROCKSDB_LITE
Status DBImpl::GetUpdatesSince(
    SequenceNumber seq, unique_ptr<TransactionLogIterator>* iter,
    const TransactionLogIterator::ReadOptions& read_options) {

  RecordTick(stats_, GET_UPDATES_SINCE_CALLS);
  if (seq > versions_->LastSequence()) {
    return Status::NotFound("Requested sequence not yet written in the db");
  }
  return wal_manager_.GetUpdatesSince(seq, iter, read_options, versions_.get());
}

Status DBImpl::DeleteFile(std::string name) {
  uint64_t number;
  FileType type;
  WalFileType log_type;
  if (!ParseFileName(name, &number, &type, &log_type) ||
      (type != kTableFile && type != kLogFile)) {
    Log(InfoLogLevel::ERROR_LEVEL, db_options_.info_log,
        "DeleteFile %s failed.\n", name.c_str());
    return Status::InvalidArgument("Invalid file name");
  }

  Status status;
  if (type == kLogFile) {
    // Only allow deleting archived log files
    if (log_type != kArchivedLogFile) {
      Log(InfoLogLevel::ERROR_LEVEL, db_options_.info_log,
          "DeleteFile %s failed - not archived log.\n",
          name.c_str());
      return Status::NotSupported("Delete only supported for archived logs");
    }
    status = env_->DeleteFile(db_options_.wal_dir + "/" + name.c_str());
    if (!status.ok()) {
      Log(InfoLogLevel::ERROR_LEVEL, db_options_.info_log,
          "DeleteFile %s failed -- %s.\n",
          name.c_str(), status.ToString().c_str());
    }
    return status;
  }

  int level;
  FileMetaData* metadata;
  ColumnFamilyData* cfd;
  VersionEdit edit;
  JobContext job_context(next_job_id_.fetch_add(1), true);
  {
    InstrumentedMutexLock l(&mutex_);
    status = versions_->GetMetadataForFile(number, &level, &metadata, &cfd);
    if (!status.ok()) {
      Log(InfoLogLevel::WARN_LEVEL, db_options_.info_log,
          "DeleteFile %s failed. File not found\n", name.c_str());
      job_context.Clean();
      return Status::InvalidArgument("File not found");
    }
    assert(level < cfd->NumberLevels());

    // If the file is being compacted no need to delete.
    if (metadata->being_compacted) {
      Log(InfoLogLevel::INFO_LEVEL, db_options_.info_log,
          "DeleteFile %s Skipped. File about to be compacted\n", name.c_str());
      job_context.Clean();
      return Status::OK();
    }

    // Only the files in the last level can be deleted externally.
    // This is to make sure that any deletion tombstones are not
    // lost. Check that the level passed is the last level.
    auto* vstoreage = cfd->current()->storage_info();
    for (int i = level + 1; i < cfd->NumberLevels(); i++) {
      if (vstoreage->NumLevelFiles(i) != 0) {
        Log(InfoLogLevel::WARN_LEVEL, db_options_.info_log,
            "DeleteFile %s FAILED. File not in last level\n", name.c_str());
        job_context.Clean();
        return Status::InvalidArgument("File not in last level");
      }
    }
    // if level == 0, it has to be the oldest file
    if (level == 0 &&
        vstoreage->LevelFiles(0).back()->fd.GetNumber() != number) {
      Log(InfoLogLevel::WARN_LEVEL, db_options_.info_log,
          "DeleteFile %s failed ---"
          " target file in level 0 must be the oldest.", name.c_str());
      job_context.Clean();
      return Status::InvalidArgument("File in level 0, but not oldest");
    }
    edit.SetColumnFamily(cfd->GetID());
    edit.DeleteFile(level, number);
    status = versions_->LogAndApply(cfd, *cfd->GetLatestMutableCFOptions(),
                                    &edit, &mutex_, directories_.GetDbDir());
    if (status.ok()) {
      InstallSuperVersionAndScheduleWorkWrapper(
          cfd, &job_context, *cfd->GetLatestMutableCFOptions());
    }
    FindObsoleteFiles(&job_context, false);
  }  // lock released here

  LogFlush(db_options_.info_log);
  // remove files outside the db-lock
  if (job_context.HaveSomethingToDelete()) {
    // Call PurgeObsoleteFiles() without holding mutex.
    PurgeObsoleteFiles(job_context);
  }
  job_context.Clean();
  return status;
}

void DBImpl::GetLiveFilesMetaData(std::vector<LiveFileMetaData>* metadata) {
  InstrumentedMutexLock l(&mutex_);
  versions_->GetLiveFilesMetaData(metadata);
}

void DBImpl::GetColumnFamilyMetaData(
    ColumnFamilyHandle* column_family,
    ColumnFamilyMetaData* cf_meta) {
  assert(column_family);
  auto* cfd = reinterpret_cast<ColumnFamilyHandleImpl*>(column_family)->cfd();
  auto* sv = GetAndRefSuperVersion(cfd);
  sv->current->GetColumnFamilyMetaData(cf_meta);
  ReturnAndCleanupSuperVersion(cfd, sv);
}

#endif  // ROCKSDB_LITE

Status DBImpl::CheckConsistency() {
  mutex_.AssertHeld();
  std::vector<LiveFileMetaData> metadata;
  versions_->GetLiveFilesMetaData(&metadata);

  std::string corruption_messages;
  for (const auto& md : metadata) {
    // md.name has a leading "/".
    std::string file_path = md.db_path + md.name;

    uint64_t fsize = 0;
    Status s = env_->GetFileSize(file_path, &fsize);
    if (!s.ok()) {
      corruption_messages +=
          "Can't access " + md.name + ": " + s.ToString() + "\n";
    } else if (fsize != md.size) {
      corruption_messages += "Sst file size mismatch: " + file_path +
                             ". Size recorded in manifest " +
                             ToString(md.size) + ", actual size " +
                             ToString(fsize) + "\n";
    }
  }
  if (corruption_messages.size() == 0) {
    return Status::OK();
  } else {
    return Status::Corruption(corruption_messages);
  }
}

Status DBImpl::GetDbIdentity(std::string& identity) const {
  std::string idfilename = IdentityFileName(dbname_);
  const EnvOptions soptions;
  unique_ptr<SequentialFileReader> id_file_reader;
  Status s;
  {
    unique_ptr<SequentialFile> idfile;
    s = env_->NewSequentialFile(idfilename, &idfile, soptions);
    if (!s.ok()) {
      return s;
    }
    id_file_reader.reset(new SequentialFileReader(std::move(idfile)));
  }

  uint64_t file_size;
  s = env_->GetFileSize(idfilename, &file_size);
  if (!s.ok()) {
    return s;
  }
  char* buffer = reinterpret_cast<char*>(alloca(file_size));
  Slice id;
  s = id_file_reader->Read(static_cast<size_t>(file_size), &id, buffer);
  if (!s.ok()) {
    return s;
  }
  identity.assign(id.ToString());
  // If last character is '\n' remove it from identity
  if (identity.size() > 0 && identity.back() == '\n') {
    identity.pop_back();
  }
  return s;
}

// Default implementations of convenience methods that subclasses of DB
// can call if they wish
Status DB::Put(const WriteOptions& opt, ColumnFamilyHandle* column_family,
               const Slice& key, const Slice& value) {
  // Pre-allocate size of write batch conservatively.
  // 8 bytes are taken by header, 4 bytes for count, 1 byte for type,
  // and we allocate 11 extra bytes for key length, as well as value length.
  WriteBatch batch(key.size() + value.size() + 24);
  batch.Put(column_family, key, value);
  return Write(opt, &batch);
}

Status DB::Delete(const WriteOptions& opt, ColumnFamilyHandle* column_family,
                  const Slice& key) {
  WriteBatch batch;
  batch.Delete(column_family, key);
  return Write(opt, &batch);
}

Status DB::Merge(const WriteOptions& opt, ColumnFamilyHandle* column_family,
                 const Slice& key, const Slice& value) {
  WriteBatch batch;
  batch.Merge(column_family, key, value);
  return Write(opt, &batch);
}

// Default implementation -- returns not supported status
Status DB::CreateColumnFamily(const ColumnFamilyOptions& cf_options,
                              const std::string& column_family_name,
                              ColumnFamilyHandle** handle) {
  return Status::NotSupported("");
}
Status DB::DropColumnFamily(ColumnFamilyHandle* column_family) {
  return Status::NotSupported("");
}

DB::~DB() { }

Status DB::Open(const Options& options, const std::string& dbname, DB** dbptr) {
  DBOptions db_options(options);
  ColumnFamilyOptions cf_options(options);
  std::vector<ColumnFamilyDescriptor> column_families;
  column_families.push_back(
      ColumnFamilyDescriptor(kDefaultColumnFamilyName, cf_options));
  std::vector<ColumnFamilyHandle*> handles;
  Status s = DB::Open(db_options, dbname, column_families, &handles, dbptr);
  if (s.ok()) {
    assert(handles.size() == 1);
    // i can delete the handle since DBImpl is always holding a reference to
    // default column family
    delete handles[0];
  }
  return s;
}

Status DB::Open(const DBOptions& db_options, const std::string& dbname,
                const std::vector<ColumnFamilyDescriptor>& column_families,
                std::vector<ColumnFamilyHandle*>* handles, DB** dbptr) {
  Status s = SanitizeOptionsByTable(db_options, column_families);
  if (!s.ok()) {
    return s;
  }

  for (auto& cfd : column_families) {
    s = CheckCompressionSupported(cfd.options);
    if (!s.ok()) {
      return s;
    }
    if (db_options.db_paths.size() > 1) {
      if ((cfd.options.compaction_style != kCompactionStyleUniversal) &&
          (cfd.options.compaction_style != kCompactionStyleLevel)) {
        return Status::NotSupported(
            "More than one DB paths are only supported in "
            "universal and level compaction styles. ");
      }
    }
  }

  if (db_options.db_paths.size() > 4) {
    return Status::NotSupported(
        "More than four DB paths are not supported yet. ");
  }

  *dbptr = nullptr;
  handles->clear();

  size_t max_write_buffer_size = 0;
  for (auto cf : column_families) {
    max_write_buffer_size =
        std::max(max_write_buffer_size, cf.options.write_buffer_size);
  }

  DBImpl* impl = new DBImpl(db_options, dbname);
  s = impl->env_->CreateDirIfMissing(impl->db_options_.wal_dir);
  if (s.ok()) {
    for (auto db_path : impl->db_options_.db_paths) {
      s = impl->env_->CreateDirIfMissing(db_path.path);
      if (!s.ok()) {
        break;
      }
    }
  }

  if (!s.ok()) {
    delete impl;
    return s;
  }

  s = impl->CreateArchivalDirectory();
  if (!s.ok()) {
    delete impl;
    return s;
  }
  impl->mutex_.Lock();
  // Handles create_if_missing, error_if_exists
  s = impl->Recover(column_families);
  if (s.ok()) {
    uint64_t new_log_number = impl->versions_->NewFileNumber();
    unique_ptr<WritableFile> lfile;
    EnvOptions soptions(db_options);
    EnvOptions opt_env_options =
        impl->db_options_.env->OptimizeForLogWrite(soptions, impl->db_options_);
    s = impl->db_options_.env->NewWritableFile(
        LogFileName(impl->db_options_.wal_dir, new_log_number), &lfile,
        opt_env_options);
    if (s.ok()) {
      lfile->SetPreallocationBlockSize(1.1 * max_write_buffer_size);
      impl->logfile_number_ = new_log_number;
      unique_ptr<WritableFileWriter> file_writer(
          new WritableFileWriter(std::move(lfile), opt_env_options));
      impl->logs_.emplace_back(new_log_number,
                               new log::Writer(std::move(file_writer)));

      // set column family handles
      for (auto cf : column_families) {
        auto cfd =
            impl->versions_->GetColumnFamilySet()->GetColumnFamily(cf.name);
        if (cfd != nullptr) {
          handles->push_back(
              new ColumnFamilyHandleImpl(cfd, impl, &impl->mutex_));
          impl->NewThreadStatusCfInfo(cfd);
        } else {
          if (db_options.create_missing_column_families) {
            // missing column family, create it
            ColumnFamilyHandle* handle;
            impl->mutex_.Unlock();
            s = impl->CreateColumnFamily(cf.options, cf.name, &handle);
            impl->mutex_.Lock();
            if (s.ok()) {
              handles->push_back(handle);
            } else {
              break;
            }
          } else {
            s = Status::InvalidArgument("Column family not found: ", cf.name);
            break;
          }
        }
      }
    }
    if (s.ok()) {
      for (auto cfd : *impl->versions_->GetColumnFamilySet()) {
        delete impl->InstallSuperVersionAndScheduleWork(
            cfd, nullptr, *cfd->GetLatestMutableCFOptions());
      }
      impl->alive_log_files_.push_back(
          DBImpl::LogFileNumberSize(impl->logfile_number_));
      impl->DeleteObsoleteFiles();
      s = impl->directories_.GetDbDir()->Fsync();
    }
  }

  if (s.ok()) {
    for (auto cfd : *impl->versions_->GetColumnFamilySet()) {
      if (cfd->ioptions()->compaction_style == kCompactionStyleFIFO) {
        auto* vstorage = cfd->current()->storage_info();
        for (int i = 1; i < vstorage->num_levels(); ++i) {
          int num_files = vstorage->NumLevelFiles(i);
          if (num_files > 0) {
            s = Status::InvalidArgument(
                "Not all files are at level 0. Cannot "
                "open with FIFO compaction style.");
            break;
          }
        }
      }
      if (!cfd->mem()->IsSnapshotSupported()) {
        impl->is_snapshot_supported_ = false;
      }
      if (cfd->ioptions()->merge_operator != nullptr &&
          !cfd->mem()->IsMergeOperatorSupported()) {
        s = Status::InvalidArgument(
            "The memtable of column family %s does not support merge operator "
            "its options.merge_operator is non-null", cfd->GetName().c_str());
      }
      if (!s.ok()) {
        break;
      }
    }
  }
  TEST_SYNC_POINT("DBImpl::Open:Opened");
  if (s.ok()) {
    impl->opened_successfully_ = true;
    impl->MaybeScheduleFlushOrCompaction();
  }
  impl->mutex_.Unlock();

  if (s.ok()) {
    Log(InfoLogLevel::INFO_LEVEL, impl->db_options_.info_log, "DB pointer %p",
        impl);
    *dbptr = impl;
  } else {
    for (auto* h : *handles) {
      delete h;
    }
    handles->clear();
    delete impl;
  }
  return s;
}

Status DB::ListColumnFamilies(const DBOptions& db_options,
                              const std::string& name,
                              std::vector<std::string>* column_families) {
  return VersionSet::ListColumnFamilies(column_families, name, db_options.env);
}

Snapshot::~Snapshot() {
}

Status DestroyDB(const std::string& dbname, const Options& options) {
  const InternalKeyComparator comparator(options.comparator);
  const Options& soptions(SanitizeOptions(dbname, &comparator, options));
  Env* env = soptions.env;
  std::vector<std::string> filenames;

  // Ignore error in case directory does not exist
  env->GetChildren(dbname, &filenames);

  FileLock* lock;
  const std::string lockname = LockFileName(dbname);
  Status result = env->LockFile(lockname, &lock);
  if (result.ok()) {
    uint64_t number;
    FileType type;
    InfoLogPrefix info_log_prefix(!options.db_log_dir.empty(), dbname);
    for (size_t i = 0; i < filenames.size(); i++) {
      if (ParseFileName(filenames[i], &number, info_log_prefix.prefix, &type) &&
          type != kDBLockFile) {  // Lock file will be deleted at end
        Status del;
        std::string path_to_delete = dbname + "/" + filenames[i];
        if (type == kMetaDatabase) {
          del = DestroyDB(path_to_delete, options);
        } else if (type == kTableFile) {
          del = DeleteOrMoveToTrash(&options, path_to_delete);
        } else {
          del = env->DeleteFile(path_to_delete);
        }
        if (result.ok() && !del.ok()) {
          result = del;
        }
      }
    }

    for (size_t path_id = 0; path_id < options.db_paths.size(); path_id++) {
      const auto& db_path = options.db_paths[path_id];
      env->GetChildren(db_path.path, &filenames);
      for (size_t i = 0; i < filenames.size(); i++) {
        if (ParseFileName(filenames[i], &number, &type) &&
            type == kTableFile) {  // Lock file will be deleted at end
          Status del;
          std::string table_path = db_path.path + "/" + filenames[i];
          if (path_id == 0) {
            del = DeleteOrMoveToTrash(&options, table_path);
          } else {
            del = env->DeleteFile(table_path);
          }
          if (result.ok() && !del.ok()) {
            result = del;
          }
        }
      }
    }

    std::vector<std::string> walDirFiles;
    std::string archivedir = ArchivalDirectory(dbname);
    if (dbname != soptions.wal_dir) {
      env->GetChildren(soptions.wal_dir, &walDirFiles);
      archivedir = ArchivalDirectory(soptions.wal_dir);
    }

    // Delete log files in the WAL dir
    for (const auto& file : walDirFiles) {
      if (ParseFileName(file, &number, &type) && type == kLogFile) {
        Status del = env->DeleteFile(soptions.wal_dir + "/" + file);
        if (result.ok() && !del.ok()) {
          result = del;
        }
      }
    }

    std::vector<std::string> archiveFiles;
    env->GetChildren(archivedir, &archiveFiles);
    // Delete archival files.
    for (size_t i = 0; i < archiveFiles.size(); ++i) {
      if (ParseFileName(archiveFiles[i], &number, &type) &&
          type == kLogFile) {
        Status del = env->DeleteFile(archivedir + "/" + archiveFiles[i]);
        if (result.ok() && !del.ok()) {
          result = del;
        }
      }
    }
    // ignore case where no archival directory is present.
    env->DeleteDir(archivedir);

    env->UnlockFile(lock);  // Ignore error since state is already gone
    env->DeleteFile(lockname);
    env->DeleteDir(dbname);  // Ignore error in case dir contains other files
    env->DeleteDir(soptions.wal_dir);
  }
  return result;
}

#if ROCKSDB_USING_THREAD_STATUS

void DBImpl::NewThreadStatusCfInfo(
    ColumnFamilyData* cfd) const {
  if (db_options_.enable_thread_tracking) {
    ThreadStatusUtil::NewColumnFamilyInfo(this, cfd);
  }
}

void DBImpl::EraseThreadStatusCfInfo(
    ColumnFamilyData* cfd) const {
  if (db_options_.enable_thread_tracking) {
    ThreadStatusUtil::EraseColumnFamilyInfo(cfd);
  }
}

void DBImpl::EraseThreadStatusDbInfo() const {
  if (db_options_.enable_thread_tracking) {
    ThreadStatusUtil::EraseDatabaseInfo(this);
  }
}

#else
void DBImpl::NewThreadStatusCfInfo(
    ColumnFamilyData* cfd) const {
}

void DBImpl::EraseThreadStatusCfInfo(
    ColumnFamilyData* cfd) const {
}

void DBImpl::EraseThreadStatusDbInfo() const {
}
#endif  // ROCKSDB_USING_THREAD_STATUS

//
// A global method that can dump out the build version
void DumpRocksDBBuildVersion(Logger * log) {
#if !defined(IOS_CROSS_COMPILE)
  // if we compile with Xcode, we don't run build_detect_vesion, so we don't
  // generate util/build_version.cc
  Warn(log,
      "RocksDB version: %d.%d.%d\n", ROCKSDB_MAJOR, ROCKSDB_MINOR,
      ROCKSDB_PATCH);
  Warn(log, "Git sha %s", rocksdb_build_git_sha);
  Warn(log, "Compile date %s", rocksdb_build_compile_date);
#endif
}

#ifndef ROCKSDB_LITE
SequenceNumber DBImpl::GetEarliestMemTableSequenceNumber(SuperVersion* sv,
                                                         bool include_history) {
  // Find the earliest sequence number that we know we can rely on reading
  // from the memtable without needing to check sst files.
  SequenceNumber earliest_seq =
      sv->imm->GetEarliestSequenceNumber(include_history);
  if (earliest_seq == kMaxSequenceNumber) {
    earliest_seq = sv->mem->GetEarliestSequenceNumber();
  }
  assert(sv->mem->GetEarliestSequenceNumber() >= earliest_seq);

  return earliest_seq;
}
#endif  // ROCKSDB_LITE

#ifndef ROCKSDB_LITE
Status DBImpl::GetLatestSequenceForKeyFromMemtable(SuperVersion* sv,
                                                   const Slice& key,
                                                   SequenceNumber* seq) {
  Status s;
  std::string value;
  MergeContext merge_context;

  SequenceNumber current_seq = versions_->LastSequence();
  LookupKey lkey(key, current_seq);

  *seq = kMaxSequenceNumber;

  // Check if there is a record for this key in the latest memtable
  sv->mem->Get(lkey, &value, &s, &merge_context, seq);

  if (!(s.ok() || s.IsNotFound() || s.IsMergeInProgress())) {
    // unexpected error reading memtable.
    Log(InfoLogLevel::ERROR_LEVEL, db_options_.info_log,
        "Unexpected status returned from MemTable::Get: %s\n",
        s.ToString().c_str());

    return s;
  }

  if (*seq != kMaxSequenceNumber) {
    // Found a sequence number, no need to check immutable memtables
    return Status::OK();
  }

  // Check if there is a record for this key in the immutable memtables
  sv->imm->Get(lkey, &value, &s, &merge_context, seq);

  if (!(s.ok() || s.IsNotFound() || s.IsMergeInProgress())) {
    // unexpected error reading memtable.
    Log(InfoLogLevel::ERROR_LEVEL, db_options_.info_log,
        "Unexpected status returned from MemTableList::Get: %s\n",
        s.ToString().c_str());

    return s;
  }

  if (*seq != kMaxSequenceNumber) {
    // Found a sequence number, no need to check memtable history
    return Status::OK();
  }

  // Check if there is a record for this key in the immutable memtables
  sv->imm->GetFromHistory(lkey, &value, &s, &merge_context, seq);

  if (!(s.ok() || s.IsNotFound() || s.IsMergeInProgress())) {
    // unexpected error reading memtable.
    Log(InfoLogLevel::ERROR_LEVEL, db_options_.info_log,
        "Unexpected status returned from MemTableList::GetFromHistory: %s\n",
        s.ToString().c_str());

    return s;
  }

  return Status::OK();
}
#endif  // ROCKSDB_LITE

}  // namespace rocksdb
