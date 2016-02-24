// Copyright (c) 2013, Facebook, Inc.  All rights reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.
// Copyright (c) 2011 The LevelDB Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file. See the AUTHORS file for names of contributors.

#ifndef ROCKSDB_LITE
#pragma once
#include "rocksdb/slice_transform.h"
#include "rocksdb/memtablerep.h"

namespace rocksdb {

class HashCuckooRepFactory : public MemTableRepFactory {
 public:
  // maxinum number of hash functions used in the cuckoo hash.
  static const unsigned int kMaxHashCount = 10;

  explicit HashCuckooRepFactory(size_t write_buffer_size,
                                size_t average_data_size,
                                unsigned int hash_function_count)
      : write_buffer_size_(write_buffer_size),
        average_data_size_(average_data_size),
        hash_function_count_(hash_function_count) {}

  virtual ~HashCuckooRepFactory() {}

  virtual MemTableRep* CreateMemTableRep(
      const MemTableRep::KeyComparator& compare, MemTableAllocator* allocator,
      const SliceTransform* transform, Logger* logger) override;

  virtual const char* Name() const override { return "HashCuckooRepFactory"; }

 private:
  size_t write_buffer_size_;
  size_t average_data_size_;
  const unsigned int hash_function_count_;
};
}  // namespace rocksdb
#endif  // ROCKSDB_LITE
