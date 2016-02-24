// Copyright (c) 2014, Facebook, Inc.  All rights reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.
package org.rocksdb;

/**
 * Config for rate limiter, which is used to control write rate of flush and
 * compaction.
 */
public abstract class RateLimiterConfig {
  /**
   * This function should only be called by
   * {@link org.rocksdb.DBOptions#setRateLimiter(long, long)}, which will
   * create a c++ shared-pointer to the c++ {@code RateLimiter} that is associated
   * with a Java RateLimiterConfig.
   *
   * @see org.rocksdb.DBOptions#setRateLimiter(long, long)
   *
   * @return native handle address to rate limiter instance.
   */
  abstract protected long newRateLimiterHandle();
}
