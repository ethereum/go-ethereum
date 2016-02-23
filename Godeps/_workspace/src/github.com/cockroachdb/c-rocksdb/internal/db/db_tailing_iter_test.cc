//  Copyright (c) 2013, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.
//
// Copyright (c) 2011 The LevelDB Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file. See the AUTHORS file for names of contributors.

// Introduction of SyncPoint effectively disabled building and running this test
// in Release build.
// which is a pity, it is a good test
#if !(defined NDEBUG) || !defined(OS_WIN)

#include "db/forward_iterator.h"
#include "port/stack_trace.h"
#include "util/db_test_util.h"

namespace rocksdb {

class DBTestTailingIterator : public DBTestBase {
 public:
  DBTestTailingIterator() : DBTestBase("/db_tailing_iterator_test") {}
};

TEST_F(DBTestTailingIterator, TailingIteratorSingle) {
  ReadOptions read_options;
  read_options.tailing = true;

  std::unique_ptr<Iterator> iter(db_->NewIterator(read_options));
  iter->SeekToFirst();
  ASSERT_TRUE(!iter->Valid());

  // add a record and check that iter can see it
  ASSERT_OK(db_->Put(WriteOptions(), "mirko", "fodor"));
  iter->SeekToFirst();
  ASSERT_TRUE(iter->Valid());
  ASSERT_EQ(iter->key().ToString(), "mirko");

  iter->Next();
  ASSERT_TRUE(!iter->Valid());
}

TEST_F(DBTestTailingIterator, TailingIteratorKeepAdding) {
  CreateAndReopenWithCF({"pikachu"}, CurrentOptions());
  ReadOptions read_options;
  read_options.tailing = true;

  std::unique_ptr<Iterator> iter(db_->NewIterator(read_options, handles_[1]));
  std::string value(1024, 'a');

  const int num_records = 10000;
  for (int i = 0; i < num_records; ++i) {
    char buf[32];
    snprintf(buf, sizeof(buf), "%016d", i);

    Slice key(buf, 16);
    ASSERT_OK(Put(1, key, value));

    iter->Seek(key);
    ASSERT_TRUE(iter->Valid());
    ASSERT_EQ(iter->key().compare(key), 0);
  }
}

TEST_F(DBTestTailingIterator, TailingIteratorSeekToNext) {
  CreateAndReopenWithCF({"pikachu"}, CurrentOptions());
  ReadOptions read_options;
  read_options.tailing = true;

  std::unique_ptr<Iterator> iter(db_->NewIterator(read_options, handles_[1]));
  std::unique_ptr<Iterator> itern(db_->NewIterator(read_options, handles_[1]));
  std::string value(1024, 'a');

  const int num_records = 1000;
  for (int i = 1; i < num_records; ++i) {
    char buf1[32];
    char buf2[32];
    snprintf(buf1, sizeof(buf1), "00a0%016d", i * 5);

    Slice key(buf1, 20);
    ASSERT_OK(Put(1, key, value));

    if (i % 100 == 99) {
      ASSERT_OK(Flush(1));
    }

    snprintf(buf2, sizeof(buf2), "00a0%016d", i * 5 - 2);
    Slice target(buf2, 20);
    iter->Seek(target);
    ASSERT_TRUE(iter->Valid());
    ASSERT_EQ(iter->key().compare(key), 0);
    if (i == 1) {
      itern->SeekToFirst();
    } else {
      itern->Next();
    }
    ASSERT_TRUE(itern->Valid());
    ASSERT_EQ(itern->key().compare(key), 0);
  }
  rocksdb::SyncPoint::GetInstance()->ClearAllCallBacks();
  rocksdb::SyncPoint::GetInstance()->DisableProcessing();
  for (int i = 2 * num_records; i > 0; --i) {
    char buf1[32];
    char buf2[32];
    snprintf(buf1, sizeof(buf1), "00a0%016d", i * 5);

    Slice key(buf1, 20);
    ASSERT_OK(Put(1, key, value));

    if (i % 100 == 99) {
      ASSERT_OK(Flush(1));
    }

    snprintf(buf2, sizeof(buf2), "00a0%016d", i * 5 - 2);
    Slice target(buf2, 20);
    iter->Seek(target);
    ASSERT_TRUE(iter->Valid());
    ASSERT_EQ(iter->key().compare(key), 0);
  }
}

TEST_F(DBTestTailingIterator, TailingIteratorTrimSeekToNext) {
  const uint64_t k150KB = 150 * 1024;
  Options options;
  options.write_buffer_size = k150KB;
  options.max_write_buffer_number = 3;
  options.min_write_buffer_number_to_merge = 2;
  CreateAndReopenWithCF({"pikachu"}, options);
  ReadOptions read_options;
  read_options.tailing = true;
  int num_iters, deleted_iters;

  char bufe[32];
  snprintf(bufe, sizeof(bufe), "00b0%016d", 0);
  Slice keyu(bufe, 20);
  read_options.iterate_upper_bound = &keyu;
  std::unique_ptr<Iterator> iter(db_->NewIterator(read_options, handles_[1]));
  std::unique_ptr<Iterator> itern(db_->NewIterator(read_options, handles_[1]));
  std::unique_ptr<Iterator> iterh(db_->NewIterator(read_options, handles_[1]));
  std::string value(1024, 'a');
  bool file_iters_deleted = false;
  rocksdb::SyncPoint::GetInstance()->SetCallBack(
      "ForwardIterator::SeekInternal:Return", [&](void* arg) {
        ForwardIterator* fiter = reinterpret_cast<ForwardIterator*>(arg);
        ASSERT_TRUE(!file_iters_deleted ||
                    fiter->TEST_CheckDeletedIters(&deleted_iters, &num_iters));
      });
  rocksdb::SyncPoint::GetInstance()->SetCallBack(
      "ForwardIterator::Next:Return", [&](void* arg) {
        ForwardIterator* fiter = reinterpret_cast<ForwardIterator*>(arg);
        ASSERT_TRUE(!file_iters_deleted ||
                    fiter->TEST_CheckDeletedIters(&deleted_iters, &num_iters));
      });
  rocksdb::SyncPoint::GetInstance()->EnableProcessing();
  const int num_records = 1000;
  for (int i = 1; i < num_records; ++i) {
    char buf1[32];
    char buf2[32];
    char buf3[32];
    char buf4[32];
    snprintf(buf1, sizeof(buf1), "00a0%016d", i * 5);
    snprintf(buf3, sizeof(buf1), "00b0%016d", i * 5);

    Slice key(buf1, 20);
    ASSERT_OK(Put(1, key, value));
    Slice keyn(buf3, 20);
    ASSERT_OK(Put(1, keyn, value));

    if (i % 100 == 99) {
      ASSERT_OK(Flush(1));
      dbfull()->TEST_WaitForCompact();
      if (i == 299) {
        file_iters_deleted = true;
      }
      snprintf(buf4, sizeof(buf1), "00a0%016d", i * 5 / 2);
      Slice target(buf4, 20);
      iterh->Seek(target);
      ASSERT_TRUE(iter->Valid());
      for (int j = (i + 1) * 5 / 2; j < i * 5; j += 5) {
        iterh->Next();
        ASSERT_TRUE(iterh->Valid());
      }
      if (i == 299) {
        file_iters_deleted = false;
      }
    }

    file_iters_deleted = true;
    snprintf(buf2, sizeof(buf2), "00a0%016d", i * 5 - 2);
    Slice target(buf2, 20);
    iter->Seek(target);
    ASSERT_TRUE(iter->Valid());
    ASSERT_EQ(iter->key().compare(key), 0);
    ASSERT_LE(num_iters, 1);
    if (i == 1) {
      itern->SeekToFirst();
    } else {
      itern->Next();
    }
    ASSERT_TRUE(itern->Valid());
    ASSERT_EQ(itern->key().compare(key), 0);
    ASSERT_LE(num_iters, 1);
    file_iters_deleted = false;
  }
  iter = 0;
  itern = 0;
  iterh = 0;
  BlockBasedTableOptions table_options;
  table_options.no_block_cache = true;
  table_options.block_cache_compressed = nullptr;
  options.table_factory.reset(NewBlockBasedTableFactory(table_options));
  ReopenWithColumnFamilies({"default", "pikachu"}, options);
  read_options.read_tier = kBlockCacheTier;
  std::unique_ptr<Iterator> iteri(db_->NewIterator(read_options, handles_[1]));
  char buf5[32];
  snprintf(buf5, sizeof(buf5), "00a0%016d", (num_records / 2) * 5 - 2);
  Slice target1(buf5, 20);
  iteri->Seek(target1);
  ASSERT_TRUE(iteri->status().IsIncomplete());
  iteri = 0;

  read_options.read_tier = kReadAllTier;
  options.table_factory.reset(NewBlockBasedTableFactory());
  ReopenWithColumnFamilies({"default", "pikachu"}, options);
  iter.reset(db_->NewIterator(read_options, handles_[1]));
  for (int i = 2 * num_records; i > 0; --i) {
    char buf1[32];
    char buf2[32];
    snprintf(buf1, sizeof(buf1), "00a0%016d", i * 5);

    Slice key(buf1, 20);
    ASSERT_OK(Put(1, key, value));

    if (i % 100 == 99) {
      ASSERT_OK(Flush(1));
    }

    snprintf(buf2, sizeof(buf2), "00a0%016d", i * 5 - 2);
    Slice target(buf2, 20);
    iter->Seek(target);
    ASSERT_TRUE(iter->Valid());
    ASSERT_EQ(iter->key().compare(key), 0);
  }
}

TEST_F(DBTestTailingIterator, TailingIteratorDeletes) {
  CreateAndReopenWithCF({"pikachu"}, CurrentOptions());
  ReadOptions read_options;
  read_options.tailing = true;

  std::unique_ptr<Iterator> iter(db_->NewIterator(read_options, handles_[1]));

  // write a single record, read it using the iterator, then delete it
  ASSERT_OK(Put(1, "0test", "test"));
  iter->SeekToFirst();
  ASSERT_TRUE(iter->Valid());
  ASSERT_EQ(iter->key().ToString(), "0test");
  ASSERT_OK(Delete(1, "0test"));

  // write many more records
  const int num_records = 10000;
  std::string value(1024, 'A');

  for (int i = 0; i < num_records; ++i) {
    char buf[32];
    snprintf(buf, sizeof(buf), "1%015d", i);

    Slice key(buf, 16);
    ASSERT_OK(Put(1, key, value));
  }

  // force a flush to make sure that no records are read from memtable
  ASSERT_OK(Flush(1));

  // skip "0test"
  iter->Next();

  // make sure we can read all new records using the existing iterator
  int count = 0;
  for (; iter->Valid(); iter->Next(), ++count) ;

  ASSERT_EQ(count, num_records);
}

TEST_F(DBTestTailingIterator, TailingIteratorPrefixSeek) {
  XFUNC_TEST("", "dbtest_prefix", prefix_skip1, XFuncPoint::SetSkip,
             kSkipNoPrefix);
  ReadOptions read_options;
  read_options.tailing = true;

  Options options = CurrentOptions();
  options.env = env_;
  options.create_if_missing = true;
  options.disable_auto_compactions = true;
  options.prefix_extractor.reset(NewFixedPrefixTransform(2));
  options.memtable_factory.reset(NewHashSkipListRepFactory(16));
  DestroyAndReopen(options);
  CreateAndReopenWithCF({"pikachu"}, options);

  std::unique_ptr<Iterator> iter(db_->NewIterator(read_options, handles_[1]));
  ASSERT_OK(Put(1, "0101", "test"));

  ASSERT_OK(Flush(1));

  ASSERT_OK(Put(1, "0202", "test"));

  // Seek(0102) shouldn't find any records since 0202 has a different prefix
  iter->Seek("0102");
  ASSERT_TRUE(!iter->Valid());

  iter->Seek("0202");
  ASSERT_TRUE(iter->Valid());
  ASSERT_EQ(iter->key().ToString(), "0202");

  iter->Next();
  ASSERT_TRUE(!iter->Valid());
  XFUNC_TEST("", "dbtest_prefix", prefix_skip1, XFuncPoint::SetSkip, 0);
}

TEST_F(DBTestTailingIterator, TailingIteratorIncomplete) {
  CreateAndReopenWithCF({"pikachu"}, CurrentOptions());
  ReadOptions read_options;
  read_options.tailing = true;
  read_options.read_tier = kBlockCacheTier;

  std::string key("key");
  std::string value("value");

  ASSERT_OK(db_->Put(WriteOptions(), key, value));

  std::unique_ptr<Iterator> iter(db_->NewIterator(read_options));
  iter->SeekToFirst();
  // we either see the entry or it's not in cache
  ASSERT_TRUE(iter->Valid() || iter->status().IsIncomplete());

  ASSERT_OK(db_->CompactRange(CompactRangeOptions(), nullptr, nullptr));
  iter->SeekToFirst();
  // should still be true after compaction
  ASSERT_TRUE(iter->Valid() || iter->status().IsIncomplete());
}

TEST_F(DBTestTailingIterator, TailingIteratorSeekToSame) {
  Options options = CurrentOptions();
  options.compaction_style = kCompactionStyleUniversal;
  options.write_buffer_size = 1000;
  CreateAndReopenWithCF({"pikachu"}, options);

  ReadOptions read_options;
  read_options.tailing = true;

  const int NROWS = 10000;
  // Write rows with keys 00000, 00002, 00004 etc.
  for (int i = 0; i < NROWS; ++i) {
    char buf[100];
    snprintf(buf, sizeof(buf), "%05d", 2*i);
    std::string key(buf);
    std::string value("value");
    ASSERT_OK(db_->Put(WriteOptions(), key, value));
  }

  std::unique_ptr<Iterator> iter(db_->NewIterator(read_options));
  // Seek to 00001.  We expect to find 00002.
  std::string start_key = "00001";
  iter->Seek(start_key);
  ASSERT_TRUE(iter->Valid());

  std::string found = iter->key().ToString();
  ASSERT_EQ("00002", found);

  // Now seek to the same key.  The iterator should remain in the same
  // position.
  iter->Seek(found);
  ASSERT_TRUE(iter->Valid());
  ASSERT_EQ(found, iter->key().ToString());
}

// Sets iterate_upper_bound and verifies that ForwardIterator doesn't call
// Seek() on immutable iterators when target key is >= prev_key and all
// iterators, including the memtable iterator, are over the upper bound.
TEST_F(DBTestTailingIterator, TailingIteratorUpperBound) {
  CreateAndReopenWithCF({"pikachu"}, CurrentOptions());

  const Slice upper_bound("20", 3);
  ReadOptions read_options;
  read_options.tailing = true;
  read_options.iterate_upper_bound = &upper_bound;

  ASSERT_OK(Put(1, "11", "11"));
  ASSERT_OK(Put(1, "12", "12"));
  ASSERT_OK(Put(1, "22", "22"));
  ASSERT_OK(Flush(1));  // flush all those keys to an immutable SST file

  // Add another key to the memtable.
  ASSERT_OK(Put(1, "21", "21"));

  std::unique_ptr<Iterator> it(db_->NewIterator(read_options, handles_[1]));
  it->Seek("12");
  ASSERT_TRUE(it->Valid());
  ASSERT_EQ("12", it->key().ToString());

  it->Next();
  // Not valid since "21" is over the upper bound.
  ASSERT_FALSE(it->Valid());

  // This keeps track of the number of times NeedToSeekImmutable() was true.
  int immutable_seeks = 0;
  rocksdb::SyncPoint::GetInstance()->SetCallBack(
      "ForwardIterator::SeekInternal:Immutable",
      [&](void* arg) { ++immutable_seeks; });

  // Seek to 13. This should not require any immutable seeks.
  rocksdb::SyncPoint::GetInstance()->EnableProcessing();
  it->Seek("13");
  rocksdb::SyncPoint::GetInstance()->DisableProcessing();

  ASSERT_FALSE(it->Valid());
  ASSERT_EQ(0, immutable_seeks);
}

TEST_F(DBTestTailingIterator, ManagedTailingIteratorSingle) {
  ReadOptions read_options;
  read_options.tailing = true;
  read_options.managed = true;

  std::unique_ptr<Iterator> iter(db_->NewIterator(read_options));
  iter->SeekToFirst();
  ASSERT_TRUE(!iter->Valid());

  // add a record and check that iter can see it
  ASSERT_OK(db_->Put(WriteOptions(), "mirko", "fodor"));
  iter->SeekToFirst();
  ASSERT_TRUE(iter->Valid());
  ASSERT_EQ(iter->key().ToString(), "mirko");

  iter->Next();
  ASSERT_TRUE(!iter->Valid());
}

TEST_F(DBTestTailingIterator, ManagedTailingIteratorKeepAdding) {
  CreateAndReopenWithCF({"pikachu"}, CurrentOptions());
  ReadOptions read_options;
  read_options.tailing = true;
  read_options.managed = true;

  std::unique_ptr<Iterator> iter(db_->NewIterator(read_options, handles_[1]));
  std::string value(1024, 'a');

  const int num_records = 10000;
  for (int i = 0; i < num_records; ++i) {
    char buf[32];
    snprintf(buf, sizeof(buf), "%016d", i);

    Slice key(buf, 16);
    ASSERT_OK(Put(1, key, value));

    iter->Seek(key);
    ASSERT_TRUE(iter->Valid());
    ASSERT_EQ(iter->key().compare(key), 0);
  }
}

TEST_F(DBTestTailingIterator, ManagedTailingIteratorSeekToNext) {
  CreateAndReopenWithCF({"pikachu"}, CurrentOptions());
  ReadOptions read_options;
  read_options.tailing = true;
  read_options.managed = true;

  std::unique_ptr<Iterator> iter(db_->NewIterator(read_options, handles_[1]));
  std::string value(1024, 'a');

  const int num_records = 1000;
  for (int i = 1; i < num_records; ++i) {
    char buf1[32];
    char buf2[32];
    snprintf(buf1, sizeof(buf1), "00a0%016d", i * 5);

    Slice key(buf1, 20);
    ASSERT_OK(Put(1, key, value));

    if (i % 100 == 99) {
      ASSERT_OK(Flush(1));
    }

    snprintf(buf2, sizeof(buf2), "00a0%016d", i * 5 - 2);
    Slice target(buf2, 20);
    iter->Seek(target);
    ASSERT_TRUE(iter->Valid());
    ASSERT_EQ(iter->key().compare(key), 0);
  }
  for (int i = 2 * num_records; i > 0; --i) {
    char buf1[32];
    char buf2[32];
    snprintf(buf1, sizeof(buf1), "00a0%016d", i * 5);

    Slice key(buf1, 20);
    ASSERT_OK(Put(1, key, value));

    if (i % 100 == 99) {
      ASSERT_OK(Flush(1));
    }

    snprintf(buf2, sizeof(buf2), "00a0%016d", i * 5 - 2);
    Slice target(buf2, 20);
    iter->Seek(target);
    ASSERT_TRUE(iter->Valid());
    ASSERT_EQ(iter->key().compare(key), 0);
  }
}

TEST_F(DBTestTailingIterator, ManagedTailingIteratorDeletes) {
  CreateAndReopenWithCF({"pikachu"}, CurrentOptions());
  ReadOptions read_options;
  read_options.tailing = true;
  read_options.managed = true;

  std::unique_ptr<Iterator> iter(db_->NewIterator(read_options, handles_[1]));

  // write a single record, read it using the iterator, then delete it
  ASSERT_OK(Put(1, "0test", "test"));
  iter->SeekToFirst();
  ASSERT_TRUE(iter->Valid());
  ASSERT_EQ(iter->key().ToString(), "0test");
  ASSERT_OK(Delete(1, "0test"));

  // write many more records
  const int num_records = 10000;
  std::string value(1024, 'A');

  for (int i = 0; i < num_records; ++i) {
    char buf[32];
    snprintf(buf, sizeof(buf), "1%015d", i);

    Slice key(buf, 16);
    ASSERT_OK(Put(1, key, value));
  }

  // force a flush to make sure that no records are read from memtable
  ASSERT_OK(Flush(1));

  // skip "0test"
  iter->Next();

  // make sure we can read all new records using the existing iterator
  int count = 0;
  for (; iter->Valid(); iter->Next(), ++count) {
  }

  ASSERT_EQ(count, num_records);
}

TEST_F(DBTestTailingIterator, ManagedTailingIteratorPrefixSeek) {
  XFUNC_TEST("", "dbtest_prefix", prefix_skip1, XFuncPoint::SetSkip,
             kSkipNoPrefix);
  ReadOptions read_options;
  read_options.tailing = true;
  read_options.managed = true;

  Options options = CurrentOptions();
  options.env = env_;
  options.create_if_missing = true;
  options.disable_auto_compactions = true;
  options.prefix_extractor.reset(NewFixedPrefixTransform(2));
  options.memtable_factory.reset(NewHashSkipListRepFactory(16));
  DestroyAndReopen(options);
  CreateAndReopenWithCF({"pikachu"}, options);

  std::unique_ptr<Iterator> iter(db_->NewIterator(read_options, handles_[1]));
  ASSERT_OK(Put(1, "0101", "test"));

  ASSERT_OK(Flush(1));

  ASSERT_OK(Put(1, "0202", "test"));

  // Seek(0102) shouldn't find any records since 0202 has a different prefix
  iter->Seek("0102");
  ASSERT_TRUE(!iter->Valid());

  iter->Seek("0202");
  ASSERT_TRUE(iter->Valid());
  ASSERT_EQ(iter->key().ToString(), "0202");

  iter->Next();
  ASSERT_TRUE(!iter->Valid());
  XFUNC_TEST("", "dbtest_prefix", prefix_skip1, XFuncPoint::SetSkip, 0);
}

TEST_F(DBTestTailingIterator, ManagedTailingIteratorIncomplete) {
  CreateAndReopenWithCF({"pikachu"}, CurrentOptions());
  ReadOptions read_options;
  read_options.tailing = true;
  read_options.managed = true;
  read_options.read_tier = kBlockCacheTier;

  std::string key = "key";
  std::string value = "value";

  ASSERT_OK(db_->Put(WriteOptions(), key, value));

  std::unique_ptr<Iterator> iter(db_->NewIterator(read_options));
  iter->SeekToFirst();
  // we either see the entry or it's not in cache
  ASSERT_TRUE(iter->Valid() || iter->status().IsIncomplete());

  ASSERT_OK(db_->CompactRange(CompactRangeOptions(), nullptr, nullptr));
  iter->SeekToFirst();
  // should still be true after compaction
  ASSERT_TRUE(iter->Valid() || iter->status().IsIncomplete());
}

TEST_F(DBTestTailingIterator, ManagedTailingIteratorSeekToSame) {
  Options options = CurrentOptions();
  options.compaction_style = kCompactionStyleUniversal;
  options.write_buffer_size = 1000;
  CreateAndReopenWithCF({"pikachu"}, options);

  ReadOptions read_options;
  read_options.tailing = true;
  read_options.managed = true;

  const int NROWS = 10000;
  // Write rows with keys 00000, 00002, 00004 etc.
  for (int i = 0; i < NROWS; ++i) {
    char buf[100];
    snprintf(buf, sizeof(buf), "%05d", 2 * i);
    std::string key(buf);
    std::string value("value");
    ASSERT_OK(db_->Put(WriteOptions(), key, value));
  }

  std::unique_ptr<Iterator> iter(db_->NewIterator(read_options));
  // Seek to 00001.  We expect to find 00002.
  std::string start_key = "00001";
  iter->Seek(start_key);
  ASSERT_TRUE(iter->Valid());

  std::string found = iter->key().ToString();
  ASSERT_EQ("00002", found);

  // Now seek to the same key.  The iterator should remain in the same
  // position.
  iter->Seek(found);
  ASSERT_TRUE(iter->Valid());
  ASSERT_EQ(found, iter->key().ToString());
}

}  // namespace rocksdb

#endif  // !(defined NDEBUG) || !defined(OS_WIN)

int main(int argc, char** argv) {
#if !(defined NDEBUG) || !defined(OS_WIN)
  rocksdb::port::InstallStackTraceHandler();
  ::testing::InitGoogleTest(&argc, argv);
  return RUN_ALL_TESTS();
#else
  return 0;
#endif
}
