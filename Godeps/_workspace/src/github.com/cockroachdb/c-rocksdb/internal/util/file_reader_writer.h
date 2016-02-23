//  Copyright (c) 2013, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.
//
// Copyright (c) 2011 The LevelDB Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file. See the AUTHORS file for names of contributors.
#pragma once
#include "rocksdb/env.h"

namespace rocksdb {

class Statistics;
class HistogramImpl;

std::unique_ptr<RandomAccessFile> NewReadaheadRandomAccessFile(
    std::unique_ptr<RandomAccessFile> file, size_t readahead_size);

class SequentialFileReader {
 private:
  std::unique_ptr<SequentialFile> file_;

 public:
  explicit SequentialFileReader(std::unique_ptr<SequentialFile>&& _file)
      : file_(std::move(_file)) {}
  Status Read(size_t n, Slice* result, char* scratch);

  Status Skip(uint64_t n);

  SequentialFile* file() { return file_.get(); }
};

class RandomAccessFileReader {
 private:
  std::unique_ptr<RandomAccessFile> file_;
  Env* env_;
  Statistics* stats_;
  uint32_t hist_type_;
  HistogramImpl* file_read_hist_;

 public:
  explicit RandomAccessFileReader(std::unique_ptr<RandomAccessFile>&& raf,
                                  Env* env = nullptr,
                                  Statistics* stats = nullptr,
                                  uint32_t hist_type = 0,
                                  HistogramImpl* file_read_hist = nullptr)
      : file_(std::move(raf)),
        env_(env),
        stats_(stats),
        hist_type_(hist_type),
        file_read_hist_(file_read_hist) {}

  Status Read(uint64_t offset, size_t n, Slice* result, char* scratch) const;

  RandomAccessFile* file() { return file_.get(); }
};

// Use posix write to write data to a file.
class WritableFileWriter {
 private:
  std::unique_ptr<WritableFile> writable_file_;
  size_t cursize_;          // current size of cached data in buf_
  size_t capacity_;         // max size of buf_
  unique_ptr<char[]> buf_;  // a buffer to cache writes
  uint64_t filesize_;
  bool pending_sync_;
  bool pending_fsync_;
  bool direct_io_;
  uint64_t last_sync_size_;
  uint64_t bytes_per_sync_;
  RateLimiter* rate_limiter_;

 public:
  explicit WritableFileWriter(std::unique_ptr<WritableFile>&& file,
                              const EnvOptions& options)
      : writable_file_(std::move(file)),
        cursize_(0),
        capacity_(65536),
        buf_(new char[capacity_]),
        filesize_(0),
        pending_sync_(false),
        pending_fsync_(false),
        direct_io_(writable_file_->UseDirectIO()),
        last_sync_size_(0),
        bytes_per_sync_(options.bytes_per_sync),
        rate_limiter_(options.rate_limiter) {}

  ~WritableFileWriter() { Flush(); }
  Status Append(const Slice& data);

  Status Flush();

  Status Close();

  Status Sync(bool use_fsync);

  // Sync only the data that was already Flush()ed. Safe to call concurrently
  // with Append() and Flush(). If !writable_file_->IsSyncThreadSafe(),
  // returns NotSupported status.
  Status SyncWithoutFlush(bool use_fsync);

  uint64_t GetFileSize() { return filesize_; }

  Status InvalidateCache(size_t offset, size_t length) {
    return writable_file_->InvalidateCache(offset, length);
  }

  WritableFile* writable_file() const { return writable_file_.get(); }

 private:
  Status RangeSync(off_t offset, off_t nbytes);
  size_t RequestToken(size_t bytes);
  Status SyncInternal(bool use_fsync);
};
}  // namespace rocksdb
