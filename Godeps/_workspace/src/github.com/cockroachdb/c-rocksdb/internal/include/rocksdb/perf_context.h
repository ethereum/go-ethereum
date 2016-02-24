// Copyright (c) 2013, Facebook, Inc.  All rights reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.

#ifndef STORAGE_ROCKSDB_INCLUDE_PERF_CONTEXT_H
#define STORAGE_ROCKSDB_INCLUDE_PERF_CONTEXT_H

#include <stdint.h>
#include <string>

#include "rocksdb/perf_level.h"

namespace rocksdb {

// A thread local context for gathering performance counter efficiently
// and transparently.
// Use SetPerfLevel(PerfLevel::kEnableTime) to enable time stats.

struct PerfContext {

  void Reset(); // reset all performance counters to zero

  std::string ToString() const;

  uint64_t user_key_comparison_count; // total number of user key comparisons
  uint64_t block_cache_hit_count;     // total number of block cache hits
  uint64_t block_read_count;          // total number of block reads (with IO)
  uint64_t block_read_byte;           // total number of bytes from block reads
  uint64_t block_read_time;           // total time spent on block reads
  uint64_t block_checksum_time;       // total time spent on block checksum
  uint64_t block_decompress_time;     // total time spent on block decompression
  // total number of internal keys skipped over during iteration (overwritten or
  // deleted, to be more specific, hidden by a put or delete of the same key)
  uint64_t internal_key_skipped_count;
  // total number of deletes skipped over during iteration
  uint64_t internal_delete_skipped_count;

  uint64_t get_snapshot_time;          // total time spent on getting snapshot
  uint64_t get_from_memtable_time;     // total time spent on querying memtables
  uint64_t get_from_memtable_count;    // number of mem tables queried
  // total time spent after Get() finds a key
  uint64_t get_post_process_time;
  uint64_t get_from_output_files_time; // total time reading from output files
  // total time spent on seeking memtable
  uint64_t seek_on_memtable_time;
  // number of seeks issued on memtable
  uint64_t seek_on_memtable_count;
  // total time spent on seeking child iters
  uint64_t seek_child_seek_time;
  // number of seek issued in child iterators
  uint64_t seek_child_seek_count;
  uint64_t seek_min_heap_time;         // total time spent on the merge heap
  // total time spent on seeking the internal entries
  uint64_t seek_internal_seek_time;
  // total time spent on iterating internal entries to find the next user entry
  uint64_t find_next_user_entry_time;

  // total time spent on writing to WAL
  uint64_t write_wal_time;
  // total time spent on writing to mem tables
  uint64_t write_memtable_time;
  // total time spent on delaying write
  uint64_t write_delay_time;
  // total time spent on writing a record, excluding the above three times
  uint64_t write_pre_and_post_process_time;

  uint64_t db_mutex_lock_nanos;      // time spent on acquiring DB mutex.
  // Time spent on waiting with a condition variable created with DB mutex.
  uint64_t db_condition_wait_nanos;
  // Time spent on merge operator.
  uint64_t merge_operator_time_nanos;

  // Time spent on reading index block from block cache or SST file
  uint64_t read_index_block_nanos;
  // Time spent on reading filter block from block cache or SST file
  uint64_t read_filter_block_nanos;
  // Time spent on creating data block iterator
  uint64_t new_table_block_iter_nanos;
  // Time spent on creating a iterator of an SST file.
  uint64_t new_table_iterator_nanos;
  // Time spent on seeking a key in data/index blocks
  uint64_t block_seek_nanos;
  // Time spent on finding or creating a table reader
  uint64_t find_table_nanos;
};

#if defined(NPERF_CONTEXT) || defined(IOS_CROSS_COMPILE)
extern PerfContext perf_context;
#elif _WIN32
extern __declspec(thread) PerfContext perf_context;
#else
extern __thread PerfContext perf_context;
#endif

}

#endif
