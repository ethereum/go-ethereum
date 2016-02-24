//  Copyright (c) 2015, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.

#ifndef ROCKSDB_LITE

#include "utilities/transactions/transaction_db_impl.h"

#include <string>
#include <vector>

#include "db/db_impl.h"
#include "rocksdb/db.h"
#include "rocksdb/options.h"
#include "rocksdb/utilities/transaction_db.h"
#include "utilities/transactions/transaction_db_mutex_impl.h"
#include "utilities/transactions/transaction_impl.h"

namespace rocksdb {

TransactionDBImpl::TransactionDBImpl(DB* db,
                                     const TransactionDBOptions& txn_db_options)
    : TransactionDB(db),
      txn_db_options_(txn_db_options),
      lock_mgr_(txn_db_options_.num_stripes, txn_db_options.max_num_locks,
                txn_db_options_.custom_mutex_factory
                    ? txn_db_options_.custom_mutex_factory
                    : std::shared_ptr<TransactionDBMutexFactory>(
                          new TransactionDBMutexFactoryImpl())) {}

Transaction* TransactionDBImpl::BeginTransaction(
    const WriteOptions& write_options, const TransactionOptions& txn_options) {
  Transaction* txn = new TransactionImpl(this, write_options, txn_options);

  return txn;
}

TransactionDBOptions TransactionDBImpl::ValidateTxnDBOptions(
    const TransactionDBOptions& txn_db_options) {
  TransactionDBOptions validated = txn_db_options;

  if (txn_db_options.num_stripes == 0) {
    validated.num_stripes = 1;
  }

  return validated;
}

Status TransactionDB::Open(const Options& options,
                           const TransactionDBOptions& txn_db_options,
                           const std::string& dbname, TransactionDB** dbptr) {
  DBOptions db_options(options);
  ColumnFamilyOptions cf_options(options);
  std::vector<ColumnFamilyDescriptor> column_families;
  column_families.push_back(
      ColumnFamilyDescriptor(kDefaultColumnFamilyName, cf_options));
  std::vector<ColumnFamilyHandle*> handles;
  Status s = TransactionDB::Open(db_options, txn_db_options, dbname,
                                 column_families, &handles, dbptr);
  if (s.ok()) {
    assert(handles.size() == 1);
    // i can delete the handle since DBImpl is always holding a reference to
    // default column family
    delete handles[0];
  }

  return s;
}

Status TransactionDB::Open(
    const DBOptions& db_options, const TransactionDBOptions& txn_db_options,
    const std::string& dbname,
    const std::vector<ColumnFamilyDescriptor>& column_families,
    std::vector<ColumnFamilyHandle*>* handles, TransactionDB** dbptr) {
  Status s;
  DB* db;

  std::vector<ColumnFamilyDescriptor> column_families_copy = column_families;

  // Enable MemTable History if not already enabled
  for (auto& column_family : column_families_copy) {
    ColumnFamilyOptions* options = &column_family.options;

    if (options->max_write_buffer_number_to_maintain == 0) {
      // Setting to -1 will set the History size to max_write_buffer_number.
      options->max_write_buffer_number_to_maintain = -1;
    }
  }

  s = DB::Open(db_options, dbname, column_families, handles, &db);

  if (s.ok()) {
    TransactionDBImpl* txn_db = new TransactionDBImpl(
        db, TransactionDBImpl::ValidateTxnDBOptions(txn_db_options));

    for (auto cf_ptr : *handles) {
      txn_db->AddColumnFamily(cf_ptr);
    }

    *dbptr = txn_db;
  }

  return s;
}

// Let TransactionLockMgr know that this column family exists so it can
// allocate a LockMap for it.
void TransactionDBImpl::AddColumnFamily(const ColumnFamilyHandle* handle) {
  lock_mgr_.AddColumnFamily(handle->GetID());
}

Status TransactionDBImpl::CreateColumnFamily(
    const ColumnFamilyOptions& options, const std::string& column_family_name,
    ColumnFamilyHandle** handle) {
  InstrumentedMutexLock l(&column_family_mutex_);

  Status s = db_->CreateColumnFamily(options, column_family_name, handle);
  if (s.ok()) {
    lock_mgr_.AddColumnFamily((*handle)->GetID());
  }

  return s;
}

// Let TransactionLockMgr know that it can deallocate the LockMap for this
// column family.
Status TransactionDBImpl::DropColumnFamily(ColumnFamilyHandle* column_family) {
  InstrumentedMutexLock l(&column_family_mutex_);

  Status s = db_->DropColumnFamily(column_family);
  if (s.ok()) {
    lock_mgr_.RemoveColumnFamily(column_family->GetID());
  }

  return s;
}

Status TransactionDBImpl::TryLock(TransactionImpl* txn, uint32_t cfh_id,
                                  const std::string& key) {
  return lock_mgr_.TryLock(txn, cfh_id, key, GetEnv());
}

void TransactionDBImpl::UnLock(TransactionImpl* txn,
                               const TransactionKeyMap* keys) {
  lock_mgr_.UnLock(txn, keys, GetEnv());
}

void TransactionDBImpl::UnLock(TransactionImpl* txn, uint32_t cfh_id,
                               const std::string& key) {
  lock_mgr_.UnLock(txn, cfh_id, key, GetEnv());
}

// Used when wrapping DB write operations in a transaction
Transaction* TransactionDBImpl::BeginInternalTransaction(
    const WriteOptions& options) {
  TransactionOptions txn_options;
  Transaction* txn = BeginTransaction(options, txn_options);

  assert(dynamic_cast<TransactionImpl*>(txn) != nullptr);
  auto txn_impl = reinterpret_cast<TransactionImpl*>(txn);

  // Use default timeout for non-transactional writes
  txn_impl->SetLockTimeout(txn_db_options_.default_lock_timeout);

  return txn;
}

// All user Put, Merge, Delete, and Write requests must be intercepted to make
// sure that they lock all keys that they are writing to avoid causing conflicts
// with any concurent transactions. The easiest way to do this is to wrap all
// write operations in a transaction.
//
// Put(), Merge(), and Delete() only lock a single key per call.  Write() will
// sort its keys before locking them.  This guarantees that TransactionDB write
// methods cannot deadlock with eachother (but still could deadlock with a
// Transaction).
Status TransactionDBImpl::Put(const WriteOptions& options,
                              ColumnFamilyHandle* column_family,
                              const Slice& key, const Slice& val) {
  Status s;

  Transaction* txn = BeginInternalTransaction(options);

  // Since the client didn't create a transaction, they don't care about
  // conflict checking for this write.  So we just need to do PutUntracked().
  s = txn->PutUntracked(column_family, key, val);

  if (s.ok()) {
    s = txn->Commit();
  }

  delete txn;

  return s;
}

Status TransactionDBImpl::Delete(const WriteOptions& wopts,
                                 ColumnFamilyHandle* column_family,
                                 const Slice& key) {
  Status s;

  Transaction* txn = BeginInternalTransaction(wopts);

  // Since the client didn't create a transaction, they don't care about
  // conflict checking for this write.  So we just need to do
  // DeleteUntracked().
  s = txn->DeleteUntracked(column_family, key);

  if (s.ok()) {
    s = txn->Commit();
  }

  delete txn;

  return s;
}

Status TransactionDBImpl::Merge(const WriteOptions& options,
                                ColumnFamilyHandle* column_family,
                                const Slice& key, const Slice& value) {
  Status s;

  Transaction* txn = BeginInternalTransaction(options);

  // Since the client didn't create a transaction, they don't care about
  // conflict checking for this write.  So we just need to do
  // MergeUntracked().
  s = txn->MergeUntracked(column_family, key, value);

  if (s.ok()) {
    s = txn->Commit();
  }

  delete txn;

  return s;
}

Status TransactionDBImpl::Write(const WriteOptions& opts, WriteBatch* updates) {
  // Need to lock all keys in this batch to prevent write conflicts with
  // concurrent transactions.
  Transaction* txn = BeginInternalTransaction(opts);

  assert(dynamic_cast<TransactionImpl*>(txn) != nullptr);
  auto txn_impl = reinterpret_cast<TransactionImpl*>(txn);

  // Since commitBatch sorts the keys before locking, concurrent Write()
  // operations will not cause a deadlock.
  // In order to avoid a deadlock with a concurrent Transaction, Transactions
  // should use a lock timeout.
  Status s = txn_impl->CommitBatch(updates);

  delete txn;

  return s;
}

}  //  namespace rocksdb
#endif  // ROCKSDB_LITE
