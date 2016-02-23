//  Copyright (c) 2013, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.
//
// Copyright (c) 2011 The LevelDB Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file. See the AUTHORS file for names of contributors.

#include "db/db_iter.h"
#include <stdexcept>
#include <deque>
#include <string>
#include <limits>

#include "db/filename.h"
#include "db/dbformat.h"
#include "rocksdb/env.h"
#include "rocksdb/options.h"
#include "rocksdb/iterator.h"
#include "rocksdb/merge_operator.h"
#include "port/port.h"
#include "util/arena.h"
#include "util/logging.h"
#include "util/mutexlock.h"
#include "util/perf_context_imp.h"

namespace rocksdb {

#if 0
static void DumpInternalIter(Iterator* iter) {
  for (iter->SeekToFirst(); iter->Valid(); iter->Next()) {
    ParsedInternalKey k;
    if (!ParseInternalKey(iter->key(), &k)) {
      fprintf(stderr, "Corrupt '%s'\n", EscapeString(iter->key()).c_str());
    } else {
      fprintf(stderr, "@ '%s'\n", k.DebugString().c_str());
    }
  }
}
#endif

// Memtables and sstables that make the DB representation contain
// (userkey,seq,type) => uservalue entries.  DBIter
// combines multiple entries for the same userkey found in the DB
// representation into a single entry while accounting for sequence
// numbers, deletion markers, overwrites, etc.
class DBIter: public Iterator {
 public:
  // The following is grossly complicated. TODO: clean it up
  // Which direction is the iterator currently moving?
  // (1) When moving forward, the internal iterator is positioned at
  //     the exact entry that yields this->key(), this->value()
  // (2) When moving backwards, the internal iterator is positioned
  //     just before all entries whose user key == this->key().
  enum Direction {
    kForward,
    kReverse
  };

  DBIter(Env* env, const ImmutableCFOptions& ioptions,
         const Comparator* cmp, Iterator* iter, SequenceNumber s,
         bool arena_mode, uint64_t max_sequential_skip_in_iterations,
         const Slice* iterate_upper_bound = nullptr)
      : arena_mode_(arena_mode),
        env_(env),
        logger_(ioptions.info_log),
        user_comparator_(cmp),
        user_merge_operator_(ioptions.merge_operator),
        iter_(iter),
        sequence_(s),
        direction_(kForward),
        valid_(false),
        current_entry_is_merged_(false),
        statistics_(ioptions.statistics),
        iterate_upper_bound_(iterate_upper_bound) {
    RecordTick(statistics_, NO_ITERATORS);
    prefix_extractor_ = ioptions.prefix_extractor;
    max_skip_ = max_sequential_skip_in_iterations;
  }
  virtual ~DBIter() {
    RecordTick(statistics_, NO_ITERATORS, -1);
    if (!arena_mode_) {
      delete iter_;
    } else {
      iter_->~Iterator();
    }
  }
  virtual void SetIter(Iterator* iter) {
    assert(iter_ == nullptr);
    iter_ = iter;
  }
  virtual bool Valid() const override { return valid_; }
  virtual Slice key() const override {
    assert(valid_);
    return saved_key_.GetKey();
  }
  virtual Slice value() const override {
    assert(valid_);
    return (direction_ == kForward && !current_entry_is_merged_) ?
      iter_->value() : saved_value_;
  }
  virtual Status status() const override {
    if (status_.ok()) {
      return iter_->status();
    } else {
      return status_;
    }
  }

  virtual void Next() override;
  virtual void Prev() override;
  virtual void Seek(const Slice& target) override;
  virtual void SeekToFirst() override;
  virtual void SeekToLast() override;

 private:
  void ReverseToBackward();
  void PrevInternal();
  void FindParseableKey(ParsedInternalKey* ikey, Direction direction);
  bool FindValueForCurrentKey();
  bool FindValueForCurrentKeyUsingSeek();
  void FindPrevUserKey();
  void FindNextUserKey();
  inline void FindNextUserEntry(bool skipping);
  void FindNextUserEntryInternal(bool skipping);
  bool ParseKey(ParsedInternalKey* key);
  void MergeValuesNewToOld();

  inline void ClearSavedValue() {
    if (saved_value_.capacity() > 1048576) {
      std::string empty;
      swap(empty, saved_value_);
    } else {
      saved_value_.clear();
    }
  }

  const SliceTransform* prefix_extractor_;
  bool arena_mode_;
  Env* const env_;
  Logger* logger_;
  const Comparator* const user_comparator_;
  const MergeOperator* const user_merge_operator_;
  Iterator* iter_;
  SequenceNumber const sequence_;

  Status status_;
  IterKey saved_key_;
  std::string saved_value_;
  Direction direction_;
  bool valid_;
  bool current_entry_is_merged_;
  Statistics* statistics_;
  uint64_t max_skip_;
  const Slice* iterate_upper_bound_;

  // No copying allowed
  DBIter(const DBIter&);
  void operator=(const DBIter&);
};

inline bool DBIter::ParseKey(ParsedInternalKey* ikey) {
  if (!ParseInternalKey(iter_->key(), ikey)) {
    status_ = Status::Corruption("corrupted internal key in DBIter");
    Log(InfoLogLevel::ERROR_LEVEL,
        logger_, "corrupted internal key in DBIter: %s",
        iter_->key().ToString(true).c_str());
    return false;
  } else {
    return true;
  }
}

void DBIter::Next() {
  assert(valid_);

  if (direction_ == kReverse) {
    FindNextUserKey();
    direction_ = kForward;
    if (!iter_->Valid()) {
      iter_->SeekToFirst();
    }
  }

  // If the current value is merged, we might already hit end of iter_
  if (!iter_->Valid()) {
    valid_ = false;
    return;
  }
  FindNextUserEntry(true /* skipping the current user key */);
}

// PRE: saved_key_ has the current user key if skipping
// POST: saved_key_ should have the next user key if valid_,
//       if the current entry is a result of merge
//           current_entry_is_merged_ => true
//           saved_value_             => the merged value
//
// NOTE: In between, saved_key_ can point to a user key that has
//       a delete marker
inline void DBIter::FindNextUserEntry(bool skipping) {
  PERF_TIMER_GUARD(find_next_user_entry_time);
  FindNextUserEntryInternal(skipping);
}

// Actual implementation of DBIter::FindNextUserEntry()
void DBIter::FindNextUserEntryInternal(bool skipping) {
  // Loop until we hit an acceptable entry to yield
  assert(iter_->Valid());
  assert(direction_ == kForward);
  current_entry_is_merged_ = false;
  uint64_t num_skipped = 0;
  do {
    ParsedInternalKey ikey;

    if (ParseKey(&ikey)) {
      if (iterate_upper_bound_ != nullptr &&
          user_comparator_->Compare(ikey.user_key, *iterate_upper_bound_) >= 0) {
        break;
      }

      if (ikey.sequence <= sequence_) {
        if (skipping &&
           user_comparator_->Compare(ikey.user_key, saved_key_.GetKey()) <= 0) {
          num_skipped++;  // skip this entry
          PERF_COUNTER_ADD(internal_key_skipped_count, 1);
        } else {
          switch (ikey.type) {
            case kTypeDeletion:
              // Arrange to skip all upcoming entries for this key since
              // they are hidden by this deletion.
              saved_key_.SetKey(ikey.user_key);
              skipping = true;
              num_skipped = 0;
              PERF_COUNTER_ADD(internal_delete_skipped_count, 1);
              break;
            case kTypeValue:
              valid_ = true;
              saved_key_.SetKey(ikey.user_key);
              return;
            case kTypeMerge:
              // By now, we are sure the current ikey is going to yield a value
              saved_key_.SetKey(ikey.user_key);
              current_entry_is_merged_ = true;
              valid_ = true;
              MergeValuesNewToOld();  // Go to a different state machine
              return;
            default:
              assert(false);
              break;
          }
        }
      }
    }
    // If we have sequentially iterated via numerous keys and still not
    // found the next user-key, then it is better to seek so that we can
    // avoid too many key comparisons. We seek to the last occurrence of
    // our current key by looking for sequence number 0 and type deletion
    // (the smallest type).
    if (skipping && num_skipped > max_skip_) {
      num_skipped = 0;
      std::string last_key;
      AppendInternalKey(&last_key, ParsedInternalKey(saved_key_.GetKey(), 0,
                                                     kTypeDeletion));
      iter_->Seek(last_key);
      RecordTick(statistics_, NUMBER_OF_RESEEKS_IN_ITERATION);
    } else {
      iter_->Next();
    }
  } while (iter_->Valid());
  valid_ = false;
}

// Merge values of the same user key starting from the current iter_ position
// Scan from the newer entries to older entries.
// PRE: iter_->key() points to the first merge type entry
//      saved_key_ stores the user key
// POST: saved_value_ has the merged value for the user key
//       iter_ points to the next entry (or invalid)
void DBIter::MergeValuesNewToOld() {
  if (!user_merge_operator_) {
    Log(InfoLogLevel::ERROR_LEVEL,
        logger_, "Options::merge_operator is null.");
    status_ = Status::InvalidArgument("user_merge_operator_ must be set.");
    valid_ = false;
    return;
  }

  // Start the merge process by pushing the first operand
  std::deque<std::string> operands;
  operands.push_front(iter_->value().ToString());

  ParsedInternalKey ikey;
  for (iter_->Next(); iter_->Valid(); iter_->Next()) {
    if (!ParseKey(&ikey)) {
      // skip corrupted key
      continue;
    }

    if (!user_comparator_->Equal(ikey.user_key, saved_key_.GetKey())) {
      // hit the next user key, stop right here
      break;
    }

    if (kTypeDeletion == ikey.type) {
      // hit a delete with the same user key, stop right here
      // iter_ is positioned after delete
      iter_->Next();
      break;
    }

    if (kTypeValue == ikey.type) {
      // hit a put, merge the put value with operands and store the
      // final result in saved_value_. We are done!
      // ignore corruption if there is any.
      const Slice val = iter_->value();
      {
        StopWatchNano timer(env_, statistics_ != nullptr);
        PERF_TIMER_GUARD(merge_operator_time_nanos);
        user_merge_operator_->FullMerge(ikey.user_key, &val, operands,
                                        &saved_value_, logger_);
        RecordTick(statistics_, MERGE_OPERATION_TOTAL_TIME,
                   timer.ElapsedNanos());
      }
      // iter_ is positioned after put
      iter_->Next();
      return;
    }

    if (kTypeMerge == ikey.type) {
      // hit a merge, add the value as an operand and run associative merge.
      // when complete, add result to operands and continue.
      const Slice& val = iter_->value();
      operands.push_front(val.ToString());
    }
  }

  {
    StopWatchNano timer(env_, statistics_ != nullptr);
    PERF_TIMER_GUARD(merge_operator_time_nanos);
    // we either exhausted all internal keys under this user key, or hit
    // a deletion marker.
    // feed null as the existing value to the merge operator, such that
    // client can differentiate this scenario and do things accordingly.
    user_merge_operator_->FullMerge(saved_key_.GetKey(), nullptr, operands,
                                    &saved_value_, logger_);
    RecordTick(statistics_, MERGE_OPERATION_TOTAL_TIME, timer.ElapsedNanos());
  }
}

void DBIter::Prev() {
  assert(valid_);
  if (direction_ == kForward) {
    ReverseToBackward();
  }
  PrevInternal();
}

void DBIter::ReverseToBackward() {
  if (current_entry_is_merged_) {
    // Not placed in the same key. Need to call Prev() until finding the
    // previous key.
    if (!iter_->Valid()) {
      iter_->SeekToLast();
    }
    ParsedInternalKey ikey;
    FindParseableKey(&ikey, kReverse);
    while (iter_->Valid() &&
           user_comparator_->Compare(ikey.user_key, saved_key_.GetKey()) > 0) {
      iter_->Prev();
      FindParseableKey(&ikey, kReverse);
    }
  }
#ifndef NDEBUG
  if (iter_->Valid()) {
    ParsedInternalKey ikey;
    assert(ParseKey(&ikey));
    assert(user_comparator_->Compare(ikey.user_key, saved_key_.GetKey()) <= 0);
  }
#endif

  FindPrevUserKey();
  direction_ = kReverse;
}

void DBIter::PrevInternal() {
  if (!iter_->Valid()) {
    valid_ = false;
    return;
  }

  ParsedInternalKey ikey;

  while (iter_->Valid()) {
    saved_key_.SetKey(ExtractUserKey(iter_->key()));
    if (FindValueForCurrentKey()) {
      valid_ = true;
      if (!iter_->Valid()) {
        return;
      }
      FindParseableKey(&ikey, kReverse);
      if (user_comparator_->Equal(ikey.user_key, saved_key_.GetKey())) {
        FindPrevUserKey();
      }
      return;
    }
    if (!iter_->Valid()) {
      break;
    }
    FindParseableKey(&ikey, kReverse);
    if (user_comparator_->Equal(ikey.user_key, saved_key_.GetKey())) {
      FindPrevUserKey();
    }
  }
  // We haven't found any key - iterator is not valid
  assert(!iter_->Valid());
  valid_ = false;
}

// This function checks, if the entry with biggest sequence_number <= sequence_
// is non kTypeDeletion. If it's not, we save value in saved_value_
bool DBIter::FindValueForCurrentKey() {
  assert(iter_->Valid());
  // Contains operands for merge operator.
  std::deque<std::string> operands;
  // last entry before merge (could be kTypeDeletion or kTypeValue)
  ValueType last_not_merge_type = kTypeDeletion;
  ValueType last_key_entry_type = kTypeDeletion;

  ParsedInternalKey ikey;
  FindParseableKey(&ikey, kReverse);

  size_t num_skipped = 0;
  while (iter_->Valid() && ikey.sequence <= sequence_ &&
         user_comparator_->Equal(ikey.user_key, saved_key_.GetKey())) {
    // We iterate too much: let's use Seek() to avoid too much key comparisons
    if (num_skipped >= max_skip_) {
      return FindValueForCurrentKeyUsingSeek();
    }

    last_key_entry_type = ikey.type;
    switch (last_key_entry_type) {
      case kTypeValue:
        operands.clear();
        saved_value_ = iter_->value().ToString();
        last_not_merge_type = kTypeValue;
        break;
      case kTypeDeletion:
        operands.clear();
        last_not_merge_type = kTypeDeletion;
        PERF_COUNTER_ADD(internal_delete_skipped_count, 1);
        break;
      case kTypeMerge:
        assert(user_merge_operator_ != nullptr);
        operands.push_back(iter_->value().ToString());
        break;
      default:
        assert(false);
    }

    PERF_COUNTER_ADD(internal_key_skipped_count, 1);
    assert(user_comparator_->Equal(ikey.user_key, saved_key_.GetKey()));
    iter_->Prev();
    ++num_skipped;
    FindParseableKey(&ikey, kReverse);
  }

  switch (last_key_entry_type) {
    case kTypeDeletion:
      valid_ = false;
      return false;
    case kTypeMerge:
      if (last_not_merge_type == kTypeDeletion) {
        StopWatchNano timer(env_, statistics_ != nullptr);
        PERF_TIMER_GUARD(merge_operator_time_nanos);
        user_merge_operator_->FullMerge(saved_key_.GetKey(), nullptr, operands,
                                        &saved_value_, logger_);
        RecordTick(statistics_, MERGE_OPERATION_TOTAL_TIME,
                   timer.ElapsedNanos());
      } else {
        assert(last_not_merge_type == kTypeValue);
        std::string last_put_value = saved_value_;
        Slice temp_slice(last_put_value);
        {
          StopWatchNano timer(env_, statistics_ != nullptr);
          PERF_TIMER_GUARD(merge_operator_time_nanos);
          user_merge_operator_->FullMerge(saved_key_.GetKey(), &temp_slice,
                                          operands, &saved_value_, logger_);
          RecordTick(statistics_, MERGE_OPERATION_TOTAL_TIME,
                     timer.ElapsedNanos());
        }
      }
      break;
    case kTypeValue:
      // do nothing - we've already has value in saved_value_
      break;
    default:
      assert(false);
      break;
  }
  valid_ = true;
  return true;
}

// This function is used in FindValueForCurrentKey.
// We use Seek() function instead of Prev() to find necessary value
bool DBIter::FindValueForCurrentKeyUsingSeek() {
  std::string last_key;
  AppendInternalKey(&last_key, ParsedInternalKey(saved_key_.GetKey(), sequence_,
                                                 kValueTypeForSeek));
  iter_->Seek(last_key);
  RecordTick(statistics_, NUMBER_OF_RESEEKS_IN_ITERATION);

  // assume there is at least one parseable key for this user key
  ParsedInternalKey ikey;
  FindParseableKey(&ikey, kForward);

  if (ikey.type == kTypeValue || ikey.type == kTypeDeletion) {
    if (ikey.type == kTypeValue) {
      saved_value_ = iter_->value().ToString();
      valid_ = true;
      return true;
    }
    valid_ = false;
    return false;
  }

  // kTypeMerge. We need to collect all kTypeMerge values and save them
  // in operands
  std::deque<std::string> operands;
  while (iter_->Valid() &&
         user_comparator_->Equal(ikey.user_key, saved_key_.GetKey()) &&
         ikey.type == kTypeMerge) {
    operands.push_front(iter_->value().ToString());
    iter_->Next();
    FindParseableKey(&ikey, kForward);
  }

  if (!iter_->Valid() ||
      !user_comparator_->Equal(ikey.user_key, saved_key_.GetKey()) ||
      ikey.type == kTypeDeletion) {
    {
      StopWatchNano timer(env_, statistics_ != nullptr);
      PERF_TIMER_GUARD(merge_operator_time_nanos);
      user_merge_operator_->FullMerge(saved_key_.GetKey(), nullptr, operands,
                                      &saved_value_, logger_);
      RecordTick(statistics_, MERGE_OPERATION_TOTAL_TIME, timer.ElapsedNanos());
    }
    // Make iter_ valid and point to saved_key_
    if (!iter_->Valid() ||
        !user_comparator_->Equal(ikey.user_key, saved_key_.GetKey())) {
      iter_->Seek(last_key);
      RecordTick(statistics_, NUMBER_OF_RESEEKS_IN_ITERATION);
    }
    valid_ = true;
    return true;
  }

  const Slice& val = iter_->value();
  {
    StopWatchNano timer(env_, statistics_ != nullptr);
    PERF_TIMER_GUARD(merge_operator_time_nanos);
    user_merge_operator_->FullMerge(saved_key_.GetKey(), &val, operands,
                                    &saved_value_, logger_);
    RecordTick(statistics_, MERGE_OPERATION_TOTAL_TIME, timer.ElapsedNanos());
  }
  valid_ = true;
  return true;
}

// Used in Next to change directions
// Go to next user key
// Don't use Seek(),
// because next user key will be very close
void DBIter::FindNextUserKey() {
  if (!iter_->Valid()) {
    return;
  }
  ParsedInternalKey ikey;
  FindParseableKey(&ikey, kForward);
  while (iter_->Valid() &&
         !user_comparator_->Equal(ikey.user_key, saved_key_.GetKey())) {
    iter_->Next();
    FindParseableKey(&ikey, kForward);
  }
}

// Go to previous user_key
void DBIter::FindPrevUserKey() {
  if (!iter_->Valid()) {
    return;
  }
  size_t num_skipped = 0;
  ParsedInternalKey ikey;
  FindParseableKey(&ikey, kReverse);
  int cmp;
  while (iter_->Valid() && ((cmp = user_comparator_->Compare(
                                 ikey.user_key, saved_key_.GetKey())) == 0 ||
                            (cmp > 0 && ikey.sequence > sequence_))) {
    if (cmp == 0) {
      if (num_skipped >= max_skip_) {
        num_skipped = 0;
        IterKey last_key;
        last_key.SetInternalKey(ParsedInternalKey(
            saved_key_.GetKey(), kMaxSequenceNumber, kValueTypeForSeek));
        iter_->Seek(last_key.GetKey());
        RecordTick(statistics_, NUMBER_OF_RESEEKS_IN_ITERATION);
      } else {
        ++num_skipped;
      }
    }
    iter_->Prev();
    FindParseableKey(&ikey, kReverse);
  }
}

// Skip all unparseable keys
void DBIter::FindParseableKey(ParsedInternalKey* ikey, Direction direction) {
  while (iter_->Valid() && !ParseKey(ikey)) {
    if (direction == kReverse) {
      iter_->Prev();
    } else {
      iter_->Next();
    }
  }
}

void DBIter::Seek(const Slice& target) {
  StopWatch sw(env_, statistics_, DB_SEEK);
  saved_key_.Clear();
  // now savved_key is used to store internal key.
  saved_key_.SetInternalKey(target, sequence_);

  {
    PERF_TIMER_GUARD(seek_internal_seek_time);
    iter_->Seek(saved_key_.GetKey());
  }

  if (iter_->Valid()) {
    direction_ = kForward;
    ClearSavedValue();
    FindNextUserEntry(false /* not skipping */);
  } else {
    valid_ = false;
  }
}

void DBIter::SeekToFirst() {
  // Don't use iter_::Seek() if we set a prefix extractor
  // because prefix seek will be used.
  if (prefix_extractor_ != nullptr) {
    max_skip_ = std::numeric_limits<uint64_t>::max();
  }
  direction_ = kForward;
  ClearSavedValue();

  {
    PERF_TIMER_GUARD(seek_internal_seek_time);
    iter_->SeekToFirst();
  }

  if (iter_->Valid()) {
    FindNextUserEntry(false /* not skipping */);
  } else {
    valid_ = false;
  }
}

void DBIter::SeekToLast() {
  // Don't use iter_::Seek() if we set a prefix extractor
  // because prefix seek will be used.
  if (prefix_extractor_ != nullptr) {
    max_skip_ = std::numeric_limits<uint64_t>::max();
  }
  direction_ = kReverse;
  ClearSavedValue();

  {
    PERF_TIMER_GUARD(seek_internal_seek_time);
    iter_->SeekToLast();
  }
  // When the iterate_upper_bound is set to a value,
  // it will seek to the last key before the
  // ReadOptions.iterate_upper_bound
  if (iter_->Valid() && iterate_upper_bound_ != nullptr) {
    saved_key_.SetKey(*iterate_upper_bound_);
    std::string last_key;
    AppendInternalKey(&last_key,
                      ParsedInternalKey(saved_key_.GetKey(), kMaxSequenceNumber,
                                        kValueTypeForSeek));

    iter_->Seek(last_key);

    if (!iter_->Valid()) {
      iter_->SeekToLast();
    } else {
      iter_->Prev();
      if (!iter_->Valid()) {
        valid_ = false;
        return;
      }
    }
  }
  PrevInternal();
}

Iterator* NewDBIterator(Env* env, const ImmutableCFOptions& ioptions,
                        const Comparator* user_key_comparator,
                        Iterator* internal_iter,
                        const SequenceNumber& sequence,
                        uint64_t max_sequential_skip_in_iterations,
                        const Slice* iterate_upper_bound) {
  return new DBIter(env, ioptions, user_key_comparator, internal_iter, sequence,
                    false, max_sequential_skip_in_iterations,
                    iterate_upper_bound);
}

ArenaWrappedDBIter::~ArenaWrappedDBIter() { db_iter_->~DBIter(); }

void ArenaWrappedDBIter::SetDBIter(DBIter* iter) { db_iter_ = iter; }

void ArenaWrappedDBIter::SetIterUnderDBIter(Iterator* iter) {
  static_cast<DBIter*>(db_iter_)->SetIter(iter);
}

inline bool ArenaWrappedDBIter::Valid() const { return db_iter_->Valid(); }
inline void ArenaWrappedDBIter::SeekToFirst() { db_iter_->SeekToFirst(); }
inline void ArenaWrappedDBIter::SeekToLast() { db_iter_->SeekToLast(); }
inline void ArenaWrappedDBIter::Seek(const Slice& target) {
  db_iter_->Seek(target);
}
inline void ArenaWrappedDBIter::Next() { db_iter_->Next(); }
inline void ArenaWrappedDBIter::Prev() { db_iter_->Prev(); }
inline Slice ArenaWrappedDBIter::key() const { return db_iter_->key(); }
inline Slice ArenaWrappedDBIter::value() const { return db_iter_->value(); }
inline Status ArenaWrappedDBIter::status() const { return db_iter_->status(); }
void ArenaWrappedDBIter::RegisterCleanup(CleanupFunction function, void* arg1,
                                         void* arg2) {
  db_iter_->RegisterCleanup(function, arg1, arg2);
}

ArenaWrappedDBIter* NewArenaWrappedDbIterator(
    Env* env, const ImmutableCFOptions& ioptions,
    const Comparator* user_key_comparator,
    const SequenceNumber& sequence,
    uint64_t max_sequential_skip_in_iterations,
    const Slice* iterate_upper_bound) {
  ArenaWrappedDBIter* iter = new ArenaWrappedDBIter();
  Arena* arena = iter->GetArena();
  auto mem = arena->AllocateAligned(sizeof(DBIter));
  DBIter* db_iter = new (mem) DBIter(env, ioptions, user_key_comparator,
      nullptr, sequence, true, max_sequential_skip_in_iterations,
      iterate_upper_bound);

  iter->SetDBIter(db_iter);

  return iter;
}

}  // namespace rocksdb
