// Copyright (c) 2014, Facebook, Inc.  All rights reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.
//
// This file implements the "bridge" between Java and C++.

#include <jni.h>
#include <stdio.h>
#include <stdlib.h>

#include "include/org_rocksdb_Snapshot.h"
#include "rocksdb/db.h"
#include "rocksjni/portal.h"

/*
 * Class:     org_rocksdb_Snapshot
 * Method:    getSequenceNumber
 * Signature: (J)J
 */
jlong Java_org_rocksdb_Snapshot_getSequenceNumber(JNIEnv* env,
    jobject jobj, jlong jsnapshot_handle) {
  auto* snapshot = reinterpret_cast<rocksdb::Snapshot*>(
      jsnapshot_handle);
  return snapshot->GetSequenceNumber();
}
