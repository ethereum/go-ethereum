// Copyright (c) 2014, Facebook, Inc.  All rights reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.
//
// This file implements the callback "bridge" between Java and C++ for
// rocksdb::Comparator.

#include "rocksjni/writebatchhandlerjnicallback.h"
#include "rocksjni/portal.h"

namespace rocksdb {
WriteBatchHandlerJniCallback::WriteBatchHandlerJniCallback(
    JNIEnv* env, jobject jWriteBatchHandler)
    : m_env(env) {

  // Note: we want to access the Java WriteBatchHandler instance
  // across multiple method calls, so we create a global ref
  m_jWriteBatchHandler = env->NewGlobalRef(jWriteBatchHandler);

  m_jPutMethodId = WriteBatchHandlerJni::getPutMethodId(env);
  m_jMergeMethodId = WriteBatchHandlerJni::getMergeMethodId(env);
  m_jDeleteMethodId = WriteBatchHandlerJni::getDeleteMethodId(env);
  m_jLogDataMethodId = WriteBatchHandlerJni::getLogDataMethodId(env);
  m_jContinueMethodId = WriteBatchHandlerJni::getContinueMethodId(env);
}

void WriteBatchHandlerJniCallback::Put(const Slice& key, const Slice& value) {
  const jbyteArray j_key = sliceToJArray(key);
  const jbyteArray j_value = sliceToJArray(value);

  m_env->CallVoidMethod(
      m_jWriteBatchHandler,
      m_jPutMethodId,
      j_key,
      j_value);

  m_env->DeleteLocalRef(j_value);
  m_env->DeleteLocalRef(j_key);
}

void WriteBatchHandlerJniCallback::Merge(const Slice& key, const Slice& value) {
  const jbyteArray j_key = sliceToJArray(key);
  const jbyteArray j_value = sliceToJArray(value);

  m_env->CallVoidMethod(
      m_jWriteBatchHandler,
      m_jMergeMethodId,
      j_key,
      j_value);

  m_env->DeleteLocalRef(j_value);
  m_env->DeleteLocalRef(j_key);
}

void WriteBatchHandlerJniCallback::Delete(const Slice& key) {
  const jbyteArray j_key = sliceToJArray(key);

  m_env->CallVoidMethod(
      m_jWriteBatchHandler,
      m_jDeleteMethodId,
      j_key);

  m_env->DeleteLocalRef(j_key);
}

void WriteBatchHandlerJniCallback::LogData(const Slice& blob) {
  const jbyteArray j_blob = sliceToJArray(blob);

  m_env->CallVoidMethod(
      m_jWriteBatchHandler,
      m_jLogDataMethodId,
      j_blob);

  m_env->DeleteLocalRef(j_blob);
}

bool WriteBatchHandlerJniCallback::Continue() {
  jboolean jContinue = m_env->CallBooleanMethod(
      m_jWriteBatchHandler,
      m_jContinueMethodId);

  return static_cast<bool>(jContinue == JNI_TRUE);
}

/*
 * Creates a Java Byte Array from the data in a Slice
 *
 * When calling this function
 * you must remember to call env->DeleteLocalRef
 * on the result after you have finished with it
 */
jbyteArray WriteBatchHandlerJniCallback::sliceToJArray(const Slice& s) {
  jbyteArray ja = m_env->NewByteArray(static_cast<jsize>(s.size()));
  m_env->SetByteArrayRegion(
      ja, 0, static_cast<jsize>(s.size()),
      reinterpret_cast<const jbyte*>(s.data()));
  return ja;
}

WriteBatchHandlerJniCallback::~WriteBatchHandlerJniCallback() {
  m_env->DeleteGlobalRef(m_jWriteBatchHandler);
}
}  // namespace rocksdb
