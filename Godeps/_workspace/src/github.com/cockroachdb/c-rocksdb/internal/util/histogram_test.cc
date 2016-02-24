//  Copyright (c) 2013, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.
//
#include "util/histogram.h"

#include "util/testharness.h"

namespace rocksdb {

class HistogramTest : public testing::Test {};

TEST_F(HistogramTest, BasicOperation) {
  HistogramImpl histogram;
  for (uint64_t i = 1; i <= 100; i++) {
    histogram.Add(i);
  }

  {
    double median = histogram.Median();
    // ASSERT_LE(median, 50);
    ASSERT_GT(median, 0);
  }

  {
    double percentile100 = histogram.Percentile(100.0);
    ASSERT_LE(percentile100, 100.0);
    ASSERT_GT(percentile100, 0.0);
    double percentile99 = histogram.Percentile(99.0);
    double percentile85 = histogram.Percentile(85.0);
    ASSERT_LE(percentile99, 99.0);
    ASSERT_TRUE(percentile99 >= percentile85);
  }

  ASSERT_EQ(histogram.Average(), 50.5); // avg is acurately calculated.
}

TEST_F(HistogramTest, EmptyHistogram) {
  HistogramImpl histogram;
  ASSERT_EQ(histogram.Median(), 0.0);
  ASSERT_EQ(histogram.Percentile(85.0), 0.0);
  ASSERT_EQ(histogram.Average(), 0.0);
}

TEST_F(HistogramTest, ClearHistogram) {
  HistogramImpl histogram;
  for (uint64_t i = 1; i <= 100; i++) {
    histogram.Add(i);
  }
  histogram.Clear();
  ASSERT_EQ(histogram.Median(), 0);
  ASSERT_EQ(histogram.Percentile(85.0), 0);
  ASSERT_EQ(histogram.Average(), 0);
}

}  // namespace rocksdb

int main(int argc, char** argv) {
  ::testing::InitGoogleTest(&argc, argv);
  return RUN_ALL_TESTS();
}
