//  Copyright (c) 2013, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.
//
// Copyright (c) 2011 The LevelDB Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file. See the AUTHORS file for names of contributors.

#include <inttypes.h>
#include <stdio.h>

#include <algorithm>
#include <iostream>
#include <map>
#include <memory>
#include <string>
#include <vector>

#include "db/dbformat.h"
#include "db/memtable.h"
#include "db/write_batch_internal.h"
#include "db/writebuffer.h"
#include "rocksdb/cache.h"
#include "rocksdb/db.h"
#include "rocksdb/env.h"
#include "rocksdb/iterator.h"
#include "rocksdb/memtablerep.h"
#include "rocksdb/perf_context.h"
#include "rocksdb/slice_transform.h"
#include "rocksdb/statistics.h"
#include "table/block.h"
#include "table/block_based_table_builder.h"
#include "table/block_based_table_factory.h"
#include "table/block_based_table_reader.h"
#include "table/block_builder.h"
#include "table/format.h"
#include "table/get_context.h"
#include "table/meta_blocks.h"
#include "table/plain_table_factory.h"
#include "util/compression.h"
#include "util/random.h"
#include "util/scoped_arena_iterator.h"
#include "util/statistics.h"
#include "util/stl_wrappers.h"
#include "util/string_util.h"
#include "util/testharness.h"
#include "util/testutil.h"

namespace rocksdb {

extern const uint64_t kLegacyBlockBasedTableMagicNumber;
extern const uint64_t kLegacyPlainTableMagicNumber;
extern const uint64_t kBlockBasedTableMagicNumber;
extern const uint64_t kPlainTableMagicNumber;

namespace {

// Return reverse of "key".
// Used to test non-lexicographic comparators.
std::string Reverse(const Slice& key) {
  auto rev = key.ToString();
  std::reverse(rev.begin(), rev.end());
  return rev;
}

class ReverseKeyComparator : public Comparator {
 public:
  virtual const char* Name() const override {
    return "rocksdb.ReverseBytewiseComparator";
  }

  virtual int Compare(const Slice& a, const Slice& b) const override {
    return BytewiseComparator()->Compare(Reverse(a), Reverse(b));
  }

  virtual void FindShortestSeparator(std::string* start,
                                     const Slice& limit) const override {
    std::string s = Reverse(*start);
    std::string l = Reverse(limit);
    BytewiseComparator()->FindShortestSeparator(&s, l);
    *start = Reverse(s);
  }

  virtual void FindShortSuccessor(std::string* key) const override {
    std::string s = Reverse(*key);
    BytewiseComparator()->FindShortSuccessor(&s);
    *key = Reverse(s);
  }
};

ReverseKeyComparator reverse_key_comparator;

void Increment(const Comparator* cmp, std::string* key) {
  if (cmp == BytewiseComparator()) {
    key->push_back('\0');
  } else {
    assert(cmp == &reverse_key_comparator);
    std::string rev = Reverse(*key);
    rev.push_back('\0');
    *key = Reverse(rev);
  }
}

}  // namespace

// Helper class for tests to unify the interface between
// BlockBuilder/TableBuilder and Block/Table.
class Constructor {
 public:
  explicit Constructor(const Comparator* cmp)
      : data_(stl_wrappers::LessOfComparator(cmp)) {}
  virtual ~Constructor() { }

  void Add(const std::string& key, const Slice& value) {
    data_[key] = value.ToString();
  }

  // Finish constructing the data structure with all the keys that have
  // been added so far.  Returns the keys in sorted order in "*keys"
  // and stores the key/value pairs in "*kvmap"
  void Finish(const Options& options, const ImmutableCFOptions& ioptions,
              const BlockBasedTableOptions& table_options,
              const InternalKeyComparator& internal_comparator,
              std::vector<std::string>* keys, stl_wrappers::KVMap* kvmap) {
    last_internal_key_ = &internal_comparator;
    *kvmap = data_;
    keys->clear();
    for (const auto& kv : data_) {
      keys->push_back(kv.first);
    }
    data_.clear();
    Status s = FinishImpl(options, ioptions, table_options,
                          internal_comparator, *kvmap);
    ASSERT_TRUE(s.ok()) << s.ToString();
  }

  // Construct the data structure from the data in "data"
  virtual Status FinishImpl(const Options& options,
                            const ImmutableCFOptions& ioptions,
                            const BlockBasedTableOptions& table_options,
                            const InternalKeyComparator& internal_comparator,
                            const stl_wrappers::KVMap& data) = 0;

  virtual Iterator* NewIterator() const = 0;

  virtual const stl_wrappers::KVMap& data() { return data_; }

  virtual bool IsArenaMode() const { return false; }

  virtual DB* db() const { return nullptr; }  // Overridden in DBConstructor

  virtual bool AnywayDeleteIterator() const { return false; }

 protected:
  const InternalKeyComparator* last_internal_key_;

 private:
  stl_wrappers::KVMap data_;
};

class BlockConstructor: public Constructor {
 public:
  explicit BlockConstructor(const Comparator* cmp)
      : Constructor(cmp),
        comparator_(cmp),
        block_(nullptr) { }
  ~BlockConstructor() {
    delete block_;
  }
  virtual Status FinishImpl(const Options& options,
                            const ImmutableCFOptions& ioptions,
                            const BlockBasedTableOptions& table_options,
                            const InternalKeyComparator& internal_comparator,
                            const stl_wrappers::KVMap& kv_map) override {
    delete block_;
    block_ = nullptr;
    BlockBuilder builder(table_options.block_restart_interval);

    for (const auto kv : kv_map) {
      builder.Add(kv.first, kv.second);
    }
    // Open the block
    data_ = builder.Finish().ToString();
    BlockContents contents;
    contents.data = data_;
    contents.cachable = false;
    block_ = new Block(std::move(contents));
    return Status::OK();
  }
  virtual Iterator* NewIterator() const override {
    return block_->NewIterator(comparator_);
  }

 private:
  const Comparator* comparator_;
  std::string data_;
  Block* block_;

  BlockConstructor();
};

// A helper class that converts internal format keys into user keys
class KeyConvertingIterator: public Iterator {
 public:
  KeyConvertingIterator(Iterator* iter, bool arena_mode = false)
      : iter_(iter), arena_mode_(arena_mode) {}
  virtual ~KeyConvertingIterator() {
    if (arena_mode_) {
      iter_->~Iterator();
    } else {
      delete iter_;
    }
  }
  virtual bool Valid() const override { return iter_->Valid(); }
  virtual void Seek(const Slice& target) override {
    ParsedInternalKey ikey(target, kMaxSequenceNumber, kTypeValue);
    std::string encoded;
    AppendInternalKey(&encoded, ikey);
    iter_->Seek(encoded);
  }
  virtual void SeekToFirst() override { iter_->SeekToFirst(); }
  virtual void SeekToLast() override { iter_->SeekToLast(); }
  virtual void Next() override { iter_->Next(); }
  virtual void Prev() override { iter_->Prev(); }

  virtual Slice key() const override {
    assert(Valid());
    ParsedInternalKey parsed_key;
    if (!ParseInternalKey(iter_->key(), &parsed_key)) {
      status_ = Status::Corruption("malformed internal key");
      return Slice("corrupted key");
    }
    return parsed_key.user_key;
  }

  virtual Slice value() const override { return iter_->value(); }
  virtual Status status() const override {
    return status_.ok() ? iter_->status() : status_;
  }

 private:
  mutable Status status_;
  Iterator* iter_;
  bool arena_mode_;

  // No copying allowed
  KeyConvertingIterator(const KeyConvertingIterator&);
  void operator=(const KeyConvertingIterator&);
};

class TableConstructor: public Constructor {
 public:
  explicit TableConstructor(const Comparator* cmp,
                            bool convert_to_internal_key = false)
      : Constructor(cmp),
        convert_to_internal_key_(convert_to_internal_key) {}
  ~TableConstructor() { Reset(); }

  virtual Status FinishImpl(const Options& options,
                            const ImmutableCFOptions& ioptions,
                            const BlockBasedTableOptions& table_options,
                            const InternalKeyComparator& internal_comparator,
                            const stl_wrappers::KVMap& kv_map) override {
    Reset();
    file_writer_.reset(test::GetWritableFileWriter(new test::StringSink()));
    unique_ptr<TableBuilder> builder;
    std::vector<std::unique_ptr<IntTblPropCollectorFactory>>
        int_tbl_prop_collector_factories;
    builder.reset(ioptions.table_factory->NewTableBuilder(
        TableBuilderOptions(ioptions, internal_comparator,
                            &int_tbl_prop_collector_factories,
                            options.compression, CompressionOptions(), false),
        file_writer_.get()));

    for (const auto kv : kv_map) {
      if (convert_to_internal_key_) {
        ParsedInternalKey ikey(kv.first, kMaxSequenceNumber, kTypeValue);
        std::string encoded;
        AppendInternalKey(&encoded, ikey);
        builder->Add(encoded, kv.second);
      } else {
        builder->Add(kv.first, kv.second);
      }
      EXPECT_TRUE(builder->status().ok());
    }
    Status s = builder->Finish();
    file_writer_->Flush();
    EXPECT_TRUE(s.ok()) << s.ToString();

    EXPECT_EQ(GetSink()->contents().size(), builder->FileSize());

    // Open the table
    uniq_id_ = cur_uniq_id_++;
    file_reader_.reset(test::GetRandomAccessFileReader(new test::StringSource(
        GetSink()->contents(), uniq_id_, ioptions.allow_mmap_reads)));
    return ioptions.table_factory->NewTableReader(
        ioptions, soptions, internal_comparator, std::move(file_reader_),
        GetSink()->contents().size(), &table_reader_);
  }

  virtual Iterator* NewIterator() const override {
    ReadOptions ro;
    Iterator* iter = table_reader_->NewIterator(ro);
    if (convert_to_internal_key_) {
      return new KeyConvertingIterator(iter);
    } else {
      return iter;
    }
  }

  uint64_t ApproximateOffsetOf(const Slice& key) const {
    return table_reader_->ApproximateOffsetOf(key);
  }

  virtual Status Reopen(const ImmutableCFOptions& ioptions) {
    file_reader_.reset(test::GetRandomAccessFileReader(new test::StringSource(
        GetSink()->contents(), uniq_id_, ioptions.allow_mmap_reads)));
    return ioptions.table_factory->NewTableReader(
        ioptions, soptions, *last_internal_key_, std::move(file_reader_),
        GetSink()->contents().size(), &table_reader_);
  }

  virtual TableReader* GetTableReader() {
    return table_reader_.get();
  }

  virtual bool AnywayDeleteIterator() const override {
    return convert_to_internal_key_;
  }

 private:
  void Reset() {
    uniq_id_ = 0;
    table_reader_.reset();
    file_writer_.reset();
    file_reader_.reset();
  }

  test::StringSink* GetSink() {
    return static_cast<test::StringSink*>(file_writer_->writable_file());
  }

  uint64_t uniq_id_;
  unique_ptr<WritableFileWriter> file_writer_;
  unique_ptr<RandomAccessFileReader> file_reader_;
  unique_ptr<TableReader> table_reader_;
  bool convert_to_internal_key_;

  TableConstructor();

  static uint64_t cur_uniq_id_;
  const EnvOptions soptions;
};
uint64_t TableConstructor::cur_uniq_id_ = 1;

class MemTableConstructor: public Constructor {
 public:
  explicit MemTableConstructor(const Comparator* cmp, WriteBuffer* wb)
      : Constructor(cmp),
        internal_comparator_(cmp),
        write_buffer_(wb),
        table_factory_(new SkipListFactory) {
    options_.memtable_factory = table_factory_;
    ImmutableCFOptions ioptions(options_);
    memtable_ = new MemTable(internal_comparator_, ioptions,
                             MutableCFOptions(options_, ioptions), wb,
                             kMaxSequenceNumber);
    memtable_->Ref();
  }
  ~MemTableConstructor() {
    delete memtable_->Unref();
  }
  virtual Status FinishImpl(const Options&, const ImmutableCFOptions& ioptions,
                            const BlockBasedTableOptions& table_options,
                            const InternalKeyComparator& internal_comparator,
                            const stl_wrappers::KVMap& kv_map) override {
    delete memtable_->Unref();
    ImmutableCFOptions mem_ioptions(ioptions);
    memtable_ = new MemTable(internal_comparator_, mem_ioptions,
                             MutableCFOptions(options_, mem_ioptions),
                             write_buffer_, kMaxSequenceNumber);
    memtable_->Ref();
    int seq = 1;
    for (const auto kv : kv_map) {
      memtable_->Add(seq, kTypeValue, kv.first, kv.second);
      seq++;
    }
    return Status::OK();
  }
  virtual Iterator* NewIterator() const override {
    return new KeyConvertingIterator(
        memtable_->NewIterator(ReadOptions(), &arena_), true);
  }

  virtual bool AnywayDeleteIterator() const override { return true; }

  virtual bool IsArenaMode() const override { return true; }

 private:
  mutable Arena arena_;
  InternalKeyComparator internal_comparator_;
  Options options_;
  WriteBuffer* write_buffer_;
  MemTable* memtable_;
  std::shared_ptr<SkipListFactory> table_factory_;
};

class DBConstructor: public Constructor {
 public:
  explicit DBConstructor(const Comparator* cmp)
      : Constructor(cmp),
        comparator_(cmp) {
    db_ = nullptr;
    NewDB();
  }
  ~DBConstructor() {
    delete db_;
  }
  virtual Status FinishImpl(const Options& options,
                            const ImmutableCFOptions& ioptions,
                            const BlockBasedTableOptions& table_options,
                            const InternalKeyComparator& internal_comparator,
                            const stl_wrappers::KVMap& kv_map) override {
    delete db_;
    db_ = nullptr;
    NewDB();
    for (const auto kv : kv_map) {
      WriteBatch batch;
      batch.Put(kv.first, kv.second);
      EXPECT_TRUE(db_->Write(WriteOptions(), &batch).ok());
    }
    return Status::OK();
  }
  virtual Iterator* NewIterator() const override {
    return db_->NewIterator(ReadOptions());
  }

  virtual DB* db() const override { return db_; }

 private:
  void NewDB() {
    std::string name = test::TmpDir() + "/table_testdb";

    Options options;
    options.comparator = comparator_;
    Status status = DestroyDB(name, options);
    ASSERT_TRUE(status.ok()) << status.ToString();

    options.create_if_missing = true;
    options.error_if_exists = true;
    options.write_buffer_size = 10000;  // Something small to force merging
    status = DB::Open(options, name, &db_);
    ASSERT_TRUE(status.ok()) << status.ToString();
  }

  const Comparator* comparator_;
  DB* db_;
};

enum TestType {
  BLOCK_BASED_TABLE_TEST,
#ifndef ROCKSDB_LITE
  PLAIN_TABLE_SEMI_FIXED_PREFIX,
  PLAIN_TABLE_FULL_STR_PREFIX,
  PLAIN_TABLE_TOTAL_ORDER,
#endif  // !ROCKSDB_LITE
  BLOCK_TEST,
  MEMTABLE_TEST,
  DB_TEST
};

struct TestArgs {
  TestType type;
  bool reverse_compare;
  int restart_interval;
  CompressionType compression;
  uint32_t format_version;
};

static std::vector<TestArgs> GenerateArgList() {
  std::vector<TestArgs> test_args;
  std::vector<TestType> test_types = {
      BLOCK_BASED_TABLE_TEST,
#ifndef ROCKSDB_LITE
      PLAIN_TABLE_SEMI_FIXED_PREFIX,
      PLAIN_TABLE_FULL_STR_PREFIX,
      PLAIN_TABLE_TOTAL_ORDER,
#endif  // !ROCKSDB_LITE
      BLOCK_TEST,
      MEMTABLE_TEST, DB_TEST};
  std::vector<bool> reverse_compare_types = {false, true};
  std::vector<int> restart_intervals = {16, 1, 1024};

  // Only add compression if it is supported
  std::vector<std::pair<CompressionType, bool>> compression_types;
  compression_types.emplace_back(kNoCompression, false);
  if (Snappy_Supported()) {
    compression_types.emplace_back(kSnappyCompression, false);
  }
  if (Zlib_Supported()) {
    compression_types.emplace_back(kZlibCompression, false);
    compression_types.emplace_back(kZlibCompression, true);
  }
  if (BZip2_Supported()) {
    compression_types.emplace_back(kBZip2Compression, false);
    compression_types.emplace_back(kBZip2Compression, true);
  }
  if (LZ4_Supported()) {
    compression_types.emplace_back(kLZ4Compression, false);
    compression_types.emplace_back(kLZ4Compression, true);
    compression_types.emplace_back(kLZ4HCCompression, false);
    compression_types.emplace_back(kLZ4HCCompression, true);
  }
  if (ZSTD_Supported()) {
    compression_types.emplace_back(kZSTDNotFinalCompression, false);
    compression_types.emplace_back(kZSTDNotFinalCompression, true);
  }

  for (auto test_type : test_types) {
    for (auto reverse_compare : reverse_compare_types) {
#ifndef ROCKSDB_LITE
      if (test_type == PLAIN_TABLE_SEMI_FIXED_PREFIX ||
          test_type == PLAIN_TABLE_FULL_STR_PREFIX) {
        // Plain table doesn't use restart index or compression.
        TestArgs one_arg;
        one_arg.type = test_type;
        one_arg.reverse_compare = reverse_compare;
        one_arg.restart_interval = restart_intervals[0];
        one_arg.compression = compression_types[0].first;
        test_args.push_back(one_arg);
        continue;
      }
#endif  // !ROCKSDB_LITE

      for (auto restart_interval : restart_intervals) {
        for (auto compression_type : compression_types) {
          TestArgs one_arg;
          one_arg.type = test_type;
          one_arg.reverse_compare = reverse_compare;
          one_arg.restart_interval = restart_interval;
          one_arg.compression = compression_type.first;
          one_arg.format_version = compression_type.second ? 2 : 1;
          test_args.push_back(one_arg);
        }
      }
    }
  }
  return test_args;
}

// In order to make all tests run for plain table format, including
// those operating on empty keys, create a new prefix transformer which
// return fixed prefix if the slice is not shorter than the prefix length,
// and the full slice if it is shorter.
class FixedOrLessPrefixTransform : public SliceTransform {
 private:
  const size_t prefix_len_;

 public:
  explicit FixedOrLessPrefixTransform(size_t prefix_len) :
      prefix_len_(prefix_len) {
  }

  virtual const char* Name() const override { return "rocksdb.FixedPrefix"; }

  virtual Slice Transform(const Slice& src) const override {
    assert(InDomain(src));
    if (src.size() < prefix_len_) {
      return src;
    }
    return Slice(src.data(), prefix_len_);
  }

  virtual bool InDomain(const Slice& src) const override { return true; }

  virtual bool InRange(const Slice& dst) const override {
    return (dst.size() <= prefix_len_);
  }
};

class HarnessTest : public testing::Test {
 public:
  HarnessTest()
      : ioptions_(options_),
        constructor_(nullptr),
        write_buffer_(options_.db_write_buffer_size) {}

  void Init(const TestArgs& args) {
    delete constructor_;
    constructor_ = nullptr;
    options_ = Options();
    options_.compression = args.compression;
    // Use shorter block size for tests to exercise block boundary
    // conditions more.
    if (args.reverse_compare) {
      options_.comparator = &reverse_key_comparator;
    }

    internal_comparator_.reset(
        new test::PlainInternalKeyComparator(options_.comparator));

    support_prev_ = true;
    only_support_prefix_seek_ = false;
    switch (args.type) {
      case BLOCK_BASED_TABLE_TEST:
        table_options_.flush_block_policy_factory.reset(
            new FlushBlockBySizePolicyFactory());
        table_options_.block_size = 256;
        table_options_.block_restart_interval = args.restart_interval;
        table_options_.format_version = args.format_version;
        options_.table_factory.reset(
            new BlockBasedTableFactory(table_options_));
        constructor_ = new TableConstructor(options_.comparator);
        break;
// Plain table is not supported in ROCKSDB_LITE
#ifndef ROCKSDB_LITE
      case PLAIN_TABLE_SEMI_FIXED_PREFIX:
        support_prev_ = false;
        only_support_prefix_seek_ = true;
        options_.prefix_extractor.reset(new FixedOrLessPrefixTransform(2));
        options_.allow_mmap_reads = true;
        options_.table_factory.reset(NewPlainTableFactory());
        constructor_ = new TableConstructor(options_.comparator, true);
        internal_comparator_.reset(
            new InternalKeyComparator(options_.comparator));
        break;
      case PLAIN_TABLE_FULL_STR_PREFIX:
        support_prev_ = false;
        only_support_prefix_seek_ = true;
        options_.prefix_extractor.reset(NewNoopTransform());
        options_.allow_mmap_reads = true;
        options_.table_factory.reset(NewPlainTableFactory());
        constructor_ = new TableConstructor(options_.comparator, true);
        internal_comparator_.reset(
            new InternalKeyComparator(options_.comparator));
        break;
      case PLAIN_TABLE_TOTAL_ORDER:
        support_prev_ = false;
        only_support_prefix_seek_ = false;
        options_.prefix_extractor = nullptr;
        options_.allow_mmap_reads = true;

        {
          PlainTableOptions plain_table_options;
          plain_table_options.user_key_len = kPlainTableVariableLength;
          plain_table_options.bloom_bits_per_key = 0;
          plain_table_options.hash_table_ratio = 0;

          options_.table_factory.reset(
              NewPlainTableFactory(plain_table_options));
        }
        constructor_ = new TableConstructor(options_.comparator, true);
        internal_comparator_.reset(
            new InternalKeyComparator(options_.comparator));
        break;
#endif  // !ROCKSDB_LITE
      case BLOCK_TEST:
        table_options_.block_size = 256;
        options_.table_factory.reset(
            new BlockBasedTableFactory(table_options_));
        constructor_ = new BlockConstructor(options_.comparator);
        break;
      case MEMTABLE_TEST:
        table_options_.block_size = 256;
        options_.table_factory.reset(
            new BlockBasedTableFactory(table_options_));
        constructor_ = new MemTableConstructor(options_.comparator,
                                               &write_buffer_);
        break;
      case DB_TEST:
        table_options_.block_size = 256;
        options_.table_factory.reset(
            new BlockBasedTableFactory(table_options_));
        constructor_ = new DBConstructor(options_.comparator);
        break;
    }
    ioptions_ = ImmutableCFOptions(options_);
  }

  ~HarnessTest() { delete constructor_; }

  void Add(const std::string& key, const std::string& value) {
    constructor_->Add(key, value);
  }

  void Test(Random* rnd) {
    std::vector<std::string> keys;
    stl_wrappers::KVMap data;
    constructor_->Finish(options_, ioptions_, table_options_,
                         *internal_comparator_, &keys, &data);

    TestForwardScan(keys, data);
    if (support_prev_) {
      TestBackwardScan(keys, data);
    }
    TestRandomAccess(rnd, keys, data);
  }

  void TestForwardScan(const std::vector<std::string>& keys,
                       const stl_wrappers::KVMap& data) {
    Iterator* iter = constructor_->NewIterator();
    ASSERT_TRUE(!iter->Valid());
    iter->SeekToFirst();
    for (stl_wrappers::KVMap::const_iterator model_iter = data.begin();
         model_iter != data.end(); ++model_iter) {
      ASSERT_EQ(ToString(data, model_iter), ToString(iter));
      iter->Next();
    }
    ASSERT_TRUE(!iter->Valid());
    if (constructor_->IsArenaMode() && !constructor_->AnywayDeleteIterator()) {
      iter->~Iterator();
    } else {
      delete iter;
    }
  }

  void TestBackwardScan(const std::vector<std::string>& keys,
                        const stl_wrappers::KVMap& data) {
    Iterator* iter = constructor_->NewIterator();
    ASSERT_TRUE(!iter->Valid());
    iter->SeekToLast();
    for (stl_wrappers::KVMap::const_reverse_iterator model_iter = data.rbegin();
         model_iter != data.rend(); ++model_iter) {
      ASSERT_EQ(ToString(data, model_iter), ToString(iter));
      iter->Prev();
    }
    ASSERT_TRUE(!iter->Valid());
    if (constructor_->IsArenaMode() && !constructor_->AnywayDeleteIterator()) {
      iter->~Iterator();
    } else {
      delete iter;
    }
  }

  void TestRandomAccess(Random* rnd, const std::vector<std::string>& keys,
                        const stl_wrappers::KVMap& data) {
    static const bool kVerbose = false;
    Iterator* iter = constructor_->NewIterator();
    ASSERT_TRUE(!iter->Valid());
    stl_wrappers::KVMap::const_iterator model_iter = data.begin();
    if (kVerbose) fprintf(stderr, "---\n");
    for (int i = 0; i < 200; i++) {
      const int toss = rnd->Uniform(support_prev_ ? 5 : 3);
      switch (toss) {
        case 0: {
          if (iter->Valid()) {
            if (kVerbose) fprintf(stderr, "Next\n");
            iter->Next();
            ++model_iter;
            ASSERT_EQ(ToString(data, model_iter), ToString(iter));
          }
          break;
        }

        case 1: {
          if (kVerbose) fprintf(stderr, "SeekToFirst\n");
          iter->SeekToFirst();
          model_iter = data.begin();
          ASSERT_EQ(ToString(data, model_iter), ToString(iter));
          break;
        }

        case 2: {
          std::string key = PickRandomKey(rnd, keys);
          model_iter = data.lower_bound(key);
          if (kVerbose) fprintf(stderr, "Seek '%s'\n",
                                EscapeString(key).c_str());
          iter->Seek(Slice(key));
          ASSERT_EQ(ToString(data, model_iter), ToString(iter));
          break;
        }

        case 3: {
          if (iter->Valid()) {
            if (kVerbose) fprintf(stderr, "Prev\n");
            iter->Prev();
            if (model_iter == data.begin()) {
              model_iter = data.end();   // Wrap around to invalid value
            } else {
              --model_iter;
            }
            ASSERT_EQ(ToString(data, model_iter), ToString(iter));
          }
          break;
        }

        case 4: {
          if (kVerbose) fprintf(stderr, "SeekToLast\n");
          iter->SeekToLast();
          if (keys.empty()) {
            model_iter = data.end();
          } else {
            std::string last = data.rbegin()->first;
            model_iter = data.lower_bound(last);
          }
          ASSERT_EQ(ToString(data, model_iter), ToString(iter));
          break;
        }
      }
    }
    if (constructor_->IsArenaMode() && !constructor_->AnywayDeleteIterator()) {
      iter->~Iterator();
    } else {
      delete iter;
    }
  }

  std::string ToString(const stl_wrappers::KVMap& data,
                       const stl_wrappers::KVMap::const_iterator& it) {
    if (it == data.end()) {
      return "END";
    } else {
      return "'" + it->first + "->" + it->second + "'";
    }
  }

  std::string ToString(const stl_wrappers::KVMap& data,
                       const stl_wrappers::KVMap::const_reverse_iterator& it) {
    if (it == data.rend()) {
      return "END";
    } else {
      return "'" + it->first + "->" + it->second + "'";
    }
  }

  std::string ToString(const Iterator* it) {
    if (!it->Valid()) {
      return "END";
    } else {
      return "'" + it->key().ToString() + "->" + it->value().ToString() + "'";
    }
  }

  std::string PickRandomKey(Random* rnd, const std::vector<std::string>& keys) {
    if (keys.empty()) {
      return "foo";
    } else {
      const int index = rnd->Uniform(static_cast<int>(keys.size()));
      std::string result = keys[index];
      switch (rnd->Uniform(support_prev_ ? 3 : 1)) {
        case 0:
          // Return an existing key
          break;
        case 1: {
          // Attempt to return something smaller than an existing key
          if (result.size() > 0 && result[result.size() - 1] > '\0'
              && (!only_support_prefix_seek_
                  || options_.prefix_extractor->Transform(result).size()
                  < result.size())) {
            result[result.size() - 1]--;
          }
          break;
      }
        case 2: {
          // Return something larger than an existing key
          Increment(options_.comparator, &result);
          break;
        }
      }
      return result;
    }
  }

  // Returns nullptr if not running against a DB
  DB* db() const { return constructor_->db(); }

 private:
  Options options_ = Options();
  ImmutableCFOptions ioptions_;
  BlockBasedTableOptions table_options_ = BlockBasedTableOptions();
  Constructor* constructor_;
  WriteBuffer write_buffer_;
  bool support_prev_;
  bool only_support_prefix_seek_;
  shared_ptr<InternalKeyComparator> internal_comparator_;
};

static bool Between(uint64_t val, uint64_t low, uint64_t high) {
  bool result = (val >= low) && (val <= high);
  if (!result) {
    fprintf(stderr, "Value %llu is not in range [%llu, %llu]\n",
            (unsigned long long)(val),
            (unsigned long long)(low),
            (unsigned long long)(high));
  }
  return result;
}

// Tests against all kinds of tables
class TableTest : public testing::Test {
 public:
  const InternalKeyComparator& GetPlainInternalComparator(
      const Comparator* comp) {
    if (!plain_internal_comparator) {
      plain_internal_comparator.reset(
          new test::PlainInternalKeyComparator(comp));
    }
    return *plain_internal_comparator;
  }

 private:
  std::unique_ptr<InternalKeyComparator> plain_internal_comparator;
};

class GeneralTableTest : public TableTest {};
class BlockBasedTableTest : public TableTest {};
class PlainTableTest : public TableTest {};
class TablePropertyTest : public testing::Test {};

// This test serves as the living tutorial for the prefix scan of user collected
// properties.
TEST_F(TablePropertyTest, PrefixScanTest) {
  UserCollectedProperties props{{"num.111.1", "1"},
                                {"num.111.2", "2"},
                                {"num.111.3", "3"},
                                {"num.333.1", "1"},
                                {"num.333.2", "2"},
                                {"num.333.3", "3"},
                                {"num.555.1", "1"},
                                {"num.555.2", "2"},
                                {"num.555.3", "3"}, };

  // prefixes that exist
  for (const std::string& prefix : {"num.111", "num.333", "num.555"}) {
    int num = 0;
    for (auto pos = props.lower_bound(prefix);
         pos != props.end() &&
             pos->first.compare(0, prefix.size(), prefix) == 0;
         ++pos) {
      ++num;
      auto key = prefix + "." + ToString(num);
      ASSERT_EQ(key, pos->first);
      ASSERT_EQ(ToString(num), pos->second);
    }
    ASSERT_EQ(3, num);
  }

  // prefixes that don't exist
  for (const std::string& prefix :
       {"num.000", "num.222", "num.444", "num.666"}) {
    auto pos = props.lower_bound(prefix);
    ASSERT_TRUE(pos == props.end() ||
                pos->first.compare(0, prefix.size(), prefix) != 0);
  }
}

// This test include all the basic checks except those for index size and block
// size, which will be conducted in separated unit tests.
TEST_F(BlockBasedTableTest, BasicBlockBasedTableProperties) {
  TableConstructor c(BytewiseComparator());

  c.Add("a1", "val1");
  c.Add("b2", "val2");
  c.Add("c3", "val3");
  c.Add("d4", "val4");
  c.Add("e5", "val5");
  c.Add("f6", "val6");
  c.Add("g7", "val7");
  c.Add("h8", "val8");
  c.Add("j9", "val9");

  std::vector<std::string> keys;
  stl_wrappers::KVMap kvmap;
  Options options;
  options.compression = kNoCompression;
  BlockBasedTableOptions table_options;
  table_options.block_restart_interval = 1;
  options.table_factory.reset(NewBlockBasedTableFactory(table_options));

  const ImmutableCFOptions ioptions(options);
  c.Finish(options, ioptions, table_options,
           GetPlainInternalComparator(options.comparator), &keys, &kvmap);

  auto& props = *c.GetTableReader()->GetTableProperties();
  ASSERT_EQ(kvmap.size(), props.num_entries);

  auto raw_key_size = kvmap.size() * 2ul;
  auto raw_value_size = kvmap.size() * 4ul;

  ASSERT_EQ(raw_key_size, props.raw_key_size);
  ASSERT_EQ(raw_value_size, props.raw_value_size);
  ASSERT_EQ(1ul, props.num_data_blocks);
  ASSERT_EQ("", props.filter_policy_name);  // no filter policy is used

  // Verify data size.
  BlockBuilder block_builder(1);
  for (const auto& item : kvmap) {
    block_builder.Add(item.first, item.second);
  }
  Slice content = block_builder.Finish();
  ASSERT_EQ(content.size() + kBlockTrailerSize, props.data_size);
}

TEST_F(BlockBasedTableTest, FilterPolicyNameProperties) {
  TableConstructor c(BytewiseComparator(), true);
  c.Add("a1", "val1");
  std::vector<std::string> keys;
  stl_wrappers::KVMap kvmap;
  BlockBasedTableOptions table_options;
  table_options.filter_policy.reset(NewBloomFilterPolicy(10));
  Options options;
  options.table_factory.reset(NewBlockBasedTableFactory(table_options));

  const ImmutableCFOptions ioptions(options);
  c.Finish(options, ioptions, table_options,
           GetPlainInternalComparator(options.comparator), &keys, &kvmap);
  auto& props = *c.GetTableReader()->GetTableProperties();
  ASSERT_EQ("rocksdb.BuiltinBloomFilter", props.filter_policy_name);
}

//
// BlockBasedTableTest::PrefetchTest
//
void AssertKeysInCache(BlockBasedTable* table_reader,
                       const std::vector<std::string>& keys_in_cache,
                       const std::vector<std::string>& keys_not_in_cache) {
  for (auto key : keys_in_cache) {
    ASSERT_TRUE(table_reader->TEST_KeyInCache(ReadOptions(), key));
  }

  for (auto key : keys_not_in_cache) {
    ASSERT_TRUE(!table_reader->TEST_KeyInCache(ReadOptions(), key));
  }
}

void PrefetchRange(TableConstructor* c, Options* opt,
                   BlockBasedTableOptions* table_options,
                   const std::vector<std::string>& keys, const char* key_begin,
                   const char* key_end,
                   const std::vector<std::string>& keys_in_cache,
                   const std::vector<std::string>& keys_not_in_cache,
                   const Status expected_status = Status::OK()) {
  // reset the cache and reopen the table
  table_options->block_cache = NewLRUCache(16 * 1024 * 1024);
  opt->table_factory.reset(NewBlockBasedTableFactory(*table_options));
  const ImmutableCFOptions ioptions2(*opt);
  ASSERT_OK(c->Reopen(ioptions2));

  // prefetch
  auto* table_reader = dynamic_cast<BlockBasedTable*>(c->GetTableReader());
  // empty string replacement is a trick so we don't crash the test
  Slice begin(key_begin ? key_begin : "");
  Slice end(key_end ? key_end : "");
  Status s = table_reader->Prefetch(key_begin ? &begin : nullptr,
                                    key_end ? &end : nullptr);
  ASSERT_TRUE(s.code() == expected_status.code());

  // assert our expectation in cache warmup
  AssertKeysInCache(table_reader, keys_in_cache, keys_not_in_cache);
}

TEST_F(BlockBasedTableTest, PrefetchTest) {
  // The purpose of this test is to test the prefetching operation built into
  // BlockBasedTable.
  Options opt;
  unique_ptr<InternalKeyComparator> ikc;
  ikc.reset(new test::PlainInternalKeyComparator(opt.comparator));
  opt.compression = kNoCompression;
  BlockBasedTableOptions table_options;
  table_options.block_size = 1024;
  // big enough so we don't ever lose cached values.
  table_options.block_cache = NewLRUCache(16 * 1024 * 1024);
  opt.table_factory.reset(NewBlockBasedTableFactory(table_options));

  TableConstructor c(BytewiseComparator());
  c.Add("k01", "hello");
  c.Add("k02", "hello2");
  c.Add("k03", std::string(10000, 'x'));
  c.Add("k04", std::string(200000, 'x'));
  c.Add("k05", std::string(300000, 'x'));
  c.Add("k06", "hello3");
  c.Add("k07", std::string(100000, 'x'));
  std::vector<std::string> keys;
  stl_wrappers::KVMap kvmap;
  const ImmutableCFOptions ioptions(opt);
  c.Finish(opt, ioptions, table_options, *ikc, &keys, &kvmap);

  // We get the following data spread :
  //
  // Data block         Index
  // ========================
  // [ k01 k02 k03 ]    k03
  // [ k04         ]    k04
  // [ k05         ]    k05
  // [ k06 k07     ]    k07


  // Simple
  PrefetchRange(&c, &opt, &table_options, keys,
                /*key_range=*/ "k01", "k05",
                /*keys_in_cache=*/ {"k01", "k02", "k03", "k04", "k05"},
                /*keys_not_in_cache=*/ {"k06", "k07"});
  PrefetchRange(&c, &opt, &table_options, keys,
                "k01", "k01",
                {"k01", "k02", "k03"},
                {"k04", "k05", "k06", "k07"});
  // odd
  PrefetchRange(&c, &opt, &table_options, keys,
                "a", "z",
                {"k01", "k02", "k03", "k04", "k05", "k06", "k07"},
                {});
  PrefetchRange(&c, &opt, &table_options, keys,
                "k00", "k00",
                {"k01", "k02", "k03"},
                {"k04", "k05", "k06", "k07"});
  // Edge cases
  PrefetchRange(&c, &opt, &table_options, keys,
                "k00", "k06",
                {"k01", "k02", "k03", "k04", "k05", "k06", "k07"},
                {});
  PrefetchRange(&c, &opt, &table_options, keys,
                "k00", "zzz",
                {"k01", "k02", "k03", "k04", "k05", "k06", "k07"},
                {});
  // null keys
  PrefetchRange(&c, &opt, &table_options, keys,
                nullptr, nullptr,
                {"k01", "k02", "k03", "k04", "k05", "k06", "k07"},
                {});
  PrefetchRange(&c, &opt, &table_options, keys,
                "k04", nullptr,
                {"k04", "k05", "k06", "k07"},
                {"k01", "k02", "k03"});
  PrefetchRange(&c, &opt, &table_options, keys,
                nullptr, "k05",
                {"k01", "k02", "k03", "k04", "k05"},
                {"k06", "k07"});
  // invalid
  PrefetchRange(&c, &opt, &table_options, keys,
                "k06", "k00", {}, {},
                Status::InvalidArgument(Slice("k06 "), Slice("k07")));
}

TEST_F(BlockBasedTableTest, TotalOrderSeekOnHashIndex) {
  BlockBasedTableOptions table_options;
  for (int i = 0; i < 4; ++i) {
    Options options;
    // Make each key/value an individual block
    table_options.block_size = 64;
    switch (i) {
    case 0:
      // Binary search index
      table_options.index_type = BlockBasedTableOptions::kBinarySearch;
      options.table_factory.reset(new BlockBasedTableFactory(table_options));
      break;
    case 1:
      // Hash search index
      table_options.index_type = BlockBasedTableOptions::kHashSearch;
      options.table_factory.reset(new BlockBasedTableFactory(table_options));
      options.prefix_extractor.reset(NewFixedPrefixTransform(4));
      break;
    case 2:
      // Hash search index with hash_index_allow_collision
      table_options.index_type = BlockBasedTableOptions::kHashSearch;
      table_options.hash_index_allow_collision = true;
      options.table_factory.reset(new BlockBasedTableFactory(table_options));
      options.prefix_extractor.reset(NewFixedPrefixTransform(4));
      break;
    case 3:
    default:
      // Hash search index with filter policy
      table_options.index_type = BlockBasedTableOptions::kHashSearch;
      table_options.filter_policy.reset(NewBloomFilterPolicy(10));
      options.table_factory.reset(new BlockBasedTableFactory(table_options));
      options.prefix_extractor.reset(NewFixedPrefixTransform(4));
      break;
    }

    TableConstructor c(BytewiseComparator(), true);
    c.Add("aaaa1", std::string('a', 56));
    c.Add("bbaa1", std::string('a', 56));
    c.Add("cccc1", std::string('a', 56));
    c.Add("bbbb1", std::string('a', 56));
    c.Add("baaa1", std::string('a', 56));
    c.Add("abbb1", std::string('a', 56));
    c.Add("cccc2", std::string('a', 56));
    std::vector<std::string> keys;
    stl_wrappers::KVMap kvmap;
    const ImmutableCFOptions ioptions(options);
    c.Finish(options, ioptions, table_options,
             GetPlainInternalComparator(options.comparator), &keys, &kvmap);
    auto props = c.GetTableReader()->GetTableProperties();
    ASSERT_EQ(7u, props->num_data_blocks);
    auto* reader = c.GetTableReader();
    ReadOptions ro;
    ro.total_order_seek = true;
    std::unique_ptr<Iterator> iter(reader->NewIterator(ro));

    iter->Seek(InternalKey("b", 0, kTypeValue).Encode());
    ASSERT_OK(iter->status());
    ASSERT_TRUE(iter->Valid());
    ASSERT_EQ("baaa1", ExtractUserKey(iter->key()).ToString());
    iter->Next();
    ASSERT_OK(iter->status());
    ASSERT_TRUE(iter->Valid());
    ASSERT_EQ("bbaa1", ExtractUserKey(iter->key()).ToString());

    iter->Seek(InternalKey("bb", 0, kTypeValue).Encode());
    ASSERT_OK(iter->status());
    ASSERT_TRUE(iter->Valid());
    ASSERT_EQ("bbaa1", ExtractUserKey(iter->key()).ToString());
    iter->Next();
    ASSERT_OK(iter->status());
    ASSERT_TRUE(iter->Valid());
    ASSERT_EQ("bbbb1", ExtractUserKey(iter->key()).ToString());

    iter->Seek(InternalKey("bbb", 0, kTypeValue).Encode());
    ASSERT_OK(iter->status());
    ASSERT_TRUE(iter->Valid());
    ASSERT_EQ("bbbb1", ExtractUserKey(iter->key()).ToString());
    iter->Next();
    ASSERT_OK(iter->status());
    ASSERT_TRUE(iter->Valid());
    ASSERT_EQ("cccc1", ExtractUserKey(iter->key()).ToString());
  }
}

static std::string RandomString(Random* rnd, int len) {
  std::string r;
  test::RandomString(rnd, len, &r);
  return r;
}

void AddInternalKey(TableConstructor* c, const std::string& prefix,
                    int suffix_len = 800) {
  static Random rnd(1023);
  InternalKey k(prefix + RandomString(&rnd, 800), 0, kTypeValue);
  c->Add(k.Encode().ToString(), "v");
}

TEST_F(TableTest, HashIndexTest) {
  TableConstructor c(BytewiseComparator());

  // keys with prefix length 3, make sure the key/value is big enough to fill
  // one block
  AddInternalKey(&c, "0015");
  AddInternalKey(&c, "0035");

  AddInternalKey(&c, "0054");
  AddInternalKey(&c, "0055");

  AddInternalKey(&c, "0056");
  AddInternalKey(&c, "0057");

  AddInternalKey(&c, "0058");
  AddInternalKey(&c, "0075");

  AddInternalKey(&c, "0076");
  AddInternalKey(&c, "0095");

  std::vector<std::string> keys;
  stl_wrappers::KVMap kvmap;
  Options options;
  options.prefix_extractor.reset(NewFixedPrefixTransform(3));
  BlockBasedTableOptions table_options;
  table_options.index_type = BlockBasedTableOptions::kHashSearch;
  table_options.hash_index_allow_collision = true;
  table_options.block_size = 1700;
  table_options.block_cache = NewLRUCache(1024);
  options.table_factory.reset(NewBlockBasedTableFactory(table_options));

  std::unique_ptr<InternalKeyComparator> comparator(
      new InternalKeyComparator(BytewiseComparator()));
  const ImmutableCFOptions ioptions(options);
  c.Finish(options, ioptions, table_options, *comparator, &keys, &kvmap);
  auto reader = c.GetTableReader();

  auto props = reader->GetTableProperties();
  ASSERT_EQ(5u, props->num_data_blocks);

  std::unique_ptr<Iterator> hash_iter(reader->NewIterator(ReadOptions()));

  // -- Find keys do not exist, but have common prefix.
  std::vector<std::string> prefixes = {"001", "003", "005", "007", "009"};
  std::vector<std::string> lower_bound = {keys[0], keys[1], keys[2],
                                          keys[7], keys[9], };

  // find the lower bound of the prefix
  for (size_t i = 0; i < prefixes.size(); ++i) {
    hash_iter->Seek(InternalKey(prefixes[i], 0, kTypeValue).Encode());
    ASSERT_OK(hash_iter->status());
    ASSERT_TRUE(hash_iter->Valid());

    // seek the first element in the block
    ASSERT_EQ(lower_bound[i], hash_iter->key().ToString());
    ASSERT_EQ("v", hash_iter->value().ToString());
  }

  // find the upper bound of prefixes
  std::vector<std::string> upper_bound = {keys[1], keys[2], keys[7], keys[9], };

  // find existing keys
  for (const auto& item : kvmap) {
    auto ukey = ExtractUserKey(item.first).ToString();
    hash_iter->Seek(ukey);

    // ASSERT_OK(regular_iter->status());
    ASSERT_OK(hash_iter->status());

    // ASSERT_TRUE(regular_iter->Valid());
    ASSERT_TRUE(hash_iter->Valid());

    ASSERT_EQ(item.first, hash_iter->key().ToString());
    ASSERT_EQ(item.second, hash_iter->value().ToString());
  }

  for (size_t i = 0; i < prefixes.size(); ++i) {
    // the key is greater than any existing keys.
    auto key = prefixes[i] + "9";
    hash_iter->Seek(InternalKey(key, 0, kTypeValue).Encode());

    ASSERT_OK(hash_iter->status());
    if (i == prefixes.size() - 1) {
      // last key
      ASSERT_TRUE(!hash_iter->Valid());
    } else {
      ASSERT_TRUE(hash_iter->Valid());
      // seek the first element in the block
      ASSERT_EQ(upper_bound[i], hash_iter->key().ToString());
      ASSERT_EQ("v", hash_iter->value().ToString());
    }
  }

  // find keys with prefix that don't match any of the existing prefixes.
  std::vector<std::string> non_exist_prefixes = {"002", "004", "006", "008"};
  for (const auto& prefix : non_exist_prefixes) {
    hash_iter->Seek(InternalKey(prefix, 0, kTypeValue).Encode());
    // regular_iter->Seek(prefix);

    ASSERT_OK(hash_iter->status());
    // Seek to non-existing prefixes should yield either invalid, or a
    // key with prefix greater than the target.
    if (hash_iter->Valid()) {
      Slice ukey = ExtractUserKey(hash_iter->key());
      Slice ukey_prefix = options.prefix_extractor->Transform(ukey);
      ASSERT_TRUE(BytewiseComparator()->Compare(prefix, ukey_prefix) < 0);
    }
  }
}

// It's very hard to figure out the index block size of a block accurately.
// To make sure we get the index size, we just make sure as key number
// grows, the filter block size also grows.
TEST_F(BlockBasedTableTest, IndexSizeStat) {
  uint64_t last_index_size = 0;

  // we need to use random keys since the pure human readable texts
  // may be well compressed, resulting insignifcant change of index
  // block size.
  Random rnd(test::RandomSeed());
  std::vector<std::string> keys;

  for (int i = 0; i < 100; ++i) {
    keys.push_back(RandomString(&rnd, 10000));
  }

  // Each time we load one more key to the table. the table index block
  // size is expected to be larger than last time's.
  for (size_t i = 1; i < keys.size(); ++i) {
    TableConstructor c(BytewiseComparator());
    for (size_t j = 0; j < i; ++j) {
      c.Add(keys[j], "val");
    }

    std::vector<std::string> ks;
    stl_wrappers::KVMap kvmap;
    Options options;
    options.compression = kNoCompression;
    BlockBasedTableOptions table_options;
    table_options.block_restart_interval = 1;
    options.table_factory.reset(NewBlockBasedTableFactory(table_options));

    const ImmutableCFOptions ioptions(options);
    c.Finish(options, ioptions, table_options,
             GetPlainInternalComparator(options.comparator), &ks, &kvmap);
    auto index_size = c.GetTableReader()->GetTableProperties()->index_size;
    ASSERT_GT(index_size, last_index_size);
    last_index_size = index_size;
  }
}

TEST_F(BlockBasedTableTest, NumBlockStat) {
  Random rnd(test::RandomSeed());
  TableConstructor c(BytewiseComparator());
  Options options;
  options.compression = kNoCompression;
  BlockBasedTableOptions table_options;
  table_options.block_restart_interval = 1;
  table_options.block_size = 1000;
  options.table_factory.reset(NewBlockBasedTableFactory(table_options));

  for (int i = 0; i < 10; ++i) {
    // the key/val are slightly smaller than block size, so that each block
    // holds roughly one key/value pair.
    c.Add(RandomString(&rnd, 900), "val");
  }

  std::vector<std::string> ks;
  stl_wrappers::KVMap kvmap;
  const ImmutableCFOptions ioptions(options);
  c.Finish(options, ioptions, table_options,
           GetPlainInternalComparator(options.comparator), &ks, &kvmap);
  ASSERT_EQ(kvmap.size(),
            c.GetTableReader()->GetTableProperties()->num_data_blocks);
}

// A simple tool that takes the snapshot of block cache statistics.
class BlockCachePropertiesSnapshot {
 public:
  explicit BlockCachePropertiesSnapshot(Statistics* statistics) {
    block_cache_miss = statistics->getTickerCount(BLOCK_CACHE_MISS);
    block_cache_hit = statistics->getTickerCount(BLOCK_CACHE_HIT);
    index_block_cache_miss = statistics->getTickerCount(BLOCK_CACHE_INDEX_MISS);
    index_block_cache_hit = statistics->getTickerCount(BLOCK_CACHE_INDEX_HIT);
    data_block_cache_miss = statistics->getTickerCount(BLOCK_CACHE_DATA_MISS);
    data_block_cache_hit = statistics->getTickerCount(BLOCK_CACHE_DATA_HIT);
    filter_block_cache_miss =
        statistics->getTickerCount(BLOCK_CACHE_FILTER_MISS);
    filter_block_cache_hit = statistics->getTickerCount(BLOCK_CACHE_FILTER_HIT);
  }

  void AssertIndexBlockStat(int64_t expected_index_block_cache_miss,
                            int64_t expected_index_block_cache_hit) {
    ASSERT_EQ(expected_index_block_cache_miss, index_block_cache_miss);
    ASSERT_EQ(expected_index_block_cache_hit, index_block_cache_hit);
  }

  void AssertFilterBlockStat(int64_t expected_filter_block_cache_miss,
                             int64_t expected_filter_block_cache_hit) {
    ASSERT_EQ(expected_filter_block_cache_miss, filter_block_cache_miss);
    ASSERT_EQ(expected_filter_block_cache_hit, filter_block_cache_hit);
  }

  // Check if the fetched props matches the expected ones.
  // TODO(kailiu) Use this only when you disabled filter policy!
  void AssertEqual(int64_t expected_index_block_cache_miss,
                   int64_t expected_index_block_cache_hit,
                   int64_t expected_data_block_cache_miss,
                   int64_t expected_data_block_cache_hit) const {
    ASSERT_EQ(expected_index_block_cache_miss, index_block_cache_miss);
    ASSERT_EQ(expected_index_block_cache_hit, index_block_cache_hit);
    ASSERT_EQ(expected_data_block_cache_miss, data_block_cache_miss);
    ASSERT_EQ(expected_data_block_cache_hit, data_block_cache_hit);
    ASSERT_EQ(expected_index_block_cache_miss + expected_data_block_cache_miss,
              block_cache_miss);
    ASSERT_EQ(expected_index_block_cache_hit + expected_data_block_cache_hit,
              block_cache_hit);
  }

 private:
  int64_t block_cache_miss = 0;
  int64_t block_cache_hit = 0;
  int64_t index_block_cache_miss = 0;
  int64_t index_block_cache_hit = 0;
  int64_t data_block_cache_miss = 0;
  int64_t data_block_cache_hit = 0;
  int64_t filter_block_cache_miss = 0;
  int64_t filter_block_cache_hit = 0;
};

// Make sure, by default, index/filter blocks were pre-loaded (meaning we won't
// use block cache to store them).
TEST_F(BlockBasedTableTest, BlockCacheDisabledTest) {
  Options options;
  options.create_if_missing = true;
  options.statistics = CreateDBStatistics();
  BlockBasedTableOptions table_options;
  table_options.block_cache = NewLRUCache(1024);
  table_options.filter_policy.reset(NewBloomFilterPolicy(10));
  options.table_factory.reset(new BlockBasedTableFactory(table_options));
  std::vector<std::string> keys;
  stl_wrappers::KVMap kvmap;

  TableConstructor c(BytewiseComparator(), true);
  c.Add("key", "value");
  const ImmutableCFOptions ioptions(options);
  c.Finish(options, ioptions, table_options,
           GetPlainInternalComparator(options.comparator), &keys, &kvmap);

  // preloading filter/index blocks is enabled.
  auto reader = dynamic_cast<BlockBasedTable*>(c.GetTableReader());
  ASSERT_TRUE(reader->TEST_filter_block_preloaded());
  ASSERT_TRUE(reader->TEST_index_reader_preloaded());

  {
    // nothing happens in the beginning
    BlockCachePropertiesSnapshot props(options.statistics.get());
    props.AssertIndexBlockStat(0, 0);
    props.AssertFilterBlockStat(0, 0);
  }

  {
    GetContext get_context(options.comparator, nullptr, nullptr, nullptr,
                           GetContext::kNotFound, Slice(), nullptr, nullptr,
                           nullptr, nullptr);
    // a hack that just to trigger BlockBasedTable::GetFilter.
    reader->Get(ReadOptions(), "non-exist-key", &get_context);
    BlockCachePropertiesSnapshot props(options.statistics.get());
    props.AssertIndexBlockStat(0, 0);
    props.AssertFilterBlockStat(0, 0);
  }
}

// Due to the difficulities of the intersaction between statistics, this test
// only tests the case when "index block is put to block cache"
TEST_F(BlockBasedTableTest, FilterBlockInBlockCache) {
  // -- Table construction
  Options options;
  options.create_if_missing = true;
  options.statistics = CreateDBStatistics();

  // Enable the cache for index/filter blocks
  BlockBasedTableOptions table_options;
  table_options.block_cache = NewLRUCache(1024);
  table_options.cache_index_and_filter_blocks = true;
  options.table_factory.reset(new BlockBasedTableFactory(table_options));
  std::vector<std::string> keys;
  stl_wrappers::KVMap kvmap;

  TableConstructor c(BytewiseComparator());
  c.Add("key", "value");
  const ImmutableCFOptions ioptions(options);
  c.Finish(options, ioptions, table_options,
           GetPlainInternalComparator(options.comparator), &keys, &kvmap);
  // preloading filter/index blocks is prohibited.
  auto* reader = dynamic_cast<BlockBasedTable*>(c.GetTableReader());
  ASSERT_TRUE(!reader->TEST_filter_block_preloaded());
  ASSERT_TRUE(!reader->TEST_index_reader_preloaded());

  // -- PART 1: Open with regular block cache.
  // Since block_cache is disabled, no cache activities will be involved.
  unique_ptr<Iterator> iter;

  // At first, no block will be accessed.
  {
    BlockCachePropertiesSnapshot props(options.statistics.get());
    // index will be added to block cache.
    props.AssertEqual(1,  // index block miss
                      0, 0, 0);
  }

  // Only index block will be accessed
  {
    iter.reset(c.NewIterator());
    BlockCachePropertiesSnapshot props(options.statistics.get());
    // NOTE: to help better highlight the "detla" of each ticker, I use
    // <last_value> + <added_value> to indicate the increment of changed
    // value; other numbers remain the same.
    props.AssertEqual(1, 0 + 1,  // index block hit
                      0, 0);
  }

  // Only data block will be accessed
  {
    iter->SeekToFirst();
    BlockCachePropertiesSnapshot props(options.statistics.get());
    props.AssertEqual(1, 1, 0 + 1,  // data block miss
                      0);
  }

  // Data block will be in cache
  {
    iter.reset(c.NewIterator());
    iter->SeekToFirst();
    BlockCachePropertiesSnapshot props(options.statistics.get());
    props.AssertEqual(1, 1 + 1, /* index block hit */
                      1, 0 + 1 /* data block hit */);
  }
  // release the iterator so that the block cache can reset correctly.
  iter.reset();

  // -- PART 2: Open with very small block cache
  // In this test, no block will ever get hit since the block cache is
  // too small to fit even one entry.
  table_options.block_cache = NewLRUCache(1);
  options.statistics = CreateDBStatistics();
  options.table_factory.reset(new BlockBasedTableFactory(table_options));
  const ImmutableCFOptions ioptions2(options);
  c.Reopen(ioptions2);
  {
    BlockCachePropertiesSnapshot props(options.statistics.get());
    props.AssertEqual(1,  // index block miss
                      0, 0, 0);
  }

  {
    // Both index and data block get accessed.
    // It first cache index block then data block. But since the cache size
    // is only 1, index block will be purged after data block is inserted.
    iter.reset(c.NewIterator());
    BlockCachePropertiesSnapshot props(options.statistics.get());
    props.AssertEqual(1 + 1,  // index block miss
                      0, 0,   // data block miss
                      0);
  }

  {
    // SeekToFirst() accesses data block. With similar reason, we expect data
    // block's cache miss.
    iter->SeekToFirst();
    BlockCachePropertiesSnapshot props(options.statistics.get());
    props.AssertEqual(2, 0, 0 + 1,  // data block miss
                      0);
  }
  iter.reset();

  // -- PART 3: Open table with bloom filter enabled but not in SST file
  table_options.block_cache = NewLRUCache(4096);
  table_options.cache_index_and_filter_blocks = false;
  options.table_factory.reset(NewBlockBasedTableFactory(table_options));

  TableConstructor c3(BytewiseComparator());
  std::string user_key = "k01";
  InternalKey internal_key(user_key, 0, kTypeValue);
  c3.Add(internal_key.Encode().ToString(), "hello");
  ImmutableCFOptions ioptions3(options);
  // Generate table without filter policy
  c3.Finish(options, ioptions3, table_options,
           GetPlainInternalComparator(options.comparator), &keys, &kvmap);
  // Open table with filter policy
  table_options.filter_policy.reset(NewBloomFilterPolicy(1));
  options.table_factory.reset(new BlockBasedTableFactory(table_options));
  options.statistics = CreateDBStatistics();
  ImmutableCFOptions ioptions4(options);
  ASSERT_OK(c3.Reopen(ioptions4));
  reader = dynamic_cast<BlockBasedTable*>(c3.GetTableReader());
  ASSERT_TRUE(!reader->TEST_filter_block_preloaded());
  std::string value;
  GetContext get_context(options.comparator, nullptr, nullptr, nullptr,
                         GetContext::kNotFound, user_key, &value, nullptr,
                         nullptr, nullptr);
  ASSERT_OK(reader->Get(ReadOptions(), user_key, &get_context));
  ASSERT_EQ(value, "hello");
  BlockCachePropertiesSnapshot props(options.statistics.get());
  props.AssertFilterBlockStat(0, 0);
}

TEST_F(BlockBasedTableTest, BlockReadCountTest) {
  // bloom_filter_type = 0 -- block-based filter
  // bloom_filter_type = 0 -- full filter
  for (int bloom_filter_type = 0; bloom_filter_type < 2; ++bloom_filter_type) {
    for (int index_and_filter_in_cache = 0; index_and_filter_in_cache < 2;
         ++index_and_filter_in_cache) {
      Options options;
      options.create_if_missing = true;

      BlockBasedTableOptions table_options;
      table_options.block_cache = NewLRUCache(1, 0);
      table_options.cache_index_and_filter_blocks = index_and_filter_in_cache;
      table_options.filter_policy.reset(
          NewBloomFilterPolicy(10, bloom_filter_type == 0));
      options.table_factory.reset(new BlockBasedTableFactory(table_options));
      std::vector<std::string> keys;
      stl_wrappers::KVMap kvmap;

      TableConstructor c(BytewiseComparator());
      std::string user_key = "k04";
      InternalKey internal_key(user_key, 0, kTypeValue);
      std::string encoded_key = internal_key.Encode().ToString();
      c.Add(encoded_key, "hello");
      ImmutableCFOptions ioptions(options);
      // Generate table with filter policy
      c.Finish(options, ioptions, table_options,
               GetPlainInternalComparator(options.comparator), &keys, &kvmap);
      auto reader = c.GetTableReader();
      std::string value;
      GetContext get_context(options.comparator, nullptr, nullptr, nullptr,
                             GetContext::kNotFound, user_key, &value, nullptr,
                             nullptr, nullptr);
      perf_context.Reset();
      ASSERT_OK(reader->Get(ReadOptions(), encoded_key, &get_context));
      if (index_and_filter_in_cache) {
        // data, index and filter block
        ASSERT_EQ(perf_context.block_read_count, 3);
      } else {
        // just the data block
        ASSERT_EQ(perf_context.block_read_count, 1);
      }
      ASSERT_EQ(get_context.State(), GetContext::kFound);
      ASSERT_EQ(value, "hello");

      // Get non-existing key
      user_key = "does-not-exist";
      internal_key = InternalKey(user_key, 0, kTypeValue);
      encoded_key = internal_key.Encode().ToString();

      get_context = GetContext(options.comparator, nullptr, nullptr, nullptr,
                               GetContext::kNotFound, user_key, &value, nullptr,
                               nullptr, nullptr);
      perf_context.Reset();
      ASSERT_OK(reader->Get(ReadOptions(), encoded_key, &get_context));
      ASSERT_EQ(get_context.State(), GetContext::kNotFound);

      if (index_and_filter_in_cache) {
        if (bloom_filter_type == 0) {
          // with block-based, we read index and then the filter
          ASSERT_EQ(perf_context.block_read_count, 2);
        } else {
          // with full-filter, we read filter first and then we stop
          ASSERT_EQ(perf_context.block_read_count, 1);
        }
      } else {
        // filter is already in memory and it figures out that the key doesn't
        // exist
        ASSERT_EQ(perf_context.block_read_count, 0);
      }
    }
  }
}

TEST_F(BlockBasedTableTest, BlockCacheLeak) {
  // Check that when we reopen a table we don't lose access to blocks already
  // in the cache. This test checks whether the Table actually makes use of the
  // unique ID from the file.

  Options opt;
  unique_ptr<InternalKeyComparator> ikc;
  ikc.reset(new test::PlainInternalKeyComparator(opt.comparator));
  opt.compression = kNoCompression;
  BlockBasedTableOptions table_options;
  table_options.block_size = 1024;
  // big enough so we don't ever lose cached values.
  table_options.block_cache = NewLRUCache(16 * 1024 * 1024);
  opt.table_factory.reset(NewBlockBasedTableFactory(table_options));

  TableConstructor c(BytewiseComparator());
  c.Add("k01", "hello");
  c.Add("k02", "hello2");
  c.Add("k03", std::string(10000, 'x'));
  c.Add("k04", std::string(200000, 'x'));
  c.Add("k05", std::string(300000, 'x'));
  c.Add("k06", "hello3");
  c.Add("k07", std::string(100000, 'x'));
  std::vector<std::string> keys;
  stl_wrappers::KVMap kvmap;
  const ImmutableCFOptions ioptions(opt);
  c.Finish(opt, ioptions, table_options, *ikc, &keys, &kvmap);

  unique_ptr<Iterator> iter(c.NewIterator());
  iter->SeekToFirst();
  while (iter->Valid()) {
    iter->key();
    iter->value();
    iter->Next();
  }
  ASSERT_OK(iter->status());

  const ImmutableCFOptions ioptions1(opt);
  ASSERT_OK(c.Reopen(ioptions1));
  auto table_reader = dynamic_cast<BlockBasedTable*>(c.GetTableReader());
  for (const std::string& key : keys) {
    ASSERT_TRUE(table_reader->TEST_KeyInCache(ReadOptions(), key));
  }

  // rerun with different block cache
  table_options.block_cache = NewLRUCache(16 * 1024 * 1024);
  opt.table_factory.reset(NewBlockBasedTableFactory(table_options));
  const ImmutableCFOptions ioptions2(opt);
  ASSERT_OK(c.Reopen(ioptions2));
  table_reader = dynamic_cast<BlockBasedTable*>(c.GetTableReader());
  for (const std::string& key : keys) {
    ASSERT_TRUE(!table_reader->TEST_KeyInCache(ReadOptions(), key));
  }
}

// Plain table is not supported in ROCKSDB_LITE
#ifndef ROCKSDB_LITE
TEST_F(PlainTableTest, BasicPlainTableProperties) {
  PlainTableOptions plain_table_options;
  plain_table_options.user_key_len = 8;
  plain_table_options.bloom_bits_per_key = 8;
  plain_table_options.hash_table_ratio = 0;

  PlainTableFactory factory(plain_table_options);
  test::StringSink sink;
  unique_ptr<WritableFileWriter> file_writer(
      test::GetWritableFileWriter(new test::StringSink()));
  Options options;
  const ImmutableCFOptions ioptions(options);
  InternalKeyComparator ikc(options.comparator);
  std::vector<std::unique_ptr<IntTblPropCollectorFactory>>
      int_tbl_prop_collector_factories;
  std::unique_ptr<TableBuilder> builder(factory.NewTableBuilder(
      TableBuilderOptions(ioptions, ikc, &int_tbl_prop_collector_factories,
                          kNoCompression, CompressionOptions(), false),
      file_writer.get()));

  for (char c = 'a'; c <= 'z'; ++c) {
    std::string key(8, c);
    key.append("\1       ");  // PlainTable expects internal key structure
    std::string value(28, c + 42);
    builder->Add(key, value);
  }
  ASSERT_OK(builder->Finish());
  file_writer->Flush();

  test::StringSink* ss =
    static_cast<test::StringSink*>(file_writer->writable_file());
  unique_ptr<RandomAccessFileReader> file_reader(
      test::GetRandomAccessFileReader(
          new test::StringSource(ss->contents(), 72242, true)));

  TableProperties* props = nullptr;
  auto s = ReadTableProperties(file_reader.get(), ss->contents().size(),
                               kPlainTableMagicNumber, Env::Default(), nullptr,
                               &props);
  std::unique_ptr<TableProperties> props_guard(props);
  ASSERT_OK(s);

  ASSERT_EQ(0ul, props->index_size);
  ASSERT_EQ(0ul, props->filter_size);
  ASSERT_EQ(16ul * 26, props->raw_key_size);
  ASSERT_EQ(28ul * 26, props->raw_value_size);
  ASSERT_EQ(26ul, props->num_entries);
  ASSERT_EQ(1ul, props->num_data_blocks);
}
#endif  // !ROCKSDB_LITE

TEST_F(GeneralTableTest, ApproximateOffsetOfPlain) {
  TableConstructor c(BytewiseComparator());
  c.Add("k01", "hello");
  c.Add("k02", "hello2");
  c.Add("k03", std::string(10000, 'x'));
  c.Add("k04", std::string(200000, 'x'));
  c.Add("k05", std::string(300000, 'x'));
  c.Add("k06", "hello3");
  c.Add("k07", std::string(100000, 'x'));
  std::vector<std::string> keys;
  stl_wrappers::KVMap kvmap;
  Options options;
  test::PlainInternalKeyComparator internal_comparator(options.comparator);
  options.compression = kNoCompression;
  BlockBasedTableOptions table_options;
  table_options.block_size = 1024;
  const ImmutableCFOptions ioptions(options);
  c.Finish(options, ioptions, table_options, internal_comparator,
           &keys, &kvmap);

  ASSERT_TRUE(Between(c.ApproximateOffsetOf("abc"),       0,      0));
  ASSERT_TRUE(Between(c.ApproximateOffsetOf("k01"),       0,      0));
  ASSERT_TRUE(Between(c.ApproximateOffsetOf("k01a"),      0,      0));
  ASSERT_TRUE(Between(c.ApproximateOffsetOf("k02"),       0,      0));
  ASSERT_TRUE(Between(c.ApproximateOffsetOf("k03"),       0,      0));
  ASSERT_TRUE(Between(c.ApproximateOffsetOf("k04"),   10000,  11000));
  ASSERT_TRUE(Between(c.ApproximateOffsetOf("k04a"), 210000, 211000));
  ASSERT_TRUE(Between(c.ApproximateOffsetOf("k05"),  210000, 211000));
  ASSERT_TRUE(Between(c.ApproximateOffsetOf("k06"),  510000, 511000));
  ASSERT_TRUE(Between(c.ApproximateOffsetOf("k07"),  510000, 511000));
  ASSERT_TRUE(Between(c.ApproximateOffsetOf("xyz"),  610000, 612000));
}

static void DoCompressionTest(CompressionType comp) {
  Random rnd(301);
  TableConstructor c(BytewiseComparator());
  std::string tmp;
  c.Add("k01", "hello");
  c.Add("k02", test::CompressibleString(&rnd, 0.25, 10000, &tmp));
  c.Add("k03", "hello3");
  c.Add("k04", test::CompressibleString(&rnd, 0.25, 10000, &tmp));
  std::vector<std::string> keys;
  stl_wrappers::KVMap kvmap;
  Options options;
  test::PlainInternalKeyComparator ikc(options.comparator);
  options.compression = comp;
  BlockBasedTableOptions table_options;
  table_options.block_size = 1024;
  const ImmutableCFOptions ioptions(options);
  c.Finish(options, ioptions, table_options, ikc, &keys, &kvmap);

  ASSERT_TRUE(Between(c.ApproximateOffsetOf("abc"),       0,      0));
  ASSERT_TRUE(Between(c.ApproximateOffsetOf("k01"),       0,      0));
  ASSERT_TRUE(Between(c.ApproximateOffsetOf("k02"),       0,      0));
  ASSERT_TRUE(Between(c.ApproximateOffsetOf("k03"),    2000,   3000));
  ASSERT_TRUE(Between(c.ApproximateOffsetOf("k04"),    2000,   3000));
  ASSERT_TRUE(Between(c.ApproximateOffsetOf("xyz"),    4000,   6100));
}

TEST_F(GeneralTableTest, ApproximateOffsetOfCompressed) {
  std::vector<CompressionType> compression_state;
  if (!Snappy_Supported()) {
    fprintf(stderr, "skipping snappy compression tests\n");
  } else {
    compression_state.push_back(kSnappyCompression);
  }

  if (!Zlib_Supported()) {
    fprintf(stderr, "skipping zlib compression tests\n");
  } else {
    compression_state.push_back(kZlibCompression);
  }

  // TODO(kailiu) DoCompressionTest() doesn't work with BZip2.
  /*
  if (!BZip2_Supported()) {
    fprintf(stderr, "skipping bzip2 compression tests\n");
  } else {
    compression_state.push_back(kBZip2Compression);
  }
  */

  if (!LZ4_Supported()) {
    fprintf(stderr, "skipping lz4 and lz4hc compression tests\n");
  } else {
    compression_state.push_back(kLZ4Compression);
    compression_state.push_back(kLZ4HCCompression);
  }

  for (auto state : compression_state) {
    DoCompressionTest(state);
  }
}

TEST_F(HarnessTest, Randomized) {
  std::vector<TestArgs> args = GenerateArgList();
  for (unsigned int i = 0; i < args.size(); i++) {
    Init(args[i]);
    Random rnd(test::RandomSeed() + 5);
    for (int num_entries = 0; num_entries < 2000;
         num_entries += (num_entries < 50 ? 1 : 200)) {
      if ((num_entries % 10) == 0) {
        fprintf(stderr, "case %d of %d: num_entries = %d\n", (i + 1),
                static_cast<int>(args.size()), num_entries);
      }
      for (int e = 0; e < num_entries; e++) {
        std::string v;
        Add(test::RandomKey(&rnd, rnd.Skewed(4)),
            test::RandomString(&rnd, rnd.Skewed(5), &v).ToString());
      }
      Test(&rnd);
    }
  }
}

TEST_F(HarnessTest, RandomizedLongDB) {
  Random rnd(test::RandomSeed());
  TestArgs args = { DB_TEST, false, 16, kNoCompression, 0 };
  Init(args);
  int num_entries = 100000;
  for (int e = 0; e < num_entries; e++) {
    std::string v;
    Add(test::RandomKey(&rnd, rnd.Skewed(4)),
        test::RandomString(&rnd, rnd.Skewed(5), &v).ToString());
  }
  Test(&rnd);

  // We must have created enough data to force merging
  int files = 0;
  for (int level = 0; level < db()->NumberLevels(); level++) {
    std::string value;
    char name[100];
    snprintf(name, sizeof(name), "rocksdb.num-files-at-level%d", level);
    ASSERT_TRUE(db()->GetProperty(name, &value));
    files += atoi(value.c_str());
  }
  ASSERT_GT(files, 0);
}

class MemTableTest : public testing::Test {};

TEST_F(MemTableTest, Simple) {
  InternalKeyComparator cmp(BytewiseComparator());
  auto table_factory = std::make_shared<SkipListFactory>();
  Options options;
  options.memtable_factory = table_factory;
  ImmutableCFOptions ioptions(options);
  WriteBuffer wb(options.db_write_buffer_size);
  MemTable* memtable =
      new MemTable(cmp, ioptions, MutableCFOptions(options, ioptions), &wb,
                   kMaxSequenceNumber);
  memtable->Ref();
  WriteBatch batch;
  WriteBatchInternal::SetSequence(&batch, 100);
  batch.Put(std::string("k1"), std::string("v1"));
  batch.Put(std::string("k2"), std::string("v2"));
  batch.Put(std::string("k3"), std::string("v3"));
  batch.Put(std::string("largekey"), std::string("vlarge"));
  ColumnFamilyMemTablesDefault cf_mems_default(memtable);
  ASSERT_TRUE(WriteBatchInternal::InsertInto(&batch, &cf_mems_default).ok());

  Arena arena;
  ScopedArenaIterator iter(memtable->NewIterator(ReadOptions(), &arena));
  iter->SeekToFirst();
  while (iter->Valid()) {
    fprintf(stderr, "key: '%s' -> '%s'\n",
            iter->key().ToString().c_str(),
            iter->value().ToString().c_str());
    iter->Next();
  }

  delete memtable->Unref();
}

// Test the empty key
TEST_F(HarnessTest, SimpleEmptyKey) {
  auto args = GenerateArgList();
  for (const auto& arg : args) {
    Init(arg);
    Random rnd(test::RandomSeed() + 1);
    Add("", "v");
    Test(&rnd);
  }
}

TEST_F(HarnessTest, SimpleSingle) {
  auto args = GenerateArgList();
  for (const auto& arg : args) {
    Init(arg);
    Random rnd(test::RandomSeed() + 2);
    Add("abc", "v");
    Test(&rnd);
  }
}

TEST_F(HarnessTest, SimpleMulti) {
  auto args = GenerateArgList();
  for (const auto& arg : args) {
    Init(arg);
    Random rnd(test::RandomSeed() + 3);
    Add("abc", "v");
    Add("abcd", "v");
    Add("ac", "v2");
    Test(&rnd);
  }
}

TEST_F(HarnessTest, SimpleSpecialKey) {
  auto args = GenerateArgList();
  for (const auto& arg : args) {
    Init(arg);
    Random rnd(test::RandomSeed() + 4);
    Add("\xff\xff", "v3");
    Test(&rnd);
  }
}

TEST_F(HarnessTest, FooterTests) {
  {
    // upconvert legacy block based
    std::string encoded;
    Footer footer(kLegacyBlockBasedTableMagicNumber, 0);
    BlockHandle meta_index(10, 5), index(20, 15);
    footer.set_metaindex_handle(meta_index);
    footer.set_index_handle(index);
    footer.EncodeTo(&encoded);
    Footer decoded_footer;
    Slice encoded_slice(encoded);
    decoded_footer.DecodeFrom(&encoded_slice);
    ASSERT_EQ(decoded_footer.table_magic_number(), kBlockBasedTableMagicNumber);
    ASSERT_EQ(decoded_footer.checksum(), kCRC32c);
    ASSERT_EQ(decoded_footer.metaindex_handle().offset(), meta_index.offset());
    ASSERT_EQ(decoded_footer.metaindex_handle().size(), meta_index.size());
    ASSERT_EQ(decoded_footer.index_handle().offset(), index.offset());
    ASSERT_EQ(decoded_footer.index_handle().size(), index.size());
    ASSERT_EQ(decoded_footer.version(), 0U);
  }
  {
    // xxhash block based
    std::string encoded;
    Footer footer(kBlockBasedTableMagicNumber, 1);
    BlockHandle meta_index(10, 5), index(20, 15);
    footer.set_metaindex_handle(meta_index);
    footer.set_index_handle(index);
    footer.set_checksum(kxxHash);
    footer.EncodeTo(&encoded);
    Footer decoded_footer;
    Slice encoded_slice(encoded);
    decoded_footer.DecodeFrom(&encoded_slice);
    ASSERT_EQ(decoded_footer.table_magic_number(), kBlockBasedTableMagicNumber);
    ASSERT_EQ(decoded_footer.checksum(), kxxHash);
    ASSERT_EQ(decoded_footer.metaindex_handle().offset(), meta_index.offset());
    ASSERT_EQ(decoded_footer.metaindex_handle().size(), meta_index.size());
    ASSERT_EQ(decoded_footer.index_handle().offset(), index.offset());
    ASSERT_EQ(decoded_footer.index_handle().size(), index.size());
    ASSERT_EQ(decoded_footer.version(), 1U);
  }
// Plain table is not supported in ROCKSDB_LITE
#ifndef ROCKSDB_LITE
  {
    // upconvert legacy plain table
    std::string encoded;
    Footer footer(kLegacyPlainTableMagicNumber, 0);
    BlockHandle meta_index(10, 5), index(20, 15);
    footer.set_metaindex_handle(meta_index);
    footer.set_index_handle(index);
    footer.EncodeTo(&encoded);
    Footer decoded_footer;
    Slice encoded_slice(encoded);
    decoded_footer.DecodeFrom(&encoded_slice);
    ASSERT_EQ(decoded_footer.table_magic_number(), kPlainTableMagicNumber);
    ASSERT_EQ(decoded_footer.checksum(), kCRC32c);
    ASSERT_EQ(decoded_footer.metaindex_handle().offset(), meta_index.offset());
    ASSERT_EQ(decoded_footer.metaindex_handle().size(), meta_index.size());
    ASSERT_EQ(decoded_footer.index_handle().offset(), index.offset());
    ASSERT_EQ(decoded_footer.index_handle().size(), index.size());
    ASSERT_EQ(decoded_footer.version(), 0U);
  }
  {
    // xxhash block based
    std::string encoded;
    Footer footer(kPlainTableMagicNumber, 1);
    BlockHandle meta_index(10, 5), index(20, 15);
    footer.set_metaindex_handle(meta_index);
    footer.set_index_handle(index);
    footer.set_checksum(kxxHash);
    footer.EncodeTo(&encoded);
    Footer decoded_footer;
    Slice encoded_slice(encoded);
    decoded_footer.DecodeFrom(&encoded_slice);
    ASSERT_EQ(decoded_footer.table_magic_number(), kPlainTableMagicNumber);
    ASSERT_EQ(decoded_footer.checksum(), kxxHash);
    ASSERT_EQ(decoded_footer.metaindex_handle().offset(), meta_index.offset());
    ASSERT_EQ(decoded_footer.metaindex_handle().size(), meta_index.size());
    ASSERT_EQ(decoded_footer.index_handle().offset(), index.offset());
    ASSERT_EQ(decoded_footer.index_handle().size(), index.size());
    ASSERT_EQ(decoded_footer.version(), 1U);
  }
#endif  // !ROCKSDB_LITE
  {
    // version == 2
    std::string encoded;
    Footer footer(kBlockBasedTableMagicNumber, 2);
    BlockHandle meta_index(10, 5), index(20, 15);
    footer.set_metaindex_handle(meta_index);
    footer.set_index_handle(index);
    footer.EncodeTo(&encoded);
    Footer decoded_footer;
    Slice encoded_slice(encoded);
    decoded_footer.DecodeFrom(&encoded_slice);
    ASSERT_EQ(decoded_footer.table_magic_number(), kBlockBasedTableMagicNumber);
    ASSERT_EQ(decoded_footer.checksum(), kCRC32c);
    ASSERT_EQ(decoded_footer.metaindex_handle().offset(), meta_index.offset());
    ASSERT_EQ(decoded_footer.metaindex_handle().size(), meta_index.size());
    ASSERT_EQ(decoded_footer.index_handle().offset(), index.offset());
    ASSERT_EQ(decoded_footer.index_handle().size(), index.size());
    ASSERT_EQ(decoded_footer.version(), 2U);
  }
}

}  // namespace rocksdb

int main(int argc, char** argv) {
  ::testing::InitGoogleTest(&argc, argv);
  return RUN_ALL_TESTS();
}
