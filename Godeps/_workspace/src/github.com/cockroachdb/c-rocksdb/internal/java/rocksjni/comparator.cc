// Copyright (c) 2014, Facebook, Inc.  All rights reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.
//
// This file implements the "bridge" between Java and C++ for
// rocksdb::Comparator.

#include <stdio.h>
#include <stdlib.h>
#include <jni.h>
#include <string>
#include <functional>

#include "include/org_rocksdb_AbstractComparator.h"
#include "include/org_rocksdb_Comparator.h"
#include "include/org_rocksdb_DirectComparator.h"
#include "rocksjni/comparatorjnicallback.h"
#include "rocksjni/portal.h"

// <editor-fold desc="org.rocksdb.AbstractComparator>

/*
 * Class:     org_rocksdb_AbstractComparator
 * Method:    disposeInternal
 * Signature: (J)V
 */
void Java_org_rocksdb_AbstractComparator_disposeInternal(
    JNIEnv* env, jobject jobj, jlong handle) {
  delete reinterpret_cast<rocksdb::BaseComparatorJniCallback*>(handle);
}
// </editor-fold>

// <editor-fold desc="org.rocksdb.Comparator>

/*
 * Class:     org_rocksdb_Comparator
 * Method:    createNewComparator0
 * Signature: ()V
 */
void Java_org_rocksdb_Comparator_createNewComparator0(
    JNIEnv* env, jobject jobj, jlong copt_handle) {
  const rocksdb::ComparatorJniCallbackOptions* copt =
    reinterpret_cast<rocksdb::ComparatorJniCallbackOptions*>(copt_handle);
  const rocksdb::ComparatorJniCallback* c =
    new rocksdb::ComparatorJniCallback(env, jobj, copt);
  rocksdb::AbstractComparatorJni::setHandle(env, jobj, c);
}
// </editor-fold>

// <editor-fold desc="org.rocksdb.DirectComparator>

/*
 * Class:     org_rocksdb_DirectComparator
 * Method:    createNewDirectComparator0
 * Signature: ()V
 */
void Java_org_rocksdb_DirectComparator_createNewDirectComparator0(
    JNIEnv* env, jobject jobj, jlong copt_handle) {
  const rocksdb::ComparatorJniCallbackOptions* copt =
    reinterpret_cast<rocksdb::ComparatorJniCallbackOptions*>(copt_handle);
  const rocksdb::DirectComparatorJniCallback* c =
    new rocksdb::DirectComparatorJniCallback(env, jobj, copt);
  rocksdb::AbstractComparatorJni::setHandle(env, jobj, c);
}
// </editor-fold>
