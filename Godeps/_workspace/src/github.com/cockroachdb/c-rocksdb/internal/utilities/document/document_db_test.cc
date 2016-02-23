//  Copyright (c) 2013, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.

#ifndef ROCKSDB_LITE

#include <algorithm>

#include "rocksdb/utilities/json_document.h"
#include "rocksdb/utilities/document_db.h"

#include "util/testharness.h"
#include "util/testutil.h"

namespace rocksdb {

class DocumentDBTest : public testing::Test {
 public:
  DocumentDBTest() {
    dbname_ = test::TmpDir() + "/document_db_test";
    DestroyDB(dbname_, Options());
  }
  ~DocumentDBTest() {
    delete db_;
    DestroyDB(dbname_, Options());
  }

  void AssertCursorIDs(Cursor* cursor, std::vector<int64_t> expected) {
    std::vector<int64_t> got;
    while (cursor->Valid()) {
      ASSERT_TRUE(cursor->Valid());
      ASSERT_TRUE(cursor->document().Contains("_id"));
      got.push_back(cursor->document()["_id"].GetInt64());
      cursor->Next();
    }
    std::sort(expected.begin(), expected.end());
    std::sort(got.begin(), got.end());
    ASSERT_TRUE(got == expected);
  }

  // converts ' to ", so that we don't have to escape " all over the place
  std::string ConvertQuotes(const std::string& input) {
    std::string output;
    for (auto x : input) {
      if (x == '\'') {
        output.push_back('\"');
      } else {
        output.push_back(x);
      }
    }
    return output;
  }

  void CreateIndexes(std::vector<DocumentDB::IndexDescriptor> indexes) {
    for (auto i : indexes) {
      ASSERT_OK(db_->CreateIndex(WriteOptions(), i));
    }
  }

  JSONDocument* Parse(const std::string& doc) {
    return JSONDocument::ParseJSON(ConvertQuotes(doc).c_str());
  }

  std::string dbname_;
  DocumentDB* db_;
};

TEST_F(DocumentDBTest, SimpleQueryTest) {
  DocumentDBOptions options;
  DocumentDB::IndexDescriptor index;
  index.description = Parse("{\"name\": 1}");
  index.name = "name_index";

  ASSERT_OK(DocumentDB::Open(options, dbname_, {}, &db_));
  CreateIndexes({index});
  delete db_;
  // now there is index present
  ASSERT_OK(DocumentDB::Open(options, dbname_, {index}, &db_));
  delete index.description;

  std::vector<std::string> json_objects = {
      "{\"_id\': 1, \"name\": \"One\"}",   "{\"_id\": 2, \"name\": \"Two\"}",
      "{\"_id\": 3, \"name\": \"Three\"}", "{\"_id\": 4, \"name\": \"Four\"}"};

  for (auto& json : json_objects) {
    std::unique_ptr<JSONDocument> document(Parse(json));
    ASSERT_TRUE(document.get() != nullptr);
    ASSERT_OK(db_->Insert(WriteOptions(), *document));
  }

  // inserting a document with existing primary key should return failure
  {
    std::unique_ptr<JSONDocument> document(Parse(json_objects[0]));
    ASSERT_TRUE(document.get() != nullptr);
    Status s = db_->Insert(WriteOptions(), *document);
    ASSERT_TRUE(s.IsInvalidArgument());
  }

  // find equal to "Two"
  {
    std::unique_ptr<JSONDocument> query(
        Parse("[{'$filter': {'name': 'Two', '$index': 'name_index'}}]"));
    std::unique_ptr<Cursor> cursor(db_->Query(ReadOptions(), *query));
    AssertCursorIDs(cursor.get(), {2});
  }

  // find less than "Three"
  {
    std::unique_ptr<JSONDocument> query(Parse(
        "[{'$filter': {'name': {'$lt': 'Three'}, '$index': "
        "'name_index'}}]"));
    std::unique_ptr<Cursor> cursor(db_->Query(ReadOptions(), *query));

    AssertCursorIDs(cursor.get(), {1, 4});
  }

  // find less than "Three" without index
  {
    std::unique_ptr<JSONDocument> query(
        Parse("[{'$filter': {'name': {'$lt': 'Three'} }}]"));
    std::unique_ptr<Cursor> cursor(db_->Query(ReadOptions(), *query));
    AssertCursorIDs(cursor.get(), {1, 4});
  }

  // remove less or equal to "Three"
  {
    std::unique_ptr<JSONDocument> query(
        Parse("{'name': {'$lte': 'Three'}, '$index': 'name_index'}"));
    ASSERT_OK(db_->Remove(ReadOptions(), WriteOptions(), *query));
  }

  // find all -- only "Two" left, everything else should be deleted
  {
    std::unique_ptr<JSONDocument> query(Parse("[]"));
    std::unique_ptr<Cursor> cursor(db_->Query(ReadOptions(), *query));
    AssertCursorIDs(cursor.get(), {2});
  }
}

TEST_F(DocumentDBTest, ComplexQueryTest) {
  DocumentDBOptions options;
  DocumentDB::IndexDescriptor priority_index;
  priority_index.description = Parse("{'priority': 1}");
  priority_index.name = "priority";
  DocumentDB::IndexDescriptor job_name_index;
  job_name_index.description = Parse("{'job_name': 1}");
  job_name_index.name = "job_name";
  DocumentDB::IndexDescriptor progress_index;
  progress_index.description = Parse("{'progress': 1}");
  progress_index.name = "progress";

  ASSERT_OK(DocumentDB::Open(options, dbname_, {}, &db_));
  CreateIndexes({priority_index, progress_index});
  delete priority_index.description;
  delete progress_index.description;

  std::vector<std::string> json_objects = {
      "{'_id': 1, 'job_name': 'play', 'priority': 10, 'progress': 14.2}",
      "{'_id': 2, 'job_name': 'white', 'priority': 2, 'progress': 45.1}",
      "{'_id': 3, 'job_name': 'straw', 'priority': 5, 'progress': 83.2}",
      "{'_id': 4, 'job_name': 'temporary', 'priority': 3, 'progress': 14.9}",
      "{'_id': 5, 'job_name': 'white', 'priority': 4, 'progress': 44.2}",
      "{'_id': 6, 'job_name': 'tea', 'priority': 1, 'progress': 12.4}",
      "{'_id': 7, 'job_name': 'delete', 'priority': 2, 'progress': 77.54}",
      "{'_id': 8, 'job_name': 'rock', 'priority': 3, 'progress': 93.24}",
      "{'_id': 9, 'job_name': 'steady', 'priority': 3, 'progress': 9.1}",
      "{'_id': 10, 'job_name': 'white', 'priority': 1, 'progress': 61.4}",
      "{'_id': 11, 'job_name': 'who', 'priority': 4, 'progress': 39.41}",
      "{'_id': 12, 'job_name': 'who', 'priority': -1, 'progress': 39.42}",
      "{'_id': 13, 'job_name': 'who', 'priority': -2, 'progress': 39.42}", };

  // add index on the fly!
  CreateIndexes({job_name_index});
  delete job_name_index.description;

  for (auto& json : json_objects) {
    std::unique_ptr<JSONDocument> document(Parse(json));
    ASSERT_TRUE(document != nullptr);
    ASSERT_OK(db_->Insert(WriteOptions(), *document));
  }

  // 2 < priority < 4 AND progress > 10.0, index priority
  {
    std::unique_ptr<JSONDocument> query(Parse(
        "[{'$filter': {'priority': {'$lt': 4, '$gt': 2}, 'progress': {'$gt': "
        "10.0}, '$index': 'priority'}}]"));
    std::unique_ptr<Cursor> cursor(db_->Query(ReadOptions(), *query));
    AssertCursorIDs(cursor.get(), {4, 8});
  }

  // -1 <= priority <= 1, index priority
  {
    std::unique_ptr<JSONDocument> query(Parse(
        "[{'$filter': {'priority': {'$lte': 1, '$gte': -1},"
        " '$index': 'priority'}}]"));
    std::unique_ptr<Cursor> cursor(db_->Query(ReadOptions(), *query));
    AssertCursorIDs(cursor.get(), {6, 10, 12});
  }

  // 2 < priority < 4 AND progress > 10.0, index progress
  {
    std::unique_ptr<JSONDocument> query(Parse(
        "[{'$filter': {'priority': {'$lt': 4, '$gt': 2}, 'progress': {'$gt': "
        "10.0}, '$index': 'progress'}}]"));
    std::unique_ptr<Cursor> cursor(db_->Query(ReadOptions(), *query));
    AssertCursorIDs(cursor.get(), {4, 8});
  }

  // job_name == 'white' AND priority >= 2, index job_name
  {
    std::unique_ptr<JSONDocument> query(Parse(
        "[{'$filter': {'job_name': 'white', 'priority': {'$gte': "
        "2}, '$index': 'job_name'}}]"));
    std::unique_ptr<Cursor> cursor(db_->Query(ReadOptions(), *query));
    AssertCursorIDs(cursor.get(), {2, 5});
  }

  // 35.0 <= progress < 65.5, index progress
  {
    std::unique_ptr<JSONDocument> query(Parse(
        "[{'$filter': {'progress': {'$gt': 5.0, '$gte': 35.0, '$lt': 65.5}, "
        "'$index': 'progress'}}]"));
    std::unique_ptr<Cursor> cursor(db_->Query(ReadOptions(), *query));
    AssertCursorIDs(cursor.get(), {2, 5, 10, 11, 12, 13});
  }

  // 2 < priority <= 4, index priority
  {
    std::unique_ptr<JSONDocument> query(Parse(
        "[{'$filter': {'priority': {'$gt': 2, '$lt': 8, '$lte': 4}, "
        "'$index': 'priority'}}]"));
    std::unique_ptr<Cursor> cursor(db_->Query(ReadOptions(), *query));
    AssertCursorIDs(cursor.get(), {4, 5, 8, 9, 11});
  }

  // Delete all whose progress is bigger than 50%
  {
    std::unique_ptr<JSONDocument> query(
        Parse("{'progress': {'$gt': 50.0}, '$index': 'progress'}"));
    ASSERT_OK(db_->Remove(ReadOptions(), WriteOptions(), *query));
  }

  // 2 < priority < 6, index priority
  {
    std::unique_ptr<JSONDocument> query(Parse(
        "[{'$filter': {'priority': {'$gt': 2, '$lt': 6}, "
        "'$index': 'priority'}}]"));
    std::unique_ptr<Cursor> cursor(db_->Query(ReadOptions(), *query));
    AssertCursorIDs(cursor.get(), {4, 5, 9, 11});
  }

  // update set priority to 10 where job_name is 'white'
  {
    std::unique_ptr<JSONDocument> query(Parse("{'job_name': 'white'}"));
    std::unique_ptr<JSONDocument> update(Parse("{'$set': {'priority': 10}}"));
    ASSERT_OK(db_->Update(ReadOptions(), WriteOptions(), *query, *update));
  }

  // update twice: set priority to 15 where job_name is 'white'
  {
    std::unique_ptr<JSONDocument> query(Parse("{'job_name': 'white'}"));
    std::unique_ptr<JSONDocument> update(Parse("{'$set': {'priority': 10},"
                                               "'$set': {'priority': 15}}"));
    ASSERT_OK(db_->Update(ReadOptions(), WriteOptions(), *query, *update));
  }

  // update twice: set priority to 15 and
  // progress to 40 where job_name is 'white'
  {
    std::unique_ptr<JSONDocument> query(Parse("{'job_name': 'white'}"));
    std::unique_ptr<JSONDocument> update(
        Parse("{'$set': {'priority': 10, 'progress': 35},"
              "'$set': {'priority': 15, 'progress': 40}}"));
    ASSERT_OK(db_->Update(ReadOptions(), WriteOptions(), *query, *update));
  }

  // priority < 0
  {
    std::unique_ptr<JSONDocument> query(
        Parse("[{'$filter': {'priority': {'$lt': 0}, '$index': 'priority'}}]"));
    std::unique_ptr<Cursor> cursor(db_->Query(ReadOptions(), *query));
    ASSERT_OK(cursor->status());
    AssertCursorIDs(cursor.get(), {12, 13});
  }

  // -2 < priority < 0
  {
    std::unique_ptr<JSONDocument> query(
        Parse("[{'$filter': {'priority': {'$gt': -2, '$lt': 0},"
        " '$index': 'priority'}}]"));
    std::unique_ptr<Cursor> cursor(db_->Query(ReadOptions(), *query));
    ASSERT_OK(cursor->status());
    AssertCursorIDs(cursor.get(), {12});
  }

  // -2 <= priority < 0
  {
    std::unique_ptr<JSONDocument> query(
        Parse("[{'$filter': {'priority': {'$gte': -2, '$lt': 0},"
        " '$index': 'priority'}}]"));
    std::unique_ptr<Cursor> cursor(db_->Query(ReadOptions(), *query));
    ASSERT_OK(cursor->status());
    AssertCursorIDs(cursor.get(), {12, 13});
  }

  // 4 < priority
  {
    std::unique_ptr<JSONDocument> query(
        Parse("[{'$filter': {'priority': {'$gt': 4}, '$index': 'priority'}}]"));
    std::unique_ptr<Cursor> cursor(db_->Query(ReadOptions(), *query));
    ASSERT_OK(cursor->status());
    AssertCursorIDs(cursor.get(), {1, 2, 5});
  }

  Status s = db_->DropIndex("doesnt-exist");
  ASSERT_TRUE(!s.ok());
  ASSERT_OK(db_->DropIndex("priority"));
}

}  //  namespace rocksdb

int main(int argc, char** argv) {
  ::testing::InitGoogleTest(&argc, argv);
  return RUN_ALL_TESTS();
}

#else
#include <stdio.h>

int main(int argc, char** argv) {
  fprintf(stderr, "SKIPPED as DocumentDB is not supported in ROCKSDB_LITE\n");
  return 0;
}

#endif  // !ROCKSDB_LITE
