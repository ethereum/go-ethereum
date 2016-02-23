//  Copyright (c) 2013, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.
//
#ifdef USE_HDFS
#ifndef ROCKSDB_HDFS_FILE_C
#define ROCKSDB_HDFS_FILE_C

#include <algorithm>
#include <stdio.h>
#include <sys/time.h>
#include <time.h>
#include <iostream>
#include <sstream>
#include "rocksdb/env.h"
#include "rocksdb/status.h"
#include "hdfs/env_hdfs.h"

#define HDFS_EXISTS 0
#define HDFS_DOESNT_EXIST -1
#define HDFS_SUCCESS 0

//
// This file defines an HDFS environment for rocksdb. It uses the libhdfs
// api to access HDFS. All HDFS files created by one instance of rocksdb
// will reside on the same HDFS cluster.
//

namespace rocksdb {

namespace {

// Log error message
static Status IOError(const std::string& context, int err_number) {
  return Status::IOError(context, strerror(err_number));
}

// assume that there is one global logger for now. It is not thread-safe,
// but need not be because the logger is initialized at db-open time.
static Logger* mylog = nullptr;

// Used for reading a file from HDFS. It implements both sequential-read
// access methods as well as random read access methods.
class HdfsReadableFile : virtual public SequentialFile,
                         virtual public RandomAccessFile {
 private:
  hdfsFS fileSys_;
  std::string filename_;
  hdfsFile hfile_;

 public:
  HdfsReadableFile(hdfsFS fileSys, const std::string& fname)
      : fileSys_(fileSys), filename_(fname), hfile_(nullptr) {
    Log(InfoLogLevel::DEBUG_LEVEL, mylog,
        "[hdfs] HdfsReadableFile opening file %s\n",
        filename_.c_str());
    hfile_ = hdfsOpenFile(fileSys_, filename_.c_str(), O_RDONLY, 0, 0, 0);
    Log(InfoLogLevel::DEBUG_LEVEL, mylog,
        "[hdfs] HdfsReadableFile opened file %s hfile_=0x%p\n",
        filename_.c_str(), hfile_);
  }

  virtual ~HdfsReadableFile() {
    Log(InfoLogLevel::DEBUG_LEVEL, mylog,
        "[hdfs] HdfsReadableFile closing file %s\n",
        filename_.c_str());
    hdfsCloseFile(fileSys_, hfile_);
    Log(InfoLogLevel::DEBUG_LEVEL, mylog,
        "[hdfs] HdfsReadableFile closed file %s\n",
        filename_.c_str());
    hfile_ = nullptr;
  }

  bool isValid() {
    return hfile_ != nullptr;
  }

  // sequential access, read data at current offset in file
  virtual Status Read(size_t n, Slice* result, char* scratch) {
    Status s;
    Log(InfoLogLevel::DEBUG_LEVEL, mylog,
        "[hdfs] HdfsReadableFile reading %s %ld\n",
        filename_.c_str(), n);

    char* buffer = scratch;
    size_t total_bytes_read = 0;
    tSize bytes_read = 0;
    tSize remaining_bytes = (tSize)n;

    // Read a total of n bytes repeatedly until we hit error or eof
    while (remaining_bytes > 0) {
      bytes_read = hdfsRead(fileSys_, hfile_, buffer, remaining_bytes);
      if (bytes_read <= 0) {
        break;
      }
      assert(bytes_read <= remaining_bytes);

      total_bytes_read += bytes_read;
      remaining_bytes -= bytes_read;
      buffer += bytes_read;
    }
    assert(total_bytes_read <= n);

    Log(InfoLogLevel::DEBUG_LEVEL, mylog,
        "[hdfs] HdfsReadableFile read %s\n", filename_.c_str());

    if (bytes_read < 0) {
      s = IOError(filename_, errno);
    } else {
      *result = Slice(scratch, total_bytes_read);
    }

    return s;
  }

  // random access, read data from specified offset in file
  virtual Status Read(uint64_t offset, size_t n, Slice* result,
                      char* scratch) const {
    Status s;
    Log(InfoLogLevel::DEBUG_LEVEL, mylog,
        "[hdfs] HdfsReadableFile preading %s\n", filename_.c_str());
    ssize_t bytes_read = hdfsPread(fileSys_, hfile_, offset,
                                   (void*)scratch, (tSize)n);
    Log(InfoLogLevel::DEBUG_LEVEL, mylog,
        "[hdfs] HdfsReadableFile pread %s\n", filename_.c_str());
    *result = Slice(scratch, (bytes_read < 0) ? 0 : bytes_read);
    if (bytes_read < 0) {
      // An error: return a non-ok status
      s = IOError(filename_, errno);
    }
    return s;
  }

  virtual Status Skip(uint64_t n) {
    Log(InfoLogLevel::DEBUG_LEVEL, mylog,
        "[hdfs] HdfsReadableFile skip %s\n", filename_.c_str());
    // get current offset from file
    tOffset current = hdfsTell(fileSys_, hfile_);
    if (current < 0) {
      return IOError(filename_, errno);
    }
    // seek to new offset in file
    tOffset newoffset = current + n;
    int val = hdfsSeek(fileSys_, hfile_, newoffset);
    if (val < 0) {
      return IOError(filename_, errno);
    }
    return Status::OK();
  }

 private:

  // returns true if we are at the end of file, false otherwise
  bool feof() {
    Log(InfoLogLevel::DEBUG_LEVEL, mylog,
        "[hdfs] HdfsReadableFile feof %s\n", filename_.c_str());
    if (hdfsTell(fileSys_, hfile_) == fileSize()) {
      return true;
    }
    return false;
  }

  // the current size of the file
  tOffset fileSize() {
    Log(InfoLogLevel::DEBUG_LEVEL, mylog,
        "[hdfs] HdfsReadableFile fileSize %s\n", filename_.c_str());
    hdfsFileInfo* pFileInfo = hdfsGetPathInfo(fileSys_, filename_.c_str());
    tOffset size = 0L;
    if (pFileInfo != nullptr) {
      size = pFileInfo->mSize;
      hdfsFreeFileInfo(pFileInfo, 1);
    } else {
      throw HdfsFatalException("fileSize on unknown file " + filename_);
    }
    return size;
  }
};

// Appends to an existing file in HDFS.
class HdfsWritableFile: public WritableFile {
 private:
  hdfsFS fileSys_;
  std::string filename_;
  hdfsFile hfile_;

 public:
  HdfsWritableFile(hdfsFS fileSys, const std::string& fname)
      : fileSys_(fileSys), filename_(fname) , hfile_(nullptr) {
    Log(InfoLogLevel::DEBUG_LEVEL, mylog,
        "[hdfs] HdfsWritableFile opening %s\n", filename_.c_str());
    hfile_ = hdfsOpenFile(fileSys_, filename_.c_str(), O_WRONLY, 0, 0, 0);
    Log(InfoLogLevel::DEBUG_LEVEL, mylog,
        "[hdfs] HdfsWritableFile opened %s\n", filename_.c_str());
    assert(hfile_ != nullptr);
  }
  virtual ~HdfsWritableFile() {
    if (hfile_ != nullptr) {
      Log(InfoLogLevel::DEBUG_LEVEL, mylog,
          "[hdfs] HdfsWritableFile closing %s\n", filename_.c_str());
      hdfsCloseFile(fileSys_, hfile_);
      Log(InfoLogLevel::DEBUG_LEVEL, mylog,
          "[hdfs] HdfsWritableFile closed %s\n", filename_.c_str());
      hfile_ = nullptr;
    }
  }

  // If the file was successfully created, then this returns true.
  // Otherwise returns false.
  bool isValid() {
    return hfile_ != nullptr;
  }

  // The name of the file, mostly needed for debug logging.
  const std::string& getName() {
    return filename_;
  }

  virtual Status Append(const Slice& data) {
    Log(InfoLogLevel::DEBUG_LEVEL, mylog,
        "[hdfs] HdfsWritableFile Append %s\n", filename_.c_str());
    const char* src = data.data();
    size_t left = data.size();
    size_t ret = hdfsWrite(fileSys_, hfile_, src, left);
    Log(InfoLogLevel::DEBUG_LEVEL, mylog,
        "[hdfs] HdfsWritableFile Appended %s\n", filename_.c_str());
    if (ret != left) {
      return IOError(filename_, errno);
    }
    return Status::OK();
  }

  virtual Status Flush() {
    return Status::OK();
  }

  virtual Status Sync() {
    Status s;
    Log(InfoLogLevel::DEBUG_LEVEL, mylog,
        "[hdfs] HdfsWritableFile Sync %s\n", filename_.c_str());
    if (hdfsFlush(fileSys_, hfile_) == -1) {
      return IOError(filename_, errno);
    }
    if (hdfsHSync(fileSys_, hfile_) == -1) {
      return IOError(filename_, errno);
    }
    Log(InfoLogLevel::DEBUG_LEVEL, mylog,
        "[hdfs] HdfsWritableFile Synced %s\n", filename_.c_str());
    return Status::OK();
  }

  // This is used by HdfsLogger to write data to the debug log file
  virtual Status Append(const char* src, size_t size) {
    if (hdfsWrite(fileSys_, hfile_, src, size) != (tSize)size) {
      return IOError(filename_, errno);
    }
    return Status::OK();
  }

  virtual Status Close() {
    Log(InfoLogLevel::DEBUG_LEVEL, mylog,
        "[hdfs] HdfsWritableFile closing %s\n", filename_.c_str());
    if (hdfsCloseFile(fileSys_, hfile_) != 0) {
      return IOError(filename_, errno);
    }
    Log(InfoLogLevel::DEBUG_LEVEL, mylog,
        "[hdfs] HdfsWritableFile closed %s\n", filename_.c_str());
    hfile_ = nullptr;
    return Status::OK();
  }
};

// The object that implements the debug logs to reside in HDFS.
class HdfsLogger : public Logger {
 private:
  HdfsWritableFile* file_;
  uint64_t (*gettid_)();  // Return the thread id for the current thread

 public:
  HdfsLogger(HdfsWritableFile* f, uint64_t (*gettid)())
      : file_(f), gettid_(gettid) {
    Log(InfoLogLevel::DEBUG_LEVEL, mylog,
        "[hdfs] HdfsLogger opened %s\n",
        file_->getName().c_str());
  }

  virtual ~HdfsLogger() {
    Log(InfoLogLevel::DEBUG_LEVEL, mylog,
        "[hdfs] HdfsLogger closed %s\n",
        file_->getName().c_str());
    delete file_;
    if (mylog != nullptr && mylog == this) {
      mylog = nullptr;
    }
  }

  virtual void Logv(const char* format, va_list ap) {
    const uint64_t thread_id = (*gettid_)();

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
                    "%04d/%02d/%02d-%02d:%02d:%02d.%06d %llx ",
                    t.tm_year + 1900,
                    t.tm_mon + 1,
                    t.tm_mday,
                    t.tm_hour,
                    t.tm_min,
                    t.tm_sec,
                    static_cast<int>(now_tv.tv_usec),
                    static_cast<long long unsigned int>(thread_id));

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
      file_->Append(base, p-base);
      file_->Flush();
      if (base != buffer) {
        delete[] base;
      }
      break;
    }
  }
};

}  // namespace

// Finally, the hdfs environment

const std::string HdfsEnv::kProto = "hdfs://";
const std::string HdfsEnv::pathsep = "/";

// open a file for sequential reading
Status HdfsEnv::NewSequentialFile(const std::string& fname,
                                  unique_ptr<SequentialFile>* result,
                                  const EnvOptions& options) {
  result->reset();
  HdfsReadableFile* f = new HdfsReadableFile(fileSys_, fname);
  if (f == nullptr || !f->isValid()) {
    delete f;
    *result = nullptr;
    return IOError(fname, errno);
  }
  result->reset(dynamic_cast<SequentialFile*>(f));
  return Status::OK();
}

// open a file for random reading
Status HdfsEnv::NewRandomAccessFile(const std::string& fname,
                                    unique_ptr<RandomAccessFile>* result,
                                    const EnvOptions& options) {
  result->reset();
  HdfsReadableFile* f = new HdfsReadableFile(fileSys_, fname);
  if (f == nullptr || !f->isValid()) {
    delete f;
    *result = nullptr;
    return IOError(fname, errno);
  }
  result->reset(dynamic_cast<RandomAccessFile*>(f));
  return Status::OK();
}

// create a new file for writing
Status HdfsEnv::NewWritableFile(const std::string& fname,
                                unique_ptr<WritableFile>* result,
                                const EnvOptions& options) {
  result->reset();
  Status s;
  HdfsWritableFile* f = new HdfsWritableFile(fileSys_, fname);
  if (f == nullptr || !f->isValid()) {
    delete f;
    *result = nullptr;
    return IOError(fname, errno);
  }
  result->reset(dynamic_cast<WritableFile*>(f));
  return Status::OK();
}

class HdfsDirectory : public Directory {
 public:
  explicit HdfsDirectory(int fd) : fd_(fd) {}
  ~HdfsDirectory() {}

  virtual Status Fsync() { return Status::OK(); }

 private:
  int fd_;
};

Status HdfsEnv::NewDirectory(const std::string& name,
                             unique_ptr<Directory>* result) {
  int value = hdfsExists(fileSys_, name.c_str());
  switch (value) {
    case HDFS_EXISTS:
      result->reset(new HdfsDirectory(0));
      return Status::OK();
    default:  // fail if the directory doesn't exist
      Log(InfoLogLevel::FATAL_LEVEL,
          mylog, "NewDirectory hdfsExists call failed");
      throw HdfsFatalException("hdfsExists call failed with error " +
                               ToString(value) + " on path " + name +
                               ".\n");
  }
}

Status HdfsEnv::FileExists(const std::string& fname) {
  int value = hdfsExists(fileSys_, fname.c_str());
  switch (value) {
    case HDFS_EXISTS:
      return Status::OK();
    case HDFS_DOESNT_EXIST:
      return Status::NotFound();
    default:  // anything else should be an error
      Log(InfoLogLevel::FATAL_LEVEL,
          mylog, "FileExists hdfsExists call failed");
      return Status::IOError("hdfsExists call failed with error " +
                             ToString(value) + " on path " + fname + ".\n");
  }
}

Status HdfsEnv::GetChildren(const std::string& path,
                            std::vector<std::string>* result) {
  int value = hdfsExists(fileSys_, path.c_str());
  switch (value) {
    case HDFS_EXISTS: {  // directory exists
    int numEntries = 0;
    hdfsFileInfo* pHdfsFileInfo = 0;
    pHdfsFileInfo = hdfsListDirectory(fileSys_, path.c_str(), &numEntries);
    if (numEntries >= 0) {
      for(int i = 0; i < numEntries; i++) {
        char* pathname = pHdfsFileInfo[i].mName;
        char* filename = rindex(pathname, '/');
        if (filename != nullptr) {
          result->push_back(filename+1);
        }
      }
      if (pHdfsFileInfo != nullptr) {
        hdfsFreeFileInfo(pHdfsFileInfo, numEntries);
      }
    } else {
      // numEntries < 0 indicates error
      Log(InfoLogLevel::FATAL_LEVEL, mylog,
          "hdfsListDirectory call failed with error ");
      throw HdfsFatalException(
          "hdfsListDirectory call failed negative error.\n");
    }
    break;
  }
  case HDFS_DOESNT_EXIST:  // directory does not exist, exit
    break;
  default:          // anything else should be an error
    Log(InfoLogLevel::FATAL_LEVEL, mylog,
        "GetChildren hdfsExists call failed");
    throw HdfsFatalException("hdfsExists call failed with error " +
                             ToString(value) + ".\n");
  }
  return Status::OK();
}

Status HdfsEnv::DeleteFile(const std::string& fname) {
  if (hdfsDelete(fileSys_, fname.c_str(), 1) == 0) {
    return Status::OK();
  }
  return IOError(fname, errno);
};

Status HdfsEnv::CreateDir(const std::string& name) {
  if (hdfsCreateDirectory(fileSys_, name.c_str()) == 0) {
    return Status::OK();
  }
  return IOError(name, errno);
};

Status HdfsEnv::CreateDirIfMissing(const std::string& name) {
  const int value = hdfsExists(fileSys_, name.c_str());
  //  Not atomic. state might change b/w hdfsExists and CreateDir.
  switch (value) {
    case HDFS_EXISTS:
    return Status::OK();
    case HDFS_DOESNT_EXIST:
    return CreateDir(name);
    default:  // anything else should be an error
      Log(InfoLogLevel::FATAL_LEVEL, mylog,
          "CreateDirIfMissing hdfsExists call failed");
      throw HdfsFatalException("hdfsExists call failed with error " +
                               ToString(value) + ".\n");
  }
};

Status HdfsEnv::DeleteDir(const std::string& name) {
  return DeleteFile(name);
};

Status HdfsEnv::GetFileSize(const std::string& fname, uint64_t* size) {
  *size = 0L;
  hdfsFileInfo* pFileInfo = hdfsGetPathInfo(fileSys_, fname.c_str());
  if (pFileInfo != nullptr) {
    *size = pFileInfo->mSize;
    hdfsFreeFileInfo(pFileInfo, 1);
    return Status::OK();
  }
  return IOError(fname, errno);
}

Status HdfsEnv::GetFileModificationTime(const std::string& fname,
                                        uint64_t* time) {
  hdfsFileInfo* pFileInfo = hdfsGetPathInfo(fileSys_, fname.c_str());
  if (pFileInfo != nullptr) {
    *time = static_cast<uint64_t>(pFileInfo->mLastMod);
    hdfsFreeFileInfo(pFileInfo, 1);
    return Status::OK();
  }
  return IOError(fname, errno);

}

// The rename is not atomic. HDFS does not allow a renaming if the
// target already exists. So, we delete the target before attempting the
// rename.
Status HdfsEnv::RenameFile(const std::string& src, const std::string& target) {
  hdfsDelete(fileSys_, target.c_str(), 1);
  if (hdfsRename(fileSys_, src.c_str(), target.c_str()) == 0) {
    return Status::OK();
  }
  return IOError(src, errno);
}

Status HdfsEnv::LockFile(const std::string& fname, FileLock** lock) {
  // there isn's a very good way to atomically check and create
  // a file via libhdfs
  *lock = nullptr;
  return Status::OK();
}

Status HdfsEnv::UnlockFile(FileLock* lock) {
  return Status::OK();
}

Status HdfsEnv::NewLogger(const std::string& fname,
                          shared_ptr<Logger>* result) {
  HdfsWritableFile* f = new HdfsWritableFile(fileSys_, fname);
  if (f == nullptr || !f->isValid()) {
    delete f;
    *result = nullptr;
    return IOError(fname, errno);
  }
  HdfsLogger* h = new HdfsLogger(f, &HdfsEnv::gettid);
  result->reset(h);
  if (mylog == nullptr) {
    // mylog = h; // uncomment this for detailed logging
  }
  return Status::OK();
}

}  // namespace rocksdb

#endif // ROCKSDB_HDFS_FILE_C

#else // USE_HDFS

// dummy placeholders used when HDFS is not available
#include "rocksdb/env.h"
#include "hdfs/env_hdfs.h"
namespace rocksdb {
 Status HdfsEnv::NewSequentialFile(const std::string& fname,
                                   unique_ptr<SequentialFile>* result,
                                   const EnvOptions& options) {
   return Status::NotSupported("Not compiled with hdfs support");
 }
}

#endif
