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

#include <unordered_map>
#include <inttypes.h>

#include "rocksdb/cache.h"
#include "rocksdb/convenience.h"
#include "rocksdb/options.h"
#include "rocksdb/table.h"
#include "rocksdb/utilities/leveldb_options.h"
#include "table/block_based_table_factory.h"
#include "util/random.h"
#include "util/testharness.h"

#ifndef GFLAGS
bool FLAGS_enable_print = false;
#else
#include <gflags/gflags.h>
using GFLAGS::ParseCommandLineFlags;
DEFINE_bool(enable_print, false, "Print options generated to console.");
#endif  // GFLAGS

namespace rocksdb {

class OptionsTest : public testing::Test {};

class StderrLogger : public Logger {
 public:
  using Logger::Logv;
  virtual void Logv(const char* format, va_list ap) override {
    vprintf(format, ap);
    printf("\n");
  }
};

Options PrintAndGetOptions(size_t total_write_buffer_limit,
                           int read_amplification_threshold,
                           int write_amplification_threshold,
                           uint64_t target_db_size = 68719476736) {
  StderrLogger logger;

  if (FLAGS_enable_print) {
    printf("---- total_write_buffer_limit: %" ROCKSDB_PRIszt
           " "
           "read_amplification_threshold: %d write_amplification_threshold: %d "
           "target_db_size %" PRIu64 " ----\n",
           total_write_buffer_limit, read_amplification_threshold,
           write_amplification_threshold, target_db_size);
  }

  Options options =
      GetOptions(total_write_buffer_limit, read_amplification_threshold,
                 write_amplification_threshold, target_db_size);
  if (FLAGS_enable_print) {
    options.Dump(&logger);
    printf("-------------------------------------\n\n\n");
  }
  return options;
}

TEST_F(OptionsTest, LooseCondition) {
  Options options;
  PrintAndGetOptions(static_cast<size_t>(10) * 1024 * 1024 * 1024, 100, 100);

  // Less mem table memory budget
  PrintAndGetOptions(32 * 1024 * 1024, 100, 100);

  // Tight read amplification
  options = PrintAndGetOptions(128 * 1024 * 1024, 8, 100);
  ASSERT_EQ(options.compaction_style, kCompactionStyleLevel);

#ifndef ROCKSDB_LITE  // Universal compaction is not supported in ROCKSDB_LITE
  // Tight write amplification
  options = PrintAndGetOptions(128 * 1024 * 1024, 64, 10);
  ASSERT_EQ(options.compaction_style, kCompactionStyleUniversal);
#endif  // !ROCKSDB_LITE

  // Both tight amplifications
  PrintAndGetOptions(128 * 1024 * 1024, 4, 8);
}

#ifndef ROCKSDB_LITE  // GetOptionsFromMap is not supported in ROCKSDB_LITE
TEST_F(OptionsTest, GetOptionsFromMapTest) {
  std::unordered_map<std::string, std::string> cf_options_map = {
      {"write_buffer_size", "1"},
      {"max_write_buffer_number", "2"},
      {"min_write_buffer_number_to_merge", "3"},
      {"max_write_buffer_number_to_maintain", "99"},
      {"compression", "kSnappyCompression"},
      {"compression_per_level",
       "kNoCompression:"
       "kSnappyCompression:"
       "kZlibCompression:"
       "kBZip2Compression:"
       "kLZ4Compression:"
       "kLZ4HCCompression:"
       "kZSTDNotFinalCompression"},
      {"compression_opts", "4:5:6"},
      {"num_levels", "7"},
      {"level0_file_num_compaction_trigger", "8"},
      {"level0_slowdown_writes_trigger", "9"},
      {"level0_stop_writes_trigger", "10"},
      {"target_file_size_base", "12"},
      {"target_file_size_multiplier", "13"},
      {"max_bytes_for_level_base", "14"},
      {"level_compaction_dynamic_level_bytes", "true"},
      {"max_bytes_for_level_multiplier", "15"},
      {"max_bytes_for_level_multiplier_additional", "16:17:18"},
      {"expanded_compaction_factor", "19"},
      {"source_compaction_factor", "20"},
      {"max_grandparent_overlap_factor", "21"},
      {"soft_rate_limit", "1.1"},
      {"hard_rate_limit", "2.1"},
      {"arena_block_size", "22"},
      {"disable_auto_compactions", "true"},
      {"compaction_style", "kCompactionStyleLevel"},
      {"verify_checksums_in_compaction", "false"},
      {"compaction_options_fifo", "23"},
      {"filter_deletes", "0"},
      {"max_sequential_skip_in_iterations", "24"},
      {"inplace_update_support", "true"},
      {"compaction_measure_io_stats", "true"},
      {"inplace_update_num_locks", "25"},
      {"memtable_prefix_bloom_bits", "26"},
      {"memtable_prefix_bloom_probes", "27"},
      {"memtable_prefix_bloom_huge_page_tlb_size", "28"},
      {"bloom_locality", "29"},
      {"max_successive_merges", "30"},
      {"min_partial_merge_operands", "31"},
      {"prefix_extractor", "fixed:31"},
      {"optimize_filters_for_hits", "true"},
  };

  std::unordered_map<std::string, std::string> db_options_map = {
      {"create_if_missing", "false"},
      {"create_missing_column_families", "true"},
      {"error_if_exists", "false"},
      {"paranoid_checks", "true"},
      {"max_open_files", "32"},
      {"max_total_wal_size", "33"},
      {"disable_data_sync", "false"},
      {"use_fsync", "true"},
      {"db_log_dir", "/db_log_dir"},
      {"wal_dir", "/wal_dir"},
      {"delete_obsolete_files_period_micros", "34"},
      {"max_background_compactions", "35"},
      {"max_background_flushes", "36"},
      {"max_log_file_size", "37"},
      {"log_file_time_to_roll", "38"},
      {"keep_log_file_num", "39"},
      {"max_manifest_file_size", "40"},
      {"table_cache_numshardbits", "41"},
      {"WAL_ttl_seconds", "43"},
      {"WAL_size_limit_MB", "44"},
      {"manifest_preallocation_size", "45"},
      {"allow_os_buffer", "false"},
      {"allow_mmap_reads", "true"},
      {"allow_mmap_writes", "false"},
      {"is_fd_close_on_exec", "true"},
      {"skip_log_error_on_recovery", "false"},
      {"stats_dump_period_sec", "46"},
      {"advise_random_on_open", "true"},
      {"use_adaptive_mutex", "false"},
      {"new_table_reader_for_compaction_inputs", "true"},
      {"compaction_readahead_size", "100"},
      {"bytes_per_sync", "47"},
      {"wal_bytes_per_sync", "48"}, };

  ColumnFamilyOptions base_cf_opt;
  ColumnFamilyOptions new_cf_opt;
  ASSERT_OK(GetColumnFamilyOptionsFromMap(
            base_cf_opt, cf_options_map, &new_cf_opt));
  ASSERT_EQ(new_cf_opt.write_buffer_size, 1U);
  ASSERT_EQ(new_cf_opt.max_write_buffer_number, 2);
  ASSERT_EQ(new_cf_opt.min_write_buffer_number_to_merge, 3);
  ASSERT_EQ(new_cf_opt.max_write_buffer_number_to_maintain, 99);
  ASSERT_EQ(new_cf_opt.compression, kSnappyCompression);
  ASSERT_EQ(new_cf_opt.compression_per_level.size(), 7U);
  ASSERT_EQ(new_cf_opt.compression_per_level[0], kNoCompression);
  ASSERT_EQ(new_cf_opt.compression_per_level[1], kSnappyCompression);
  ASSERT_EQ(new_cf_opt.compression_per_level[2], kZlibCompression);
  ASSERT_EQ(new_cf_opt.compression_per_level[3], kBZip2Compression);
  ASSERT_EQ(new_cf_opt.compression_per_level[4], kLZ4Compression);
  ASSERT_EQ(new_cf_opt.compression_per_level[5], kLZ4HCCompression);
  ASSERT_EQ(new_cf_opt.compression_per_level[6], kZSTDNotFinalCompression);
  ASSERT_EQ(new_cf_opt.compression_opts.window_bits, 4);
  ASSERT_EQ(new_cf_opt.compression_opts.level, 5);
  ASSERT_EQ(new_cf_opt.compression_opts.strategy, 6);
  ASSERT_EQ(new_cf_opt.num_levels, 7);
  ASSERT_EQ(new_cf_opt.level0_file_num_compaction_trigger, 8);
  ASSERT_EQ(new_cf_opt.level0_slowdown_writes_trigger, 9);
  ASSERT_EQ(new_cf_opt.level0_stop_writes_trigger, 10);
  ASSERT_EQ(new_cf_opt.target_file_size_base, static_cast<uint64_t>(12));
  ASSERT_EQ(new_cf_opt.target_file_size_multiplier, 13);
  ASSERT_EQ(new_cf_opt.max_bytes_for_level_base, 14U);
  ASSERT_EQ(new_cf_opt.level_compaction_dynamic_level_bytes, true);
  ASSERT_EQ(new_cf_opt.max_bytes_for_level_multiplier, 15);
  ASSERT_EQ(new_cf_opt.max_bytes_for_level_multiplier_additional.size(), 3U);
  ASSERT_EQ(new_cf_opt.max_bytes_for_level_multiplier_additional[0], 16);
  ASSERT_EQ(new_cf_opt.max_bytes_for_level_multiplier_additional[1], 17);
  ASSERT_EQ(new_cf_opt.max_bytes_for_level_multiplier_additional[2], 18);
  ASSERT_EQ(new_cf_opt.expanded_compaction_factor, 19);
  ASSERT_EQ(new_cf_opt.source_compaction_factor, 20);
  ASSERT_EQ(new_cf_opt.max_grandparent_overlap_factor, 21);
  ASSERT_EQ(new_cf_opt.soft_rate_limit, 1.1);
  ASSERT_EQ(new_cf_opt.hard_rate_limit, 2.1);
  ASSERT_EQ(new_cf_opt.arena_block_size, 22U);
  ASSERT_EQ(new_cf_opt.disable_auto_compactions, true);
  ASSERT_EQ(new_cf_opt.compaction_style, kCompactionStyleLevel);
  ASSERT_EQ(new_cf_opt.verify_checksums_in_compaction, false);
  ASSERT_EQ(new_cf_opt.compaction_options_fifo.max_table_files_size,
            static_cast<uint64_t>(23));
  ASSERT_EQ(new_cf_opt.filter_deletes, false);
  ASSERT_EQ(new_cf_opt.max_sequential_skip_in_iterations,
            static_cast<uint64_t>(24));
  ASSERT_EQ(new_cf_opt.inplace_update_support, true);
  ASSERT_EQ(new_cf_opt.inplace_update_num_locks, 25U);
  ASSERT_EQ(new_cf_opt.memtable_prefix_bloom_bits, 26U);
  ASSERT_EQ(new_cf_opt.memtable_prefix_bloom_probes, 27U);
  ASSERT_EQ(new_cf_opt.memtable_prefix_bloom_huge_page_tlb_size, 28U);
  ASSERT_EQ(new_cf_opt.bloom_locality, 29U);
  ASSERT_EQ(new_cf_opt.max_successive_merges, 30U);
  ASSERT_EQ(new_cf_opt.min_partial_merge_operands, 31U);
  ASSERT_TRUE(new_cf_opt.prefix_extractor != nullptr);
  ASSERT_EQ(new_cf_opt.optimize_filters_for_hits, true);
  ASSERT_EQ(std::string(new_cf_opt.prefix_extractor->Name()),
            "rocksdb.FixedPrefix.31");

  cf_options_map["write_buffer_size"] = "hello";
  ASSERT_NOK(GetColumnFamilyOptionsFromMap(
             base_cf_opt, cf_options_map, &new_cf_opt));
  cf_options_map["write_buffer_size"] = "1";
  ASSERT_OK(GetColumnFamilyOptionsFromMap(
            base_cf_opt, cf_options_map, &new_cf_opt));
  cf_options_map["unknown_option"] = "1";
  ASSERT_NOK(GetColumnFamilyOptionsFromMap(
             base_cf_opt, cf_options_map, &new_cf_opt));

  DBOptions base_db_opt;
  DBOptions new_db_opt;
  ASSERT_OK(GetDBOptionsFromMap(base_db_opt, db_options_map, &new_db_opt));
  ASSERT_EQ(new_db_opt.create_if_missing, false);
  ASSERT_EQ(new_db_opt.create_missing_column_families, true);
  ASSERT_EQ(new_db_opt.error_if_exists, false);
  ASSERT_EQ(new_db_opt.paranoid_checks, true);
  ASSERT_EQ(new_db_opt.max_open_files, 32);
  ASSERT_EQ(new_db_opt.max_total_wal_size, static_cast<uint64_t>(33));
  ASSERT_EQ(new_db_opt.disableDataSync, false);
  ASSERT_EQ(new_db_opt.use_fsync, true);
  ASSERT_EQ(new_db_opt.db_log_dir, "/db_log_dir");
  ASSERT_EQ(new_db_opt.wal_dir, "/wal_dir");
  ASSERT_EQ(new_db_opt.delete_obsolete_files_period_micros,
            static_cast<uint64_t>(34));
  ASSERT_EQ(new_db_opt.max_background_compactions, 35);
  ASSERT_EQ(new_db_opt.max_background_flushes, 36);
  ASSERT_EQ(new_db_opt.max_log_file_size, 37U);
  ASSERT_EQ(new_db_opt.log_file_time_to_roll, 38U);
  ASSERT_EQ(new_db_opt.keep_log_file_num, 39U);
  ASSERT_EQ(new_db_opt.max_manifest_file_size, static_cast<uint64_t>(40));
  ASSERT_EQ(new_db_opt.table_cache_numshardbits, 41);
  ASSERT_EQ(new_db_opt.WAL_ttl_seconds, static_cast<uint64_t>(43));
  ASSERT_EQ(new_db_opt.WAL_size_limit_MB, static_cast<uint64_t>(44));
  ASSERT_EQ(new_db_opt.manifest_preallocation_size, 45U);
  ASSERT_EQ(new_db_opt.allow_os_buffer, false);
  ASSERT_EQ(new_db_opt.allow_mmap_reads, true);
  ASSERT_EQ(new_db_opt.allow_mmap_writes, false);
  ASSERT_EQ(new_db_opt.is_fd_close_on_exec, true);
  ASSERT_EQ(new_db_opt.skip_log_error_on_recovery, false);
  ASSERT_EQ(new_db_opt.stats_dump_period_sec, 46U);
  ASSERT_EQ(new_db_opt.advise_random_on_open, true);
  ASSERT_EQ(new_db_opt.use_adaptive_mutex, false);
  ASSERT_EQ(new_db_opt.new_table_reader_for_compaction_inputs, true);
  ASSERT_EQ(new_db_opt.compaction_readahead_size, 100);
  ASSERT_EQ(new_db_opt.bytes_per_sync, static_cast<uint64_t>(47));
  ASSERT_EQ(new_db_opt.wal_bytes_per_sync, static_cast<uint64_t>(48));
}
#endif  // !ROCKSDB_LITE

#ifndef ROCKSDB_LITE  // GetColumnFamilyOptionsFromString is not supported in
                      // ROCKSDB_LITE
TEST_F(OptionsTest, GetColumnFamilyOptionsFromStringTest) {
  ColumnFamilyOptions base_cf_opt;
  ColumnFamilyOptions new_cf_opt;
  base_cf_opt.table_factory.reset();
  ASSERT_OK(GetColumnFamilyOptionsFromString(base_cf_opt, "", &new_cf_opt));
  ASSERT_OK(GetColumnFamilyOptionsFromString(base_cf_opt,
            "write_buffer_size=5", &new_cf_opt));
  ASSERT_EQ(new_cf_opt.write_buffer_size, 5U);
  ASSERT_TRUE(new_cf_opt.table_factory == nullptr);
  ASSERT_OK(GetColumnFamilyOptionsFromString(base_cf_opt,
            "write_buffer_size=6;", &new_cf_opt));
  ASSERT_EQ(new_cf_opt.write_buffer_size, 6U);
  ASSERT_OK(GetColumnFamilyOptionsFromString(base_cf_opt,
            "  write_buffer_size =  7  ", &new_cf_opt));
  ASSERT_EQ(new_cf_opt.write_buffer_size, 7U);
  ASSERT_OK(GetColumnFamilyOptionsFromString(base_cf_opt,
            "  write_buffer_size =  8 ; ", &new_cf_opt));
  ASSERT_EQ(new_cf_opt.write_buffer_size, 8U);
  ASSERT_OK(GetColumnFamilyOptionsFromString(base_cf_opt,
            "write_buffer_size=9;max_write_buffer_number=10", &new_cf_opt));
  ASSERT_EQ(new_cf_opt.write_buffer_size, 9U);
  ASSERT_EQ(new_cf_opt.max_write_buffer_number, 10);
  ASSERT_OK(GetColumnFamilyOptionsFromString(base_cf_opt,
            "write_buffer_size=11; max_write_buffer_number  =  12 ;",
            &new_cf_opt));
  ASSERT_EQ(new_cf_opt.write_buffer_size, 11U);
  ASSERT_EQ(new_cf_opt.max_write_buffer_number, 12);
  // Wrong name "max_write_buffer_number_"
  ASSERT_NOK(GetColumnFamilyOptionsFromString(base_cf_opt,
             "write_buffer_size=13;max_write_buffer_number_=14;",
              &new_cf_opt));
  // Wrong key/value pair
  ASSERT_NOK(GetColumnFamilyOptionsFromString(base_cf_opt,
             "write_buffer_size=13;max_write_buffer_number;", &new_cf_opt));
  // Error Paring value
  ASSERT_NOK(GetColumnFamilyOptionsFromString(base_cf_opt,
             "write_buffer_size=13;max_write_buffer_number=;", &new_cf_opt));
  // Missing option name
  ASSERT_NOK(GetColumnFamilyOptionsFromString(base_cf_opt,
             "write_buffer_size=13; =100;", &new_cf_opt));

  const int64_t kilo = 1024UL;
  const int64_t mega = 1024 * kilo;
  const int64_t giga = 1024 * mega;
  const int64_t tera = 1024 * giga;

  // Units (k)
  ASSERT_OK(GetColumnFamilyOptionsFromString(base_cf_opt,
            "memtable_prefix_bloom_bits=14k;max_write_buffer_number=-15K",
            &new_cf_opt));
  ASSERT_EQ(new_cf_opt.memtable_prefix_bloom_bits, 14UL * kilo);
  ASSERT_EQ(new_cf_opt.max_write_buffer_number, -15 * kilo);
  // Units (m)
  ASSERT_OK(GetColumnFamilyOptionsFromString(base_cf_opt,
            "max_write_buffer_number=16m;inplace_update_num_locks=17M",
            &new_cf_opt));
  ASSERT_EQ(new_cf_opt.max_write_buffer_number, 16 * mega);
  ASSERT_EQ(new_cf_opt.inplace_update_num_locks, 17 * mega);
  // Units (g)
  ASSERT_OK(GetColumnFamilyOptionsFromString(
      base_cf_opt,
      "write_buffer_size=18g;prefix_extractor=capped:8;"
      "arena_block_size=19G",
      &new_cf_opt));

  ASSERT_EQ(new_cf_opt.write_buffer_size, 18 * giga);
  ASSERT_EQ(new_cf_opt.arena_block_size, 19 * giga);
  ASSERT_TRUE(new_cf_opt.prefix_extractor.get() != nullptr);
  std::string prefix_name(new_cf_opt.prefix_extractor->Name());
  ASSERT_EQ(prefix_name, "rocksdb.CappedPrefix.8");

  // Units (t)
  ASSERT_OK(GetColumnFamilyOptionsFromString(base_cf_opt,
            "write_buffer_size=20t;arena_block_size=21T", &new_cf_opt));
  ASSERT_EQ(new_cf_opt.write_buffer_size, 20 * tera);
  ASSERT_EQ(new_cf_opt.arena_block_size, 21 * tera);

  // Nested block based table options
  // Emtpy
  ASSERT_OK(GetColumnFamilyOptionsFromString(base_cf_opt,
            "write_buffer_size=10;max_write_buffer_number=16;"
            "block_based_table_factory={};arena_block_size=1024",
            &new_cf_opt));
  ASSERT_TRUE(new_cf_opt.table_factory != nullptr);
  // Non-empty
  ASSERT_OK(GetColumnFamilyOptionsFromString(base_cf_opt,
            "write_buffer_size=10;max_write_buffer_number=16;"
            "block_based_table_factory={block_cache=1M;block_size=4;};"
            "arena_block_size=1024",
            &new_cf_opt));
  ASSERT_TRUE(new_cf_opt.table_factory != nullptr);
  // Last one
  ASSERT_OK(GetColumnFamilyOptionsFromString(base_cf_opt,
            "write_buffer_size=10;max_write_buffer_number=16;"
            "block_based_table_factory={block_cache=1M;block_size=4;}",
            &new_cf_opt));
  ASSERT_TRUE(new_cf_opt.table_factory != nullptr);
  // Mismatch curly braces
  ASSERT_NOK(GetColumnFamilyOptionsFromString(base_cf_opt,
             "write_buffer_size=10;max_write_buffer_number=16;"
             "block_based_table_factory={{{block_size=4;};"
             "arena_block_size=1024",
             &new_cf_opt));
  // Unexpected chars after closing curly brace
  ASSERT_NOK(GetColumnFamilyOptionsFromString(base_cf_opt,
             "write_buffer_size=10;max_write_buffer_number=16;"
             "block_based_table_factory={block_size=4;}};"
             "arena_block_size=1024",
             &new_cf_opt));
  ASSERT_NOK(GetColumnFamilyOptionsFromString(base_cf_opt,
             "write_buffer_size=10;max_write_buffer_number=16;"
             "block_based_table_factory={block_size=4;}xdfa;"
             "arena_block_size=1024",
             &new_cf_opt));
  ASSERT_NOK(GetColumnFamilyOptionsFromString(base_cf_opt,
             "write_buffer_size=10;max_write_buffer_number=16;"
             "block_based_table_factory={block_size=4;}xdfa",
             &new_cf_opt));
  // Invalid block based table option
  ASSERT_NOK(GetColumnFamilyOptionsFromString(base_cf_opt,
             "write_buffer_size=10;max_write_buffer_number=16;"
             "block_based_table_factory={xx_block_size=4;}",
             &new_cf_opt));
  ASSERT_OK(GetColumnFamilyOptionsFromString(base_cf_opt,
           "optimize_filters_for_hits=true",
           &new_cf_opt));
  ASSERT_OK(GetColumnFamilyOptionsFromString(base_cf_opt,
            "optimize_filters_for_hits=false",
            &new_cf_opt));
  ASSERT_NOK(GetColumnFamilyOptionsFromString(base_cf_opt,
              "optimize_filters_for_hits=junk",
              &new_cf_opt));
}
#endif  // !ROCKSDB_LITE

#ifndef ROCKSDB_LITE  // GetBlockBasedTableOptionsFromString is not supported
TEST_F(OptionsTest, GetBlockBasedTableOptionsFromString) {
  BlockBasedTableOptions table_opt;
  BlockBasedTableOptions new_opt;
  // make sure default values are overwritten by something else
  ASSERT_OK(GetBlockBasedTableOptionsFromString(table_opt,
            "cache_index_and_filter_blocks=1;index_type=kHashSearch;"
            "checksum=kxxHash;hash_index_allow_collision=1;no_block_cache=1;"
            "block_cache=1M;block_cache_compressed=1k;block_size=1024;"
            "block_size_deviation=8;block_restart_interval=4;"
            "filter_policy=bloomfilter:4:true;whole_key_filtering=1",
            &new_opt));
  ASSERT_TRUE(new_opt.cache_index_and_filter_blocks);
  ASSERT_EQ(new_opt.index_type, BlockBasedTableOptions::kHashSearch);
  ASSERT_EQ(new_opt.checksum, ChecksumType::kxxHash);
  ASSERT_TRUE(new_opt.hash_index_allow_collision);
  ASSERT_TRUE(new_opt.no_block_cache);
  ASSERT_TRUE(new_opt.block_cache != nullptr);
  ASSERT_EQ(new_opt.block_cache->GetCapacity(), 1024UL*1024UL);
  ASSERT_TRUE(new_opt.block_cache_compressed != nullptr);
  ASSERT_EQ(new_opt.block_cache_compressed->GetCapacity(), 1024UL);
  ASSERT_EQ(new_opt.block_size, 1024UL);
  ASSERT_EQ(new_opt.block_size_deviation, 8);
  ASSERT_EQ(new_opt.block_restart_interval, 4);
  ASSERT_TRUE(new_opt.filter_policy != nullptr);

  // unknown option
  ASSERT_NOK(GetBlockBasedTableOptionsFromString(table_opt,
             "cache_index_and_filter_blocks=1;index_type=kBinarySearch;"
             "bad_option=1",
             &new_opt));

  // unrecognized index type
  ASSERT_NOK(GetBlockBasedTableOptionsFromString(table_opt,
             "cache_index_and_filter_blocks=1;index_type=kBinarySearchXX",
             &new_opt));

  // unrecognized checksum type
  ASSERT_NOK(GetBlockBasedTableOptionsFromString(table_opt,
             "cache_index_and_filter_blocks=1;checksum=kxxHashXX",
             &new_opt));

  // unrecognized filter policy name
  ASSERT_NOK(GetBlockBasedTableOptionsFromString(table_opt,
             "cache_index_and_filter_blocks=1;"
             "filter_policy=bloomfilterxx:4:true",
             &new_opt));
  // unrecognized filter policy config
  ASSERT_NOK(GetBlockBasedTableOptionsFromString(table_opt,
             "cache_index_and_filter_blocks=1;"
             "filter_policy=bloomfilter:4",
             &new_opt));
}
#endif  // !ROCKSDB_LITE

#ifndef ROCKSDB_LITE  // GetOptionsFromString is not supported in RocksDB Lite
TEST_F(OptionsTest, GetOptionsFromStringTest) {
  Options base_options, new_options;
  base_options.write_buffer_size = 20;
  base_options.min_write_buffer_number_to_merge = 15;
  BlockBasedTableOptions block_based_table_options;
  block_based_table_options.cache_index_and_filter_blocks = true;
  base_options.table_factory.reset(
      NewBlockBasedTableFactory(block_based_table_options));
  ASSERT_OK(GetOptionsFromString(
      base_options,
      "write_buffer_size=10;max_write_buffer_number=16;"
      "block_based_table_factory={block_cache=1M;block_size=4;};"
      "create_if_missing=true;max_open_files=1;rate_limiter_bytes_per_sec=1024",
      &new_options));

  ASSERT_EQ(new_options.write_buffer_size, 10U);
  ASSERT_EQ(new_options.max_write_buffer_number, 16);
  BlockBasedTableOptions new_block_based_table_options =
      dynamic_cast<BlockBasedTableFactory*>(new_options.table_factory.get())
          ->GetTableOptions();
  ASSERT_EQ(new_block_based_table_options.block_cache->GetCapacity(), 1U << 20);
  ASSERT_EQ(new_block_based_table_options.block_size, 4U);
  // don't overwrite block based table options
  ASSERT_TRUE(new_block_based_table_options.cache_index_and_filter_blocks);

  ASSERT_EQ(new_options.create_if_missing, true);
  ASSERT_EQ(new_options.max_open_files, 1);
  ASSERT_TRUE(new_options.rate_limiter.get() != nullptr);
}

namespace {
void VerifyDBOptions(const DBOptions& base_opt, const DBOptions& new_opt) {
  // boolean options
  ASSERT_EQ(base_opt.advise_random_on_open, new_opt.advise_random_on_open);
  ASSERT_EQ(base_opt.allow_mmap_reads, new_opt.allow_mmap_reads);
  ASSERT_EQ(base_opt.allow_mmap_writes, new_opt.allow_mmap_writes);
  ASSERT_EQ(base_opt.allow_os_buffer, new_opt.allow_os_buffer);
  ASSERT_EQ(base_opt.create_if_missing, new_opt.create_if_missing);
  ASSERT_EQ(base_opt.create_missing_column_families,
            new_opt.create_missing_column_families);
  ASSERT_EQ(base_opt.disableDataSync, new_opt.disableDataSync);
  ASSERT_EQ(base_opt.enable_thread_tracking, new_opt.enable_thread_tracking);
  ASSERT_EQ(base_opt.error_if_exists, new_opt.error_if_exists);
  ASSERT_EQ(base_opt.is_fd_close_on_exec, new_opt.is_fd_close_on_exec);
  ASSERT_EQ(base_opt.paranoid_checks, new_opt.paranoid_checks);
  ASSERT_EQ(base_opt.skip_log_error_on_recovery,
            new_opt.skip_log_error_on_recovery);
  ASSERT_EQ(base_opt.skip_stats_update_on_db_open,
            new_opt.skip_stats_update_on_db_open);
  ASSERT_EQ(base_opt.use_adaptive_mutex, new_opt.use_adaptive_mutex);
  ASSERT_EQ(base_opt.use_fsync, new_opt.use_fsync);

  // int options
  ASSERT_EQ(base_opt.max_background_compactions,
            new_opt.max_background_compactions);
  ASSERT_EQ(base_opt.max_background_flushes, new_opt.max_background_flushes);
  ASSERT_EQ(base_opt.max_file_opening_threads,
            new_opt.max_file_opening_threads);
  ASSERT_EQ(base_opt.max_open_files, new_opt.max_open_files);
  ASSERT_EQ(base_opt.table_cache_numshardbits,
            new_opt.table_cache_numshardbits);

  // size_t options
  ASSERT_EQ(base_opt.db_write_buffer_size, new_opt.db_write_buffer_size);
  ASSERT_EQ(base_opt.keep_log_file_num, new_opt.keep_log_file_num);
  ASSERT_EQ(base_opt.log_file_time_to_roll, new_opt.log_file_time_to_roll);
  ASSERT_EQ(base_opt.manifest_preallocation_size,
            new_opt.manifest_preallocation_size);
  ASSERT_EQ(base_opt.max_log_file_size, new_opt.max_log_file_size);

  // std::string options
  ASSERT_EQ(base_opt.db_log_dir, new_opt.db_log_dir);
  ASSERT_EQ(base_opt.wal_dir, new_opt.wal_dir);

  // uint32_t options
  ASSERT_EQ(base_opt.max_subcompactions, new_opt.max_subcompactions);

  // uint64_t options
  ASSERT_EQ(base_opt.WAL_size_limit_MB, new_opt.WAL_size_limit_MB);
  ASSERT_EQ(base_opt.WAL_ttl_seconds, new_opt.WAL_ttl_seconds);
  ASSERT_EQ(base_opt.bytes_per_sync, new_opt.bytes_per_sync);
  ASSERT_EQ(base_opt.delayed_write_rate, new_opt.delayed_write_rate);
  ASSERT_EQ(base_opt.delete_obsolete_files_period_micros,
            new_opt.delete_obsolete_files_period_micros);
  ASSERT_EQ(base_opt.max_manifest_file_size, new_opt.max_manifest_file_size);
  ASSERT_EQ(base_opt.max_total_wal_size, new_opt.max_total_wal_size);
  ASSERT_EQ(base_opt.wal_bytes_per_sync, new_opt.wal_bytes_per_sync);

  // unsigned int options
  ASSERT_EQ(base_opt.stats_dump_period_sec, new_opt.stats_dump_period_sec);
}
}  // namespace

TEST_F(OptionsTest, DBOptionsSerialization) {
  Options base_options, new_options;
  Random rnd(301);

  // Phase 1: Make big change in base_options
  // boolean options
  base_options.advise_random_on_open = rnd.Uniform(2);
  base_options.allow_mmap_reads = rnd.Uniform(2);
  base_options.allow_mmap_writes = rnd.Uniform(2);
  base_options.allow_os_buffer = rnd.Uniform(2);
  base_options.create_if_missing = rnd.Uniform(2);
  base_options.create_missing_column_families = rnd.Uniform(2);
  base_options.disableDataSync = rnd.Uniform(2);
  base_options.enable_thread_tracking = rnd.Uniform(2);
  base_options.error_if_exists = rnd.Uniform(2);
  base_options.is_fd_close_on_exec = rnd.Uniform(2);
  base_options.paranoid_checks = rnd.Uniform(2);
  base_options.skip_log_error_on_recovery = rnd.Uniform(2);
  base_options.skip_stats_update_on_db_open = rnd.Uniform(2);
  base_options.use_adaptive_mutex = rnd.Uniform(2);
  base_options.use_fsync = rnd.Uniform(2);

  // int options
  base_options.max_background_compactions = rnd.Uniform(100);
  base_options.max_background_flushes = rnd.Uniform(100);
  base_options.max_file_opening_threads = rnd.Uniform(100);
  base_options.max_open_files = rnd.Uniform(100);
  base_options.table_cache_numshardbits = rnd.Uniform(100);

  // size_t options
  base_options.db_write_buffer_size = rnd.Uniform(10000);
  base_options.keep_log_file_num = rnd.Uniform(10000);
  base_options.log_file_time_to_roll = rnd.Uniform(10000);
  base_options.manifest_preallocation_size = rnd.Uniform(10000);
  base_options.max_log_file_size = rnd.Uniform(10000);

  // std::string options
  base_options.db_log_dir = "path/to/db_log_dir";
  base_options.wal_dir = "path/to/wal_dir";

  // uint32_t options
  base_options.max_subcompactions = rnd.Uniform(100000);

  // uint64_t options
  static const uint64_t uint_max = static_cast<uint64_t>(UINT_MAX);
  base_options.WAL_size_limit_MB = uint_max + rnd.Uniform(100000);
  base_options.WAL_ttl_seconds = uint_max + rnd.Uniform(100000);
  base_options.bytes_per_sync = uint_max + rnd.Uniform(100000);
  base_options.delayed_write_rate = uint_max + rnd.Uniform(100000);
  base_options.delete_obsolete_files_period_micros =
      uint_max + rnd.Uniform(100000);
  base_options.max_manifest_file_size = uint_max + rnd.Uniform(100000);
  base_options.max_total_wal_size = uint_max + rnd.Uniform(100000);
  base_options.wal_bytes_per_sync = uint_max + rnd.Uniform(100000);

  // unsigned int options
  base_options.stats_dump_period_sec = rnd.Uniform(100000);

  // Phase 2: obtain a string from base_option
  std::string base_opt_string;
  ASSERT_OK(GetStringFromDBOptions(base_options, &base_opt_string));

  // Phase 3: Set new_options from the derived string and expect
  //          new_options == base_options
  ASSERT_OK(GetDBOptionsFromString(DBOptions(), base_opt_string, &new_options));
  VerifyDBOptions(base_options, new_options);
}

namespace {
void VerifyDouble(double a, double b) { ASSERT_LT(fabs(a - b), 0.00001); }

void VerifyColumnFamilyOptions(const ColumnFamilyOptions& base_opt,
                               const ColumnFamilyOptions& new_opt) {
  // custom type options
  ASSERT_EQ(base_opt.compaction_style, new_opt.compaction_style);

  // boolean options
  ASSERT_EQ(base_opt.compaction_measure_io_stats,
            new_opt.compaction_measure_io_stats);
  ASSERT_EQ(base_opt.disable_auto_compactions,
            new_opt.disable_auto_compactions);
  ASSERT_EQ(base_opt.filter_deletes, new_opt.filter_deletes);
  ASSERT_EQ(base_opt.inplace_update_support, new_opt.inplace_update_support);
  ASSERT_EQ(base_opt.level_compaction_dynamic_level_bytes,
            new_opt.level_compaction_dynamic_level_bytes);
  ASSERT_EQ(base_opt.optimize_filters_for_hits,
            new_opt.optimize_filters_for_hits);
  ASSERT_EQ(base_opt.paranoid_file_checks, new_opt.paranoid_file_checks);
  ASSERT_EQ(base_opt.purge_redundant_kvs_while_flush,
            new_opt.purge_redundant_kvs_while_flush);
  ASSERT_EQ(base_opt.verify_checksums_in_compaction,
            new_opt.verify_checksums_in_compaction);

  // double options
  VerifyDouble(base_opt.hard_rate_limit, new_opt.hard_rate_limit);
  VerifyDouble(base_opt.soft_rate_limit, new_opt.soft_rate_limit);

  // int options
  ASSERT_EQ(base_opt.expanded_compaction_factor,
            new_opt.expanded_compaction_factor);
  ASSERT_EQ(base_opt.level0_file_num_compaction_trigger,
            new_opt.level0_file_num_compaction_trigger);
  ASSERT_EQ(base_opt.level0_slowdown_writes_trigger,
            new_opt.level0_slowdown_writes_trigger);
  ASSERT_EQ(base_opt.level0_stop_writes_trigger,
            new_opt.level0_stop_writes_trigger);
  ASSERT_EQ(base_opt.max_bytes_for_level_multiplier,
            new_opt.max_bytes_for_level_multiplier);
  ASSERT_EQ(base_opt.max_grandparent_overlap_factor,
            new_opt.max_grandparent_overlap_factor);
  ASSERT_EQ(base_opt.max_mem_compaction_level,
            new_opt.max_mem_compaction_level);
  ASSERT_EQ(base_opt.max_write_buffer_number, new_opt.max_write_buffer_number);
  ASSERT_EQ(base_opt.max_write_buffer_number_to_maintain,
            new_opt.max_write_buffer_number_to_maintain);
  ASSERT_EQ(base_opt.min_write_buffer_number_to_merge,
            new_opt.min_write_buffer_number_to_merge);
  ASSERT_EQ(base_opt.num_levels, new_opt.num_levels);
  ASSERT_EQ(base_opt.source_compaction_factor,
            new_opt.source_compaction_factor);
  ASSERT_EQ(base_opt.target_file_size_multiplier,
            new_opt.target_file_size_multiplier);

  // size_t options
  ASSERT_EQ(base_opt.arena_block_size, new_opt.arena_block_size);
  ASSERT_EQ(base_opt.inplace_update_num_locks,
            new_opt.inplace_update_num_locks);
  ASSERT_EQ(base_opt.max_successive_merges, new_opt.max_successive_merges);
  ASSERT_EQ(base_opt.memtable_prefix_bloom_huge_page_tlb_size,
            new_opt.memtable_prefix_bloom_huge_page_tlb_size);
  ASSERT_EQ(base_opt.write_buffer_size, new_opt.write_buffer_size);

  // uint32_t options
  ASSERT_EQ(base_opt.bloom_locality, new_opt.bloom_locality);
  ASSERT_EQ(base_opt.memtable_prefix_bloom_bits,
            new_opt.memtable_prefix_bloom_bits);
  ASSERT_EQ(base_opt.memtable_prefix_bloom_probes,
            new_opt.memtable_prefix_bloom_probes);
  ASSERT_EQ(base_opt.min_partial_merge_operands,
            new_opt.min_partial_merge_operands);
  ASSERT_EQ(base_opt.max_bytes_for_level_base,
            new_opt.max_bytes_for_level_base);

  // uint64_t options
  ASSERT_EQ(base_opt.max_sequential_skip_in_iterations,
            new_opt.max_sequential_skip_in_iterations);
  ASSERT_EQ(base_opt.target_file_size_base, new_opt.target_file_size_base);

  // unsigned int options
  ASSERT_EQ(base_opt.rate_limit_delay_max_milliseconds,
            new_opt.rate_limit_delay_max_milliseconds);
}
}  // namespace

TEST_F(OptionsTest, ColumnFamilyOptionsSerialization) {
  ColumnFamilyOptions base_opt, new_opt;
  Random rnd(302);
  // Phase 1: randomly assign base_opt
  // custom type options
  base_opt.compaction_style = (CompactionStyle)(rnd.Uniform(4));

  // boolean options
  base_opt.compaction_measure_io_stats = rnd.Uniform(2);
  base_opt.disable_auto_compactions = rnd.Uniform(2);
  base_opt.filter_deletes = rnd.Uniform(2);
  base_opt.inplace_update_support = rnd.Uniform(2);
  base_opt.level_compaction_dynamic_level_bytes = rnd.Uniform(2);
  base_opt.optimize_filters_for_hits = rnd.Uniform(2);
  base_opt.paranoid_file_checks = rnd.Uniform(2);
  base_opt.purge_redundant_kvs_while_flush = rnd.Uniform(2);
  base_opt.verify_checksums_in_compaction = rnd.Uniform(2);

  // double options
  base_opt.hard_rate_limit = static_cast<double>(rnd.Uniform(10000)) / 13;
  base_opt.soft_rate_limit = static_cast<double>(rnd.Uniform(10000)) / 13;

  // int options
  base_opt.expanded_compaction_factor = rnd.Uniform(100);
  base_opt.level0_file_num_compaction_trigger = rnd.Uniform(100);
  base_opt.level0_slowdown_writes_trigger = rnd.Uniform(100);
  base_opt.level0_stop_writes_trigger = rnd.Uniform(100);
  base_opt.max_bytes_for_level_multiplier = rnd.Uniform(100);
  base_opt.max_grandparent_overlap_factor = rnd.Uniform(100);
  base_opt.max_mem_compaction_level = rnd.Uniform(100);
  base_opt.max_write_buffer_number = rnd.Uniform(100);
  base_opt.max_write_buffer_number_to_maintain = rnd.Uniform(100);
  base_opt.min_write_buffer_number_to_merge = rnd.Uniform(100);
  base_opt.num_levels = rnd.Uniform(100);
  base_opt.source_compaction_factor = rnd.Uniform(100);
  base_opt.target_file_size_multiplier = rnd.Uniform(100);

  // size_t options
  base_opt.arena_block_size = rnd.Uniform(10000);
  base_opt.inplace_update_num_locks = rnd.Uniform(10000);
  base_opt.max_successive_merges = rnd.Uniform(10000);
  base_opt.memtable_prefix_bloom_huge_page_tlb_size = rnd.Uniform(10000);
  base_opt.write_buffer_size = rnd.Uniform(10000);

  // uint32_t options
  base_opt.bloom_locality = rnd.Uniform(10000);
  base_opt.memtable_prefix_bloom_bits = rnd.Uniform(10000);
  base_opt.memtable_prefix_bloom_probes = rnd.Uniform(10000);
  base_opt.min_partial_merge_operands = rnd.Uniform(10000);
  base_opt.max_bytes_for_level_base = rnd.Uniform(10000);

  // uint64_t options
  static const uint64_t uint_max = static_cast<uint64_t>(UINT_MAX);
  base_opt.max_sequential_skip_in_iterations = uint_max + rnd.Uniform(10000);
  base_opt.target_file_size_base = uint_max + rnd.Uniform(10000);

  // unsigned int options
  base_opt.rate_limit_delay_max_milliseconds = rnd.Uniform(10000);

  // Phase 2: obtain a string from base_opt
  std::string base_opt_string;
  ASSERT_OK(GetStringFromColumnFamilyOptions(base_opt, &base_opt_string));

  // Phase 3: Set new_opt from the derived string and expect
  //          new_opt == base_opt
  ASSERT_OK(GetColumnFamilyOptionsFromString(ColumnFamilyOptions(),
                                             base_opt_string, &new_opt));
  VerifyColumnFamilyOptions(base_opt, new_opt);
}

#endif  // !ROCKSDB_LITE


Status StringToMap(
    const std::string& opts_str,
    std::unordered_map<std::string, std::string>* opts_map);

#ifndef ROCKSDB_LITE  // StringToMap is not supported in ROCKSDB_LITE
TEST_F(OptionsTest, StringToMapTest) {
  std::unordered_map<std::string, std::string> opts_map;
  // Regular options
  ASSERT_OK(StringToMap("k1=v1;k2=v2;k3=v3", &opts_map));
  ASSERT_EQ(opts_map["k1"], "v1");
  ASSERT_EQ(opts_map["k2"], "v2");
  ASSERT_EQ(opts_map["k3"], "v3");
  // Value with '='
  opts_map.clear();
  ASSERT_OK(StringToMap("k1==v1;k2=v2=;", &opts_map));
  ASSERT_EQ(opts_map["k1"], "=v1");
  ASSERT_EQ(opts_map["k2"], "v2=");
  // Overwrriten option
  opts_map.clear();
  ASSERT_OK(StringToMap("k1=v1;k1=v2;k3=v3", &opts_map));
  ASSERT_EQ(opts_map["k1"], "v2");
  ASSERT_EQ(opts_map["k3"], "v3");
  // Empty value
  opts_map.clear();
  ASSERT_OK(StringToMap("k1=v1;k2=;k3=v3;k4=", &opts_map));
  ASSERT_EQ(opts_map["k1"], "v1");
  ASSERT_TRUE(opts_map.find("k2") != opts_map.end());
  ASSERT_EQ(opts_map["k2"], "");
  ASSERT_EQ(opts_map["k3"], "v3");
  ASSERT_TRUE(opts_map.find("k4") != opts_map.end());
  ASSERT_EQ(opts_map["k4"], "");
  opts_map.clear();
  ASSERT_OK(StringToMap("k1=v1;k2=;k3=v3;k4=   ", &opts_map));
  ASSERT_EQ(opts_map["k1"], "v1");
  ASSERT_TRUE(opts_map.find("k2") != opts_map.end());
  ASSERT_EQ(opts_map["k2"], "");
  ASSERT_EQ(opts_map["k3"], "v3");
  ASSERT_TRUE(opts_map.find("k4") != opts_map.end());
  ASSERT_EQ(opts_map["k4"], "");
  opts_map.clear();
  ASSERT_OK(StringToMap("k1=v1;k2=;k3=", &opts_map));
  ASSERT_EQ(opts_map["k1"], "v1");
  ASSERT_TRUE(opts_map.find("k2") != opts_map.end());
  ASSERT_EQ(opts_map["k2"], "");
  ASSERT_TRUE(opts_map.find("k3") != opts_map.end());
  ASSERT_EQ(opts_map["k3"], "");
  opts_map.clear();
  ASSERT_OK(StringToMap("k1=v1;k2=;k3=;", &opts_map));
  ASSERT_EQ(opts_map["k1"], "v1");
  ASSERT_TRUE(opts_map.find("k2") != opts_map.end());
  ASSERT_EQ(opts_map["k2"], "");
  ASSERT_TRUE(opts_map.find("k3") != opts_map.end());
  ASSERT_EQ(opts_map["k3"], "");
  // Regular nested options
  opts_map.clear();
  ASSERT_OK(StringToMap("k1=v1;k2={nk1=nv1;nk2=nv2};k3=v3", &opts_map));
  ASSERT_EQ(opts_map["k1"], "v1");
  ASSERT_EQ(opts_map["k2"], "nk1=nv1;nk2=nv2");
  ASSERT_EQ(opts_map["k3"], "v3");
  // Multi-level nested options
  opts_map.clear();
  ASSERT_OK(StringToMap("k1=v1;k2={nk1=nv1;nk2={nnk1=nnk2}};"
                        "k3={nk1={nnk1={nnnk1=nnnv1;nnnk2;nnnv2}}};k4=v4",
                        &opts_map));
  ASSERT_EQ(opts_map["k1"], "v1");
  ASSERT_EQ(opts_map["k2"], "nk1=nv1;nk2={nnk1=nnk2}");
  ASSERT_EQ(opts_map["k3"], "nk1={nnk1={nnnk1=nnnv1;nnnk2;nnnv2}}");
  ASSERT_EQ(opts_map["k4"], "v4");
  // Garbage inside curly braces
  opts_map.clear();
  ASSERT_OK(StringToMap("k1=v1;k2={dfad=};k3={=};k4=v4",
                        &opts_map));
  ASSERT_EQ(opts_map["k1"], "v1");
  ASSERT_EQ(opts_map["k2"], "dfad=");
  ASSERT_EQ(opts_map["k3"], "=");
  ASSERT_EQ(opts_map["k4"], "v4");
  // Empty nested options
  opts_map.clear();
  ASSERT_OK(StringToMap("k1=v1;k2={};", &opts_map));
  ASSERT_EQ(opts_map["k1"], "v1");
  ASSERT_EQ(opts_map["k2"], "");
  opts_map.clear();
  ASSERT_OK(StringToMap("k1=v1;k2={{{{}}}{}{}};", &opts_map));
  ASSERT_EQ(opts_map["k1"], "v1");
  ASSERT_EQ(opts_map["k2"], "{{{}}}{}{}");
  // With random spaces
  opts_map.clear();
  ASSERT_OK(StringToMap("  k1 =  v1 ; k2= {nk1=nv1; nk2={nnk1=nnk2}}  ; "
                        "k3={  {   } }; k4= v4  ",
                        &opts_map));
  ASSERT_EQ(opts_map["k1"], "v1");
  ASSERT_EQ(opts_map["k2"], "nk1=nv1; nk2={nnk1=nnk2}");
  ASSERT_EQ(opts_map["k3"], "{   }");
  ASSERT_EQ(opts_map["k4"], "v4");

  // Empty key
  ASSERT_NOK(StringToMap("k1=v1;k2=v2;=", &opts_map));
  ASSERT_NOK(StringToMap("=v1;k2=v2", &opts_map));
  ASSERT_NOK(StringToMap("k1=v1;k2v2;", &opts_map));
  ASSERT_NOK(StringToMap("k1=v1;k2=v2;fadfa", &opts_map));
  ASSERT_NOK(StringToMap("k1=v1;k2=v2;;", &opts_map));
  // Mismatch curly braces
  ASSERT_NOK(StringToMap("k1=v1;k2={;k3=v3", &opts_map));
  ASSERT_NOK(StringToMap("k1=v1;k2={{};k3=v3", &opts_map));
  ASSERT_NOK(StringToMap("k1=v1;k2={}};k3=v3", &opts_map));
  ASSERT_NOK(StringToMap("k1=v1;k2={{}{}}};k3=v3", &opts_map));
  // However this is valid!
  opts_map.clear();
  ASSERT_OK(StringToMap("k1=v1;k2=};k3=v3", &opts_map));
  ASSERT_EQ(opts_map["k1"], "v1");
  ASSERT_EQ(opts_map["k2"], "}");
  ASSERT_EQ(opts_map["k3"], "v3");

  // Invalid chars after closing curly brace
  ASSERT_NOK(StringToMap("k1=v1;k2={{}}{};k3=v3", &opts_map));
  ASSERT_NOK(StringToMap("k1=v1;k2={{}}cfda;k3=v3", &opts_map));
  ASSERT_NOK(StringToMap("k1=v1;k2={{}}  cfda;k3=v3", &opts_map));
  ASSERT_NOK(StringToMap("k1=v1;k2={{}}  cfda", &opts_map));
  ASSERT_NOK(StringToMap("k1=v1;k2={{}}{}", &opts_map));
  ASSERT_NOK(StringToMap("k1=v1;k2={{dfdl}adfa}{}", &opts_map));
}
#endif  // ROCKSDB_LITE

#ifndef ROCKSDB_LITE  // StringToMap is not supported in ROCKSDB_LITE
TEST_F(OptionsTest, StringToMapRandomTest) {
  std::unordered_map<std::string, std::string> opts_map;
  // Make sure segfault is not hit by semi-random strings

  std::vector<std::string> bases = {
      "a={aa={};tt={xxx={}}};c=defff",
      "a={aa={};tt={xxx={}}};c=defff;d={{}yxx{}3{xx}}",
      "abc={{}{}{}{{{}}}{{}{}{}{}{}{}{}"};

  for (std::string base : bases) {
    for (int rand_seed = 301; rand_seed < 401; rand_seed++) {
      Random rnd(rand_seed);
      for (int attempt = 0; attempt < 10; attempt++) {
        std::string str = base;
        // Replace random position to space
        size_t pos = static_cast<size_t>(
            rnd.Uniform(static_cast<int>(base.size())));
        str[pos] = ' ';
        Status s = StringToMap(str, &opts_map);
        ASSERT_TRUE(s.ok() || s.IsInvalidArgument());
        opts_map.clear();
      }
    }
  }

  // Random Construct a string
  std::vector<char> chars = {'{', '}', ' ', '=', ';', 'c'};
  for (int rand_seed = 301; rand_seed < 1301; rand_seed++) {
    Random rnd(rand_seed);
    int len = rnd.Uniform(30);
    std::string str = "";
    for (int attempt = 0; attempt < len; attempt++) {
      // Add a random character
      size_t pos = static_cast<size_t>(
          rnd.Uniform(static_cast<int>(chars.size())));
      str.append(1, chars[pos]);
    }
    Status s = StringToMap(str, &opts_map);
    ASSERT_TRUE(s.ok() || s.IsInvalidArgument());
    s = StringToMap("name=" + str, &opts_map);
    ASSERT_TRUE(s.ok() || s.IsInvalidArgument());
    opts_map.clear();
  }
}
#endif  // !ROCKSDB_LITE

TEST_F(OptionsTest, ConvertOptionsTest) {
  LevelDBOptions leveldb_opt;
  Options converted_opt = ConvertOptions(leveldb_opt);

  ASSERT_EQ(converted_opt.create_if_missing, leveldb_opt.create_if_missing);
  ASSERT_EQ(converted_opt.error_if_exists, leveldb_opt.error_if_exists);
  ASSERT_EQ(converted_opt.paranoid_checks, leveldb_opt.paranoid_checks);
  ASSERT_EQ(converted_opt.env, leveldb_opt.env);
  ASSERT_EQ(converted_opt.info_log.get(), leveldb_opt.info_log);
  ASSERT_EQ(converted_opt.write_buffer_size, leveldb_opt.write_buffer_size);
  ASSERT_EQ(converted_opt.max_open_files, leveldb_opt.max_open_files);
  ASSERT_EQ(converted_opt.compression, leveldb_opt.compression);

  std::shared_ptr<BlockBasedTableFactory> table_factory =
      std::dynamic_pointer_cast<BlockBasedTableFactory>(
          converted_opt.table_factory);

  ASSERT_TRUE(table_factory.get() != nullptr);

  const BlockBasedTableOptions table_opt = table_factory->GetTableOptions();

  ASSERT_EQ(table_opt.block_cache->GetCapacity(), 8UL << 20);
  ASSERT_EQ(table_opt.block_size, leveldb_opt.block_size);
  ASSERT_EQ(table_opt.block_restart_interval,
            leveldb_opt.block_restart_interval);
  ASSERT_EQ(table_opt.filter_policy.get(), leveldb_opt.filter_policy);
}

}  // namespace rocksdb

int main(int argc, char** argv) {
  ::testing::InitGoogleTest(&argc, argv);
#ifdef GFLAGS
  ParseCommandLineFlags(&argc, &argv, true);
#endif  // GFLAGS
  return RUN_ALL_TESTS();
}
