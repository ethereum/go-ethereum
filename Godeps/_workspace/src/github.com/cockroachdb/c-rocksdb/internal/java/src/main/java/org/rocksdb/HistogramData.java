// Copyright (c) 2014, Facebook, Inc.  All rights reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.

package org.rocksdb;

public class HistogramData {
  private final double median_;
  private final double percentile95_;
  private final double percentile99_;
  private final double average_;
  private final double standardDeviation_;

  public HistogramData(final double median, final double percentile95,
      final double percentile99, final double average,
      final double standardDeviation) {
    median_ = median;
    percentile95_ = percentile95;
    percentile99_ = percentile99;
    average_ = average;
    standardDeviation_ = standardDeviation;
  }

  public double getMedian() {
    return median_;
  }

  public double getPercentile95() {
    return percentile95_;
  }

  public double getPercentile99() {
    return percentile99_;
  }

  public double getAverage() {
    return average_;
  }

  public double getStandardDeviation() {
    return standardDeviation_;
  }
}
