// Copyright (c) 2015, Facebook, Inc.  All rights reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.

#ifndef ROCKSDB_LITE

#pragma once

#include <string>

#include "rocksdb/compaction_filter.h"
#include "rocksdb/slice.h"

namespace rocksdb {

class RemoveEmptyValueCompactionFilter : public CompactionFilter {
 public:
    const char* Name() const override;
    bool Filter(int level,
        const Slice& key,
        const Slice& existing_value,
        std::string* new_value,
        bool* value_changed) const override;
};
}  // namespace rocksdb
#endif  // !ROCKSDB_LITE
