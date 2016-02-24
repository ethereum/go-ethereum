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
#include <vector>

#include "db/column_family.h"
#include "db/job_context.h"
#include "db/version_set.h"
#include "rocksdb/status.h"

namespace rocksdb {

#ifndef ROCKSDB_LITE
Status DBImpl::SuggestCompactRange(ColumnFamilyHandle* column_family,
                                   const Slice* begin, const Slice* end) {
  auto cfh = reinterpret_cast<ColumnFamilyHandleImpl*>(column_family);
  auto cfd = cfh->cfd();
  InternalKey start_key, end_key;
  if (begin != nullptr) {
    start_key.SetMaxPossibleForUserKey(*begin);
  }
  if (end != nullptr) {
    end_key.SetMinPossibleForUserKey(*end);
  }
  {
    InstrumentedMutexLock l(&mutex_);
    auto vstorage = cfd->current()->storage_info();
    for (int level = 0; level < vstorage->num_non_empty_levels() - 1; ++level) {
      std::vector<FileMetaData*> inputs;
      vstorage->GetOverlappingInputs(
          level, begin == nullptr ? nullptr : &start_key,
          end == nullptr ? nullptr : &end_key, &inputs);
      for (auto f : inputs) {
        f->marked_for_compaction = true;
      }
    }
    // Since we have some more files to compact, we should also recompute
    // compaction score
    vstorage->ComputeCompactionScore(*cfd->GetLatestMutableCFOptions(),
                                     CompactionOptionsFIFO());
    SchedulePendingCompaction(cfd);
    MaybeScheduleFlushOrCompaction();
  }
  return Status::OK();
}

Status DBImpl::PromoteL0(ColumnFamilyHandle* column_family, int target_level) {
  assert(column_family);

  if (target_level < 1) {
    Log(InfoLogLevel::INFO_LEVEL, db_options_.info_log,
        "PromoteL0 FAILED. Invalid target level %d\n", target_level);
    return Status::InvalidArgument("Invalid target level");
  }

  Status status;
  VersionEdit edit;
  JobContext job_context(next_job_id_.fetch_add(1), true);
  {
    InstrumentedMutexLock l(&mutex_);
    auto* cfd = static_cast<ColumnFamilyHandleImpl*>(column_family)->cfd();
    const auto* vstorage = cfd->current()->storage_info();

    if (target_level >= vstorage->num_levels()) {
      Log(InfoLogLevel::INFO_LEVEL, db_options_.info_log,
          "PromoteL0 FAILED. Target level %d does not exist\n", target_level);
      job_context.Clean();
      return Status::InvalidArgument("Target level does not exist");
    }

    // Sort L0 files by range.
    const InternalKeyComparator* icmp = &cfd->internal_comparator();
    auto l0_files = vstorage->LevelFiles(0);
    std::sort(l0_files.begin(), l0_files.end(),
              [icmp](FileMetaData* f1, FileMetaData* f2) {
                return icmp->Compare(f1->largest, f2->largest) < 0;
              });

    // Check that no L0 file is being compacted and that they have
    // non-overlapping ranges.
    for (size_t i = 0; i < l0_files.size(); ++i) {
      auto f = l0_files[i];
      if (f->being_compacted) {
        Log(InfoLogLevel::INFO_LEVEL, db_options_.info_log,
            "PromoteL0 FAILED. File %" PRIu64 " being compacted\n",
            f->fd.GetNumber());
        job_context.Clean();
        return Status::InvalidArgument("PromoteL0 called during L0 compaction");
      }

      if (i == 0) continue;
      auto prev_f = l0_files[i - 1];
      if (icmp->Compare(prev_f->largest, f->smallest) >= 0) {
        Log(InfoLogLevel::INFO_LEVEL, db_options_.info_log,
            "PromoteL0 FAILED. Files %" PRIu64 " and %" PRIu64
            " have overlapping ranges\n",
            prev_f->fd.GetNumber(), f->fd.GetNumber());
        job_context.Clean();
        return Status::InvalidArgument("L0 has overlapping files");
      }
    }

    // Check that all levels up to target_level are empty.
    for (int level = 1; level <= target_level; ++level) {
      if (vstorage->NumLevelFiles(level) > 0) {
        Log(InfoLogLevel::INFO_LEVEL, db_options_.info_log,
            "PromoteL0 FAILED. Level %d not empty\n", level);
        job_context.Clean();
        return Status::InvalidArgument(
            "All levels up to target_level "
            "must be empty");
      }
    }

    edit.SetColumnFamily(cfd->GetID());
    for (const auto& f : l0_files) {
      edit.DeleteFile(0, f->fd.GetNumber());
      edit.AddFile(target_level, f->fd.GetNumber(), f->fd.GetPathId(),
                   f->fd.GetFileSize(), f->smallest, f->largest,
                   f->smallest_seqno, f->largest_seqno,
                   f->marked_for_compaction);
    }

    status = versions_->LogAndApply(cfd, *cfd->GetLatestMutableCFOptions(),
                                    &edit, &mutex_, directories_.GetDbDir());
    if (status.ok()) {
      InstallSuperVersionAndScheduleWorkWrapper(
          cfd, &job_context, *cfd->GetLatestMutableCFOptions());
    }
  }  // lock released here
  LogFlush(db_options_.info_log);
  job_context.Clean();

  return status;
}
#endif  // ROCKSDB_LITE

}  // namespace rocksdb
