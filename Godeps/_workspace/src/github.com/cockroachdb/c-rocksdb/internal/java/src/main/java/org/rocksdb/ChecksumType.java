// Copyright (c) 2014, Facebook, Inc.  All rights reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.

package org.rocksdb;

/**
 * Checksum types used in conjunction with BlockBasedTable.
 */
public enum ChecksumType {
  /**
   * Not implemented yet.
   */
  kNoChecksum((byte) 0),
  /**
   * CRC32 Checksum
   */
  kCRC32c((byte) 1),
  /**
   * XX Hash
   */
  kxxHash((byte) 2);

  /**
   * Returns the byte value of the enumerations value
   *
   * @return byte representation
   */
  public byte getValue() {
    return value_;
  }

  private ChecksumType(byte value) {
    value_ = value;
  }

  private final byte value_;
}
