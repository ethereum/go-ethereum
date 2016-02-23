//  Copyright (c) 2013, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.

#ifndef GFLAGS
#include <cstdio>
int main() {
  fprintf(stderr, "Please install gflags to run rocksdb tools\n");
  return 1;
}
#else

#ifndef __STDC_FORMAT_MACROS
#define __STDC_FORMAT_MACROS
#endif

#include <inttypes.h>
#include <algorithm>
#include <gflags/gflags.h>

#include "dynamic_bloom.h"
#include "port/port.h"
#include "util/arena.h"
#include "util/logging.h"
#include "util/testharness.h"
#include "util/testutil.h"
#include "util/stop_watch.h"

using GFLAGS::ParseCommandLineFlags;

DEFINE_int32(bits_per_key, 10, "");
DEFINE_int32(num_probes, 6, "");
DEFINE_bool(enable_perf, false, "");

namespace rocksdb {

static Slice Key(uint64_t i, char* buffer) {
  memcpy(buffer, &i, sizeof(i));
  return Slice(buffer, sizeof(i));
}

class DynamicBloomTest : public testing::Test {};

TEST_F(DynamicBloomTest, EmptyFilter) {
  Arena arena;
  DynamicBloom bloom1(&arena, 100, 0, 2);
  ASSERT_TRUE(!bloom1.MayContain("hello"));
  ASSERT_TRUE(!bloom1.MayContain("world"));

  DynamicBloom bloom2(&arena, CACHE_LINE_SIZE * 8 * 2 - 1, 1, 2);
  ASSERT_TRUE(!bloom2.MayContain("hello"));
  ASSERT_TRUE(!bloom2.MayContain("world"));
}

TEST_F(DynamicBloomTest, Small) {
  Arena arena;
  DynamicBloom bloom1(&arena, 100, 0, 2);
  bloom1.Add("hello");
  bloom1.Add("world");
  ASSERT_TRUE(bloom1.MayContain("hello"));
  ASSERT_TRUE(bloom1.MayContain("world"));
  ASSERT_TRUE(!bloom1.MayContain("x"));
  ASSERT_TRUE(!bloom1.MayContain("foo"));

  DynamicBloom bloom2(&arena, CACHE_LINE_SIZE * 8 * 2 - 1, 1, 2);
  bloom2.Add("hello");
  bloom2.Add("world");
  ASSERT_TRUE(bloom2.MayContain("hello"));
  ASSERT_TRUE(bloom2.MayContain("world"));
  ASSERT_TRUE(!bloom2.MayContain("x"));
  ASSERT_TRUE(!bloom2.MayContain("foo"));
}

static uint32_t NextNum(uint32_t num) {
  if (num < 10) {
    num += 1;
  } else if (num < 100) {
    num += 10;
  } else if (num < 1000) {
    num += 100;
  } else {
    num += 1000;
  }
  return num;
}

TEST_F(DynamicBloomTest, VaryingLengths) {
  char buffer[sizeof(uint64_t)];

  // Count number of filters that significantly exceed the false positive rate
  int mediocre_filters = 0;
  int good_filters = 0;
  uint32_t num_probes = static_cast<uint32_t>(FLAGS_num_probes);

  fprintf(stderr, "bits_per_key: %d  num_probes: %d\n",
          FLAGS_bits_per_key, num_probes);

  for (uint32_t enable_locality = 0; enable_locality < 2; ++enable_locality) {
    for (uint32_t num = 1; num <= 10000; num = NextNum(num)) {
      uint32_t bloom_bits = 0;
      Arena arena;
      if (enable_locality == 0) {
        bloom_bits = std::max(num * FLAGS_bits_per_key, 64U);
      } else {
        bloom_bits = std::max(num * FLAGS_bits_per_key,
                              enable_locality * CACHE_LINE_SIZE * 8);
      }
      DynamicBloom bloom(&arena, bloom_bits, enable_locality, num_probes);
      for (uint64_t i = 0; i < num; i++) {
        bloom.Add(Key(i, buffer));
        ASSERT_TRUE(bloom.MayContain(Key(i, buffer)));
      }

      // All added keys must match
      for (uint64_t i = 0; i < num; i++) {
        ASSERT_TRUE(bloom.MayContain(Key(i, buffer)))
          << "Num " << num << "; key " << i;
      }

      // Check false positive rate

      int result = 0;
      for (uint64_t i = 0; i < 10000; i++) {
        if (bloom.MayContain(Key(i + 1000000000, buffer))) {
          result++;
        }
      }
      double rate = result / 10000.0;

      fprintf(stderr,
              "False positives: %5.2f%% @ num = %6u, bloom_bits = %6u, "
              "enable locality?%u\n",
              rate * 100.0, num, bloom_bits, enable_locality);

      if (rate > 0.0125)
        mediocre_filters++;  // Allowed, but not too often
      else
        good_filters++;
    }

    fprintf(stderr, "Filters: %d good, %d mediocre\n",
            good_filters, mediocre_filters);
    ASSERT_LE(mediocre_filters, good_filters/5);
  }
}

TEST_F(DynamicBloomTest, perf) {
  StopWatchNano timer(Env::Default());
  uint32_t num_probes = static_cast<uint32_t>(FLAGS_num_probes);

  if (!FLAGS_enable_perf) {
    return;
  }

  for (uint32_t m = 1; m <= 8; ++m) {
    Arena arena;
    const uint32_t num_keys = m * 8 * 1024 * 1024;
    fprintf(stderr, "testing %" PRIu32 "M keys\n", m * 8);

    DynamicBloom std_bloom(&arena, num_keys * 10, 0, num_probes);

    timer.Start();
    for (uint32_t i = 1; i <= num_keys; ++i) {
      std_bloom.Add(Slice(reinterpret_cast<const char*>(&i), 8));
    }

    uint64_t elapsed = timer.ElapsedNanos();
    fprintf(stderr, "standard bloom, avg add latency %" PRIu64 "\n",
            elapsed / num_keys);

    uint32_t count = 0;
    timer.Start();
    for (uint32_t i = 1; i <= num_keys; ++i) {
      if (std_bloom.MayContain(Slice(reinterpret_cast<const char*>(&i), 8))) {
        ++count;
      }
    }
    ASSERT_EQ(count, num_keys);
    elapsed = timer.ElapsedNanos();
    fprintf(stderr, "standard bloom, avg query latency %" PRIu64 "\n",
            elapsed / count);

    // Locality enabled version
    DynamicBloom blocked_bloom(&arena, num_keys * 10, 1, num_probes);

      timer.Start();
      for (uint32_t i = 1; i <= num_keys; ++i) {
        blocked_bloom.Add(Slice(reinterpret_cast<const char*>(&i), 8));
      }

      elapsed = timer.ElapsedNanos();
      fprintf(stderr,
              "blocked bloom(enable locality), avg add latency %" PRIu64 "\n",
              elapsed / num_keys);

      count = 0;
      timer.Start();
      for (uint32_t i = 1; i <= num_keys; ++i) {
        if (blocked_bloom.MayContain(
                Slice(reinterpret_cast<const char*>(&i), 8))) {
          ++count;
        }
      }

      elapsed = timer.ElapsedNanos();
      fprintf(stderr,
              "blocked bloom(enable locality), avg query latency %" PRIu64 "\n",
              elapsed / count);
      ASSERT_TRUE(count == num_keys);
    }
}

}  // namespace rocksdb

int main(int argc, char** argv) {
  ::testing::InitGoogleTest(&argc, argv);
  ParseCommandLineFlags(&argc, &argv, true);

  return RUN_ALL_TESTS();
}

#endif  // GFLAGS
