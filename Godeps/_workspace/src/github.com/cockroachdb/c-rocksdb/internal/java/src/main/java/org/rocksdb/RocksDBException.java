// Copyright (c) 2013, Facebook, Inc.  All rights reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.

package org.rocksdb;

/**
 * A RocksDBException encapsulates the error of an operation.  This exception
 * type is used to describe an internal error from the c++ rocksdb library.
 */
public class RocksDBException extends Exception {
  /**
   * The private construct used by a set of public static factory method.
   *
   * @param msg the specified error message.
   */
  public RocksDBException(final String msg) {
    super(msg);
  }
}
