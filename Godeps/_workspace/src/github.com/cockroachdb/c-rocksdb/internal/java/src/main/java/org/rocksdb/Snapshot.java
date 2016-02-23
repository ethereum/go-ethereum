// Copyright (c) 2014, Facebook, Inc.  All rights reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.

package org.rocksdb;

/**
 * Snapshot of database
 */
public class Snapshot extends RocksObject {
  Snapshot(final long nativeHandle) {
    super();
    nativeHandle_ = nativeHandle;
  }

  /**
   * Return the associated sequence number;
   *
   * @return the associated sequence number of
   *     this snapshot.
   */
  public long getSequenceNumber() {
    assert(isInitialized());
    return getSequenceNumber(nativeHandle_);
  }

  /**
   * Dont release C++ Snapshot pointer. The pointer
   * to the snapshot is released by the database
   * instance.
   */
  @Override protected void disposeInternal() {
  }

  private native long getSequenceNumber(long handle);
}
