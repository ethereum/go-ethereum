//  Copyright (c) 2013, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.
//
// Copyright (c) 2011 The LevelDB Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file. See the AUTHORS file for names of contributors.

#include "db/builder.h"

#include <algorithm>
#include <deque>
#include <vector>

#include "db/compaction_iterator.h"
#include "db/dbformat.h"
#include "db/filename.h"
#include "db/internal_stats.h"
#include "db/merge_helper.h"
#include "db/table_cache.h"
#include "db/version_edit.h"
#include "rocksdb/db.h"
#include "rocksdb/env.h"
#include "rocksdb/iterator.h"
#include "rocksdb/options.h"
#include "rocksdb/table.h"
#include "table/block_based_table_builder.h"
#include "util/file_reader_writer.h"
#include "util/iostats_context_imp.h"
#include "util/stop_watch.h"
#include "util/thread_status_util.h"

namespace rocksdb {

class TableFactory;

TableBuilder* NewTableBuilder(
    const ImmutableCFOptions& ioptions,
    const InternalKeyComparator& internal_comparator,
    const std::vector<std::unique_ptr<IntTblPropCollectorFactory>>*
        int_tbl_prop_collector_factories,
    WritableFileWriter* file, const CompressionType compression_type,
    const CompressionOptions& compression_opts, const bool skip_filters) {
  return ioptions.table_factory->NewTableBuilder(
      TableBuilderOptions(ioptions, internal_comparator,
                          int_tbl_prop_collector_factories, compression_type,
                          compression_opts, skip_filters),
      file);
}

Status BuildTable(
    const std::string& dbname, Env* env, const ImmutableCFOptions& ioptions,
    const EnvOptions& env_options, TableCache* table_cache, Iterator* iter,
    FileMetaData* meta, const InternalKeyComparator& internal_comparator,
    const std::vector<std::unique_ptr<IntTblPropCollectorFactory>>*
        int_tbl_prop_collector_factories,
    std::vector<SequenceNumber> snapshots, const CompressionType compression,
    const CompressionOptions& compression_opts, bool paranoid_file_checks,
    InternalStats* internal_stats, const Env::IOPriority io_priority,
    TableProperties* table_properties) {
  // Reports the IOStats for flush for every following bytes.
  const size_t kReportFlushIOStatsEvery = 1048576;
  Status s;
  meta->fd.file_size = 0;
  iter->SeekToFirst();

  std::string fname = TableFileName(ioptions.db_paths, meta->fd.GetNumber(),
                                    meta->fd.GetPathId());
  if (iter->Valid()) {
    TableBuilder* builder;
    unique_ptr<WritableFileWriter> file_writer;
    {
      unique_ptr<WritableFile> file;
      s = env->NewWritableFile(fname, &file, env_options);
      if (!s.ok()) {
        return s;
      }
      file->SetIOPriority(io_priority);

      file_writer.reset(new WritableFileWriter(std::move(file), env_options));

      builder = NewTableBuilder(
          ioptions, internal_comparator, int_tbl_prop_collector_factories,
          file_writer.get(), compression, compression_opts);
    }

    MergeHelper merge(internal_comparator.user_comparator(),
                      ioptions.merge_operator, ioptions.info_log,
                      ioptions.min_partial_merge_operands,
                      true /* internal key corruption is not ok */);

    CompactionIterator c_iter(iter, internal_comparator.user_comparator(),
                              &merge, kMaxSequenceNumber, &snapshots, env,
                              true /* internal key corruption is not ok */);
    c_iter.SeekToFirst();
    for (; c_iter.Valid(); c_iter.Next()) {
      const Slice& key = c_iter.key();
      const Slice& value = c_iter.value();
      builder->Add(key, value);
      meta->UpdateBoundaries(key, c_iter.ikey().sequence);

      // TODO(noetzli): Update stats after flush, too.
      if (io_priority == Env::IO_HIGH &&
          IOSTATS(bytes_written) >= kReportFlushIOStatsEvery) {
        ThreadStatusUtil::SetThreadOperationProperty(
            ThreadStatus::FLUSH_BYTES_WRITTEN, IOSTATS(bytes_written));
      }
    }

    // Finish and check for builder errors
    s = c_iter.status();
    if (s.ok()) {
      s = builder->Finish();
    } else {
      builder->Abandon();
    }
    if (s.ok()) {
      meta->fd.file_size = builder->FileSize();
      meta->marked_for_compaction = builder->NeedCompact();
      assert(meta->fd.GetFileSize() > 0);
      if (table_properties) {
        *table_properties = builder->GetTableProperties();
      }
    }
    delete builder;

    // Finish and check for file errors
    if (s.ok() && !ioptions.disable_data_sync) {
      StopWatch sw(env, ioptions.statistics, TABLE_SYNC_MICROS);
      file_writer->Sync(ioptions.use_fsync);
    }
    if (s.ok()) {
      s = file_writer->Close();
    }

    if (s.ok()) {
      // Verify that the table is usable
      Iterator* it = table_cache->NewIterator(
          ReadOptions(), env_options, internal_comparator, meta->fd, nullptr,
          (internal_stats == nullptr) ? nullptr
                                      : internal_stats->GetFileReadHist(0),
          false);
      s = it->status();
      if (s.ok() && paranoid_file_checks) {
        for (it->SeekToFirst(); it->Valid(); it->Next()) {}
        s = it->status();
      }

      delete it;
    }
  }

  // Check for input iterator errors
  if (!iter->status().ok()) {
    s = iter->status();
  }

  if (s.ok() && meta->fd.GetFileSize() > 0) {
    // Keep it
  } else {
    env->DeleteFile(fname);
  }
  return s;
}

}  // namespace rocksdb
