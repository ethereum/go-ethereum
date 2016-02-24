//  Copyright (c) 2013, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.
//
// Copyright (c) 2011 The LevelDB Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file. See the AUTHORS file for names of contributors.

#include <stdio.h>

#ifndef ROCKSDB_LITE
#include <algorithm>
#include <cmath>
#include <vector>

#include "rocksdb/table.h"
#include "rocksdb/utilities/table_properties_collectors.h"
#include "util/random.h"
#include "utilities/table_properties_collectors/compact_on_deletion_collector.h"

int main(int argc, char** argv) {
  const int kWindowSizes[] =
      {1000, 10000, 10000, 127, 128, 129, 255, 256, 257, 2, 10000};
  const int kDeletionTriggers[] =
      {500, 9500, 4323, 47, 61, 128, 250, 250, 250, 2, 2};

  std::vector<int> window_sizes;
  std::vector<int> deletion_triggers;
  // deterministic tests
  for (int test = 0; test < 9; ++test) {
    window_sizes.emplace_back(kWindowSizes[test]);
    deletion_triggers.emplace_back(kDeletionTriggers[test]);
  }

  // randomize tests
  rocksdb::Random rnd(301);
  const int kMaxTestSize = 100000l;
  for (int random_test = 0; random_test < 100; random_test++) {
    int window_size = rnd.Uniform(kMaxTestSize) + 1;
    int deletion_trigger = rnd.Uniform(window_size);
    window_sizes.emplace_back(window_size);
    deletion_triggers.emplace_back(deletion_trigger);
  }

  assert(window_sizes.size() == deletion_triggers.size());

  for (size_t test = 0; test < window_sizes.size(); ++test) {
    const int kBucketSize = 128;
    const int kWindowSize = window_sizes[test];
    const int kPaddedWindowSize =
        kBucketSize * ((window_sizes[test] + kBucketSize - 1) / kBucketSize);
    const int kNumDeletionTrigger = deletion_triggers[test];
    const int kBias = (kNumDeletionTrigger + kBucketSize - 1) / kBucketSize;
    // Simple test
    {
      std::unique_ptr<rocksdb::TablePropertiesCollector> collector;
      auto factory = rocksdb::NewCompactOnDeletionCollectorFactory(
          kWindowSize, kNumDeletionTrigger);
      collector.reset(
          factory->CreateTablePropertiesCollector());
      const int kSample = 10;
      for (int delete_rate = 0; delete_rate <= kSample; ++delete_rate) {
        int deletions = 0;
        for (int i = 0; i < kPaddedWindowSize; ++i) {
          if (i % kSample < delete_rate) {
            collector->AddUserKey("hello", "rocksdb",
                                  rocksdb::kEntryDelete, 0, 0);
            deletions++;
          } else {
            collector->AddUserKey("hello", "rocksdb",
                                  rocksdb::kEntryPut, 0, 0);
          }
        }
        if (collector->NeedCompact() !=
            (deletions >= kNumDeletionTrigger) &&
            std::abs(deletions - kNumDeletionTrigger) > kBias) {
          fprintf(stderr, "[Error] collector->NeedCompact() != (%d >= %d)"
                  " with kWindowSize = %d and kNumDeletionTrigger = %d\n",
                  deletions, kNumDeletionTrigger,
                  kWindowSize, kNumDeletionTrigger);
          assert(false);
        }
        collector->Finish(nullptr);
      }
    }

    // Only one section of a file satisfies the compaction trigger
    {
      std::unique_ptr<rocksdb::TablePropertiesCollector> collector;
      auto factory = rocksdb::NewCompactOnDeletionCollectorFactory(
          kWindowSize, kNumDeletionTrigger);
      collector.reset(
          factory->CreateTablePropertiesCollector());
      const int kSample = 10;
      for (int delete_rate = 0; delete_rate <= kSample; ++delete_rate) {
        int deletions = 0;
        for (int section = 0; section < 5; ++section) {
          int initial_entries = rnd.Uniform(kWindowSize) + kWindowSize;
          for (int i = 0; i < initial_entries; ++i) {
            collector->AddUserKey("hello", "rocksdb",
                                  rocksdb::kEntryPut, 0, 0);
          }
        }
        for (int i = 0; i < kPaddedWindowSize; ++i) {
          if (i % kSample < delete_rate) {
            collector->AddUserKey("hello", "rocksdb",
                                  rocksdb::kEntryDelete, 0, 0);
            deletions++;
          } else {
            collector->AddUserKey("hello", "rocksdb",
                                  rocksdb::kEntryPut, 0, 0);
          }
        }
        for (int section = 0; section < 5; ++section) {
          int ending_entries = rnd.Uniform(kWindowSize) + kWindowSize;
          for (int i = 0; i < ending_entries; ++i) {
            collector->AddUserKey("hello", "rocksdb",
                                  rocksdb::kEntryPut, 0, 0);
          }
        }
        if (collector->NeedCompact() != (deletions >= kNumDeletionTrigger) &&
            std::abs(deletions - kNumDeletionTrigger) > kBias) {
          fprintf(stderr, "[Error] collector->NeedCompact() %d != (%d >= %d)"
                  " with kWindowSize = %d, kNumDeletionTrigger = %d\n",
                  collector->NeedCompact(),
                  deletions, kNumDeletionTrigger, kWindowSize,
                  kNumDeletionTrigger);
          assert(false);
        }
        collector->Finish(nullptr);
      }
    }

    // TEST 3:  Issues a lots of deletes, but their density is not
    // high enough to trigger compaction.
    {
      std::unique_ptr<rocksdb::TablePropertiesCollector> collector;
      auto factory = rocksdb::NewCompactOnDeletionCollectorFactory(
          kWindowSize, kNumDeletionTrigger);
      collector.reset(
          factory->CreateTablePropertiesCollector());
      assert(collector->NeedCompact() == false);
      // Insert "kNumDeletionTrigger * 0.95" deletions for every
      // "kWindowSize" and verify compaction is not needed.
      const int kDeletionsPerSection = kNumDeletionTrigger * 95 / 100;
      if (kDeletionsPerSection >= 0) {
        for (int section = 0; section < 200; ++section) {
          for (int i = 0; i < kPaddedWindowSize; ++i) {
            if (i < kDeletionsPerSection) {
              collector->AddUserKey("hello", "rocksdb",
                                    rocksdb::kEntryDelete, 0, 0);
            } else {
              collector->AddUserKey("hello", "rocksdb",
                                    rocksdb::kEntryPut, 0, 0);
            }
          }
        }
        if (collector->NeedCompact() &&
            std::abs(kDeletionsPerSection - kNumDeletionTrigger) > kBias) {
          fprintf(stderr, "[Error] collector->NeedCompact() != false"
                  " with kWindowSize = %d and kNumDeletionTrigger = %d\n",
                  kWindowSize, kNumDeletionTrigger);
          assert(false);
        }
        collector->Finish(nullptr);
      }
    }
  }
  fprintf(stderr, "PASSED\n");
}
#else
int main(int argc, char** argv) {
  fprintf(stderr, "SKIPPED as RocksDBLite does not include utilities.\n");
  return 0;
}
#endif  // !ROCKSDB_LITE
