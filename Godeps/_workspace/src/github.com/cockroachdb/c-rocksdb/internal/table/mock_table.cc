// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file. See the AUTHORS file for names of contributors.
//  Copyright (c) 2013, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.

#include "table/mock_table.h"

#include "db/dbformat.h"
#include "port/port.h"
#include "rocksdb/table_properties.h"
#include "table/get_context.h"
#include "util/coding.h"
#include "util/file_reader_writer.h"

namespace rocksdb {
namespace mock {

namespace {

const InternalKeyComparator icmp_(BytewiseComparator());

}  // namespace

stl_wrappers::KVMap MakeMockFile(
    std::initializer_list<std::pair<const std::string, std::string>> l) {
  return stl_wrappers::KVMap(l, stl_wrappers::LessOfComparator(&icmp_));
}

Iterator* MockTableReader::NewIterator(const ReadOptions&, Arena* arena) {
  return new MockTableIterator(table_);
}

Status MockTableReader::Get(const ReadOptions&, const Slice& key,
                            GetContext* get_context) {
  std::unique_ptr<MockTableIterator> iter(new MockTableIterator(table_));
  for (iter->Seek(key); iter->Valid(); iter->Next()) {
    ParsedInternalKey parsed_key;
    if (!ParseInternalKey(iter->key(), &parsed_key)) {
      return Status::Corruption(Slice());
    }

    if (!get_context->SaveValue(parsed_key, iter->value())) {
      break;
    }
  }
  return Status::OK();
}

std::shared_ptr<const TableProperties> MockTableReader::GetTableProperties()
    const {
  return std::shared_ptr<const TableProperties>(new TableProperties());
}

MockTableFactory::MockTableFactory() : next_id_(1) {}

Status MockTableFactory::NewTableReader(
    const ImmutableCFOptions& ioptions, const EnvOptions& env_options,
    const InternalKeyComparator& internal_key,
    unique_ptr<RandomAccessFileReader>&& file, uint64_t file_size,
    unique_ptr<TableReader>* table_reader) const {
  uint32_t id = GetIDFromFile(file.get());

  MutexLock lock_guard(&file_system_.mutex);

  auto it = file_system_.files.find(id);
  if (it == file_system_.files.end()) {
    return Status::IOError("Mock file not found");
  }

  table_reader->reset(new MockTableReader(it->second));

  return Status::OK();
}

TableBuilder* MockTableFactory::NewTableBuilder(
    const TableBuilderOptions& table_builder_options,
    WritableFileWriter* file) const {
  uint32_t id = GetAndWriteNextID(file->writable_file());

  return new MockTableBuilder(id, &file_system_);
}

Status MockTableFactory::CreateMockTable(Env* env, const std::string& fname,
                                         stl_wrappers::KVMap file_contents) {
  std::unique_ptr<WritableFile> file;
  auto s = env->NewWritableFile(fname, &file, EnvOptions());
  if (!s.ok()) {
    return s;
  }

  uint32_t id = GetAndWriteNextID(file.get());
  file_system_.files.insert({id, std::move(file_contents)});
  return Status::OK();
}

uint32_t MockTableFactory::GetAndWriteNextID(WritableFile* file) const {
  uint32_t next_id = next_id_.fetch_add(1);
  char buf[4];
  EncodeFixed32(buf, next_id);
  file->Append(Slice(buf, 4));
  return next_id;
}

uint32_t MockTableFactory::GetIDFromFile(RandomAccessFileReader* file) const {
  char buf[4];
  Slice result;
  file->Read(0, 4, &result, buf);
  assert(result.size() == 4);
  return DecodeFixed32(buf);
}

void MockTableFactory::AssertSingleFile(
    const stl_wrappers::KVMap& file_contents) {
  ASSERT_EQ(file_system_.files.size(), 1U);
  ASSERT_TRUE(file_contents == file_system_.files.begin()->second);
}

void MockTableFactory::AssertLatestFile(
    const stl_wrappers::KVMap& file_contents) {
  ASSERT_GE(file_system_.files.size(), 1U);
  auto latest = file_system_.files.end();
  --latest;

  if (file_contents != latest->second) {
    std::cout << "Wrong content! Content of latest file:" << std::endl;
    for (const auto& kv : latest->second) {
      ParsedInternalKey ikey;
      std::string key, value;
      std::tie(key, value) = kv;
      ParseInternalKey(Slice(key), &ikey);
      std::cout << ikey.DebugString(false) << " -> " << value << std::endl;
    }
    ASSERT_TRUE(false);
  }
}

}  // namespace mock
}  // namespace rocksdb
