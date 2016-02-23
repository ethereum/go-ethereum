//  Copyright (c) 2015, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.

#include "util/delete_scheduler_impl.h"

#include <thread>
#include <vector>

#include "port/port.h"
#include "rocksdb/env.h"
#include "util/mutexlock.h"
#include "util/sync_point.h"

namespace rocksdb {

DeleteSchedulerImpl::DeleteSchedulerImpl(Env* env, const std::string& trash_dir,
                                         int64_t rate_bytes_per_sec,
                                         std::shared_ptr<Logger> info_log)
    : env_(env),
      trash_dir_(trash_dir),
      rate_bytes_per_sec_(rate_bytes_per_sec),
      pending_files_(0),
      closing_(false),
      cv_(&mu_),
      info_log_(info_log) {
  if (rate_bytes_per_sec_ == 0) {
    // Rate limiting is disabled
    bg_thread_.reset();
  } else {
    bg_thread_.reset(
        new std::thread(&DeleteSchedulerImpl::BackgroundEmptyTrash, this));
  }
}

DeleteSchedulerImpl::~DeleteSchedulerImpl() {
  {
    MutexLock l(&mu_);
    closing_ = true;
    cv_.SignalAll();
  }
  if (bg_thread_) {
    bg_thread_->join();
  }
}

Status DeleteSchedulerImpl::DeleteFile(const std::string& file_path) {
  if (rate_bytes_per_sec_ == 0) {
    // Rate limiting is disabled
    return env_->DeleteFile(file_path);
  }

  // Move file to trash
  std::string path_in_trash;
  Status s = MoveToTrash(file_path, &path_in_trash);
  if (!s.ok()) {
    Log(InfoLogLevel::ERROR_LEVEL, info_log_,
        "Failed to move %s to trash directory (%s)", file_path.c_str(),
        trash_dir_.c_str());
    return env_->DeleteFile(file_path);
  }

  // Add file to delete queue
  {
    MutexLock l(&mu_);
    queue_.push(path_in_trash);
    pending_files_++;
    if (pending_files_ == 1) {
      cv_.SignalAll();
    }
  }
  return s;
}

std::map<std::string, Status> DeleteSchedulerImpl::GetBackgroundErrors() {
  MutexLock l(&mu_);
  return bg_errors_;
}

Status DeleteSchedulerImpl::MoveToTrash(const std::string& file_path,
                                        std::string* path_in_trash) {
  Status s;
  // Figure out the name of the file in trash folder
  size_t idx = file_path.rfind("/");
  if (idx == std::string::npos || idx == file_path.size() - 1) {
    return Status::InvalidArgument("file_path is corrupted");
  }
  *path_in_trash = trash_dir_ + file_path.substr(idx);
  std::string unique_suffix = "";

  if (*path_in_trash == file_path) {
    // This file is already in trash
    return s;
  }

  // TODO(tec) : Implement Env::RenameFileIfNotExist and remove
  //             file_move_mu mutex.
  MutexLock l(&file_move_mu_);
  while (true) {
    s = env_->FileExists(*path_in_trash + unique_suffix);
    if (s.IsNotFound()) {
      // We found a path for our file in trash
      *path_in_trash += unique_suffix;
      s = env_->RenameFile(file_path, *path_in_trash);
      break;
    } else if (s.ok()) {
      // Name conflict, generate new random suffix
      unique_suffix = env_->GenerateUniqueId();
    } else {
      // Error during FileExists call, we cannot continue
      break;
    }
  }
  return s;
}

void DeleteSchedulerImpl::BackgroundEmptyTrash() {
  TEST_SYNC_POINT("DeleteSchedulerImpl::BackgroundEmptyTrash");

  while (true) {
    MutexLock l(&mu_);
    while (queue_.empty() && !closing_) {
      cv_.Wait();
    }

    if (closing_) {
      return;
    }

    // Delete all files in queue_
    uint64_t start_time = env_->NowMicros();
    uint64_t total_deleted_bytes = 0;
    while (!queue_.empty() && !closing_) {
      std::string path_in_trash = queue_.front();
      queue_.pop();

      // We dont need to hold the lock while deleting the file
      mu_.Unlock();
      uint64_t deleted_bytes = 0;
      // Delete file from trash and update total_penlty value
      Status s = DeleteTrashFile(path_in_trash,  &deleted_bytes);
      total_deleted_bytes += deleted_bytes;
      mu_.Lock();

      if (!s.ok()) {
        bg_errors_[path_in_trash] = s;
      }

      // Apply penlty if necessary
      uint64_t total_penlty =
          ((total_deleted_bytes * kMicrosInSecond) / rate_bytes_per_sec_);
      while (!closing_ && !cv_.TimedWait(start_time + total_penlty)) {}
      TEST_SYNC_POINT_CALLBACK("DeleteSchedulerImpl::BackgroundEmptyTrash:Wait",
                               &total_penlty);

      pending_files_--;
      if (pending_files_ == 0) {
        // Unblock WaitForEmptyTrash since there are no more files waiting
        // to be deleted
        cv_.SignalAll();
      }
    }
  }
}

Status DeleteSchedulerImpl::DeleteTrashFile(const std::string& path_in_trash,
                                            uint64_t* deleted_bytes) {
  uint64_t file_size;
  Status s = env_->GetFileSize(path_in_trash, &file_size);
  if (s.ok()) {
    TEST_SYNC_POINT("DeleteSchedulerImpl::DeleteTrashFile:DeleteFile");
    s = env_->DeleteFile(path_in_trash);
  }

  if (!s.ok()) {
    // Error while getting file size or while deleting
    Log(InfoLogLevel::ERROR_LEVEL, info_log_,
        "Failed to delete %s from trash -- %s", path_in_trash.c_str(),
        s.ToString().c_str());
    *deleted_bytes = 0;
  } else {
    *deleted_bytes = file_size;
  }

  return s;
}

void DeleteSchedulerImpl::WaitForEmptyTrash() {
  MutexLock l(&mu_);
  while (pending_files_ > 0 && !closing_) {
    cv_.Wait();
  }
}

DeleteScheduler* NewDeleteScheduler(Env* env, const std::string& trash_dir,
                                    int64_t rate_bytes_per_sec,
                                    std::shared_ptr<Logger> info_log,
                                    bool delete_exisitng_trash,
                                    Status* status) {
  DeleteScheduler* res =
      new DeleteSchedulerImpl(env, trash_dir, rate_bytes_per_sec, info_log);

  Status s;
  if (trash_dir != "") {
    s = env->CreateDirIfMissing(trash_dir);
    if (s.ok() && delete_exisitng_trash) {
      std::vector<std::string> files_in_trash;
      s = env->GetChildren(trash_dir, &files_in_trash);
      if (s.ok()) {
        for (const std::string& trash_file : files_in_trash) {
          if (trash_file == "." || trash_file == "..") {
            continue;
          }
          Status file_delete = res->DeleteFile(trash_dir + "/" + trash_file);
          if (s.ok() && !file_delete.ok()) {
            s = file_delete;
          }
        }
      }
    }
  }

  if (status) {
    *status = s;
  }

  return res;
}

}  // namespace rocksdb
