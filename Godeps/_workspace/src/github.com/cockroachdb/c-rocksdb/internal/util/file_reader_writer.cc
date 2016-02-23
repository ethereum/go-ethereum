//  Copyright (c) 2013, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.
//
// Copyright (c) 2011 The LevelDB Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file. See the AUTHORS file for names of contributors.

#include "util/file_reader_writer.h"

#include <algorithm>
#include <mutex>

#include "port/port.h"
#include "util/histogram.h"
#include "util/iostats_context_imp.h"
#include "util/random.h"
#include "util/rate_limiter.h"
#include "util/sync_point.h"

namespace rocksdb {
Status SequentialFileReader::Read(size_t n, Slice* result, char* scratch) {
  Status s = file_->Read(n, result, scratch);
  IOSTATS_ADD(bytes_read, result->size());
  return s;
}

Status SequentialFileReader::Skip(uint64_t n) { return file_->Skip(n); }

Status RandomAccessFileReader::Read(uint64_t offset, size_t n, Slice* result,
                                    char* scratch) const {
  Status s;
  uint64_t elapsed = 0;
  {
    StopWatch sw(env_, stats_, hist_type_,
                 (stats_ != nullptr) ? &elapsed : nullptr);
    IOSTATS_TIMER_GUARD(read_nanos);
    s = file_->Read(offset, n, result, scratch);
    IOSTATS_ADD_IF_POSITIVE(bytes_read, result->size());
  }
  if (stats_ != nullptr && file_read_hist_ != nullptr) {
    file_read_hist_->Add(elapsed);
  }
  return s;
}

Status WritableFileWriter::Append(const Slice& data) {
  const char* src = data.data();
  size_t left = data.size();
  Status s;
  pending_sync_ = true;
  pending_fsync_ = true;

  TEST_KILL_RANDOM(rocksdb_kill_odds * REDUCE_ODDS2);

  {
    IOSTATS_TIMER_GUARD(prepare_write_nanos);
    TEST_SYNC_POINT("WritableFileWriter::Append:BeforePrepareWrite");
    writable_file_->PrepareWrite(static_cast<size_t>(GetFileSize()), left);
  }
  // if there is no space in the cache, then flush
  if (cursize_ + left > capacity_) {
    s = Flush();
    if (!s.ok()) {
      return s;
    }
    // Increase the buffer size, but capped at 1MB
    if (capacity_ < (1 << 20)) {
      capacity_ *= 2;
      buf_.reset(new char[capacity_]);
    }
    assert(cursize_ == 0);
  }

  // if the write fits into the cache, then write to cache
  // otherwise do a write() syscall to write to OS buffers.
  if (cursize_ + left <= capacity_) {
    memcpy(buf_.get() + cursize_, src, left);
    cursize_ += left;
  } else {
    while (left != 0) {
      size_t size = RequestToken(left);
      {
        IOSTATS_TIMER_GUARD(write_nanos);
        s = writable_file_->Append(Slice(src, size));
        if (!s.ok()) {
          return s;
        }
      }
      IOSTATS_ADD(bytes_written, size);
      TEST_KILL_RANDOM(rocksdb_kill_odds);

      left -= size;
      src += size;
    }
  }
  TEST_KILL_RANDOM(rocksdb_kill_odds);
  filesize_ += data.size();
  return Status::OK();
}

Status WritableFileWriter::Close() {
  Status s;
  s = Flush();  // flush cache to OS
  if (!s.ok()) {
    return s;
  }

  TEST_KILL_RANDOM(rocksdb_kill_odds);
  return writable_file_->Close();
}

// write out the cached data to the OS cache
Status WritableFileWriter::Flush() {
  TEST_KILL_RANDOM(rocksdb_kill_odds * REDUCE_ODDS2);
  size_t left = cursize_;
  char* src = buf_.get();
  while (left != 0) {
    size_t size = RequestToken(left);
    {
      IOSTATS_TIMER_GUARD(write_nanos);
      TEST_SYNC_POINT("WritableFileWriter::Flush:BeforeAppend");
      Status s = writable_file_->Append(Slice(src, size));
      if (!s.ok()) {
        return s;
      }
    }
    IOSTATS_ADD(bytes_written, size);
    TEST_KILL_RANDOM(rocksdb_kill_odds * REDUCE_ODDS2);
    left -= size;
    src += size;
  }
  cursize_ = 0;

  writable_file_->Flush();

  // sync OS cache to disk for every bytes_per_sync_
  // TODO: give log file and sst file different options (log
  // files could be potentially cached in OS for their whole
  // life time, thus we might not want to flush at all).

  // We try to avoid sync to the last 1MB of data. For two reasons:
  // (1) avoid rewrite the same page that is modified later.
  // (2) for older version of OS, write can block while writing out
  //     the page.
  // Xfs does neighbor page flushing outside of the specified ranges. We
  // need to make sure sync range is far from the write offset.
  if (!direct_io_ && bytes_per_sync_) {
    uint64_t kBytesNotSyncRange = 1024 * 1024;  // recent 1MB is not synced.
    uint64_t kBytesAlignWhenSync = 4 * 1024;    // Align 4KB.
    if (filesize_ > kBytesNotSyncRange) {
      uint64_t offset_sync_to = filesize_ - kBytesNotSyncRange;
      offset_sync_to -= offset_sync_to % kBytesAlignWhenSync;
      assert(offset_sync_to >= last_sync_size_);
      if (offset_sync_to > 0 &&
          offset_sync_to - last_sync_size_ >= bytes_per_sync_) {
        RangeSync(last_sync_size_, offset_sync_to - last_sync_size_);
        last_sync_size_ = offset_sync_to;
      }
    }
  }

  return Status::OK();
}

Status WritableFileWriter::Sync(bool use_fsync) {
  Status s = Flush();
  if (!s.ok()) {
    return s;
  }
  TEST_KILL_RANDOM(rocksdb_kill_odds);
  if (!direct_io_ && pending_sync_) {
    s = SyncInternal(use_fsync);
    if (!s.ok()) {
      return s;
    }
  }
  TEST_KILL_RANDOM(rocksdb_kill_odds);
  pending_sync_ = false;
  if (use_fsync) {
    pending_fsync_ = false;
  }
  return Status::OK();
}

Status WritableFileWriter::SyncWithoutFlush(bool use_fsync) {
  if (!writable_file_->IsSyncThreadSafe()) {
    return Status::NotSupported(
      "Can't WritableFileWriter::SyncWithoutFlush() because "
      "WritableFile::IsSyncThreadSafe() is false");
  }
  TEST_SYNC_POINT("WritableFileWriter::SyncWithoutFlush:1");
  Status s = SyncInternal(use_fsync);
  TEST_SYNC_POINT("WritableFileWriter::SyncWithoutFlush:2");
  return s;
}

Status WritableFileWriter::SyncInternal(bool use_fsync) {
  Status s;
  IOSTATS_TIMER_GUARD(fsync_nanos);
  TEST_SYNC_POINT("WritableFileWriter::SyncInternal:0");
  if (use_fsync) {
    s = writable_file_->Fsync();
  } else {
    s = writable_file_->Sync();
  }
  return s;
}

Status WritableFileWriter::RangeSync(off_t offset, off_t nbytes) {
  IOSTATS_TIMER_GUARD(range_sync_nanos);
  TEST_SYNC_POINT("WritableFileWriter::RangeSync:0");
  return writable_file_->RangeSync(offset, nbytes);
}

size_t WritableFileWriter::RequestToken(size_t bytes) {
  Env::IOPriority io_priority;
  if (rate_limiter_&&(io_priority = writable_file_->GetIOPriority()) <
      Env::IO_TOTAL) {
    bytes = std::min(bytes,
                     static_cast<size_t>(rate_limiter_->GetSingleBurstBytes()));
    rate_limiter_->Request(bytes, io_priority);
  }
  return bytes;
}

namespace {
class ReadaheadRandomAccessFile : public RandomAccessFile {
 public:
  ReadaheadRandomAccessFile(std::unique_ptr<RandomAccessFile> file,
                            size_t readahead_size)
      : file_(std::move(file)),
        readahead_size_(readahead_size),
        buffer_(new char[readahead_size_]),
        buffer_offset_(0),
        buffer_len_(0) {}

  virtual Status Read(uint64_t offset, size_t n, Slice* result,
                      char* scratch) const override {
    if (n >= readahead_size_) {
      return file_->Read(offset, n, result, scratch);
    }

    std::unique_lock<std::mutex> lk(lock_);

    size_t copied = 0;
    // if offset between [buffer_offset_, buffer_offset_ + buffer_len>
    if (offset >= buffer_offset_ && offset < buffer_len_ + buffer_offset_) {
      uint64_t offset_in_buffer = offset - buffer_offset_;
      copied = std::min(static_cast<uint64_t>(buffer_len_) - offset_in_buffer,
                        static_cast<uint64_t>(n));
      memcpy(scratch, buffer_.get() + offset_in_buffer, copied);
      if (copied == n) {
        // fully cached
        *result = Slice(scratch, n);
        return Status::OK();
      }
    }
    Slice readahead_result;
    Status s = file_->Read(offset + copied, readahead_size_, &readahead_result,
                           buffer_.get());
    if (!s.ok()) {
      return s;
    }

    auto left_to_copy = std::min(readahead_result.size(), n - copied);
    memcpy(scratch + copied, readahead_result.data(), left_to_copy);
    *result = Slice(scratch, copied + left_to_copy);

    if (readahead_result.data() == buffer_.get()) {
      buffer_offset_ = offset + copied;
      buffer_len_ = readahead_result.size();
    } else {
      buffer_len_ = 0;
    }

    return Status::OK();
  }

  virtual size_t GetUniqueId(char* id, size_t max_size) const override {
    return file_->GetUniqueId(id, max_size);
  }

  virtual void Hint(AccessPattern pattern) override { file_->Hint(pattern); }

  virtual Status InvalidateCache(size_t offset, size_t length) override {
    return file_->InvalidateCache(offset, length);
  }

 private:
  std::unique_ptr<RandomAccessFile> file_;
  size_t readahead_size_;

  mutable std::mutex lock_;
  mutable std::unique_ptr<char[]> buffer_;
  mutable uint64_t buffer_offset_;
  mutable size_t buffer_len_;
};
}  // namespace

std::unique_ptr<RandomAccessFile> NewReadaheadRandomAccessFile(
    std::unique_ptr<RandomAccessFile> file, size_t readahead_size) {
  std::unique_ptr<ReadaheadRandomAccessFile> wrapped_file(
      new ReadaheadRandomAccessFile(std::move(file), readahead_size));
  return std::move(wrapped_file);
}

}  // namespace rocksdb
