//  Copyright (c) 2015, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.

#include "rocksdb/snapshot.h"

#include "rocksdb/db.h"

namespace rocksdb {

ManagedSnapshot::ManagedSnapshot(DB* db) : db_(db),
                                           snapshot_(db->GetSnapshot()) {}

ManagedSnapshot::~ManagedSnapshot() {
  if (snapshot_) {
    db_->ReleaseSnapshot(snapshot_);
  }
}

const Snapshot* ManagedSnapshot::snapshot() { return snapshot_;}

}  // namespace rocksdb
