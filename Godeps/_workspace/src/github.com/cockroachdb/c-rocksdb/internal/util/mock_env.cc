//  Copyright (c) 2014, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.
//
// Copyright (c) 2011 The LevelDB Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file. See the AUTHORS file for names of contributors.

#include "util/mock_env.h"
#include "port/sys_time.h"
#include <algorithm>
#include <chrono>
#include "util/rate_limiter.h"
#include "util/random.h"
#include "util/murmurhash.h"

namespace rocksdb {

class MemFile {
 public:
  explicit MemFile(Env* env, const std::string& fn, bool _is_lock_file = false)
      : env_(env),
        fn_(fn),
        refs_(0),
        is_lock_file_(_is_lock_file),
        locked_(false),
        size_(0),
        modified_time_(Now()),
        rnd_(static_cast<uint32_t>(
            MurmurHash(fn.data(), static_cast<int>(fn.size()), 0))),
        fsynced_bytes_(0) {}

  void Ref() {
    MutexLock lock(&mutex_);
    ++refs_;
  }

  bool is_lock_file() const { return is_lock_file_; }

  bool Lock() {
    assert(is_lock_file_);
    MutexLock lock(&mutex_);
    if (locked_) {
      return false;
    } else {
      locked_ = true;
      return true;
    }
  }

  void Unlock() {
    assert(is_lock_file_);
    MutexLock lock(&mutex_);
    locked_ = false;
  }

  void Unref() {
    bool do_delete = false;
    {
      MutexLock lock(&mutex_);
      --refs_;
      assert(refs_ >= 0);
      if (refs_ <= 0) {
        do_delete = true;
      }
    }

    if (do_delete) {
      delete this;
    }
  }

  uint64_t Size() const {
    return size_;
  }

  void Truncate(size_t size) {
    MutexLock lock(&mutex_);
    if (size < size_) {
      data_.resize(size);
      size_ = size;
    }
  }

  void CorruptBuffer() {
    if (fsynced_bytes_ >= size_) {
      return;
    }
    uint64_t buffered_bytes = size_ - fsynced_bytes_;
    uint64_t start =
        fsynced_bytes_ + rnd_.Uniform(static_cast<int>(buffered_bytes));
    uint64_t end = std::min(start + 512, size_.load());
    MutexLock lock(&mutex_);
    for (uint64_t pos = start; pos < end; ++pos) {
      data_[pos] = static_cast<char>(rnd_.Uniform(256));
    }
  }

  Status Read(uint64_t offset, size_t n, Slice* result, char* scratch) const {
    MutexLock lock(&mutex_);
    if (offset > Size()) {
      return Status::IOError("Offset greater than file size.");
    }
    const uint64_t available = Size() - offset;
    if (n > available) {
      n = available;
    }
    if (n == 0) {
      *result = Slice();
      return Status::OK();
    }
    if (scratch) {
      memcpy(scratch, &(data_[offset]), n);
      *result = Slice(scratch, n);
    } else {
      *result = Slice(&(data_[offset]), n);
    }
    return Status::OK();
  }

  Status Append(const Slice& data) {
    MutexLock lock(&mutex_);
    data_.append(data.data(), data.size());
    size_ = data_.size();
    modified_time_ = Now();
    return Status::OK();
  }

  Status Fsync() {
    fsynced_bytes_ = size_.load();
    return Status::OK();
  }

  uint64_t ModifiedTime() const {
    return modified_time_;
  }

 private:
  uint64_t Now() {
    int64_t unix_time;
    auto s = env_->GetCurrentTime(&unix_time);
    assert(s.ok());
    return static_cast<uint64_t>(unix_time);
  }

  // Private since only Unref() should be used to delete it.
  ~MemFile() {
    assert(refs_ == 0);
  }

  // No copying allowed.
  MemFile(const MemFile&);
  void operator=(const MemFile&);

  Env* env_;
  const std::string fn_;
  mutable port::Mutex mutex_;
  int refs_;
  bool is_lock_file_;
  bool locked_;

  // Data written into this file, all bytes before fsynced_bytes are
  // persistent.
  std::string data_;
  std::atomic<uint64_t> size_;
  std::atomic<uint64_t> modified_time_;

  Random rnd_;
  std::atomic<uint64_t> fsynced_bytes_;
};

namespace {

class MockSequentialFile : public SequentialFile {
 public:
  explicit MockSequentialFile(MemFile* file) : file_(file), pos_(0) {
    file_->Ref();
  }

  ~MockSequentialFile() {
    file_->Unref();
  }

  virtual Status Read(size_t n, Slice* result, char* scratch) override {
    Status s = file_->Read(pos_, n, result, scratch);
    if (s.ok()) {
      pos_ += result->size();
    }
    return s;
  }

  virtual Status Skip(uint64_t n) override {
    if (pos_ > file_->Size()) {
      return Status::IOError("pos_ > file_->Size()");
    }
    const size_t available = file_->Size() - pos_;
    if (n > available) {
      n = available;
    }
    pos_ += n;
    return Status::OK();
  }

 private:
  MemFile* file_;
  size_t pos_;
};

class MockRandomAccessFile : public RandomAccessFile {
 public:
  explicit MockRandomAccessFile(MemFile* file) : file_(file) {
    file_->Ref();
  }

  ~MockRandomAccessFile() {
    file_->Unref();
  }

  virtual Status Read(uint64_t offset, size_t n, Slice* result,
                      char* scratch) const override {
    return file_->Read(offset, n, result, scratch);
  }

 private:
  MemFile* file_;
};

class MockWritableFile : public WritableFile {
 public:
  MockWritableFile(MemFile* file, RateLimiter* rate_limiter)
    : file_(file),
      rate_limiter_(rate_limiter) {
    file_->Ref();
  }

  ~MockWritableFile() {
    file_->Unref();
  }

  virtual Status Append(const Slice& data) override {
    uint64_t bytes_written = 0;
    while (bytes_written < data.size()) {
      auto bytes = RequestToken(data.size() - bytes_written);
      Status s = file_->Append(Slice(data.data() + bytes_written, bytes));
      if (!s.ok()) {
        return s;
      }
      bytes_written += bytes;
    }
    return Status::OK();
  }

  virtual Status Close() override { return file_->Fsync(); }

  virtual Status Flush() override { return Status::OK(); }

  virtual Status Sync() override { return file_->Fsync(); }

  virtual uint64_t GetFileSize() override { return file_->Size(); }

 private:
  inline size_t RequestToken(size_t bytes) {
    if (rate_limiter_ && io_priority_ < Env::IO_TOTAL) {
      bytes = std::min(bytes,
          static_cast<size_t>(rate_limiter_->GetSingleBurstBytes()));
      rate_limiter_->Request(bytes, io_priority_);
    }
    return bytes;
  }

  MemFile* file_;
  RateLimiter* rate_limiter_;
};

class MockEnvDirectory : public Directory {
 public:
  virtual Status Fsync() override { return Status::OK(); }
};

class MockEnvFileLock : public FileLock {
 public:
  explicit MockEnvFileLock(const std::string& fname)
    : fname_(fname) {}

  std::string FileName() const {
    return fname_;
  }

 private:
  const std::string fname_;
};

class TestMemLogger : public Logger {
 private:
  std::unique_ptr<WritableFile> file_;
  std::atomic_size_t log_size_;
  static const uint64_t flush_every_seconds_ = 5;
  std::atomic_uint_fast64_t last_flush_micros_;
  Env* env_;
  bool flush_pending_;

 public:
  TestMemLogger(std::unique_ptr<WritableFile> f, Env* env,
                const InfoLogLevel log_level = InfoLogLevel::ERROR_LEVEL)
      : Logger(log_level),
        file_(std::move(f)),
        log_size_(0),
        last_flush_micros_(0),
        env_(env),
        flush_pending_(false) {}
  virtual ~TestMemLogger() {
  }

  virtual void Flush() override {
    if (flush_pending_) {
      flush_pending_ = false;
    }
    last_flush_micros_ = env_->NowMicros();
  }

  using Logger::Logv;
  virtual void Logv(const char* format, va_list ap) override {
    // We try twice: the first time with a fixed-size stack allocated buffer,
    // and the second time with a much larger dynamically allocated buffer.
    char buffer[500];
    for (int iter = 0; iter < 2; iter++) {
      char* base;
      int bufsize;
      if (iter == 0) {
        bufsize = sizeof(buffer);
        base = buffer;
      } else {
        bufsize = 30000;
        base = new char[bufsize];
      }
      char* p = base;
      char* limit = base + bufsize;

      struct timeval now_tv;
      gettimeofday(&now_tv, nullptr);
      const time_t seconds = now_tv.tv_sec;
      struct tm t;
      localtime_r(&seconds, &t);
      p += snprintf(p, limit - p,
                    "%04d/%02d/%02d-%02d:%02d:%02d.%06d ",
                    t.tm_year + 1900,
                    t.tm_mon + 1,
                    t.tm_mday,
                    t.tm_hour,
                    t.tm_min,
                    t.tm_sec,
                    static_cast<int>(now_tv.tv_usec));

      // Print the message
      if (p < limit) {
        va_list backup_ap;
        va_copy(backup_ap, ap);
        p += vsnprintf(p, limit - p, format, backup_ap);
        va_end(backup_ap);
      }

      // Truncate to available space if necessary
      if (p >= limit) {
        if (iter == 0) {
          continue;       // Try again with larger buffer
        } else {
          p = limit - 1;
        }
      }

      // Add newline if necessary
      if (p == base || p[-1] != '\n') {
        *p++ = '\n';
      }

      assert(p <= limit);
      const size_t write_size = p - base;

      file_->Append(Slice(base, write_size));
      flush_pending_ = true;
      log_size_ += write_size;
      uint64_t now_micros = static_cast<uint64_t>(now_tv.tv_sec) * 1000000 +
        now_tv.tv_usec;
      if (now_micros - last_flush_micros_ >= flush_every_seconds_ * 1000000) {
        flush_pending_ = false;
        last_flush_micros_ = now_micros;
      }
      if (base != buffer) {
        delete[] base;
      }
      break;
    }
  }
  size_t GetLogFileSize() const override { return log_size_; }
};

}  // Anonymous namespace

MockEnv::MockEnv(Env* base_env) : EnvWrapper(base_env), fake_sleep_micros_(0) {}

MockEnv::~MockEnv() {
  for (FileSystem::iterator i = file_map_.begin(); i != file_map_.end(); ++i) {
    i->second->Unref();
  }
}

  // Partial implementation of the Env interface.
Status MockEnv::NewSequentialFile(const std::string& fname,
                                     unique_ptr<SequentialFile>* result,
                                     const EnvOptions& soptions) {
  auto fn = NormalizePath(fname);
  MutexLock lock(&mutex_);
  if (file_map_.find(fn) == file_map_.end()) {
    *result = NULL;
    return Status::IOError(fn, "File not found");
  }
  auto* f = file_map_[fn];
  if (f->is_lock_file()) {
    return Status::InvalidArgument(fn, "Cannot open a lock file.");
  }
  result->reset(new MockSequentialFile(f));
  return Status::OK();
}

Status MockEnv::NewRandomAccessFile(const std::string& fname,
                                       unique_ptr<RandomAccessFile>* result,
                                       const EnvOptions& soptions) {
  auto fn = NormalizePath(fname);
  MutexLock lock(&mutex_);
  if (file_map_.find(fn) == file_map_.end()) {
    *result = NULL;
    return Status::IOError(fn, "File not found");
  }
  auto* f = file_map_[fn];
  if (f->is_lock_file()) {
    return Status::InvalidArgument(fn, "Cannot open a lock file.");
  }
  result->reset(new MockRandomAccessFile(f));
  return Status::OK();
}

Status MockEnv::NewWritableFile(const std::string& fname,
                                   unique_ptr<WritableFile>* result,
                                   const EnvOptions& env_options) {
  auto fn = NormalizePath(fname);
  MutexLock lock(&mutex_);
  if (file_map_.find(fn) != file_map_.end()) {
    DeleteFileInternal(fn);
  }
  MemFile* file = new MemFile(this, fn, false);
  file->Ref();
  file_map_[fn] = file;

  result->reset(new MockWritableFile(file, env_options.rate_limiter));
  return Status::OK();
}

Status MockEnv::NewDirectory(const std::string& name,
                                unique_ptr<Directory>* result) {
  result->reset(new MockEnvDirectory());
  return Status::OK();
}

Status MockEnv::FileExists(const std::string& fname) {
  auto fn = NormalizePath(fname);
  MutexLock lock(&mutex_);
  if (file_map_.find(fn) != file_map_.end()) {
    // File exists
    return Status::OK();
  }
  // Now also check if fn exists as a dir
  for (const auto& iter : file_map_) {
    const std::string& filename = iter.first;
    if (filename.size() >= fn.size() + 1 &&
        filename[fn.size()] == '/' &&
        Slice(filename).starts_with(Slice(fn))) {
      return Status::OK();
    }
  }
  return Status::NotFound();
}

Status MockEnv::GetChildren(const std::string& dir,
                               std::vector<std::string>* result) {
  auto d = NormalizePath(dir);
  {
    MutexLock lock(&mutex_);
    result->clear();
    for (const auto& iter : file_map_) {
      const std::string& filename = iter.first;

      if (filename.size() >= d.size() + 1 && filename[d.size()] == '/' &&
          Slice(filename).starts_with(Slice(d))) {
        size_t next_slash = filename.find('/', d.size() + 1);
        if (next_slash != std::string::npos) {
          result->push_back(filename.substr(
                d.size() + 1, next_slash - d.size() - 1));
        } else {
          result->push_back(filename.substr(d.size() + 1));
        }
      }
    }
  }
  result->erase(std::unique(result->begin(), result->end()), result->end());
  return Status::OK();
}

void MockEnv::DeleteFileInternal(const std::string& fname) {
  assert(fname == NormalizePath(fname));
  const auto& pair = file_map_.find(fname);
  if (pair != file_map_.end()) {
    pair->second->Unref();
    file_map_.erase(fname);
  }
}

Status MockEnv::DeleteFile(const std::string& fname) {
  auto fn = NormalizePath(fname);
  MutexLock lock(&mutex_);
  if (file_map_.find(fn) == file_map_.end()) {
    return Status::IOError(fn, "File not found");
  }

  DeleteFileInternal(fn);
  return Status::OK();
}

Status MockEnv::CreateDir(const std::string& dirname) {
  return Status::OK();
}

Status MockEnv::CreateDirIfMissing(const std::string& dirname) {
  return Status::OK();
}

Status MockEnv::DeleteDir(const std::string& dirname) {
  return Status::OK();
}

Status MockEnv::GetFileSize(const std::string& fname, uint64_t* file_size) {
  auto fn = NormalizePath(fname);
  MutexLock lock(&mutex_);
  auto iter = file_map_.find(fn);
  if (iter == file_map_.end()) {
    return Status::IOError(fn, "File not found");
  }

  *file_size = iter->second->Size();
  return Status::OK();
}

Status MockEnv::GetFileModificationTime(const std::string& fname,
                                           uint64_t* time) {
  auto fn = NormalizePath(fname);
  MutexLock lock(&mutex_);
  auto iter = file_map_.find(fn);
  if (iter == file_map_.end()) {
    return Status::IOError(fn, "File not found");
  }
  *time = iter->second->ModifiedTime();
  return Status::OK();
}

Status MockEnv::RenameFile(const std::string& src, const std::string& dest) {
  auto s = NormalizePath(src);
  auto t = NormalizePath(dest);
  MutexLock lock(&mutex_);
  if (file_map_.find(s) == file_map_.end()) {
    return Status::IOError(s, "File not found");
  }

  DeleteFileInternal(t);
  file_map_[t] = file_map_[s];
  file_map_.erase(s);
  return Status::OK();
}

Status MockEnv::LinkFile(const std::string& src, const std::string& dest) {
  auto s = NormalizePath(src);
  auto t = NormalizePath(dest);
  MutexLock lock(&mutex_);
  if (file_map_.find(s) == file_map_.end()) {
    return Status::IOError(s, "File not found");
  }

  DeleteFileInternal(t);
  file_map_[t] = file_map_[s];
  return Status::OK();
}

Status MockEnv::NewLogger(const std::string& fname,
                             shared_ptr<Logger>* result) {
  auto fn = NormalizePath(fname);
  MutexLock lock(&mutex_);
  auto iter = file_map_.find(fn);
  MemFile* file = nullptr;
  if (iter == file_map_.end()) {
    file = new MemFile(this, fn, false);
    file->Ref();
    file_map_[fn] = file;
  } else {
    file = iter->second;
  }
  std::unique_ptr<WritableFile> f(new MockWritableFile(file, nullptr));
  result->reset(new TestMemLogger(std::move(f), this));
  return Status::OK();
}

Status MockEnv::LockFile(const std::string& fname, FileLock** flock) {
  auto fn = NormalizePath(fname);
  {
    MutexLock lock(&mutex_);
    if (file_map_.find(fn) != file_map_.end()) {
      if (!file_map_[fn]->is_lock_file()) {
        return Status::InvalidArgument(fname, "Not a lock file.");
      }
      if (!file_map_[fn]->Lock()) {
        return Status::IOError(fn, "Lock is already held.");
      }
    } else {
      auto* file = new MemFile(this, fn, true);
      file->Ref();
      file->Lock();
      file_map_[fn] = file;
    }
  }
  *flock = new MockEnvFileLock(fn);
  return Status::OK();
}

Status MockEnv::UnlockFile(FileLock* flock) {
  std::string fn = dynamic_cast<MockEnvFileLock*>(flock)->FileName();
  {
    MutexLock lock(&mutex_);
    if (file_map_.find(fn) != file_map_.end()) {
      if (!file_map_[fn]->is_lock_file()) {
        return Status::InvalidArgument(fn, "Not a lock file.");
      }
      file_map_[fn]->Unlock();
    }
  }
  delete flock;
  return Status::OK();
}

Status MockEnv::GetTestDirectory(std::string* path) {
  *path = "/test";
  return Status::OK();
}

Status MockEnv::GetCurrentTime(int64_t* unix_time) {
  auto s = EnvWrapper::GetCurrentTime(unix_time);
  *unix_time += fake_sleep_micros_.load() / (1000 * 1000);
  return s;
}

uint64_t MockEnv::NowMicros() {
  return EnvWrapper::NowMicros() + fake_sleep_micros_.load();
}

uint64_t MockEnv::NowNanos() {
  return EnvWrapper::NowNanos() + fake_sleep_micros_.load() * 1000;
}

// Non-virtual functions, specific to MockEnv
Status MockEnv::Truncate(const std::string& fname, size_t size) {
  auto fn = NormalizePath(fname);
  MutexLock lock(&mutex_);
  auto iter = file_map_.find(fn);
  if (iter == file_map_.end()) {
    return Status::IOError(fn, "File not found");
  }
  iter->second->Truncate(size);
  return Status::OK();
}

Status MockEnv::CorruptBuffer(const std::string& fname) {
  auto fn = NormalizePath(fname);
  MutexLock lock(&mutex_);
  auto iter = file_map_.find(fn);
  if (iter == file_map_.end()) {
    return Status::IOError(fn, "File not found");
  }
  iter->second->CorruptBuffer();
  return Status::OK();
}

std::string MockEnv::NormalizePath(const std::string path) {
  std::string dst;
  for (auto c : path) {
    if (!dst.empty() && c == '/' && dst.back() == '/') {
      continue;
    }
    dst.push_back(c);
  }
  return dst;
}

void MockEnv::FakeSleepForMicroseconds(int64_t micros) {
  fake_sleep_micros_.fetch_add(micros);
}

}  // namespace rocksdb
