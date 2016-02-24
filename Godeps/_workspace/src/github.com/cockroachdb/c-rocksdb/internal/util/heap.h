//  Copyright (c) 2013, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.

#pragma once

#include <algorithm>
#include <cstdint>
#include <functional>
#include "util/autovector.h"

namespace rocksdb {

// Binary heap implementation optimized for use in multi-way merge sort.
// Comparison to std::priority_queue:
// - In libstdc++, std::priority_queue::pop() usually performs just over logN
//   comparisons but never fewer.
// - std::priority_queue does not have a replace-top operation, requiring a
//   pop+push.  If the replacement element is the new top, this requires
//   around 2logN comparisons.
// - This heap's pop() uses a "schoolbook" downheap which requires up to ~2logN
//   comparisons.
// - This heap provides a replace_top() operation which requires [1, 2logN]
//   comparisons.  When the replacement element is also the new top, this
//   takes just 1 or 2 comparisons.
//
// The last property can yield an order-of-magnitude performance improvement
// when merge-sorting real-world non-random data.  If the merge operation is
// likely to take chunks of elements from the same input stream, only 1
// comparison per element is needed.  In RocksDB-land, this happens when
// compacting a database where keys are not randomly distributed across L0
// files but nearby keys are likely to be in the same L0 file.
//
// The container uses the same counterintuitive ordering as
// std::priority_queue: the comparison operator is expected to provide the
// less-than relation, but top() will return the maximum.

template<typename T, typename Compare = std::less<T>>
class BinaryHeap {
 public:
  BinaryHeap() { }
  explicit BinaryHeap(Compare cmp) : cmp_(std::move(cmp)) { }

  void push(const T& value) {
    data_.push_back(value);
    upheap(data_.size() - 1);
  }

  void push(T&& value) {
    data_.push_back(std::move(value));
    upheap(data_.size() - 1);
  }

  const T& top() const {
    assert(!empty());
    return data_.front();
  }

  void replace_top(const T& value) {
    assert(!empty());
    data_.front() = value;
    downheap(get_root());
  }

  void replace_top(T&& value) {
    assert(!empty());
    data_.front() = std::move(value);
    downheap(get_root());
  }

  void pop() {
    assert(!empty());
    data_.front() = std::move(data_.back());
    data_.pop_back();
    if (!empty()) {
      downheap(get_root());
    }
  }

  void swap(BinaryHeap &other) {
    std::swap(cmp_, other.cmp_);
    data_.swap(other.data_);
  }

  void clear() {
    data_.clear();
  }

  bool empty() const {
    return data_.empty();
  }

 private:
  static inline size_t get_root() { return 0; }
  static inline size_t get_parent(size_t index) { return (index - 1) / 2; }
  static inline size_t get_left(size_t index) { return 2 * index + 1; }
  static inline size_t get_right(size_t index) { return 2 * index + 2; }

  void upheap(size_t index) {
    T v = std::move(data_[index]);
    while (index > get_root()) {
      const size_t parent = get_parent(index);
      if (!cmp_(data_[parent], v)) {
        break;
      }
      data_[index] = std::move(data_[parent]);
      index = parent;
    }
    data_[index] = std::move(v);
  }

  void downheap(size_t index) {
    T v = std::move(data_[index]);
    while (1) {
      const size_t left_child = get_left(index);
      if (get_left(index) >= data_.size()) {
        break;
      }
      const size_t right_child = left_child + 1;
      assert(right_child == get_right(index));
      size_t picked_child = left_child;
      if (right_child < data_.size() &&
          cmp_(data_[left_child], data_[right_child])) {
        picked_child = right_child;
      }
      if (!cmp_(v, data_[picked_child])) {
        break;
      }
      data_[index] = std::move(data_[picked_child]);
      index = picked_child;
    }
    data_[index] = std::move(v);
  }

  Compare cmp_;
  autovector<T> data_;
};

}  // namespace rocksdb
