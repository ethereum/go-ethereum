// Copyright (c) 2015, Facebook, Inc.  All rights reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.

#pragma once

#ifndef ROCKSDB_LITE

#include <string>
#include <vector>

#include "rocksdb/comparator.h"
#include "rocksdb/db.h"
#include "rocksdb/status.h"

namespace rocksdb {

class Iterator;
class TransactionDB;
class WriteBatchWithIndex;

// Provides BEGIN/COMMIT/ROLLBACK transactions.
//
// To use transactions, you must first create either an OptimisticTransactionDB
// or a TransactionDB.  See examples/[optimistic_]transaction_example.cc for
// more information.
//
// To create a transaction, use [Optimistic]TransactionDB::BeginTransaction().
//
// It is up to the caller to synchronize access to this object.
//
// See examples/transaction_example.cc for some simple examples.
//
// TODO(agiardullo): Not yet implemented
//  -PerfContext statistics
//  -Support for using Transactions with DBWithTTL
class Transaction {
 public:
  virtual ~Transaction() {}

  // If a transaction has a snapshot set, the transaction will ensure that
  // any keys successfully written(or fetched via GetForUpdate()) have not
  // been modified outside of this transaction since the time the snapshot was
  // set.
  // If a snapshot has not been set, the transaction guarantees that keys have
  // not been modified since the time each key was first written (or fetched via
  // GetForUpdate()).
  //
  // Using SetSnapshot() will provide stricter isolation guarantees at the
  // expense of potentially more transaction failures due to conflicts with
  // other writes.
  //
  // Calling SetSnapshot() has no effect on keys written before this function
  // has been called.
  //
  // SetSnapshot() may be called multiple times if you would like to change
  // the snapshot used for different operations in this transaction.
  //
  // Calling SetSnapshot will not affect the version of Data returned by Get()
  // methods.  See Transaction::Get() for more details.
  virtual void SetSnapshot() = 0;

  // Returns the Snapshot created by the last call to SetSnapshot().
  //
  // REQUIRED: The returned Snapshot is only valid up until the next time
  // SetSnapshot() is called or the Transaction is deleted.
  virtual const Snapshot* GetSnapshot() const = 0;

  // Write all batched keys to the db atomically.
  //
  // Returns OK on success.
  //
  // May return any error status that could be returned by DB:Write().
  //
  // If this transaction was created by an OptimisticTransactionDB(),
  // Status::Busy() may be returned if the transaction could not guarantee
  // that there are no write conflicts.  Status::TryAgain() may be returned
  // if the memtable history size is not large enough
  //  (See max_write_buffer_number_to_maintain).
  //
  // If this transaction was created by a TransactionDB(), Status::Expired()
  // may be returned if this transaction has lived for longer than
  // TransactionOptions.expiration.
  virtual Status Commit() = 0;

  // Discard all batched writes in this transaction.
  virtual void Rollback() = 0;

  // Records the state of the transaction for future calls to
  // RollbackToSavePoint().  May be called multiple times to set multiple save
  // points.
  virtual void SetSavePoint() = 0;

  // Undo all operations in this transaction (Put, Merge, Delete, PutLogData)
  // since the most recent call to SetSavePoint() and removes the most recent
  // SetSavePoint().
  // If there is no previous call to SetSavePoint(), returns Status::NotFound()
  virtual Status RollbackToSavePoint() = 0;

  // This function is similar to DB::Get() except it will also read pending
  // changes in this transaction.  Currently, this function will return
  // Status::MergeInProgress if the most recent write to the queried key in
  // this batch is a Merge.
  //
  // If read_options.snapshot is not set, the current version of the key will
  // be read.  Calling SetSnapshot() does not affect the version of the data
  // returned.
  //
  // Note that setting read_options.snapshot will affect what is read from the
  // DB but will NOT change which keys are read from this transaction (the keys
  // in this transaction do not yet belong to any snapshot and will be fetched
  // regardless).
  virtual Status Get(const ReadOptions& options,
                     ColumnFamilyHandle* column_family, const Slice& key,
                     std::string* value) = 0;

  virtual Status Get(const ReadOptions& options, const Slice& key,
                     std::string* value) = 0;

  virtual std::vector<Status> MultiGet(
      const ReadOptions& options,
      const std::vector<ColumnFamilyHandle*>& column_family,
      const std::vector<Slice>& keys, std::vector<std::string>* values) = 0;

  virtual std::vector<Status> MultiGet(const ReadOptions& options,
                                       const std::vector<Slice>& keys,
                                       std::vector<std::string>* values) = 0;

  // Read this key and ensure that this transaction will only
  // be able to be committed if this key is not written outside this
  // transaction after it has first been read (or after the snapshot if a
  // snapshot is set in this transaction).  The transaction behavior is the
  // same regardless of whether the key exists or not.
  //
  // Note: Currently, this function will return Status::MergeInProgress
  // if the most recent write to the queried key in this batch is a Merge.
  //
  // The values returned by this function are similar to Transaction::Get().
  // If value==nullptr, then this function will not read any data, but will
  // still ensure that this key cannot be written to by outside of this
  // transaction.
  //
  // If this transaction was created by an OptimisticTransaction, GetForUpdate()
  // could cause commit() to fail.  Otherwise, it could return any error
  // that could be returned by DB::Get().
  //
  // If this transaction was created by a TransactionDB, it can return
  // Status::OK() on success,
  // Status::Busy() if there is a write conflict,
  // Status::TimedOut() if a lock could not be acquired,
  // Status::TryAgain() if the memtable history size is not large enough
  //  (See max_write_buffer_number_to_maintain)
  // Status::MergeInProgress() if merge operations cannot be resolved.
  // or other errors if this key could not be read.
  virtual Status GetForUpdate(const ReadOptions& options,
                              ColumnFamilyHandle* column_family,
                              const Slice& key, std::string* value) = 0;

  virtual Status GetForUpdate(const ReadOptions& options, const Slice& key,
                              std::string* value) = 0;

  virtual std::vector<Status> MultiGetForUpdate(
      const ReadOptions& options,
      const std::vector<ColumnFamilyHandle*>& column_family,
      const std::vector<Slice>& keys, std::vector<std::string>* values) = 0;

  virtual std::vector<Status> MultiGetForUpdate(
      const ReadOptions& options, const std::vector<Slice>& keys,
      std::vector<std::string>* values) = 0;

  // Returns an iterator that will iterate on all keys in the default
  // column family including both keys in the DB and uncommitted keys in this
  // transaction.
  //
  // Setting read_options.snapshot will affect what is read from the
  // DB but will NOT change which keys are read from this transaction (the keys
  // in this transaction do not yet belong to any snapshot and will be fetched
  // regardless).
  //
  // Caller is reponsible for deleting the returned Iterator.
  //
  // The returned iterator is only valid until Commit(), Rollback(), or
  // RollbackToSavePoint() is called.
  // NOTE: Transaction::Put/Merge/Delete will currently invalidate this iterator
  // until
  // the following issue is fixed:
  // https://github.com/facebook/rocksdb/issues/616
  virtual Iterator* GetIterator(const ReadOptions& read_options) = 0;

  virtual Iterator* GetIterator(const ReadOptions& read_options,
                                ColumnFamilyHandle* column_family) = 0;

  // Put, Merge, and Delete behave similarly to their corresponding
  // functions in WriteBatch, but will also do conflict checking on the
  // keys being written.
  //
  // If this Transaction was created on an OptimisticTransactionDB, these
  // functions should always return Status::OK().
  //
  // If this Transaction was created on a TransactionDB, the status returned
  // can be:
  // Status::OK() on success,
  // Status::Busy() if there is a write conflict,
  // Status::TimedOut() if a lock could not be acquired,
  // Status::TryAgain() if the memtable history size is not large enough
  //  (See max_write_buffer_number_to_maintain)
  // or other errors on unexpected failures.
  virtual Status Put(ColumnFamilyHandle* column_family, const Slice& key,
                     const Slice& value) = 0;
  virtual Status Put(const Slice& key, const Slice& value) = 0;
  virtual Status Put(ColumnFamilyHandle* column_family, const SliceParts& key,
                     const SliceParts& value) = 0;
  virtual Status Put(const SliceParts& key, const SliceParts& value) = 0;

  virtual Status Merge(ColumnFamilyHandle* column_family, const Slice& key,
                       const Slice& value) = 0;
  virtual Status Merge(const Slice& key, const Slice& value) = 0;

  virtual Status Delete(ColumnFamilyHandle* column_family,
                        const Slice& key) = 0;
  virtual Status Delete(const Slice& key) = 0;
  virtual Status Delete(ColumnFamilyHandle* column_family,
                        const SliceParts& key) = 0;
  virtual Status Delete(const SliceParts& key) = 0;

  // PutUntracked() will write a Put to the batch of operations to be committed
  // in this transaction.  This write will only happen if this transaction
  // gets committed successfully.  But unlike Transaction::Put(),
  // no conflict checking will be done for this key.
  //
  // If this Transaction was created on a TransactionDB, this function will
  // still acquire locks necessary to make sure this write doesn't cause
  // conflicts in other transactions and may return Status::Busy().
  virtual Status PutUntracked(ColumnFamilyHandle* column_family,
                              const Slice& key, const Slice& value) = 0;
  virtual Status PutUntracked(const Slice& key, const Slice& value) = 0;
  virtual Status PutUntracked(ColumnFamilyHandle* column_family,
                              const SliceParts& key,
                              const SliceParts& value) = 0;
  virtual Status PutUntracked(const SliceParts& key,
                              const SliceParts& value) = 0;

  virtual Status MergeUntracked(ColumnFamilyHandle* column_family,
                                const Slice& key, const Slice& value) = 0;
  virtual Status MergeUntracked(const Slice& key, const Slice& value) = 0;

  virtual Status DeleteUntracked(ColumnFamilyHandle* column_family,
                                 const Slice& key) = 0;

  virtual Status DeleteUntracked(const Slice& key) = 0;
  virtual Status DeleteUntracked(ColumnFamilyHandle* column_family,
                                 const SliceParts& key) = 0;
  virtual Status DeleteUntracked(const SliceParts& key) = 0;

  // Similar to WriteBatch::PutLogData
  virtual void PutLogData(const Slice& blob) = 0;

  // Returns the number of distinct Keys being tracked by this transaction.
  // If this transaction was created by a TransactinDB, this is the number of
  // keys that are currently locked by this transaction.
  // If this transaction was created by an OptimisticTransactionDB, this is the
  // number of keys that need to be checked for conflicts at commit time.
  virtual uint64_t GetNumKeys() const = 0;

  // Returns the number of Puts/Deletes/Merges that have been applied to this
  // transaction so far.
  virtual uint64_t GetNumPuts() const = 0;
  virtual uint64_t GetNumDeletes() const = 0;
  virtual uint64_t GetNumMerges() const = 0;

  // Returns the elapsed time in milliseconds since this Transaction began.
  virtual uint64_t GetElapsedTime() const = 0;

  // Fetch the underlying write batch that contains all pending changes to be
  // committed.
  //
  // Note:  You should not write or delete anything from the batch directly and
  // should only use the the functions in the Transaction class to
  // write to this transaction.
  virtual WriteBatchWithIndex* GetWriteBatch() = 0;

  // Change the value of TransactionOptions.lock_timeout (in milliseconds) for
  // this transaction.
  // Has no effect on OptimisticTransactions.
  virtual void SetLockTimeout(int64_t timeout) = 0;

 protected:
  explicit Transaction(const TransactionDB* db) {}
  Transaction() {}

 private:
  // No copying allowed
  Transaction(const Transaction&);
  void operator=(const Transaction&);
};

}  // namespace rocksdb

#endif  // ROCKSDB_LITE
