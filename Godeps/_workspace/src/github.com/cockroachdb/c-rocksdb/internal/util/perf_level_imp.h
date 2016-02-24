//  Copyright (c) 2013, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.
//
#pragma once
#include "rocksdb/perf_level.h"
#include "port/port.h"

namespace rocksdb {

#if defined(IOS_CROSS_COMPILE)
extern PerfLevel perf_level;
#else
extern __thread PerfLevel perf_level;
#endif

}  // namespace rocksdb
