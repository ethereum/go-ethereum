//  Copyright (c) 2013, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.

#pragma once
#ifndef ROCKSDB_LITE

#include "rocksdb/slice.h"
#include "db/dbformat.h"

namespace rocksdb {

class WritableFile;
struct ParsedInternalKey;

// Helper class to write out a key to an output file
// Actual data format of the key is documented in plain_table_factory.h
class PlainTableKeyEncoder {
 public:
  explicit PlainTableKeyEncoder(EncodingType encoding_type,
                                uint32_t user_key_len,
                                const SliceTransform* prefix_extractor,
                                size_t index_sparseness)
      : encoding_type_((prefix_extractor != nullptr) ? encoding_type : kPlain),
        fixed_user_key_len_(user_key_len),
        prefix_extractor_(prefix_extractor),
        index_sparseness_((index_sparseness > 1) ? index_sparseness : 1),
        key_count_for_prefix_(0) {}
  // key: the key to write out, in the format of internal key.
  // file: the output file to write out
  // offset: offset in the file. Needs to be updated after appending bytes
  //         for the key
  // meta_bytes_buf: buffer for extra meta bytes
  // meta_bytes_buf_size: offset to append extra meta bytes. Will be updated
  //                      if meta_bytes_buf is updated.
  Status AppendKey(const Slice& key, WritableFileWriter* file, uint64_t* offset,
                   char* meta_bytes_buf, size_t* meta_bytes_buf_size);

  // Return actual encoding type to be picked
  EncodingType GetEncodingType() { return encoding_type_; }

 private:
  EncodingType encoding_type_;
  uint32_t fixed_user_key_len_;
  const SliceTransform* prefix_extractor_;
  const size_t index_sparseness_;
  size_t key_count_for_prefix_;
  IterKey pre_prefix_;
};

// A helper class to decode keys from input buffer
// Actual data format of the key is documented in plain_table_factory.h
class PlainTableKeyDecoder {
 public:
  explicit PlainTableKeyDecoder(EncodingType encoding_type,
                                uint32_t user_key_len,
                                const SliceTransform* prefix_extractor)
      : encoding_type_(encoding_type),
        prefix_len_(0),
        fixed_user_key_len_(user_key_len),
        prefix_extractor_(prefix_extractor),
        in_prefix_(false) {}
  // Find the next key.
  // start: char array where the key starts.
  // limit: boundary of the char array
  // parsed_key: the output of the result key
  // internal_key: if not null, fill with the output of the result key in
  //               un-parsed format
  // bytes_read: how many bytes read from start. Output
  // seekable: whether key can be read from this place. Used when building
  //           indexes. Output.
  Status NextKey(const char* start, const char* limit,
                 ParsedInternalKey* parsed_key, Slice* internal_key,
                 size_t* bytes_read, bool* seekable = nullptr);
  EncodingType encoding_type_;
  uint32_t prefix_len_;
  uint32_t fixed_user_key_len_;
  Slice saved_user_key_;
  IterKey cur_key_;
  const SliceTransform* prefix_extractor_;
  bool in_prefix_;

 private:
  Status NextPlainEncodingKey(const char* start, const char* limit,
                              ParsedInternalKey* parsed_key,
                              Slice* internal_key, size_t* bytes_read,
                              bool* seekable = nullptr);
  Status NextPrefixEncodingKey(const char* start, const char* limit,
                               ParsedInternalKey* parsed_key,
                               Slice* internal_key, size_t* bytes_read,
                               bool* seekable = nullptr);
};

}  // namespace rocksdb

#endif  // ROCKSDB_LITE
