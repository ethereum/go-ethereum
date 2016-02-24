// Copyright (c) 2011 The LevelDB Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file. See the AUTHORS file for names of contributors.

#ifndef ROCKSDB_LITE

#include "table/plain_table_reader.h"

#include <string>
#include <vector>

#include "db/dbformat.h"

#include "rocksdb/cache.h"
#include "rocksdb/comparator.h"
#include "rocksdb/env.h"
#include "rocksdb/filter_policy.h"
#include "rocksdb/options.h"
#include "rocksdb/statistics.h"

#include "table/block.h"
#include "table/bloom_block.h"
#include "table/filter_block.h"
#include "table/format.h"
#include "table/meta_blocks.h"
#include "table/two_level_iterator.h"
#include "table/plain_table_factory.h"
#include "table/plain_table_key_coding.h"
#include "table/get_context.h"

#include "util/arena.h"
#include "util/coding.h"
#include "util/dynamic_bloom.h"
#include "util/hash.h"
#include "util/histogram.h"
#include "util/murmurhash.h"
#include "util/perf_context_imp.h"
#include "util/stop_watch.h"
#include "util/string_util.h"


namespace rocksdb {

namespace {

// Safely getting a uint32_t element from a char array, where, starting from
// `base`, every 4 bytes are considered as an fixed 32 bit integer.
inline uint32_t GetFixed32Element(const char* base, size_t offset) {
  return DecodeFixed32(base + offset * sizeof(uint32_t));
}
}  // namespace

// Iterator to iterate IndexedTable
class PlainTableIterator : public Iterator {
 public:
  explicit PlainTableIterator(PlainTableReader* table, bool use_prefix_seek);
  ~PlainTableIterator();

  bool Valid() const override;

  void SeekToFirst() override;

  void SeekToLast() override;

  void Seek(const Slice& target) override;

  void Next() override;

  void Prev() override;

  Slice key() const override;

  Slice value() const override;

  Status status() const override;

 private:
  PlainTableReader* table_;
  PlainTableKeyDecoder decoder_;
  bool use_prefix_seek_;
  uint32_t offset_;
  uint32_t next_offset_;
  Slice key_;
  Slice value_;
  Status status_;
  // No copying allowed
  PlainTableIterator(const PlainTableIterator&) = delete;
  void operator=(const Iterator&) = delete;
};

extern const uint64_t kPlainTableMagicNumber;
PlainTableReader::PlainTableReader(const ImmutableCFOptions& ioptions,
                                   unique_ptr<RandomAccessFileReader>&& file,
                                   const EnvOptions& storage_options,
                                   const InternalKeyComparator& icomparator,
                                   EncodingType encoding_type,
                                   uint64_t file_size,
                                   const TableProperties* table_properties)
    : internal_comparator_(icomparator),
      encoding_type_(encoding_type),
      full_scan_mode_(false),
      data_end_offset_(static_cast<uint32_t>(table_properties->data_size)),
      user_key_len_(static_cast<uint32_t>(table_properties->fixed_key_len)),
      prefix_extractor_(ioptions.prefix_extractor),
      enable_bloom_(false),
      bloom_(6, nullptr),
      ioptions_(ioptions),
      file_(std::move(file)),
      file_size_(file_size),
      table_properties_(nullptr) {}

PlainTableReader::~PlainTableReader() {
}

Status PlainTableReader::Open(const ImmutableCFOptions& ioptions,
                              const EnvOptions& env_options,
                              const InternalKeyComparator& internal_comparator,
                              unique_ptr<RandomAccessFileReader>&& file,
                              uint64_t file_size,
                              unique_ptr<TableReader>* table_reader,
                              const int bloom_bits_per_key,
                              double hash_table_ratio, size_t index_sparseness,
                              size_t huge_page_tlb_size, bool full_scan_mode) {
  assert(ioptions.allow_mmap_reads);
  if (file_size > PlainTableIndex::kMaxFileSize) {
    return Status::NotSupported("File is too large for PlainTableReader!");
  }

  TableProperties* props = nullptr;
  auto s = ReadTableProperties(file.get(), file_size, kPlainTableMagicNumber,
                               ioptions.env, ioptions.info_log, &props);
  if (!s.ok()) {
    return s;
  }

  assert(hash_table_ratio >= 0.0);
  auto& user_props = props->user_collected_properties;
  auto prefix_extractor_in_file =
      user_props.find(PlainTablePropertyNames::kPrefixExtractorName);

  if (!full_scan_mode && prefix_extractor_in_file != user_props.end()) {
    if (!ioptions.prefix_extractor) {
      return Status::InvalidArgument(
          "Prefix extractor is missing when opening a PlainTable built "
          "using a prefix extractor");
    } else if (prefix_extractor_in_file->second.compare(
                   ioptions.prefix_extractor->Name()) != 0) {
      return Status::InvalidArgument(
          "Prefix extractor given doesn't match the one used to build "
          "PlainTable");
    }
  }

  EncodingType encoding_type = kPlain;
  auto encoding_type_prop =
      user_props.find(PlainTablePropertyNames::kEncodingType);
  if (encoding_type_prop != user_props.end()) {
    encoding_type = static_cast<EncodingType>(
        DecodeFixed32(encoding_type_prop->second.c_str()));
  }

  std::unique_ptr<PlainTableReader> new_reader(new PlainTableReader(
      ioptions, std::move(file), env_options, internal_comparator,
      encoding_type, file_size, props));

  s = new_reader->MmapDataFile();
  if (!s.ok()) {
    return s;
  }

  if (!full_scan_mode) {
    s = new_reader->PopulateIndex(props, bloom_bits_per_key, hash_table_ratio,
                                  index_sparseness, huge_page_tlb_size);
    if (!s.ok()) {
      return s;
    }
  } else {
    // Flag to indicate it is a full scan mode so that none of the indexes
    // can be used.
    new_reader->full_scan_mode_ = true;
  }

  *table_reader = std::move(new_reader);
  return s;
}

void PlainTableReader::SetupForCompaction() {
}

Iterator* PlainTableReader::NewIterator(const ReadOptions& options,
                                        Arena* arena) {
  if (options.total_order_seek && !IsTotalOrderMode()) {
    return NewErrorIterator(
        Status::InvalidArgument("total_order_seek not supported"), arena);
  }
  if (arena == nullptr) {
    return new PlainTableIterator(this, prefix_extractor_ != nullptr);
  } else {
    auto mem = arena->AllocateAligned(sizeof(PlainTableIterator));
    return new (mem) PlainTableIterator(this, prefix_extractor_ != nullptr);
  }
}

Status PlainTableReader::PopulateIndexRecordList(
    PlainTableIndexBuilder* index_builder, vector<uint32_t>* prefix_hashes) {
  Slice prev_key_prefix_slice;
  uint32_t pos = data_start_offset_;

  bool is_first_record = true;
  Slice key_prefix_slice;
  PlainTableKeyDecoder decoder(encoding_type_, user_key_len_,
                               ioptions_.prefix_extractor);
  while (pos < data_end_offset_) {
    uint32_t key_offset = pos;
    ParsedInternalKey key;
    Slice value_slice;
    bool seekable = false;
    Status s = Next(&decoder, &pos, &key, nullptr, &value_slice, &seekable);
    if (!s.ok()) {
      return s;
    }

    key_prefix_slice = GetPrefix(key);
    if (enable_bloom_) {
      bloom_.AddHash(GetSliceHash(key.user_key));
    } else {
      if (is_first_record || prev_key_prefix_slice != key_prefix_slice) {
        if (!is_first_record) {
          prefix_hashes->push_back(GetSliceHash(prev_key_prefix_slice));
        }
        prev_key_prefix_slice = key_prefix_slice;
      }
    }

    index_builder->AddKeyPrefix(GetPrefix(key), key_offset);

    if (!seekable && is_first_record) {
      return Status::Corruption("Key for a prefix is not seekable");
    }

    is_first_record = false;
  }

  prefix_hashes->push_back(GetSliceHash(key_prefix_slice));
  auto s = index_.InitFromRawData(index_builder->Finish());
  return s;
}

void PlainTableReader::AllocateAndFillBloom(int bloom_bits_per_key,
                                            int num_prefixes,
                                            size_t huge_page_tlb_size,
                                            vector<uint32_t>* prefix_hashes) {
  if (!IsTotalOrderMode()) {
    uint32_t bloom_total_bits = num_prefixes * bloom_bits_per_key;
    if (bloom_total_bits > 0) {
      enable_bloom_ = true;
      bloom_.SetTotalBits(&arena_, bloom_total_bits, ioptions_.bloom_locality,
                          huge_page_tlb_size, ioptions_.info_log);
      FillBloom(prefix_hashes);
    }
  }
}

void PlainTableReader::FillBloom(vector<uint32_t>* prefix_hashes) {
  assert(bloom_.IsInitialized());
  for (auto prefix_hash : *prefix_hashes) {
    bloom_.AddHash(prefix_hash);
  }
}

Status PlainTableReader::MmapDataFile() {
  // Get mmapped memory to file_data_.
  return file_->Read(0, file_size_, &file_data_, nullptr);
}

Status PlainTableReader::PopulateIndex(TableProperties* props,
                                       int bloom_bits_per_key,
                                       double hash_table_ratio,
                                       size_t index_sparseness,
                                       size_t huge_page_tlb_size) {
  assert(props != nullptr);
  table_properties_.reset(props);

  BlockContents bloom_block_contents;
  auto s = ReadMetaBlock(file_.get(), file_size_, kPlainTableMagicNumber,
                         ioptions_.env, BloomBlockBuilder::kBloomBlock,
                         &bloom_block_contents);
  bool index_in_file = s.ok();

  BlockContents index_block_contents;
  s = ReadMetaBlock(file_.get(), file_size_, kPlainTableMagicNumber,
      ioptions_.env, PlainTableIndexBuilder::kPlainTableIndexBlock,
      &index_block_contents);

  index_in_file &= s.ok();

  Slice* bloom_block;
  if (index_in_file) {
    bloom_block = &bloom_block_contents.data;
  } else {
    bloom_block = nullptr;
  }

  // index_in_file == true only if there are kBloomBlock and
  // kPlainTableIndexBlock
  // in file

  Slice* index_block;
  if (index_in_file) {
    index_block = &index_block_contents.data;
  } else {
    index_block = nullptr;
  }

  if ((ioptions_.prefix_extractor == nullptr) &&
      (hash_table_ratio != 0)) {
    // ioptions.prefix_extractor is requried for a hash-based look-up.
    return Status::NotSupported(
        "PlainTable requires a prefix extractor enable prefix hash mode.");
  }

  // First, read the whole file, for every kIndexIntervalForSamePrefixKeys rows
  // for a prefix (starting from the first one), generate a record of (hash,
  // offset) and append it to IndexRecordList, which is a data structure created
  // to store them.

  if (!index_in_file) {
    // Allocate bloom filter here for total order mode.
    if (IsTotalOrderMode()) {
      uint32_t num_bloom_bits =
          static_cast<uint32_t>(table_properties_->num_entries) *
          bloom_bits_per_key;
      if (num_bloom_bits > 0) {
        enable_bloom_ = true;
        bloom_.SetTotalBits(&arena_, num_bloom_bits, ioptions_.bloom_locality,
                            huge_page_tlb_size, ioptions_.info_log);
      }
    }
  } else {
    enable_bloom_ = true;
    auto num_blocks_property = props->user_collected_properties.find(
        PlainTablePropertyNames::kNumBloomBlocks);

    uint32_t num_blocks = 0;
    if (num_blocks_property != props->user_collected_properties.end()) {
      Slice temp_slice(num_blocks_property->second);
      if (!GetVarint32(&temp_slice, &num_blocks)) {
        num_blocks = 0;
      }
    }
    // cast away const qualifier, because bloom_ won't be changed
    bloom_.SetRawData(
        const_cast<unsigned char*>(
            reinterpret_cast<const unsigned char*>(bloom_block->data())),
        static_cast<uint32_t>(bloom_block->size()) * 8, num_blocks);
  }

  PlainTableIndexBuilder index_builder(&arena_, ioptions_, index_sparseness,
                                       hash_table_ratio, huge_page_tlb_size);

  std::vector<uint32_t> prefix_hashes;
  if (!index_in_file) {
    s = PopulateIndexRecordList(&index_builder, &prefix_hashes);
    if (!s.ok()) {
      return s;
    }
  } else {
    s = index_.InitFromRawData(*index_block);
    if (!s.ok()) {
      return s;
    }
  }

  if (!index_in_file) {
    // Calculated bloom filter size and allocate memory for
    // bloom filter based on the number of prefixes, then fill it.
    AllocateAndFillBloom(bloom_bits_per_key, index_.GetNumPrefixes(),
                         huge_page_tlb_size, &prefix_hashes);
  }

  // Fill two table properties.
  if (!index_in_file) {
    props->user_collected_properties["plain_table_hash_table_size"] =
        ToString(index_.GetIndexSize() * PlainTableIndex::kOffsetLen);
    props->user_collected_properties["plain_table_sub_index_size"] =
        ToString(index_.GetSubIndexSize());
  } else {
    props->user_collected_properties["plain_table_hash_table_size"] =
        ToString(0);
    props->user_collected_properties["plain_table_sub_index_size"] =
        ToString(0);
  }

  return Status::OK();
}

Status PlainTableReader::GetOffset(const Slice& target, const Slice& prefix,
                                   uint32_t prefix_hash, bool& prefix_matched,
                                   uint32_t* offset) const {
  prefix_matched = false;
  uint32_t prefix_index_offset;
  auto res = index_.GetOffset(prefix_hash, &prefix_index_offset);
  if (res == PlainTableIndex::kNoPrefixForBucket) {
    *offset = data_end_offset_;
    return Status::OK();
  } else if (res == PlainTableIndex::kDirectToFile) {
    *offset = prefix_index_offset;
    return Status::OK();
  }

  // point to sub-index, need to do a binary search
  uint32_t upper_bound;
  const char* base_ptr =
      index_.GetSubIndexBasePtrAndUpperBound(prefix_index_offset, &upper_bound);
  uint32_t low = 0;
  uint32_t high = upper_bound;
  ParsedInternalKey mid_key;
  ParsedInternalKey parsed_target;
  if (!ParseInternalKey(target, &parsed_target)) {
    return Status::Corruption(Slice());
  }

  // The key is between [low, high). Do a binary search between it.
  while (high - low > 1) {
    uint32_t mid = (high + low) / 2;
    uint32_t file_offset = GetFixed32Element(base_ptr, mid);
    size_t tmp;
    Status s = PlainTableKeyDecoder(encoding_type_, user_key_len_,
                                    ioptions_.prefix_extractor)
                   .NextKey(file_data_.data() + file_offset,
                            file_data_.data() + data_end_offset_, &mid_key,
                            nullptr, &tmp);
    if (!s.ok()) {
      return s;
    }
    int cmp_result = internal_comparator_.Compare(mid_key, parsed_target);
    if (cmp_result < 0) {
      low = mid;
    } else {
      if (cmp_result == 0) {
        // Happen to have found the exact key or target is smaller than the
        // first key after base_offset.
        prefix_matched = true;
        *offset = file_offset;
        return Status::OK();
      } else {
        high = mid;
      }
    }
  }
  // Both of the key at the position low or low+1 could share the same
  // prefix as target. We need to rule out one of them to avoid to go
  // to the wrong prefix.
  ParsedInternalKey low_key;
  size_t tmp;
  uint32_t low_key_offset = GetFixed32Element(base_ptr, low);
  Status s = PlainTableKeyDecoder(encoding_type_, user_key_len_,
                                  ioptions_.prefix_extractor)
                 .NextKey(file_data_.data() + low_key_offset,
                          file_data_.data() + data_end_offset_, &low_key,
                          nullptr, &tmp);
  if (!s.ok()) {
    return s;
  }

  if (GetPrefix(low_key) == prefix) {
    prefix_matched = true;
    *offset = low_key_offset;
  } else if (low + 1 < upper_bound) {
    // There is possible a next prefix, return it
    prefix_matched = false;
    *offset = GetFixed32Element(base_ptr, low + 1);
  } else {
    // target is larger than a key of the last prefix in this bucket
    // but with a different prefix. Key does not exist.
    *offset = data_end_offset_;
  }
  return Status::OK();
}

bool PlainTableReader::MatchBloom(uint32_t hash) const {
  return !enable_bloom_ || bloom_.MayContainHash(hash);
}


Status PlainTableReader::Next(PlainTableKeyDecoder* decoder, uint32_t* offset,
                              ParsedInternalKey* parsed_key,
                              Slice* internal_key, Slice* value,
                              bool* seekable) const {
  if (*offset == data_end_offset_) {
    *offset = data_end_offset_;
    return Status::OK();
  }

  if (*offset > data_end_offset_) {
    return Status::Corruption("Offset is out of file size");
  }

  const char* start = file_data_.data() + *offset;
  size_t bytes_for_key;
  Status s =
      decoder->NextKey(start, file_data_.data() + data_end_offset_, parsed_key,
                       internal_key, &bytes_for_key, seekable);
  if (!s.ok()) {
    return s;
  }
  uint32_t value_size;
  const char* value_ptr = GetVarint32Ptr(
      start + bytes_for_key, file_data_.data() + data_end_offset_, &value_size);
  if (value_ptr == nullptr) {
    return Status::Corruption(
        "Unexpected EOF when reading the next value's size.");
  }
  *offset = *offset + static_cast<uint32_t>(value_ptr - start) + value_size;
  if (*offset > data_end_offset_) {
    return Status::Corruption("Unexpected EOF when reading the next value. ");
  }
  *value = Slice(value_ptr, value_size);

  return Status::OK();
}

void PlainTableReader::Prepare(const Slice& target) {
  if (enable_bloom_) {
    uint32_t prefix_hash = GetSliceHash(GetPrefix(target));
    bloom_.Prefetch(prefix_hash);
  }
}

Status PlainTableReader::Get(const ReadOptions& ro, const Slice& target,
                             GetContext* get_context) {
  // Check bloom filter first.
  Slice prefix_slice;
  uint32_t prefix_hash;
  if (IsTotalOrderMode()) {
    if (full_scan_mode_) {
      status_ =
          Status::InvalidArgument("Get() is not allowed in full scan mode.");
    }
    // Match whole user key for bloom filter check.
    if (!MatchBloom(GetSliceHash(GetUserKey(target)))) {
      return Status::OK();
    }
    // in total order mode, there is only one bucket 0, and we always use empty
    // prefix.
    prefix_slice = Slice();
    prefix_hash = 0;
  } else {
    prefix_slice = GetPrefix(target);
    prefix_hash = GetSliceHash(prefix_slice);
    if (!MatchBloom(prefix_hash)) {
      return Status::OK();
    }
  }
  uint32_t offset;
  bool prefix_match;
  Status s =
      GetOffset(target, prefix_slice, prefix_hash, prefix_match, &offset);
  if (!s.ok()) {
    return s;
  }
  ParsedInternalKey found_key;
  ParsedInternalKey parsed_target;
  if (!ParseInternalKey(target, &parsed_target)) {
    return Status::Corruption(Slice());
  }
  Slice found_value;
  PlainTableKeyDecoder decoder(encoding_type_, user_key_len_,
                               ioptions_.prefix_extractor);
  while (offset < data_end_offset_) {
    s = Next(&decoder, &offset, &found_key, nullptr, &found_value);
    if (!s.ok()) {
      return s;
    }
    if (!prefix_match) {
      // Need to verify prefix for the first key found if it is not yet
      // checked.
      if (GetPrefix(found_key) != prefix_slice) {
        return Status::OK();
      }
      prefix_match = true;
    }
    // TODO(ljin): since we know the key comparison result here,
    // can we enable the fast path?
    if (internal_comparator_.Compare(found_key, parsed_target) >= 0) {
      if (!get_context->SaveValue(found_key, found_value)) {
        break;
      }
    }
  }
  return Status::OK();
}

uint64_t PlainTableReader::ApproximateOffsetOf(const Slice& key) {
  return 0;
}

PlainTableIterator::PlainTableIterator(PlainTableReader* table,
                                       bool use_prefix_seek)
    : table_(table),
      decoder_(table_->encoding_type_, table_->user_key_len_,
               table_->prefix_extractor_),
      use_prefix_seek_(use_prefix_seek) {
  next_offset_ = offset_ = table_->data_end_offset_;
}

PlainTableIterator::~PlainTableIterator() {
}

bool PlainTableIterator::Valid() const {
  return offset_ < table_->data_end_offset_
      && offset_ >= table_->data_start_offset_;
}

void PlainTableIterator::SeekToFirst() {
  next_offset_ = table_->data_start_offset_;
  if (next_offset_ >= table_->data_end_offset_) {
    next_offset_ = offset_ = table_->data_end_offset_;
  } else {
    Next();
  }
}

void PlainTableIterator::SeekToLast() {
  assert(false);
  status_ = Status::NotSupported("SeekToLast() is not supported in PlainTable");
}

void PlainTableIterator::Seek(const Slice& target) {
  // If the user doesn't set prefix seek option and we are not able to do a
  // total Seek(). assert failure.
  if (!use_prefix_seek_) {
    if (table_->full_scan_mode_) {
      status_ =
          Status::InvalidArgument("Seek() is not allowed in full scan mode.");
      offset_ = next_offset_ = table_->data_end_offset_;
      return;
    } else if (table_->GetIndexSize() > 1) {
      assert(false);
      status_ = Status::NotSupported(
          "PlainTable cannot issue non-prefix seek unless in total order "
          "mode.");
      offset_ = next_offset_ = table_->data_end_offset_;
      return;
    }
  }

  Slice prefix_slice = table_->GetPrefix(target);
  uint32_t prefix_hash = 0;
  // Bloom filter is ignored in total-order mode.
  if (!table_->IsTotalOrderMode()) {
    prefix_hash = GetSliceHash(prefix_slice);
    if (!table_->MatchBloom(prefix_hash)) {
      offset_ = next_offset_ = table_->data_end_offset_;
      return;
    }
  }
  bool prefix_match;
  status_ = table_->GetOffset(target, prefix_slice, prefix_hash, prefix_match,
                              &next_offset_);
  if (!status_.ok()) {
    offset_ = next_offset_ = table_->data_end_offset_;
    return;
  }

  if (next_offset_ < table_-> data_end_offset_) {
    for (Next(); status_.ok() && Valid(); Next()) {
      if (!prefix_match) {
        // Need to verify the first key's prefix
        if (table_->GetPrefix(key()) != prefix_slice) {
          offset_ = next_offset_ = table_->data_end_offset_;
          break;
        }
        prefix_match = true;
      }
      if (table_->internal_comparator_.Compare(key(), target) >= 0) {
        break;
      }
    }
  } else {
    offset_ = table_->data_end_offset_;
  }
}

void PlainTableIterator::Next() {
  offset_ = next_offset_;
  if (offset_ < table_->data_end_offset_) {
    Slice tmp_slice;
    ParsedInternalKey parsed_key;
    status_ =
        table_->Next(&decoder_, &next_offset_, &parsed_key, &key_, &value_);
    if (!status_.ok()) {
      offset_ = next_offset_ = table_->data_end_offset_;
    }
  }
}

void PlainTableIterator::Prev() {
  assert(false);
}

Slice PlainTableIterator::key() const {
  assert(Valid());
  return key_;
}

Slice PlainTableIterator::value() const {
  assert(Valid());
  return value_;
}

Status PlainTableIterator::status() const {
  return status_;
}

}  // namespace rocksdb
#endif  // ROCKSDB_LITE
