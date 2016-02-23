// Copyright (c) 2014, Facebook, Inc.  All rights reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.

#pragma once

#include <string>
#include "rocksdb/env.h"

namespace rocksdb {

// This is internal API that will make hacking on flashcache easier. Not sure if
// we need to expose this to public users, probably not
extern int FlashcacheBlacklistCurrentThread(Env* flashcache_aware_env);
extern int FlashcacheWhitelistCurrentThread(Env* flashcache_aware_env);

}  // namespace rocksdb
