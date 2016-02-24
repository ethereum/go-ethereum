//  Copyright (c) 2014, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.
//
// Copyright (c) 2011 The LevelDB Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file. See the AUTHORS file for names of contributors.

#include "util/rate_limiter.h"
#include "rocksdb/env.h"

namespace rocksdb {


// Pending request
struct GenericRateLimiter::Req {
  explicit Req(int64_t _bytes, port::Mutex* _mu)
      : bytes(_bytes), cv(_mu), granted(false) {}
  int64_t bytes;
  port::CondVar cv;
  bool granted;
};

GenericRateLimiter::GenericRateLimiter(int64_t rate_bytes_per_sec,
                                       int64_t refill_period_us,
                                       int32_t fairness)
    : refill_period_us_(refill_period_us),
      refill_bytes_per_period_(
          CalculateRefillBytesPerPeriod(rate_bytes_per_sec)),
      env_(Env::Default()),
      stop_(false),
      exit_cv_(&request_mutex_),
      requests_to_wait_(0),
      available_bytes_(0),
      next_refill_us_(env_->NowMicros()),
      fairness_(fairness > 100 ? 100 : fairness),
      rnd_((uint32_t)time(nullptr)),
      leader_(nullptr) {
  total_requests_[0] = 0;
  total_requests_[1] = 0;
  total_bytes_through_[0] = 0;
  total_bytes_through_[1] = 0;
}

GenericRateLimiter::~GenericRateLimiter() {
  MutexLock g(&request_mutex_);
  stop_ = true;
  requests_to_wait_ = static_cast<int32_t>(queue_[Env::IO_LOW].size() +
                                           queue_[Env::IO_HIGH].size());
  for (auto& r : queue_[Env::IO_HIGH]) {
    r->cv.Signal();
  }
  for (auto& r : queue_[Env::IO_LOW]) {
    r->cv.Signal();
  }
  while (requests_to_wait_ > 0) {
    exit_cv_.Wait();
  }
}

// This API allows user to dynamically change rate limiter's bytes per second.
void GenericRateLimiter::SetBytesPerSecond(int64_t bytes_per_second) {
  assert(bytes_per_second > 0);
  refill_bytes_per_period_.store(
      CalculateRefillBytesPerPeriod(bytes_per_second),
      std::memory_order_relaxed);
}

void GenericRateLimiter::Request(int64_t bytes, const Env::IOPriority pri) {
  assert(bytes <= refill_bytes_per_period_.load(std::memory_order_relaxed));

  MutexLock g(&request_mutex_);
  if (stop_) {
    return;
  }

  ++total_requests_[pri];

  if (available_bytes_ >= bytes) {
    // Refill thread assigns quota and notifies requests waiting on
    // the queue under mutex. So if we get here, that means nobody
    // is waiting?
    available_bytes_ -= bytes;
    total_bytes_through_[pri] += bytes;
    return;
  }

  // Request cannot be satisfied at this moment, enqueue
  Req r(bytes, &request_mutex_);
  queue_[pri].push_back(&r);

  do {
    bool timedout = false;
    // Leader election, candidates can be:
    // (1) a new incoming request,
    // (2) a previous leader, whose quota has not been not assigned yet due
    //     to lower priority
    // (3) a previous waiter at the front of queue, who got notified by
    //     previous leader
    if (leader_ == nullptr &&
        ((!queue_[Env::IO_HIGH].empty() &&
            &r == queue_[Env::IO_HIGH].front()) ||
         (!queue_[Env::IO_LOW].empty() &&
            &r == queue_[Env::IO_LOW].front()))) {
      leader_ = &r;
      timedout = r.cv.TimedWait(next_refill_us_);
    } else {
      // Not at the front of queue or an leader has already been elected
      r.cv.Wait();
    }

    // request_mutex_ is held from now on
    if (stop_) {
      --requests_to_wait_;
      exit_cv_.Signal();
      return;
    }

    // Make sure the waken up request is always the header of its queue
    assert(r.granted ||
           (!queue_[Env::IO_HIGH].empty() &&
            &r == queue_[Env::IO_HIGH].front()) ||
           (!queue_[Env::IO_LOW].empty() &&
            &r == queue_[Env::IO_LOW].front()));
    assert(leader_ == nullptr ||
           (!queue_[Env::IO_HIGH].empty() &&
            leader_ == queue_[Env::IO_HIGH].front()) ||
           (!queue_[Env::IO_LOW].empty() &&
            leader_ == queue_[Env::IO_LOW].front()));

    if (leader_ == &r) {
      // Waken up from TimedWait()
      if (timedout) {
        // Time to do refill!
        Refill();

        // Re-elect a new leader regardless. This is to simplify the
        // election handling.
        leader_ = nullptr;

        // Notify the header of queue if current leader is going away
        if (r.granted) {
          // Current leader already got granted with quota. Notify header
          // of waiting queue to participate next round of election.
          assert((queue_[Env::IO_HIGH].empty() ||
                    &r != queue_[Env::IO_HIGH].front()) &&
                 (queue_[Env::IO_LOW].empty() ||
                    &r != queue_[Env::IO_LOW].front()));
          if (!queue_[Env::IO_HIGH].empty()) {
            queue_[Env::IO_HIGH].front()->cv.Signal();
          } else if (!queue_[Env::IO_LOW].empty()) {
            queue_[Env::IO_LOW].front()->cv.Signal();
          }
          // Done
          break;
        }
      } else {
        // Spontaneous wake up, need to continue to wait
        assert(!r.granted);
        leader_ = nullptr;
      }
    } else {
      // Waken up by previous leader:
      // (1) if requested quota is granted, it is done.
      // (2) if requested quota is not granted, this means current thread
      // was picked as a new leader candidate (previous leader got quota).
      // It needs to participate leader election because a new request may
      // come in before this thread gets waken up. So it may actually need
      // to do Wait() again.
      assert(!timedout);
    }
  } while (!r.granted);
}

void GenericRateLimiter::Refill() {
  next_refill_us_ = env_->NowMicros() + refill_period_us_;
  // Carry over the left over quota from the last period
  auto refill_bytes_per_period =
      refill_bytes_per_period_.load(std::memory_order_relaxed);
  if (available_bytes_ < refill_bytes_per_period) {
    available_bytes_ += refill_bytes_per_period;
  }

  int use_low_pri_first = rnd_.OneIn(fairness_) ? 0 : 1;
  for (int q = 0; q < 2; ++q) {
    auto use_pri = (use_low_pri_first == q) ? Env::IO_LOW : Env::IO_HIGH;
    auto* queue = &queue_[use_pri];
    while (!queue->empty()) {
      auto* next_req = queue->front();
      if (available_bytes_ < next_req->bytes) {
        break;
      }
      available_bytes_ -= next_req->bytes;
      total_bytes_through_[use_pri] += next_req->bytes;
      queue->pop_front();

      next_req->granted = true;
      if (next_req != leader_) {
        // Quota granted, signal the thread
        next_req->cv.Signal();
      }
    }
  }
}

RateLimiter* NewGenericRateLimiter(
    int64_t rate_bytes_per_sec, int64_t refill_period_us, int32_t fairness) {
  assert(rate_bytes_per_sec > 0);
  assert(refill_period_us > 0);
  assert(fairness > 0);
  return new GenericRateLimiter(
      rate_bytes_per_sec, refill_period_us, fairness);
}

}  // namespace rocksdb
