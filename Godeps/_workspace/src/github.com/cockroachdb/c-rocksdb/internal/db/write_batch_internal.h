//  Copyright (c) 2013, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.
//
// Copyright (c) 2011 The LevelDB Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file. See the AUTHORS file for names of contributors.

#pragma once
#include "rocksdb/types.h"
#include "rocksdb/write_batch.h"
#include "rocksdb/db.h"
#include "rocksdb/options.h"

namespace rocksdb {

class MemTable;

class ColumnFamilyMemTables {
 public:
  virtual ~ColumnFamilyMemTables() {}
  virtual bool Seek(uint32_t column_family_id) = 0;
  // returns true if the update to memtable should be ignored
  // (useful when recovering from log whose updates have already
  // been processed)
  virtual uint64_t GetLogNumber() const = 0;
  virtual MemTable* GetMemTable() const = 0;
  virtual ColumnFamilyHandle* GetColumnFamilyHandle() = 0;
  virtual void CheckMemtableFull() = 0;
};

class ColumnFamilyMemTablesDefault : public ColumnFamilyMemTables {
 public:
  explicit ColumnFamilyMemTablesDefault(MemTable* mem)
      : ok_(false), mem_(mem) {}

  bool Seek(uint32_t column_family_id) override {
    ok_ = (column_family_id == 0);
    return ok_;
  }

  uint64_t GetLogNumber() const override { return 0; }

  MemTable* GetMemTable() const override {
    assert(ok_);
    return mem_;
  }

  ColumnFamilyHandle* GetColumnFamilyHandle() override { return nullptr; }

  void CheckMemtableFull() override {}

 private:
  bool ok_;
  MemTable* mem_;
};

// WriteBatchInternal provides static methods for manipulating a
// WriteBatch that we don't want in the public WriteBatch interface.
class WriteBatchInternal {
 public:
  // WriteBatch methods with column_family_id instead of ColumnFamilyHandle*
  static void Put(WriteBatch* batch, uint32_t column_family_id,
                  const Slice& key, const Slice& value);

  static void Put(WriteBatch* batch, uint32_t column_family_id,
                  const SliceParts& key, const SliceParts& value);

  static void Delete(WriteBatch* batch, uint32_t column_family_id,
                     const SliceParts& key);

  static void Delete(WriteBatch* batch, uint32_t column_family_id,
                     const Slice& key);

  static void Merge(WriteBatch* batch, uint32_t column_family_id,
                    const Slice& key, const Slice& value);

  static void Merge(WriteBatch* batch, uint32_t column_family_id,
                    const SliceParts& key, const SliceParts& value);

  // Return the number of entries in the batch.
  static int Count(const WriteBatch* batch);

  // Set the count for the number of entries in the batch.
  static void SetCount(WriteBatch* batch, int n);

  // Return the seqeunce number for the start of this batch.
  static SequenceNumber Sequence(const WriteBatch* batch);

  // Store the specified number as the seqeunce number for the start of
  // this batch.
  static void SetSequence(WriteBatch* batch, SequenceNumber seq);

  // Returns the offset of the first entry in the batch.
  // This offset is only valid if the batch is not empty.
  static size_t GetFirstOffset(WriteBatch* batch);

  static Slice Contents(const WriteBatch* batch) {
    return Slice(batch->rep_);
  }

  static size_t ByteSize(const WriteBatch* batch) {
    return batch->rep_.size();
  }

  static void SetContents(WriteBatch* batch, const Slice& contents);

  // Inserts batch entries into memtable
  // If dont_filter_deletes is false AND options.filter_deletes is true,
  // then --> Drops deletes in batch if db->KeyMayExist returns false
  // If ignore_missing_column_families == true. WriteBatch referencing
  // non-existing column family should be ignored.
  // However, if ignore_missing_column_families == false, any WriteBatch
  // referencing non-existing column family will return a InvalidArgument()
  // failure.
  //
  // If log_number is non-zero, the memtable will be updated only if
  // memtables->GetLogNumber() >= log_number
  static Status InsertInto(const WriteBatch* batch,
                           ColumnFamilyMemTables* memtables,
                           bool ignore_missing_column_families = false,
                           uint64_t log_number = 0, DB* db = nullptr,
                           const bool dont_filter_deletes = true);

  static void Append(WriteBatch* dst, const WriteBatch* src);
};

}  // namespace rocksdb
