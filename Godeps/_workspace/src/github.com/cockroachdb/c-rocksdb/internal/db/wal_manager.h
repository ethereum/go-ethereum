//  Copyright (c) 2013, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.
//
// Copyright (c) 2011 The LevelDB Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file. See the AUTHORS file for names of contributors.
#pragma once

#include <atomic>
#include <deque>
#include <limits>
#include <set>
#include <utility>
#include <vector>
#include <string>
#include <memory>

#include "port/port.h"

#include "rocksdb/env.h"
#include "rocksdb/options.h"
#include "rocksdb/types.h"
#include "rocksdb/transaction_log.h"
#include "rocksdb/status.h"

#include "db/version_set.h"

namespace rocksdb {

#ifndef ROCKSDB_LITE
class WalManager {
 public:
  WalManager(const DBOptions& db_options, const EnvOptions& env_options)
      : db_options_(db_options),
        env_options_(env_options),
        env_(db_options.env),
        purge_wal_files_last_run_(0) {}

  Status GetSortedWalFiles(VectorLogPtr& files);

  Status GetUpdatesSince(
      SequenceNumber seq_number, std::unique_ptr<TransactionLogIterator>* iter,
      const TransactionLogIterator::ReadOptions& read_options,
      VersionSet* version_set);

  void PurgeObsoleteWALFiles();

  void ArchiveWALFile(const std::string& fname, uint64_t number);

  Status TEST_ReadFirstRecord(const WalFileType type, const uint64_t number,
                              SequenceNumber* sequence) {
    return ReadFirstRecord(type, number, sequence);
  }

  Status TEST_ReadFirstLine(const std::string& fname,
                            SequenceNumber* sequence) {
    return ReadFirstLine(fname, sequence);
  }

 private:
  Status GetSortedWalsOfType(const std::string& path, VectorLogPtr& log_files,
                             WalFileType type);
  // Requires: all_logs should be sorted with earliest log file first
  // Retains all log files in all_logs which contain updates with seq no.
  // Greater Than or Equal to the requested SequenceNumber.
  Status RetainProbableWalFiles(VectorLogPtr& all_logs,
                                const SequenceNumber target);

  Status ReadFirstRecord(const WalFileType type, const uint64_t number,
                         SequenceNumber* sequence);

  Status ReadFirstLine(const std::string& fname, SequenceNumber* sequence);

  // ------- state from DBImpl ------
  const DBOptions& db_options_;
  const EnvOptions& env_options_;
  Env* env_;

  // ------- WalManager state -------
  // cache for ReadFirstRecord() calls
  std::unordered_map<uint64_t, SequenceNumber> read_first_record_cache_;
  port::Mutex read_first_record_cache_mutex_;

  // last time when PurgeObsoleteWALFiles ran.
  uint64_t purge_wal_files_last_run_;

  // obsolete files will be deleted every this seconds if ttl deletion is
  // enabled and archive size_limit is disabled.
  static const uint64_t kDefaultIntervalToDeleteObsoleteWAL = 600;
};

#endif  // ROCKSDB_LITE
}  // namespace rocksdb
