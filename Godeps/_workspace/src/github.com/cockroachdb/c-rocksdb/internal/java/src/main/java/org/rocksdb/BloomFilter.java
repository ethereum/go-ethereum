// Copyright (c) 2014, Facebook, Inc.  All rights reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.

package org.rocksdb;

/**
 * Bloom filter policy that uses a bloom filter with approximately
 * the specified number of bits per key.
 *
 * <p>
 * Note: if you are using a custom comparator that ignores some parts
 * of the keys being compared, you must not use this {@code BloomFilter}
 * and must provide your own FilterPolicy that also ignores the
 * corresponding parts of the keys. For example, if the comparator
 * ignores trailing spaces, it would be incorrect to use a
 * FilterPolicy (like {@code BloomFilter}) that does not ignore
 * trailing spaces in keys.</p>
 */
public class BloomFilter extends Filter {

  private static final int DEFAULT_BITS_PER_KEY = 10;
  private static final boolean DEFAULT_MODE = true;
  private final int bitsPerKey_;
  private final boolean useBlockBasedMode_;

  /**
   * BloomFilter constructor
   *
   * <p>
   * Callers must delete the result after any database that is using the
   * result has been closed.</p>
   */
  public BloomFilter() {
    this(DEFAULT_BITS_PER_KEY, DEFAULT_MODE);
  }

  /**
   * BloomFilter constructor
   *
   * <p>
   * bits_per_key: bits per key in bloom filter. A good value for bits_per_key
   * is 10, which yields a filter with ~ 1% false positive rate.
   * </p>
   * <p>
   * Callers must delete the result after any database that is using the
   * result has been closed.</p>
   *
   * @param bitsPerKey number of bits to use
   */
  public BloomFilter(final int bitsPerKey) {
    this(bitsPerKey, DEFAULT_MODE);
  }

  /**
   * BloomFilter constructor
   *
   * <p>
   * bits_per_key: bits per key in bloom filter. A good value for bits_per_key
   * is 10, which yields a filter with ~ 1% false positive rate.
   * <p><strong>default bits_per_key</strong>: 10</p>
   *
   * <p>use_block_based_builder: use block based filter rather than full filter.
   * If you want to builder full filter, it needs to be set to false.
   * </p>
   * <p><strong>default mode: block based filter</strong></p>
   * <p>
   * Callers must delete the result after any database that is using the
   * result has been closed.</p>
   *
   * @param bitsPerKey number of bits to use
   * @param useBlockBasedMode use block based mode or full filter mode
   */
  public BloomFilter(final int bitsPerKey, final boolean useBlockBasedMode) {
    super();
    bitsPerKey_ = bitsPerKey;
    useBlockBasedMode_ = useBlockBasedMode;
    createNewFilter();
  }

  @Override
  protected final void createNewFilter() {
    createNewBloomFilter(bitsPerKey_, useBlockBasedMode_);
  }

  private native void createNewBloomFilter(int bitsKeyKey,
      boolean useBlockBasedMode);
}
