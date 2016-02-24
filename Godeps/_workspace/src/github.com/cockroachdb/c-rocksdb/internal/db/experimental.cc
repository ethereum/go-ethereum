//  Copyright (c) 2014, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.

#include "rocksdb/experimental.h"

#include "db/db_impl.h"

namespace rocksdb {
namespace experimental {

#ifndef ROCKSDB_LITE

Status SuggestCompactRange(DB* db, ColumnFamilyHandle* column_family,
                           const Slice* begin, const Slice* end) {
  auto dbimpl = dynamic_cast<DBImpl*>(db);
  if (dbimpl == nullptr) {
    return Status::InvalidArgument("Didn't recognize DB object");
  }

  return dbimpl->SuggestCompactRange(column_family, begin, end);
}

Status PromoteL0(DB* db, ColumnFamilyHandle* column_family, int target_level) {
  auto dbimpl = dynamic_cast<DBImpl*>(db);
  if (dbimpl == nullptr) {
    return Status::InvalidArgument("Didn't recognize DB object");
  }
  return dbimpl->PromoteL0(column_family, target_level);
}

#else  // ROCKSDB_LITE

Status SuggestCompactRange(DB* db, ColumnFamilyHandle* column_family,
                           const Slice* begin, const Slice* end) {
  return Status::NotSupported("Not supported in RocksDB LITE");
}

Status PromoteL0(DB* db, ColumnFamilyHandle* column_family, int target_level) {
  return Status::NotSupported("Not supported in RocksDB LITE");
}

#endif  // ROCKSDB_LITE

Status SuggestCompactRange(DB* db, const Slice* begin, const Slice* end) {
  return SuggestCompactRange(db, db->DefaultColumnFamily(), begin, end);
}

}  // namespace experimental
}  // namespace rocksdb
