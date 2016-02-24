//  Copyright (c) 2013, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.

#include <algorithm>
#include <string>
#include <vector>

#include "db/merge_helper.h"
#include "rocksdb/comparator.h"
#include "util/coding.h"
#include "util/testharness.h"
#include "util/testutil.h"
#include "utilities/merge_operators.h"

namespace rocksdb {

class MergeHelperTest : public testing::Test {
 public:
  MergeHelperTest() = default;
  ~MergeHelperTest() = default;

  Status RunUInt64MergeHelper(SequenceNumber stop_before, bool at_bottom) {
    InitIterator();
    merge_op_ = MergeOperators::CreateUInt64AddOperator();
    merge_helper_.reset(new MergeHelper(BytewiseComparator(), merge_op_.get(),
                                        nullptr, 2U, false));
    return merge_helper_->MergeUntil(iter_.get(), stop_before, at_bottom,
                                     nullptr, Env::Default());
  }

  Status RunStringAppendMergeHelper(SequenceNumber stop_before,
                                    bool at_bottom) {
    InitIterator();
    merge_op_ = MergeOperators::CreateStringAppendTESTOperator();
    merge_helper_.reset(new MergeHelper(BytewiseComparator(), merge_op_.get(),
                                        nullptr, 2U, false));
    return merge_helper_->MergeUntil(iter_.get(), stop_before, at_bottom,
                                     nullptr, Env::Default());
  }

  std::string Key(const std::string& user_key, const SequenceNumber& seq,
      const ValueType& t) {
    return InternalKey(user_key, seq, t).Encode().ToString();
  }

  void AddKeyVal(const std::string& user_key, const SequenceNumber& seq,
                 const ValueType& t, const std::string& val,
                 bool corrupt = false) {
    InternalKey ikey = InternalKey(user_key, seq, t);
    if (corrupt) {
      test::CorruptKeyType(&ikey);
    }
    ks_.push_back(ikey.Encode().ToString());
    vs_.push_back(val);
  }

  void InitIterator() {
    iter_.reset(new test::VectorIterator(ks_, vs_));
    iter_->SeekToFirst();
  }

  std::string EncodeInt(uint64_t x) {
    std::string result;
    PutFixed64(&result, x);
    return result;
  }

  std::unique_ptr<test::VectorIterator> iter_;
  std::shared_ptr<MergeOperator> merge_op_;
  std::unique_ptr<MergeHelper> merge_helper_;
  std::vector<std::string> ks_;
  std::vector<std::string> vs_;
};

// If MergeHelper encounters a new key on the last level, we know that
// the key has no more history and it can merge keys.
TEST_F(MergeHelperTest, MergeAtBottomSuccess) {
  AddKeyVal("a", 20, kTypeMerge, EncodeInt(1U));
  AddKeyVal("a", 10, kTypeMerge, EncodeInt(3U));
  AddKeyVal("b", 10, kTypeMerge, EncodeInt(4U));  // <- Iterator after merge

  ASSERT_TRUE(RunUInt64MergeHelper(0, true).ok());
  ASSERT_EQ(ks_[2], iter_->key());
  ASSERT_EQ(Key("a", 20, kTypeValue), merge_helper_->keys()[0]);
  ASSERT_EQ(EncodeInt(4U), merge_helper_->values()[0]);
  ASSERT_EQ(1U, merge_helper_->keys().size());
  ASSERT_EQ(1U, merge_helper_->values().size());
}

// Merging with a value results in a successful merge.
TEST_F(MergeHelperTest, MergeValue) {
  AddKeyVal("a", 40, kTypeMerge, EncodeInt(1U));
  AddKeyVal("a", 30, kTypeMerge, EncodeInt(3U));
  AddKeyVal("a", 20, kTypeValue, EncodeInt(4U));  // <- Iterator after merge
  AddKeyVal("a", 10, kTypeMerge, EncodeInt(1U));

  ASSERT_TRUE(RunUInt64MergeHelper(0, false).ok());
  ASSERT_EQ(ks_[3], iter_->key());
  ASSERT_EQ(Key("a", 40, kTypeValue), merge_helper_->keys()[0]);
  ASSERT_EQ(EncodeInt(8U), merge_helper_->values()[0]);
  ASSERT_EQ(1U, merge_helper_->keys().size());
  ASSERT_EQ(1U, merge_helper_->values().size());
}

// Merging stops before a snapshot.
TEST_F(MergeHelperTest, SnapshotBeforeValue) {
  AddKeyVal("a", 50, kTypeMerge, EncodeInt(1U));
  AddKeyVal("a", 40, kTypeMerge, EncodeInt(3U));  // <- Iterator after merge
  AddKeyVal("a", 30, kTypeMerge, EncodeInt(1U));
  AddKeyVal("a", 20, kTypeValue, EncodeInt(4U));
  AddKeyVal("a", 10, kTypeMerge, EncodeInt(1U));

  ASSERT_TRUE(RunUInt64MergeHelper(31, true).IsMergeInProgress());
  ASSERT_EQ(ks_[2], iter_->key());
  ASSERT_EQ(Key("a", 50, kTypeMerge), merge_helper_->keys()[0]);
  ASSERT_EQ(EncodeInt(4U), merge_helper_->values()[0]);
  ASSERT_EQ(1U, merge_helper_->keys().size());
  ASSERT_EQ(1U, merge_helper_->values().size());
}

// MergeHelper preserves the operand stack for merge operators that
// cannot do a partial merge.
TEST_F(MergeHelperTest, NoPartialMerge) {
  AddKeyVal("a", 50, kTypeMerge, "v2");
  AddKeyVal("a", 40, kTypeMerge, "v");  // <- Iterator after merge
  AddKeyVal("a", 30, kTypeMerge, "v");

  ASSERT_TRUE(RunStringAppendMergeHelper(31, true).IsMergeInProgress());
  ASSERT_EQ(ks_[2], iter_->key());
  ASSERT_EQ(Key("a", 40, kTypeMerge), merge_helper_->keys()[0]);
  ASSERT_EQ("v", merge_helper_->values()[0]);
  ASSERT_EQ(Key("a", 50, kTypeMerge), merge_helper_->keys()[1]);
  ASSERT_EQ("v2", merge_helper_->values()[1]);
  ASSERT_EQ(2U, merge_helper_->keys().size());
  ASSERT_EQ(2U, merge_helper_->values().size());
}

// A single operand can not be merged.
TEST_F(MergeHelperTest, SingleOperand) {
  AddKeyVal("a", 50, kTypeMerge, EncodeInt(1U));

  ASSERT_TRUE(RunUInt64MergeHelper(31, true).IsMergeInProgress());
  ASSERT_FALSE(iter_->Valid());
  ASSERT_EQ(Key("a", 50, kTypeMerge), merge_helper_->keys()[0]);
  ASSERT_EQ(EncodeInt(1U), merge_helper_->values()[0]);
  ASSERT_EQ(1U, merge_helper_->keys().size());
  ASSERT_EQ(1U, merge_helper_->values().size());
}

// Merging with a deletion turns the deletion into a value
TEST_F(MergeHelperTest, MergeDeletion) {
  AddKeyVal("a", 30, kTypeMerge, EncodeInt(3U));
  AddKeyVal("a", 20, kTypeDeletion, "");

  ASSERT_TRUE(RunUInt64MergeHelper(15, false).ok());
  ASSERT_FALSE(iter_->Valid());
  ASSERT_EQ(Key("a", 30, kTypeValue), merge_helper_->keys()[0]);
  ASSERT_EQ(EncodeInt(3U), merge_helper_->values()[0]);
  ASSERT_EQ(1U, merge_helper_->keys().size());
  ASSERT_EQ(1U, merge_helper_->values().size());
}

// The merge helper stops upon encountering a corrupt key
TEST_F(MergeHelperTest, CorruptKey) {
  AddKeyVal("a", 30, kTypeMerge, EncodeInt(3U));
  AddKeyVal("a", 25, kTypeMerge, EncodeInt(1U));
  // Corrupt key
  AddKeyVal("a", 20, kTypeDeletion, "", true);  // <- Iterator after merge

  ASSERT_TRUE(RunUInt64MergeHelper(15, false).IsMergeInProgress());
  ASSERT_EQ(ks_[2], iter_->key());
  ASSERT_EQ(Key("a", 30, kTypeMerge), merge_helper_->keys()[0]);
  ASSERT_EQ(EncodeInt(4U), merge_helper_->values()[0]);
  ASSERT_EQ(1U, merge_helper_->keys().size());
  ASSERT_EQ(1U, merge_helper_->values().size());
}

}  // namespace rocksdb

int main(int argc, char** argv) {
  ::testing::InitGoogleTest(&argc, argv);
  return RUN_ALL_TESTS();
}
