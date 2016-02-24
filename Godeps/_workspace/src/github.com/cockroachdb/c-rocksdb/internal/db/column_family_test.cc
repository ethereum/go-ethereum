//  Copyright (c) 2013, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.
//
// Copyright (c) 2011 The LevelDB Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file. See the AUTHORS file for names of contributors.

#include <algorithm>
#include <vector>
#include <string>
#include <thread>

#include "db/db_impl.h"
#include "rocksdb/db.h"
#include "rocksdb/env.h"
#include "rocksdb/iterator.h"
#include "util/string_util.h"
#include "util/testharness.h"
#include "util/testutil.h"
#include "util/coding.h"
#include "util/sync_point.h"
#include "utilities/merge_operators.h"

namespace rocksdb {

namespace {
std::string RandomString(Random* rnd, int len) {
  std::string r;
  test::RandomString(rnd, len, &r);
  return r;
}
}  // anonymous namespace

// counts how many operations were performed
class EnvCounter : public EnvWrapper {
 public:
  explicit EnvCounter(Env* base)
      : EnvWrapper(base), num_new_writable_file_(0) {}
  int GetNumberOfNewWritableFileCalls() {
    return num_new_writable_file_;
  }
  Status NewWritableFile(const std::string& f, unique_ptr<WritableFile>* r,
                         const EnvOptions& soptions) override {
    ++num_new_writable_file_;
    return EnvWrapper::NewWritableFile(f, r, soptions);
  }

 private:
  int num_new_writable_file_;
};

class ColumnFamilyTest : public testing::Test {
 public:
  ColumnFamilyTest() : rnd_(139) {
    env_ = new EnvCounter(Env::Default());
    dbname_ = test::TmpDir() + "/column_family_test";
    db_options_.create_if_missing = true;
    db_options_.env = env_;
    DestroyDB(dbname_, Options(db_options_, column_family_options_));
  }

  ~ColumnFamilyTest() {
    delete env_;
  }

  void Close() {
    for (auto h : handles_) {
      delete h;
    }
    handles_.clear();
    names_.clear();
    delete db_;
    db_ = nullptr;
  }

  Status TryOpen(std::vector<std::string> cf,
                 std::vector<ColumnFamilyOptions> options = {}) {
    std::vector<ColumnFamilyDescriptor> column_families;
    names_.clear();
    for (size_t i = 0; i < cf.size(); ++i) {
      column_families.push_back(ColumnFamilyDescriptor(
          cf[i], options.size() == 0 ? column_family_options_ : options[i]));
      names_.push_back(cf[i]);
    }
    return DB::Open(db_options_, dbname_, column_families, &handles_, &db_);
  }

  Status OpenReadOnly(std::vector<std::string> cf,
                         std::vector<ColumnFamilyOptions> options = {}) {
    std::vector<ColumnFamilyDescriptor> column_families;
    names_.clear();
    for (size_t i = 0; i < cf.size(); ++i) {
      column_families.push_back(ColumnFamilyDescriptor(
          cf[i], options.size() == 0 ? column_family_options_ : options[i]));
      names_.push_back(cf[i]);
    }
    return DB::OpenForReadOnly(db_options_, dbname_, column_families, &handles_,
                               &db_);
  }

#ifndef ROCKSDB_LITE  // ReadOnlyDB is not supported
  void AssertOpenReadOnly(std::vector<std::string> cf,
                    std::vector<ColumnFamilyOptions> options = {}) {
    ASSERT_OK(OpenReadOnly(cf, options));
  }
#endif  // !ROCKSDB_LITE


  void Open(std::vector<std::string> cf,
            std::vector<ColumnFamilyOptions> options = {}) {
    ASSERT_OK(TryOpen(cf, options));
  }

  void Open() {
    Open({"default"});
  }

  DBImpl* dbfull() { return reinterpret_cast<DBImpl*>(db_); }

  int GetProperty(int cf, std::string property) {
    std::string value;
    EXPECT_TRUE(dbfull()->GetProperty(handles_[cf], property, &value));
#ifndef CYGWIN
    return std::stoi(value);
#else
    return std::strtol(value.c_str(), 0 /* off */, 10 /* base */);
#endif
  }

  void Destroy() {
    for (auto h : handles_) {
      delete h;
    }
    handles_.clear();
    names_.clear();
    delete db_;
    db_ = nullptr;
    ASSERT_OK(DestroyDB(dbname_, Options(db_options_, column_family_options_)));
  }

  void CreateColumnFamilies(
      const std::vector<std::string>& cfs,
      const std::vector<ColumnFamilyOptions> options = {}) {
    int cfi = static_cast<int>(handles_.size());
    handles_.resize(cfi + cfs.size());
    names_.resize(cfi + cfs.size());
    for (size_t i = 0; i < cfs.size(); ++i) {
      ASSERT_OK(db_->CreateColumnFamily(
          options.size() == 0 ? column_family_options_ : options[i], cfs[i],
          &handles_[cfi]));
      names_[cfi] = cfs[i];
      cfi++;
    }
  }

  void Reopen(const std::vector<ColumnFamilyOptions> options = {}) {
    std::vector<std::string> names;
    for (auto name : names_) {
      if (name != "") {
        names.push_back(name);
      }
    }
    Close();
    assert(options.size() == 0 || names.size() == options.size());
    Open(names, options);
  }

  void CreateColumnFamiliesAndReopen(const std::vector<std::string>& cfs) {
    CreateColumnFamilies(cfs);
    Reopen();
  }

  void DropColumnFamilies(const std::vector<int>& cfs) {
    for (auto cf : cfs) {
      ASSERT_OK(db_->DropColumnFamily(handles_[cf]));
      delete handles_[cf];
      handles_[cf] = nullptr;
      names_[cf] = "";
    }
  }

  void PutRandomData(int cf, int num, int key_value_size) {
    for (int i = 0; i < num; ++i) {
      // 10 bytes for key, rest is value
      ASSERT_OK(Put(cf, test::RandomKey(&rnd_, 10),
                    RandomString(&rnd_, key_value_size - 10)));
    }
  }

  void WaitForFlush(int cf) {
#ifndef ROCKSDB_LITE  // TEST functions are not supported in lite
    ASSERT_OK(dbfull()->TEST_WaitForFlushMemTable(handles_[cf]));
#endif  // !ROCKSDB_LITE
  }

  void WaitForCompaction() {
#ifndef ROCKSDB_LITE  // TEST functions are not supported in lite
    ASSERT_OK(dbfull()->TEST_WaitForCompact());
#endif  // !ROCKSDB_LITE
  }

  uint64_t MaxTotalInMemoryState() {
#ifndef ROCKSDB_LITE
    return dbfull()->TEST_MaxTotalInMemoryState();
#else
    return 0;
#endif  // !ROCKSDB_LITE
  }

  void AssertMaxTotalInMemoryState(uint64_t value) {
    ASSERT_EQ(value, MaxTotalInMemoryState());
  }

  Status Put(int cf, const std::string& key, const std::string& value) {
    return db_->Put(WriteOptions(), handles_[cf], Slice(key), Slice(value));
  }
  Status Merge(int cf, const std::string& key, const std::string& value) {
    return db_->Merge(WriteOptions(), handles_[cf], Slice(key), Slice(value));
  }
  Status Flush(int cf) {
    return db_->Flush(FlushOptions(), handles_[cf]);
  }

  std::string Get(int cf, const std::string& key) {
    ReadOptions options;
    options.verify_checksums = true;
    std::string result;
    Status s = db_->Get(options, handles_[cf], Slice(key), &result);
    if (s.IsNotFound()) {
      result = "NOT_FOUND";
    } else if (!s.ok()) {
      result = s.ToString();
    }
    return result;
  }

  void CompactAll(int cf) {
    ASSERT_OK(db_->CompactRange(CompactRangeOptions(), handles_[cf], nullptr,
                                nullptr));
  }

  void Compact(int cf, const Slice& start, const Slice& limit) {
    ASSERT_OK(
        db_->CompactRange(CompactRangeOptions(), handles_[cf], &start, &limit));
  }

  int NumTableFilesAtLevel(int level, int cf) {
    return GetProperty(cf,
                       "rocksdb.num-files-at-level" + ToString(level));
  }

#ifndef ROCKSDB_LITE
  // Return spread of files per level
  std::string FilesPerLevel(int cf) {
    std::string result;
    int last_non_zero_offset = 0;
    for (int level = 0; level < dbfull()->NumberLevels(handles_[cf]); level++) {
      int f = NumTableFilesAtLevel(level, cf);
      char buf[100];
      snprintf(buf, sizeof(buf), "%s%d", (level ? "," : ""), f);
      result += buf;
      if (f > 0) {
        last_non_zero_offset = static_cast<int>(result.size());
      }
    }
    result.resize(last_non_zero_offset);
    return result;
  }
#endif

  void AssertFilesPerLevel(const std::string& value, int cf) {
#ifndef ROCKSDB_LITE
    ASSERT_EQ(value, FilesPerLevel(cf));
#endif
  }

#ifndef ROCKSDB_LITE  // GetLiveFilesMetaData is not supported
  int CountLiveFiles() {
    std::vector<LiveFileMetaData> metadata;
    db_->GetLiveFilesMetaData(&metadata);
    return static_cast<int>(metadata.size());
  }
#endif  // !ROCKSDB_LITE

  void AssertCountLiveFiles(int expected_value) {
#ifndef ROCKSDB_LITE
    ASSERT_EQ(expected_value, CountLiveFiles());
#endif
  }

  // Do n memtable flushes, each of which produces an sstable
  // covering the range [small,large].
  void MakeTables(int cf, int n, const std::string& small,
                  const std::string& large) {
    for (int i = 0; i < n; i++) {
      ASSERT_OK(Put(cf, small, "begin"));
      ASSERT_OK(Put(cf, large, "end"));
      ASSERT_OK(db_->Flush(FlushOptions(), handles_[cf]));
    }
  }

#ifndef ROCKSDB_LITE  // GetSortedWalFiles is not supported
  int CountLiveLogFiles() {
    int micros_wait_for_log_deletion = 20000;
    env_->SleepForMicroseconds(micros_wait_for_log_deletion);
    int ret = 0;
    VectorLogPtr wal_files;
    Status s;
    // GetSortedWalFiles is a flakey function -- it gets all the wal_dir
    // children files and then later checks for their existence. if some of the
    // log files doesn't exist anymore, it reports an error. it does all of this
    // without DB mutex held, so if a background process deletes the log file
    // while the function is being executed, it returns an error. We retry the
    // function 10 times to avoid the error failing the test
    for (int retries = 0; retries < 10; ++retries) {
      wal_files.clear();
      s = db_->GetSortedWalFiles(wal_files);
      if (s.ok()) {
        break;
      }
    }
    EXPECT_OK(s);
    for (const auto& wal : wal_files) {
      if (wal->Type() == kAliveLogFile) {
        ++ret;
      }
    }
    return ret;
    return 0;
  }
#endif  // !ROCKSDB_LITE

  void AssertCountLiveLogFiles(int value) {
#ifndef ROCKSDB_LITE  // GetSortedWalFiles is not supported
    ASSERT_EQ(value, CountLiveLogFiles());
#endif  // !ROCKSDB_LITE
  }

  void AssertNumberOfImmutableMemtables(std::vector<int> num_per_cf) {
    assert(num_per_cf.size() == handles_.size());

#ifndef ROCKSDB_LITE  // GetProperty is not supported in lite
    for (size_t i = 0; i < num_per_cf.size(); ++i) {
      ASSERT_EQ(num_per_cf[i], GetProperty(static_cast<int>(i),
                                           "rocksdb.num-immutable-mem-table"));
    }
#endif  // !ROCKSDB_LITE
  }

  void CopyFile(const std::string& source, const std::string& destination,
                uint64_t size = 0) {
    const EnvOptions soptions;
    unique_ptr<SequentialFile> srcfile;
    ASSERT_OK(env_->NewSequentialFile(source, &srcfile, soptions));
    unique_ptr<WritableFile> destfile;
    ASSERT_OK(env_->NewWritableFile(destination, &destfile, soptions));

    if (size == 0) {
      // default argument means copy everything
      ASSERT_OK(env_->GetFileSize(source, &size));
    }

    char buffer[4096];
    Slice slice;
    while (size > 0) {
      uint64_t one = std::min(uint64_t(sizeof(buffer)), size);
      ASSERT_OK(srcfile->Read(one, &slice, buffer));
      ASSERT_OK(destfile->Append(slice));
      size -= slice.size();
    }
    ASSERT_OK(destfile->Close());
  }

  std::vector<ColumnFamilyHandle*> handles_;
  std::vector<std::string> names_;
  ColumnFamilyOptions column_family_options_;
  DBOptions db_options_;
  std::string dbname_;
  DB* db_ = nullptr;
  EnvCounter* env_;
  Random rnd_;
};

TEST_F(ColumnFamilyTest, DontReuseColumnFamilyID) {
  for (int iter = 0; iter < 3; ++iter) {
    Open();
    CreateColumnFamilies({"one", "two", "three"});
    for (size_t i = 0; i < handles_.size(); ++i) {
      auto cfh = reinterpret_cast<ColumnFamilyHandleImpl*>(handles_[i]);
      ASSERT_EQ(i, cfh->GetID());
    }
    if (iter == 1) {
      Reopen();
    }
    DropColumnFamilies({3});
    Reopen();
    if (iter == 2) {
      // this tests if max_column_family is correctly persisted with
      // WriteSnapshot()
      Reopen();
    }
    CreateColumnFamilies({"three2"});
    // ID 3 that was used for dropped column family "three" should not be reused
    auto cfh3 = reinterpret_cast<ColumnFamilyHandleImpl*>(handles_[3]);
    ASSERT_EQ(4U, cfh3->GetID());
    Close();
    Destroy();
  }
}

TEST_F(ColumnFamilyTest, AddDrop) {
  Open();
  CreateColumnFamilies({"one", "two", "three"});
  ASSERT_EQ("NOT_FOUND", Get(1, "fodor"));
  ASSERT_EQ("NOT_FOUND", Get(2, "fodor"));
  DropColumnFamilies({2});
  ASSERT_EQ("NOT_FOUND", Get(1, "fodor"));
  CreateColumnFamilies({"four"});
  ASSERT_EQ("NOT_FOUND", Get(3, "fodor"));
  ASSERT_OK(Put(1, "fodor", "mirko"));
  ASSERT_EQ("mirko", Get(1, "fodor"));
  ASSERT_EQ("NOT_FOUND", Get(3, "fodor"));
  Close();
  ASSERT_TRUE(TryOpen({"default"}).IsInvalidArgument());
  Open({"default", "one", "three", "four"});
  DropColumnFamilies({1});
  Reopen();
  Close();

  std::vector<std::string> families;
  ASSERT_OK(DB::ListColumnFamilies(db_options_, dbname_, &families));
  sort(families.begin(), families.end());
  ASSERT_TRUE(families ==
              std::vector<std::string>({"default", "four", "three"}));
}

TEST_F(ColumnFamilyTest, DropTest) {
  // first iteration - dont reopen DB before dropping
  // second iteration - reopen DB before dropping
  for (int iter = 0; iter < 2; ++iter) {
    Open({"default"});
    CreateColumnFamiliesAndReopen({"pikachu"});
    for (int i = 0; i < 100; ++i) {
      ASSERT_OK(Put(1, ToString(i), "bar" + ToString(i)));
    }
    ASSERT_OK(Flush(1));

    if (iter == 1) {
      Reopen();
    }
    ASSERT_EQ("bar1", Get(1, "1"));

    AssertCountLiveFiles(1);
    DropColumnFamilies({1});
    // make sure that all files are deleted when we drop the column family
    AssertCountLiveFiles(0);
    Destroy();
  }
}

TEST_F(ColumnFamilyTest, WriteBatchFailure) {
  Open();
  CreateColumnFamiliesAndReopen({"one", "two"});
  WriteBatch batch;
  batch.Put(handles_[0], Slice("existing"), Slice("column-family"));
  batch.Put(handles_[1], Slice("non-existing"), Slice("column-family"));
  ASSERT_OK(db_->Write(WriteOptions(), &batch));
  DropColumnFamilies({1});
  WriteOptions woptions_ignore_missing_cf;
  woptions_ignore_missing_cf.ignore_missing_column_families = true;
  batch.Put(handles_[0], Slice("still here"), Slice("column-family"));
  ASSERT_OK(db_->Write(woptions_ignore_missing_cf, &batch));
  ASSERT_EQ("column-family", Get(0, "still here"));
  Status s = db_->Write(WriteOptions(), &batch);
  ASSERT_TRUE(s.IsInvalidArgument());
  Close();
}

TEST_F(ColumnFamilyTest, ReadWrite) {
  Open();
  CreateColumnFamiliesAndReopen({"one", "two"});
  ASSERT_OK(Put(0, "foo", "v1"));
  ASSERT_OK(Put(0, "bar", "v2"));
  ASSERT_OK(Put(1, "mirko", "v3"));
  ASSERT_OK(Put(0, "foo", "v2"));
  ASSERT_OK(Put(2, "fodor", "v5"));

  for (int iter = 0; iter <= 3; ++iter) {
    ASSERT_EQ("v2", Get(0, "foo"));
    ASSERT_EQ("v2", Get(0, "bar"));
    ASSERT_EQ("v3", Get(1, "mirko"));
    ASSERT_EQ("v5", Get(2, "fodor"));
    ASSERT_EQ("NOT_FOUND", Get(0, "fodor"));
    ASSERT_EQ("NOT_FOUND", Get(1, "fodor"));
    ASSERT_EQ("NOT_FOUND", Get(2, "foo"));
    if (iter <= 1) {
      Reopen();
    }
  }
  Close();
}

TEST_F(ColumnFamilyTest, IgnoreRecoveredLog) {
  std::string backup_logs = dbname_ + "/backup_logs";

  // delete old files in backup_logs directory
  ASSERT_OK(env_->CreateDirIfMissing(dbname_));
  ASSERT_OK(env_->CreateDirIfMissing(backup_logs));
  std::vector<std::string> old_files;
  env_->GetChildren(backup_logs, &old_files);
  for (auto& file : old_files) {
    if (file != "." && file != "..") {
      env_->DeleteFile(backup_logs + "/" + file);
    }
  }

  column_family_options_.merge_operator =
      MergeOperators::CreateUInt64AddOperator();
  db_options_.wal_dir = dbname_ + "/logs";
  Destroy();
  Open();
  CreateColumnFamilies({"cf1", "cf2"});

  // fill up the DB
  std::string one, two, three;
  PutFixed64(&one, 1);
  PutFixed64(&two, 2);
  PutFixed64(&three, 3);
  ASSERT_OK(Merge(0, "foo", one));
  ASSERT_OK(Merge(1, "mirko", one));
  ASSERT_OK(Merge(0, "foo", one));
  ASSERT_OK(Merge(2, "bla", one));
  ASSERT_OK(Merge(2, "fodor", one));
  ASSERT_OK(Merge(0, "bar", one));
  ASSERT_OK(Merge(2, "bla", one));
  ASSERT_OK(Merge(1, "mirko", two));
  ASSERT_OK(Merge(1, "franjo", one));

  // copy the logs to backup
  std::vector<std::string> logs;
  env_->GetChildren(db_options_.wal_dir, &logs);
  for (auto& log : logs) {
    if (log != ".." && log != ".") {
      CopyFile(db_options_.wal_dir + "/" + log, backup_logs + "/" + log);
    }
  }

  // recover the DB
  Close();

  // 1. check consistency
  // 2. copy the logs from backup back to WAL dir. if the recovery happens
  // again on the same log files, this should lead to incorrect results
  // due to applying merge operator twice
  // 3. check consistency
  for (int iter = 0; iter < 2; ++iter) {
    // assert consistency
    Open({"default", "cf1", "cf2"});
    ASSERT_EQ(two, Get(0, "foo"));
    ASSERT_EQ(one, Get(0, "bar"));
    ASSERT_EQ(three, Get(1, "mirko"));
    ASSERT_EQ(one, Get(1, "franjo"));
    ASSERT_EQ(one, Get(2, "fodor"));
    ASSERT_EQ(two, Get(2, "bla"));
    Close();

    if (iter == 0) {
      // copy the logs from backup back to wal dir
      for (auto& log : logs) {
        if (log != ".." && log != ".") {
          CopyFile(backup_logs + "/" + log, db_options_.wal_dir + "/" + log);
        }
      }
    }
  }
}

TEST_F(ColumnFamilyTest, FlushTest) {
  Open();
  CreateColumnFamiliesAndReopen({"one", "two"});
  ASSERT_OK(Put(0, "foo", "v1"));
  ASSERT_OK(Put(0, "bar", "v2"));
  ASSERT_OK(Put(1, "mirko", "v3"));
  ASSERT_OK(Put(0, "foo", "v2"));
  ASSERT_OK(Put(2, "fodor", "v5"));

  for (int j = 0; j < 2; j++) {
    ReadOptions ro;
    std::vector<Iterator*> iterators;
    // Hold super version.
    if (j == 0) {
      ASSERT_OK(db_->NewIterators(ro, handles_, &iterators));
    }

    for (int i = 0; i < 3; ++i) {
      uint64_t max_total_in_memory_state =
          MaxTotalInMemoryState();
      Flush(i);
      AssertMaxTotalInMemoryState(max_total_in_memory_state);
    }
    ASSERT_OK(Put(1, "foofoo", "bar"));
    ASSERT_OK(Put(0, "foofoo", "bar"));

    for (auto* it : iterators) {
      delete it;
    }
  }
  Reopen();

  for (int iter = 0; iter <= 2; ++iter) {
    ASSERT_EQ("v2", Get(0, "foo"));
    ASSERT_EQ("v2", Get(0, "bar"));
    ASSERT_EQ("v3", Get(1, "mirko"));
    ASSERT_EQ("v5", Get(2, "fodor"));
    ASSERT_EQ("NOT_FOUND", Get(0, "fodor"));
    ASSERT_EQ("NOT_FOUND", Get(1, "fodor"));
    ASSERT_EQ("NOT_FOUND", Get(2, "foo"));
    if (iter <= 1) {
      Reopen();
    }
  }
  Close();
}

// Makes sure that obsolete log files get deleted
TEST_F(ColumnFamilyTest, LogDeletionTest) {
  db_options_.max_total_wal_size = std::numeric_limits<uint64_t>::max();
  column_family_options_.arena_block_size = 4 * 1024;
  column_family_options_.write_buffer_size = 100000;  // 100KB
  Open();
  CreateColumnFamilies({"one", "two", "three", "four"});
  // Each bracket is one log file. if number is in (), it means
  // we don't need it anymore (it's been flushed)
  // []
  AssertCountLiveLogFiles(0);
  PutRandomData(0, 1, 100);
  // [0]
  PutRandomData(1, 1, 100);
  // [0, 1]
  PutRandomData(1, 1000, 100);
  WaitForFlush(1);
  // [0, (1)] [1]
  AssertCountLiveLogFiles(2);
  PutRandomData(0, 1, 100);
  // [0, (1)] [0, 1]
  AssertCountLiveLogFiles(2);
  PutRandomData(2, 1, 100);
  // [0, (1)] [0, 1, 2]
  PutRandomData(2, 1000, 100);
  WaitForFlush(2);
  // [0, (1)] [0, 1, (2)] [2]
  AssertCountLiveLogFiles(3);
  PutRandomData(2, 1000, 100);
  WaitForFlush(2);
  // [0, (1)] [0, 1, (2)] [(2)] [2]
  AssertCountLiveLogFiles(4);
  PutRandomData(3, 1, 100);
  // [0, (1)] [0, 1, (2)] [(2)] [2, 3]
  PutRandomData(1, 1, 100);
  // [0, (1)] [0, 1, (2)] [(2)] [1, 2, 3]
  AssertCountLiveLogFiles(4);
  PutRandomData(1, 1000, 100);
  WaitForFlush(1);
  // [0, (1)] [0, (1), (2)] [(2)] [(1), 2, 3] [1]
  AssertCountLiveLogFiles(5);
  PutRandomData(0, 1000, 100);
  WaitForFlush(0);
  // [(0), (1)] [(0), (1), (2)] [(2)] [(1), 2, 3] [1, (0)] [0]
  // delete obsolete logs -->
  // [(1), 2, 3] [1, (0)] [0]
  AssertCountLiveLogFiles(3);
  PutRandomData(0, 1000, 100);
  WaitForFlush(0);
  // [(1), 2, 3] [1, (0)], [(0)] [0]
  AssertCountLiveLogFiles(4);
  PutRandomData(1, 1000, 100);
  WaitForFlush(1);
  // [(1), 2, 3] [(1), (0)] [(0)] [0, (1)] [1]
  AssertCountLiveLogFiles(5);
  PutRandomData(2, 1000, 100);
  WaitForFlush(2);
  // [(1), (2), 3] [(1), (0)] [(0)] [0, (1)] [1, (2)], [2]
  AssertCountLiveLogFiles(6);
  PutRandomData(3, 1000, 100);
  WaitForFlush(3);
  // [(1), (2), (3)] [(1), (0)] [(0)] [0, (1)] [1, (2)], [2, (3)] [3]
  // delete obsolete logs -->
  // [0, (1)] [1, (2)], [2, (3)] [3]
  AssertCountLiveLogFiles(4);
  Close();
}

// Makes sure that obsolete log files get deleted
TEST_F(ColumnFamilyTest, DifferentWriteBufferSizes) {
  // disable flushing stale column families
  db_options_.max_total_wal_size = std::numeric_limits<uint64_t>::max();
  Open();
  CreateColumnFamilies({"one", "two", "three"});
  ColumnFamilyOptions default_cf, one, two, three;
  // setup options. all column families have max_write_buffer_number setup to 10
  // "default" -> 100KB memtable, start flushing immediatelly
  // "one" -> 200KB memtable, start flushing with two immutable memtables
  // "two" -> 1MB memtable, start flushing with three immutable memtables
  // "three" -> 90KB memtable, start flushing with four immutable memtables
  default_cf.write_buffer_size = 100000;
  default_cf.arena_block_size = 4 * 4096;
  default_cf.max_write_buffer_number = 10;
  default_cf.min_write_buffer_number_to_merge = 1;
  default_cf.max_write_buffer_number_to_maintain = 0;
  one.write_buffer_size = 200000;
  one.arena_block_size = 4 * 4096;
  one.max_write_buffer_number = 10;
  one.min_write_buffer_number_to_merge = 2;
  one.max_write_buffer_number_to_maintain = 1;
  two.write_buffer_size = 1000000;
  two.arena_block_size = 4 * 4096;
  two.max_write_buffer_number = 10;
  two.min_write_buffer_number_to_merge = 3;
  two.max_write_buffer_number_to_maintain = 2;
  three.write_buffer_size = 4096 * 22 + 2048;
  three.arena_block_size = 4096;
  three.max_write_buffer_number = 10;
  three.min_write_buffer_number_to_merge = 4;
  three.max_write_buffer_number_to_maintain = -1;

  Reopen({default_cf, one, two, three});

  int micros_wait_for_flush = 10000;
  PutRandomData(0, 100, 1000);
  WaitForFlush(0);
  AssertNumberOfImmutableMemtables({0, 0, 0, 0});
  AssertCountLiveLogFiles(1);
  PutRandomData(1, 200, 1000);
  env_->SleepForMicroseconds(micros_wait_for_flush);
  AssertNumberOfImmutableMemtables({0, 1, 0, 0});
  AssertCountLiveLogFiles(2);
  PutRandomData(2, 1000, 1000);
  env_->SleepForMicroseconds(micros_wait_for_flush);
  AssertNumberOfImmutableMemtables({0, 1, 1, 0});
  AssertCountLiveLogFiles(3);
  PutRandomData(2, 1000, 1000);
  env_->SleepForMicroseconds(micros_wait_for_flush);
  AssertNumberOfImmutableMemtables({0, 1, 2, 0});
  AssertCountLiveLogFiles(4);
  PutRandomData(3, 91, 990);
  env_->SleepForMicroseconds(micros_wait_for_flush);
  AssertNumberOfImmutableMemtables({0, 1, 2, 1});
  AssertCountLiveLogFiles(5);
  PutRandomData(3, 90, 990);
  env_->SleepForMicroseconds(micros_wait_for_flush);
  AssertNumberOfImmutableMemtables({0, 1, 2, 2});
  AssertCountLiveLogFiles(6);
  PutRandomData(3, 90, 990);
  env_->SleepForMicroseconds(micros_wait_for_flush);
  AssertNumberOfImmutableMemtables({0, 1, 2, 3});
  AssertCountLiveLogFiles(7);
  PutRandomData(0, 100, 1000);
  WaitForFlush(0);
  AssertNumberOfImmutableMemtables({0, 1, 2, 3});
  AssertCountLiveLogFiles(8);
  PutRandomData(2, 100, 10000);
  WaitForFlush(2);
  AssertNumberOfImmutableMemtables({0, 1, 0, 3});
  AssertCountLiveLogFiles(9);
  PutRandomData(3, 90, 990);
  WaitForFlush(3);
  AssertNumberOfImmutableMemtables({0, 1, 0, 0});
  AssertCountLiveLogFiles(10);
  PutRandomData(3, 90, 990);
  env_->SleepForMicroseconds(micros_wait_for_flush);
  AssertNumberOfImmutableMemtables({0, 1, 0, 1});
  AssertCountLiveLogFiles(11);
  PutRandomData(1, 200, 1000);
  WaitForFlush(1);
  AssertNumberOfImmutableMemtables({0, 0, 0, 1});
  AssertCountLiveLogFiles(5);
  PutRandomData(3, 90 * 3, 990);
  WaitForFlush(3);
  PutRandomData(3, 90 * 4, 990);
  WaitForFlush(3);
  AssertNumberOfImmutableMemtables({0, 0, 0, 0});
  AssertCountLiveLogFiles(12);
  PutRandomData(0, 100, 1000);
  WaitForFlush(0);
  AssertNumberOfImmutableMemtables({0, 0, 0, 0});
  AssertCountLiveLogFiles(12);
  PutRandomData(2, 3 * 1000, 1000);
  WaitForFlush(2);
  AssertNumberOfImmutableMemtables({0, 0, 0, 0});
  AssertCountLiveLogFiles(12);
  PutRandomData(1, 2*200, 1000);
  WaitForFlush(1);
  AssertNumberOfImmutableMemtables({0, 0, 0, 0});
  AssertCountLiveLogFiles(7);
  Close();
}

#ifndef ROCKSDB_LITE  // Cuckoo is not supported in lite
TEST_F(ColumnFamilyTest, MemtableNotSupportSnapshot) {
  Open();
  auto* s1 = dbfull()->GetSnapshot();
  ASSERT_TRUE(s1 != nullptr);
  dbfull()->ReleaseSnapshot(s1);

  // Add a column family that doesn't support snapshot
  ColumnFamilyOptions first;
  first.memtable_factory.reset(NewHashCuckooRepFactory(1024 * 1024));
  CreateColumnFamilies({"first"}, {first});
  auto* s2 = dbfull()->GetSnapshot();
  ASSERT_TRUE(s2 == nullptr);

  // Add a column family that supports snapshot. Snapshot stays not supported.
  ColumnFamilyOptions second;
  CreateColumnFamilies({"second"}, {second});
  auto* s3 = dbfull()->GetSnapshot();
  ASSERT_TRUE(s3 == nullptr);
  Close();
}
#endif  // !ROCKSDB_LITE

TEST_F(ColumnFamilyTest, DifferentMergeOperators) {
  Open();
  CreateColumnFamilies({"first", "second"});
  ColumnFamilyOptions default_cf, first, second;
  first.merge_operator = MergeOperators::CreateUInt64AddOperator();
  second.merge_operator = MergeOperators::CreateStringAppendOperator();
  Reopen({default_cf, first, second});

  std::string one, two, three;
  PutFixed64(&one, 1);
  PutFixed64(&two, 2);
  PutFixed64(&three, 3);

  ASSERT_OK(Put(0, "foo", two));
  ASSERT_OK(Put(0, "foo", one));
  ASSERT_TRUE(Merge(0, "foo", two).IsNotSupported());
  ASSERT_EQ(Get(0, "foo"), one);

  ASSERT_OK(Put(1, "foo", two));
  ASSERT_OK(Put(1, "foo", one));
  ASSERT_OK(Merge(1, "foo", two));
  ASSERT_EQ(Get(1, "foo"), three);

  ASSERT_OK(Put(2, "foo", two));
  ASSERT_OK(Put(2, "foo", one));
  ASSERT_OK(Merge(2, "foo", two));
  ASSERT_EQ(Get(2, "foo"), one + "," + two);
  Close();
}

TEST_F(ColumnFamilyTest, DifferentCompactionStyles) {
  Open();
  CreateColumnFamilies({"one", "two"});
  ColumnFamilyOptions default_cf, one, two;
  db_options_.max_open_files = 20;  // only 10 files in file cache
  db_options_.disableDataSync = true;

  default_cf.compaction_style = kCompactionStyleLevel;
  default_cf.num_levels = 3;
  default_cf.write_buffer_size = 64 << 10;  // 64KB
  default_cf.target_file_size_base = 30 << 10;
  default_cf.source_compaction_factor = 100;
  BlockBasedTableOptions table_options;
  table_options.no_block_cache = true;
  default_cf.table_factory.reset(NewBlockBasedTableFactory(table_options));

  one.compaction_style = kCompactionStyleUniversal;

  one.num_levels = 1;
  // trigger compaction if there are >= 4 files
  one.level0_file_num_compaction_trigger = 4;
  one.write_buffer_size = 120000;

  two.compaction_style = kCompactionStyleLevel;
  two.num_levels = 4;
  two.level0_file_num_compaction_trigger = 3;
  two.write_buffer_size = 100000;

  Reopen({default_cf, one, two});

  // SETUP column family "one" -- universal style
  for (int i = 0; i < one.level0_file_num_compaction_trigger - 1; ++i) {
    PutRandomData(1, 10, 12000);
    PutRandomData(1, 1, 10);
    WaitForFlush(1);
    AssertFilesPerLevel(ToString(i + 1), 1);
  }

  // SETUP column family "two" -- level style with 4 levels
  for (int i = 0; i < two.level0_file_num_compaction_trigger - 1; ++i) {
    PutRandomData(2, 10, 12000);
    PutRandomData(2, 1, 10);
    WaitForFlush(2);
    AssertFilesPerLevel(ToString(i + 1), 2);
  }

  // TRIGGER compaction "one"
  PutRandomData(1, 10, 12000);
  PutRandomData(1, 1, 10);

  // TRIGGER compaction "two"
  PutRandomData(2, 10, 12000);
  PutRandomData(2, 1, 10);

  // WAIT for compactions
  WaitForCompaction();

  // VERIFY compaction "one"
  AssertFilesPerLevel("1", 1);

  // VERIFY compaction "two"
  AssertFilesPerLevel("0,1", 2);
  CompactAll(2);
  AssertFilesPerLevel("0,1", 2);

  Close();
}

#ifndef ROCKSDB_LITE  // Tailing interator not supported
namespace {
std::string IterStatus(Iterator* iter) {
  std::string result;
  if (iter->Valid()) {
    result = iter->key().ToString() + "->" + iter->value().ToString();
  } else {
    result = "(invalid)";
  }
  return result;
}
}  // anonymous namespace

TEST_F(ColumnFamilyTest, NewIteratorsTest) {
  // iter == 0 -- no tailing
  // iter == 2 -- tailing
  for (int iter = 0; iter < 2; ++iter) {
    Open();
    CreateColumnFamiliesAndReopen({"one", "two"});
    ASSERT_OK(Put(0, "a", "b"));
    ASSERT_OK(Put(1, "b", "a"));
    ASSERT_OK(Put(2, "c", "m"));
    ASSERT_OK(Put(2, "v", "t"));
    std::vector<Iterator*> iterators;
    ReadOptions options;
    options.tailing = (iter == 1);
    ASSERT_OK(db_->NewIterators(options, handles_, &iterators));

    for (auto it : iterators) {
      it->SeekToFirst();
    }
    ASSERT_EQ(IterStatus(iterators[0]), "a->b");
    ASSERT_EQ(IterStatus(iterators[1]), "b->a");
    ASSERT_EQ(IterStatus(iterators[2]), "c->m");

    ASSERT_OK(Put(1, "x", "x"));

    for (auto it : iterators) {
      it->Next();
    }

    ASSERT_EQ(IterStatus(iterators[0]), "(invalid)");
    if (iter == 0) {
      // no tailing
      ASSERT_EQ(IterStatus(iterators[1]), "(invalid)");
    } else {
      // tailing
      ASSERT_EQ(IterStatus(iterators[1]), "x->x");
    }
    ASSERT_EQ(IterStatus(iterators[2]), "v->t");

    for (auto it : iterators) {
      delete it;
    }
    Destroy();
  }
}
#endif  // !ROCKSDB_LITE

#ifndef ROCKSDB_LITE  // ReadOnlyDB is not supported
TEST_F(ColumnFamilyTest, ReadOnlyDBTest) {
  Open();
  CreateColumnFamiliesAndReopen({"one", "two", "three", "four"});
  ASSERT_OK(Put(0, "a", "b"));
  ASSERT_OK(Put(1, "foo", "bla"));
  ASSERT_OK(Put(2, "foo", "blabla"));
  ASSERT_OK(Put(3, "foo", "blablabla"));
  ASSERT_OK(Put(4, "foo", "blablablabla"));

  DropColumnFamilies({2});
  Close();
  // open only a subset of column families
  AssertOpenReadOnly({"default", "one", "four"});
  ASSERT_EQ("NOT_FOUND", Get(0, "foo"));
  ASSERT_EQ("bla", Get(1, "foo"));
  ASSERT_EQ("blablablabla", Get(2, "foo"));


  // test newiterators
  {
    std::vector<Iterator*> iterators;
    ASSERT_OK(db_->NewIterators(ReadOptions(), handles_, &iterators));
    for (auto it : iterators) {
      it->SeekToFirst();
    }
    ASSERT_EQ(IterStatus(iterators[0]), "a->b");
    ASSERT_EQ(IterStatus(iterators[1]), "foo->bla");
    ASSERT_EQ(IterStatus(iterators[2]), "foo->blablablabla");
    for (auto it : iterators) {
      it->Next();
    }
    ASSERT_EQ(IterStatus(iterators[0]), "(invalid)");
    ASSERT_EQ(IterStatus(iterators[1]), "(invalid)");
    ASSERT_EQ(IterStatus(iterators[2]), "(invalid)");

    for (auto it : iterators) {
      delete it;
    }
  }

  Close();
  // can't open dropped column family
  Status s = OpenReadOnly({"default", "one", "two"});
  ASSERT_TRUE(!s.ok());

  // Can't open without specifying default column family
  s = OpenReadOnly({"one", "four"});
  ASSERT_TRUE(!s.ok());
}
#endif  // !ROCKSDB_LITE

TEST_F(ColumnFamilyTest, DontRollEmptyLogs) {
  Open();
  CreateColumnFamiliesAndReopen({"one", "two", "three", "four"});

  for (size_t i = 0; i < handles_.size(); ++i) {
    PutRandomData(static_cast<int>(i), 10, 100);
  }
  int num_writable_file_start = env_->GetNumberOfNewWritableFileCalls();
  // this will trigger the flushes
  for (int i = 0; i <= 4; ++i) {
    ASSERT_OK(Flush(i));
  }

  for (int i = 0; i < 4; ++i) {
    WaitForFlush(i);
  }
  int total_new_writable_files =
      env_->GetNumberOfNewWritableFileCalls() - num_writable_file_start;
  ASSERT_EQ(static_cast<size_t>(total_new_writable_files), handles_.size() + 1);
  Close();
}

TEST_F(ColumnFamilyTest, FlushStaleColumnFamilies) {
  Open();
  CreateColumnFamilies({"one", "two"});
  ColumnFamilyOptions default_cf, one, two;
  default_cf.write_buffer_size = 100000;  // small write buffer size
  default_cf.arena_block_size = 4096;
  default_cf.disable_auto_compactions = true;
  one.disable_auto_compactions = true;
  two.disable_auto_compactions = true;
  db_options_.max_total_wal_size = 210000;

  Reopen({default_cf, one, two});

  PutRandomData(2, 1, 10);  // 10 bytes
  for (int i = 0; i < 2; ++i) {
    PutRandomData(0, 100, 1000);  // flush
    WaitForFlush(0);

    AssertCountLiveFiles(i + 1);
  }
  // third flush. now, CF [two] should be detected as stale and flushed
  // column family 1 should not be flushed since it's empty
  PutRandomData(0, 100, 1000);  // flush
  WaitForFlush(0);
  WaitForFlush(2);
  // 3 files for default column families, 1 file for column family [two], zero
  // files for column family [one], because it's empty
  AssertCountLiveFiles(4);
  Close();
}

TEST_F(ColumnFamilyTest, CreateMissingColumnFamilies) {
  Status s = TryOpen({"one", "two"});
  ASSERT_TRUE(!s.ok());
  db_options_.create_missing_column_families = true;
  s = TryOpen({"default", "one", "two"});
  ASSERT_TRUE(s.ok());
  Close();
}

TEST_F(ColumnFamilyTest, SanitizeOptions) {
  DBOptions db_options;
  for (int s = kCompactionStyleLevel; s <= kCompactionStyleUniversal; ++s) {
    for (int l = 0; l <= 2; l++) {
      for (int i = 1; i <= 3; i++) {
        for (int j = 1; j <= 3; j++) {
          for (int k = 1; k <= 3; k++) {
            ColumnFamilyOptions original;
            original.compaction_style = static_cast<CompactionStyle>(s);
            original.num_levels = l;
            original.level0_stop_writes_trigger = i;
            original.level0_slowdown_writes_trigger = j;
            original.level0_file_num_compaction_trigger = k;
            original.write_buffer_size =
                l * 4 * 1024 * 1024 + i * 1024 * 1024 + j * 1024 + k;

            ColumnFamilyOptions result =
                SanitizeOptions(db_options, nullptr, original);
            ASSERT_TRUE(result.level0_stop_writes_trigger >=
                        result.level0_slowdown_writes_trigger);
            ASSERT_TRUE(result.level0_slowdown_writes_trigger >=
                        result.level0_file_num_compaction_trigger);
            ASSERT_TRUE(result.level0_file_num_compaction_trigger ==
                        original.level0_file_num_compaction_trigger);
            if (s == kCompactionStyleLevel) {
              ASSERT_GE(result.num_levels, 2);
            } else {
              ASSERT_GE(result.num_levels, 1);
              if (original.num_levels >= 1) {
                ASSERT_EQ(result.num_levels, original.num_levels);
              }
            }

            // Make sure Sanitize options sets arena_block_size to 1/8 of
            // the write_buffer_size, rounded up to a multiple of 4k.
            size_t expected_arena_block_size =
                l * 4 * 1024 * 1024 / 8 + i * 1024 * 1024 / 8;
            if (j + k != 0) {
              // not a multiple of 4k, round up 4k
              expected_arena_block_size += 4 * 1024;
            }
            ASSERT_EQ(expected_arena_block_size, result.arena_block_size);
          }
        }
      }
    }
  }
}

TEST_F(ColumnFamilyTest, ReadDroppedColumnFamily) {
  // iter 0 -- drop CF, don't reopen
  // iter 1 -- delete CF, reopen
  for (int iter = 0; iter < 2; ++iter) {
    db_options_.create_missing_column_families = true;
    db_options_.max_open_files = 20;
    // delete obsolete files always
    db_options_.delete_obsolete_files_period_micros = 0;
    Open({"default", "one", "two"});
    ColumnFamilyOptions options;
    options.level0_file_num_compaction_trigger = 100;
    options.level0_slowdown_writes_trigger = 200;
    options.level0_stop_writes_trigger = 200;
    options.write_buffer_size = 100000;  // small write buffer size
    Reopen({options, options, options});

    // 1MB should create ~10 files for each CF
    int kKeysNum = 10000;
    PutRandomData(0, kKeysNum, 100);
    PutRandomData(1, kKeysNum, 100);
    PutRandomData(2, kKeysNum, 100);

    if (iter == 0) {
      // Drop CF two
      ASSERT_OK(db_->DropColumnFamily(handles_[2]));
    } else {
      // delete CF two
      delete handles_[2];
      handles_[2] = nullptr;
    }

    // Add bunch more data to other CFs
    PutRandomData(0, kKeysNum, 100);
    PutRandomData(1, kKeysNum, 100);

    if (iter == 1) {
      Reopen();
    }

    // Since we didn't delete CF handle, RocksDB's contract guarantees that
    // we're still able to read dropped CF
    for (int i = 0; i < 3; ++i) {
      std::unique_ptr<Iterator> iterator(
          db_->NewIterator(ReadOptions(), handles_[i]));
      int count = 0;
      for (iterator->SeekToFirst(); iterator->Valid(); iterator->Next()) {
        ASSERT_OK(iterator->status());
        ++count;
      }
      ASSERT_OK(iterator->status());
      ASSERT_EQ(count, kKeysNum * ((i == 2) ? 1 : 2));
    }

    Close();
    Destroy();
  }
}

TEST_F(ColumnFamilyTest, FlushAndDropRaceCondition) {
  db_options_.create_missing_column_families = true;
  Open({"default", "one"});
  ColumnFamilyOptions options;
  options.level0_file_num_compaction_trigger = 100;
  options.level0_slowdown_writes_trigger = 200;
  options.level0_stop_writes_trigger = 200;
  options.max_write_buffer_number = 20;
  options.write_buffer_size = 100000;  // small write buffer size
  Reopen({options, options});

  rocksdb::SyncPoint::GetInstance()->LoadDependency(
      {{"VersionSet::LogAndApply::ColumnFamilyDrop:1"
        "FlushJob::InstallResults"},
       {"FlushJob::InstallResults",
        "VersionSet::LogAndApply::ColumnFamilyDrop:2", }});

  rocksdb::SyncPoint::GetInstance()->EnableProcessing();
  test::SleepingBackgroundTask sleeping_task;

  env_->Schedule(&test::SleepingBackgroundTask::DoSleepTask, &sleeping_task,
                 Env::Priority::HIGH);

  // 1MB should create ~10 files for each CF
  int kKeysNum = 10000;
  PutRandomData(1, kKeysNum, 100);

  std::vector<std::thread> threads;
  threads.emplace_back([&] { ASSERT_OK(db_->DropColumnFamily(handles_[1])); });

  sleeping_task.WakeUp();
  sleeping_task.WaitUntilDone();
  sleeping_task.Reset();
  // now we sleep again. this is just so we're certain that flush job finished
  env_->Schedule(&test::SleepingBackgroundTask::DoSleepTask, &sleeping_task,
                 Env::Priority::HIGH);
  sleeping_task.WakeUp();
  sleeping_task.WaitUntilDone();

  {
    // Since we didn't delete CF handle, RocksDB's contract guarantees that
    // we're still able to read dropped CF
    std::unique_ptr<Iterator> iterator(
        db_->NewIterator(ReadOptions(), handles_[1]));
    int count = 0;
    for (iterator->SeekToFirst(); iterator->Valid(); iterator->Next()) {
      ASSERT_OK(iterator->status());
      ++count;
    }
    ASSERT_OK(iterator->status());
    ASSERT_EQ(count, kKeysNum);
  }
  for (auto& t : threads) {
    t.join();
  }

  Close();
  Destroy();
  rocksdb::SyncPoint::GetInstance()->DisableProcessing();
}

}  // namespace rocksdb

int main(int argc, char** argv) {
  ::testing::InitGoogleTest(&argc, argv);
  return RUN_ALL_TESTS();
}
