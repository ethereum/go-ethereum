// Copyright (c) 2015, Facebook, Inc.  All rights reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.

package org.rocksdb;

/**
 * RocksDB memory environment.
 */
public class RocksMemEnv extends Env {

  /**
   * <p>Creates a new RocksDB environment that stores its data
   * in memory and delegates all non-file-storage tasks to
   * base_env. The caller must delete the result when it is
   * no longer needed.</p>
   *
   * <p>{@code *base_env} must remain live while the result is in use.</p>
   */
  public RocksMemEnv() {
    super();
    nativeHandle_ = createMemEnv();
  }

  @Override
  protected void disposeInternal() {
    disposeInternal(nativeHandle_);
  }

  private static native long createMemEnv();
  private native void disposeInternal(long handle);
}
