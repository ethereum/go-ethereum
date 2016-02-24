//  Copyright (c) 2013, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.
//
// Copyright (c) 2011 The LevelDB Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file. See the AUTHORS file for names of contributors.


#include "table/block_based_table_factory.h"

#include <memory>
#include <string>
#include <stdint.h>

#include "port/port.h"
#include "rocksdb/flush_block_policy.h"
#include "rocksdb/cache.h"
#include "table/block_based_table_builder.h"
#include "table/block_based_table_reader.h"
#include "table/format.h"

namespace rocksdb {

BlockBasedTableFactory::BlockBasedTableFactory(
    const BlockBasedTableOptions& table_options)
    : table_options_(table_options) {
  if (table_options_.flush_block_policy_factory == nullptr) {
    table_options_.flush_block_policy_factory.reset(
        new FlushBlockBySizePolicyFactory());
  }
  if (table_options_.no_block_cache) {
    table_options_.block_cache.reset();
  } else if (table_options_.block_cache == nullptr) {
    table_options_.block_cache = NewLRUCache(8 << 20);
  }
  if (table_options_.block_size_deviation < 0 ||
      table_options_.block_size_deviation > 100) {
    table_options_.block_size_deviation = 0;
  }
}

Status BlockBasedTableFactory::NewTableReader(
    const ImmutableCFOptions& ioptions, const EnvOptions& soptions,
    const InternalKeyComparator& internal_comparator,
    unique_ptr<RandomAccessFileReader>&& file, uint64_t file_size,
    unique_ptr<TableReader>* table_reader, const bool prefetch_enabled) const {
  return BlockBasedTable::Open(ioptions, soptions, table_options_,
                               internal_comparator, std::move(file), file_size,
                               table_reader, prefetch_enabled);
}

TableBuilder* BlockBasedTableFactory::NewTableBuilder(
    const TableBuilderOptions& table_builder_options,
    WritableFileWriter* file) const {
  auto table_builder = new BlockBasedTableBuilder(
      table_builder_options.ioptions, table_options_,
      table_builder_options.internal_comparator,
      table_builder_options.int_tbl_prop_collector_factories, file,
      table_builder_options.compression_type,
      table_builder_options.compression_opts,
      table_builder_options.skip_filters);

  return table_builder;
}

Status BlockBasedTableFactory::SanitizeOptions(
    const DBOptions& db_opts,
    const ColumnFamilyOptions& cf_opts) const {
  if (table_options_.index_type == BlockBasedTableOptions::kHashSearch &&
      cf_opts.prefix_extractor == nullptr) {
    return Status::InvalidArgument("Hash index is specified for block-based "
        "table, but prefix_extractor is not given");
  }
  if (table_options_.cache_index_and_filter_blocks &&
      table_options_.no_block_cache) {
    return Status::InvalidArgument("Enable cache_index_and_filter_blocks, "
        ", but block cache is disabled");
  }
  if (!BlockBasedTableSupportedVersion(table_options_.format_version)) {
    return Status::InvalidArgument(
        "Unsupported BlockBasedTable format_version. Please check "
        "include/rocksdb/table.h for more info");
  }
  return Status::OK();
}

std::string BlockBasedTableFactory::GetPrintableTableOptions() const {
  std::string ret;
  ret.reserve(20000);
  const int kBufferSize = 200;
  char buffer[kBufferSize];

  snprintf(buffer, kBufferSize, "  flush_block_policy_factory: %s (%p)\n",
           table_options_.flush_block_policy_factory->Name(),
           table_options_.flush_block_policy_factory.get());
  ret.append(buffer);
  snprintf(buffer, kBufferSize, "  cache_index_and_filter_blocks: %d\n",
           table_options_.cache_index_and_filter_blocks);
  ret.append(buffer);
  snprintf(buffer, kBufferSize, "  index_type: %d\n",
           table_options_.index_type);
  ret.append(buffer);
  snprintf(buffer, kBufferSize, "  hash_index_allow_collision: %d\n",
           table_options_.hash_index_allow_collision);
  ret.append(buffer);
  snprintf(buffer, kBufferSize, "  checksum: %d\n",
           table_options_.checksum);
  ret.append(buffer);
  snprintf(buffer, kBufferSize, "  no_block_cache: %d\n",
           table_options_.no_block_cache);
  ret.append(buffer);
  snprintf(buffer, kBufferSize, "  block_cache: %p\n",
           table_options_.block_cache.get());
  ret.append(buffer);
  if (table_options_.block_cache) {
    snprintf(buffer, kBufferSize, "  block_cache_size: %" ROCKSDB_PRIszt "\n",
             table_options_.block_cache->GetCapacity());
    ret.append(buffer);
  }
  snprintf(buffer, kBufferSize, "  block_cache_compressed: %p\n",
           table_options_.block_cache_compressed.get());
  ret.append(buffer);
  if (table_options_.block_cache_compressed) {
    snprintf(buffer, kBufferSize,
             "  block_cache_compressed_size: %" ROCKSDB_PRIszt "\n",
             table_options_.block_cache_compressed->GetCapacity());
    ret.append(buffer);
  }
  snprintf(buffer, kBufferSize, "  block_size: %" ROCKSDB_PRIszt "\n",
           table_options_.block_size);
  ret.append(buffer);
  snprintf(buffer, kBufferSize, "  block_size_deviation: %d\n",
           table_options_.block_size_deviation);
  ret.append(buffer);
  snprintf(buffer, kBufferSize, "  block_restart_interval: %d\n",
           table_options_.block_restart_interval);
  ret.append(buffer);
  snprintf(buffer, kBufferSize, "  filter_policy: %s\n",
           table_options_.filter_policy == nullptr ?
             "nullptr" : table_options_.filter_policy->Name());
  ret.append(buffer);
  snprintf(buffer, kBufferSize, "  whole_key_filtering: %d\n",
           table_options_.whole_key_filtering);
  snprintf(buffer, kBufferSize, "  format_version: %d\n",
           table_options_.format_version);
  ret.append(buffer);
  return ret;
}

const BlockBasedTableOptions& BlockBasedTableFactory::GetTableOptions() const {
  return table_options_;
}

TableFactory* NewBlockBasedTableFactory(
    const BlockBasedTableOptions& table_options) {
  return new BlockBasedTableFactory(table_options);
}

const std::string BlockBasedTablePropertyNames::kIndexType =
    "rocksdb.block.based.table.index.type";
const std::string BlockBasedTablePropertyNames::kWholeKeyFiltering =
    "rocksdb.block.based.table.whole.key.filtering";
const std::string BlockBasedTablePropertyNames::kPrefixFiltering =
    "rocksdb.block.based.table.prefix.filtering";
const std::string kHashIndexPrefixesBlock = "rocksdb.hashindex.prefixes";
const std::string kHashIndexPrefixesMetadataBlock =
    "rocksdb.hashindex.metadata";
const std::string kPropTrue = "1";
const std::string kPropFalse = "0";

}  // namespace rocksdb
