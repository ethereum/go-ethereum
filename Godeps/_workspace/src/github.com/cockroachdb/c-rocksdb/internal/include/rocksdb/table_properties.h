// Copyright (c) 2011 The LevelDB Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file. See the AUTHORS file for names of contributors.
#pragma once

#include <stdint.h>
#include <string>
#include <map>
#include "rocksdb/status.h"
#include "rocksdb/types.h"

namespace rocksdb {

// -- Table Properties
// Other than basic table properties, each table may also have the user
// collected properties.
// The value of the user-collected properties are encoded as raw bytes --
// users have to interprete these values by themselves.
// Note: To do prefix seek/scan in `UserCollectedProperties`, you can do
// something similar to:
//
// UserCollectedProperties props = ...;
// for (auto pos = props.lower_bound(prefix);
//      pos != props.end() && pos->first.compare(0, prefix.size(), prefix) == 0;
//      ++pos) {
//   ...
// }
typedef std::map<std::string, std::string> UserCollectedProperties;

// TableProperties contains a bunch of read-only properties of its associated
// table.
struct TableProperties {
 public:
  // the total size of all data blocks.
  uint64_t data_size = 0;
  // the size of index block.
  uint64_t index_size = 0;
  // the size of filter block.
  uint64_t filter_size = 0;
  // total raw key size
  uint64_t raw_key_size = 0;
  // total raw value size
  uint64_t raw_value_size = 0;
  // the number of blocks in this table
  uint64_t num_data_blocks = 0;
  // the number of entries in this table
  uint64_t num_entries = 0;
  // format version, reserved for backward compatibility
  uint64_t format_version = 0;
  // If 0, key is variable length. Otherwise number of bytes for each key.
  uint64_t fixed_key_len = 0;

  // The name of the filter policy used in this table.
  // If no filter policy is used, `filter_policy_name` will be an empty string.
  std::string filter_policy_name;

  // user collected properties
  UserCollectedProperties user_collected_properties;

  // convert this object to a human readable form
  //   @prop_delim: delimiter for each property.
  std::string ToString(const std::string& prop_delim = "; ",
                       const std::string& kv_delim = "=") const;

  // Aggregate the numerical member variables of the specified
  // TableProperties.
  void Add(const TableProperties& tp);
};

// table properties' human-readable names in the property block.
struct TablePropertiesNames {
  static const std::string kDataSize;
  static const std::string kIndexSize;
  static const std::string kFilterSize;
  static const std::string kRawKeySize;
  static const std::string kRawValueSize;
  static const std::string kNumDataBlocks;
  static const std::string kNumEntries;
  static const std::string kFormatVersion;
  static const std::string kFixedKeyLen;
  static const std::string kFilterPolicy;
};

extern const std::string kPropertiesBlock;

enum EntryType {
  kEntryPut,
  kEntryDelete,
  kEntryMerge,
  kEntryOther,
};

// `TablePropertiesCollector` provides the mechanism for users to collect
// their own properties that they are interested in. This class is essentially
// a collection of callback functions that will be invoked during table
// building. It is construced with TablePropertiesCollectorFactory. The methods
// don't need to be thread-safe, as we will create exactly one
// TablePropertiesCollector object per table and then call it sequentially
class TablePropertiesCollector {
 public:
  virtual ~TablePropertiesCollector() {}

  // DEPRECATE User defined collector should implement AddUserKey(), though
  //           this old function still works for backward compatible reason.
  // Add() will be called when a new key/value pair is inserted into the table.
  // @params key    the user key that is inserted into the table.
  // @params value  the value that is inserted into the table.
  virtual Status Add(const Slice& key, const Slice& value) {
    return Status::InvalidArgument(
        "TablePropertiesCollector::Add() deprecated.");
  }

  // AddUserKey() will be called when a new key/value pair is inserted into the
  // table.
  // @params key    the user key that is inserted into the table.
  // @params value  the value that is inserted into the table.
  // @params file_size  file size up to now
  virtual Status AddUserKey(const Slice& key, const Slice& value,
                            EntryType type, SequenceNumber seq,
                            uint64_t file_size) {
    // For backwards-compatibility.
    return Add(key, value);
  }

  // Finish() will be called when a table has already been built and is ready
  // for writing the properties block.
  // @params properties  User will add their collected statistics to
  // `properties`.
  virtual Status Finish(UserCollectedProperties* properties) = 0;

  // Return the human-readable properties, where the key is property name and
  // the value is the human-readable form of value.
  virtual UserCollectedProperties GetReadableProperties() const = 0;

  // The name of the properties collector can be used for debugging purpose.
  virtual const char* Name() const = 0;

  // EXPERIMENTAL Return whether the output file should be further compacted
  virtual bool NeedCompact() const { return false; }
};

// Constructs TablePropertiesCollector. Internals create a new
// TablePropertiesCollector for each new table
class TablePropertiesCollectorFactory {
 public:
  virtual ~TablePropertiesCollectorFactory() {}
  // has to be thread-safe
  virtual TablePropertiesCollector* CreateTablePropertiesCollector() = 0;

  // The name of the properties collector can be used for debugging purpose.
  virtual const char* Name() const = 0;
};

// Extra properties
// Below is a list of non-basic properties that are collected by database
// itself. Especially some properties regarding to the internal keys (which
// is unknown to `table`).
extern uint64_t GetDeletedKeys(const UserCollectedProperties& props);

}  // namespace rocksdb
