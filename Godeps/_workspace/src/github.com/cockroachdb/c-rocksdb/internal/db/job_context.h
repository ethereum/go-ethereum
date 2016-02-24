//  Copyright (c) 2013, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.
//
// Copyright (c) 2011 The LevelDB Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file. See the AUTHORS file for names of contributors.

#pragma once

#include <string>
#include <vector>

#include "db/column_family.h"
#include "db/log_writer.h"

namespace rocksdb {

class MemTable;

struct JobContext {
  inline bool HaveSomethingToDelete() const {
    return full_scan_candidate_files.size() || sst_delete_files.size() ||
           log_delete_files.size() || new_superversion != nullptr ||
           superversions_to_free.size() > 0 || memtables_to_free.size() > 0 ||
           logs_to_free.size() > 0;
  }

  // Structure to store information for candidate files to delete.
  struct CandidateFileInfo {
    std::string file_name;
    uint32_t path_id;
    CandidateFileInfo(std::string name, uint32_t path)
        : file_name(std::move(name)), path_id(path) {}
    bool operator==(const CandidateFileInfo& other) const {
      return file_name == other.file_name && path_id == other.path_id;
    }
  };

  // Unique job id
  int job_id;

  // a list of all files that we'll consider deleting
  // (every once in a while this is filled up with all files
  // in the DB directory)
  // (filled only if we're doing full scan)
  std::vector<CandidateFileInfo> full_scan_candidate_files;

  // the list of all live sst files that cannot be deleted
  std::vector<FileDescriptor> sst_live;

  // a list of sst files that we need to delete
  std::vector<FileMetaData*> sst_delete_files;

  // a list of log files that we need to delete
  std::vector<uint64_t> log_delete_files;

  // a list of memtables to be free
  autovector<MemTable*> memtables_to_free;

  autovector<SuperVersion*> superversions_to_free;

  autovector<log::Writer*> logs_to_free;

  SuperVersion* new_superversion;  // if nullptr no new superversion

  // the current manifest_file_number, log_number and prev_log_number
  // that corresponds to the set of files in 'live'.
  uint64_t manifest_file_number;
  uint64_t pending_manifest_file_number;
  uint64_t log_number;
  uint64_t prev_log_number;

  uint64_t min_pending_output = 0;

  explicit JobContext(int _job_id, bool create_superversion = false) {
    job_id = _job_id;
    manifest_file_number = 0;
    pending_manifest_file_number = 0;
    log_number = 0;
    prev_log_number = 0;
    new_superversion = create_superversion ? new SuperVersion() : nullptr;
  }

  // For non-empty JobContext Clean() has to be called at least once before
  // before destruction (see asserts in ~JobContext()). Should be called with
  // unlocked DB mutex. Destructor doesn't call Clean() to avoid accidentally
  // doing potentially slow Clean() with locked DB mutex.
  void Clean() {
    // free pending memtables
    for (auto m : memtables_to_free) {
      delete m;
    }
    // free superversions
    for (auto s : superversions_to_free) {
      delete s;
    }
    for (auto l : logs_to_free) {
      delete l;
    }
    // if new_superversion was not used, it will be non-nullptr and needs
    // to be freed here
    delete new_superversion;

    memtables_to_free.clear();
    superversions_to_free.clear();
    logs_to_free.clear();
    new_superversion = nullptr;
  }

  ~JobContext() {
    assert(memtables_to_free.size() == 0);
    assert(superversions_to_free.size() == 0);
    assert(new_superversion == nullptr);
    assert(logs_to_free.size() == 0);
  }
};

}  // namespace rocksdb
