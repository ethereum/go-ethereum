// Copyright (c) 2013, Facebook, Inc.  All rights reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.
//
// The implementation of ThreadStatus.
//
// Note that we make get and set access to ThreadStatusData lockless.
// As a result, ThreadStatusData as a whole is not atomic.  However,
// we guarantee consistent ThreadStatusData all the time whenever
// user call GetThreadList().  This consistency guarantee is done
// by having the following constraint in the internal implementation
// of set and get order:
//
// 1. When reset any information in ThreadStatusData, always start from
//    clearing up the lower-level information first.
// 2. When setting any information in ThreadStatusData, always start from
//    setting the higher-level information.
// 3. When returning ThreadStatusData to the user, fields are fetched from
//    higher-level to lower-level.  In addition, where there's a nullptr
//    in one field, then all fields that has lower-level than that field
//    should be ignored.
//
// The high to low level information would be:
// thread_id > thread_type > db > cf > operation > state
//
// This means user might not always get full information, but whenever
// returned by the GetThreadList() is guaranteed to be consistent.
#pragma once
#include <atomic>
#include <list>
#include <memory>
#include <mutex>
#include <string>
#include <unordered_map>
#include <unordered_set>
#include <vector>

#include "rocksdb/status.h"
#include "rocksdb/thread_status.h"
#include "port/port.h"
#include "util/thread_operation.h"

namespace rocksdb {

class ColumnFamilyHandle;

// The structure that keeps constant information about a column family.
struct ConstantColumnFamilyInfo {
#if ROCKSDB_USING_THREAD_STATUS
 public:
  ConstantColumnFamilyInfo(
      const void* _db_key,
      const std::string& _db_name,
      const std::string& _cf_name) :
      db_key(_db_key), db_name(_db_name), cf_name(_cf_name) {}
  const void* db_key;
  const std::string db_name;
  const std::string cf_name;
#endif  // ROCKSDB_USING_THREAD_STATUS
};

// the internal data-structure that is used to reflect the current
// status of a thread using a set of atomic pointers.
struct ThreadStatusData {
#if ROCKSDB_USING_THREAD_STATUS
  explicit ThreadStatusData() : enable_tracking(false) {
    thread_id.store(0);
    thread_type.store(ThreadStatus::USER);
    cf_key.store(nullptr);
    operation_type.store(ThreadStatus::OP_UNKNOWN);
    op_start_time.store(0);
    state_type.store(ThreadStatus::STATE_UNKNOWN);
  }

  // A flag to indicate whether the thread tracking is enabled
  // in the current thread.  This value will be updated based on whether
  // the associated Options::enable_thread_tracking is set to true
  // in ThreadStatusUtil::SetColumnFamily().
  //
  // If set to false, then SetThreadOperation and SetThreadState
  // will be no-op.
  bool enable_tracking;

  std::atomic<uint64_t> thread_id;
  std::atomic<ThreadStatus::ThreadType> thread_type;
  std::atomic<void*> cf_key;
  std::atomic<ThreadStatus::OperationType> operation_type;
  std::atomic<uint64_t> op_start_time;
  std::atomic<ThreadStatus::OperationStage> operation_stage;
  std::atomic<uint64_t> op_properties[ThreadStatus::kNumOperationProperties];
  std::atomic<ThreadStatus::StateType> state_type;
#endif  // ROCKSDB_USING_THREAD_STATUS
};

// The class that stores and updates the status of the current thread
// using a thread-local ThreadStatusData.
//
// In most of the case, you should use ThreadStatusUtil to update
// the status of the current thread instead of using ThreadSatusUpdater
// directly.
//
// @see ThreadStatusUtil
class ThreadStatusUpdater {
 public:
  ThreadStatusUpdater() {}

  // Releases all ThreadStatusData of all active threads.
  virtual ~ThreadStatusUpdater() {}

  // Unregister the current thread.
  void UnregisterThread();

  // Reset the status of the current thread.  This includes resetting
  // ColumnFamilyInfoKey, ThreadOperation, and ThreadState.
  void ResetThreadStatus();

  // Set the id of the current thread.
  void SetThreadID(uint64_t thread_id);

  // Register the current thread for tracking.
  void RegisterThread(ThreadStatus::ThreadType ttype, uint64_t thread_id);

  // Update the column-family info of the current thread by setting
  // its thread-local pointer of ThreadStateInfo to the correct entry.
  void SetColumnFamilyInfoKey(const void* cf_key);

  // returns the column family info key.
  const void* GetColumnFamilyInfoKey();

  // Update the thread operation of the current thread.
  void SetThreadOperation(const ThreadStatus::OperationType type);

  // The start time of the current thread operation.  It is in the format
  // of micro-seconds since some fixed point in time.
  void SetOperationStartTime(const uint64_t start_time);

  // Set the "i"th property of the current operation.
  //
  // NOTE: Our practice here is to set all the thread operation properties
  //       and stage before we set thread operation, and thread operation
  //       will be set in std::memory_order_release.  This is to ensure
  //       whenever a thread operation is not OP_UNKNOWN, we will always
  //       have a consistent information on its properties.
  void SetThreadOperationProperty(
      int i, uint64_t value);

  // Increase the "i"th property of the current operation with
  // the specified delta.
  void IncreaseThreadOperationProperty(
      int i, uint64_t delta);

  // Update the thread operation stage of the current thread.
  ThreadStatus::OperationStage SetThreadOperationStage(
      const ThreadStatus::OperationStage stage);

  // Clear thread operation of the current thread.
  void ClearThreadOperation();

  // Reset all thread-operation-properties to 0.
  void ClearThreadOperationProperties();

  // Update the thread state of the current thread.
  void SetThreadState(const ThreadStatus::StateType type);

  // Clear the thread state of the current thread.
  void ClearThreadState();

  // Obtain the status of all active registered threads.
  Status GetThreadList(
      std::vector<ThreadStatus>* thread_list);

  // Create an entry in the global ColumnFamilyInfo table for the
  // specified column family.  This function should be called only
  // when the current thread does not hold db_mutex.
  void NewColumnFamilyInfo(
      const void* db_key, const std::string& db_name,
      const void* cf_key, const std::string& cf_name);

  // Erase all ConstantColumnFamilyInfo that is associated with the
  // specified db instance.  This function should be called only when
  // the current thread does not hold db_mutex.
  void EraseDatabaseInfo(const void* db_key);

  // Erase the ConstantColumnFamilyInfo that is associated with the
  // specified ColumnFamilyData.  This function should be called only
  // when the current thread does not hold db_mutex.
  void EraseColumnFamilyInfo(const void* cf_key);

  // Verifies whether the input ColumnFamilyHandles matches
  // the information stored in the current cf_info_map.
  void TEST_VerifyColumnFamilyInfoMap(
      const std::vector<ColumnFamilyHandle*>& handles,
      bool check_exist);

 protected:
#if ROCKSDB_USING_THREAD_STATUS
  // The thread-local variable for storing thread status.
  static __thread ThreadStatusData* thread_status_data_;

  // Returns the pointer to the thread status data only when the
  // thread status data is non-null and has enable_tracking == true.
  ThreadStatusData* GetLocalThreadStatus();

  // Directly returns the pointer to thread_status_data_ without
  // checking whether enabling_tracking is true of not.
  ThreadStatusData* Get() {
    return thread_status_data_;
  }

  // The mutex that protects cf_info_map and db_key_map.
  std::mutex thread_list_mutex_;

  // The current status data of all active threads.
  std::unordered_set<ThreadStatusData*> thread_data_set_;

  // A global map that keeps the column family information.  It is stored
  // globally instead of inside DB is to avoid the situation where DB is
  // closing while GetThreadList function already get the pointer to its
  // CopnstantColumnFamilyInfo.
  std::unordered_map<
      const void*, std::unique_ptr<ConstantColumnFamilyInfo>> cf_info_map_;

  // A db_key to cf_key map that allows erasing elements in cf_info_map
  // associated to the same db_key faster.
  std::unordered_map<
      const void*, std::unordered_set<const void*>> db_key_map_;

#else
  static ThreadStatusData* thread_status_data_;
#endif  // ROCKSDB_USING_THREAD_STATUS
};

}  // namespace rocksdb
