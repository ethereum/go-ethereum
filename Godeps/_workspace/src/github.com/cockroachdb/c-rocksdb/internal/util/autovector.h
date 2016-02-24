//  Copyright (c) 2013, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.
#pragma once

#include <algorithm>
#include <cassert>
#include <stdexcept>
#include <iterator>
#include <vector>

namespace rocksdb {

#ifdef ROCKSDB_LITE
template <class T, size_t kSize = 8>
class autovector : public std::vector<T> {};
#else
// A vector that leverages pre-allocated stack-based array to achieve better
// performance for array with small amount of items.
//
// The interface resembles that of vector, but with less features since we aim
// to solve the problem that we have in hand, rather than implementing a
// full-fledged generic container.
//
// Currently we don't support:
//  * reserve()/shrink_to_fit()
//     If used correctly, in most cases, people should not touch the
//     underlying vector at all.
//  * random insert()/erase(), please only use push_back()/pop_back().
//  * No move/swap operations. Each autovector instance has a
//     stack-allocated array and if we want support move/swap operations, we
//     need to copy the arrays other than just swapping the pointers. In this
//     case we'll just explicitly forbid these operations since they may
//     lead users to make false assumption by thinking they are inexpensive
//     operations.
//
// Naming style of public methods almost follows that of the STL's.
template <class T, size_t kSize = 8>
class autovector {
 public:
  // General STL-style container member types.
  typedef T value_type;
  typedef typename std::vector<T>::difference_type difference_type;
  typedef typename std::vector<T>::size_type size_type;
  typedef value_type& reference;
  typedef const value_type& const_reference;
  typedef value_type* pointer;
  typedef const value_type* const_pointer;

  // This class is the base for regular/const iterator
  template <class TAutoVector, class TValueType>
  class iterator_impl {
   public:
    // -- iterator traits
    typedef iterator_impl<TAutoVector, TValueType> self_type;
    typedef TValueType value_type;
    typedef TValueType& reference;
    typedef TValueType* pointer;
    typedef typename TAutoVector::difference_type difference_type;
    typedef std::random_access_iterator_tag iterator_category;

    iterator_impl(TAutoVector* vect, size_t index)
        : vect_(vect), index_(index) {};
    iterator_impl(const iterator_impl&) = default;
    ~iterator_impl() {}
    iterator_impl& operator=(const iterator_impl&) = default;

    // -- Advancement
    // ++iterator
    self_type& operator++() {
      ++index_;
      return *this;
    }

    // iterator++
    self_type operator++(int) {
      auto old = *this;
      ++index_;
      return old;
    }

    // --iterator
    self_type& operator--() {
      --index_;
      return *this;
    }

    // iterator--
    self_type operator--(int) {
      auto old = *this;
      --index_;
      return old;
    }

    self_type operator-(difference_type len) {
      return self_type(vect_, index_ - len);
    }

    difference_type operator-(const self_type& other) {
      assert(vect_ == other.vect_);
      return index_ - other.index_;
    }

    self_type operator+(difference_type len) {
      return self_type(vect_, index_ + len);
    }

    self_type& operator+=(difference_type len) {
      index_ += len;
      return *this;
    }

    self_type& operator-=(difference_type len) {
      index_ -= len;
      return *this;
    }

    // -- Reference
    reference operator*() {
      assert(vect_->size() >= index_);
      return (*vect_)[index_];
    }
    pointer operator->() {
      assert(vect_->size() >= index_);
      return &(*vect_)[index_];
    }

    // -- Logical Operators
    bool operator==(const self_type& other) const {
      assert(vect_ == other.vect_);
      return index_ == other.index_;
    }

    bool operator!=(const self_type& other) const { return !(*this == other); }

    bool operator>(const self_type& other) const {
      assert(vect_ == other.vect_);
      return index_ > other.index_;
    }

    bool operator<(const self_type& other) const {
      assert(vect_ == other.vect_);
      return index_ < other.index_;
    }

    bool operator>=(const self_type& other) const {
      assert(vect_ == other.vect_);
      return index_ >= other.index_;
    }

    bool operator<=(const self_type& other) const {
      assert(vect_ == other.vect_);
      return index_ <= other.index_;
    }

   private:
    TAutoVector* vect_ = nullptr;
    size_t index_ = 0;
  };

  typedef iterator_impl<autovector, value_type> iterator;
  typedef iterator_impl<const autovector, const value_type> const_iterator;
  typedef std::reverse_iterator<iterator> reverse_iterator;
  typedef std::reverse_iterator<const_iterator> const_reverse_iterator;

  autovector() = default;
  ~autovector() = default;

  // -- Immutable operations
  // Indicate if all data resides in in-stack data structure.
  bool only_in_stack() const {
    // If no element was inserted at all, the vector's capacity will be `0`.
    return vect_.capacity() == 0;
  }

  size_type size() const { return num_stack_items_ + vect_.size(); }

  // resize does not guarantee anything about the contents of the newly
  // available elements
  void resize(size_type n) {
    if (n > kSize) {
      vect_.resize(n - kSize);
      num_stack_items_ = kSize;
    } else {
      vect_.clear();
      num_stack_items_ = n;
    }
  }

  bool empty() const { return size() == 0; }

  const_reference operator[](size_type n) const {
    assert(n < size());
    return n < kSize ? values_[n] : vect_[n - kSize];
  }

  reference operator[](size_type n) {
    assert(n < size());
    return n < kSize ? values_[n] : vect_[n - kSize];
  }

  const_reference at(size_type n) const {
    assert(n < size());
    return (*this)[n];
  }

  reference at(size_type n) {
    assert(n < size());
    return (*this)[n];
  }

  reference front() {
    assert(!empty());
    return *begin();
  }

  const_reference front() const {
    assert(!empty());
    return *begin();
  }

  reference back() {
    assert(!empty());
    return *(end() - 1);
  }

  const_reference back() const {
    assert(!empty());
    return *(end() - 1);
  }

  // -- Mutable Operations
  void push_back(T&& item) {
    if (num_stack_items_ < kSize) {
      values_[num_stack_items_++] = std::move(item);
    } else {
      vect_.push_back(item);
    }
  }

  void push_back(const T& item) {
    if (num_stack_items_ < kSize) {
      values_[num_stack_items_++] = item;
    } else {
      vect_.push_back(item);
    }
  }

  template <class... Args>
  void emplace_back(Args&&... args) {
    push_back(value_type(args...));
  }

  void pop_back() {
    assert(!empty());
    if (!vect_.empty()) {
      vect_.pop_back();
    } else {
      --num_stack_items_;
    }
  }

  void clear() {
    num_stack_items_ = 0;
    vect_.clear();
  }

  // -- Copy and Assignment
  autovector& assign(const autovector& other);

  autovector(const autovector& other) { assign(other); }

  autovector& operator=(const autovector& other) { return assign(other); }

  // move operation are disallowed since it is very hard to make sure both
  // autovectors are allocated from the same function stack.
  autovector& operator=(autovector&& other) = delete;
  autovector(autovector&& other) = delete;

  // -- Iterator Operations
  iterator begin() { return iterator(this, 0); }

  const_iterator begin() const { return const_iterator(this, 0); }

  iterator end() { return iterator(this, this->size()); }

  const_iterator end() const { return const_iterator(this, this->size()); }

  reverse_iterator rbegin() { return reverse_iterator(end()); }

  const_reverse_iterator rbegin() const {
    return const_reverse_iterator(end());
  }

  reverse_iterator rend() { return reverse_iterator(begin()); }

  const_reverse_iterator rend() const {
    return const_reverse_iterator(begin());
  }

 private:
  size_type num_stack_items_ = 0;  // current number of items
  value_type values_[kSize];       // the first `kSize` items
  // used only if there are more than `kSize` items.
  std::vector<T> vect_;
};

template <class T, size_t kSize>
autovector<T, kSize>& autovector<T, kSize>::assign(const autovector& other) {
  // copy the internal vector
  vect_.assign(other.vect_.begin(), other.vect_.end());

  // copy array
  num_stack_items_ = other.num_stack_items_;
  std::copy(other.values_, other.values_ + num_stack_items_, values_);

  return *this;
}
#endif  // ROCKSDB_LITE
}  // namespace rocksdb
