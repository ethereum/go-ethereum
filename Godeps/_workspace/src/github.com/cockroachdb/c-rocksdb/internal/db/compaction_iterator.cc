// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file. See the AUTHORS file for names of contributors.
//  Copyright (c) 2013, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.

#include "db/compaction_iterator.h"

namespace rocksdb {

CompactionIterator::CompactionIterator(
    Iterator* input, const Comparator* cmp, MergeHelper* merge_helper,
    SequenceNumber last_sequence, std::vector<SequenceNumber>* snapshots,
    Env* env, bool expect_valid_internal_key, Statistics* stats,
    Compaction* compaction, const CompactionFilter* compaction_filter,
    LogBuffer* log_buffer)
    : input_(input),
      cmp_(cmp),
      merge_helper_(merge_helper),
      snapshots_(snapshots),
      env_(env),
      expect_valid_internal_key_(expect_valid_internal_key),
      stats_(stats),
      compaction_(compaction),
      compaction_filter_(compaction_filter),
      log_buffer_(log_buffer),
      merge_out_iter_(merge_helper_) {
  assert(compaction_filter_ == nullptr || compaction_ != nullptr);
  bottommost_level_ =
      compaction_ == nullptr ? false : compaction_->bottommost_level();
  if (compaction_ != nullptr) {
    level_ptrs_ = std::vector<size_t>(compaction_->number_levels(), 0);
  }

  if (snapshots_->size() == 0) {
    // optimize for fast path if there are no snapshots
    visible_at_tip_ = last_sequence;
    earliest_snapshot_ = visible_at_tip_;
    latest_snapshot_ = 0;
  } else {
    visible_at_tip_ = 0;
    earliest_snapshot_ = snapshots_->at(0);
    latest_snapshot_ = snapshots_->back();
  }
}

void CompactionIterator::ResetRecordCounts() {
  iter_stats_.num_record_drop_user = 0;
  iter_stats_.num_record_drop_hidden = 0;
  iter_stats_.num_record_drop_obsolete = 0;
}

void CompactionIterator::SeekToFirst() {
  NextFromInput();
  PrepareOutput();
}

void CompactionIterator::Next() {
  // If there is a merge output, return it before continuing to process the
  // input.
  if (merge_out_iter_.Valid()) {
    merge_out_iter_.Next();

    // Check if we returned all records of the merge output.
    if (merge_out_iter_.Valid()) {
      key_ = merge_out_iter_.key();
      value_ = merge_out_iter_.value();
      bool valid_key __attribute__((__unused__)) =
          ParseInternalKey(key_, &ikey_);
      // MergeUntil stops when it encounters a corrupt key and does not
      // include them in the result, so we expect the keys here to be valid.
      assert(valid_key);
      valid_ = true;
    } else {
      // MergeHelper moves the iterator to the first record after the merged
      // records, so even though we reached the end of the merge output, we do
      // not want to advance the iterator.
      NextFromInput();
    }
  } else {
    // Only advance the input iterator if there is no merge output.
    input_->Next();
    NextFromInput();
  }

  PrepareOutput();
}

void CompactionIterator::NextFromInput() {
  valid_ = false;

  while (input_->Valid()) {
    key_ = input_->key();
    value_ = input_->value();
    iter_stats_.num_input_records++;

    if (!ParseInternalKey(key_, &ikey_)) {
      // If `expect_valid_internal_key_` is false, return the corrupted key
      // and let the caller decide what to do with it.
      // TODO(noetzli): We should have a more elegant solution for this.
      if (expect_valid_internal_key_) {
        assert(!"corrupted internal key is not expected");
        break;
      }
      current_user_key_.Clear();
      has_current_user_key_ = false;
      current_user_key_sequence_ = kMaxSequenceNumber;
      current_user_key_snapshot_ = 0;
      iter_stats_.num_input_corrupt_records++;
      valid_ = true;
      break;
    }

    // Update input statistics
    if (ikey_.type == kTypeDeletion) {
      iter_stats_.num_input_deletion_records++;
    }
    iter_stats_.total_input_raw_key_bytes += key_.size();
    iter_stats_.total_input_raw_value_bytes += value_.size();

    if (!has_current_user_key_ ||
        cmp_->Compare(ikey_.user_key, current_user_key_.GetKey()) != 0) {
      // First occurrence of this user key
      current_user_key_.SetKey(ikey_.user_key);
      has_current_user_key_ = true;
      current_user_key_sequence_ = kMaxSequenceNumber;
      current_user_key_snapshot_ = 0;
      // apply the compaction filter to the first occurrence of the user key
      if (compaction_filter_ != nullptr && ikey_.type == kTypeValue &&
          (visible_at_tip_ || ikey_.sequence > latest_snapshot_)) {
        // If the user has specified a compaction filter and the sequence
        // number is greater than any external snapshot, then invoke the
        // filter. If the return value of the compaction filter is true,
        // replace the entry with a deletion marker.
        bool value_changed = false;
        bool to_delete = false;
        compaction_filter_value_.clear();
        {
          StopWatchNano timer(env_, true);
          to_delete = compaction_filter_->Filter(
              compaction_->level(), ikey_.user_key, value_,
              &compaction_filter_value_, &value_changed);
          iter_stats_.total_filter_time +=
              env_ != nullptr ? timer.ElapsedNanos() : 0;
        }
        if (to_delete) {
          // make a copy of the original key and convert it to a delete
          delete_key_.SetInternalKey(ExtractUserKey(key_), ikey_.sequence,
                                     kTypeDeletion);
          // anchor the key again
          key_ = delete_key_.GetKey();
          // needed because ikey_ is backed by key
          ParseInternalKey(key_, &ikey_);
          // no value associated with delete
          value_.clear();
          iter_stats_.num_record_drop_user++;
        } else if (value_changed) {
          value_ = compaction_filter_value_;
        }
      }
    }

    // If there are no snapshots, then this kv affect visibility at tip.
    // Otherwise, search though all existing snapshots to find the earliest
    // snapshot that is affected by this kv.
    SequenceNumber last_sequence __attribute__((__unused__)) =
        current_user_key_sequence_;
    current_user_key_sequence_ = ikey_.sequence;
    SequenceNumber last_snapshot = current_user_key_snapshot_;
    SequenceNumber prev_snapshot = 0;  // 0 means no previous snapshot
    current_user_key_snapshot_ =
        visible_at_tip_ ? visible_at_tip_ : findEarliestVisibleSnapshot(
                                                ikey_.sequence, &prev_snapshot);

    if (last_snapshot == current_user_key_snapshot_) {
      // If the earliest snapshot is which this key is visible in
      // is the same as the visibility of a previous instance of the
      // same key, then this kv is not visible in any snapshot.
      // Hidden by an newer entry for same user key
      // TODO: why not > ?
      assert(last_sequence >= current_user_key_sequence_);
      ++iter_stats_.num_record_drop_hidden;  // (A)
    } else if (compaction_ != nullptr && ikey_.type == kTypeDeletion &&
               ikey_.sequence <= earliest_snapshot_ &&
               compaction_->KeyNotExistsBeyondOutputLevel(ikey_.user_key,
                                                          &level_ptrs_)) {
      // TODO(noetzli): This is the only place where we use compaction_
      // (besides the constructor). We should probably get rid of this
      // dependency and find a way to do similar filtering during flushes.
      //
      // For this user key:
      // (1) there is no data in higher levels
      // (2) data in lower levels will have larger sequence numbers
      // (3) data in layers that are being compacted here and have
      //     smaller sequence numbers will be dropped in the next
      //     few iterations of this loop (by rule (A) above).
      // Therefore this deletion marker is obsolete and can be dropped.
      ++iter_stats_.num_record_drop_obsolete;
    } else if (ikey_.type == kTypeMerge) {
      if (!merge_helper_->HasOperator()) {
        LogToBuffer(log_buffer_, "Options::merge_operator is null.");
        status_ = Status::InvalidArgument(
            "merge_operator is not properly initialized.");
        return;
      }

      // We know the merge type entry is not hidden, otherwise we would
      // have hit (A)
      // We encapsulate the merge related state machine in a different
      // object to minimize change to the existing flow.
      merge_helper_->MergeUntil(input_, prev_snapshot, bottommost_level_,
                                stats_, env_);
      merge_out_iter_.SeekToFirst();

      // NOTE: key, value, and ikey_ refer to old entries.
      //       These will be correctly set below.
      key_ = merge_out_iter_.key();
      value_ = merge_out_iter_.value();
      bool valid_key __attribute__((__unused__)) =
          ParseInternalKey(key_, &ikey_);
      // MergeUntil stops when it encounters a corrupt key and does not
      // include them in the result, so we expect the keys here to valid.
      assert(valid_key);
      valid_ = true;
      break;
    } else {
      valid_ = true;
      break;
    }

    input_->Next();
  }
}

void CompactionIterator::PrepareOutput() {
  // Zeroing out the sequence number leads to better compression.
  // If this is the bottommost level (no files in lower levels)
  // and the earliest snapshot is larger than this seqno
  // then we can squash the seqno to zero.
  if (bottommost_level_ && valid_ && ikey_.sequence < earliest_snapshot_ &&
      ikey_.type != kTypeMerge) {
    assert(ikey_.type != kTypeDeletion);
    // make a copy because updating in place would cause problems
    // with the priority queue that is managing the input key iterator
    updated_key_.assign(key_.data(), key_.size());
    UpdateInternalKey(&updated_key_, (uint64_t)0, ikey_.type);
    key_ = Slice(updated_key_);
  }
}

inline SequenceNumber CompactionIterator::findEarliestVisibleSnapshot(
    SequenceNumber in, SequenceNumber* prev_snapshot) {
  assert(snapshots_->size());
  SequenceNumber prev __attribute__((unused)) = 0;
  for (const auto cur : *snapshots_) {
    assert(prev <= cur);
    if (cur >= in) {
      *prev_snapshot = prev;
      return cur;
    }
    prev = cur;
    assert(prev);
  }
  *prev_snapshot = prev;
  return kMaxSequenceNumber;
}

}  // namespace rocksdb
