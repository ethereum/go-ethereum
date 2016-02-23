package org.rocksdb;

/**
 * The config for vector memtable representation.
 */
public class VectorMemTableConfig extends MemTableConfig {
  public static final int DEFAULT_RESERVED_SIZE = 0;

  /**
   * VectorMemTableConfig constructor
   */
  public VectorMemTableConfig() {
    reservedSize_ = DEFAULT_RESERVED_SIZE;
  }

  /**
   * Set the initial size of the vector that will be used
   * by the memtable created based on this config.
   *
   * @param size the initial size of the vector.
   * @return the reference to the current config.
   */
  public VectorMemTableConfig setReservedSize(final int size) {
    reservedSize_ = size;
    return this;
  }

  /**
   * Returns the initial size of the vector used by the memtable
   * created based on this config.
   *
   * @return the initial size of the vector.
   */
  public int reservedSize() {
    return reservedSize_;
  }

  @Override protected long newMemTableFactoryHandle() {
    return newMemTableFactoryHandle(reservedSize_);
  }

  private native long newMemTableFactoryHandle(long reservedSize)
      throws IllegalArgumentException;
  private int reservedSize_;
}
