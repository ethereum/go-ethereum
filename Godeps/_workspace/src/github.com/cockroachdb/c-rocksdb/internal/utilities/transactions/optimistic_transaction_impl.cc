//  Copyright (c) 2015, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.

#ifndef ROCKSDB_LITE

#include "utilities/transactions/optimistic_transaction_impl.h"

#include <algorithm>
#include <string>
#include <vector>

#include "db/column_family.h"
#include "db/db_impl.h"
#include "rocksdb/comparator.h"
#include "rocksdb/db.h"
#include "rocksdb/status.h"
#include "rocksdb/utilities/optimistic_transaction_db.h"
#include "util/string_util.h"
#include "utilities/transactions/transaction_util.h"

namespace rocksdb {

struct WriteOptions;

OptimisticTransactionImpl::OptimisticTransactionImpl(
    OptimisticTransactionDB* txn_db, const WriteOptions& write_options,
    const OptimisticTransactionOptions& txn_options)
    : TransactionBaseImpl(txn_db->GetBaseDB(), write_options), txn_db_(txn_db) {
  if (txn_options.set_snapshot) {
    SetSnapshot();
  }
}

OptimisticTransactionImpl::~OptimisticTransactionImpl() {
}

void OptimisticTransactionImpl::Clear() {
  TransactionBaseImpl::Clear();
}

Status OptimisticTransactionImpl::Commit() {
  // Set up callback which will call CheckTransactionForConflicts() to
  // check whether this transaction is safe to be committed.
  OptimisticTransactionCallback callback(this);

  DBImpl* db_impl = dynamic_cast<DBImpl*>(db_->GetRootDB());
  if (db_impl == nullptr) {
    // This should only happen if we support creating transactions from
    // a StackableDB and someone overrides GetRootDB().
    return Status::InvalidArgument(
        "DB::GetRootDB() returned an unexpected DB class");
  }

  Status s = db_impl->WriteWithCallback(
      write_options_, write_batch_->GetWriteBatch(), &callback);

  if (s.ok()) {
    Clear();
  }

  return s;
}

void OptimisticTransactionImpl::Rollback() { Clear(); }

// Record this key so that we can check it for conflicts at commit time.
Status OptimisticTransactionImpl::TryLock(ColumnFamilyHandle* column_family,
                                          const Slice& key, bool untracked) {
  if (untracked) {
    return Status::OK();
  }
  uint32_t cfh_id = GetColumnFamilyID(column_family);

  SequenceNumber seq;
  if (snapshot_) {
    seq = snapshot_->snapshot()->GetSequenceNumber();
  } else {
    seq = db_->GetLatestSequenceNumber();
  }

  std::string key_str = key.ToString();

  TrackKey(cfh_id, key_str, seq);

  // Always return OK. Confilct checking will happen at commit time.
  return Status::OK();
}

// Returns OK if it is safe to commit this transaction.  Returns Status::Busy
// if there are read or write conflicts that would prevent us from committing OR
// if we can not determine whether there would be any such conflicts.
//
// Should only be called on writer thread in order to avoid any race conditions
// in detecting
// write conflicts.
Status OptimisticTransactionImpl::CheckTransactionForConflicts(DB* db) {
  Status result;

  assert(dynamic_cast<DBImpl*>(db) != nullptr);
  auto db_impl = reinterpret_cast<DBImpl*>(db);

  return TransactionUtil::CheckKeysForConflicts(db_impl, GetTrackedKeys());
}

}  // namespace rocksdb

#endif  // ROCKSDB_LITE
