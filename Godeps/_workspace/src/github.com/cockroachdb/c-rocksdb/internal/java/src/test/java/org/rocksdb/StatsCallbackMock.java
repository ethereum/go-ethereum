// Copyright (c) 2014, Facebook, Inc.  All rights reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.

package org.rocksdb;

public class StatsCallbackMock implements StatisticsCollectorCallback {
  public int tickerCallbackCount = 0;
  public int histCallbackCount = 0;

  public void tickerCallback(TickerType tickerType, long tickerCount) {
    tickerCallbackCount++;
  }

  public void histogramCallback(HistogramType histType,
      HistogramData histData) {
    histCallbackCount++;
  }
}
