// Copyright (c) 2013, Facebook, Inc. All rights reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.

#include "table/block_hash_index.h"

#include <algorithm>

#include "rocksdb/comparator.h"
#include "rocksdb/iterator.h"
#include "rocksdb/slice_transform.h"
#include "util/coding.h"

namespace rocksdb {

Status CreateBlockHashIndex(const SliceTransform* hash_key_extractor,
                            const Slice& prefixes, const Slice& prefix_meta,
                            BlockHashIndex** hash_index) {
  uint64_t pos = 0;
  auto meta_pos = prefix_meta;
  Status s;
  *hash_index = new BlockHashIndex(
      hash_key_extractor,
      false /* external module manages memory space for prefixes */);

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
    Slice prefix(prefixes.data() + pos, prefix_size);
    (*hash_index)->Add(prefix, entry_index, num_blocks);

    pos += prefix_size;
  }

  if (s.ok() && pos != prefixes.size()) {
    s = Status::Corruption("Corrupted prefix meta block");
  }

  if (!s.ok()) {
    delete *hash_index;
  }

  return s;
}

BlockHashIndex* CreateBlockHashIndexOnTheFly(
    Iterator* index_iter, Iterator* data_iter, const uint32_t num_restarts,
    const Comparator* comparator, const SliceTransform* hash_key_extractor) {
  assert(hash_key_extractor);
  auto hash_index = new BlockHashIndex(
      hash_key_extractor,
      true /* hash_index will copy prefix when Add() is called */);
  uint32_t current_restart_index = 0;

  std::string pending_entry_prefix;
  // pending_block_num == 0 also implies there is no entry inserted at all.
  uint32_t pending_block_num = 0;
  uint32_t pending_entry_index = 0;

  // scan all the entries and create a hash index based on their prefixes.
  data_iter->SeekToFirst();
  for (index_iter->SeekToFirst();
       index_iter->Valid() && current_restart_index < num_restarts;
       index_iter->Next()) {
    Slice last_key_in_block = index_iter->key();
    assert(data_iter->Valid() && data_iter->status().ok());

    // scan through all entries within a data block.
    while (data_iter->Valid() &&
           comparator->Compare(data_iter->key(), last_key_in_block) <= 0) {
      auto key_prefix = hash_key_extractor->Transform(data_iter->key());
      bool is_first_entry = pending_block_num == 0;

      // Keys may share the prefix
      if (is_first_entry || pending_entry_prefix != key_prefix) {
        if (!is_first_entry) {
          bool succeeded = hash_index->Add(
              pending_entry_prefix, pending_entry_index, pending_block_num);
          if (!succeeded) {
            delete hash_index;
            return nullptr;
          }
        }

        // update the status.
        // needs a hard copy otherwise the underlying data changes all the time.
        pending_entry_prefix = key_prefix.ToString();
        pending_block_num = 1;
        pending_entry_index = current_restart_index;
      } else {
        // entry number increments when keys share the prefix reside in
        // different data blocks.
        auto last_restart_index = pending_entry_index + pending_block_num - 1;
        assert(last_restart_index <= current_restart_index);
        if (last_restart_index != current_restart_index) {
          ++pending_block_num;
        }
      }
      data_iter->Next();
    }

    ++current_restart_index;
  }

  // make sure all entries has been scaned.
  assert(!index_iter->Valid());
  assert(!data_iter->Valid());

  if (pending_block_num > 0) {
    auto succeeded = hash_index->Add(pending_entry_prefix, pending_entry_index,
                                     pending_block_num);
    if (!succeeded) {
      delete hash_index;
      return nullptr;
    }
  }

  return hash_index;
}

bool BlockHashIndex::Add(const Slice& prefix, uint32_t restart_index,
                         uint32_t num_blocks) {
  auto prefix_to_insert = prefix;
  if (kOwnPrefixes) {
    auto prefix_ptr = arena_.Allocate(prefix.size());
    // MSVC reports C4996 Function call with parameters that may be
    // unsafe when using std::copy with a output iterator - pointer
    memcpy(prefix_ptr, prefix.data(), prefix.size());
    prefix_to_insert = Slice(prefix_ptr, prefix.size());
  }
  auto result = restart_indices_.insert(
      {prefix_to_insert, RestartIndex(restart_index, num_blocks)});
  return result.second;
}

const BlockHashIndex::RestartIndex* BlockHashIndex::GetRestartIndex(
    const Slice& key) {
  auto key_prefix = hash_key_extractor_->Transform(key);

  auto pos = restart_indices_.find(key_prefix);
  if (pos == restart_indices_.end()) {
    return nullptr;
  }

  return &pos->second;
}

}  // namespace rocksdb
