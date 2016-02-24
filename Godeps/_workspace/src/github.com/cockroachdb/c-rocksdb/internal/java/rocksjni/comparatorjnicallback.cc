// Copyright (c) 2014, Facebook, Inc.  All rights reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.
//
// This file implements the callback "bridge" between Java and C++ for
// rocksdb::Comparator.

#include "rocksjni/comparatorjnicallback.h"
#include "rocksjni/portal.h"

namespace rocksdb {
BaseComparatorJniCallback::BaseComparatorJniCallback(
    JNIEnv* env, jobject jComparator,
    const ComparatorJniCallbackOptions* copt)
    : mtx_compare(new port::Mutex(copt->use_adaptive_mutex)),
    mtx_findShortestSeparator(new port::Mutex(copt->use_adaptive_mutex)) {
  // Note: Comparator methods may be accessed by multiple threads,
  // so we ref the jvm not the env
  const jint rs __attribute__((unused)) = env->GetJavaVM(&m_jvm);
  assert(rs == JNI_OK);

  // Note: we want to access the Java Comparator instance
  // across multiple method calls, so we create a global ref
  m_jComparator = env->NewGlobalRef(jComparator);

  // Note: The name of a Comparator will not change during it's lifetime,
  // so we cache it in a global var
  jmethodID jNameMethodId = AbstractComparatorJni::getNameMethodId(env);
  jstring jsName = (jstring)env->CallObjectMethod(m_jComparator, jNameMethodId);
  m_name = JniUtil::copyString(env, jsName);  // also releases jsName

  m_jCompareMethodId = AbstractComparatorJni::getCompareMethodId(env);
  m_jFindShortestSeparatorMethodId =
    AbstractComparatorJni::getFindShortestSeparatorMethodId(env);
  m_jFindShortSuccessorMethodId =
    AbstractComparatorJni::getFindShortSuccessorMethodId(env);
}

/**
 * Attach/Get a JNIEnv for the current native thread
 */
JNIEnv* BaseComparatorJniCallback::getJniEnv() const {
  JNIEnv *env;
  jint rs __attribute__((unused)) =
      m_jvm->AttachCurrentThread(reinterpret_cast<void**>(&env), NULL);
  assert(rs == JNI_OK);
  return env;
}

const char* BaseComparatorJniCallback::Name() const {
  return m_name.c_str();
}

int BaseComparatorJniCallback::Compare(const Slice& a, const Slice& b) const {
  JNIEnv* m_env = getJniEnv();

  // TODO(adamretter): slice objects can potentially be cached using thread
  // local variables to avoid locking. Could make this configurable depending on
  // performance.
  mtx_compare->Lock();

  AbstractSliceJni::setHandle(m_env, m_jSliceA, &a);
  AbstractSliceJni::setHandle(m_env, m_jSliceB, &b);
  jint result =
    m_env->CallIntMethod(m_jComparator, m_jCompareMethodId, m_jSliceA,
      m_jSliceB);

  mtx_compare->Unlock();

  m_jvm->DetachCurrentThread();

  return result;
}

void BaseComparatorJniCallback::FindShortestSeparator(
  std::string* start, const Slice& limit) const {
  if (start == nullptr) {
    return;
  }

  JNIEnv* m_env = getJniEnv();

  const char* startUtf = start->c_str();
  jstring jsStart = m_env->NewStringUTF(startUtf);

  // TODO(adamretter): slice object can potentially be cached using thread local
  // variable to avoid locking. Could make this configurable depending on
  // performance.
  mtx_findShortestSeparator->Lock();

  AbstractSliceJni::setHandle(m_env, m_jSliceLimit, &limit);
  jstring jsResultStart =
    (jstring)m_env->CallObjectMethod(m_jComparator,
      m_jFindShortestSeparatorMethodId, jsStart, m_jSliceLimit);

  mtx_findShortestSeparator->Unlock();

  m_env->DeleteLocalRef(jsStart);

  if (jsResultStart != nullptr) {
    // update start with result
    *start =
      JniUtil::copyString(m_env, jsResultStart);  // also releases jsResultStart
  }

  m_jvm->DetachCurrentThread();
}

void BaseComparatorJniCallback::FindShortSuccessor(std::string* key) const {
  if (key == nullptr) {
    return;
  }

  JNIEnv* m_env = getJniEnv();

  const char* keyUtf = key->c_str();
  jstring jsKey = m_env->NewStringUTF(keyUtf);

  jstring jsResultKey =
    (jstring)m_env->CallObjectMethod(m_jComparator,
      m_jFindShortSuccessorMethodId, jsKey);

  m_env->DeleteLocalRef(jsKey);

  if (jsResultKey != nullptr) {
    // updates key with result, also releases jsResultKey.
    *key = JniUtil::copyString(m_env, jsResultKey);
  }

  m_jvm->DetachCurrentThread();
}

BaseComparatorJniCallback::~BaseComparatorJniCallback() {
  JNIEnv* m_env = getJniEnv();

  m_env->DeleteGlobalRef(m_jComparator);

  // Note: do not need to explicitly detach, as this function is effectively
  // called from the Java class's disposeInternal method, and so already
  // has an attached thread, getJniEnv above is just a no-op Attach to get
  // the env jvm->DetachCurrentThread();
}

ComparatorJniCallback::ComparatorJniCallback(
    JNIEnv* env, jobject jComparator,
    const ComparatorJniCallbackOptions* copt) :
    BaseComparatorJniCallback(env, jComparator, copt) {
  m_jSliceA = env->NewGlobalRef(SliceJni::construct0(env));
  m_jSliceB = env->NewGlobalRef(SliceJni::construct0(env));
  m_jSliceLimit = env->NewGlobalRef(SliceJni::construct0(env));
}

ComparatorJniCallback::~ComparatorJniCallback() {
  JNIEnv* m_env = getJniEnv();
  m_env->DeleteGlobalRef(m_jSliceA);
  m_env->DeleteGlobalRef(m_jSliceB);
  m_env->DeleteGlobalRef(m_jSliceLimit);
}

DirectComparatorJniCallback::DirectComparatorJniCallback(
    JNIEnv* env, jobject jComparator,
    const ComparatorJniCallbackOptions* copt) :
    BaseComparatorJniCallback(env, jComparator, copt) {
  m_jSliceA = env->NewGlobalRef(DirectSliceJni::construct0(env));
  m_jSliceB = env->NewGlobalRef(DirectSliceJni::construct0(env));
  m_jSliceLimit = env->NewGlobalRef(DirectSliceJni::construct0(env));
}

DirectComparatorJniCallback::~DirectComparatorJniCallback() {
  JNIEnv* m_env = getJniEnv();
  m_env->DeleteGlobalRef(m_jSliceA);
  m_env->DeleteGlobalRef(m_jSliceB);
  m_env->DeleteGlobalRef(m_jSliceLimit);
}
}  // namespace rocksdb
