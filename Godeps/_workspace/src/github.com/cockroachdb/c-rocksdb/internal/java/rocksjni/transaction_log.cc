// Copyright (c) 2014, Facebook, Inc.  All rights reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.
//
// This file implements the "bridge" between Java and C++ and enables
// calling c++ rocksdb::Iterator methods from Java side.

#include <jni.h>
#include <stdio.h>
#include <stdlib.h>

#include "include/org_rocksdb_TransactionLogIterator.h"
#include "rocksdb/transaction_log.h"
#include "rocksjni/portal.h"

/*
 * Class:     org_rocksdb_TransactionLogIterator
 * Method:    disposeInternal
 * Signature: (J)V
 */
void Java_org_rocksdb_TransactionLogIterator_disposeInternal(
    JNIEnv* env, jobject jobj, jlong handle) {
  delete reinterpret_cast<rocksdb::TransactionLogIterator*>(handle);
}

/*
 * Class:     org_rocksdb_TransactionLogIterator
 * Method:    isValid
 * Signature: (J)Z
 */
jboolean Java_org_rocksdb_TransactionLogIterator_isValid(
    JNIEnv* env, jobject jobj, jlong handle) {
  return reinterpret_cast<rocksdb::TransactionLogIterator*>(handle)->Valid();
}

/*
 * Class:     org_rocksdb_TransactionLogIterator
 * Method:    next
 * Signature: (J)V
 */
void Java_org_rocksdb_TransactionLogIterator_next(
    JNIEnv* env, jobject jobj, jlong handle) {
  reinterpret_cast<rocksdb::TransactionLogIterator*>(handle)->Next();
}

/*
 * Class:     org_rocksdb_TransactionLogIterator
 * Method:    status
 * Signature: (J)V
 */
void Java_org_rocksdb_TransactionLogIterator_status(
    JNIEnv* env, jobject jobj, jlong handle) {
  rocksdb::Status s = reinterpret_cast<
      rocksdb::TransactionLogIterator*>(handle)->status();
  if (!s.ok()) {
    rocksdb::RocksDBExceptionJni::ThrowNew(env, s);
  }
}

/*
 * Class:     org_rocksdb_TransactionLogIterator
 * Method:    getBatch
 * Signature: (J)Lorg/rocksdb/TransactionLogIterator$BatchResult
 */
jobject Java_org_rocksdb_TransactionLogIterator_getBatch(
    JNIEnv* env, jobject jobj, jlong handle) {
  rocksdb::BatchResult batch_result =
      reinterpret_cast<rocksdb::TransactionLogIterator*>(handle)->GetBatch();
  jclass jclazz = env->FindClass(
      "org/rocksdb/TransactionLogIterator$BatchResult");
  assert(jclazz != nullptr);
  jmethodID mid = env->GetMethodID(
      jclazz, "<init>", "(Lorg/rocksdb/TransactionLogIterator;JJ)V");
  assert(mid != nullptr);
  return env->NewObject(jclazz, mid, jobj,
      batch_result.sequence, batch_result.writeBatchPtr.release());
}
