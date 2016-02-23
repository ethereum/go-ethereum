// Copyright (c) 2015, Facebook, Inc.  All rights reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.
// Copyright (c) 2011 The LevelDB Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file. See the AUTHORS file for names of contributors.

#pragma once

namespace rocksdb {

class Slice;
class Status;
class ColumnFamilyHandle;
class WriteBatch;
struct SliceParts;

// Abstract base class that defines the basic interface for a write batch.
// See WriteBatch for a basic implementation and WrithBatchWithIndex for an
// indexed implemenation.
class WriteBatchBase {
 public:
  virtual ~WriteBatchBase() {}

  // Store the mapping "key->value" in the database.
  virtual void Put(ColumnFamilyHandle* column_family, const Slice& key,
                   const Slice& value) = 0;
  virtual void Put(const Slice& key, const Slice& value) = 0;

  // Variant of Put() that gathers output like writev(2).  The key and value
  // that will be written to the database are concatentations of arrays of
  // slices.
  virtual void Put(ColumnFamilyHandle* column_family, const SliceParts& key,
                   const SliceParts& value);
  virtual void Put(const SliceParts& key, const SliceParts& value);

  // Merge "value" with the existing value of "key" in the database.
  // "key->merge(existing, value)"
  virtual void Merge(ColumnFamilyHandle* column_family, const Slice& key,
                     const Slice& value) = 0;
  virtual void Merge(const Slice& key, const Slice& value) = 0;

  // variant that takes SliceParts
  virtual void Merge(ColumnFamilyHandle* column_family, const SliceParts& key,
                     const SliceParts& value);
  virtual void Merge(const SliceParts& key, const SliceParts& value);

  // If the database contains a mapping for "key", erase it.  Else do nothing.
  virtual void Delete(ColumnFamilyHandle* column_family, const Slice& key) = 0;
  virtual void Delete(const Slice& key) = 0;

  // variant that takes SliceParts
  virtual void Delete(ColumnFamilyHandle* column_family, const SliceParts& key);
  virtual void Delete(const SliceParts& key);

  // Append a blob of arbitrary size to the records in this batch. The blob will
  // be stored in the transaction log but not in any other file. In particular,
  // it will not be persisted to the SST files. When iterating over this
  // WriteBatch, WriteBatch::Handler::LogData will be called with the contents
  // of the blob as it is encountered. Blobs, puts, deletes, and merges will be
  // encountered in the same order in thich they were inserted. The blob will
  // NOT consume sequence number(s) and will NOT increase the count of the batch
  //
  // Example application: add timestamps to the transaction log for use in
  // replication.
  virtual void PutLogData(const Slice& blob) = 0;

  // Clear all updates buffered in this batch.
  virtual void Clear() = 0;

  // Covert this batch into a WriteBatch.  This is an abstracted way of
  // converting any WriteBatchBase(eg WriteBatchWithIndex) into a basic
  // WriteBatch.
  virtual WriteBatch* GetWriteBatch() = 0;

  // Records the state of the batch for future calls to RollbackToSavePoint().
  // May be called multiple times to set multiple save points.
  virtual void SetSavePoint() = 0;

  // Remove all entries in this batch (Put, Merge, Delete, PutLogData) since the
  // most recent call to SetSavePoint() and removes the most recent save point.
  // If there is no previous call to SetSavePoint(), behaves the same as
  // Clear().
  virtual Status RollbackToSavePoint() = 0;
};

}  // namespace rocksdb
