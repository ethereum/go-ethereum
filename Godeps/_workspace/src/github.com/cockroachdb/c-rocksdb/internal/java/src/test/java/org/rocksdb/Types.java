// Copyright (c) 2014, Facebook, Inc.  All rights reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.

package org.rocksdb;

/**
 * Simple type conversion methods
 * for use in tests
 */
public class Types {

  /**
   * Convert first 4 bytes of a byte array to an int
   *
   * @param data The byte array
   *
   * @return An integer
   */
  public static int byteToInt(final byte data[]) {
    return (data[0] & 0xff) |
        ((data[1] & 0xff) << 8) |
        ((data[2] & 0xff) << 16) |
        ((data[3] & 0xff) << 24);
  }

  /**
   * Convert an int to 4 bytes
   *
   * @param v The int
   *
   * @return A byte array containing 4 bytes
   */
  public static byte[] intToByte(final int v) {
    return new byte[] {
        (byte)((v >>> 0) & 0xff),
        (byte)((v >>> 8) & 0xff),
        (byte)((v >>> 16) & 0xff),
        (byte)((v >>> 24) & 0xff)
    };
  }
}
