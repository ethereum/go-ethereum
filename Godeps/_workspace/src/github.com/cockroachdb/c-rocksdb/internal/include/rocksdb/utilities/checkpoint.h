// Copyright (c) 2013, Facebook, Inc.  All rights reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.
//
// A checkpoint is an openable snapshot of a database at a point in time.

#pragma once
#ifndef ROCKSDB_LITE

#include <string>
#include "rocksdb/status.h"

namespace rocksdb {

class DB;

class Checkpoint {
 public:
  // Creates a Checkpoint object to be used for creating openable sbapshots
  static Status Create(DB* db, Checkpoint** checkpoint_ptr);

  // Builds an openable snapshot of RocksDB on the same disk, which
  // accepts an output directory on the same disk, and under the directory
  // (1) hard-linked SST files pointing to existing live SST files
  // SST files will be copied if output directory is on a different filesystem
  // (2) a copied manifest files and other files
  // The directory should not already exist and will be created by this API.
  // The directory will be an absolute path
  virtual Status CreateCheckpoint(const std::string& checkpoint_dir);

  virtual ~Checkpoint() {}
};

}  // namespace rocksdb
#endif  // !ROCKSDB_LITE
