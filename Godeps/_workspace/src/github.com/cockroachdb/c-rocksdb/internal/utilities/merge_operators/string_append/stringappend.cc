/**
 * A MergeOperator for rocksdb that implements string append.
 * @author Deon Nicholas (dnicholas@fb.com)
 * Copyright 2013 Facebook
 */

#include "stringappend.h"

#include <memory>
#include <assert.h>

#include "rocksdb/slice.h"
#include "rocksdb/merge_operator.h"
#include "utilities/merge_operators.h"

namespace rocksdb {

// Constructor: also specify the delimiter character.
StringAppendOperator::StringAppendOperator(char delim_char)
    : delim_(delim_char) {
}

// Implementation for the merge operation (concatenates two strings)
bool StringAppendOperator::Merge(const Slice& key,
                                 const Slice* existing_value,
                                 const Slice& value,
                                 std::string* new_value,
                                 Logger* logger) const {

  // Clear the *new_value for writing.
  assert(new_value);
  new_value->clear();

  if (!existing_value) {
    // No existing_value. Set *new_value = value
    new_value->assign(value.data(),value.size());
  } else {
    // Generic append (existing_value != null).
    // Reserve *new_value to correct size, and apply concatenation.
    new_value->reserve(existing_value->size() + 1 + value.size());
    new_value->assign(existing_value->data(),existing_value->size());
    new_value->append(1,delim_);
    new_value->append(value.data(), value.size());
  }

  return true;
}

const char* StringAppendOperator::Name() const  {
  return "StringAppendOperator";
}

std::shared_ptr<MergeOperator> MergeOperators::CreateStringAppendOperator() {
  return std::make_shared<StringAppendOperator>(',');
}

} // namespace rocksdb



