//  Copyright (c) 2013, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.

#include "db/merge_helper.h"

#include <stdio.h>
#include <string>

#include "db/dbformat.h"
#include "rocksdb/comparator.h"
#include "rocksdb/db.h"
#include "rocksdb/merge_operator.h"
#include "util/perf_context_imp.h"
#include "util/statistics.h"
#include "util/stop_watch.h"

namespace rocksdb {

// TODO(agiardullo): Clean up merge callsites to use this func
Status MergeHelper::TimedFullMerge(const Slice& key, const Slice* value,
                                   const std::deque<std::string>& operands,
                                   const MergeOperator* merge_operator,
                                   Statistics* statistics, Env* env,
                                   Logger* logger, std::string* result) {
  if (operands.size() == 0) {
    result->assign(value->data(), value->size());
    return Status::OK();
  }

  if (merge_operator == nullptr) {
    return Status::NotSupported("Provide a merge_operator when opening DB");
  }

  // Setup to time the merge
  StopWatchNano timer(env, statistics != nullptr);
  PERF_TIMER_GUARD(merge_operator_time_nanos);

  // Do the merge
  bool success =
      merge_operator->FullMerge(key, value, operands, result, logger);

  RecordTick(statistics, MERGE_OPERATION_TOTAL_TIME,
             env != nullptr ? timer.ElapsedNanos() : 0);

  if (!success) {
    RecordTick(statistics, NUMBER_MERGE_FAILURES);
    return Status::Corruption("Error: Could not perform merge.");
  }

  return Status::OK();
}

// PRE:  iter points to the first merge type entry
// POST: iter points to the first entry beyond the merge process (or the end)
//       keys_, operands_ are updated to reflect the merge result.
//       keys_ stores the list of keys encountered while merging.
//       operands_ stores the list of merge operands encountered while merging.
//       keys_[i] corresponds to operands_[i] for each i.
Status MergeHelper::MergeUntil(Iterator* iter, const SequenceNumber stop_before,
                               const bool at_bottom, Statistics* stats,
                               Env* env_) {
  // Get a copy of the internal key, before it's invalidated by iter->Next()
  // Also maintain the list of merge operands seen.
  assert(HasOperator());
  keys_.clear();
  operands_.clear();
  keys_.push_front(iter->key().ToString());
  operands_.push_front(iter->value().ToString());
  assert(user_merge_operator_);

  // We need to parse the internal key again as the parsed key is
  // backed by the internal key!
  // Assume no internal key corruption as it has been successfully parsed
  // by the caller.
  // Invariant: keys_.back() will not change. Hence, orig_ikey is always valid.
  ParsedInternalKey orig_ikey;
  ParseInternalKey(keys_.back(), &orig_ikey);

  Status s;
  bool hit_the_next_user_key = false;
  for (iter->Next(); iter->Valid(); iter->Next()) {
    ParsedInternalKey ikey;
    assert(operands_.size() >= 1);        // Should be invariants!
    assert(keys_.size() == operands_.size());

    if (!ParseInternalKey(iter->key(), &ikey)) {
      // stop at corrupted key
      if (assert_valid_internal_key_) {
        assert(!"corrupted internal key is not expected");
      }
      break;
    } else if (!user_comparator_->Equal(ikey.user_key, orig_ikey.user_key)) {
      // hit a different user key, stop right here
      hit_the_next_user_key = true;
      break;
    } else if (stop_before && ikey.sequence <= stop_before) {
      // hit an entry that's visible by the previous snapshot, can't touch that
      break;
    }

    // At this point we are guaranteed that we need to process this key.

    assert(ikey.type <= kValueTypeForSeek);
    if (ikey.type != kTypeMerge) {
      // hit a put/delete
      //   => merge the put value or a nullptr with operands_
      //   => store result in operands_.back() (and update keys_.back())
      //   => change the entry type to kTypeValue for keys_.back()
      // We are done! Success!
      //
      // TODO(noetzli) If the merge operator returns false, we are currently
      // (almost) silently dropping the put/delete. That's probably not what we
      // want.
      const Slice val = iter->value();
      const Slice* val_ptr = (kTypeValue == ikey.type) ? &val : nullptr;
      std::string merge_result;
      s = TimedFullMerge(ikey.user_key, val_ptr, operands_,
                         user_merge_operator_, stats, env_, logger_,
                         &merge_result);

      // We store the result in keys_.back() and operands_.back()
      // if nothing went wrong (i.e.: no operand corruption on disk)
      if (s.ok()) {
        // The original key encountered
        std::string original_key = std::move(keys_.back());
        orig_ikey.type = kTypeValue;
        UpdateInternalKey(&original_key, orig_ikey.sequence, orig_ikey.type);
        keys_.clear();
        operands_.clear();
        keys_.emplace_front(std::move(original_key));
        operands_.emplace_front(std::move(merge_result));
      }

      // move iter to the next entry
      iter->Next();
      return s;
    } else {
      // hit a merge
      //   => merge the operand into the front of the operands_ list
      //   => use the user's associative merge function to determine how.
      //   => then continue because we haven't yet seen a Put/Delete.
      assert(!operands_.empty()); // Should have at least one element in it

      // keep queuing keys and operands until we either meet a put / delete
      // request or later did a partial merge.
      keys_.push_front(iter->key().ToString());
      operands_.push_front(iter->value().ToString());
    }
  }

  // We are sure we have seen this key's entire history if we are at the
  // last level and exhausted all internal keys of this user key.
  // NOTE: !iter->Valid() does not necessarily mean we hit the
  // beginning of a user key, as versions of a user key might be
  // split into multiple files (even files on the same level)
  // and some files might not be included in the compaction/merge.
  //
  // There are also cases where we have seen the root of history of this
  // key without being sure of it. Then, we simply miss the opportunity
  // to combine the keys. Since VersionSet::SetupOtherInputs() always makes
  // sure that all merge-operands on the same level get compacted together,
  // this will simply lead to these merge operands moving to the next level.
  //
  // So, we only perform the following logic (to merge all operands together
  // without a Put/Delete) if we are certain that we have seen the end of key.
  bool surely_seen_the_beginning = hit_the_next_user_key && at_bottom;
  if (surely_seen_the_beginning) {
    // do a final merge with nullptr as the existing value and say
    // bye to the merge type (it's now converted to a Put)
    assert(kTypeMerge == orig_ikey.type);
    assert(operands_.size() >= 1);
    assert(operands_.size() == keys_.size());
    std::string merge_result;
    s = TimedFullMerge(orig_ikey.user_key, nullptr, operands_,
                       user_merge_operator_, stats, env_, logger_,
                       &merge_result);
    if (s.ok()) {
      // The original key encountered
      std::string original_key = std::move(keys_.back());
      orig_ikey.type = kTypeValue;
      UpdateInternalKey(&original_key, orig_ikey.sequence, orig_ikey.type);
      keys_.clear();
      operands_.clear();
      keys_.emplace_front(std::move(original_key));
      operands_.emplace_front(std::move(merge_result));
    }
  } else {
    // We haven't seen the beginning of the key nor a Put/Delete.
    // Attempt to use the user's associative merge function to
    // merge the stacked merge operands into a single operand.
    //
    // TODO(noetzli) The docblock of MergeUntil suggests that a successful
    // partial merge returns Status::OK(). Should we change the status code
    // after a successful partial merge?
    s = Status::MergeInProgress();
    if (operands_.size() >= 2 &&
        operands_.size() >= min_partial_merge_operands_) {
      bool merge_success = false;
      std::string merge_result;
      {
        StopWatchNano timer(env_, stats != nullptr);
        PERF_TIMER_GUARD(merge_operator_time_nanos);
        merge_success = user_merge_operator_->PartialMergeMulti(
            orig_ikey.user_key,
            std::deque<Slice>(operands_.begin(), operands_.end()),
            &merge_result, logger_);
        RecordTick(stats, MERGE_OPERATION_TOTAL_TIME,
                   env_ != nullptr ? timer.ElapsedNanos() : 0);
      }
      if (merge_success) {
        // Merging of operands (associative merge) was successful.
        // Replace operands with the merge result
        operands_.clear();
        operands_.emplace_front(std::move(merge_result));
        keys_.erase(keys_.begin(), keys_.end() - 1);
      }
    }
  }

  return s;
}

MergeOutputIterator::MergeOutputIterator(const MergeHelper* merge_helper)
    : merge_helper_(merge_helper) {
  it_keys_ = merge_helper_->keys().rend();
  it_values_ = merge_helper_->values().rend();
}

void MergeOutputIterator::SeekToFirst() {
  const auto& keys = merge_helper_->keys();
  const auto& values = merge_helper_->values();
  assert(keys.size() > 0);
  assert(keys.size() == values.size());
  it_keys_ = keys.rbegin();
  it_values_ = values.rbegin();
}

void MergeOutputIterator::Next() {
  ++it_keys_;
  ++it_values_;
}

} // namespace rocksdb
