//  Copyright (c) 2013, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.
//
// Copyright (c) 2011 The LevelDB Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file. See the AUTHORS file for names of contributors.

#include <string>
#include "db/file_indexer.h"
#include "db/dbformat.h"
#include "db/version_edit.h"
#include "port/stack_trace.h"
#include "rocksdb/comparator.h"
#include "util/testharness.h"
#include "util/testutil.h"

namespace rocksdb {

class IntComparator : public Comparator {
 public:
  int Compare(const Slice& a, const Slice& b) const override {
    assert(a.size() == 8);
    assert(b.size() == 8);
    int64_t diff = *reinterpret_cast<const int64_t*>(a.data()) -
                   *reinterpret_cast<const int64_t*>(b.data());
    if (diff < 0) {
      return -1;
    } else if (diff == 0) {
      return 0;
    } else {
      return 1;
    }
  }

  const char* Name() const override { return "IntComparator"; }

  void FindShortestSeparator(std::string* start,
                             const Slice& limit) const override {}

  void FindShortSuccessor(std::string* key) const override {}
};

class FileIndexerTest : public testing::Test {
 public:
  FileIndexerTest()
      : kNumLevels(4), files(new std::vector<FileMetaData*>[kNumLevels]) {}

  ~FileIndexerTest() {
    ClearFiles();
    delete[] files;
  }

  void AddFile(int level, int64_t smallest, int64_t largest) {
    auto* f = new FileMetaData();
    f->smallest = IntKey(smallest);
    f->largest = IntKey(largest);
    files[level].push_back(f);
  }

  InternalKey IntKey(int64_t v) {
    return InternalKey(Slice(reinterpret_cast<char*>(&v), 8), 0, kTypeValue);
  }

  void ClearFiles() {
    for (uint32_t i = 0; i < kNumLevels; ++i) {
      for (auto* f : files[i]) {
        delete f;
      }
      files[i].clear();
    }
  }

  void GetNextLevelIndex(const uint32_t level, const uint32_t file_index,
      const int cmp_smallest, const int cmp_largest, int32_t* left_index,
      int32_t* right_index) {
    *left_index = 100;
    *right_index = 100;
    indexer->GetNextLevelIndex(level, file_index, cmp_smallest, cmp_largest,
                               left_index, right_index);
  }

  int32_t left = 100;
  int32_t right = 100;
  const uint32_t kNumLevels;
  IntComparator ucmp;
  FileIndexer* indexer;

  std::vector<FileMetaData*>* files;
};

// Case 0: Empty
TEST_F(FileIndexerTest, Empty) {
  Arena arena;
  indexer = new FileIndexer(&ucmp);
  indexer->UpdateIndex(&arena, 0, files);
  delete indexer;
}

// Case 1: no overlap, files are on the left of next level files
TEST_F(FileIndexerTest, no_overlap_left) {
  Arena arena;
  indexer = new FileIndexer(&ucmp);
  // level 1
  AddFile(1, 100, 200);
  AddFile(1, 300, 400);
  AddFile(1, 500, 600);
  // level 2
  AddFile(2, 1500, 1600);
  AddFile(2, 1601, 1699);
  AddFile(2, 1700, 1800);
  // level 3
  AddFile(3, 2500, 2600);
  AddFile(3, 2601, 2699);
  AddFile(3, 2700, 2800);
  indexer->UpdateIndex(&arena, kNumLevels, files);
  for (uint32_t level = 1; level < 3; ++level) {
    for (uint32_t f = 0; f < 3; ++f) {
      GetNextLevelIndex(level, f, -1, -1, &left, &right);
      ASSERT_EQ(0, left);
      ASSERT_EQ(-1, right);
      GetNextLevelIndex(level, f, 0, -1, &left, &right);
      ASSERT_EQ(0, left);
      ASSERT_EQ(-1, right);
      GetNextLevelIndex(level, f, 1, -1, &left, &right);
      ASSERT_EQ(0, left);
      ASSERT_EQ(-1, right);
      GetNextLevelIndex(level, f, 1, 0, &left, &right);
      ASSERT_EQ(0, left);
      ASSERT_EQ(-1, right);
      GetNextLevelIndex(level, f, 1, 1, &left, &right);
      ASSERT_EQ(0, left);
      ASSERT_EQ(2, right);
    }
  }
  delete indexer;
  ClearFiles();
}

// Case 2: no overlap, files are on the right of next level files
TEST_F(FileIndexerTest, no_overlap_right) {
  Arena arena;
  indexer = new FileIndexer(&ucmp);
  // level 1
  AddFile(1, 2100, 2200);
  AddFile(1, 2300, 2400);
  AddFile(1, 2500, 2600);
  // level 2
  AddFile(2, 1500, 1600);
  AddFile(2, 1501, 1699);
  AddFile(2, 1700, 1800);
  // level 3
  AddFile(3, 500, 600);
  AddFile(3, 501, 699);
  AddFile(3, 700, 800);
  indexer->UpdateIndex(&arena, kNumLevels, files);
  for (uint32_t level = 1; level < 3; ++level) {
    for (uint32_t f = 0; f < 3; ++f) {
      GetNextLevelIndex(level, f, -1, -1, &left, &right);
      ASSERT_EQ(f == 0 ? 0 : 3, left);
      ASSERT_EQ(2, right);
      GetNextLevelIndex(level, f, 0, -1, &left, &right);
      ASSERT_EQ(3, left);
      ASSERT_EQ(2, right);
      GetNextLevelIndex(level, f, 1, -1, &left, &right);
      ASSERT_EQ(3, left);
      ASSERT_EQ(2, right);
      GetNextLevelIndex(level, f, 1, -1, &left, &right);
      ASSERT_EQ(3, left);
      ASSERT_EQ(2, right);
      GetNextLevelIndex(level, f, 1, 0, &left, &right);
      ASSERT_EQ(3, left);
      ASSERT_EQ(2, right);
      GetNextLevelIndex(level, f, 1, 1, &left, &right);
      ASSERT_EQ(3, left);
      ASSERT_EQ(2, right);
    }
  }
  delete indexer;
}

// Case 3: empty L2
TEST_F(FileIndexerTest, empty_L2) {
  Arena arena;
  indexer = new FileIndexer(&ucmp);
  for (uint32_t i = 1; i < kNumLevels; ++i) {
    ASSERT_EQ(0U, indexer->LevelIndexSize(i));
  }
  // level 1
  AddFile(1, 2100, 2200);
  AddFile(1, 2300, 2400);
  AddFile(1, 2500, 2600);
  // level 3
  AddFile(3, 500, 600);
  AddFile(3, 501, 699);
  AddFile(3, 700, 800);
  indexer->UpdateIndex(&arena, kNumLevels, files);
  for (uint32_t f = 0; f < 3; ++f) {
    GetNextLevelIndex(1, f, -1, -1, &left, &right);
    ASSERT_EQ(0, left);
    ASSERT_EQ(-1, right);
    GetNextLevelIndex(1, f, 0, -1, &left, &right);
    ASSERT_EQ(0, left);
    ASSERT_EQ(-1, right);
    GetNextLevelIndex(1, f, 1, -1, &left, &right);
    ASSERT_EQ(0, left);
    ASSERT_EQ(-1, right);
    GetNextLevelIndex(1, f, 1, -1, &left, &right);
    ASSERT_EQ(0, left);
    ASSERT_EQ(-1, right);
    GetNextLevelIndex(1, f, 1, 0, &left, &right);
    ASSERT_EQ(0, left);
    ASSERT_EQ(-1, right);
    GetNextLevelIndex(1, f, 1, 1, &left, &right);
    ASSERT_EQ(0, left);
    ASSERT_EQ(-1, right);
  }
  delete indexer;
  ClearFiles();
}

// Case 4: mixed
TEST_F(FileIndexerTest, mixed) {
  Arena arena;
  indexer = new FileIndexer(&ucmp);
  // level 1
  AddFile(1, 100, 200);
  AddFile(1, 250, 400);
  AddFile(1, 450, 500);
  // level 2
  AddFile(2, 100, 150);  // 0
  AddFile(2, 200, 250);  // 1
  AddFile(2, 251, 300);  // 2
  AddFile(2, 301, 350);  // 3
  AddFile(2, 500, 600);  // 4
  // level 3
  AddFile(3, 0, 50);
  AddFile(3, 100, 200);
  AddFile(3, 201, 250);
  indexer->UpdateIndex(&arena, kNumLevels, files);
  // level 1, 0
  GetNextLevelIndex(1, 0, -1, -1, &left, &right);
  ASSERT_EQ(0, left);
  ASSERT_EQ(0, right);
  GetNextLevelIndex(1, 0, 0, -1, &left, &right);
  ASSERT_EQ(0, left);
  ASSERT_EQ(0, right);
  GetNextLevelIndex(1, 0, 1, -1, &left, &right);
  ASSERT_EQ(0, left);
  ASSERT_EQ(1, right);
  GetNextLevelIndex(1, 0, 1, 0, &left, &right);
  ASSERT_EQ(1, left);
  ASSERT_EQ(1, right);
  GetNextLevelIndex(1, 0, 1, 1, &left, &right);
  ASSERT_EQ(1, left);
  ASSERT_EQ(4, right);
  // level 1, 1
  GetNextLevelIndex(1, 1, -1, -1, &left, &right);
  ASSERT_EQ(1, left);
  ASSERT_EQ(1, right);
  GetNextLevelIndex(1, 1, 0, -1, &left, &right);
  ASSERT_EQ(1, left);
  ASSERT_EQ(1, right);
  GetNextLevelIndex(1, 1, 1, -1, &left, &right);
  ASSERT_EQ(1, left);
  ASSERT_EQ(3, right);
  GetNextLevelIndex(1, 1, 1, 0, &left, &right);
  ASSERT_EQ(4, left);
  ASSERT_EQ(3, right);
  GetNextLevelIndex(1, 1, 1, 1, &left, &right);
  ASSERT_EQ(4, left);
  ASSERT_EQ(4, right);
  // level 1, 2
  GetNextLevelIndex(1, 2, -1, -1, &left, &right);
  ASSERT_EQ(4, left);
  ASSERT_EQ(3, right);
  GetNextLevelIndex(1, 2, 0, -1, &left, &right);
  ASSERT_EQ(4, left);
  ASSERT_EQ(3, right);
  GetNextLevelIndex(1, 2, 1, -1, &left, &right);
  ASSERT_EQ(4, left);
  ASSERT_EQ(4, right);
  GetNextLevelIndex(1, 2, 1, 0, &left, &right);
  ASSERT_EQ(4, left);
  ASSERT_EQ(4, right);
  GetNextLevelIndex(1, 2, 1, 1, &left, &right);
  ASSERT_EQ(4, left);
  ASSERT_EQ(4, right);
  // level 2, 0
  GetNextLevelIndex(2, 0, -1, -1, &left, &right);
  ASSERT_EQ(0, left);
  ASSERT_EQ(1, right);
  GetNextLevelIndex(2, 0, 0, -1, &left, &right);
  ASSERT_EQ(1, left);
  ASSERT_EQ(1, right);
  GetNextLevelIndex(2, 0, 1, -1, &left, &right);
  ASSERT_EQ(1, left);
  ASSERT_EQ(1, right);
  GetNextLevelIndex(2, 0, 1, 0, &left, &right);
  ASSERT_EQ(1, left);
  ASSERT_EQ(1, right);
  GetNextLevelIndex(2, 0, 1, 1, &left, &right);
  ASSERT_EQ(1, left);
  ASSERT_EQ(2, right);
  // level 2, 1
  GetNextLevelIndex(2, 1, -1, -1, &left, &right);
  ASSERT_EQ(1, left);
  ASSERT_EQ(1, right);
  GetNextLevelIndex(2, 1, 0, -1, &left, &right);
  ASSERT_EQ(1, left);
  ASSERT_EQ(1, right);
  GetNextLevelIndex(2, 1, 1, -1, &left, &right);
  ASSERT_EQ(1, left);
  ASSERT_EQ(2, right);
  GetNextLevelIndex(2, 1, 1, 0, &left, &right);
  ASSERT_EQ(2, left);
  ASSERT_EQ(2, right);
  GetNextLevelIndex(2, 1, 1, 1, &left, &right);
  ASSERT_EQ(2, left);
  ASSERT_EQ(2, right);
  // level 2, [2 - 4], no overlap
  for (uint32_t f = 2; f <= 4; ++f) {
    GetNextLevelIndex(2, f, -1, -1, &left, &right);
    ASSERT_EQ(f == 2 ? 2 : 3, left);
    ASSERT_EQ(2, right);
    GetNextLevelIndex(2, f, 0, -1, &left, &right);
    ASSERT_EQ(3, left);
    ASSERT_EQ(2, right);
    GetNextLevelIndex(2, f, 1, -1, &left, &right);
    ASSERT_EQ(3, left);
    ASSERT_EQ(2, right);
    GetNextLevelIndex(2, f, 1, 0, &left, &right);
    ASSERT_EQ(3, left);
    ASSERT_EQ(2, right);
    GetNextLevelIndex(2, f, 1, 1, &left, &right);
    ASSERT_EQ(3, left);
    ASSERT_EQ(2, right);
  }
  delete indexer;
  ClearFiles();
}

}  // namespace rocksdb

int main(int argc, char** argv) {
  rocksdb::port::InstallStackTraceHandler();
  ::testing::InitGoogleTest(&argc, argv);
  return RUN_ALL_TESTS();
}
