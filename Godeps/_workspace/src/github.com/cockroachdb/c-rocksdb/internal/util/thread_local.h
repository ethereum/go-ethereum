//  Copyright (c) 2013, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.
//
// Copyright (c) 2011 The LevelDB Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file. See the AUTHORS file for names of contributors.

#pragma once

#include <atomic>
#include <memory>
#include <unordered_map>
#include <vector>

#include "util/autovector.h"
#include "port/port.h"

#ifndef ROCKSDB_SUPPORT_THREAD_LOCAL
#define ROCKSDB_SUPPORT_THREAD_LOCAL \
  !defined(OS_WIN) && !defined(OS_MACOSX) && !defined(IOS_CROSS_COMPILE)
#endif

namespace rocksdb {

// Cleanup function that will be called for a stored thread local
// pointer (if not NULL) when one of the following happens:
// (1) a thread terminates
// (2) a ThreadLocalPtr is destroyed
typedef void (*UnrefHandler)(void* ptr);

// ThreadLocalPtr stores only values of pointer type.  Different from
// the usual thread-local-storage, ThreadLocalPtr has the ability to
// distinguish data coming from different threads and different
// ThreadLocalPtr instances.  For example, if a regular thread_local
// variable A is declared in DBImpl, two DBImpl objects would share
// the same A.  However, a ThreadLocalPtr that is defined under the
// scope of DBImpl can avoid such confliction.  As a result, its memory
// usage would be O(# of threads * # of ThreadLocalPtr instances).
class ThreadLocalPtr {
 public:
  explicit ThreadLocalPtr(UnrefHandler handler = nullptr);

  ~ThreadLocalPtr();

  // Return the current pointer stored in thread local
  void* Get() const;

  // Set a new pointer value to the thread local storage.
  void Reset(void* ptr);

  // Atomically swap the supplied ptr and return the previous value
  void* Swap(void* ptr);

  // Atomically compare the stored value with expected. Set the new
  // pointer value to thread local only if the comparision is true.
  // Otherwise, expected returns the stored value.
  // Return true on success, false on failure
  bool CompareAndSwap(void* ptr, void*& expected);

  // Reset all thread local data to replacement, and return non-nullptr
  // data for all existing threads
  void Scrape(autovector<void*>* ptrs, void* const replacement);

 protected:
  struct Entry {
    Entry() : ptr(nullptr) {}
    Entry(const Entry& e) : ptr(e.ptr.load(std::memory_order_relaxed)) {}
    std::atomic<void*> ptr;
  };

  // This is the structure that is declared as "thread_local" storage.
  // The vector keep list of atomic pointer for all instances for "current"
  // thread. The vector is indexed by an Id that is unique in process and
  // associated with one ThreadLocalPtr instance. The Id is assigned by a
  // global StaticMeta singleton. So if we instantiated 3 ThreadLocalPtr
  // instances, each thread will have a ThreadData with a vector of size 3:
  //     ---------------------------------------------------
  //     |          | instance 1 | instance 2 | instnace 3 |
  //     ---------------------------------------------------
  //     | thread 1 |    void*   |    void*   |    void*   | <- ThreadData
  //     ---------------------------------------------------
  //     | thread 2 |    void*   |    void*   |    void*   | <- ThreadData
  //     ---------------------------------------------------
  //     | thread 3 |    void*   |    void*   |    void*   | <- ThreadData
  //     ---------------------------------------------------
  struct ThreadData {
    ThreadData() : entries() {}
    std::vector<Entry> entries;
    ThreadData* next;
    ThreadData* prev;
  };

  class StaticMeta {
   public:
    StaticMeta();

    // Return the next available Id
    uint32_t GetId();
    // Return the next availabe Id without claiming it
    uint32_t PeekId() const;
    // Return the given Id back to the free pool. This also triggers
    // UnrefHandler for associated pointer value (if not NULL) for all threads.
    void ReclaimId(uint32_t id);

    // Return the pointer value for the given id for the current thread.
    void* Get(uint32_t id) const;
    // Reset the pointer value for the given id for the current thread.
    // It triggers UnrefHanlder if the id has existing pointer value.
    void Reset(uint32_t id, void* ptr);
    // Atomically swap the supplied ptr and return the previous value
    void* Swap(uint32_t id, void* ptr);
    // Atomically compare and swap the provided value only if it equals
    // to expected value.
    bool CompareAndSwap(uint32_t id, void* ptr, void*& expected);
    // Reset all thread local data to replacement, and return non-nullptr
    // data for all existing threads
    void Scrape(uint32_t id, autovector<void*>* ptrs, void* const replacement);

    // Register the UnrefHandler for id
    void SetHandler(uint32_t id, UnrefHandler handler);

   private:
    // Get UnrefHandler for id with acquiring mutex
    // REQUIRES: mutex locked
    UnrefHandler GetHandler(uint32_t id);

    // Triggered before a thread terminates
    static void OnThreadExit(void* ptr);

    // Add current thread's ThreadData to the global chain
    // REQUIRES: mutex locked
    void AddThreadData(ThreadData* d);

    // Remove current thread's ThreadData from the global chain
    // REQUIRES: mutex locked
    void RemoveThreadData(ThreadData* d);

    static ThreadData* GetThreadLocal();

    uint32_t next_instance_id_;
    // Used to recycle Ids in case ThreadLocalPtr is instantiated and destroyed
    // frequently. This also prevents it from blowing up the vector space.
    autovector<uint32_t> free_instance_ids_;
    // Chain all thread local structure together. This is necessary since
    // when one ThreadLocalPtr gets destroyed, we need to loop over each
    // thread's version of pointer corresponding to that instance and
    // call UnrefHandler for it.
    ThreadData head_;

    std::unordered_map<uint32_t, UnrefHandler> handler_map_;

    // protect inst, next_instance_id_, free_instance_ids_, head_,
    // ThreadData.entries
    static port::Mutex mutex_;
#if ROCKSDB_SUPPORT_THREAD_LOCAL
    // Thread local storage
    static __thread ThreadData* tls_;
#endif

    // Used to make thread exit trigger possible if !defined(OS_MACOSX).
    // Otherwise, used to retrieve thread data.
    pthread_key_t pthread_key_;
  };

  static StaticMeta* Instance();

  const uint32_t id_;
};

}  // namespace rocksdb
