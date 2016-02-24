//  Copyright (c) 2013, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.
//
// Copyright (c) 2012 Facebook.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

#ifndef ROCKSDB_LITE

#ifndef __STDC_FORMAT_MACROS
#define __STDC_FORMAT_MACROS
#endif

#include <inttypes.h>
#include <algorithm>
#include <string>
#include <stdint.h>
#include "db/db_impl.h"
#include "db/filename.h"
#include "db/job_context.h"
#include "db/version_set.h"
#include "rocksdb/db.h"
#include "rocksdb/env.h"
#include "port/port.h"
#include "util/mutexlock.h"
#include "util/sync_point.h"
#include "util/file_util.h"

namespace rocksdb {

Status DBImpl::DisableFileDeletions() {
  InstrumentedMutexLock l(&mutex_);
  ++disable_delete_obsolete_files_;
  if (disable_delete_obsolete_files_ == 1) {
    Log(InfoLogLevel::INFO_LEVEL, db_options_.info_log,
        "File Deletions Disabled");
  } else {
    Log(InfoLogLevel::WARN_LEVEL, db_options_.info_log,
        "File Deletions Disabled, but already disabled. Counter: %d",
        disable_delete_obsolete_files_);
  }
  return Status::OK();
}

Status DBImpl::EnableFileDeletions(bool force) {
  // Job id == 0 means that this is not our background process, but rather
  // user thread
  JobContext job_context(0);
  bool should_purge_files = false;
  {
    InstrumentedMutexLock l(&mutex_);
    if (force) {
      // if force, we need to enable file deletions right away
      disable_delete_obsolete_files_ = 0;
    } else if (disable_delete_obsolete_files_ > 0) {
      --disable_delete_obsolete_files_;
    }
    if (disable_delete_obsolete_files_ == 0)  {
      Log(InfoLogLevel::INFO_LEVEL, db_options_.info_log,
          "File Deletions Enabled");
      should_purge_files = true;
      FindObsoleteFiles(&job_context, true);
    } else {
      Log(InfoLogLevel::WARN_LEVEL, db_options_.info_log,
          "File Deletions Enable, but not really enabled. Counter: %d",
          disable_delete_obsolete_files_);
    }
  }
  if (should_purge_files)  {
    PurgeObsoleteFiles(job_context);
  }
  job_context.Clean();
  LogFlush(db_options_.info_log);
  return Status::OK();
}

int DBImpl::IsFileDeletionsEnabled() const {
  return disable_delete_obsolete_files_;
}

Status DBImpl::GetLiveFiles(std::vector<std::string>& ret,
                            uint64_t* manifest_file_size,
                            bool flush_memtable) {

  *manifest_file_size = 0;

  mutex_.Lock();

  if (flush_memtable) {
    // flush all dirty data to disk.
    Status status;
    for (auto cfd : *versions_->GetColumnFamilySet()) {
      if (cfd->IsDropped()) {
        continue;
      }
      cfd->Ref();
      mutex_.Unlock();
      status = FlushMemTable(cfd, FlushOptions());
      TEST_SYNC_POINT("DBImpl::GetLiveFiles:1");
      TEST_SYNC_POINT("DBImpl::GetLiveFiles:2");
      mutex_.Lock();
      cfd->Unref();
      if (!status.ok()) {
        break;
      }
    }
    versions_->GetColumnFamilySet()->FreeDeadColumnFamilies();

    if (!status.ok()) {
      mutex_.Unlock();
      Log(InfoLogLevel::ERROR_LEVEL, db_options_.info_log,
          "Cannot Flush data %s\n", status.ToString().c_str());
      return status;
    }
  }

  // Make a set of all of the live *.sst files
  std::vector<FileDescriptor> live;
  for (auto cfd : *versions_->GetColumnFamilySet()) {
    if (cfd->IsDropped()) {
      continue;
    }
    cfd->current()->AddLiveFiles(&live);
  }

  ret.clear();
  ret.reserve(live.size() + 2); //*.sst + CURRENT + MANIFEST

  // create names of the live files. The names are not absolute
  // paths, instead they are relative to dbname_;
  for (auto live_file : live) {
    ret.push_back(MakeTableFileName("", live_file.GetNumber()));
  }

  ret.push_back(CurrentFileName(""));
  ret.push_back(DescriptorFileName("", versions_->manifest_file_number()));

  // find length of manifest file while holding the mutex lock
  *manifest_file_size = versions_->manifest_file_size();

  mutex_.Unlock();
  return Status::OK();
}

Status DBImpl::GetSortedWalFiles(VectorLogPtr& files) {
  return wal_manager_.GetSortedWalFiles(files);
}

}

#endif  // ROCKSDB_LITE
