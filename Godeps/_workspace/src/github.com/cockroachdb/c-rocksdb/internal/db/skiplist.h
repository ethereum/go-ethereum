//  Copyright (c) 2013, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.
//
// Copyright (c) 2011 The LevelDB Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file. See the AUTHORS file for names of contributors.
//
// Thread safety
// -------------
//
// Writes require external synchronization, most likely a mutex.
// Reads require a guarantee that the SkipList will not be destroyed
// while the read is in progress.  Apart from that, reads progress
// without any internal locking or synchronization.
//
// Invariants:
//
// (1) Allocated nodes are never deleted until the SkipList is
// destroyed.  This is trivially guaranteed by the code since we
// never delete any skip list nodes.
//
// (2) The contents of a Node except for the next/prev pointers are
// immutable after the Node has been linked into the SkipList.
// Only Insert() modifies the list, and it is careful to initialize
// a node and use release-stores to publish the nodes in one or
// more lists.
//
// ... prev vs. next pointer ordering ...
//

#pragma once
#include <assert.h>
#include <atomic>
#include <stdlib.h>
#include "port/port.h"
#include "util/allocator.h"
#include "util/random.h"

namespace rocksdb {

template<typename Key, class Comparator>
class SkipList {
 private:
  struct Node;

 public:
  // Create a new SkipList object that will use "cmp" for comparing keys,
  // and will allocate memory using "*allocator".  Objects allocated in the
  // allocator must remain allocated for the lifetime of the skiplist object.
  explicit SkipList(Comparator cmp, Allocator* allocator,
                    int32_t max_height = 12, int32_t branching_factor = 4);

  // Insert key into the list.
  // REQUIRES: nothing that compares equal to key is currently in the list.
  void Insert(const Key& key);

  // Returns true iff an entry that compares equal to key is in the list.
  bool Contains(const Key& key) const;

  // Return estimated number of entries smaller than `key`.
  uint64_t EstimateCount(const Key& key) const;

  // Iteration over the contents of a skip list
  class Iterator {
   public:
    // Initialize an iterator over the specified list.
    // The returned iterator is not valid.
    explicit Iterator(const SkipList* list);

    // Change the underlying skiplist used for this iterator
    // This enables us not changing the iterator without deallocating
    // an old one and then allocating a new one
    void SetList(const SkipList* list);

    // Returns true iff the iterator is positioned at a valid node.
    bool Valid() const;

    // Returns the key at the current position.
    // REQUIRES: Valid()
    const Key& key() const;

    // Advances to the next position.
    // REQUIRES: Valid()
    void Next();

    // Advances to the previous position.
    // REQUIRES: Valid()
    void Prev();

    // Advance to the first entry with a key >= target
    void Seek(const Key& target);

    // Position at the first entry in list.
    // Final state of iterator is Valid() iff list is not empty.
    void SeekToFirst();

    // Position at the last entry in list.
    // Final state of iterator is Valid() iff list is not empty.
    void SeekToLast();

   private:
    const SkipList* list_;
    Node* node_;
    // Intentionally copyable
  };

 private:
  const int32_t kMaxHeight_;
  const int32_t kBranching_;

  // Immutable after construction
  Comparator const compare_;
  Allocator* const allocator_;    // Allocator used for allocations of nodes

  Node* const head_;

  // Modified only by Insert().  Read racily by readers, but stale
  // values are ok.
  std::atomic<int> max_height_;  // Height of the entire list

  // Used for optimizing sequential insert patterns.  Tricky.  prev_[i] for
  // i up to max_height_ is the predecessor of prev_[0] and prev_height_
  // is the height of prev_[0].  prev_[0] can only be equal to head before
  // insertion, in which case max_height_ and prev_height_ are 1.
  Node** prev_;
  int32_t prev_height_;

  inline int GetMaxHeight() const {
    return max_height_.load(std::memory_order_relaxed);
  }

  // Read/written only by Insert().
  Random rnd_;

  Node* NewNode(const Key& key, int height);
  int RandomHeight();
  bool Equal(const Key& a, const Key& b) const { return (compare_(a, b) == 0); }

  // Return true if key is greater than the data stored in "n"
  bool KeyIsAfterNode(const Key& key, Node* n) const;

  // Returns the earliest node with a key >= key.
  // Return nullptr if there is no such node.
  Node* FindGreaterOrEqual(const Key& key) const;

  // Return the latest node with a key < key.
  // Return head_ if there is no such node.
  // Fills prev[level] with pointer to previous node at "level" for every
  // level in [0..max_height_-1], if prev is non-null.
  Node* FindLessThan(const Key& key, Node** prev = nullptr) const;

  // Return the last node in the list.
  // Return head_ if list is empty.
  Node* FindLast() const;

  // No copying allowed
  SkipList(const SkipList&);
  void operator=(const SkipList&);
};

// Implementation details follow
template<typename Key, class Comparator>
struct SkipList<Key, Comparator>::Node {
  explicit Node(const Key& k) : key(k) { }

  Key const key;

  // Accessors/mutators for links.  Wrapped in methods so we can
  // add the appropriate barriers as necessary.
  Node* Next(int n) {
    assert(n >= 0);
    // Use an 'acquire load' so that we observe a fully initialized
    // version of the returned Node.
    return (next_[n].load(std::memory_order_acquire));
  }
  void SetNext(int n, Node* x) {
    assert(n >= 0);
    // Use a 'release store' so that anybody who reads through this
    // pointer observes a fully initialized version of the inserted node.
    next_[n].store(x, std::memory_order_release);
  }

  // No-barrier variants that can be safely used in a few locations.
  Node* NoBarrier_Next(int n) {
    assert(n >= 0);
    return next_[n].load(std::memory_order_relaxed);
  }
  void NoBarrier_SetNext(int n, Node* x) {
    assert(n >= 0);
    next_[n].store(x, std::memory_order_relaxed);
  }

 private:
  // Array of length equal to the node height.  next_[0] is lowest level link.
  std::atomic<Node*> next_[1];
};

template<typename Key, class Comparator>
typename SkipList<Key, Comparator>::Node*
SkipList<Key, Comparator>::NewNode(const Key& key, int height) {
  char* mem = allocator_->AllocateAligned(
      sizeof(Node) + sizeof(std::atomic<Node*>) * (height - 1));
  return new (mem) Node(key);
}

template<typename Key, class Comparator>
inline SkipList<Key, Comparator>::Iterator::Iterator(const SkipList* list) {
  SetList(list);
}

template<typename Key, class Comparator>
inline void SkipList<Key, Comparator>::Iterator::SetList(const SkipList* list) {
  list_ = list;
  node_ = nullptr;
}

template<typename Key, class Comparator>
inline bool SkipList<Key, Comparator>::Iterator::Valid() const {
  return node_ != nullptr;
}

template<typename Key, class Comparator>
inline const Key& SkipList<Key, Comparator>::Iterator::key() const {
  assert(Valid());
  return node_->key;
}

template<typename Key, class Comparator>
inline void SkipList<Key, Comparator>::Iterator::Next() {
  assert(Valid());
  node_ = node_->Next(0);
}

template<typename Key, class Comparator>
inline void SkipList<Key, Comparator>::Iterator::Prev() {
  // Instead of using explicit "prev" links, we just search for the
  // last node that falls before key.
  assert(Valid());
  node_ = list_->FindLessThan(node_->key);
  if (node_ == list_->head_) {
    node_ = nullptr;
  }
}

template<typename Key, class Comparator>
inline void SkipList<Key, Comparator>::Iterator::Seek(const Key& target) {
  node_ = list_->FindGreaterOrEqual(target);
}

template<typename Key, class Comparator>
inline void SkipList<Key, Comparator>::Iterator::SeekToFirst() {
  node_ = list_->head_->Next(0);
}

template<typename Key, class Comparator>
inline void SkipList<Key, Comparator>::Iterator::SeekToLast() {
  node_ = list_->FindLast();
  if (node_ == list_->head_) {
    node_ = nullptr;
  }
}

template<typename Key, class Comparator>
int SkipList<Key, Comparator>::RandomHeight() {
  // Increase height with probability 1 in kBranching
  int height = 1;
  while (height < kMaxHeight_ && ((rnd_.Next() % kBranching_) == 0)) {
    height++;
  }
  assert(height > 0);
  assert(height <= kMaxHeight_);
  return height;
}

template<typename Key, class Comparator>
bool SkipList<Key, Comparator>::KeyIsAfterNode(const Key& key, Node* n) const {
  // nullptr n is considered infinite
  return (n != nullptr) && (compare_(n->key, key) < 0);
}

template<typename Key, class Comparator>
typename SkipList<Key, Comparator>::Node* SkipList<Key, Comparator>::
  FindGreaterOrEqual(const Key& key) const {
  // Note: It looks like we could reduce duplication by implementing
  // this function as FindLessThan(key)->Next(0), but we wouldn't be able
  // to exit early on equality and the result wouldn't even be correct.
  // A concurrent insert might occur after FindLessThan(key) but before
  // we get a chance to call Next(0).
  Node* x = head_;
  int level = GetMaxHeight() - 1;
  Node* last_bigger = nullptr;
  while (true) {
    Node* next = x->Next(level);
    // Make sure the lists are sorted
    assert(x == head_ || next == nullptr || KeyIsAfterNode(next->key, x));
    // Make sure we haven't overshot during our search
    assert(x == head_ || KeyIsAfterNode(key, x));
    int cmp = (next == nullptr || next == last_bigger)
        ? 1 : compare_(next->key, key);
    if (cmp == 0 || (cmp > 0 && level == 0)) {
      return next;
    } else if (cmp < 0) {
      // Keep searching in this list
      x = next;
    } else {
      // Switch to next list, reuse compare_() result
      last_bigger = next;
      level--;
    }
  }
}

template<typename Key, class Comparator>
typename SkipList<Key, Comparator>::Node*
SkipList<Key, Comparator>::FindLessThan(const Key& key, Node** prev) const {
  Node* x = head_;
  int level = GetMaxHeight() - 1;
  // KeyIsAfter(key, last_not_after) is definitely false
  Node* last_not_after = nullptr;
  while (true) {
    Node* next = x->Next(level);
    assert(x == head_ || next == nullptr || KeyIsAfterNode(next->key, x));
    assert(x == head_ || KeyIsAfterNode(key, x));
    if (next != last_not_after && KeyIsAfterNode(key, next)) {
      // Keep searching in this list
      x = next;
    } else {
      if (prev != nullptr) {
        prev[level] = x;
      }
      if (level == 0) {
        return x;
      } else {
        // Switch to next list, reuse KeyIUsAfterNode() result
        last_not_after = next;
        level--;
      }
    }
  }
}

template<typename Key, class Comparator>
typename SkipList<Key, Comparator>::Node* SkipList<Key, Comparator>::FindLast()
    const {
  Node* x = head_;
  int level = GetMaxHeight() - 1;
  while (true) {
    Node* next = x->Next(level);
    if (next == nullptr) {
      if (level == 0) {
        return x;
      } else {
        // Switch to next list
        level--;
      }
    } else {
      x = next;
    }
  }
}

template <typename Key, class Comparator>
uint64_t SkipList<Key, Comparator>::EstimateCount(const Key& key) const {
  uint64_t count = 0;

  Node* x = head_;
  int level = GetMaxHeight() - 1;
  while (true) {
    assert(x == head_ || compare_(x->key, key) < 0);
    Node* next = x->Next(level);
    if (next == nullptr || compare_(next->key, key) >= 0) {
      if (level == 0) {
        return count;
      } else {
        // Switch to next list
        count *= kBranching_;
        level--;
      }
    } else {
      x = next;
      count++;
    }
  }
}

template <typename Key, class Comparator>
SkipList<Key, Comparator>::SkipList(const Comparator cmp, Allocator* allocator,
                                    int32_t max_height,
                                    int32_t branching_factor)
    : kMaxHeight_(max_height),
      kBranching_(branching_factor),
      compare_(cmp),
      allocator_(allocator),
      head_(NewNode(0 /* any key will do */, max_height)),
      max_height_(1),
      prev_height_(1),
      rnd_(0xdeadbeef) {
  assert(kMaxHeight_ > 0);
  assert(kBranching_ > 0);
  // Allocate the prev_ Node* array, directly from the passed-in allocator.
  // prev_ does not need to be freed, as its life cycle is tied up with
  // the allocator as a whole.
  prev_ = reinterpret_cast<Node**>(
            allocator_->AllocateAligned(sizeof(Node*) * kMaxHeight_));
  for (int i = 0; i < kMaxHeight_; i++) {
    head_->SetNext(i, nullptr);
    prev_[i] = head_;
  }
}

template<typename Key, class Comparator>
void SkipList<Key, Comparator>::Insert(const Key& key) {
  // fast path for sequential insertion
  if (!KeyIsAfterNode(key, prev_[0]->NoBarrier_Next(0)) &&
      (prev_[0] == head_ || KeyIsAfterNode(key, prev_[0]))) {
    assert(prev_[0] != head_ || (prev_height_ == 1 && GetMaxHeight() == 1));

    // Outside of this method prev_[1..max_height_] is the predecessor
    // of prev_[0], and prev_height_ refers to prev_[0].  Inside Insert
    // prev_[0..max_height - 1] is the predecessor of key.  Switch from
    // the external state to the internal
    for (int i = 1; i < prev_height_; i++) {
      prev_[i] = prev_[0];
    }
  } else {
    // TODO(opt): we could use a NoBarrier predecessor search as an
    // optimization for architectures where memory_order_acquire needs
    // a synchronization instruction.  Doesn't matter on x86
    FindLessThan(key, prev_);
  }

  // Our data structure does not allow duplicate insertion
  assert(prev_[0]->Next(0) == nullptr || !Equal(key, prev_[0]->Next(0)->key));

  int height = RandomHeight();
  if (height > GetMaxHeight()) {
    for (int i = GetMaxHeight(); i < height; i++) {
      prev_[i] = head_;
    }
    //fprintf(stderr, "Change height from %d to %d\n", max_height_, height);

    // It is ok to mutate max_height_ without any synchronization
    // with concurrent readers.  A concurrent reader that observes
    // the new value of max_height_ will see either the old value of
    // new level pointers from head_ (nullptr), or a new value set in
    // the loop below.  In the former case the reader will
    // immediately drop to the next level since nullptr sorts after all
    // keys.  In the latter case the reader will use the new node.
    max_height_.store(height, std::memory_order_relaxed);
  }

  Node* x = NewNode(key, height);
  for (int i = 0; i < height; i++) {
    // NoBarrier_SetNext() suffices since we will add a barrier when
    // we publish a pointer to "x" in prev[i].
    x->NoBarrier_SetNext(i, prev_[i]->NoBarrier_Next(i));
    prev_[i]->SetNext(i, x);
  }
  prev_[0] = x;
  prev_height_ = height;
}

template<typename Key, class Comparator>
bool SkipList<Key, Comparator>::Contains(const Key& key) const {
  Node* x = FindGreaterOrEqual(key);
  if (x != nullptr && Equal(key, x->key)) {
    return true;
  } else {
    return false;
  }
}

}  // namespace rocksdb
