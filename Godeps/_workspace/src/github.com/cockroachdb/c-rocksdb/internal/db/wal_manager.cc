//  Copyright (c) 2013, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.
//
// Copyright (c) 2011 The LevelDB Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file. See the AUTHORS file for names of contributors.

#include "db/wal_manager.h"

#ifndef __STDC_FORMAT_MACROS
#define __STDC_FORMAT_MACROS
#endif

#include <inttypes.h>
#include <algorithm>
#include <vector>
#include <memory>

#include "db/filename.h"
#include "db/transaction_log_impl.h"
#include "db/log_reader.h"
#include "db/log_writer.h"
#include "db/write_batch_internal.h"
#include "port/port.h"
#include "rocksdb/env.h"
#include "rocksdb/options.h"
#include "rocksdb/write_batch.h"
#include "util/coding.h"
#include "util/file_reader_writer.h"
#include "util/logging.h"
#include "util/mutexlock.h"
#include "util/sync_point.h"
#include "util/string_util.h"

namespace rocksdb {

#ifndef ROCKSDB_LITE

Status WalManager::GetSortedWalFiles(VectorLogPtr& files) {
  // First get sorted files in db dir, then get sorted files from archived
  // dir, to avoid a race condition where a log file is moved to archived
  // dir in between.
  Status s;
  // list wal files in main db dir.
  VectorLogPtr logs;
  s = GetSortedWalsOfType(db_options_.wal_dir, logs, kAliveLogFile);
  if (!s.ok()) {
    return s;
  }

  // Reproduce the race condition where a log file is moved
  // to archived dir, between these two sync points, used in
  // (DBTest,TransactionLogIteratorRace)
  TEST_SYNC_POINT("WalManager::GetSortedWalFiles:1");
  TEST_SYNC_POINT("WalManager::GetSortedWalFiles:2");

  files.clear();
  // list wal files in archive dir.
  std::string archivedir = ArchivalDirectory(db_options_.wal_dir);
  Status exists = env_->FileExists(archivedir);
  if (exists.ok()) {
    s = GetSortedWalsOfType(archivedir, files, kArchivedLogFile);
    if (!s.ok()) {
      return s;
    }
  } else if (!exists.IsNotFound()) {
    assert(s.IsIOError());
    return s;
  }

  uint64_t latest_archived_log_number = 0;
  if (!files.empty()) {
    latest_archived_log_number = files.back()->LogNumber();
    Log(InfoLogLevel::INFO_LEVEL, db_options_.info_log,
        "Latest Archived log: %" PRIu64,
        latest_archived_log_number);
  }

  files.reserve(files.size() + logs.size());
  for (auto& log : logs) {
    if (log->LogNumber() > latest_archived_log_number) {
      files.push_back(std::move(log));
    } else {
      // When the race condition happens, we could see the
      // same log in both db dir and archived dir. Simply
      // ignore the one in db dir. Note that, if we read
      // archived dir first, we would have missed the log file.
      Log(InfoLogLevel::WARN_LEVEL, db_options_.info_log,
          "%s already moved to archive", log->PathName().c_str());
    }
  }

  return s;
}

Status WalManager::GetUpdatesSince(
    SequenceNumber seq, std::unique_ptr<TransactionLogIterator>* iter,
    const TransactionLogIterator::ReadOptions& read_options,
    VersionSet* version_set) {

  //  Get all sorted Wal Files.
  //  Do binary search and open files and find the seq number.

  std::unique_ptr<VectorLogPtr> wal_files(new VectorLogPtr);
  Status s = GetSortedWalFiles(*wal_files);
  if (!s.ok()) {
    return s;
  }

  s = RetainProbableWalFiles(*wal_files, seq);
  if (!s.ok()) {
    return s;
  }
  iter->reset(new TransactionLogIteratorImpl(
      db_options_.wal_dir, &db_options_, read_options, env_options_, seq,
      std::move(wal_files), version_set));
  return (*iter)->status();
}

// 1. Go through all archived files and
//    a. if ttl is enabled, delete outdated files
//    b. if archive size limit is enabled, delete empty files,
//        compute file number and size.
// 2. If size limit is enabled:
//    a. compute how many files should be deleted
//    b. get sorted non-empty archived logs
//    c. delete what should be deleted
void WalManager::PurgeObsoleteWALFiles() {
  bool const ttl_enabled = db_options_.WAL_ttl_seconds > 0;
  bool const size_limit_enabled = db_options_.WAL_size_limit_MB > 0;
  if (!ttl_enabled && !size_limit_enabled) {
    return;
  }

  int64_t current_time;
  Status s = env_->GetCurrentTime(&current_time);
  if (!s.ok()) {
    Log(InfoLogLevel::ERROR_LEVEL, db_options_.info_log,
        "Can't get current time: %s", s.ToString().c_str());
    assert(false);
    return;
  }
  uint64_t const now_seconds = static_cast<uint64_t>(current_time);
  uint64_t const time_to_check = (ttl_enabled && !size_limit_enabled)
                                     ? db_options_.WAL_ttl_seconds / 2
                                     : kDefaultIntervalToDeleteObsoleteWAL;

  if (purge_wal_files_last_run_ + time_to_check > now_seconds) {
    return;
  }

  purge_wal_files_last_run_ = now_seconds;

  std::string archival_dir = ArchivalDirectory(db_options_.wal_dir);
  std::vector<std::string> files;
  s = env_->GetChildren(archival_dir, &files);
  if (!s.ok()) {
    Log(InfoLogLevel::ERROR_LEVEL, db_options_.info_log,
        "Can't get archive files: %s", s.ToString().c_str());
    assert(false);
    return;
  }

  size_t log_files_num = 0;
  uint64_t log_file_size = 0;

  for (auto& f : files) {
    uint64_t number;
    FileType type;
    if (ParseFileName(f, &number, &type) && type == kLogFile) {
      std::string const file_path = archival_dir + "/" + f;
      if (ttl_enabled) {
        uint64_t file_m_time;
        s = env_->GetFileModificationTime(file_path, &file_m_time);
        if (!s.ok()) {
          Log(InfoLogLevel::WARN_LEVEL, db_options_.info_log,
              "Can't get file mod time: %s: %s",
              file_path.c_str(), s.ToString().c_str());
          continue;
        }
        if (now_seconds - file_m_time > db_options_.WAL_ttl_seconds) {
          s = env_->DeleteFile(file_path);
          if (!s.ok()) {
            Log(InfoLogLevel::WARN_LEVEL, db_options_.info_log,
                "Can't delete file: %s: %s",
                file_path.c_str(), s.ToString().c_str());
            continue;
          } else {
            MutexLock l(&read_first_record_cache_mutex_);
            read_first_record_cache_.erase(number);
          }
          continue;
        }
      }

      if (size_limit_enabled) {
        uint64_t file_size;
        s = env_->GetFileSize(file_path, &file_size);
        if (!s.ok()) {
          Log(InfoLogLevel::ERROR_LEVEL, db_options_.info_log,
              "Unable to get file size: %s: %s",
              file_path.c_str(), s.ToString().c_str());
          return;
        } else {
          if (file_size > 0) {
            log_file_size = std::max(log_file_size, file_size);
            ++log_files_num;
          } else {
            s = env_->DeleteFile(file_path);
            if (!s.ok()) {
              Log(InfoLogLevel::WARN_LEVEL, db_options_.info_log,
                  "Unable to delete file: %s: %s",
                  file_path.c_str(), s.ToString().c_str());
              continue;
            } else {
              MutexLock l(&read_first_record_cache_mutex_);
              read_first_record_cache_.erase(number);
            }
          }
        }
      }
    }
  }

  if (0 == log_files_num || !size_limit_enabled) {
    return;
  }

  size_t const files_keep_num =
      db_options_.WAL_size_limit_MB * 1024 * 1024 / log_file_size;
  if (log_files_num <= files_keep_num) {
    return;
  }

  size_t files_del_num = log_files_num - files_keep_num;
  VectorLogPtr archived_logs;
  GetSortedWalsOfType(archival_dir, archived_logs, kArchivedLogFile);

  if (files_del_num > archived_logs.size()) {
    Log(InfoLogLevel::WARN_LEVEL, db_options_.info_log,
        "Trying to delete more archived log files than "
        "exist. Deleting all");
    files_del_num = archived_logs.size();
  }

  for (size_t i = 0; i < files_del_num; ++i) {
    std::string const file_path = archived_logs[i]->PathName();
    s = env_->DeleteFile(db_options_.wal_dir + "/" + file_path);
    if (!s.ok()) {
      Log(InfoLogLevel::WARN_LEVEL, db_options_.info_log,
          "Unable to delete file: %s: %s", file_path.c_str(),
          s.ToString().c_str());
      continue;
    } else {
      MutexLock l(&read_first_record_cache_mutex_);
      read_first_record_cache_.erase(archived_logs[i]->LogNumber());
    }
  }
}

void WalManager::ArchiveWALFile(const std::string& fname, uint64_t number) {
  auto archived_log_name = ArchivedLogFileName(db_options_.wal_dir, number);
  // The sync point below is used in (DBTest,TransactionLogIteratorRace)
  TEST_SYNC_POINT("WalManager::PurgeObsoleteFiles:1");
  Status s = env_->RenameFile(fname, archived_log_name);
  // The sync point below is used in (DBTest,TransactionLogIteratorRace)
  TEST_SYNC_POINT("WalManager::PurgeObsoleteFiles:2");
  Log(InfoLogLevel::INFO_LEVEL, db_options_.info_log,
      "Move log file %s to %s -- %s\n", fname.c_str(),
      archived_log_name.c_str(), s.ToString().c_str());
}

namespace {
struct CompareLogByPointer {
  bool operator()(const std::unique_ptr<LogFile>& a,
                  const std::unique_ptr<LogFile>& b) {
    LogFileImpl* a_impl = dynamic_cast<LogFileImpl*>(a.get());
    LogFileImpl* b_impl = dynamic_cast<LogFileImpl*>(b.get());
    return *a_impl < *b_impl;
  }
};
}

Status WalManager::GetSortedWalsOfType(const std::string& path,
                                       VectorLogPtr& log_files,
                                       WalFileType log_type) {
  std::vector<std::string> all_files;
  const Status status = env_->GetChildren(path, &all_files);
  if (!status.ok()) {
    return status;
  }
  log_files.reserve(all_files.size());
  for (const auto& f : all_files) {
    uint64_t number;
    FileType type;
    if (ParseFileName(f, &number, &type) && type == kLogFile) {
      SequenceNumber sequence;
      Status s = ReadFirstRecord(log_type, number, &sequence);
      if (!s.ok()) {
        return s;
      }
      if (sequence == 0) {
        // empty file
        continue;
      }

      // Reproduce the race condition where a log file is moved
      // to archived dir, between these two sync points, used in
      // (DBTest,TransactionLogIteratorRace)
      TEST_SYNC_POINT("WalManager::GetSortedWalsOfType:1");
      TEST_SYNC_POINT("WalManager::GetSortedWalsOfType:2");

      uint64_t size_bytes;
      s = env_->GetFileSize(LogFileName(path, number), &size_bytes);
      // re-try in case the alive log file has been moved to archive.
      std::string archived_file = ArchivedLogFileName(path, number);
      if (!s.ok() && log_type == kAliveLogFile &&
          env_->FileExists(archived_file).ok()) {
        s = env_->GetFileSize(archived_file, &size_bytes);
        if (!s.ok() && env_->FileExists(archived_file).IsNotFound()) {
          // oops, the file just got deleted from archived dir! move on
          s = Status::OK();
          continue;
        }
      }
      if (!s.ok()) {
        return s;
      }

      log_files.push_back(std::move(std::unique_ptr<LogFile>(
          new LogFileImpl(number, log_type, sequence, size_bytes))));
    }
  }
  CompareLogByPointer compare_log_files;
  std::sort(log_files.begin(), log_files.end(), compare_log_files);
  return status;
}

Status WalManager::RetainProbableWalFiles(VectorLogPtr& all_logs,
                                          const SequenceNumber target) {
  int64_t start = 0;  // signed to avoid overflow when target is < first file.
  int64_t end = static_cast<int64_t>(all_logs.size()) - 1;
  // Binary Search. avoid opening all files.
  while (end >= start) {
    int64_t mid = start + (end - start) / 2;  // Avoid overflow.
    SequenceNumber current_seq_num = all_logs.at(mid)->StartSequence();
    if (current_seq_num == target) {
      end = mid;
      break;
    } else if (current_seq_num < target) {
      start = mid + 1;
    } else {
      end = mid - 1;
    }
  }
  // end could be -ve.
  size_t start_index = std::max(static_cast<int64_t>(0), end);
  // The last wal file is always included
  all_logs.erase(all_logs.begin(), all_logs.begin() + start_index);
  return Status::OK();
}

Status WalManager::ReadFirstRecord(const WalFileType type,
                                   const uint64_t number,
                                   SequenceNumber* sequence) {
  *sequence = 0;
  if (type != kAliveLogFile && type != kArchivedLogFile) {
    Log(InfoLogLevel::ERROR_LEVEL, db_options_.info_log,
        "[WalManger] Unknown file type %s", ToString(type).c_str());
    return Status::NotSupported(
        "File Type Not Known " + ToString(type));
  }
  {
    MutexLock l(&read_first_record_cache_mutex_);
    auto itr = read_first_record_cache_.find(number);
    if (itr != read_first_record_cache_.end()) {
      *sequence = itr->second;
      return Status::OK();
    }
  }
  Status s;
  if (type == kAliveLogFile) {
    std::string fname = LogFileName(db_options_.wal_dir, number);
    s = ReadFirstLine(fname, sequence);
    if (env_->FileExists(fname).ok() && !s.ok()) {
      // return any error that is not caused by non-existing file
      return s;
    }
  }

  if (type == kArchivedLogFile || !s.ok()) {
    //  check if the file got moved to archive.
    std::string archived_file =
        ArchivedLogFileName(db_options_.wal_dir, number);
    s = ReadFirstLine(archived_file, sequence);
    // maybe the file was deleted from archive dir. If that's the case, return
    // Status::OK(). The caller with identify this as empty file because
    // *sequence == 0
    if (!s.ok() && env_->FileExists(archived_file).IsNotFound()) {
      return Status::OK();
    }
  }

  if (s.ok() && *sequence != 0) {
    MutexLock l(&read_first_record_cache_mutex_);
    read_first_record_cache_.insert({number, *sequence});
  }
  return s;
}

// the function returns status.ok() and sequence == 0 if the file exists, but is
// empty
Status WalManager::ReadFirstLine(const std::string& fname,
                                 SequenceNumber* sequence) {
  struct LogReporter : public log::Reader::Reporter {
    Env* env;
    Logger* info_log;
    const char* fname;

    Status* status;
    bool ignore_error;  // true if db_options_.paranoid_checks==false
    virtual void Corruption(size_t bytes, const Status& s) override {
      Log(InfoLogLevel::WARN_LEVEL, info_log,
          "[WalManager] %s%s: dropping %d bytes; %s",
          (this->ignore_error ? "(ignoring error) " : ""), fname,
          static_cast<int>(bytes), s.ToString().c_str());
      if (this->status->ok()) {
        // only keep the first error
        *this->status = s;
      }
    }
  };

  std::unique_ptr<SequentialFile> file;
  Status status = env_->NewSequentialFile(fname, &file, env_options_);
  unique_ptr<SequentialFileReader> file_reader(
      new SequentialFileReader(std::move(file)));

  if (!status.ok()) {
    return status;
  }

  LogReporter reporter;
  reporter.env = env_;
  reporter.info_log = db_options_.info_log.get();
  reporter.fname = fname.c_str();
  reporter.status = &status;
  reporter.ignore_error = !db_options_.paranoid_checks;
  log::Reader reader(std::move(file_reader), &reporter, true /*checksum*/,
                     0 /*initial_offset*/);
  std::string scratch;
  Slice record;

  if (reader.ReadRecord(&record, &scratch) &&
      (status.ok() || !db_options_.paranoid_checks)) {
    if (record.size() < 12) {
      reporter.Corruption(record.size(),
                          Status::Corruption("log record too small"));
      // TODO read record's till the first no corrupt entry?
    } else {
      WriteBatch batch;
      WriteBatchInternal::SetContents(&batch, record);
      *sequence = WriteBatchInternal::Sequence(&batch);
      return Status::OK();
    }
  }

  // ReadRecord returns false on EOF, which means that the log file is empty. we
  // return status.ok() in that case and set sequence number to 0
  *sequence = 0;
  return status;
}

#endif  // ROCKSDB_LITE
}  // namespace rocksdb
