// Copyright (c) 2014, Facebook, Inc.  All rights reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.

package org.rocksdb;

import org.junit.Test;


public class CompressionOptionsTest
{
  @Test
  public void getCompressionType() {
    for (CompressionType compressionType : CompressionType.values()) {
      String libraryName = compressionType.getLibraryName();
      compressionType.equals(CompressionType.getCompressionType(
          libraryName));
    }
  }
}
