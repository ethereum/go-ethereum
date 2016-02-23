// Copyright (c) 2014, Facebook, Inc.  All rights reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.

#include "rocksdb/utilities/flashcache.h"

#include "utilities/flashcache/flashcache.h"

#ifdef OS_LINUX
#include <fcntl.h>
#include <sys/ioctl.h>
#include <sys/stat.h>
#include <sys/syscall.h>
#include <unistd.h>

#include "third-party/flashcache/flashcache_ioctl.h"
#endif

namespace rocksdb {

#if !defined(ROCKSDB_LITE) && defined(OS_LINUX)
// Most of the code that handles flashcache is copied from websql's branch of
// mysql-5.6
class FlashcacheAwareEnv : public EnvWrapper {
 public:
  FlashcacheAwareEnv(Env* base, int cachedev_fd)
      : EnvWrapper(base), cachedev_fd_(cachedev_fd) {
    pid_t pid = getpid();
    /* cleanup previous whitelistings */
    if (ioctl(cachedev_fd_, FLASHCACHEDELALLWHITELIST, &pid) < 0) {
      cachedev_fd_ = -1;
      fprintf(stderr, "ioctl del-all-whitelist for flashcache failed\n");
      return;
    }
    if (ioctl(cachedev_fd_, FLASHCACHEADDWHITELIST, &pid) < 0) {
      fprintf(stderr, "ioctl add-whitelist for flashcache failed\n");
    }
  }

  ~FlashcacheAwareEnv() {
    // cachedev_fd_ is -1 if it's unitialized
    if (cachedev_fd_ != -1) {
      pid_t pid = getpid();
      if (ioctl(cachedev_fd_, FLASHCACHEDELWHITELIST, &pid) < 0) {
        fprintf(stderr, "ioctl del-whitelist for flashcache failed\n");
      }
    }
  }

  static int BlacklistCurrentThread(int cachedev_fd) {
    pid_t pid = static_cast<pid_t>(syscall(SYS_gettid));
    return ioctl(cachedev_fd, FLASHCACHEADDNCPID, &pid);
  }

  static int WhitelistCurrentThread(int cachedev_fd) {
    pid_t pid = static_cast<pid_t>(syscall(SYS_gettid));
    return ioctl(cachedev_fd, FLASHCACHEDELNCPID, &pid);
  }

  int GetFlashCacheFileDescriptor() { return cachedev_fd_; }

  struct Arg {
    Arg(void (*f)(void* arg), void* a, int _cachedev_fd)
        : original_function_(f), original_arg_(a), cachedev_fd(_cachedev_fd) {}

    void (*original_function_)(void* arg);
    void* original_arg_;
    int cachedev_fd;
  };

  static void BgThreadWrapper(void* a) {
    Arg* arg = reinterpret_cast<Arg*>(a);
    if (arg->cachedev_fd != -1) {
      if (BlacklistCurrentThread(arg->cachedev_fd) < 0) {
        fprintf(stderr, "ioctl add-nc-pid for flashcache failed\n");
      }
    }
    arg->original_function_(arg->original_arg_);
    if (arg->cachedev_fd != -1) {
      if (WhitelistCurrentThread(arg->cachedev_fd) < 0) {
        fprintf(stderr, "ioctl del-nc-pid for flashcache failed\n");
      }
    }
    delete arg;
  }

  int UnSchedule(void* arg, Priority pri) override {
    // no unschedule for you
    return 0;
  }

  void Schedule(void (*f)(void* arg), void* a, Priority pri,
                void* tag = nullptr) override {
    EnvWrapper::Schedule(&BgThreadWrapper, new Arg(f, a, cachedev_fd_), pri,
                         tag);
  }

 private:
  int cachedev_fd_;
};

std::unique_ptr<Env> NewFlashcacheAwareEnv(Env* base,
                                           const int cachedev_fd) {
  std::unique_ptr<Env> ret(new FlashcacheAwareEnv(base, cachedev_fd));
  return std::move(ret);
}

int FlashcacheBlacklistCurrentThread(Env* flashcache_aware_env) {
  int fd = dynamic_cast<FlashcacheAwareEnv*>(flashcache_aware_env)
               ->GetFlashCacheFileDescriptor();
  if (fd == -1) {
    return -1;
  }
  return FlashcacheAwareEnv::BlacklistCurrentThread(fd);
}
int FlashcacheWhitelistCurrentThread(Env* flashcache_aware_env) {
  int fd = dynamic_cast<FlashcacheAwareEnv*>(flashcache_aware_env)
               ->GetFlashCacheFileDescriptor();
  if (fd == -1) {
    return -1;
  }
  return FlashcacheAwareEnv::WhitelistCurrentThread(fd);
}

#else   // !defined(ROCKSDB_LITE) && defined(OS_LINUX)
std::unique_ptr<Env> NewFlashcacheAwareEnv(Env* base,
                                           const int cachedev_fd) {
  return nullptr;
}
int FlashcacheBlacklistCurrentThread(Env* flashcache_aware_env) { return -1; }
int FlashcacheWhitelistCurrentThread(Env* flashcache_aware_env) { return -1; }

#endif  // !defined(ROCKSDB_LITE) && defined(OS_LINUX)

}  // namespace rocksdb
