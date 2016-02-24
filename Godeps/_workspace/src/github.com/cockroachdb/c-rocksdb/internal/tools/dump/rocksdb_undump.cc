//  Copyright (c) 2014, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.

#include <cstring>
#include <iostream>

#include "rocksdb/db.h"
#include "rocksdb/env.h"
#include "util/coding.h"

void usage(const char *name) {
  std::cout << "usage: " << name << " <dumpfile> <rocksdb>" << std::endl;
}

int main(int argc, char **argv) {
  rocksdb::DB *dbptr;
  rocksdb::Options options;
  rocksdb::Status status;
  rocksdb::Env *env;
  std::unique_ptr<rocksdb::SequentialFile> dumpfile;
  rocksdb::Slice slice;
  char scratch8[8];

  static const char *magicstr = "ROCKDUMP";
  static const char versionstr[8] = {0, 0, 0, 0, 0, 0, 0, 1};

  if (argc != 3) {
    usage(argv[0]);
    exit(1);
  }

  env = rocksdb::Env::Default();

  status = env->NewSequentialFile(argv[1], &dumpfile, rocksdb::EnvOptions());
  if (!status.ok()) {
    std::cerr << "Unable to open dump file '" << argv[1]
              << "' for reading: " << status.ToString() << std::endl;
    exit(1);
  }

  status = dumpfile->Read(8, &slice, scratch8);
  if (!status.ok() || slice.size() != 8 ||
      memcmp(slice.data(), magicstr, 8) != 0) {
    std::cerr << "File '" << argv[1] << "' is not a recognizable dump file."
              << std::endl;
    exit(1);
  }

  status = dumpfile->Read(8, &slice, scratch8);
  if (!status.ok() || slice.size() != 8 ||
      memcmp(slice.data(), versionstr, 8) != 0) {
    std::cerr << "File '" << argv[1] << "' version not recognized."
              << std::endl;
    exit(1);
  }

  status = dumpfile->Read(4, &slice, scratch8);
  if (!status.ok() || slice.size() != 4) {
    std::cerr << "Unable to read info blob size." << std::endl;
    exit(1);
  }
  uint32_t infosize = rocksdb::DecodeFixed32(slice.data());
  status = dumpfile->Skip(infosize);
  if (!status.ok()) {
    std::cerr << "Unable to skip info blob: " << status.ToString() << std::endl;
    exit(1);
  }

  options.create_if_missing = true;
  status = rocksdb::DB::Open(options, argv[2], &dbptr);
  if (!status.ok()) {
    std::cerr << "Unable to open database '" << argv[2]
              << "' for writing: " << status.ToString() << std::endl;
    exit(1);
  }

  const std::unique_ptr<rocksdb::DB> db(dbptr);

  uint32_t last_keysize = 64;
  size_t last_valsize = 1 << 20;
  std::unique_ptr<char[]> keyscratch(new char[last_keysize]);
  std::unique_ptr<char[]> valscratch(new char[last_valsize]);

  while (1) {
    uint32_t keysize, valsize;
    rocksdb::Slice keyslice;
    rocksdb::Slice valslice;

    status = dumpfile->Read(4, &slice, scratch8);
    if (!status.ok() || slice.size() != 4) break;
    keysize = rocksdb::DecodeFixed32(slice.data());
    if (keysize > last_keysize) {
      while (keysize > last_keysize) last_keysize *= 2;
      keyscratch = std::unique_ptr<char[]>(new char[last_keysize]);
    }

    status = dumpfile->Read(keysize, &keyslice, keyscratch.get());
    if (!status.ok() || keyslice.size() != keysize) {
      std::cerr << "Key read failure: "
                << (status.ok() ? "insufficient data" : status.ToString())
                << std::endl;
      exit(1);
    }

    status = dumpfile->Read(4, &slice, scratch8);
    if (!status.ok() || slice.size() != 4) {
      std::cerr << "Unable to read value size: "
                << (status.ok() ? "insufficient data" : status.ToString())
                << std::endl;
      exit(1);
    }
    valsize = rocksdb::DecodeFixed32(slice.data());
    if (valsize > last_valsize) {
      while (valsize > last_valsize) last_valsize *= 2;
      valscratch = std::unique_ptr<char[]>(new char[last_valsize]);
    }

    status = dumpfile->Read(valsize, &valslice, valscratch.get());
    if (!status.ok() || valslice.size() != valsize) {
      std::cerr << "Unable to read value: "
                << (status.ok() ? "insufficient data" : status.ToString())
                << std::endl;
      exit(1);
    }

    status = db->Put(rocksdb::WriteOptions(), keyslice, valslice);
    if (!status.ok()) {
      fprintf(stderr, "Unable to write database entry\n");
      exit(1);
    }
  }

  return 0;
}
