// Copyright (c) 2014, Facebook, Inc.  All rights reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.
package org.rocksdb;

/**
 * The config for plain table sst format.
 *
 * <p>PlainTable is a RocksDB's SST file format optimized for low query
 * latency on pure-memory or really low-latency media.</p>
 *
 * <p>It also support prefix hash feature.</p>
 */
public class PlainTableConfig extends TableFormatConfig {
  public static final int VARIABLE_LENGTH = 0;
  public static final int DEFAULT_BLOOM_BITS_PER_KEY = 10;
  public static final double DEFAULT_HASH_TABLE_RATIO = 0.75;
  public static final int DEFAULT_INDEX_SPARSENESS = 16;
  public static final int DEFAULT_HUGE_TLB_SIZE = 0;
  public static final EncodingType DEFAULT_ENCODING_TYPE =
      EncodingType.kPlain;
  public static final boolean DEFAULT_FULL_SCAN_MODE = false;
  public static final boolean DEFAULT_STORE_INDEX_IN_FILE
      = false;

  public PlainTableConfig() {
    keySize_ = VARIABLE_LENGTH;
    bloomBitsPerKey_ = DEFAULT_BLOOM_BITS_PER_KEY;
    hashTableRatio_ = DEFAULT_HASH_TABLE_RATIO;
    indexSparseness_ = DEFAULT_INDEX_SPARSENESS;
    hugePageTlbSize_ = DEFAULT_HUGE_TLB_SIZE;
    encodingType_ = DEFAULT_ENCODING_TYPE;
    fullScanMode_ = DEFAULT_FULL_SCAN_MODE;
    storeIndexInFile_ = DEFAULT_STORE_INDEX_IN_FILE;
  }

  /**
   * <p>Set the length of the user key. If it is set to be
   * VARIABLE_LENGTH, then it indicates the user keys are
   * of variable length.</p>
   *
   * <p>Otherwise,all the keys need to have the same length
   * in byte.</p>
   *
   * <p>DEFAULT: VARIABLE_LENGTH</p>
   *
   * @param keySize the length of the user key.
   * @return the reference to the current config.
   */
  public PlainTableConfig setKeySize(int keySize) {
    keySize_ = keySize;
    return this;
  }

  /**
   * @return the specified size of the user key.  If VARIABLE_LENGTH,
   *     then it indicates variable-length key.
   */
  public int keySize() {
    return keySize_;
  }

  /**
   * Set the number of bits per key used by the internal bloom filter
   * in the plain table sst format.
   *
   * @param bitsPerKey the number of bits per key for bloom filer.
   * @return the reference to the current config.
   */
  public PlainTableConfig setBloomBitsPerKey(int bitsPerKey) {
    bloomBitsPerKey_ = bitsPerKey;
    return this;
  }

  /**
   * @return the number of bits per key used for the bloom filter.
   */
  public int bloomBitsPerKey() {
    return bloomBitsPerKey_;
  }

  /**
   * hashTableRatio is the desired utilization of the hash table used
   * for prefix hashing.  The ideal ratio would be the number of
   * prefixes / the number of hash buckets.  If this value is set to
   * zero, then hash table will not be used.
   *
   * @param ratio the hash table ratio.
   * @return the reference to the current config.
   */
  public PlainTableConfig setHashTableRatio(double ratio) {
    hashTableRatio_ = ratio;
    return this;
  }

  /**
   * @return the hash table ratio.
   */
  public double hashTableRatio() {
    return hashTableRatio_;
  }

  /**
   * Index sparseness determines the index interval for keys inside the
   * same prefix.  This number is equal to the maximum number of linear
   * search required after hash and binary search.  If it's set to 0,
   * then each key will be indexed.
   *
   * @param sparseness the index sparseness.
   * @return the reference to the current config.
   */
  public PlainTableConfig setIndexSparseness(int sparseness) {
    indexSparseness_ = sparseness;
    return this;
  }

  /**
   * @return the index sparseness.
   */
  public long indexSparseness() {
    return indexSparseness_;
  }

  /**
   * <p>huge_page_tlb_size: if &le;0, allocate hash indexes and blooms
   * from malloc otherwise from huge page TLB.</p>
   *
   * <p>The user needs to reserve huge pages for it to be allocated,
   * like: {@code sysctl -w vm.nr_hugepages=20}</p>
   *
   * <p>See linux doc Documentation/vm/hugetlbpage.txt</p>
   *
   * @param hugePageTlbSize huge page tlb size
   * @return the reference to the current config.
   */
  public PlainTableConfig setHugePageTlbSize(int hugePageTlbSize) {
    this.hugePageTlbSize_ = hugePageTlbSize;
    return this;
  }

  /**
   * Returns the value for huge page tlb size
   *
   * @return hugePageTlbSize
   */
  public int hugePageTlbSize() {
    return hugePageTlbSize_;
  }

  /**
   * Sets the encoding type.
   *
   * <p>This setting determines how to encode
   * the keys. See enum {@link EncodingType} for
   * the choices.</p>
   *
   * <p>The value will determine how to encode keys
   * when writing to a new SST file. This value will be stored
   * inside the SST file which will be used when reading from
   * the file, which makes it possible for users to choose
   * different encoding type when reopening a DB. Files with
   * different encoding types can co-exist in the same DB and
   * can be read.</p>
   *
   * @param encodingType {@link org.rocksdb.EncodingType} value.
   * @return the reference to the current config.
   */
  public PlainTableConfig setEncodingType(EncodingType encodingType) {
    this.encodingType_ = encodingType;
    return this;
  }

  /**
   * Returns the active EncodingType
   *
   * @return currently set encoding type
   */
  public EncodingType encodingType() {
    return encodingType_;
  }

  /**
   * Set full scan mode, if true the whole file will be read
   * one record by one without using the index.
   *
   * @param fullScanMode boolean value indicating if full
   *     scan mode shall be enabled.
   * @return the reference to the current config.
   */
  public PlainTableConfig setFullScanMode(boolean fullScanMode) {
    this.fullScanMode_ = fullScanMode;
    return this;
  }

  /**
   * Return if full scan mode is active
   * @return boolean value indicating if the full scan mode is
   *     enabled.
   */
  public boolean fullScanMode() {
    return fullScanMode_;
  }

  /**
   * <p>If set to true: compute plain table index and bloom
   * filter during file building and store it in file.
   * When reading file, index will be mmaped instead
   * of doing recomputation.</p>
   *
   * @param storeIndexInFile value indicating if index shall
   *     be stored in a file
   * @return the reference to the current config.
   */
  public PlainTableConfig setStoreIndexInFile(boolean storeIndexInFile) {
    this.storeIndexInFile_ = storeIndexInFile;
    return this;
  }

  /**
   * Return a boolean value indicating if index shall be stored
   * in a file.
   *
   * @return currently set value for store index in file.
   */
  public boolean storeIndexInFile() {
    return storeIndexInFile_;
  }

  @Override protected long newTableFactoryHandle() {
    return newTableFactoryHandle(keySize_, bloomBitsPerKey_,
        hashTableRatio_, indexSparseness_, hugePageTlbSize_,
        encodingType_.getValue(), fullScanMode_,
        storeIndexInFile_);
  }

  private native long newTableFactoryHandle(
      int keySize, int bloomBitsPerKey,
      double hashTableRatio, int indexSparseness,
      int hugePageTlbSize, byte encodingType,
      boolean fullScanMode, boolean storeIndexInFile);

  private int keySize_;
  private int bloomBitsPerKey_;
  private double hashTableRatio_;
  private int indexSparseness_;
  private int hugePageTlbSize_;
  private EncodingType encodingType_;
  private boolean fullScanMode_;
  private boolean storeIndexInFile_;
}
