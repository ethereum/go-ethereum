/**
 * An persistent map : key -> (list of strings), using rocksdb merge.
 * This file is a test-harness / use-case for the StringAppendOperator.
 *
 * @author Deon Nicholas (dnicholas@fb.com)
 * Copyright 2013 Facebook, Inc.
*/

#include <iostream>
#include <map>

#include "rocksdb/db.h"
#include "rocksdb/merge_operator.h"
#include "rocksdb/utilities/db_ttl.h"
#include "utilities/merge_operators.h"
#include "utilities/merge_operators/string_append/stringappend.h"
#include "utilities/merge_operators/string_append/stringappend2.h"
#include "util/testharness.h"
#include "util/random.h"

using namespace rocksdb;

namespace rocksdb {

// Path to the database on file system
const std::string kDbName = test::TmpDir() + "/stringappend_test";

namespace {
// OpenDb opens a (possibly new) rocksdb database with a StringAppendOperator
std::shared_ptr<DB> OpenNormalDb(char delim_char) {
  DB* db;
  Options options;
  options.create_if_missing = true;
  options.merge_operator.reset(new StringAppendOperator(delim_char));
  EXPECT_OK(DB::Open(options, kDbName, &db));
  return std::shared_ptr<DB>(db);
}

#ifndef ROCKSDB_LITE  // TtlDb is not supported in Lite
// Open a TtlDB with a non-associative StringAppendTESTOperator
std::shared_ptr<DB> OpenTtlDb(char delim_char) {
  DBWithTTL* db;
  Options options;
  options.create_if_missing = true;
  options.merge_operator.reset(new StringAppendTESTOperator(delim_char));
  EXPECT_OK(DBWithTTL::Open(options, kDbName, &db, 123456));
  return std::shared_ptr<DB>(db);
}
#endif  // !ROCKSDB_LITE
}  // namespace

/// StringLists represents a set of string-lists, each with a key-index.
/// Supports Append(list, string) and Get(list)
class StringLists {
 public:

  //Constructor: specifies the rocksdb db
  /* implicit */
  StringLists(std::shared_ptr<DB> db)
      : db_(db),
        merge_option_(),
        get_option_() {
    assert(db);
  }

  // Append string val onto the list defined by key; return true on success
  bool Append(const std::string& key, const std::string& val){
    Slice valSlice(val.data(), val.size());
    auto s = db_->Merge(merge_option_, key, valSlice);

    if (s.ok()) {
      return true;
    } else {
      std::cerr << "ERROR " << s.ToString() << std::endl;
      return false;
    }
  }

  // Returns the list of strings associated with key (or "" if does not exist)
  bool Get(const std::string& key, std::string* const result){
    assert(result != nullptr); // we should have a place to store the result
    auto s = db_->Get(get_option_, key, result);

    if (s.ok()) {
      return true;
    }

    // Either key does not exist, or there is some error.
    *result = "";       // Always return empty string (just for convention)

    //NotFound is okay; just return empty (similar to std::map)
    //But network or db errors, etc, should fail the test (or at least yell)
    if (!s.IsNotFound()) {
      std::cerr << "ERROR " << s.ToString() << std::endl;
    }

    // Always return false if s.ok() was not true
    return false;
  }


 private:
  std::shared_ptr<DB> db_;
  WriteOptions merge_option_;
  ReadOptions get_option_;

};


// The class for unit-testing
class StringAppendOperatorTest : public testing::Test {
 public:
  StringAppendOperatorTest() {
    DestroyDB(kDbName, Options());    // Start each test with a fresh DB
  }

  typedef std::shared_ptr<DB> (* OpenFuncPtr)(char);

  // Allows user to open databases with different configurations.
  // e.g.: Can open a DB or a TtlDB, etc.
  static void SetOpenDbFunction(OpenFuncPtr func) {
    OpenDb = func;
  }

 protected:
  static OpenFuncPtr OpenDb;
};
StringAppendOperatorTest::OpenFuncPtr StringAppendOperatorTest::OpenDb = nullptr;

// THE TEST CASES BEGIN HERE

TEST_F(StringAppendOperatorTest, IteratorTest) {
  auto db_ = OpenDb(',');
  StringLists slists(db_);

  slists.Append("k1", "v1");
  slists.Append("k1", "v2");
  slists.Append("k1", "v3");

  slists.Append("k2", "a1");
  slists.Append("k2", "a2");
  slists.Append("k2", "a3");

  std::string res;
  std::unique_ptr<rocksdb::Iterator> it(db_->NewIterator(ReadOptions()));
  std::string k1("k1");
  std::string k2("k2");
  bool first = true;
  for (it->Seek(k1); it->Valid(); it->Next()) {
    res = it->value().ToString();
    if (first) {
      ASSERT_EQ(res, "v1,v2,v3");
      first = false;
    } else {
      ASSERT_EQ(res, "a1,a2,a3");
    }
  }
  slists.Append("k2", "a4");
  slists.Append("k1", "v4");

  // Snapshot should still be the same. Should ignore a4 and v4.
  first = true;
  for (it->Seek(k1); it->Valid(); it->Next()) {
    res = it->value().ToString();
    if (first) {
      ASSERT_EQ(res, "v1,v2,v3");
      first = false;
    } else {
      ASSERT_EQ(res, "a1,a2,a3");
    }
  }


  // Should release the snapshot and be aware of the new stuff now
  it.reset(db_->NewIterator(ReadOptions()));
  first = true;
  for (it->Seek(k1); it->Valid(); it->Next()) {
    res = it->value().ToString();
    if (first) {
      ASSERT_EQ(res, "v1,v2,v3,v4");
      first = false;
    } else {
      ASSERT_EQ(res, "a1,a2,a3,a4");
    }
  }

  // start from k2 this time.
  for (it->Seek(k2); it->Valid(); it->Next()) {
    res = it->value().ToString();
    if (first) {
      ASSERT_EQ(res, "v1,v2,v3,v4");
      first = false;
    } else {
      ASSERT_EQ(res, "a1,a2,a3,a4");
    }
  }

  slists.Append("k3", "g1");

  it.reset(db_->NewIterator(ReadOptions()));
  first = true;
  std::string k3("k3");
  for(it->Seek(k2); it->Valid(); it->Next()) {
    res = it->value().ToString();
    if (first) {
      ASSERT_EQ(res, "a1,a2,a3,a4");
      first = false;
    } else {
      ASSERT_EQ(res, "g1");
    }
  }
  for(it->Seek(k3); it->Valid(); it->Next()) {
    res = it->value().ToString();
    if (first) {
      // should not be hit
      ASSERT_EQ(res, "a1,a2,a3,a4");
      first = false;
    } else {
      ASSERT_EQ(res, "g1");
    }
  }

}

TEST_F(StringAppendOperatorTest, SimpleTest) {
  auto db = OpenDb(',');
  StringLists slists(db);

  slists.Append("k1", "v1");
  slists.Append("k1", "v2");
  slists.Append("k1", "v3");

  std::string res;
  bool status = slists.Get("k1", &res);

  ASSERT_TRUE(status);
  ASSERT_EQ(res, "v1,v2,v3");
}

TEST_F(StringAppendOperatorTest, SimpleDelimiterTest) {
  auto db = OpenDb('|');
  StringLists slists(db);

  slists.Append("k1", "v1");
  slists.Append("k1", "v2");
  slists.Append("k1", "v3");

  std::string res;
  slists.Get("k1", &res);
  ASSERT_EQ(res, "v1|v2|v3");
}

TEST_F(StringAppendOperatorTest, OneValueNoDelimiterTest) {
  auto db = OpenDb('!');
  StringLists slists(db);

  slists.Append("random_key", "single_val");

  std::string res;
  slists.Get("random_key", &res);
  ASSERT_EQ(res, "single_val");
}

TEST_F(StringAppendOperatorTest, VariousKeys) {
  auto db = OpenDb('\n');
  StringLists slists(db);

  slists.Append("c", "asdasd");
  slists.Append("a", "x");
  slists.Append("b", "y");
  slists.Append("a", "t");
  slists.Append("a", "r");
  slists.Append("b", "2");
  slists.Append("c", "asdasd");

  std::string a, b, c;
  bool sa, sb, sc;
  sa = slists.Get("a", &a);
  sb = slists.Get("b", &b);
  sc = slists.Get("c", &c);

  ASSERT_TRUE(sa && sb && sc); // All three keys should have been found

  ASSERT_EQ(a, "x\nt\nr");
  ASSERT_EQ(b, "y\n2");
  ASSERT_EQ(c, "asdasd\nasdasd");
}

// Generate semi random keys/words from a small distribution.
TEST_F(StringAppendOperatorTest, RandomMixGetAppend) {
  auto db = OpenDb(' ');
  StringLists slists(db);

  // Generate a list of random keys and values
  const int kWordCount = 15;
  std::string words[] = {"sdasd", "triejf", "fnjsdfn", "dfjisdfsf", "342839",
                         "dsuha", "mabuais", "sadajsid", "jf9834hf", "2d9j89",
                         "dj9823jd", "a", "dk02ed2dh", "$(jd4h984$(*", "mabz"};
  const int kKeyCount = 6;
  std::string keys[] = {"dhaiusdhu", "denidw", "daisda", "keykey", "muki",
                        "shzassdianmd"};

  // Will store a local copy of all data in order to verify correctness
  std::map<std::string, std::string> parallel_copy;

  // Generate a bunch of random queries (Append and Get)!
  enum query_t  { APPEND_OP, GET_OP, NUM_OPS };
  Random randomGen(1337);       //deterministic seed; always get same results!

  const int kNumQueries = 30;
  for (int q=0; q<kNumQueries; ++q) {
    // Generate a random query (Append or Get) and random parameters
    query_t query = (query_t)randomGen.Uniform((int)NUM_OPS);
    std::string key = keys[randomGen.Uniform((int)kKeyCount)];
    std::string word = words[randomGen.Uniform((int)kWordCount)];

    // Apply the query and any checks.
    if (query == APPEND_OP) {

      // Apply the rocksdb test-harness Append defined above
      slists.Append(key, word);  //apply the rocksdb append

      // Apply the similar "Append" to the parallel copy
      if (parallel_copy[key].size() > 0) {
        parallel_copy[key] += " " + word;
      } else {
        parallel_copy[key] = word;
      }

    } else if (query == GET_OP) {
      // Assumes that a non-existent key just returns <empty>
      std::string res;
      slists.Get(key, &res);
      ASSERT_EQ(res, parallel_copy[key]);
    }

  }

}

TEST_F(StringAppendOperatorTest, BIGRandomMixGetAppend) {
  auto db = OpenDb(' ');
  StringLists slists(db);

  // Generate a list of random keys and values
  const int kWordCount = 15;
  std::string words[] = {"sdasd", "triejf", "fnjsdfn", "dfjisdfsf", "342839",
                         "dsuha", "mabuais", "sadajsid", "jf9834hf", "2d9j89",
                         "dj9823jd", "a", "dk02ed2dh", "$(jd4h984$(*", "mabz"};
  const int kKeyCount = 6;
  std::string keys[] = {"dhaiusdhu", "denidw", "daisda", "keykey", "muki",
                        "shzassdianmd"};

  // Will store a local copy of all data in order to verify correctness
  std::map<std::string, std::string> parallel_copy;

  // Generate a bunch of random queries (Append and Get)!
  enum query_t  { APPEND_OP, GET_OP, NUM_OPS };
  Random randomGen(9138204);       // deterministic seed

  const int kNumQueries = 1000;
  for (int q=0; q<kNumQueries; ++q) {
    // Generate a random query (Append or Get) and random parameters
    query_t query = (query_t)randomGen.Uniform((int)NUM_OPS);
    std::string key = keys[randomGen.Uniform((int)kKeyCount)];
    std::string word = words[randomGen.Uniform((int)kWordCount)];

    //Apply the query and any checks.
    if (query == APPEND_OP) {

      // Apply the rocksdb test-harness Append defined above
      slists.Append(key, word);  //apply the rocksdb append

      // Apply the similar "Append" to the parallel copy
      if (parallel_copy[key].size() > 0) {
        parallel_copy[key] += " " + word;
      } else {
        parallel_copy[key] = word;
      }

    } else if (query == GET_OP) {
      // Assumes that a non-existent key just returns <empty>
      std::string res;
      slists.Get(key, &res);
      ASSERT_EQ(res, parallel_copy[key]);
    }

  }

}

TEST_F(StringAppendOperatorTest, PersistentVariousKeys) {
  // Perform the following operations in limited scope
  {
    auto db = OpenDb('\n');
    StringLists slists(db);

    slists.Append("c", "asdasd");
    slists.Append("a", "x");
    slists.Append("b", "y");
    slists.Append("a", "t");
    slists.Append("a", "r");
    slists.Append("b", "2");
    slists.Append("c", "asdasd");

    std::string a, b, c;
    slists.Get("a", &a);
    slists.Get("b", &b);
    slists.Get("c", &c);

    ASSERT_EQ(a, "x\nt\nr");
    ASSERT_EQ(b, "y\n2");
    ASSERT_EQ(c, "asdasd\nasdasd");
  }

  // Reopen the database (the previous changes should persist / be remembered)
  {
    auto db = OpenDb('\n');
    StringLists slists(db);

    slists.Append("c", "bbnagnagsx");
    slists.Append("a", "sa");
    slists.Append("b", "df");
    slists.Append("a", "gh");
    slists.Append("a", "jk");
    slists.Append("b", "l;");
    slists.Append("c", "rogosh");

    // The previous changes should be on disk (L0)
    // The most recent changes should be in memory (MemTable)
    // Hence, this will test both Get() paths.
    std::string a, b, c;
    slists.Get("a", &a);
    slists.Get("b", &b);
    slists.Get("c", &c);

    ASSERT_EQ(a, "x\nt\nr\nsa\ngh\njk");
    ASSERT_EQ(b, "y\n2\ndf\nl;");
    ASSERT_EQ(c, "asdasd\nasdasd\nbbnagnagsx\nrogosh");
  }

  // Reopen the database (the previous changes should persist / be remembered)
  {
    auto db = OpenDb('\n');
    StringLists slists(db);

    // All changes should be on disk. This will test VersionSet Get()
    std::string a, b, c;
    slists.Get("a", &a);
    slists.Get("b", &b);
    slists.Get("c", &c);

    ASSERT_EQ(a, "x\nt\nr\nsa\ngh\njk");
    ASSERT_EQ(b, "y\n2\ndf\nl;");
    ASSERT_EQ(c, "asdasd\nasdasd\nbbnagnagsx\nrogosh");
  }
}

TEST_F(StringAppendOperatorTest, PersistentFlushAndCompaction) {
  // Perform the following operations in limited scope
  {
    auto db = OpenDb('\n');
    StringLists slists(db);
    std::string a, b, c;
    bool success;

    // Append, Flush, Get
    slists.Append("c", "asdasd");
    db->Flush(rocksdb::FlushOptions());
    success = slists.Get("c", &c);
    ASSERT_TRUE(success);
    ASSERT_EQ(c, "asdasd");

    // Append, Flush, Append, Get
    slists.Append("a", "x");
    slists.Append("b", "y");
    db->Flush(rocksdb::FlushOptions());
    slists.Append("a", "t");
    slists.Append("a", "r");
    slists.Append("b", "2");

    success = slists.Get("a", &a);
    assert(success == true);
    ASSERT_EQ(a, "x\nt\nr");

    success = slists.Get("b", &b);
    assert(success == true);
    ASSERT_EQ(b, "y\n2");

    // Append, Get
    success = slists.Append("c", "asdasd");
    assert(success);
    success = slists.Append("b", "monkey");
    assert(success);

    // I omit the "assert(success)" checks here.
    slists.Get("a", &a);
    slists.Get("b", &b);
    slists.Get("c", &c);

    ASSERT_EQ(a, "x\nt\nr");
    ASSERT_EQ(b, "y\n2\nmonkey");
    ASSERT_EQ(c, "asdasd\nasdasd");
  }

  // Reopen the database (the previous changes should persist / be remembered)
  {
    auto db = OpenDb('\n');
    StringLists slists(db);
    std::string a, b, c;

    // Get (Quick check for persistence of previous database)
    slists.Get("a", &a);
    ASSERT_EQ(a, "x\nt\nr");

    //Append, Compact, Get
    slists.Append("c", "bbnagnagsx");
    slists.Append("a", "sa");
    slists.Append("b", "df");
    db->CompactRange(CompactRangeOptions(), nullptr, nullptr);
    slists.Get("a", &a);
    slists.Get("b", &b);
    slists.Get("c", &c);
    ASSERT_EQ(a, "x\nt\nr\nsa");
    ASSERT_EQ(b, "y\n2\nmonkey\ndf");
    ASSERT_EQ(c, "asdasd\nasdasd\nbbnagnagsx");

    // Append, Get
    slists.Append("a", "gh");
    slists.Append("a", "jk");
    slists.Append("b", "l;");
    slists.Append("c", "rogosh");
    slists.Get("a", &a);
    slists.Get("b", &b);
    slists.Get("c", &c);
    ASSERT_EQ(a, "x\nt\nr\nsa\ngh\njk");
    ASSERT_EQ(b, "y\n2\nmonkey\ndf\nl;");
    ASSERT_EQ(c, "asdasd\nasdasd\nbbnagnagsx\nrogosh");

    // Compact, Get
    db->CompactRange(CompactRangeOptions(), nullptr, nullptr);
    ASSERT_EQ(a, "x\nt\nr\nsa\ngh\njk");
    ASSERT_EQ(b, "y\n2\nmonkey\ndf\nl;");
    ASSERT_EQ(c, "asdasd\nasdasd\nbbnagnagsx\nrogosh");

    // Append, Flush, Compact, Get
    slists.Append("b", "afcg");
    db->Flush(rocksdb::FlushOptions());
    db->CompactRange(CompactRangeOptions(), nullptr, nullptr);
    slists.Get("b", &b);
    ASSERT_EQ(b, "y\n2\nmonkey\ndf\nl;\nafcg");
  }
}

TEST_F(StringAppendOperatorTest, SimpleTestNullDelimiter) {
  auto db = OpenDb('\0');
  StringLists slists(db);

  slists.Append("k1", "v1");
  slists.Append("k1", "v2");
  slists.Append("k1", "v3");

  std::string res;
  bool status = slists.Get("k1", &res);
  ASSERT_TRUE(status);

  // Construct the desired string. Default constructor doesn't like '\0' chars.
  std::string checker("v1,v2,v3");    // Verify that the string is right size.
  checker[2] = '\0';                  // Use null delimiter instead of comma.
  checker[5] = '\0';
  assert(checker.size() == 8);        // Verify it is still the correct size

  // Check that the rocksdb result string matches the desired string
  assert(res.size() == checker.size());
  ASSERT_EQ(res, checker);
}

} // namespace rocksdb

int main(int argc, char** argv) {
  ::testing::InitGoogleTest(&argc, argv);
  // Run with regular database
  int result;
  {
    fprintf(stderr, "Running tests with regular db and operator.\n");
    StringAppendOperatorTest::SetOpenDbFunction(&OpenNormalDb);
    result = RUN_ALL_TESTS();
  }

#ifndef ROCKSDB_LITE  // TtlDb is not supported in Lite
  // Run with TTL
  {
    fprintf(stderr, "Running tests with ttl db and generic operator.\n");
    StringAppendOperatorTest::SetOpenDbFunction(&OpenTtlDb);
    result |= RUN_ALL_TESTS();
  }
#endif  // !ROCKSDB_LITE

  return result;
}
