//  Copyright (c) 2013, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.

#ifndef __STDC_FORMAT_MACROS
#define __STDC_FORMAT_MACROS
#endif

#include <inttypes.h>
#include <stdio.h>
#include <string>
#include <algorithm>
#include <vector>

#include "db/filename.h"
#include "rocksdb/options.h"
#include "rocksdb/env.h"
#include "util/db_info_dumper.h"

namespace rocksdb {

void DumpDBFileSummary(const DBOptions& options, const std::string& dbname) {
  if (options.info_log == nullptr) {
    return;
  }

  auto* env = options.env;
  uint64_t number = 0;
  FileType type = kInfoLogFile;

  std::vector<std::string> files;
  uint64_t file_num = 0;
  uint64_t file_size;
  std::string file_info, wal_info;

  Warn(options.info_log, "DB SUMMARY\n");
  // Get files in dbname dir
  if (!env->GetChildren(dbname, &files).ok()) {
    Error(options.info_log,
          "Error when reading %s dir\n", dbname.c_str());
  }
  std::sort(files.begin(), files.end());
  for (std::string file : files) {
    if (!ParseFileName(file, &number, &type)) {
      continue;
    }
    switch (type) {
      case kCurrentFile:
        Warn(options.info_log,
            "CURRENT file:  %s\n", file.c_str());
        break;
      case kIdentityFile:
        Warn(options.info_log,
            "IDENTITY file:  %s\n", file.c_str());
        break;
      case kDescriptorFile:
        env->GetFileSize(dbname + "/" + file, &file_size);
        Warn(options.info_log,
            "MANIFEST file:  %s size: %" PRIu64 " Bytes\n",
            file.c_str(), file_size);
        break;
      case kLogFile:
        env->GetFileSize(dbname + "/" + file, &file_size);
        char str[16];
        snprintf(str, sizeof(str), "%" PRIu64, file_size);
        wal_info.append(file).append(" size: ").
            append(str).append(" ; ");
        break;
      case kTableFile:
        if (++file_num < 10) {
          file_info.append(file).append(" ");
        }
        break;
      default:
        break;
    }
  }

  // Get sst files in db_path dir
  for (auto& db_path : options.db_paths) {
    if (dbname.compare(db_path.path) != 0) {
      if (!env->GetChildren(db_path.path, &files).ok()) {
        Error(options.info_log,
            "Error when reading %s dir\n",
            db_path.path.c_str());
        continue;
      }
      std::sort(files.begin(), files.end());
      for (std::string file : files) {
        if (ParseFileName(file, &number, &type)) {
          if (type == kTableFile && ++file_num < 10) {
            file_info.append(file).append(" ");
          }
        }
      }
    }
    Warn(options.info_log,
        "SST files in %s dir, Total Num: %" PRIu64 ", files: %s\n",
        db_path.path.c_str(), file_num, file_info.c_str());
    file_num = 0;
    file_info.clear();
  }

  // Get wal file in wal_dir
  if (dbname.compare(options.wal_dir) != 0) {
    if (!env->GetChildren(options.wal_dir, &files).ok()) {
      Error(options.info_log,
          "Error when reading %s dir\n",
          options.wal_dir.c_str());
      return;
    }
    wal_info.clear();
    for (std::string file : files) {
      if (ParseFileName(file, &number, &type)) {
        if (type == kLogFile) {
          env->GetFileSize(options.wal_dir + "/" + file, &file_size);
          char str[16];
          snprintf(str, sizeof(str), "%" PRIu64, file_size);
          wal_info.append(file).append(" size: ").
              append(str).append(" ; ");
        }
      }
    }
  }
  Warn(options.info_log,
      "Write Ahead Log file in %s: %s\n",
      options.wal_dir.c_str(), wal_info.c_str());
}
}  // namespace rocksdb
