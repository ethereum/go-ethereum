//  Copyright (c) 2014, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.
//
// Copyright (c) 2011 The LevelDB Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file. See the AUTHORS file for names of contributors.

#ifndef __STDC_FORMAT_MACROS
#define __STDC_FORMAT_MACROS
#endif

#include <inttypes.h>
#include <limits>
#include "util/testharness.h"
#include "util/rate_limiter.h"
#include "util/random.h"
#include "rocksdb/env.h"

namespace rocksdb {

class RateLimiterTest : public testing::Test {};

TEST_F(RateLimiterTest, StartStop) {
  std::unique_ptr<RateLimiter> limiter(new GenericRateLimiter(100, 100, 10));
}

TEST_F(RateLimiterTest, Rate) {
  auto* env = Env::Default();
  struct Arg {
    Arg(int32_t _target_rate, int _burst)
        : limiter(new GenericRateLimiter(_target_rate, 100 * 1000, 10)),
          request_size(_target_rate / 10),
          burst(_burst) {}
    std::unique_ptr<RateLimiter> limiter;
    int32_t request_size;
    int burst;
  };

  auto writer = [](void* p) {
    auto* thread_env = Env::Default();
    auto* arg = static_cast<Arg*>(p);
    // Test for 2 seconds
    auto until = thread_env->NowMicros() + 2 * 1000000;
    Random r((uint32_t)(thread_env->NowNanos() %
                        std::numeric_limits<uint32_t>::max()));
    while (thread_env->NowMicros() < until) {
      for (int i = 0; i < static_cast<int>(r.Skewed(arg->burst) + 1); ++i) {
        arg->limiter->Request(r.Uniform(arg->request_size - 1) + 1,
                              Env::IO_HIGH);
      }
      arg->limiter->Request(r.Uniform(arg->request_size - 1) + 1, Env::IO_LOW);
    }
  };

  for (int i = 1; i <= 16; i *= 2) {
    int32_t target = i * 1024 * 10;
    Arg arg(target, i / 4 + 1);
    int64_t old_total_bytes_through = 0;
    for (int iter = 1; iter <= 2; ++iter) {
      // second iteration changes the target dynamically
      if (iter == 2) {
        target *= 2;
        arg.limiter->SetBytesPerSecond(target);
      }
      auto start = env->NowMicros();
      for (int t = 0; t < i; ++t) {
        env->StartThread(writer, &arg);
      }
      env->WaitForJoin();

      auto elapsed = env->NowMicros() - start;
      double rate =
          (arg.limiter->GetTotalBytesThrough() - old_total_bytes_through) *
          1000000.0 / elapsed;
      old_total_bytes_through = arg.limiter->GetTotalBytesThrough();
      fprintf(stderr,
              "request size [1 - %" PRIi32 "], limit %" PRIi32
              " KB/sec, actual rate: %lf KB/sec, elapsed %.2lf seconds\n",
              arg.request_size - 1, target / 1024, rate / 1024,
              elapsed / 1000000.0);

      ASSERT_GE(rate / target, 0.9);
      ASSERT_LE(rate / target, 1.1);
    }
  }
}

}  // namespace rocksdb

int main(int argc, char** argv) {
  ::testing::InitGoogleTest(&argc, argv);
  return RUN_ALL_TESTS();
}
