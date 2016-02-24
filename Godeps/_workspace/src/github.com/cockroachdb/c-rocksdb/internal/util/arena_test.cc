//  Copyright (c) 2013, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.
//
// Copyright (c) 2011 The LevelDB Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file. See the AUTHORS file for names of contributors.

#include "util/arena.h"
#include "util/random.h"
#include "util/testharness.h"

namespace rocksdb {

namespace {
const size_t kHugePageSize = 2 * 1024 * 1024;
}  // namespace
class ArenaTest : public testing::Test {};

TEST_F(ArenaTest, Empty) { Arena arena0; }

namespace {
bool CheckMemoryAllocated(size_t allocated, size_t expected) {
  // The value returned by Arena::MemoryAllocatedBytes() may be greater than
  // the requested memory. We choose a somewhat arbitrary upper bound of
  // max_expected = expected * 1.1 to detect critical overallocation.
  size_t max_expected = expected * 1.1;
  return allocated >= expected && allocated <= max_expected;
}

void MemoryAllocatedBytesTest(size_t huge_page_size) {
  const int N = 17;
  size_t req_sz;  // requested size
  size_t bsz = 32 * 1024;  // block size
  size_t expected_memory_allocated;

  Arena arena(bsz, huge_page_size);

  // requested size > quarter of a block:
  //   allocate requested size separately
  req_sz = 12 * 1024;
  for (int i = 0; i < N; i++) {
    arena.Allocate(req_sz);
  }
  expected_memory_allocated = req_sz * N + Arena::kInlineSize;
  ASSERT_PRED2(CheckMemoryAllocated, arena.MemoryAllocatedBytes(),
               expected_memory_allocated);

  arena.Allocate(Arena::kInlineSize - 1);

  // requested size < quarter of a block:
  //   allocate a block with the default size, then try to use unused part
  //   of the block. So one new block will be allocated for the first
  //   Allocate(99) call. All the remaining calls won't lead to new allocation.
  req_sz = 99;
  for (int i = 0; i < N; i++) {
    arena.Allocate(req_sz);
  }
  if (huge_page_size) {
    ASSERT_TRUE(
        CheckMemoryAllocated(arena.MemoryAllocatedBytes(),
                             expected_memory_allocated + bsz) ||
        CheckMemoryAllocated(arena.MemoryAllocatedBytes(),
                             expected_memory_allocated + huge_page_size));
  } else {
    expected_memory_allocated += bsz;
    ASSERT_PRED2(CheckMemoryAllocated, arena.MemoryAllocatedBytes(),
                 expected_memory_allocated);
  }

  // requested size > size of a block:
  //   allocate requested size separately
  expected_memory_allocated = arena.MemoryAllocatedBytes();
  req_sz = 8 * 1024 * 1024;
  for (int i = 0; i < N; i++) {
    arena.Allocate(req_sz);
  }
  expected_memory_allocated += req_sz * N;
  ASSERT_PRED2(CheckMemoryAllocated, arena.MemoryAllocatedBytes(),
               expected_memory_allocated);
}

// Make sure we didn't count the allocate but not used memory space in
// Arena::ApproximateMemoryUsage()
static void ApproximateMemoryUsageTest(size_t huge_page_size) {
  const size_t kBlockSize = 4096;
  const size_t kEntrySize = kBlockSize / 8;
  const size_t kZero = 0;
  Arena arena(kBlockSize, huge_page_size);
  ASSERT_EQ(kZero, arena.ApproximateMemoryUsage());

  // allocate inline bytes
  arena.AllocateAligned(8);
  arena.AllocateAligned(Arena::kInlineSize / 2 - 16);
  arena.AllocateAligned(Arena::kInlineSize / 2);
  ASSERT_EQ(arena.ApproximateMemoryUsage(), Arena::kInlineSize - 8);
  ASSERT_PRED2(CheckMemoryAllocated, arena.MemoryAllocatedBytes(),
               Arena::kInlineSize);

  auto num_blocks = kBlockSize / kEntrySize;

  // first allocation
  arena.AllocateAligned(kEntrySize);
  auto mem_usage = arena.MemoryAllocatedBytes();
  if (huge_page_size) {
    ASSERT_TRUE(
        CheckMemoryAllocated(mem_usage, kBlockSize + Arena::kInlineSize) ||
        CheckMemoryAllocated(mem_usage, huge_page_size + Arena::kInlineSize));
  } else {
    ASSERT_PRED2(CheckMemoryAllocated, mem_usage,
                 kBlockSize + Arena::kInlineSize);
  }
  auto usage = arena.ApproximateMemoryUsage();
  ASSERT_LT(usage, mem_usage);
  for (size_t i = 1; i < num_blocks; ++i) {
    arena.AllocateAligned(kEntrySize);
    ASSERT_EQ(mem_usage, arena.MemoryAllocatedBytes());
    ASSERT_EQ(arena.ApproximateMemoryUsage(), usage + kEntrySize);
    usage = arena.ApproximateMemoryUsage();
  }
  if (huge_page_size) {
    ASSERT_TRUE(usage > mem_usage ||
                usage + huge_page_size - kBlockSize == mem_usage);
  } else {
    ASSERT_GT(usage, mem_usage);
  }
}

static void SimpleTest(size_t huge_page_size) {
  std::vector<std::pair<size_t, char*>> allocated;
  Arena arena(Arena::kMinBlockSize, huge_page_size);
  const int N = 100000;
  size_t bytes = 0;
  Random rnd(301);
  for (int i = 0; i < N; i++) {
    size_t s;
    if (i % (N / 10) == 0) {
      s = i;
    } else {
      s = rnd.OneIn(4000)
              ? rnd.Uniform(6000)
              : (rnd.OneIn(10) ? rnd.Uniform(100) : rnd.Uniform(20));
    }
    if (s == 0) {
      // Our arena disallows size 0 allocations.
      s = 1;
    }
    char* r;
    if (rnd.OneIn(10)) {
      r = arena.AllocateAligned(s);
    } else {
      r = arena.Allocate(s);
    }

    for (unsigned int b = 0; b < s; b++) {
      // Fill the "i"th allocation with a known bit pattern
      r[b] = i % 256;
    }
    bytes += s;
    allocated.push_back(std::make_pair(s, r));
    ASSERT_GE(arena.ApproximateMemoryUsage(), bytes);
    if (i > N / 10) {
      ASSERT_LE(arena.ApproximateMemoryUsage(), bytes * 1.10);
    }
  }
  for (unsigned int i = 0; i < allocated.size(); i++) {
    size_t num_bytes = allocated[i].first;
    const char* p = allocated[i].second;
    for (unsigned int b = 0; b < num_bytes; b++) {
      // Check the "i"th allocation for the known bit pattern
      ASSERT_EQ(int(p[b]) & 0xff, (int)(i % 256));
    }
  }
}
}  // namespace

TEST_F(ArenaTest, MemoryAllocatedBytes) {
  MemoryAllocatedBytesTest(0);
  MemoryAllocatedBytesTest(kHugePageSize);
}

TEST_F(ArenaTest, ApproximateMemoryUsage) {
  ApproximateMemoryUsageTest(0);
  ApproximateMemoryUsageTest(kHugePageSize);
}

TEST_F(ArenaTest, Simple) {
  SimpleTest(0);
  SimpleTest(kHugePageSize);
}
}  // namespace rocksdb

int main(int argc, char** argv) {
  ::testing::InitGoogleTest(&argc, argv);
  return RUN_ALL_TESTS();
}
