// Copyright (c) 2014, Facebook, Inc.  All rights reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.

package org.rocksdb;

/**
 * Comparators are used by RocksDB to determine
 * the ordering of keys.
 *
 * This class is package private, implementers
 * should extend either of the public abstract classes:
 *   @see org.rocksdb.Comparator
 *   @see org.rocksdb.DirectComparator
 */
public abstract class AbstractComparator<T extends AbstractSlice<?>>
    extends RocksObject {

  /**
   * The name of the comparator.  Used to check for comparator
   * mismatches (i.e., a DB created with one comparator is
   * accessed using a different comparator).
   *
   * A new name should be used whenever
   * the comparator implementation changes in a way that will cause
   * the relative ordering of any two keys to change.
   *
   * Names starting with "rocksdb." are reserved and should not be used.
   *
   * @return The name of this comparator implementation
   */
  public abstract String name();

  /**
   * Three-way key comparison
   *
   *  @param a Slice access to first key
   *  @param b Slice access to second key
   *
   *  @return Should return either:
   *    1) &lt; 0 if "a" &lt; "b"
   *    2) == 0 if "a" == "b"
   *    3) &gt; 0 if "a" &gt; "b"
   */
  public abstract int compare(final T a, final T b);

  /**
   * <p>Used to reduce the space requirements
   * for internal data structures like index blocks.</p>
   *
   * <p>If start &lt; limit, you may return a new start which is a
   * shorter string in [start, limit).</p>
   *
   * <p>Simple comparator implementations may return null if they
   * wish to use start unchanged. i.e., an implementation of
   * this method that does nothing is correct.</p>
   *
   * @param start String
   * @param limit of type T
   *
   * @return a shorter start, or null
   */
  public String findShortestSeparator(final String start, final T limit) {
      return null;
  }

  /**
   * <p>Used to reduce the space requirements
   * for internal data structures like index blocks.</p>
   *
   * <p>You may return a new short key (key1) where
   * key1 &ge; key.</p>
   *
   * <p>Simple comparator implementations may return null if they
   * wish to leave the key unchanged. i.e., an implementation of
   * this method that does nothing is correct.</p>
   *
   * @param key String
   *
   * @return a shorter key, or null
   */
  public String findShortSuccessor(final String key) {
      return null;
  }

  /**
   * Deletes underlying C++ comparator pointer.
   *
   * Note that this function should be called only after all
   * RocksDB instances referencing the comparator are closed.
   * Otherwise an undefined behavior will occur.
   */
  @Override protected void disposeInternal() {
    assert(isInitialized());
    disposeInternal(nativeHandle_);
  }

  private native void disposeInternal(long handle);
}
