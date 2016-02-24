//  Copyright (c) 2014, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.
//
// Copyright (c) 2011 The LevelDB Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file. See the AUTHORS file for names of contributors.

#include "rocksdb/utilities/leveldb_options.h"
#include "rocksdb/cache.h"
#include "rocksdb/comparator.h"
#include "rocksdb/env.h"
#include "rocksdb/filter_policy.h"
#include "rocksdb/options.h"
#include "rocksdb/table.h"

namespace rocksdb {

LevelDBOptions::LevelDBOptions()
    : comparator(BytewiseComparator()),
      create_if_missing(false),
      error_if_exists(false),
      paranoid_checks(false),
      env(Env::Default()),
      info_log(nullptr),
      write_buffer_size(4 << 20),
      max_open_files(1000),
      block_cache(nullptr),
      block_size(4096),
      block_restart_interval(16),
      compression(kSnappyCompression),
      filter_policy(nullptr) {}

Options ConvertOptions(const LevelDBOptions& leveldb_options) {
  Options options = Options();
  options.create_if_missing = leveldb_options.create_if_missing;
  options.error_if_exists = leveldb_options.error_if_exists;
  options.paranoid_checks = leveldb_options.paranoid_checks;
  options.env = leveldb_options.env;
  options.info_log.reset(leveldb_options.info_log);
  options.write_buffer_size = leveldb_options.write_buffer_size;
  options.max_open_files = leveldb_options.max_open_files;
  options.compression = leveldb_options.compression;

  BlockBasedTableOptions table_options;
  table_options.block_cache.reset(leveldb_options.block_cache);
  table_options.block_size = leveldb_options.block_size;
  table_options.block_restart_interval = leveldb_options.block_restart_interval;
  table_options.filter_policy.reset(leveldb_options.filter_policy);
  options.table_factory.reset(NewBlockBasedTableFactory(table_options));

  return options;
}

}  // namespace rocksdb
