//  Copyright (c) 2014, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.

#include <mutex>
#include <condition_variable>

#include "util/thread_status_updater.h"
#include "util/testharness.h"
#include "rocksdb/db.h"

#if ROCKSDB_USING_THREAD_STATUS

namespace rocksdb {

class SimulatedBackgroundTask {
 public:
  SimulatedBackgroundTask(
      const void* db_key, const std::string& db_name,
      const void* cf_key, const std::string& cf_name,
      const ThreadStatus::OperationType operation_type =
          ThreadStatus::OP_UNKNOWN,
      const ThreadStatus::StateType state_type =
          ThreadStatus::STATE_UNKNOWN)
      : db_key_(db_key), db_name_(db_name),
        cf_key_(cf_key), cf_name_(cf_name),
        operation_type_(operation_type), state_type_(state_type),
        should_run_(true), running_count_(0) {
    Env::Default()->GetThreadStatusUpdater()->NewColumnFamilyInfo(
        db_key_, db_name_, cf_key_, cf_name_);
  }

  ~SimulatedBackgroundTask() {
    Env::Default()->GetThreadStatusUpdater()->EraseDatabaseInfo(db_key_);
  }

  void Run() {
    std::unique_lock<std::mutex> l(mutex_);
    running_count_++;
    Env::Default()->GetThreadStatusUpdater()->SetColumnFamilyInfoKey(cf_key_);
    Env::Default()->GetThreadStatusUpdater()->SetThreadOperation(
        operation_type_);
    Env::Default()->GetThreadStatusUpdater()->SetThreadState(state_type_);
    while (should_run_) {
      bg_cv_.wait(l);
    }
    Env::Default()->GetThreadStatusUpdater()->ClearThreadState();
    Env::Default()->GetThreadStatusUpdater()->ClearThreadOperation();
    Env::Default()->GetThreadStatusUpdater()->SetColumnFamilyInfoKey(0);
    running_count_--;
    bg_cv_.notify_all();
  }

  void FinishAllTasks() {
    std::unique_lock<std::mutex> l(mutex_);
    should_run_ = false;
    bg_cv_.notify_all();
  }

  void WaitUntilScheduled(int job_count, Env* env) {
    while (running_count_ < job_count) {
      env->SleepForMicroseconds(1000);
    }
  }

  void WaitUntilDone() {
    std::unique_lock<std::mutex> l(mutex_);
    while (running_count_ > 0) {
      bg_cv_.wait(l);
    }
  }

  static void DoSimulatedTask(void* arg) {
    reinterpret_cast<SimulatedBackgroundTask*>(arg)->Run();
  }

 private:
  const void* db_key_;
  const std::string db_name_;
  const void* cf_key_;
  const std::string cf_name_;
  const ThreadStatus::OperationType operation_type_;
  const ThreadStatus::StateType state_type_;
  std::mutex mutex_;
  std::condition_variable bg_cv_;
  bool should_run_;
  std::atomic<int> running_count_;
};

class ThreadListTest : public testing::Test {
 public:
  ThreadListTest() {
  }
};

TEST_F(ThreadListTest, GlobalTables) {
  // verify the global tables for operations and states are properly indexed.
  for (int type = 0; type != ThreadStatus::NUM_OP_TYPES; ++type) {
    ASSERT_EQ(global_operation_table[type].type, type);
    ASSERT_EQ(global_operation_table[type].name,
              ThreadStatus::GetOperationName(
                  ThreadStatus::OperationType(type)));
  }

  for (int type = 0; type != ThreadStatus::NUM_STATE_TYPES; ++type) {
    ASSERT_EQ(global_state_table[type].type, type);
    ASSERT_EQ(global_state_table[type].name,
              ThreadStatus::GetStateName(
                  ThreadStatus::StateType(type)));
  }

  for (int stage = 0; stage != ThreadStatus::NUM_OP_STAGES; ++stage) {
    ASSERT_EQ(global_op_stage_table[stage].stage, stage);
    ASSERT_EQ(global_op_stage_table[stage].name,
              ThreadStatus::GetOperationStageName(
                  ThreadStatus::OperationStage(stage)));
  }
}

TEST_F(ThreadListTest, SimpleColumnFamilyInfoTest) {
  Env* env = Env::Default();
  const int kHighPriorityThreads = 3;
  const int kLowPriorityThreads = 5;
  const int kSimulatedHighPriThreads = kHighPriorityThreads - 1;
  const int kSimulatedLowPriThreads = kLowPriorityThreads / 3;
  env->SetBackgroundThreads(kHighPriorityThreads, Env::HIGH);
  env->SetBackgroundThreads(kLowPriorityThreads, Env::LOW);

  SimulatedBackgroundTask running_task(
      reinterpret_cast<void*>(1234), "running",
      reinterpret_cast<void*>(5678), "pikachu");

  for (int test = 0; test < kSimulatedHighPriThreads; ++test) {
    env->Schedule(&SimulatedBackgroundTask::DoSimulatedTask,
        &running_task, Env::Priority::HIGH);
  }
  for (int test = 0; test < kSimulatedLowPriThreads; ++test) {
    env->Schedule(&SimulatedBackgroundTask::DoSimulatedTask,
        &running_task, Env::Priority::LOW);
  }
  running_task.WaitUntilScheduled(
      kSimulatedHighPriThreads + kSimulatedLowPriThreads, env);

  std::vector<ThreadStatus> thread_list;

  // Verify the number of running threads in each pool.
  env->GetThreadList(&thread_list);
  int running_count[ThreadStatus::NUM_THREAD_TYPES] = {0};
  for (auto thread_status : thread_list) {
    if (thread_status.cf_name == "pikachu" &&
        thread_status.db_name == "running") {
      running_count[thread_status.thread_type]++;
    }
  }
  ASSERT_EQ(
      running_count[ThreadStatus::HIGH_PRIORITY],
      kSimulatedHighPriThreads);
  ASSERT_EQ(
      running_count[ThreadStatus::LOW_PRIORITY],
      kSimulatedLowPriThreads);
  ASSERT_EQ(
      running_count[ThreadStatus::USER], 0);

  running_task.FinishAllTasks();
  running_task.WaitUntilDone();

  // Verify none of the threads are running
  env->GetThreadList(&thread_list);

  for (int i = 0; i < ThreadStatus::NUM_THREAD_TYPES; ++i) {
    running_count[i] = 0;
  }
  for (auto thread_status : thread_list) {
    if (thread_status.cf_name == "pikachu" &&
        thread_status.db_name == "running") {
      running_count[thread_status.thread_type]++;
    }
  }

  ASSERT_EQ(
      running_count[ThreadStatus::HIGH_PRIORITY], 0);
  ASSERT_EQ(
      running_count[ThreadStatus::LOW_PRIORITY], 0);
  ASSERT_EQ(
      running_count[ThreadStatus::USER], 0);
}

namespace {
  void UpdateStatusCounts(
      const std::vector<ThreadStatus>& thread_list,
      int operation_counts[], int state_counts[]) {
    for (auto thread_status : thread_list) {
      operation_counts[thread_status.operation_type]++;
      state_counts[thread_status.state_type]++;
    }
  }

  void VerifyAndResetCounts(
      const int correct_counts[], int collected_counts[], int size) {
    for (int i = 0; i < size; ++i) {
      ASSERT_EQ(collected_counts[i], correct_counts[i]);
      collected_counts[i] = 0;
    }
  }

  void UpdateCount(
      int operation_counts[], int from_event, int to_event, int amount) {
    operation_counts[from_event] -= amount;
    operation_counts[to_event] += amount;
  }
}  // namespace

TEST_F(ThreadListTest, SimpleEventTest) {
  Env* env = Env::Default();

  // simulated tasks
  const int kFlushWriteTasks = 3;
  SimulatedBackgroundTask flush_write_task(
      reinterpret_cast<void*>(1234), "running",
      reinterpret_cast<void*>(5678), "pikachu",
      ThreadStatus::OP_FLUSH);

  const int kCompactionWriteTasks = 4;
  SimulatedBackgroundTask compaction_write_task(
      reinterpret_cast<void*>(1234), "running",
      reinterpret_cast<void*>(5678), "pikachu",
      ThreadStatus::OP_COMPACTION);

  const int kCompactionReadTasks = 5;
  SimulatedBackgroundTask compaction_read_task(
      reinterpret_cast<void*>(1234), "running",
      reinterpret_cast<void*>(5678), "pikachu",
      ThreadStatus::OP_COMPACTION);

  const int kCompactionWaitTasks = 6;
  SimulatedBackgroundTask compaction_wait_task(
      reinterpret_cast<void*>(1234), "running",
      reinterpret_cast<void*>(5678), "pikachu",
      ThreadStatus::OP_COMPACTION);

  // setup right answers
  int correct_operation_counts[ThreadStatus::NUM_OP_TYPES] = {0};
  correct_operation_counts[ThreadStatus::OP_FLUSH] =
      kFlushWriteTasks;
  correct_operation_counts[ThreadStatus::OP_COMPACTION] =
      kCompactionWriteTasks + kCompactionReadTasks + kCompactionWaitTasks;

  env->SetBackgroundThreads(
      correct_operation_counts[ThreadStatus::OP_FLUSH], Env::HIGH);
  env->SetBackgroundThreads(
      correct_operation_counts[ThreadStatus::OP_COMPACTION], Env::LOW);

  // schedule the simulated tasks
  for (int t = 0; t < kFlushWriteTasks; ++t) {
    env->Schedule(&SimulatedBackgroundTask::DoSimulatedTask,
        &flush_write_task, Env::Priority::HIGH);
  }
  flush_write_task.WaitUntilScheduled(kFlushWriteTasks, env);

  for (int t = 0; t < kCompactionWriteTasks; ++t) {
    env->Schedule(&SimulatedBackgroundTask::DoSimulatedTask,
        &compaction_write_task, Env::Priority::LOW);
  }
  compaction_write_task.WaitUntilScheduled(kCompactionWriteTasks, env);

  for (int t = 0; t < kCompactionReadTasks; ++t) {
    env->Schedule(&SimulatedBackgroundTask::DoSimulatedTask,
        &compaction_read_task, Env::Priority::LOW);
  }
  compaction_read_task.WaitUntilScheduled(kCompactionReadTasks, env);

  for (int t = 0; t < kCompactionWaitTasks; ++t) {
    env->Schedule(&SimulatedBackgroundTask::DoSimulatedTask,
        &compaction_wait_task, Env::Priority::LOW);
  }
  compaction_wait_task.WaitUntilScheduled(kCompactionWaitTasks, env);

  // verify the thread-status
  int operation_counts[ThreadStatus::NUM_OP_TYPES] = {0};
  int state_counts[ThreadStatus::NUM_STATE_TYPES] = {0};

  std::vector<ThreadStatus> thread_list;
  env->GetThreadList(&thread_list);
  UpdateStatusCounts(thread_list, operation_counts, state_counts);
  VerifyAndResetCounts(correct_operation_counts, operation_counts,
                       ThreadStatus::NUM_OP_TYPES);

  // terminate compaction-wait tasks and see if the thread-status
  // reflects this update
  compaction_wait_task.FinishAllTasks();
  compaction_wait_task.WaitUntilDone();
  UpdateCount(correct_operation_counts, ThreadStatus::OP_COMPACTION,
              ThreadStatus::OP_UNKNOWN, kCompactionWaitTasks);

  env->GetThreadList(&thread_list);
  UpdateStatusCounts(thread_list, operation_counts, state_counts);
  VerifyAndResetCounts(correct_operation_counts, operation_counts,
                       ThreadStatus::NUM_OP_TYPES);

  // terminate flush-write tasks and see if the thread-status
  // reflects this update
  flush_write_task.FinishAllTasks();
  flush_write_task.WaitUntilDone();
  UpdateCount(correct_operation_counts, ThreadStatus::OP_FLUSH,
              ThreadStatus::OP_UNKNOWN, kFlushWriteTasks);

  env->GetThreadList(&thread_list);
  UpdateStatusCounts(thread_list, operation_counts, state_counts);
  VerifyAndResetCounts(correct_operation_counts, operation_counts,
                       ThreadStatus::NUM_OP_TYPES);

  // terminate compaction-write tasks and see if the thread-status
  // reflects this update
  compaction_write_task.FinishAllTasks();
  compaction_write_task.WaitUntilDone();
  UpdateCount(correct_operation_counts, ThreadStatus::OP_COMPACTION,
              ThreadStatus::OP_UNKNOWN, kCompactionWriteTasks);

  env->GetThreadList(&thread_list);
  UpdateStatusCounts(thread_list, operation_counts, state_counts);
  VerifyAndResetCounts(correct_operation_counts, operation_counts,
                       ThreadStatus::NUM_OP_TYPES);

  // terminate compaction-write tasks and see if the thread-status
  // reflects this update
  compaction_read_task.FinishAllTasks();
  compaction_read_task.WaitUntilDone();
  UpdateCount(correct_operation_counts, ThreadStatus::OP_COMPACTION,
              ThreadStatus::OP_UNKNOWN, kCompactionReadTasks);

  env->GetThreadList(&thread_list);
  UpdateStatusCounts(thread_list, operation_counts, state_counts);
  VerifyAndResetCounts(correct_operation_counts, operation_counts,
                       ThreadStatus::NUM_OP_TYPES);
}

}  // namespace rocksdb

int main(int argc, char** argv) {
  ::testing::InitGoogleTest(&argc, argv);
  return RUN_ALL_TESTS();
}

#else

int main(int argc, char** argv) {
  ::testing::InitGoogleTest(&argc, argv);
  return 0;
}

#endif  // ROCKSDB_USING_THREAD_STATUS
