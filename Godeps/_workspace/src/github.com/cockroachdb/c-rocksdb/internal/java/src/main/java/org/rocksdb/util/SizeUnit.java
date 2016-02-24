// Copyright (c) 2014, Facebook, Inc.  All rights reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.

package org.rocksdb.util;

public class SizeUnit {
  public static final long KB = 1024L;
  public static final long MB = KB * KB;
  public static final long GB = KB * MB;
  public static final long TB = KB * GB;
  public static final long PB = KB * TB;

  private SizeUnit() {}
}
