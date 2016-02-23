// Copyright (c) 2014, Facebook, Inc.  All rights reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.

package org.rocksdb;

/**
 * Base class implementation for Rocks Iterators
 * in the Java API
 *
 * <p>Multiple threads can invoke const methods on an RocksIterator without
 * external synchronization, but if any of the threads may call a
 * non-const method, all threads accessing the same RocksIterator must use
 * external synchronization.</p>
 *
 * @param <P> The type of the Parent Object from which the Rocks Iterator was
 *          created. This is used by disposeInternal to avoid double-free
 *          issues with the underlying C++ object.
 * @see org.rocksdb.RocksObject
 */
public abstract class AbstractRocksIterator<P extends RocksObject>
    extends RocksObject implements RocksIteratorInterface {
  final P parent_;

  protected AbstractRocksIterator(final P parent,
      final long nativeHandle) {
    super();
    nativeHandle_ = nativeHandle;
    // parent must point to a valid RocksDB instance.
    assert (parent != null);
    // RocksIterator must hold a reference to the related parent instance
    // to guarantee that while a GC cycle starts RocksIterator instances
    // are freed prior to parent instances.
    parent_ = parent;
  }

  @Override
  public boolean isValid() {
    assert (isInitialized());
    return isValid0(nativeHandle_);
  }

  @Override
  public void seekToFirst() {
    assert (isInitialized());
    seekToFirst0(nativeHandle_);
  }

  @Override
  public void seekToLast() {
    assert (isInitialized());
    seekToLast0(nativeHandle_);
  }

  @Override
  public void seek(byte[] target) {
    assert (isInitialized());
    seek0(nativeHandle_, target, target.length);
  }

  @Override
  public void next() {
    assert (isInitialized());
    next0(nativeHandle_);
  }

  @Override
  public void prev() {
    assert (isInitialized());
    prev0(nativeHandle_);
  }

  @Override
  public void status() throws RocksDBException {
    assert (isInitialized());
    status0(nativeHandle_);
  }

  /**
   * <p>Deletes underlying C++ iterator pointer.</p>
   *
   * <p>Note: the underlying handle can only be safely deleted if the parent
   * instance related to a certain RocksIterator is still valid and initialized.
   * Therefore {@code disposeInternal()} checks if the parent is initialized
   * before freeing the native handle.</p>
   */
  @Override
  protected void disposeInternal() {
    synchronized (parent_) {
      assert (isInitialized());
      if (parent_.isInitialized()) {
        disposeInternal(nativeHandle_);
      }
    }
  }

  abstract void disposeInternal(long handle);
  abstract boolean isValid0(long handle);
  abstract void seekToFirst0(long handle);
  abstract void seekToLast0(long handle);
  abstract void next0(long handle);
  abstract void prev0(long handle);
  abstract void seek0(long handle, byte[] target, int targetLen);
  abstract void status0(long handle) throws RocksDBException;
}
