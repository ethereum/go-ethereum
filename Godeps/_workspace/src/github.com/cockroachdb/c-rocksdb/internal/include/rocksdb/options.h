// Copyright (c) 2013, Facebook, Inc.  All rights reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.
// Copyright (c) 2011 The LevelDB Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file. See the AUTHORS file for names of contributors.

#ifndef STORAGE_ROCKSDB_INCLUDE_OPTIONS_H_
#define STORAGE_ROCKSDB_INCLUDE_OPTIONS_H_

#include <stddef.h>
#include <stdint.h>
#include <string>
#include <memory>
#include <vector>
#include <limits>
#include <unordered_map>

#include "rocksdb/version.h"
#include "rocksdb/listener.h"
#include "rocksdb/universal_compaction.h"

#ifdef max
#undef max
#endif

namespace rocksdb {

class Cache;
class CompactionFilter;
class CompactionFilterFactory;
class CompactionFilterFactoryV2;
class Comparator;
class Env;
enum InfoLogLevel : unsigned char;
class FilterPolicy;
class Logger;
class MergeOperator;
class Snapshot;
class TableFactory;
class MemTableRepFactory;
class TablePropertiesCollectorFactory;
class RateLimiter;
class DeleteScheduler;
class Slice;
class SliceTransform;
class Statistics;
class InternalKeyComparator;

// DB contents are stored in a set of blocks, each of which holds a
// sequence of key,value pairs.  Each block may be compressed before
// being stored in a file.  The following enum describes which
// compression method (if any) is used to compress a block.
enum CompressionType : char {
  // NOTE: do not change the values of existing entries, as these are
  // part of the persistent format on disk.
  kNoCompression = 0x0,
  kSnappyCompression = 0x1,
  kZlibCompression = 0x2,
  kBZip2Compression = 0x3,
  kLZ4Compression = 0x4,
  kLZ4HCCompression = 0x5,
  // zstd format is not finalized yet so it's subject to changes.
  kZSTDNotFinalCompression = 0x40,
};

enum CompactionStyle : char {
  // level based compaction style
  kCompactionStyleLevel = 0x0,
  // Universal compaction style
  // Not supported in ROCKSDB_LITE.
  kCompactionStyleUniversal = 0x1,
  // FIFO compaction style
  // Not supported in ROCKSDB_LITE
  kCompactionStyleFIFO = 0x2,
  // Disable background compaction. Compaction jobs are submitted
  // via CompactFiles().
  // Not supported in ROCKSDB_LITE
  kCompactionStyleNone = 0x3,
};

enum class WALRecoveryMode : char {
  // Original levelDB recovery
  // We tolerate incomplete record in trailing data on all logs
  // Use case : This is legacy behavior (default)
  kTolerateCorruptedTailRecords = 0x00,
  // Recover from clean shutdown
  // We don't expect to find any corruption in the WAL
  // Use case : This is ideal for unit tests and rare applications that
  // can require high consistency gaurantee
  kAbsoluteConsistency = 0x01,
  // Recover to point-in-time consistency
  // We stop the WAL playback on discovering WAL inconsistency
  // Use case : Ideal for systems that have disk controller cache like
  // hard disk, SSD without super capacitor that store related data
  kPointInTimeRecovery = 0x02,
  // Recovery after a disaster
  // We ignore any corruption in the  WAL and try to salvage as much data as
  // possible
  // Use case : Ideal for last ditch effort to recover data or systems that
  // operate with low grade unrelated data
  kSkipAnyCorruptedRecords = 0x03,
};

struct CompactionOptionsFIFO {
  // once the total sum of table files reaches this, we will delete the oldest
  // table file
  // Default: 1GB
  uint64_t max_table_files_size;

  CompactionOptionsFIFO() : max_table_files_size(1 * 1024 * 1024 * 1024) {}
};

// Compression options for different compression algorithms like Zlib
struct CompressionOptions {
  int window_bits;
  int level;
  int strategy;
  CompressionOptions() : window_bits(-14), level(-1), strategy(0) {}
  CompressionOptions(int wbits, int _lev, int _strategy)
      : window_bits(wbits), level(_lev), strategy(_strategy) {}
};

enum UpdateStatus {    // Return status For inplace update callback
  UPDATE_FAILED   = 0, // Nothing to update
  UPDATED_INPLACE = 1, // Value updated inplace
  UPDATED         = 2, // No inplace update. Merged value set
};

struct DbPath {
  std::string path;
  uint64_t target_size;  // Target size of total files under the path, in byte.

  DbPath() : target_size(0) {}
  DbPath(const std::string& p, uint64_t t) : path(p), target_size(t) {}
};

struct Options;

struct ColumnFamilyOptions {
  // Some functions that make it easier to optimize RocksDB

  // Use this if you don't need to keep the data sorted, i.e. you'll never use
  // an iterator, only Put() and Get() API calls
  //
  // Not supported in ROCKSDB_LITE
  ColumnFamilyOptions* OptimizeForPointLookup(
      uint64_t block_cache_size_mb);

  // Default values for some parameters in ColumnFamilyOptions are not
  // optimized for heavy workloads and big datasets, which means you might
  // observe write stalls under some conditions. As a starting point for tuning
  // RocksDB options, use the following two functions:
  // * OptimizeLevelStyleCompaction -- optimizes level style compaction
  // * OptimizeUniversalStyleCompaction -- optimizes universal style compaction
  // Universal style compaction is focused on reducing Write Amplification
  // Factor for big data sets, but increases Space Amplification. You can learn
  // more about the different styles here:
  // https://github.com/facebook/rocksdb/wiki/Rocksdb-Architecture-Guide
  // Make sure to also call IncreaseParallelism(), which will provide the
  // biggest performance gains.
  // Note: we might use more memory than memtable_memory_budget during high
  // write rate period
  //
  // OptimizeUniversalStyleCompaction is not supported in ROCKSDB_LITE
  ColumnFamilyOptions* OptimizeLevelStyleCompaction(
      uint64_t memtable_memory_budget = 512 * 1024 * 1024);
  ColumnFamilyOptions* OptimizeUniversalStyleCompaction(
      uint64_t memtable_memory_budget = 512 * 1024 * 1024);

  // -------------------
  // Parameters that affect behavior

  // Comparator used to define the order of keys in the table.
  // Default: a comparator that uses lexicographic byte-wise ordering
  //
  // REQUIRES: The client must ensure that the comparator supplied
  // here has the same name and orders keys *exactly* the same as the
  // comparator provided to previous open calls on the same DB.
  const Comparator* comparator;

  // REQUIRES: The client must provide a merge operator if Merge operation
  // needs to be accessed. Calling Merge on a DB without a merge operator
  // would result in Status::NotSupported. The client must ensure that the
  // merge operator supplied here has the same name and *exactly* the same
  // semantics as the merge operator provided to previous open calls on
  // the same DB. The only exception is reserved for upgrade, where a DB
  // previously without a merge operator is introduced to Merge operation
  // for the first time. It's necessary to specify a merge operator when
  // openning the DB in this case.
  // Default: nullptr
  std::shared_ptr<MergeOperator> merge_operator;

  // A single CompactionFilter instance to call into during compaction.
  // Allows an application to modify/delete a key-value during background
  // compaction.
  //
  // If the client requires a new compaction filter to be used for different
  // compaction runs, it can specify compaction_filter_factory instead of this
  // option.  The client should specify only one of the two.
  // compaction_filter takes precedence over compaction_filter_factory if
  // client specifies both.
  //
  // If multithreaded compaction is being used, the supplied CompactionFilter
  // instance may be used from different threads concurrently and so should be
  // thread-safe.
  //
  // Default: nullptr
  const CompactionFilter* compaction_filter;

  // This is a factory that provides compaction filter objects which allow
  // an application to modify/delete a key-value during background compaction.
  //
  // A new filter will be created on each compaction run.  If multithreaded
  // compaction is being used, each created CompactionFilter will only be used
  // from a single thread and so does not need to be thread-safe.
  //
  // Default: nullptr
  std::shared_ptr<CompactionFilterFactory> compaction_filter_factory;

  // This is deprecated. Talk to us if you depend on
  // compaction_filter_factory_v2 and we'll put it back
  // std::shared_ptr<CompactionFilterFactoryV2> compaction_filter_factory_v2;

  // -------------------
  // Parameters that affect performance

  // Amount of data to build up in memory (backed by an unsorted log
  // on disk) before converting to a sorted on-disk file.
  //
  // Larger values increase performance, especially during bulk loads.
  // Up to max_write_buffer_number write buffers may be held in memory
  // at the same time,
  // so you may wish to adjust this parameter to control memory usage.
  // Also, a larger write buffer will result in a longer recovery time
  // the next time the database is opened.
  //
  // Note that write_buffer_size is enforced per column family.
  // See db_write_buffer_size for sharing memory across column families.
  //
  // Default: 4MB
  //
  // Dynamically changeable through SetOptions() API
  size_t write_buffer_size;

  // The maximum number of write buffers that are built up in memory.
  // The default and the minimum number is 2, so that when 1 write buffer
  // is being flushed to storage, new writes can continue to the other
  // write buffer.
  //
  // Default: 2
  //
  // Dynamically changeable through SetOptions() API
  int max_write_buffer_number;

  // The minimum number of write buffers that will be merged together
  // before writing to storage.  If set to 1, then
  // all write buffers are fushed to L0 as individual files and this increases
  // read amplification because a get request has to check in all of these
  // files. Also, an in-memory merge may result in writing lesser
  // data to storage if there are duplicate records in each of these
  // individual write buffers.  Default: 1
  int min_write_buffer_number_to_merge;

  // The total maximum number of write buffers to maintain in memory including
  // copies of buffers that have already been flushed.  Unlike
  // max_write_buffer_number, this parameter does not affect flushing.
  // This controls the minimum amount of write history that will be available
  // in memory for conflict checking when Transactions are used.
  // If this value is too low, some transactions may fail at commit time due
  // to not being able to determine whether there were any write conflicts.
  //
  // Setting this value to 0 will cause write buffers to be freed immediately
  // after they are flushed.
  // If this value is set to -1, 'max_write_buffer_number' will be used.
  //
  // Default:
  // If using a TransactionDB/OptimisticTransactionDB, the default value will
  // be set to the value of 'max_write_buffer_number' if it is not explicitly
  // set by the user.  Otherwise, the default is 0.
  int max_write_buffer_number_to_maintain;

  // Compress blocks using the specified compression algorithm.  This
  // parameter can be changed dynamically.
  //
  // Default: kSnappyCompression, if it's supported. If snappy is not linked
  // with the library, the default is kNoCompression.
  //
  // Typical speeds of kSnappyCompression on an Intel(R) Core(TM)2 2.4GHz:
  //    ~200-500MB/s compression
  //    ~400-800MB/s decompression
  // Note that these speeds are significantly faster than most
  // persistent storage speeds, and therefore it is typically never
  // worth switching to kNoCompression.  Even if the input data is
  // incompressible, the kSnappyCompression implementation will
  // efficiently detect that and will switch to uncompressed mode.
  CompressionType compression;

  // Different levels can have different compression policies. There
  // are cases where most lower levels would like to use quick compression
  // algorithms while the higher levels (which have more data) use
  // compression algorithms that have better compression but could
  // be slower. This array, if non-empty, should have an entry for
  // each level of the database; these override the value specified in
  // the previous field 'compression'.
  //
  // NOTICE if level_compaction_dynamic_level_bytes=true,
  // compression_per_level[0] still determines L0, but other elements
  // of the array are based on base level (the level L0 files are merged
  // to), and may not match the level users see from info log for metadata.
  // If L0 files are merged to level-n, then, for i>0, compression_per_level[i]
  // determines compaction type for level n+i-1.
  // For example, if we have three 5 levels, and we determine to merge L0
  // data to L4 (which means L1..L3 will be empty), then the new files go to
  // L4 uses compression type compression_per_level[1].
  // If now L0 is merged to L2. Data goes to L2 will be compressed
  // according to compression_per_level[1], L3 using compression_per_level[2]
  // and L4 using compression_per_level[3]. Compaction for each level can
  // change when data grows.
  std::vector<CompressionType> compression_per_level;

  // different options for compression algorithms
  CompressionOptions compression_opts;

  // If non-nullptr, use the specified function to determine the
  // prefixes for keys.  These prefixes will be placed in the filter.
  // Depending on the workload, this can reduce the number of read-IOP
  // cost for scans when a prefix is passed via ReadOptions to
  // db.NewIterator().  For prefix filtering to work properly,
  // "prefix_extractor" and "comparator" must be such that the following
  // properties hold:
  //
  // 1) key.starts_with(prefix(key))
  // 2) Compare(prefix(key), key) <= 0.
  // 3) If Compare(k1, k2) <= 0, then Compare(prefix(k1), prefix(k2)) <= 0
  // 4) prefix(prefix(key)) == prefix(key)
  //
  // Default: nullptr
  std::shared_ptr<const SliceTransform> prefix_extractor;

  // Number of levels for this database
  int num_levels;

  // Number of files to trigger level-0 compaction. A value <0 means that
  // level-0 compaction will not be triggered by number of files at all.
  //
  // Default: 4
  //
  // Dynamically changeable through SetOptions() API
  int level0_file_num_compaction_trigger;

  // Soft limit on number of level-0 files. We start slowing down writes at this
  // point. A value <0 means that no writing slow down will be triggered by
  // number of files in level-0.
  //
  // Dynamically changeable through SetOptions() API
  int level0_slowdown_writes_trigger;

  // Maximum number of level-0 files.  We stop writes at this point.
  //
  // Dynamically changeable through SetOptions() API
  int level0_stop_writes_trigger;

  // This does not do anything anymore. Deprecated.
  int max_mem_compaction_level;

  // Target file size for compaction.
  // target_file_size_base is per-file size for level-1.
  // Target file size for level L can be calculated by
  // target_file_size_base * (target_file_size_multiplier ^ (L-1))
  // For example, if target_file_size_base is 2MB and
  // target_file_size_multiplier is 10, then each file on level-1 will
  // be 2MB, and each file on level 2 will be 20MB,
  // and each file on level-3 will be 200MB.
  //
  // Default: 2MB.
  //
  // Dynamically changeable through SetOptions() API
  uint64_t target_file_size_base;

  // By default target_file_size_multiplier is 1, which means
  // by default files in different levels will have similar size.
  //
  // Dynamically changeable through SetOptions() API
  int target_file_size_multiplier;

  // Control maximum total data size for a level.
  // max_bytes_for_level_base is the max total for level-1.
  // Maximum number of bytes for level L can be calculated as
  // (max_bytes_for_level_base) * (max_bytes_for_level_multiplier ^ (L-1))
  // For example, if max_bytes_for_level_base is 20MB, and if
  // max_bytes_for_level_multiplier is 10, total data size for level-1
  // will be 20MB, total file size for level-2 will be 200MB,
  // and total file size for level-3 will be 2GB.
  //
  // Default: 10MB.
  //
  // Dynamically changeable through SetOptions() API
  uint64_t max_bytes_for_level_base;

  // If true, RocksDB will pick target size of each level dynamically.
  // We will pick a base level b >= 1. L0 will be directly merged into level b,
  // instead of always into level 1. Level 1 to b-1 need to be empty.
  // We try to pick b and its target size so that
  // 1. target size is in the range of
  //   (max_bytes_for_level_base / max_bytes_for_level_multiplier,
  //    max_bytes_for_level_base]
  // 2. target size of the last level (level num_levels-1) equals to extra size
  //    of the level.
  // At the same time max_bytes_for_level_multiplier and
  // max_bytes_for_level_multiplier_additional are still satisfied.
  //
  // With this option on, from an empty DB, we make last level the base level,
  // which means merging L0 data into the last level, until it exceeds
  // max_bytes_for_level_base. And then we make the second last level to be
  // base level, to start to merge L0 data to second last level, with its
  // target size to be 1/max_bytes_for_level_multiplier of the last level's
  // extra size. After the data accumulates more so that we need to move the
  // base level to the third last one, and so on.
  //
  // For example, assume max_bytes_for_level_multiplier=10, num_levels=6,
  // and max_bytes_for_level_base=10MB.
  // Target sizes of level 1 to 5 starts with:
  // [- - - - 10MB]
  // with base level is level. Target sizes of level 1 to 4 are not applicable
  // because they will not be used.
  // Until the size of Level 5 grows to more than 10MB, say 11MB, we make
  // base target to level 4 and now the targets looks like:
  // [- - - 1.1MB 11MB]
  // While data are accumulated, size targets are tuned based on actual data
  // of level 5. When level 5 has 50MB of data, the target is like:
  // [- - - 5MB 50MB]
  // Until level 5's actual size is more than 100MB, say 101MB. Now if we keep
  // level 4 to be the base level, its target size needs to be 10.1MB, which
  // doesn't satisfy the target size range. So now we make level 3 the target
  // size and the target sizes of the levels look like:
  // [- - 1.01MB 10.1MB 101MB]
  // In the same way, while level 5 further grows, all levels' targets grow,
  // like
  // [- - 5MB 50MB 500MB]
  // Until level 5 exceeds 1000MB and becomes 1001MB, we make level 2 the
  // base level and make levels' target sizes like this:
  // [- 1.001MB 10.01MB 100.1MB 1001MB]
  // and go on...
  //
  // By doing it, we give max_bytes_for_level_multiplier a priority against
  // max_bytes_for_level_base, for a more predictable LSM tree shape. It is
  // useful to limit worse case space amplification.
  //
  // max_bytes_for_level_multiplier_additional is ignored with this flag on.
  //
  // Turning this feature on or off for an existing DB can cause unexpected
  // LSM tree structure so it's not recommended.
  //
  // NOTE: this option is experimental
  //
  // Default: false
  bool level_compaction_dynamic_level_bytes;

  // Default: 10.
  //
  // Dynamically changeable through SetOptions() API
  int max_bytes_for_level_multiplier;

  // Different max-size multipliers for different levels.
  // These are multiplied by max_bytes_for_level_multiplier to arrive
  // at the max-size of each level.
  //
  // Default: 1
  //
  // Dynamically changeable through SetOptions() API
  std::vector<int> max_bytes_for_level_multiplier_additional;

  // Maximum number of bytes in all compacted files.  We avoid expanding
  // the lower level file set of a compaction if it would make the
  // total compaction cover more than
  // (expanded_compaction_factor * targetFileSizeLevel()) many bytes.
  //
  // Dynamically changeable through SetOptions() API
  int expanded_compaction_factor;

  // Maximum number of bytes in all source files to be compacted in a
  // single compaction run. We avoid picking too many files in the
  // source level so that we do not exceed the total source bytes
  // for compaction to exceed
  // (source_compaction_factor * targetFileSizeLevel()) many bytes.
  // Default:1, i.e. pick maxfilesize amount of data as the source of
  // a compaction.
  //
  // Dynamically changeable through SetOptions() API
  int source_compaction_factor;

  // Control maximum bytes of overlaps in grandparent (i.e., level+2) before we
  // stop building a single file in a level->level+1 compaction.
  //
  // Dynamically changeable through SetOptions() API
  int max_grandparent_overlap_factor;

  // Puts are delayed to options.delayed_write_rate when any level has a
  // compaction score that exceeds soft_rate_limit. This is ignored when == 0.0.
  // CONSTRAINT: soft_rate_limit <= hard_rate_limit. If this constraint does not
  // hold, RocksDB will set soft_rate_limit = hard_rate_limit
  //
  // Default: 0 (disabled)
  //
  // Dynamically changeable through SetOptions() API
  double soft_rate_limit;

  // DEPRECATED -- this options is no longer usde
  double hard_rate_limit;

  // DEPRECATED -- this options is no longer used
  unsigned int rate_limit_delay_max_milliseconds;

  // size of one block in arena memory allocation.
  // If <= 0, a proper value is automatically calculated (usually 1/8 of
  // writer_buffer_size, rounded up to a multiple of 4KB).
  //
  // There are two additonal restriction of the The specified size:
  // (1) size should be in the range of [4096, 2 << 30] and
  // (2) be the multiple of the CPU word (which helps with the memory
  // alignment).
  //
  // We'll automatically check and adjust the size number to make sure it
  // conforms to the restrictions.
  //
  // Default: 0
  //
  // Dynamically changeable through SetOptions() API
  size_t arena_block_size;

  // Disable automatic compactions. Manual compactions can still
  // be issued on this column family
  //
  // Dynamically changeable through SetOptions() API
  bool disable_auto_compactions;

  // DEPREACTED
  // Does not have any effect.
  bool purge_redundant_kvs_while_flush;

  // The compaction style. Default: kCompactionStyleLevel
  CompactionStyle compaction_style;

  // If true, compaction will verify checksum on every read that happens
  // as part of compaction
  //
  // Default: true
  //
  // Dynamically changeable through SetOptions() API
  bool verify_checksums_in_compaction;

  // The options needed to support Universal Style compactions
  CompactionOptionsUniversal compaction_options_universal;

  // The options for FIFO compaction style
  CompactionOptionsFIFO compaction_options_fifo;

  // Use KeyMayExist API to filter deletes when this is true.
  // If KeyMayExist returns false, i.e. the key definitely does not exist, then
  // the delete is a noop. KeyMayExist only incurs in-memory look up.
  // This optimization avoids writing the delete to storage when appropriate.
  //
  // Default: false
  //
  // Dynamically changeable through SetOptions() API
  bool filter_deletes;

  // An iteration->Next() sequentially skips over keys with the same
  // user-key unless this option is set. This number specifies the number
  // of keys (with the same userkey) that will be sequentially
  // skipped before a reseek is issued.
  //
  // Default: 8
  //
  // Dynamically changeable through SetOptions() API
  uint64_t max_sequential_skip_in_iterations;

  // This is a factory that provides MemTableRep objects.
  // Default: a factory that provides a skip-list-based implementation of
  // MemTableRep.
  std::shared_ptr<MemTableRepFactory> memtable_factory;

  // This is a factory that provides TableFactory objects.
  // Default: a block-based table factory that provides a default
  // implementation of TableBuilder and TableReader with default
  // BlockBasedTableOptions.
  std::shared_ptr<TableFactory> table_factory;

  // Block-based table related options are moved to BlockBasedTableOptions.
  // Related options that were originally here but now moved include:
  //   no_block_cache
  //   block_cache
  //   block_cache_compressed
  //   block_size
  //   block_size_deviation
  //   block_restart_interval
  //   filter_policy
  //   whole_key_filtering
  // If you'd like to customize some of these options, you will need to
  // use NewBlockBasedTableFactory() to construct a new table factory.

  // This option allows user to to collect their own interested statistics of
  // the tables.
  // Default: empty vector -- no user-defined statistics collection will be
  // performed.
  typedef std::vector<std::shared_ptr<TablePropertiesCollectorFactory>>
      TablePropertiesCollectorFactories;
  TablePropertiesCollectorFactories table_properties_collector_factories;

  // Allows thread-safe inplace updates. If this is true, there is no way to
  // achieve point-in-time consistency using snapshot or iterator (assuming
  // concurrent updates). Hence iterator and multi-get will return results
  // which are not consistent as of any point-in-time.
  // If inplace_callback function is not set,
  //   Put(key, new_value) will update inplace the existing_value iff
  //   * key exists in current memtable
  //   * new sizeof(new_value) <= sizeof(existing_value)
  //   * existing_value for that key is a put i.e. kTypeValue
  // If inplace_callback function is set, check doc for inplace_callback.
  // Default: false.
  bool inplace_update_support;

  // Number of locks used for inplace update
  // Default: 10000, if inplace_update_support = true, else 0.
  //
  // Dynamically changeable through SetOptions() API
  size_t inplace_update_num_locks;

  // existing_value - pointer to previous value (from both memtable and sst).
  //                  nullptr if key doesn't exist
  // existing_value_size - pointer to size of existing_value).
  //                       nullptr if key doesn't exist
  // delta_value - Delta value to be merged with the existing_value.
  //               Stored in transaction logs.
  // merged_value - Set when delta is applied on the previous value.

  // Applicable only when inplace_update_support is true,
  // this callback function is called at the time of updating the memtable
  // as part of a Put operation, lets say Put(key, delta_value). It allows the
  // 'delta_value' specified as part of the Put operation to be merged with
  // an 'existing_value' of the key in the database.

  // If the merged value is smaller in size that the 'existing_value',
  // then this function can update the 'existing_value' buffer inplace and
  // the corresponding 'existing_value'_size pointer, if it wishes to.
  // The callback should return UpdateStatus::UPDATED_INPLACE.
  // In this case. (In this case, the snapshot-semantics of the rocksdb
  // Iterator is not atomic anymore).

  // If the merged value is larger in size than the 'existing_value' or the
  // application does not wish to modify the 'existing_value' buffer inplace,
  // then the merged value should be returned via *merge_value. It is set by
  // merging the 'existing_value' and the Put 'delta_value'. The callback should
  // return UpdateStatus::UPDATED in this case. This merged value will be added
  // to the memtable.

  // If merging fails or the application does not wish to take any action,
  // then the callback should return UpdateStatus::UPDATE_FAILED.

  // Please remember that the original call from the application is Put(key,
  // delta_value). So the transaction log (if enabled) will still contain (key,
  // delta_value). The 'merged_value' is not stored in the transaction log.
  // Hence the inplace_callback function should be consistent across db reopens.

  // Default: nullptr
  UpdateStatus (*inplace_callback)(char* existing_value,
                                   uint32_t* existing_value_size,
                                   Slice delta_value,
                                   std::string* merged_value);

  // if prefix_extractor is set and bloom_bits is not 0, create prefix bloom
  // for memtable
  //
  // Dynamically changeable through SetOptions() API
  uint32_t memtable_prefix_bloom_bits;

  // number of hash probes per key
  //
  // Dynamically changeable through SetOptions() API
  uint32_t memtable_prefix_bloom_probes;

  // Page size for huge page TLB for bloom in memtable. If <=0, not allocate
  // from huge page TLB but from malloc.
  // Need to reserve huge pages for it to be allocated. For example:
  //      sysctl -w vm.nr_hugepages=20
  // See linux doc Documentation/vm/hugetlbpage.txt
  //
  // Dynamically changeable through SetOptions() API
  size_t memtable_prefix_bloom_huge_page_tlb_size;

  // Control locality of bloom filter probes to improve cache miss rate.
  // This option only applies to memtable prefix bloom and plaintable
  // prefix bloom. It essentially limits every bloom checking to one cache line.
  // This optimization is turned off when set to 0, and positive number to turn
  // it on.
  // Default: 0
  uint32_t bloom_locality;

  // Maximum number of successive merge operations on a key in the memtable.
  //
  // When a merge operation is added to the memtable and the maximum number of
  // successive merges is reached, the value of the key will be calculated and
  // inserted into the memtable instead of the merge operation. This will
  // ensure that there are never more than max_successive_merges merge
  // operations in the memtable.
  //
  // Default: 0 (disabled)
  //
  // Dynamically changeable through SetOptions() API
  size_t max_successive_merges;

  // The number of partial merge operands to accumulate before partial
  // merge will be performed. Partial merge will not be called
  // if the list of values to merge is less than min_partial_merge_operands.
  //
  // If min_partial_merge_operands < 2, then it will be treated as 2.
  //
  // Default: 2
  uint32_t min_partial_merge_operands;

  // This flag specifies that the implementation should optimize the filters
  // mainly for cases where keys are found rather than also optimize for keys
  // missed. This would be used in cases where the application knows that
  // there are very few misses or the performance in the case of misses is not
  // important.
  //
  // For now, this flag allows us to not store filters for the last level i.e
  // the largest level which contains data of the LSM store. For keys which
  // are hits, the filters in this level are not useful because we will search
  // for the data anyway. NOTE: the filters in other levels are still useful
  // even for key hit because they tell us whether to look in that level or go
  // to the higher level.
  //
  // Default: false
  bool optimize_filters_for_hits;

  // After writing every SST file, reopen it and read all the keys.
  // Default: false
  bool paranoid_file_checks;

  // Measure IO stats in compactions, if true.
  // Default: false
  bool compaction_measure_io_stats;

  // Create ColumnFamilyOptions with default values for all fields
  ColumnFamilyOptions();
  // Create ColumnFamilyOptions from Options
  explicit ColumnFamilyOptions(const Options& options);

  void Dump(Logger* log) const;
};

struct DBOptions {
  // Some functions that make it easier to optimize RocksDB

#ifndef ROCKSDB_LITE
  // By default, RocksDB uses only one background thread for flush and
  // compaction. Calling this function will set it up such that total of
  // `total_threads` is used. Good value for `total_threads` is the number of
  // cores. You almost definitely want to call this function if your system is
  // bottlenecked by RocksDB.
  DBOptions* IncreaseParallelism(int total_threads = 16);
#endif  // ROCKSDB_LITE

  // If true, the database will be created if it is missing.
  // Default: false
  bool create_if_missing;

  // If true, missing column families will be automatically created.
  // Default: false
  bool create_missing_column_families;

  // If true, an error is raised if the database already exists.
  // Default: false
  bool error_if_exists;

  // If true, RocksDB will aggressively check consistency of the data.
  // Also, if any of the  writes to the database fails (Put, Delete, Merge,
  // Write), the database will switch to read-only mode and fail all other
  // Write operations.
  // In most cases you want this to be set to true.
  // Default: true
  bool paranoid_checks;

  // Use the specified object to interact with the environment,
  // e.g. to read/write files, schedule background work, etc.
  // Default: Env::Default()
  Env* env;

  // Use to control write rate of flush and compaction. Flush has higher
  // priority than compaction. Rate limiting is disabled if nullptr.
  // If rate limiter is enabled, bytes_per_sync is set to 1MB by default.
  // Default: nullptr
  std::shared_ptr<RateLimiter> rate_limiter;

  // Use to control files deletion rate, can be used among multiple
  // RocksDB instances. delete_scheduler is only used to delete table files that
  // need to be deleted from the first db_path (db_name if db_paths is empty),
  // other files types and other db_paths wont be affected by delete_scheduler.
  // Default: nullptr (disabled)
  std::shared_ptr<DeleteScheduler> delete_scheduler;

  // Any internal progress/error information generated by the db will
  // be written to info_log if it is non-nullptr, or to a file stored
  // in the same directory as the DB contents if info_log is nullptr.
  // Default: nullptr
  std::shared_ptr<Logger> info_log;

  InfoLogLevel info_log_level;

  // Number of open files that can be used by the DB.  You may need to
  // increase this if your database has a large working set. Value -1 means
  // files opened are always kept open. You can estimate number of files based
  // on target_file_size_base and target_file_size_multiplier for level-based
  // compaction. For universal-style compaction, you can usually set it to -1.
  // Default: 5000 or ulimit value of max open files (whichever is smaller)
  int max_open_files;

  // If max_open_files is -1, DB will open all files on DB::Open(). You can
  // use this option to increase the number of threads used to open the files.
  // Default: 1
  int max_file_opening_threads;

  // Once write-ahead logs exceed this size, we will start forcing the flush of
  // column families whose memtables are backed by the oldest live WAL file
  // (i.e. the ones that are causing all the space amplification). If set to 0
  // (default), we will dynamically choose the WAL size limit to be
  // [sum of all write_buffer_size * max_write_buffer_number] * 4
  // Default: 0
  uint64_t max_total_wal_size;

  // If non-null, then we should collect metrics about database operations
  // Statistics objects should not be shared between DB instances as
  // it does not use any locks to prevent concurrent updates.
  std::shared_ptr<Statistics> statistics;

  // If true, then the contents of manifest and data files are not synced
  // to stable storage. Their contents remain in the OS buffers till the
  // OS decides to flush them. This option is good for bulk-loading
  // of data. Once the bulk-loading is complete, please issue a
  // sync to the OS to flush all dirty buffesrs to stable storage.
  // Default: false
  bool disableDataSync;

  // If true, then every store to stable storage will issue a fsync.
  // If false, then every store to stable storage will issue a fdatasync.
  // This parameter should be set to true while storing data to
  // filesystem like ext3 that can lose files after a reboot.
  // Default: false
  bool use_fsync;

  // A list of paths where SST files can be put into, with its target size.
  // Newer data is placed into paths specified earlier in the vector while
  // older data gradually moves to paths specified later in the vector.
  //
  // For example, you have a flash device with 10GB allocated for the DB,
  // as well as a hard drive of 2TB, you should config it to be:
  //   [{"/flash_path", 10GB}, {"/hard_drive", 2TB}]
  //
  // The system will try to guarantee data under each path is close to but
  // not larger than the target size. But current and future file sizes used
  // by determining where to place a file are based on best-effort estimation,
  // which means there is a chance that the actual size under the directory
  // is slightly more than target size under some workloads. User should give
  // some buffer room for those cases.
  //
  // If none of the paths has sufficient room to place a file, the file will
  // be placed to the last path anyway, despite to the target size.
  //
  // Placing newer data to ealier paths is also best-efforts. User should
  // expect user files to be placed in higher levels in some extreme cases.
  //
  // If left empty, only one path will be used, which is db_name passed when
  // opening the DB.
  // Default: empty
  std::vector<DbPath> db_paths;

  // This specifies the info LOG dir.
  // If it is empty, the log files will be in the same dir as data.
  // If it is non empty, the log files will be in the specified dir,
  // and the db data dir's absolute path will be used as the log file
  // name's prefix.
  std::string db_log_dir;

  // This specifies the absolute dir path for write-ahead logs (WAL).
  // If it is empty, the log files will be in the same dir as data,
  //   dbname is used as the data dir by default
  // If it is non empty, the log files will be in kept the specified dir.
  // When destroying the db,
  //   all log files in wal_dir and the dir itself is deleted
  std::string wal_dir;

  // The periodicity when obsolete files get deleted. The default
  // value is 6 hours. The files that get out of scope by compaction
  // process will still get automatically delete on every compaction,
  // regardless of this setting
  uint64_t delete_obsolete_files_period_micros;

  // Maximum number of concurrent background compaction jobs, submitted to
  // the default LOW priority thread pool.
  // If you're increasing this, also consider increasing number of threads in
  // LOW priority thread pool. For more information, see
  // Env::SetBackgroundThreads
  // Default: 1
  int max_background_compactions;

  // This integer represents the maximum number of threads that will
  // concurrently perform a compaction job by breaking it into multiple,
  // smaller ones that are run simultaneously.
  // Default: 1 (i.e. no subcompactions)
  uint32_t max_subcompactions;

  // Maximum number of concurrent background memtable flush jobs, submitted to
  // the HIGH priority thread pool.
  //
  // By default, all background jobs (major compaction and memtable flush) go
  // to the LOW priority pool. If this option is set to a positive number,
  // memtable flush jobs will be submitted to the HIGH priority pool.
  // It is important when the same Env is shared by multiple db instances.
  // Without a separate pool, long running major compaction jobs could
  // potentially block memtable flush jobs of other db instances, leading to
  // unnecessary Put stalls.
  //
  // If you're increasing this, also consider increasing number of threads in
  // HIGH priority thread pool. For more information, see
  // Env::SetBackgroundThreads
  // Default: 1
  int max_background_flushes;

  // Specify the maximal size of the info log file. If the log file
  // is larger than `max_log_file_size`, a new info log file will
  // be created.
  // If max_log_file_size == 0, all logs will be written to one
  // log file.
  size_t max_log_file_size;

  // Time for the info log file to roll (in seconds).
  // If specified with non-zero value, log file will be rolled
  // if it has been active longer than `log_file_time_to_roll`.
  // Default: 0 (disabled)
  size_t log_file_time_to_roll;

  // Maximal info log files to be kept.
  // Default: 1000
  size_t keep_log_file_num;

  // manifest file is rolled over on reaching this limit.
  // The older manifest file be deleted.
  // The default value is MAX_INT so that roll-over does not take place.
  uint64_t max_manifest_file_size;

  // Number of shards used for table cache.
  int table_cache_numshardbits;

  // DEPRECATED
  // int table_cache_remove_scan_count_limit;

  // The following two fields affect how archived logs will be deleted.
  // 1. If both set to 0, logs will be deleted asap and will not get into
  //    the archive.
  // 2. If WAL_ttl_seconds is 0 and WAL_size_limit_MB is not 0,
  //    WAL files will be checked every 10 min and if total size is greater
  //    then WAL_size_limit_MB, they will be deleted starting with the
  //    earliest until size_limit is met. All empty files will be deleted.
  // 3. If WAL_ttl_seconds is not 0 and WAL_size_limit_MB is 0, then
  //    WAL files will be checked every WAL_ttl_secondsi / 2 and those that
  //    are older than WAL_ttl_seconds will be deleted.
  // 4. If both are not 0, WAL files will be checked every 10 min and both
  //    checks will be performed with ttl being first.
  uint64_t WAL_ttl_seconds;
  uint64_t WAL_size_limit_MB;

  // Number of bytes to preallocate (via fallocate) the manifest
  // files.  Default is 4mb, which is reasonable to reduce random IO
  // as well as prevent overallocation for mounts that preallocate
  // large amounts of data (such as xfs's allocsize option).
  size_t manifest_preallocation_size;

  // Data being read from file storage may be buffered in the OS
  // Default: true
  bool allow_os_buffer;

  // Allow the OS to mmap file for reading sst tables. Default: false
  bool allow_mmap_reads;

  // Allow the OS to mmap file for writing.
  // DB::SyncWAL() only works if this is set to false.
  // Default: false
  bool allow_mmap_writes;

  // Disable child process inherit open files. Default: true
  bool is_fd_close_on_exec;

  // DEPRECATED -- this options is no longer used
  bool skip_log_error_on_recovery;

  // if not zero, dump rocksdb.stats to LOG every stats_dump_period_sec
  // Default: 600 (10 min)
  unsigned int stats_dump_period_sec;

  // If set true, will hint the underlying file system that the file
  // access pattern is random, when a sst file is opened.
  // Default: true
  bool advise_random_on_open;

  // Amount of data to build up in memtables across all column
  // families before writing to disk.
  //
  // This is distinct from write_buffer_size, which enforces a limit
  // for a single memtable.
  //
  // This feature is disabled by default. Specify a non-zero value
  // to enable it.
  //
  // Default: 0 (disabled)
  size_t db_write_buffer_size;

  // Specify the file access pattern once a compaction is started.
  // It will be applied to all input files of a compaction.
  // Default: NORMAL
  enum AccessHint {
      NONE,
      NORMAL,
      SEQUENTIAL,
      WILLNEED
  };
  AccessHint access_hint_on_compaction_start;

  // If true, always create a new file descriptor and new table reader
  // for compaction inputs. Turn this parameter on may introduce extra
  // memory usage in the table reader, if it allocates extra memory
  // for indexes. This will allow file descriptor prefetch options
  // to be set for compaction input files and not to impact file
  // descriptors for the same file used by user queries.
  // Suggest to enable BlockBasedTableOptions.cache_index_and_filter_blocks
  // for this mode if using block-based table.
  //
  // Default: false
  bool new_table_reader_for_compaction_inputs;

  // If non-zero, we perform bigger reads when doing compaction. If you're
  // running RocksDB on spinning disks, you should set this to at least 2MB.
  // That way RocksDB's compaction is doing sequential instead of random reads.
  //
  // When non-zero, we also force new_table_reader_for_compaction_inputs to
  // true.
  //
  // Default: 0
  size_t compaction_readahead_size;

  // Use adaptive mutex, which spins in the user space before resorting
  // to kernel. This could reduce context switch when the mutex is not
  // heavily contended. However, if the mutex is hot, we could end up
  // wasting spin time.
  // Default: false
  bool use_adaptive_mutex;

  // Create DBOptions with default values for all fields
  DBOptions();
  // Create DBOptions from Options
  explicit DBOptions(const Options& options);

  void Dump(Logger* log) const;

  // Allows OS to incrementally sync files to disk while they are being
  // written, asynchronously, in the background. This operation can be used
  // to smooth out write I/Os over time. Users shouldn't reply on it for
  // persistency guarantee.
  // Issue one request for every bytes_per_sync written. 0 turns it off.
  // Default: 0
  //
  // You may consider using rate_limiter to regulate write rate to device.
  // When rate limiter is enabled, it automatically enables bytes_per_sync
  // to 1MB.
  //
  // This option applies to table files
  uint64_t bytes_per_sync;

  // Same as bytes_per_sync, but applies to WAL files
  // Default: 0, turned off
  uint64_t wal_bytes_per_sync;

  // A vector of EventListeners which call-back functions will be called
  // when specific RocksDB event happens.
  std::vector<std::shared_ptr<EventListener>> listeners;

  // If true, then the status of the threads involved in this DB will
  // be tracked and available via GetThreadList() API.
  //
  // Default: false
  bool enable_thread_tracking;

  // The limited write rate to DB if soft_rate_limit or
  // level0_slowdown_writes_trigger is triggered. It is calcualted using
  // size of user write requests before compression.
  // Unit: byte per second.
  //
  // Default: 1MB/s
  uint64_t delayed_write_rate;

  // If true, then DB::Open() will not update the statistics used to optimize
  // compaction decision by loading table properties from many files.
  // Turning off this feature will improve DBOpen time espcially in
  // disk environment.
  //
  // Default: false
  bool skip_stats_update_on_db_open;

  // Recovery mode to control the consistency while replaying WAL
  // Default: kTolerateCorruptedTailRecords
  WALRecoveryMode wal_recovery_mode;

  // A global cache for table-level rows.
  // Default: nullptr (disabled)
  // Not supported in ROCKSDB_LITE mode!
  std::shared_ptr<Cache> row_cache;
};

// Options to control the behavior of a database (passed to DB::Open)
struct Options : public DBOptions, public ColumnFamilyOptions {
  // Create an Options object with default values for all fields.
  Options() :
    DBOptions(),
    ColumnFamilyOptions() {}

  Options(const DBOptions& db_options,
          const ColumnFamilyOptions& column_family_options)
      : DBOptions(db_options), ColumnFamilyOptions(column_family_options) {}

  void Dump(Logger* log) const;

  void DumpCFOptions(Logger* log) const;

  // Set appropriate parameters for bulk loading.
  // The reason that this is a function that returns "this" instead of a
  // constructor is to enable chaining of multiple similar calls in the future.
  //

  // All data will be in level 0 without any automatic compaction.
  // It's recommended to manually call CompactRange(NULL, NULL) before reading
  // from the database, because otherwise the read can be very slow.
  Options* PrepareForBulkLoad();
};

//
// An application can issue a read request (via Get/Iterators) and specify
// if that read should process data that ALREADY resides on a specified cache
// level. For example, if an application specifies kBlockCacheTier then the
// Get call will process data that is already processed in the memtable or
// the block cache. It will not page in data from the OS cache or data that
// resides in storage.
enum ReadTier {
  kReadAllTier = 0x0,    // data in memtable, block cache, OS cache or storage
  kBlockCacheTier = 0x1  // data in memtable or block cache
};

// Options that control read operations
struct ReadOptions {
  // If true, all data read from underlying storage will be
  // verified against corresponding checksums.
  // Default: true
  bool verify_checksums;

  // Should the "data block"/"index block"/"filter block" read for this
  // iteration be cached in memory?
  // Callers may wish to set this field to false for bulk scans.
  // Default: true
  bool fill_cache;

  // If this option is set and memtable implementation allows, Seek
  // might only return keys with the same prefix as the seek-key
  //
  // ! DEPRECATED: prefix_seek is on by default when prefix_extractor
  // is configured
  // bool prefix_seek;

  // If "snapshot" is non-nullptr, read as of the supplied snapshot
  // (which must belong to the DB that is being read and which must
  // not have been released).  If "snapshot" is nullptr, use an impliicit
  // snapshot of the state at the beginning of this read operation.
  // Default: nullptr
  const Snapshot* snapshot;

  // If "prefix" is non-nullptr, and ReadOptions is being passed to
  // db.NewIterator, only return results when the key begins with this
  // prefix.  This field is ignored by other calls (e.g., Get).
  // Options.prefix_extractor must also be set, and
  // prefix_extractor.InRange(prefix) must be true.  The iterator
  // returned by NewIterator when this option is set will behave just
  // as if the underlying store did not contain any non-matching keys,
  // with two exceptions.  Seek() only accepts keys starting with the
  // prefix, and SeekToLast() is not supported.  prefix filter with this
  // option will sometimes reduce the number of read IOPs.
  // Default: nullptr
  //
  // ! DEPRECATED
  // const Slice* prefix;

  // "iterate_upper_bound" defines the extent upto which the forward iterator
  // can returns entries. Once the bound is reached, Valid() will be false.
  // "iterate_upper_bound" is exclusive ie the bound value is
  // not a valid entry.  If iterator_extractor is not null, the Seek target
  // and iterator_upper_bound need to have the same prefix.
  // This is because ordering is not guaranteed outside of prefix domain.
  // There is no lower bound on the iterator. If needed, that can be easily
  // implemented
  //
  // Default: nullptr
  const Slice* iterate_upper_bound;

  // Specify if this read request should process data that ALREADY
  // resides on a particular cache. If the required data is not
  // found at the specified cache, then Status::Incomplete is returned.
  // Default: kReadAllTier
  ReadTier read_tier;

  // Specify to create a tailing iterator -- a special iterator that has a
  // view of the complete database (i.e. it can also be used to read newly
  // added data) and is optimized for sequential reads. It will return records
  // that were inserted into the database after the creation of the iterator.
  // Default: false
  // Not supported in ROCKSDB_LITE mode!
  bool tailing;

  // Specify to create a managed iterator -- a special iterator that
  // uses less resources by having the ability to free its underlying
  // resources on request.
  // Default: false
  // Not supported in ROCKSDB_LITE mode!
  bool managed;

  // Enable a total order seek regardless of index format (e.g. hash index)
  // used in the table. Some table format (e.g. plain table) may not support
  // this option.
  bool total_order_seek;

  ReadOptions();
  ReadOptions(bool cksum, bool cache);
};

// Options that control write operations
struct WriteOptions {
  // If true, the write will be flushed from the operating system
  // buffer cache (by calling WritableFile::Sync()) before the write
  // is considered complete.  If this flag is true, writes will be
  // slower.
  //
  // If this flag is false, and the machine crashes, some recent
  // writes may be lost.  Note that if it is just the process that
  // crashes (i.e., the machine does not reboot), no writes will be
  // lost even if sync==false.
  //
  // In other words, a DB write with sync==false has similar
  // crash semantics as the "write()" system call.  A DB write
  // with sync==true has similar crash semantics to a "write()"
  // system call followed by "fdatasync()".
  //
  // Default: false
  bool sync;

  // If true, writes will not first go to the write ahead log,
  // and the write may got lost after a crash.
  bool disableWAL;

  // The option is deprecated. It's not used anymore.
  uint64_t timeout_hint_us;

  // If true and if user is trying to write to column families that don't exist
  // (they were dropped),  ignore the write (don't return an error). If there
  // are multiple writes in a WriteBatch, other writes will succeed.
  // Default: false
  bool ignore_missing_column_families;

  WriteOptions()
      : sync(false),
        disableWAL(false),
        timeout_hint_us(0),
        ignore_missing_column_families(false) {}
};

// Options that control flush operations
struct FlushOptions {
  // If true, the flush will wait until the flush is done.
  // Default: true
  bool wait;

  FlushOptions() : wait(true) {}
};

// Get options based on some guidelines. Now only tune parameter based on
// flush/compaction and fill default parameters for other parameters.
// total_write_buffer_limit: budget for memory spent for mem tables
// read_amplification_threshold: comfortable value of read amplification
// write_amplification_threshold: comfortable value of write amplification.
// target_db_size: estimated total DB size.
extern Options GetOptions(size_t total_write_buffer_limit,
                          int read_amplification_threshold = 8,
                          int write_amplification_threshold = 32,
                          uint64_t target_db_size = 68719476736 /* 64GB */);

// CompactionOptions are used in CompactFiles() call.
struct CompactionOptions {
  // Compaction output compression type
  // Default: snappy
  CompressionType compression;
  // Compaction will create files of size `output_file_size_limit`.
  // Default: MAX, which means that compaction will create a single file
  uint64_t output_file_size_limit;

  CompactionOptions()
      : compression(kSnappyCompression),
        output_file_size_limit(std::numeric_limits<uint64_t>::max()) {}
};

// For level based compaction, we can configure if we want to skip/force
// bottommost level compaction.
enum class BottommostLevelCompaction {
  // Skip bottommost level compaction
  kSkip,
  // Only compact bottommost level if there is a compaction filter
  // This is the default option
  kIfHaveCompactionFilter,
  // Always compact bottommost level
  kForce,
};

// CompactRangeOptions is used by CompactRange() call.
struct CompactRangeOptions {
  // If true, compacted files will be moved to the minimum level capable
  // of holding the data or given level (specified non-negative target_level).
  bool change_level = false;
  // If change_level is true and target_level have non-negative value, compacted
  // files will be moved to target_level.
  int target_level = -1;
  // Compaction outputs will be placed in options.db_paths[target_path_id].
  // Behavior is undefined if target_path_id is out of range.
  uint32_t target_path_id = 0;
  // By default level based compaction will only compact the bottommost level
  // if there is a compaction filter
  BottommostLevelCompaction bottommost_level_compaction =
      BottommostLevelCompaction::kIfHaveCompactionFilter;
};
}  // namespace rocksdb

#endif  // STORAGE_ROCKSDB_INCLUDE_OPTIONS_H_
