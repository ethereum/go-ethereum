//  Copyright (c) 2014, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.
#pragma once

#include <assert.h>
#include <condition_variable>
#include <mutex>
#include <string>
#include <unordered_set>
#include <unordered_map>
#include <vector>

// This is only set from db_stress.cc and for testing only.
// If non-zero, kill at various points in source code with probability 1/this
extern int rocksdb_kill_odds;

#ifdef NDEBUG
// empty in release build
#define TEST_KILL_RANDOM(rocksdb_kill_odds)
#else

namespace rocksdb {
// Kill the process with probablity 1/odds for testing.
extern void TestKillRandom(int odds, const std::string& srcfile, int srcline);

// To avoid crashing always at some frequently executed codepaths (during
// kill random test), use this factor to reduce odds
#define REDUCE_ODDS 2
#define REDUCE_ODDS2 4

#define TEST_KILL_RANDOM(rocksdb_kill_odds)                  \
  {                                                          \
    if (rocksdb_kill_odds > 0) {                             \
      TestKillRandom(rocksdb_kill_odds, __FILE__, __LINE__); \
    }                                                        \
  }
}  // namespace rocksdb
#endif

#ifdef NDEBUG
#define TEST_SYNC_POINT(x)
#define TEST_SYNC_POINT_CALLBACK(x, y)
#else

namespace rocksdb {

// This class provides facility to reproduce race conditions deterministically
// in unit tests.
// Developer could specify sync points in the codebase via TEST_SYNC_POINT.
// Each sync point represents a position in the execution stream of a thread.
// In the unit test, 'Happens After' relationship among sync points could be
// setup via SyncPoint::LoadDependency, to reproduce a desired interleave of
// threads execution.
// Refer to (DBTest,TransactionLogIteratorRace), for an exmaple use case.

class SyncPoint {
 public:
  static SyncPoint* GetInstance();

  struct Dependency {
    std::string predecessor;
    std::string successor;
  };
  // call once at the beginning of a test to setup the dependency between
  // sync points
  void LoadDependency(const std::vector<Dependency>& dependencies);

  // Set up a call back function in sync point.
  void SetCallBack(const std::string point,
                   std::function<void(void*)> callback);
  // Clear all call back functions.
  void ClearAllCallBacks();

  // enable sync point processing (disabled on startup)
  void EnableProcessing();

  // disable sync point processing
  void DisableProcessing();

  // remove the execution trace of all sync points
  void ClearTrace();

  // triggered by TEST_SYNC_POINT, blocking execution until all predecessors
  // are executed.
  // And/or call registered callback functionn, with argument `cb_arg`
  void Process(const std::string& point, void* cb_arg = nullptr);

  // TODO: it might be useful to provide a function that blocks until all
  // sync points are cleared.

 private:
  bool PredecessorsAllCleared(const std::string& point);

  // successor/predecessor map loaded from LoadDependency
  std::unordered_map<std::string, std::vector<std::string>> successors_;
  std::unordered_map<std::string, std::vector<std::string>> predecessors_;
  std::unordered_map<std::string, std::function<void(void*)> > callbacks_;

  std::mutex mutex_;
  std::condition_variable cv_;
  // sync points that have been passed through
  std::unordered_set<std::string> cleared_points_;
  bool enabled_ = false;
  int num_callbacks_running_ = 0;
};

}  // namespace rocksdb

// Use TEST_SYNC_POINT to specify sync points inside code base.
// Sync points can have happens-after depedency on other sync points,
// configured at runtime via SyncPoint::LoadDependency. This could be
// utilized to re-produce race conditions between threads.
// See TransactionLogIteratorRace in db_test.cc for an example use case.
// TEST_SYNC_POINT is no op in release build.
#define TEST_SYNC_POINT(x) rocksdb::SyncPoint::GetInstance()->Process(x)
#define TEST_SYNC_POINT_CALLBACK(x, y) \
  rocksdb::SyncPoint::GetInstance()->Process(x, y)
#endif  // NDEBUG
