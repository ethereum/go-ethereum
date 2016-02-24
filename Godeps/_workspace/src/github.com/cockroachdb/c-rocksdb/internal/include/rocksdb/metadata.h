// Copyright (c) 2014, Facebook, Inc.  All rights reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.

#pragma once

#include <stdint.h>

#include <limits>
#include <string>
#include <vector>

#include "rocksdb/types.h"

namespace rocksdb {
struct ColumnFamilyMetaData;
struct LevelMetaData;
struct SstFileMetaData;

// The metadata that describes a column family.
struct ColumnFamilyMetaData {
  ColumnFamilyMetaData() : size(0), name("") {}
  ColumnFamilyMetaData(const std::string& _name, uint64_t _size,
                       const std::vector<LevelMetaData>&& _levels) :
      size(_size), name(_name), levels(_levels) {}

  // The size of this column family in bytes, which is equal to the sum of
  // the file size of its "levels".
  uint64_t size;
  // The number of files in this column family.
  size_t file_count;
  // The name of the column family.
  std::string name;
  // The metadata of all levels in this column family.
  std::vector<LevelMetaData> levels;
};

// The metadata that describes a level.
struct LevelMetaData {
  LevelMetaData(int _level, uint64_t _size,
                const std::vector<SstFileMetaData>&& _files) :
      level(_level), size(_size),
      files(_files) {}

  // The level which this meta data describes.
  const int level;
  // The size of this level in bytes, which is equal to the sum of
  // the file size of its "files".
  const uint64_t size;
  // The metadata of all sst files in this level.
  const std::vector<SstFileMetaData> files;
};

// The metadata that describes a SST file.
struct SstFileMetaData {
  SstFileMetaData() {}
  SstFileMetaData(const std::string& _file_name,
                  const std::string& _path, uint64_t _size,
                  SequenceNumber _smallest_seqno,
                  SequenceNumber _largest_seqno,
                  const std::string& _smallestkey,
                  const std::string& _largestkey,
                  bool _being_compacted) :
    size(_size), name(_file_name),
    db_path(_path), smallest_seqno(_smallest_seqno), largest_seqno(_largest_seqno),
    smallestkey(_smallestkey), largestkey(_largestkey),
    being_compacted(_being_compacted) {}

  // File size in bytes.
  uint64_t size;
  // The name of the file.
  std::string name;
  // The full path where the file locates.
  std::string db_path;

  SequenceNumber smallest_seqno;  // Smallest sequence number in file.
  SequenceNumber largest_seqno;   // Largest sequence number in file.
  std::string smallestkey;     // Smallest user defined key in the file.
  std::string largestkey;      // Largest user defined key in the file.
  bool being_compacted;  // true if the file is currently being compacted.
};

// The full set of metadata associated with each SST file.
struct LiveFileMetaData : SstFileMetaData {
  std::string column_family_name;  // Name of the column family
  int level;               // Level at which this file resides.
};



}  // namespace rocksdb
