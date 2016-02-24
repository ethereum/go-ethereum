//  Copyright (c) 2013, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.
//
// Copyright (c) 2011 The LevelDB Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file. See the AUTHORS file for names of contributors.

#ifndef OS_WIN
#include <sys/ioctl.h>
#endif
#include <sys/types.h>

#include <iostream>
#include <unordered_set>
#include <atomic>
#include <list>

#ifdef OS_LINUX
#include <linux/fs.h>
#include <stdlib.h>
#include <sys/stat.h>
#include <unistd.h>
#endif

#ifdef ROCKSDB_FALLOCATE_PRESENT
#include <errno.h>
#include <fcntl.h>
#endif

#include "rocksdb/env.h"
#include "port/port.h"
#include "util/coding.h"
#include "util/log_buffer.h"
#include "util/mutexlock.h"
#include "util/string_util.h"
#include "util/testharness.h"

namespace rocksdb {

static const int kDelayMicros = 100000;

class EnvPosixTest : public testing::Test {
 private:
  port::Mutex mu_;
  std::string events_;

 public:
  Env* env_;
  EnvPosixTest() : env_(Env::Default()) { }
};

static void SetBool(void* ptr) {
  reinterpret_cast<std::atomic<bool>*>(ptr)
      ->store(true, std::memory_order_relaxed);
}

class SleepingBackgroundTask {
 public:
  explicit SleepingBackgroundTask()
      : bg_cv_(&mutex_), should_sleep_(true), sleeping_(false) {}
  void DoSleep() {
    MutexLock l(&mutex_);
    sleeping_ = true;
    while (should_sleep_) {
      bg_cv_.Wait();
    }
    sleeping_ = false;
    bg_cv_.SignalAll();
  }

  void WakeUp() {
    MutexLock l(&mutex_);
    should_sleep_ = false;
    bg_cv_.SignalAll();

    while (sleeping_) {
      bg_cv_.Wait();
    }
  }

  bool IsSleeping() {
    MutexLock l(&mutex_);
    return sleeping_;
  }

  static void DoSleepTask(void* arg) {
    reinterpret_cast<SleepingBackgroundTask*>(arg)->DoSleep();
  }

 private:
  port::Mutex mutex_;
  port::CondVar bg_cv_;  // Signalled when background work finishes
  bool should_sleep_;
  bool sleeping_;
};

TEST_F(EnvPosixTest, RunImmediately) {
  std::atomic<bool> called(false);
  env_->Schedule(&SetBool, &called);
  Env::Default()->SleepForMicroseconds(kDelayMicros);
  ASSERT_TRUE(called.load(std::memory_order_relaxed));
}

TEST_F(EnvPosixTest, UnSchedule) {
  std::atomic<bool> called(false);
  env_->SetBackgroundThreads(1, Env::LOW);

  /* Block the low priority queue */
  SleepingBackgroundTask sleeping_task, sleeping_task1;
  env_->Schedule(&SleepingBackgroundTask::DoSleepTask, &sleeping_task,
                 Env::Priority::LOW);

  /* Schedule another task */
  env_->Schedule(&SleepingBackgroundTask::DoSleepTask, &sleeping_task1,
                 Env::Priority::LOW, &sleeping_task1);

  /* Remove it with a different tag  */
  ASSERT_EQ(0, env_->UnSchedule(&called, Env::Priority::LOW));

  /* Remove it from the queue with the right tag */
  ASSERT_EQ(1, env_->UnSchedule(&sleeping_task1, Env::Priority::LOW));

  // Unblock background thread
  sleeping_task.WakeUp();

  /* Schedule another task */
  env_->Schedule(&SetBool, &called);
  for (int i = 0; i < kDelayMicros; i++) {
    if (called.load(std::memory_order_relaxed)) {
      break;
    }
    Env::Default()->SleepForMicroseconds(1);
  }
  ASSERT_TRUE(called.load(std::memory_order_relaxed));

  ASSERT_TRUE(!sleeping_task.IsSleeping() && !sleeping_task1.IsSleeping());
}

TEST_F(EnvPosixTest, RunMany) {
  std::atomic<int> last_id(0);

  struct CB {
    std::atomic<int>* last_id_ptr;  // Pointer to shared slot
    int id;                         // Order# for the execution of this callback

    CB(std::atomic<int>* p, int i) : last_id_ptr(p), id(i) {}

    static void Run(void* v) {
      CB* cb = reinterpret_cast<CB*>(v);
      int cur = cb->last_id_ptr->load(std::memory_order_relaxed);
      ASSERT_EQ(cb->id - 1, cur);
      cb->last_id_ptr->store(cb->id, std::memory_order_release);
    }
  };

  // Schedule in different order than start time
  CB cb1(&last_id, 1);
  CB cb2(&last_id, 2);
  CB cb3(&last_id, 3);
  CB cb4(&last_id, 4);
  env_->Schedule(&CB::Run, &cb1);
  env_->Schedule(&CB::Run, &cb2);
  env_->Schedule(&CB::Run, &cb3);
  env_->Schedule(&CB::Run, &cb4);

  Env::Default()->SleepForMicroseconds(kDelayMicros);
  int cur = last_id.load(std::memory_order_acquire);
  ASSERT_EQ(4, cur);
}

struct State {
  port::Mutex mu;
  int val;
  int num_running;
};

static void ThreadBody(void* arg) {
  State* s = reinterpret_cast<State*>(arg);
  s->mu.Lock();
  s->val += 1;
  s->num_running -= 1;
  s->mu.Unlock();
}

TEST_F(EnvPosixTest, StartThread) {
  State state;
  state.val = 0;
  state.num_running = 3;
  for (int i = 0; i < 3; i++) {
    env_->StartThread(&ThreadBody, &state);
  }
  while (true) {
    state.mu.Lock();
    int num = state.num_running;
    state.mu.Unlock();
    if (num == 0) {
      break;
    }
    Env::Default()->SleepForMicroseconds(kDelayMicros);
  }
  ASSERT_EQ(state.val, 3);
}

TEST_F(EnvPosixTest, TwoPools) {
  class CB {
   public:
    CB(const std::string& pool_name, int pool_size)
        : mu_(),
          num_running_(0),
          num_finished_(0),
          pool_size_(pool_size),
          pool_name_(pool_name) { }

    static void Run(void* v) {
      CB* cb = reinterpret_cast<CB*>(v);
      cb->Run();
    }

    void Run() {
      {
        MutexLock l(&mu_);
        num_running_++;
        // make sure we don't have more than pool_size_ jobs running.
        ASSERT_LE(num_running_, pool_size_.load());
      }

      // sleep for 1 sec
      Env::Default()->SleepForMicroseconds(1000000);

      {
        MutexLock l(&mu_);
        num_running_--;
        num_finished_++;
      }
    }

    int NumFinished() {
      MutexLock l(&mu_);
      return num_finished_;
    }

    void Reset(int pool_size) {
      pool_size_.store(pool_size);
      num_finished_ = 0;
    }

   private:
    port::Mutex mu_;
    int num_running_;
    int num_finished_;
    std::atomic<int> pool_size_;
    std::string pool_name_;
  };

  const int kLowPoolSize = 2;
  const int kHighPoolSize = 4;
  const int kJobs = 8;

  CB low_pool_job("low", kLowPoolSize);
  CB high_pool_job("high", kHighPoolSize);

  env_->SetBackgroundThreads(kLowPoolSize);
  env_->SetBackgroundThreads(kHighPoolSize, Env::Priority::HIGH);

  ASSERT_EQ(0U, env_->GetThreadPoolQueueLen(Env::Priority::LOW));
  ASSERT_EQ(0U, env_->GetThreadPoolQueueLen(Env::Priority::HIGH));

  // schedule same number of jobs in each pool
  for (int i = 0; i < kJobs; i++) {
    env_->Schedule(&CB::Run, &low_pool_job);
    env_->Schedule(&CB::Run, &high_pool_job, Env::Priority::HIGH);
  }
  // Wait a short while for the jobs to be dispatched.
  Env::Default()->SleepForMicroseconds(kDelayMicros);
  ASSERT_EQ((unsigned int)(kJobs - kLowPoolSize),
            env_->GetThreadPoolQueueLen());
  ASSERT_EQ((unsigned int)(kJobs - kLowPoolSize),
            env_->GetThreadPoolQueueLen(Env::Priority::LOW));
  ASSERT_EQ((unsigned int)(kJobs - kHighPoolSize),
            env_->GetThreadPoolQueueLen(Env::Priority::HIGH));

  // wait for all jobs to finish
  while (low_pool_job.NumFinished() < kJobs ||
         high_pool_job.NumFinished() < kJobs) {
    env_->SleepForMicroseconds(kDelayMicros);
  }

  ASSERT_EQ(0U, env_->GetThreadPoolQueueLen(Env::Priority::LOW));
  ASSERT_EQ(0U, env_->GetThreadPoolQueueLen(Env::Priority::HIGH));

  // call IncBackgroundThreadsIfNeeded to two pools. One increasing and
  // the other decreasing
  env_->IncBackgroundThreadsIfNeeded(kLowPoolSize - 1, Env::Priority::LOW);
  env_->IncBackgroundThreadsIfNeeded(kHighPoolSize + 1, Env::Priority::HIGH);
  high_pool_job.Reset(kHighPoolSize + 1);
  low_pool_job.Reset(kLowPoolSize);

  // schedule same number of jobs in each pool
  for (int i = 0; i < kJobs; i++) {
    env_->Schedule(&CB::Run, &low_pool_job);
    env_->Schedule(&CB::Run, &high_pool_job, Env::Priority::HIGH);
  }
  // Wait a short while for the jobs to be dispatched.
  Env::Default()->SleepForMicroseconds(kDelayMicros);
  ASSERT_EQ((unsigned int)(kJobs - kLowPoolSize),
            env_->GetThreadPoolQueueLen());
  ASSERT_EQ((unsigned int)(kJobs - kLowPoolSize),
            env_->GetThreadPoolQueueLen(Env::Priority::LOW));
  ASSERT_EQ((unsigned int)(kJobs - (kHighPoolSize + 1)),
            env_->GetThreadPoolQueueLen(Env::Priority::HIGH));

  // wait for all jobs to finish
  while (low_pool_job.NumFinished() < kJobs ||
         high_pool_job.NumFinished() < kJobs) {
    env_->SleepForMicroseconds(kDelayMicros);
  }

  env_->SetBackgroundThreads(kHighPoolSize, Env::Priority::HIGH);
}

TEST_F(EnvPosixTest, DecreaseNumBgThreads) {
  std::vector<SleepingBackgroundTask> tasks(10);

  // Set number of thread to 1 first.
  env_->SetBackgroundThreads(1, Env::Priority::HIGH);
  Env::Default()->SleepForMicroseconds(kDelayMicros);

  // Schedule 3 tasks. 0 running; Task 1, 2 waiting.
  for (size_t i = 0; i < 3; i++) {
    env_->Schedule(&SleepingBackgroundTask::DoSleepTask, &tasks[i],
                   Env::Priority::HIGH);
    Env::Default()->SleepForMicroseconds(kDelayMicros);
  }
  ASSERT_EQ(2U, env_->GetThreadPoolQueueLen(Env::Priority::HIGH));
  ASSERT_TRUE(tasks[0].IsSleeping());
  ASSERT_TRUE(!tasks[1].IsSleeping());
  ASSERT_TRUE(!tasks[2].IsSleeping());

  // Increase to 2 threads. Task 0, 1 running; 2 waiting
  env_->SetBackgroundThreads(2, Env::Priority::HIGH);
  Env::Default()->SleepForMicroseconds(kDelayMicros);
  ASSERT_EQ(1U, env_->GetThreadPoolQueueLen(Env::Priority::HIGH));
  ASSERT_TRUE(tasks[0].IsSleeping());
  ASSERT_TRUE(tasks[1].IsSleeping());
  ASSERT_TRUE(!tasks[2].IsSleeping());

  // Shrink back to 1 thread. Still task 0, 1 running, 2 waiting
  env_->SetBackgroundThreads(1, Env::Priority::HIGH);
  Env::Default()->SleepForMicroseconds(kDelayMicros);
  ASSERT_EQ(1U, env_->GetThreadPoolQueueLen(Env::Priority::HIGH));
  ASSERT_TRUE(tasks[0].IsSleeping());
  ASSERT_TRUE(tasks[1].IsSleeping());
  ASSERT_TRUE(!tasks[2].IsSleeping());

  // The last task finishes. Task 0 running, 2 waiting.
  tasks[1].WakeUp();
  Env::Default()->SleepForMicroseconds(kDelayMicros);
  ASSERT_EQ(1U, env_->GetThreadPoolQueueLen(Env::Priority::HIGH));
  ASSERT_TRUE(tasks[0].IsSleeping());
  ASSERT_TRUE(!tasks[1].IsSleeping());
  ASSERT_TRUE(!tasks[2].IsSleeping());

  // Increase to 5 threads. Task 0 and 2 running.
  env_->SetBackgroundThreads(5, Env::Priority::HIGH);
  Env::Default()->SleepForMicroseconds(kDelayMicros);
  ASSERT_EQ((unsigned int)0, env_->GetThreadPoolQueueLen(Env::Priority::HIGH));
  ASSERT_TRUE(tasks[0].IsSleeping());
  ASSERT_TRUE(tasks[2].IsSleeping());

  // Change number of threads a couple of times while there is no sufficient
  // tasks.
  env_->SetBackgroundThreads(7, Env::Priority::HIGH);
  Env::Default()->SleepForMicroseconds(kDelayMicros);
  tasks[2].WakeUp();
  ASSERT_EQ(0U, env_->GetThreadPoolQueueLen(Env::Priority::HIGH));
  env_->SetBackgroundThreads(3, Env::Priority::HIGH);
  Env::Default()->SleepForMicroseconds(kDelayMicros);
  ASSERT_EQ(0U, env_->GetThreadPoolQueueLen(Env::Priority::HIGH));
  env_->SetBackgroundThreads(4, Env::Priority::HIGH);
  Env::Default()->SleepForMicroseconds(kDelayMicros);
  ASSERT_EQ(0U, env_->GetThreadPoolQueueLen(Env::Priority::HIGH));
  env_->SetBackgroundThreads(5, Env::Priority::HIGH);
  Env::Default()->SleepForMicroseconds(kDelayMicros);
  ASSERT_EQ(0U, env_->GetThreadPoolQueueLen(Env::Priority::HIGH));
  env_->SetBackgroundThreads(4, Env::Priority::HIGH);
  Env::Default()->SleepForMicroseconds(kDelayMicros);
  ASSERT_EQ(0U, env_->GetThreadPoolQueueLen(Env::Priority::HIGH));

  Env::Default()->SleepForMicroseconds(kDelayMicros * 50);

  // Enqueue 5 more tasks. Thread pool size now is 4.
  // Task 0, 3, 4, 5 running;6, 7 waiting.
  for (size_t i = 3; i < 8; i++) {
    env_->Schedule(&SleepingBackgroundTask::DoSleepTask, &tasks[i],
                   Env::Priority::HIGH);
  }
  Env::Default()->SleepForMicroseconds(kDelayMicros);
  ASSERT_EQ(2U, env_->GetThreadPoolQueueLen(Env::Priority::HIGH));
  ASSERT_TRUE(tasks[3].IsSleeping());
  ASSERT_TRUE(tasks[4].IsSleeping());
  ASSERT_TRUE(tasks[5].IsSleeping());
  ASSERT_TRUE(!tasks[6].IsSleeping());
  ASSERT_TRUE(!tasks[7].IsSleeping());

  // Wake up task 0, 3 and 4. Task 5, 6, 7 running.
  tasks[0].WakeUp();
  tasks[3].WakeUp();
  tasks[4].WakeUp();

  Env::Default()->SleepForMicroseconds(kDelayMicros);
  ASSERT_EQ((unsigned int)0, env_->GetThreadPoolQueueLen(Env::Priority::HIGH));
  for (size_t i = 5; i < 8; i++) {
    ASSERT_TRUE(tasks[i].IsSleeping());
  }

  // Shrink back to 1 thread. Still task 5, 6, 7 running
  env_->SetBackgroundThreads(1, Env::Priority::HIGH);
  Env::Default()->SleepForMicroseconds(kDelayMicros);
  ASSERT_TRUE(tasks[5].IsSleeping());
  ASSERT_TRUE(tasks[6].IsSleeping());
  ASSERT_TRUE(tasks[7].IsSleeping());

  // Wake up task  6. Task 5, 7 running
  tasks[6].WakeUp();
  Env::Default()->SleepForMicroseconds(kDelayMicros);
  ASSERT_TRUE(tasks[5].IsSleeping());
  ASSERT_TRUE(!tasks[6].IsSleeping());
  ASSERT_TRUE(tasks[7].IsSleeping());

  // Wake up threads 7. Task 5 running
  tasks[7].WakeUp();
  Env::Default()->SleepForMicroseconds(kDelayMicros);
  ASSERT_TRUE(!tasks[7].IsSleeping());

  // Enqueue thread 8 and 9. Task 5 running; one of 8, 9 might be running.
  env_->Schedule(&SleepingBackgroundTask::DoSleepTask, &tasks[8],
                 Env::Priority::HIGH);
  env_->Schedule(&SleepingBackgroundTask::DoSleepTask, &tasks[9],
                 Env::Priority::HIGH);
  Env::Default()->SleepForMicroseconds(kDelayMicros);
  ASSERT_GT(env_->GetThreadPoolQueueLen(Env::Priority::HIGH), (unsigned int)0);
  ASSERT_TRUE(!tasks[8].IsSleeping() || !tasks[9].IsSleeping());

  // Increase to 4 threads. Task 5, 8, 9 running.
  env_->SetBackgroundThreads(4, Env::Priority::HIGH);
  Env::Default()->SleepForMicroseconds(kDelayMicros);
  ASSERT_EQ((unsigned int)0, env_->GetThreadPoolQueueLen(Env::Priority::HIGH));
  ASSERT_TRUE(tasks[8].IsSleeping());
  ASSERT_TRUE(tasks[9].IsSleeping());

  // Shrink to 1 thread
  env_->SetBackgroundThreads(1, Env::Priority::HIGH);

  // Wake up thread 9.
  tasks[9].WakeUp();
  Env::Default()->SleepForMicroseconds(kDelayMicros);
  ASSERT_TRUE(!tasks[9].IsSleeping());
  ASSERT_TRUE(tasks[8].IsSleeping());

  // Wake up thread 8
  tasks[8].WakeUp();
  Env::Default()->SleepForMicroseconds(kDelayMicros);
  ASSERT_TRUE(!tasks[8].IsSleeping());

  // Wake up the last thread
  tasks[5].WakeUp();

  Env::Default()->SleepForMicroseconds(kDelayMicros);
  ASSERT_TRUE(!tasks[5].IsSleeping());
}

#ifdef OS_LINUX
// Travis doesn't support fallocate or getting unique ID from files for whatever
// reason.
#ifndef TRAVIS

namespace {
bool IsSingleVarint(const std::string& s) {
  Slice slice(s);

  uint64_t v;
  if (!GetVarint64(&slice, &v)) {
    return false;
  }

  return slice.size() == 0;
}

bool IsUniqueIDValid(const std::string& s) {
  return !s.empty() && !IsSingleVarint(s);
}

const size_t MAX_ID_SIZE = 100;
char temp_id[MAX_ID_SIZE];


}  // namespace

// Determine whether we can use the FS_IOC_GETVERSION ioctl
// on a file in directory DIR.  Create a temporary file therein,
// try to apply the ioctl (save that result), cleanup and
// return the result.  Return true if it is supported, and
// false if anything fails.
// Note that this function "knows" that dir has just been created
// and is empty, so we create a simply-named test file: "f".
bool ioctl_support__FS_IOC_GETVERSION(const std::string& dir) {
  const std::string file = dir + "/f";
  int fd;
  do {
    fd = open(file.c_str(), O_CREAT | O_RDWR | O_TRUNC, 0644);
  } while (fd < 0 && errno == EINTR);
  long int version;
  bool ok = (fd >= 0 && ioctl(fd, FS_IOC_GETVERSION, &version) >= 0);

  close(fd);
  unlink(file.c_str());

  return ok;
}

// To ensure that Env::GetUniqueId-related tests work correctly, the files
// should be stored in regular storage like "hard disk" or "flash device",
// and not on a tmpfs file system (like /dev/shm and /tmp on some systems).
// Otherwise we cannot get the correct id.
//
// This function serves as the replacement for test::TmpDir(), which may be
// customized to be on a file system that doesn't work with GetUniqueId().

class IoctlFriendlyTmpdir {
 public:
  explicit IoctlFriendlyTmpdir() {
    char dir_buf[100];
    std::list<std::string> candidate_dir_list = {"/var/tmp", "/tmp"};

    const char *fmt = "%s/rocksdb.XXXXXX";
    const char *tmp = getenv("TEST_IOCTL_FRIENDLY_TMPDIR");
    // If $TEST_IOCTL_FRIENDLY_TMPDIR/rocksdb.XXXXXX fits, use
    // $TEST_IOCTL_FRIENDLY_TMPDIR; subtract 2 for the "%s", and
    // add 1 for the trailing NUL byte.
    if (tmp && strlen(tmp) + strlen(fmt) - 2 + 1 <= sizeof dir_buf) {
      // use $TEST_IOCTL_FRIENDLY_TMPDIR value
      candidate_dir_list.push_front(tmp);
    }

    for (const std::string& d : candidate_dir_list) {
      snprintf(dir_buf, sizeof dir_buf, fmt, d.c_str());
      if (mkdtemp(dir_buf)) {
        if (ioctl_support__FS_IOC_GETVERSION(dir_buf)) {
          dir_ = dir_buf;
          return;
        } else {
          // Diagnose ioctl-related failure only if this is the
          // directory specified via that envvar.
          if (tmp == d) {
            fprintf(stderr, "TEST_IOCTL_FRIENDLY_TMPDIR-specified directory is "
                    "not suitable: %s\n", d.c_str());
          }
          rmdir(dir_buf);  // ignore failure
        }
      } else {
        // mkdtemp failed: diagnose it, but don't give up.
        fprintf(stderr, "mkdtemp(%s/...) failed: %s\n", d.c_str(),
                strerror(errno));
      }
    }

    fprintf(stderr, "failed to find an ioctl-friendly temporary directory;"
            " specify one via the TEST_IOCTL_FRIENDLY_TMPDIR envvar\n");
    std::abort();
  }

  ~IoctlFriendlyTmpdir() {
    rmdir(dir_.c_str());
  }
  const std::string& name() {
    return dir_;
  }

 private:
  std::string dir_;
};


// Only works in linux platforms
TEST_F(EnvPosixTest, RandomAccessUniqueID) {
  // Create file.
  const EnvOptions soptions;
  IoctlFriendlyTmpdir ift;
  std::string fname = ift.name() + "/testfile";
  unique_ptr<WritableFile> wfile;
  ASSERT_OK(env_->NewWritableFile(fname, &wfile, soptions));

  unique_ptr<RandomAccessFile> file;

  // Get Unique ID
  ASSERT_OK(env_->NewRandomAccessFile(fname, &file, soptions));
  size_t id_size = file->GetUniqueId(temp_id, MAX_ID_SIZE);
  ASSERT_TRUE(id_size > 0);
  std::string unique_id1(temp_id, id_size);
  ASSERT_TRUE(IsUniqueIDValid(unique_id1));

  // Get Unique ID again
  ASSERT_OK(env_->NewRandomAccessFile(fname, &file, soptions));
  id_size = file->GetUniqueId(temp_id, MAX_ID_SIZE);
  ASSERT_TRUE(id_size > 0);
  std::string unique_id2(temp_id, id_size);
  ASSERT_TRUE(IsUniqueIDValid(unique_id2));

  // Get Unique ID again after waiting some time.
  env_->SleepForMicroseconds(1000000);
  ASSERT_OK(env_->NewRandomAccessFile(fname, &file, soptions));
  id_size = file->GetUniqueId(temp_id, MAX_ID_SIZE);
  ASSERT_TRUE(id_size > 0);
  std::string unique_id3(temp_id, id_size);
  ASSERT_TRUE(IsUniqueIDValid(unique_id3));

  // Check IDs are the same.
  ASSERT_EQ(unique_id1, unique_id2);
  ASSERT_EQ(unique_id2, unique_id3);

  // Delete the file
  env_->DeleteFile(fname);
}

// only works in linux platforms
#ifdef ROCKSDB_FALLOCATE_PRESENT
TEST_F(EnvPosixTest, AllocateTest) {
  IoctlFriendlyTmpdir ift;
  std::string fname = ift.name() + "/preallocate_testfile";

  // Try fallocate in a file to see whether the target file system supports it.
  // Skip the test if fallocate is not supported.
  std::string fname_test_fallocate = ift.name() + "/preallocate_testfile_2";
  int fd = -1;
  do {
    fd = open(fname_test_fallocate.c_str(), O_CREAT | O_RDWR | O_TRUNC, 0644);
  } while (fd < 0 && errno == EINTR);
  ASSERT_GT(fd, 0);

  int alloc_status = fallocate(fd, 0, 0, 1);

  int err_number = 0;
  if (alloc_status != 0) {
    err_number = errno;
    fprintf(stderr, "Warning: fallocate() fails, %s\n", strerror(err_number));
  }
  close(fd);
  ASSERT_OK(env_->DeleteFile(fname_test_fallocate));
  if (alloc_status != 0 && err_number == EOPNOTSUPP) {
    // The filesystem containing the file does not support fallocate
    return;
  }

  EnvOptions soptions;
  soptions.use_mmap_writes = false;
  unique_ptr<WritableFile> wfile;
  ASSERT_OK(env_->NewWritableFile(fname, &wfile, soptions));

  // allocate 100 MB
  size_t kPreallocateSize = 100 * 1024 * 1024;
  size_t kBlockSize = 512;
  size_t kPageSize = 4096;
  std::string data(1024 * 1024, 'a');
  wfile->SetPreallocationBlockSize(kPreallocateSize);
  wfile->PrepareWrite(wfile->GetFileSize(), data.size());
  ASSERT_OK(wfile->Append(Slice(data)));
  ASSERT_OK(wfile->Flush());

  struct stat f_stat;
  stat(fname.c_str(), &f_stat);
  ASSERT_EQ((unsigned int)data.size(), f_stat.st_size);
  // verify that blocks are preallocated
  // Note here that we don't check the exact number of blocks preallocated --
  // we only require that number of allocated blocks is at least what we expect.
  // It looks like some FS give us more blocks that we asked for. That's fine.
  // It might be worth investigating further.
  ASSERT_LE((unsigned int)(kPreallocateSize / kBlockSize), f_stat.st_blocks);

  // close the file, should deallocate the blocks
  wfile.reset();

  stat(fname.c_str(), &f_stat);
  ASSERT_EQ((unsigned int)data.size(), f_stat.st_size);
  // verify that preallocated blocks were deallocated on file close
  // Because the FS might give us more blocks, we add a full page to the size
  // and expect the number of blocks to be less or equal to that.
  ASSERT_GE((f_stat.st_size + kPageSize + kBlockSize - 1) / kBlockSize, (unsigned int)f_stat.st_blocks);
}
#endif  // ROCKSDB_FALLOCATE_PRESENT

// Returns true if any of the strings in ss are the prefix of another string.
bool HasPrefix(const std::unordered_set<std::string>& ss) {
  for (const std::string& s: ss) {
    if (s.empty()) {
      return true;
    }
    for (size_t i = 1; i < s.size(); ++i) {
      if (ss.count(s.substr(0, i)) != 0) {
        return true;
      }
    }
  }
  return false;
}

// Only works in linux platforms
TEST_F(EnvPosixTest, RandomAccessUniqueIDConcurrent) {
  // Check whether a bunch of concurrently existing files have unique IDs.
  const EnvOptions soptions;

  // Create the files
  IoctlFriendlyTmpdir ift;
  std::vector<std::string> fnames;
  for (int i = 0; i < 1000; ++i) {
    fnames.push_back(ift.name() + "/" + "testfile" + ToString(i));

    // Create file.
    unique_ptr<WritableFile> wfile;
    ASSERT_OK(env_->NewWritableFile(fnames[i], &wfile, soptions));
  }

  // Collect and check whether the IDs are unique.
  std::unordered_set<std::string> ids;
  for (const std::string fname: fnames) {
    unique_ptr<RandomAccessFile> file;
    std::string unique_id;
    ASSERT_OK(env_->NewRandomAccessFile(fname, &file, soptions));
    size_t id_size = file->GetUniqueId(temp_id, MAX_ID_SIZE);
    ASSERT_TRUE(id_size > 0);
    unique_id = std::string(temp_id, id_size);
    ASSERT_TRUE(IsUniqueIDValid(unique_id));

    ASSERT_TRUE(ids.count(unique_id) == 0);
    ids.insert(unique_id);
  }

  // Delete the files
  for (const std::string fname: fnames) {
    ASSERT_OK(env_->DeleteFile(fname));
  }

  ASSERT_TRUE(!HasPrefix(ids));
}

// Only works in linux platforms
TEST_F(EnvPosixTest, RandomAccessUniqueIDDeletes) {
  const EnvOptions soptions;

  IoctlFriendlyTmpdir ift;
  std::string fname = ift.name() + "/" + "testfile";

  // Check that after file is deleted we don't get same ID again in a new file.
  std::unordered_set<std::string> ids;
  for (int i = 0; i < 1000; ++i) {
    // Create file.
    {
      unique_ptr<WritableFile> wfile;
      ASSERT_OK(env_->NewWritableFile(fname, &wfile, soptions));
    }

    // Get Unique ID
    std::string unique_id;
    {
      unique_ptr<RandomAccessFile> file;
      ASSERT_OK(env_->NewRandomAccessFile(fname, &file, soptions));
      size_t id_size = file->GetUniqueId(temp_id, MAX_ID_SIZE);
      ASSERT_TRUE(id_size > 0);
      unique_id = std::string(temp_id, id_size);
    }

    ASSERT_TRUE(IsUniqueIDValid(unique_id));
    ASSERT_TRUE(ids.count(unique_id) == 0);
    ids.insert(unique_id);

    // Delete the file
    ASSERT_OK(env_->DeleteFile(fname));
  }

  ASSERT_TRUE(!HasPrefix(ids));
}

// Only works in linux platforms
TEST_F(EnvPosixTest, InvalidateCache) {
  const EnvOptions soptions;
  std::string fname = test::TmpDir() + "/" + "testfile";

  // Create file.
  {
    unique_ptr<WritableFile> wfile;
    ASSERT_OK(env_->NewWritableFile(fname, &wfile, soptions));
    ASSERT_OK(wfile.get()->Append(Slice("Hello world")));
    ASSERT_OK(wfile.get()->InvalidateCache(0, 0));
    ASSERT_OK(wfile.get()->Close());
  }

  // Random Read
  {
    unique_ptr<RandomAccessFile> file;
    char scratch[100];
    Slice result;
    ASSERT_OK(env_->NewRandomAccessFile(fname, &file, soptions));
    ASSERT_OK(file.get()->Read(0, 11, &result, scratch));
    ASSERT_EQ(memcmp(scratch, "Hello world", 11), 0);
    ASSERT_OK(file.get()->InvalidateCache(0, 11));
    ASSERT_OK(file.get()->InvalidateCache(0, 0));
  }

  // Sequential Read
  {
    unique_ptr<SequentialFile> file;
    char scratch[100];
    Slice result;
    ASSERT_OK(env_->NewSequentialFile(fname, &file, soptions));
    ASSERT_OK(file.get()->Read(11, &result, scratch));
    ASSERT_EQ(memcmp(scratch, "Hello world", 11), 0);
    ASSERT_OK(file.get()->InvalidateCache(0, 11));
    ASSERT_OK(file.get()->InvalidateCache(0, 0));
  }
  // Delete the file
  ASSERT_OK(env_->DeleteFile(fname));
}
#endif  // not TRAVIS
#endif  // OS_LINUX

class TestLogger : public Logger {
 public:
  using Logger::Logv;
  virtual void Logv(const char* format, va_list ap) override {
    log_count++;

    char new_format[550];
    std::fill_n(new_format, sizeof(new_format), '2');
    {
      va_list backup_ap;
      va_copy(backup_ap, ap);
      int n = vsnprintf(new_format, sizeof(new_format) - 1, format, backup_ap);
      // 48 bytes for extra information + bytes allocated

// When we have n == -1 there is not a terminating zero expected
#ifdef OS_WIN
      if (n < 0) {
        char_0_count++;
      }
#endif

      if (new_format[0] == '[') {
        // "[DEBUG] "
        ASSERT_TRUE(n <= 56 + (512 - static_cast<int>(sizeof(struct timeval))));
      } else {
        ASSERT_TRUE(n <= 48 + (512 - static_cast<int>(sizeof(struct timeval))));
      }
      va_end(backup_ap);
    }

    for (size_t i = 0; i < sizeof(new_format); i++) {
      if (new_format[i] == 'x') {
        char_x_count++;
      } else if (new_format[i] == '\0') {
        char_0_count++;
      }
    }
  }
  int log_count;
  int char_x_count;
  int char_0_count;
};

TEST_F(EnvPosixTest, LogBufferTest) {
  TestLogger test_logger;
  test_logger.SetInfoLogLevel(InfoLogLevel::INFO_LEVEL);
  test_logger.log_count = 0;
  test_logger.char_x_count = 0;
  test_logger.char_0_count = 0;
  LogBuffer log_buffer(InfoLogLevel::INFO_LEVEL, &test_logger);
  LogBuffer log_buffer_debug(DEBUG_LEVEL, &test_logger);

  char bytes200[200];
  std::fill_n(bytes200, sizeof(bytes200), '1');
  bytes200[sizeof(bytes200) - 1] = '\0';
  char bytes600[600];
  std::fill_n(bytes600, sizeof(bytes600), '1');
  bytes600[sizeof(bytes600) - 1] = '\0';
  char bytes9000[9000];
  std::fill_n(bytes9000, sizeof(bytes9000), '1');
  bytes9000[sizeof(bytes9000) - 1] = '\0';

  LogToBuffer(&log_buffer, "x%sx", bytes200);
  LogToBuffer(&log_buffer, "x%sx", bytes600);
  LogToBuffer(&log_buffer, "x%sx%sx%sx", bytes200, bytes200, bytes200);
  LogToBuffer(&log_buffer, "x%sx%sx", bytes200, bytes600);
  LogToBuffer(&log_buffer, "x%sx%sx", bytes600, bytes9000);

  LogToBuffer(&log_buffer_debug, "x%sx", bytes200);
  test_logger.SetInfoLogLevel(DEBUG_LEVEL);
  LogToBuffer(&log_buffer_debug, "x%sx%sx%sx", bytes600, bytes9000, bytes200);

  ASSERT_EQ(0, test_logger.log_count);
  log_buffer.FlushBufferToLog();
  log_buffer_debug.FlushBufferToLog();
  ASSERT_EQ(6, test_logger.log_count);
  ASSERT_EQ(6, test_logger.char_0_count);
  ASSERT_EQ(10, test_logger.char_x_count);
}

class TestLogger2 : public Logger {
 public:
  explicit TestLogger2(size_t max_log_size) : max_log_size_(max_log_size) {}
  using Logger::Logv;
  virtual void Logv(const char* format, va_list ap) override {
    char new_format[2000];
    std::fill_n(new_format, sizeof(new_format), '2');
    {
      va_list backup_ap;
      va_copy(backup_ap, ap);
      int n = vsnprintf(new_format, sizeof(new_format) - 1, format, backup_ap);
      // 48 bytes for extra information + bytes allocated
      ASSERT_TRUE(
          n <= 48 + static_cast<int>(max_log_size_ - sizeof(struct timeval)));
      ASSERT_TRUE(n > static_cast<int>(max_log_size_ - sizeof(struct timeval)));
      va_end(backup_ap);
    }
  }
  size_t max_log_size_;
};

TEST_F(EnvPosixTest, LogBufferMaxSizeTest) {
  char bytes9000[9000];
  std::fill_n(bytes9000, sizeof(bytes9000), '1');
  bytes9000[sizeof(bytes9000) - 1] = '\0';

  for (size_t max_log_size = 256; max_log_size <= 1024;
       max_log_size += 1024 - 256) {
    TestLogger2 test_logger(max_log_size);
    test_logger.SetInfoLogLevel(InfoLogLevel::INFO_LEVEL);
    LogBuffer log_buffer(InfoLogLevel::INFO_LEVEL, &test_logger);
    LogToBuffer(&log_buffer, max_log_size, "%s", bytes9000);
    log_buffer.FlushBufferToLog();
  }
}

TEST_F(EnvPosixTest, Preallocation) {
  const std::string src = test::TmpDir() + "/" + "testfile";
  unique_ptr<WritableFile> srcfile;
  const EnvOptions soptions;
  ASSERT_OK(env_->NewWritableFile(src, &srcfile, soptions));
  srcfile->SetPreallocationBlockSize(1024 * 1024);

  // No writes should mean no preallocation
  size_t block_size, last_allocated_block;
  srcfile->GetPreallocationStatus(&block_size, &last_allocated_block);
  ASSERT_EQ(last_allocated_block, 0UL);

  // Small write should preallocate one block
  std::string str = "test";
  srcfile->PrepareWrite(srcfile->GetFileSize(), str.size());
  srcfile->Append(str);
  srcfile->GetPreallocationStatus(&block_size, &last_allocated_block);
  ASSERT_EQ(last_allocated_block, 1UL);

  // Write an entire preallocation block, make sure we increased by two.
  std::string buf(block_size, ' ');
  srcfile->PrepareWrite(srcfile->GetFileSize(), buf.size());
  srcfile->Append(buf);
  srcfile->GetPreallocationStatus(&block_size, &last_allocated_block);
  ASSERT_EQ(last_allocated_block, 2UL);

  // Write five more blocks at once, ensure we're where we need to be.
  buf = std::string(block_size * 5, ' ');
  srcfile->PrepareWrite(srcfile->GetFileSize(), buf.size());
  srcfile->Append(buf);
  srcfile->GetPreallocationStatus(&block_size, &last_allocated_block);
  ASSERT_EQ(last_allocated_block, 7UL);
}

// Test that all WritableFileWrapper forwards all calls to WritableFile.
TEST_F(EnvPosixTest, WritableFileWrapper) {
  class Base : public WritableFile {
   public:
    mutable int *step_;

    void inc(int x) const {
      EXPECT_EQ(x, (*step_)++);
    }

    explicit Base(int* step) : step_(step) {
      inc(0);
    }

    Status Append(const Slice& data) override { inc(1); return Status::OK(); }
    Status Close() override { inc(2); return Status::OK(); }
    Status Flush() override { inc(3); return Status::OK(); }
    Status Sync() override { inc(4); return Status::OK(); }
    Status Fsync() override { inc(5); return Status::OK(); }
    void SetIOPriority(Env::IOPriority pri) override { inc(6); }
    uint64_t GetFileSize() override { inc(7); return 0; }
    void GetPreallocationStatus(size_t* block_size,
                                size_t* last_allocated_block) override {
      inc(8);
    }
    size_t GetUniqueId(char* id, size_t max_size) const override {
      inc(9);
      return 0;
    }
    Status InvalidateCache(size_t offset, size_t length) override {
      inc(10);
      return Status::OK();
    }

   protected:
    Status Allocate(off_t offset, off_t len) override {
      inc(11);
      return Status::OK();
    }
    Status RangeSync(off_t offset, off_t nbytes) override {
      inc(12);
      return Status::OK();
    }

   public:
    ~Base() {
      inc(13);
    }
  };

  class Wrapper : public WritableFileWrapper {
   public:
    explicit Wrapper(WritableFile* target) : WritableFileWrapper(target) {}

    void CallProtectedMethods() {
      Allocate(0, 0);
      RangeSync(0, 0);
    }
  };

  int step = 0;

  {
    Base b(&step);
    Wrapper w(&b);
    w.Append(Slice());
    w.Close();
    w.Flush();
    w.Sync();
    w.Fsync();
    w.SetIOPriority(Env::IOPriority::IO_HIGH);
    w.GetFileSize();
    w.GetPreallocationStatus(nullptr, nullptr);
    w.GetUniqueId(nullptr, 0);
    w.InvalidateCache(0, 0);
    w.CallProtectedMethods();
  }

  EXPECT_EQ(14, step);
}

}  // namespace rocksdb

int main(int argc, char** argv) {
  ::testing::InitGoogleTest(&argc, argv);
  return RUN_ALL_TESTS();
}
