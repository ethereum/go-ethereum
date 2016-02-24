//  Copyright (c) 2015, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.

#ifndef ROCKSDB_LITE

#include <string>
#include <vector>

#include "utilities/transactions/optimistic_transaction_db_impl.h"

#include "db/db_impl.h"
#include "rocksdb/db.h"
#include "rocksdb/options.h"
#include "rocksdb/utilities/optimistic_transaction_db.h"
#include "utilities/transactions/optimistic_transaction_impl.h"

namespace rocksdb {

Transaction* OptimisticTransactionDBImpl::BeginTransaction(
    const WriteOptions& write_options,
    const OptimisticTransactionOptions& txn_options) {
  Transaction* txn =
      new OptimisticTransactionImpl(this, write_options, txn_options);

  return txn;
}

Status OptimisticTransactionDB::Open(const Options& options,
                                     const std::string& dbname,
                                     OptimisticTransactionDB** dbptr) {
  DBOptions db_options(options);
  ColumnFamilyOptions cf_options(options);
  std::vector<ColumnFamilyDescriptor> column_families;
  column_families.push_back(
      ColumnFamilyDescriptor(kDefaultColumnFamilyName, cf_options));
  std::vector<ColumnFamilyHandle*> handles;
  Status s = Open(db_options, dbname, column_families, &handles, dbptr);
  if (s.ok()) {
    assert(handles.size() == 1);
    // i can delete the handle since DBImpl is always holding a reference to
    // default column family
    delete handles[0];
  }

  return s;
}

Status OptimisticTransactionDB::Open(
    const DBOptions& db_options, const std::string& dbname,
    const std::vector<ColumnFamilyDescriptor>& column_families,
    std::vector<ColumnFamilyHandle*>* handles,
    OptimisticTransactionDB** dbptr) {
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

  s = DB::Open(db_options, dbname, column_families_copy, handles, &db);

  if (s.ok()) {
    *dbptr = new OptimisticTransactionDBImpl(db);
  }

  return s;
}

}  //  namespace rocksdb
#endif  // ROCKSDB_LITE
