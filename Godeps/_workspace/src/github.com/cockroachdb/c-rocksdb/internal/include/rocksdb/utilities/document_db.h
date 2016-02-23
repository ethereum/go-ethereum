//  Copyright (c) 2013, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.

#pragma once
#ifndef ROCKSDB_LITE

#include <string>
#include <vector>

#include "rocksdb/utilities/stackable_db.h"
#include "rocksdb/utilities/json_document.h"
#include "rocksdb/db.h"

namespace rocksdb {

// IMPORTANT: DocumentDB is a work in progress. It is unstable and we might
// change the API without warning. Talk to RocksDB team before using this in
// production ;)

// DocumentDB is a layer on top of RocksDB that provides a very simple JSON API.
// When creating a DB, you specify a list of indexes you want to keep on your
// data. You can insert a JSON document to the DB, which is automatically
// indexed. Every document added to the DB needs to have "_id" field which is
// automatically indexed and is an unique primary key. All other indexes are
// non-unique.

// NOTE: field names in the JSON are NOT allowed to start with '$' or
// contain '.'. We don't currently enforce that rule, but will start behaving
// badly.

// Cursor is what you get as a result of executing query. To get all
// results from a query, call Next() on a Cursor while  Valid() returns true
class Cursor {
 public:
  Cursor() = default;
  virtual ~Cursor() {}

  virtual bool Valid() const = 0;
  virtual void Next() = 0;
  // Lifecycle of the returned JSONDocument is until the next Next() call
  virtual const JSONDocument& document() const = 0;
  virtual Status status() const = 0;

 private:
  // No copying allowed
  Cursor(const Cursor&);
  void operator=(const Cursor&);
};

struct DocumentDBOptions {
  int background_threads = 4;
  uint64_t memtable_size = 128 * 1024 * 1024;    // 128 MB
  uint64_t cache_size = 1 * 1024 * 1024 * 1024;  // 1 GB
};

// TODO(icanadi) Add `JSONDocument* info` parameter to all calls that can be
// used by the caller to get more information about the call execution (number
// of dropped records, number of updated records, etc.)
class DocumentDB : public StackableDB {
 public:
  struct IndexDescriptor {
    // Currently, you can only define an index on a single field. To specify an
    // index on a field X, set index description to JSON "{X: 1}"
    // Currently the value needs to be 1, which means ascending.
    // In the future, we plan to also support indexes on multiple keys, where
    // you could mix ascending sorting (1) with descending sorting indexes (-1)
    JSONDocument* description;
    std::string name;
  };

  // Open DocumentDB with specified indexes. The list of indexes has to be
  // complete, i.e. include all indexes present in the DB, except the primary
  // key index.
  // Otherwise, Open() will return an error
  static Status Open(const DocumentDBOptions& options, const std::string& name,
                     const std::vector<IndexDescriptor>& indexes,
                     DocumentDB** db, bool read_only = false);

  explicit DocumentDB(DB* db) : StackableDB(db) {}

  // Create a new index. It will stop all writes for the duration of the call.
  // All current documents in the DB are scanned and corresponding index entries
  // are created
  virtual Status CreateIndex(const WriteOptions& write_options,
                             const IndexDescriptor& index) = 0;

  // Drop an index. Client is responsible to make sure that index is not being
  // used by currently executing queries
  virtual Status DropIndex(const std::string& name) = 0;

  // Insert a document to the DB. The document needs to have a primary key "_id"
  // which can either be a string or an integer. Otherwise the write will fail
  // with InvalidArgument.
  virtual Status Insert(const WriteOptions& options,
                        const JSONDocument& document) = 0;

  // Deletes all documents matching a filter atomically
  virtual Status Remove(const ReadOptions& read_options,
                        const WriteOptions& write_options,
                        const JSONDocument& query) = 0;

  // Does this sequence of operations:
  // 1. Find all documents matching a filter
  // 2. For all documents, atomically:
  // 2.1. apply the update operators
  // 2.2. update the secondary indexes
  //
  // Currently only $set update operator is supported.
  // Syntax is: {$set: {key1: value1, key2: value2, etc...}}
  // This operator will change a document's key1 field to value1, key2 to
  // value2, etc. New values will be set even if a document didn't have an entry
  // for the specified key.
  //
  // You can not change a primary key of a document.
  //
  // Update example: Update({id: {$gt: 5}, $index: id}, {$set: {enabled: true}})
  virtual Status Update(const ReadOptions& read_options,
                        const WriteOptions& write_options,
                        const JSONDocument& filter,
                        const JSONDocument& updates) = 0;

  // query has to be an array in which every element is an operator. Currently
  // only $filter operator is supported. Syntax of $filter operator is:
  // {$filter: {key1: condition1, key2: condition2, etc.}} where conditions can
  // be either:
  // 1) a single value in which case the condition is equality condition, or
  // 2) a defined operators, like {$gt: 4}, which will match all documents that
  // have key greater than 4.
  //
  // Supported operators are:
  // 1) $gt -- greater than
  // 2) $gte -- greater than or equal
  // 3) $lt -- less than
  // 4) $lte -- less than or equal
  // If you want the filter to use an index, you need to specify it like this:
  // {$filter: {...(conditions)..., $index: index_name}}
  //
  // Example query:
  // * [{$filter: {name: John, age: {$gte: 18}, $index: age}}]
  // will return all Johns whose age is greater or equal to 18 and it will use
  // index "age" to satisfy the query.
  virtual Cursor* Query(const ReadOptions& read_options,
                        const JSONDocument& query) = 0;
};

}  // namespace rocksdb
#endif  // ROCKSDB_LITE
