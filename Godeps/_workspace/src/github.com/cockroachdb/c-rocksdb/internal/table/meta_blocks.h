//  Copyright (c) 2013, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.
#pragma once

#include <map>
#include <memory>
#include <string>
#include <vector>

#include "db/builder.h"
#include "db/table_properties_collector.h"
#include "rocksdb/comparator.h"
#include "rocksdb/options.h"
#include "rocksdb/slice.h"
#include "table/block_builder.h"
#include "table/format.h"
#include "util/stl_wrappers.h"

namespace rocksdb {

class BlockBuilder;
class BlockHandle;
class Env;
class Footer;
class Logger;
class RandomAccessFile;
struct TableProperties;

class MetaIndexBuilder {
 public:
  MetaIndexBuilder(const MetaIndexBuilder&) = delete;
  MetaIndexBuilder& operator=(const MetaIndexBuilder&) = delete;

  MetaIndexBuilder();
  void Add(const std::string& key, const BlockHandle& handle);

  // Write all the added key/value pairs to the block and return the contents
  // of the block.
  Slice Finish();

 private:
  // store the sorted key/handle of the metablocks.
  stl_wrappers::KVMap meta_block_handles_;
  std::unique_ptr<BlockBuilder> meta_index_block_;
};

class PropertyBlockBuilder {
 public:
  PropertyBlockBuilder(const PropertyBlockBuilder&) = delete;
  PropertyBlockBuilder& operator=(const PropertyBlockBuilder&) = delete;

  PropertyBlockBuilder();

  void AddTableProperty(const TableProperties& props);
  void Add(const std::string& key, uint64_t value);
  void Add(const std::string& key, const std::string& value);
  void Add(const UserCollectedProperties& user_collected_properties);

  // Write all the added entries to the block and return the block contents
  Slice Finish();

 private:
  std::unique_ptr<BlockBuilder> properties_block_;
  stl_wrappers::KVMap props_;
};

// Were we encounter any error occurs during user-defined statistics collection,
// we'll write the warning message to info log.
void LogPropertiesCollectionError(
    Logger* info_log, const std::string& method, const std::string& name);

// Utility functions help table builder to trigger batch events for user
// defined property collectors.
// Return value indicates if there is any error occurred; if error occurred,
// the warning message will be logged.
// NotifyCollectTableCollectorsOnAdd() triggers the `Add` event for all
// property collectors.
bool NotifyCollectTableCollectorsOnAdd(
    const Slice& key, const Slice& value, uint64_t file_size,
    const std::vector<std::unique_ptr<IntTblPropCollector>>& collectors,
    Logger* info_log);

// NotifyCollectTableCollectorsOnAdd() triggers the `Finish` event for all
// property collectors. The collected properties will be added to `builder`.
bool NotifyCollectTableCollectorsOnFinish(
    const std::vector<std::unique_ptr<IntTblPropCollector>>& collectors,
    Logger* info_log, PropertyBlockBuilder* builder);

// Read the properties from the table.
// @returns a status to indicate if the operation succeeded. On success,
//          *table_properties will point to a heap-allocated TableProperties
//          object, otherwise value of `table_properties` will not be modified.
Status ReadProperties(const Slice& handle_value, RandomAccessFileReader* file,
                      const Footer& footer, Env* env, Logger* logger,
                      TableProperties** table_properties);

// Directly read the properties from the properties block of a plain table.
// @returns a status to indicate if the operation succeeded. On success,
//          *table_properties will point to a heap-allocated TableProperties
//          object, otherwise value of `table_properties` will not be modified.
Status ReadTableProperties(RandomAccessFileReader* file, uint64_t file_size,
                           uint64_t table_magic_number, Env* env,
                           Logger* info_log, TableProperties** properties);

// Find the meta block from the meta index block.
Status FindMetaBlock(Iterator* meta_index_iter,
                     const std::string& meta_block_name,
                     BlockHandle* block_handle);

// Find the meta block
Status FindMetaBlock(RandomAccessFileReader* file, uint64_t file_size,
                     uint64_t table_magic_number, Env* env,
                     const std::string& meta_block_name,
                     BlockHandle* block_handle);

// Read the specified meta block with name meta_block_name
// from `file` and initialize `contents` with contents of this block.
// Return Status::OK in case of success.
Status ReadMetaBlock(RandomAccessFileReader* file, uint64_t file_size,
                     uint64_t table_magic_number, Env* env,
                     const std::string& meta_block_name,
                     BlockContents* contents);

}  // namespace rocksdb
