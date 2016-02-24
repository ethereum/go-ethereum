// Copyright (c) 2011 The LevelDB Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file. See the AUTHORS file for names of contributors.

#include "rocksdb/env.h"
#include "rocksdb/status.h"
#include "port/port.h"
#include "util/mutexlock.h"
#include <map>
#include <string.h>
#include <string>
#include <vector>

namespace rocksdb {

#ifndef ROCKSDB_LITE

namespace {

std::string NormalizeFileName(const std::string fname) {
  if (fname.find("//") == std::string::npos) {
    return fname;
  }
  std::string out_name = "";
  bool is_slash = false;
  for (char c : fname) {
    if (c == '/' && is_slash) {
      continue;
    }
    out_name.append(1, c);
    if (c == '/') {
      is_slash = true;
    } else {
      is_slash = false;
    }
  }
  return out_name;
}

class FileState {
 public:
  // FileStates are reference counted. The initial reference count is zero
  // and the caller must call Ref() at least once.
  FileState() : refs_(0), size_(0) {}

  // Increase the reference count.
  void Ref() {
    MutexLock lock(&refs_mutex_);
    ++refs_;
  }

  // Decrease the reference count. Delete if this is the last reference.
  void Unref() {
    bool do_delete = false;

    {
      MutexLock lock(&refs_mutex_);
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

  uint64_t Size() const { return size_; }

  Status Read(uint64_t offset, size_t n, Slice* result, char* scratch) const {
    if (offset > size_) {
      return Status::IOError("Offset greater than file size.");
    }
    const uint64_t available = size_ - offset;
    if (n > available) {
      n = available;
    }
    if (n == 0) {
      *result = Slice();
      return Status::OK();
    }

    size_t block = offset / kBlockSize;
    size_t block_offset = offset % kBlockSize;

    if (n <= kBlockSize - block_offset) {
      // The requested bytes are all in the first block.
      *result = Slice(blocks_[block] + block_offset, n);
      return Status::OK();
    }

    size_t bytes_to_copy = n;
    char* dst = scratch;

    while (bytes_to_copy > 0) {
      size_t avail = kBlockSize - block_offset;
      if (avail > bytes_to_copy) {
        avail = bytes_to_copy;
      }
      memcpy(dst, blocks_[block] + block_offset, avail);

      bytes_to_copy -= avail;
      dst += avail;
      block++;
      block_offset = 0;
    }

    *result = Slice(scratch, n);
    return Status::OK();
  }

  Status Append(const Slice& data) {
    const char* src = data.data();
    size_t src_len = data.size();

    while (src_len > 0) {
      size_t avail;
      size_t offset = size_ % kBlockSize;

      if (offset != 0) {
        // There is some room in the last block.
        avail = kBlockSize - offset;
      } else {
        // No room in the last block; push new one.
        blocks_.push_back(new char[kBlockSize]);
        avail = kBlockSize;
      }

      if (avail > src_len) {
        avail = src_len;
      }
      memcpy(blocks_.back() + offset, src, avail);
      src_len -= avail;
      src += avail;
      size_ += avail;
    }

    return Status::OK();
  }

 private:
  // Private since only Unref() should be used to delete it.
  ~FileState() {
    for (std::vector<char*>::iterator i = blocks_.begin(); i != blocks_.end();
         ++i) {
      delete [] *i;
    }
  }

  // No copying allowed.
  FileState(const FileState&);
  void operator=(const FileState&);

  port::Mutex refs_mutex_;
  int refs_;  // Protected by refs_mutex_;

  // The following fields are not protected by any mutex. They are only mutable
  // while the file is being written, and concurrent access is not allowed
  // to writable files.
  std::vector<char*> blocks_;
  uint64_t size_;

  enum { kBlockSize = 8 * 1024 };
};

class SequentialFileImpl : public SequentialFile {
 public:
  explicit SequentialFileImpl(FileState* file) : file_(file), pos_(0) {
    file_->Ref();
  }

  ~SequentialFileImpl() {
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
  FileState* file_;
  size_t pos_;
};

class RandomAccessFileImpl : public RandomAccessFile {
 public:
  explicit RandomAccessFileImpl(FileState* file) : file_(file) {
    file_->Ref();
  }

  ~RandomAccessFileImpl() {
    file_->Unref();
  }

  virtual Status Read(uint64_t offset, size_t n, Slice* result,
                      char* scratch) const override {
    return file_->Read(offset, n, result, scratch);
  }

 private:
  FileState* file_;
};

class WritableFileImpl : public WritableFile {
 public:
  WritableFileImpl(FileState* file) : file_(file) {
    file_->Ref();
  }

  ~WritableFileImpl() {
    file_->Unref();
  }

  virtual Status Append(const Slice& data) override {
    return file_->Append(data);
  }

  virtual Status Close() override { return Status::OK(); }
  virtual Status Flush() override { return Status::OK(); }
  virtual Status Sync() override { return Status::OK(); }

 private:
  FileState* file_;
};

class InMemoryDirectory : public Directory {
 public:
  virtual Status Fsync() override { return Status::OK(); }
};

class InMemoryEnv : public EnvWrapper {
 public:
  explicit InMemoryEnv(Env* base_env) : EnvWrapper(base_env) { }

  virtual ~InMemoryEnv() {
    for (FileSystem::iterator i = file_map_.begin(); i != file_map_.end(); ++i){
      i->second->Unref();
    }
  }

  // Partial implementation of the Env interface.
  virtual Status NewSequentialFile(const std::string& fname,
                                   unique_ptr<SequentialFile>* result,
                                   const EnvOptions& soptions) override {
    std::string nfname = NormalizeFileName(fname);
    MutexLock lock(&mutex_);
    if (file_map_.find(fname) == file_map_.end()) {
      *result = NULL;
      return Status::IOError(fname, "File not found");
    }

    result->reset(new SequentialFileImpl(file_map_[nfname]));
    return Status::OK();
  }

  virtual Status NewRandomAccessFile(const std::string& fname,
                                     unique_ptr<RandomAccessFile>* result,
                                     const EnvOptions& soptions) override {
    std::string nfname = NormalizeFileName(fname);
    MutexLock lock(&mutex_);
    if (file_map_.find(nfname) == file_map_.end()) {
      *result = NULL;
      return Status::IOError(fname, "File not found");
    }

    result->reset(new RandomAccessFileImpl(file_map_[nfname]));
    return Status::OK();
  }

  virtual Status NewWritableFile(const std::string& fname,
                                 unique_ptr<WritableFile>* result,
                                 const EnvOptions& soptions) override {
    std::string nfname = NormalizeFileName(fname);
    MutexLock lock(&mutex_);
    if (file_map_.find(nfname) != file_map_.end()) {
      DeleteFileInternal(nfname);
    }

    FileState* file = new FileState();
    file->Ref();
    file_map_[nfname] = file;

    result->reset(new WritableFileImpl(file));
    return Status::OK();
  }

  virtual Status NewDirectory(const std::string& name,
                              unique_ptr<Directory>* result) override {
    result->reset(new InMemoryDirectory());
    return Status::OK();
  }

  virtual Status FileExists(const std::string& fname) override {
    std::string nfname = NormalizeFileName(fname);
    MutexLock lock(&mutex_);
    if (file_map_.find(nfname) != file_map_.end()) {
      return Status::OK();
    } else {
      return Status::NotFound();
    }
  }

  virtual Status GetChildren(const std::string& dir,
                             std::vector<std::string>* result) override {
    MutexLock lock(&mutex_);
    result->clear();

    for (FileSystem::iterator i = file_map_.begin(); i != file_map_.end(); ++i){
      const std::string& filename = i->first;

      if (filename.size() >= dir.size() + 1 && filename[dir.size()] == '/' &&
          Slice(filename).starts_with(Slice(dir))) {
        result->push_back(filename.substr(dir.size() + 1));
      }
    }

    return Status::OK();
  }

  void DeleteFileInternal(const std::string& fname) {
    if (file_map_.find(fname) == file_map_.end()) {
      return;
    }

    file_map_[fname]->Unref();
    file_map_.erase(fname);
  }

  virtual Status DeleteFile(const std::string& fname) override {
    std::string nfname = NormalizeFileName(fname);
    MutexLock lock(&mutex_);
    if (file_map_.find(nfname) == file_map_.end()) {
      return Status::IOError(fname, "File not found");
    }

    DeleteFileInternal(nfname);
    return Status::OK();
  }

  virtual Status CreateDir(const std::string& dirname) override {
    return Status::OK();
  }

  virtual Status CreateDirIfMissing(const std::string& dirname) override {
    return Status::OK();
  }

  virtual Status DeleteDir(const std::string& dirname) override {
    return Status::OK();
  }

  virtual Status GetFileSize(const std::string& fname,
                             uint64_t* file_size) override {
    std::string nfname = NormalizeFileName(fname);
    MutexLock lock(&mutex_);

    if (file_map_.find(nfname) == file_map_.end()) {
      return Status::IOError(fname, "File not found");
    }

    *file_size = file_map_[nfname]->Size();
    return Status::OK();
  }

  virtual Status GetFileModificationTime(const std::string& fname,
                                         uint64_t* time) override {
    return Status::NotSupported("getFileMTime", "Not supported in MemEnv");
  }

  virtual Status RenameFile(const std::string& src,
                            const std::string& dest) override {
    std::string nsrc = NormalizeFileName(src);
    std::string ndest = NormalizeFileName(dest);
    MutexLock lock(&mutex_);
    if (file_map_.find(nsrc) == file_map_.end()) {
      return Status::IOError(src, "File not found");
    }

    DeleteFileInternal(dest);
    file_map_[ndest] = file_map_[nsrc];
    file_map_.erase(nsrc);
    return Status::OK();
  }

  virtual Status LockFile(const std::string& fname, FileLock** lock) override {
    *lock = new FileLock;
    return Status::OK();
  }

  virtual Status UnlockFile(FileLock* lock) override {
    delete lock;
    return Status::OK();
  }

  virtual Status GetTestDirectory(std::string* path) override {
    *path = "/test";
    return Status::OK();
  }

 private:
  // Map from filenames to FileState objects, representing a simple file system.
  typedef std::map<std::string, FileState*> FileSystem;
  port::Mutex mutex_;
  FileSystem file_map_;  // Protected by mutex_.
};

}  // namespace

Env* NewMemEnv(Env* base_env) {
  return new InMemoryEnv(base_env);
}

#else  // ROCKSDB_LITE

Env* NewMemEnv(Env* base_env) { return nullptr; }

#endif  // !ROCKSDB_LITE

}  // namespace rocksdb
