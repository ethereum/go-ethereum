//  Copyright (c) 2013, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.
//
// Logger implementation that can be shared by all environments
// where enough posix functionality is available.

#pragma once
#include <list>

#include "db/filename.h"
#include "port/port.h"
#include "port/util_logger.h"

namespace rocksdb {

// Rolls the log file by size and/or time
class AutoRollLogger : public Logger {
 public:
  AutoRollLogger(Env* env, const std::string& dbname,
                 const std::string& db_log_dir, size_t log_max_size,
                 size_t log_file_time_to_roll,
                 const InfoLogLevel log_level = InfoLogLevel::INFO_LEVEL)
      : Logger(log_level),
        dbname_(dbname),
        db_log_dir_(db_log_dir),
        env_(env),
        status_(Status::OK()),
        kMaxLogFileSize(log_max_size),
        kLogFileTimeToRoll(log_file_time_to_roll),
        cached_now(static_cast<uint64_t>(env_->NowMicros() * 1e-6)),
        ctime_(cached_now),
        cached_now_access_count(0),
        call_NowMicros_every_N_records_(100),
        mutex_() {
    env->GetAbsolutePath(dbname, &db_absolute_path_);
    log_fname_ = InfoLogFileName(dbname_, db_absolute_path_, db_log_dir_);
    RollLogFile();
    ResetLogger();
  }

  using Logger::Logv;
  void Logv(const char* format, va_list ap) override;

  // Write a header entry to the log. All header information will be written
  // again every time the log rolls over.
  virtual void LogHeader(const char* format, va_list ap) override;

  // check if the logger has encountered any problem.
  Status GetStatus() {
    return status_;
  }

  size_t GetLogFileSize() const override { return logger_->GetLogFileSize(); }

  void Flush() override {
    if (logger_) {
      logger_->Flush();
    }
  }

  virtual ~AutoRollLogger() {
  }

  void SetCallNowMicrosEveryNRecords(uint64_t call_NowMicros_every_N_records) {
    call_NowMicros_every_N_records_ = call_NowMicros_every_N_records;
  }

  // Expose the log file path for testing purpose
  std::string TEST_log_fname() const {
    return log_fname_;
  }

 private:
  bool LogExpired();
  Status ResetLogger();
  void RollLogFile();
  // Log message to logger without rolling
  void LogInternal(const char* format, ...);
  // Serialize the va_list to a string
  std::string ValistToString(const char* format, va_list args) const;
  // Write the logs marked as headers to the new log file
  void WriteHeaderInfo();

  std::string log_fname_; // Current active info log's file name.
  std::string dbname_;
  std::string db_log_dir_;
  std::string db_absolute_path_;
  Env* env_;
  std::shared_ptr<Logger> logger_;
  // current status of the logger
  Status status_;
  const size_t kMaxLogFileSize;
  const size_t kLogFileTimeToRoll;
  // header information
  std::list<std::string> headers_;
  // to avoid frequent env->NowMicros() calls, we cached the current time
  uint64_t cached_now;
  uint64_t ctime_;
  uint64_t cached_now_access_count;
  uint64_t call_NowMicros_every_N_records_;
  port::Mutex mutex_;
};

// Facade to craete logger automatically
Status CreateLoggerFromOptions(
    const std::string& dbname,
    const std::string& db_log_dir,
    Env* env,
    const DBOptions& options,
    std::shared_ptr<Logger>* logger);

}  // namespace rocksdb
