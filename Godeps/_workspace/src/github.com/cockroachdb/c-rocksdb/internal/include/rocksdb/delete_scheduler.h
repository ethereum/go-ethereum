//  Copyright (c) 2015, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.

#pragma once

#include <map>
#include <string>

#include "rocksdb/status.h"

namespace rocksdb {

class Env;
class Logger;

// DeleteScheduler allow the DB to enforce a rate limit on file deletion,
// Instead of deleteing files immediately, files are moved to trash_dir
// and deleted in a background thread that apply sleep penlty between deletes
// if they are happening in a rate faster than rate_bytes_per_sec,
//
// Rate limiting can be turned off by setting rate_bytes_per_sec = 0, In this
// case DeleteScheduler will delete files immediately.
class DeleteScheduler {
 public:
  virtual ~DeleteScheduler() {}

  // Return delete rate limit in bytes per second
  virtual int64_t GetRateBytesPerSecond() = 0;

  // Move file to trash directory and schedule it's deletion
  virtual Status DeleteFile(const std::string& fname) = 0;

  // Return a map containing errors that happened in the background thread
  // file_path => error status
  virtual std::map<std::string, Status> GetBackgroundErrors() = 0;

  // Wait for all files being deleteing in the background to finish or for
  // destructor to be called.
  virtual void WaitForEmptyTrash() = 0;
};

// Create a new DeleteScheduler that can be shared among multiple RocksDB
// instances to control the file deletion rate.
//
// @env: Pointer to Env object, please see "rocksdb/env.h".
// @trash_dir: Path to the directory where deleted files will be moved into
//    to be deleted in a background thread while applying rate limiting. If this
//    directory dont exist, it will be created. This directory should not be
//    used by any other process or any other DeleteScheduler.
// @rate_bytes_per_sec: How many bytes should be deleted per second, If this
//    value is set to 1024 (1 Kb / sec) and we deleted a file of size 4 Kb
//    in 1 second, we will wait for another 3 seconds before we delete other
//    files, Set to 0 to disable rate limiting.
// @info_log: If not nullptr, info_log will be used to log errors.
// @delete_exisitng_trash: If set to true, the newly created DeleteScheduler
//    will delete files that already exist in trash_dir.
// @status: If not nullptr, status will contain any errors that happened during
//    creating the missing trash_dir or deleting existing files in trash.
extern DeleteScheduler* NewDeleteScheduler(
    Env* env, const std::string& trash_dir, int64_t rate_bytes_per_sec,
    std::shared_ptr<Logger> info_log = nullptr,
    bool delete_exisitng_trash = true, Status* status = nullptr);

}  // namespace rocksdb
