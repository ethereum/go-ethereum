//  Copyright (c) 2015, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.

#pragma once

#include <map>
#include <queue>
#include <string>
#include <thread>

#include "port/port.h"

#include "rocksdb/delete_scheduler.h"
#include "rocksdb/status.h"

namespace rocksdb {

class Env;
class Logger;

class DeleteSchedulerImpl : public DeleteScheduler {
 public:
  DeleteSchedulerImpl(Env* env, const std::string& trash_dir,
                      int64_t rate_bytes_per_sec,
                      std::shared_ptr<Logger> info_log);

  ~DeleteSchedulerImpl();

  // Return delete rate limit in bytes per second
  int64_t GetRateBytesPerSecond() { return rate_bytes_per_sec_; }

  // Move file to trash directory and schedule it's deletion
  Status DeleteFile(const std::string& fname);

  // Wait for all files being deleteing in the background to finish or for
  // destructor to be called.
  void WaitForEmptyTrash();

  // Return a map containing errors that happened in BackgroundEmptyTrash
  // file_path => error status
  std::map<std::string, Status> GetBackgroundErrors();

 private:
  Status MoveToTrash(const std::string& file_path, std::string* path_in_trash);

  Status DeleteTrashFile(const std::string& path_in_trash,
                         uint64_t* deleted_bytes);

  void BackgroundEmptyTrash();

  Env* env_;
  // Path to the trash directory
  std::string trash_dir_;
  // Maximum number of bytes that should be deleted per second
  int64_t rate_bytes_per_sec_;
  // Mutex to protect queue_, pending_files_, bg_errors_, closing_
  port::Mutex mu_;
  // Queue of files in trash that need to be deleted
  std::queue<std::string> queue_;
  // Number of files in trash that are waiting to be deleted
  int32_t pending_files_;
  // Errors that happened in BackgroundEmptyTrash (file_path => error)
  std::map<std::string, Status> bg_errors_;
  // Set to true in ~DeleteSchedulerImpl() to force BackgroundEmptyTrash to stop
  bool closing_;
  // Condition variable signaled in these conditions
  //    - pending_files_ value change from 0 => 1
  //    - pending_files_ value change from 1 => 0
  //    - closing_ value is set to true
  port::CondVar cv_;
  // Background thread running BackgroundEmptyTrash
  std::unique_ptr<std::thread> bg_thread_;
  // Mutex to protect threads from file name conflicts
  port::Mutex file_move_mu_;
  std::shared_ptr<Logger> info_log_;
  static const uint64_t kMicrosInSecond = 1000 * 1000LL;
};

}  // namespace rocksdb
