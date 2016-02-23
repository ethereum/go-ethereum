//  Copyright (c) 2014, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.

#ifndef ROCKSDB_LITE

#ifndef __STDC_FORMAT_MACROS
#define __STDC_FORMAT_MACROS
#endif

#include <inttypes.h>

#include "table/plain_table_index.h"
#include "util/coding.h"
#include "util/hash.h"

namespace rocksdb {

namespace {
inline uint32_t GetBucketIdFromHash(uint32_t hash, uint32_t num_buckets) {
  assert(num_buckets > 0);
  return hash % num_buckets;
}
}

Status PlainTableIndex::InitFromRawData(Slice data) {
  if (!GetVarint32(&data, &index_size_)) {
    return Status::Corruption("Couldn't read the index size!");
  }
  assert(index_size_ > 0);
  if (!GetVarint32(&data, &num_prefixes_)) {
    return Status::Corruption("Couldn't read the index size!");
  }
  sub_index_size_ =
      static_cast<uint32_t>(data.size()) - index_size_ * kOffsetLen;

  char* index_data_begin = const_cast<char*>(data.data());
  index_ = reinterpret_cast<uint32_t*>(index_data_begin);
  sub_index_ = reinterpret_cast<char*>(index_ + index_size_);
  return Status::OK();
}

PlainTableIndex::IndexSearchResult PlainTableIndex::GetOffset(
    uint32_t prefix_hash, uint32_t* bucket_value) const {
  int bucket = GetBucketIdFromHash(prefix_hash, index_size_);
  *bucket_value = index_[bucket];
  if ((*bucket_value & kSubIndexMask) == kSubIndexMask) {
    *bucket_value ^= kSubIndexMask;
    return kSubindex;
  }
  if (*bucket_value >= kMaxFileSize) {
    return kNoPrefixForBucket;
  } else {
    // point directly to the file
    return kDirectToFile;
  }
}

void PlainTableIndexBuilder::IndexRecordList::AddRecord(uint32_t hash,
                                                        uint32_t offset) {
  if (num_records_in_current_group_ == kNumRecordsPerGroup) {
    current_group_ = AllocateNewGroup();
    num_records_in_current_group_ = 0;
  }
  auto& new_record = current_group_[num_records_in_current_group_++];
  new_record.hash = hash;
  new_record.offset = offset;
  new_record.next = nullptr;
}

void PlainTableIndexBuilder::AddKeyPrefix(Slice key_prefix_slice,
                                          uint32_t key_offset) {
  if (is_first_record_ || prev_key_prefix_ != key_prefix_slice.ToString()) {
    ++num_prefixes_;
    if (!is_first_record_) {
      keys_per_prefix_hist_.Add(num_keys_per_prefix_);
    }
    num_keys_per_prefix_ = 0;
    prev_key_prefix_ = key_prefix_slice.ToString();
    prev_key_prefix_hash_ = GetSliceHash(key_prefix_slice);
    due_index_ = true;
  }

  if (due_index_) {
    // Add an index key for every kIndexIntervalForSamePrefixKeys keys
    record_list_.AddRecord(prev_key_prefix_hash_, key_offset);
    due_index_ = false;
  }

  num_keys_per_prefix_++;
  if (index_sparseness_ == 0 || num_keys_per_prefix_ % index_sparseness_ == 0) {
    due_index_ = true;
  }
  is_first_record_ = false;
}

Slice PlainTableIndexBuilder::Finish() {
  AllocateIndex();
  std::vector<IndexRecord*> hash_to_offsets(index_size_, nullptr);
  std::vector<uint32_t> entries_per_bucket(index_size_, 0);
  BucketizeIndexes(&hash_to_offsets, &entries_per_bucket);

  keys_per_prefix_hist_.Add(num_keys_per_prefix_);
  Log(InfoLogLevel::INFO_LEVEL, ioptions_.info_log,
      "Number of Keys per prefix Histogram: %s",
      keys_per_prefix_hist_.ToString().c_str());

  // From the temp data structure, populate indexes.
  return FillIndexes(hash_to_offsets, entries_per_bucket);
}

void PlainTableIndexBuilder::AllocateIndex() {
  if (prefix_extractor_ == nullptr || hash_table_ratio_ <= 0) {
    // Fall back to pure binary search if the user fails to specify a prefix
    // extractor.
    index_size_ = 1;
  } else {
    double hash_table_size_multipier = 1.0 / hash_table_ratio_;
    index_size_ = num_prefixes_ * hash_table_size_multipier + 1;
    assert(index_size_ > 0);
  }
}

void PlainTableIndexBuilder::BucketizeIndexes(
    std::vector<IndexRecord*>* hash_to_offsets,
    std::vector<uint32_t>* entries_per_bucket) {
  bool first = true;
  uint32_t prev_hash = 0;
  size_t num_records = record_list_.GetNumRecords();
  for (size_t i = 0; i < num_records; i++) {
    IndexRecord* index_record = record_list_.At(i);
    uint32_t cur_hash = index_record->hash;
    if (first || prev_hash != cur_hash) {
      prev_hash = cur_hash;
      first = false;
    }
    uint32_t bucket = GetBucketIdFromHash(cur_hash, index_size_);
    IndexRecord* prev_bucket_head = (*hash_to_offsets)[bucket];
    index_record->next = prev_bucket_head;
    (*hash_to_offsets)[bucket] = index_record;
    (*entries_per_bucket)[bucket]++;
  }

  sub_index_size_ = 0;
  for (auto entry_count : *entries_per_bucket) {
    if (entry_count <= 1) {
      continue;
    }
    // Only buckets with more than 1 entry will have subindex.
    sub_index_size_ += VarintLength(entry_count);
    // total bytes needed to store these entries' in-file offsets.
    sub_index_size_ += entry_count * PlainTableIndex::kOffsetLen;
  }
}

Slice PlainTableIndexBuilder::FillIndexes(
    const std::vector<IndexRecord*>& hash_to_offsets,
    const std::vector<uint32_t>& entries_per_bucket) {
  Log(InfoLogLevel::DEBUG_LEVEL, ioptions_.info_log,
      "Reserving %" PRIu32 " bytes for plain table's sub_index",
      sub_index_size_);
  auto total_allocate_size = GetTotalSize();
  char* allocated = arena_->AllocateAligned(
      total_allocate_size, huge_page_tlb_size_, ioptions_.info_log);

  auto temp_ptr = EncodeVarint32(allocated, index_size_);
  uint32_t* index =
      reinterpret_cast<uint32_t*>(EncodeVarint32(temp_ptr, num_prefixes_));
  char* sub_index = reinterpret_cast<char*>(index + index_size_);

  uint32_t sub_index_offset = 0;
  for (uint32_t i = 0; i < index_size_; i++) {
    uint32_t num_keys_for_bucket = entries_per_bucket[i];
    switch (num_keys_for_bucket) {
      case 0:
        // No key for bucket
        index[i] = PlainTableIndex::kMaxFileSize;
        break;
      case 1:
        // point directly to the file offset
        index[i] = hash_to_offsets[i]->offset;
        break;
      default:
        // point to second level indexes.
        index[i] = sub_index_offset | PlainTableIndex::kSubIndexMask;
        char* prev_ptr = &sub_index[sub_index_offset];
        char* cur_ptr = EncodeVarint32(prev_ptr, num_keys_for_bucket);
        sub_index_offset += (cur_ptr - prev_ptr);
        char* sub_index_pos = &sub_index[sub_index_offset];
        IndexRecord* record = hash_to_offsets[i];
        int j;
        for (j = num_keys_for_bucket - 1; j >= 0 && record;
             j--, record = record->next) {
          EncodeFixed32(sub_index_pos + j * sizeof(uint32_t), record->offset);
        }
        assert(j == -1 && record == nullptr);
        sub_index_offset += PlainTableIndex::kOffsetLen * num_keys_for_bucket;
        assert(sub_index_offset <= sub_index_size_);
        break;
    }
  }
  assert(sub_index_offset == sub_index_size_);

  Log(InfoLogLevel::DEBUG_LEVEL, ioptions_.info_log,
      "hash table size: %d, suffix_map length %" ROCKSDB_PRIszt, index_size_,
      sub_index_size_);
  return Slice(allocated, GetTotalSize());
}

const std::string PlainTableIndexBuilder::kPlainTableIndexBlock =
    "PlainTableIndexBlock";
};  // namespace rocksdb

#endif  // ROCKSDB_LITE
