//  Copyright (c) 2013, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.
//
// Copyright (c) 2011 The LevelDB Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file. See the AUTHORS file for names of contributors.
//
// See port_example.h for documentation for the following types/functions.

#ifndef STORAGE_LEVELDB_PORT_PORT_WIN_H_
#define STORAGE_LEVELDB_PORT_PORT_WIN_H_

// Always want minimum headers
#ifndef WIN32_LEAN_AND_MEAN
#define WIN32_LEAN_AND_MEAN
#endif

// Assume that for everywhere
#undef PLATFORM_IS_LITTLE_ENDIAN
#define PLATFORM_IS_LITTLE_ENDIAN true

#include <windows.h>
#include <string>
#include <string.h>
#include <mutex>
#include <condition_variable>

#include <stdint.h>

#include "rocksdb/options.h"

#undef min
#undef max
#undef DeleteFile
#undef GetCurrentTime


#ifndef strcasecmp
#define strcasecmp _stricmp
#endif

// defined in stdio.h
#ifndef snprintf
#define snprintf _snprintf
#endif

#undef GetCurrentTime
#undef DeleteFile

typedef SSIZE_T ssize_t;

// size_t printf formatting named in the manner of C99 standard formatting
// strings such as PRIu64
// in fact, we could use that one
#ifndef ROCKSDB_PRIszt
#define ROCKSDB_PRIszt "Iu"
#endif

#define ROCKSDB_NOEXCEPT

#define __attribute__(A)

#ifdef ZLIB
#include <zlib.h>
#endif

#ifdef BZIP2
#include <bzlib.h>
#endif

#if defined(LZ4)
#include <lz4.h>
#include <lz4hc.h>
#endif

#ifdef SNAPPY
#include <snappy.h>
#endif

// Thread local storage on Linux
// There is thread_local in C++11
#ifndef __thread
#define __thread __declspec(thread)
#endif

#ifndef PLATFORM_IS_LITTLE_ENDIAN
#define PLATFORM_IS_LITTLE_ENDIAN (__BYTE_ORDER == __LITTLE_ENDIAN)
#endif

namespace rocksdb {

#define PREFETCH(addr, rw, locality)
std::string GetWindowsErrSz(DWORD err);

namespace port {

// For use at db/file_indexer.h kLevelMaxIndex
const int kMaxInt32 = INT32_MAX;
const uint64_t kMaxUint64 = UINT64_MAX;

const bool kLittleEndian = true;

class CondVar;

class Mutex {
 public:
  /* implicit */ Mutex(bool adaptive = false);
  ~Mutex();

  void Lock();
  void Unlock();

  // this will assert if the mutex is not locked
  // it does NOT verify that mutex is held by a calling thread
  void AssertHeld();

  std::unique_lock<std::mutex>& getLock() { return lock; }

 private:
  friend class CondVar;
  std::mutex m_mutex;
  std::unique_lock<std::mutex> lock;
#ifndef NDEBUG
  bool locked_;
#endif

  // No copying
  Mutex(const Mutex&);
  void operator=(const Mutex&);
};

class RWMutex {
 public:
  RWMutex() { InitializeSRWLock(&srwLock_); }

  void ReadLock() { AcquireSRWLockShared(&srwLock_); }

  void WriteLock() { AcquireSRWLockExclusive(&srwLock_); }

  void ReadUnlock() { ReleaseSRWLockShared(&srwLock_); }

  void WriteUnlock() { ReleaseSRWLockExclusive(&srwLock_); }

  // Empty as in POSIX
  void AssertHeld() {}

 private:
  SRWLOCK srwLock_;
  // No copying allowed
  RWMutex(const RWMutex&);
  void operator=(const RWMutex&);
};

class CondVar {
 public:
  explicit CondVar(Mutex* mu);
  ~CondVar();
  void Wait();
  bool TimedWait(uint64_t expiration_time);
  void Signal();
  void SignalAll();

 private:
  std::condition_variable cv_;
  Mutex* mu_;
};

typedef std::once_flag OnceType;
#define LEVELDB_ONCE_INIT std::once_flag::once_flag();
extern void InitOnce(OnceType* once, void (*initializer)());

#define CACHE_LINE_SIZE 64U

#ifdef min
#undef min
#endif
#ifdef max
#undef max
#endif

// For Thread Local Storage abstraction
typedef DWORD pthread_key_t;

inline int pthread_key_create(pthread_key_t* key, void (*destructor)(void*)) {
  // Not used
  (void)destructor;

  pthread_key_t k = TlsAlloc();
  if (TLS_OUT_OF_INDEXES == k) {
    return ENOMEM;
  }

  *key = k;
  return 0;
}

inline int pthread_key_delete(pthread_key_t key) {
  if (!TlsFree(key)) {
    return EINVAL;
  }
  return 0;
}

inline int pthread_setspecific(pthread_key_t key, const void* value) {
  if (!TlsSetValue(key, const_cast<void*>(value))) {
    return ENOMEM;
  }
  return 0;
}

inline void* pthread_getspecific(pthread_key_t key) {
  void* result = TlsGetValue(key);
  if (!result) {
    if (GetLastError() != ERROR_SUCCESS) {
      errno = EINVAL;
    } else {
      errno = NOERROR;
    }
  }
  return result;
}

// UNIX equiv although errno numbers will be off
// using C-runtime to implement. Note, this does not
// feel space with zeros in case the file is extended.
int truncate(const char* path, int64_t length);
void Crash(const std::string& srcfile, int srcline);
extern int GetMaxOpenFiles();

}  // namespace port

using port::pthread_key_t;
using port::pthread_key_create;
using port::pthread_key_delete;
using port::pthread_setspecific;
using port::pthread_getspecific;
using port::truncate;

}  // namespace rocksdb

#endif  // STORAGE_LEVELDB_PORT_PORT_POSIX_H_
