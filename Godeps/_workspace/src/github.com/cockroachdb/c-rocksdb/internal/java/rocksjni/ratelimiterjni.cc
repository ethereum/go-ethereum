// Copyright (c) 2014, Facebook, Inc.  All rights reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.
//
// This file implements the "bridge" between Java and C++ for RateLimiter.

#include "rocksjni/portal.h"
#include "include/org_rocksdb_GenericRateLimiterConfig.h"
#include "rocksdb/rate_limiter.h"

/*
 * Class:     org_rocksdb_GenericRateLimiterConfig
 * Method:    newRateLimiterHandle
 * Signature: (JJI)J
 */
jlong Java_org_rocksdb_GenericRateLimiterConfig_newRateLimiterHandle(
    JNIEnv* env, jobject jobj, jlong jrate_bytes_per_second,
    jlong jrefill_period_micros, jint jfairness) {
  return reinterpret_cast<jlong>(rocksdb::NewGenericRateLimiter(
      static_cast<int64_t>(jrate_bytes_per_second),
      static_cast<int64_t>(jrefill_period_micros),
      static_cast<int32_t>(jfairness)));
}
