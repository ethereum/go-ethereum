// Copyright (c) 2013, Facebook, Inc.  All rights reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.
// Copyright (c) 2011 The LevelDB Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file. See the AUTHORS file for names of contributors.
//
// WriteBatch holds a collection of updates to apply atomically to a DB.
//
// The updates are applied in the order in which they are added
// to the WriteBatch.  For example, the value of "key" will be "v3"
// after the following batch is written:
//
//    batch.Put("key", "v1");
//    batch.Delete("key");
//    batch.Put("key", "v2");
//    batch.Put("key", "v3");
//
// Multiple threads can invoke const methods on a WriteBatch without
// external synchronization, but if any of the threads may call a
// non-const method, all threads accessing the same WriteBatch must use
// external synchronization.

#ifndef STORAGE_ROCKSDB_INCLUDE_WRITE_BATCH_H_
#define STORAGE_ROCKSDB_INCLUDE_WRITE_BATCH_H_

#include <stack>
#include <string>
#include <stdint.h>
#include "rocksdb/status.h"
#include "rocksdb/write_batch_base.h"

namespace rocksdb {

class Slice;
class ColumnFamilyHandle;
struct SavePoints;
struct SliceParts;

class WriteBatch : public WriteBatchBase {
 public:
  explicit WriteBatch(size_t reserved_bytes = 0);
  ~WriteBatch();

  using WriteBatchBase::Put;
  // Store the mapping "key->value" in the database.
  void Put(ColumnFamilyHandle* column_family, const Slice& key,
           const Slice& value) override;
  void Put(const Slice& key, const Slice& value) override {
    Put(nullptr, key, value);
  }

  // Variant of Put() that gathers output like writev(2).  The key and value
  // that will be written to the database are concatentations of arrays of
  // slices.
  void Put(ColumnFamilyHandle* column_family, const SliceParts& key,
           const SliceParts& value) override;
  void Put(const SliceParts& key, const SliceParts& value) override {
    Put(nullptr, key, value);
  }

  using WriteBatchBase::Merge;
  // Merge "value" with the existing value of "key" in the database.
  // "key->merge(existing, value)"
  void Merge(ColumnFamilyHandle* column_family, const Slice& key,
             const Slice& value) override;
  void Merge(const Slice& key, const Slice& value) override {
    Merge(nullptr, key, value);
  }

  // variant that takes SliceParts
  void Merge(ColumnFamilyHandle* column_family, const SliceParts& key,
             const SliceParts& value) override;
  void Merge(const SliceParts& key, const SliceParts& value) override {
    Merge(nullptr, key, value);
  }

  using WriteBatchBase::Delete;
  // If the database contains a mapping for "key", erase it.  Else do nothing.
  void Delete(ColumnFamilyHandle* column_family, const Slice& key) override;
  void Delete(const Slice& key) override { Delete(nullptr, key); }

  // variant that takes SliceParts
  void Delete(ColumnFamilyHandle* column_family,
              const SliceParts& key) override;
  void Delete(const SliceParts& key) override { Delete(nullptr, key); }

  using WriteBatchBase::PutLogData;
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
  void PutLogData(const Slice& blob) override;

  using WriteBatchBase::Clear;
  // Clear all updates buffered in this batch.
  void Clear() override;

  // Records the state of the batch for future calls to RollbackToSavePoint().
  // May be called multiple times to set multiple save points.
  void SetSavePoint() override;

  // Remove all entries in this batch (Put, Merge, Delete, PutLogData) since the
  // most recent call to SetSavePoint() and removes the most recent save point.
  // If there is no previous call to SetSavePoint(), Status::NotFound()
  // will be returned.
  // Oterwise returns Status::OK().
  Status RollbackToSavePoint() override;

  // Support for iterating over the contents of a batch.
  class Handler {
   public:
    virtual ~Handler();
    // default implementation will just call Put without column family for
    // backwards compatibility. If the column family is not default,
    // the function is noop
    virtual Status PutCF(uint32_t column_family_id, const Slice& key,
                         const Slice& value) {
      if (column_family_id == 0) {
        // Put() historically doesn't return status. We didn't want to be
        // backwards incompatible so we didn't change the return status
        // (this is a public API). We do an ordinary get and return Status::OK()
        Put(key, value);
        return Status::OK();
      }
      return Status::InvalidArgument(
          "non-default column family and PutCF not implemented");
    }
    virtual void Put(const Slice& key, const Slice& value) {}

    // Merge and LogData are not pure virtual. Otherwise, we would break
    // existing clients of Handler on a source code level. The default
    // implementation of Merge does nothing.
    virtual Status MergeCF(uint32_t column_family_id, const Slice& key,
                           const Slice& value) {
      if (column_family_id == 0) {
        Merge(key, value);
        return Status::OK();
      }
      return Status::InvalidArgument(
          "non-default column family and MergeCF not implemented");
    }
    virtual void Merge(const Slice& key, const Slice& value) {}

    // The default implementation of LogData does nothing.
    virtual void LogData(const Slice& blob);
    virtual Status DeleteCF(uint32_t column_family_id, const Slice& key) {
      if (column_family_id == 0) {
        Delete(key);
        return Status::OK();
      }
      return Status::InvalidArgument(
          "non-default column family and DeleteCF not implemented");
    }
    virtual void Delete(const Slice& key) {}

    // Continue is called by WriteBatch::Iterate. If it returns false,
    // iteration is halted. Otherwise, it continues iterating. The default
    // implementation always returns true.
    virtual bool Continue();
  };
  Status Iterate(Handler* handler) const;

  // Retrieve the serialized version of this batch.
  const std::string& Data() const { return rep_; }

  // Retrieve data size of the batch.
  size_t GetDataSize() const { return rep_.size(); }

  // Returns the number of updates in the batch
  int Count() const;

  using WriteBatchBase::GetWriteBatch;
  WriteBatch* GetWriteBatch() override { return this; }

  // Constructor with a serialized string object
  explicit WriteBatch(std::string rep) : save_points_(nullptr), rep_(rep) {}

 private:
  friend class WriteBatchInternal;
  SavePoints* save_points_;

 protected:
  std::string rep_;  // See comment in write_batch.cc for the format of rep_

  // Intentionally copyable
};

}  // namespace rocksdb

#endif  // STORAGE_ROCKSDB_INCLUDE_WRITE_BATCH_H_
