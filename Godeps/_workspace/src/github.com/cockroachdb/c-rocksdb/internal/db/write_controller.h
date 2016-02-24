//  Copyright (c) 2013, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.

#pragma once

#include <stdint.h>

#include <memory>

namespace rocksdb {

class Env;
class WriteControllerToken;

// WriteController is controlling write stalls in our write code-path. Write
// stalls happen when compaction can't keep up with write rate.
// All of the methods here (including WriteControllerToken's destructors) need
// to be called while holding DB mutex
class WriteController {
 public:
  explicit WriteController(uint64_t delayed_write_rate = 1024u * 1024u * 32u)
      : total_stopped_(0),
        total_delayed_(0),
        bytes_left_(0),
        last_refill_time_(0) {
    set_delayed_write_rate(delayed_write_rate);
  }
  ~WriteController() = default;

  // When an actor (column family) requests a stop token, all writes will be
  // stopped until the stop token is released (deleted)
  std::unique_ptr<WriteControllerToken> GetStopToken();
  // When an actor (column family) requests a delay token, total delay for all
  // writes to the DB will be controlled under the delayed write rate. Every
  // write needs to call GetDelay() with number of bytes writing to the DB,
  // which returns number of microseconds to sleep.
  std::unique_ptr<WriteControllerToken> GetDelayToken();

  // these two metods are querying the state of the WriteController
  bool IsStopped() const;
  bool NeedsDelay() const { return total_delayed_ > 0; }
  // return how many microseconds the caller needs to sleep after the call
  // num_bytes: how many number of bytes to put into the DB.
  // Prerequisite: DB mutex held.
  uint64_t GetDelay(Env* env, uint64_t num_bytes);
  void set_delayed_write_rate(uint64_t delayed_write_rate) {
    delayed_write_rate_ = delayed_write_rate;
    if (delayed_write_rate_ == 0) {
      // avoid divide 0
      delayed_write_rate_ = 1U;
    }
  }

 private:
  friend class WriteControllerToken;
  friend class StopWriteToken;
  friend class DelayWriteToken;

  int total_stopped_;
  int total_delayed_;
  uint64_t bytes_left_;
  uint64_t last_refill_time_;
  uint64_t delayed_write_rate_;
};

class WriteControllerToken {
 public:
  explicit WriteControllerToken(WriteController* controller)
      : controller_(controller) {}
  virtual ~WriteControllerToken() {}

 protected:
  WriteController* controller_;

 private:
  // no copying allowed
  WriteControllerToken(const WriteControllerToken&) = delete;
  void operator=(const WriteControllerToken&) = delete;
};

class StopWriteToken : public WriteControllerToken {
 public:
  explicit StopWriteToken(WriteController* controller)
      : WriteControllerToken(controller) {}
  virtual ~StopWriteToken();
};

class DelayWriteToken : public WriteControllerToken {
 public:
  explicit DelayWriteToken(WriteController* controller)
      : WriteControllerToken(controller) {}
  virtual ~DelayWriteToken();
};

}  // namespace rocksdb
