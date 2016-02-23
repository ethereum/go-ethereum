//  Copyright (c) 2013, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.
//
// Copyright (c) 2012 The LevelDB Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file. See the AUTHORS file for names of contributors.

#include <algorithm>
#include "table/block_based_filter_block.h"

#include "db/dbformat.h"
#include "rocksdb/filter_policy.h"
#include "util/coding.h"
#include "util/string_util.h"

namespace rocksdb {

namespace {
bool SamePrefix(const SliceTransform* prefix_extractor,
                const Slice& key1, const Slice& key2) {
  if (!prefix_extractor->InDomain(key1) &&
      !prefix_extractor->InDomain(key2)) {
    return true;
  } else if (!prefix_extractor->InDomain(key1) ||
             !prefix_extractor->InDomain(key2)) {
    return false;
  } else {
    return (prefix_extractor->Transform(key1) ==
            prefix_extractor->Transform(key2));
  }
}

void AppendItem(std::string* props, const std::string& key,
                const std::string& value) {
  char cspace = ' ';
  std::string value_str("");
  size_t i = 0;
  const size_t dataLength = 64;
  const size_t tabLength = 2;
  const size_t offLength = 16;

  value_str.append(&value[i], std::min(size_t(dataLength), value.size()));
  i += dataLength;
  while (i < value.size()) {
    value_str.append("\n");
    value_str.append(offLength, cspace);
    value_str.append(&value[i], std::min(size_t(dataLength), value.size() - i));
    i += dataLength;
  }

  std::string result("");
  if (key.size() < (offLength - tabLength))
    result.append(size_t((offLength - tabLength)) - key.size(), cspace);
  result.append(key);

  props->append(result + ": " + value_str + "\n");
}

template <class TKey>
void AppendItem(std::string* props, const TKey& key, const std::string& value) {
  std::string key_str = rocksdb::ToString(key);
  AppendItem(props, key_str, value);
}
}  // namespace


// See doc/table_format.txt for an explanation of the filter block format.

// Generate new filter every 2KB of data
static const size_t kFilterBaseLg = 11;
static const size_t kFilterBase = 1 << kFilterBaseLg;

BlockBasedFilterBlockBuilder::BlockBasedFilterBlockBuilder(
    const SliceTransform* prefix_extractor,
    const BlockBasedTableOptions& table_opt)
    : policy_(table_opt.filter_policy.get()),
      prefix_extractor_(prefix_extractor),
      whole_key_filtering_(table_opt.whole_key_filtering) {
  assert(policy_);
}

void BlockBasedFilterBlockBuilder::StartBlock(uint64_t block_offset) {
  uint64_t filter_index = (block_offset / kFilterBase);
  assert(filter_index >= filter_offsets_.size());
  while (filter_index > filter_offsets_.size()) {
    GenerateFilter();
  }
}

void BlockBasedFilterBlockBuilder::Add(const Slice& key) {
  added_to_start_ = 0;
  if (whole_key_filtering_) {
    AddKey(key);
    added_to_start_ = 1;
  }
  if (prefix_extractor_ && prefix_extractor_->InDomain(key)) {
    AddPrefix(key);
  }
}

// Add key to filter if needed
inline void BlockBasedFilterBlockBuilder::AddKey(const Slice& key) {
  start_.push_back(entries_.size());
  entries_.append(key.data(), key.size());
}

// Add prefix to filter if needed
inline void BlockBasedFilterBlockBuilder::AddPrefix(const Slice& key) {
  // get slice for most recently added entry
  Slice prev;
  if (start_.size() > added_to_start_) {
    size_t prev_start = start_[start_.size() - 1 - added_to_start_];
    const char* base = entries_.data() + prev_start;
    size_t length = entries_.size() - prev_start;
    prev = Slice(base, length);
  }

  // this assumes prefix(prefix(key)) == prefix(key), as the last
  // entry in entries_ may be either a key or prefix, and we use
  // prefix(last entry) to get the prefix of the last key.
  if (prev.size() == 0 || !SamePrefix(prefix_extractor_, key, prev)) {
    Slice prefix = prefix_extractor_->Transform(key);
    start_.push_back(entries_.size());
    entries_.append(prefix.data(), prefix.size());
  }
}

Slice BlockBasedFilterBlockBuilder::Finish() {
  if (!start_.empty()) {
    GenerateFilter();
  }

  // Append array of per-filter offsets
  const uint32_t array_offset = static_cast<uint32_t>(result_.size());
  for (size_t i = 0; i < filter_offsets_.size(); i++) {
    PutFixed32(&result_, filter_offsets_[i]);
  }

  PutFixed32(&result_, array_offset);
  result_.push_back(kFilterBaseLg);  // Save encoding parameter in result
  return Slice(result_);
}

void BlockBasedFilterBlockBuilder::GenerateFilter() {
  const size_t num_entries = start_.size();
  if (num_entries == 0) {
    // Fast path if there are no keys for this filter
    filter_offsets_.push_back(static_cast<uint32_t>(result_.size()));
    return;
  }

  // Make list of keys from flattened key structure
  start_.push_back(entries_.size());  // Simplify length computation
  tmp_entries_.resize(num_entries);
  for (size_t i = 0; i < num_entries; i++) {
    const char* base = entries_.data() + start_[i];
    size_t length = start_[i + 1] - start_[i];
    tmp_entries_[i] = Slice(base, length);
  }

  // Generate filter for current set of keys and append to result_.
  filter_offsets_.push_back(static_cast<uint32_t>(result_.size()));
  policy_->CreateFilter(&tmp_entries_[0], static_cast<int>(num_entries),
                        &result_);

  tmp_entries_.clear();
  entries_.clear();
  start_.clear();
}

BlockBasedFilterBlockReader::BlockBasedFilterBlockReader(
    const SliceTransform* prefix_extractor,
    const BlockBasedTableOptions& table_opt, bool whole_key_filtering,
    BlockContents&& contents)
    : policy_(table_opt.filter_policy.get()),
      prefix_extractor_(prefix_extractor),
      whole_key_filtering_(whole_key_filtering),
      data_(nullptr),
      offset_(nullptr),
      num_(0),
      base_lg_(0),
      contents_(std::move(contents)) {
  assert(policy_);
  size_t n = contents_.data.size();
  if (n < 5) return;  // 1 byte for base_lg_ and 4 for start of offset array
  base_lg_ = contents_.data[n - 1];
  uint32_t last_word = DecodeFixed32(contents_.data.data() + n - 5);
  if (last_word > n - 5) return;
  data_ = contents_.data.data();
  offset_ = data_ + last_word;
  num_ = (n - 5 - last_word) / 4;
}

bool BlockBasedFilterBlockReader::KeyMayMatch(const Slice& key,
                                              uint64_t block_offset) {
  assert(block_offset != kNotValid);
  if (!whole_key_filtering_) {
    return true;
  }
  return MayMatch(key, block_offset);
}

bool BlockBasedFilterBlockReader::PrefixMayMatch(const Slice& prefix,
                                                 uint64_t block_offset) {
  assert(block_offset != kNotValid);
  if (!prefix_extractor_) {
    return true;
  }
  return MayMatch(prefix, block_offset);
}

bool BlockBasedFilterBlockReader::MayMatch(const Slice& entry,
                                           uint64_t block_offset) {
  uint64_t index = block_offset >> base_lg_;
  if (index < num_) {
    uint32_t start = DecodeFixed32(offset_ + index * 4);
    uint32_t limit = DecodeFixed32(offset_ + index * 4 + 4);
    if (start <= limit && limit <= (uint32_t)(offset_ - data_)) {
      Slice filter = Slice(data_ + start, limit - start);
      return policy_->KeyMayMatch(entry, filter);
    } else if (start == limit) {
      // Empty filters do not match any entries
      return false;
    }
  }
  return true;  // Errors are treated as potential matches
}

size_t BlockBasedFilterBlockReader::ApproximateMemoryUsage() const {
  return num_ * 4 + 5 + (offset_ - data_);
}

std::string BlockBasedFilterBlockReader::ToString() const {
  std::string result, filter_meta;
  result.reserve(1024);

  std::string s_bo("Block offset"), s_hd("Hex dump"), s_fb("# filter blocks");
  AppendItem(&result, s_fb, rocksdb::ToString(num_));
  AppendItem(&result, s_bo, s_hd);

  for (size_t index = 0; index < num_; index++) {
    uint32_t start = DecodeFixed32(offset_ + index * 4);
    uint32_t limit = DecodeFixed32(offset_ + index * 4 + 4);

    if (start != limit) {
      result.append(" filter block # " + rocksdb::ToString(index + 1) + "\n");
      Slice filter = Slice(data_ + start, limit - start);
      AppendItem(&result, start, filter.ToString(true));
    }
  }
  return result;
}
}  // namespace rocksdb
