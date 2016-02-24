// Copyright (c) 2014, Facebook, Inc.  All rights reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.

package org.rocksdb;

/**
 * Statistics to analyze the performance of a db. Pointer for statistics object
 * is managed by Options class.
 */
public class Statistics {

  private final long statsHandle_;

  public Statistics(final long statsHandle) {
    statsHandle_ = statsHandle;
  }

  public long getTickerCount(TickerType tickerType) {
    assert(isInitialized());
    return getTickerCount0(tickerType.getValue(), statsHandle_);
  }

  public HistogramData geHistogramData(final HistogramType histogramType) {
    assert(isInitialized());
    return geHistogramData0(
        histogramType.getValue(), statsHandle_);
  }

  private boolean isInitialized() {
    return (statsHandle_ != 0);
  }

  private native long getTickerCount0(int tickerType, long handle);
  private native HistogramData geHistogramData0(int histogramType, long handle);
}
