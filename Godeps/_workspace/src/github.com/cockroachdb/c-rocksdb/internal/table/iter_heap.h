//  Copyright (c) 2013, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.
//

#pragma once

#include "rocksdb/comparator.h"
#include "table/iterator_wrapper.h"

namespace rocksdb {

// When used with std::priority_queue, this comparison functor puts the
// iterator with the max/largest key on top.
class MaxIteratorComparator {
 public:
  MaxIteratorComparator(const Comparator* comparator) :
    comparator_(comparator) {}

  bool operator()(IteratorWrapper* a, IteratorWrapper* b) const {
    return comparator_->Compare(a->key(), b->key()) < 0;
  }
 private:
  const Comparator* comparator_;
};

// When used with std::priority_queue, this comparison functor puts the
// iterator with the min/smallest key on top.
class MinIteratorComparator {
 public:
  MinIteratorComparator(const Comparator* comparator) :
    comparator_(comparator) {}

  bool operator()(IteratorWrapper* a, IteratorWrapper* b) const {
    return comparator_->Compare(a->key(), b->key()) > 0;
  }
 private:
  const Comparator* comparator_;
};

}  // namespace rocksdb
