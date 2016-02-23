//  Copyright (c) 2014, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.

#ifndef GFLAGS
#include <cstdio>
int main() {
  fprintf(stderr, "Please install gflags to run rocksdb tools\n");
  return 1;
}
#else

#ifndef __STDC_FORMAT_MACROS
#define __STDC_FORMAT_MACROS
#endif

#include <inttypes.h>
#include <gflags/gflags.h>
#include <iostream>

#include "rocksdb/db.h"
#include "rocksdb/env.h"
#include "util/coding.h"

DEFINE_bool(anonymous, false, "Output an empty information blob.");

void usage(const char* name) {
  std::cout << "usage: " << name << " [--anonymous] <db> <dumpfile>"
            << std::endl;
}

int main(int argc, char** argv) {
  rocksdb::DB* dbptr;
  rocksdb::Options options;
  rocksdb::Status status;
  std::unique_ptr<rocksdb::WritableFile> dumpfile;
  char hostname[1024];
  int64_t timesec;
  std::string abspath;
  char json[4096];

  GFLAGS::ParseCommandLineFlags(&argc, &argv, true);

  static const char* magicstr = "ROCKDUMP";
  static const char versionstr[8] = {0, 0, 0, 0, 0, 0, 0, 1};

  if (argc != 3) {
    usage(argv[0]);
    exit(1);
  }

  rocksdb::Env* env = rocksdb::Env::Default();

  // Open the database
  options.create_if_missing = false;
  status = rocksdb::DB::OpenForReadOnly(options, argv[1], &dbptr);
  if (!status.ok()) {
    std::cerr << "Unable to open database '" << argv[1]
              << "' for reading: " << status.ToString() << std::endl;
    exit(1);
  }

  const std::unique_ptr<rocksdb::DB> db(dbptr);

  status = env->NewWritableFile(argv[2], &dumpfile, rocksdb::EnvOptions());
  if (!status.ok()) {
    std::cerr << "Unable to open dump file '" << argv[2]
              << "' for writing: " << status.ToString() << std::endl;
    exit(1);
  }

  rocksdb::Slice magicslice(magicstr, 8);
  status = dumpfile->Append(magicslice);
  if (!status.ok()) {
    std::cerr << "Append failed: " << status.ToString() << std::endl;
    exit(1);
  }

  rocksdb::Slice versionslice(versionstr, 8);
  status = dumpfile->Append(versionslice);
  if (!status.ok()) {
    std::cerr << "Append failed: " << status.ToString() << std::endl;
    exit(1);
  }

  if (FLAGS_anonymous) {
    snprintf(json, sizeof(json), "{}");
  } else {
    status = env->GetHostName(hostname, sizeof(hostname));
    status = env->GetCurrentTime(&timesec);
    status = env->GetAbsolutePath(argv[1], &abspath);
    snprintf(json, sizeof(json),
             "{ \"database-path\": \"%s\", \"hostname\": \"%s\", "
             "\"creation-time\": %" PRIi64 " }",
             abspath.c_str(), hostname, timesec);
  }

  rocksdb::Slice infoslice(json, strlen(json));
  char infosize[4];
  rocksdb::EncodeFixed32(infosize, (uint32_t)infoslice.size());
  rocksdb::Slice infosizeslice(infosize, 4);
  status = dumpfile->Append(infosizeslice);
  if (!status.ok()) {
    std::cerr << "Append failed: " << status.ToString() << std::endl;
    exit(1);
  }
  status = dumpfile->Append(infoslice);
  if (!status.ok()) {
    std::cerr << "Append failed: " << status.ToString() << std::endl;
    exit(1);
  }

  const std::unique_ptr<rocksdb::Iterator> it(
      db->NewIterator(rocksdb::ReadOptions()));
  for (it->SeekToFirst(); it->Valid(); it->Next()) {
    char keysize[4];
    rocksdb::EncodeFixed32(keysize, (uint32_t)it->key().size());
    rocksdb::Slice keysizeslice(keysize, 4);
    status = dumpfile->Append(keysizeslice);
    if (!status.ok()) {
      std::cerr << "Append failed: " << status.ToString() << std::endl;
      exit(1);
    }
    status = dumpfile->Append(it->key());
    if (!status.ok()) {
      std::cerr << "Append failed: " << status.ToString() << std::endl;
      exit(1);
    }

    char valsize[4];
    rocksdb::EncodeFixed32(valsize, (uint32_t)it->value().size());
    rocksdb::Slice valsizeslice(valsize, 4);
    status = dumpfile->Append(valsizeslice);
    if (!status.ok()) {
      std::cerr << "Append failed: " << status.ToString() << std::endl;
      exit(1);
    }
    status = dumpfile->Append(it->value());
    if (!status.ok()) {
      std::cerr << "Append failed: " << status.ToString() << std::endl;
      exit(1);
    }
  }
  if (!it->status().ok()) {
    std::cerr << "Database iteration failed: " << status.ToString()
              << std::endl;
    exit(1);
  }

  return 0;
}

#endif  // GFLAGS
