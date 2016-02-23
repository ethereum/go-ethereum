//  Copyright (c) 2015, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.

#ifndef ROCKSDB_LITE

#include "utilities/transactions/transaction_db_mutex_impl.h"

#include <chrono>
#include <condition_variable>
#include <functional>
#include <mutex>

#include "rocksdb/utilities/transaction_db_mutex.h"

namespace rocksdb {

class TransactionDBMutexImpl : public TransactionDBMutex {
 public:
  TransactionDBMutexImpl() {}
  ~TransactionDBMutexImpl() {}

  Status Lock() override;

  Status TryLockFor(int64_t timeout_time) override;

  void UnLock() override { mutex_.unlock(); }

  friend class TransactionDBCondVarImpl;

 private:
  std::timed_mutex mutex_;
};

class TransactionDBCondVarImpl : public TransactionDBCondVar {
 public:
  TransactionDBCondVarImpl() {}
  ~TransactionDBCondVarImpl() {}

  Status Wait(std::shared_ptr<TransactionDBMutex> mutex) override;

  Status WaitFor(std::shared_ptr<TransactionDBMutex> mutex,
                 int64_t timeout_time) override;

  void Notify() override { cv_.notify_one(); }

  void NotifyAll() override { cv_.notify_all(); }

 private:
  std::condition_variable_any cv_;
};

std::shared_ptr<TransactionDBMutex>
TransactionDBMutexFactoryImpl::AllocateMutex() {
  return std::shared_ptr<TransactionDBMutex>(new TransactionDBMutexImpl());
}

std::shared_ptr<TransactionDBCondVar>
TransactionDBMutexFactoryImpl::AllocateCondVar() {
  return std::shared_ptr<TransactionDBCondVar>(new TransactionDBCondVarImpl());
}

Status TransactionDBMutexImpl::Lock() {
  mutex_.lock();
  return Status::OK();
}

Status TransactionDBMutexImpl::TryLockFor(int64_t timeout_time) {
  bool locked = true;

  if (timeout_time < 0) {
    // If timeout is negative, we wait indefinitely to acquire the lock
    mutex_.lock();
  } else if (timeout_time == 0) {
    locked = mutex_.try_lock();
  } else {
    // Attempt to acquire the lock unless we timeout
    auto duration = std::chrono::microseconds(timeout_time);
    locked = mutex_.try_lock_for(duration);
  }

  if (!locked) {
    // timeout acquiring mutex
    return Status::TimedOut(Status::SubCode::kMutexTimeout);
  }

  return Status::OK();
}

Status TransactionDBCondVarImpl::Wait(
    std::shared_ptr<TransactionDBMutex> mutex) {
  auto mutex_impl = reinterpret_cast<TransactionDBMutexImpl*>(mutex.get());
  cv_.wait(mutex_impl->mutex_);
  return Status::OK();
}

Status TransactionDBCondVarImpl::WaitFor(
    std::shared_ptr<TransactionDBMutex> mutex, int64_t timeout_time) {
  auto mutex_impl = reinterpret_cast<TransactionDBMutexImpl*>(mutex.get());

  if (timeout_time < 0) {
    // If timeout is negative, do not use a timeout
    cv_.wait(mutex_impl->mutex_);
  } else {
    auto duration = std::chrono::microseconds(timeout_time);
    auto cv_status = cv_.wait_for(mutex_impl->mutex_, duration);

    // Check if the wait stopped due to timing out.
    if (cv_status == std::cv_status::timeout) {
      return Status::TimedOut(Status::SubCode::kMutexTimeout);
    }
  }

  // CV was signaled, or we spuriously woke up (but didn't time out)
  return Status::OK();
}

}  // namespace rocksdb

#endif  // ROCKSDB_LITE
