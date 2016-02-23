//  Copyright (c) 2013, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.
//
// Copyright (c) 2012 The LevelDB Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file. See the AUTHORS file for names of contributors.

#include "table/block_based_filter_block.h"

#include "rocksdb/filter_policy.h"
#include "util/coding.h"
#include "util/hash.h"
#include "util/logging.h"
#include "util/testharness.h"
#include "util/testutil.h"

namespace rocksdb {

// For testing: emit an array with one hash value per key
class TestHashFilter : public FilterPolicy {
 public:
  virtual const char* Name() const override { return "TestHashFilter"; }

  virtual void CreateFilter(const Slice* keys, int n,
                            std::string* dst) const override {
    for (int i = 0; i < n; i++) {
      uint32_t h = Hash(keys[i].data(), keys[i].size(), 1);
      PutFixed32(dst, h);
    }
  }

  virtual bool KeyMayMatch(const Slice& key,
                           const Slice& filter) const override {
    uint32_t h = Hash(key.data(), key.size(), 1);
    for (unsigned int i = 0; i + 4 <= filter.size(); i += 4) {
      if (h == DecodeFixed32(filter.data() + i)) {
        return true;
      }
    }
    return false;
  }
};

class FilterBlockTest : public testing::Test {
 public:
  TestHashFilter policy_;
  BlockBasedTableOptions table_options_;

  FilterBlockTest() {
    table_options_.filter_policy.reset(new TestHashFilter());
  }
};

TEST_F(FilterBlockTest, EmptyBuilder) {
  BlockBasedFilterBlockBuilder builder(nullptr, table_options_);
  BlockContents block(builder.Finish(), false, kNoCompression);
  ASSERT_EQ("\\x00\\x00\\x00\\x00\\x0b", EscapeString(block.data));
  BlockBasedFilterBlockReader reader(nullptr, table_options_, true,
                                     std::move(block));
  ASSERT_TRUE(reader.KeyMayMatch("foo", 0));
  ASSERT_TRUE(reader.KeyMayMatch("foo", 100000));
}

TEST_F(FilterBlockTest, SingleChunk) {
  BlockBasedFilterBlockBuilder builder(nullptr, table_options_);
  builder.StartBlock(100);
  builder.Add("foo");
  builder.Add("bar");
  builder.Add("box");
  builder.StartBlock(200);
  builder.Add("box");
  builder.StartBlock(300);
  builder.Add("hello");
  BlockContents block(builder.Finish(), false, kNoCompression);
  BlockBasedFilterBlockReader reader(nullptr, table_options_, true,
                                     std::move(block));
  ASSERT_TRUE(reader.KeyMayMatch("foo", 100));
  ASSERT_TRUE(reader.KeyMayMatch("bar", 100));
  ASSERT_TRUE(reader.KeyMayMatch("box", 100));
  ASSERT_TRUE(reader.KeyMayMatch("hello", 100));
  ASSERT_TRUE(reader.KeyMayMatch("foo", 100));
  ASSERT_TRUE(!reader.KeyMayMatch("missing", 100));
  ASSERT_TRUE(!reader.KeyMayMatch("other", 100));
}

TEST_F(FilterBlockTest, MultiChunk) {
  BlockBasedFilterBlockBuilder builder(nullptr, table_options_);

  // First filter
  builder.StartBlock(0);
  builder.Add("foo");
  builder.StartBlock(2000);
  builder.Add("bar");

  // Second filter
  builder.StartBlock(3100);
  builder.Add("box");

  // Third filter is empty

  // Last filter
  builder.StartBlock(9000);
  builder.Add("box");
  builder.Add("hello");

  BlockContents block(builder.Finish(), false, kNoCompression);
  BlockBasedFilterBlockReader reader(nullptr, table_options_, true,
                                     std::move(block));

  // Check first filter
  ASSERT_TRUE(reader.KeyMayMatch("foo", 0));
  ASSERT_TRUE(reader.KeyMayMatch("bar", 2000));
  ASSERT_TRUE(!reader.KeyMayMatch("box", 0));
  ASSERT_TRUE(!reader.KeyMayMatch("hello", 0));

  // Check second filter
  ASSERT_TRUE(reader.KeyMayMatch("box", 3100));
  ASSERT_TRUE(!reader.KeyMayMatch("foo", 3100));
  ASSERT_TRUE(!reader.KeyMayMatch("bar", 3100));
  ASSERT_TRUE(!reader.KeyMayMatch("hello", 3100));

  // Check third filter (empty)
  ASSERT_TRUE(!reader.KeyMayMatch("foo", 4100));
  ASSERT_TRUE(!reader.KeyMayMatch("bar", 4100));
  ASSERT_TRUE(!reader.KeyMayMatch("box", 4100));
  ASSERT_TRUE(!reader.KeyMayMatch("hello", 4100));

  // Check last filter
  ASSERT_TRUE(reader.KeyMayMatch("box", 9000));
  ASSERT_TRUE(reader.KeyMayMatch("hello", 9000));
  ASSERT_TRUE(!reader.KeyMayMatch("foo", 9000));
  ASSERT_TRUE(!reader.KeyMayMatch("bar", 9000));
}

// Test for block based filter block
// use new interface in FilterPolicy to create filter builder/reader
class BlockBasedFilterBlockTest : public testing::Test {
 public:
  BlockBasedTableOptions table_options_;

  BlockBasedFilterBlockTest() {
    table_options_.filter_policy.reset(NewBloomFilterPolicy(10));
  }

  ~BlockBasedFilterBlockTest() {}
};

TEST_F(BlockBasedFilterBlockTest, BlockBasedEmptyBuilder) {
  FilterBlockBuilder* builder = new BlockBasedFilterBlockBuilder(
      nullptr, table_options_);
  BlockContents block(builder->Finish(), false, kNoCompression);
  ASSERT_EQ("\\x00\\x00\\x00\\x00\\x0b", EscapeString(block.data));
  FilterBlockReader* reader = new BlockBasedFilterBlockReader(
      nullptr, table_options_, true, std::move(block));
  ASSERT_TRUE(reader->KeyMayMatch("foo", 0));
  ASSERT_TRUE(reader->KeyMayMatch("foo", 100000));

  delete builder;
  delete reader;
}

TEST_F(BlockBasedFilterBlockTest, BlockBasedSingleChunk) {
  FilterBlockBuilder* builder = new BlockBasedFilterBlockBuilder(
      nullptr, table_options_);
  builder->StartBlock(100);
  builder->Add("foo");
  builder->Add("bar");
  builder->Add("box");
  builder->StartBlock(200);
  builder->Add("box");
  builder->StartBlock(300);
  builder->Add("hello");
  BlockContents block(builder->Finish(), false, kNoCompression);
  FilterBlockReader* reader = new BlockBasedFilterBlockReader(
      nullptr, table_options_, true, std::move(block));
  ASSERT_TRUE(reader->KeyMayMatch("foo", 100));
  ASSERT_TRUE(reader->KeyMayMatch("bar", 100));
  ASSERT_TRUE(reader->KeyMayMatch("box", 100));
  ASSERT_TRUE(reader->KeyMayMatch("hello", 100));
  ASSERT_TRUE(reader->KeyMayMatch("foo", 100));
  ASSERT_TRUE(!reader->KeyMayMatch("missing", 100));
  ASSERT_TRUE(!reader->KeyMayMatch("other", 100));

  delete builder;
  delete reader;
}

TEST_F(BlockBasedFilterBlockTest, BlockBasedMultiChunk) {
  FilterBlockBuilder* builder = new BlockBasedFilterBlockBuilder(
      nullptr, table_options_);

  // First filter
  builder->StartBlock(0);
  builder->Add("foo");
  builder->StartBlock(2000);
  builder->Add("bar");

  // Second filter
  builder->StartBlock(3100);
  builder->Add("box");

  // Third filter is empty

  // Last filter
  builder->StartBlock(9000);
  builder->Add("box");
  builder->Add("hello");

  BlockContents block(builder->Finish(), false, kNoCompression);
  FilterBlockReader* reader = new BlockBasedFilterBlockReader(
      nullptr, table_options_, true, std::move(block));

  // Check first filter
  ASSERT_TRUE(reader->KeyMayMatch("foo", 0));
  ASSERT_TRUE(reader->KeyMayMatch("bar", 2000));
  ASSERT_TRUE(!reader->KeyMayMatch("box", 0));
  ASSERT_TRUE(!reader->KeyMayMatch("hello", 0));

  // Check second filter
  ASSERT_TRUE(reader->KeyMayMatch("box", 3100));
  ASSERT_TRUE(!reader->KeyMayMatch("foo", 3100));
  ASSERT_TRUE(!reader->KeyMayMatch("bar", 3100));
  ASSERT_TRUE(!reader->KeyMayMatch("hello", 3100));

  // Check third filter (empty)
  ASSERT_TRUE(!reader->KeyMayMatch("foo", 4100));
  ASSERT_TRUE(!reader->KeyMayMatch("bar", 4100));
  ASSERT_TRUE(!reader->KeyMayMatch("box", 4100));
  ASSERT_TRUE(!reader->KeyMayMatch("hello", 4100));

  // Check last filter
  ASSERT_TRUE(reader->KeyMayMatch("box", 9000));
  ASSERT_TRUE(reader->KeyMayMatch("hello", 9000));
  ASSERT_TRUE(!reader->KeyMayMatch("foo", 9000));
  ASSERT_TRUE(!reader->KeyMayMatch("bar", 9000));

  delete builder;
  delete reader;
}

}  // namespace rocksdb

int main(int argc, char** argv) {
  ::testing::InitGoogleTest(&argc, argv);
  return RUN_ALL_TESTS();
}
