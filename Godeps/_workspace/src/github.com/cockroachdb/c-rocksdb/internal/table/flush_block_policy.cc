//  Copyright (c) 2013, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.

#include "rocksdb/options.h"
#include "rocksdb/flush_block_policy.h"
#include "rocksdb/slice.h"
#include "table/block_builder.h"

#include <cassert>

namespace rocksdb {

// Flush block by size
class FlushBlockBySizePolicy : public FlushBlockPolicy {
 public:
  // @params block_size:           Approximate size of user data packed per
  //                               block.
  // @params block_size_deviation: This is used to close a block before it
  //                               reaches the configured
  FlushBlockBySizePolicy(const uint64_t block_size,
                         const uint64_t block_size_deviation,
                         const BlockBuilder& data_block_builder) :
      block_size_(block_size),
      block_size_deviation_(block_size_deviation),
      data_block_builder_(data_block_builder) {
  }

  virtual bool Update(const Slice& key,
                      const Slice& value) override {
    // it makes no sense to flush when the data block is empty
    if (data_block_builder_.empty()) {
      return false;
    }

    auto curr_size = data_block_builder_.CurrentSizeEstimate();

    // Do flush if one of the below two conditions is true:
    // 1) if the current estimated size already exceeds the block size,
    // 2) block_size_deviation is set and the estimated size after appending
    // the kv will exceed the block size and the current size is under the
    // the deviation.
    return curr_size >= block_size_ || BlockAlmostFull(key, value);
  }

 private:
  bool BlockAlmostFull(const Slice& key, const Slice& value) const {
    const auto curr_size = data_block_builder_.CurrentSizeEstimate();
    const auto estimated_size_after =
      data_block_builder_.EstimateSizeAfterKV(key, value);

    return
      estimated_size_after > block_size_ &&
      block_size_deviation_ > 0 &&
      curr_size * 100 > block_size_ * (100 - block_size_deviation_);
  }

  const uint64_t block_size_;
  const uint64_t block_size_deviation_;
  const BlockBuilder& data_block_builder_;
};

FlushBlockPolicy* FlushBlockBySizePolicyFactory::NewFlushBlockPolicy(
    const BlockBasedTableOptions& table_options,
    const BlockBuilder& data_block_builder) const {
  return new FlushBlockBySizePolicy(
      table_options.block_size, table_options.block_size_deviation,
      data_block_builder);
}

}  // namespace rocksdb
