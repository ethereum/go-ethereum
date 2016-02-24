// Copyright (c) 2014, Facebook, Inc.  All rights reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.

package org.rocksdb;

/**
 * RocksObject is the base-class of all RocksDB classes that has a pointer to
 * some c++ {@code rocksdb} object.
 *
 * <p>
 * RocksObject has {@code dispose()} function, which releases its associated c++
 * resource.</p>
 * <p>
 * This function can be either called manually, or being called automatically
 * during the regular Java GC process. However, since Java may wrongly assume a
 * RocksObject only contains a long member variable and think it is small in size,
 * Java may give {@code RocksObject} low priority in the GC process. For this, it is
 * suggested to call {@code dispose()} manually. However, it is safe to let
 * {@code RocksObject} go out-of-scope without manually calling {@code dispose()}
 * as {@code dispose()} will be called in the finalizer during the
 * regular GC process.</p>
 */
public abstract class RocksObject {
  protected RocksObject() {
    nativeHandle_ = 0;
    owningHandle_ = true;
  }

  /**
   * Release the c++ object manually pointed by the native handle.
   * <p>
   * Note that {@code dispose()} will also be called during the GC process
   * if it was not called before its {@code RocksObject} went out-of-scope.
   * However, since Java may wrongly wrongly assume those objects are
   * small in that they seems to only hold a long variable. As a result,
   * they might have low priority in the GC process.  To prevent this,
   * it is suggested to call {@code dispose()} manually.
   * </p>
   * <p>
   * Note that once an instance of {@code RocksObject} has been disposed,
   * calling its function will lead undefined behavior.
   * </p>
   */
  public final synchronized void dispose() {
    if (isOwningNativeHandle() && isInitialized()) {
      disposeInternal();
    }
    nativeHandle_ = 0;
    disOwnNativeHandle();
  }

  /**
   * The helper function of {@code dispose()} which all subclasses of
   * {@code RocksObject} must implement to release their associated
   * C++ resource.
   */
  protected abstract void disposeInternal();

  /**
   * Revoke ownership of the native object.
   * <p>
   * This will prevent the object from attempting to delete the underlying
   * native object in its finalizer. This must be used when another object
   * takes over ownership of the native object or both will attempt to delete
   * the underlying object when garbage collected.
   * <p>
   * When {@code disOwnNativeHandle()} is called, {@code dispose()} will simply set
   * {@code nativeHandle_} to 0 without releasing its associated C++ resource.
   * As a result, incorrectly use this function may cause memory leak, and this
   * function call will not affect the return value of {@code isInitialized()}.
   * </p>
   * @see #dispose()
   * @see #isInitialized()
   */
  protected void disOwnNativeHandle() {
    owningHandle_ = false;
  }

  /**
   * Returns true if the current {@code RocksObject} is responsible to release
   * its native handle.
   *
   * @return true if the current {@code RocksObject} is responsible to release
   *     its native handle.
   *
   * @see #disOwnNativeHandle()
   * @see #dispose()
   */
  protected boolean isOwningNativeHandle() {
    return owningHandle_;
  }

  /**
   * Returns true if the associated native handle has been initialized.
   *
   * @return true if the associated native handle has been initialized.
   *
   * @see #dispose()
   */
  protected boolean isInitialized() {
    return (nativeHandle_ != 0);
  }

  /**
   * Simply calls {@code dispose()} and release its c++ resource if it has not
   * yet released.
   */
  @Override protected void finalize() throws Throwable {
    dispose();
    super.finalize();
  }

  /**
   * A long variable holding c++ pointer pointing to some RocksDB C++ object.
   */
  protected long nativeHandle_;

  /**
   * A flag indicating whether the current {@code RocksObject} is responsible to
   * release the c++ object stored in its {@code nativeHandle_}.
   */
  private boolean owningHandle_;
}
