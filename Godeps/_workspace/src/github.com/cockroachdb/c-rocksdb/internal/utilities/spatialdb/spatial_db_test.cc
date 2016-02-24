//  Copyright (c) 2013, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.

#ifndef ROCKSDB_LITE

#include <vector>
#include <string>
#include <set>

#include "rocksdb/utilities/spatial_db.h"
#include "util/compression.h"
#include "util/testharness.h"
#include "util/testutil.h"
#include "util/random.h"

namespace rocksdb {
namespace spatial {

class SpatialDBTest : public testing::Test {
 public:
  SpatialDBTest() {
    dbname_ = test::TmpDir() + "/spatial_db_test";
    DestroyDB(dbname_, Options());
  }

  void AssertCursorResults(BoundingBox<double> bbox, const std::string& index,
                           const std::vector<std::string>& blobs) {
    Cursor* c = db_->Query(ReadOptions(), bbox, index);
    ASSERT_OK(c->status());
    std::multiset<std::string> b;
    for (auto x : blobs) {
      b.insert(x);
    }

    while (c->Valid()) {
      auto itr = b.find(c->blob().ToString());
      ASSERT_TRUE(itr != b.end());
      b.erase(itr);
      c->Next();
    }
    ASSERT_EQ(b.size(), 0U);
    ASSERT_OK(c->status());
    delete c;
  }

  std::string dbname_;
  SpatialDB* db_;
};

TEST_F(SpatialDBTest, FeatureSetSerializeTest) {
  if (!LZ4_Supported()) {
    return;
  }
  FeatureSet fs;

  fs.Set("a", std::string("b"));
  fs.Set("x", static_cast<uint64_t>(3));
  fs.Set("y", false);
  fs.Set("n", Variant());  // null
  fs.Set("m", 3.25);

  ASSERT_TRUE(fs.Find("w") == fs.end());
  ASSERT_TRUE(fs.Find("x") != fs.end());
  ASSERT_TRUE((*fs.Find("x")).second == Variant(static_cast<uint64_t>(3)));
  ASSERT_TRUE((*fs.Find("y")).second != Variant(true));
  std::set<std::string> keys({"a", "x", "y", "n", "m"});
  for (const auto& x : fs) {
    ASSERT_TRUE(keys.find(x.first) != keys.end());
    keys.erase(x.first);
  }
  ASSERT_EQ(keys.size(), 0U);

  std::string serialized;
  fs.Serialize(&serialized);

  FeatureSet deserialized;
  ASSERT_TRUE(deserialized.Deserialize(serialized));

  ASSERT_TRUE(deserialized.Contains("a"));
  ASSERT_EQ(deserialized.Get("a").type(), Variant::kString);
  ASSERT_EQ(deserialized.Get("a").get_string(), "b");
  ASSERT_TRUE(deserialized.Contains("x"));
  ASSERT_EQ(deserialized.Get("x").type(), Variant::kInt);
  ASSERT_EQ(deserialized.Get("x").get_int(), static_cast<uint64_t>(3));
  ASSERT_TRUE(deserialized.Contains("y"));
  ASSERT_EQ(deserialized.Get("y").type(), Variant::kBool);
  ASSERT_EQ(deserialized.Get("y").get_bool(), false);
  ASSERT_TRUE(deserialized.Contains("n"));
  ASSERT_EQ(deserialized.Get("n").type(), Variant::kNull);
  ASSERT_TRUE(deserialized.Contains("m"));
  ASSERT_EQ(deserialized.Get("m").type(), Variant::kDouble);
  ASSERT_EQ(deserialized.Get("m").get_double(), 3.25);

  // corrupted serialization
  serialized = serialized.substr(0, serialized.size() - 3);
  deserialized.Clear();
  ASSERT_TRUE(!deserialized.Deserialize(serialized));
}

TEST_F(SpatialDBTest, TestNextID) {
  if (!LZ4_Supported()) {
    return;
  }
  ASSERT_OK(SpatialDB::Create(
      SpatialDBOptions(), dbname_,
      {SpatialIndexOptions("simple", BoundingBox<double>(0, 0, 100, 100), 2)}));

  ASSERT_OK(SpatialDB::Open(SpatialDBOptions(), dbname_, &db_));
  ASSERT_OK(db_->Insert(WriteOptions(), BoundingBox<double>(5, 5, 10, 10),
                        "one", FeatureSet(), {"simple"}));
  ASSERT_OK(db_->Insert(WriteOptions(), BoundingBox<double>(10, 10, 15, 15),
                        "two", FeatureSet(), {"simple"}));
  delete db_;

  ASSERT_OK(SpatialDB::Open(SpatialDBOptions(), dbname_, &db_));
  ASSERT_OK(db_->Insert(WriteOptions(), BoundingBox<double>(55, 55, 65, 65),
                        "three", FeatureSet(), {"simple"}));
  delete db_;

  ASSERT_OK(SpatialDB::Open(SpatialDBOptions(), dbname_, &db_));
  AssertCursorResults(BoundingBox<double>(0, 0, 100, 100), "simple",
                      {"one", "two", "three"});
  delete db_;
}

TEST_F(SpatialDBTest, FeatureSetTest) {
  if (!LZ4_Supported()) {
    return;
  }
  ASSERT_OK(SpatialDB::Create(
      SpatialDBOptions(), dbname_,
      {SpatialIndexOptions("simple", BoundingBox<double>(0, 0, 100, 100), 2)}));
  ASSERT_OK(SpatialDB::Open(SpatialDBOptions(), dbname_, &db_));

  FeatureSet fs;
  fs.Set("a", std::string("b"));
  fs.Set("c", std::string("d"));

  ASSERT_OK(db_->Insert(WriteOptions(), BoundingBox<double>(5, 5, 10, 10),
                        "one", fs, {"simple"}));

  Cursor* c =
      db_->Query(ReadOptions(), BoundingBox<double>(5, 5, 10, 10), "simple");

  ASSERT_TRUE(c->Valid());
  ASSERT_EQ(c->blob().compare("one"), 0);
  FeatureSet returned = c->feature_set();
  ASSERT_TRUE(returned.Contains("a"));
  ASSERT_TRUE(!returned.Contains("b"));
  ASSERT_TRUE(returned.Contains("c"));
  ASSERT_EQ(returned.Get("a").type(), Variant::kString);
  ASSERT_EQ(returned.Get("a").get_string(), "b");
  ASSERT_EQ(returned.Get("c").type(), Variant::kString);
  ASSERT_EQ(returned.Get("c").get_string(), "d");

  c->Next();
  ASSERT_TRUE(!c->Valid());

  delete c;
  delete db_;
}

TEST_F(SpatialDBTest, SimpleTest) {
  if (!LZ4_Supported()) {
    return;
  }
  // iter 0 -- not read only
  // iter 1 -- read only
  for (int iter = 0; iter < 2; ++iter) {
    DestroyDB(dbname_, Options());
    ASSERT_OK(SpatialDB::Create(
        SpatialDBOptions(), dbname_,
        {SpatialIndexOptions("index", BoundingBox<double>(0, 0, 128, 128),
                             3)}));
    ASSERT_OK(SpatialDB::Open(SpatialDBOptions(), dbname_, &db_));

    ASSERT_OK(db_->Insert(WriteOptions(), BoundingBox<double>(33, 17, 63, 79),
                          "one", FeatureSet(), {"index"}));
    ASSERT_OK(db_->Insert(WriteOptions(), BoundingBox<double>(65, 65, 111, 111),
                          "two", FeatureSet(), {"index"}));
    ASSERT_OK(db_->Insert(WriteOptions(), BoundingBox<double>(1, 49, 127, 63),
                          "three", FeatureSet(), {"index"}));
    ASSERT_OK(db_->Insert(WriteOptions(), BoundingBox<double>(20, 100, 21, 101),
                          "four", FeatureSet(), {"index"}));
    ASSERT_OK(db_->Insert(WriteOptions(), BoundingBox<double>(81, 33, 127, 63),
                          "five", FeatureSet(), {"index"}));
    ASSERT_OK(db_->Insert(WriteOptions(), BoundingBox<double>(1, 65, 47, 95),
                          "six", FeatureSet(), {"index"}));

    if (iter == 1) {
      delete db_;
      ASSERT_OK(SpatialDB::Open(SpatialDBOptions(), dbname_, &db_, true));
    }

    AssertCursorResults(BoundingBox<double>(33, 17, 47, 31), "index", {"one"});
    AssertCursorResults(BoundingBox<double>(17, 33, 79, 63), "index",
                        {"one", "three"});
    AssertCursorResults(BoundingBox<double>(17, 81, 63, 111), "index",
                        {"four", "six"});
    AssertCursorResults(BoundingBox<double>(85, 86, 85, 86), "index", {"two"});
    AssertCursorResults(BoundingBox<double>(33, 1, 127, 111), "index",
                        {"one", "two", "three", "five", "six"});
    // even though the bounding box doesn't intersect, we got "four" back
    // because
    // it's in the same tile
    AssertCursorResults(BoundingBox<double>(18, 98, 19, 99), "index", {"four"});
    AssertCursorResults(BoundingBox<double>(130, 130, 131, 131), "index", {});
    AssertCursorResults(BoundingBox<double>(81, 17, 127, 31), "index", {});
    AssertCursorResults(BoundingBox<double>(90, 50, 91, 51), "index",
                        {"three", "five"});

    delete db_;
  }
}

namespace {
std::string RandomStr(Random* rnd) {
  std::string r;
  for (int k = 0; k < 10; ++k) {
    r.push_back(rnd->Uniform(26) + 'a');
  }
  return r;
}

BoundingBox<int> RandomBoundingBox(int limit, Random* rnd, int max_size) {
  BoundingBox<int> r;
  r.min_x = rnd->Uniform(limit - 1);
  r.min_y = rnd->Uniform(limit - 1);
  r.max_x = r.min_x + rnd->Uniform(std::min(limit - 1 - r.min_x, max_size)) + 1;
  r.max_y = r.min_y + rnd->Uniform(std::min(limit - 1 - r.min_y, max_size)) + 1;
  return r;
}

BoundingBox<double> ScaleBB(BoundingBox<int> b, double step) {
  return BoundingBox<double>(b.min_x * step + 1, b.min_y * step + 1,
                             (b.max_x + 1) * step - 1,
                             (b.max_y + 1) * step - 1);
}

}  // namespace

TEST_F(SpatialDBTest, RandomizedTest) {
  if (!LZ4_Supported()) {
    return;
  }
  Random rnd(301);
  std::vector<std::pair<std::string, BoundingBox<int>>> elements;

  BoundingBox<double> spatial_index_bounds(0, 0, (1LL << 32), (1LL << 32));
  ASSERT_OK(SpatialDB::Create(
      SpatialDBOptions(), dbname_,
      {SpatialIndexOptions("index", spatial_index_bounds, 7)}));
  ASSERT_OK(SpatialDB::Open(SpatialDBOptions(), dbname_, &db_));
  double step = (1LL << 32) / (1 << 7);

  for (int i = 0; i < 1000; ++i) {
    std::string blob = RandomStr(&rnd);
    BoundingBox<int> bbox = RandomBoundingBox(128, &rnd, 10);
    ASSERT_OK(db_->Insert(WriteOptions(), ScaleBB(bbox, step), blob,
                          FeatureSet(), {"index"}));
    elements.push_back(make_pair(blob, bbox));
  }

  // parallel
  db_->Compact(2);
  // serial
  db_->Compact(1);

  for (int i = 0; i < 1000; ++i) {
    BoundingBox<int> int_bbox = RandomBoundingBox(128, &rnd, 10);
    BoundingBox<double> double_bbox = ScaleBB(int_bbox, step);
    std::vector<std::string> blobs;
    for (auto e : elements) {
      if (e.second.Intersects(int_bbox)) {
        blobs.push_back(e.first);
      }
    }
    AssertCursorResults(double_bbox, "index", blobs);
  }

  delete db_;
}

}  // namespace spatial
}  // namespace rocksdb

int main(int argc, char** argv) {
  ::testing::InitGoogleTest(&argc, argv);
  return RUN_ALL_TESTS();
}

#else
#include <stdio.h>

int main(int argc, char** argv) {
  fprintf(stderr, "SKIPPED as SpatialDB is not supported in ROCKSDB_LITE\n");
  return 0;
}

#endif  // !ROCKSDB_LITE
