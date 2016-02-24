// Copyright (c) 2013, Facebook, Inc.  All rights reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.
// Copyright (c) 2012 The LevelDB Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file. See the AUTHORS file for names of contributors.
//
// Class for specifying user-defined functions which perform a
// transformation on a slice.  It is not required that every slice
// belong to the domain and/or range of a function.  Subclasses should
// define InDomain and InRange to determine which slices are in either
// of these sets respectively.

#ifndef STORAGE_ROCKSDB_INCLUDE_SLICE_TRANSFORM_H_
#define STORAGE_ROCKSDB_INCLUDE_SLICE_TRANSFORM_H_

#include <string>

namespace rocksdb {

class Slice;

class SliceTransform {
 public:
  virtual ~SliceTransform() {};

  // Return the name of this transformation.
  virtual const char* Name() const = 0;

  // transform a src in domain to a dst in the range
  virtual Slice Transform(const Slice& src) const = 0;

  // determine whether this is a valid src upon the function applies
  virtual bool InDomain(const Slice& src) const = 0;

  // determine whether dst=Transform(src) for some src
  virtual bool InRange(const Slice& dst) const = 0;

  // Transform(s)=Transform(`prefix`) for any s with `prefix` as a prefix.
  //
  // This function is not used by RocksDB, but for users. If users pass
  // Options by string to RocksDB, they might not know what prefix extractor
  // they are using. This function is to help users can determine:
  //   if they want to iterate all keys prefixing `prefix`, whetherit is
  //   safe to use prefix bloom filter and seek to key `prefix`.
  // If this function returns true, this means a user can Seek() to a prefix
  // using the bloom filter. Otherwise, user needs to skip the bloom filter
  // by setting ReadOptions.total_order_seek = true.
  //
  // Here is an example: Suppose we implement a slice transform that returns
  // the first part of the string after spliting it using deimiter ",":
  // 1. SameResultWhenAppended("abc,") should return true. If aplying prefix
  //    bloom filter using it, all slices matching "abc:.*" will be extracted
  //    to "abc,", so any SST file or memtable containing any of those key
  //    will not be filtered out.
  // 2. SameResultWhenAppended("abc") should return false. A user will not
  //    guaranteed to see all the keys matching "abc.*" if a user seek to "abc"
  //    against a DB with the same setting. If one SST file only contains
  //    "abcd,e", the file can be filtered out and the key will be invisible.
  //
  // i.e., an implementation always returning false is safe.
  virtual bool SameResultWhenAppended(const Slice& prefix) const {
    return false;
  }
};

extern const SliceTransform* NewFixedPrefixTransform(size_t prefix_len);

extern const SliceTransform* NewCappedPrefixTransform(size_t cap_len);

extern const SliceTransform* NewNoopTransform();

}

#endif  // STORAGE_ROCKSDB_INCLUDE_SLICE_TRANSFORM_H_
