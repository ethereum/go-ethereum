//  Copyright (c) 2014, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.

#include "table/bloom_block.h"

#include <string>
#include "rocksdb/slice.h"
#include "util/dynamic_bloom.h"

namespace rocksdb {

void BloomBlockBuilder::AddKeysHashes(const std::vector<uint32_t>& keys_hashes) {
  for (auto hash : keys_hashes) {
    bloom_.AddHash(hash);
  }
}

Slice BloomBlockBuilder::Finish() { return bloom_.GetRawData(); }

const std::string BloomBlockBuilder::kBloomBlock = "kBloomBlock";
}  // namespace rocksdb
