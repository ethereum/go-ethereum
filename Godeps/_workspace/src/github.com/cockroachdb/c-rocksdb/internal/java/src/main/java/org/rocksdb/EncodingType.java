// Copyright (c) 2014, Facebook, Inc.  All rights reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.

package org.rocksdb;

/**
 * EncodingType
 *
 * <p>The value will determine how to encode keys
 * when writing to a new SST file.</p>
 *
 * <p>This value will be stored
 * inside the SST file which will be used when reading from
 * the file, which makes it possible for users to choose
 * different encoding type when reopening a DB. Files with
 * different encoding types can co-exist in the same DB and
 * can be read.</p>
 */
public enum EncodingType {
  /**
   * Always write full keys without any special encoding.
   */
  kPlain((byte) 0),
  /**
   * <p>Find opportunity to write the same prefix once for multiple rows.
   * In some cases, when a key follows a previous key with the same prefix,
   * instead of writing out the full key, it just writes out the size of the
   * shared prefix, as well as other bytes, to save some bytes.</p>
   *
   * <p>When using this option, the user is required to use the same prefix
   * extractor to make sure the same prefix will be extracted from the same key.
   * The Name() value of the prefix extractor will be stored in the file. When
   * reopening the file, the name of the options.prefix_extractor given will be
   * bitwise compared to the prefix extractors stored in the file. An error
   * will be returned if the two don't match.</p>
   */
  kPrefix((byte) 1);

  /**
   * Returns the byte value of the enumerations value
   *
   * @return byte representation
   */
  public byte getValue() {
    return value_;
  }

  private EncodingType(byte value) {
    value_ = value;
  }

  private final byte value_;
}
