// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file. See the AUTHORS file for names of contributors.
//  Copyright (c) 2013, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.
//
// Copyright (c) 2011 The LevelDB Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file. See the AUTHORS file for names of contributors.

#ifndef ROCKSDB_LITE

#include <algorithm>
#include <set>

#include "db/db_impl.h"
#include "db/filename.h"
#include "db/version_set.h"
#include "db/write_batch_internal.h"
#include "rocksdb/cache.h"
#include "rocksdb/compaction_filter.h"
#include "rocksdb/db.h"
#include "rocksdb/env.h"
#include "rocksdb/filter_policy.h"
#include "rocksdb/slice_transform.h"
#include "rocksdb/table.h"
#include "table/meta_blocks.h"
#include "table/bloom_block.h"
#include "table/plain_table_factory.h"
#include "table/plain_table_reader.h"
#include "util/hash.h"
#include "util/logging.h"
#include "util/mutexlock.h"
#include "util/string_util.h"
#include "util/testharness.h"
#include "util/testutil.h"
#include "utilities/merge_operators.h"

using std::unique_ptr;

namespace rocksdb {

class PlainTableDBTest : public testing::Test {
 protected:
 private:
  std::string dbname_;
  Env* env_;
  DB* db_;

  Options last_options_;

 public:
  PlainTableDBTest() : env_(Env::Default()) {
    dbname_ = test::TmpDir() + "/plain_table_db_test";
    EXPECT_OK(DestroyDB(dbname_, Options()));
    db_ = nullptr;
    Reopen();
  }

  ~PlainTableDBTest() {
    delete db_;
    EXPECT_OK(DestroyDB(dbname_, Options()));
  }

  // Return the current option configuration.
  Options CurrentOptions() {
    Options options;

    PlainTableOptions plain_table_options;
    plain_table_options.user_key_len = 0;
    plain_table_options.bloom_bits_per_key = 2;
    plain_table_options.hash_table_ratio = 0.8;
    plain_table_options.index_sparseness = 3;
    plain_table_options.huge_page_tlb_size = 0;
    plain_table_options.encoding_type = kPrefix;
    plain_table_options.full_scan_mode = false;
    plain_table_options.store_index_in_file = false;

    options.table_factory.reset(NewPlainTableFactory(plain_table_options));
    options.memtable_factory.reset(NewHashLinkListRepFactory(4, 0, 3, true));

    options.prefix_extractor.reset(NewFixedPrefixTransform(8));
    options.allow_mmap_reads = true;
    return options;
  }

  DBImpl* dbfull() {
    return reinterpret_cast<DBImpl*>(db_);
  }

  void Reopen(Options* options = nullptr) {
    ASSERT_OK(TryReopen(options));
  }

  void Close() {
    delete db_;
    db_ = nullptr;
  }

  void DestroyAndReopen(Options* options = nullptr) {
    //Destroy using last options
    Destroy(&last_options_);
    ASSERT_OK(TryReopen(options));
  }

  void Destroy(Options* options) {
    delete db_;
    db_ = nullptr;
    ASSERT_OK(DestroyDB(dbname_, *options));
  }

  Status PureReopen(Options* options, DB** db) {
    return DB::Open(*options, dbname_, db);
  }

  Status TryReopen(Options* options = nullptr) {
    delete db_;
    db_ = nullptr;
    Options opts;
    if (options != nullptr) {
      opts = *options;
    } else {
      opts = CurrentOptions();
      opts.create_if_missing = true;
    }
    last_options_ = opts;

    return DB::Open(opts, dbname_, &db_);
  }

  Status Put(const Slice& k, const Slice& v) {
    return db_->Put(WriteOptions(), k, v);
  }

  Status Delete(const std::string& k) {
    return db_->Delete(WriteOptions(), k);
  }

  std::string Get(const std::string& k, const Snapshot* snapshot = nullptr) {
    ReadOptions options;
    options.snapshot = snapshot;
    std::string result;
    Status s = db_->Get(options, k, &result);
    if (s.IsNotFound()) {
      result = "NOT_FOUND";
    } else if (!s.ok()) {
      result = s.ToString();
    }
    return result;
  }


  int NumTableFilesAtLevel(int level) {
    std::string property;
    EXPECT_TRUE(db_->GetProperty(
        "rocksdb.num-files-at-level" + NumberToString(level), &property));
    return atoi(property.c_str());
  }

  // Return spread of files per level
  std::string FilesPerLevel() {
    std::string result;
    size_t last_non_zero_offset = 0;
    for (int level = 0; level < db_->NumberLevels(); level++) {
      int f = NumTableFilesAtLevel(level);
      char buf[100];
      snprintf(buf, sizeof(buf), "%s%d", (level ? "," : ""), f);
      result += buf;
      if (f > 0) {
        last_non_zero_offset = result.size();
      }
    }
    result.resize(last_non_zero_offset);
    return result;
  }

  std::string IterStatus(Iterator* iter) {
    std::string result;
    if (iter->Valid()) {
      result = iter->key().ToString() + "->" + iter->value().ToString();
    } else {
      result = "(invalid)";
    }
    return result;
  }
};

TEST_F(PlainTableDBTest, Empty) {
  ASSERT_TRUE(dbfull() != nullptr);
  ASSERT_EQ("NOT_FOUND", Get("0000000000000foo"));
}

extern const uint64_t kPlainTableMagicNumber;

class TestPlainTableReader : public PlainTableReader {
 public:
  TestPlainTableReader(const EnvOptions& env_options,
                       const InternalKeyComparator& icomparator,
                       EncodingType encoding_type, uint64_t file_size,
                       int bloom_bits_per_key, double hash_table_ratio,
                       size_t index_sparseness,
                       const TableProperties* table_properties,
                       unique_ptr<RandomAccessFileReader>&& file,
                       const ImmutableCFOptions& ioptions,
                       bool* expect_bloom_not_match, bool store_index_in_file)
      : PlainTableReader(ioptions, std::move(file), env_options, icomparator,
                         encoding_type, file_size, table_properties),
        expect_bloom_not_match_(expect_bloom_not_match) {
    Status s = MmapDataFile();
    EXPECT_TRUE(s.ok());

    s = PopulateIndex(const_cast<TableProperties*>(table_properties),
                      bloom_bits_per_key, hash_table_ratio, index_sparseness,
                      2 * 1024 * 1024);
    EXPECT_TRUE(s.ok());

    TableProperties* props = const_cast<TableProperties*>(table_properties);
    if (store_index_in_file) {
      auto bloom_version_ptr = props->user_collected_properties.find(
          PlainTablePropertyNames::kBloomVersion);
      EXPECT_TRUE(bloom_version_ptr != props->user_collected_properties.end());
      EXPECT_EQ(bloom_version_ptr->second, std::string("1"));
      if (ioptions.bloom_locality > 0) {
        auto num_blocks_ptr = props->user_collected_properties.find(
            PlainTablePropertyNames::kNumBloomBlocks);
        EXPECT_TRUE(num_blocks_ptr != props->user_collected_properties.end());
      }
    }
  }

  virtual ~TestPlainTableReader() {}

 private:
  virtual bool MatchBloom(uint32_t hash) const override {
    bool ret = PlainTableReader::MatchBloom(hash);
    if (*expect_bloom_not_match_) {
      EXPECT_TRUE(!ret);
    } else {
      EXPECT_TRUE(ret);
    }
    return ret;
  }
  bool* expect_bloom_not_match_;
};

extern const uint64_t kPlainTableMagicNumber;
class TestPlainTableFactory : public PlainTableFactory {
 public:
  explicit TestPlainTableFactory(bool* expect_bloom_not_match,
                                 const PlainTableOptions& options)
      : PlainTableFactory(options),
        bloom_bits_per_key_(options.bloom_bits_per_key),
        hash_table_ratio_(options.hash_table_ratio),
        index_sparseness_(options.index_sparseness),
        store_index_in_file_(options.store_index_in_file),
        expect_bloom_not_match_(expect_bloom_not_match) {}

  Status NewTableReader(const ImmutableCFOptions& ioptions,
                        const EnvOptions& env_options,
                        const InternalKeyComparator& internal_comparator,
                        unique_ptr<RandomAccessFileReader>&& file,
                        uint64_t file_size,
                        unique_ptr<TableReader>* table) const override {
    TableProperties* props = nullptr;
    auto s = ReadTableProperties(file.get(), file_size, kPlainTableMagicNumber,
                                 ioptions.env, ioptions.info_log, &props);
    EXPECT_TRUE(s.ok());

    if (store_index_in_file_) {
      BlockHandle bloom_block_handle;
      s = FindMetaBlock(file.get(), file_size, kPlainTableMagicNumber,
                        ioptions.env, BloomBlockBuilder::kBloomBlock,
                        &bloom_block_handle);
      EXPECT_TRUE(s.ok());

      BlockHandle index_block_handle;
      s = FindMetaBlock(
          file.get(), file_size, kPlainTableMagicNumber, ioptions.env,
          PlainTableIndexBuilder::kPlainTableIndexBlock, &index_block_handle);
      EXPECT_TRUE(s.ok());
    }

    auto& user_props = props->user_collected_properties;
    auto encoding_type_prop =
        user_props.find(PlainTablePropertyNames::kEncodingType);
    assert(encoding_type_prop != user_props.end());
    EncodingType encoding_type = static_cast<EncodingType>(
        DecodeFixed32(encoding_type_prop->second.c_str()));

    std::unique_ptr<PlainTableReader> new_reader(new TestPlainTableReader(
        env_options, internal_comparator, encoding_type, file_size,
        bloom_bits_per_key_, hash_table_ratio_, index_sparseness_, props,
        std::move(file), ioptions, expect_bloom_not_match_,
        store_index_in_file_));

    *table = std::move(new_reader);
    return s;
  }

 private:
  int bloom_bits_per_key_;
  double hash_table_ratio_;
  size_t index_sparseness_;
  bool store_index_in_file_;
  bool* expect_bloom_not_match_;
};

TEST_F(PlainTableDBTest, Flush) {
  for (size_t huge_page_tlb_size = 0; huge_page_tlb_size <= 2 * 1024 * 1024;
       huge_page_tlb_size += 2 * 1024 * 1024) {
    for (EncodingType encoding_type : {kPlain, kPrefix}) {
    for (int bloom_bits = 0; bloom_bits <= 117; bloom_bits += 117) {
      for (int total_order = 0; total_order <= 1; total_order++) {
        for (int store_index_in_file = 0; store_index_in_file <= 1;
             ++store_index_in_file) {
          if (!bloom_bits && store_index_in_file) {
            continue;
          }

          Options options = CurrentOptions();
          options.create_if_missing = true;
          // Set only one bucket to force bucket conflict.
          // Test index interval for the same prefix to be 1, 2 and 4
          if (total_order) {
            options.prefix_extractor.reset();

            PlainTableOptions plain_table_options;
            plain_table_options.user_key_len = 0;
            plain_table_options.bloom_bits_per_key = bloom_bits;
            plain_table_options.hash_table_ratio = 0;
            plain_table_options.index_sparseness = 2;
            plain_table_options.huge_page_tlb_size = huge_page_tlb_size;
            plain_table_options.encoding_type = encoding_type;
            plain_table_options.full_scan_mode = false;
            plain_table_options.store_index_in_file = store_index_in_file;

            options.table_factory.reset(
                NewPlainTableFactory(plain_table_options));
          } else {
            PlainTableOptions plain_table_options;
            plain_table_options.user_key_len = 0;
            plain_table_options.bloom_bits_per_key = bloom_bits;
            plain_table_options.hash_table_ratio = 0.75;
            plain_table_options.index_sparseness = 16;
            plain_table_options.huge_page_tlb_size = huge_page_tlb_size;
            plain_table_options.encoding_type = encoding_type;
            plain_table_options.full_scan_mode = false;
            plain_table_options.store_index_in_file = store_index_in_file;

            options.table_factory.reset(
                NewPlainTableFactory(plain_table_options));
          }
          DestroyAndReopen(&options);
          uint64_t int_num;
          ASSERT_TRUE(dbfull()->GetIntProperty(
              "rocksdb.estimate-table-readers-mem", &int_num));
          ASSERT_EQ(int_num, 0U);

          ASSERT_OK(Put("1000000000000foo", "v1"));
          ASSERT_OK(Put("0000000000000bar", "v2"));
          ASSERT_OK(Put("1000000000000foo", "v3"));
          dbfull()->TEST_FlushMemTable();

          ASSERT_TRUE(dbfull()->GetIntProperty(
              "rocksdb.estimate-table-readers-mem", &int_num));
          ASSERT_GT(int_num, 0U);

          TablePropertiesCollection ptc;
          reinterpret_cast<DB*>(dbfull())->GetPropertiesOfAllTables(&ptc);
          ASSERT_EQ(1U, ptc.size());
          auto row = ptc.begin();
          auto tp = row->second;

          if (!store_index_in_file) {
            ASSERT_EQ(total_order ? "4" : "12",
                      (tp->user_collected_properties)
                          .at("plain_table_hash_table_size"));
            ASSERT_EQ("0", (tp->user_collected_properties)
                               .at("plain_table_sub_index_size"));
          } else {
            ASSERT_EQ("0", (tp->user_collected_properties)
                               .at("plain_table_hash_table_size"));
            ASSERT_EQ("0", (tp->user_collected_properties)
                               .at("plain_table_sub_index_size"));
          }
          ASSERT_EQ("v3", Get("1000000000000foo"));
          ASSERT_EQ("v2", Get("0000000000000bar"));
        }
        }
      }
    }
  }
}

TEST_F(PlainTableDBTest, Flush2) {
  for (size_t huge_page_tlb_size = 0; huge_page_tlb_size <= 2 * 1024 * 1024;
       huge_page_tlb_size += 2 * 1024 * 1024) {
    for (EncodingType encoding_type : {kPlain, kPrefix}) {
    for (int bloom_bits = 0; bloom_bits <= 117; bloom_bits += 117) {
      for (int total_order = 0; total_order <= 1; total_order++) {
        for (int store_index_in_file = 0; store_index_in_file <= 1;
             ++store_index_in_file) {
          if (encoding_type == kPrefix && total_order) {
            continue;
          }
          if (!bloom_bits && store_index_in_file) {
            continue;
          }
          if (total_order && store_index_in_file) {
          continue;
        }
        bool expect_bloom_not_match = false;
        Options options = CurrentOptions();
        options.create_if_missing = true;
        // Set only one bucket to force bucket conflict.
        // Test index interval for the same prefix to be 1, 2 and 4
        PlainTableOptions plain_table_options;
        if (total_order) {
          options.prefix_extractor = nullptr;
          plain_table_options.hash_table_ratio = 0;
          plain_table_options.index_sparseness = 2;
        } else {
          plain_table_options.hash_table_ratio = 0.75;
          plain_table_options.index_sparseness = 16;
        }
        plain_table_options.user_key_len = kPlainTableVariableLength;
        plain_table_options.bloom_bits_per_key = bloom_bits;
        plain_table_options.huge_page_tlb_size = huge_page_tlb_size;
        plain_table_options.encoding_type = encoding_type;
        plain_table_options.store_index_in_file = store_index_in_file;
        options.table_factory.reset(new TestPlainTableFactory(
            &expect_bloom_not_match, plain_table_options));

        DestroyAndReopen(&options);
        ASSERT_OK(Put("0000000000000bar", "b"));
        ASSERT_OK(Put("1000000000000foo", "v1"));
        dbfull()->TEST_FlushMemTable();

        ASSERT_OK(Put("1000000000000foo", "v2"));
        dbfull()->TEST_FlushMemTable();
        ASSERT_EQ("v2", Get("1000000000000foo"));

        ASSERT_OK(Put("0000000000000eee", "v3"));
        dbfull()->TEST_FlushMemTable();
        ASSERT_EQ("v3", Get("0000000000000eee"));

        ASSERT_OK(Delete("0000000000000bar"));
        dbfull()->TEST_FlushMemTable();
        ASSERT_EQ("NOT_FOUND", Get("0000000000000bar"));

        ASSERT_OK(Put("0000000000000eee", "v5"));
        ASSERT_OK(Put("9000000000000eee", "v5"));
        dbfull()->TEST_FlushMemTable();
        ASSERT_EQ("v5", Get("0000000000000eee"));

        // Test Bloom Filter
        if (bloom_bits > 0) {
          // Neither key nor value should exist.
          expect_bloom_not_match = true;
          ASSERT_EQ("NOT_FOUND", Get("5_not00000000bar"));
          // Key doesn't exist any more but prefix exists.
          if (total_order) {
            ASSERT_EQ("NOT_FOUND", Get("1000000000000not"));
            ASSERT_EQ("NOT_FOUND", Get("0000000000000not"));
          }
          expect_bloom_not_match = false;
        }
      }
      }
    }
    }
  }
}

TEST_F(PlainTableDBTest, Iterator) {
  for (size_t huge_page_tlb_size = 0; huge_page_tlb_size <= 2 * 1024 * 1024;
       huge_page_tlb_size += 2 * 1024 * 1024) {
    for (EncodingType encoding_type : {kPlain, kPrefix}) {
    for (int bloom_bits = 0; bloom_bits <= 117; bloom_bits += 117) {
      for (int total_order = 0; total_order <= 1; total_order++) {
        if (encoding_type == kPrefix && total_order == 1) {
          continue;
        }
        bool expect_bloom_not_match = false;
        Options options = CurrentOptions();
        options.create_if_missing = true;
        // Set only one bucket to force bucket conflict.
        // Test index interval for the same prefix to be 1, 2 and 4
        if (total_order) {
          options.prefix_extractor = nullptr;

          PlainTableOptions plain_table_options;
          plain_table_options.user_key_len = 16;
          plain_table_options.bloom_bits_per_key = bloom_bits;
          plain_table_options.hash_table_ratio = 0;
          plain_table_options.index_sparseness = 2;
          plain_table_options.huge_page_tlb_size = huge_page_tlb_size;
          plain_table_options.encoding_type = encoding_type;

          options.table_factory.reset(new TestPlainTableFactory(
              &expect_bloom_not_match, plain_table_options));
        } else {
          PlainTableOptions plain_table_options;
          plain_table_options.user_key_len = 16;
          plain_table_options.bloom_bits_per_key = bloom_bits;
          plain_table_options.hash_table_ratio = 0.75;
          plain_table_options.index_sparseness = 16;
          plain_table_options.huge_page_tlb_size = huge_page_tlb_size;
          plain_table_options.encoding_type = encoding_type;

          options.table_factory.reset(new TestPlainTableFactory(
              &expect_bloom_not_match, plain_table_options));
        }
        DestroyAndReopen(&options);

        ASSERT_OK(Put("1000000000foo002", "v_2"));
        ASSERT_OK(Put("0000000000000bar", "random"));
        ASSERT_OK(Put("1000000000foo001", "v1"));
        ASSERT_OK(Put("3000000000000bar", "bar_v"));
        ASSERT_OK(Put("1000000000foo003", "v__3"));
        ASSERT_OK(Put("1000000000foo004", "v__4"));
        ASSERT_OK(Put("1000000000foo005", "v__5"));
        ASSERT_OK(Put("1000000000foo007", "v__7"));
        ASSERT_OK(Put("1000000000foo008", "v__8"));
        dbfull()->TEST_FlushMemTable();
        ASSERT_EQ("v1", Get("1000000000foo001"));
        ASSERT_EQ("v__3", Get("1000000000foo003"));
        Iterator* iter = dbfull()->NewIterator(ReadOptions());
        iter->Seek("1000000000foo000");
        ASSERT_TRUE(iter->Valid());
        ASSERT_EQ("1000000000foo001", iter->key().ToString());
        ASSERT_EQ("v1", iter->value().ToString());

        iter->Next();
        ASSERT_TRUE(iter->Valid());
        ASSERT_EQ("1000000000foo002", iter->key().ToString());
        ASSERT_EQ("v_2", iter->value().ToString());

        iter->Next();
        ASSERT_TRUE(iter->Valid());
        ASSERT_EQ("1000000000foo003", iter->key().ToString());
        ASSERT_EQ("v__3", iter->value().ToString());

        iter->Next();
        ASSERT_TRUE(iter->Valid());
        ASSERT_EQ("1000000000foo004", iter->key().ToString());
        ASSERT_EQ("v__4", iter->value().ToString());

        iter->Seek("3000000000000bar");
        ASSERT_TRUE(iter->Valid());
        ASSERT_EQ("3000000000000bar", iter->key().ToString());
        ASSERT_EQ("bar_v", iter->value().ToString());

        iter->Seek("1000000000foo000");
        ASSERT_TRUE(iter->Valid());
        ASSERT_EQ("1000000000foo001", iter->key().ToString());
        ASSERT_EQ("v1", iter->value().ToString());

        iter->Seek("1000000000foo005");
        ASSERT_TRUE(iter->Valid());
        ASSERT_EQ("1000000000foo005", iter->key().ToString());
        ASSERT_EQ("v__5", iter->value().ToString());

        iter->Seek("1000000000foo006");
        ASSERT_TRUE(iter->Valid());
        ASSERT_EQ("1000000000foo007", iter->key().ToString());
        ASSERT_EQ("v__7", iter->value().ToString());

        iter->Seek("1000000000foo008");
        ASSERT_TRUE(iter->Valid());
        ASSERT_EQ("1000000000foo008", iter->key().ToString());
        ASSERT_EQ("v__8", iter->value().ToString());

        if (total_order == 0) {
          iter->Seek("1000000000foo009");
          ASSERT_TRUE(iter->Valid());
          ASSERT_EQ("3000000000000bar", iter->key().ToString());
        }

        // Test Bloom Filter
        if (bloom_bits > 0) {
          if (!total_order) {
            // Neither key nor value should exist.
            expect_bloom_not_match = true;
            iter->Seek("2not000000000bar");
            ASSERT_TRUE(!iter->Valid());
            ASSERT_EQ("NOT_FOUND", Get("2not000000000bar"));
            expect_bloom_not_match = false;
          } else {
            expect_bloom_not_match = true;
            ASSERT_EQ("NOT_FOUND", Get("2not000000000bar"));
            expect_bloom_not_match = false;
          }
        }

        delete iter;
      }
    }
    }
  }
}

namespace {
std::string MakeLongKey(size_t length, char c) {
  return std::string(length, c);
}
}  // namespace

TEST_F(PlainTableDBTest, IteratorLargeKeys) {
  Options options = CurrentOptions();

  PlainTableOptions plain_table_options;
  plain_table_options.user_key_len = 0;
  plain_table_options.bloom_bits_per_key = 0;
  plain_table_options.hash_table_ratio = 0;

  options.table_factory.reset(NewPlainTableFactory(plain_table_options));
  options.create_if_missing = true;
  options.prefix_extractor.reset();
  DestroyAndReopen(&options);

  std::string key_list[] = {
      MakeLongKey(30, '0'),
      MakeLongKey(16, '1'),
      MakeLongKey(32, '2'),
      MakeLongKey(60, '3'),
      MakeLongKey(90, '4'),
      MakeLongKey(50, '5'),
      MakeLongKey(26, '6')
  };

  for (size_t i = 0; i < 7; i++) {
    ASSERT_OK(Put(key_list[i], ToString(i)));
  }

  dbfull()->TEST_FlushMemTable();

  Iterator* iter = dbfull()->NewIterator(ReadOptions());
  iter->Seek(key_list[0]);

  for (size_t i = 0; i < 7; i++) {
    ASSERT_TRUE(iter->Valid());
    ASSERT_EQ(key_list[i], iter->key().ToString());
    ASSERT_EQ(ToString(i), iter->value().ToString());
    iter->Next();
  }

  ASSERT_TRUE(!iter->Valid());

  delete iter;
}

namespace {
std::string MakeLongKeyWithPrefix(size_t length, char c) {
  return "00000000" + std::string(length - 8, c);
}
}  // namespace

TEST_F(PlainTableDBTest, IteratorLargeKeysWithPrefix) {
  Options options = CurrentOptions();

  PlainTableOptions plain_table_options;
  plain_table_options.user_key_len = 16;
  plain_table_options.bloom_bits_per_key = 0;
  plain_table_options.hash_table_ratio = 0.8;
  plain_table_options.index_sparseness = 3;
  plain_table_options.huge_page_tlb_size = 0;
  plain_table_options.encoding_type = kPrefix;

  options.table_factory.reset(NewPlainTableFactory(plain_table_options));
  options.create_if_missing = true;
  DestroyAndReopen(&options);

  std::string key_list[] = {
      MakeLongKeyWithPrefix(30, '0'), MakeLongKeyWithPrefix(16, '1'),
      MakeLongKeyWithPrefix(32, '2'), MakeLongKeyWithPrefix(60, '3'),
      MakeLongKeyWithPrefix(90, '4'), MakeLongKeyWithPrefix(50, '5'),
      MakeLongKeyWithPrefix(26, '6')};

  for (size_t i = 0; i < 7; i++) {
    ASSERT_OK(Put(key_list[i], ToString(i)));
  }

  dbfull()->TEST_FlushMemTable();

  Iterator* iter = dbfull()->NewIterator(ReadOptions());
  iter->Seek(key_list[0]);

  for (size_t i = 0; i < 7; i++) {
    ASSERT_TRUE(iter->Valid());
    ASSERT_EQ(key_list[i], iter->key().ToString());
    ASSERT_EQ(ToString(i), iter->value().ToString());
    iter->Next();
  }

  ASSERT_TRUE(!iter->Valid());

  delete iter;
}

TEST_F(PlainTableDBTest, IteratorReverseSuffixComparator) {
  Options options = CurrentOptions();
  options.create_if_missing = true;
  // Set only one bucket to force bucket conflict.
  // Test index interval for the same prefix to be 1, 2 and 4
  test::SimpleSuffixReverseComparator comp;
  options.comparator = &comp;
  DestroyAndReopen(&options);

  ASSERT_OK(Put("1000000000foo002", "v_2"));
  ASSERT_OK(Put("0000000000000bar", "random"));
  ASSERT_OK(Put("1000000000foo001", "v1"));
  ASSERT_OK(Put("3000000000000bar", "bar_v"));
  ASSERT_OK(Put("1000000000foo003", "v__3"));
  ASSERT_OK(Put("1000000000foo004", "v__4"));
  ASSERT_OK(Put("1000000000foo005", "v__5"));
  ASSERT_OK(Put("1000000000foo007", "v__7"));
  ASSERT_OK(Put("1000000000foo008", "v__8"));
  dbfull()->TEST_FlushMemTable();
  ASSERT_EQ("v1", Get("1000000000foo001"));
  ASSERT_EQ("v__3", Get("1000000000foo003"));
  Iterator* iter = dbfull()->NewIterator(ReadOptions());
  iter->Seek("1000000000foo009");
  ASSERT_TRUE(iter->Valid());
  ASSERT_EQ("1000000000foo008", iter->key().ToString());
  ASSERT_EQ("v__8", iter->value().ToString());

  iter->Next();
  ASSERT_TRUE(iter->Valid());
  ASSERT_EQ("1000000000foo007", iter->key().ToString());
  ASSERT_EQ("v__7", iter->value().ToString());

  iter->Next();
  ASSERT_TRUE(iter->Valid());
  ASSERT_EQ("1000000000foo005", iter->key().ToString());
  ASSERT_EQ("v__5", iter->value().ToString());

  iter->Next();
  ASSERT_TRUE(iter->Valid());
  ASSERT_EQ("1000000000foo004", iter->key().ToString());
  ASSERT_EQ("v__4", iter->value().ToString());

  iter->Seek("3000000000000bar");
  ASSERT_TRUE(iter->Valid());
  ASSERT_EQ("3000000000000bar", iter->key().ToString());
  ASSERT_EQ("bar_v", iter->value().ToString());

  iter->Seek("1000000000foo005");
  ASSERT_TRUE(iter->Valid());
  ASSERT_EQ("1000000000foo005", iter->key().ToString());
  ASSERT_EQ("v__5", iter->value().ToString());

  iter->Seek("1000000000foo006");
  ASSERT_TRUE(iter->Valid());
  ASSERT_EQ("1000000000foo005", iter->key().ToString());
  ASSERT_EQ("v__5", iter->value().ToString());

  iter->Seek("1000000000foo008");
  ASSERT_TRUE(iter->Valid());
  ASSERT_EQ("1000000000foo008", iter->key().ToString());
  ASSERT_EQ("v__8", iter->value().ToString());

  iter->Seek("1000000000foo000");
  ASSERT_TRUE(iter->Valid());
  ASSERT_EQ("3000000000000bar", iter->key().ToString());

  delete iter;
}

TEST_F(PlainTableDBTest, HashBucketConflict) {
  for (size_t huge_page_tlb_size = 0; huge_page_tlb_size <= 2 * 1024 * 1024;
       huge_page_tlb_size += 2 * 1024 * 1024) {
    for (unsigned char i = 1; i <= 3; i++) {
      Options options = CurrentOptions();
      options.create_if_missing = true;
      // Set only one bucket to force bucket conflict.
      // Test index interval for the same prefix to be 1, 2 and 4

      PlainTableOptions plain_table_options;
      plain_table_options.user_key_len = 16;
      plain_table_options.bloom_bits_per_key = 0;
      plain_table_options.hash_table_ratio = 0;
      plain_table_options.index_sparseness = 2 ^ i;
      plain_table_options.huge_page_tlb_size = huge_page_tlb_size;

      options.table_factory.reset(NewPlainTableFactory(plain_table_options));

      DestroyAndReopen(&options);
      ASSERT_OK(Put("5000000000000fo0", "v1"));
      ASSERT_OK(Put("5000000000000fo1", "v2"));
      ASSERT_OK(Put("5000000000000fo2", "v"));
      ASSERT_OK(Put("2000000000000fo0", "v3"));
      ASSERT_OK(Put("2000000000000fo1", "v4"));
      ASSERT_OK(Put("2000000000000fo2", "v"));
      ASSERT_OK(Put("2000000000000fo3", "v"));

      dbfull()->TEST_FlushMemTable();

      ASSERT_EQ("v1", Get("5000000000000fo0"));
      ASSERT_EQ("v2", Get("5000000000000fo1"));
      ASSERT_EQ("v3", Get("2000000000000fo0"));
      ASSERT_EQ("v4", Get("2000000000000fo1"));

      ASSERT_EQ("NOT_FOUND", Get("5000000000000bar"));
      ASSERT_EQ("NOT_FOUND", Get("2000000000000bar"));
      ASSERT_EQ("NOT_FOUND", Get("5000000000000fo8"));
      ASSERT_EQ("NOT_FOUND", Get("2000000000000fo8"));

      ReadOptions ro;
      Iterator* iter = dbfull()->NewIterator(ro);

      iter->Seek("5000000000000fo0");
      ASSERT_TRUE(iter->Valid());
      ASSERT_EQ("5000000000000fo0", iter->key().ToString());
      iter->Next();
      ASSERT_TRUE(iter->Valid());
      ASSERT_EQ("5000000000000fo1", iter->key().ToString());

      iter->Seek("5000000000000fo1");
      ASSERT_TRUE(iter->Valid());
      ASSERT_EQ("5000000000000fo1", iter->key().ToString());

      iter->Seek("2000000000000fo0");
      ASSERT_TRUE(iter->Valid());
      ASSERT_EQ("2000000000000fo0", iter->key().ToString());
      iter->Next();
      ASSERT_TRUE(iter->Valid());
      ASSERT_EQ("2000000000000fo1", iter->key().ToString());

      iter->Seek("2000000000000fo1");
      ASSERT_TRUE(iter->Valid());
      ASSERT_EQ("2000000000000fo1", iter->key().ToString());

      iter->Seek("2000000000000bar");
      ASSERT_TRUE(iter->Valid());
      ASSERT_EQ("2000000000000fo0", iter->key().ToString());

      iter->Seek("5000000000000bar");
      ASSERT_TRUE(iter->Valid());
      ASSERT_EQ("5000000000000fo0", iter->key().ToString());

      iter->Seek("2000000000000fo8");
      ASSERT_TRUE(!iter->Valid() ||
                  options.comparator->Compare(iter->key(), "20000001") > 0);

      iter->Seek("5000000000000fo8");
      ASSERT_TRUE(!iter->Valid());

      iter->Seek("1000000000000fo2");
      ASSERT_TRUE(!iter->Valid());

      iter->Seek("3000000000000fo2");
      ASSERT_TRUE(!iter->Valid());

      iter->Seek("8000000000000fo2");
      ASSERT_TRUE(!iter->Valid());

      delete iter;
    }
  }
}

TEST_F(PlainTableDBTest, HashBucketConflictReverseSuffixComparator) {
  for (size_t huge_page_tlb_size = 0; huge_page_tlb_size <= 2 * 1024 * 1024;
       huge_page_tlb_size += 2 * 1024 * 1024) {
    for (unsigned char i = 1; i <= 3; i++) {
      Options options = CurrentOptions();
      options.create_if_missing = true;
      test::SimpleSuffixReverseComparator comp;
      options.comparator = &comp;
      // Set only one bucket to force bucket conflict.
      // Test index interval for the same prefix to be 1, 2 and 4

      PlainTableOptions plain_table_options;
      plain_table_options.user_key_len = 16;
      plain_table_options.bloom_bits_per_key = 0;
      plain_table_options.hash_table_ratio = 0;
      plain_table_options.index_sparseness = 2 ^ i;
      plain_table_options.huge_page_tlb_size = huge_page_tlb_size;

      options.table_factory.reset(NewPlainTableFactory(plain_table_options));
      DestroyAndReopen(&options);
      ASSERT_OK(Put("5000000000000fo0", "v1"));
      ASSERT_OK(Put("5000000000000fo1", "v2"));
      ASSERT_OK(Put("5000000000000fo2", "v"));
      ASSERT_OK(Put("2000000000000fo0", "v3"));
      ASSERT_OK(Put("2000000000000fo1", "v4"));
      ASSERT_OK(Put("2000000000000fo2", "v"));
      ASSERT_OK(Put("2000000000000fo3", "v"));

      dbfull()->TEST_FlushMemTable();

      ASSERT_EQ("v1", Get("5000000000000fo0"));
      ASSERT_EQ("v2", Get("5000000000000fo1"));
      ASSERT_EQ("v3", Get("2000000000000fo0"));
      ASSERT_EQ("v4", Get("2000000000000fo1"));

      ASSERT_EQ("NOT_FOUND", Get("5000000000000bar"));
      ASSERT_EQ("NOT_FOUND", Get("2000000000000bar"));
      ASSERT_EQ("NOT_FOUND", Get("5000000000000fo8"));
      ASSERT_EQ("NOT_FOUND", Get("2000000000000fo8"));

      ReadOptions ro;
      Iterator* iter = dbfull()->NewIterator(ro);

      iter->Seek("5000000000000fo1");
      ASSERT_TRUE(iter->Valid());
      ASSERT_EQ("5000000000000fo1", iter->key().ToString());
      iter->Next();
      ASSERT_TRUE(iter->Valid());
      ASSERT_EQ("5000000000000fo0", iter->key().ToString());

      iter->Seek("5000000000000fo1");
      ASSERT_TRUE(iter->Valid());
      ASSERT_EQ("5000000000000fo1", iter->key().ToString());

      iter->Seek("2000000000000fo1");
      ASSERT_TRUE(iter->Valid());
      ASSERT_EQ("2000000000000fo1", iter->key().ToString());
      iter->Next();
      ASSERT_TRUE(iter->Valid());
      ASSERT_EQ("2000000000000fo0", iter->key().ToString());

      iter->Seek("2000000000000fo1");
      ASSERT_TRUE(iter->Valid());
      ASSERT_EQ("2000000000000fo1", iter->key().ToString());

      iter->Seek("2000000000000var");
      ASSERT_TRUE(iter->Valid());
      ASSERT_EQ("2000000000000fo3", iter->key().ToString());

      iter->Seek("5000000000000var");
      ASSERT_TRUE(iter->Valid());
      ASSERT_EQ("5000000000000fo2", iter->key().ToString());

      std::string seek_key = "2000000000000bar";
      iter->Seek(seek_key);
      ASSERT_TRUE(!iter->Valid() ||
                  options.prefix_extractor->Transform(iter->key()) !=
                      options.prefix_extractor->Transform(seek_key));

      iter->Seek("1000000000000fo2");
      ASSERT_TRUE(!iter->Valid());

      iter->Seek("3000000000000fo2");
      ASSERT_TRUE(!iter->Valid());

      iter->Seek("8000000000000fo2");
      ASSERT_TRUE(!iter->Valid());

      delete iter;
    }
  }
}

TEST_F(PlainTableDBTest, NonExistingKeyToNonEmptyBucket) {
  Options options = CurrentOptions();
  options.create_if_missing = true;
  // Set only one bucket to force bucket conflict.
  // Test index interval for the same prefix to be 1, 2 and 4
  PlainTableOptions plain_table_options;
  plain_table_options.user_key_len = 16;
  plain_table_options.bloom_bits_per_key = 0;
  plain_table_options.hash_table_ratio = 0;
  plain_table_options.index_sparseness = 5;

  options.table_factory.reset(NewPlainTableFactory(plain_table_options));
  DestroyAndReopen(&options);
  ASSERT_OK(Put("5000000000000fo0", "v1"));
  ASSERT_OK(Put("5000000000000fo1", "v2"));
  ASSERT_OK(Put("5000000000000fo2", "v3"));

  dbfull()->TEST_FlushMemTable();

  ASSERT_EQ("v1", Get("5000000000000fo0"));
  ASSERT_EQ("v2", Get("5000000000000fo1"));
  ASSERT_EQ("v3", Get("5000000000000fo2"));

  ASSERT_EQ("NOT_FOUND", Get("8000000000000bar"));
  ASSERT_EQ("NOT_FOUND", Get("1000000000000bar"));

  Iterator* iter = dbfull()->NewIterator(ReadOptions());

  iter->Seek("5000000000000bar");
  ASSERT_TRUE(iter->Valid());
  ASSERT_EQ("5000000000000fo0", iter->key().ToString());

  iter->Seek("5000000000000fo8");
  ASSERT_TRUE(!iter->Valid());

  iter->Seek("1000000000000fo2");
  ASSERT_TRUE(!iter->Valid());

  iter->Seek("8000000000000fo2");
  ASSERT_TRUE(!iter->Valid());

  delete iter;
}

static std::string Key(int i) {
  char buf[100];
  snprintf(buf, sizeof(buf), "key_______%06d", i);
  return std::string(buf);
}

static std::string RandomString(Random* rnd, int len) {
  std::string r;
  test::RandomString(rnd, len, &r);
  return r;
}

TEST_F(PlainTableDBTest, CompactionTrigger) {
  Options options = CurrentOptions();
  options.write_buffer_size = 120 << 10;  // 100KB
  options.num_levels = 3;
  options.level0_file_num_compaction_trigger = 3;
  Reopen(&options);

  Random rnd(301);

  for (int num = 0; num < options.level0_file_num_compaction_trigger - 1;
      num++) {
    std::vector<std::string> values;
    // Write 120KB (10 values, each 12K)
    for (int i = 0; i < 10; i++) {
      values.push_back(RandomString(&rnd, 12000));
      ASSERT_OK(Put(Key(i), values[i]));
    }
    ASSERT_OK(Put(Key(999), ""));
    dbfull()->TEST_WaitForFlushMemTable();
    ASSERT_EQ(NumTableFilesAtLevel(0), num + 1);
  }

  //generate one more file in level-0, and should trigger level-0 compaction
  std::vector<std::string> values;
  for (int i = 0; i < 12; i++) {
    values.push_back(RandomString(&rnd, 10000));
    ASSERT_OK(Put(Key(i), values[i]));
  }
  ASSERT_OK(Put(Key(999), ""));
  dbfull()->TEST_WaitForCompact();

  ASSERT_EQ(NumTableFilesAtLevel(0), 0);
  ASSERT_EQ(NumTableFilesAtLevel(1), 1);
}

TEST_F(PlainTableDBTest, AdaptiveTable) {
  Options options = CurrentOptions();
  options.create_if_missing = true;

  options.table_factory.reset(NewPlainTableFactory());
  DestroyAndReopen(&options);

  ASSERT_OK(Put("1000000000000foo", "v1"));
  ASSERT_OK(Put("0000000000000bar", "v2"));
  ASSERT_OK(Put("1000000000000foo", "v3"));
  dbfull()->TEST_FlushMemTable();

  options.create_if_missing = false;
  std::shared_ptr<TableFactory> dummy_factory;
  std::shared_ptr<TableFactory> block_based_factory(
      NewBlockBasedTableFactory());
  options.table_factory.reset(NewAdaptiveTableFactory(
      block_based_factory, dummy_factory, dummy_factory));
  Reopen(&options);
  ASSERT_EQ("v3", Get("1000000000000foo"));
  ASSERT_EQ("v2", Get("0000000000000bar"));

  ASSERT_OK(Put("2000000000000foo", "v4"));
  ASSERT_OK(Put("3000000000000bar", "v5"));
  dbfull()->TEST_FlushMemTable();
  ASSERT_EQ("v4", Get("2000000000000foo"));
  ASSERT_EQ("v5", Get("3000000000000bar"));

  Reopen(&options);
  ASSERT_EQ("v3", Get("1000000000000foo"));
  ASSERT_EQ("v2", Get("0000000000000bar"));
  ASSERT_EQ("v4", Get("2000000000000foo"));
  ASSERT_EQ("v5", Get("3000000000000bar"));

  options.table_factory.reset(NewBlockBasedTableFactory());
  Reopen(&options);
  ASSERT_NE("v3", Get("1000000000000foo"));

  options.table_factory.reset(NewPlainTableFactory());
  Reopen(&options);
  ASSERT_NE("v5", Get("3000000000000bar"));
}

}  // namespace rocksdb

int main(int argc, char** argv) {
  ::testing::InitGoogleTest(&argc, argv);
  return RUN_ALL_TESTS();
}

#else
#include <stdio.h>

int main(int argc, char** argv) {
  fprintf(stderr, "SKIPPED as plain table is not supported in ROCKSDB_LITE\n");
  return 0;
}

#endif  // !ROCKSDB_LITE
