//  Copyright (c) 2013, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.

#pragma once
#ifndef ROCKSDB_LITE

#include <string>
#include <vector>

#include "rocksdb/utilities/stackable_db.h"
#include "rocksdb/db.h"

namespace rocksdb {

// Database with TTL support.
//
// USE-CASES:
// This API should be used to open the db when key-values inserted are
//  meant to be removed from the db in a non-strict 'ttl' amount of time
//  Therefore, this guarantees that key-values inserted will remain in the
//  db for >= ttl amount of time and the db will make efforts to remove the
//  key-values as soon as possible after ttl seconds of their insertion.
//
// BEHAVIOUR:
// TTL is accepted in seconds
// (int32_t)Timestamp(creation) is suffixed to values in Put internally
// Expired TTL values deleted in compaction only:(Timestamp+ttl<time_now)
// Get/Iterator may return expired entries(compaction not run on them yet)
// Different TTL may be used during different Opens
// Example: Open1 at t=0 with ttl=4 and insert k1,k2, close at t=2
//          Open2 at t=3 with ttl=5. Now k1,k2 should be deleted at t>=5
// read_only=true opens in the usual read-only mode. Compactions will not be
//  triggered(neither manual nor automatic), so no expired entries removed
//
// CONSTRAINTS:
// Not specifying/passing or non-positive TTL behaves like TTL = infinity
//
// !!!WARNING!!!:
// Calling DB::Open directly to re-open a db created by this API will get
//  corrupt values(timestamp suffixed) and no ttl effect will be there
//  during the second Open, so use this API consistently to open the db
// Be careful when passing ttl with a small positive value because the
//  whole database may be deleted in a small amount of time

class DBWithTTL : public StackableDB {
 public:
  virtual Status CreateColumnFamilyWithTtl(
      const ColumnFamilyOptions& options, const std::string& column_family_name,
      ColumnFamilyHandle** handle, int ttl) = 0;

  static Status Open(const Options& options, const std::string& dbname,
                     DBWithTTL** dbptr, int32_t ttl = 0,
                     bool read_only = false);

  static Status Open(const DBOptions& db_options, const std::string& dbname,
                     const std::vector<ColumnFamilyDescriptor>& column_families,
                     std::vector<ColumnFamilyHandle*>* handles,
                     DBWithTTL** dbptr, std::vector<int32_t> ttls,
                     bool read_only = false);

 protected:
  explicit DBWithTTL(DB* db) : StackableDB(db) {}
};

}  // namespace rocksdb
#endif  // ROCKSDB_LITE
