// Copyright (c) 2014, Facebook, Inc.  All rights reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.

package org.rocksdb;

/**
 * ColumnFamilyHandle class to hold handles to underlying rocksdb
 * ColumnFamily Pointers.
 */
public class ColumnFamilyHandle extends RocksObject {
  ColumnFamilyHandle(final RocksDB rocksDB,
      final long nativeHandle) {
    super();
    nativeHandle_ = nativeHandle;
    // rocksDB must point to a valid RocksDB instance;
    assert(rocksDB != null);
    // ColumnFamilyHandle must hold a reference to the related RocksDB instance
    // to guarantee that while a GC cycle starts ColumnFamilyHandle instances
    // are freed prior to RocksDB instances.
    rocksDB_ = rocksDB;
  }

  /**
   * <p>Deletes underlying C++ iterator pointer.</p>
   *
   * <p>Note: the underlying handle can only be safely deleted if the RocksDB
   * instance related to a certain ColumnFamilyHandle is still valid and initialized.
   * Therefore {@code disposeInternal()} checks if the RocksDB is initialized
   * before freeing the native handle.</p>
   */
  @Override protected void disposeInternal() {
    synchronized (rocksDB_) {
      assert (isInitialized());
      if (rocksDB_.isInitialized()) {
        disposeInternal(nativeHandle_);
      }
    }
  }

  private native void disposeInternal(long handle);

  private final RocksDB rocksDB_;
}
