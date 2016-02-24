// Copyright (c) 2014, Facebook, Inc. All rights reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.

#ifndef ROCKSDB_LITE
#include "table/cuckoo_table_factory.h"

#include "db/dbformat.h"
#include "table/cuckoo_table_builder.h"
#include "table/cuckoo_table_reader.h"

namespace rocksdb {

Status CuckooTableFactory::NewTableReader(
    const ImmutableCFOptions& ioptions, const EnvOptions& env_options,
    const InternalKeyComparator& icomp,
    std::unique_ptr<RandomAccessFileReader>&& file, uint64_t file_size,
    std::unique_ptr<TableReader>* table) const {
  std::unique_ptr<CuckooTableReader> new_reader(new CuckooTableReader(ioptions,
      std::move(file), file_size, icomp.user_comparator(), nullptr));
  Status s = new_reader->status();
  if (s.ok()) {
    *table = std::move(new_reader);
  }
  return s;
}

TableBuilder* CuckooTableFactory::NewTableBuilder(
    const TableBuilderOptions& table_builder_options,
    WritableFileWriter* file) const {
  // Ignore the skipFIlters flag. Does not apply to this file format
  //

  // TODO: change builder to take the option struct
  return new CuckooTableBuilder(
      file, table_options_.hash_table_ratio, 64,
      table_options_.max_search_depth,
      table_builder_options.internal_comparator.user_comparator(),
      table_options_.cuckoo_block_size, table_options_.use_module_hash,
      table_options_.identity_as_first_hash, nullptr);
}

std::string CuckooTableFactory::GetPrintableTableOptions() const {
  std::string ret;
  ret.reserve(2000);
  const int kBufferSize = 200;
  char buffer[kBufferSize];

  snprintf(buffer, kBufferSize, "  hash_table_ratio: %lf\n",
           table_options_.hash_table_ratio);
  ret.append(buffer);
  snprintf(buffer, kBufferSize, "  max_search_depth: %u\n",
           table_options_.max_search_depth);
  ret.append(buffer);
  snprintf(buffer, kBufferSize, "  cuckoo_block_size: %u\n",
           table_options_.cuckoo_block_size);
  ret.append(buffer);
  snprintf(buffer, kBufferSize, "  identity_as_first_hash: %d\n",
           table_options_.identity_as_first_hash);
  ret.append(buffer);
  return ret;
}

TableFactory* NewCuckooTableFactory(const CuckooTableOptions& table_options) {
  return new CuckooTableFactory(table_options);
}

}  // namespace rocksdb
#endif  // ROCKSDB_LITE
