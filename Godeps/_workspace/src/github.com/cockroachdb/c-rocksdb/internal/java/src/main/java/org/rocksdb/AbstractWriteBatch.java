// Copyright (c) 2014, Facebook, Inc.  All rights reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.

package org.rocksdb;

public abstract class AbstractWriteBatch extends RocksObject implements WriteBatchInterface {

  @Override
  public int count() {
    assert (isInitialized());
    return count0();
  }

  @Override
  public void put(byte[] key, byte[] value) {
    assert (isInitialized());
    put(key, key.length, value, value.length);
  }

  @Override
  public void put(ColumnFamilyHandle columnFamilyHandle, byte[] key, byte[] value) {
    assert (isInitialized());
    put(key, key.length, value, value.length, columnFamilyHandle.nativeHandle_);
  }

  @Override
  public void merge(byte[] key, byte[] value) {
    assert (isInitialized());
    merge(key, key.length, value, value.length);
  }

  @Override
  public void merge(ColumnFamilyHandle columnFamilyHandle, byte[] key, byte[] value) {
    assert (isInitialized());
    merge(key, key.length, value, value.length, columnFamilyHandle.nativeHandle_);
  }

  @Override
  public void remove(byte[] key) {
    assert (isInitialized());
    remove(key, key.length);
  }

  @Override
  public void remove(ColumnFamilyHandle columnFamilyHandle, byte[] key) {
    assert (isInitialized());
    remove(key, key.length, columnFamilyHandle.nativeHandle_);
  }

  @Override
  public void putLogData(byte[] blob) {
    assert (isInitialized());
    putLogData(blob, blob.length);
  }

  @Override
  public void clear() {
    assert (isInitialized());
    clear0();
  }

  /**
   * Delete the c++ side pointer.
   */
  @Override
  protected void disposeInternal() {
    assert (isInitialized());
    disposeInternal(nativeHandle_);
  }

  abstract void disposeInternal(long handle);

  abstract int count0();

  abstract void put(byte[] key, int keyLen, byte[] value, int valueLen);

  abstract void put(byte[] key, int keyLen, byte[] value, int valueLen, long cfHandle);

  abstract void merge(byte[] key, int keyLen, byte[] value, int valueLen);

  abstract void merge(byte[] key, int keyLen, byte[] value, int valueLen, long cfHandle);

  abstract void remove(byte[] key, int keyLen);

  abstract void remove(byte[] key, int keyLen, long cfHandle);

  abstract void putLogData(byte[] blob, int blobLen);

  abstract void clear0();
}
