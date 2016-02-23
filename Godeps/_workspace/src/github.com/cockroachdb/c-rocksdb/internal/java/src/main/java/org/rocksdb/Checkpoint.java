// Copyright (c) 2014, Facebook, Inc.  All rights reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.

package org.rocksdb;

/**
 * Provides Checkpoint functionality. Checkpoints
 * provide persistent snapshots of RocksDB databases.
 */
public class Checkpoint extends RocksObject {

  /**
   * Creates a Checkpoint object to be used for creating open-able
   * snapshots.
   *
   * @param db {@link RocksDB} instance.
   * @return a Checkpoint instance.
   *
   * @throws java.lang.IllegalArgumentException if {@link RocksDB}
   *     instance is null.
   * @throws java.lang.IllegalStateException if {@link RocksDB}
   *     instance is not initialized.
   */
  public static Checkpoint create(final RocksDB db) {
    if (db == null) {
      throw new IllegalArgumentException(
          "RocksDB instance shall not be null.");
    } else if (!db.isInitialized()) {
      throw new IllegalStateException(
          "RocksDB instance must be initialized.");
    }
    Checkpoint checkpoint = new Checkpoint(db);
    return checkpoint;
  }

  /**
   * <p>Builds an open-able snapshot of RocksDB on the same disk, which
   * accepts an output directory on the same disk, and under the directory
   * (1) hard-linked SST files pointing to existing live SST files
   * (2) a copied manifest files and other files</p>
   *
   * @param checkpointPath path to the folder where the snapshot is going
   *     to be stored.
   * @throws RocksDBException thrown if an error occurs within the native
   *     part of the library.
   */
  public void createCheckpoint(final String checkpointPath)
      throws RocksDBException {
    createCheckpoint(nativeHandle_, checkpointPath);
  }

  @Override
  protected void disposeInternal() {
    disposeInternal(nativeHandle_);
  }

  private Checkpoint(RocksDB db) {
    super();
    nativeHandle_ = newCheckpoint(db.nativeHandle_);
    db_ = db;
  }

  private RocksDB db_;

  private static native long newCheckpoint(long dbHandle);
  private native void disposeInternal(long handle);

  private native void createCheckpoint(long handle, String checkpointPath)
      throws RocksDBException;
}
