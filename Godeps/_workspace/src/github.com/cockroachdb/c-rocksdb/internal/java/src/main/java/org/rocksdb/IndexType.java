// Copyright (c) 2014, Facebook, Inc.  All rights reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.

package org.rocksdb;

/**
 * IndexType used in conjunction with BlockBasedTable.
 */
public enum IndexType {
  /**
   * A space efficient index block that is optimized for
   * binary-search-based index.
   */
  kBinarySearch((byte) 0),
  /**
   * The hash index, if enabled, will do the hash lookup when
   * {@code Options.prefix_extractor} is provided.
   */
  kHashSearch((byte) 1);

  /**
   * Returns the byte value of the enumerations value
   *
   * @return byte representation
   */
  public byte getValue() {
    return value_;
  }

  private IndexType(byte value) {
    value_ = value;
  }

  private final byte value_;
}
