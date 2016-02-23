//  Copyright (c) 2013, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.
//
#pragma once

#ifdef FAILED
#undef FAILED
#endif

namespace rocksdb {

class LDBCommandExecuteResult {
public:
  enum State {
    EXEC_NOT_STARTED = 0, EXEC_SUCCEED = 1, EXEC_FAILED = 2,
  };

  LDBCommandExecuteResult() : state_(EXEC_NOT_STARTED), message_("") {}

  LDBCommandExecuteResult(State state, std::string& msg) :
    state_(state), message_(msg) {}

  std::string ToString() {
    std::string ret;
    switch (state_) {
    case EXEC_SUCCEED:
      break;
    case EXEC_FAILED:
      ret.append("Failed: ");
      break;
    case EXEC_NOT_STARTED:
      ret.append("Not started: ");
    }
    if (!message_.empty()) {
      ret.append(message_);
    }
    return ret;
  }

  void Reset() {
    state_ = EXEC_NOT_STARTED;
    message_ = "";
  }

  bool IsSucceed() {
    return state_ == EXEC_SUCCEED;
  }

  bool IsNotStarted() {
    return state_ == EXEC_NOT_STARTED;
  }

  bool IsFailed() {
    return state_ == EXEC_FAILED;
  }

  static LDBCommandExecuteResult Succeed(std::string msg) {
    return LDBCommandExecuteResult(EXEC_SUCCEED, msg);
  }

  static LDBCommandExecuteResult Failed(std::string msg) {
    return LDBCommandExecuteResult(EXEC_FAILED, msg);
  }

private:
  State state_;
  std::string message_;

  bool operator==(const LDBCommandExecuteResult&);
  bool operator!=(const LDBCommandExecuteResult&);
};

}
