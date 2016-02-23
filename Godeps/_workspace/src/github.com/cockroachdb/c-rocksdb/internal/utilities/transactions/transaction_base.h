// Copyright (c) 2015, Facebook, Inc.  All rights reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.

#pragma once

#ifndef ROCKSDB_LITE

#include <stack>
#include <string>
#include <vector>

#include "rocksdb/db.h"
#include "rocksdb/slice.h"
#include "rocksdb/snapshot.h"
#include "rocksdb/status.h"
#include "rocksdb/types.h"
#include "rocksdb/utilities/transaction.h"
#include "rocksdb/utilities/transaction_db.h"
#include "rocksdb/utilities/write_batch_with_index.h"
#include "utilities/transactions/transaction_util.h"

namespace rocksdb {

class TransactionBaseImpl : public Transaction {
 public:
  TransactionBaseImpl(DB* db, const WriteOptions& write_options);

  virtual ~TransactionBaseImpl();

  // Remove pending operations queued in this transaction.
  virtual void Clear();

  // Called before executing Put, Merge, Delete, and GetForUpdate.  If TryLock
  // returns non-OK, the Put/Merge/Delete/GetForUpdate will be failed.
  // untracked will be true if called from PutUntracked, DeleteUntracked, or
  // MergeUntracked.
  virtual Status TryLock(ColumnFamilyHandle* column_family, const Slice& key,
                         bool untracked = false) = 0;

  void SetSavePoint() override;

  Status RollbackToSavePoint() override;

  Status Get(const ReadOptions& options, ColumnFamilyHandle* column_family,
             const Slice& key, std::string* value) override;

  Status Get(const ReadOptions& options, const Slice& key,
             std::string* value) override {
    return Get(options, db_->DefaultColumnFamily(), key, value);
  }

  Status GetForUpdate(const ReadOptions& options,
                      ColumnFamilyHandle* column_family, const Slice& key,
                      std::string* value) override;

  Status GetForUpdate(const ReadOptions& options, const Slice& key,
                      std::string* value) override {
    return GetForUpdate(options, db_->DefaultColumnFamily(), key, value);
  }

  std::vector<Status> MultiGet(
      const ReadOptions& options,
      const std::vector<ColumnFamilyHandle*>& column_family,
      const std::vector<Slice>& keys,
      std::vector<std::string>* values) override;

  std::vector<Status> MultiGet(const ReadOptions& options,
                               const std::vector<Slice>& keys,
                               std::vector<std::string>* values) override {
    return MultiGet(options, std::vector<ColumnFamilyHandle*>(
                                 keys.size(), db_->DefaultColumnFamily()),
                    keys, values);
  }

  std::vector<Status> MultiGetForUpdate(
      const ReadOptions& options,
      const std::vector<ColumnFamilyHandle*>& column_family,
      const std::vector<Slice>& keys,
      std::vector<std::string>* values) override;

  std::vector<Status> MultiGetForUpdate(
      const ReadOptions& options, const std::vector<Slice>& keys,
      std::vector<std::string>* values) override {
    return MultiGetForUpdate(options,
                             std::vector<ColumnFamilyHandle*>(
                                 keys.size(), db_->DefaultColumnFamily()),
                             keys, values);
  }

  Iterator* GetIterator(const ReadOptions& read_options) override;
  Iterator* GetIterator(const ReadOptions& read_options,
                        ColumnFamilyHandle* column_family) override;

  Status Put(ColumnFamilyHandle* column_family, const Slice& key,
             const Slice& value) override;
  Status Put(const Slice& key, const Slice& value) override {
    return Put(nullptr, key, value);
  }

  Status Put(ColumnFamilyHandle* column_family, const SliceParts& key,
             const SliceParts& value) override;
  Status Put(const SliceParts& key, const SliceParts& value) override {
    return Put(nullptr, key, value);
  }

  Status Merge(ColumnFamilyHandle* column_family, const Slice& key,
               const Slice& value) override;
  Status Merge(const Slice& key, const Slice& value) override {
    return Merge(nullptr, key, value);
  }

  Status Delete(ColumnFamilyHandle* column_family, const Slice& key) override;
  Status Delete(const Slice& key) override { return Delete(nullptr, key); }
  Status Delete(ColumnFamilyHandle* column_family,
                const SliceParts& key) override;
  Status Delete(const SliceParts& key) override { return Delete(nullptr, key); }

  Status PutUntracked(ColumnFamilyHandle* column_family, const Slice& key,
                      const Slice& value) override;
  Status PutUntracked(const Slice& key, const Slice& value) override {
    return PutUntracked(nullptr, key, value);
  }

  Status PutUntracked(ColumnFamilyHandle* column_family, const SliceParts& key,
                      const SliceParts& value) override;
  Status PutUntracked(const SliceParts& key, const SliceParts& value) override {
    return PutUntracked(nullptr, key, value);
  }

  Status MergeUntracked(ColumnFamilyHandle* column_family, const Slice& key,
                        const Slice& value) override;
  Status MergeUntracked(const Slice& key, const Slice& value) override {
    return MergeUntracked(nullptr, key, value);
  }

  Status DeleteUntracked(ColumnFamilyHandle* column_family,
                         const Slice& key) override;
  Status DeleteUntracked(const Slice& key) override {
    return DeleteUntracked(nullptr, key);
  }
  Status DeleteUntracked(ColumnFamilyHandle* column_family,
                         const SliceParts& key) override;
  Status DeleteUntracked(const SliceParts& key) override {
    return DeleteUntracked(nullptr, key);
  }

  void PutLogData(const Slice& blob) override;

  WriteBatchWithIndex* GetWriteBatch() override;

  virtual void SetLockTimeout(int64_t timeout) override { /* Do nothing */
  }

  const Snapshot* GetSnapshot() const override {
    return snapshot_ ? snapshot_->snapshot() : nullptr;
  }

  void SetSnapshot() override;

  uint64_t GetElapsedTime() const override;

  uint64_t GetNumPuts() const override;

  uint64_t GetNumDeletes() const override;

  uint64_t GetNumMerges() const override;

  uint64_t GetNumKeys() const override;

  // Get list of keys in this transaction that must not have any conflicts
  // with writes in other transactions.
  const TransactionKeyMap& GetTrackedKeys() const { return tracked_keys_; }

 protected:
  // Add a key to the list of tracked keys.
  // seqno is the earliest seqno this key was involved with this transaction.
  void TrackKey(uint32_t cfh_id, const std::string& key, SequenceNumber seqno);

  const TransactionKeyMap* GetTrackedKeysSinceSavePoint();

  DB* const db_;

  const WriteOptions write_options_;

  const Comparator* cmp_;

  // Records writes pending in this transaction
  std::unique_ptr<WriteBatchWithIndex> write_batch_;

  // Stores that time the txn was constructed, in microseconds.
  const uint64_t start_time_;

  // Stores the current snapshot that was was set by SetSnapshot or null if
  // no snapshot is currently set.
  std::shared_ptr<ManagedSnapshot> snapshot_;

  // Count of various operations pending in this transaction
  uint64_t num_puts_ = 0;
  uint64_t num_deletes_ = 0;
  uint64_t num_merges_ = 0;

  struct SavePoint {
    std::shared_ptr<ManagedSnapshot> snapshot_;
    uint64_t num_puts_;
    uint64_t num_deletes_;
    uint64_t num_merges_;

    // Record all keys tracked since the last savepoint
    TransactionKeyMap new_keys_;

    SavePoint(std::shared_ptr<ManagedSnapshot> snapshot, uint64_t num_puts,
              uint64_t num_deletes, uint64_t num_merges)
        : snapshot_(snapshot),
          num_puts_(num_puts),
          num_deletes_(num_deletes),
          num_merges_(num_merges) {}
  };

 private:
  // Stack of the Snapshot saved at each save point.  Saved snapshots may be
  // nullptr if there was no snapshot at the time SetSavePoint() was called.
  std::unique_ptr<std::stack<TransactionBaseImpl::SavePoint>> save_points_;

  // Map from column_family_id to map of keys that are involved in this
  // transaction.
  // Pessimistic Transactions will do conflict checking before adding a key
  // by calling TrackKey().
  // Optimistic Transactions will wait till commit time to do conflict checking.
  TransactionKeyMap tracked_keys_;

  Status TryLock(ColumnFamilyHandle* column_family, const SliceParts& key,
                 bool untracked = false);
};

}  // namespace rocksdb

#endif  // ROCKSDB_LITE
