package org.rocksdb;

/**
 * The config for skip-list memtable representation.
 */
public class SkipListMemTableConfig extends MemTableConfig {

  public static final long DEFAULT_LOOKAHEAD = 0;

  /**
   * SkipListMemTableConfig constructor
   */
  public SkipListMemTableConfig() {
    lookahead_ = DEFAULT_LOOKAHEAD;
  }

  /**
   * Sets lookahead for SkipList
   *
   * @param lookahead If non-zero, each iterator's seek operation
   *     will start the search from the previously visited record
   *     (doing at most 'lookahead' steps). This is an
   *     optimization for the access pattern including many
   *     seeks with consecutive keys.
   * @return the current instance of SkipListMemTableConfig
   */
  public SkipListMemTableConfig setLookahead(final long lookahead) {
    lookahead_ = lookahead;
    return this;
  }

  /**
   * Returns the currently set lookahead value.
   *
   * @return lookahead value
   */
  public long lookahead() {
    return lookahead_;
  }


  @Override protected long newMemTableFactoryHandle() {
    return newMemTableFactoryHandle0(lookahead_);
  }

  private native long newMemTableFactoryHandle0(long lookahead)
      throws IllegalArgumentException;

  private long lookahead_;
}
