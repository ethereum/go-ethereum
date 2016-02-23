// Copyright (c) 2014, Facebook, Inc.  All rights reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.
package org.rocksdb;

/**
 * Config for rate limiter, which is used to control write rate of flush and
 * compaction.
 *
 * @see RateLimiterConfig
 */
public class GenericRateLimiterConfig extends RateLimiterConfig {
  private static final long DEFAULT_REFILL_PERIOD_MICROS = (100 * 1000);
  private static final int DEFAULT_FAIRNESS = 10;

  /**
   * GenericRateLimiterConfig constructor
   *
   * @param rateBytesPerSecond this is the only parameter you want to set
   *     most of the time. It controls the total write rate of compaction
   *     and flush in bytes per second. Currently, RocksDB does not enforce
   *     rate limit for anything other than flush and compaction, e.g. write to WAL.
   * @param refillPeriodMicros this controls how often tokens are refilled. For example,
   *     when rate_bytes_per_sec is set to 10MB/s and refill_period_us is set to
   *     100ms, then 1MB is refilled every 100ms internally. Larger value can lead to
   *     burstier writes while smaller value introduces more CPU overhead.
   *     The default should work for most cases.
   * @param fairness RateLimiter accepts high-pri requests and low-pri requests.
   *     A low-pri request is usually blocked in favor of hi-pri request. Currently,
   *     RocksDB assigns low-pri to request from compaction and high-pri to request
   *     from flush. Low-pri requests can get blocked if flush requests come in
   *     continuously. This fairness parameter grants low-pri requests permission by
   *     fairness chance even though high-pri requests exist to avoid starvation.
   *     You should be good by leaving it at default 10.
   */
  public GenericRateLimiterConfig(final long rateBytesPerSecond,
      final long refillPeriodMicros, final int fairness) {
    rateBytesPerSecond_ = rateBytesPerSecond;
    refillPeriodMicros_ = refillPeriodMicros;
    fairness_ = fairness;
  }

  /**
   * GenericRateLimiterConfig constructor
   *
   * @param rateBytesPerSecond this is the only parameter you want to set
   *     most of the time. It controls the total write rate of compaction
   *     and flush in bytes per second. Currently, RocksDB does not enforce
   *     rate limit for anything other than flush and compaction, e.g. write to WAL.
   */
  public GenericRateLimiterConfig(final long rateBytesPerSecond) {
    this(rateBytesPerSecond, DEFAULT_REFILL_PERIOD_MICROS, DEFAULT_FAIRNESS);
  }

  @Override protected long newRateLimiterHandle() {
    return newRateLimiterHandle(rateBytesPerSecond_, refillPeriodMicros_,
        fairness_);
  }

  private native long newRateLimiterHandle(long rateBytesPerSecond,
      long refillPeriodMicros, int fairness);
  private final long rateBytesPerSecond_;
  private final long refillPeriodMicros_;
  private final int fairness_;
}
