// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file. See the AUTHORS file for names of contributors.
//  Copyright (c) 2013, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.
#pragma once

#include <algorithm>
#include <deque>
#include <string>
#include <vector>

#include "db/compaction.h"
#include "db/merge_helper.h"
#include "rocksdb/compaction_filter.h"
#include "util/log_buffer.h"

namespace rocksdb {

struct CompactionIteratorStats {
  // Compaction statistics
  int64_t num_record_drop_user = 0;
  int64_t num_record_drop_hidden = 0;
  int64_t num_record_drop_obsolete = 0;
  uint64_t total_filter_time = 0;

  // Input statistics
  // TODO(noetzli): The stats are incomplete. They are lacking everything
  // consumed by MergeHelper.
  uint64_t num_input_records = 0;
  uint64_t num_input_deletion_records = 0;
  uint64_t num_input_corrupt_records = 0;
  uint64_t total_input_raw_key_bytes = 0;
  uint64_t total_input_raw_value_bytes = 0;
};

class CompactionIterator {
 public:
  CompactionIterator(Iterator* input, const Comparator* cmp,
                     MergeHelper* merge_helper, SequenceNumber last_sequence,
                     std::vector<SequenceNumber>* snapshots, Env* env,
                     bool expect_valid_internal_key,
                     Statistics* stats = nullptr,
                     Compaction* compaction = nullptr,
                     const CompactionFilter* compaction_filter = nullptr,
                     LogBuffer* log_buffer = nullptr);

  void ResetRecordCounts();

  // Seek to the beginning of the compaction iterator output.
  //
  // REQUIRED: Call only once.
  void SeekToFirst();

  // Produces the next record in the compaction.
  //
  // REQUIRED: SeekToFirst() has been called.
  void Next();

  // Getters
  const Slice& key() const { return key_; }
  const Slice& value() const { return value_; }
  const Status& status() const { return status_; }
  const ParsedInternalKey& ikey() const { return ikey_; }
  bool Valid() const { return valid_; }
  Slice user_key() const { return current_user_key_.GetKey(); }
  const CompactionIteratorStats& iter_stats() const { return iter_stats_; }

 private:
  // Processes the input stream to find the next output
  void NextFromInput();

  // Do last preparations before presenting the output to the callee. At this
  // point this only zeroes out the sequence number if possible for better
  // compression.
  void PrepareOutput();

  // Given a sequence number, return the sequence number of the
  // earliest snapshot that this sequence number is visible in.
  // The snapshots themselves are arranged in ascending order of
  // sequence numbers.
  // Employ a sequential search because the total number of
  // snapshots are typically small.
  inline SequenceNumber findEarliestVisibleSnapshot(
      SequenceNumber in, SequenceNumber* prev_snapshot);

  Iterator* input_;
  const Comparator* cmp_;
  MergeHelper* merge_helper_;
  const std::vector<SequenceNumber>* snapshots_;
  Env* env_;
  bool expect_valid_internal_key_;
  Statistics* stats_;
  Compaction* compaction_;
  const CompactionFilter* compaction_filter_;
  LogBuffer* log_buffer_;
  bool bottommost_level_;
  bool valid_ = false;
  SequenceNumber visible_at_tip_;
  SequenceNumber earliest_snapshot_;
  SequenceNumber latest_snapshot_;

  // State
  Slice key_;
  Slice value_;
  Status status_;
  ParsedInternalKey ikey_;
  bool has_current_user_key_ = false;
  IterKey current_user_key_;
  SequenceNumber current_user_key_sequence_;
  SequenceNumber current_user_key_snapshot_;
  MergeOutputIterator merge_out_iter_;
  std::string updated_key_;
  std::string compaction_filter_value_;
  IterKey delete_key_;
  // "level_ptrs" holds indices that remember which file of an associated
  // level we were last checking during the last call to compaction->
  // KeyNotExistsBeyondOutputLevel(). This allows future calls to the function
  // to pick off where it left off since each subcompaction's key range is
  // increasing so a later call to the function must be looking for a key that
  // is in or beyond the last file checked during the previous call
  std::vector<size_t> level_ptrs_;
  CompactionIteratorStats iter_stats_;
};
}  // namespace rocksdb
