// Copyright (c) 2011 The LevelDB Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file. See the AUTHORS file for names of contributors.

#ifndef ROCKSDB_LITE
#include "table/adaptive_table_factory.h"

#include "table/format.h"
#include "port/port.h"

namespace rocksdb {

AdaptiveTableFactory::AdaptiveTableFactory(
    std::shared_ptr<TableFactory> table_factory_to_write,
    std::shared_ptr<TableFactory> block_based_table_factory,
    std::shared_ptr<TableFactory> plain_table_factory,
    std::shared_ptr<TableFactory> cuckoo_table_factory)
    : table_factory_to_write_(table_factory_to_write),
      block_based_table_factory_(block_based_table_factory),
      plain_table_factory_(plain_table_factory),
      cuckoo_table_factory_(cuckoo_table_factory) {
  if (!table_factory_to_write_) {
    table_factory_to_write_ = block_based_table_factory_;
  }
  if (!plain_table_factory_) {
    plain_table_factory_.reset(NewPlainTableFactory());
  }
  if (!block_based_table_factory_) {
    block_based_table_factory_.reset(NewBlockBasedTableFactory());
  }
  if (!cuckoo_table_factory_) {
    cuckoo_table_factory_.reset(NewCuckooTableFactory());
  }
}

extern const uint64_t kPlainTableMagicNumber;
extern const uint64_t kLegacyPlainTableMagicNumber;
extern const uint64_t kBlockBasedTableMagicNumber;
extern const uint64_t kLegacyBlockBasedTableMagicNumber;
extern const uint64_t kCuckooTableMagicNumber;

Status AdaptiveTableFactory::NewTableReader(
    const ImmutableCFOptions& ioptions, const EnvOptions& env_options,
    const InternalKeyComparator& icomp,
    unique_ptr<RandomAccessFileReader>&& file, uint64_t file_size,
    unique_ptr<TableReader>* table) const {
  Footer footer;
  auto s = ReadFooterFromFile(file.get(), file_size, &footer);
  if (!s.ok()) {
    return s;
  }
  if (footer.table_magic_number() == kPlainTableMagicNumber ||
      footer.table_magic_number() == kLegacyPlainTableMagicNumber) {
    return plain_table_factory_->NewTableReader(
        ioptions, env_options, icomp, std::move(file), file_size, table);
  } else if (footer.table_magic_number() == kBlockBasedTableMagicNumber ||
      footer.table_magic_number() == kLegacyBlockBasedTableMagicNumber) {
    return block_based_table_factory_->NewTableReader(
        ioptions, env_options, icomp, std::move(file), file_size, table);
  } else if (footer.table_magic_number() == kCuckooTableMagicNumber) {
    return cuckoo_table_factory_->NewTableReader(
        ioptions, env_options, icomp, std::move(file), file_size, table);
  } else {
    return Status::NotSupported("Unidentified table format");
  }
}

TableBuilder* AdaptiveTableFactory::NewTableBuilder(
    const TableBuilderOptions& table_builder_options,
    WritableFileWriter* file) const {
  return table_factory_to_write_->NewTableBuilder(table_builder_options, file);
}

std::string AdaptiveTableFactory::GetPrintableTableOptions() const {
  std::string ret;
  ret.reserve(20000);
  const int kBufferSize = 200;
  char buffer[kBufferSize];

  if (!table_factory_to_write_) {
    snprintf(buffer, kBufferSize, "  write factory (%s) options:\n%s\n",
             table_factory_to_write_->Name(),
             table_factory_to_write_->GetPrintableTableOptions().c_str());
    ret.append(buffer);
  }
  if (!plain_table_factory_) {
    snprintf(buffer, kBufferSize, "  %s options:\n%s\n",
             plain_table_factory_->Name(),
             plain_table_factory_->GetPrintableTableOptions().c_str());
    ret.append(buffer);
  }
  if (!block_based_table_factory_) {
    snprintf(buffer, kBufferSize, "  %s options:\n%s\n",
             block_based_table_factory_->Name(),
             block_based_table_factory_->GetPrintableTableOptions().c_str());
    ret.append(buffer);
  }
  if (!cuckoo_table_factory_) {
    snprintf(buffer, kBufferSize, "  %s options:\n%s\n",
             cuckoo_table_factory_->Name(),
             cuckoo_table_factory_->GetPrintableTableOptions().c_str());
    ret.append(buffer);
  }
  return ret;
}

extern TableFactory* NewAdaptiveTableFactory(
    std::shared_ptr<TableFactory> table_factory_to_write,
    std::shared_ptr<TableFactory> block_based_table_factory,
    std::shared_ptr<TableFactory> plain_table_factory,
    std::shared_ptr<TableFactory> cuckoo_table_factory) {
  return new AdaptiveTableFactory(table_factory_to_write,
      block_based_table_factory, plain_table_factory, cuckoo_table_factory);
}

}  // namespace rocksdb
#endif  // ROCKSDB_LITE
