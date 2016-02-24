// Copyright (c) 2013, Facebook, Inc.  All rights reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.
// Copyright (c) 2011 The LevelDB Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file. See the AUTHORS file for names of contributors.
#pragma once

#include "rocksdb/iterator.h"

namespace rocksdb {
class ScopedArenaIterator {
 public:
  explicit ScopedArenaIterator(Iterator* iter = nullptr) : iter_(iter) {}

  Iterator* operator->() { return iter_; }

  void set(Iterator* iter) { iter_ = iter; }

  Iterator* get() { return iter_; }

  ~ScopedArenaIterator() { iter_->~Iterator(); }

 private:
  Iterator* iter_;
};
}  // namespace rocksdb
