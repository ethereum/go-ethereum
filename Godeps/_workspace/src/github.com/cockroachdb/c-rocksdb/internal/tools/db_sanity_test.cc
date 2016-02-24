//  Copyright (c) 2013, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.

#include <cstdio>
#include <cstdlib>
#include <vector>
#include <memory>

#include "rocksdb/db.h"
#include "rocksdb/options.h"
#include "rocksdb/env.h"
#include "rocksdb/slice.h"
#include "rocksdb/status.h"
#include "rocksdb/comparator.h"
#include "rocksdb/table.h"
#include "rocksdb/slice_transform.h"
#include "rocksdb/filter_policy.h"
#include "port/port.h"
#include "util/string_util.h"

namespace rocksdb {

class SanityTest {
 public:
  explicit SanityTest(const std::string& path)
      : env_(Env::Default()), path_(path) {
    env_->CreateDirIfMissing(path);
  }
  virtual ~SanityTest() {}

  virtual std::string Name() const = 0;
  virtual Options GetOptions() const = 0;

  Status Create() {
    Options options = GetOptions();
    options.create_if_missing = true;
    std::string dbname = path_ + Name();
    DestroyDB(dbname, options);
    DB* db = nullptr;
    Status s = DB::Open(options, dbname, &db);
    std::unique_ptr<DB> db_guard(db);
    if (!s.ok()) {
      return s;
    }
    for (int i = 0; i < 1000000; ++i) {
      std::string k = "key" + ToString(i);
      std::string v = "value" + ToString(i);
      s = db->Put(WriteOptions(), Slice(k), Slice(v));
      if (!s.ok()) {
        return s;
      }
    }
    return db->Flush(FlushOptions());
  }
  Status Verify() {
    DB* db = nullptr;
    std::string dbname = path_ + Name();
    Status s = DB::Open(GetOptions(), dbname, &db);
    std::unique_ptr<DB> db_guard(db);
    if (!s.ok()) {
      return s;
    }
    for (int i = 0; i < 1000000; ++i) {
      std::string k = "key" + ToString(i);
      std::string v = "value" + ToString(i);
      std::string result;
      s = db->Get(ReadOptions(), Slice(k), &result);
      if (!s.ok()) {
        return s;
      }
      if (result != v) {
        return Status::Corruption("Unexpected value for key " + k);
      }
    }
    return Status::OK();
  }

 private:
  Env* env_;
  std::string const path_;
};

class SanityTestBasic : public SanityTest {
 public:
  explicit SanityTestBasic(const std::string& path) : SanityTest(path) {}
  virtual Options GetOptions() const override {
    Options options;
    options.create_if_missing = true;
    return options;
  }
  virtual std::string Name() const override { return "Basic"; }
};

class SanityTestSpecialComparator : public SanityTest {
 public:
  explicit SanityTestSpecialComparator(const std::string& path)
      : SanityTest(path) {
    options_.comparator = new NewComparator();
  }
  ~SanityTestSpecialComparator() { delete options_.comparator; }
  virtual Options GetOptions() const override { return options_; }
  virtual std::string Name() const override { return "SpecialComparator"; }

 private:
  class NewComparator : public Comparator {
   public:
    virtual const char* Name() const override {
      return "rocksdb.NewComparator";
    }
    virtual int Compare(const Slice& a, const Slice& b) const override {
      return BytewiseComparator()->Compare(a, b);
    }
    virtual void FindShortestSeparator(std::string* s,
                                       const Slice& l) const override {
      BytewiseComparator()->FindShortestSeparator(s, l);
    }
    virtual void FindShortSuccessor(std::string* key) const override {
      BytewiseComparator()->FindShortSuccessor(key);
    }
  };
  Options options_;
};

class SanityTestZlibCompression : public SanityTest {
 public:
  explicit SanityTestZlibCompression(const std::string& path)
      : SanityTest(path) {
    options_.compression = kZlibCompression;
  }
  virtual Options GetOptions() const override { return options_; }
  virtual std::string Name() const override { return "ZlibCompression"; }

 private:
  Options options_;
};

class SanityTestZlibCompressionVersion2 : public SanityTest {
 public:
  explicit SanityTestZlibCompressionVersion2(const std::string& path)
      : SanityTest(path) {
    options_.compression = kZlibCompression;
    BlockBasedTableOptions table_options;
#if ROCKSDB_MAJOR > 3 || (ROCKSDB_MAJOR == 3 && ROCKSDB_MINOR >= 10)
    table_options.format_version = 2;
#endif
    options_.table_factory.reset(NewBlockBasedTableFactory(table_options));
  }
  virtual Options GetOptions() const override { return options_; }
  virtual std::string Name() const override {
    return "ZlibCompressionVersion2";
  }

 private:
  Options options_;
};

class SanityTestLZ4Compression : public SanityTest {
 public:
  explicit SanityTestLZ4Compression(const std::string& path)
      : SanityTest(path) {
    options_.compression = kLZ4Compression;
  }
  virtual Options GetOptions() const override { return options_; }
  virtual std::string Name() const override { return "LZ4Compression"; }

 private:
  Options options_;
};

class SanityTestLZ4HCCompression : public SanityTest {
 public:
  explicit SanityTestLZ4HCCompression(const std::string& path)
      : SanityTest(path) {
    options_.compression = kLZ4HCCompression;
  }
  virtual Options GetOptions() const override { return options_; }
  virtual std::string Name() const override { return "LZ4HCCompression"; }

 private:
  Options options_;
};

class SanityTestZSTDCompression : public SanityTest {
 public:
  explicit SanityTestZSTDCompression(const std::string& path)
      : SanityTest(path) {
    options_.compression = kZSTDNotFinalCompression;
  }
  virtual Options GetOptions() const override { return options_; }
  virtual std::string Name() const override { return "ZSTDCompression"; }

 private:
  Options options_;
};

#ifndef ROCKSDB_LITE
class SanityTestPlainTableFactory : public SanityTest {
 public:
  explicit SanityTestPlainTableFactory(const std::string& path)
      : SanityTest(path) {
    options_.table_factory.reset(NewPlainTableFactory());
    options_.prefix_extractor.reset(NewFixedPrefixTransform(2));
    options_.allow_mmap_reads = true;
  }
  ~SanityTestPlainTableFactory() {}
  virtual Options GetOptions() const override { return options_; }
  virtual std::string Name() const override { return "PlainTable"; }

 private:
  Options options_;
};
#endif  // ROCKSDB_LITE

class SanityTestBloomFilter : public SanityTest {
 public:
  explicit SanityTestBloomFilter(const std::string& path) : SanityTest(path) {
    BlockBasedTableOptions table_options;
    table_options.filter_policy.reset(NewBloomFilterPolicy(10));
    options_.table_factory.reset(NewBlockBasedTableFactory(table_options));
  }
  ~SanityTestBloomFilter() {}
  virtual Options GetOptions() const override { return options_; }
  virtual std::string Name() const override { return "BloomFilter"; }

 private:
  Options options_;
};

namespace {
bool RunSanityTests(const std::string& command, const std::string& path) {
  std::vector<SanityTest*> sanity_tests = {
      new SanityTestBasic(path),
      new SanityTestSpecialComparator(path),
      new SanityTestZlibCompression(path),
      new SanityTestZlibCompressionVersion2(path),
      new SanityTestLZ4Compression(path),
      new SanityTestLZ4HCCompression(path),
      new SanityTestZSTDCompression(path),
#ifndef ROCKSDB_LITE
      new SanityTestPlainTableFactory(path),
#endif  // ROCKSDB_LITE
      new SanityTestBloomFilter(path)};

  if (command == "create") {
    fprintf(stderr, "Creating...\n");
  } else {
    fprintf(stderr, "Verifying...\n");
  }
  bool result = true;
  for (auto sanity_test : sanity_tests) {
    Status s;
    fprintf(stderr, "%s -- ", sanity_test->Name().c_str());
    if (command == "create") {
      s = sanity_test->Create();
    } else {
      assert(command == "verify");
      s = sanity_test->Verify();
    }
    fprintf(stderr, "%s\n", s.ToString().c_str());
    if (!s.ok()) {
      fprintf(stderr, "FAIL\n");
      result = false;
    }

    delete sanity_test;
  }
  return result;
}
}  // namespace

}  // namespace rocksdb

int main(int argc, char** argv) {
  std::string path, command;
  bool ok = (argc == 3);
  if (ok) {
    path = std::string(argv[1]);
    command = std::string(argv[2]);
    ok = (command == "create" || command == "verify");
  }
  if (!ok) {
    fprintf(stderr, "Usage: %s <path> [create|verify] \n", argv[0]);
    exit(1);
  }
  if (path.back() != '/') {
    path += "/";
  }

  bool sanity_ok = rocksdb::RunSanityTests(command, path);

  return sanity_ok ? 0 : 1;
}
