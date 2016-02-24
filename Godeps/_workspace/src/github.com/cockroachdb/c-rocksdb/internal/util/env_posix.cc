//  Copyright (c) 2013, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.
//
// Copyright (c) 2011 The LevelDB Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file. See the AUTHORS file for names of contributors.

#include <deque>
#include <set>
#include <dirent.h>
#include <errno.h>
#include <fcntl.h>
#include <pthread.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/ioctl.h>
#include <sys/mman.h>
#include <sys/stat.h>
#ifdef OS_LINUX
#include <sys/statfs.h>
#include <sys/syscall.h>
#endif
#include <sys/time.h>
#include <sys/types.h>
#include <time.h>
#include <unistd.h>
#if defined(OS_LINUX)
#include <linux/fs.h>
#endif
#include <signal.h>
#include <algorithm>
#include "rocksdb/env.h"
#include "rocksdb/slice.h"
#include "port/port.h"
#include "util/coding.h"
#include "util/logging.h"
#include "util/posix_logger.h"
#include "util/random.h"
#include "util/iostats_context_imp.h"
#include "util/string_util.h"
#include "util/sync_point.h"
#include "util/thread_status_updater.h"
#include "util/thread_status_util.h"

// Get nano time includes
#if defined(OS_LINUX) || defined(OS_FREEBSD)
#elif defined(__MACH__)
#include <mach/clock.h>
#include <mach/mach.h>
#else
#include <chrono>
#endif

#if !defined(TMPFS_MAGIC)
#define TMPFS_MAGIC 0x01021994
#endif
#if !defined(XFS_SUPER_MAGIC)
#define XFS_SUPER_MAGIC 0x58465342
#endif
#if !defined(EXT4_SUPER_MAGIC)
#define EXT4_SUPER_MAGIC 0xEF53
#endif

// For non linux platform, the following macros are used only as place
// holder.
#if !(defined OS_LINUX) && !(defined CYGWIN)
#define POSIX_FADV_NORMAL 0 /* [MC1] no further special treatment */
#define POSIX_FADV_RANDOM 1 /* [MC1] expect random page refs */
#define POSIX_FADV_SEQUENTIAL 2 /* [MC1] expect sequential page refs */
#define POSIX_FADV_WILLNEED 3 /* [MC1] will need these pages */
#define POSIX_FADV_DONTNEED 4 /* [MC1] dont need these pages */
#endif


namespace rocksdb {

namespace {

// A wrapper for fadvise, if the platform doesn't support fadvise,
// it will simply return Status::NotSupport.
int Fadvise(int fd, off_t offset, size_t len, int advice) {
#ifdef OS_LINUX
  return posix_fadvise(fd, offset, len, advice);
#else
  return 0;  // simply do nothing.
#endif
}

ThreadStatusUpdater* CreateThreadStatusUpdater() {
  return new ThreadStatusUpdater();
}

// list of pathnames that are locked
static std::set<std::string> lockedFiles;
static port::Mutex mutex_lockedFiles;

static Status IOError(const std::string& context, int err_number) {
  return Status::IOError(context, strerror(err_number));
}

#if defined(OS_LINUX)
namespace {
  static size_t GetUniqueIdFromFile(int fd, char* id, size_t max_size) {
    if (max_size < kMaxVarint64Length*3) {
      return 0;
    }

    struct stat buf;
    int result = fstat(fd, &buf);
    if (result == -1) {
      return 0;
    }

    long version = 0;
    result = ioctl(fd, FS_IOC_GETVERSION, &version);
    if (result == -1) {
      return 0;
    }
    uint64_t uversion = (uint64_t)version;

    char* rid = id;
    rid = EncodeVarint64(rid, buf.st_dev);
    rid = EncodeVarint64(rid, buf.st_ino);
    rid = EncodeVarint64(rid, uversion);
    assert(rid >= id);
    return static_cast<size_t>(rid-id);
  }
}
#endif

class PosixSequentialFile: public SequentialFile {
 private:
  std::string filename_;
  FILE* file_;
  int fd_;
  bool use_os_buffer_;

 public:
  PosixSequentialFile(const std::string& fname, FILE* f,
      const EnvOptions& options)
      : filename_(fname), file_(f), fd_(fileno(f)),
        use_os_buffer_(options.use_os_buffer) {
  }
  virtual ~PosixSequentialFile() { fclose(file_); }

  virtual Status Read(size_t n, Slice* result, char* scratch) override {
    Status s;
    size_t r = 0;
    do {
      r = fread_unlocked(scratch, 1, n, file_);
    } while (r == 0 && ferror(file_) && errno == EINTR);
    *result = Slice(scratch, r);
    if (r < n) {
      if (feof(file_)) {
        // We leave status as ok if we hit the end of the file
        // We also clear the error so that the reads can continue
        // if a new data is written to the file
        clearerr(file_);
      } else {
        // A partial read with an error: return a non-ok status
        s = IOError(filename_, errno);
      }
    }
    if (!use_os_buffer_) {
      // we need to fadvise away the entire range of pages because
      // we do not want readahead pages to be cached.
      Fadvise(fd_, 0, 0, POSIX_FADV_DONTNEED); // free OS pages
    }
    return s;
  }

  virtual Status Skip(uint64_t n) override {
    if (fseek(file_, static_cast<long int>(n), SEEK_CUR)) {
      return IOError(filename_, errno);
    }
    return Status::OK();
  }

  virtual Status InvalidateCache(size_t offset, size_t length) override {
#ifndef OS_LINUX
    return Status::OK();
#else
    // free OS pages
    int ret = Fadvise(fd_, offset, length, POSIX_FADV_DONTNEED);
    if (ret == 0) {
      return Status::OK();
    }
    return IOError(filename_, errno);
#endif
  }
};

// pread() based random-access
class PosixRandomAccessFile: public RandomAccessFile {
 private:
  std::string filename_;
  int fd_;
  bool use_os_buffer_;

 public:
  PosixRandomAccessFile(const std::string& fname, int fd,
                        const EnvOptions& options)
      : filename_(fname), fd_(fd), use_os_buffer_(options.use_os_buffer) {
    assert(!options.use_mmap_reads || sizeof(void*) < 8);
  }
  virtual ~PosixRandomAccessFile() { close(fd_); }

  virtual Status Read(uint64_t offset, size_t n, Slice* result,
                      char* scratch) const override {
    Status s;
    ssize_t r = -1;
    size_t left = n;
    char* ptr = scratch;
    while (left > 0) {
      r = pread(fd_, ptr, left, static_cast<off_t>(offset));

      if (r <= 0) {
        if (errno == EINTR) {
          continue;
        }
        break;
      }
      ptr += r;
      offset += r;
      left -= r;
    }

    *result = Slice(scratch, (r < 0) ? 0 : n - left);
    if (r < 0) {
      // An error: return a non-ok status
      s = IOError(filename_, errno);
    }
    if (!use_os_buffer_) {
      // we need to fadvise away the entire range of pages because
      // we do not want readahead pages to be cached.
      Fadvise(fd_, 0, 0, POSIX_FADV_DONTNEED); // free OS pages
    }
    return s;
  }

#ifdef OS_LINUX
  virtual size_t GetUniqueId(char* id, size_t max_size) const override {
    return GetUniqueIdFromFile(fd_, id, max_size);
  }
#endif

  virtual void Hint(AccessPattern pattern) override {
    switch(pattern) {
      case NORMAL:
        Fadvise(fd_, 0, 0, POSIX_FADV_NORMAL);
        break;
      case RANDOM:
        Fadvise(fd_, 0, 0, POSIX_FADV_RANDOM);
        break;
      case SEQUENTIAL:
        Fadvise(fd_, 0, 0, POSIX_FADV_SEQUENTIAL);
        break;
      case WILLNEED:
        Fadvise(fd_, 0, 0, POSIX_FADV_WILLNEED);
        break;
      case DONTNEED:
        Fadvise(fd_, 0, 0, POSIX_FADV_DONTNEED);
        break;
      default:
        assert(false);
        break;
    }
  }

  virtual Status InvalidateCache(size_t offset, size_t length) override {
#ifndef OS_LINUX
    return Status::OK();
#else
    // free OS pages
    int ret = Fadvise(fd_, offset, length, POSIX_FADV_DONTNEED);
    if (ret == 0) {
      return Status::OK();
    }
    return IOError(filename_, errno);
#endif
  }
};

// mmap() based random-access
class PosixMmapReadableFile: public RandomAccessFile {
 private:
  int fd_;
  std::string filename_;
  void* mmapped_region_;
  size_t length_;

 public:
  // base[0,length-1] contains the mmapped contents of the file.
  PosixMmapReadableFile(const int fd, const std::string& fname,
                        void* base, size_t length,
                        const EnvOptions& options)
      : fd_(fd), filename_(fname), mmapped_region_(base), length_(length) {
    fd_ = fd_ + 0;  // suppress the warning for used variables
    assert(options.use_mmap_reads);
    assert(options.use_os_buffer);
  }
  virtual ~PosixMmapReadableFile() {
    int ret = munmap(mmapped_region_, length_);
    if (ret != 0) {
      fprintf(stdout, "failed to munmap %p length %" ROCKSDB_PRIszt " \n",
              mmapped_region_, length_);
    }
  }

  virtual Status Read(uint64_t offset, size_t n, Slice* result,
                      char* scratch) const override {
    Status s;
    if (offset > length_) {
      *result = Slice();
      return IOError(filename_, EINVAL);
    } else if (offset + n > length_) {
      n = length_ - offset;
    }
    *result = Slice(reinterpret_cast<char*>(mmapped_region_) + offset, n);
    return s;
  }
  virtual Status InvalidateCache(size_t offset, size_t length) override {
#ifndef OS_LINUX
    return Status::OK();
#else
    // free OS pages
    int ret = Fadvise(fd_, offset, length, POSIX_FADV_DONTNEED);
    if (ret == 0) {
      return Status::OK();
    }
    return IOError(filename_, errno);
#endif
  }
};

// We preallocate up to an extra megabyte and use memcpy to append new
// data to the file.  This is safe since we either properly close the
// file before reading from it, or for log files, the reading code
// knows enough to skip zero suffixes.
class PosixMmapFile : public WritableFile {
 private:
  std::string filename_;
  int fd_;
  size_t page_size_;
  size_t map_size_;       // How much extra memory to map at a time
  char* base_;            // The mapped region
  char* limit_;           // Limit of the mapped region
  char* dst_;             // Where to write next  (in range [base_,limit_])
  char* last_sync_;       // Where have we synced up to
  uint64_t file_offset_;  // Offset of base_ in file
#ifdef ROCKSDB_FALLOCATE_PRESENT
  bool fallocate_with_keep_size_;
#endif

  // Roundup x to a multiple of y
  static size_t Roundup(size_t x, size_t y) {
    return ((x + y - 1) / y) * y;
  }

  size_t TruncateToPageBoundary(size_t s) {
    s -= (s & (page_size_ - 1));
    assert((s % page_size_) == 0);
    return s;
  }

  Status UnmapCurrentRegion() {
    TEST_KILL_RANDOM(rocksdb_kill_odds);
    if (base_ != nullptr) {
      int munmap_status = munmap(base_, limit_ - base_);
      if (munmap_status != 0) {
        return IOError(filename_, munmap_status);
      }
      file_offset_ += limit_ - base_;
      base_ = nullptr;
      limit_ = nullptr;
      last_sync_ = nullptr;
      dst_ = nullptr;

      // Increase the amount we map the next time, but capped at 1MB
      if (map_size_ < (1<<20)) {
        map_size_ *= 2;
      }
    }
    return Status::OK();
  }

  Status MapNewRegion() {
#ifdef ROCKSDB_FALLOCATE_PRESENT
    assert(base_ == nullptr);

    TEST_KILL_RANDOM(rocksdb_kill_odds);
    // we can't fallocate with FALLOC_FL_KEEP_SIZE here
    {
      IOSTATS_TIMER_GUARD(allocate_nanos);
      int alloc_status = fallocate(fd_, 0, file_offset_, map_size_);
      if (alloc_status != 0) {
        // fallback to posix_fallocate
        alloc_status = posix_fallocate(fd_, file_offset_, map_size_);
      }
      if (alloc_status != 0) {
        return Status::IOError("Error allocating space to file : " + filename_ +
          "Error : " + strerror(alloc_status));
      }
    }

    TEST_KILL_RANDOM(rocksdb_kill_odds);
    void* ptr = mmap(nullptr, map_size_, PROT_READ | PROT_WRITE, MAP_SHARED,
                     fd_, file_offset_);
    if (ptr == MAP_FAILED) {
      return Status::IOError("MMap failed on " + filename_);
    }
    TEST_KILL_RANDOM(rocksdb_kill_odds);

    base_ = reinterpret_cast<char*>(ptr);
    limit_ = base_ + map_size_;
    dst_ = base_;
    last_sync_ = base_;
    return Status::OK();
#else
    return Status::NotSupported("This platform doesn't support fallocate()");
#endif
  }

  Status Msync() {
    if (dst_ == last_sync_) {
      return Status::OK();
    }
    // Find the beginnings of the pages that contain the first and last
    // bytes to be synced.
    size_t p1 = TruncateToPageBoundary(last_sync_ - base_);
    size_t p2 = TruncateToPageBoundary(dst_ - base_ - 1);
    last_sync_ = dst_;
    TEST_KILL_RANDOM(rocksdb_kill_odds);
    if (msync(base_ + p1, p2 - p1 + page_size_, MS_SYNC) < 0) {
      return IOError(filename_, errno);
    }
    return Status::OK();
  }

 public:
  PosixMmapFile(const std::string& fname, int fd, size_t page_size,
                const EnvOptions& options)
      : filename_(fname),
        fd_(fd),
        page_size_(page_size),
        map_size_(Roundup(65536, page_size)),
        base_(nullptr),
        limit_(nullptr),
        dst_(nullptr),
        last_sync_(nullptr),
        file_offset_(0) {
#ifdef ROCKSDB_FALLOCATE_PRESENT
    fallocate_with_keep_size_ = options.fallocate_with_keep_size;
#endif
    assert((page_size & (page_size - 1)) == 0);
    assert(options.use_mmap_writes);
  }


  ~PosixMmapFile() {
    if (fd_ >= 0) {
      PosixMmapFile::Close();
    }
  }

  virtual Status Append(const Slice& data) override {
    const char* src = data.data();
    size_t left = data.size();
    while (left > 0) {
      assert(base_ <= dst_);
      assert(dst_ <= limit_);
      size_t avail = limit_ - dst_;
      if (avail == 0) {
        Status s = UnmapCurrentRegion();
        if (!s.ok()) {
          return s;
        }
        s = MapNewRegion();
        if (!s.ok()) {
          return s;
        }
        TEST_KILL_RANDOM(rocksdb_kill_odds);
      }

      size_t n = (left <= avail) ? left : avail;
      memcpy(dst_, src, n);
      dst_ += n;
      src += n;
      left -= n;
    }
    return Status::OK();
  }

  virtual Status Close() override {
    Status s;
    size_t unused = limit_ - dst_;

    s = UnmapCurrentRegion();
    if (!s.ok()) {
      s = IOError(filename_, errno);
    } else if (unused > 0) {
      // Trim the extra space at the end of the file
      if (ftruncate(fd_, file_offset_ - unused) < 0) {
        s = IOError(filename_, errno);
      }
    }

    if (close(fd_) < 0) {
      if (s.ok()) {
        s = IOError(filename_, errno);
      }
    }

    fd_ = -1;
    base_ = nullptr;
    limit_ = nullptr;
    return s;
  }

  virtual Status Flush() override {
    return Status::OK();
  }

  virtual Status Sync() override {
    if (fdatasync(fd_) < 0) {
      return IOError(filename_, errno);
    }

    return Msync();
  }

  /**
   * Flush data as well as metadata to stable storage.
   */
  virtual Status Fsync() override {
    if (fsync(fd_) < 0) {
      return IOError(filename_, errno);
    }

    return Msync();
  }

  /**
   * Get the size of valid data in the file. This will not match the
   * size that is returned from the filesystem because we use mmap
   * to extend file by map_size every time.
   */
  virtual uint64_t GetFileSize() override {
    size_t used = dst_ - base_;
    return file_offset_ + used;
  }

  virtual Status InvalidateCache(size_t offset, size_t length) override {
#ifndef OS_LINUX
    return Status::OK();
#else
    // free OS pages
    int ret = Fadvise(fd_, offset, length, POSIX_FADV_DONTNEED);
    if (ret == 0) {
      return Status::OK();
    }
    return IOError(filename_, errno);
#endif
  }

#ifdef ROCKSDB_FALLOCATE_PRESENT
  virtual Status Allocate(off_t offset, off_t len) override {
    TEST_KILL_RANDOM(rocksdb_kill_odds);
    int alloc_status = fallocate(
        fd_, fallocate_with_keep_size_ ? FALLOC_FL_KEEP_SIZE : 0, offset, len);
    if (alloc_status == 0) {
      return Status::OK();
    } else {
      return IOError(filename_, errno);
    }
  }
#endif
};

// Use posix write to write data to a file.
class PosixWritableFile : public WritableFile {
 private:
  const std::string filename_;
  int fd_;
  uint64_t filesize_;
#ifdef ROCKSDB_FALLOCATE_PRESENT
  bool fallocate_with_keep_size_;
#endif

 public:
  PosixWritableFile(const std::string& fname, int fd, const EnvOptions& options)
      : filename_(fname), fd_(fd), filesize_(0) {
#ifdef ROCKSDB_FALLOCATE_PRESENT
    fallocate_with_keep_size_ = options.fallocate_with_keep_size;
#endif
    assert(!options.use_mmap_writes);
  }

  ~PosixWritableFile() {
    if (fd_ >= 0) {
      PosixWritableFile::Close();
    }
  }

  virtual Status Append(const Slice& data) override {
    const char* src = data.data();
    size_t left = data.size();
    Status s;
      while (left != 0) {
        ssize_t done = write(fd_, src, left);
        if (done < 0) {
          if (errno == EINTR) {
            continue;
          }
          return IOError(filename_, errno);
        }
        left -= done;
        src += done;
      }
      filesize_ += data.size();
    return Status::OK();
  }

  virtual Status Close() override {
    Status s;

    size_t block_size;
    size_t last_allocated_block;
    GetPreallocationStatus(&block_size, &last_allocated_block);
    if (last_allocated_block > 0) {
      // trim the extra space preallocated at the end of the file
      // NOTE(ljin): we probably don't want to surface failure as an IOError,
      // but it will be nice to log these errors.
      int dummy __attribute__((unused));
      dummy = ftruncate(fd_, filesize_);
#ifdef ROCKSDB_FALLOCATE_PRESENT
      // in some file systems, ftruncate only trims trailing space if the
      // new file size is smaller than the current size. Calling fallocate
      // with FALLOC_FL_PUNCH_HOLE flag to explicitly release these unused
      // blocks. FALLOC_FL_PUNCH_HOLE is supported on at least the following
      // filesystems:
      //   XFS (since Linux 2.6.38)
      //   ext4 (since Linux 3.0)
      //   Btrfs (since Linux 3.7)
      //   tmpfs (since Linux 3.5)
      // We ignore error since failure of this operation does not affect
      // correctness.
      IOSTATS_TIMER_GUARD(allocate_nanos);
      fallocate(fd_, FALLOC_FL_KEEP_SIZE | FALLOC_FL_PUNCH_HOLE,
                filesize_, block_size * last_allocated_block - filesize_);
#endif
    }

    if (close(fd_) < 0) {
      s = IOError(filename_, errno);
    }
    fd_ = -1;
    return s;
  }

  // write out the cached data to the OS cache
  virtual Status Flush() override {
    return Status::OK();
  }

  virtual Status Sync() override {
    if (fdatasync(fd_) < 0) {
      return IOError(filename_, errno);
    }
    return Status::OK();
  }

  virtual Status Fsync() override {
    if (fsync(fd_) < 0) {
      return IOError(filename_, errno);
    }
    return Status::OK();
  }

  virtual bool IsSyncThreadSafe() const override {
    return true;
  }

  virtual uint64_t GetFileSize() override { return filesize_; }

  virtual Status InvalidateCache(size_t offset, size_t length) override {
#ifndef OS_LINUX
    return Status::OK();
#else
    // free OS pages
    int ret = Fadvise(fd_, offset, length, POSIX_FADV_DONTNEED);
    if (ret == 0) {
      return Status::OK();
    }
    return IOError(filename_, errno);
#endif
  }

#ifdef ROCKSDB_FALLOCATE_PRESENT
  virtual Status Allocate(off_t offset, off_t len) override {
    TEST_KILL_RANDOM(rocksdb_kill_odds);
    IOSTATS_TIMER_GUARD(allocate_nanos);
    int alloc_status;
    alloc_status = fallocate(
        fd_, fallocate_with_keep_size_ ? FALLOC_FL_KEEP_SIZE : 0, offset, len);
    if (alloc_status == 0) {
      return Status::OK();
    } else {
      return IOError(filename_, errno);
    }
  }

  virtual Status RangeSync(off_t offset, off_t nbytes) override {
    if (sync_file_range(fd_, offset, nbytes, SYNC_FILE_RANGE_WRITE) == 0) {
      return Status::OK();
    } else {
      return IOError(filename_, errno);
    }
  }
  virtual size_t GetUniqueId(char* id, size_t max_size) const override {
    return GetUniqueIdFromFile(fd_, id, max_size);
  }
#endif
};

class PosixDirectory : public Directory {
 public:
  explicit PosixDirectory(int fd) : fd_(fd) {}
  ~PosixDirectory() {
    close(fd_);
  }

  virtual Status Fsync() override {
    if (fsync(fd_) == -1) {
      return IOError("directory", errno);
    }
    return Status::OK();
  }

 private:
  int fd_;
};

static int LockOrUnlock(const std::string& fname, int fd, bool lock) {
  mutex_lockedFiles.Lock();
  if (lock) {
    // If it already exists in the lockedFiles set, then it is already locked,
    // and fail this lock attempt. Otherwise, insert it into lockedFiles.
    // This check is needed because fcntl() does not detect lock conflict
    // if the fcntl is issued by the same thread that earlier acquired
    // this lock.
    if (lockedFiles.insert(fname).second == false) {
      mutex_lockedFiles.Unlock();
      errno = ENOLCK;
      return -1;
    }
  } else {
    // If we are unlocking, then verify that we had locked it earlier,
    // it should already exist in lockedFiles. Remove it from lockedFiles.
    if (lockedFiles.erase(fname) != 1) {
      mutex_lockedFiles.Unlock();
      errno = ENOLCK;
      return -1;
    }
  }
  errno = 0;
  struct flock f;
  memset(&f, 0, sizeof(f));
  f.l_type = (lock ? F_WRLCK : F_UNLCK);
  f.l_whence = SEEK_SET;
  f.l_start = 0;
  f.l_len = 0;        // Lock/unlock entire file
  int value = fcntl(fd, F_SETLK, &f);
  if (value == -1 && lock) {
    // if there is an error in locking, then remove the pathname from lockedfiles
    lockedFiles.erase(fname);
  }
  mutex_lockedFiles.Unlock();
  return value;
}

class PosixFileLock : public FileLock {
 public:
  int fd_;
  std::string filename;
};

void PthreadCall(const char* label, int result) {
  if (result != 0) {
    fprintf(stderr, "pthread %s: %s\n", label, strerror(result));
    abort();
  }
}

class PosixEnv : public Env {
 public:
  PosixEnv();

  virtual ~PosixEnv() {
    for (const auto tid : threads_to_join_) {
      pthread_join(tid, nullptr);
    }
    for (int pool_id = 0; pool_id < Env::Priority::TOTAL; ++pool_id) {
      thread_pools_[pool_id].JoinAllThreads();
    }
    // All threads must be joined before the deletion of
    // thread_status_updater_.
    delete thread_status_updater_;
  }

  void SetFD_CLOEXEC(int fd, const EnvOptions* options) {
    if ((options == nullptr || options->set_fd_cloexec) && fd > 0) {
      fcntl(fd, F_SETFD, fcntl(fd, F_GETFD) | FD_CLOEXEC);
    }
  }

  virtual Status NewSequentialFile(const std::string& fname,
                                   unique_ptr<SequentialFile>* result,
                                   const EnvOptions& options) override {
    result->reset();
    FILE* f = nullptr;
    do {
      IOSTATS_TIMER_GUARD(open_nanos);
      f = fopen(fname.c_str(), "r");
    } while (f == nullptr && errno == EINTR);
    if (f == nullptr) {
      *result = nullptr;
      return IOError(fname, errno);
    } else {
      int fd = fileno(f);
      SetFD_CLOEXEC(fd, &options);
      result->reset(new PosixSequentialFile(fname, f, options));
      return Status::OK();
    }
  }

  virtual Status NewRandomAccessFile(const std::string& fname,
                                     unique_ptr<RandomAccessFile>* result,
                                     const EnvOptions& options) override {
    result->reset();
    Status s;
    int fd;
    {
      IOSTATS_TIMER_GUARD(open_nanos);
      fd = open(fname.c_str(), O_RDONLY);
    }
    SetFD_CLOEXEC(fd, &options);
    if (fd < 0) {
      s = IOError(fname, errno);
    } else if (options.use_mmap_reads && sizeof(void*) >= 8) {
      // Use of mmap for random reads has been removed because it
      // kills performance when storage is fast.
      // Use mmap when virtual address-space is plentiful.
      uint64_t size;
      s = GetFileSize(fname, &size);
      if (s.ok()) {
        void* base = mmap(nullptr, size, PROT_READ, MAP_SHARED, fd, 0);
        if (base != MAP_FAILED) {
          result->reset(new PosixMmapReadableFile(fd, fname, base,
                                                  size, options));
        } else {
          s = IOError(fname, errno);
        }
      }
      close(fd);
    } else {
      result->reset(new PosixRandomAccessFile(fname, fd, options));
    }
    return s;
  }

  virtual Status NewWritableFile(const std::string& fname,
                                 unique_ptr<WritableFile>* result,
                                 const EnvOptions& options) override {
    result->reset();
    Status s;
    int fd = -1;
    do {
      IOSTATS_TIMER_GUARD(open_nanos);
      fd = open(fname.c_str(), O_CREAT | O_RDWR | O_TRUNC, 0644);
    } while (fd < 0 && errno == EINTR);
    if (fd < 0) {
      s = IOError(fname, errno);
    } else {
      SetFD_CLOEXEC(fd, &options);
      if (options.use_mmap_writes) {
        if (!checkedDiskForMmap_) {
          // this will be executed once in the program's lifetime.
          // do not use mmapWrite on non ext-3/xfs/tmpfs systems.
          if (!SupportsFastAllocate(fname)) {
            forceMmapOff = true;
          }
          checkedDiskForMmap_ = true;
        }
      }
      if (options.use_mmap_writes && !forceMmapOff) {
        result->reset(new PosixMmapFile(fname, fd, page_size_, options));
      } else {
        // disable mmap writes
        EnvOptions no_mmap_writes_options = options;
        no_mmap_writes_options.use_mmap_writes = false;

        result->reset(new PosixWritableFile(fname, fd, no_mmap_writes_options));
      }
    }
    return s;
  }

  virtual Status NewDirectory(const std::string& name,
                              unique_ptr<Directory>* result) override {
    result->reset();
    int fd;
    {
      IOSTATS_TIMER_GUARD(open_nanos);
      fd = open(name.c_str(), 0);
    }
    if (fd < 0) {
      return IOError(name, errno);
    } else {
      result->reset(new PosixDirectory(fd));
    }
    return Status::OK();
  }

  virtual Status FileExists(const std::string& fname) override {
    int result = access(fname.c_str(), F_OK);

    if (result == 0) {
      return Status::OK();
    }

    switch (errno) {
      case EACCES:
      case ELOOP:
      case ENAMETOOLONG:
      case ENOENT:
      case ENOTDIR:
        return Status::NotFound();
      default:
        assert(result == EIO || result == ENOMEM);
        return Status::IOError("Unexpected error(" + ToString(result) +
                               ") accessing file `" + fname + "' ");
    }
  }

  virtual Status GetChildren(const std::string& dir,
                             std::vector<std::string>* result) override {
    result->clear();
    DIR* d = opendir(dir.c_str());
    if (d == nullptr) {
      return IOError(dir, errno);
    }
    struct dirent* entry;
    while ((entry = readdir(d)) != nullptr) {
      result->push_back(entry->d_name);
    }
    closedir(d);
    return Status::OK();
  }

  virtual Status DeleteFile(const std::string& fname) override {
    Status result;
    if (unlink(fname.c_str()) != 0) {
      result = IOError(fname, errno);
    }
    return result;
  };

  virtual Status CreateDir(const std::string& name) override {
    Status result;
    if (mkdir(name.c_str(), 0755) != 0) {
      result = IOError(name, errno);
    }
    return result;
  };

  virtual Status CreateDirIfMissing(const std::string& name) override {
    Status result;
    if (mkdir(name.c_str(), 0755) != 0) {
      if (errno != EEXIST) {
        result = IOError(name, errno);
      } else if (!DirExists(name)) { // Check that name is actually a
                                     // directory.
        // Message is taken from mkdir
        result = Status::IOError("`"+name+"' exists but is not a directory");
      }
    }
    return result;
  };

  virtual Status DeleteDir(const std::string& name) override {
    Status result;
    if (rmdir(name.c_str()) != 0) {
      result = IOError(name, errno);
    }
    return result;
  };

  virtual Status GetFileSize(const std::string& fname,
                             uint64_t* size) override {
    Status s;
    struct stat sbuf;
    if (stat(fname.c_str(), &sbuf) != 0) {
      *size = 0;
      s = IOError(fname, errno);
    } else {
      *size = sbuf.st_size;
    }
    return s;
  }

  virtual Status GetFileModificationTime(const std::string& fname,
                                         uint64_t* file_mtime) override {
    struct stat s;
    if (stat(fname.c_str(), &s) !=0) {
      return IOError(fname, errno);
    }
    *file_mtime = static_cast<uint64_t>(s.st_mtime);
    return Status::OK();
  }
  virtual Status RenameFile(const std::string& src,
                            const std::string& target) override {
    Status result;
    if (rename(src.c_str(), target.c_str()) != 0) {
      result = IOError(src, errno);
    }
    return result;
  }

  virtual Status LinkFile(const std::string& src,
                          const std::string& target) override {
    Status result;
    if (link(src.c_str(), target.c_str()) != 0) {
      if (errno == EXDEV) {
        return Status::NotSupported("No cross FS links allowed");
      }
      result = IOError(src, errno);
    }
    return result;
  }

  virtual Status LockFile(const std::string& fname, FileLock** lock) override {
    *lock = nullptr;
    Status result;
    int fd;
    {
      IOSTATS_TIMER_GUARD(open_nanos);
      fd = open(fname.c_str(), O_RDWR | O_CREAT, 0644);
    }
    if (fd < 0) {
      result = IOError(fname, errno);
    } else if (LockOrUnlock(fname, fd, true) == -1) {
      result = IOError("lock " + fname, errno);
      close(fd);
    } else {
      SetFD_CLOEXEC(fd, nullptr);
      PosixFileLock* my_lock = new PosixFileLock;
      my_lock->fd_ = fd;
      my_lock->filename = fname;
      *lock = my_lock;
    }
    return result;
  }

  virtual Status UnlockFile(FileLock* lock) override {
    PosixFileLock* my_lock = reinterpret_cast<PosixFileLock*>(lock);
    Status result;
    if (LockOrUnlock(my_lock->filename, my_lock->fd_, false) == -1) {
      result = IOError("unlock", errno);
    }
    close(my_lock->fd_);
    delete my_lock;
    return result;
  }

  virtual void Schedule(void (*function)(void* arg1), void* arg,
                        Priority pri = LOW, void* tag = nullptr) override;

  virtual int UnSchedule(void* arg, Priority pri) override;

  virtual void StartThread(void (*function)(void* arg), void* arg) override;

  virtual void WaitForJoin() override;

  virtual unsigned int GetThreadPoolQueueLen(Priority pri = LOW) const override;

  virtual Status GetTestDirectory(std::string* result) override {
    const char* env = getenv("TEST_TMPDIR");
    if (env && env[0] != '\0') {
      *result = env;
    } else {
      char buf[100];
      snprintf(buf, sizeof(buf), "/tmp/rocksdbtest-%d", int(geteuid()));
      *result = buf;
    }
    // Directory may already exist
    CreateDir(*result);
    return Status::OK();
  }

  virtual Status GetThreadList(
      std::vector<ThreadStatus>* thread_list) override {
    assert(thread_status_updater_);
    return thread_status_updater_->GetThreadList(thread_list);
  }

  static uint64_t gettid(pthread_t tid) {
    uint64_t thread_id = 0;
    memcpy(&thread_id, &tid, std::min(sizeof(thread_id), sizeof(tid)));
    return thread_id;
  }

  static uint64_t gettid() {
    pthread_t tid = pthread_self();
    return gettid(tid);
  }

  virtual uint64_t GetThreadID() const override {
    return gettid(pthread_self());
  }

  virtual Status NewLogger(const std::string& fname,
                           shared_ptr<Logger>* result) override {
    FILE* f;
    {
      IOSTATS_TIMER_GUARD(open_nanos);
      f = fopen(fname.c_str(), "w");
    }
    if (f == nullptr) {
      result->reset();
      return IOError(fname, errno);
    } else {
      int fd = fileno(f);
#ifdef ROCKSDB_FALLOCATE_PRESENT
      fallocate(fd, FALLOC_FL_KEEP_SIZE, 0, 4 * 1024 * 1024);
#endif
      SetFD_CLOEXEC(fd, nullptr);
      result->reset(new PosixLogger(f, &PosixEnv::gettid, this));
      return Status::OK();
    }
  }

  virtual uint64_t NowMicros() override {
    struct timeval tv;
    gettimeofday(&tv, nullptr);
    return static_cast<uint64_t>(tv.tv_sec) * 1000000 + tv.tv_usec;
  }

  virtual uint64_t NowNanos() override {
#if defined(OS_LINUX) || defined(OS_FREEBSD)
    struct timespec ts;
    clock_gettime(CLOCK_MONOTONIC, &ts);
    return static_cast<uint64_t>(ts.tv_sec) * 1000000000 + ts.tv_nsec;
#elif defined(__MACH__)
    clock_serv_t cclock;
    mach_timespec_t ts;
    host_get_clock_service(mach_host_self(), CALENDAR_CLOCK, &cclock);
    clock_get_time(cclock, &ts);
    mach_port_deallocate(mach_task_self(), cclock);
    return static_cast<uint64_t>(ts.tv_sec) * 1000000000 + ts.tv_nsec;
#else
    return std::chrono::duration_cast<std::chrono::nanoseconds>(
       std::chrono::steady_clock::now().time_since_epoch()).count();
#endif
  }

  virtual void SleepForMicroseconds(int micros) override { usleep(micros); }

  virtual Status GetHostName(char* name, uint64_t len) override {
    int ret = gethostname(name, static_cast<size_t>(len));
    if (ret < 0) {
      if (errno == EFAULT || errno == EINVAL)
        return Status::InvalidArgument(strerror(errno));
      else
        return IOError("GetHostName", errno);
    }
    return Status::OK();
  }

  virtual Status GetCurrentTime(int64_t* unix_time) override {
    time_t ret = time(nullptr);
    if (ret == (time_t) -1) {
      return IOError("GetCurrentTime", errno);
    }
    *unix_time = (int64_t) ret;
    return Status::OK();
  }

  virtual Status GetAbsolutePath(const std::string& db_path,
                                 std::string* output_path) override {
    if (db_path.find('/') == 0) {
      *output_path = db_path;
      return Status::OK();
    }

    char the_path[256];
    char* ret = getcwd(the_path, 256);
    if (ret == nullptr) {
      return Status::IOError(strerror(errno));
    }

    *output_path = ret;
    return Status::OK();
  }

  // Allow increasing the number of worker threads.
  virtual void SetBackgroundThreads(int num, Priority pri) override {
    assert(pri >= Priority::LOW && pri <= Priority::HIGH);
    thread_pools_[pri].SetBackgroundThreads(num);
  }

  // Allow increasing the number of worker threads.
  virtual void IncBackgroundThreadsIfNeeded(int num, Priority pri) override {
    assert(pri >= Priority::LOW && pri <= Priority::HIGH);
    thread_pools_[pri].IncBackgroundThreadsIfNeeded(num);
  }

  virtual void LowerThreadPoolIOPriority(Priority pool = LOW) override {
    assert(pool >= Priority::LOW && pool <= Priority::HIGH);
#ifdef OS_LINUX
    thread_pools_[pool].LowerIOPriority();
#endif
  }

  virtual std::string TimeToString(uint64_t secondsSince1970) override {
    const time_t seconds = (time_t)secondsSince1970;
    struct tm t;
    int maxsize = 64;
    std::string dummy;
    dummy.reserve(maxsize);
    dummy.resize(maxsize);
    char* p = &dummy[0];
    localtime_r(&seconds, &t);
    snprintf(p, maxsize,
             "%04d/%02d/%02d-%02d:%02d:%02d ",
             t.tm_year + 1900,
             t.tm_mon + 1,
             t.tm_mday,
             t.tm_hour,
             t.tm_min,
             t.tm_sec);
    return dummy;
  }

  EnvOptions OptimizeForLogWrite(const EnvOptions& env_options,
                                 const DBOptions& db_options) const override {
    EnvOptions optimized = env_options;
    optimized.use_mmap_writes = false;
    optimized.bytes_per_sync = db_options.wal_bytes_per_sync;
    // TODO(icanadi) it's faster if fallocate_with_keep_size is false, but it
    // breaks TransactionLogIteratorStallAtLastRecord unit test. Fix the unit
    // test and make this false
    optimized.fallocate_with_keep_size = true;
    return optimized;
  }

  EnvOptions OptimizeForManifestWrite(
      const EnvOptions& env_options) const override {
    EnvOptions optimized = env_options;
    optimized.use_mmap_writes = false;
    optimized.fallocate_with_keep_size = true;
    return optimized;
  }

 private:
  bool checkedDiskForMmap_;
  bool forceMmapOff; // do we override Env options?


  // Returns true iff the named directory exists and is a directory.
  virtual bool DirExists(const std::string& dname) {
    struct stat statbuf;
    if (stat(dname.c_str(), &statbuf) == 0) {
      return S_ISDIR(statbuf.st_mode);
    }
    return false; // stat() failed return false
  }

  bool SupportsFastAllocate(const std::string& path) {
#ifdef ROCKSDB_FALLOCATE_PRESENT
    struct statfs s;
    if (statfs(path.c_str(), &s)){
      return false;
    }
    switch (s.f_type) {
      case EXT4_SUPER_MAGIC:
        return true;
      case XFS_SUPER_MAGIC:
        return true;
      case TMPFS_MAGIC:
        return true;
      default:
        return false;
    }
#else
    return false;
#endif
  }

  size_t page_size_;


  class ThreadPool {
   public:
    ThreadPool()
        : total_threads_limit_(1),
          bgthreads_(0),
          queue_(),
          queue_len_(0),
          exit_all_threads_(false),
          low_io_priority_(false),
          env_(nullptr) {
      PthreadCall("mutex_init", pthread_mutex_init(&mu_, nullptr));
      PthreadCall("cvar_init", pthread_cond_init(&bgsignal_, nullptr));
    }

    ~ThreadPool() {
      assert(bgthreads_.size() == 0U);
    }

    void JoinAllThreads() {
      PthreadCall("lock", pthread_mutex_lock(&mu_));
      assert(!exit_all_threads_);
      exit_all_threads_ = true;
      PthreadCall("signalall", pthread_cond_broadcast(&bgsignal_));
      PthreadCall("unlock", pthread_mutex_unlock(&mu_));
      for (const auto tid : bgthreads_) {
        pthread_join(tid, nullptr);
      }
      bgthreads_.clear();
    }

    void SetHostEnv(Env* env) {
      env_ = env;
    }

    void LowerIOPriority() {
#ifdef OS_LINUX
      PthreadCall("lock", pthread_mutex_lock(&mu_));
      low_io_priority_ = true;
      PthreadCall("unlock", pthread_mutex_unlock(&mu_));
#endif
    }

    // Return true if there is at least one thread needs to terminate.
    bool HasExcessiveThread() {
      return static_cast<int>(bgthreads_.size()) > total_threads_limit_;
    }

    // Return true iff the current thread is the excessive thread to terminate.
    // Always terminate the running thread that is added last, even if there are
    // more than one thread to terminate.
    bool IsLastExcessiveThread(size_t thread_id) {
      return HasExcessiveThread() && thread_id == bgthreads_.size() - 1;
    }

    // Is one of the threads to terminate.
    bool IsExcessiveThread(size_t thread_id) {
      return static_cast<int>(thread_id) >= total_threads_limit_;
    }

    // Return the thread priority.
    // This would allow its member-thread to know its priority.
    Env::Priority GetThreadPriority() {
      return priority_;
    }

    // Set the thread priority.
    void SetThreadPriority(Env::Priority priority) {
      priority_ = priority;
    }

    void BGThread(size_t thread_id) {
      bool low_io_priority = false;
      while (true) {
        // Wait until there is an item that is ready to run
        PthreadCall("lock", pthread_mutex_lock(&mu_));
        // Stop waiting if the thread needs to do work or needs to terminate.
        while (!exit_all_threads_ && !IsLastExcessiveThread(thread_id) &&
               (queue_.empty() || IsExcessiveThread(thread_id))) {
          PthreadCall("wait", pthread_cond_wait(&bgsignal_, &mu_));
        }
        if (exit_all_threads_) { // mechanism to let BG threads exit safely
          PthreadCall("unlock", pthread_mutex_unlock(&mu_));
          break;
        }
        if (IsLastExcessiveThread(thread_id)) {
          // Current thread is the last generated one and is excessive.
          // We always terminate excessive thread in the reverse order of
          // generation time.
          auto terminating_thread = bgthreads_.back();
          pthread_detach(terminating_thread);
          bgthreads_.pop_back();
          if (HasExcessiveThread()) {
            // There is still at least more excessive thread to terminate.
            WakeUpAllThreads();
          }
          PthreadCall("unlock", pthread_mutex_unlock(&mu_));
          break;
        }
        void (*function)(void*) = queue_.front().function;
        void* arg = queue_.front().arg;
        queue_.pop_front();
        queue_len_.store(static_cast<unsigned int>(queue_.size()),
                         std::memory_order_relaxed);

        bool decrease_io_priority = (low_io_priority != low_io_priority_);
        PthreadCall("unlock", pthread_mutex_unlock(&mu_));

#ifdef OS_LINUX
        if (decrease_io_priority) {
          #define IOPRIO_CLASS_SHIFT               (13)
          #define IOPRIO_PRIO_VALUE(class, data)   \
              (((class) << IOPRIO_CLASS_SHIFT) | data)
          // Put schedule into IOPRIO_CLASS_IDLE class (lowest)
          // These system calls only have an effect when used in conjunction
          // with an I/O scheduler that supports I/O priorities. As at
          // kernel 2.6.17 the only such scheduler is the Completely
          // Fair Queuing (CFQ) I/O scheduler.
          // To change scheduler:
          //  echo cfq > /sys/block/<device_name>/queue/schedule
          // Tunables to consider:
          //  /sys/block/<device_name>/queue/slice_idle
          //  /sys/block/<device_name>/queue/slice_sync
          syscall(SYS_ioprio_set,
                  1,  // IOPRIO_WHO_PROCESS
                  0,  // current thread
                  IOPRIO_PRIO_VALUE(3, 0));
          low_io_priority = true;
        }
#else
        (void)decrease_io_priority; // avoid 'unused variable' error
#endif
        (*function)(arg);
      }
    }

    // Helper struct for passing arguments when creating threads.
    struct BGThreadMetadata {
      ThreadPool* thread_pool_;
      size_t thread_id_;  // Thread count in the thread.
      explicit BGThreadMetadata(ThreadPool* thread_pool, size_t thread_id)
          : thread_pool_(thread_pool), thread_id_(thread_id) {}
    };

    static void* BGThreadWrapper(void* arg) {
      BGThreadMetadata* meta = reinterpret_cast<BGThreadMetadata*>(arg);
      size_t thread_id = meta->thread_id_;
      ThreadPool* tp = meta->thread_pool_;
#if ROCKSDB_USING_THREAD_STATUS
      // for thread-status
      ThreadStatusUtil::RegisterThread(tp->env_,
          (tp->GetThreadPriority() == Env::Priority::HIGH ?
              ThreadStatus::HIGH_PRIORITY :
              ThreadStatus::LOW_PRIORITY));
#endif
      delete meta;
      tp->BGThread(thread_id);
#if ROCKSDB_USING_THREAD_STATUS
      ThreadStatusUtil::UnregisterThread();
#endif
      return nullptr;
    }

    void WakeUpAllThreads() {
      PthreadCall("signalall", pthread_cond_broadcast(&bgsignal_));
    }

    void SetBackgroundThreadsInternal(int num, bool allow_reduce) {
      PthreadCall("lock", pthread_mutex_lock(&mu_));
      if (exit_all_threads_) {
        PthreadCall("unlock", pthread_mutex_unlock(&mu_));
        return;
      }
      if (num > total_threads_limit_ ||
          (num < total_threads_limit_ && allow_reduce)) {
        total_threads_limit_ = std::max(1, num);
        WakeUpAllThreads();
        StartBGThreads();
      }
      PthreadCall("unlock", pthread_mutex_unlock(&mu_));
    }

    void IncBackgroundThreadsIfNeeded(int num) {
      SetBackgroundThreadsInternal(num, false);
    }

    void SetBackgroundThreads(int num) {
      SetBackgroundThreadsInternal(num, true);
    }

    void StartBGThreads() {
      // Start background thread if necessary
      while ((int)bgthreads_.size() < total_threads_limit_) {
        pthread_t t;
        PthreadCall(
            "create thread",
            pthread_create(&t, nullptr, &ThreadPool::BGThreadWrapper,
                           new BGThreadMetadata(this, bgthreads_.size())));

        // Set the thread name to aid debugging
#if defined(_GNU_SOURCE) && defined(__GLIBC_PREREQ)
#if __GLIBC_PREREQ(2, 12)
        char name_buf[16];
        snprintf(name_buf, sizeof name_buf, "rocksdb:bg%" ROCKSDB_PRIszt,
                 bgthreads_.size());
        name_buf[sizeof name_buf - 1] = '\0';
        pthread_setname_np(t, name_buf);
#endif
#endif

        bgthreads_.push_back(t);
      }
    }

    void Schedule(void (*function)(void* arg1), void* arg, void* tag) {
      PthreadCall("lock", pthread_mutex_lock(&mu_));

      if (exit_all_threads_) {
        PthreadCall("unlock", pthread_mutex_unlock(&mu_));
        return;
      }

      StartBGThreads();

      // Add to priority queue
      queue_.push_back(BGItem());
      queue_.back().function = function;
      queue_.back().arg = arg;
      queue_.back().tag = tag;
      queue_len_.store(static_cast<unsigned int>(queue_.size()),
                       std::memory_order_relaxed);

      if (!HasExcessiveThread()) {
        // Wake up at least one waiting thread.
        PthreadCall("signal", pthread_cond_signal(&bgsignal_));
      } else {
        // Need to wake up all threads to make sure the one woken
        // up is not the one to terminate.
        WakeUpAllThreads();
      }

      PthreadCall("unlock", pthread_mutex_unlock(&mu_));
    }

    int UnSchedule(void* arg) {
      int count = 0;
      PthreadCall("lock", pthread_mutex_lock(&mu_));

      // Remove from priority queue
      BGQueue::iterator it = queue_.begin();
      while (it != queue_.end()) {
        if (arg == (*it).tag) {
          it = queue_.erase(it);
          count++;
        } else {
          it++;
        }
      }
      queue_len_.store(static_cast<unsigned int>(queue_.size()),
                       std::memory_order_relaxed);
      PthreadCall("unlock", pthread_mutex_unlock(&mu_));
      return count;
    }

    unsigned int GetQueueLen() const {
      return queue_len_.load(std::memory_order_relaxed);
    }

   private:
    // Entry per Schedule() call
    struct BGItem {
      void* arg;
      void (*function)(void*);
      void* tag;
    };
    typedef std::deque<BGItem> BGQueue;

    pthread_mutex_t mu_;
    pthread_cond_t bgsignal_;
    int total_threads_limit_;
    std::vector<pthread_t> bgthreads_;
    BGQueue queue_;
    std::atomic_uint queue_len_;  // Queue length. Used for stats reporting
    bool exit_all_threads_;
    bool low_io_priority_;
    Env::Priority priority_;
    Env* env_;
  };

  std::vector<ThreadPool> thread_pools_;

  pthread_mutex_t mu_;
  std::vector<pthread_t> threads_to_join_;

};

PosixEnv::PosixEnv() : checkedDiskForMmap_(false),
                       forceMmapOff(false),
                       page_size_(getpagesize()),
                       thread_pools_(Priority::TOTAL) {
  PthreadCall("mutex_init", pthread_mutex_init(&mu_, nullptr));
  for (int pool_id = 0; pool_id < Env::Priority::TOTAL; ++pool_id) {
    thread_pools_[pool_id].SetThreadPriority(
        static_cast<Env::Priority>(pool_id));
    // This allows later initializing the thread-local-env of each thread.
    thread_pools_[pool_id].SetHostEnv(this);
  }
  thread_status_updater_ = CreateThreadStatusUpdater();
}

void PosixEnv::Schedule(void (*function)(void* arg1), void* arg, Priority pri,
                        void* tag) {
  assert(pri >= Priority::LOW && pri <= Priority::HIGH);
  thread_pools_[pri].Schedule(function, arg, tag);
}

int PosixEnv::UnSchedule(void* arg, Priority pri) {
  return thread_pools_[pri].UnSchedule(arg);
}

unsigned int PosixEnv::GetThreadPoolQueueLen(Priority pri) const {
  assert(pri >= Priority::LOW && pri <= Priority::HIGH);
  return thread_pools_[pri].GetQueueLen();
}

struct StartThreadState {
  void (*user_function)(void*);
  void* arg;
};

static void* StartThreadWrapper(void* arg) {
  StartThreadState* state = reinterpret_cast<StartThreadState*>(arg);
  state->user_function(state->arg);
  delete state;
  return nullptr;
}

void PosixEnv::StartThread(void (*function)(void* arg), void* arg) {
  pthread_t t;
  StartThreadState* state = new StartThreadState;
  state->user_function = function;
  state->arg = arg;
  PthreadCall("start thread",
              pthread_create(&t, nullptr,  &StartThreadWrapper, state));
  PthreadCall("lock", pthread_mutex_lock(&mu_));
  threads_to_join_.push_back(t);
  PthreadCall("unlock", pthread_mutex_unlock(&mu_));
}

void PosixEnv::WaitForJoin() {
  for (const auto tid : threads_to_join_) {
    pthread_join(tid, nullptr);
  }
  threads_to_join_.clear();
}

}  // namespace

std::string Env::GenerateUniqueId() {
  std::string uuid_file = "/proc/sys/kernel/random/uuid";

  Status s = FileExists(uuid_file);
  if (s.ok()) {
    std::string uuid;
    s = ReadFileToString(this, uuid_file, &uuid);
    if (s.ok()) {
      return uuid;
    }
  }
  // Could not read uuid_file - generate uuid using "nanos-random"
  Random64 r(time(nullptr));
  uint64_t random_uuid_portion =
    r.Uniform(std::numeric_limits<uint64_t>::max());
  uint64_t nanos_uuid_portion = NowNanos();
  char uuid2[200];
  snprintf(uuid2,
           200,
           "%lx-%lx",
           (unsigned long)nanos_uuid_portion,
           (unsigned long)random_uuid_portion);
  return uuid2;
}

Env* Env::Default() {
  static PosixEnv default_env;
  return &default_env;
}

}  // namespace rocksdb
