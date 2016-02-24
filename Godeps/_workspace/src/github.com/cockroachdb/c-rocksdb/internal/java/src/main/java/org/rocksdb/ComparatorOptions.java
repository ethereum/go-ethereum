package org.rocksdb;

/**
 * This class controls the behaviour
 * of Java implementations of
 * AbstractComparator
 *
 * Note that dispose() must be called before a ComparatorOptions
 * instance becomes out-of-scope to release the allocated memory in C++.
 */
public class ComparatorOptions extends RocksObject {
  public ComparatorOptions() {
    super();
    newComparatorOptions();
  }

  /**
   * Use adaptive mutex, which spins in the user space before resorting
   * to kernel. This could reduce context switch when the mutex is not
   * heavily contended. However, if the mutex is hot, we could end up
   * wasting spin time.
   * Default: false
   *
   * @return true if adaptive mutex is used.
   */
  public boolean useAdaptiveMutex() {
    assert(isInitialized());
    return useAdaptiveMutex(nativeHandle_);
  }

  /**
   * Use adaptive mutex, which spins in the user space before resorting
   * to kernel. This could reduce context switch when the mutex is not
   * heavily contended. However, if the mutex is hot, we could end up
   * wasting spin time.
   * Default: false
   *
   * @param useAdaptiveMutex true if adaptive mutex is used.
   * @return the reference to the current comparator options.
   */
  public ComparatorOptions setUseAdaptiveMutex(final boolean useAdaptiveMutex) {
    assert (isInitialized());
    setUseAdaptiveMutex(nativeHandle_, useAdaptiveMutex);
    return this;
  }

  @Override protected void disposeInternal() {
    assert(isInitialized());
    disposeInternal(nativeHandle_);
  }

  private native void newComparatorOptions();
  private native boolean useAdaptiveMutex(final long handle);
  private native void setUseAdaptiveMutex(final long handle,
      final boolean useAdaptiveMutex);
  private native void disposeInternal(long handle);
}
