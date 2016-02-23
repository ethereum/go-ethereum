//  Copyright (c) 2013, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.
//
#pragma once
#include "rocksdb/env.h"
#include "util/perf_level_imp.h"
#include "util/stop_watch.h"

namespace rocksdb {

class PerfStepTimer {
 public:
  PerfStepTimer(uint64_t* metric)
    : enabled_(perf_level >= PerfLevel::kEnableTime),
      env_(enabled_ ? Env::Default() : nullptr),
      start_(0),
      metric_(metric) {
  }

  ~PerfStepTimer() {
    Stop();
  }

  void Start() {
    if (enabled_) {
      start_ = env_->NowNanos();
    }
  }

  void Measure() {
    if (start_) {
      uint64_t now = env_->NowNanos();
      *metric_ += now - start_;
      start_ = now;
    }
  }

  void Stop() {
    if (start_) {
      *metric_ += env_->NowNanos() - start_;
      start_ = 0;
    }
  }

 private:
  const bool enabled_;
  Env* const env_;
  uint64_t start_;
  uint64_t* metric_;
};

}  // namespace rocksdb
