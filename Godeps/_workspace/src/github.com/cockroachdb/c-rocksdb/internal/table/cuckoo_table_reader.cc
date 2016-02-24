//  Copyright (c) 2014, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.
//
// Copyright (c) 2011 The LevelDB Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file. See the AUTHORS file for names of contributors.

#ifndef ROCKSDB_LITE
#include "table/cuckoo_table_reader.h"

#include <algorithm>
#include <limits>
#include <string>
#include <utility>
#include <vector>
#include "rocksdb/iterator.h"
#include "rocksdb/table.h"
#include "table/meta_blocks.h"
#include "table/cuckoo_table_factory.h"
#include "table/get_context.h"
#include "util/arena.h"
#include "util/coding.h"

namespace rocksdb {
namespace {
const uint64_t CACHE_LINE_MASK = ~((uint64_t)CACHE_LINE_SIZE - 1);
const uint32_t kInvalidIndex = std::numeric_limits<uint32_t>::max();
}

extern const uint64_t kCuckooTableMagicNumber;

CuckooTableReader::CuckooTableReader(
    const ImmutableCFOptions& ioptions,
    std::unique_ptr<RandomAccessFileReader>&& file, uint64_t file_size,
    const Comparator* comparator,
    uint64_t (*get_slice_hash)(const Slice&, uint32_t, uint64_t))
    : file_(std::move(file)),
      ucomp_(comparator),
      get_slice_hash_(get_slice_hash) {
  if (!ioptions.allow_mmap_reads) {
    status_ = Status::InvalidArgument("File is not mmaped");
  }
  TableProperties* props = nullptr;
  status_ = ReadTableProperties(file_.get(), file_size, kCuckooTableMagicNumber,
      ioptions.env, ioptions.info_log, &props);
  if (!status_.ok()) {
    return;
  }
  table_props_.reset(props);
  auto& user_props = props->user_collected_properties;
  auto hash_funs = user_props.find(CuckooTablePropertyNames::kNumHashFunc);
  if (hash_funs == user_props.end()) {
    status_ = Status::Corruption("Number of hash functions not found");
    return;
  }
  num_hash_func_ = *reinterpret_cast<const uint32_t*>(hash_funs->second.data());
  auto unused_key = user_props.find(CuckooTablePropertyNames::kEmptyKey);
  if (unused_key == user_props.end()) {
    status_ = Status::Corruption("Empty bucket value not found");
    return;
  }
  unused_key_ = unused_key->second;

  key_length_ = static_cast<uint32_t>(props->fixed_key_len);
  auto user_key_len = user_props.find(CuckooTablePropertyNames::kUserKeyLength);
  if (user_key_len == user_props.end()) {
    status_ = Status::Corruption("User key length not found");
    return;
  }
  user_key_length_ = *reinterpret_cast<const uint32_t*>(
      user_key_len->second.data());

  auto value_length = user_props.find(CuckooTablePropertyNames::kValueLength);
  if (value_length == user_props.end()) {
    status_ = Status::Corruption("Value length not found");
    return;
  }
  value_length_ = *reinterpret_cast<const uint32_t*>(
      value_length->second.data());
  bucket_length_ = key_length_ + value_length_;

  auto hash_table_size = user_props.find(
      CuckooTablePropertyNames::kHashTableSize);
  if (hash_table_size == user_props.end()) {
    status_ = Status::Corruption("Hash table size not found");
    return;
  }
  table_size_ = *reinterpret_cast<const uint64_t*>(
      hash_table_size->second.data());

  auto is_last_level = user_props.find(CuckooTablePropertyNames::kIsLastLevel);
  if (is_last_level == user_props.end()) {
    status_ = Status::Corruption("Is last level not found");
    return;
  }
  is_last_level_ = *reinterpret_cast<const bool*>(is_last_level->second.data());

  auto identity_as_first_hash = user_props.find(
      CuckooTablePropertyNames::kIdentityAsFirstHash);
  if (identity_as_first_hash == user_props.end()) {
    status_ = Status::Corruption("identity as first hash not found");
    return;
  }
  identity_as_first_hash_ = *reinterpret_cast<const bool*>(
      identity_as_first_hash->second.data());

  auto use_module_hash = user_props.find(
      CuckooTablePropertyNames::kUseModuleHash);
  if (use_module_hash == user_props.end()) {
    status_ = Status::Corruption("hash type is not found");
    return;
  }
  use_module_hash_ = *reinterpret_cast<const bool*>(
      use_module_hash->second.data());
  auto cuckoo_block_size = user_props.find(
      CuckooTablePropertyNames::kCuckooBlockSize);
  if (cuckoo_block_size == user_props.end()) {
    status_ = Status::Corruption("Cuckoo block size not found");
    return;
  }
  cuckoo_block_size_ = *reinterpret_cast<const uint32_t*>(
      cuckoo_block_size->second.data());
  cuckoo_block_bytes_minus_one_ = cuckoo_block_size_ * bucket_length_ - 1;
  status_ = file_->Read(0, file_size, &file_data_, nullptr);
}

Status CuckooTableReader::Get(const ReadOptions& readOptions, const Slice& key,
                              GetContext* get_context) {
  assert(key.size() == key_length_ + (is_last_level_ ? 8 : 0));
  Slice user_key = ExtractUserKey(key);
  for (uint32_t hash_cnt = 0; hash_cnt < num_hash_func_; ++hash_cnt) {
    uint64_t offset = bucket_length_ * CuckooHash(
        user_key, hash_cnt, use_module_hash_, table_size_,
        identity_as_first_hash_, get_slice_hash_);
    const char* bucket = &file_data_.data()[offset];
    for (uint32_t block_idx = 0; block_idx < cuckoo_block_size_;
         ++block_idx, bucket += bucket_length_) {
      if (ucomp_->Equal(Slice(unused_key_.data(), user_key.size()),
                        Slice(bucket, user_key.size()))) {
        return Status::OK();
      }
      // Here, we compare only the user key part as we support only one entry
      // per user key and we don't support sanpshot.
      if (ucomp_->Equal(user_key, Slice(bucket, user_key.size()))) {
        Slice value(bucket + key_length_, value_length_);
        if (is_last_level_) {
          get_context->SaveValue(value);
        } else {
          Slice full_key(bucket, key_length_);
          ParsedInternalKey found_ikey;
          ParseInternalKey(full_key, &found_ikey);
          get_context->SaveValue(found_ikey, value);
        }
        // We don't support merge operations. So, we return here.
        return Status::OK();
      }
    }
  }
  return Status::OK();
}

void CuckooTableReader::Prepare(const Slice& key) {
  // Prefetch the first Cuckoo Block.
  Slice user_key = ExtractUserKey(key);
  uint64_t addr = reinterpret_cast<uint64_t>(file_data_.data()) +
    bucket_length_ * CuckooHash(user_key, 0, use_module_hash_, table_size_,
                                identity_as_first_hash_, nullptr);
  uint64_t end_addr = addr + cuckoo_block_bytes_minus_one_;
  for (addr &= CACHE_LINE_MASK; addr < end_addr; addr += CACHE_LINE_SIZE) {
    PREFETCH(reinterpret_cast<const char*>(addr), 0, 3);
  }
}

class CuckooTableIterator : public Iterator {
 public:
  explicit CuckooTableIterator(CuckooTableReader* reader);
  ~CuckooTableIterator() {}
  bool Valid() const override;
  void SeekToFirst() override;
  void SeekToLast() override;
  void Seek(const Slice& target) override;
  void Next() override;
  void Prev() override;
  Slice key() const override;
  Slice value() const override;
  Status status() const override { return status_; }
  void InitIfNeeded();

 private:
  struct BucketComparator {
    BucketComparator(const Slice& file_data, const Comparator* ucomp,
                     uint32_t bucket_len, uint32_t user_key_len,
                     const Slice& target = Slice())
      : file_data_(file_data),
        ucomp_(ucomp),
        bucket_len_(bucket_len),
        user_key_len_(user_key_len),
        target_(target) {}
    bool operator()(const uint32_t first, const uint32_t second) const {
      const char* first_bucket =
        (first == kInvalidIndex) ? target_.data() :
                                   &file_data_.data()[first * bucket_len_];
      const char* second_bucket =
        (second == kInvalidIndex) ? target_.data() :
                                    &file_data_.data()[second * bucket_len_];
      return ucomp_->Compare(Slice(first_bucket, user_key_len_),
                             Slice(second_bucket, user_key_len_)) < 0;
    }
   private:
    const Slice file_data_;
    const Comparator* ucomp_;
    const uint32_t bucket_len_;
    const uint32_t user_key_len_;
    const Slice target_;
  };

  const BucketComparator bucket_comparator_;
  void PrepareKVAtCurrIdx();
  CuckooTableReader* reader_;
  bool initialized_;
  Status status_;
  // Contains a map of keys to bucket_id sorted in key order.
  std::vector<uint32_t> sorted_bucket_ids_;
  // We assume that the number of items can be stored in uint32 (4 Billion).
  uint32_t curr_key_idx_;
  Slice curr_value_;
  IterKey curr_key_;
  // No copying allowed
  CuckooTableIterator(const CuckooTableIterator&) = delete;
  void operator=(const Iterator&) = delete;
};

CuckooTableIterator::CuckooTableIterator(CuckooTableReader* reader)
  : bucket_comparator_(reader->file_data_, reader->ucomp_,
                       reader->bucket_length_, reader->user_key_length_),
    reader_(reader),
    initialized_(false),
    curr_key_idx_(kInvalidIndex) {
  sorted_bucket_ids_.clear();
  curr_value_.clear();
  curr_key_.Clear();
}

void CuckooTableIterator::InitIfNeeded() {
  if (initialized_) {
    return;
  }
  sorted_bucket_ids_.reserve(reader_->GetTableProperties()->num_entries);
  uint64_t num_buckets = reader_->table_size_ + reader_->cuckoo_block_size_ - 1;
  assert(num_buckets < kInvalidIndex);
  const char* bucket = reader_->file_data_.data();
  for (uint32_t bucket_id = 0; bucket_id < num_buckets; ++bucket_id) {
    if (Slice(bucket, reader_->key_length_) != Slice(reader_->unused_key_)) {
      sorted_bucket_ids_.push_back(bucket_id);
    }
    bucket += reader_->bucket_length_;
  }
  assert(sorted_bucket_ids_.size() ==
      reader_->GetTableProperties()->num_entries);
  std::sort(sorted_bucket_ids_.begin(), sorted_bucket_ids_.end(),
            bucket_comparator_);
  curr_key_idx_ = kInvalidIndex;
  initialized_ = true;
}

void CuckooTableIterator::SeekToFirst() {
  InitIfNeeded();
  curr_key_idx_ = 0;
  PrepareKVAtCurrIdx();
}

void CuckooTableIterator::SeekToLast() {
  InitIfNeeded();
  curr_key_idx_ = static_cast<uint32_t>(sorted_bucket_ids_.size()) - 1;
  PrepareKVAtCurrIdx();
}

void CuckooTableIterator::Seek(const Slice& target) {
  InitIfNeeded();
  const BucketComparator seek_comparator(
      reader_->file_data_, reader_->ucomp_,
      reader_->bucket_length_, reader_->user_key_length_,
      ExtractUserKey(target));
  auto seek_it = std::lower_bound(sorted_bucket_ids_.begin(),
      sorted_bucket_ids_.end(),
      kInvalidIndex,
      seek_comparator);
  curr_key_idx_ =
      static_cast<uint32_t>(std::distance(sorted_bucket_ids_.begin(), seek_it));
  PrepareKVAtCurrIdx();
}

bool CuckooTableIterator::Valid() const {
  return curr_key_idx_ < sorted_bucket_ids_.size();
}

void CuckooTableIterator::PrepareKVAtCurrIdx() {
  if (!Valid()) {
    curr_value_.clear();
    curr_key_.Clear();
    return;
  }
  uint32_t id = sorted_bucket_ids_[curr_key_idx_];
  const char* offset = reader_->file_data_.data() +
                       id * reader_->bucket_length_;
  if (reader_->is_last_level_) {
    // Always return internal key.
    curr_key_.SetInternalKey(Slice(offset, reader_->user_key_length_),
                             0, kTypeValue);
  } else {
    curr_key_.SetKey(Slice(offset, reader_->key_length_));
  }
  curr_value_ = Slice(offset + reader_->key_length_, reader_->value_length_);
}

void CuckooTableIterator::Next() {
  if (!Valid()) {
    curr_value_.clear();
    curr_key_.Clear();
    return;
  }
  ++curr_key_idx_;
  PrepareKVAtCurrIdx();
}

void CuckooTableIterator::Prev() {
  if (curr_key_idx_ == 0) {
    curr_key_idx_ = static_cast<uint32_t>(sorted_bucket_ids_.size());
  }
  if (!Valid()) {
    curr_value_.clear();
    curr_key_.Clear();
    return;
  }
  --curr_key_idx_;
  PrepareKVAtCurrIdx();
}

Slice CuckooTableIterator::key() const {
  assert(Valid());
  return curr_key_.GetKey();
}

Slice CuckooTableIterator::value() const {
  assert(Valid());
  return curr_value_;
}

extern Iterator* NewErrorIterator(const Status& status, Arena* arena);

Iterator* CuckooTableReader::NewIterator(
    const ReadOptions& read_options, Arena* arena) {
  if (!status().ok()) {
    return NewErrorIterator(
        Status::Corruption("CuckooTableReader status is not okay."), arena);
  }
  if (read_options.total_order_seek) {
    return NewErrorIterator(
        Status::InvalidArgument("total_order_seek is not supported."), arena);
  }
  CuckooTableIterator* iter;
  if (arena == nullptr) {
    iter = new CuckooTableIterator(this);
  } else {
    auto iter_mem = arena->AllocateAligned(sizeof(CuckooTableIterator));
    iter = new (iter_mem) CuckooTableIterator(this);
  }
  return iter;
}

size_t CuckooTableReader::ApproximateMemoryUsage() const { return 0; }

}  // namespace rocksdb
#endif
