//  Copyright (c) 2013, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.
//
// Copyright (c) 2011 The LevelDB Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file. See the AUTHORS file for names of contributors.
//

#pragma once
#include "db/version_set.h"

#include <vector>
#include <string>

class ColumnFamilyData;

namespace rocksdb {

class MemTableList;
class DBImpl;

// IMPORTANT: If you add a new property here, also add it to the list in
//            include/rocksdb/db.h
enum DBPropertyType : uint32_t {
  kUnknown,
  kNumFilesAtLevel,  // Number of files at a specific level
  kLevelStats,       // Return number of files and total sizes of each level
  kCFStats,          // Return general statitistics of CF
  kDBStats,          // Return general statitistics of DB
  kStats,            // Return general statitistics of both DB and CF
  kSsTables,         // Return a human readable string of current SST files
  kStartIntTypes,    // ---- Dummy value to indicate the start of integer values
  kNumImmutableMemTable,         // Return number of immutable mem tables that
                                 // have not been flushed.
  kNumImmutableMemTableFlushed,  // Return number of immutable mem tables
                                 // in memory that have already been flushed
  kMemtableFlushPending,         // Return 1 if mem table flushing is pending,
                                 // otherwise 0.
  kCompactionPending,      // Return 1 if a compaction is pending. Otherwise 0.
  kBackgroundErrors,       // Return accumulated background errors encountered.
  kCurSizeActiveMemTable,  // Return current size of the active memtable
  kCurSizeAllMemTables,    // Return current size of unflushed
                           // (active + immutable) memtables
  kSizeAllMemTables,       // Return current size of all (active + immutable
                           // + pinned) memtables
  kNumEntriesInMutableMemtable,    // Return number of deletes in the mutable
                                   // memtable.
  kNumEntriesInImmutableMemtable,  // Return sum of number of entries in all
                                   // the immutable mem tables.
  kNumDeletesInMutableMemtable,    // Return number of entries in the mutable
                                   // memtable.
  kNumDeletesInImmutableMemtable,  // Return sum of number of deletes in all
                                   // the immutable mem tables.
  kEstimatedNumKeys,  // Estimated total number of keys in the database.
  kEstimatedUsageByTableReaders,  // Estimated memory by table readers.
  kIsFileDeletionEnabled,         // Equals disable_delete_obsolete_files_,
                                  // 0 means file deletions enabled
  kNumSnapshots,                  // Number of snapshots in the system
  kOldestSnapshotTime,            // Unix timestamp of the first snapshot
  kNumLiveVersions,
  kEstimateLiveDataSize,            // Estimated amount of live data in bytes
  kTotalSstFilesSize,               // Total size of all sst files.
  kBaseLevel,                       // The level that L0 data is compacted to
  kEstimatePendingCompactionBytes,  // Estimated bytes to compaction
  kAggregatedTableProperties,  // Return a string that contains the aggregated
                               // table properties.
  kAggregatedTablePropertiesAtLevel,  // Return a string that contains the
                                      // aggregated
  // table properties at the specified level.
};

extern DBPropertyType GetPropertyType(const Slice& property,
                                      bool* is_int_property,
                                      bool* need_out_of_mutex);


#ifndef ROCKSDB_LITE
class InternalStats {
 public:
  enum InternalCFStatsType {
    LEVEL0_SLOWDOWN,
    MEMTABLE_COMPACTION,
    LEVEL0_NUM_FILES,
    WRITE_STALLS_ENUM_MAX,
    BYTES_FLUSHED,
    INTERNAL_CF_STATS_ENUM_MAX,
  };

  enum InternalDBStatsType {
    WAL_FILE_BYTES,
    WAL_FILE_SYNCED,
    BYTES_WRITTEN,
    NUMBER_KEYS_WRITTEN,
    WRITE_DONE_BY_OTHER,
    WRITE_DONE_BY_SELF,
    WRITE_WITH_WAL,
    WRITE_STALL_MICROS,
    INTERNAL_DB_STATS_ENUM_MAX,
  };

  InternalStats(int num_levels, Env* env, ColumnFamilyData* cfd)
      : db_stats_(INTERNAL_DB_STATS_ENUM_MAX),
        cf_stats_value_(INTERNAL_CF_STATS_ENUM_MAX),
        cf_stats_count_(INTERNAL_CF_STATS_ENUM_MAX),
        comp_stats_(num_levels),
        stall_leveln_slowdown_count_hard_(num_levels),
        stall_leveln_slowdown_count_soft_(num_levels),
        file_read_latency_(num_levels),
        bg_error_count_(0),
        number_levels_(num_levels),
        env_(env),
        cfd_(cfd),
        started_at_(env->NowMicros()) {
    for (int i = 0; i< INTERNAL_DB_STATS_ENUM_MAX; ++i) {
      db_stats_[i] = 0;
    }
    for (int i = 0; i< INTERNAL_CF_STATS_ENUM_MAX; ++i) {
      cf_stats_value_[i] = 0;
      cf_stats_count_[i] = 0;
    }
    for (int i = 0; i < num_levels; ++i) {
      stall_leveln_slowdown_count_hard_[i] = 0;
      stall_leveln_slowdown_count_soft_[i] = 0;
    }
  }

  // Per level compaction stats.  comp_stats_[level] stores the stats for
  // compactions that produced data for the specified "level".
  struct CompactionStats {
    uint64_t micros;

    // The number of bytes read from all non-output levels
    uint64_t bytes_read_non_output_levels;

    // The number of bytes read from the compaction output level.
    uint64_t bytes_read_output_level;

    // Total number of bytes written during compaction
    uint64_t bytes_written;

    // Total number of bytes moved to the output level
    uint64_t bytes_moved;

    // The number of compaction input files in all non-output levels.
    int num_input_files_in_non_output_levels;

    // The number of compaction input files in the output level.
    int num_input_files_in_output_level;

    // The number of compaction output files.
    int num_output_files;

    // Total incoming entries during compaction between levels N and N+1
    uint64_t num_input_records;

    // Accumulated diff number of entries
    // (num input entries - num output entires) for compaction  levels N and N+1
    uint64_t num_dropped_records;

    // Number of compactions done
    int count;

    explicit CompactionStats(int _count = 0)
        : micros(0),
          bytes_read_non_output_levels(0),
          bytes_read_output_level(0),
          bytes_written(0),
          bytes_moved(0),
          num_input_files_in_non_output_levels(0),
          num_input_files_in_output_level(0),
          num_output_files(0),
          num_input_records(0),
          num_dropped_records(0),
          count(_count) {}

    explicit CompactionStats(const CompactionStats& c)
        : micros(c.micros),
          bytes_read_non_output_levels(c.bytes_read_non_output_levels),
          bytes_read_output_level(c.bytes_read_output_level),
          bytes_written(c.bytes_written),
          bytes_moved(c.bytes_moved),
          num_input_files_in_non_output_levels(
              c.num_input_files_in_non_output_levels),
          num_input_files_in_output_level(
              c.num_input_files_in_output_level),
          num_output_files(c.num_output_files),
          num_input_records(c.num_input_records),
          num_dropped_records(c.num_dropped_records),
          count(c.count) {}

    void Add(const CompactionStats& c) {
      this->micros += c.micros;
      this->bytes_read_non_output_levels += c.bytes_read_non_output_levels;
      this->bytes_read_output_level += c.bytes_read_output_level;
      this->bytes_written += c.bytes_written;
      this->bytes_moved += c.bytes_moved;
      this->num_input_files_in_non_output_levels +=
          c.num_input_files_in_non_output_levels;
      this->num_input_files_in_output_level +=
          c.num_input_files_in_output_level;
      this->num_output_files += c.num_output_files;
      this->num_input_records += c.num_input_records;
      this->num_dropped_records += c.num_dropped_records;
      this->count += c.count;
    }

    void Subtract(const CompactionStats& c) {
      this->micros -= c.micros;
      this->bytes_read_non_output_levels -= c.bytes_read_non_output_levels;
      this->bytes_read_output_level -= c.bytes_read_output_level;
      this->bytes_written -= c.bytes_written;
      this->bytes_moved -= c.bytes_moved;
      this->num_input_files_in_non_output_levels -=
          c.num_input_files_in_non_output_levels;
      this->num_input_files_in_output_level -=
          c.num_input_files_in_output_level;
      this->num_output_files -= c.num_output_files;
      this->num_input_records -= c.num_input_records;
      this->num_dropped_records -= c.num_dropped_records;
      this->count -= c.count;
    }
  };

  void AddCompactionStats(int level, const CompactionStats& stats) {
    comp_stats_[level].Add(stats);
  }

  void IncBytesMoved(int level, uint64_t amount) {
    comp_stats_[level].bytes_moved += amount;
  }

  void RecordLevelNSlowdown(int level, bool soft) {
    if (soft) {
      ++stall_leveln_slowdown_count_soft_[level];
    } else {
      ++stall_leveln_slowdown_count_hard_[level];
    }
  }

  void AddCFStats(InternalCFStatsType type, uint64_t value) {
    cf_stats_value_[type] += value;
    ++cf_stats_count_[type];
  }

  void AddDBStats(InternalDBStatsType type, uint64_t value) {
    db_stats_[type] += value;
  }

  HistogramImpl* GetFileReadHist(int level) {
    return &file_read_latency_[level];
  }

  uint64_t GetBackgroundErrorCount() const { return bg_error_count_; }

  uint64_t BumpAndGetBackgroundErrorCount() { return ++bg_error_count_; }

  bool GetStringProperty(DBPropertyType property_type, const Slice& property,
                         std::string* value);

  bool GetIntProperty(DBPropertyType property_type, uint64_t* value,
                      DBImpl* db) const;

  bool GetIntPropertyOutOfMutex(DBPropertyType property_type, Version* version,
                                uint64_t* value) const;

 private:
  void DumpDBStats(std::string* value);
  void DumpCFStats(std::string* value);

  // Per-DB stats
  std::vector<uint64_t> db_stats_;
  // Per-ColumnFamily stats
  std::vector<uint64_t> cf_stats_value_;
  std::vector<uint64_t> cf_stats_count_;
  // Per-ColumnFamily/level compaction stats
  std::vector<CompactionStats> comp_stats_;
  // These count the number of microseconds for which MakeRoomForWrite stalls.
  std::vector<uint64_t> stall_leveln_slowdown_count_hard_;
  std::vector<uint64_t> stall_leveln_slowdown_count_soft_;
  std::vector<HistogramImpl> file_read_latency_;

  // Used to compute per-interval statistics
  struct CFStatsSnapshot {
    // ColumnFamily-level stats
    CompactionStats comp_stats;
    uint64_t ingest_bytes;            // Bytes written to L0
    uint64_t stall_count;             // Stall count

    CFStatsSnapshot()
        : comp_stats(0),
          ingest_bytes(0),
          stall_count(0) {}
  } cf_stats_snapshot_;

  struct DBStatsSnapshot {
    // DB-level stats
    uint64_t ingest_bytes;            // Bytes written by user
    uint64_t wal_bytes;               // Bytes written to WAL
    uint64_t wal_synced;              // Number of times WAL is synced
    uint64_t write_with_wal;          // Number of writes that request WAL
    // These count the number of writes processed by the calling thread or
    // another thread.
    uint64_t write_other;
    uint64_t write_self;
    // Stats from compaction jobs - bytes written, bytes read, duration.
    uint64_t compact_bytes_write;
    uint64_t compact_bytes_read;
    uint64_t compact_micros;
    // Total number of keys written. write_self and write_other measure number
    // of write requests written, Each of the write request can contain updates
    // to multiple keys. num_keys_written is total number of keys updated by all
    // those writes.
    uint64_t num_keys_written;
    // Total time writes delayed by stalls.
    uint64_t write_stall_micros;
    double seconds_up;

    DBStatsSnapshot()
        : ingest_bytes(0),
          wal_bytes(0),
          wal_synced(0),
          write_with_wal(0),
          write_other(0),
          write_self(0),
          compact_bytes_write(0),
          compact_bytes_read(0),
          compact_micros(0),
          num_keys_written(0),
          write_stall_micros(0),
          seconds_up(0) {}
  } db_stats_snapshot_;

  // Total number of background errors encountered. Every time a flush task
  // or compaction task fails, this counter is incremented. The failure can
  // be caused by any possible reason, including file system errors, out of
  // resources, or input file corruption. Failing when retrying the same flush
  // or compaction will cause the counter to increase too.
  uint64_t bg_error_count_;

  const int number_levels_;
  Env* env_;
  ColumnFamilyData* cfd_;
  const uint64_t started_at_;
};

#else

class InternalStats {
 public:
  enum InternalCFStatsType {
    LEVEL0_SLOWDOWN,
    MEMTABLE_COMPACTION,
    LEVEL0_NUM_FILES,
    WRITE_STALLS_ENUM_MAX,
    BYTES_FLUSHED,
    INTERNAL_CF_STATS_ENUM_MAX,
  };

  enum InternalDBStatsType {
    WAL_FILE_BYTES,
    WAL_FILE_SYNCED,
    BYTES_WRITTEN,
    NUMBER_KEYS_WRITTEN,
    WRITE_DONE_BY_OTHER,
    WRITE_DONE_BY_SELF,
    WRITE_WITH_WAL,
    WRITE_STALL_MICROS,
    INTERNAL_DB_STATS_ENUM_MAX,
  };

  InternalStats(int num_levels, Env* env, ColumnFamilyData* cfd) {}

  struct CompactionStats {
    uint64_t micros;
    uint64_t bytes_read_non_output_levels;
    uint64_t bytes_read_output_level;
    uint64_t bytes_written;
    uint64_t bytes_moved;
    int num_input_files_in_non_output_levels;
    int num_input_files_in_output_level;
    int num_output_files;
    uint64_t num_input_records;
    uint64_t num_dropped_records;
    int count;

    explicit CompactionStats(int _count = 0) {}

    explicit CompactionStats(const CompactionStats& c) {}

    void Add(const CompactionStats& c) {}

    void Subtract(const CompactionStats& c) {}
  };

  void AddCompactionStats(int level, const CompactionStats& stats) {}

  void IncBytesMoved(int level, uint64_t amount) {}

  void RecordLevelNSlowdown(int level, bool soft) {}

  void AddCFStats(InternalCFStatsType type, uint64_t value) {}

  void AddDBStats(InternalDBStatsType type, uint64_t value) {}

  HistogramImpl* GetFileReadHist(int level) { return nullptr; }

  uint64_t GetBackgroundErrorCount() const { return 0; }

  uint64_t BumpAndGetBackgroundErrorCount() { return 0; }

  bool GetStringProperty(DBPropertyType property_type, const Slice& property,
                         std::string* value) { return false; }

  bool GetIntProperty(DBPropertyType property_type, uint64_t* value,
                      DBImpl* db) const { return false; }

  bool GetIntPropertyOutOfMutex(DBPropertyType property_type, Version* version,
                                uint64_t* value) const { return false; }
};
#endif  // !ROCKSDB_LITE

}  // namespace rocksdb
