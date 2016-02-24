// Copyright (c) 2014, Facebook, Inc.  All rights reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.

#pragma once

#include <string>
#include "rocksdb/env.h"

namespace rocksdb {

// This API is experimental. We will mark it stable once we run it in production
// for a while.
// NewFlashcacheAwareEnv() creates and Env that blacklists all background
// threads (used for flush and compaction) from using flashcache to cache their
// reads. Reads from compaction thread don't need to be cached because they are
// going to be soon made obsolete (due to nature of compaction)
// Usually you would pass Env::Default() as base.
// cachedev_fd is a file descriptor of the flashcache device. Caller has to
// open flashcache device before calling this API.
extern std::unique_ptr<Env> NewFlashcacheAwareEnv(
    Env* base, const int cachedev_fd);

}  // namespace rocksdb
