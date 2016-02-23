package org.rocksdb;

/**
 * The config for hash skip-list mem-table representation.
 * Such mem-table representation contains a fix-sized array of
 * buckets, where each bucket points to a skiplist (or null if the
 * bucket is empty).
 *
 * Note that since this mem-table representation relies on the
 * key prefix, it is required to invoke one of the usePrefixExtractor
 * functions to specify how to extract key prefix given a key.
 * If proper prefix-extractor is not set, then RocksDB will
 * use the default memtable representation (SkipList) instead
 * and post a warning in the LOG.
 */
public class HashSkipListMemTableConfig extends MemTableConfig {
  public static final int DEFAULT_BUCKET_COUNT = 1000000;
  public static final int DEFAULT_BRANCHING_FACTOR = 4;
  public static final int DEFAULT_HEIGHT = 4;

  /**
   * HashSkipListMemTableConfig constructor
   */
  public HashSkipListMemTableConfig() {
    bucketCount_ = DEFAULT_BUCKET_COUNT;
    branchingFactor_ = DEFAULT_BRANCHING_FACTOR;
    height_ = DEFAULT_HEIGHT;
  }

  /**
   * Set the number of hash buckets used in the hash skiplist memtable.
   * Default = 1000000.
   *
   * @param count the number of hash buckets used in the hash
   *    skiplist memtable.
   * @return the reference to the current HashSkipListMemTableConfig.
   */
  public HashSkipListMemTableConfig setBucketCount(
      final long count) {
    bucketCount_ = count;
    return this;
  }

  /**
   * @return the number of hash buckets
   */
  public long bucketCount() {
    return bucketCount_;
  }

  /**
   * Set the height of the skip list.  Default = 4.
   *
   * @param height height to set.
   *
   * @return the reference to the current HashSkipListMemTableConfig.
   */
  public HashSkipListMemTableConfig setHeight(final int height) {
    height_ = height;
    return this;
  }

  /**
   * @return the height of the skip list.
   */
  public int height() {
    return height_;
  }

  /**
   * Set the branching factor used in the hash skip-list memtable.
   * This factor controls the probabilistic size ratio between adjacent
   * links in the skip list.
   *
   * @param bf the probabilistic size ratio between adjacent link
   *     lists in the skip list.
   * @return the reference to the current HashSkipListMemTableConfig.
   */
  public HashSkipListMemTableConfig setBranchingFactor(
      final int bf) {
    branchingFactor_ = bf;
    return this;
  }

  /**
   * @return branching factor, the probabilistic size ratio between
   *     adjacent links in the skip list.
   */
  public int branchingFactor() {
    return branchingFactor_;
  }

  @Override protected long newMemTableFactoryHandle() {
    return newMemTableFactoryHandle(
        bucketCount_, height_, branchingFactor_);
  }

  private native long newMemTableFactoryHandle(
      long bucketCount, int height, int branchingFactor)
      throws IllegalArgumentException;

  private long bucketCount_;
  private int branchingFactor_;
  private int height_;
}
