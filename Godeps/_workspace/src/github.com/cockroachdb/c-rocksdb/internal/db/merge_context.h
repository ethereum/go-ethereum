//  Copyright (c) 2013, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.
//
#pragma once
#include "db/dbformat.h"
#include "rocksdb/slice.h"
#include <string>
#include <deque>

namespace rocksdb {

const std::deque<std::string> empty_operand_list;

// The merge context for merging a user key.
// When doing a Get(), DB will create such a class and pass it when
// issuing Get() operation to memtables and version_set. The operands
// will be fetched from the context when issuing partial of full merge.
class MergeContext {
public:
  // Clear all the operands
  void Clear() {
    if (operand_list) {
      operand_list->clear();
    }
  }
  // Replace all operands with merge_result, which are expected to be the
  // merge result of them.
  void PushPartialMergeResult(std::string& merge_result) {
    assert (operand_list);
    operand_list->clear();
    operand_list->push_front(std::move(merge_result));
  }
  // Push a merge operand
  void PushOperand(const Slice& operand_slice) {
    Initialize();
    operand_list->push_front(operand_slice.ToString());
  }
  // return total number of operands in the list
  size_t GetNumOperands() const {
    if (!operand_list) {
      return 0;
    }
    return operand_list->size();
  }
  // Get the operand at the index.
  Slice GetOperand(int index) const {
    assert (operand_list);
    return (*operand_list)[index];
  }
  // Return all the operands.
  const std::deque<std::string>& GetOperands() const {
    if (!operand_list) {
      return empty_operand_list;
    }
    return *operand_list;
  }
private:
  void Initialize() {
    if (!operand_list) {
      operand_list.reset(new std::deque<std::string>());
    }
  }
  std::unique_ptr<std::deque<std::string>> operand_list;
};

} // namespace rocksdb
