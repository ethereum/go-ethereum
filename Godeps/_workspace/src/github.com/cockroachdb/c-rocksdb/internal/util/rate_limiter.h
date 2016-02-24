//  Copyright (c) 2014, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.
//
// Copyright (c) 2011 The LevelDB Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file. See the AUTHORS file for names of contributors.

#pragma once

#include <atomic>
#include <deque>
#include "port/port.h"
#include "util/mutexlock.h"
#include "util/random.h"
#include "rocksdb/env.h"
#include "rocksdb/rate_limiter.h"

namespace rocksdb {

class GenericRateLimiter : public RateLimiter {
 public:
  GenericRateLimiter(int64_t refill_bytes,
      int64_t refill_period_us, int32_t fairness);

  virtual ~GenericRateLimiter();

  // This API allows user to dynamically change rate limiter's bytes per second.
  virtual void SetBytesPerSecond(int64_t bytes_per_second) override;

  // Request for token to write bytes. If this request can not be satisfied,
  // the call is blocked. Caller is responsible to make sure
  // bytes <= GetSingleBurstBytes()
  virtual void Request(const int64_t bytes, const Env::IOPriority pri) override;

  virtual int64_t GetSingleBurstBytes() const override {
    return refill_bytes_per_period_.load(std::memory_order_relaxed);
  }

  virtual int64_t GetTotalBytesThrough(
      const Env::IOPriority pri = Env::IO_TOTAL) const override {
    MutexLock g(&request_mutex_);
    if (pri == Env::IO_TOTAL) {
      return total_bytes_through_[Env::IO_LOW] +
             total_bytes_through_[Env::IO_HIGH];
    }
    return total_bytes_through_[pri];
  }

  virtual int64_t GetTotalRequests(
      const Env::IOPriority pri = Env::IO_TOTAL) const override {
    MutexLock g(&request_mutex_);
    if (pri == Env::IO_TOTAL) {
      return total_requests_[Env::IO_LOW] + total_requests_[Env::IO_HIGH];
    }
    return total_requests_[pri];
  }

 private:
  void Refill();
  int64_t CalculateRefillBytesPerPeriod(int64_t rate_bytes_per_sec) {
    return rate_bytes_per_sec * refill_period_us_ / 1000000.0;
  }

  // This mutex guard all internal states
  mutable port::Mutex request_mutex_;

  const int64_t refill_period_us_;
  // This variable can be changed dynamically.
  std::atomic<int64_t> refill_bytes_per_period_;
  Env* const env_;

  bool stop_;
  port::CondVar exit_cv_;
  int32_t requests_to_wait_;

  int64_t total_requests_[Env::IO_TOTAL];
  int64_t total_bytes_through_[Env::IO_TOTAL];
  int64_t available_bytes_;
  int64_t next_refill_us_;

  int32_t fairness_;
  Random rnd_;

  struct Req;
  Req* leader_;
  std::deque<Req*> queue_[Env::IO_TOTAL];
};

}  // namespace rocksdb
