package org.rocksdb;

/**
 * FlushOptions to be passed to flush operations of
 * {@link org.rocksdb.RocksDB}.
 */
public class FlushOptions extends RocksObject {

  /**
   * Construct a new instance of FlushOptions.
   */
  public FlushOptions(){
    super();
    newFlushOptions();
  }

  /**
   * Set if the flush operation shall block until it terminates.
   *
   * @param waitForFlush boolean value indicating if the flush
   *     operations waits for termination of the flush process.
   *
   * @return instance of current FlushOptions.
   */
  public FlushOptions setWaitForFlush(final boolean waitForFlush) {
    assert(isInitialized());
    setWaitForFlush(nativeHandle_, waitForFlush);
    return this;
  }

  /**
   * Wait for flush to finished.
   *
   * @return boolean value indicating if the flush operation
   *     waits for termination of the flush process.
   */
  public boolean waitForFlush() {
    assert(isInitialized());
    return waitForFlush(nativeHandle_);
  }

  @Override protected void disposeInternal() {
    disposeInternal(nativeHandle_);
  }

  private native void newFlushOptions();
  private native void disposeInternal(long handle);
  private native void setWaitForFlush(long handle,
      boolean wait);
  private native boolean waitForFlush(long handle);
}
