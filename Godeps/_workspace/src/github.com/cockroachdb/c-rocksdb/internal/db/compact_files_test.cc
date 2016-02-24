//  Copyright (c) 2014, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.

#ifndef ROCKSDB_LITE

#include <mutex>
#include <string>
#include <vector>

#include "rocksdb/db.h"
#include "rocksdb/env.h"
#include "util/string_util.h"
#include "util/testharness.h"

namespace rocksdb {

class CompactFilesTest : public testing::Test {
 public:
  CompactFilesTest() {
    env_ = Env::Default();
    db_name_ = test::TmpDir(env_) + "/compact_files_test";
  }

  std::string db_name_;
  Env* env_;
};

// A class which remembers the name of each flushed file.
class FlushedFileCollector : public EventListener {
 public:
  FlushedFileCollector() {}
  ~FlushedFileCollector() {}

  virtual void OnFlushCompleted(
      DB* db, const FlushJobInfo& info) override {
    std::lock_guard<std::mutex> lock(mutex_);
    flushed_files_.push_back(info.file_path);
  }

  std::vector<std::string> GetFlushedFiles() {
    std::lock_guard<std::mutex> lock(mutex_);
    std::vector<std::string> result;
    for (auto fname : flushed_files_) {
      result.push_back(fname);
    }
    return result;
  }

 private:
  std::vector<std::string> flushed_files_;
  std::mutex mutex_;
};

TEST_F(CompactFilesTest, ObsoleteFiles) {
  Options options;
  // to trigger compaction more easily
  const int kWriteBufferSize = 10000;
  options.create_if_missing = true;
  // Disable RocksDB background compaction.
  options.compaction_style = kCompactionStyleNone;
  // Small slowdown and stop trigger for experimental purpose.
  options.level0_slowdown_writes_trigger = 20;
  options.level0_stop_writes_trigger = 20;
  options.write_buffer_size = kWriteBufferSize;
  options.max_write_buffer_number = 2;
  options.compression = kNoCompression;

  // Add listener
  FlushedFileCollector* collector = new FlushedFileCollector();
  options.listeners.emplace_back(collector);

  DB* db = nullptr;
  DestroyDB(db_name_, options);
  Status s = DB::Open(options, db_name_, &db);
  assert(s.ok());
  assert(db);

  // create couple files
  for (int i = 1000; i < 2000; ++i) {
    db->Put(WriteOptions(), ToString(i),
            std::string(kWriteBufferSize / 10, 'a' + (i % 26)));
  }

  auto l0_files = collector->GetFlushedFiles();
  CompactionOptions compact_opt;
  compact_opt.compression = kNoCompression;
  compact_opt.output_file_size_limit = kWriteBufferSize * 5;
  ASSERT_OK(db->CompactFiles(CompactionOptions(), l0_files, 1));

  // verify all compaction input files are deleted
  for (auto fname : l0_files) {
    ASSERT_EQ(Status::NotFound(), env_->FileExists(fname));
  }
  delete db;
}

}  // namespace rocksdb

int main(int argc, char** argv) {
  ::testing::InitGoogleTest(&argc, argv);
  return RUN_ALL_TESTS();
}

#else
#include <stdio.h>

int main(int argc, char** argv) {
  fprintf(stderr,
          "SKIPPED as DBImpl::CompactFiles is not supported in ROCKSDB_LITE\n");
  return 0;
}

#endif  // !ROCKSDB_LITE
