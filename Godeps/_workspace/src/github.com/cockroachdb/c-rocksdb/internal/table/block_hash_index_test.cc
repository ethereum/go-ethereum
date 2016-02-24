// Copyright (c) 2013, Facebook, Inc. All rights reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.

#include <map>
#include <memory>
#include <vector>

#include "rocksdb/comparator.h"
#include "rocksdb/iterator.h"
#include "rocksdb/slice_transform.h"
#include "table/block_hash_index.h"
#include "util/testharness.h"
#include "util/testutil.h"

namespace rocksdb {

typedef std::map<std::string, std::string> Data;

class MapIterator : public Iterator {
 public:
  explicit MapIterator(const Data& data) : data_(data), pos_(data_.end()) {}

  virtual bool Valid() const override { return pos_ != data_.end(); }

  virtual void SeekToFirst() override { pos_ = data_.begin(); }

  virtual void SeekToLast() override {
    pos_ = data_.end();
    --pos_;
  }

  virtual void Seek(const Slice& target) override {
    pos_ = data_.find(target.ToString());
  }

  virtual void Next() override { ++pos_; }

  virtual void Prev() override { --pos_; }

  virtual Slice key() const override { return pos_->first; }

  virtual Slice value() const override { return pos_->second; }

  virtual Status status() const override { return Status::OK(); }

 private:
  const Data& data_;
  Data::const_iterator pos_;
};

class BlockTest : public testing::Test {};

TEST_F(BlockTest, BasicTest) {
  const size_t keys_per_block = 4;
  const size_t prefix_size = 2;
  std::vector<std::string> keys = {/* block 1 */
                                   "0101", "0102", "0103", "0201",
                                   /* block 2 */
                                   "0202", "0203", "0301", "0401",
                                   /* block 3 */
                                   "0501", "0601", "0701", "0801",
                                   /* block 4 */
                                   "0802", "0803", "0804", "0805",
                                   /* block 5 */
                                   "0806", "0807", "0808", "0809", };

  Data data_entries;
  for (const auto key : keys) {
    data_entries.insert({key, key});
  }

  Data index_entries;
  for (size_t i = 3; i < keys.size(); i += keys_per_block) {
    // simply ignore the value part
    index_entries.insert({keys[i], ""});
  }

  MapIterator data_iter(data_entries);
  MapIterator index_iter(index_entries);

  auto prefix_extractor = NewFixedPrefixTransform(prefix_size);
  std::unique_ptr<BlockHashIndex> block_hash_index(CreateBlockHashIndexOnTheFly(
      &index_iter, &data_iter, static_cast<uint32_t>(index_entries.size()),
      BytewiseComparator(), prefix_extractor));

  std::map<std::string, BlockHashIndex::RestartIndex> expected = {
      {"01xx", BlockHashIndex::RestartIndex(0, 1)},
      {"02yy", BlockHashIndex::RestartIndex(0, 2)},
      {"03zz", BlockHashIndex::RestartIndex(1, 1)},
      {"04pp", BlockHashIndex::RestartIndex(1, 1)},
      {"05ww", BlockHashIndex::RestartIndex(2, 1)},
      {"06xx", BlockHashIndex::RestartIndex(2, 1)},
      {"07pp", BlockHashIndex::RestartIndex(2, 1)},
      {"08xz", BlockHashIndex::RestartIndex(2, 3)}, };

  const BlockHashIndex::RestartIndex* index = nullptr;
  // search existed prefixes
  for (const auto& item : expected) {
    index = block_hash_index->GetRestartIndex(item.first);
    ASSERT_TRUE(index != nullptr);
    ASSERT_EQ(item.second.first_index, index->first_index);
    ASSERT_EQ(item.second.num_blocks, index->num_blocks);
  }

  // search non exist prefixes
  ASSERT_TRUE(!block_hash_index->GetRestartIndex("00xx"));
  ASSERT_TRUE(!block_hash_index->GetRestartIndex("10yy"));
  ASSERT_TRUE(!block_hash_index->GetRestartIndex("20zz"));

  delete prefix_extractor;
}

}  // namespace rocksdb

int main(int argc, char** argv) {
  ::testing::InitGoogleTest(&argc, argv);
  return RUN_ALL_TESTS();
}
