// Copyright (c) 2013, Facebook, Inc.  All rights reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.

#include <memory>

#include "rocksdb/env.h"
#include "rocksdb/merge_operator.h"
#include "rocksdb/slice.h"
#include "util/coding.h"
#include "utilities/merge_operators.h"

using namespace rocksdb;

namespace { // anonymous namespace

// A 'model' merge operator with uint64 addition semantics
// Implemented as an AssociativeMergeOperator for simplicity and example.
class UInt64AddOperator : public AssociativeMergeOperator {
 public:
  virtual bool Merge(const Slice& key,
                     const Slice* existing_value,
                     const Slice& value,
                     std::string* new_value,
                     Logger* logger) const override {
    uint64_t orig_value = 0;
    if (existing_value){
      orig_value = DecodeInteger(*existing_value, logger);
    }
    uint64_t operand = DecodeInteger(value, logger);

    assert(new_value);
    new_value->clear();
    PutFixed64(new_value, orig_value + operand);

    return true;  // Return true always since corruption will be treated as 0
  }

  virtual const char* Name() const override {
    return "UInt64AddOperator";
  }

 private:
  // Takes the string and decodes it into a uint64_t
  // On error, prints a message and returns 0
  uint64_t DecodeInteger(const Slice& value, Logger* logger) const {
    uint64_t result = 0;

    if (value.size() == sizeof(uint64_t)) {
      result = DecodeFixed64(value.data());
    } else if (logger != nullptr) {
      // If value is corrupted, treat it as 0
      Log(InfoLogLevel::ERROR_LEVEL, logger,
          "uint64 value corruption, size: %" ROCKSDB_PRIszt
          " > %" ROCKSDB_PRIszt,
          value.size(), sizeof(uint64_t));
    }

    return result;
  }

};

}

namespace rocksdb {

std::shared_ptr<MergeOperator> MergeOperators::CreateUInt64AddOperator() {
  return std::make_shared<UInt64AddOperator>();
}

}
