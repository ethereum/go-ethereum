package org.rocksdb;

/**
 * <p>A TransactionLogIterator is used to iterate over the transactions in a db.
 * One run of the iterator is continuous, i.e. the iterator will stop at the
 * beginning of any gap in sequences.</p>
 */
public class TransactionLogIterator extends RocksObject {

  /**
   * <p>An iterator is either positioned at a WriteBatch
   * or not valid. This method returns true if the iterator
   * is valid. Can read data from a valid iterator.</p>
   *
   * @return true if iterator position is valid.
   */
  public boolean isValid() {
    return isValid(nativeHandle_);
  }

  /**
   * <p>Moves the iterator to the next WriteBatch.
   * <strong>REQUIRES</strong>: Valid() to be true.</p>
   */
  public void next() {
    next(nativeHandle_);
  }

  /**
   * <p>Throws RocksDBException if something went wrong.</p>
   *
   * @throws org.rocksdb.RocksDBException if something went
   *     wrong in the underlying C++ code.
   */
  public void status() throws RocksDBException {
    status(nativeHandle_);
  }

  /**
   * <p>If iterator position is valid, return the current
   * write_batch and the sequence number of the earliest
   * transaction contained in the batch.</p>
   *
   * <p>ONLY use if Valid() is true and status() is OK.</p>
   *
   * @return {@link org.rocksdb.TransactionLogIterator.BatchResult}
   *     instance.
   */
  public BatchResult getBatch() {
    assert(isValid());
    return getBatch(nativeHandle_);
  }

  /**
   * <p>TransactionLogIterator constructor.</p>
   *
   * @param nativeHandle address to native address.
   */
  TransactionLogIterator(final long nativeHandle) {
    super();
    nativeHandle_ = nativeHandle;
  }

  @Override protected void disposeInternal() {
    disposeInternal(nativeHandle_);
  }

  /**
   * <p>BatchResult represents a data structure returned
   * by a TransactionLogIterator containing a sequence
   * number and a {@link WriteBatch} instance.</p>
   */
  public final class BatchResult {
    /**
     * <p>Constructor of BatchResult class.</p>
     *
     * @param sequenceNumber related to this BatchResult instance.
     * @param nativeHandle to {@link org.rocksdb.WriteBatch}
     *     native instance.
     */
    public BatchResult(final long sequenceNumber,
        final long nativeHandle) {
      sequenceNumber_ = sequenceNumber;
      writeBatch_ = new WriteBatch(nativeHandle);
    }

    /**
     * <p>Return sequence number related to this BatchResult.</p>
     *
     * @return Sequence number.
     */
    public long sequenceNumber() {
      return sequenceNumber_;
    }

    /**
     * <p>Return contained {@link org.rocksdb.WriteBatch}
     * instance</p>
     *
     * @return {@link org.rocksdb.WriteBatch} instance.
     */
    public WriteBatch writeBatch() {
      return writeBatch_;
    }

    private final long sequenceNumber_;
    private final WriteBatch writeBatch_;
  }

  private native void disposeInternal(long handle);
  private native boolean isValid(long handle);
  private native void next(long handle);
  private native void status(long handle)
      throws RocksDBException;
  private native BatchResult getBatch(long handle);
}
