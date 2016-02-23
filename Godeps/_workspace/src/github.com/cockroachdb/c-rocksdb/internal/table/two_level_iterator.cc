//  Copyright (c) 2013, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.
//
// Copyright (c) 2011 The LevelDB Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file. See the AUTHORS file for names of contributors.

#include "table/two_level_iterator.h"

#include "rocksdb/options.h"
#include "rocksdb/table.h"
#include "table/block.h"
#include "table/format.h"
#include "util/arena.h"

namespace rocksdb {

namespace {

class TwoLevelIterator: public Iterator {
 public:
  explicit TwoLevelIterator(TwoLevelIteratorState* state,
                            Iterator* first_level_iter,
                            bool need_free_iter_and_state);

  virtual ~TwoLevelIterator() {
    first_level_iter_.DeleteIter(!need_free_iter_and_state_);
    second_level_iter_.DeleteIter(false);
    if (need_free_iter_and_state_) {
      delete state_;
    } else {
      state_->~TwoLevelIteratorState();
    }
  }

  virtual void Seek(const Slice& target) override;
  virtual void SeekToFirst() override;
  virtual void SeekToLast() override;
  virtual void Next() override;
  virtual void Prev() override;

  virtual bool Valid() const override { return second_level_iter_.Valid(); }
  virtual Slice key() const override {
    assert(Valid());
    return second_level_iter_.key();
  }
  virtual Slice value() const override {
    assert(Valid());
    return second_level_iter_.value();
  }
  virtual Status status() const override {
    // It'd be nice if status() returned a const Status& instead of a Status
    if (!first_level_iter_.status().ok()) {
      return first_level_iter_.status();
    } else if (second_level_iter_.iter() != nullptr &&
               !second_level_iter_.status().ok()) {
      return second_level_iter_.status();
    } else {
      return status_;
    }
  }

 private:
  void SaveError(const Status& s) {
    if (status_.ok() && !s.ok()) status_ = s;
  }
  void SkipEmptyDataBlocksForward();
  void SkipEmptyDataBlocksBackward();
  void SetSecondLevelIterator(Iterator* iter);
  void InitDataBlock();

  TwoLevelIteratorState* state_;
  IteratorWrapper first_level_iter_;
  IteratorWrapper second_level_iter_;  // May be nullptr
  bool need_free_iter_and_state_;
  Status status_;
  // If second_level_iter is non-nullptr, then "data_block_handle_" holds the
  // "index_value" passed to block_function_ to create the second_level_iter.
  std::string data_block_handle_;
};

TwoLevelIterator::TwoLevelIterator(TwoLevelIteratorState* state,
                                   Iterator* first_level_iter,
                                   bool need_free_iter_and_state)
    : state_(state),
      first_level_iter_(first_level_iter),
      need_free_iter_and_state_(need_free_iter_and_state) {}

void TwoLevelIterator::Seek(const Slice& target) {
  if (state_->check_prefix_may_match &&
      !state_->PrefixMayMatch(target)) {
    SetSecondLevelIterator(nullptr);
    return;
  }
  first_level_iter_.Seek(target);

  InitDataBlock();
  if (second_level_iter_.iter() != nullptr) {
    second_level_iter_.Seek(target);
  }
  SkipEmptyDataBlocksForward();
}

void TwoLevelIterator::SeekToFirst() {
  first_level_iter_.SeekToFirst();
  InitDataBlock();
  if (second_level_iter_.iter() != nullptr) {
    second_level_iter_.SeekToFirst();
  }
  SkipEmptyDataBlocksForward();
}

void TwoLevelIterator::SeekToLast() {
  first_level_iter_.SeekToLast();
  InitDataBlock();
  if (second_level_iter_.iter() != nullptr) {
    second_level_iter_.SeekToLast();
  }
  SkipEmptyDataBlocksBackward();
}

void TwoLevelIterator::Next() {
  assert(Valid());
  second_level_iter_.Next();
  SkipEmptyDataBlocksForward();
}

void TwoLevelIterator::Prev() {
  assert(Valid());
  second_level_iter_.Prev();
  SkipEmptyDataBlocksBackward();
}


void TwoLevelIterator::SkipEmptyDataBlocksForward() {
  while (second_level_iter_.iter() == nullptr ||
         (!second_level_iter_.Valid() &&
         !second_level_iter_.status().IsIncomplete())) {
    // Move to next block
    if (!first_level_iter_.Valid()) {
      SetSecondLevelIterator(nullptr);
      return;
    }
    first_level_iter_.Next();
    InitDataBlock();
    if (second_level_iter_.iter() != nullptr) {
      second_level_iter_.SeekToFirst();
    }
  }
}

void TwoLevelIterator::SkipEmptyDataBlocksBackward() {
  while (second_level_iter_.iter() == nullptr ||
         (!second_level_iter_.Valid() &&
         !second_level_iter_.status().IsIncomplete())) {
    // Move to next block
    if (!first_level_iter_.Valid()) {
      SetSecondLevelIterator(nullptr);
      return;
    }
    first_level_iter_.Prev();
    InitDataBlock();
    if (second_level_iter_.iter() != nullptr) {
      second_level_iter_.SeekToLast();
    }
  }
}

void TwoLevelIterator::SetSecondLevelIterator(Iterator* iter) {
  if (second_level_iter_.iter() != nullptr) {
    SaveError(second_level_iter_.status());
  }
  second_level_iter_.Set(iter);
}

void TwoLevelIterator::InitDataBlock() {
  if (!first_level_iter_.Valid()) {
    SetSecondLevelIterator(nullptr);
  } else {
    Slice handle = first_level_iter_.value();
    if (second_level_iter_.iter() != nullptr &&
        !second_level_iter_.status().IsIncomplete() &&
        handle.compare(data_block_handle_) == 0) {
      // second_level_iter is already constructed with this iterator, so
      // no need to change anything
    } else {
      Iterator* iter = state_->NewSecondaryIterator(handle);
      data_block_handle_.assign(handle.data(), handle.size());
      SetSecondLevelIterator(iter);
    }
  }
}

}  // namespace

Iterator* NewTwoLevelIterator(TwoLevelIteratorState* state,
                              Iterator* first_level_iter, Arena* arena,
                              bool need_free_iter_and_state) {
  if (arena == nullptr) {
    return new TwoLevelIterator(state, first_level_iter,
                                need_free_iter_and_state);
  } else {
    auto mem = arena->AllocateAligned(sizeof(TwoLevelIterator));
    return new (mem)
        TwoLevelIterator(state, first_level_iter, need_free_iter_and_state);
  }
}

}  // namespace rocksdb
