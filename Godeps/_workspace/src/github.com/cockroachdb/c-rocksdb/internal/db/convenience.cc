//  Copyright (c) 2013, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.
//
// Copyright (c) 2012 Facebook.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

#ifndef ROCKSDB_LITE

#include "rocksdb/convenience.h"

#include "db/db_impl.h"

namespace rocksdb {

void CancelAllBackgroundWork(DB* db, bool wait) {
  (dynamic_cast<DBImpl*>(db))->CancelAllBackgroundWork(wait);
}
}  // namespace rocksdb

#endif  // ROCKSDB_LITE
