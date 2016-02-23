// Copyright (c) 2014, Facebook, Inc.  All rights reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.

package org.rocksdb;

/**
 * <p>A RocksEnv is an interface used by the rocksdb implementation to access
 * operating system functionality like the filesystem etc.</p>
 *
 * <p>All Env implementations are safe for concurrent access from
 * multiple threads without any external synchronization.</p>
 */
public class RocksEnv extends Env {

  /**
   * <p>Package-private constructor that uses the specified native handle
   * to construct a RocksEnv.</p>
   *
   * <p>Note that the ownership of the input handle
   * belongs to the caller, and the newly created RocksEnv will not take
   * the ownership of the input handle.  As a result, calling
   * {@code dispose()} of the created RocksEnv will be no-op.</p>
   */
  RocksEnv(final long handle) {
    super();
    nativeHandle_ = handle;
    disOwnNativeHandle();
  }

  /**
   * <p>The helper function of {@link #dispose()} which all subclasses of
   * {@link RocksObject} must implement to release their associated C++
   * resource.</p>
   *
   * <p><strong>Note:</strong> this class is used to use the default
   * RocksEnv with RocksJava. The default env allocation is managed
   * by C++.</p>
   */
  @Override protected void disposeInternal() {
  }
}
