//  Copyright (c) 2013, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.
//
// Copyright (c) 2011 The LevelDB Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file. See the AUTHORS file for names of contributors.

#if !defined(OS_WIN) && !defined(WIN32) && !defined(_WIN32)
#error Windows Specific Code
#endif

#include "port/win/port_win.h"

#include <io.h>
#include "port/dirent.h"
#include "port/sys_time.h"

#include <cstdlib>
#include <stdio.h>
#include <assert.h>
#include <string.h>

#include <memory>
#include <exception>
#include <chrono>

#include "util/logging.h"

namespace rocksdb {
namespace port {

void gettimeofday(struct timeval* tv, struct timezone* /* tz */) {
  using namespace std::chrono;

  microseconds usNow(
      duration_cast<microseconds>(system_clock::now().time_since_epoch()));

  seconds secNow(duration_cast<seconds>(usNow));

  tv->tv_sec = secNow.count();
  tv->tv_usec = usNow.count() - duration_cast<microseconds>(secNow).count();
}

Mutex::Mutex(bool adaptive) : lock(m_mutex, std::defer_lock) {}

Mutex::~Mutex() {}

void Mutex::Lock() {
  lock.lock();
#ifndef NDEBUG
  locked_ = true;
#endif
}

void Mutex::Unlock() {
#ifndef NDEBUG
  locked_ = false;
#endif
  lock.unlock();
}

void Mutex::AssertHeld() {
#ifndef NDEBUG
  assert(locked_);
#endif
}

CondVar::CondVar(Mutex* mu) : mu_(mu) {}

CondVar::~CondVar() {}

void CondVar::Wait() {
#ifndef NDEBUG
  mu_->locked_ = false;
#endif
  cv_.wait(mu_->getLock());
#ifndef NDEBUG
  mu_->locked_ = true;
#endif
}

bool CondVar::TimedWait(uint64_t abs_time_us) {
#ifndef NDEBUG
  mu_->locked_ = false;
#endif

  using namespace std::chrono;

  // MSVC++ library implements wait_until in terms of wait_for so
  // there is not an absolute wait anyway.
  microseconds usAbsTime(abs_time_us);

  microseconds usNow(
      duration_cast<microseconds>(system_clock::now().time_since_epoch()));
  microseconds relTimeUs =
      (usAbsTime > usNow) ? (usAbsTime - usNow) : microseconds::zero();

  std::cv_status cvStatus = cv_.wait_for(mu_->getLock(), relTimeUs);

#ifndef NDEBUG
  mu_->locked_ = true;
#endif

  if (cvStatus == std::cv_status::timeout) {
    return true;
  }

  return false;
}

void CondVar::Signal() { cv_.notify_one(); }

void CondVar::SignalAll() { cv_.notify_all(); }

void InitOnce(OnceType* once, void (*initializer)()) {
  std::call_once(*once, initializer);
}

// Private structure, exposed only by pointer
struct DIR {
  intptr_t handle_;
  bool firstread_;
  struct __finddata64_t data_;
  dirent entry_;

  DIR() : handle_(-1), firstread_(true) {}

  DIR(const DIR&) = delete;
  DIR& operator=(const DIR&) = delete;

  ~DIR() {
    if (-1 != handle_) {
      _findclose(handle_);
    }
  }
};

DIR* opendir(const char* name) {
  if (!name || *name == 0) {
    errno = ENOENT;
    return nullptr;
  }

  std::string pattern(name);
  pattern.append("\\").append("*");

  std::unique_ptr<DIR> dir(new DIR);

  dir->handle_ = _findfirst64(pattern.c_str(), &dir->data_);

  if (dir->handle_ == -1) {
    return nullptr;
  }

  strncpy_s(dir->entry_.d_name, dir->data_.name, strlen(dir->data_.name));

  return dir.release();
}

struct dirent* readdir(DIR* dirp) {
  if (!dirp || dirp->handle_ == -1) {
    errno = EBADF;
    return nullptr;
  }

  if (dirp->firstread_) {
    dirp->firstread_ = false;
    return &dirp->entry_;
  }

  auto ret = _findnext64(dirp->handle_, &dirp->data_);

  if (ret != 0) {
    return nullptr;
  }

  strncpy_s(dirp->entry_.d_name, dirp->data_.name, strlen(dirp->data_.name));

  return &dirp->entry_;
}

int closedir(DIR* dirp) {
  delete dirp;
  return 0;
}

int truncate(const char* path, int64_t len) {
  if (path == nullptr) {
    errno = EFAULT;
    return -1;
  }

  if (len < 0) {
    errno = EINVAL;
    return -1;
  }

  HANDLE hFile =
      CreateFile(path, GENERIC_READ | GENERIC_WRITE,
                 FILE_SHARE_READ | FILE_SHARE_WRITE | FILE_SHARE_DELETE,
                 NULL,           // Security attrs
                 OPEN_EXISTING,  // Truncate existing file only
                 FILE_ATTRIBUTE_NORMAL, NULL);

  if (INVALID_HANDLE_VALUE == hFile) {
    auto lastError = GetLastError();
    if (lastError == ERROR_FILE_NOT_FOUND) {
      errno = ENOENT;
    } else if (lastError == ERROR_ACCESS_DENIED) {
      errno = EACCES;
    } else {
      errno = EIO;
    }
    return -1;
  }

  int result = 0;
  FILE_END_OF_FILE_INFO end_of_file;
  end_of_file.EndOfFile.QuadPart = len;

  if (!SetFileInformationByHandle(hFile, FileEndOfFileInfo, &end_of_file,
                                  sizeof(FILE_END_OF_FILE_INFO))) {
    errno = EIO;
    result = -1;
  }

  CloseHandle(hFile);
  return result;
}

void Crash(const std::string& srcfile, int srcline) {
  fprintf(stdout, "Crashing at %s:%d\n", srcfile.c_str(), srcline);
  fflush(stdout);
  abort();
}

int GetMaxOpenFiles() { return -1; }

}  // namespace port
}  // namespace rocksdb

#ifdef JEMALLOC

#include "jemalloc/jemalloc.h"

namespace rocksdb {

namespace port {

__declspec(noinline) void WINAPI InitializeJemalloc() {
  je_init();
  atexit(je_uninit);
}

}  // port
}  // rocksdb

extern "C" {

#ifdef _WIN64

#pragma comment(linker, "/INCLUDE:p_rocksdb_init_jemalloc")

typedef void(WINAPI* CRT_Startup_Routine)(void);

// .CRT section is merged with .rdata on x64 so it must be constant data.
// must be of external linkage
// We put this into XCT since we want to run this earlier than C++ static
// constructors
// which are placed into XCU
#pragma const_seg(".CRT$XCT")
extern const CRT_Startup_Routine p_rocksdb_init_jemalloc;
const CRT_Startup_Routine p_rocksdb_init_jemalloc =
    rocksdb::port::InitializeJemalloc;
#pragma const_seg()

#else  // _WIN64

// x86 untested

#pragma comment(linker, "/INCLUDE:_p_rocksdb_init_jemalloc")

#pragma section(".CRT$XCT", read)
JEMALLOC_SECTION(".CRT$XCT") JEMALLOC_ATTR(used) static const void(
    WINAPI* p_rocksdb_init_jemalloc)(void) = rocksdb::port::InitializeJemalloc;

#endif  // _WIN64

}  // extern "C"

// Global operators to be replaced by a linker

void* operator new(size_t size) {
  void* p = je_malloc(size);
  if (!p) {
    throw std::bad_alloc();
  }
  return p;
}

void* operator new[](size_t size) {
  void* p = je_malloc(size);
  if (!p) {
    throw std::bad_alloc();
  }
  return p;
}

void operator delete(void* p) { je_free(p); }

void operator delete[](void* p) { je_free(p); }

#endif  // JEMALLOC
