//  Copyright (c) 2013, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.

#ifndef ROCKSDB_LITE

#include <map>
#include <string>

#include "rocksdb/cache.h"
#include "rocksdb/write_batch.h"

#include "db/wal_manager.h"
#include "db/log_writer.h"
#include "db/column_family.h"
#include "db/version_set.h"
#include "db/writebuffer.h"
#include "util/file_reader_writer.h"
#include "util/mock_env.h"
#include "util/string_util.h"
#include "util/testharness.h"
#include "util/testutil.h"
#include "table/mock_table.h"
#include "db/db_impl.h"

namespace rocksdb {

// TODO(icanadi) mock out VersionSet
// TODO(icanadi) move other WalManager-specific tests from db_test here
class WalManagerTest : public testing::Test {
 public:
  WalManagerTest()
      : env_(new MockEnv(Env::Default())),
        dbname_(test::TmpDir() + "/wal_manager_test"),
        table_cache_(NewLRUCache(50000, 16)),
        write_buffer_(db_options_.db_write_buffer_size),
        current_log_number_(0) {
    DestroyDB(dbname_, Options());
  }

  void Init() {
    ASSERT_OK(env_->CreateDirIfMissing(dbname_));
    ASSERT_OK(env_->CreateDirIfMissing(ArchivalDirectory(dbname_)));
    db_options_.db_paths.emplace_back(dbname_,
                                      std::numeric_limits<uint64_t>::max());
    db_options_.wal_dir = dbname_;
    db_options_.env = env_.get();

    versions_.reset(new VersionSet(dbname_, &db_options_, env_options_,
                                   table_cache_.get(), &write_buffer_,
                                   &write_controller_));

    wal_manager_.reset(new WalManager(db_options_, env_options_));
  }

  void Reopen() {
    wal_manager_.reset(new WalManager(db_options_, env_options_));
  }

  // NOT thread safe
  void Put(const std::string& key, const std::string& value) {
    assert(current_log_writer_.get() != nullptr);
    uint64_t seq =  versions_->LastSequence() + 1;
    WriteBatch batch;
    batch.Put(key, value);
    WriteBatchInternal::SetSequence(&batch, seq);
    current_log_writer_->AddRecord(WriteBatchInternal::Contents(&batch));
    versions_->SetLastSequence(seq);
  }

  // NOT thread safe
  void RollTheLog(bool archived) {
    current_log_number_++;
    std::string fname = ArchivedLogFileName(dbname_, current_log_number_);
    unique_ptr<WritableFile> file;
    ASSERT_OK(env_->NewWritableFile(fname, &file, env_options_));
    unique_ptr<WritableFileWriter> file_writer(
        new WritableFileWriter(std::move(file), env_options_));
    current_log_writer_.reset(new log::Writer(std::move(file_writer)));
  }

  void CreateArchiveLogs(int num_logs, int entries_per_log) {
    for (int i = 1; i <= num_logs; ++i) {
      RollTheLog(true);
      for (int k = 0; k < entries_per_log; ++k) {
        Put(ToString(k), std::string(1024, 'a'));
      }
    }
  }

  std::unique_ptr<TransactionLogIterator> OpenTransactionLogIter(
      const SequenceNumber seq) {
    unique_ptr<TransactionLogIterator> iter;
    Status status = wal_manager_->GetUpdatesSince(
        seq, &iter, TransactionLogIterator::ReadOptions(), versions_.get());
    EXPECT_OK(status);
    return std::move(iter);
  }

  std::unique_ptr<MockEnv> env_;
  std::string dbname_;
  WriteController write_controller_;
  EnvOptions env_options_;
  std::shared_ptr<Cache> table_cache_;
  DBOptions db_options_;
  WriteBuffer write_buffer_;
  std::unique_ptr<VersionSet> versions_;
  std::unique_ptr<WalManager> wal_manager_;

  std::unique_ptr<log::Writer> current_log_writer_;
  uint64_t current_log_number_;
};

TEST_F(WalManagerTest, ReadFirstRecordCache) {
  Init();
  std::string path = dbname_ + "/000001.log";
  unique_ptr<WritableFile> file;
  ASSERT_OK(env_->NewWritableFile(path, &file, EnvOptions()));

  SequenceNumber s;
  ASSERT_OK(wal_manager_->TEST_ReadFirstLine(path, &s));
  ASSERT_EQ(s, 0U);

  ASSERT_OK(wal_manager_->TEST_ReadFirstRecord(kAliveLogFile, 1, &s));
  ASSERT_EQ(s, 0U);

  unique_ptr<WritableFileWriter> file_writer(
      new WritableFileWriter(std::move(file), EnvOptions()));
  log::Writer writer(std::move(file_writer));
  WriteBatch batch;
  batch.Put("foo", "bar");
  WriteBatchInternal::SetSequence(&batch, 10);
  writer.AddRecord(WriteBatchInternal::Contents(&batch));

  // TODO(icanadi) move SpecialEnv outside of db_test, so we can reuse it here.
  // Waiting for lei to finish with db_test
  // env_->count_sequential_reads_ = true;
  // sequential_read_counter_ sanity test
  // ASSERT_EQ(env_->sequential_read_counter_.Read(), 0);

  ASSERT_OK(wal_manager_->TEST_ReadFirstRecord(kAliveLogFile, 1, &s));
  ASSERT_EQ(s, 10U);
  // did a read
  // TODO(icanadi) move SpecialEnv outside of db_test, so we can reuse it here
  // ASSERT_EQ(env_->sequential_read_counter_.Read(), 1);

  ASSERT_OK(wal_manager_->TEST_ReadFirstRecord(kAliveLogFile, 1, &s));
  ASSERT_EQ(s, 10U);
  // no new reads since the value is cached
  // TODO(icanadi) move SpecialEnv outside of db_test, so we can reuse it here
  // ASSERT_EQ(env_->sequential_read_counter_.Read(), 1);
}

namespace {
uint64_t GetLogDirSize(std::string dir_path, Env* env) {
  uint64_t dir_size = 0;
  std::vector<std::string> files;
  env->GetChildren(dir_path, &files);
  for (auto& f : files) {
    uint64_t number;
    FileType type;
    if (ParseFileName(f, &number, &type) && type == kLogFile) {
      std::string const file_path = dir_path + "/" + f;
      uint64_t file_size;
      env->GetFileSize(file_path, &file_size);
      dir_size += file_size;
    }
  }
  return dir_size;
}
std::vector<std::uint64_t> ListSpecificFiles(
    Env* env, const std::string& path, const FileType expected_file_type) {
  std::vector<std::string> files;
  std::vector<uint64_t> file_numbers;
  env->GetChildren(path, &files);
  uint64_t number;
  FileType type;
  for (size_t i = 0; i < files.size(); ++i) {
    if (ParseFileName(files[i], &number, &type)) {
      if (type == expected_file_type) {
        file_numbers.push_back(number);
      }
    }
  }
  return std::move(file_numbers);
}

int CountRecords(TransactionLogIterator* iter) {
  int count = 0;
  SequenceNumber lastSequence = 0;
  BatchResult res;
  while (iter->Valid()) {
    res = iter->GetBatch();
    EXPECT_TRUE(res.sequence > lastSequence);
    ++count;
    lastSequence = res.sequence;
    EXPECT_OK(iter->status());
    iter->Next();
  }
  return count;
}
}  // namespace

TEST_F(WalManagerTest, WALArchivalSizeLimit) {
  db_options_.WAL_ttl_seconds = 0;
  db_options_.WAL_size_limit_MB = 1000;
  Init();

  // TEST : Create WalManager with huge size limit and no ttl.
  // Create some archived files and call PurgeObsoleteWALFiles().
  // Count the archived log files that survived.
  // Assert that all of them did.
  // Change size limit. Re-open WalManager.
  // Assert that archive is not greater than WAL_size_limit_MB after
  // PurgeObsoleteWALFiles()
  // Set ttl and time_to_check_ to small values. Re-open db.
  // Assert that there are no archived logs left.

  std::string archive_dir = ArchivalDirectory(dbname_);
  CreateArchiveLogs(20, 5000);

  std::vector<std::uint64_t> log_files =
      ListSpecificFiles(env_.get(), archive_dir, kLogFile);
  ASSERT_EQ(log_files.size(), 20U);

  db_options_.WAL_size_limit_MB = 8;
  Reopen();
  wal_manager_->PurgeObsoleteWALFiles();

  uint64_t archive_size = GetLogDirSize(archive_dir, env_.get());
  ASSERT_TRUE(archive_size <= db_options_.WAL_size_limit_MB * 1024 * 1024);

  db_options_.WAL_ttl_seconds = 1;
  env_->FakeSleepForMicroseconds(2 * 1000 * 1000);
  Reopen();
  wal_manager_->PurgeObsoleteWALFiles();

  log_files = ListSpecificFiles(env_.get(), archive_dir, kLogFile);
  ASSERT_TRUE(log_files.empty());
}

TEST_F(WalManagerTest, WALArchivalTtl) {
  db_options_.WAL_ttl_seconds = 1000;
  Init();

  // TEST : Create WalManager with a ttl and no size limit.
  // Create some archived log files and call PurgeObsoleteWALFiles().
  // Assert that files are not deleted
  // Reopen db with small ttl.
  // Assert that all archived logs was removed.

  std::string archive_dir = ArchivalDirectory(dbname_);
  CreateArchiveLogs(20, 5000);

  std::vector<uint64_t> log_files =
      ListSpecificFiles(env_.get(), archive_dir, kLogFile);
  ASSERT_GT(log_files.size(), 0U);

  db_options_.WAL_ttl_seconds = 1;
  env_->FakeSleepForMicroseconds(3 * 1000 * 1000);
  Reopen();
  wal_manager_->PurgeObsoleteWALFiles();

  log_files = ListSpecificFiles(env_.get(), archive_dir, kLogFile);
  ASSERT_TRUE(log_files.empty());
}

TEST_F(WalManagerTest, TransactionLogIteratorMoveOverZeroFiles) {
  Init();
  RollTheLog(false);
  Put("key1", std::string(1024, 'a'));
  // Create a zero record WAL file.
  RollTheLog(false);
  RollTheLog(false);

  Put("key2", std::string(1024, 'a'));

  auto iter = OpenTransactionLogIter(0);
  ASSERT_EQ(2, CountRecords(iter.get()));
}

TEST_F(WalManagerTest, TransactionLogIteratorJustEmptyFile) {
  Init();
  RollTheLog(false);
  auto iter = OpenTransactionLogIter(0);
  // Check that an empty iterator is returned
  ASSERT_TRUE(!iter->Valid());
}

}  // namespace rocksdb

int main(int argc, char** argv) {
  ::testing::InitGoogleTest(&argc, argv);
  return RUN_ALL_TESTS();
}

#else
#include <stdio.h>

int main(int argc, char** argv) {
  fprintf(stderr, "SKIPPED as WalManager is not supported in ROCKSDB_LITE\n");
  return 0;
}

#endif  // !ROCKSDB_LITE
