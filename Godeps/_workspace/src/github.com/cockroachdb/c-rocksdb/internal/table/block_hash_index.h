// Copyright (c) 2013, Facebook, Inc. All rights reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.
#pragma once

#include <string>
#include <unordered_map>

#include "rocksdb/status.h"
#include "util/arena.h"
#include "util/murmurhash.h"

namespace rocksdb {

class Comparator;
class Iterator;
class Slice;
class SliceTransform;

// Build a hash-based index to speed up the lookup for "index block".
// BlockHashIndex accepts a key and, if found, returns its restart index within
// that index block.
class BlockHashIndex {
 public:
  // Represents a restart index in the index block's restart array.
  struct RestartIndex {
    explicit RestartIndex(uint32_t _first_index, uint32_t _num_blocks = 1)
        : first_index(_first_index), num_blocks(_num_blocks) {}

    // For a given prefix, what is the restart index for the first data block
    // that contains it.
    uint32_t first_index = 0;

    // How many data blocks contains this prefix?
    uint32_t num_blocks = 1;
  };

  // @params own_prefixes indicate if we should take care the memory space for
  // the `key_prefix`
  // passed by Add()
  explicit BlockHashIndex(const SliceTransform* hash_key_extractor,
                          bool own_prefixes)
      : hash_key_extractor_(hash_key_extractor), kOwnPrefixes(own_prefixes) {}

  // Maps a key to its restart first_index.
  // Returns nullptr if the restart first_index is found
  const RestartIndex* GetRestartIndex(const Slice& key);

  bool Add(const Slice& key_prefix, uint32_t restart_index,
           uint32_t num_blocks);

  size_t ApproximateMemoryUsage() const {
    return arena_.ApproximateMemoryUsage();
  }

 private:
  const SliceTransform* hash_key_extractor_;
  std::unordered_map<Slice, RestartIndex, murmur_hash> restart_indices_;

  Arena arena_;
  bool kOwnPrefixes;
};

// Create hash index by reading from the metadata blocks.
// @params prefixes: a sequence of prefixes.
// @params prefix_meta: contains the "metadata" to of the prefixes.
Status CreateBlockHashIndex(const SliceTransform* hash_key_extractor,
                            const Slice& prefixes, const Slice& prefix_meta,
                            BlockHashIndex** hash_index);

// Create hash index by scanning the entries in index as well as the whole
// dataset.
// @params index_iter: an iterator with the pointer to the first entry in a
//                     block.
// @params data_iter: an iterator that can scan all the entries reside in a
//                     table.
// @params num_restarts: used for correctness verification.
// @params hash_key_extractor: extract the hashable part of a given key.
// On error, nullptr will be returned.
BlockHashIndex* CreateBlockHashIndexOnTheFly(
    Iterator* index_iter, Iterator* data_iter, const uint32_t num_restarts,
    const Comparator* comparator, const SliceTransform* hash_key_extractor);

}  // namespace rocksdb
