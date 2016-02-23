//  Copyright (c) 2013, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.

#ifndef ROCKSDB_LITE

#include <map>
#include <set>
#include <string>

#include "rocksdb/utilities/json_document.h"

#include "util/testutil.h"
#include "util/testharness.h"

namespace rocksdb {
namespace {
void AssertField(const JSONDocument& json, const std::string& field) {
  ASSERT_TRUE(json.Contains(field));
  ASSERT_TRUE(json[field].IsNull());
}

void AssertField(const JSONDocument& json, const std::string& field,
                 const std::string& expected) {
  ASSERT_TRUE(json.Contains(field));
  ASSERT_TRUE(json[field].IsString());
  ASSERT_EQ(expected, json[field].GetString());
}

void AssertField(const JSONDocument& json, const std::string& field,
                 int64_t expected) {
  ASSERT_TRUE(json.Contains(field));
  ASSERT_TRUE(json[field].IsInt64());
  ASSERT_EQ(expected, json[field].GetInt64());
}

void AssertField(const JSONDocument& json, const std::string& field,
                 bool expected) {
  ASSERT_TRUE(json.Contains(field));
  ASSERT_TRUE(json[field].IsBool());
  ASSERT_EQ(expected, json[field].GetBool());
}

void AssertField(const JSONDocument& json, const std::string& field,
                 double expected) {
  ASSERT_TRUE(json.Contains(field));
  ASSERT_TRUE(json[field].IsDouble());
  ASSERT_EQ(expected, json[field].GetDouble());
}
}  // namespace

class JSONDocumentTest : public testing::Test {
 public:
  JSONDocumentTest()
  : rnd_(101)
  {}

  void AssertSampleJSON(const JSONDocument& json) {
    AssertField(json, "title", std::string("json"));
    AssertField(json, "type", std::string("object"));
    // properties
    ASSERT_TRUE(json.Contains("properties"));
    ASSERT_TRUE(json["properties"].Contains("flags"));
    ASSERT_TRUE(json["properties"]["flags"].IsArray());
    ASSERT_EQ(3u, json["properties"]["flags"].Count());
    ASSERT_TRUE(json["properties"]["flags"][0].IsInt64());
    ASSERT_EQ(10, json["properties"]["flags"][0].GetInt64());
    ASSERT_TRUE(json["properties"]["flags"][1].IsString());
    ASSERT_EQ("parse", json["properties"]["flags"][1].GetString());
    ASSERT_TRUE(json["properties"]["flags"][2].IsObject());
    AssertField(json["properties"]["flags"][2], "tag", std::string("no"));
    AssertField(json["properties"]["flags"][2], std::string("status"));
    AssertField(json["properties"], "age", 110.5e-4);
    AssertField(json["properties"], "depth", static_cast<int64_t>(-10));
    // test iteration
    std::set<std::string> expected({"flags", "age", "depth"});
    for (auto item : json["properties"].Items()) {
      auto iter = expected.find(item.first);
      ASSERT_TRUE(iter != expected.end());
      expected.erase(iter);
    }
    ASSERT_EQ(0U, expected.size());
    ASSERT_TRUE(json.Contains("latlong"));
    ASSERT_TRUE(json["latlong"].IsArray());
    ASSERT_EQ(2u, json["latlong"].Count());
    ASSERT_TRUE(json["latlong"][0].IsDouble());
    ASSERT_EQ(53.25, json["latlong"][0].GetDouble());
    ASSERT_TRUE(json["latlong"][1].IsDouble());
    ASSERT_EQ(43.75, json["latlong"][1].GetDouble());
    AssertField(json, "enabled", true);
  }

  const std::string kSampleJSON =
      "{ \"title\" : \"json\", \"type\" : \"object\", \"properties\" : { "
      "\"flags\": [10, \"parse\", {\"tag\": \"no\", \"status\": null}], "
      "\"age\": 110.5e-4, \"depth\": -10 }, \"latlong\": [53.25, 43.75], "
      "\"enabled\": true }";

  const std::string kSampleJSONDifferent =
      "{ \"title\" : \"json\", \"type\" : \"object\", \"properties\" : { "
      "\"flags\": [10, \"parse\", {\"tag\": \"no\", \"status\": 2}], "
      "\"age\": 110.5e-4, \"depth\": -10 }, \"latlong\": [53.25, 43.75], "
      "\"enabled\": true }";

  Random rnd_;
};

TEST_F(JSONDocumentTest, MakeNullTest) {
  JSONDocument x;
  ASSERT_TRUE(x.IsNull());
  ASSERT_TRUE(x.IsOwner());
  ASSERT_TRUE(!x.IsBool());
}

TEST_F(JSONDocumentTest, MakeBoolTest) {
  {
    JSONDocument x(true);
    ASSERT_TRUE(x.IsOwner());
    ASSERT_TRUE(x.IsBool());
    ASSERT_TRUE(!x.IsInt64());
    ASSERT_EQ(x.GetBool(), true);
  }

  {
    JSONDocument x(false);
    ASSERT_TRUE(x.IsOwner());
    ASSERT_TRUE(x.IsBool());
    ASSERT_TRUE(!x.IsInt64());
    ASSERT_EQ(x.GetBool(), false);
  }
}

TEST_F(JSONDocumentTest, MakeInt64Test) {
  JSONDocument x(static_cast<int64_t>(16));
  ASSERT_TRUE(x.IsInt64());
  ASSERT_TRUE(x.IsInt64());
  ASSERT_TRUE(!x.IsBool());
  ASSERT_TRUE(x.IsOwner());
  ASSERT_EQ(x.GetInt64(), 16);
}

TEST_F(JSONDocumentTest, MakeStringTest) {
  JSONDocument x("string");
  ASSERT_TRUE(x.IsOwner());
  ASSERT_TRUE(x.IsString());
  ASSERT_TRUE(!x.IsBool());
  ASSERT_EQ(x.GetString(), "string");
}

TEST_F(JSONDocumentTest, MakeDoubleTest) {
  JSONDocument x(5.6);
  ASSERT_TRUE(x.IsOwner());
  ASSERT_TRUE(x.IsDouble());
  ASSERT_TRUE(!x.IsBool());
  ASSERT_EQ(x.GetDouble(), 5.6);
}

TEST_F(JSONDocumentTest, MakeByTypeTest) {
  {
    JSONDocument x(JSONDocument::kNull);
    ASSERT_TRUE(x.IsNull());
  }
  {
    JSONDocument x(JSONDocument::kBool);
    ASSERT_TRUE(x.IsBool());
  }
  {
    JSONDocument x(JSONDocument::kString);
    ASSERT_TRUE(x.IsString());
  }
  {
    JSONDocument x(JSONDocument::kInt64);
    ASSERT_TRUE(x.IsInt64());
  }
  {
    JSONDocument x(JSONDocument::kDouble);
    ASSERT_TRUE(x.IsDouble());
  }
  {
    JSONDocument x(JSONDocument::kObject);
    ASSERT_TRUE(x.IsObject());
  }
  {
    JSONDocument x(JSONDocument::kArray);
    ASSERT_TRUE(x.IsArray());
  }
}

TEST_F(JSONDocumentTest, Parsing) {
  std::unique_ptr<JSONDocument> parsed_json(
          JSONDocument::ParseJSON(kSampleJSON.c_str()));
  ASSERT_TRUE(parsed_json->IsOwner());
  ASSERT_TRUE(parsed_json != nullptr);
  AssertSampleJSON(*parsed_json);

  // test deep copying
  JSONDocument copied_json_document(*parsed_json);
  AssertSampleJSON(copied_json_document);
  ASSERT_TRUE(copied_json_document == *parsed_json);

  std::unique_ptr<JSONDocument> parsed_different_sample(
      JSONDocument::ParseJSON(kSampleJSONDifferent.c_str()));
  ASSERT_TRUE(parsed_different_sample != nullptr);
  ASSERT_TRUE(!(*parsed_different_sample == copied_json_document));

  // parse error
  const std::string kFaultyJSON =
      kSampleJSON.substr(0, kSampleJSON.size() - 10);
  ASSERT_TRUE(JSONDocument::ParseJSON(kFaultyJSON.c_str()) == nullptr);
}

TEST_F(JSONDocumentTest, Serialization) {
  std::unique_ptr<JSONDocument> parsed_json(
            JSONDocument::ParseJSON(kSampleJSON.c_str()));
  ASSERT_TRUE(parsed_json != nullptr);
  ASSERT_TRUE(parsed_json->IsOwner());
  std::string serialized;
  parsed_json->Serialize(&serialized);

  std::unique_ptr<JSONDocument> deserialized_json(
            JSONDocument::Deserialize(Slice(serialized)));
  ASSERT_TRUE(deserialized_json != nullptr);
  AssertSampleJSON(*deserialized_json);

  // deserialization failure
  ASSERT_TRUE(JSONDocument::Deserialize(
                  Slice(serialized.data(), serialized.size() - 10)) == nullptr);
}

TEST_F(JSONDocumentTest, OperatorEqualsTest) {
  // kNull
  ASSERT_TRUE(JSONDocument() == JSONDocument());

  // kBool
  ASSERT_TRUE(JSONDocument(false) != JSONDocument());
  ASSERT_TRUE(JSONDocument(false) == JSONDocument(false));
  ASSERT_TRUE(JSONDocument(true) == JSONDocument(true));
  ASSERT_TRUE(JSONDocument(false) != JSONDocument(true));

  // kString
  ASSERT_TRUE(JSONDocument("test") != JSONDocument());
  ASSERT_TRUE(JSONDocument("test") == JSONDocument("test"));

  // kInt64
  ASSERT_TRUE(JSONDocument(static_cast<int64_t>(15)) != JSONDocument());
  ASSERT_TRUE(JSONDocument(static_cast<int64_t>(15)) !=
              JSONDocument(static_cast<int64_t>(14)));
  ASSERT_TRUE(JSONDocument(static_cast<int64_t>(15)) ==
              JSONDocument(static_cast<int64_t>(15)));

  unique_ptr<JSONDocument> arrayWithInt8Doc(JSONDocument::ParseJSON("[8]"));
  ASSERT_TRUE(arrayWithInt8Doc != nullptr);
  ASSERT_TRUE(arrayWithInt8Doc->IsArray());
  ASSERT_TRUE((*arrayWithInt8Doc)[0].IsInt64());
  ASSERT_TRUE((*arrayWithInt8Doc)[0] == JSONDocument(static_cast<int64_t>(8)));

  unique_ptr<JSONDocument> arrayWithInt16Doc(JSONDocument::ParseJSON("[512]"));
  ASSERT_TRUE(arrayWithInt16Doc != nullptr);
  ASSERT_TRUE(arrayWithInt16Doc->IsArray());
  ASSERT_TRUE((*arrayWithInt16Doc)[0].IsInt64());
  ASSERT_TRUE((*arrayWithInt16Doc)[0] ==
              JSONDocument(static_cast<int64_t>(512)));

  unique_ptr<JSONDocument> arrayWithInt32Doc(
    JSONDocument::ParseJSON("[1000000]"));
  ASSERT_TRUE(arrayWithInt32Doc != nullptr);
  ASSERT_TRUE(arrayWithInt32Doc->IsArray());
  ASSERT_TRUE((*arrayWithInt32Doc)[0].IsInt64());
  ASSERT_TRUE((*arrayWithInt32Doc)[0] ==
               JSONDocument(static_cast<int64_t>(1000000)));

  // kDouble
  ASSERT_TRUE(JSONDocument(15.) != JSONDocument());
  ASSERT_TRUE(JSONDocument(15.) != JSONDocument(14.));
  ASSERT_TRUE(JSONDocument(15.) == JSONDocument(15.));
}

TEST_F(JSONDocumentTest, JSONDocumentBuilderTest) {
  unique_ptr<JSONDocument> parsedArray(
    JSONDocument::ParseJSON("[1, [123, \"a\", \"b\"], {\"b\":\"c\"}]"));
  ASSERT_TRUE(parsedArray != nullptr);

  JSONDocumentBuilder builder;
  ASSERT_TRUE(builder.WriteStartArray());
  ASSERT_TRUE(builder.WriteJSONDocument(1));

  ASSERT_TRUE(builder.WriteStartArray());
    ASSERT_TRUE(builder.WriteJSONDocument(123));
    ASSERT_TRUE(builder.WriteJSONDocument("a"));
    ASSERT_TRUE(builder.WriteJSONDocument("b"));
  ASSERT_TRUE(builder.WriteEndArray());

  ASSERT_TRUE(builder.WriteStartObject());
    ASSERT_TRUE(builder.WriteKeyValue("b", "c"));
  ASSERT_TRUE(builder.WriteEndObject());

  ASSERT_TRUE(builder.WriteEndArray());

  ASSERT_TRUE(*parsedArray == builder.GetJSONDocument());
}

TEST_F(JSONDocumentTest, OwnershipTest) {
  std::unique_ptr<JSONDocument> parsed(
          JSONDocument::ParseJSON(kSampleJSON.c_str()));
  ASSERT_TRUE(parsed != nullptr);
  ASSERT_TRUE(parsed->IsOwner());

  // Copy constructor from owner -> owner
  JSONDocument copy_constructor(*parsed);
  ASSERT_TRUE(copy_constructor.IsOwner());

  // Copy constructor from non-owner -> non-owner
  JSONDocument non_owner((*parsed)["properties"]);
  ASSERT_TRUE(!non_owner.IsOwner());

  // Move constructor from owner -> owner
  JSONDocument moved_from_owner(std::move(copy_constructor));
  ASSERT_TRUE(moved_from_owner.IsOwner());

  // Move constructor from non-owner -> non-owner
  JSONDocument moved_from_non_owner(std::move(non_owner));
  ASSERT_TRUE(!moved_from_non_owner.IsOwner());
}

}  //  namespace rocksdb

int main(int argc, char** argv) {
  ::testing::InitGoogleTest(&argc, argv);
  return RUN_ALL_TESTS();
}

#else
#include <stdio.h>

int main(int argc, char** argv) {
  fprintf(stderr, "SKIPPED as JSONDocument is not supported in ROCKSDB_LITE\n");
  return 0;
}

#endif  // !ROCKSDB_LITE
