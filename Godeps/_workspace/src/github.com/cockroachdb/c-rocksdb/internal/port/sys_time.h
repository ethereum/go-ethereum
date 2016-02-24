//  Copyright (c) 2013, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.
//
// Copyright (c) 2011 The LevelDB Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file. See the AUTHORS file for names of contributors.

// This file is a portable substitute for sys/time.h which does not exist on
// Windows

#ifndef STORAGE_LEVELDB_PORT_SYS_TIME_H_
#define STORAGE_LEVELDB_PORT_SYS_TIME_H_

#if defined(OS_WIN) && defined(_MSC_VER)

#include <time.h>

namespace rocksdb {

namespace port {

// Avoid including winsock2.h for this definition
typedef struct timeval {
  long tv_sec;
  long tv_usec;
} timeval;

void gettimeofday(struct timeval* tv, struct timezone* tz);

inline struct tm* localtime_r(const time_t* timep, struct tm* result) {
  errno_t ret = localtime_s(result, timep);
  return (ret == 0) ? result : NULL;
}
}

using port::timeval;
using port::gettimeofday;
using port::localtime_r;
}

#else
#include <time.h>
#include <sys/time.h>
#endif

#endif  // STORAGE_LEVELDB_PORT_SYS_TIME_H_
