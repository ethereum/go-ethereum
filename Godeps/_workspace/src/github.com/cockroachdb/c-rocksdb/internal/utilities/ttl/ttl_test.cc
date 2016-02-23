// Copyright (c) 2011 The LevelDB Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file. See the AUTHORS file for names of contributors.

#ifndef ROCKSDB_LITE

#include <memory>
#include "rocksdb/compaction_filter.h"
#include "rocksdb/utilities/db_ttl.h"
#include "util/testharness.h"
#include "util/logging.h"
#include <map>
#ifndef OS_WIN
#include <unistd.h>
#endif

namespace rocksdb {

namespace {

typedef std::map<std::string, std::string> KVMap;

enum BatchOperation { OP_PUT = 0, OP_DELETE = 1 };
}

class SpecialTimeEnv : public EnvWrapper {
 public:
  explicit SpecialTimeEnv(Env* base) : EnvWrapper(base) {
    base->GetCurrentTime(&current_time_);
  }

  void Sleep(int64_t sleep_time) { current_time_ += sleep_time; }
  virtual Status GetCurrentTime(int64_t* current_time) override {
    *current_time = current_time_;
    return Status::OK();
  }

 private:
  int64_t current_time_;
};

class TtlTest : public testing::Test {
 public:
  TtlTest() {
    env_.reset(new SpecialTimeEnv(Env::Default()));
    dbname_ = test::TmpDir() + "/db_ttl";
    options_.create_if_missing = true;
    options_.env = env_.get();
    // ensure that compaction is kicked in to always strip timestamp from kvs
    options_.max_grandparent_overlap_factor = 0;
    // compaction should take place always from level0 for determinism
    db_ttl_ = nullptr;
    DestroyDB(dbname_, Options());
  }

  ~TtlTest() {
    CloseTtl();
    DestroyDB(dbname_, Options());
  }

  // Open database with TTL support when TTL not provided with db_ttl_ pointer
  void OpenTtl() {
    ASSERT_TRUE(db_ttl_ ==
                nullptr);  //  db should be closed before opening again
    ASSERT_OK(DBWithTTL::Open(options_, dbname_, &db_ttl_));
  }

  // Open database with TTL support when TTL provided with db_ttl_ pointer
  void OpenTtl(int32_t ttl) {
    ASSERT_TRUE(db_ttl_ == nullptr);
    ASSERT_OK(DBWithTTL::Open(options_, dbname_, &db_ttl_, ttl));
  }

  // Open with TestFilter compaction filter
  void OpenTtlWithTestCompaction(int32_t ttl) {
    options_.compaction_filter_factory =
      std::shared_ptr<CompactionFilterFactory>(
          new TestFilterFactory(kSampleSize_, kNewValue_));
    OpenTtl(ttl);
  }

  // Open database with TTL support in read_only mode
  void OpenReadOnlyTtl(int32_t ttl) {
    ASSERT_TRUE(db_ttl_ == nullptr);
    ASSERT_OK(DBWithTTL::Open(options_, dbname_, &db_ttl_, ttl, true));
  }

  void CloseTtl() {
    delete db_ttl_;
    db_ttl_ = nullptr;
  }

  // Populates and returns a kv-map
  void MakeKVMap(int64_t num_entries) {
    kvmap_.clear();
    int digits = 1;
    for (int64_t dummy = num_entries; dummy /= 10; ++digits) {
    }
    int digits_in_i = 1;
    for (int64_t i = 0; i < num_entries; i++) {
      std::string key = "key";
      std::string value = "value";
      if (i % 10 == 0) {
        digits_in_i++;
      }
      for(int j = digits_in_i; j < digits; j++) {
        key.append("0");
        value.append("0");
      }
      AppendNumberTo(&key, i);
      AppendNumberTo(&value, i);
      kvmap_[key] = value;
    }
    ASSERT_EQ(static_cast<int64_t>(kvmap_.size()),
              num_entries);  // check all insertions done
  }

  // Makes a write-batch with key-vals from kvmap_ and 'Write''s it
  void MakePutWriteBatch(const BatchOperation* batch_ops, int64_t num_ops) {
    ASSERT_LE(num_ops, static_cast<int64_t>(kvmap_.size()));
    static WriteOptions wopts;
    static FlushOptions flush_opts;
    WriteBatch batch;
    kv_it_ = kvmap_.begin();
    for (int64_t i = 0; i < num_ops && kv_it_ != kvmap_.end(); i++, ++kv_it_) {
      switch (batch_ops[i]) {
        case OP_PUT:
          batch.Put(kv_it_->first, kv_it_->second);
          break;
        case OP_DELETE:
          batch.Delete(kv_it_->first);
          break;
        default:
          ASSERT_TRUE(false);
      }
    }
    db_ttl_->Write(wopts, &batch);
    db_ttl_->Flush(flush_opts);
  }

  // Puts num_entries starting from start_pos_map from kvmap_ into the database
  void PutValues(int64_t start_pos_map, int64_t num_entries, bool flush = true,
                 ColumnFamilyHandle* cf = nullptr) {
    ASSERT_TRUE(db_ttl_);
    ASSERT_LE(start_pos_map + num_entries, static_cast<int64_t>(kvmap_.size()));
    static WriteOptions wopts;
    static FlushOptions flush_opts;
    kv_it_ = kvmap_.begin();
    advance(kv_it_, start_pos_map);
    for (int64_t i = 0; kv_it_ != kvmap_.end() && i < num_entries;
         i++, ++kv_it_) {
      ASSERT_OK(cf == nullptr
                    ? db_ttl_->Put(wopts, kv_it_->first, kv_it_->second)
                    : db_ttl_->Put(wopts, cf, kv_it_->first, kv_it_->second));
    }
    // Put a mock kv at the end because CompactionFilter doesn't delete last key
    ASSERT_OK(cf == nullptr ? db_ttl_->Put(wopts, "keymock", "valuemock")
                            : db_ttl_->Put(wopts, cf, "keymock", "valuemock"));
    if (flush) {
      if (cf == nullptr) {
        db_ttl_->Flush(flush_opts);
      } else {
        db_ttl_->Flush(flush_opts, cf);
      }
    }
  }

  // Runs a manual compaction
  void ManualCompact(ColumnFamilyHandle* cf = nullptr) {
    if (cf == nullptr) {
      db_ttl_->CompactRange(CompactRangeOptions(), nullptr, nullptr);
    } else {
      db_ttl_->CompactRange(CompactRangeOptions(), cf, nullptr, nullptr);
    }
  }

  // checks the whole kvmap_ to return correct values using KeyMayExist
  void SimpleKeyMayExistCheck() {
    static ReadOptions ropts;
    bool value_found;
    std::string val;
    for(auto &kv : kvmap_) {
      bool ret = db_ttl_->KeyMayExist(ropts, kv.first, &val, &value_found);
      if (ret == false || value_found == false) {
        fprintf(stderr, "KeyMayExist could not find key=%s in the database but"
                        " should have\n", kv.first.c_str());
        ASSERT_TRUE(false);
      } else if (val.compare(kv.second) != 0) {
        fprintf(stderr, " value for key=%s present in database is %s but"
                        " should be %s\n", kv.first.c_str(), val.c_str(),
                        kv.second.c_str());
        ASSERT_TRUE(false);
      }
    }
  }

  // checks the whole kvmap_ to return correct values using MultiGet
  void SimpleMultiGetTest() {
    static ReadOptions ropts;
    std::vector<Slice> keys;
    std::vector<std::string> values;

    for (auto& kv : kvmap_) {
      keys.emplace_back(kv.first);
    }

    auto statuses = db_ttl_->MultiGet(ropts, keys, &values);
    size_t i = 0;
    for (auto& kv : kvmap_) {
      ASSERT_OK(statuses[i]);
      ASSERT_EQ(values[i], kv.second);
      ++i;
    }
  }

  // Sleeps for slp_tim then runs a manual compaction
  // Checks span starting from st_pos from kvmap_ in the db and
  // Gets should return true if check is true and false otherwise
  // Also checks that value that we got is the same as inserted; and =kNewValue
  //   if test_compaction_change is true
  void SleepCompactCheck(int slp_tim, int64_t st_pos, int64_t span,
                         bool check = true, bool test_compaction_change = false,
                         ColumnFamilyHandle* cf = nullptr) {
    ASSERT_TRUE(db_ttl_);

    env_->Sleep(slp_tim);
    ManualCompact(cf);
    static ReadOptions ropts;
    kv_it_ = kvmap_.begin();
    advance(kv_it_, st_pos);
    std::string v;
    for (int64_t i = 0; kv_it_ != kvmap_.end() && i < span; i++, ++kv_it_) {
      Status s = (cf == nullptr) ? db_ttl_->Get(ropts, kv_it_->first, &v)
                                 : db_ttl_->Get(ropts, cf, kv_it_->first, &v);
      if (s.ok() != check) {
        fprintf(stderr, "key=%s ", kv_it_->first.c_str());
        if (!s.ok()) {
          fprintf(stderr, "is absent from db but was expected to be present\n");
        } else {
          fprintf(stderr, "is present in db but was expected to be absent\n");
        }
        ASSERT_TRUE(false);
      } else if (s.ok()) {
          if (test_compaction_change && v.compare(kNewValue_) != 0) {
            fprintf(stderr, " value for key=%s present in database is %s but "
                            " should be %s\n", kv_it_->first.c_str(), v.c_str(),
                            kNewValue_.c_str());
            ASSERT_TRUE(false);
          } else if (!test_compaction_change && v.compare(kv_it_->second) !=0) {
            fprintf(stderr, " value for key=%s present in database is %s but "
                            " should be %s\n", kv_it_->first.c_str(), v.c_str(),
                            kv_it_->second.c_str());
            ASSERT_TRUE(false);
          }
      }
    }
  }

  // Similar as SleepCompactCheck but uses TtlIterator to read from db
  void SleepCompactCheckIter(int slp, int st_pos, int64_t span,
                             bool check = true) {
    ASSERT_TRUE(db_ttl_);
    env_->Sleep(slp);
    ManualCompact();
    static ReadOptions ropts;
    Iterator *dbiter = db_ttl_->NewIterator(ropts);
    kv_it_ = kvmap_.begin();
    advance(kv_it_, st_pos);

    dbiter->Seek(kv_it_->first);
    if (!check) {
      if (dbiter->Valid()) {
        ASSERT_NE(dbiter->value().compare(kv_it_->second), 0);
      }
    } else {  // dbiter should have found out kvmap_[st_pos]
      for (int64_t i = st_pos; kv_it_ != kvmap_.end() && i < st_pos + span;
           i++, ++kv_it_) {
        ASSERT_TRUE(dbiter->Valid());
        ASSERT_EQ(dbiter->value().compare(kv_it_->second), 0);
        dbiter->Next();
      }
    }
    delete dbiter;
  }

  class TestFilter : public CompactionFilter {
   public:
    TestFilter(const int64_t kSampleSize, const std::string& kNewValue)
      : kSampleSize_(kSampleSize),
        kNewValue_(kNewValue) {
    }

    // Works on keys of the form "key<number>"
    // Drops key if number at the end of key is in [0, kSampleSize_/3),
    // Keeps key if it is in [kSampleSize_/3, 2*kSampleSize_/3),
    // Change value if it is in [2*kSampleSize_/3, kSampleSize_)
    // Eg. kSampleSize_=6. Drop:key0-1...Keep:key2-3...Change:key4-5...
    virtual bool Filter(int level, const Slice& key,
                        const Slice& value, std::string* new_value,
                        bool* value_changed) const override {
      assert(new_value != nullptr);

      std::string search_str = "0123456789";
      std::string key_string = key.ToString();
      size_t pos = key_string.find_first_of(search_str);
      int num_key_end;
      if (pos != std::string::npos) {
        auto key_substr = key_string.substr(pos, key.size() - pos);
#ifndef CYGWIN
        num_key_end = std::stoi(key_substr);
#else
        num_key_end = std::strtol(key_substr.c_str(), 0, 10);
#endif

      } else {
        return false; // Keep keys not matching the format "key<NUMBER>"
      }

      int64_t partition = kSampleSize_ / 3;
      if (num_key_end < partition) {
        return true;
      } else if (num_key_end < partition * 2) {
        return false;
      } else {
        *new_value = kNewValue_;
        *value_changed = true;
        return false;
      }
    }

    virtual const char* Name() const override {
      return "TestFilter";
    }

   private:
    const int64_t kSampleSize_;
    const std::string kNewValue_;
  };

  class TestFilterFactory : public CompactionFilterFactory {
    public:
      TestFilterFactory(const int64_t kSampleSize, const std::string& kNewValue)
        : kSampleSize_(kSampleSize),
          kNewValue_(kNewValue) {
      }

      virtual std::unique_ptr<CompactionFilter> CreateCompactionFilter(
          const CompactionFilter::Context& context) override {
        return std::unique_ptr<CompactionFilter>(
            new TestFilter(kSampleSize_, kNewValue_));
      }

      virtual const char* Name() const override {
        return "TestFilterFactory";
      }

    private:
      const int64_t kSampleSize_;
      const std::string kNewValue_;
  };


  // Choose carefully so that Put, Gets & Compaction complete in 1 second buffer
  static const int64_t kSampleSize_ = 100;
  std::string dbname_;
  DBWithTTL* db_ttl_;
  unique_ptr<SpecialTimeEnv> env_;

 private:
  Options options_;
  KVMap kvmap_;
  KVMap::iterator kv_it_;
  const std::string kNewValue_ = "new_value";
  unique_ptr<CompactionFilter> test_comp_filter_;
}; // class TtlTest

// If TTL is non positive or not provided, the behaviour is TTL = infinity
// This test opens the db 3 times with such default behavior and inserts a
// bunch of kvs each time. All kvs should accumulate in the db till the end
// Partitions the sample-size provided into 3 sets over boundary1 and boundary2
TEST_F(TtlTest, NoEffect) {
  MakeKVMap(kSampleSize_);
  int64_t boundary1 = kSampleSize_ / 3;
  int64_t boundary2 = 2 * boundary1;

  OpenTtl();
  PutValues(0, boundary1);                       //T=0: Set1 never deleted
  SleepCompactCheck(1, 0, boundary1);            //T=1: Set1 still there
  CloseTtl();

  OpenTtl(0);
  PutValues(boundary1, boundary2 - boundary1);   //T=1: Set2 never deleted
  SleepCompactCheck(1, 0, boundary2);            //T=2: Sets1 & 2 still there
  CloseTtl();

  OpenTtl(-1);
  PutValues(boundary2, kSampleSize_ - boundary2); //T=3: Set3 never deleted
  SleepCompactCheck(1, 0, kSampleSize_, true);    //T=4: Sets 1,2,3 still there
  CloseTtl();
}

// Puts a set of values and checks its presence using Get during ttl
TEST_F(TtlTest, PresentDuringTTL) {
  MakeKVMap(kSampleSize_);

  OpenTtl(2);                                 // T=0:Open the db with ttl = 2
  PutValues(0, kSampleSize_);                  // T=0:Insert Set1. Delete at t=2
  SleepCompactCheck(1, 0, kSampleSize_, true); // T=1:Set1 should still be there
  CloseTtl();
}

// Puts a set of values and checks its absence using Get after ttl
TEST_F(TtlTest, AbsentAfterTTL) {
  MakeKVMap(kSampleSize_);

  OpenTtl(1);                                  // T=0:Open the db with ttl = 2
  PutValues(0, kSampleSize_);                  // T=0:Insert Set1. Delete at t=2
  SleepCompactCheck(2, 0, kSampleSize_, false); // T=2:Set1 should not be there
  CloseTtl();
}

// Resets the timestamp of a set of kvs by updating them and checks that they
// are not deleted according to the old timestamp
TEST_F(TtlTest, ResetTimestamp) {
  MakeKVMap(kSampleSize_);

  OpenTtl(3);
  PutValues(0, kSampleSize_);            // T=0: Insert Set1. Delete at t=3
  env_->Sleep(2);                        // T=2
  PutValues(0, kSampleSize_);            // T=2: Insert Set1. Delete at t=5
  SleepCompactCheck(2, 0, kSampleSize_); // T=4: Set1 should still be there
  CloseTtl();
}

// Similar to PresentDuringTTL but uses Iterator
TEST_F(TtlTest, IterPresentDuringTTL) {
  MakeKVMap(kSampleSize_);

  OpenTtl(2);
  PutValues(0, kSampleSize_);                 // T=0: Insert. Delete at t=2
  SleepCompactCheckIter(1, 0, kSampleSize_);  // T=1: Set should be there
  CloseTtl();
}

// Similar to AbsentAfterTTL but uses Iterator
TEST_F(TtlTest, IterAbsentAfterTTL) {
  MakeKVMap(kSampleSize_);

  OpenTtl(1);
  PutValues(0, kSampleSize_);                      // T=0: Insert. Delete at t=1
  SleepCompactCheckIter(2, 0, kSampleSize_, false); // T=2: Should not be there
  CloseTtl();
}

// Checks presence while opening the same db more than once with the same ttl
// Note: The second open will open the same db
TEST_F(TtlTest, MultiOpenSamePresent) {
  MakeKVMap(kSampleSize_);

  OpenTtl(2);
  PutValues(0, kSampleSize_);                   // T=0: Insert. Delete at t=2
  CloseTtl();

  OpenTtl(2);                                  // T=0. Delete at t=2
  SleepCompactCheck(1, 0, kSampleSize_);        // T=1: Set should be there
  CloseTtl();
}

// Checks absence while opening the same db more than once with the same ttl
// Note: The second open will open the same db
TEST_F(TtlTest, MultiOpenSameAbsent) {
  MakeKVMap(kSampleSize_);

  OpenTtl(1);
  PutValues(0, kSampleSize_);                   // T=0: Insert. Delete at t=1
  CloseTtl();

  OpenTtl(1);                                  // T=0.Delete at t=1
  SleepCompactCheck(2, 0, kSampleSize_, false); // T=2: Set should not be there
  CloseTtl();
}

// Checks presence while opening the same db more than once with bigger ttl
TEST_F(TtlTest, MultiOpenDifferent) {
  MakeKVMap(kSampleSize_);

  OpenTtl(1);
  PutValues(0, kSampleSize_);            // T=0: Insert. Delete at t=1
  CloseTtl();

  OpenTtl(3);                           // T=0: Set deleted at t=3
  SleepCompactCheck(2, 0, kSampleSize_); // T=2: Set should be there
  CloseTtl();
}

// Checks presence during ttl in read_only mode
TEST_F(TtlTest, ReadOnlyPresentForever) {
  MakeKVMap(kSampleSize_);

  OpenTtl(1);                                 // T=0:Open the db normally
  PutValues(0, kSampleSize_);                  // T=0:Insert Set1. Delete at t=1
  CloseTtl();

  OpenReadOnlyTtl(1);
  SleepCompactCheck(2, 0, kSampleSize_);       // T=2:Set1 should still be there
  CloseTtl();
}

// Checks whether WriteBatch works well with TTL
// Puts all kvs in kvmap_ in a batch and writes first, then deletes first half
TEST_F(TtlTest, WriteBatchTest) {
  MakeKVMap(kSampleSize_);
  BatchOperation batch_ops[kSampleSize_];
  for (int i = 0; i < kSampleSize_; i++) {
    batch_ops[i] = OP_PUT;
  }

  OpenTtl(2);
  MakePutWriteBatch(batch_ops, kSampleSize_);
  for (int i = 0; i < kSampleSize_ / 2; i++) {
    batch_ops[i] = OP_DELETE;
  }
  MakePutWriteBatch(batch_ops, kSampleSize_ / 2);
  SleepCompactCheck(0, 0, kSampleSize_ / 2, false);
  SleepCompactCheck(0, kSampleSize_ / 2, kSampleSize_ - kSampleSize_ / 2);
  CloseTtl();
}

// Checks user's compaction filter for correctness with TTL logic
TEST_F(TtlTest, CompactionFilter) {
  MakeKVMap(kSampleSize_);

  OpenTtlWithTestCompaction(1);
  PutValues(0, kSampleSize_);                  // T=0:Insert Set1. Delete at t=1
  // T=2: TTL logic takes precedence over TestFilter:-Set1 should not be there
  SleepCompactCheck(2, 0, kSampleSize_, false);
  CloseTtl();

  OpenTtlWithTestCompaction(3);
  PutValues(0, kSampleSize_);                   // T=0:Insert Set1.
  int64_t partition = kSampleSize_ / 3;
  SleepCompactCheck(1, 0, partition, false);                  // Part dropped
  SleepCompactCheck(0, partition, partition);                 // Part kept
  SleepCompactCheck(0, 2 * partition, partition, true, true); // Part changed
  CloseTtl();
}

// Insert some key-values which KeyMayExist should be able to get and check that
// values returned are fine
TEST_F(TtlTest, KeyMayExist) {
  MakeKVMap(kSampleSize_);

  OpenTtl();
  PutValues(0, kSampleSize_, false);

  SimpleKeyMayExistCheck();

  CloseTtl();
}

TEST_F(TtlTest, MultiGetTest) {
  MakeKVMap(kSampleSize_);

  OpenTtl();
  PutValues(0, kSampleSize_, false);

  SimpleMultiGetTest();

  CloseTtl();
}

TEST_F(TtlTest, ColumnFamiliesTest) {
  DB* db;
  Options options;
  options.create_if_missing = true;
  options.env = env_.get();

  DB::Open(options, dbname_, &db);
  ColumnFamilyHandle* handle;
  ASSERT_OK(db->CreateColumnFamily(ColumnFamilyOptions(options),
                                   "ttl_column_family", &handle));

  delete handle;
  delete db;

  std::vector<ColumnFamilyDescriptor> column_families;
  column_families.push_back(ColumnFamilyDescriptor(
      kDefaultColumnFamilyName, ColumnFamilyOptions(options)));
  column_families.push_back(ColumnFamilyDescriptor(
      "ttl_column_family", ColumnFamilyOptions(options)));

  std::vector<ColumnFamilyHandle*> handles;

  ASSERT_OK(DBWithTTL::Open(DBOptions(options), dbname_, column_families,
                            &handles, &db_ttl_, {3, 5}, false));
  ASSERT_EQ(handles.size(), 2U);
  ColumnFamilyHandle* new_handle;
  ASSERT_OK(db_ttl_->CreateColumnFamilyWithTtl(options, "ttl_column_family_2",
                                               &new_handle, 2));
  handles.push_back(new_handle);

  MakeKVMap(kSampleSize_);
  PutValues(0, kSampleSize_, false, handles[0]);
  PutValues(0, kSampleSize_, false, handles[1]);
  PutValues(0, kSampleSize_, false, handles[2]);

  // everything should be there after 1 second
  SleepCompactCheck(1, 0, kSampleSize_, true, false, handles[0]);
  SleepCompactCheck(0, 0, kSampleSize_, true, false, handles[1]);
  SleepCompactCheck(0, 0, kSampleSize_, true, false, handles[2]);

  // only column family 1 should be alive after 4 seconds
  SleepCompactCheck(3, 0, kSampleSize_, false, false, handles[0]);
  SleepCompactCheck(0, 0, kSampleSize_, true, false, handles[1]);
  SleepCompactCheck(0, 0, kSampleSize_, false, false, handles[2]);

  // nothing should be there after 6 seconds
  SleepCompactCheck(2, 0, kSampleSize_, false, false, handles[0]);
  SleepCompactCheck(0, 0, kSampleSize_, false, false, handles[1]);
  SleepCompactCheck(0, 0, kSampleSize_, false, false, handles[2]);

  for (auto h : handles) {
    delete h;
  }
  delete db_ttl_;
  db_ttl_ = nullptr;
}

} //  namespace rocksdb

// A black-box test for the ttl wrapper around rocksdb
int main(int argc, char** argv) {
  ::testing::InitGoogleTest(&argc, argv);
  return RUN_ALL_TESTS();
}

#else
#include <stdio.h>

int main(int argc, char** argv) {
  fprintf(stderr, "SKIPPED as DBWithTTL is not supported in ROCKSDB_LITE\n");
  return 0;
}

#endif  // !ROCKSDB_LITE
