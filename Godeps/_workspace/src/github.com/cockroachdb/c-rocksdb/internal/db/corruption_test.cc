//  Copyright (c) 2013, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.
//
// Copyright (c) 2011 The LevelDB Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file. See the AUTHORS file for names of contributors.

#include "rocksdb/db.h"

#include <errno.h>
#include <fcntl.h>
#include <sys/stat.h>
#include <sys/types.h>
#include "rocksdb/cache.h"
#include "rocksdb/env.h"
#include "rocksdb/table.h"
#include "rocksdb/write_batch.h"
#include "db/db_impl.h"
#include "db/filename.h"
#include "db/log_format.h"
#include "db/version_set.h"
#include "util/logging.h"
#include "util/testharness.h"
#include "util/testutil.h"

namespace rocksdb {

static const int kValueSize = 1000;

class CorruptionTest : public testing::Test {
 public:
  test::ErrorEnv env_;
  std::string dbname_;
  shared_ptr<Cache> tiny_cache_;
  Options options_;
  DB* db_;

  CorruptionTest() {
    tiny_cache_ = NewLRUCache(100);
    options_.env = &env_;
    dbname_ = test::TmpDir() + "/corruption_test";
    DestroyDB(dbname_, options_);

    db_ = nullptr;
    options_.create_if_missing = true;
    BlockBasedTableOptions table_options;
    table_options.block_size_deviation = 0;  // make unit test pass for now
    options_.table_factory.reset(NewBlockBasedTableFactory(table_options));
    Reopen();
    options_.create_if_missing = false;
  }

  ~CorruptionTest() {
     delete db_;
     DestroyDB(dbname_, Options());
  }

  void CloseDb() {
    delete db_;
    db_ = nullptr;
  }

  Status TryReopen(Options* options = nullptr) {
    delete db_;
    db_ = nullptr;
    Options opt = (options ? *options : options_);
    opt.env = &env_;
    opt.arena_block_size = 4096;
    BlockBasedTableOptions table_options;
    table_options.block_cache = tiny_cache_;
    table_options.block_size_deviation = 0;
    opt.table_factory.reset(NewBlockBasedTableFactory(table_options));
    return DB::Open(opt, dbname_, &db_);
  }

  void Reopen(Options* options = nullptr) {
    ASSERT_OK(TryReopen(options));
  }

  void RepairDB() {
    delete db_;
    db_ = nullptr;
    ASSERT_OK(::rocksdb::RepairDB(dbname_, options_));
  }

  void Build(int n) {
    std::string key_space, value_space;
    WriteBatch batch;
    for (int i = 0; i < n; i++) {
      //if ((i % 100) == 0) fprintf(stderr, "@ %d of %d\n", i, n);
      Slice key = Key(i, &key_space);
      batch.Clear();
      batch.Put(key, Value(i, &value_space));
      ASSERT_OK(db_->Write(WriteOptions(), &batch));
    }
  }

  void Check(int min_expected, int max_expected) {
    unsigned int next_expected = 0;
    int missed = 0;
    int bad_keys = 0;
    int bad_values = 0;
    int correct = 0;
    std::string value_space;
    // Do not verify checksums. If we verify checksums then the
    // db itself will raise errors because data is corrupted.
    // Instead, we want the reads to be successful and this test
    // will detect whether the appropriate corruptions have
    // occurred.
    Iterator* iter = db_->NewIterator(ReadOptions(false, true));
    for (iter->SeekToFirst(); iter->Valid(); iter->Next()) {
      uint64_t key;
      Slice in(iter->key());
      if (!ConsumeDecimalNumber(&in, &key) ||
          !in.empty() ||
          key < next_expected) {
        bad_keys++;
        continue;
      }
      missed += (key - next_expected);
      next_expected = static_cast<unsigned int>(key + 1);
      if (iter->value() != Value(static_cast<int>(key), &value_space)) {
        bad_values++;
      } else {
        correct++;
      }
    }
    delete iter;

    fprintf(stderr,
            "expected=%d..%d; got=%d; bad_keys=%d; bad_values=%d; missed=%d\n",
            min_expected, max_expected, correct, bad_keys, bad_values, missed);
    ASSERT_LE(min_expected, correct);
    ASSERT_GE(max_expected, correct);
  }

  void CorruptFile(const std::string& fname, int offset, int bytes_to_corrupt) {
    struct stat sbuf;
    if (stat(fname.c_str(), &sbuf) != 0) {
      const char* msg = strerror(errno);
      ASSERT_TRUE(false) << fname << ": " << msg;
    }

    if (offset < 0) {
      // Relative to end of file; make it absolute
      if (-offset > sbuf.st_size) {
        offset = 0;
      } else {
        offset = static_cast<int>(sbuf.st_size + offset);
      }
    }
    if (offset > sbuf.st_size) {
      offset = static_cast<int>(sbuf.st_size);
    }
    if (offset + bytes_to_corrupt > sbuf.st_size) {
      bytes_to_corrupt = static_cast<int>(sbuf.st_size - offset);
    }

    // Do it
    std::string contents;
    Status s = ReadFileToString(Env::Default(), fname, &contents);
    ASSERT_TRUE(s.ok()) << s.ToString();
    for (int i = 0; i < bytes_to_corrupt; i++) {
      contents[i + offset] ^= 0x80;
    }
    s = WriteStringToFile(Env::Default(), contents, fname);
    ASSERT_TRUE(s.ok()) << s.ToString();
  }

  void Corrupt(FileType filetype, int offset, int bytes_to_corrupt) {
    // Pick file to corrupt
    std::vector<std::string> filenames;
    ASSERT_OK(env_.GetChildren(dbname_, &filenames));
    uint64_t number;
    FileType type;
    std::string fname;
    int picked_number = -1;
    for (unsigned int i = 0; i < filenames.size(); i++) {
      if (ParseFileName(filenames[i], &number, &type) &&
          type == filetype &&
          static_cast<int>(number) > picked_number) {  // Pick latest file
        fname = dbname_ + "/" + filenames[i];
        picked_number = static_cast<int>(number);
      }
    }
    ASSERT_TRUE(!fname.empty()) << filetype;

    CorruptFile(fname, offset, bytes_to_corrupt);
  }

  // corrupts exactly one file at level `level`. if no file found at level,
  // asserts
  void CorruptTableFileAtLevel(int level, int offset, int bytes_to_corrupt) {
    std::vector<LiveFileMetaData> metadata;
    db_->GetLiveFilesMetaData(&metadata);
    for (const auto& m : metadata) {
      if (m.level == level) {
        CorruptFile(dbname_ + "/" + m.name, offset, bytes_to_corrupt);
        return;
      }
    }
    ASSERT_TRUE(false) << "no file found at level";
  }


  int Property(const std::string& name) {
    std::string property;
    int result;
    if (db_->GetProperty(name, &property) &&
        sscanf(property.c_str(), "%d", &result) == 1) {
      return result;
    } else {
      return -1;
    }
  }

  // Return the ith key
  Slice Key(int i, std::string* storage) {
    char buf[100];
    snprintf(buf, sizeof(buf), "%016d", i);
    storage->assign(buf, strlen(buf));
    return Slice(*storage);
  }

  // Return the value to associate with the specified key
  Slice Value(int k, std::string* storage) {
    Random r(k);
    return test::RandomString(&r, kValueSize, storage);
  }
};

TEST_F(CorruptionTest, Recovery) {
  Build(100);
  Check(100, 100);
#ifdef OS_WIN
  // On Wndows OS Disk cache does not behave properly
  // We do not call FlushBuffers on every Flush. If we do not close
  // the log file prior to the corruption we end up with the first
  // block not corrupted but only the second. However, under the debugger
  // things work just fine but never pass when running normally
  // For that reason people may want to run with unbuffered I/O. That option
  // is not available for WAL though.
  CloseDb();
#endif
  Corrupt(kLogFile, 19, 1);      // WriteBatch tag for first record
  Corrupt(kLogFile, log::kBlockSize + 1000, 1);  // Somewhere in second block
  ASSERT_TRUE(!TryReopen().ok());
  options_.paranoid_checks = false;
  Reopen(&options_);

  // The 64 records in the first two log blocks are completely lost.
  Check(36, 36);
}

TEST_F(CorruptionTest, RecoverWriteError) {
  env_.writable_file_error_ = true;
  Status s = TryReopen();
  ASSERT_TRUE(!s.ok());
}

TEST_F(CorruptionTest, NewFileErrorDuringWrite) {
  // Do enough writing to force minor compaction
  env_.writable_file_error_ = true;
  const int num =
      static_cast<int>(3 + (Options().write_buffer_size / kValueSize));
  std::string value_storage;
  Status s;
  bool failed = false;
  for (int i = 0; i < num; i++) {
    WriteBatch batch;
    batch.Put("a", Value(100, &value_storage));
    s = db_->Write(WriteOptions(), &batch);
    if (!s.ok()) {
      failed = true;
    }
    ASSERT_TRUE(!failed || !s.ok());
  }
  ASSERT_TRUE(!s.ok());
  ASSERT_GE(env_.num_writable_file_errors_, 1);
  env_.writable_file_error_ = false;
  Reopen();
}

TEST_F(CorruptionTest, TableFile) {
  Build(100);
  DBImpl* dbi = reinterpret_cast<DBImpl*>(db_);
  dbi->TEST_FlushMemTable();
  dbi->TEST_CompactRange(0, nullptr, nullptr);
  dbi->TEST_CompactRange(1, nullptr, nullptr);

  Corrupt(kTableFile, 100, 1);
  Check(99, 99);
}

TEST_F(CorruptionTest, TableFileIndexData) {
  Build(10000);  // Enough to build multiple Tables
  DBImpl* dbi = reinterpret_cast<DBImpl*>(db_);
  dbi->TEST_FlushMemTable();

  Corrupt(kTableFile, -2000, 500);
  Reopen();
  Check(5000, 9999);
}

TEST_F(CorruptionTest, MissingDescriptor) {
  Build(1000);
  RepairDB();
  Reopen();
  Check(1000, 1000);
}

TEST_F(CorruptionTest, SequenceNumberRecovery) {
  ASSERT_OK(db_->Put(WriteOptions(), "foo", "v1"));
  ASSERT_OK(db_->Put(WriteOptions(), "foo", "v2"));
  ASSERT_OK(db_->Put(WriteOptions(), "foo", "v3"));
  ASSERT_OK(db_->Put(WriteOptions(), "foo", "v4"));
  ASSERT_OK(db_->Put(WriteOptions(), "foo", "v5"));
  RepairDB();
  Reopen();
  std::string v;
  ASSERT_OK(db_->Get(ReadOptions(), "foo", &v));
  ASSERT_EQ("v5", v);
  // Write something.  If sequence number was not recovered properly,
  // it will be hidden by an earlier write.
  ASSERT_OK(db_->Put(WriteOptions(), "foo", "v6"));
  ASSERT_OK(db_->Get(ReadOptions(), "foo", &v));
  ASSERT_EQ("v6", v);
  Reopen();
  ASSERT_OK(db_->Get(ReadOptions(), "foo", &v));
  ASSERT_EQ("v6", v);
}

TEST_F(CorruptionTest, CorruptedDescriptor) {
  ASSERT_OK(db_->Put(WriteOptions(), "foo", "hello"));
  DBImpl* dbi = reinterpret_cast<DBImpl*>(db_);
  dbi->TEST_FlushMemTable();
  dbi->TEST_CompactRange(0, nullptr, nullptr);

  Corrupt(kDescriptorFile, 0, 1000);
  Status s = TryReopen();
  ASSERT_TRUE(!s.ok());

  RepairDB();
  Reopen();
  std::string v;
  ASSERT_OK(db_->Get(ReadOptions(), "foo", &v));
  ASSERT_EQ("hello", v);
}

TEST_F(CorruptionTest, CompactionInputError) {
  Options options;
  Reopen(&options);
  Build(10);
  DBImpl* dbi = reinterpret_cast<DBImpl*>(db_);
  dbi->TEST_FlushMemTable();
  dbi->TEST_CompactRange(0, nullptr, nullptr);
  dbi->TEST_CompactRange(1, nullptr, nullptr);
  ASSERT_EQ(1, Property("rocksdb.num-files-at-level2"));

  Corrupt(kTableFile, 100, 1);
  Check(9, 9);

  // Force compactions by writing lots of values
  Build(10000);
  Check(10000, 10000);
}

TEST_F(CorruptionTest, CompactionInputErrorParanoid) {
  Options options;
  options.paranoid_checks = true;
  options.write_buffer_size = 131072;
  options.max_write_buffer_number = 2;
  Reopen(&options);
  DBImpl* dbi = reinterpret_cast<DBImpl*>(db_);

  // Fill levels >= 1
  for (int level = 1; level < dbi->NumberLevels(); level++) {
    dbi->Put(WriteOptions(), "", "begin");
    dbi->Put(WriteOptions(), "~", "end");
    dbi->TEST_FlushMemTable();
    for (int comp_level = 0; comp_level < dbi->NumberLevels() - level;
         ++comp_level) {
      dbi->TEST_CompactRange(comp_level, nullptr, nullptr);
    }
  }

  Reopen(&options);

  dbi = reinterpret_cast<DBImpl*>(db_);
  Build(10);
  dbi->TEST_FlushMemTable();
  dbi->TEST_WaitForCompact();
  ASSERT_EQ(1, Property("rocksdb.num-files-at-level0"));

  CorruptTableFileAtLevel(0, 100, 1);
  Check(9, 9);

  // Write must eventually fail because of corrupted table
  Status s;
  std::string tmp1, tmp2;
  bool failed = false;
  for (int i = 0; i < 10000; i++) {
    s = db_->Put(WriteOptions(), Key(i, &tmp1), Value(i, &tmp2));
    if (!s.ok()) {
      failed = true;
    }
    // if one write failed, every subsequent write must fail, too
    ASSERT_TRUE(!failed || !s.ok()) << "write did not fail in a corrupted db";
  }
  ASSERT_TRUE(!s.ok()) << "write did not fail in corrupted paranoid db";
}

TEST_F(CorruptionTest, UnrelatedKeys) {
  Build(10);
  DBImpl* dbi = reinterpret_cast<DBImpl*>(db_);
  dbi->TEST_FlushMemTable();
  Corrupt(kTableFile, 100, 1);

  std::string tmp1, tmp2;
  ASSERT_OK(db_->Put(WriteOptions(), Key(1000, &tmp1), Value(1000, &tmp2)));
  std::string v;
  ASSERT_OK(db_->Get(ReadOptions(), Key(1000, &tmp1), &v));
  ASSERT_EQ(Value(1000, &tmp2).ToString(), v);
  dbi->TEST_FlushMemTable();
  ASSERT_OK(db_->Get(ReadOptions(), Key(1000, &tmp1), &v));
  ASSERT_EQ(Value(1000, &tmp2).ToString(), v);
}

TEST_F(CorruptionTest, FileSystemStateCorrupted) {
  for (int iter = 0; iter < 2; ++iter) {
    Options options;
    options.paranoid_checks = true;
    options.create_if_missing = true;
    Reopen(&options);
    Build(10);
    ASSERT_OK(db_->Flush(FlushOptions()));
    DBImpl* dbi = reinterpret_cast<DBImpl*>(db_);
    std::vector<LiveFileMetaData> metadata;
    dbi->GetLiveFilesMetaData(&metadata);
    ASSERT_GT(metadata.size(), size_t(0));
    std::string filename = dbname_ + metadata[0].name;

    delete db_;
    db_ = nullptr;

    if (iter == 0) {  // corrupt file size
      unique_ptr<WritableFile> file;
      env_.NewWritableFile(filename, &file, EnvOptions());
      file->Append(Slice("corrupted sst"));
      file.reset();
    } else {  // delete the file
      env_.DeleteFile(filename);
    }

    Status x = TryReopen(&options);
    ASSERT_TRUE(x.IsCorruption());
    DestroyDB(dbname_, options_);
    Reopen(&options);
  }
}

}  // namespace rocksdb

int main(int argc, char** argv) {
  ::testing::InitGoogleTest(&argc, argv);
  return RUN_ALL_TESTS();
}
