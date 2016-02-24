//  Copyright (c) 2013, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.
//
#pragma once
namespace rocksdb {
namespace port {

// Install a signal handler to print callstack on the following signals:
// SIGILL SIGSEGV SIGBUS SIGABRT
// Currently supports linux only. No-op otherwise.
void InstallStackTraceHandler();

// Prints stack, skips skip_first_frames frames
void PrintStack(int first_frames_to_skip = 0);

}  // namespace port
}  // namespace rocksdb
