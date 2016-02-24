//  Copyright (c) 2013, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.
//
// Copyright (c) 2011 The LevelDB Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file. See the AUTHORS file for names of contributors.

#pragma once
#include "rocksdb/statistics.h"

#include <cassert>
#include <string>
#include <vector>
#include <map>

#include <string.h>

namespace rocksdb {

class HistogramBucketMapper {
 public:

  HistogramBucketMapper();

  // converts a value to the bucket index.
  size_t IndexForValue(const uint64_t value) const;
  // number of buckets required.

  size_t BucketCount() const {
    return bucketValues_.size();
  }

  uint64_t LastValue() const {
    return maxBucketValue_;
  }

  uint64_t FirstValue() const {
    return minBucketValue_;
  }

  uint64_t BucketLimit(const size_t bucketNumber) const {
    assert(bucketNumber < BucketCount());
    return bucketValues_[bucketNumber];
  }

 private:
  const std::vector<uint64_t> bucketValues_;
  const uint64_t maxBucketValue_;
  const uint64_t minBucketValue_;
  std::map<uint64_t, uint64_t> valueIndexMap_;
};

class HistogramImpl {
 public:
  HistogramImpl() { memset(buckets_, 0, sizeof(buckets_)); }
  virtual void Clear();
  virtual bool Empty();
  virtual void Add(uint64_t value);
  void Merge(const HistogramImpl& other);

  virtual std::string ToString() const;

  virtual double Median() const;
  virtual double Percentile(double p) const;
  virtual double Average() const;
  virtual double StandardDeviation() const;
  virtual void Data(HistogramData * const data) const;

  virtual ~HistogramImpl() {}

 private:
  // To be able to use HistogramImpl as thread local variable, its constructor
  // has to be static. That's why we're using manually values from BucketMapper
  double min_ = 1000000000;  // this is BucketMapper:LastValue()
  double max_ = 0;
  double num_ = 0;
  double sum_ = 0;
  double sum_squares_ = 0;
  uint64_t buckets_[138];  // this is BucketMapper::BucketCount()
};

}  // namespace rocksdb
