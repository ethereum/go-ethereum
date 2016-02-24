//  Copyright (c) 2013, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.
//
// Copyright (c) 2011 The LevelDB Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file. See the AUTHORS file for names of contributors.

#include "table/merger.h"

#include <vector>

#include "rocksdb/comparator.h"
#include "rocksdb/iterator.h"
#include "rocksdb/options.h"
#include "table/iter_heap.h"
#include "table/iterator_wrapper.h"
#include "util/arena.h"
#include "util/heap.h"
#include "util/stop_watch.h"
#include "util/sync_point.h"
#include "util/perf_context_imp.h"
#include "util/autovector.h"

namespace rocksdb {
// Without anonymous namespace here, we fail the warning -Wmissing-prototypes
namespace {
typedef BinaryHeap<IteratorWrapper*, MaxIteratorComparator> MergerMaxIterHeap;
typedef BinaryHeap<IteratorWrapper*, MinIteratorComparator> MergerMinIterHeap;
}  // namespace

const size_t kNumIterReserve = 4;

class MergingIterator : public Iterator {
 public:
  MergingIterator(const Comparator* comparator, Iterator** children, int n,
                  bool is_arena_mode)
      : is_arena_mode_(is_arena_mode),
        comparator_(comparator),
        current_(nullptr),
        direction_(kForward),
        minHeap_(comparator_) {
    children_.resize(n);
    for (int i = 0; i < n; i++) {
      children_[i].Set(children[i]);
    }
    for (auto& child : children_) {
      if (child.Valid()) {
        minHeap_.push(&child);
      }
    }
    current_ = CurrentForward();
  }

  virtual void AddIterator(Iterator* iter) {
    assert(direction_ == kForward);
    children_.emplace_back(iter);
    auto new_wrapper = children_.back();
    if (new_wrapper.Valid()) {
      minHeap_.push(&new_wrapper);
      current_ = CurrentForward();
    }
  }

  virtual ~MergingIterator() {
    for (auto& child : children_) {
      child.DeleteIter(is_arena_mode_);
    }
  }

  virtual bool Valid() const override { return (current_ != nullptr); }

  virtual void SeekToFirst() override {
    ClearHeaps();
    for (auto& child : children_) {
      child.SeekToFirst();
      if (child.Valid()) {
        minHeap_.push(&child);
      }
    }
    direction_ = kForward;
    current_ = CurrentForward();
  }

  virtual void SeekToLast() override {
    ClearHeaps();
    InitMaxHeap();
    for (auto& child : children_) {
      child.SeekToLast();
      if (child.Valid()) {
        maxHeap_->push(&child);
      }
    }
    direction_ = kReverse;
    current_ = CurrentReverse();
  }

  virtual void Seek(const Slice& target) override {
    ClearHeaps();
    for (auto& child : children_) {
      {
        PERF_TIMER_GUARD(seek_child_seek_time);
        child.Seek(target);
      }
      PERF_COUNTER_ADD(seek_child_seek_count, 1);

      if (child.Valid()) {
        PERF_TIMER_GUARD(seek_min_heap_time);
        minHeap_.push(&child);
      }
    }
    direction_ = kForward;
    {
      PERF_TIMER_GUARD(seek_min_heap_time);
      current_ = CurrentForward();
    }
  }

  virtual void Next() override {
    assert(Valid());

    // Ensure that all children are positioned after key().
    // If we are moving in the forward direction, it is already
    // true for all of the non-current children since current_ is
    // the smallest child and key() == current_->key().
    if (direction_ != kForward) {
      // Otherwise, advance the non-current children.  We advance current_
      // just after the if-block.
      ClearHeaps();
      for (auto& child : children_) {
        if (&child != current_) {
          child.Seek(key());
          if (child.Valid() && comparator_->Equal(key(), child.key())) {
            child.Next();
          }
        }
        if (child.Valid()) {
          minHeap_.push(&child);
        }
      }
      direction_ = kForward;
      // The loop advanced all non-current children to be > key() so current_
      // should still be strictly the smallest key.
      assert(current_ == CurrentForward());
    }

    // For the heap modifications below to be correct, current_ must be the
    // current top of the heap.
    assert(current_ == CurrentForward());

    // as the current points to the current record. move the iterator forward.
    current_->Next();
    if (current_->Valid()) {
      // current is still valid after the Next() call above.  Call
      // replace_top() to restore the heap property.  When the same child
      // iterator yields a sequence of keys, this is cheap.
      minHeap_.replace_top(current_);
    } else {
      // current stopped being valid, remove it from the heap.
      minHeap_.pop();
    }
    current_ = CurrentForward();
  }

  virtual void Prev() override {
    assert(Valid());
    // Ensure that all children are positioned before key().
    // If we are moving in the reverse direction, it is already
    // true for all of the non-current children since current_ is
    // the largest child and key() == current_->key().
    if (direction_ != kReverse) {
      // Otherwise, retreat the non-current children.  We retreat current_
      // just after the if-block.
      ClearHeaps();
      InitMaxHeap();
      for (auto& child : children_) {
        if (&child != current_) {
          child.Seek(key());
          if (child.Valid()) {
            // Child is at first entry >= key().  Step back one to be < key()
            TEST_SYNC_POINT_CALLBACK("MergeIterator::Prev:BeforePrev", &child);
            child.Prev();
          } else {
            // Child has no entries >= key().  Position at last entry.
            TEST_SYNC_POINT("MergeIterator::Prev:BeforeSeekToLast");
            child.SeekToLast();
          }
        }
        if (child.Valid()) {
          maxHeap_->push(&child);
        }
      }
      direction_ = kReverse;
      // Note that we don't do assert(current_ == CurrentReverse()) here
      // because it is possible to have some keys larger than the seek-key
      // inserted between Seek() and SeekToLast(), which makes current_ not
      // equal to CurrentReverse().
      current_ = CurrentReverse();
    }

    // For the heap modifications below to be correct, current_ must be the
    // current top of the heap.
    assert(current_ == CurrentReverse());

    current_->Prev();
    if (current_->Valid()) {
      // current is still valid after the Prev() call above.  Call
      // replace_top() to restore the heap property.  When the same child
      // iterator yields a sequence of keys, this is cheap.
      maxHeap_->replace_top(current_);
    } else {
      // current stopped being valid, remove it from the heap.
      maxHeap_->pop();
    }
    current_ = CurrentReverse();
  }

  virtual Slice key() const override {
    assert(Valid());
    return current_->key();
  }

  virtual Slice value() const override {
    assert(Valid());
    return current_->value();
  }

  virtual Status status() const override {
    Status s;
    for (auto& child : children_) {
      s = child.status();
      if (!s.ok()) {
        break;
      }
    }
    return s;
  }

 private:
  // Clears heaps for both directions, used when changing direction or seeking
  void ClearHeaps();
  // Ensures that maxHeap_ is initialized when starting to go in the reverse
  // direction
  void InitMaxHeap();

  bool is_arena_mode_;
  const Comparator* comparator_;
  autovector<IteratorWrapper, kNumIterReserve> children_;

  // Cached pointer to child iterator with the current key, or nullptr if no
  // child iterators are valid.  This is the top of minHeap_ or maxHeap_
  // depending on the direction.
  IteratorWrapper* current_;
  // Which direction is the iterator moving?
  enum Direction {
    kForward,
    kReverse
  };
  Direction direction_;
  MergerMinIterHeap minHeap_;
  // Max heap is used for reverse iteration, which is way less common than
  // forward.  Lazily initialize it to save memory.
  std::unique_ptr<MergerMaxIterHeap> maxHeap_;

  IteratorWrapper* CurrentForward() const {
    assert(direction_ == kForward);
    return !minHeap_.empty() ? minHeap_.top() : nullptr;
  }

  IteratorWrapper* CurrentReverse() const {
    assert(direction_ == kReverse);
    assert(maxHeap_);
    return !maxHeap_->empty() ? maxHeap_->top() : nullptr;
  }
};

void MergingIterator::ClearHeaps() {
  minHeap_.clear();
  if (maxHeap_) {
    maxHeap_->clear();
  }
}

void MergingIterator::InitMaxHeap() {
  if (!maxHeap_) {
    maxHeap_.reset(new MergerMaxIterHeap(comparator_));
  }
}

Iterator* NewMergingIterator(const Comparator* cmp, Iterator** list, int n,
                             Arena* arena) {
  assert(n >= 0);
  if (n == 0) {
    return NewEmptyIterator(arena);
  } else if (n == 1) {
    return list[0];
  } else {
    if (arena == nullptr) {
      return new MergingIterator(cmp, list, n, false);
    } else {
      auto mem = arena->AllocateAligned(sizeof(MergingIterator));
      return new (mem) MergingIterator(cmp, list, n, true);
    }
  }
}

MergeIteratorBuilder::MergeIteratorBuilder(const Comparator* comparator,
                                           Arena* a)
    : first_iter(nullptr), use_merging_iter(false), arena(a) {

  auto mem = arena->AllocateAligned(sizeof(MergingIterator));
  merge_iter = new (mem) MergingIterator(comparator, nullptr, 0, true);
}

void MergeIteratorBuilder::AddIterator(Iterator* iter) {
  if (!use_merging_iter && first_iter != nullptr) {
    merge_iter->AddIterator(first_iter);
    use_merging_iter = true;
  }
  if (use_merging_iter) {
    merge_iter->AddIterator(iter);
  } else {
    first_iter = iter;
  }
}

Iterator* MergeIteratorBuilder::Finish() {
  if (!use_merging_iter) {
    return first_iter;
  } else {
    auto ret = merge_iter;
    merge_iter = nullptr;
    return ret;
  }
}

}  // namespace rocksdb
