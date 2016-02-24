//  Copyright (c) 2013, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.
//
// Copyright (c) 2011 The LevelDB Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file. See the AUTHORS file for names of contributors.
//
// See port_example.h for documentation for the following types/functions.

#ifndef STORAGE_LEVELDB_PORT_DIRENT_H_
#define STORAGE_LEVELDB_PORT_DIRENT_H_

#ifdef ROCKSDB_PLATFORM_POSIX
#include <dirent.h>
#include <sys/types.h>
#elif defined(OS_WIN)

namespace rocksdb {
namespace port {

struct dirent {
  char d_name[_MAX_PATH]; /* filename */
};

struct DIR;

DIR* opendir(const char* name);

dirent* readdir(DIR* dirp);

int closedir(DIR* dirp);

}  // namespace port

using port::dirent;
using port::DIR;
using port::opendir;
using port::readdir;
using port::closedir;

}  // namespace rocksdb

#endif  // OS_WIN

#endif  // STORAGE_LEVELDB_PORT_DIRENT_H_
