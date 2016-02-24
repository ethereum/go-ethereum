// Copyright (c) 2014, Facebook, Inc.  All rights reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.

#pragma once

#include <string>
#include <stdexcept>
#include "rocksdb/options.h"
#include "rocksdb/status.h"
#include "util/mutable_cf_options.h"

namespace rocksdb {

Status GetMutableOptionsFromStrings(
    const MutableCFOptions& base_options,
    const std::unordered_map<std::string, std::string>& options_map,
    MutableCFOptions* new_options);

enum class OptionType {
  kBoolean,
  kInt,
  kUInt,
  kUInt32T,
  kUInt64T,
  kSizeT,
  kString,
  kDouble,
  kCompactionStyle,
  kUnknown
};

// A struct for storing constant option information such as option name,
// option type, and offset.
struct OptionTypeInfo {
  int offset;
  OptionType type;
};

static std::unordered_map<std::string, OptionTypeInfo> db_options_type_info = {
    /*
     // not yet supported
      AccessHint access_hint_on_compaction_start;
      Env* env;
      InfoLogLevel info_log_level;
      WALRecoveryMode wal_recovery_mode;
      std::shared_ptr<Cache> row_cache;
      std::shared_ptr<DeleteScheduler> delete_scheduler;
      std::shared_ptr<Logger> info_log;
      std::shared_ptr<RateLimiter> rate_limiter;
      std::shared_ptr<Statistics> statistics;
      std::vector<DbPath> db_paths;
      std::vector<std::shared_ptr<EventListener>> listeners;
     */
    {"advise_random_on_open",
     {offsetof(struct DBOptions, advise_random_on_open), OptionType::kBoolean}},
    {"allow_mmap_reads",
     {offsetof(struct DBOptions, allow_mmap_reads), OptionType::kBoolean}},
    {"allow_mmap_writes",
     {offsetof(struct DBOptions, allow_mmap_writes), OptionType::kBoolean}},
    {"allow_os_buffer",
     {offsetof(struct DBOptions, allow_os_buffer), OptionType::kBoolean}},
    {"create_if_missing",
     {offsetof(struct DBOptions, create_if_missing), OptionType::kBoolean}},
    {"create_missing_column_families",
     {offsetof(struct DBOptions, create_missing_column_families),
      OptionType::kBoolean}},
    {"disableDataSync",
     {offsetof(struct DBOptions, disableDataSync), OptionType::kBoolean}},
    {"disable_data_sync",  // for compatibility
     {offsetof(struct DBOptions, disableDataSync), OptionType::kBoolean}},
    {"enable_thread_tracking",
     {offsetof(struct DBOptions, enable_thread_tracking),
      OptionType::kBoolean}},
    {"error_if_exists",
     {offsetof(struct DBOptions, error_if_exists), OptionType::kBoolean}},
    {"is_fd_close_on_exec",
     {offsetof(struct DBOptions, is_fd_close_on_exec), OptionType::kBoolean}},
    {"paranoid_checks",
     {offsetof(struct DBOptions, paranoid_checks), OptionType::kBoolean}},
    {"skip_log_error_on_recovery",
     {offsetof(struct DBOptions, skip_log_error_on_recovery),
      OptionType::kBoolean}},
    {"skip_stats_update_on_db_open",
     {offsetof(struct DBOptions, skip_stats_update_on_db_open),
      OptionType::kBoolean}},
    {"new_table_reader_for_compaction_inputs",
     {offsetof(struct DBOptions, new_table_reader_for_compaction_inputs),
      OptionType::kBoolean}},
    {"compaction_readahead_size",
     {offsetof(struct DBOptions, compaction_readahead_size),
      OptionType::kSizeT}},
    {"use_adaptive_mutex",
     {offsetof(struct DBOptions, use_adaptive_mutex), OptionType::kBoolean}},
    {"use_fsync",
     {offsetof(struct DBOptions, use_fsync), OptionType::kBoolean}},
    {"max_background_compactions",
     {offsetof(struct DBOptions, max_background_compactions),
      OptionType::kInt}},
    {"max_background_flushes",
     {offsetof(struct DBOptions, max_background_flushes), OptionType::kInt}},
    {"max_file_opening_threads",
     {offsetof(struct DBOptions, max_file_opening_threads), OptionType::kInt}},
    {"max_open_files",
     {offsetof(struct DBOptions, max_open_files), OptionType::kInt}},
    {"table_cache_numshardbits",
     {offsetof(struct DBOptions, table_cache_numshardbits), OptionType::kInt}},
    {"db_write_buffer_size",
     {offsetof(struct DBOptions, db_write_buffer_size), OptionType::kSizeT}},
    {"keep_log_file_num",
     {offsetof(struct DBOptions, keep_log_file_num), OptionType::kSizeT}},
    {"log_file_time_to_roll",
     {offsetof(struct DBOptions, log_file_time_to_roll), OptionType::kSizeT}},
    {"manifest_preallocation_size",
     {offsetof(struct DBOptions, manifest_preallocation_size),
      OptionType::kSizeT}},
    {"max_log_file_size",
     {offsetof(struct DBOptions, max_log_file_size), OptionType::kSizeT}},
    {"db_log_dir",
     {offsetof(struct DBOptions, db_log_dir), OptionType::kString}},
    {"wal_dir", {offsetof(struct DBOptions, wal_dir), OptionType::kString}},
    {"max_subcompactions",
     {offsetof(struct DBOptions, max_subcompactions), OptionType::kUInt32T}},
    {"WAL_size_limit_MB",
     {offsetof(struct DBOptions, WAL_size_limit_MB), OptionType::kUInt64T}},
    {"WAL_ttl_seconds",
     {offsetof(struct DBOptions, WAL_ttl_seconds), OptionType::kUInt64T}},
    {"bytes_per_sync",
     {offsetof(struct DBOptions, bytes_per_sync), OptionType::kUInt64T}},
    {"delayed_write_rate",
     {offsetof(struct DBOptions, delayed_write_rate), OptionType::kUInt64T}},
    {"delete_obsolete_files_period_micros",
     {offsetof(struct DBOptions, delete_obsolete_files_period_micros),
      OptionType::kUInt64T}},
    {"max_manifest_file_size",
     {offsetof(struct DBOptions, max_manifest_file_size),
      OptionType::kUInt64T}},
    {"max_total_wal_size",
     {offsetof(struct DBOptions, max_total_wal_size), OptionType::kUInt64T}},
    {"wal_bytes_per_sync",
     {offsetof(struct DBOptions, wal_bytes_per_sync), OptionType::kUInt64T}},
    {"stats_dump_period_sec",
     {offsetof(struct DBOptions, stats_dump_period_sec), OptionType::kUInt}}};

static std::unordered_map<std::string, OptionTypeInfo> cf_options_type_info = {
    /* not yet supported
    CompactionOptionsFIFO compaction_options_fifo;
    CompactionOptionsUniversal compaction_options_universal;
    CompressionOptions compression_opts;
    CompressionType compression;
    TablePropertiesCollectorFactories table_properties_collector_factories;
    typedef std::vector<std::shared_ptr<TablePropertiesCollectorFactory>>
        TablePropertiesCollectorFactories;
    UpdateStatus (*inplace_callback)(char* existing_value,
                                     uint34_t* existing_value_size,
                                     Slice delta_value,
                                     std::string* merged_value);
    const CompactionFilter* compaction_filter;
    const Comparator* comparator;
    std::shared_ptr<CompactionFilterFactory> compaction_filter_factory;
    std::shared_ptr<MemTableRepFactory> memtable_factory;
    std::shared_ptr<MergeOperator> merge_operator;
    std::shared_ptr<TableFactory> table_factory;
    std::shared_ptr<const SliceTransform> prefix_extractor;
    std::vector<CompressionType> compression_per_level;
    std::vector<int> max_bytes_for_level_multiplier_additional;
     */
    {"compaction_measure_io_stats",
     {offsetof(struct ColumnFamilyOptions, compaction_measure_io_stats),
      OptionType::kBoolean}},
    {"disable_auto_compactions",
     {offsetof(struct ColumnFamilyOptions, disable_auto_compactions),
      OptionType::kBoolean}},
    {"filter_deletes",
     {offsetof(struct ColumnFamilyOptions, filter_deletes),
      OptionType::kBoolean}},
    {"inplace_update_support",
     {offsetof(struct ColumnFamilyOptions, inplace_update_support),
      OptionType::kBoolean}},
    {"level_compaction_dynamic_level_bytes",
     {offsetof(struct ColumnFamilyOptions,
               level_compaction_dynamic_level_bytes),
      OptionType::kBoolean}},
    {"optimize_filters_for_hits",
     {offsetof(struct ColumnFamilyOptions, optimize_filters_for_hits),
      OptionType::kBoolean}},
    {"paranoid_file_checks",
     {offsetof(struct ColumnFamilyOptions, paranoid_file_checks),
      OptionType::kBoolean}},
    {"purge_redundant_kvs_while_flush",
     {offsetof(struct ColumnFamilyOptions, purge_redundant_kvs_while_flush),
      OptionType::kBoolean}},
    {"verify_checksums_in_compaction",
     {offsetof(struct ColumnFamilyOptions, verify_checksums_in_compaction),
      OptionType::kBoolean}},
    {"hard_rate_limit",
     {offsetof(struct ColumnFamilyOptions, hard_rate_limit),
      OptionType::kDouble}},
    {"soft_rate_limit",
     {offsetof(struct ColumnFamilyOptions, soft_rate_limit),
      OptionType::kDouble}},
    {"expanded_compaction_factor",
     {offsetof(struct ColumnFamilyOptions, expanded_compaction_factor),
      OptionType::kInt}},
    {"level0_file_num_compaction_trigger",
     {offsetof(struct ColumnFamilyOptions, level0_file_num_compaction_trigger),
      OptionType::kInt}},
    {"level0_slowdown_writes_trigger",
     {offsetof(struct ColumnFamilyOptions, level0_slowdown_writes_trigger),
      OptionType::kInt}},
    {"level0_stop_writes_trigger",
     {offsetof(struct ColumnFamilyOptions, level0_stop_writes_trigger),
      OptionType::kInt}},
    {"max_bytes_for_level_multiplier",
     {offsetof(struct ColumnFamilyOptions, max_bytes_for_level_multiplier),
      OptionType::kInt}},
    {"max_grandparent_overlap_factor",
     {offsetof(struct ColumnFamilyOptions, max_grandparent_overlap_factor),
      OptionType::kInt}},
    {"max_mem_compaction_level",
     {offsetof(struct ColumnFamilyOptions, max_mem_compaction_level),
      OptionType::kInt}},
    {"max_write_buffer_number",
     {offsetof(struct ColumnFamilyOptions, max_write_buffer_number),
      OptionType::kInt}},
    {"max_write_buffer_number_to_maintain",
     {offsetof(struct ColumnFamilyOptions, max_write_buffer_number_to_maintain),
      OptionType::kInt}},
    {"min_write_buffer_number_to_merge",
     {offsetof(struct ColumnFamilyOptions, min_write_buffer_number_to_merge),
      OptionType::kInt}},
    {"num_levels",
     {offsetof(struct ColumnFamilyOptions, num_levels), OptionType::kInt}},
    {"source_compaction_factor",
     {offsetof(struct ColumnFamilyOptions, source_compaction_factor),
      OptionType::kInt}},
    {"target_file_size_multiplier",
     {offsetof(struct ColumnFamilyOptions, target_file_size_multiplier),
      OptionType::kInt}},
    {"arena_block_size",
     {offsetof(struct ColumnFamilyOptions, arena_block_size),
      OptionType::kSizeT}},
    {"inplace_update_num_locks",
     {offsetof(struct ColumnFamilyOptions, inplace_update_num_locks),
      OptionType::kSizeT}},
    {"max_successive_merges",
     {offsetof(struct ColumnFamilyOptions, max_successive_merges),
      OptionType::kSizeT}},
    {"memtable_prefix_bloom_huge_page_tlb_size",
     {offsetof(struct ColumnFamilyOptions,
               memtable_prefix_bloom_huge_page_tlb_size),
      OptionType::kSizeT}},
    {"write_buffer_size",
     {offsetof(struct ColumnFamilyOptions, write_buffer_size),
      OptionType::kSizeT}},
    {"bloom_locality",
     {offsetof(struct ColumnFamilyOptions, bloom_locality),
      OptionType::kUInt32T}},
    {"memtable_prefix_bloom_bits",
     {offsetof(struct ColumnFamilyOptions, memtable_prefix_bloom_bits),
      OptionType::kUInt32T}},
    {"memtable_prefix_bloom_probes",
     {offsetof(struct ColumnFamilyOptions, memtable_prefix_bloom_probes),
      OptionType::kUInt32T}},
    {"min_partial_merge_operands",
     {offsetof(struct ColumnFamilyOptions, min_partial_merge_operands),
      OptionType::kUInt32T}},
    {"max_bytes_for_level_base",
     {offsetof(struct ColumnFamilyOptions, max_bytes_for_level_base),
      OptionType::kUInt64T}},
    {"max_sequential_skip_in_iterations",
     {offsetof(struct ColumnFamilyOptions, max_sequential_skip_in_iterations),
      OptionType::kUInt64T}},
    {"target_file_size_base",
     {offsetof(struct ColumnFamilyOptions, target_file_size_base),
      OptionType::kUInt64T}},
    {"rate_limit_delay_max_milliseconds",
     {offsetof(struct ColumnFamilyOptions, rate_limit_delay_max_milliseconds),
      OptionType::kUInt}},
    {"compaction_style",
     {offsetof(struct ColumnFamilyOptions, compaction_style),
      OptionType::kCompactionStyle}}};

}  // namespace rocksdb
