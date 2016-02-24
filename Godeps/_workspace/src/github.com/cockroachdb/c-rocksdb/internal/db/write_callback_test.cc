//  Copyright (c) 2015, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.

#ifndef ROCKSDB_LITE

#include <string>

#include "db/db_impl.h"
#include "db/write_callback.h"
#include "rocksdb/db.h"
#include "rocksdb/write_batch.h"
#include "util/logging.h"
#include "util/testharness.h"

using std::string;

namespace rocksdb {

class WriteCallbackTest : public testing::Test {
 public:
  string dbname;

  WriteCallbackTest() {
    dbname = test::TmpDir() + "/write_callback_testdb";
  }
};

class WriteCallbackTestWriteCallback1 : public WriteCallback {
 public:
  bool was_called = false;

  Status Callback(DB *db) override {
    was_called = true;

    // Make sure db is a DBImpl
    DBImpl* db_impl = dynamic_cast<DBImpl*> (db);
    if (db_impl == nullptr) {
      return Status::InvalidArgument("");
    }

    return Status::OK();
  }
};

class WriteCallbackTestWriteCallback2 : public WriteCallback {
 public:
  Status Callback(DB *db) override {
    return Status::Busy();
  }
};

TEST_F(WriteCallbackTest, WriteCallBackTest) {
  Options options;
  WriteOptions write_options;
  ReadOptions read_options;
  string value;
  DB* db;
  DBImpl* db_impl;

  options.create_if_missing = true;
  Status s = DB::Open(options, dbname, &db);
  ASSERT_OK(s);

  db_impl = dynamic_cast<DBImpl*> (db);
  ASSERT_TRUE(db_impl);

  WriteBatch wb;

  wb.Put("a", "value.a");
  wb.Delete("x");

  // Test a simple Write
  s = db->Write(write_options, &wb);
  ASSERT_OK(s);

  s = db->Get(read_options, "a", &value);
  ASSERT_OK(s);
  ASSERT_EQ("value.a", value);

  // Test WriteWithCallback
  WriteCallbackTestWriteCallback1 callback1;
  WriteBatch wb2;

  wb2.Put("a", "value.a2");

  s = db_impl->WriteWithCallback(write_options, &wb2, &callback1);
  ASSERT_OK(s);
  ASSERT_TRUE(callback1.was_called);

  s = db->Get(read_options, "a", &value);
  ASSERT_OK(s);
  ASSERT_EQ("value.a2", value);

  // Test WriteWithCallback for a callback that fails
  WriteCallbackTestWriteCallback2 callback2;
  WriteBatch wb3;

  wb3.Put("a", "value.a3");

  s = db_impl->WriteWithCallback(write_options, &wb3, &callback2);
  ASSERT_NOK(s);

  s = db->Get(read_options, "a", &value);
  ASSERT_OK(s);
  ASSERT_EQ("value.a2", value);

  delete db;
  DestroyDB(dbname, options);
}

}  // namespace rocksdb

int main(int argc, char** argv) {
  ::testing::InitGoogleTest(&argc, argv);
  return RUN_ALL_TESTS();
}

#else
#include <stdio.h>

int main(int argc, char** argv) {
  fprintf(stderr,
          "SKIPPED as WriteWithCallback is not supported in ROCKSDB_LITE\n");
  return 0;
}

#endif  // !ROCKSDB_LITE
