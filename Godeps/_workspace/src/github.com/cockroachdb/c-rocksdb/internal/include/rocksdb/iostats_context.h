// Copyright (c) 2014, Facebook, Inc.  All rights reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.
#pragma once

#include <stdint.h>
#include <string>

#include "rocksdb/perf_level.h"

// A thread local context for gathering io-stats efficiently and transparently.
// Use SetPerfLevel(PerfLevel::kEnableTime) to enable time stats.

namespace rocksdb {

struct IOStatsContext {
  // reset all io-stats counter to zero
  void Reset();

  std::string ToString() const;

  // the thread pool id
  uint64_t thread_pool_id;

  // number of bytes that has been written.
  uint64_t bytes_written;
  // number of bytes that has been read.
  uint64_t bytes_read;

  // time spent in open() and fopen().
  uint64_t open_nanos;
  // time spent in fallocate().
  uint64_t allocate_nanos;
  // time spent in write() and pwrite().
  uint64_t write_nanos;
  // time spent in read() and pread()
  uint64_t read_nanos;
  // time spent in sync_file_range().
  uint64_t range_sync_nanos;
  // time spent in fsync
  uint64_t fsync_nanos;
  // time spent in preparing write (fallocate etc).
  uint64_t prepare_write_nanos;
  // time spent in Logger::Logv().
  uint64_t logger_nanos;
};

#ifndef IOS_CROSS_COMPILE
# ifdef _WIN32
extern __declspec(thread) IOStatsContext iostats_context;
# else
extern __thread IOStatsContext iostats_context;
# endif
#endif  // IOS_CROSS_COMPILE

}  // namespace rocksdb
