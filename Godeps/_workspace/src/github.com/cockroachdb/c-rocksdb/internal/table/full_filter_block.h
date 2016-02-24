//  Copyright (c) 2014, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.

#pragma once

#include <stddef.h>
#include <stdint.h>
#include <memory>
#include <string>
#include <vector>
#include "rocksdb/options.h"
#include "rocksdb/slice.h"
#include "rocksdb/slice_transform.h"
#include "db/dbformat.h"
#include "util/hash.h"
#include "table/filter_block.h"

namespace rocksdb {

class FilterPolicy;
class FilterBitsBuilder;
class FilterBitsReader;

// A FullFilterBlockBuilder is used to construct a full filter for a
// particular Table.  It generates a single string which is stored as
// a special block in the Table.
// The format of full filter block is:
// +----------------------------------------------------------------+
// |              full filter for all keys in sst file              |
// +----------------------------------------------------------------+
// The full filter can be very large. At the end of it, we put
// num_probes: how many hash functions are used in bloom filter
//
class FullFilterBlockBuilder : public FilterBlockBuilder {
 public:
  explicit FullFilterBlockBuilder(const SliceTransform* prefix_extractor,
                                  bool whole_key_filtering,
                                  FilterBitsBuilder* filter_bits_builder);
  // bits_builder is created in filter_policy, it should be passed in here
  // directly. and be deleted here
  ~FullFilterBlockBuilder() {}

  virtual bool IsBlockBased() override { return false; }
  virtual void StartBlock(uint64_t block_offset) override {}
  virtual void Add(const Slice& key) override;
  virtual Slice Finish() override;

 private:
  // important: all of these might point to invalid addresses
  // at the time of destruction of this filter block. destructor
  // should NOT dereference them.
  const SliceTransform* prefix_extractor_;
  bool whole_key_filtering_;

  uint32_t num_added_;
  std::unique_ptr<FilterBitsBuilder> filter_bits_builder_;
  std::unique_ptr<const char[]> filter_data_;

  void AddKey(const Slice& key);
  void AddPrefix(const Slice& key);

  // No copying allowed
  FullFilterBlockBuilder(const FullFilterBlockBuilder&);
  void operator=(const FullFilterBlockBuilder&);
};

// A FilterBlockReader is used to parse filter from SST table.
// KeyMayMatch and PrefixMayMatch would trigger filter checking
class FullFilterBlockReader : public FilterBlockReader {
 public:
  // REQUIRES: "contents" and filter_bits_reader must stay live
  // while *this is live.
  explicit FullFilterBlockReader(const SliceTransform* prefix_extractor,
                                 bool whole_key_filtering,
                                 const Slice& contents,
                                 FilterBitsReader* filter_bits_reader);
  explicit FullFilterBlockReader(const SliceTransform* prefix_extractor,
                                 bool whole_key_filtering,
                                 BlockContents&& contents,
                                 FilterBitsReader* filter_bits_reader);

  // bits_reader is created in filter_policy, it should be passed in here
  // directly. and be deleted here
  ~FullFilterBlockReader() {}

  virtual bool IsBlockBased() override { return false; }
  virtual bool KeyMayMatch(const Slice& key,
                           uint64_t block_offset = kNotValid) override;
  virtual bool PrefixMayMatch(const Slice& prefix,
                              uint64_t block_offset = kNotValid) override;
  virtual size_t ApproximateMemoryUsage() const override;

 private:
  const SliceTransform* prefix_extractor_;
  bool whole_key_filtering_;

  std::unique_ptr<FilterBitsReader> filter_bits_reader_;
  Slice contents_;
  BlockContents block_contents_;
  std::unique_ptr<const char[]> filter_data_;

  bool MayMatch(const Slice& entry);

  // No copying allowed
  FullFilterBlockReader(const FullFilterBlockReader&);
  void operator=(const FullFilterBlockReader&);
};

}  // namespace rocksdb
