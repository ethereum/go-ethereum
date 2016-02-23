// Copyright (c) 2014, Facebook, Inc. All rights reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.

#include "table/block_prefix_index.h"

#include <vector>

#include "rocksdb/comparator.h"
#include "rocksdb/slice.h"
#include "rocksdb/slice_transform.h"
#include "util/arena.h"
#include "util/coding.h"
#include "util/hash.h"

namespace rocksdb {

inline uint32_t Hash(const Slice& s) {
  return rocksdb::Hash(s.data(), s.size(), 0);
}

inline uint32_t PrefixToBucket(const Slice& prefix, uint32_t num_buckets) {
  return Hash(prefix) % num_buckets;
}

// The prefix block index is simply a bucket array, with each entry pointing to
// the blocks that span the prefixes hashed to this bucket.
//
// To reduce memory footprint, if there is only one block per bucket, the entry
// stores the block id directly. If there are more than one blocks per bucket,
// because of hash collision or a single prefix spanning multiple blocks,
// the entry points to an array of block ids. The block array is an array of
// uint32_t's. The first uint32_t indicates the total number of blocks, followed
// by the block ids.
//
// To differentiate the two cases, the high order bit of the entry indicates
// whether it is a 'pointer' into a separate block array.
// 0x7FFFFFFF is reserved for empty bucket.

const uint32_t kNoneBlock = 0x7FFFFFFF;
const uint32_t kBlockArrayMask = 0x80000000;

inline bool IsNone(uint32_t block_id) {
  return block_id == kNoneBlock;
}

inline bool IsBlockId(uint32_t block_id) {
  return (block_id & kBlockArrayMask) == 0;
}

inline uint32_t DecodeIndex(uint32_t block_id) {
  uint32_t index = block_id ^ kBlockArrayMask;
  assert(index < kBlockArrayMask);
  return index;
}

inline uint32_t EncodeIndex(uint32_t index) {
  assert(index < kBlockArrayMask);
  return index | kBlockArrayMask;
}

// temporary storage for prefix information during index building
struct PrefixRecord {
  Slice prefix;
  uint32_t start_block;
  uint32_t end_block;
  uint32_t num_blocks;
  PrefixRecord* next;
};

class BlockPrefixIndex::Builder {
 public:
  explicit Builder(const SliceTransform* internal_prefix_extractor)
      : internal_prefix_extractor_(internal_prefix_extractor) {}

  void Add(const Slice& key_prefix, uint32_t start_block,
           uint32_t num_blocks) {
    PrefixRecord* record = reinterpret_cast<PrefixRecord*>(
      arena_.AllocateAligned(sizeof(PrefixRecord)));
    record->prefix = key_prefix;
    record->start_block = start_block;
    record->end_block = start_block + num_blocks - 1;
    record->num_blocks = num_blocks;
    prefixes_.push_back(record);
  }

  BlockPrefixIndex* Finish() {
    // For now, use roughly 1:1 prefix to bucket ratio.
    uint32_t num_buckets = static_cast<uint32_t>(prefixes_.size()) + 1;

    // Collect prefix records that hash to the same bucket, into a single
    // linklist.
    std::vector<PrefixRecord*> prefixes_per_bucket(num_buckets, nullptr);
    std::vector<uint32_t> num_blocks_per_bucket(num_buckets, 0);
    for (PrefixRecord* current : prefixes_) {
      uint32_t bucket = PrefixToBucket(current->prefix, num_buckets);
      // merge the prefix block span if the first block of this prefix is
      // connected to the last block of the previous prefix.
      PrefixRecord* prev = prefixes_per_bucket[bucket];
      if (prev) {
        assert(current->start_block >= prev->end_block);
        auto distance = current->start_block - prev->end_block;
        if (distance <= 1) {
          prev->end_block = current->end_block;
          prev->num_blocks = prev->end_block - prev->start_block + 1;
          num_blocks_per_bucket[bucket] += (current->num_blocks + distance - 1);
          continue;
        }
      }
      current->next = prev;
      prefixes_per_bucket[bucket] = current;
      num_blocks_per_bucket[bucket] += current->num_blocks;
    }

    // Calculate the block array buffer size
    uint32_t total_block_array_entries = 0;
    for (uint32_t i = 0; i < num_buckets; i++) {
      uint32_t num_blocks = num_blocks_per_bucket[i];
      if (num_blocks > 1) {
        total_block_array_entries += (num_blocks + 1);
      }
    }

    // Populate the final prefix block index
    uint32_t* block_array_buffer = new uint32_t[total_block_array_entries];
    uint32_t* buckets = new uint32_t[num_buckets];
    uint32_t offset = 0;
    for (uint32_t i = 0; i < num_buckets; i++) {
      uint32_t num_blocks = num_blocks_per_bucket[i];
      if (num_blocks == 0) {
        assert(prefixes_per_bucket[i] == nullptr);
        buckets[i] = kNoneBlock;
      } else if (num_blocks == 1) {
        assert(prefixes_per_bucket[i] != nullptr);
        assert(prefixes_per_bucket[i]->next == nullptr);
        buckets[i] = prefixes_per_bucket[i]->start_block;
      } else {
        assert(prefixes_per_bucket[i] != nullptr);
        buckets[i] = EncodeIndex(offset);
        block_array_buffer[offset] = num_blocks;
        uint32_t* last_block = &block_array_buffer[offset + num_blocks];
        auto current = prefixes_per_bucket[i];
        // populate block ids from largest to smallest
        while (current != nullptr) {
          for (uint32_t iter = 0; iter < current->num_blocks; iter++) {
            *last_block = current->end_block - iter;
            last_block--;
          }
          current = current->next;
        }
        assert(last_block == &block_array_buffer[offset]);
        offset += (num_blocks + 1);
      }
    }

    assert(offset == total_block_array_entries);

    return new BlockPrefixIndex(internal_prefix_extractor_, num_buckets,
                                buckets, total_block_array_entries,
                                block_array_buffer);
  }

 private:
  const SliceTransform* internal_prefix_extractor_;

  std::vector<PrefixRecord*> prefixes_;
  Arena arena_;
};


Status BlockPrefixIndex::Create(const SliceTransform* internal_prefix_extractor,
                                const Slice& prefixes, const Slice& prefix_meta,
                                BlockPrefixIndex** prefix_index) {
  uint64_t pos = 0;
  auto meta_pos = prefix_meta;
  Status s;
  Builder builder(internal_prefix_extractor);

  while (!meta_pos.empty()) {
    uint32_t prefix_size = 0;
    uint32_t entry_index = 0;
    uint32_t num_blocks = 0;
    if (!GetVarint32(&meta_pos, &prefix_size) ||
        !GetVarint32(&meta_pos, &entry_index) ||
        !GetVarint32(&meta_pos, &num_blocks)) {
      s = Status::Corruption(
          "Corrupted prefix meta block: unable to read from it.");
      break;
    }
    if (pos + prefix_size > prefixes.size()) {
      s = Status::Corruption(
        "Corrupted prefix meta block: size inconsistency.");
      break;
    }
    Slice prefix(prefixes.data() + pos, prefix_size);
    builder.Add(prefix, entry_index, num_blocks);

    pos += prefix_size;
  }

  if (s.ok() && pos != prefixes.size()) {
    s = Status::Corruption("Corrupted prefix meta block");
  }

  if (s.ok()) {
    *prefix_index = builder.Finish();
  }

  return s;
}

uint32_t BlockPrefixIndex::GetBlocks(const Slice& key,
                                     uint32_t** blocks) {
  Slice prefix = internal_prefix_extractor_->Transform(key);

  uint32_t bucket = PrefixToBucket(prefix, num_buckets_);
  uint32_t block_id = buckets_[bucket];

  if (IsNone(block_id)) {
    return 0;
  } else if (IsBlockId(block_id)) {
    *blocks = &buckets_[bucket];
    return 1;
  } else {
    uint32_t index = DecodeIndex(block_id);
    assert(index < num_block_array_buffer_entries_);
    *blocks = &block_array_buffer_[index+1];
    uint32_t num_blocks = block_array_buffer_[index];
    assert(num_blocks > 1);
    assert(index + num_blocks < num_block_array_buffer_entries_);
    return num_blocks;
  }
}

}  // namespace rocksdb
