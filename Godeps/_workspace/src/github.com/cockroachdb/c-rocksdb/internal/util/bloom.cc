//  Copyright (c) 2014, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.
//
// Copyright (c) 2012 The LevelDB Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file. See the AUTHORS file for names of contributors.

#include "rocksdb/filter_policy.h"

#include "rocksdb/slice.h"
#include "table/block_based_filter_block.h"
#include "table/full_filter_block.h"
#include "util/hash.h"
#include "util/coding.h"

namespace rocksdb {

class BlockBasedFilterBlockBuilder;
class FullFilterBlockBuilder;

namespace {
class FullFilterBitsBuilder : public FilterBitsBuilder {
 public:
  explicit FullFilterBitsBuilder(const size_t bits_per_key,
                                 const size_t num_probes)
      : bits_per_key_(bits_per_key),
        num_probes_(num_probes) {
    assert(bits_per_key_);
  }

  ~FullFilterBitsBuilder() {}

  virtual void AddKey(const Slice& key) override {
    uint32_t hash = BloomHash(key);
    if (hash_entries_.size() == 0 || hash != hash_entries_.back()) {
      hash_entries_.push_back(hash);
    }
  }

  // Create a filter that for hashes [0, n-1], the filter is allocated here
  // When creating filter, it is ensured that
  // total_bits = num_lines * CACHE_LINE_SIZE * 8
  // dst len is >= 5, 1 for num_probes, 4 for num_lines
  // Then total_bits = (len - 5) * 8, and cache_line_size could be calculated
  // +----------------------------------------------------------------+
  // |              filter data with length total_bits/8              |
  // +----------------------------------------------------------------+
  // |                                                                |
  // | ...                                                            |
  // |                                                                |
  // +----------------------------------------------------------------+
  // | ...                | num_probes : 1 byte | num_lines : 4 bytes |
  // +----------------------------------------------------------------+
  virtual Slice Finish(std::unique_ptr<const char[]>* buf) override {
    uint32_t total_bits, num_lines;
    char* data = ReserveSpace(static_cast<int>(hash_entries_.size()),
                              &total_bits, &num_lines);
    assert(data);

    if (total_bits != 0 && num_lines != 0) {
      for (auto h : hash_entries_) {
        AddHash(h, data, num_lines, total_bits);
      }
    }
    data[total_bits/8] = static_cast<char>(num_probes_);
    EncodeFixed32(data + total_bits/8 + 1, static_cast<uint32_t>(num_lines));

    const char* const_data = data;
    buf->reset(const_data);
    hash_entries_.clear();

    return Slice(data, total_bits / 8 + 5);
  }

 private:
  size_t bits_per_key_;
  size_t num_probes_;
  std::vector<uint32_t> hash_entries_;

  // Get totalbits that optimized for cpu cache line
  uint32_t GetTotalBitsForLocality(uint32_t total_bits);

  // Reserve space for new filter
  char* ReserveSpace(const int num_entry, uint32_t* total_bits,
      uint32_t* num_lines);

  // Assuming single threaded access to this function.
  void AddHash(uint32_t h, char* data, uint32_t num_lines,
      uint32_t total_bits);

  // No Copy allowed
  FullFilterBitsBuilder(const FullFilterBitsBuilder&);
  void operator=(const FullFilterBitsBuilder&);
};

uint32_t FullFilterBitsBuilder::GetTotalBitsForLocality(uint32_t total_bits) {
  uint32_t num_lines =
      (total_bits + CACHE_LINE_SIZE * 8 - 1) / (CACHE_LINE_SIZE * 8);

  // Make num_lines an odd number to make sure more bits are involved
  // when determining which block.
  if (num_lines % 2 == 0) {
    num_lines++;
  }
  return num_lines * (CACHE_LINE_SIZE * 8);
}

char* FullFilterBitsBuilder::ReserveSpace(const int num_entry,
    uint32_t* total_bits, uint32_t* num_lines) {
  assert(bits_per_key_);
  char* data = nullptr;
  if (num_entry != 0) {
    uint32_t total_bits_tmp = num_entry * static_cast<uint32_t>(bits_per_key_);

    *total_bits = GetTotalBitsForLocality(total_bits_tmp);
    *num_lines = *total_bits / (CACHE_LINE_SIZE * 8);
    assert(*total_bits > 0 && *total_bits % 8 == 0);
  } else {
    // filter is empty, just leave space for metadata
    *total_bits = 0;
    *num_lines = 0;
  }

  // Reserve space for Filter
  uint32_t sz = *total_bits / 8;
  sz += 5;  // 4 bytes for num_lines, 1 byte for num_probes

  data = new char[sz];
  memset(data, 0, sz);
  return data;
}

inline void FullFilterBitsBuilder::AddHash(uint32_t h, char* data,
    uint32_t num_lines, uint32_t total_bits) {
  assert(num_lines > 0 && total_bits > 0);

  const uint32_t delta = (h >> 17) | (h << 15);  // Rotate right 17 bits
  uint32_t b = (h % num_lines) * (CACHE_LINE_SIZE * 8);

  for (uint32_t i = 0; i < num_probes_; ++i) {
    // Since CACHE_LINE_SIZE is defined as 2^n, this line will be optimized
    // to a simple operation by compiler.
    const uint32_t bitpos = b + (h % (CACHE_LINE_SIZE * 8));
    data[bitpos / 8] |= (1 << (bitpos % 8));

    h += delta;
  }
}

class FullFilterBitsReader : public FilterBitsReader {
 public:
  explicit FullFilterBitsReader(const Slice& contents)
      : data_(const_cast<char*>(contents.data())),
        data_len_(static_cast<uint32_t>(contents.size())),
        num_probes_(0),
        num_lines_(0) {
    assert(data_);
    GetFilterMeta(contents, &num_probes_, &num_lines_);
    // Sanitize broken parameter
    if (num_lines_ != 0 && (data_len_-5) % num_lines_ != 0) {
      num_lines_ = 0;
      num_probes_ = 0;
    }
  }

  ~FullFilterBitsReader() {}

  virtual bool MayMatch(const Slice& entry) override {
    if (data_len_ <= 5) {   // remain same with original filter
      return false;
    }
    // Other Error params, including a broken filter, regarded as match
    if (num_probes_ == 0 || num_lines_ == 0) return true;
    uint32_t hash = BloomHash(entry);
    return HashMayMatch(hash, Slice(data_, data_len_),
                        num_probes_, num_lines_);
  }

 private:
  // Filter meta data
  char* data_;
  uint32_t data_len_;
  size_t num_probes_;
  uint32_t num_lines_;

  // Get num_probes, and num_lines from filter
  // If filter format broken, set both to 0.
  void GetFilterMeta(const Slice& filter, size_t* num_probes,
                             uint32_t* num_lines);

  // "filter" contains the data appended by a preceding call to
  // CreateFilterFromHash() on this class.  This method must return true if
  // the key was in the list of keys passed to CreateFilter().
  // This method may return true or false if the key was not on the
  // list, but it should aim to return false with a high probability.
  //
  // hash: target to be checked
  // filter: the whole filter, including meta data bytes
  // num_probes: number of probes, read before hand
  // num_lines: filter metadata, read before hand
  // Before calling this function, need to ensure the input meta data
  // is valid.
  bool HashMayMatch(const uint32_t& hash, const Slice& filter,
      const size_t& num_probes, const uint32_t& num_lines);

  // No Copy allowed
  FullFilterBitsReader(const FullFilterBitsReader&);
  void operator=(const FullFilterBitsReader&);
};

void FullFilterBitsReader::GetFilterMeta(const Slice& filter,
    size_t* num_probes, uint32_t* num_lines) {
  uint32_t len = static_cast<uint32_t>(filter.size());
  if (len <= 5) {
    // filter is empty or broken
    *num_probes = 0;
    *num_lines = 0;
    return;
  }

  *num_probes = filter.data()[len - 5];
  *num_lines = DecodeFixed32(filter.data() + len - 4);
}

bool FullFilterBitsReader::HashMayMatch(const uint32_t& hash,
    const Slice& filter, const size_t& num_probes,
    const uint32_t& num_lines) {
  uint32_t len = static_cast<uint32_t>(filter.size());
  if (len <= 5) return false;  // remain the same with original filter

  // It is ensured the params are valid before calling it
  assert(num_probes != 0);
  assert(num_lines != 0 && (len - 5) % num_lines == 0);
  uint32_t cache_line_size = (len - 5) / num_lines;
  const char* data = filter.data();

  uint32_t h = hash;
  const uint32_t delta = (h >> 17) | (h << 15);  // Rotate right 17 bits
  uint32_t b = (h % num_lines) * (cache_line_size * 8);

  for (uint32_t i = 0; i < num_probes; ++i) {
    // Since CACHE_LINE_SIZE is defined as 2^n, this line will be optimized
    //  to a simple and operation by compiler.
    const uint32_t bitpos = b + (h % (cache_line_size * 8));
    if (((data[bitpos / 8]) & (1 << (bitpos % 8))) == 0) {
      return false;
    }

    h += delta;
  }

  return true;
}

// An implementation of filter policy
class BloomFilterPolicy : public FilterPolicy {
 public:
  explicit BloomFilterPolicy(int bits_per_key, bool use_block_based_builder)
      : bits_per_key_(bits_per_key), hash_func_(BloomHash),
        use_block_based_builder_(use_block_based_builder) {
    initialize();
  }

  ~BloomFilterPolicy() {
  }

  virtual const char* Name() const override {
    return "rocksdb.BuiltinBloomFilter";
  }

  virtual void CreateFilter(const Slice* keys, int n,
                            std::string* dst) const override {
    // Compute bloom filter size (in both bits and bytes)
    size_t bits = n * bits_per_key_;

    // For small n, we can see a very high false positive rate.  Fix it
    // by enforcing a minimum bloom filter length.
    if (bits < 64) bits = 64;

    size_t bytes = (bits + 7) / 8;
    bits = bytes * 8;

    const size_t init_size = dst->size();
    dst->resize(init_size + bytes, 0);
    dst->push_back(static_cast<char>(num_probes_));  // Remember # of probes
    char* array = &(*dst)[init_size];
    for (size_t i = 0; i < (size_t)n; i++) {
      // Use double-hashing to generate a sequence of hash values.
      // See analysis in [Kirsch,Mitzenmacher 2006].
      uint32_t h = hash_func_(keys[i]);
      const uint32_t delta = (h >> 17) | (h << 15);  // Rotate right 17 bits
      for (size_t j = 0; j < num_probes_; j++) {
        const uint32_t bitpos = h % bits;
        array[bitpos/8] |= (1 << (bitpos % 8));
        h += delta;
      }
    }
  }

  virtual bool KeyMayMatch(const Slice& key,
                           const Slice& bloom_filter) const override {
    const size_t len = bloom_filter.size();
    if (len < 2) return false;

    const char* array = bloom_filter.data();
    const size_t bits = (len - 1) * 8;

    // Use the encoded k so that we can read filters generated by
    // bloom filters created using different parameters.
    const size_t k = array[len-1];
    if (k > 30) {
      // Reserved for potentially new encodings for short bloom filters.
      // Consider it a match.
      return true;
    }

    uint32_t h = hash_func_(key);
    const uint32_t delta = (h >> 17) | (h << 15);  // Rotate right 17 bits
    for (size_t j = 0; j < k; j++) {
      const uint32_t bitpos = h % bits;
      if ((array[bitpos/8] & (1 << (bitpos % 8))) == 0) return false;
      h += delta;
    }
    return true;
  }

  virtual FilterBitsBuilder* GetFilterBitsBuilder() const override {
    if (use_block_based_builder_) {
      return nullptr;
    }

    return new FullFilterBitsBuilder(bits_per_key_, num_probes_);
  }

  virtual FilterBitsReader* GetFilterBitsReader(const Slice& contents)
      const override {
    return new FullFilterBitsReader(contents);
  }

  // If choose to use block based builder
  bool UseBlockBasedBuilder() { return use_block_based_builder_; }

 private:
  size_t bits_per_key_;
  size_t num_probes_;
  uint32_t (*hash_func_)(const Slice& key);

  const bool use_block_based_builder_;

  void initialize() {
    // We intentionally round down to reduce probing cost a little bit
    num_probes_ = static_cast<size_t>(bits_per_key_ * 0.69);  // 0.69 =~ ln(2)
    if (num_probes_ < 1) num_probes_ = 1;
    if (num_probes_ > 30) num_probes_ = 30;
  }
};

}  // namespace

const FilterPolicy* NewBloomFilterPolicy(int bits_per_key,
                                         bool use_block_based_builder) {
  return new BloomFilterPolicy(bits_per_key, use_block_based_builder);
}

}  // namespace rocksdb
