//  Copyright (c) 2013, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.

#include <atomic>

#include "rocksdb/env.h"
#include "port/port.h"
#include "util/autovector.h"
#include "util/thread_local.h"
#include "util/testharness.h"
#include "util/testutil.h"

namespace rocksdb {

class ThreadLocalTest : public testing::Test {
 public:
  ThreadLocalTest() : env_(Env::Default()) {}

  Env* env_;
};

namespace {

struct Params {
  Params(port::Mutex* m, port::CondVar* c, int* u, int n,
         UnrefHandler handler = nullptr)
      : mu(m),
        cv(c),
        unref(u),
        total(n),
        started(0),
        completed(0),
        doWrite(false),
        tls1(handler),
        tls2(nullptr) {}

  port::Mutex* mu;
  port::CondVar* cv;
  int* unref;
  int total;
  int started;
  int completed;
  bool doWrite;
  ThreadLocalPtr tls1;
  ThreadLocalPtr* tls2;
};

class IDChecker : public ThreadLocalPtr {
 public:
  static uint32_t PeekId() { return Instance()->PeekId(); }
};

}  // anonymous namespace

TEST_F(ThreadLocalTest, UniqueIdTest) {
  port::Mutex mu;
  port::CondVar cv(&mu);

  ASSERT_EQ(IDChecker::PeekId(), 0u);
  // New ThreadLocal instance bumps id by 1
  {
    // Id used 0
    Params p1(&mu, &cv, nullptr, 1u);
    ASSERT_EQ(IDChecker::PeekId(), 1u);
    // Id used 1
    Params p2(&mu, &cv, nullptr, 1u);
    ASSERT_EQ(IDChecker::PeekId(), 2u);
    // Id used 2
    Params p3(&mu, &cv, nullptr, 1u);
    ASSERT_EQ(IDChecker::PeekId(), 3u);
    // Id used 3
    Params p4(&mu, &cv, nullptr, 1u);
    ASSERT_EQ(IDChecker::PeekId(), 4u);
  }
  // id 3, 2, 1, 0 are in the free queue in order
  ASSERT_EQ(IDChecker::PeekId(), 0u);

  // pick up 0
  Params p1(&mu, &cv, nullptr, 1u);
  ASSERT_EQ(IDChecker::PeekId(), 1u);
  // pick up 1
  Params* p2 = new Params(&mu, &cv, nullptr, 1u);
  ASSERT_EQ(IDChecker::PeekId(), 2u);
  // pick up 2
  Params p3(&mu, &cv, nullptr, 1u);
  ASSERT_EQ(IDChecker::PeekId(), 3u);
  // return up 1
  delete p2;
  ASSERT_EQ(IDChecker::PeekId(), 1u);
  // Now we have 3, 1 in queue
  // pick up 1
  Params p4(&mu, &cv, nullptr, 1u);
  ASSERT_EQ(IDChecker::PeekId(), 3u);
  // pick up 3
  Params p5(&mu, &cv, nullptr, 1u);
  // next new id
  ASSERT_EQ(IDChecker::PeekId(), 4u);
  // After exit, id sequence in queue:
  // 3, 1, 2, 0
}

TEST_F(ThreadLocalTest, SequentialReadWriteTest) {
  // global id list carries over 3, 1, 2, 0
  ASSERT_EQ(IDChecker::PeekId(), 0u);

  port::Mutex mu;
  port::CondVar cv(&mu);
  Params p(&mu, &cv, nullptr, 1);
  ThreadLocalPtr tls2;
  p.tls2 = &tls2;

  auto func = [](void* ptr) {
    auto& params = *static_cast<Params*>(ptr);

    ASSERT_TRUE(params.tls1.Get() == nullptr);
    params.tls1.Reset(reinterpret_cast<int*>(1));
    ASSERT_TRUE(params.tls1.Get() == reinterpret_cast<int*>(1));
    params.tls1.Reset(reinterpret_cast<int*>(2));
    ASSERT_TRUE(params.tls1.Get() == reinterpret_cast<int*>(2));

    ASSERT_TRUE(params.tls2->Get() == nullptr);
    params.tls2->Reset(reinterpret_cast<int*>(1));
    ASSERT_TRUE(params.tls2->Get() == reinterpret_cast<int*>(1));
    params.tls2->Reset(reinterpret_cast<int*>(2));
    ASSERT_TRUE(params.tls2->Get() == reinterpret_cast<int*>(2));

    params.mu->Lock();
    ++(params.completed);
    params.cv->SignalAll();
    params.mu->Unlock();
  };

  for (int iter = 0; iter < 1024; ++iter) {
    ASSERT_EQ(IDChecker::PeekId(), 1u);
    // Another new thread, read/write should not see value from previous thread
    env_->StartThread(func, static_cast<void*>(&p));
    mu.Lock();
    while (p.completed != iter + 1) {
      cv.Wait();
    }
    mu.Unlock();
    ASSERT_EQ(IDChecker::PeekId(), 1u);
  }
}

TEST_F(ThreadLocalTest, ConcurrentReadWriteTest) {
  // global id list carries over 3, 1, 2, 0
  ASSERT_EQ(IDChecker::PeekId(), 0u);

  ThreadLocalPtr tls2;
  port::Mutex mu1;
  port::CondVar cv1(&mu1);
  Params p1(&mu1, &cv1, nullptr, 16);
  p1.tls2 = &tls2;

  port::Mutex mu2;
  port::CondVar cv2(&mu2);
  Params p2(&mu2, &cv2, nullptr, 16);
  p2.doWrite = true;
  p2.tls2 = &tls2;

  auto func = [](void* ptr) {
    auto& p = *static_cast<Params*>(ptr);

    p.mu->Lock();
    int own = ++(p.started);
    p.cv->SignalAll();
    while (p.started != p.total) {
      p.cv->Wait();
    }
    p.mu->Unlock();

    // Let write threads write a different value from the read threads
    if (p.doWrite) {
      own += 8192;
    }

    ASSERT_TRUE(p.tls1.Get() == nullptr);
    ASSERT_TRUE(p.tls2->Get() == nullptr);

    auto* env = Env::Default();
    auto start = env->NowMicros();

    p.tls1.Reset(reinterpret_cast<int*>(own));
    p.tls2->Reset(reinterpret_cast<int*>(own + 1));
    // Loop for 1 second
    while (env->NowMicros() - start < 1000 * 1000) {
      for (int iter = 0; iter < 100000; ++iter) {
        ASSERT_TRUE(p.tls1.Get() == reinterpret_cast<int*>(own));
        ASSERT_TRUE(p.tls2->Get() == reinterpret_cast<int*>(own + 1));
        if (p.doWrite) {
          p.tls1.Reset(reinterpret_cast<int*>(own));
          p.tls2->Reset(reinterpret_cast<int*>(own + 1));
        }
      }
    }

    p.mu->Lock();
    ++(p.completed);
    p.cv->SignalAll();
    p.mu->Unlock();
  };

  // Initiate 2 instnaces: one keeps writing and one keeps reading.
  // The read instance should not see data from the write instance.
  // Each thread local copy of the value are also different from each
  // other.
  for (int th = 0; th < p1.total; ++th) {
    env_->StartThread(func, static_cast<void*>(&p1));
  }
  for (int th = 0; th < p2.total; ++th) {
    env_->StartThread(func, static_cast<void*>(&p2));
  }

  mu1.Lock();
  while (p1.completed != p1.total) {
    cv1.Wait();
  }
  mu1.Unlock();

  mu2.Lock();
  while (p2.completed != p2.total) {
    cv2.Wait();
  }
  mu2.Unlock();

  ASSERT_EQ(IDChecker::PeekId(), 3u);
}

TEST_F(ThreadLocalTest, Unref) {
  ASSERT_EQ(IDChecker::PeekId(), 0u);

  auto unref = [](void* ptr) {
    auto& p = *static_cast<Params*>(ptr);
    p.mu->Lock();
    ++(*p.unref);
    p.mu->Unlock();
  };

  // Case 0: no unref triggered if ThreadLocalPtr is never accessed
  auto func0 = [](void* ptr) {
    auto& p = *static_cast<Params*>(ptr);

    p.mu->Lock();
    ++(p.started);
    p.cv->SignalAll();
    while (p.started != p.total) {
      p.cv->Wait();
    }
    p.mu->Unlock();
  };

  for (int th = 1; th <= 128; th += th) {
    port::Mutex mu;
    port::CondVar cv(&mu);
    int unref_count = 0;
    Params p(&mu, &cv, &unref_count, th, unref);

    for (int i = 0; i < p.total; ++i) {
      env_->StartThread(func0, static_cast<void*>(&p));
    }
    env_->WaitForJoin();
    ASSERT_EQ(unref_count, 0);
  }

  // Case 1: unref triggered by thread exit
  auto func1 = [](void* ptr) {
    auto& p = *static_cast<Params*>(ptr);

    p.mu->Lock();
    ++(p.started);
    p.cv->SignalAll();
    while (p.started != p.total) {
      p.cv->Wait();
    }
    p.mu->Unlock();

    ASSERT_TRUE(p.tls1.Get() == nullptr);
    ASSERT_TRUE(p.tls2->Get() == nullptr);

    p.tls1.Reset(ptr);
    p.tls2->Reset(ptr);

    p.tls1.Reset(ptr);
    p.tls2->Reset(ptr);
  };

  for (int th = 1; th <= 128; th += th) {
    port::Mutex mu;
    port::CondVar cv(&mu);
    int unref_count = 0;
    ThreadLocalPtr tls2(unref);
    Params p(&mu, &cv, &unref_count, th, unref);
    p.tls2 = &tls2;

    for (int i = 0; i < p.total; ++i) {
      env_->StartThread(func1, static_cast<void*>(&p));
    }

    env_->WaitForJoin();

    // N threads x 2 ThreadLocal instance cleanup on thread exit
    ASSERT_EQ(unref_count, 2 * p.total);
  }

  // Case 2: unref triggered by ThreadLocal instance destruction
  auto func2 = [](void* ptr) {
    auto& p = *static_cast<Params*>(ptr);

    p.mu->Lock();
    ++(p.started);
    p.cv->SignalAll();
    while (p.started != p.total) {
      p.cv->Wait();
    }
    p.mu->Unlock();

    ASSERT_TRUE(p.tls1.Get() == nullptr);
    ASSERT_TRUE(p.tls2->Get() == nullptr);

    p.tls1.Reset(ptr);
    p.tls2->Reset(ptr);

    p.tls1.Reset(ptr);
    p.tls2->Reset(ptr);

    p.mu->Lock();
    ++(p.completed);
    p.cv->SignalAll();

    // Waiting for instruction to exit thread
    while (p.completed != 0) {
      p.cv->Wait();
    }
    p.mu->Unlock();
  };

  for (int th = 1; th <= 128; th += th) {
    port::Mutex mu;
    port::CondVar cv(&mu);
    int unref_count = 0;
    Params p(&mu, &cv, &unref_count, th, unref);
    p.tls2 = new ThreadLocalPtr(unref);

    for (int i = 0; i < p.total; ++i) {
      env_->StartThread(func2, static_cast<void*>(&p));
    }

    // Wait for all threads to finish using Params
    mu.Lock();
    while (p.completed != p.total) {
      cv.Wait();
    }
    mu.Unlock();

    // Now destroy one ThreadLocal instance
    delete p.tls2;
    p.tls2 = nullptr;
    // instance destroy for N threads
    ASSERT_EQ(unref_count, p.total);

    // Signal to exit
    mu.Lock();
    p.completed = 0;
    cv.SignalAll();
    mu.Unlock();
    env_->WaitForJoin();
    // additional N threads exit unref for the left instance
    ASSERT_EQ(unref_count, 2 * p.total);
  }
}

TEST_F(ThreadLocalTest, Swap) {
  ThreadLocalPtr tls;
  tls.Reset(reinterpret_cast<void*>(1));
  ASSERT_EQ(reinterpret_cast<int64_t>(tls.Swap(nullptr)), 1);
  ASSERT_TRUE(tls.Swap(reinterpret_cast<void*>(2)) == nullptr);
  ASSERT_EQ(reinterpret_cast<int64_t>(tls.Get()), 2);
  ASSERT_EQ(reinterpret_cast<int64_t>(tls.Swap(reinterpret_cast<void*>(3))), 2);
}

TEST_F(ThreadLocalTest, Scrape) {
  auto unref = [](void* ptr) {
    auto& p = *static_cast<Params*>(ptr);
    p.mu->Lock();
    ++(*p.unref);
    p.mu->Unlock();
  };

  auto func = [](void* ptr) {
    auto& p = *static_cast<Params*>(ptr);

    ASSERT_TRUE(p.tls1.Get() == nullptr);
    ASSERT_TRUE(p.tls2->Get() == nullptr);

    p.tls1.Reset(ptr);
    p.tls2->Reset(ptr);

    p.tls1.Reset(ptr);
    p.tls2->Reset(ptr);

    p.mu->Lock();
    ++(p.completed);
    p.cv->SignalAll();

    // Waiting for instruction to exit thread
    while (p.completed != 0) {
      p.cv->Wait();
    }
    p.mu->Unlock();
  };

  for (int th = 1; th <= 128; th += th) {
    port::Mutex mu;
    port::CondVar cv(&mu);
    int unref_count = 0;
    Params p(&mu, &cv, &unref_count, th, unref);
    p.tls2 = new ThreadLocalPtr(unref);

    for (int i = 0; i < p.total; ++i) {
      env_->StartThread(func, static_cast<void*>(&p));
    }

    // Wait for all threads to finish using Params
    mu.Lock();
    while (p.completed != p.total) {
      cv.Wait();
    }
    mu.Unlock();

    ASSERT_EQ(unref_count, 0);

    // Scrape all thread local data. No unref at thread
    // exit or ThreadLocalPtr destruction
    autovector<void*> ptrs;
    p.tls1.Scrape(&ptrs, nullptr);
    p.tls2->Scrape(&ptrs, nullptr);
    delete p.tls2;
    // Signal to exit
    mu.Lock();
    p.completed = 0;
    cv.SignalAll();
    mu.Unlock();
    env_->WaitForJoin();

    ASSERT_EQ(unref_count, 0);
  }
}

TEST_F(ThreadLocalTest, CompareAndSwap) {
  ThreadLocalPtr tls;
  ASSERT_TRUE(tls.Swap(reinterpret_cast<void*>(1)) == nullptr);
  void* expected = reinterpret_cast<void*>(1);
  // Swap in 2
  ASSERT_TRUE(tls.CompareAndSwap(reinterpret_cast<void*>(2), expected));
  expected = reinterpret_cast<void*>(100);
  // Fail Swap, still 2
  ASSERT_TRUE(!tls.CompareAndSwap(reinterpret_cast<void*>(2), expected));
  ASSERT_EQ(expected, reinterpret_cast<void*>(2));
  // Swap in 3
  expected = reinterpret_cast<void*>(2);
  ASSERT_TRUE(tls.CompareAndSwap(reinterpret_cast<void*>(3), expected));
  ASSERT_EQ(tls.Get(), reinterpret_cast<void*>(3));
}

}  // namespace rocksdb

int main(int argc, char** argv) {
  ::testing::InitGoogleTest(&argc, argv);
  return RUN_ALL_TESTS();
}
