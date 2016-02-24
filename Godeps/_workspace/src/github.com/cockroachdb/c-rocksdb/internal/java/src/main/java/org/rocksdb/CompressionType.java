// Copyright (c) 2014, Facebook, Inc.  All rights reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.

package org.rocksdb;

/**
 * Enum CompressionType
 *
 * <p>DB contents are stored in a set of blocks, each of which holds a
 * sequence of key,value pairs. Each block may be compressed before
 * being stored in a file. The following enum describes which
 * compression method (if any) is used to compress a block.</p>
 */
public enum CompressionType {

  NO_COMPRESSION((byte) 0, null),
  SNAPPY_COMPRESSION((byte) 1, "snappy"),
  ZLIB_COMPRESSION((byte) 2, "z"),
  BZLIB2_COMPRESSION((byte) 3, "bzip2"),
  LZ4_COMPRESSION((byte) 4, "lz4"),
  LZ4HC_COMPRESSION((byte) 5, "lz4hc");

  /**
   * <p>Get the CompressionType enumeration value by
   * passing the library name to this method.</p>
   *
   * <p>If library cannot be found the enumeration
   * value {@code NO_COMPRESSION} will be returned.</p>
   *
   * @param libraryName compression library name.
   *
   * @return CompressionType instance.
   */
  public static CompressionType getCompressionType(String libraryName) {
    if (libraryName != null) {
      for (CompressionType compressionType : CompressionType.values()) {
        if (compressionType.getLibraryName() != null &&
            compressionType.getLibraryName().equals(libraryName)) {
          return compressionType;
        }
      }
    }
    return CompressionType.NO_COMPRESSION;
  }

  /**
   * <p>Get the CompressionType enumeration value by
   * passing the byte identifier to this method.</p>
   *
   * <p>If library cannot be found the enumeration
   * value {@code NO_COMPRESSION} will be returned.</p>
   *
   * @param byteIdentifier of CompressionType.
   *
   * @return CompressionType instance.
   */
  public static CompressionType getCompressionType(byte byteIdentifier) {
    for (CompressionType compressionType : CompressionType.values()) {
      if (compressionType.getValue() == byteIdentifier) {
        return compressionType;
      }
    }
    return CompressionType.NO_COMPRESSION;
  }

  /**
   * <p>Returns the byte value of the enumerations value.</p>
   *
   * @return byte representation
   */
  public byte getValue() {
    return value_;
  }

  /**
   * <p>Returns the library name of the compression type
   * identified by the enumeration value.</p>
   *
   * @return library name
   */
  public String getLibraryName() {
    return libraryName_;
  }

  private CompressionType(byte value, final String libraryName) {
        value_ = value;
        libraryName_ = libraryName;
  }

  private final byte value_;
  private final String libraryName_;
}
