// Copyright (c) 2014, Facebook, Inc.  All rights reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.

package org.rocksdb;

/**
 * Callback interface provided to StatisticsCollector.
 *
 * Thread safety:
 * StatisticsCollector doesn't make any guarantees about thread safety.
 * If the same reference of StatisticsCollectorCallback is passed to multiple
 * StatisticsCollector references, then its the responsibility of the
 * user to make StatisticsCollectorCallback's implementation thread-safe.
 *
 */
public interface StatisticsCollectorCallback {
  /**
   * Callback function to get ticker values.
   * @param tickerType Ticker type.
   * @param tickerCount Value of ticker type.
  */
  void tickerCallback(TickerType tickerType, long tickerCount);

  /**
   * Callback function to get histogram values.
   * @param histType Histogram type.
   * @param histData Histogram data.
  */
  void histogramCallback(HistogramType histType, HistogramData histData);
}
