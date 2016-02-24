// Copyright (c) 2014, Facebook, Inc. All rights reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.

#ifndef ROCKSDB_LITE

#ifndef GFLAGS
#include <cstdio>
int main() {
  fprintf(stderr, "Please install gflags to run this test\n");
  return 1;
}
#else

#ifndef __STDC_FORMAT_MACROS
#define __STDC_FORMAT_MACROS
#endif

#include <inttypes.h>
#include <gflags/gflags.h>
#include <vector>
#include <string>
#include <map>

#include "table/meta_blocks.h"
#include "table/cuckoo_table_builder.h"
#include "table/cuckoo_table_reader.h"
#include "table/cuckoo_table_factory.h"
#include "table/get_context.h"
#include "util/arena.h"
#include "util/random.h"
#include "util/string_util.h"
#include "util/testharness.h"
#include "util/testutil.h"

using GFLAGS::ParseCommandLineFlags;
using GFLAGS::SetUsageMessage;

DEFINE_string(file_dir, "", "Directory where the files will be created"
    " for benchmark. Added for using tmpfs.");
DEFINE_bool(enable_perf, false, "Run Benchmark Tests too.");
DEFINE_bool(write, false,
    "Should write new values to file in performance tests?");
DEFINE_bool(identity_as_first_hash, true, "use identity as first hash");

namespace rocksdb {

namespace {
const uint32_t kNumHashFunc = 10;
// Methods, variables related to Hash functions.
std::unordered_map<std::string, std::vector<uint64_t>> hash_map;

void AddHashLookups(const std::string& s, uint64_t bucket_id,
        uint32_t num_hash_fun) {
  std::vector<uint64_t> v;
  for (uint32_t i = 0; i < num_hash_fun; i++) {
    v.push_back(bucket_id + i);
  }
  hash_map[s] = v;
}

uint64_t GetSliceHash(const Slice& s, uint32_t index,
    uint64_t max_num_buckets) {
  return hash_map[s.ToString()][index];
}
}  // namespace

class CuckooReaderTest : public testing::Test {
 public:
  using testing::Test::SetUp;

  CuckooReaderTest() {
    options.allow_mmap_reads = true;
    env = options.env;
    env_options = EnvOptions(options);
  }

  void SetUp(int num) {
    num_items = num;
    hash_map.clear();
    keys.clear();
    keys.resize(num_items);
    user_keys.clear();
    user_keys.resize(num_items);
    values.clear();
    values.resize(num_items);
  }

  std::string NumToStr(int64_t i) {
    return std::string(reinterpret_cast<char*>(&i), sizeof(i));
  }

  void CreateCuckooFileAndCheckReader(
      const Comparator* ucomp = BytewiseComparator()) {
    std::unique_ptr<WritableFile> writable_file;
    ASSERT_OK(env->NewWritableFile(fname, &writable_file, env_options));
    unique_ptr<WritableFileWriter> file_writer(
        new WritableFileWriter(std::move(writable_file), env_options));

    CuckooTableBuilder builder(file_writer.get(), 0.9, kNumHashFunc, 100, ucomp,
                               2, false, false, GetSliceHash);
    ASSERT_OK(builder.status());
    for (uint32_t key_idx = 0; key_idx < num_items; ++key_idx) {
      builder.Add(Slice(keys[key_idx]), Slice(values[key_idx]));
      ASSERT_OK(builder.status());
      ASSERT_EQ(builder.NumEntries(), key_idx + 1);
    }
    ASSERT_OK(builder.Finish());
    ASSERT_EQ(num_items, builder.NumEntries());
    file_size = builder.FileSize();
    ASSERT_OK(file_writer->Close());

    // Check reader now.
    std::unique_ptr<RandomAccessFile> read_file;
    ASSERT_OK(env->NewRandomAccessFile(fname, &read_file, env_options));
    unique_ptr<RandomAccessFileReader> file_reader(
        new RandomAccessFileReader(std::move(read_file)));
    const ImmutableCFOptions ioptions(options);
    CuckooTableReader reader(ioptions, std::move(file_reader), file_size, ucomp,
                             GetSliceHash);
    ASSERT_OK(reader.status());
    // Assume no merge/deletion
    for (uint32_t i = 0; i < num_items; ++i) {
      std::string value;
      GetContext get_context(ucomp, nullptr, nullptr, nullptr,
                             GetContext::kNotFound, Slice(user_keys[i]), &value,
                             nullptr, nullptr, nullptr);
      ASSERT_OK(reader.Get(ReadOptions(), Slice(keys[i]), &get_context));
      ASSERT_EQ(values[i], value);
    }
  }
  void UpdateKeys(bool with_zero_seqno) {
    for (uint32_t i = 0; i < num_items; i++) {
      ParsedInternalKey ikey(user_keys[i],
          with_zero_seqno ? 0 : i + 1000, kTypeValue);
      keys[i].clear();
      AppendInternalKey(&keys[i], ikey);
    }
  }

  void CheckIterator(const Comparator* ucomp = BytewiseComparator()) {
    std::unique_ptr<RandomAccessFile> read_file;
    ASSERT_OK(env->NewRandomAccessFile(fname, &read_file, env_options));
    unique_ptr<RandomAccessFileReader> file_reader(
        new RandomAccessFileReader(std::move(read_file)));
    const ImmutableCFOptions ioptions(options);
    CuckooTableReader reader(ioptions, std::move(file_reader), file_size, ucomp,
                             GetSliceHash);
    ASSERT_OK(reader.status());
    Iterator* it = reader.NewIterator(ReadOptions(), nullptr);
    ASSERT_OK(it->status());
    ASSERT_TRUE(!it->Valid());
    it->SeekToFirst();
    int cnt = 0;
    while (it->Valid()) {
      ASSERT_OK(it->status());
      ASSERT_TRUE(Slice(keys[cnt]) == it->key());
      ASSERT_TRUE(Slice(values[cnt]) == it->value());
      ++cnt;
      it->Next();
    }
    ASSERT_EQ(static_cast<uint32_t>(cnt), num_items);

    it->SeekToLast();
    cnt = static_cast<int>(num_items) - 1;
    ASSERT_TRUE(it->Valid());
    while (it->Valid()) {
      ASSERT_OK(it->status());
      ASSERT_TRUE(Slice(keys[cnt]) == it->key());
      ASSERT_TRUE(Slice(values[cnt]) == it->value());
      --cnt;
      it->Prev();
    }
    ASSERT_EQ(cnt, -1);

    cnt = static_cast<int>(num_items) / 2;
    it->Seek(keys[cnt]);
    while (it->Valid()) {
      ASSERT_OK(it->status());
      ASSERT_TRUE(Slice(keys[cnt]) == it->key());
      ASSERT_TRUE(Slice(values[cnt]) == it->value());
      ++cnt;
      it->Next();
    }
    ASSERT_EQ(static_cast<uint32_t>(cnt), num_items);
    delete it;

    Arena arena;
    it = reader.NewIterator(ReadOptions(), &arena);
    ASSERT_OK(it->status());
    ASSERT_TRUE(!it->Valid());
    it->Seek(keys[num_items/2]);
    ASSERT_TRUE(it->Valid());
    ASSERT_OK(it->status());
    ASSERT_TRUE(keys[num_items/2] == it->key());
    ASSERT_TRUE(values[num_items/2] == it->value());
    ASSERT_OK(it->status());
    it->~Iterator();
  }

  std::vector<std::string> keys;
  std::vector<std::string> user_keys;
  std::vector<std::string> values;
  uint64_t num_items;
  std::string fname;
  uint64_t file_size;
  Options options;
  Env* env;
  EnvOptions env_options;
};

TEST_F(CuckooReaderTest, WhenKeyExists) {
  SetUp(kNumHashFunc);
  fname = test::TmpDir() + "/CuckooReader_WhenKeyExists";
  for (uint64_t i = 0; i < num_items; i++) {
    user_keys[i] = "key" + NumToStr(i);
    ParsedInternalKey ikey(user_keys[i], i + 1000, kTypeValue);
    AppendInternalKey(&keys[i], ikey);
    values[i] = "value" + NumToStr(i);
    // Give disjoint hash values.
    AddHashLookups(user_keys[i], i, kNumHashFunc);
  }
  CreateCuckooFileAndCheckReader();
  // Last level file.
  UpdateKeys(true);
  CreateCuckooFileAndCheckReader();
  // Test with collision. Make all hash values collide.
  hash_map.clear();
  for (uint32_t i = 0; i < num_items; i++) {
    AddHashLookups(user_keys[i], 0, kNumHashFunc);
  }
  UpdateKeys(false);
  CreateCuckooFileAndCheckReader();
  // Last level file.
  UpdateKeys(true);
  CreateCuckooFileAndCheckReader();
}

TEST_F(CuckooReaderTest, WhenKeyExistsWithUint64Comparator) {
  SetUp(kNumHashFunc);
  fname = test::TmpDir() + "/CuckooReaderUint64_WhenKeyExists";
  for (uint64_t i = 0; i < num_items; i++) {
    user_keys[i].resize(8);
    memcpy(&user_keys[i][0], static_cast<void*>(&i), 8);
    ParsedInternalKey ikey(user_keys[i], i + 1000, kTypeValue);
    AppendInternalKey(&keys[i], ikey);
    values[i] = "value" + NumToStr(i);
    // Give disjoint hash values.
    AddHashLookups(user_keys[i], i, kNumHashFunc);
  }
  CreateCuckooFileAndCheckReader(test::Uint64Comparator());
  // Last level file.
  UpdateKeys(true);
  CreateCuckooFileAndCheckReader(test::Uint64Comparator());
  // Test with collision. Make all hash values collide.
  hash_map.clear();
  for (uint32_t i = 0; i < num_items; i++) {
    AddHashLookups(user_keys[i], 0, kNumHashFunc);
  }
  UpdateKeys(false);
  CreateCuckooFileAndCheckReader(test::Uint64Comparator());
  // Last level file.
  UpdateKeys(true);
  CreateCuckooFileAndCheckReader(test::Uint64Comparator());
}

TEST_F(CuckooReaderTest, CheckIterator) {
  SetUp(2*kNumHashFunc);
  fname = test::TmpDir() + "/CuckooReader_CheckIterator";
  for (uint64_t i = 0; i < num_items; i++) {
    user_keys[i] = "key" + NumToStr(i);
    ParsedInternalKey ikey(user_keys[i], 1000, kTypeValue);
    AppendInternalKey(&keys[i], ikey);
    values[i] = "value" + NumToStr(i);
    // Give disjoint hash values, in reverse order.
    AddHashLookups(user_keys[i], num_items-i-1, kNumHashFunc);
  }
  CreateCuckooFileAndCheckReader();
  CheckIterator();
  // Last level file.
  UpdateKeys(true);
  CreateCuckooFileAndCheckReader();
  CheckIterator();
}

TEST_F(CuckooReaderTest, CheckIteratorUint64) {
  SetUp(2*kNumHashFunc);
  fname = test::TmpDir() + "/CuckooReader_CheckIterator";
  for (uint64_t i = 0; i < num_items; i++) {
    user_keys[i].resize(8);
    memcpy(&user_keys[i][0], static_cast<void*>(&i), 8);
    ParsedInternalKey ikey(user_keys[i], 1000, kTypeValue);
    AppendInternalKey(&keys[i], ikey);
    values[i] = "value" + NumToStr(i);
    // Give disjoint hash values, in reverse order.
    AddHashLookups(user_keys[i], num_items-i-1, kNumHashFunc);
  }
  CreateCuckooFileAndCheckReader(test::Uint64Comparator());
  CheckIterator(test::Uint64Comparator());
  // Last level file.
  UpdateKeys(true);
  CreateCuckooFileAndCheckReader(test::Uint64Comparator());
  CheckIterator(test::Uint64Comparator());
}

TEST_F(CuckooReaderTest, WhenKeyNotFound) {
  // Add keys with colliding hash values.
  SetUp(kNumHashFunc);
  fname = test::TmpDir() + "/CuckooReader_WhenKeyNotFound";
  for (uint64_t i = 0; i < num_items; i++) {
    user_keys[i] = "key" + NumToStr(i);
    ParsedInternalKey ikey(user_keys[i], i + 1000, kTypeValue);
    AppendInternalKey(&keys[i], ikey);
    values[i] = "value" + NumToStr(i);
    // Make all hash values collide.
    AddHashLookups(user_keys[i], 0, kNumHashFunc);
  }
  auto* ucmp = BytewiseComparator();
  CreateCuckooFileAndCheckReader();
  std::unique_ptr<RandomAccessFile> read_file;
  ASSERT_OK(env->NewRandomAccessFile(fname, &read_file, env_options));
  unique_ptr<RandomAccessFileReader> file_reader(
      new RandomAccessFileReader(std::move(read_file)));
  const ImmutableCFOptions ioptions(options);
  CuckooTableReader reader(ioptions, std::move(file_reader), file_size, ucmp,
                           GetSliceHash);
  ASSERT_OK(reader.status());
  // Search for a key with colliding hash values.
  std::string not_found_user_key = "key" + NumToStr(num_items);
  std::string not_found_key;
  AddHashLookups(not_found_user_key, 0, kNumHashFunc);
  ParsedInternalKey ikey(not_found_user_key, 1000, kTypeValue);
  AppendInternalKey(&not_found_key, ikey);
  std::string value;
  GetContext get_context(ucmp, nullptr, nullptr, nullptr, GetContext::kNotFound,
                         Slice(not_found_key), &value, nullptr, nullptr,
                         nullptr);
  ASSERT_OK(reader.Get(ReadOptions(), Slice(not_found_key), &get_context));
  ASSERT_TRUE(value.empty());
  ASSERT_OK(reader.status());
  // Search for a key with an independent hash value.
  std::string not_found_user_key2 = "key" + NumToStr(num_items + 1);
  AddHashLookups(not_found_user_key2, kNumHashFunc, kNumHashFunc);
  ParsedInternalKey ikey2(not_found_user_key2, 1000, kTypeValue);
  std::string not_found_key2;
  AppendInternalKey(&not_found_key2, ikey2);
  GetContext get_context2(ucmp, nullptr, nullptr, nullptr,
                          GetContext::kNotFound, Slice(not_found_key2), &value,
                          nullptr, nullptr, nullptr);
  ASSERT_OK(reader.Get(ReadOptions(), Slice(not_found_key2), &get_context2));
  ASSERT_TRUE(value.empty());
  ASSERT_OK(reader.status());

  // Test read when key is unused key.
  std::string unused_key =
    reader.GetTableProperties()->user_collected_properties.at(
    CuckooTablePropertyNames::kEmptyKey);
  // Add hash values that map to empty buckets.
  AddHashLookups(ExtractUserKey(unused_key).ToString(),
      kNumHashFunc, kNumHashFunc);
  GetContext get_context3(ucmp, nullptr, nullptr, nullptr,
                          GetContext::kNotFound, Slice(unused_key), &value,
                          nullptr, nullptr, nullptr);
  ASSERT_OK(reader.Get(ReadOptions(), Slice(unused_key), &get_context3));
  ASSERT_TRUE(value.empty());
  ASSERT_OK(reader.status());
}

// Performance tests
namespace {
void GetKeys(uint64_t num, std::vector<std::string>* keys) {
  keys->clear();
  IterKey k;
  k.SetInternalKey("", 0, kTypeValue);
  std::string internal_key_suffix = k.GetKey().ToString();
  ASSERT_EQ(static_cast<size_t>(8), internal_key_suffix.size());
  for (uint64_t key_idx = 0; key_idx < num; ++key_idx) {
    uint64_t value = 2 * key_idx;
    std::string new_key(reinterpret_cast<char*>(&value), sizeof(value));
    new_key += internal_key_suffix;
    keys->push_back(new_key);
  }
}

std::string GetFileName(uint64_t num) {
  if (FLAGS_file_dir.empty()) {
    FLAGS_file_dir = test::TmpDir();
  }
  return FLAGS_file_dir + "/cuckoo_read_benchmark" +
    ToString(num/1000000) + "Mkeys";
}

// Create last level file as we are interested in measuring performance of
// last level file only.
void WriteFile(const std::vector<std::string>& keys,
    const uint64_t num, double hash_ratio) {
  Options options;
  options.allow_mmap_reads = true;
  Env* env = options.env;
  EnvOptions env_options = EnvOptions(options);
  std::string fname = GetFileName(num);

  std::unique_ptr<WritableFile> writable_file;
  ASSERT_OK(env->NewWritableFile(fname, &writable_file, env_options));
  unique_ptr<WritableFileWriter> file_writer(
      new WritableFileWriter(std::move(writable_file), env_options));
  CuckooTableBuilder builder(file_writer.get(), hash_ratio, 64, 1000,
                             test::Uint64Comparator(), 5, false,
                             FLAGS_identity_as_first_hash, nullptr);
  ASSERT_OK(builder.status());
  for (uint64_t key_idx = 0; key_idx < num; ++key_idx) {
    // Value is just a part of key.
    builder.Add(Slice(keys[key_idx]), Slice(&keys[key_idx][0], 4));
    ASSERT_EQ(builder.NumEntries(), key_idx + 1);
    ASSERT_OK(builder.status());
  }
  ASSERT_OK(builder.Finish());
  ASSERT_EQ(num, builder.NumEntries());
  ASSERT_OK(file_writer->Close());

  uint64_t file_size;
  env->GetFileSize(fname, &file_size);
  std::unique_ptr<RandomAccessFile> read_file;
  ASSERT_OK(env->NewRandomAccessFile(fname, &read_file, env_options));
  unique_ptr<RandomAccessFileReader> file_reader(
      new RandomAccessFileReader(std::move(read_file)));

  const ImmutableCFOptions ioptions(options);
  CuckooTableReader reader(ioptions, std::move(file_reader), file_size,
                           test::Uint64Comparator(), nullptr);
  ASSERT_OK(reader.status());
  ReadOptions r_options;
  std::string value;
  // Assume only the fast path is triggered
  GetContext get_context(nullptr, nullptr, nullptr, nullptr,
                         GetContext::kNotFound, Slice(), &value, nullptr,
                         nullptr, nullptr);
  for (uint64_t i = 0; i < num; ++i) {
    value.clear();
    ASSERT_OK(reader.Get(r_options, Slice(keys[i]), &get_context));
    ASSERT_TRUE(Slice(keys[i]) == Slice(&keys[i][0], 4));
  }
}

void ReadKeys(uint64_t num, uint32_t batch_size) {
  Options options;
  options.allow_mmap_reads = true;
  Env* env = options.env;
  EnvOptions env_options = EnvOptions(options);
  std::string fname = GetFileName(num);

  uint64_t file_size;
  env->GetFileSize(fname, &file_size);
  std::unique_ptr<RandomAccessFile> read_file;
  ASSERT_OK(env->NewRandomAccessFile(fname, &read_file, env_options));
  unique_ptr<RandomAccessFileReader> file_reader(
      new RandomAccessFileReader(std::move(read_file)));

  const ImmutableCFOptions ioptions(options);
  CuckooTableReader reader(ioptions, std::move(file_reader), file_size,
                           test::Uint64Comparator(), nullptr);
  ASSERT_OK(reader.status());
  const UserCollectedProperties user_props =
    reader.GetTableProperties()->user_collected_properties;
  const uint32_t num_hash_fun = *reinterpret_cast<const uint32_t*>(
      user_props.at(CuckooTablePropertyNames::kNumHashFunc).data());
  const uint64_t table_size = *reinterpret_cast<const uint64_t*>(
      user_props.at(CuckooTablePropertyNames::kHashTableSize).data());
  fprintf(stderr, "With %" PRIu64 " items, utilization is %.2f%%, number of"
      " hash functions: %u.\n", num, num * 100.0 / (table_size), num_hash_fun);
  ReadOptions r_options;

  std::vector<uint64_t> keys;
  keys.reserve(num);
  for (uint64_t i = 0; i < num; ++i) {
    keys.push_back(2 * i);
  }
  std::random_shuffle(keys.begin(), keys.end());

  std::string value;
  // Assume only the fast path is triggered
  GetContext get_context(nullptr, nullptr, nullptr, nullptr,
                         GetContext::kNotFound, Slice(), &value, nullptr,
                         nullptr, nullptr);
  uint64_t start_time = env->NowMicros();
  if (batch_size > 0) {
    for (uint64_t i = 0; i < num; i += batch_size) {
      for (uint64_t j = i; j < i+batch_size && j < num; ++j) {
        reader.Prepare(Slice(reinterpret_cast<char*>(&keys[j]), 16));
      }
      for (uint64_t j = i; j < i+batch_size && j < num; ++j) {
        reader.Get(r_options, Slice(reinterpret_cast<char*>(&keys[j]), 16),
                   &get_context);
      }
    }
  } else {
    for (uint64_t i = 0; i < num; i++) {
      reader.Get(r_options, Slice(reinterpret_cast<char*>(&keys[i]), 16),
                 &get_context);
    }
  }
  float time_per_op = (env->NowMicros() - start_time) * 1.0 / num;
  fprintf(stderr,
      "Time taken per op is %.3fus (%.1f Mqps) with batch size of %u\n",
      time_per_op, 1.0 / time_per_op, batch_size);
}
}  // namespace.

TEST_F(CuckooReaderTest, TestReadPerformance) {
  if (!FLAGS_enable_perf) {
    return;
  }
  double hash_ratio = 0.95;
  // These numbers are chosen to have a hash utilizaiton % close to
  // 0.9, 0.75, 0.6 and 0.5 respectively.
  // They all create 128 M buckets.
  std::vector<uint64_t> nums = {120*1024*1024, 100*1024*1024, 80*1024*1024,
    70*1024*1024};
#ifndef NDEBUG
  fprintf(stdout,
      "WARNING: Not compiled with DNDEBUG. Performance tests may be slow.\n");
#endif
  for (uint64_t num : nums) {
    if (FLAGS_write ||
        Env::Default()->FileExists(GetFileName(num)).IsNotFound()) {
      std::vector<std::string> all_keys;
      GetKeys(num, &all_keys);
      WriteFile(all_keys, num, hash_ratio);
    }
    ReadKeys(num, 0);
    ReadKeys(num, 10);
    ReadKeys(num, 25);
    ReadKeys(num, 50);
    ReadKeys(num, 100);
    fprintf(stderr, "\n");
  }
}
}  // namespace rocksdb

int main(int argc, char** argv) {
  ::testing::InitGoogleTest(&argc, argv);
  ParseCommandLineFlags(&argc, &argv, true);
  return RUN_ALL_TESTS();
}

#endif  // GFLAGS.

#else
#include <stdio.h>

int main(int argc, char** argv) {
  fprintf(stderr, "SKIPPED as Cuckoo table is not supported in ROCKSDB_LITE\n");
  return 0;
}

#endif  // ROCKSDB_LITE
