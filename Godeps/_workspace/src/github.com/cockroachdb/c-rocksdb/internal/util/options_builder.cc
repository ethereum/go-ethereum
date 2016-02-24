//  Copyright (c) 2013, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.

#include <math.h>
#include <algorithm>
#include "rocksdb/options.h"

namespace rocksdb {

namespace {

// For now, always use 1-0 as level bytes multiplier.
const int kBytesForLevelMultiplier = 10;
const size_t kBytesForOneMb = 1024 * 1024;

// Pick compaction style
CompactionStyle PickCompactionStyle(size_t write_buffer_size,
                                    int read_amp_threshold,
                                    int write_amp_threshold,
                                    uint64_t target_db_size) {
#ifndef ROCKSDB_LITE
  // Estimate read amplification and write amplification of two compaction
  // styles. If there is hard limit to force a choice, make the choice.
  // Otherwise, calculate a score based on threshold and expected value of
  // two styles, weighing reads 4X important than writes.
  int expected_levels = static_cast<int>(ceil(
      ::log(target_db_size / write_buffer_size) / ::log(kBytesForLevelMultiplier)));

  int expected_max_files_universal =
      static_cast<int>(ceil(log2(target_db_size / write_buffer_size)));

  const int kEstimatedLevel0FilesInLevelStyle = 2;
  // Estimate write amplification:
  // (1) 1 for every L0 file
  // (2) 2 for L1
  // (3) kBytesForLevelMultiplier for the last level. It's really hard to
  //     predict.
  // (3) kBytesForLevelMultiplier for other levels.
  int expected_write_amp_level = kEstimatedLevel0FilesInLevelStyle + 2
      + (expected_levels - 2) * kBytesForLevelMultiplier
      + kBytesForLevelMultiplier;
  int expected_read_amp_level =
      kEstimatedLevel0FilesInLevelStyle + expected_levels;

  int max_read_amp_uni = expected_max_files_universal;
  if (read_amp_threshold <= max_read_amp_uni) {
    return kCompactionStyleLevel;
  } else if (write_amp_threshold <= expected_write_amp_level) {
    return kCompactionStyleUniversal;
  }

  const double kReadWriteWeight = 4;

  double level_ratio =
      static_cast<double>(read_amp_threshold) / expected_read_amp_level *
          kReadWriteWeight +
      static_cast<double>(write_amp_threshold) / expected_write_amp_level;

  int expected_write_amp_uni = expected_max_files_universal / 2 + 2;
  int expected_read_amp_uni = expected_max_files_universal / 2 + 1;

  double uni_ratio =
      static_cast<double>(read_amp_threshold) / expected_read_amp_uni *
          kReadWriteWeight +
      static_cast<double>(write_amp_threshold) / expected_write_amp_uni;

  if (level_ratio > uni_ratio) {
    return kCompactionStyleLevel;
  } else {
    return kCompactionStyleUniversal;
  }
#else
  return kCompactionStyleLevel;
#endif  // !ROCKSDB_LITE
}

// Pick mem table size
void PickWriteBufferSize(size_t total_write_buffer_limit, Options* options) {
  const size_t kMaxWriteBufferSize = 128 * kBytesForOneMb;
  const size_t kMinWriteBufferSize = 4 * kBytesForOneMb;

  // Try to pick up a buffer size between 4MB and 128MB.
  // And try to pick 4 as the total number of write buffers.
  size_t write_buffer_size = total_write_buffer_limit / 4;
  if (write_buffer_size > kMaxWriteBufferSize) {
    write_buffer_size = kMaxWriteBufferSize;
  } else if (write_buffer_size < kMinWriteBufferSize) {
    write_buffer_size = std::min(static_cast<size_t>(kMinWriteBufferSize),
                                 total_write_buffer_limit / 2);
  }

  // Truncate to multiple of 1MB.
  if (write_buffer_size % kBytesForOneMb != 0) {
    write_buffer_size =
        (write_buffer_size / kBytesForOneMb + 1) * kBytesForOneMb;
  }

  options->write_buffer_size = write_buffer_size;
  options->max_write_buffer_number =
      static_cast<int>(total_write_buffer_limit / write_buffer_size);
  options->min_write_buffer_number_to_merge = 1;
}

#ifndef ROCKSDB_LITE
void OptimizeForUniversal(Options* options) {
  options->level0_file_num_compaction_trigger = 2;
  options->level0_slowdown_writes_trigger = 30;
  options->level0_stop_writes_trigger = 40;
  options->max_open_files = -1;
}
#endif

// Optimize parameters for level-based compaction
void OptimizeForLevel(int read_amplification_threshold,
                      int write_amplification_threshold,
                      uint64_t target_db_size, Options* options) {
  int expected_levels_one_level0_file =
      static_cast<int>(ceil(::log(target_db_size / options->write_buffer_size) /
                            ::log(kBytesForLevelMultiplier)));

  int level0_stop_writes_trigger =
      read_amplification_threshold - expected_levels_one_level0_file;

  const size_t kInitialLevel0TotalSize = 128 * kBytesForOneMb;
  const int kMaxFileNumCompactionTrigger = 4;
  const int kMinLevel0StopTrigger = 3;

  int file_num_buffer =
      kInitialLevel0TotalSize / options->write_buffer_size + 1;

  if (level0_stop_writes_trigger > file_num_buffer) {
    // Have sufficient room for multiple level 0 files
    // Try enlarge the buffer up to 1GB

    // Try to enlarge the buffer up to 1GB, if still have sufficient headroom.
    file_num_buffer *=
        1 << std::max(0, std::min(3, level0_stop_writes_trigger -
                                       file_num_buffer - 2));

    options->level0_stop_writes_trigger = level0_stop_writes_trigger;
    options->level0_slowdown_writes_trigger = level0_stop_writes_trigger - 2;
    options->level0_file_num_compaction_trigger =
        std::min(kMaxFileNumCompactionTrigger, file_num_buffer / 2);
  } else {
    options->level0_stop_writes_trigger =
        std::max(kMinLevel0StopTrigger, file_num_buffer);
    options->level0_slowdown_writes_trigger =
        options->level0_stop_writes_trigger - 1;
    options->level0_file_num_compaction_trigger = 1;
  }

  // This doesn't consider compaction and overheads of mem tables. But usually
  // it is in the same order of magnitude.
  size_t expected_level0_compaction_size =
      options->level0_file_num_compaction_trigger * options->write_buffer_size;
  // Enlarge level1 target file size if level0 compaction size is larger.
  uint64_t max_bytes_for_level_base = 10 * kBytesForOneMb;
  if (expected_level0_compaction_size > max_bytes_for_level_base) {
    max_bytes_for_level_base = expected_level0_compaction_size;
  }
  options->max_bytes_for_level_base = max_bytes_for_level_base;
  // Now always set level multiplier to be 10
  options->max_bytes_for_level_multiplier = kBytesForLevelMultiplier;

  const uint64_t kMinFileSize = 2 * kBytesForOneMb;
  // Allow at least 3-way parallelism for compaction between level 1 and 2.
  uint64_t max_file_size = max_bytes_for_level_base / 3;
  if (max_file_size < kMinFileSize) {
    options->target_file_size_base = kMinFileSize;
  } else {
    if (max_file_size % kBytesForOneMb != 0) {
      max_file_size = (max_file_size / kBytesForOneMb + 1) * kBytesForOneMb;
    }
    options->target_file_size_base = max_file_size;
  }

  // TODO: consider to tune num_levels too.
}

}  // namespace

Options GetOptions(size_t total_write_buffer_limit,
                   int read_amplification_threshold,
                   int write_amplification_threshold, uint64_t target_db_size) {
  Options options;
  PickWriteBufferSize(total_write_buffer_limit, &options);
  size_t write_buffer_size = options.write_buffer_size;
  options.compaction_style =
      PickCompactionStyle(write_buffer_size, read_amplification_threshold,
                          write_amplification_threshold, target_db_size);
#ifndef ROCKSDB_LITE
  if (options.compaction_style == kCompactionStyleUniversal) {
    OptimizeForUniversal(&options);
  } else {
#else
  {
#endif  // !ROCKSDB_LITE
    OptimizeForLevel(read_amplification_threshold,
                     write_amplification_threshold, target_db_size, &options);
  }
  return options;
}

}  // namespace rocksdb
