//  Copyright (c) 2013, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.

#pragma once

#include <stdint.h>
#include <deque>
#include <set>
#include <vector>

namespace rocksdb {

class ColumnFamilyData;

// This class is thread-compatible. It's should only be accessed from single
// write thread (between BeginWrite() and EndWrite())
class FlushScheduler {
 public:
  FlushScheduler() = default;
  ~FlushScheduler() = default;

  void ScheduleFlush(ColumnFamilyData* cfd);
  // Returns Ref()-ed column family. Client needs to Unref()
  // REQUIRES: db mutex is held (exception is single-threaded recovery)
  ColumnFamilyData* GetNextColumnFamily();

  bool Empty();

  void Clear();

 private:
  std::deque<ColumnFamilyData*> column_families_;
#ifndef NDEBUG
  std::set<ColumnFamilyData*> column_families_set_;
#endif  // NDEBUG
};

}  // namespace rocksdb
