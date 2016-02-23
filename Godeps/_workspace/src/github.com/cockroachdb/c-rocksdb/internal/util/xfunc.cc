//  Copyright (c) 2014, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.

#ifdef XFUNC
#include <string>
#include "db/db_impl.h"
#include "db/managed_iterator.h"
#include "db/write_callback.h"
#include "rocksdb/db.h"
#include "rocksdb/options.h"
#include "rocksdb/utilities/optimistic_transaction.h"
#include "rocksdb/utilities/optimistic_transaction_db.h"
#include "rocksdb/write_batch.h"
#include "util/xfunc.h"


namespace rocksdb {

std::string XFuncPoint::xfunc_test_;
bool XFuncPoint::initialized_ = false;
bool XFuncPoint::enabled_ = false;
int XFuncPoint::skip_policy_ = 0;

void GetXFTestOptions(Options* options, int skip_policy) {
  if (XFuncPoint::Check("inplace_lock_test") &&
      (!(skip_policy & kSkipNoSnapshot))) {
    options->inplace_update_support = true;
  }
}

void xf_manage_release(ManagedIterator* iter) {
  if (!(XFuncPoint::GetSkip() & kSkipNoPrefix)) {
    iter->ReleaseIter(false);
  }
}

void xf_manage_options(ReadOptions* read_options) {
  if (!XFuncPoint::Check("managed_xftest_dropold") &&
      (!XFuncPoint::Check("managed_xftest_release"))) {
    return;
  }
  read_options->managed = true;
}

void xf_manage_new(DBImpl* db, ReadOptions* read_options,
                   bool is_snapshot_supported) {
  if ((!XFuncPoint::Check("managed_xftest_dropold") &&
       (!XFuncPoint::Check("managed_xftest_release"))) ||
      (!read_options->managed)) {
    return;
  }
  if ((!read_options->tailing) && (read_options->snapshot == nullptr) &&
      (!is_snapshot_supported)) {
    read_options->managed = false;
    return;
  }
  if (db->GetOptions().prefix_extractor != nullptr) {
    if (strcmp(db->GetOptions().table_factory.get()->Name(), "PlainTable")) {
      if (!(XFuncPoint::GetSkip() & kSkipNoPrefix)) {
        read_options->total_order_seek = true;
      }
    } else {
      read_options->managed = false;
    }
  }
}

void xf_manage_create(ManagedIterator* iter) { iter->SetDropOld(false); }

void xf_transaction_set_memtable_history(
    int32_t* max_write_buffer_number_to_maintain) {
  *max_write_buffer_number_to_maintain = 10;
}

void xf_transaction_clear_memtable_history(
    int32_t* max_write_buffer_number_to_maintain) {
  *max_write_buffer_number_to_maintain = 0;
}

class XFTransactionWriteHandler : public WriteBatch::Handler {
 public:
  OptimisticTransaction* txn_;
  DBImpl* db_impl_;

  XFTransactionWriteHandler(OptimisticTransaction* txn, DBImpl* db_impl)
      : txn_(txn), db_impl_(db_impl) {}

  virtual Status PutCF(uint32_t column_family_id, const Slice& key,
                       const Slice& value) override {
    InstrumentedMutexLock l(&db_impl_->mutex_);

    ColumnFamilyHandle* cfh = db_impl_->GetColumnFamilyHandle(column_family_id);
    if (cfh == nullptr) {
      return Status::InvalidArgument(
          "XFUNC test could not find column family "
          "handle for id ",
          ToString(column_family_id));
    }

    txn_->Put(cfh, key, value);

    return Status::OK();
  }

  virtual Status MergeCF(uint32_t column_family_id, const Slice& key,
                         const Slice& value) override {
    InstrumentedMutexLock l(&db_impl_->mutex_);

    ColumnFamilyHandle* cfh = db_impl_->GetColumnFamilyHandle(column_family_id);
    if (cfh == nullptr) {
      return Status::InvalidArgument(
          "XFUNC test could not find column family "
          "handle for id ",
          ToString(column_family_id));
    }

    txn_->Merge(cfh, key, value);

    return Status::OK();
  }

  virtual Status DeleteCF(uint32_t column_family_id,
                          const Slice& key) override {
    InstrumentedMutexLock l(&db_impl_->mutex_);

    ColumnFamilyHandle* cfh = db_impl_->GetColumnFamilyHandle(column_family_id);
    if (cfh == nullptr) {
      return Status::InvalidArgument(
          "XFUNC test could not find column family "
          "handle for id ",
          ToString(column_family_id));
    }

    txn_->Delete(cfh, key);

    return Status::OK();
  }

  virtual void LogData(const Slice& blob) override { txn_->PutLogData(blob); }
};

// Whenever DBImpl::Write is called, create a transaction and do the write via
// the transaction.
void xf_transaction_write(const WriteOptions& write_options,
                          const DBOptions& db_options, WriteBatch* my_batch,
                          WriteCallback* callback, DBImpl* db_impl, Status* s,
                          bool* write_attempted) {
  if (callback != nullptr) {
    // We may already be in a transaction, don't force a transaction
    *write_attempted = false;
    return;
  }

  OptimisticTransactionDB* txn_db = new OptimisticTransactionDB(db_impl);
  OptimisticTransaction* txn =
      OptimisticTransaction::BeginTransaction(txn_db, write_options);

  XFTransactionWriteHandler handler(txn, db_impl);
  *s = my_batch->Iterate(&handler);

  if (!s->ok()) {
    Log(InfoLogLevel::ERROR_LEVEL, db_options.info_log,
        "XFUNC test could not iterate batch.  status: $s\n",
        s->ToString().c_str());
  }

  *s = txn->Commit();

  if (!s->ok()) {
    Log(InfoLogLevel::ERROR_LEVEL, db_options.info_log,
        "XFUNC test could not commit transaction.  status: $s\n",
        s->ToString().c_str());
  }

  *write_attempted = true;
  delete txn;
  delete txn_db;
}

}  // namespace rocksdb

#endif  // XFUNC
