//  Copyright (c) 2013, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.
//
#include <string>
#include <vector>
#include <cmath>
#include <iostream>
#include <fstream>
#include <iterator>
#include <algorithm>
#include "util/testharness.h"
#include "util/auto_roll_logger.h"
#include "rocksdb/db.h"
#include <sys/stat.h>
#include <errno.h>

using namespace std;

namespace rocksdb {

class AutoRollLoggerTest : public testing::Test {
 public:
  static void InitTestDb() {
#ifdef OS_WIN
    // Replace all slashes in the path so windows CompSpec does not
    // become confused
    std::string testDir(kTestDir);
    std::replace_if(testDir.begin(), testDir.end(),
                    [](char ch) { return ch == '/'; }, '\\');
    std::string deleteCmd = "if exist " + testDir + " rd /s /q " + testDir;
#else
    std::string deleteCmd = "rm -rf " + kTestDir;
#endif
    ASSERT_TRUE(system(deleteCmd.c_str()) == 0);
    Env::Default()->CreateDir(kTestDir);
  }

  void RollLogFileBySizeTest(AutoRollLogger* logger,
                             size_t log_max_size,
                             const string& log_message);
  uint64_t RollLogFileByTimeTest(AutoRollLogger* logger,
                                 size_t time,
                                 const string& log_message);

  static const string kSampleMessage;
  static const string kTestDir;
  static const string kLogFile;
  static Env* env;
};

const string AutoRollLoggerTest::kSampleMessage(
    "this is the message to be written to the log file!!");
const string AutoRollLoggerTest::kTestDir(test::TmpDir() + "/db_log_test");
const string AutoRollLoggerTest::kLogFile(test::TmpDir() + "/db_log_test/LOG");
Env* AutoRollLoggerTest::env = Env::Default();

// In this test we only want to Log some simple log message with
// no format. LogMessage() provides such a simple interface and
// avoids the [format-security] warning which occurs when you
// call Log(logger, log_message) directly.
namespace {
void LogMessage(Logger* logger, const char* message) {
  Log(logger, "%s", message);
}

void LogMessage(const InfoLogLevel log_level, Logger* logger,
                const char* message) {
  Log(log_level, logger, "%s", message);
}
}  // namespace

namespace {
void GetFileCreateTime(const std::string& fname, uint64_t* file_ctime) {
  struct stat s;
  if (stat(fname.c_str(), &s) != 0) {
    *file_ctime = (uint64_t)0;
  }
  *file_ctime = static_cast<uint64_t>(s.st_ctime);
}
}  // namespace

void AutoRollLoggerTest::RollLogFileBySizeTest(AutoRollLogger* logger,
                                               size_t log_max_size,
                                               const string& log_message) {
  logger->SetInfoLogLevel(InfoLogLevel::INFO_LEVEL);
  // measure the size of each message, which is supposed
  // to be equal or greater than log_message.size()
  LogMessage(logger, log_message.c_str());
  size_t message_size = logger->GetLogFileSize();
  size_t current_log_size = message_size;

  // Test the cases when the log file will not be rolled.
  while (current_log_size + message_size < log_max_size) {
    LogMessage(logger, log_message.c_str());
    current_log_size += message_size;
    ASSERT_EQ(current_log_size, logger->GetLogFileSize());
  }

  // Now the log file will be rolled
  LogMessage(logger, log_message.c_str());
  // Since rotation is checked before actual logging, we need to
  // trigger the rotation by logging another message.
  LogMessage(logger, log_message.c_str());

  ASSERT_TRUE(message_size == logger->GetLogFileSize());
}

uint64_t AutoRollLoggerTest::RollLogFileByTimeTest(
    AutoRollLogger* logger, size_t time, const string& log_message) {
  uint64_t expected_create_time;
  uint64_t actual_create_time;
  uint64_t total_log_size;
  EXPECT_OK(env->GetFileSize(kLogFile, &total_log_size));
  GetFileCreateTime(kLogFile, &expected_create_time);
  logger->SetCallNowMicrosEveryNRecords(0);

  // -- Write to the log for several times, which is supposed
  // to be finished before time.
  for (int i = 0; i < 10; ++i) {
     LogMessage(logger, log_message.c_str());
     EXPECT_OK(logger->GetStatus());
     // Make sure we always write to the same log file (by
     // checking the create time);
     GetFileCreateTime(kLogFile, &actual_create_time);

     // Also make sure the log size is increasing.
     EXPECT_EQ(expected_create_time, actual_create_time);
     EXPECT_GT(logger->GetLogFileSize(), total_log_size);
     total_log_size = logger->GetLogFileSize();
  }

  // -- Make the log file expire
#ifdef OS_WIN
  Sleep(static_cast<unsigned int>(time) * 1000);
#else
  sleep(static_cast<unsigned int>(time));
#endif
  LogMessage(logger, log_message.c_str());

  // At this time, the new log file should be created.
  GetFileCreateTime(kLogFile, &actual_create_time);
  EXPECT_GT(actual_create_time, expected_create_time);
  EXPECT_LT(logger->GetLogFileSize(), total_log_size);
  expected_create_time = actual_create_time;

  return expected_create_time;
}

TEST_F(AutoRollLoggerTest, RollLogFileBySize) {
    InitTestDb();
    size_t log_max_size = 1024 * 5;

    AutoRollLogger logger(Env::Default(), kTestDir, "", log_max_size, 0);

    RollLogFileBySizeTest(&logger, log_max_size,
                          kSampleMessage + ":RollLogFileBySize");
}

TEST_F(AutoRollLoggerTest, RollLogFileByTime) {
    size_t time = 2;
    size_t log_size = 1024 * 5;

    InitTestDb();
    // -- Test the existence of file during the server restart.
    ASSERT_EQ(Status::NotFound(), env->FileExists(kLogFile));
    AutoRollLogger logger(Env::Default(), kTestDir, "", log_size, time);
    ASSERT_OK(env->FileExists(kLogFile));

    RollLogFileByTimeTest(&logger, time, kSampleMessage + ":RollLogFileByTime");
}

TEST_F(AutoRollLoggerTest, OpenLogFilesMultipleTimesWithOptionLog_max_size) {
  // If only 'log_max_size' options is specified, then every time
  // when rocksdb is restarted, a new empty log file will be created.
  InitTestDb();
  // WORKAROUND:
  // avoid complier's complaint of "comparison between signed
  // and unsigned integer expressions" because literal 0 is
  // treated as "singed".
  size_t kZero = 0;
  size_t log_size = 1024;

  AutoRollLogger* logger = new AutoRollLogger(
    Env::Default(), kTestDir, "", log_size, 0);

  LogMessage(logger, kSampleMessage.c_str());
  ASSERT_GT(logger->GetLogFileSize(), kZero);
  delete logger;

  // reopens the log file and an empty log file will be created.
  logger = new AutoRollLogger(
    Env::Default(), kTestDir, "", log_size, 0);
  ASSERT_EQ(logger->GetLogFileSize(), kZero);
  delete logger;
}

TEST_F(AutoRollLoggerTest, CompositeRollByTimeAndSizeLogger) {
  size_t time = 2, log_max_size = 1024 * 5;

  InitTestDb();

  AutoRollLogger logger(Env::Default(), kTestDir, "", log_max_size, time);

  // Test the ability to roll by size
  RollLogFileBySizeTest(
      &logger, log_max_size,
      kSampleMessage + ":CompositeRollByTimeAndSizeLogger");

  // Test the ability to roll by Time
  RollLogFileByTimeTest( &logger, time,
      kSampleMessage + ":CompositeRollByTimeAndSizeLogger");
}

#ifndef OS_WIN
// TODO: does not build for Windows because of PosixLogger use below. Need to
// port
TEST_F(AutoRollLoggerTest, CreateLoggerFromOptions) {
  DBOptions options;
  shared_ptr<Logger> logger;

  // Normal logger
  ASSERT_OK(CreateLoggerFromOptions(kTestDir, "", env, options, &logger));
  ASSERT_TRUE(dynamic_cast<PosixLogger*>(logger.get()));

  // Only roll by size
  InitTestDb();
  options.max_log_file_size = 1024;
  ASSERT_OK(CreateLoggerFromOptions(kTestDir, "", env, options, &logger));
  AutoRollLogger* auto_roll_logger =
    dynamic_cast<AutoRollLogger*>(logger.get());
  ASSERT_TRUE(auto_roll_logger);
  RollLogFileBySizeTest(
      auto_roll_logger, options.max_log_file_size,
      kSampleMessage + ":CreateLoggerFromOptions - size");

  // Only roll by Time
  InitTestDb();
  options.max_log_file_size = 0;
  options.log_file_time_to_roll = 2;
  ASSERT_OK(CreateLoggerFromOptions(kTestDir, "", env, options, &logger));
  auto_roll_logger =
    dynamic_cast<AutoRollLogger*>(logger.get());
  RollLogFileByTimeTest(
      auto_roll_logger, options.log_file_time_to_roll,
      kSampleMessage + ":CreateLoggerFromOptions - time");

  // roll by both Time and size
  InitTestDb();
  options.max_log_file_size = 1024 * 5;
  options.log_file_time_to_roll = 2;
  ASSERT_OK(CreateLoggerFromOptions(kTestDir, "", env, options, &logger));
  auto_roll_logger =
    dynamic_cast<AutoRollLogger*>(logger.get());
  RollLogFileBySizeTest(
      auto_roll_logger, options.max_log_file_size,
      kSampleMessage + ":CreateLoggerFromOptions - both");
  RollLogFileByTimeTest(
      auto_roll_logger, options.log_file_time_to_roll,
      kSampleMessage + ":CreateLoggerFromOptions - both");
}
#endif

TEST_F(AutoRollLoggerTest, InfoLogLevel) {
  InitTestDb();

  size_t log_size = 8192;
  size_t log_lines = 0;
  // an extra-scope to force the AutoRollLogger to flush the log file when it
  // becomes out of scope.
  {
    AutoRollLogger logger(Env::Default(), kTestDir, "", log_size, 0);
    for (int log_level = InfoLogLevel::HEADER_LEVEL;
         log_level >= InfoLogLevel::DEBUG_LEVEL; log_level--) {
      logger.SetInfoLogLevel((InfoLogLevel)log_level);
      for (int log_type = InfoLogLevel::DEBUG_LEVEL;
           log_type <= InfoLogLevel::HEADER_LEVEL; log_type++) {
        // log messages with log level smaller than log_level will not be
        // logged.
        LogMessage((InfoLogLevel)log_type, &logger, kSampleMessage.c_str());
      }
      log_lines += InfoLogLevel::HEADER_LEVEL - log_level + 1;
    }
    for (int log_level = InfoLogLevel::HEADER_LEVEL;
         log_level >= InfoLogLevel::DEBUG_LEVEL; log_level--) {
      logger.SetInfoLogLevel((InfoLogLevel)log_level);

      // again, messages with level smaller than log_level will not be logged.
      Log(InfoLogLevel::HEADER_LEVEL, &logger, "%s", kSampleMessage.c_str());
      Debug(&logger, "%s", kSampleMessage.c_str());
      Info(&logger, "%s", kSampleMessage.c_str());
      Warn(&logger, "%s", kSampleMessage.c_str());
      Error(&logger, "%s", kSampleMessage.c_str());
      Fatal(&logger, "%s", kSampleMessage.c_str());
      log_lines += InfoLogLevel::HEADER_LEVEL - log_level + 1;
    }
  }
  std::ifstream inFile(AutoRollLoggerTest::kLogFile.c_str());
  size_t lines = std::count(std::istreambuf_iterator<char>(inFile),
                         std::istreambuf_iterator<char>(), '\n');
  ASSERT_EQ(log_lines, lines);
  inFile.close();
}

// Test the logger Header function for roll over logs
// We expect the new logs creates as roll over to carry the headers specified
static std::vector<string> GetOldFileNames(const string& path) {
  std::vector<string> ret;

  const string dirname = path.substr(/*start=*/ 0, path.find_last_of("/"));
  const string fname = path.substr(path.find_last_of("/") + 1);

  std::vector<string> children;
  Env::Default()->GetChildren(dirname, &children);

  // We know that the old log files are named [path]<something>
  // Return all entities that match the pattern
  for (auto& child : children) {
    if (fname != child && child.find(fname) == 0) {
      ret.push_back(dirname + "/" + child);
    }
  }

  return ret;
}

// Return the number of lines where a given pattern was found in the file
static size_t GetLinesCount(const string& fname, const string& pattern) {
  stringstream ssbuf;
  string line;
  size_t count = 0;

  ifstream inFile(fname.c_str());
  ssbuf << inFile.rdbuf();

  while (getline(ssbuf, line)) {
    if (line.find(pattern) != std::string::npos) {
      count++;
    }
  }

  return count;
}

TEST_F(AutoRollLoggerTest, LogHeaderTest) {
  static const size_t MAX_HEADERS = 10;
  static const size_t LOG_MAX_SIZE = 1024 * 5;
  static const std::string HEADER_STR = "Log header line";

  // test_num == 0 -> standard call to Header()
  // test_num == 1 -> call to Log() with InfoLogLevel::HEADER_LEVEL
  for (int test_num = 0; test_num < 2; test_num++) {

    InitTestDb();

    AutoRollLogger logger(Env::Default(), kTestDir, /*db_log_dir=*/ "",
                          LOG_MAX_SIZE, /*log_file_time_to_roll=*/ 0);

    if (test_num == 0) {
      // Log some headers explicitly using Header()
      for (size_t i = 0; i < MAX_HEADERS; i++) {
        Header(&logger, "%s %d", HEADER_STR.c_str(), i);
      }
    } else if (test_num == 1) {
      // HEADER_LEVEL should make this behave like calling Header()
      for (size_t i = 0; i < MAX_HEADERS; i++) {
        Log(InfoLogLevel::HEADER_LEVEL, &logger, "%s %d",
            HEADER_STR.c_str(), i);
      }
    }

    const string newfname = logger.TEST_log_fname();

    // Log enough data to cause a roll over
    int i = 0;
    for (size_t iter = 0; iter < 2; iter++) {
      while (logger.GetLogFileSize() < LOG_MAX_SIZE) {
        Info(&logger, (kSampleMessage + ":LogHeaderTest line %d").c_str(), i);
        ++i;
      }

      Info(&logger, "Rollover");
    }

    // Flush the log for the latest file
    LogFlush(&logger);

    const auto oldfiles = GetOldFileNames(newfname);

    ASSERT_EQ(oldfiles.size(), (size_t) 2);

    for (auto& oldfname : oldfiles) {
      // verify that the files rolled over
      ASSERT_NE(oldfname, newfname);
      // verify that the old log contains all the header logs
      ASSERT_EQ(GetLinesCount(oldfname, HEADER_STR), MAX_HEADERS);
    }
  }
}

TEST_F(AutoRollLoggerTest, LogFileExistence) {
  rocksdb::DB* db;
  rocksdb::Options options;
  string deleteCmd = "rm -rf " + kTestDir;
  ASSERT_EQ(system(deleteCmd.c_str()), 0);
  options.max_log_file_size = 100 * 1024 * 1024;
  options.create_if_missing = true;
  ASSERT_OK(rocksdb::DB::Open(options, kTestDir, &db));
  ASSERT_OK(env->FileExists(kLogFile));
  delete db;
}

}  // namespace rocksdb

int main(int argc, char** argv) {
  ::testing::InitGoogleTest(&argc, argv);
  return RUN_ALL_TESTS();
}
