package org.rocksdb;

/**
 * The config for hash linked list memtable representation
 * Such memtable contains a fix-sized array of buckets, where
 * each bucket points to a sorted singly-linked
 * list (or null if the bucket is empty).
 *
 * Note that since this mem-table representation relies on the
 * key prefix, it is required to invoke one of the usePrefixExtractor
 * functions to specify how to extract key prefix given a key.
 * If proper prefix-extractor is not set, then RocksDB will
 * use the default memtable representation (SkipList) instead
 * and post a warning in the LOG.
 */
public class HashLinkedListMemTableConfig extends MemTableConfig {
  public static final long DEFAULT_BUCKET_COUNT = 50000;
  public static final long DEFAULT_HUGE_PAGE_TLB_SIZE = 0;
  public static final int DEFAULT_BUCKET_ENTRIES_LOG_THRES = 4096;
  public static final boolean
      DEFAULT_IF_LOG_BUCKET_DIST_WHEN_FLUSH = true;
  public static final int DEFAUL_THRESHOLD_USE_SKIPLIST = 256;

  /**
   * HashLinkedListMemTableConfig constructor
   */
  public HashLinkedListMemTableConfig() {
    bucketCount_ = DEFAULT_BUCKET_COUNT;
    hugePageTlbSize_ = DEFAULT_HUGE_PAGE_TLB_SIZE;
    bucketEntriesLoggingThreshold_ = DEFAULT_BUCKET_ENTRIES_LOG_THRES;
    ifLogBucketDistWhenFlush_ = DEFAULT_IF_LOG_BUCKET_DIST_WHEN_FLUSH;
    thresholdUseSkiplist_ = DEFAUL_THRESHOLD_USE_SKIPLIST;
  }

  /**
   * Set the number of buckets in the fixed-size array used
   * in the hash linked-list mem-table.
   *
   * @param count the number of hash buckets.
   * @return the reference to the current HashLinkedListMemTableConfig.
   */
  public HashLinkedListMemTableConfig setBucketCount(
      final long count) {
    bucketCount_ = count;
    return this;
  }

  /**
   * Returns the number of buckets that will be used in the memtable
   * created based on this config.
   *
   * @return the number of buckets
   */
  public long bucketCount() {
    return bucketCount_;
  }

  /**
   * <p>Set the size of huge tlb or allocate the hashtable bytes from
   * malloc if {@code size <= 0}.</p>
   *
   * <p>The user needs to reserve huge pages for it to be allocated,
   * like: {@code sysctl -w vm.nr_hugepages=20}</p>
   *
   * <p>See linux documentation/vm/hugetlbpage.txt</p>
   *
   * @param size if set to {@code <= 0} hashtable bytes from malloc
   * @return the reference to the current HashLinkedListMemTableConfig.
   */
  public HashLinkedListMemTableConfig setHugePageTlbSize(
      final long size) {
    hugePageTlbSize_ = size;
    return this;
  }

  /**
   * Returns the size value of hugePageTlbSize.
   *
   * @return the hugePageTlbSize.
   */
  public long hugePageTlbSize() {
    return hugePageTlbSize_;
  }

  /**
   * If number of entries in one bucket exceeds that setting, log
   * about it.
   *
   * @param threshold - number of entries in a single bucket before
   *     logging starts.
   * @return the reference to the current HashLinkedListMemTableConfig.
   */
  public HashLinkedListMemTableConfig
      setBucketEntriesLoggingThreshold(final int threshold) {
    bucketEntriesLoggingThreshold_ = threshold;
    return this;
  }

  /**
   * Returns the maximum number of entries in one bucket before
   * logging starts.
   *
   * @return maximum number of entries in one bucket before logging
   *     starts.
   */
  public int bucketEntriesLoggingThreshold() {
    return bucketEntriesLoggingThreshold_;
  }

  /**
   * If true the distrubition of number of entries will be logged.
   *
   * @param logDistribution - boolean parameter indicating if number
   *     of entry distribution shall be logged.
   * @return the reference to the current HashLinkedListMemTableConfig.
   */
  public HashLinkedListMemTableConfig
      setIfLogBucketDistWhenFlush(final boolean logDistribution) {
    ifLogBucketDistWhenFlush_ = logDistribution;
    return this;
  }

  /**
   * Returns information about logging the distribution of
   *  number of entries on flush.
   *
   * @return if distrubtion of number of entries shall be logged.
   */
  public boolean ifLogBucketDistWhenFlush() {
    return ifLogBucketDistWhenFlush_;
  }

  /**
   * Set maximum number of entries in one bucket. Exceeding this val
   * leads to a switch from LinkedList to SkipList.
   *
   * @param threshold maximum number of entries before SkipList is
   *     used.
   * @return the reference to the current HashLinkedListMemTableConfig.
   */
  public HashLinkedListMemTableConfig
      setThresholdUseSkiplist(final int threshold) {
    thresholdUseSkiplist_ = threshold;
    return this;
  }

  /**
   * Returns entries per bucket threshold before LinkedList is
   * replaced by SkipList usage for that bucket.
   *
   * @return entries per bucket threshold before SkipList is used.
   */
  public int thresholdUseSkiplist() {
    return thresholdUseSkiplist_;
  }

  @Override protected long newMemTableFactoryHandle() {
    return newMemTableFactoryHandle(bucketCount_, hugePageTlbSize_,
        bucketEntriesLoggingThreshold_, ifLogBucketDistWhenFlush_,
        thresholdUseSkiplist_);
  }

  private native long newMemTableFactoryHandle(long bucketCount,
      long hugePageTlbSize, int bucketEntriesLoggingThreshold,
      boolean ifLogBucketDistWhenFlush, int thresholdUseSkiplist)
      throws IllegalArgumentException;

  private long bucketCount_;
  private long hugePageTlbSize_;
  private int bucketEntriesLoggingThreshold_;
  private boolean ifLogBucketDistWhenFlush_;
  private int thresholdUseSkiplist_;
}
