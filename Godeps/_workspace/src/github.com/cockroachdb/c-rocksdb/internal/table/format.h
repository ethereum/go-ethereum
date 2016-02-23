//  Copyright (c) 2013, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.
//
// Copyright (c) 2011 The LevelDB Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file. See the AUTHORS file for names of contributors.

#pragma once
#include <string>
#include <stdint.h>
#include "rocksdb/slice.h"
#include "rocksdb/status.h"
#include "rocksdb/options.h"
#include "rocksdb/table.h"

namespace rocksdb {

class Block;
class RandomAccessFile;
struct ReadOptions;

// the length of the magic number in bytes.
const int kMagicNumberLengthByte = 8;

// BlockHandle is a pointer to the extent of a file that stores a data
// block or a meta block.
class BlockHandle {
 public:
  BlockHandle();
  BlockHandle(uint64_t offset, uint64_t size);

  // The offset of the block in the file.
  uint64_t offset() const { return offset_; }
  void set_offset(uint64_t _offset) { offset_ = _offset; }

  // The size of the stored block
  uint64_t size() const { return size_; }
  void set_size(uint64_t _size) { size_ = _size; }

  void EncodeTo(std::string* dst) const;
  Status DecodeFrom(Slice* input);

  // Return a string that contains the copy of handle.
  std::string ToString(bool hex = true) const;

  // if the block handle's offset and size are both "0", we will view it
  // as a null block handle that points to no where.
  bool IsNull() const {
    return offset_ == 0 && size_ == 0;
  }

  static const BlockHandle& NullBlockHandle() {
    return kNullBlockHandle;
  }

  // Maximum encoding length of a BlockHandle
  enum { kMaxEncodedLength = 10 + 10 };

 private:
  uint64_t offset_ = 0;
  uint64_t size_ = 0;

  static const BlockHandle kNullBlockHandle;
};

inline uint32_t GetCompressFormatForVersion(CompressionType compression_type,
                                            uint32_t version) {
  // snappy is not versioned
  assert(compression_type != kSnappyCompression &&
         compression_type != kNoCompression);
  // As of version 2, we encode compressed block with
  // compress_format_version == 2. Before that, the version is 1.
  // DO NOT CHANGE THIS FUNCTION, it affects disk format
  return version >= 2 ? 2 : 1;
}

inline bool BlockBasedTableSupportedVersion(uint32_t version) {
  return version <= 2;
}

// Footer encapsulates the fixed information stored at the tail
// end of every table file.
class Footer {
 public:
  // Constructs a footer without specifying its table magic number.
  // In such case, the table magic number of such footer should be
  // initialized via @ReadFooterFromFile().
  // Use this when you plan to load Footer with DecodeFrom(). Never use this
  // when you plan to EncodeTo.
  Footer() : Footer(kInvalidTableMagicNumber, 0) {}

  // Use this constructor when you plan to write out the footer using
  // EncodeTo(). Never use this constructor with DecodeFrom().
  Footer(uint64_t table_magic_number, uint32_t version);

  // The version of the footer in this file
  uint32_t version() const { return version_; }

  // The checksum type used in this file
  ChecksumType checksum() const { return checksum_; }
  void set_checksum(const ChecksumType c) { checksum_ = c; }

  // The block handle for the metaindex block of the table
  const BlockHandle& metaindex_handle() const { return metaindex_handle_; }
  void set_metaindex_handle(const BlockHandle& h) { metaindex_handle_ = h; }

  // The block handle for the index block of the table
  const BlockHandle& index_handle() const { return index_handle_; }

  void set_index_handle(const BlockHandle& h) { index_handle_ = h; }

  uint64_t table_magic_number() const { return table_magic_number_; }

  void EncodeTo(std::string* dst) const;

  // Set the current footer based on the input slice.
  //
  // REQUIRES: table_magic_number_ is not set (i.e.,
  // HasInitializedTableMagicNumber() is true). The function will initialize the
  // magic number
  Status DecodeFrom(Slice* input);

  // Encoded length of a Footer.  Note that the serialization of a Footer will
  // always occupy at least kMinEncodedLength bytes.  If fields are changed
  // the version number should be incremented and kMaxEncodedLength should be
  // increased accordingly.
  enum {
    // Footer version 0 (legacy) will always occupy exactly this many bytes.
    // It consists of two block handles, padding, and a magic number.
    kVersion0EncodedLength = 2 * BlockHandle::kMaxEncodedLength + 8,
    // Footer of versions 1 and higher will always occupy exactly this many
    // bytes. It consists of the checksum type, two block handles, padding,
    // a version number (bigger than 1), and a magic number
    kNewVersionsEncodedLength = 1 + 2 * BlockHandle::kMaxEncodedLength + 4 + 8,
    kMinEncodedLength = kVersion0EncodedLength,
    kMaxEncodedLength = kNewVersionsEncodedLength,
  };

  static const uint64_t kInvalidTableMagicNumber = 0;

  // convert this object to a human readable form
  std::string ToString() const;

 private:
  // REQUIRES: magic number wasn't initialized.
  void set_table_magic_number(uint64_t magic_number) {
    assert(!HasInitializedTableMagicNumber());
    table_magic_number_ = magic_number;
  }

  // return true if @table_magic_number_ is set to a value different
  // from @kInvalidTableMagicNumber.
  bool HasInitializedTableMagicNumber() const {
    return (table_magic_number_ != kInvalidTableMagicNumber);
  }

  uint32_t version_;
  ChecksumType checksum_;
  BlockHandle metaindex_handle_;
  BlockHandle index_handle_;
  uint64_t table_magic_number_ = 0;
};

// Read the footer from file
// If enforce_table_magic_number != 0, ReadFooterFromFile() will return
// corruption if table_magic number is not equal to enforce_table_magic_number
Status ReadFooterFromFile(RandomAccessFileReader* file, uint64_t file_size,
                          Footer* footer,
                          uint64_t enforce_table_magic_number = 0);

// 1-byte type + 32-bit crc
static const size_t kBlockTrailerSize = 5;

struct BlockContents {
  Slice data;           // Actual contents of data
  bool cachable;        // True iff data can be cached
  CompressionType compression_type;
  std::unique_ptr<char[]> allocation;

  BlockContents() : cachable(false), compression_type(kNoCompression) {}

  BlockContents(const Slice& _data, bool _cachable,
                CompressionType _compression_type)
      : data(_data), cachable(_cachable), compression_type(_compression_type) {}

  BlockContents(std::unique_ptr<char[]>&& _data, size_t _size, bool _cachable,
                CompressionType _compression_type)
      : data(_data.get(), _size),
        cachable(_cachable),
        compression_type(_compression_type),
        allocation(std::move(_data)) {}

  BlockContents(BlockContents&& other) { *this = std::move(other); }

  BlockContents& operator=(BlockContents&& other) {
    data = std::move(other.data);
    cachable = other.cachable;
    compression_type = other.compression_type;
    allocation = std::move(other.allocation);
    return *this;
  }
};

// Read the block identified by "handle" from "file".  On failure
// return non-OK.  On success fill *result and return OK.
extern Status ReadBlockContents(RandomAccessFileReader* file,
                                const Footer& footer,
                                const ReadOptions& options,
                                const BlockHandle& handle,
                                BlockContents* contents, Env* env,
                                bool do_uncompress);

// The 'data' points to the raw block contents read in from file.
// This method allocates a new heap buffer and the raw block
// contents are uncompresed into this buffer. This buffer is
// returned via 'result' and it is upto the caller to
// free this buffer.
// For description of compress_format_version and possible values, see
// util/compression.h
extern Status UncompressBlockContents(const char* data, size_t n,
                                      BlockContents* contents,
                                      uint32_t compress_format_version);

// Implementation details follow.  Clients should ignore,

inline BlockHandle::BlockHandle()
    : BlockHandle(~static_cast<uint64_t>(0),
                  ~static_cast<uint64_t>(0)) {
}

inline BlockHandle::BlockHandle(uint64_t _offset, uint64_t _size)
    : offset_(_offset), size_(_size) {}

}  // namespace rocksdb
