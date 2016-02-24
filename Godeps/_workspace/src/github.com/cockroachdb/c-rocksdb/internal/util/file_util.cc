//  Copyright (c) 2013, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.
//
#include "util/file_util.h"

#include <string>
#include <algorithm>

#include "rocksdb/delete_scheduler.h"
#include "rocksdb/env.h"
#include "rocksdb/options.h"
#include "db/filename.h"
#include "util/file_reader_writer.h"

namespace rocksdb {

// Utility function to copy a file up to a specified length
Status CopyFile(Env* env, const std::string& source,
                const std::string& destination, uint64_t size) {
  const EnvOptions soptions;
  Status s;
  unique_ptr<SequentialFileReader> src_reader;
  unique_ptr<WritableFileWriter> dest_writer;

  {
    unique_ptr<SequentialFile> srcfile;
  s = env->NewSequentialFile(source, &srcfile, soptions);
  unique_ptr<WritableFile> destfile;
  if (s.ok()) {
    s = env->NewWritableFile(destination, &destfile, soptions);
  } else {
    return s;
  }

  if (size == 0) {
    // default argument means copy everything
    if (s.ok()) {
      s = env->GetFileSize(source, &size);
    } else {
      return s;
    }
  }
  src_reader.reset(new SequentialFileReader(std::move(srcfile)));
  dest_writer.reset(new WritableFileWriter(std::move(destfile), soptions));
  }

  char buffer[4096];
  Slice slice;
  while (size > 0) {
    uint64_t bytes_to_read =
        std::min(static_cast<uint64_t>(sizeof(buffer)), size);
    if (s.ok()) {
      s = src_reader->Read(bytes_to_read, &slice, buffer);
    }
    if (s.ok()) {
      if (slice.size() == 0) {
        return Status::Corruption("file too small");
      }
      s = dest_writer->Append(slice);
    }
    if (!s.ok()) {
      return s;
    }
    size -= slice.size();
  }
  return Status::OK();
}

Status DeleteOrMoveToTrash(const DBOptions* db_options,
                           const std::string& fname) {
  if (db_options->delete_scheduler == nullptr) {
    return db_options->env->DeleteFile(fname);
  } else {
    return db_options->delete_scheduler->DeleteFile(fname);
  }
}

}  // namespace rocksdb
