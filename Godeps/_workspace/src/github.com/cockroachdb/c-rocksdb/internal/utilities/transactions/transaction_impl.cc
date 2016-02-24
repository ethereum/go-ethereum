//  Copyright (c) 2015, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.

#ifndef ROCKSDB_LITE

#include "utilities/transactions/transaction_impl.h"

#include <map>
#include <set>
#include <string>
#include <vector>

#include "db/column_family.h"
#include "db/db_impl.h"
#include "rocksdb/comparator.h"
#include "rocksdb/db.h"
#include "rocksdb/snapshot.h"
#include "rocksdb/status.h"
#include "rocksdb/utilities/transaction_db.h"
#include "util/string_util.h"
#include "utilities/transactions/transaction_db_impl.h"
#include "utilities/transactions/transaction_util.h"

namespace rocksdb {

struct WriteOptions;

std::atomic<TransactionID> TransactionImpl::txn_id_counter_(1);

TransactionID TransactionImpl::GenTxnID() {
  return txn_id_counter_.fetch_add(1);
}

TransactionImpl::TransactionImpl(TransactionDB* txn_db,
                                 const WriteOptions& write_options,
                                 const TransactionOptions& txn_options)
    : TransactionBaseImpl(txn_db->GetBaseDB(), write_options),
      txn_db_impl_(nullptr),
      txn_id_(GenTxnID()),
      expiration_time_(txn_options.expiration >= 0
                           ? start_time_ + txn_options.expiration * 1000
                           : 0),
      lock_timeout_(txn_options.lock_timeout * 1000) {
  txn_db_impl_ = dynamic_cast<TransactionDBImpl*>(txn_db);
  assert(txn_db_impl_);

  if (lock_timeout_ < 0) {
    // Lock timeout not set, use default
    lock_timeout_ =
        txn_db_impl_->GetTxnDBOptions().transaction_lock_timeout * 1000;
  }

  if (txn_options.set_snapshot) {
    SetSnapshot();
  }
}

TransactionImpl::~TransactionImpl() {
  txn_db_impl_->UnLock(this, &GetTrackedKeys());
}

void TransactionImpl::Clear() {
  txn_db_impl_->UnLock(this, &GetTrackedKeys());
  TransactionBaseImpl::Clear();
}

bool TransactionImpl::IsExpired() const {
  if (expiration_time_ > 0) {
    if (db_->GetEnv()->NowMicros() >= expiration_time_) {
      // Transaction is expired.
      return true;
    }
  }

  return false;
}

Status TransactionImpl::CommitBatch(WriteBatch* batch) {
  TransactionKeyMap keys_to_unlock;

  Status s = LockBatch(batch, &keys_to_unlock);

  if (s.ok()) {
    s = DoCommit(batch);

    txn_db_impl_->UnLock(this, &keys_to_unlock);
  }

  return s;
}

Status TransactionImpl::Commit() {
  Status s = DoCommit(write_batch_->GetWriteBatch());

  Clear();

  return s;
}

Status TransactionImpl::DoCommit(WriteBatch* batch) {
  Status s;

  if (expiration_time_ > 0) {
    // We cannot commit a transaction that is expired as its locks might have
    // been released.
    // To avoid race conditions, we need to use a WriteCallback to check the
    // expiration time once we're on the writer thread.
    TransactionCallback callback(this);

    // Do write directly on base db as TransctionDB::Write() would attempt to
    // do conflict checking that we've already done.
    assert(dynamic_cast<DBImpl*>(db_) != nullptr);
    auto db_impl = reinterpret_cast<DBImpl*>(db_);

    s = db_impl->WriteWithCallback(write_options_, batch, &callback);
  } else {
    s = db_->Write(write_options_, batch);
  }

  return s;
}

void TransactionImpl::Rollback() { Clear(); }

Status TransactionImpl::RollbackToSavePoint() {
  // Unlock any keys locked since last transaction
  const TransactionKeyMap* keys = GetTrackedKeysSinceSavePoint();
  if (keys) {
    txn_db_impl_->UnLock(this, keys);
  }

  return TransactionBaseImpl::RollbackToSavePoint();
}

// Lock all keys in this batch.
// On success, caller should unlock keys_to_unlock
Status TransactionImpl::LockBatch(WriteBatch* batch,
                                  TransactionKeyMap* keys_to_unlock) {
  class Handler : public WriteBatch::Handler {
   public:
    // Sorted map of column_family_id to sorted set of keys.
    // Since LockBatch() always locks keys in sorted order, it cannot deadlock
    // with itself.  We're not using a comparator here since it doesn't matter
    // what the sorting is as long as it's consistent.
    std::map<uint32_t, std::set<std::string>> keys_;

    Handler() {}

    void RecordKey(uint32_t column_family_id, const Slice& key) {
      std::string key_str = key.ToString();

      auto iter = (keys_)[column_family_id].find(key_str);
      if (iter == (keys_)[column_family_id].end()) {
        // key not yet seen, store it.
        (keys_)[column_family_id].insert({std::move(key_str)});
      }
    }

    virtual Status PutCF(uint32_t column_family_id, const Slice& key,
                         const Slice& value) override {
      RecordKey(column_family_id, key);
      return Status::OK();
    }
    virtual Status MergeCF(uint32_t column_family_id, const Slice& key,
                           const Slice& value) override {
      RecordKey(column_family_id, key);
      return Status::OK();
    }
    virtual Status DeleteCF(uint32_t column_family_id,
                            const Slice& key) override {
      RecordKey(column_family_id, key);
      return Status::OK();
    }
  };

  // Iterating on this handler will add all keys in this batch into keys
  Handler handler;
  batch->Iterate(&handler);

  Status s;

  // Attempt to lock all keys
  for (const auto& cf_iter : handler.keys_) {
    uint32_t cfh_id = cf_iter.first;
    auto& cfh_keys = cf_iter.second;

    for (const auto& key_iter : cfh_keys) {
      const std::string& key = key_iter;

      s = txn_db_impl_->TryLock(this, cfh_id, key);
      if (!s.ok()) {
        break;
      }
      (*keys_to_unlock)[cfh_id].insert({std::move(key), kMaxSequenceNumber});
    }

    if (!s.ok()) {
      break;
    }
  }

  if (!s.ok()) {
    txn_db_impl_->UnLock(this, keys_to_unlock);
  }

  return s;
}

// Attempt to lock this key.
// Returns OK if the key has been successfully locked.  Non-ok, otherwise.
// If check_shapshot is true and this transaction has a snapshot set,
// this key will only be locked if there have been no writes to this key since
// the snapshot time.
Status TransactionImpl::TryLock(ColumnFamilyHandle* column_family,
                                const Slice& key, bool untracked) {
  uint32_t cfh_id = GetColumnFamilyID(column_family);
  std::string key_str = key.ToString();
  bool previously_locked;
  Status s;

  // Even though we do not care about doing conflict checking for this write,
  // we still need to take a lock to make sure we do not cause a conflict with
  // some other write.  However, we do not need to check if there have been
  // any writes since this transaction's snapshot.
  // TODO(agiardullo): could optimize by supporting shared txn locks in the
  // future
  bool check_snapshot = !untracked;
  SequenceNumber tracked_seqno = kMaxSequenceNumber;

  // Lookup whether this key has already been locked by this transaction
  const auto& tracked_keys = GetTrackedKeys();
  const auto tracked_keys_cf = tracked_keys.find(cfh_id);
  if (tracked_keys_cf == tracked_keys.end()) {
    previously_locked = false;
  } else {
    auto iter = tracked_keys_cf->second.find(key_str);
    if (iter == tracked_keys_cf->second.end()) {
      previously_locked = false;
    } else {
      previously_locked = true;
      tracked_seqno = iter->second;
    }
  }

  // lock this key if this transactions hasn't already locked it
  if (!previously_locked) {
    s = txn_db_impl_->TryLock(this, cfh_id, key_str);
  }

  if (s.ok()) {
    // If a snapshot is set, we need to make sure the key hasn't been modified
    // since the snapshot.  This must be done after we locked the key.
    if (!check_snapshot || snapshot_ == nullptr) {
      // Need to remember the earliest sequence number that we know that this
      // key has not been modified after.  This is useful if this same
      // transaction
      // later tries to lock this key again.
      if (tracked_seqno == kMaxSequenceNumber) {
        // Since we haven't checked a snapshot, we only know this key has not
        // been modified since after we locked it.
        tracked_seqno = db_->GetLatestSequenceNumber();
      }
    } else {
      // If the key has been previous validated at a sequence number earlier
      // than the curent snapshot's sequence number, we already know it has not
      // been modified.
      SequenceNumber seq = snapshot_->snapshot()->GetSequenceNumber();
      bool already_validated = tracked_seqno <= seq;

      if (!already_validated) {
        s = CheckKeySequence(column_family, key);

        if (s.ok()) {
          // Record that there have been no writes to this key after this
          // sequence.
          tracked_seqno = seq;
        } else {
          // Failed to validate key
          if (!previously_locked) {
            // Unlock key we just locked
            txn_db_impl_->UnLock(this, cfh_id, key.ToString());
          }
        }
      }
    }
  }

  if (s.ok()) {
    // Let base class know we've conflict checked this key.
    TrackKey(cfh_id, key_str, tracked_seqno);
  }

  return s;
}

// Return OK() if this key has not been modified more recently than the
// transaction snapshot_.
Status TransactionImpl::CheckKeySequence(ColumnFamilyHandle* column_family,
                                         const Slice& key) {
  Status result;
  if (snapshot_ != nullptr) {
    assert(dynamic_cast<DBImpl*>(db_) != nullptr);
    auto db_impl = reinterpret_cast<DBImpl*>(db_);

    ColumnFamilyHandle* cfh = column_family ? column_family :
      db_impl->DefaultColumnFamily();

    result = TransactionUtil::CheckKeyForConflicts(
        db_impl, cfh, key.ToString(),
        snapshot_->snapshot()->GetSequenceNumber());
  }

  return result;
}

}  // namespace rocksdb

#endif  // ROCKSDB_LITE
