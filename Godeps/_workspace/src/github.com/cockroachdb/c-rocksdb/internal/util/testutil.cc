//  Copyright (c) 2013, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.
//
// Copyright (c) 2011 The LevelDB Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file. See the AUTHORS file for names of contributors.

#include "util/testutil.h"

#include "port/port.h"
#include "util/file_reader_writer.h"
#include "util/random.h"

namespace rocksdb {
namespace test {

Slice RandomString(Random* rnd, int len, std::string* dst) {
  dst->resize(len);
  for (int i = 0; i < len; i++) {
    (*dst)[i] = static_cast<char>(' ' + rnd->Uniform(95));   // ' ' .. '~'
  }
  return Slice(*dst);
}

extern std::string RandomHumanReadableString(Random* rnd, int len) {
  std::string ret;
  ret.resize(len);
  for (int i = 0; i < len; ++i) {
    ret[i] = static_cast<char>('a' + rnd->Uniform(26));
  }
  return ret;
}

std::string RandomKey(Random* rnd, int len) {
  // Make sure to generate a wide variety of characters so we
  // test the boundary conditions for short-key optimizations.
  static const char kTestChars[] = {
    '\0', '\1', 'a', 'b', 'c', 'd', 'e', '\xfd', '\xfe', '\xff'
  };
  std::string result;
  for (int i = 0; i < len; i++) {
    result += kTestChars[rnd->Uniform(sizeof(kTestChars))];
  }
  return result;
}


extern Slice CompressibleString(Random* rnd, double compressed_fraction,
                                int len, std::string* dst) {
  int raw = static_cast<int>(len * compressed_fraction);
  if (raw < 1) raw = 1;
  std::string raw_data;
  RandomString(rnd, raw, &raw_data);

  // Duplicate the random data until we have filled "len" bytes
  dst->clear();
  while (dst->size() < (unsigned int)len) {
    dst->append(raw_data);
  }
  dst->resize(len);
  return Slice(*dst);
}

namespace {
class Uint64ComparatorImpl : public Comparator {
 public:
  Uint64ComparatorImpl() { }

  virtual const char* Name() const override {
    return "rocksdb.Uint64Comparator";
  }

  virtual int Compare(const Slice& a, const Slice& b) const override {
    assert(a.size() == sizeof(uint64_t) && b.size() == sizeof(uint64_t));
    const uint64_t* left = reinterpret_cast<const uint64_t*>(a.data());
    const uint64_t* right = reinterpret_cast<const uint64_t*>(b.data());
    if (*left == *right) {
      return 0;
    } else if (*left < *right) {
      return -1;
    } else {
      return 1;
    }
  }

  virtual void FindShortestSeparator(std::string* start,
      const Slice& limit) const override {
    return;
  }

  virtual void FindShortSuccessor(std::string* key) const override {
    return;
  }
};
}  // namespace

static port::OnceType once = LEVELDB_ONCE_INIT;
static const Comparator* uint64comp;

static void InitModule() {
  uint64comp = new Uint64ComparatorImpl;
}

const Comparator* Uint64Comparator() {
  port::InitOnce(&once, InitModule);
  return uint64comp;
}

WritableFileWriter* GetWritableFileWriter(WritableFile* wf) {
  unique_ptr<WritableFile> file(wf);
  return new WritableFileWriter(std::move(file), EnvOptions());
}

RandomAccessFileReader* GetRandomAccessFileReader(RandomAccessFile* raf) {
  unique_ptr<RandomAccessFile> file(raf);
  return new RandomAccessFileReader(std::move(file));
}

SequentialFileReader* GetSequentialFileReader(SequentialFile* se) {
  unique_ptr<SequentialFile> file(se);
  return new SequentialFileReader(std::move(file));
}

void CorruptKeyType(InternalKey* ikey) {
  std::string keystr = ikey->Encode().ToString();
  keystr[keystr.size() - 8] = kTypeLogData;
  ikey->DecodeFrom(Slice(keystr.data(), keystr.size()));
}

}  // namespace test
}  // namespace rocksdb
