// Copyright (c) 2014, Facebook, Inc.  All rights reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.

package org.rocksdb;

/**
 * Contains all information necessary to collect statistics from one instance
 * of DB statistics.
 */
public class StatsCollectorInput {
  private final Statistics _statistics;
  private final StatisticsCollectorCallback _statsCallback;

  /**
   * Constructor for StatsCollectorInput.
   *
   * @param statistics Reference of DB statistics.
   * @param statsCallback Reference of statistics callback interface.
   */
  public StatsCollectorInput(final Statistics statistics,
      final StatisticsCollectorCallback statsCallback) {
    _statistics = statistics;
    _statsCallback = statsCallback;
  }

  public Statistics getStatistics() {
    return _statistics;
  }

  public StatisticsCollectorCallback getCallback() {
    return _statsCallback;
  }
}
