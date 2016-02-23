//  Copyright (c) 2013, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.

#ifndef ROCKSDB_LITE
#include "table/plain_table_key_coding.h"

#include "db/dbformat.h"
#include "table/plain_table_factory.h"
#include "util/file_reader_writer.h"

namespace rocksdb {

namespace {

enum PlainTableEntryType : unsigned char {
  kFullKey = 0,
  kPrefixFromPreviousKey = 1,
  kKeySuffix = 2,
};

// Control byte:
// First two bits indicate type of entry
// Other bytes are inlined sizes. If all bits are 1 (0x03F), overflow bytes
// are used. key_size-0x3F will be encoded as a variint32 after this bytes.

const unsigned char kSizeInlineLimit = 0x3F;

// Return 0 for error
size_t EncodeSize(PlainTableEntryType type, uint32_t key_size,
                  char* out_buffer) {
  out_buffer[0] = type << 6;

  if (key_size < static_cast<uint32_t>(kSizeInlineLimit)) {
    // size inlined
    out_buffer[0] |= static_cast<char>(key_size);
    return 1;
  } else {
    out_buffer[0] |= kSizeInlineLimit;
    char* ptr = EncodeVarint32(out_buffer + 1, key_size - kSizeInlineLimit);
    return ptr - out_buffer;
  }
}

// Return position after the size byte(s). nullptr means error
const char* DecodeSize(const char* offset, const char* limit,
                       PlainTableEntryType* entry_type, uint32_t* key_size) {
  assert(offset < limit);
  *entry_type = static_cast<PlainTableEntryType>(
      (static_cast<unsigned char>(offset[0]) & ~kSizeInlineLimit) >> 6);
  char inline_key_size = offset[0] & kSizeInlineLimit;
  if (inline_key_size < kSizeInlineLimit) {
    *key_size = inline_key_size;
    return offset + 1;
  } else {
    uint32_t extra_size;
    const char* ptr = GetVarint32Ptr(offset + 1, limit, &extra_size);
    if (ptr == nullptr) {
      return nullptr;
    }
    *key_size = kSizeInlineLimit + extra_size;
    return ptr;
  }
}
}  // namespace

Status PlainTableKeyEncoder::AppendKey(const Slice& key,
                                       WritableFileWriter* file,
                                       uint64_t* offset, char* meta_bytes_buf,
                                       size_t* meta_bytes_buf_size) {
  ParsedInternalKey parsed_key;
  if (!ParseInternalKey(key, &parsed_key)) {
    return Status::Corruption(Slice());
  }

  Slice key_to_write = key;  // Portion of internal key to write out.

  uint32_t user_key_size = static_cast<uint32_t>(key.size() - 8);
  if (encoding_type_ == kPlain) {
    if (fixed_user_key_len_ == kPlainTableVariableLength) {
      // Write key length
      char key_size_buf[5];  // tmp buffer for key size as varint32
      char* ptr = EncodeVarint32(key_size_buf, user_key_size);
      assert(ptr <= key_size_buf + sizeof(key_size_buf));
      auto len = ptr - key_size_buf;
      Status s = file->Append(Slice(key_size_buf, len));
      if (!s.ok()) {
        return s;
      }
      *offset += len;
    }
  } else {
    assert(encoding_type_ == kPrefix);
    char size_bytes[12];
    size_t size_bytes_pos = 0;

    Slice prefix =
        prefix_extractor_->Transform(Slice(key.data(), user_key_size));
    if (key_count_for_prefix_ == 0 || prefix != pre_prefix_.GetKey() ||
        key_count_for_prefix_ % index_sparseness_ == 0) {
      key_count_for_prefix_ = 1;
      pre_prefix_.SetKey(prefix);
      size_bytes_pos += EncodeSize(kFullKey, user_key_size, size_bytes);
      Status s = file->Append(Slice(size_bytes, size_bytes_pos));
      if (!s.ok()) {
        return s;
      }
      *offset += size_bytes_pos;
    } else {
      key_count_for_prefix_++;
      if (key_count_for_prefix_ == 2) {
        // For second key within a prefix, need to encode prefix length
        size_bytes_pos +=
            EncodeSize(kPrefixFromPreviousKey,
                       static_cast<uint32_t>(pre_prefix_.GetKey().size()),
                       size_bytes + size_bytes_pos);
      }
      uint32_t prefix_len = static_cast<uint32_t>(pre_prefix_.GetKey().size());
      size_bytes_pos += EncodeSize(kKeySuffix, user_key_size - prefix_len,
                                   size_bytes + size_bytes_pos);
      Status s = file->Append(Slice(size_bytes, size_bytes_pos));
      if (!s.ok()) {
        return s;
      }
      *offset += size_bytes_pos;
      key_to_write = Slice(key.data() + prefix_len, key.size() - prefix_len);
    }
  }

  // Encode full key
  // For value size as varint32 (up to 5 bytes).
  // If the row is of value type with seqId 0, flush the special flag together
  // in this buffer to safe one file append call, which takes 1 byte.
  if (parsed_key.sequence == 0 && parsed_key.type == kTypeValue) {
    Status s =
        file->Append(Slice(key_to_write.data(), key_to_write.size() - 8));
    if (!s.ok()) {
      return s;
    }
    *offset += key_to_write.size() - 8;
    meta_bytes_buf[*meta_bytes_buf_size] = PlainTableFactory::kValueTypeSeqId0;
    *meta_bytes_buf_size += 1;
  } else {
    file->Append(key_to_write);
    *offset += key_to_write.size();
  }

  return Status::OK();
}

namespace {
Status ReadInternalKey(const char* key_ptr, const char* limit,
                       uint32_t user_key_size, ParsedInternalKey* parsed_key,
                       size_t* bytes_read, bool* internal_key_valid,
                       Slice* internal_key) {
  if (key_ptr + user_key_size + 1 >= limit) {
    return Status::Corruption("Unexpected EOF when reading the next key");
  }
  if (*(key_ptr + user_key_size) == PlainTableFactory::kValueTypeSeqId0) {
    // Special encoding for the row with seqID=0
    parsed_key->user_key = Slice(key_ptr, user_key_size);
    parsed_key->sequence = 0;
    parsed_key->type = kTypeValue;
    *bytes_read += user_key_size + 1;
    *internal_key_valid = false;
  } else {
    if (key_ptr + user_key_size + 8 >= limit) {
      return Status::Corruption(
          "Unexpected EOF when reading internal bytes of the next key");
    }
    *internal_key_valid = true;
    *internal_key = Slice(key_ptr, user_key_size + 8);
    if (!ParseInternalKey(*internal_key, parsed_key)) {
      return Status::Corruption(
          Slice("Incorrect value type found when reading the next key"));
    }
    *bytes_read += user_key_size + 8;
  }
  return Status::OK();
}
}  // namespace

Status PlainTableKeyDecoder::NextPlainEncodingKey(
    const char* start, const char* limit, ParsedInternalKey* parsed_key,
    Slice* internal_key, size_t* bytes_read, bool* seekable) {
  const char* key_ptr = start;
  uint32_t user_key_size = 0;
  if (fixed_user_key_len_ != kPlainTableVariableLength) {
    user_key_size = fixed_user_key_len_;
    key_ptr = start;
  } else {
    uint32_t tmp_size = 0;
    key_ptr = GetVarint32Ptr(start, limit, &tmp_size);
    if (key_ptr == nullptr) {
      return Status::Corruption(
          "Unexpected EOF when reading the next key's size");
    }
    user_key_size = tmp_size;
    *bytes_read = key_ptr - start;
  }
  // dummy initial value to avoid compiler complain
  bool decoded_internal_key_valid = true;
  Slice decoded_internal_key;
  Status s =
      ReadInternalKey(key_ptr, limit, user_key_size, parsed_key, bytes_read,
                      &decoded_internal_key_valid, &decoded_internal_key);
  if (!s.ok()) {
    return s;
  }
  if (internal_key != nullptr) {
    if (decoded_internal_key_valid) {
      *internal_key = decoded_internal_key;
    } else {
      // Need to copy out the internal key
      cur_key_.SetInternalKey(*parsed_key);
      *internal_key = cur_key_.GetKey();
    }
  }
  return Status::OK();
}

Status PlainTableKeyDecoder::NextPrefixEncodingKey(
    const char* start, const char* limit, ParsedInternalKey* parsed_key,
    Slice* internal_key, size_t* bytes_read, bool* seekable) {
  const char* key_ptr = start;
  PlainTableEntryType entry_type;

  bool expect_suffix = false;
  do {
    uint32_t size = 0;
    // dummy initial value to avoid compiler complain
    bool decoded_internal_key_valid = true;
    const char* pos = DecodeSize(key_ptr, limit, &entry_type, &size);
    if (pos == nullptr) {
      return Status::Corruption("Unexpected EOF when reading size of the key");
    }
    *bytes_read += pos - key_ptr;
    key_ptr = pos;

    switch (entry_type) {
      case kFullKey: {
        expect_suffix = false;
        Slice decoded_internal_key;
        Status s =
            ReadInternalKey(key_ptr, limit, size, parsed_key, bytes_read,
                            &decoded_internal_key_valid, &decoded_internal_key);
        if (!s.ok()) {
          return s;
        }
        saved_user_key_ = parsed_key->user_key;
        if (internal_key != nullptr) {
          if (decoded_internal_key_valid) {
            *internal_key = decoded_internal_key;
          } else {
            cur_key_.SetInternalKey(*parsed_key);
            *internal_key = cur_key_.GetKey();
          }
        }
        break;
      }
      case kPrefixFromPreviousKey: {
        if (seekable != nullptr) {
          *seekable = false;
        }
        prefix_len_ = size;
        assert(prefix_extractor_ == nullptr ||
               prefix_extractor_->Transform(saved_user_key_).size() ==
                   prefix_len_);
        // Need read another size flag for suffix
        expect_suffix = true;
        break;
      }
      case kKeySuffix: {
        expect_suffix = false;
        if (seekable != nullptr) {
          *seekable = false;
        }
        cur_key_.Reserve(prefix_len_ + size);

        Slice tmp_slice;
        Status s = ReadInternalKey(key_ptr, limit, size, parsed_key, bytes_read,
                                   &decoded_internal_key_valid, &tmp_slice);
        if (!s.ok()) {
          return s;
        }
        cur_key_.SetInternalKey(Slice(saved_user_key_.data(), prefix_len_),
                                *parsed_key);
        assert(
            prefix_extractor_ == nullptr ||
            prefix_extractor_->Transform(ExtractUserKey(cur_key_.GetKey())) ==
                Slice(saved_user_key_.data(), prefix_len_));
        parsed_key->user_key = ExtractUserKey(cur_key_.GetKey());
        if (internal_key != nullptr) {
          *internal_key = cur_key_.GetKey();
        }
        break;
      }
      default:
        return Status::Corruption("Identified size flag.");
    }
  } while (expect_suffix);  // Another round if suffix is expected.
  return Status::OK();
}

Status PlainTableKeyDecoder::NextKey(const char* start, const char* limit,
                                     ParsedInternalKey* parsed_key,
                                     Slice* internal_key, size_t* bytes_read,
                                     bool* seekable) {
  *bytes_read = 0;
  if (seekable != nullptr) {
    *seekable = true;
  }
  if (encoding_type_ == kPlain) {
    return NextPlainEncodingKey(start, limit, parsed_key, internal_key,
                                bytes_read, seekable);
  } else {
    assert(encoding_type_ == kPrefix);
    return NextPrefixEncodingKey(start, limit, parsed_key, internal_key,
                                 bytes_read, seekable);
  }
}

}  // namespace rocksdb
#endif  // ROCKSDB_LITE
