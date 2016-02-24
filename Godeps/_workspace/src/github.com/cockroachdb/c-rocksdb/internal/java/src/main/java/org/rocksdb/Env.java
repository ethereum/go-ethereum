// Copyright (c) 2015, Facebook, Inc.  All rights reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.

package org.rocksdb;

/**
 * Base class for all Env implementations in RocksDB.
 */
public abstract class Env extends RocksObject {
  public static final int FLUSH_POOL = 0;
  public static final int COMPACTION_POOL = 1;

  /**
   * <p>Returns the default environment suitable for the current operating
   * system.</p>
   *
   * <p>The result of {@code getDefault()} is a singleton whose ownership
   * belongs to rocksdb c++.  As a result, the returned RocksEnv will not
   * have the ownership of its c++ resource, and calling its dispose()
   * will be no-op.</p>
   *
   * @return the default {@link org.rocksdb.RocksEnv} instance.
   */
  public static Env getDefault() {
    return default_env_;
  }

  /**
   * <p>Sets the number of background worker threads of the flush pool
   * for this environment.</p>
   * <p>Default number: 1</p>
   *
   * @param num the number of threads
   *
   * @return current {@link RocksEnv} instance.
   */
  public Env setBackgroundThreads(final int num) {
    return setBackgroundThreads(num, FLUSH_POOL);
  }

  /**
   * <p>Sets the number of background worker threads of the specified thread
   * pool for this environment.</p>
   *
   * @param num the number of threads
   * @param poolID the id to specified a thread pool.  Should be either
   *     FLUSH_POOL or COMPACTION_POOL.
   *
   * <p>Default number: 1</p>
   * @return current {@link RocksEnv} instance.
   */
  public Env setBackgroundThreads(final int num, final int poolID) {
    setBackgroundThreads(nativeHandle_, num, poolID);
    return this;
  }

  /**
   * <p>Returns the length of the queue associated with the specified
   * thread pool.</p>
   *
   * @param poolID the id to specified a thread pool.  Should be either
   *     FLUSH_POOL or COMPACTION_POOL.
   *
   * @return the thread pool queue length.
   */
  public int getThreadPoolQueueLen(final int poolID) {
    return getThreadPoolQueueLen(nativeHandle_, poolID);
  }


  protected Env() {
    super();
  }

  static {
    default_env_ = new RocksEnv(getDefaultEnvInternal());
  }

  /**
   * <p>The static default Env. The ownership of its native handle
   * belongs to rocksdb c++ and is not able to be released on the Java
   * side.</p>
   */
  static Env default_env_;

  private static native long getDefaultEnvInternal();
  private native void setBackgroundThreads(
      long handle, int num, int priority);
  private native int getThreadPoolQueueLen(long handle, int poolID);
}
