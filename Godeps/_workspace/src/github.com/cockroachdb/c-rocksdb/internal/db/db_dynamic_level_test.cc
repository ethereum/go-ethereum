//  Copyright (c) 2013, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.
//
// Copyright (c) 2011 The LevelDB Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file. See the AUTHORS file for names of contributors.

// Introduction of SyncPoint effectively disabled building and running this test
// in Release build.
// which is a pity, it is a good test
#if !(defined NDEBUG) || !defined(OS_WIN)

#include "port/stack_trace.h"
#include "util/db_test_util.h"

namespace rocksdb {
class DBTestDynamicLevel : public DBTestBase {
 public:
  DBTestDynamicLevel() : DBTestBase("/db_dynamic_level_test") {}
};

TEST_F(DBTestDynamicLevel, DynamicLevelMaxBytesBase) {
  if (!Snappy_Supported() || !LZ4_Supported()) {
    return;
  }
  // Use InMemoryEnv, or it would be too slow.
  unique_ptr<Env> env(new MockEnv(env_));

  const int kNKeys = 1000;
  int keys[kNKeys];

  auto verify_func = [&]() {
    for (int i = 0; i < kNKeys; i++) {
      ASSERT_NE("NOT_FOUND", Get(Key(i)));
      ASSERT_NE("NOT_FOUND", Get(Key(kNKeys * 2 + i)));
      if (i < kNKeys / 10) {
        ASSERT_EQ("NOT_FOUND", Get(Key(kNKeys + keys[i])));
      } else {
        ASSERT_NE("NOT_FOUND", Get(Key(kNKeys + keys[i])));
      }
    }
  };

  Random rnd(301);
  for (int ordered_insert = 0; ordered_insert <= 1; ordered_insert++) {
    for (int i = 0; i < kNKeys; i++) {
      keys[i] = i;
    }
    if (ordered_insert == 0) {
      std::random_shuffle(std::begin(keys), std::end(keys));
    }
    for (int max_background_compactions = 1; max_background_compactions < 4;
         max_background_compactions += 2) {
      Options options;
      options.env = env.get();
      options.create_if_missing = true;
      options.db_write_buffer_size = 2048;
      options.write_buffer_size = 2048;
      options.max_write_buffer_number = 2;
      options.level0_file_num_compaction_trigger = 2;
      options.level0_slowdown_writes_trigger = 2;
      options.level0_stop_writes_trigger = 2;
      options.target_file_size_base = 2048;
      options.level_compaction_dynamic_level_bytes = true;
      options.max_bytes_for_level_base = 10240;
      options.max_bytes_for_level_multiplier = 4;
      options.soft_rate_limit = 1.1;
      options.max_background_compactions = max_background_compactions;
      options.num_levels = 5;

      options.compression_per_level.resize(3);
      options.compression_per_level[0] = kNoCompression;
      options.compression_per_level[1] = kLZ4Compression;
      options.compression_per_level[2] = kSnappyCompression;

      DestroyAndReopen(options);

      for (int i = 0; i < kNKeys; i++) {
        int key = keys[i];
        ASSERT_OK(Put(Key(kNKeys + key), RandomString(&rnd, 102)));
        ASSERT_OK(Put(Key(key), RandomString(&rnd, 102)));
        ASSERT_OK(Put(Key(kNKeys * 2 + key), RandomString(&rnd, 102)));
        ASSERT_OK(Delete(Key(kNKeys + keys[i / 10])));
        env_->SleepForMicroseconds(5000);
      }

      uint64_t int_prop;
      ASSERT_TRUE(db_->GetIntProperty("rocksdb.background-errors", &int_prop));
      ASSERT_EQ(0U, int_prop);

      // Verify DB
      for (int j = 0; j < 2; j++) {
        verify_func();
        if (j == 0) {
          Reopen(options);
        }
      }

      // Test compact range works
      dbfull()->CompactRange(CompactRangeOptions(), nullptr, nullptr);
      // All data should be in the last level.
      ColumnFamilyMetaData cf_meta;
      db_->GetColumnFamilyMetaData(&cf_meta);
      ASSERT_EQ(5U, cf_meta.levels.size());
      for (int i = 0; i < 4; i++) {
        ASSERT_EQ(0U, cf_meta.levels[i].files.size());
      }
      ASSERT_GT(cf_meta.levels[4U].files.size(), 0U);
      verify_func();

      Close();
    }
  }

  env_->SetBackgroundThreads(1, Env::LOW);
  env_->SetBackgroundThreads(1, Env::HIGH);
}

// Test specific cases in dynamic max bytes
TEST_F(DBTestDynamicLevel, DynamicLevelMaxBytesBase2) {
  Random rnd(301);
  int kMaxKey = 1000000;

  Options options = CurrentOptions();
  options.create_if_missing = true;
  options.db_write_buffer_size = 2048;
  options.write_buffer_size = 2048;
  options.max_write_buffer_number = 2;
  options.level0_file_num_compaction_trigger = 2;
  options.level0_slowdown_writes_trigger = 9999;
  options.level0_stop_writes_trigger = 9999;
  options.target_file_size_base = 2048;
  options.level_compaction_dynamic_level_bytes = true;
  options.max_bytes_for_level_base = 10240;
  options.max_bytes_for_level_multiplier = 4;
  options.max_background_compactions = 2;
  options.num_levels = 5;
  options.expanded_compaction_factor = 0;  // Force not expanding in compactions
  BlockBasedTableOptions table_options;
  table_options.block_size = 1024;
  options.table_factory.reset(NewBlockBasedTableFactory(table_options));

  DestroyAndReopen(options);
  ASSERT_OK(dbfull()->SetOptions({
      {"disable_auto_compactions", "true"},
  }));

  uint64_t int_prop;
  std::string str_prop;

  // Initial base level is the last level
  ASSERT_TRUE(db_->GetIntProperty("rocksdb.base-level", &int_prop));
  ASSERT_EQ(4U, int_prop);

  // Put about 7K to L0
  for (int i = 0; i < 70; i++) {
    ASSERT_OK(Put(Key(static_cast<int>(rnd.Uniform(kMaxKey))),
                  RandomString(&rnd, 80)));
  }
  ASSERT_OK(dbfull()->SetOptions({
      {"disable_auto_compactions", "false"},
  }));
  Flush();
  dbfull()->TEST_WaitForCompact();
  ASSERT_TRUE(db_->GetIntProperty("rocksdb.base-level", &int_prop));
  ASSERT_EQ(4U, int_prop);

  // Insert extra about 3.5K to L0. After they are compacted to L4, base level
  // should be changed to L3.
  ASSERT_OK(dbfull()->SetOptions({
      {"disable_auto_compactions", "true"},
  }));
  for (int i = 0; i < 70; i++) {
    ASSERT_OK(Put(Key(static_cast<int>(rnd.Uniform(kMaxKey))),
                  RandomString(&rnd, 80)));
  }

  ASSERT_OK(dbfull()->SetOptions({
      {"disable_auto_compactions", "false"},
  }));
  Flush();
  dbfull()->TEST_WaitForCompact();
  ASSERT_TRUE(db_->GetIntProperty("rocksdb.base-level", &int_prop));
  ASSERT_EQ(3U, int_prop);
  ASSERT_TRUE(db_->GetProperty("rocksdb.num-files-at-level1", &str_prop));
  ASSERT_EQ("0", str_prop);
  ASSERT_TRUE(db_->GetProperty("rocksdb.num-files-at-level2", &str_prop));
  ASSERT_EQ("0", str_prop);

  // Trigger parallel compaction, and the first one would change the base
  // level.
  // Hold compaction jobs to make sure
  rocksdb::SyncPoint::GetInstance()->SetCallBack(
      "CompactionJob::Run():Start",
      [&](void* arg) { env_->SleepForMicroseconds(100000); });
  rocksdb::SyncPoint::GetInstance()->EnableProcessing();
  ASSERT_OK(dbfull()->SetOptions({
      {"disable_auto_compactions", "true"},
  }));
  // Write about 10K more
  for (int i = 0; i < 100; i++) {
    ASSERT_OK(Put(Key(static_cast<int>(rnd.Uniform(kMaxKey))),
                  RandomString(&rnd, 80)));
  }
  ASSERT_OK(dbfull()->SetOptions({
      {"disable_auto_compactions", "false"},
  }));
  Flush();
  // Wait for 200 milliseconds before proceeding compactions to make sure two
  // parallel ones are executed.
  env_->SleepForMicroseconds(200000);
  dbfull()->TEST_WaitForCompact();
  ASSERT_TRUE(db_->GetIntProperty("rocksdb.base-level", &int_prop));
  ASSERT_EQ(3U, int_prop);
  rocksdb::SyncPoint::GetInstance()->DisableProcessing();

  // Trigger a condition that the compaction changes base level and L0->Lbase
  // happens at the same time.
  // We try to make last levels' targets to be 10K, 40K, 160K, add triggers
  // another compaction from 40K->160K.
  ASSERT_OK(dbfull()->SetOptions({
      {"disable_auto_compactions", "true"},
  }));
  // Write about 150K more
  for (int i = 0; i < 1350; i++) {
    ASSERT_OK(Put(Key(static_cast<int>(rnd.Uniform(kMaxKey))),
                  RandomString(&rnd, 80)));
  }
  ASSERT_OK(dbfull()->SetOptions({
      {"disable_auto_compactions", "false"},
  }));
  Flush();
  dbfull()->TEST_WaitForCompact();
  ASSERT_TRUE(db_->GetIntProperty("rocksdb.base-level", &int_prop));
  ASSERT_EQ(2U, int_prop);

  // Keep Writing data until base level changed 2->1. There will be L0->L2
  // compaction going on at the same time.
  rocksdb::SyncPoint::GetInstance()->EnableProcessing();
  for (int attempt = 0; attempt <= 20; attempt++) {
    // Write about 5K more data with two flushes. It should be flush to level 2
    // but when it is applied, base level is already 1.
    for (int i = 0; i < 50; i++) {
      ASSERT_OK(Put(Key(static_cast<int>(rnd.Uniform(kMaxKey))),
                    RandomString(&rnd, 80)));
    }
    Flush();

    ASSERT_TRUE(db_->GetIntProperty("rocksdb.base-level", &int_prop));
    if (int_prop == 2U) {
      env_->SleepForMicroseconds(50000);
    } else {
      break;
    }
  }
  rocksdb::SyncPoint::GetInstance()->DisableProcessing();
  rocksdb::SyncPoint::GetInstance()->ClearAllCallBacks();

  env_->SleepForMicroseconds(200000);

  ASSERT_TRUE(db_->GetIntProperty("rocksdb.base-level", &int_prop));
  ASSERT_EQ(1U, int_prop);
}

// Test specific cases in dynamic max bytes
TEST_F(DBTestDynamicLevel, DynamicLevelMaxBytesCompactRange) {
  Random rnd(301);
  int kMaxKey = 1000000;

  Options options = CurrentOptions();
  options.create_if_missing = true;
  options.db_write_buffer_size = 2048;
  options.write_buffer_size = 2048;
  options.max_write_buffer_number = 2;
  options.level0_file_num_compaction_trigger = 2;
  options.level0_slowdown_writes_trigger = 9999;
  options.level0_stop_writes_trigger = 9999;
  options.target_file_size_base = 2;
  options.level_compaction_dynamic_level_bytes = true;
  options.max_bytes_for_level_base = 10240;
  options.max_bytes_for_level_multiplier = 4;
  options.max_background_compactions = 1;
  const int kNumLevels = 5;
  options.num_levels = kNumLevels;
  options.expanded_compaction_factor = 0;  // Force not expanding in compactions
  BlockBasedTableOptions table_options;
  table_options.block_size = 1024;
  options.table_factory.reset(NewBlockBasedTableFactory(table_options));

  DestroyAndReopen(options);

  // Compact against empty DB
  dbfull()->CompactRange(CompactRangeOptions(), nullptr, nullptr);

  uint64_t int_prop;
  std::string str_prop;

  // Initial base level is the last level
  ASSERT_TRUE(db_->GetIntProperty("rocksdb.base-level", &int_prop));
  ASSERT_EQ(4U, int_prop);

  // Put about 7K to L0
  for (int i = 0; i < 140; i++) {
    ASSERT_OK(Put(Key(static_cast<int>(rnd.Uniform(kMaxKey))),
                  RandomString(&rnd, 80)));
  }
  Flush();
  dbfull()->TEST_WaitForCompact();
  if (NumTableFilesAtLevel(0) == 0) {
    // Make sure level 0 is not empty
    ASSERT_OK(Put(Key(static_cast<int>(rnd.Uniform(kMaxKey))),
                  RandomString(&rnd, 80)));
    Flush();
  }

  ASSERT_TRUE(db_->GetIntProperty("rocksdb.base-level", &int_prop));
  ASSERT_EQ(3U, int_prop);
  ASSERT_TRUE(db_->GetProperty("rocksdb.num-files-at-level1", &str_prop));
  ASSERT_EQ("0", str_prop);
  ASSERT_TRUE(db_->GetProperty("rocksdb.num-files-at-level2", &str_prop));
  ASSERT_EQ("0", str_prop);

  rocksdb::SyncPoint::GetInstance()->DisableProcessing();
  rocksdb::SyncPoint::GetInstance()->ClearAllCallBacks();

  std::set<int> output_levels;
  rocksdb::SyncPoint::GetInstance()->SetCallBack(
      "CompactionPicker::CompactRange:Return", [&](void* arg) {
        Compaction* compaction = reinterpret_cast<Compaction*>(arg);
        output_levels.insert(compaction->output_level());
      });
  rocksdb::SyncPoint::GetInstance()->EnableProcessing();

  dbfull()->CompactRange(CompactRangeOptions(), nullptr, nullptr);
  ASSERT_EQ(output_levels.size(), 2);
  ASSERT_TRUE(output_levels.find(3) != output_levels.end());
  ASSERT_TRUE(output_levels.find(4) != output_levels.end());
  ASSERT_TRUE(db_->GetProperty("rocksdb.num-files-at-level0", &str_prop));
  ASSERT_EQ("0", str_prop);
  ASSERT_TRUE(db_->GetProperty("rocksdb.num-files-at-level3", &str_prop));
  ASSERT_EQ("0", str_prop);
  // Base level is still level 3.
  ASSERT_TRUE(db_->GetIntProperty("rocksdb.base-level", &int_prop));
  ASSERT_EQ(3U, int_prop);
}

TEST_F(DBTestDynamicLevel, DynamicLevelMaxBytesBaseInc) {
  Options options = CurrentOptions();
  options.create_if_missing = true;
  options.db_write_buffer_size = 2048;
  options.write_buffer_size = 2048;
  options.max_write_buffer_number = 2;
  options.level0_file_num_compaction_trigger = 2;
  options.level0_slowdown_writes_trigger = 2;
  options.level0_stop_writes_trigger = 2;
  options.target_file_size_base = 2048;
  options.level_compaction_dynamic_level_bytes = true;
  options.max_bytes_for_level_base = 10240;
  options.max_bytes_for_level_multiplier = 4;
  options.soft_rate_limit = 1.1;
  options.max_background_compactions = 2;
  options.num_levels = 5;

  DestroyAndReopen(options);

  int non_trivial = 0;
  rocksdb::SyncPoint::GetInstance()->SetCallBack(
      "DBImpl::BackgroundCompaction:NonTrivial",
      [&](void* arg) { non_trivial++; });
  rocksdb::SyncPoint::GetInstance()->EnableProcessing();

  Random rnd(301);
  const int total_keys = 3000;
  const int random_part_size = 100;
  for (int i = 0; i < total_keys; i++) {
    std::string value = RandomString(&rnd, random_part_size);
    PutFixed32(&value, static_cast<uint32_t>(i));
    ASSERT_OK(Put(Key(i), value));
  }
  Flush();
  dbfull()->TEST_WaitForCompact();
  rocksdb::SyncPoint::GetInstance()->DisableProcessing();

  ASSERT_EQ(non_trivial, 0);

  for (int i = 0; i < total_keys; i++) {
    std::string value = Get(Key(i));
    ASSERT_EQ(DecodeFixed32(value.c_str() + random_part_size),
              static_cast<uint32_t>(i));
  }

  env_->SetBackgroundThreads(1, Env::LOW);
  env_->SetBackgroundThreads(1, Env::HIGH);
}

TEST_F(DBTestDynamicLevel, MigrateToDynamicLevelMaxBytesBase) {
  Random rnd(301);
  const int kMaxKey = 2000;

  Options options;
  options.create_if_missing = true;
  options.db_write_buffer_size = 2048;
  options.write_buffer_size = 2048;
  options.max_write_buffer_number = 8;
  options.level0_file_num_compaction_trigger = 4;
  options.level0_slowdown_writes_trigger = 4;
  options.level0_stop_writes_trigger = 8;
  options.target_file_size_base = 2048;
  options.level_compaction_dynamic_level_bytes = false;
  options.max_bytes_for_level_base = 10240;
  options.max_bytes_for_level_multiplier = 4;
  options.soft_rate_limit = 1.1;
  options.num_levels = 8;

  DestroyAndReopen(options);

  auto verify_func = [&](int num_keys, bool if_sleep) {
    for (int i = 0; i < num_keys; i++) {
      ASSERT_NE("NOT_FOUND", Get(Key(kMaxKey + i)));
      if (i < num_keys / 10) {
        ASSERT_EQ("NOT_FOUND", Get(Key(i)));
      } else {
        ASSERT_NE("NOT_FOUND", Get(Key(i)));
      }
      if (if_sleep && i % 1000 == 0) {
        // Without it, valgrind may choose not to give another
        // thread a chance to run before finishing the function,
        // causing the test to be extremely slow.
        env_->SleepForMicroseconds(1);
      }
    }
  };

  int total_keys = 1000;
  for (int i = 0; i < total_keys; i++) {
    ASSERT_OK(Put(Key(i), RandomString(&rnd, 102)));
    ASSERT_OK(Put(Key(kMaxKey + i), RandomString(&rnd, 102)));
    ASSERT_OK(Delete(Key(i / 10)));
  }
  verify_func(total_keys, false);
  dbfull()->TEST_WaitForCompact();

  options.level_compaction_dynamic_level_bytes = true;
  options.disable_auto_compactions = true;
  Reopen(options);
  verify_func(total_keys, false);

  std::atomic_bool compaction_finished;
  compaction_finished = false;
  // Issue manual compaction in one thread and still verify DB state
  // in main thread.
  std::thread t([&]() {
    CompactRangeOptions compact_options;
    compact_options.change_level = true;
    compact_options.target_level = options.num_levels - 1;
    dbfull()->CompactRange(compact_options, nullptr, nullptr);
    compaction_finished.store(true);
  });
  do {
    verify_func(total_keys, true);
  } while (!compaction_finished.load());
  t.join();

  ASSERT_OK(dbfull()->SetOptions({
      {"disable_auto_compactions", "false"},
  }));

  int total_keys2 = 2000;
  for (int i = total_keys; i < total_keys2; i++) {
    ASSERT_OK(Put(Key(i), RandomString(&rnd, 102)));
    ASSERT_OK(Put(Key(kMaxKey + i), RandomString(&rnd, 102)));
    ASSERT_OK(Delete(Key(i / 10)));
  }

  verify_func(total_keys2, false);
  dbfull()->TEST_WaitForCompact();
  verify_func(total_keys2, false);

  // Base level is not level 1
  ASSERT_EQ(NumTableFilesAtLevel(1), 0);
  ASSERT_EQ(NumTableFilesAtLevel(2), 0);
}
}  // namespace rocksdb

#endif  // !(defined NDEBUG) || !defined(OS_WIN)

int main(int argc, char** argv) {
#if !(defined NDEBUG) || !defined(OS_WIN)
  rocksdb::port::InstallStackTraceHandler();
  ::testing::InitGoogleTest(&argc, argv);
  return RUN_ALL_TESTS();
#else
  return 0;
#endif
}
