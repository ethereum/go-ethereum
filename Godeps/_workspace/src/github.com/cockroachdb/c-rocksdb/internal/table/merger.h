//  Copyright (c) 2013, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.
//
// Copyright (c) 2011 The LevelDB Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file. See the AUTHORS file for names of contributors.

#pragma once

#include "rocksdb/types.h"

namespace rocksdb {

class Comparator;
class Iterator;
class Env;
class Arena;

// Return an iterator that provided the union of the data in
// children[0,n-1].  Takes ownership of the child iterators and
// will delete them when the result iterator is deleted.
//
// The result does no duplicate suppression.  I.e., if a particular
// key is present in K child iterators, it will be yielded K times.
//
// REQUIRES: n >= 0
extern Iterator* NewMergingIterator(const Comparator* comparator,
                                    Iterator** children, int n,
                                    Arena* arena = nullptr);

class MergingIterator;

// A builder class to build a merging iterator by adding iterators one by one.
class MergeIteratorBuilder {
 public:
  // comparator: the comparator used in merging comparator
  // arena: where the merging iterator needs to be allocated from.
  explicit MergeIteratorBuilder(const Comparator* comparator, Arena* arena);
  ~MergeIteratorBuilder() {}

  // Add iter to the merging iterator.
  void AddIterator(Iterator* iter);

  // Get arena used to build the merging iterator. It is called one a child
  // iterator needs to be allocated.
  Arena* GetArena() { return arena; }

  // Return the result merging iterator.
  Iterator* Finish();

 private:
  MergingIterator* merge_iter;
  Iterator* first_iter;
  bool use_merging_iter;
  Arena* arena;
};

}  // namespace rocksdb
