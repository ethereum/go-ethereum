// Copyright (c) 2013, Facebook, Inc.  All rights reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.
#ifndef ROCKSDB_LITE
#pragma once

namespace rocksdb {

class SSTDumpTool {
 public:
  int Run(int argc, char** argv);
};

}  // namespace rocksdb

#endif  // ROCKSDB_LITE
