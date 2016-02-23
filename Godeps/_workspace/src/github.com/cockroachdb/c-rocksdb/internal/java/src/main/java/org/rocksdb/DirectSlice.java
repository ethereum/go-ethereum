// Copyright (c) 2014, Facebook, Inc.  All rights reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.

package org.rocksdb;

import java.nio.ByteBuffer;

/**
 * Base class for slices which will receive direct
 * ByteBuffer based access to the underlying data.
 *
 * ByteBuffer backed slices typically perform better with
 * larger keys and values. When using smaller keys and
 * values consider using @see org.rocksdb.Slice
 */
public class DirectSlice extends AbstractSlice<ByteBuffer> {
  //TODO(AR) only needed by WriteBatchWithIndexTest until JDK8
  public final static DirectSlice NONE = new DirectSlice();

  /**
   * Called from JNI to construct a new Java DirectSlice
   * without an underlying C++ object set
   * at creation time.
   *
   * Note: You should be aware that
   * {@see org.rocksdb.RocksObject#disOwnNativeHandle()} is intentionally
   * called from the default DirectSlice constructor, and that it is marked as
   * package-private. This is so that developers cannot construct their own default
   * DirectSlice objects (at present). As developers cannot construct their own
   * DirectSlice objects through this, they are not creating underlying C++
   * DirectSlice objects, and so there is nothing to free (dispose) from Java.
   */
  DirectSlice() {
    super();
    disOwnNativeHandle();
  }

  /**
   * Constructs a slice
   * where the data is taken from
   * a String.
   *
   * @param str The string
   */
  public DirectSlice(final String str) {
    super();
    createNewSliceFromString(str);
  }

  /**
   * Constructs a slice where the data is
   * read from the provided
   * ByteBuffer up to a certain length
   *
   * @param data The buffer containing the data
   * @param length The length of the data to use for the slice
   */
  public DirectSlice(final ByteBuffer data, final int length) {
    super();
    assert(data.isDirect());
    createNewDirectSlice0(data, length);
  }

  /**
   * Constructs a slice where the data is
   * read from the provided
   * ByteBuffer
   *
   * @param data The bugger containing the data
   */
  public DirectSlice(final ByteBuffer data) {
    super();
    assert(data.isDirect());
    createNewDirectSlice1(data);
  }

  /**
   * Retrieves the byte at a specific offset
   * from the underlying data
   *
   * @param offset The (zero-based) offset of the byte to retrieve
   *
   * @return the requested byte
   */
  public byte get(int offset) {
    assert (isInitialized());
    return get0(nativeHandle_, offset);
  }

  /**
   * Clears the backing slice
   */
  public void clear() {
    assert (isInitialized());
    clear0(nativeHandle_);
  }

  /**
   * Drops the specified {@code n}
   * number of bytes from the start
   * of the backing slice
   *
   * @param n The number of bytes to drop
   */
  public void removePrefix(final int n) {
    assert (isInitialized());
    removePrefix0(nativeHandle_, n);
  }

  private native void createNewDirectSlice0(ByteBuffer data, int length);
  private native void createNewDirectSlice1(ByteBuffer data);
  @Override protected final native ByteBuffer data0(long handle);
  private native byte get0(long handle, int offset);
  private native void clear0(long handle);
  private native void removePrefix0(long handle, int length);
}
