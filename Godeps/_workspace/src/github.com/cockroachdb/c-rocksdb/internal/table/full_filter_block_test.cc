//  Copyright (c) 2014, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.

#include "table/full_filter_block.h"

#include "rocksdb/filter_policy.h"
#include "util/coding.h"
#include "util/hash.h"
#include "util/logging.h"
#include "util/testharness.h"
#include "util/testutil.h"

namespace rocksdb {

class TestFilterBitsBuilder : public FilterBitsBuilder {
 public:
  explicit TestFilterBitsBuilder() {}

  // Add Key to filter
  virtual void AddKey(const Slice& key) override {
    hash_entries_.push_back(Hash(key.data(), key.size(), 1));
  }

  // Generate the filter using the keys that are added
  virtual Slice Finish(std::unique_ptr<const char[]>* buf) override {
    uint32_t len = static_cast<uint32_t>(hash_entries_.size()) * 4;
    char* data = new char[len];
    for (size_t i = 0; i < hash_entries_.size(); i++) {
      EncodeFixed32(data + i * 4, hash_entries_[i]);
    }
    const char* const_data = data;
    buf->reset(const_data);
    return Slice(data, len);
  }

 private:
  std::vector<uint32_t> hash_entries_;
};

class TestFilterBitsReader : public FilterBitsReader {
 public:
  explicit TestFilterBitsReader(const Slice& contents)
      : data_(contents.data()), len_(static_cast<uint32_t>(contents.size())) {}

  virtual bool MayMatch(const Slice& entry) override {
    uint32_t h = Hash(entry.data(), entry.size(), 1);
    for (size_t i = 0; i + 4 <= len_; i += 4) {
      if (h == DecodeFixed32(data_ + i)) {
        return true;
      }
    }
    return false;
  }

 private:
  const char* data_;
  uint32_t len_;
};


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

  virtual FilterBitsBuilder* GetFilterBitsBuilder() const override {
    return new TestFilterBitsBuilder();
  }

  virtual FilterBitsReader* GetFilterBitsReader(const Slice& contents)
      const override {
    return new TestFilterBitsReader(contents);
  }
};

class PluginFullFilterBlockTest : public testing::Test {
 public:
  BlockBasedTableOptions table_options_;

  PluginFullFilterBlockTest() {
    table_options_.filter_policy.reset(new TestHashFilter());
  }
};

TEST_F(PluginFullFilterBlockTest, PluginEmptyBuilder) {
  FullFilterBlockBuilder builder(
      nullptr, true, table_options_.filter_policy->GetFilterBitsBuilder());
  Slice block = builder.Finish();
  ASSERT_EQ("", EscapeString(block));

  FullFilterBlockReader reader(
      nullptr, true, block,
      table_options_.filter_policy->GetFilterBitsReader(block));
  // Remain same symantic with blockbased filter
  ASSERT_TRUE(reader.KeyMayMatch("foo"));
}

TEST_F(PluginFullFilterBlockTest, PluginSingleChunk) {
  FullFilterBlockBuilder builder(
      nullptr, true, table_options_.filter_policy->GetFilterBitsBuilder());
  builder.Add("foo");
  builder.Add("bar");
  builder.Add("box");
  builder.Add("box");
  builder.Add("hello");
  Slice block = builder.Finish();
  FullFilterBlockReader reader(
      nullptr, true, block,
      table_options_.filter_policy->GetFilterBitsReader(block));
  ASSERT_TRUE(reader.KeyMayMatch("foo"));
  ASSERT_TRUE(reader.KeyMayMatch("bar"));
  ASSERT_TRUE(reader.KeyMayMatch("box"));
  ASSERT_TRUE(reader.KeyMayMatch("hello"));
  ASSERT_TRUE(reader.KeyMayMatch("foo"));
  ASSERT_TRUE(!reader.KeyMayMatch("missing"));
  ASSERT_TRUE(!reader.KeyMayMatch("other"));
}

class FullFilterBlockTest : public testing::Test {
 public:
  BlockBasedTableOptions table_options_;

  FullFilterBlockTest() {
    table_options_.filter_policy.reset(NewBloomFilterPolicy(10, false));
  }

  ~FullFilterBlockTest() {}
};

TEST_F(FullFilterBlockTest, EmptyBuilder) {
  FullFilterBlockBuilder builder(
      nullptr, true, table_options_.filter_policy->GetFilterBitsBuilder());
  Slice block = builder.Finish();
  ASSERT_EQ("", EscapeString(block));

  FullFilterBlockReader reader(
      nullptr, true, block,
      table_options_.filter_policy->GetFilterBitsReader(block));
  // Remain same symantic with blockbased filter
  ASSERT_TRUE(reader.KeyMayMatch("foo"));
}

TEST_F(FullFilterBlockTest, SingleChunk) {
  FullFilterBlockBuilder builder(
      nullptr, true, table_options_.filter_policy->GetFilterBitsBuilder());
  builder.Add("foo");
  builder.Add("bar");
  builder.Add("box");
  builder.Add("box");
  builder.Add("hello");
  Slice block = builder.Finish();
  FullFilterBlockReader reader(
      nullptr, true, block,
      table_options_.filter_policy->GetFilterBitsReader(block));
  ASSERT_TRUE(reader.KeyMayMatch("foo"));
  ASSERT_TRUE(reader.KeyMayMatch("bar"));
  ASSERT_TRUE(reader.KeyMayMatch("box"));
  ASSERT_TRUE(reader.KeyMayMatch("hello"));
  ASSERT_TRUE(reader.KeyMayMatch("foo"));
  ASSERT_TRUE(!reader.KeyMayMatch("missing"));
  ASSERT_TRUE(!reader.KeyMayMatch("other"));
}

}  // namespace rocksdb

int main(int argc, char** argv) {
  ::testing::InitGoogleTest(&argc, argv);
  return RUN_ALL_TESTS();
}
