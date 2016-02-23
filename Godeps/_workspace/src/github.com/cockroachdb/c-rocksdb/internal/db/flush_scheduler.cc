//  Copyright (c) 2013, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.

#include "db/flush_scheduler.h"

#include <cassert>

#include "db/column_family.h"

namespace rocksdb {

void FlushScheduler::ScheduleFlush(ColumnFamilyData* cfd) {
#ifndef NDEBUG
  assert(column_families_set_.find(cfd) == column_families_set_.end());
  column_families_set_.insert(cfd);
#endif  // NDEBUG
  cfd->Ref();
  column_families_.push_back(cfd);
}

ColumnFamilyData* FlushScheduler::GetNextColumnFamily() {
  ColumnFamilyData* cfd = nullptr;
  while (column_families_.size() > 0) {
    cfd = column_families_.front();
    column_families_.pop_front();
    if (cfd->IsDropped()) {
      if (cfd->Unref()) {
        delete cfd;
        cfd = nullptr;
      }
    } else {
      break;
    }
  }
#ifndef NDEBUG
  if (cfd != nullptr) {
    auto itr = column_families_set_.find(cfd);
    assert(itr != column_families_set_.end());
    column_families_set_.erase(itr);
  }
#endif  // NDEBUG
  return cfd;
}

bool FlushScheduler::Empty() { return column_families_.empty(); }

void FlushScheduler::Clear() {
  for (auto cfd : column_families_) {
#ifndef NDEBUG
    auto itr = column_families_set_.find(cfd);
    assert(itr != column_families_set_.end());
    column_families_set_.erase(itr);
#endif  // NDEBUG
    if (cfd->Unref()) {
      delete cfd;
    }
  }
  column_families_.clear();
}

}  // namespace rocksdb
