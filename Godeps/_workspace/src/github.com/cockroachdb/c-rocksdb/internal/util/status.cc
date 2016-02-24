//  Copyright (c) 2013, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.
//
// Copyright (c) 2011 The LevelDB Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file. See the AUTHORS file for names of contributors.

#include <stdio.h>
#include "port/port.h"
#include "rocksdb/status.h"

namespace rocksdb {

const char* Status::CopyState(const char* state) {
  uint32_t size;
  memcpy(&size, state, sizeof(size));
  char* result = new char[size + 4];
  memcpy(result, state, size + 4);
  return result;
}

Status::Status(Code _code, const Slice& msg, const Slice& msg2)
    : code_(_code), subcode_(kNone) {
  assert(code_ != kOk);
  const uint32_t len1 = static_cast<uint32_t>(msg.size());
  const uint32_t len2 = static_cast<uint32_t>(msg2.size());
  const uint32_t size = len1 + (len2 ? (2 + len2) : 0);
  char* result = new char[size + 4];
  memcpy(result, &size, sizeof(size));
  memcpy(result + 4, msg.data(), len1);
  if (len2) {
    result[4 + len1] = ':';
    result[5 + len1] = ' ';
    memcpy(result + 6 + len1, msg2.data(), len2);
  }
  state_ = result;
}

std::string Status::ToString() const {
  char tmp[30];
  const char* type;
  switch (code_) {
    case kOk:
      return "OK";
    case kNotFound:
      type = "NotFound: ";
      break;
    case kCorruption:
      type = "Corruption: ";
      break;
    case kNotSupported:
      type = "Not implemented: ";
      break;
    case kInvalidArgument:
      type = "Invalid argument: ";
      break;
    case kIOError:
      type = "IO error: ";
      break;
    case kMergeInProgress:
      type = "Merge in progress: ";
      break;
    case kIncomplete:
      type = "Result incomplete: ";
      break;
    case kShutdownInProgress:
      type = "Shutdown in progress: ";
      break;
    case kTimedOut:
      type = "Operation timed out: ";
      break;
    case kAborted:
      type = "Operation aborted: ";
      break;
    case kBusy:
      type = "Resource busy: ";
      break;
    case kExpired:
      type = "Operation expired: ";
      break;
    case kTryAgain:
      type = "Operation failed. Try again.: ";
      break;
    default:
      snprintf(tmp, sizeof(tmp), "Unknown code(%d): ",
               static_cast<int>(code()));
      type = tmp;
      break;
  }
  std::string result(type);
  if (subcode_ != kNone) {
    uint32_t index = static_cast<int32_t>(subcode_);
    assert(sizeof(msgs) > index);
    result.append(msgs[index]);
  }

  if (state_ != nullptr) {
    uint32_t length;
    memcpy(&length, state_, sizeof(length));
    result.append(state_ + 4, length);
  }
  return result;
}

}  // namespace rocksdb
