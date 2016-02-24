//  Copyright (c) 2013, Facebook, Inc.  All rights reserved.
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
#include <algorithm>
#include <iostream>
#include <mutex>
#include <queue>
#include <set>
#include <thread>
#include <unordered_set>
#include <utility>

#include "db/db_impl.h"
#include "db/dbformat.h"
#include "db/filename.h"
#include "db/job_context.h"
#include "db/version_set.h"
#include "db/write_batch_internal.h"
#include "port/stack_trace.h"
#include "rocksdb/cache.h"
#include "rocksdb/compaction_filter.h"
#include "rocksdb/convenience.h"
#include "rocksdb/db.h"
#include "rocksdb/env.h"
#include "rocksdb/experimental.h"
#include "rocksdb/filter_policy.h"
#include "rocksdb/options.h"
#include "rocksdb/perf_context.h"
#include "rocksdb/slice.h"
#include "rocksdb/slice_transform.h"
#include "rocksdb/table.h"
#include "rocksdb/table_properties.h"
#include "rocksdb/thread_status.h"
#include "rocksdb/utilities/checkpoint.h"
#include "rocksdb/utilities/write_batch_with_index.h"
#include "table/block_based_table_factory.h"
#include "table/mock_table.h"
#include "table/plain_table_factory.h"
#include "util/compression.h"
#include "util/hash.h"
#include "util/hash_linklist_rep.h"
#include "util/logging.h"
#include "util/mock_env.h"
#include "util/mutexlock.h"
#include "util/rate_limiter.h"
#include "util/scoped_arena_iterator.h"
#include "util/statistics.h"
#include "util/string_util.h"
#include "util/sync_point.h"
#include "util/testharness.h"
#include "util/testutil.h"
#include "util/thread_status_util.h"
#include "util/xfunc.h"
#include "utilities/merge_operators.h"

#if !defined(IOS_CROSS_COMPILE) && (!defined(NDEBUG) || !defined(OS_WIN))
#ifndef ROCKSDB_LITE
namespace rocksdb {

static std::string RandomString(Random* rnd, int len, double ratio) {
  std::string r;
  test::CompressibleString(rnd, ratio, len, &r);
  return r;
}

std::string Key(uint64_t key, int length) {
  const int kBufSize = 1000;
  char buf[kBufSize];
  if (length > kBufSize) {
    length = kBufSize;
  }
  snprintf(buf, kBufSize, "%0*" PRIu64, length, key);
  return std::string(buf);
}

class CompactionJobStatsTest : public testing::Test,
                               public testing::WithParamInterface<bool> {
 public:
  std::string dbname_;
  std::string alternative_wal_dir_;
  Env* env_;
  DB* db_;
  std::vector<ColumnFamilyHandle*> handles_;
  uint32_t max_subcompactions_;

  Options last_options_;

  CompactionJobStatsTest() : env_(Env::Default()) {
    env_->SetBackgroundThreads(1, Env::LOW);
    env_->SetBackgroundThreads(1, Env::HIGH);
    dbname_ = test::TmpDir(env_) + "/compaction_job_stats_test";
    alternative_wal_dir_ = dbname_ + "/wal";
    Options options;
    options.create_if_missing = true;
    max_subcompactions_ = GetParam();
    options.max_subcompactions = max_subcompactions_;
    auto delete_options = options;
    delete_options.wal_dir = alternative_wal_dir_;
    EXPECT_OK(DestroyDB(dbname_, delete_options));
    // Destroy it for not alternative WAL dir is used.
    EXPECT_OK(DestroyDB(dbname_, options));
    db_ = nullptr;
    Reopen(options);
  }

  ~CompactionJobStatsTest() {
    rocksdb::SyncPoint::GetInstance()->DisableProcessing();
    rocksdb::SyncPoint::GetInstance()->LoadDependency({});
    rocksdb::SyncPoint::GetInstance()->ClearAllCallBacks();
    Close();
    Options options;
    options.db_paths.emplace_back(dbname_, 0);
    options.db_paths.emplace_back(dbname_ + "_2", 0);
    options.db_paths.emplace_back(dbname_ + "_3", 0);
    options.db_paths.emplace_back(dbname_ + "_4", 0);
    EXPECT_OK(DestroyDB(dbname_, options));
  }

  // Required if inheriting from testing::WithParamInterface<>
  static void SetUpTestCase() {}
  static void TearDownTestCase() {}

  DBImpl* dbfull() {
    return reinterpret_cast<DBImpl*>(db_);
  }

  void CreateColumnFamilies(const std::vector<std::string>& cfs,
                            const Options& options) {
    ColumnFamilyOptions cf_opts(options);
    size_t cfi = handles_.size();
    handles_.resize(cfi + cfs.size());
    for (auto cf : cfs) {
      ASSERT_OK(db_->CreateColumnFamily(cf_opts, cf, &handles_[cfi++]));
    }
  }

  void CreateAndReopenWithCF(const std::vector<std::string>& cfs,
                             const Options& options) {
    CreateColumnFamilies(cfs, options);
    std::vector<std::string> cfs_plus_default = cfs;
    cfs_plus_default.insert(cfs_plus_default.begin(), kDefaultColumnFamilyName);
    ReopenWithColumnFamilies(cfs_plus_default, options);
  }

  void ReopenWithColumnFamilies(const std::vector<std::string>& cfs,
                                const std::vector<Options>& options) {
    ASSERT_OK(TryReopenWithColumnFamilies(cfs, options));
  }

  void ReopenWithColumnFamilies(const std::vector<std::string>& cfs,
                                const Options& options) {
    ASSERT_OK(TryReopenWithColumnFamilies(cfs, options));
  }

  Status TryReopenWithColumnFamilies(
      const std::vector<std::string>& cfs,
      const std::vector<Options>& options) {
    Close();
    EXPECT_EQ(cfs.size(), options.size());
    std::vector<ColumnFamilyDescriptor> column_families;
    for (size_t i = 0; i < cfs.size(); ++i) {
      column_families.push_back(ColumnFamilyDescriptor(cfs[i], options[i]));
    }
    DBOptions db_opts = DBOptions(options[0]);
    return DB::Open(db_opts, dbname_, column_families, &handles_, &db_);
  }

  Status TryReopenWithColumnFamilies(const std::vector<std::string>& cfs,
                                     const Options& options) {
    Close();
    std::vector<Options> v_opts(cfs.size(), options);
    return TryReopenWithColumnFamilies(cfs, v_opts);
  }

  void Reopen(const Options& options) {
    ASSERT_OK(TryReopen(options));
  }

  void Close() {
    for (auto h : handles_) {
      delete h;
    }
    handles_.clear();
    delete db_;
    db_ = nullptr;
  }

  void DestroyAndReopen(const Options& options) {
    // Destroy using last options
    Destroy(last_options_);
    ASSERT_OK(TryReopen(options));
  }

  void Destroy(const Options& options) {
    Close();
    ASSERT_OK(DestroyDB(dbname_, options));
  }

  Status ReadOnlyReopen(const Options& options) {
    return DB::OpenForReadOnly(options, dbname_, &db_);
  }

  Status TryReopen(const Options& options) {
    Close();
    last_options_ = options;
    return DB::Open(options, dbname_, &db_);
  }

  Status Flush(int cf = 0) {
    if (cf == 0) {
      return db_->Flush(FlushOptions());
    } else {
      return db_->Flush(FlushOptions(), handles_[cf]);
    }
  }

  Status Put(const Slice& k, const Slice& v, WriteOptions wo = WriteOptions()) {
    return db_->Put(wo, k, v);
  }

  Status Put(int cf, const Slice& k, const Slice& v,
             WriteOptions wo = WriteOptions()) {
    return db_->Put(wo, handles_[cf], k, v);
  }

  Status Delete(const std::string& k) {
    return db_->Delete(WriteOptions(), k);
  }

  Status Delete(int cf, const std::string& k) {
    return db_->Delete(WriteOptions(), handles_[cf], k);
  }

  std::string Get(const std::string& k, const Snapshot* snapshot = nullptr) {
    ReadOptions options;
    options.verify_checksums = true;
    options.snapshot = snapshot;
    std::string result;
    Status s = db_->Get(options, k, &result);
    if (s.IsNotFound()) {
      result = "NOT_FOUND";
    } else if (!s.ok()) {
      result = s.ToString();
    }
    return result;
  }

  std::string Get(int cf, const std::string& k,
                  const Snapshot* snapshot = nullptr) {
    ReadOptions options;
    options.verify_checksums = true;
    options.snapshot = snapshot;
    std::string result;
    Status s = db_->Get(options, handles_[cf], k, &result);
    if (s.IsNotFound()) {
      result = "NOT_FOUND";
    } else if (!s.ok()) {
      result = s.ToString();
    }
    return result;
  }

  int NumTableFilesAtLevel(int level, int cf = 0) {
    std::string property;
    if (cf == 0) {
      // default cfd
      EXPECT_TRUE(db_->GetProperty(
          "rocksdb.num-files-at-level" + NumberToString(level), &property));
    } else {
      EXPECT_TRUE(db_->GetProperty(
          handles_[cf], "rocksdb.num-files-at-level" + NumberToString(level),
          &property));
    }
    return atoi(property.c_str());
  }

  // Return spread of files per level
  std::string FilesPerLevel(int cf = 0) {
    int num_levels =
        (cf == 0) ? db_->NumberLevels() : db_->NumberLevels(handles_[1]);
    std::string result;
    size_t last_non_zero_offset = 0;
    for (int level = 0; level < num_levels; level++) {
      int f = NumTableFilesAtLevel(level, cf);
      char buf[100];
      snprintf(buf, sizeof(buf), "%s%d", (level ? "," : ""), f);
      result += buf;
      if (f > 0) {
        last_non_zero_offset = result.size();
      }
    }
    result.resize(last_non_zero_offset);
    return result;
  }

  uint64_t Size(const Slice& start, const Slice& limit, int cf = 0) {
    Range r(start, limit);
    uint64_t size;
    if (cf == 0) {
      db_->GetApproximateSizes(&r, 1, &size);
    } else {
      db_->GetApproximateSizes(handles_[1], &r, 1, &size);
    }
    return size;
  }

  void Compact(int cf, const Slice& start, const Slice& limit,
               uint32_t target_path_id) {
    CompactRangeOptions compact_options;
    compact_options.target_path_id = target_path_id;
    ASSERT_OK(db_->CompactRange(compact_options, handles_[cf], &start, &limit));
  }

  void Compact(int cf, const Slice& start, const Slice& limit) {
    ASSERT_OK(
        db_->CompactRange(CompactRangeOptions(), handles_[cf], &start, &limit));
  }

  void Compact(const Slice& start, const Slice& limit) {
    ASSERT_OK(db_->CompactRange(CompactRangeOptions(), &start, &limit));
  }

  void TEST_Compact(int level, int cf, const Slice& start, const Slice& limit) {
    ASSERT_OK(dbfull()->TEST_CompactRange(level, &start, &limit, handles_[cf],
                                          true /* disallow trivial move */));
  }

  // Do n memtable compactions, each of which produces an sstable
  // covering the range [small,large].
  void MakeTables(int n, const std::string& small, const std::string& large,
                  int cf = 0) {
    for (int i = 0; i < n; i++) {
      ASSERT_OK(Put(cf, small, "begin"));
      ASSERT_OK(Put(cf, large, "end"));
      ASSERT_OK(Flush(cf));
    }
  }

  static void SetDeletionCompactionStats(
      CompactionJobStats *stats, uint64_t input_deletions,
      uint64_t expired_deletions, uint64_t records_replaced) {
    stats->num_input_deletion_records = input_deletions;
    stats->num_expired_deletion_records = expired_deletions;
    stats->num_records_replaced = records_replaced;
  }

  void MakeTableWithKeyValues(
    Random* rnd, uint64_t smallest, uint64_t largest,
    int key_size, int value_size, uint64_t interval,
    double ratio, int cf = 0) {
    for (auto key = smallest; key < largest; key += interval) {
      ASSERT_OK(Put(cf, Slice(Key(key, key_size)),
                        Slice(RandomString(rnd, value_size, ratio))));
    }
    ASSERT_OK(Flush(cf));
  }

  // This function behaves with the implicit understanding that two
  // rounds of keys are inserted into the database, as per the behavior
  // of the DeletionStatsTest.
  void SelectivelyDeleteKeys(uint64_t smallest, uint64_t largest,
    uint64_t interval, int deletion_interval, int key_size,
    uint64_t cutoff_key_num, CompactionJobStats* stats, int cf = 0) {

    // interval needs to be >= 2 so that deletion entries can be inserted
    // that are intended to not result in an actual key deletion by using
    // an offset of 1 from another existing key
    ASSERT_GE(interval, 2);

    uint64_t ctr = 1;
    uint32_t deletions_made = 0;
    uint32_t num_deleted = 0;
    uint32_t num_expired = 0;
    for (auto key = smallest; key <= largest; key += interval, ctr++) {
      if (ctr % deletion_interval == 0) {
        ASSERT_OK(Delete(cf, Key(key, key_size)));
        deletions_made++;
        num_deleted++;

        if (key > cutoff_key_num) {
          num_expired++;
        }
      }
    }

    // Insert some deletions for keys that don't exist that
    // are both in and out of the key range
    ASSERT_OK(Delete(cf, Key(smallest+1, key_size)));
    deletions_made++;

    ASSERT_OK(Delete(cf, Key(smallest-1, key_size)));
    deletions_made++;
    num_expired++;

    ASSERT_OK(Delete(cf, Key(smallest-9, key_size)));
    deletions_made++;
    num_expired++;

    ASSERT_OK(Flush(cf));
    SetDeletionCompactionStats(stats, deletions_made, num_expired,
      num_deleted);
  }
};

// An EventListener which helps verify the compaction results in
// test CompactionJobStatsTest.
class CompactionJobStatsChecker : public EventListener {
 public:
  CompactionJobStatsChecker()
      : compression_enabled_(false), verify_next_comp_io_stats_(false) {}

  size_t NumberOfUnverifiedStats() { return expected_stats_.size(); }

  void set_verify_next_comp_io_stats(bool v) { verify_next_comp_io_stats_ = v; }

  // Once a compaction completed, this function will verify the returned
  // CompactionJobInfo with the oldest CompactionJobInfo added earlier
  // in "expected_stats_" which has not yet being used for verification.
  virtual void OnCompactionCompleted(DB *db, const CompactionJobInfo& ci) {
    if (verify_next_comp_io_stats_) {
      ASSERT_GT(ci.stats.file_write_nanos, 0);
      ASSERT_GT(ci.stats.file_range_sync_nanos, 0);
      ASSERT_GT(ci.stats.file_fsync_nanos, 0);
      ASSERT_GT(ci.stats.file_prepare_write_nanos, 0);
      verify_next_comp_io_stats_ = false;
    }

    std::lock_guard<std::mutex> lock(mutex_);
    if (expected_stats_.size()) {
      Verify(ci.stats, expected_stats_.front());
      expected_stats_.pop();
    }
  }

  // A helper function which verifies whether two CompactionJobStats
  // match.  The verification of all compaction stats are done by
  // ASSERT_EQ except for the total input / output bytes, which we
  // use ASSERT_GE and ASSERT_LE with a reasonable bias ---
  // 10% in uncompressed case and 20% when compression is used.
  virtual void Verify(const CompactionJobStats& current_stats,
              const CompactionJobStats& stats) {
    // time
    ASSERT_GT(current_stats.elapsed_micros, 0U);

    ASSERT_EQ(current_stats.num_input_records,
        stats.num_input_records);
    ASSERT_EQ(current_stats.num_input_files,
        stats.num_input_files);
    ASSERT_EQ(current_stats.num_input_files_at_output_level,
        stats.num_input_files_at_output_level);

    ASSERT_EQ(current_stats.num_output_records,
        stats.num_output_records);
    ASSERT_EQ(current_stats.num_output_files,
        stats.num_output_files);

    ASSERT_EQ(current_stats.is_manual_compaction,
        stats.is_manual_compaction);

    // file size
    double kFileSizeBias = compression_enabled_ ? 0.20 : 0.10;
    ASSERT_GE(current_stats.total_input_bytes * (1.00 + kFileSizeBias),
              stats.total_input_bytes);
    ASSERT_LE(current_stats.total_input_bytes,
              stats.total_input_bytes * (1.00 + kFileSizeBias));
    ASSERT_GE(current_stats.total_output_bytes * (1.00 + kFileSizeBias),
              stats.total_output_bytes);
    ASSERT_LE(current_stats.total_output_bytes,
              stats.total_output_bytes * (1.00 + kFileSizeBias));
    ASSERT_EQ(current_stats.total_input_raw_key_bytes,
              stats.total_input_raw_key_bytes);
    ASSERT_EQ(current_stats.total_input_raw_value_bytes,
              stats.total_input_raw_value_bytes);

    ASSERT_EQ(current_stats.num_records_replaced,
        stats.num_records_replaced);

    ASSERT_EQ(current_stats.num_corrupt_keys,
        stats.num_corrupt_keys);

    ASSERT_EQ(
        std::string(current_stats.smallest_output_key_prefix),
        std::string(stats.smallest_output_key_prefix));
    ASSERT_EQ(
        std::string(current_stats.largest_output_key_prefix),
        std::string(stats.largest_output_key_prefix));
  }

  // Add an expected compaction stats, which will be used to
  // verify the CompactionJobStats returned by the OnCompactionCompleted()
  // callback.
  void AddExpectedStats(const CompactionJobStats& stats) {
    std::lock_guard<std::mutex> lock(mutex_);
    expected_stats_.push(stats);
  }

  void EnableCompression(bool flag) {
    compression_enabled_ = flag;
  }

  bool verify_next_comp_io_stats() const { return verify_next_comp_io_stats_; }

 private:
  std::mutex mutex_;
  std::queue<CompactionJobStats> expected_stats_;
  bool compression_enabled_;
  bool verify_next_comp_io_stats_;
};

// An EventListener which helps verify the compaction statistics in
// the test DeletionStatsTest.
class CompactionJobDeletionStatsChecker : public CompactionJobStatsChecker {
 public:
  // Verifies whether two CompactionJobStats match.
  void Verify(const CompactionJobStats& current_stats,
              const CompactionJobStats& stats) {
    ASSERT_EQ(
      current_stats.num_input_deletion_records,
      stats.num_input_deletion_records);
    ASSERT_EQ(
        current_stats.num_expired_deletion_records,
        stats.num_expired_deletion_records);
    ASSERT_EQ(
        current_stats.num_records_replaced,
        stats.num_records_replaced);

    ASSERT_EQ(current_stats.num_corrupt_keys,
        stats.num_corrupt_keys);
  }
};

namespace {

uint64_t EstimatedFileSize(
    uint64_t num_records, size_t key_size, size_t value_size,
    double compression_ratio = 1.0,
    size_t block_size = 4096,
    int bloom_bits_per_key = 10) {
  const size_t kPerKeyOverhead = 8;
  const size_t kFooterSize = 512;

  uint64_t data_size =
      num_records * (key_size + value_size * compression_ratio +
                     kPerKeyOverhead);

  return data_size + kFooterSize
         + num_records * bloom_bits_per_key / 8      // filter block
         + data_size * (key_size + 8) / block_size;  // index block
}

namespace {

void CopyPrefix(
    const Slice& src, size_t prefix_length, std::string* dst) {
  assert(prefix_length > 0);
  size_t length = src.size() > prefix_length ? prefix_length : src.size();
  dst->assign(src.data(), length);
}

}  // namespace

CompactionJobStats NewManualCompactionJobStats(
    const std::string& smallest_key, const std::string& largest_key,
    size_t num_input_files, size_t num_input_files_at_output_level,
    uint64_t num_input_records, size_t key_size, size_t value_size,
    size_t num_output_files, uint64_t num_output_records,
    double compression_ratio, uint64_t num_records_replaced,
    bool is_manual = true) {
  CompactionJobStats stats;
  stats.Reset();

  stats.num_input_records = num_input_records;
  stats.num_input_files = num_input_files;
  stats.num_input_files_at_output_level = num_input_files_at_output_level;

  stats.num_output_records = num_output_records;
  stats.num_output_files = num_output_files;

  stats.total_input_bytes =
      EstimatedFileSize(
          num_input_records / num_input_files,
          key_size, value_size, compression_ratio) * num_input_files;
  stats.total_output_bytes =
      EstimatedFileSize(
          num_output_records / num_output_files,
          key_size, value_size, compression_ratio) * num_output_files;
  stats.total_input_raw_key_bytes =
      num_input_records * (key_size + 8);
  stats.total_input_raw_value_bytes =
      num_input_records * value_size;

  stats.is_manual_compaction = is_manual;

  stats.num_records_replaced = num_records_replaced;

  CopyPrefix(smallest_key,
             CompactionJobStats::kMaxPrefixLength,
             &stats.smallest_output_key_prefix);
  CopyPrefix(largest_key,
             CompactionJobStats::kMaxPrefixLength,
             &stats.largest_output_key_prefix);

  return stats;
}

CompressionType GetAnyCompression() {
  if (Snappy_Supported()) {
    return kSnappyCompression;
  } else if (Zlib_Supported()) {
    return kZlibCompression;
  } else if (BZip2_Supported()) {
    return kBZip2Compression;
  } else if (LZ4_Supported()) {
    return kLZ4Compression;
  }
  return kNoCompression;
}

}  // namespace

TEST_P(CompactionJobStatsTest, CompactionJobStatsTest) {
  Random rnd(301);
  const int kBufSize = 100;
  char buf[kBufSize];
  uint64_t key_base = 100000000l;
  // Note: key_base must be multiple of num_keys_per_L0_file
  int num_keys_per_L0_file = 100;
  const int kTestScale = 8;
  const int kKeySize = 10;
  const int kValueSize = 1000;
  const double kCompressionRatio = 0.5;
  double compression_ratio = 1.0;
  uint64_t key_interval = key_base / num_keys_per_L0_file;

  // Whenever a compaction completes, this listener will try to
  // verify whether the returned CompactionJobStats matches
  // what we expect.  The expected CompactionJobStats is added
  // via AddExpectedStats().
  auto* stats_checker = new CompactionJobStatsChecker();
  Options options;
  options.listeners.emplace_back(stats_checker);
  options.create_if_missing = true;
  options.max_background_flushes = 0;
  // just enough setting to hold off auto-compaction.
  options.level0_file_num_compaction_trigger = kTestScale + 1;
  options.num_levels = 3;
  options.compression = kNoCompression;
  options.max_subcompactions = max_subcompactions_;
  options.bytes_per_sync = 512 * 1024;

  options.compaction_measure_io_stats = true;
  for (int test = 0; test < 2; ++test) {
    DestroyAndReopen(options);
    CreateAndReopenWithCF({"pikachu"}, options);

    // 1st Phase: generate "num_L0_files" L0 files.
    int num_L0_files = 0;
    for (uint64_t start_key = key_base;
                  start_key <= key_base * kTestScale;
                  start_key += key_base) {
      MakeTableWithKeyValues(
          &rnd, start_key, start_key + key_base - 1,
          kKeySize, kValueSize, key_interval,
          compression_ratio, 1);
      snprintf(buf, kBufSize, "%d", ++num_L0_files);
      ASSERT_EQ(std::string(buf), FilesPerLevel(1));
    }
    ASSERT_EQ(ToString(num_L0_files), FilesPerLevel(1));

    // 2nd Phase: perform L0 -> L1 compaction.
    int L0_compaction_count = 6;
    int count = 1;
    std::string smallest_key;
    std::string largest_key;
    for (uint64_t start_key = key_base;
         start_key <= key_base * L0_compaction_count;
         start_key += key_base, count++) {
      smallest_key = Key(start_key, 10);
      largest_key = Key(start_key + key_base - key_interval, 10);
      stats_checker->AddExpectedStats(
          NewManualCompactionJobStats(
              smallest_key, largest_key,
              1, 0, num_keys_per_L0_file,
              kKeySize, kValueSize,
              1, num_keys_per_L0_file,
              compression_ratio, 0));
      ASSERT_EQ(stats_checker->NumberOfUnverifiedStats(), 1U);
      TEST_Compact(0, 1, smallest_key, largest_key);
      snprintf(buf, kBufSize, "%d,%d", num_L0_files - count, count);
      ASSERT_EQ(std::string(buf), FilesPerLevel(1));
    }

    // compact two files into one in the last L0 -> L1 compaction
    int num_remaining_L0 = num_L0_files - L0_compaction_count;
    smallest_key = Key(key_base * (L0_compaction_count + 1), 10);
    largest_key = Key(key_base * (kTestScale + 1) - key_interval, 10);
    stats_checker->AddExpectedStats(
        NewManualCompactionJobStats(
            smallest_key, largest_key,
            num_remaining_L0,
            0, num_keys_per_L0_file * num_remaining_L0,
            kKeySize, kValueSize,
            1, num_keys_per_L0_file * num_remaining_L0,
            compression_ratio, 0));
    ASSERT_EQ(stats_checker->NumberOfUnverifiedStats(), 1U);
    TEST_Compact(0, 1, smallest_key, largest_key);

    int num_L1_files = num_L0_files - num_remaining_L0 + 1;
    num_L0_files = 0;
    snprintf(buf, kBufSize, "%d,%d", num_L0_files, num_L1_files);
    ASSERT_EQ(std::string(buf), FilesPerLevel(1));

    // 3rd Phase: generate sparse L0 files (wider key-range, same num of keys)
    int sparseness = 2;
    for (uint64_t start_key = key_base;
                  start_key <= key_base * kTestScale;
                  start_key += key_base * sparseness) {
      MakeTableWithKeyValues(
          &rnd, start_key, start_key + key_base * sparseness - 1,
          kKeySize, kValueSize,
          key_base * sparseness / num_keys_per_L0_file,
          compression_ratio, 1);
      snprintf(buf, kBufSize, "%d,%d", ++num_L0_files, num_L1_files);
      ASSERT_EQ(std::string(buf), FilesPerLevel(1));
    }

    // 4th Phase: perform L0 -> L1 compaction again, expect higher write amp
    // When subcompactions are enabled, the number of output files increases
    // by 1 because multiple threads are consuming the input and generating
    // output files without coordinating to see if the output could fit into
    // a smaller number of files like it does when it runs sequentially
    int num_output_files = options.max_subcompactions > 1 ? 2 : 1;
    for (uint64_t start_key = key_base;
         num_L0_files > 1;
         start_key += key_base * sparseness) {
      smallest_key = Key(start_key, 10);
      largest_key =
          Key(start_key + key_base * sparseness - key_interval, 10);
      stats_checker->AddExpectedStats(
          NewManualCompactionJobStats(
              smallest_key, largest_key,
              3, 2, num_keys_per_L0_file * 3,
              kKeySize, kValueSize,
              num_output_files,
              num_keys_per_L0_file * 2,  // 1/3 of the data will be updated.
              compression_ratio,
              num_keys_per_L0_file));
      ASSERT_EQ(stats_checker->NumberOfUnverifiedStats(), 1U);
      Compact(1, smallest_key, largest_key);
      if (options.max_subcompactions == 1) {
        --num_L1_files;
      }
      snprintf(buf, kBufSize, "%d,%d", --num_L0_files, num_L1_files);
      ASSERT_EQ(std::string(buf), FilesPerLevel(1));
    }

    // 5th Phase: Do a full compaction, which involves in two sub-compactions.
    // Here we expect to have 1 L0 files and 4 L1 files
    // In the first sub-compaction, we expect L0 compaction.
    smallest_key = Key(key_base, 10);
    largest_key = Key(key_base * (kTestScale + 1) - key_interval, 10);
    stats_checker->AddExpectedStats(
        NewManualCompactionJobStats(
            Key(key_base * (kTestScale + 1 - sparseness), 10), largest_key,
            2, 1, num_keys_per_L0_file * 3,
            kKeySize, kValueSize,
            1, num_keys_per_L0_file * 2,
            compression_ratio,
            num_keys_per_L0_file));
    ASSERT_EQ(stats_checker->NumberOfUnverifiedStats(), 1U);
    Compact(1, smallest_key, largest_key);

    num_L1_files = options.max_subcompactions > 1 ? 7 : 4;
    char L1_buf[4];
    snprintf(L1_buf, sizeof(L1_buf), "0,%d", num_L1_files);
    std::string L1_files(L1_buf);
    ASSERT_EQ(L1_files, FilesPerLevel(1));
    options.compression = GetAnyCompression();
    if (options.compression == kNoCompression) {
      break;
    }
    stats_checker->EnableCompression(true);
    compression_ratio = kCompressionRatio;

    for (int i = 0; i < 5; i++) {
      ASSERT_OK(Put(1, Slice(Key(key_base + i, 10)),
                    Slice(RandomString(&rnd, 512 * 1024, 1))));
    }

    ASSERT_OK(Flush(1));
    reinterpret_cast<DBImpl*>(db_)->TEST_WaitForCompact();

    stats_checker->set_verify_next_comp_io_stats(true);
    std::atomic<bool> first_prepare_write(true);
    rocksdb::SyncPoint::GetInstance()->SetCallBack(
        "WritableFileWriter::Append:BeforePrepareWrite", [&](void* arg) {
          if (first_prepare_write.load()) {
            options.env->SleepForMicroseconds(3);
            first_prepare_write.store(false);
          }
        });

    std::atomic<bool> first_flush(true);
    rocksdb::SyncPoint::GetInstance()->SetCallBack(
        "WritableFileWriter::Flush:BeforeAppend", [&](void* arg) {
          if (first_flush.load()) {
            options.env->SleepForMicroseconds(3);
            first_flush.store(false);
          }
        });

    std::atomic<bool> first_sync(true);
    rocksdb::SyncPoint::GetInstance()->SetCallBack(
        "WritableFileWriter::SyncInternal:0", [&](void* arg) {
          if (first_sync.load()) {
            options.env->SleepForMicroseconds(3);
            first_sync.store(false);
          }
        });

    std::atomic<bool> first_range_sync(true);
    rocksdb::SyncPoint::GetInstance()->SetCallBack(
        "WritableFileWriter::RangeSync:0", [&](void* arg) {
          if (first_range_sync.load()) {
            options.env->SleepForMicroseconds(3);
            first_range_sync.store(false);
          }
        });
    rocksdb::SyncPoint::GetInstance()->EnableProcessing();

    Compact(1, smallest_key, largest_key);

    ASSERT_TRUE(!stats_checker->verify_next_comp_io_stats());
    ASSERT_TRUE(!first_prepare_write.load());
    ASSERT_TRUE(!first_flush.load());
    ASSERT_TRUE(!first_sync.load());
    ASSERT_TRUE(!first_range_sync.load());
    rocksdb::SyncPoint::GetInstance()->DisableProcessing();
  }
  ASSERT_EQ(stats_checker->NumberOfUnverifiedStats(), 0U);
}

TEST_P(CompactionJobStatsTest, DeletionStatsTest) {
  Random rnd(301);
  uint64_t key_base = 100000l;
  // Note: key_base must be multiple of num_keys_per_L0_file
  int num_keys_per_L0_file = 20;
  const int kTestScale = 8;  // make sure this is even
  const int kKeySize = 10;
  const int kValueSize = 100;
  double compression_ratio = 1.0;
  uint64_t key_interval = key_base / num_keys_per_L0_file;
  uint64_t largest_key_num = key_base * (kTestScale + 1) - key_interval;
  uint64_t cutoff_key_num = key_base * (kTestScale / 2 + 1) - key_interval;
  const std::string smallest_key = Key(key_base - 10, kKeySize);
  const std::string largest_key = Key(largest_key_num + 10, kKeySize);

  // Whenever a compaction completes, this listener will try to
  // verify whether the returned CompactionJobStats matches
  // what we expect.
  auto* stats_checker = new CompactionJobDeletionStatsChecker();
  Options options;
  options.listeners.emplace_back(stats_checker);
  options.create_if_missing = true;
  options.max_background_flushes = 0;
  options.level0_file_num_compaction_trigger = kTestScale+1;
  options.num_levels = 3;
  options.compression = kNoCompression;
  options.max_bytes_for_level_multiplier = 2;
  options.max_subcompactions = max_subcompactions_;

  DestroyAndReopen(options);
  CreateAndReopenWithCF({"pikachu"}, options);

  // Stage 1: Generate several L0 files and then send them to L2 by
  // using CompactRangeOptions and CompactRange(). These files will
  // have a strict subset of the keys from the full key-range
  for (uint64_t start_key = key_base;
                start_key <= key_base * kTestScale / 2;
                start_key += key_base) {
    MakeTableWithKeyValues(
        &rnd, start_key, start_key + key_base - 1,
        kKeySize, kValueSize, key_interval,
        compression_ratio, 1);
  }

  CompactRangeOptions cr_options;
  cr_options.change_level = true;
  cr_options.target_level = 2;
  db_->CompactRange(cr_options, handles_[1], nullptr, nullptr);
  ASSERT_GT(NumTableFilesAtLevel(2, 1), 0);

  // Stage 2: Generate files including keys from the entire key range
  for (uint64_t start_key = key_base;
                start_key <= key_base * kTestScale;
                start_key += key_base) {
    MakeTableWithKeyValues(
        &rnd, start_key, start_key + key_base - 1,
        kKeySize, kValueSize, key_interval,
        compression_ratio, 1);
  }

  // Send these L0 files to L1
  TEST_Compact(0, 1, smallest_key, largest_key);
  ASSERT_GT(NumTableFilesAtLevel(1, 1), 0);

  // Add a new record and flush so now there is a L0 file
  // with a value too (not just deletions from the next step)
  ASSERT_OK(Put(1, Key(key_base-6, kKeySize), "test"));
  ASSERT_OK(Flush(1));

  // Stage 3: Generate L0 files with some deletions so now
  // there are files with the same key range in L0, L1, and L2
  int deletion_interval = 3;
  CompactionJobStats first_compaction_stats;
  SelectivelyDeleteKeys(key_base, largest_key_num,
      key_interval, deletion_interval, kKeySize, cutoff_key_num,
      &first_compaction_stats, 1);

  stats_checker->AddExpectedStats(first_compaction_stats);

  // Stage 4: Trigger compaction and verify the stats
  TEST_Compact(0, 1, smallest_key, largest_key);
}

namespace {
int GetUniversalCompactionInputUnits(uint32_t num_flushes) {
  uint32_t compaction_input_units;
  for (compaction_input_units = 1;
       num_flushes >= compaction_input_units;
       compaction_input_units *= 2) {
    if ((num_flushes & compaction_input_units) != 0) {
      return compaction_input_units > 1 ? compaction_input_units : 0;
    }
  }
  return 0;
}
}  // namespace

TEST_P(CompactionJobStatsTest, UniversalCompactionTest) {
  Random rnd(301);
  uint64_t key_base = 100000000l;
  // Note: key_base must be multiple of num_keys_per_L0_file
  int num_keys_per_table = 100;
  const uint32_t kTestScale = 8;
  const int kKeySize = 10;
  const int kValueSize = 900;
  double compression_ratio = 1.0;
  uint64_t key_interval = key_base / num_keys_per_table;

  auto* stats_checker = new CompactionJobStatsChecker();
  Options options;
  options.listeners.emplace_back(stats_checker);
  options.create_if_missing = true;
  options.num_levels = 3;
  options.compression = kNoCompression;
  options.level0_file_num_compaction_trigger = 2;
  options.target_file_size_base = num_keys_per_table * 1000;
  options.compaction_style = kCompactionStyleUniversal;
  options.compaction_options_universal.size_ratio = 1;
  options.compaction_options_universal.max_size_amplification_percent = 1000;
  options.max_subcompactions = max_subcompactions_;

  DestroyAndReopen(options);
  CreateAndReopenWithCF({"pikachu"}, options);

  // Generates the expected CompactionJobStats for each compaction
  for (uint32_t num_flushes = 2; num_flushes <= kTestScale; num_flushes++) {
    // Here we treat one newly flushed file as an unit.
    //
    // For example, if a newly flushed file is 100k, and a compaction has
    // 4 input units, then this compaction inputs 400k.
    uint32_t num_input_units = GetUniversalCompactionInputUnits(num_flushes);
    if (num_input_units == 0) {
      continue;
    }
    // The following statement determines the expected smallest key
    // based on whether it is a full compaction.  A full compaction only
    // happens when the number of flushes equals to the number of compaction
    // input runs.
    uint64_t smallest_key =
        (num_flushes == num_input_units) ?
            key_base : key_base * (num_flushes - 1);

    stats_checker->AddExpectedStats(
        NewManualCompactionJobStats(
            Key(smallest_key, 10),
            Key(smallest_key + key_base * num_input_units - key_interval, 10),
            num_input_units,
            num_input_units > 2 ? num_input_units / 2 : 0,
            num_keys_per_table * num_input_units,
            kKeySize, kValueSize,
            num_input_units,
            num_keys_per_table * num_input_units,
            1.0, 0, false));
  }
  ASSERT_EQ(stats_checker->NumberOfUnverifiedStats(), 4U);

  for (uint64_t start_key = key_base;
                start_key <= key_base * kTestScale;
                start_key += key_base) {
    MakeTableWithKeyValues(
        &rnd, start_key, start_key + key_base - 1,
        kKeySize, kValueSize, key_interval,
        compression_ratio, 1);
    reinterpret_cast<DBImpl*>(db_)->TEST_WaitForCompact();
  }
  ASSERT_EQ(stats_checker->NumberOfUnverifiedStats(), 0U);
}

INSTANTIATE_TEST_CASE_P(CompactionJobStatsTest, CompactionJobStatsTest,
                        ::testing::Values(1, 4));
}  // namespace rocksdb

int main(int argc, char** argv) {
  rocksdb::port::InstallStackTraceHandler();
  ::testing::InitGoogleTest(&argc, argv);
  return RUN_ALL_TESTS();
}

#endif  // !ROCKSDB_LITE

#else

int main(int argc, char** argv) { return 0; }
#endif  // !defined(IOS_CROSS_COMPILE)
