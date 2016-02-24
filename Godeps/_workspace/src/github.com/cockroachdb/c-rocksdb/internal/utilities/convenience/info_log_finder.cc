//  Copyright (c) 2013, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.
//
// Copyright (c) 2012 Facebook.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

#include "db/filename.h"
#include "rocksdb/env.h"
#include "rocksdb/utilities/info_log_finder.h"

namespace rocksdb {

Status GetInfoLogList(DB* db, std::vector<std::string>* info_log_list) {
  uint64_t number = 0;
  FileType type;
  std::string path;

  if (!db) {
    return Status::InvalidArgument("DB pointer is not valid");
  }

  const Options& options = db->GetOptions();
  if (!options.db_log_dir.empty()) {
    path = options.db_log_dir;
  } else {
    path = db->GetName();
  }
  InfoLogPrefix info_log_prefix(!options.db_log_dir.empty(), db->GetName());
  auto* env = options.env;
  std::vector<std::string> file_names;
  Status s = env->GetChildren(path, &file_names);

  if (!s.ok()) {
    return s;
  }

  for (auto f : file_names) {
    if (ParseFileName(f, &number, info_log_prefix.prefix, &type) &&
        (type == kInfoLogFile)) {
      info_log_list->push_back(f);
    }
  }
  return Status::OK();
}
}  // namespace rocksdb
