//  Copyright (c) 2014, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.

#pragma once

#ifndef ROCKSDB_LITE

#include <string>
#include <vector>

#include "db/dbformat.h"
#include "rocksdb/options.h"
#include "util/murmurhash.h"
#include "util/hash.h"
#include "util/arena.h"
#include "util/histogram.h"

namespace rocksdb {

// PlainTableIndex contains buckets size of index_size_, each is a
// 32-bit integer. The lower 31 bits contain an offset value (explained below)
// and the first bit of the integer indicates type of the offset.
//
// +--------------+------------------------------------------------------+
// | Flag (1 bit) | Offset to binary search buffer or file (31 bits)     +
// +--------------+------------------------------------------------------+
//
// Explanation for the "flag bit":
//
// 0 indicates that the bucket contains only one prefix (no conflict when
//   hashing this prefix), whose first row starts from this offset of the
// file.
// 1 indicates that the bucket contains more than one prefixes, or there
//   are too many rows for one prefix so we need a binary search for it. In
//   this case, the offset indicates the offset of sub_index_ holding the
//   binary search indexes of keys for those rows. Those binary search indexes
//   are organized in this way:
//
// The first 4 bytes, indicate how many indexes (N) are stored after it. After
// it, there are N 32-bit integers, each points of an offset of the file,
// which
// points to starting of a row. Those offsets need to be guaranteed to be in
// ascending order so the keys they are pointing to are also in ascending
// order
// to make sure we can use them to do binary searches. Below is visual
// presentation of a bucket.
//
// <begin>
//   number_of_records:  varint32
//   record 1 file offset:  fixedint32
//   record 2 file offset:  fixedint32
//    ....
//   record N file offset:  fixedint32
// <end>
class PlainTableIndex {
 public:
  enum IndexSearchResult {
    kNoPrefixForBucket = 0,
    kDirectToFile = 1,
    kSubindex = 2
  };

  explicit PlainTableIndex(Slice data) { InitFromRawData(data); }

  PlainTableIndex()
      : index_size_(0),
        sub_index_size_(0),
        num_prefixes_(0),
        index_(nullptr),
        sub_index_(nullptr) {}

  IndexSearchResult GetOffset(uint32_t prefix_hash,
                              uint32_t* bucket_value) const;

  Status InitFromRawData(Slice data);

  const char* GetSubIndexBasePtrAndUpperBound(uint32_t offset,
                                              uint32_t* upper_bound) const {
    const char* index_ptr = &sub_index_[offset];
    return GetVarint32Ptr(index_ptr, index_ptr + 4, upper_bound);
  }

  uint32_t GetIndexSize() const { return index_size_; }

  uint32_t GetSubIndexSize() const { return sub_index_size_; }

  uint32_t GetNumPrefixes() const { return num_prefixes_; }

  static const uint64_t kMaxFileSize = (1u << 31) - 1;
  static const uint32_t kSubIndexMask = 0x80000000;
  static const size_t kOffsetLen = sizeof(uint32_t);

 private:
  uint32_t index_size_;
  uint32_t sub_index_size_;
  uint32_t num_prefixes_;

  uint32_t* index_;
  char* sub_index_;
};

// PlainTableIndexBuilder is used to create plain table index.
// After calling Finish(), it returns Slice, which is usually
// used either to initialize PlainTableIndex or
// to save index to sst file.
// For more details about the  index, please refer to:
// https://github.com/facebook/rocksdb/wiki/PlainTable-Format
// #wiki-in-memory-index-format
class PlainTableIndexBuilder {
 public:
  PlainTableIndexBuilder(Arena* arena, const ImmutableCFOptions& ioptions,
                         size_t index_sparseness, double hash_table_ratio,
                         size_t huge_page_tlb_size)
      : arena_(arena),
        ioptions_(ioptions),
        record_list_(kRecordsPerGroup),
        is_first_record_(true),
        due_index_(false),
        num_prefixes_(0),
        num_keys_per_prefix_(0),
        prev_key_prefix_hash_(0),
        index_sparseness_(index_sparseness),
        prefix_extractor_(ioptions.prefix_extractor),
        hash_table_ratio_(hash_table_ratio),
        huge_page_tlb_size_(huge_page_tlb_size) {}

  void AddKeyPrefix(Slice key_prefix_slice, uint32_t key_offset);

  Slice Finish();

  uint32_t GetTotalSize() const {
    return VarintLength(index_size_) + VarintLength(num_prefixes_) +
           PlainTableIndex::kOffsetLen * index_size_ + sub_index_size_;
  }

  static const std::string kPlainTableIndexBlock;

 private:
  struct IndexRecord {
    uint32_t hash;    // hash of the prefix
    uint32_t offset;  // offset of a row
    IndexRecord* next;
  };

  // Helper class to track all the index records
  class IndexRecordList {
   public:
    explicit IndexRecordList(size_t num_records_per_group)
        : kNumRecordsPerGroup(num_records_per_group),
          current_group_(nullptr),
          num_records_in_current_group_(num_records_per_group) {}

    ~IndexRecordList() {
      for (size_t i = 0; i < groups_.size(); i++) {
        delete[] groups_[i];
      }
    }

    void AddRecord(uint32_t hash, uint32_t offset);

    size_t GetNumRecords() const {
      return (groups_.size() - 1) * kNumRecordsPerGroup +
             num_records_in_current_group_;
    }
    IndexRecord* At(size_t index) {
      return &(groups_[index / kNumRecordsPerGroup]
                      [index % kNumRecordsPerGroup]);
    }

   private:
    IndexRecord* AllocateNewGroup() {
      IndexRecord* result = new IndexRecord[kNumRecordsPerGroup];
      groups_.push_back(result);
      return result;
    }

    // Each group in `groups_` contains fix-sized records (determined by
    // kNumRecordsPerGroup). Which can help us minimize the cost if resizing
    // occurs.
    const size_t kNumRecordsPerGroup;
    IndexRecord* current_group_;
    // List of arrays allocated
    std::vector<IndexRecord*> groups_;
    size_t num_records_in_current_group_;
  };

  void AllocateIndex();

  // Internal helper function to bucket index record list to hash buckets.
  void BucketizeIndexes(std::vector<IndexRecord*>* hash_to_offsets,
                        std::vector<uint32_t>* entries_per_bucket);

  // Internal helper class to fill the indexes and bloom filters to internal
  // data structures.
  Slice FillIndexes(const std::vector<IndexRecord*>& hash_to_offsets,
                    const std::vector<uint32_t>& entries_per_bucket);

  Arena* arena_;
  const ImmutableCFOptions ioptions_;
  HistogramImpl keys_per_prefix_hist_;
  IndexRecordList record_list_;
  bool is_first_record_;
  bool due_index_;
  uint32_t num_prefixes_;
  uint32_t num_keys_per_prefix_;

  uint32_t prev_key_prefix_hash_;
  size_t index_sparseness_;
  uint32_t index_size_;
  uint32_t sub_index_size_;

  const SliceTransform* prefix_extractor_;
  double hash_table_ratio_;
  size_t huge_page_tlb_size_;

  std::string prev_key_prefix_;

  static const size_t kRecordsPerGroup = 256;
};

};  // namespace rocksdb

#endif  // ROCKSDB_LITE
