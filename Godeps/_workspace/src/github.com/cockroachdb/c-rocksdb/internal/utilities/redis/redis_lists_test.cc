//  Copyright (c) 2013, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.
/**
 * A test harness for the Redis API built on rocksdb.
 *
 * USAGE: Build with: "make redis_test" (in rocksdb directory).
 *        Run unit tests with: "./redis_test"
 *        Manual/Interactive user testing: "./redis_test -m"
 *        Manual user testing + restart database: "./redis_test -m -d"
 *
 * TODO:  Add LARGE random test cases to verify efficiency and scalability
 *
 * @author Deon Nicholas (dnicholas@fb.com)
 */

#ifndef ROCKSDB_LITE

#include <iostream>
#include <cctype>

#include "redis_lists.h"
#include "util/testharness.h"
#include "util/random.h"

using namespace rocksdb;
using namespace std;

namespace rocksdb {

class RedisListsTest : public testing::Test {
 public:
  static const string kDefaultDbName;
  static Options options;

  RedisListsTest() {
    options.create_if_missing = true;
  }
};

const string RedisListsTest::kDefaultDbName =
    test::TmpDir() + "/redis_lists_test";
Options RedisListsTest::options = Options();

// operator== and operator<< are defined below for vectors (lists)
// Needed for ASSERT_EQ

namespace {
void AssertListEq(const std::vector<std::string>& result,
                  const std::vector<std::string>& expected_result) {
  ASSERT_EQ(result.size(), expected_result.size());
  for (size_t i = 0; i < result.size(); ++i) {
    ASSERT_EQ(result[i], expected_result[i]);
  }
}
}  // namespace

// PushRight, Length, Index, Range
TEST_F(RedisListsTest, SimpleTest) {
  RedisLists redis(kDefaultDbName, options, true);   // Destructive

  string tempv; // Used below for all Index(), PopRight(), PopLeft()

  // Simple PushRight (should return the new length each time)
  ASSERT_EQ(redis.PushRight("k1", "v1"), 1);
  ASSERT_EQ(redis.PushRight("k1", "v2"), 2);
  ASSERT_EQ(redis.PushRight("k1", "v3"), 3);

  // Check Length and Index() functions
  ASSERT_EQ(redis.Length("k1"), 3);        // Check length
  ASSERT_TRUE(redis.Index("k1", 0, &tempv));
  ASSERT_EQ(tempv, "v1");   // Check valid indices
  ASSERT_TRUE(redis.Index("k1", 1, &tempv));
  ASSERT_EQ(tempv, "v2");
  ASSERT_TRUE(redis.Index("k1", 2, &tempv));
  ASSERT_EQ(tempv, "v3");

  // Check range function and vectors
  std::vector<std::string> result = redis.Range("k1", 0, 2);   // Get the list
  std::vector<std::string> expected_result(3);
  expected_result[0] = "v1";
  expected_result[1] = "v2";
  expected_result[2] = "v3";
  AssertListEq(result, expected_result);
}

// PushLeft, Length, Index, Range
TEST_F(RedisListsTest, SimpleTest2) {
  RedisLists redis(kDefaultDbName, options, true);   // Destructive

  string tempv; // Used below for all Index(), PopRight(), PopLeft()

  // Simple PushRight
  ASSERT_EQ(redis.PushLeft("k1", "v3"), 1);
  ASSERT_EQ(redis.PushLeft("k1", "v2"), 2);
  ASSERT_EQ(redis.PushLeft("k1", "v1"), 3);

  // Check Length and Index() functions
  ASSERT_EQ(redis.Length("k1"), 3);        // Check length
  ASSERT_TRUE(redis.Index("k1", 0, &tempv));
  ASSERT_EQ(tempv, "v1");   // Check valid indices
  ASSERT_TRUE(redis.Index("k1", 1, &tempv));
  ASSERT_EQ(tempv, "v2");
  ASSERT_TRUE(redis.Index("k1", 2, &tempv));
  ASSERT_EQ(tempv, "v3");

  // Check range function and vectors
  std::vector<std::string> result = redis.Range("k1", 0, 2);   // Get the list
  std::vector<std::string> expected_result(3);
  expected_result[0] = "v1";
  expected_result[1] = "v2";
  expected_result[2] = "v3";
  AssertListEq(result, expected_result);
}

// Exhaustive test of the Index() function
TEST_F(RedisListsTest, IndexTest) {
  RedisLists redis(kDefaultDbName, options, true);   // Destructive

  string tempv; // Used below for all Index(), PopRight(), PopLeft()

  // Empty Index check (return empty and should not crash or edit tempv)
  tempv = "yo";
  ASSERT_TRUE(!redis.Index("k1", 0, &tempv));
  ASSERT_EQ(tempv, "yo");
  ASSERT_TRUE(!redis.Index("fda", 3, &tempv));
  ASSERT_EQ(tempv, "yo");
  ASSERT_TRUE(!redis.Index("random", -12391, &tempv));
  ASSERT_EQ(tempv, "yo");

  // Simple Pushes (will yield: [v6, v4, v4, v1, v2, v3]
  redis.PushRight("k1", "v1");
  redis.PushRight("k1", "v2");
  redis.PushRight("k1", "v3");
  redis.PushLeft("k1", "v4");
  redis.PushLeft("k1", "v4");
  redis.PushLeft("k1", "v6");

  // Simple, non-negative indices
  ASSERT_TRUE(redis.Index("k1", 0, &tempv));
  ASSERT_EQ(tempv, "v6");
  ASSERT_TRUE(redis.Index("k1", 1, &tempv));
  ASSERT_EQ(tempv, "v4");
  ASSERT_TRUE(redis.Index("k1", 2, &tempv));
  ASSERT_EQ(tempv, "v4");
  ASSERT_TRUE(redis.Index("k1", 3, &tempv));
  ASSERT_EQ(tempv, "v1");
  ASSERT_TRUE(redis.Index("k1", 4, &tempv));
  ASSERT_EQ(tempv, "v2");
  ASSERT_TRUE(redis.Index("k1", 5, &tempv));
  ASSERT_EQ(tempv, "v3");

  // Negative indices
  ASSERT_TRUE(redis.Index("k1", -6, &tempv));
  ASSERT_EQ(tempv, "v6");
  ASSERT_TRUE(redis.Index("k1", -5, &tempv));
  ASSERT_EQ(tempv, "v4");
  ASSERT_TRUE(redis.Index("k1", -4, &tempv));
  ASSERT_EQ(tempv, "v4");
  ASSERT_TRUE(redis.Index("k1", -3, &tempv));
  ASSERT_EQ(tempv, "v1");
  ASSERT_TRUE(redis.Index("k1", -2, &tempv));
  ASSERT_EQ(tempv, "v2");
  ASSERT_TRUE(redis.Index("k1", -1, &tempv));
  ASSERT_EQ(tempv, "v3");

  // Out of bounds (return empty, no crash)
  ASSERT_TRUE(!redis.Index("k1", 6, &tempv));
  ASSERT_TRUE(!redis.Index("k1", 123219, &tempv));
  ASSERT_TRUE(!redis.Index("k1", -7, &tempv));
  ASSERT_TRUE(!redis.Index("k1", -129, &tempv));
}


// Exhaustive test of the Range() function
TEST_F(RedisListsTest, RangeTest) {
  RedisLists redis(kDefaultDbName, options, true);   // Destructive

  string tempv; // Used below for all Index(), PopRight(), PopLeft()

  // Simple Pushes (will yield: [v6, v4, v4, v1, v2, v3])
  redis.PushRight("k1", "v1");
  redis.PushRight("k1", "v2");
  redis.PushRight("k1", "v3");
  redis.PushLeft("k1", "v4");
  redis.PushLeft("k1", "v4");
  redis.PushLeft("k1", "v6");

  // Sanity check (check the length;  make sure it's 6)
  ASSERT_EQ(redis.Length("k1"), 6);

  // Simple range
  std::vector<std::string> res = redis.Range("k1", 1, 4);
  ASSERT_EQ((int)res.size(), 4);
  ASSERT_EQ(res[0], "v4");
  ASSERT_EQ(res[1], "v4");
  ASSERT_EQ(res[2], "v1");
  ASSERT_EQ(res[3], "v2");

  // Negative indices (i.e.: measured from the end)
  res = redis.Range("k1", 2, -1);
  ASSERT_EQ((int)res.size(), 4);
  ASSERT_EQ(res[0], "v4");
  ASSERT_EQ(res[1], "v1");
  ASSERT_EQ(res[2], "v2");
  ASSERT_EQ(res[3], "v3");

  res = redis.Range("k1", -6, -4);
  ASSERT_EQ((int)res.size(), 3);
  ASSERT_EQ(res[0], "v6");
  ASSERT_EQ(res[1], "v4");
  ASSERT_EQ(res[2], "v4");

  res = redis.Range("k1", -1, 5);
  ASSERT_EQ((int)res.size(), 1);
  ASSERT_EQ(res[0], "v3");

  // Partial / Broken indices
  res = redis.Range("k1", -3, 1000000);
  ASSERT_EQ((int)res.size(), 3);
  ASSERT_EQ(res[0], "v1");
  ASSERT_EQ(res[1], "v2");
  ASSERT_EQ(res[2], "v3");

  res = redis.Range("k1", -1000000, 1);
  ASSERT_EQ((int)res.size(), 2);
  ASSERT_EQ(res[0], "v6");
  ASSERT_EQ(res[1], "v4");

  // Invalid indices
  res = redis.Range("k1", 7, 9);
  ASSERT_EQ((int)res.size(), 0);

  res = redis.Range("k1", -8, -7);
  ASSERT_EQ((int)res.size(), 0);

  res = redis.Range("k1", 3, 2);
  ASSERT_EQ((int)res.size(), 0);

  res = redis.Range("k1", 5, -2);
  ASSERT_EQ((int)res.size(), 0);

  // Range matches Index
  res = redis.Range("k1", -6, -4);
  ASSERT_TRUE(redis.Index("k1", -6, &tempv));
  ASSERT_EQ(tempv, res[0]);
  ASSERT_TRUE(redis.Index("k1", -5, &tempv));
  ASSERT_EQ(tempv, res[1]);
  ASSERT_TRUE(redis.Index("k1", -4, &tempv));
  ASSERT_EQ(tempv, res[2]);

  // Last check
  res = redis.Range("k1", 0, -6);
  ASSERT_EQ((int)res.size(), 1);
  ASSERT_EQ(res[0], "v6");
}

// Exhaustive test for InsertBefore(), and InsertAfter()
TEST_F(RedisListsTest, InsertTest) {
  RedisLists redis(kDefaultDbName, options, true);

  string tempv; // Used below for all Index(), PopRight(), PopLeft()

  // Insert on empty list (return 0, and do not crash)
  ASSERT_EQ(redis.InsertBefore("k1", "non-exist", "a"), 0);
  ASSERT_EQ(redis.InsertAfter("k1", "other-non-exist", "c"), 0);
  ASSERT_EQ(redis.Length("k1"), 0);

  // Push some preliminary stuff [g, f, e, d, c, b, a]
  redis.PushLeft("k1", "a");
  redis.PushLeft("k1", "b");
  redis.PushLeft("k1", "c");
  redis.PushLeft("k1", "d");
  redis.PushLeft("k1", "e");
  redis.PushLeft("k1", "f");
  redis.PushLeft("k1", "g");
  ASSERT_EQ(redis.Length("k1"), 7);

  // Test InsertBefore
  int newLength = redis.InsertBefore("k1", "e", "hello");
  ASSERT_EQ(newLength, 8);
  ASSERT_EQ(redis.Length("k1"), newLength);
  ASSERT_TRUE(redis.Index("k1", 1, &tempv));
  ASSERT_EQ(tempv, "f");
  ASSERT_TRUE(redis.Index("k1", 3, &tempv));
  ASSERT_EQ(tempv, "e");
  ASSERT_TRUE(redis.Index("k1", 2, &tempv));
  ASSERT_EQ(tempv, "hello");

  // Test InsertAfter
  newLength =  redis.InsertAfter("k1", "c", "bye");
  ASSERT_EQ(newLength, 9);
  ASSERT_EQ(redis.Length("k1"), newLength);
  ASSERT_TRUE(redis.Index("k1", 6, &tempv));
  ASSERT_EQ(tempv, "bye");

  // Test bad value on InsertBefore
  newLength = redis.InsertBefore("k1", "yo", "x");
  ASSERT_EQ(newLength, 9);
  ASSERT_EQ(redis.Length("k1"), newLength);

  // Test bad value on InsertAfter
  newLength = redis.InsertAfter("k1", "xxxx", "y");
  ASSERT_EQ(newLength, 9);
  ASSERT_EQ(redis.Length("k1"), newLength);

  // Test InsertBefore beginning
  newLength = redis.InsertBefore("k1", "g", "begggggggggggggggg");
  ASSERT_EQ(newLength, 10);
  ASSERT_EQ(redis.Length("k1"), newLength);

  // Test InsertAfter end
  newLength = redis.InsertAfter("k1", "a", "enddd");
  ASSERT_EQ(newLength, 11);
  ASSERT_EQ(redis.Length("k1"), newLength);

  // Make sure nothing weird happened.
  ASSERT_TRUE(redis.Index("k1", 0, &tempv));
  ASSERT_EQ(tempv, "begggggggggggggggg");
  ASSERT_TRUE(redis.Index("k1", 1, &tempv));
  ASSERT_EQ(tempv, "g");
  ASSERT_TRUE(redis.Index("k1", 2, &tempv));
  ASSERT_EQ(tempv, "f");
  ASSERT_TRUE(redis.Index("k1", 3, &tempv));
  ASSERT_EQ(tempv, "hello");
  ASSERT_TRUE(redis.Index("k1", 4, &tempv));
  ASSERT_EQ(tempv, "e");
  ASSERT_TRUE(redis.Index("k1", 5, &tempv));
  ASSERT_EQ(tempv, "d");
  ASSERT_TRUE(redis.Index("k1", 6, &tempv));
  ASSERT_EQ(tempv, "c");
  ASSERT_TRUE(redis.Index("k1", 7, &tempv));
  ASSERT_EQ(tempv, "bye");
  ASSERT_TRUE(redis.Index("k1", 8, &tempv));
  ASSERT_EQ(tempv, "b");
  ASSERT_TRUE(redis.Index("k1", 9, &tempv));
  ASSERT_EQ(tempv, "a");
  ASSERT_TRUE(redis.Index("k1", 10, &tempv));
  ASSERT_EQ(tempv, "enddd");
}

// Exhaustive test of Set function
TEST_F(RedisListsTest, SetTest) {
  RedisLists redis(kDefaultDbName, options, true);

  string tempv; // Used below for all Index(), PopRight(), PopLeft()

  // Set on empty list (return false, and do not crash)
  ASSERT_EQ(redis.Set("k1", 7, "a"), false);
  ASSERT_EQ(redis.Set("k1", 0, "a"), false);
  ASSERT_EQ(redis.Set("k1", -49, "cx"), false);
  ASSERT_EQ(redis.Length("k1"), 0);

  // Push some preliminary stuff [g, f, e, d, c, b, a]
  redis.PushLeft("k1", "a");
  redis.PushLeft("k1", "b");
  redis.PushLeft("k1", "c");
  redis.PushLeft("k1", "d");
  redis.PushLeft("k1", "e");
  redis.PushLeft("k1", "f");
  redis.PushLeft("k1", "g");
  ASSERT_EQ(redis.Length("k1"), 7);

  // Test Regular Set
  ASSERT_TRUE(redis.Set("k1", 0, "0"));
  ASSERT_TRUE(redis.Set("k1", 3, "3"));
  ASSERT_TRUE(redis.Set("k1", 6, "6"));
  ASSERT_TRUE(redis.Set("k1", 2, "2"));
  ASSERT_TRUE(redis.Set("k1", 5, "5"));
  ASSERT_TRUE(redis.Set("k1", 1, "1"));
  ASSERT_TRUE(redis.Set("k1", 4, "4"));

  ASSERT_EQ(redis.Length("k1"), 7); // Size should not change
  ASSERT_TRUE(redis.Index("k1", 0, &tempv));
  ASSERT_EQ(tempv, "0");
  ASSERT_TRUE(redis.Index("k1", 1, &tempv));
  ASSERT_EQ(tempv, "1");
  ASSERT_TRUE(redis.Index("k1", 2, &tempv));
  ASSERT_EQ(tempv, "2");
  ASSERT_TRUE(redis.Index("k1", 3, &tempv));
  ASSERT_EQ(tempv, "3");
  ASSERT_TRUE(redis.Index("k1", 4, &tempv));
  ASSERT_EQ(tempv, "4");
  ASSERT_TRUE(redis.Index("k1", 5, &tempv));
  ASSERT_EQ(tempv, "5");
  ASSERT_TRUE(redis.Index("k1", 6, &tempv));
  ASSERT_EQ(tempv, "6");

  // Set with negative indices
  ASSERT_TRUE(redis.Set("k1", -7, "a"));
  ASSERT_TRUE(redis.Set("k1", -4, "d"));
  ASSERT_TRUE(redis.Set("k1", -1, "g"));
  ASSERT_TRUE(redis.Set("k1", -5, "c"));
  ASSERT_TRUE(redis.Set("k1", -2, "f"));
  ASSERT_TRUE(redis.Set("k1", -6, "b"));
  ASSERT_TRUE(redis.Set("k1", -3, "e"));

  ASSERT_EQ(redis.Length("k1"), 7); // Size should not change
  ASSERT_TRUE(redis.Index("k1", 0, &tempv));
  ASSERT_EQ(tempv, "a");
  ASSERT_TRUE(redis.Index("k1", 1, &tempv));
  ASSERT_EQ(tempv, "b");
  ASSERT_TRUE(redis.Index("k1", 2, &tempv));
  ASSERT_EQ(tempv, "c");
  ASSERT_TRUE(redis.Index("k1", 3, &tempv));
  ASSERT_EQ(tempv, "d");
  ASSERT_TRUE(redis.Index("k1", 4, &tempv));
  ASSERT_EQ(tempv, "e");
  ASSERT_TRUE(redis.Index("k1", 5, &tempv));
  ASSERT_EQ(tempv, "f");
  ASSERT_TRUE(redis.Index("k1", 6, &tempv));
  ASSERT_EQ(tempv, "g");

  // Bad indices (just out-of-bounds / off-by-one check)
  ASSERT_EQ(redis.Set("k1", -8, "off-by-one in negative index"), false);
  ASSERT_EQ(redis.Set("k1", 7, "off-by-one-error in positive index"), false);
  ASSERT_EQ(redis.Set("k1", 43892, "big random index should fail"), false);
  ASSERT_EQ(redis.Set("k1", -21391, "large negative index should fail"), false);

  // One last check (to make sure nothing weird happened)
  ASSERT_EQ(redis.Length("k1"), 7); // Size should not change
  ASSERT_TRUE(redis.Index("k1", 0, &tempv));
  ASSERT_EQ(tempv, "a");
  ASSERT_TRUE(redis.Index("k1", 1, &tempv));
  ASSERT_EQ(tempv, "b");
  ASSERT_TRUE(redis.Index("k1", 2, &tempv));
  ASSERT_EQ(tempv, "c");
  ASSERT_TRUE(redis.Index("k1", 3, &tempv));
  ASSERT_EQ(tempv, "d");
  ASSERT_TRUE(redis.Index("k1", 4, &tempv));
  ASSERT_EQ(tempv, "e");
  ASSERT_TRUE(redis.Index("k1", 5, &tempv));
  ASSERT_EQ(tempv, "f");
  ASSERT_TRUE(redis.Index("k1", 6, &tempv));
  ASSERT_EQ(tempv, "g");
}

// Testing Insert, Push, and Set, in a mixed environment
TEST_F(RedisListsTest, InsertPushSetTest) {
  RedisLists redis(kDefaultDbName, options, true);   // Destructive

  string tempv; // Used below for all Index(), PopRight(), PopLeft()

  // A series of pushes and insertions
  // Will result in [newbegin, z, a, aftera, x, newend]
  // Also, check the return value sometimes (should return length)
  int lengthCheck;
  lengthCheck = redis.PushLeft("k1", "a");
  ASSERT_EQ(lengthCheck, 1);
  redis.PushLeft("k1", "z");
  redis.PushRight("k1", "x");
  lengthCheck = redis.InsertAfter("k1", "a", "aftera");
  ASSERT_EQ(lengthCheck , 4);
  redis.InsertBefore("k1", "z", "newbegin");  // InsertBefore beginning of list
  redis.InsertAfter("k1", "x", "newend");     // InsertAfter end of list

  // Check
  std::vector<std::string> res = redis.Range("k1", 0, -1); // Get the list
  ASSERT_EQ((int)res.size(), 6);
  ASSERT_EQ(res[0], "newbegin");
  ASSERT_EQ(res[5], "newend");
  ASSERT_EQ(res[3], "aftera");

  // Testing duplicate values/pivots (multiple occurrences of 'a')
  ASSERT_TRUE(redis.Set("k1", 0, "a"));     // [a, z, a, aftera, x, newend]
  redis.InsertAfter("k1", "a", "happy");    // [a, happy, z, a, aftera, ...]
  ASSERT_TRUE(redis.Index("k1", 1, &tempv));
  ASSERT_EQ(tempv, "happy");
  redis.InsertBefore("k1", "a", "sad");     // [sad, a, happy, z, a, aftera, ...]
  ASSERT_TRUE(redis.Index("k1", 0, &tempv));
  ASSERT_EQ(tempv, "sad");
  ASSERT_TRUE(redis.Index("k1", 2, &tempv));
  ASSERT_EQ(tempv, "happy");
  ASSERT_TRUE(redis.Index("k1", 5, &tempv));
  ASSERT_EQ(tempv, "aftera");
  redis.InsertAfter("k1", "a", "zz");         // [sad, a, zz, happy, z, a, aftera, ...]
  ASSERT_TRUE(redis.Index("k1", 2, &tempv));
  ASSERT_EQ(tempv, "zz");
  ASSERT_TRUE(redis.Index("k1", 6, &tempv));
  ASSERT_EQ(tempv, "aftera");
  ASSERT_TRUE(redis.Set("k1", 1, "nota"));    // [sad, nota, zz, happy, z, a, ...]
  redis.InsertBefore("k1", "a", "ba");        // [sad, nota, zz, happy, z, ba, a, ...]
  ASSERT_TRUE(redis.Index("k1", 4, &tempv));
  ASSERT_EQ(tempv, "z");
  ASSERT_TRUE(redis.Index("k1", 5, &tempv));
  ASSERT_EQ(tempv, "ba");
  ASSERT_TRUE(redis.Index("k1", 6, &tempv));
  ASSERT_EQ(tempv, "a");

  // We currently have: [sad, nota, zz, happy, z, ba, a, aftera, x, newend]
  // redis.Print("k1");   // manually check

  // Test Inserting before/after non-existent values
  lengthCheck = redis.Length("k1"); // Ensure that the length doesn't change
  ASSERT_EQ(lengthCheck, 10);
  ASSERT_EQ(redis.InsertBefore("k1", "non-exist", "randval"), lengthCheck);
  ASSERT_EQ(redis.InsertAfter("k1", "nothing", "a"), lengthCheck);
  ASSERT_EQ(redis.InsertAfter("randKey", "randVal", "ranValue"), 0); // Empty
  ASSERT_EQ(redis.Length("k1"), lengthCheck); // The length should not change

  // Simply Test the Set() function
  redis.Set("k1", 5, "ba2");
  redis.InsertBefore("k1", "ba2", "beforeba2");
  ASSERT_TRUE(redis.Index("k1", 4, &tempv));
  ASSERT_EQ(tempv, "z");
  ASSERT_TRUE(redis.Index("k1", 5, &tempv));
  ASSERT_EQ(tempv, "beforeba2");
  ASSERT_TRUE(redis.Index("k1", 6, &tempv));
  ASSERT_EQ(tempv, "ba2");
  ASSERT_TRUE(redis.Index("k1", 7, &tempv));
  ASSERT_EQ(tempv, "a");

  // We have: [sad, nota, zz, happy, z, beforeba2, ba2, a, aftera, x, newend]

  // Set() with negative indices
  redis.Set("k1", -1, "endprank");
  ASSERT_TRUE(!redis.Index("k1", 11, &tempv));
  ASSERT_TRUE(redis.Index("k1", 10, &tempv));
  ASSERT_EQ(tempv, "endprank"); // Ensure Set worked correctly
  redis.Set("k1", -11, "t");
  ASSERT_TRUE(redis.Index("k1", 0, &tempv));
  ASSERT_EQ(tempv, "t");

  // Test out of bounds Set
  ASSERT_EQ(redis.Set("k1", -12, "ssd"), false);
  ASSERT_EQ(redis.Set("k1", 11, "sasd"), false);
  ASSERT_EQ(redis.Set("k1", 1200, "big"), false);
}

// Testing Trim, Pop
TEST_F(RedisListsTest, TrimPopTest) {
  RedisLists redis(kDefaultDbName, options, true);   // Destructive

  string tempv; // Used below for all Index(), PopRight(), PopLeft()

  // A series of pushes and insertions
  // Will result in [newbegin, z, a, aftera, x, newend]
  redis.PushLeft("k1", "a");
  redis.PushLeft("k1", "z");
  redis.PushRight("k1", "x");
  redis.InsertBefore("k1", "z", "newbegin");    // InsertBefore start of list
  redis.InsertAfter("k1", "x", "newend");       // InsertAfter end of list
  redis.InsertAfter("k1", "a", "aftera");

  // Simple PopLeft/Right test
  ASSERT_TRUE(redis.PopLeft("k1", &tempv));
  ASSERT_EQ(tempv, "newbegin");
  ASSERT_EQ(redis.Length("k1"), 5);
  ASSERT_TRUE(redis.Index("k1", 0, &tempv));
  ASSERT_EQ(tempv, "z");
  ASSERT_TRUE(redis.PopRight("k1", &tempv));
  ASSERT_EQ(tempv, "newend");
  ASSERT_EQ(redis.Length("k1"), 4);
  ASSERT_TRUE(redis.Index("k1", -1, &tempv));
  ASSERT_EQ(tempv, "x");

  // Now have: [z, a, aftera, x]

  // Test Trim
  ASSERT_TRUE(redis.Trim("k1", 0, -1));       // [z, a, aftera, x] (do nothing)
  ASSERT_EQ(redis.Length("k1"), 4);
  ASSERT_TRUE(redis.Trim("k1", 0, 2));                     // [z, a, aftera]
  ASSERT_EQ(redis.Length("k1"), 3);
  ASSERT_TRUE(redis.Index("k1", -1, &tempv));
  ASSERT_EQ(tempv, "aftera");
  ASSERT_TRUE(redis.Trim("k1", 1, 1));                     // [a]
  ASSERT_EQ(redis.Length("k1"), 1);
  ASSERT_TRUE(redis.Index("k1", 0, &tempv));
  ASSERT_EQ(tempv, "a");

  // Test out of bounds (empty) trim
  ASSERT_TRUE(redis.Trim("k1", 1, 0));
  ASSERT_EQ(redis.Length("k1"), 0);

  // Popping with empty list (return empty without error)
  ASSERT_TRUE(!redis.PopLeft("k1", &tempv));
  ASSERT_TRUE(!redis.PopRight("k1", &tempv));
  ASSERT_TRUE(redis.Trim("k1", 0, 5));

  // Exhaustive Trim test (negative and invalid indices)
  // Will start in [newbegin, z, a, aftera, x, newend]
  redis.PushLeft("k1", "a");
  redis.PushLeft("k1", "z");
  redis.PushRight("k1", "x");
  redis.InsertBefore("k1", "z", "newbegin");    // InsertBefore start of list
  redis.InsertAfter("k1", "x", "newend");       // InsertAfter end of list
  redis.InsertAfter("k1", "a", "aftera");
  ASSERT_TRUE(redis.Trim("k1", -6, -1));                     // Should do nothing
  ASSERT_EQ(redis.Length("k1"), 6);
  ASSERT_TRUE(redis.Trim("k1", 1, -2));
  ASSERT_TRUE(redis.Index("k1", 0, &tempv));
  ASSERT_EQ(tempv, "z");
  ASSERT_TRUE(redis.Index("k1", 3, &tempv));
  ASSERT_EQ(tempv, "x");
  ASSERT_EQ(redis.Length("k1"), 4);
  ASSERT_TRUE(redis.Trim("k1", -3, -2));
  ASSERT_EQ(redis.Length("k1"), 2);
}

// Testing Remove, RemoveFirst, RemoveLast
TEST_F(RedisListsTest, RemoveTest) {
  RedisLists redis(kDefaultDbName, options, true);   // Destructive

  string tempv; // Used below for all Index(), PopRight(), PopLeft()

  // A series of pushes and insertions
  // Will result in [newbegin, z, a, aftera, x, newend, a, a]
  redis.PushLeft("k1", "a");
  redis.PushLeft("k1", "z");
  redis.PushRight("k1", "x");
  redis.InsertBefore("k1", "z", "newbegin");    // InsertBefore start of list
  redis.InsertAfter("k1", "x", "newend");       // InsertAfter end of list
  redis.InsertAfter("k1", "a", "aftera");
  redis.PushRight("k1", "a");
  redis.PushRight("k1", "a");

  // Verify
  ASSERT_TRUE(redis.Index("k1", 0, &tempv));
  ASSERT_EQ(tempv, "newbegin");
  ASSERT_TRUE(redis.Index("k1", -1, &tempv));
  ASSERT_EQ(tempv, "a");

  // Check RemoveFirst (Remove the first two 'a')
  // Results in [newbegin, z, aftera, x, newend, a]
  int numRemoved = redis.Remove("k1", 2, "a");
  ASSERT_EQ(numRemoved, 2);
  ASSERT_TRUE(redis.Index("k1", 0, &tempv));
  ASSERT_EQ(tempv, "newbegin");
  ASSERT_TRUE(redis.Index("k1", 1, &tempv));
  ASSERT_EQ(tempv, "z");
  ASSERT_TRUE(redis.Index("k1", 4, &tempv));
  ASSERT_EQ(tempv, "newend");
  ASSERT_TRUE(redis.Index("k1", 5, &tempv));
  ASSERT_EQ(tempv, "a");
  ASSERT_EQ(redis.Length("k1"), 6);

  // Repopulate some stuff
  // Results in: [x, x, x, x, x, newbegin, z, x, aftera, x, newend, a, x]
  redis.PushLeft("k1", "x");
  redis.PushLeft("k1", "x");
  redis.PushLeft("k1", "x");
  redis.PushLeft("k1", "x");
  redis.PushLeft("k1", "x");
  redis.PushRight("k1", "x");
  redis.InsertAfter("k1", "z", "x");

  // Test removal from end
  numRemoved = redis.Remove("k1", -2, "x");
  ASSERT_EQ(numRemoved, 2);
  ASSERT_TRUE(redis.Index("k1", 8, &tempv));
  ASSERT_EQ(tempv, "aftera");
  ASSERT_TRUE(redis.Index("k1", 9, &tempv));
  ASSERT_EQ(tempv, "newend");
  ASSERT_TRUE(redis.Index("k1", 10, &tempv));
  ASSERT_EQ(tempv, "a");
  ASSERT_TRUE(!redis.Index("k1", 11, &tempv));
  numRemoved = redis.Remove("k1", -2, "x");
  ASSERT_EQ(numRemoved, 2);
  ASSERT_TRUE(redis.Index("k1", 4, &tempv));
  ASSERT_EQ(tempv, "newbegin");
  ASSERT_TRUE(redis.Index("k1", 6, &tempv));
  ASSERT_EQ(tempv, "aftera");

  // We now have: [x, x, x, x, newbegin, z, aftera, newend, a]
  ASSERT_EQ(redis.Length("k1"), 9);
  ASSERT_TRUE(redis.Index("k1", -1, &tempv));
  ASSERT_EQ(tempv, "a");
  ASSERT_TRUE(redis.Index("k1", 0, &tempv));
  ASSERT_EQ(tempv, "x");

  // Test over-shooting (removing more than there exists)
  numRemoved = redis.Remove("k1", -9000, "x");
  ASSERT_EQ(numRemoved , 4);    // Only really removed 4
  ASSERT_EQ(redis.Length("k1"), 5);
  ASSERT_TRUE(redis.Index("k1", 0, &tempv));
  ASSERT_EQ(tempv, "newbegin");
  numRemoved = redis.Remove("k1", 1, "x");
  ASSERT_EQ(numRemoved, 0);

  // Try removing ALL!
  numRemoved = redis.Remove("k1", 0, "newbegin");   // REMOVE 0 will remove all!
  ASSERT_EQ(numRemoved, 1);

  // Removal from an empty-list
  ASSERT_TRUE(redis.Trim("k1", 1, 0));
  numRemoved = redis.Remove("k1", 1, "z");
  ASSERT_EQ(numRemoved, 0);
}


// Test Multiple keys and Persistence
TEST_F(RedisListsTest, PersistenceMultiKeyTest) {
  string tempv; // Used below for all Index(), PopRight(), PopLeft()

  // Block one: populate a single key in the database
  {
    RedisLists redis(kDefaultDbName, options, true);   // Destructive

    // A series of pushes and insertions
    // Will result in [newbegin, z, a, aftera, x, newend, a, a]
    redis.PushLeft("k1", "a");
    redis.PushLeft("k1", "z");
    redis.PushRight("k1", "x");
    redis.InsertBefore("k1", "z", "newbegin");    // InsertBefore start of list
    redis.InsertAfter("k1", "x", "newend");       // InsertAfter end of list
    redis.InsertAfter("k1", "a", "aftera");
    redis.PushRight("k1", "a");
    redis.PushRight("k1", "a");

    ASSERT_TRUE(redis.Index("k1", 3, &tempv));
    ASSERT_EQ(tempv, "aftera");
  }

  // Block two: make sure changes were saved and add some other key
  {
    RedisLists redis(kDefaultDbName, options, false); // Persistent, non-destructive

    // Check
    ASSERT_EQ(redis.Length("k1"), 8);
    ASSERT_TRUE(redis.Index("k1", 3, &tempv));
    ASSERT_EQ(tempv, "aftera");

    redis.PushRight("k2", "randomkey");
    redis.PushLeft("k2", "sas");

    redis.PopLeft("k1", &tempv);
  }

  // Block three: Verify the changes from block 2
  {
    RedisLists redis(kDefaultDbName, options, false); // Persistent, non-destructive

    // Check
    ASSERT_EQ(redis.Length("k1"), 7);
    ASSERT_EQ(redis.Length("k2"), 2);
    ASSERT_TRUE(redis.Index("k1", 0, &tempv));
    ASSERT_EQ(tempv, "z");
    ASSERT_TRUE(redis.Index("k2", -2, &tempv));
    ASSERT_EQ(tempv, "sas");
  }
}

/// THE manual REDIS TEST begins here
/// THIS WILL ONLY OCCUR IF YOU RUN: ./redis_test -m

namespace {
void MakeUpper(std::string* const s) {
  int len = static_cast<int>(s->length());
  for (int i = 0; i < len; ++i) {
    (*s)[i] = toupper((*s)[i]);  // C-version defined in <ctype.h>
  }
}

/// Allows the user to enter in REDIS commands into the command-line.
/// This is useful for manual / interacticve testing / debugging.
///  Use destructive=true to clean the database before use.
///  Use destructive=false to remember the previous state (i.e.: persistent)
/// Should be called from main function.
int manual_redis_test(bool destructive){
  RedisLists redis(RedisListsTest::kDefaultDbName,
                   RedisListsTest::options,
                   destructive);

  // TODO: Right now, please use spaces to separate each word.
  //  In actual redis, you can use quotes to specify compound values
  //  Example: RPUSH mylist "this is a compound value"

  std::string command;
  while(true) {
    cin >> command;
    MakeUpper(&command);

    if (command == "LINSERT") {
      std::string k, t, p, v;
      cin >> k >> t >> p >> v;
      MakeUpper(&t);
      if (t=="BEFORE") {
        std::cout << redis.InsertBefore(k, p, v) << std::endl;
      } else if (t=="AFTER") {
        std::cout << redis.InsertAfter(k, p, v) << std::endl;
      }
    } else if (command == "LPUSH") {
      std::string k, v;
      std::cin >> k >> v;
      redis.PushLeft(k, v);
    } else if (command == "RPUSH") {
      std::string k, v;
      std::cin >> k >> v;
      redis.PushRight(k, v);
    } else if (command == "LPOP") {
      std::string k;
      std::cin >> k;
      string res;
      redis.PopLeft(k, &res);
      std::cout << res << std::endl;
    } else if (command == "RPOP") {
      std::string k;
      std::cin >> k;
      string res;
      redis.PopRight(k, &res);
      std::cout << res << std::endl;
    } else if (command == "LREM") {
      std::string k;
      int amt;
      std::string v;

      std::cin >> k >> amt >> v;
      std::cout << redis.Remove(k, amt, v) << std::endl;
    } else if (command == "LLEN") {
      std::string k;
      std::cin >> k;
      std::cout << redis.Length(k) << std::endl;
    } else if (command == "LRANGE") {
      std::string k;
      int i, j;
      std::cin >> k >> i >> j;
      std::vector<std::string> res = redis.Range(k, i, j);
      for (auto it = res.begin(); it != res.end(); ++it) {
        std::cout << " " << (*it);
      }
      std::cout << std::endl;
    } else if (command == "LTRIM") {
      std::string k;
      int i, j;
      std::cin >> k >> i >> j;
      redis.Trim(k, i, j);
    } else if (command == "LSET") {
      std::string k;
      int idx;
      std::string v;
      cin >> k >> idx >> v;
      redis.Set(k, idx, v);
    } else if (command == "LINDEX") {
      std::string k;
      int idx;
      std::cin >> k >> idx;
      string res;
      redis.Index(k, idx, &res);
      std::cout << res << std::endl;
    } else if (command == "PRINT") {      // Added by Deon
      std::string k;
      cin >> k;
      redis.Print(k);
    } else if (command == "QUIT") {
      return 0;
    } else {
      std::cout << "unknown command: " << command << std::endl;
    }
  }
}
}  // namespace

} // namespace rocksdb


// USAGE: "./redis_test" for default (unit tests)
//        "./redis_test -m" for manual testing (redis command api)
//        "./redis_test -m -d" for destructive manual test (erase db before use)


namespace {
// Check for "want" argument in the argument list
bool found_arg(int argc, char* argv[], const char* want){
  for(int i=1; i<argc; ++i){
    if (strcmp(argv[i], want) == 0) {
      return true;
    }
  }
  return false;
}
}  // namespace

// Will run unit tests.
// However, if -m is specified, it will do user manual/interactive testing
// -m -d is manual and destructive (will clear the database before use)
int main(int argc, char* argv[]) {
  ::testing::InitGoogleTest(&argc, argv);
  if (found_arg(argc, argv, "-m")) {
    bool destructive = found_arg(argc, argv, "-d");
    return rocksdb::manual_redis_test(destructive);
  } else {
    return RUN_ALL_TESTS();
  }
}

#else
#include <stdio.h>

int main(int argc, char* argv[]) {
  fprintf(stderr, "SKIPPED as redis is not supported in ROCKSDB_LITE\n");
  return 0;
}

#endif  // !ROCKSDB_LITE
