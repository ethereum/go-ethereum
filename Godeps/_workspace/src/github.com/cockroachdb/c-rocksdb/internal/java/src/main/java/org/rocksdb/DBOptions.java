// Copyright (c) 2014, Facebook, Inc.  All rights reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.

package org.rocksdb;

import java.util.Properties;

/**
 * DBOptions to control the behavior of a database.  It will be used
 * during the creation of a {@link org.rocksdb.RocksDB} (i.e., RocksDB.open()).
 *
 * If {@link #dispose()} function is not called, then it will be GC'd automatically
 * and native resources will be released as part of the process.
 */
public class DBOptions extends RocksObject implements DBOptionsInterface {
  static {
    RocksDB.loadLibrary();
  }

  /**
   * Construct DBOptions.
   *
   * This constructor will create (by allocating a block of memory)
   * an {@code rocksdb::DBOptions} in the c++ side.
   */
  public DBOptions() {
    super();
    numShardBits_ = DEFAULT_NUM_SHARD_BITS;
    newDBOptions();
  }

  /**
   * <p>Method to get a options instance by using pre-configured
   * property values. If one or many values are undefined in
   * the context of RocksDB the method will return a null
   * value.</p>
   *
   * <p><strong>Note</strong>: Property keys can be derived from
   * getter methods within the options class. Example: the method
   * {@code allowMmapReads()} has a property key:
   * {@code allow_mmap_reads}.</p>
   *
   * @param properties {@link java.util.Properties} instance.
   *
   * @return {@link org.rocksdb.DBOptions instance}
   *     or null.
   *
   * @throws java.lang.IllegalArgumentException if null or empty
   *     {@link java.util.Properties} instance is passed to the method call.
   */
  public static DBOptions getDBOptionsFromProps(
      final Properties properties) {
    if (properties == null || properties.size() == 0) {
      throw new IllegalArgumentException(
          "Properties value must contain at least one value.");
    }
    DBOptions dbOptions = null;
    StringBuilder stringBuilder = new StringBuilder();
    for (final String name : properties.stringPropertyNames()){
      stringBuilder.append(name);
      stringBuilder.append("=");
      stringBuilder.append(properties.getProperty(name));
      stringBuilder.append(";");
    }
    long handle = getDBOptionsFromProps(
        stringBuilder.toString());
    if (handle != 0){
      dbOptions = new DBOptions(handle);
    }
    return dbOptions;
  }

  @Override
  public DBOptions setIncreaseParallelism(
      final int totalThreads) {
    assert (isInitialized());
    setIncreaseParallelism(nativeHandle_, totalThreads);
    return this;
  }

  @Override
  public DBOptions setCreateIfMissing(final boolean flag) {
    assert(isInitialized());
    setCreateIfMissing(nativeHandle_, flag);
    return this;
  }

  @Override
  public boolean createIfMissing() {
    assert(isInitialized());
    return createIfMissing(nativeHandle_);
  }

  @Override
  public DBOptions setCreateMissingColumnFamilies(
      final boolean flag) {
    assert(isInitialized());
    setCreateMissingColumnFamilies(nativeHandle_, flag);
    return this;
  }

  @Override
  public boolean createMissingColumnFamilies() {
    assert(isInitialized());
    return createMissingColumnFamilies(nativeHandle_);
  }

  @Override
  public DBOptions setErrorIfExists(
      final boolean errorIfExists) {
    assert(isInitialized());
    setErrorIfExists(nativeHandle_, errorIfExists);
    return this;
  }

  @Override
  public boolean errorIfExists() {
    assert(isInitialized());
    return errorIfExists(nativeHandle_);
  }

  @Override
  public DBOptions setParanoidChecks(
      final boolean paranoidChecks) {
    assert(isInitialized());
    setParanoidChecks(nativeHandle_, paranoidChecks);
    return this;
  }

  @Override
  public boolean paranoidChecks() {
    assert(isInitialized());
    return paranoidChecks(nativeHandle_);
  }

  @Override
  public DBOptions setRateLimiterConfig(
      final RateLimiterConfig config) {
    assert(isInitialized());
    rateLimiterConfig_ = config;
    setRateLimiter(nativeHandle_, config.newRateLimiterHandle());
    return this;
  }

  @Override
  public DBOptions setLogger(final Logger logger) {
    assert(isInitialized());
    setLogger(nativeHandle_, logger.nativeHandle_);
    return this;
  }

  @Override
  public DBOptions setInfoLogLevel(
      final InfoLogLevel infoLogLevel) {
    assert(isInitialized());
    setInfoLogLevel(nativeHandle_, infoLogLevel.getValue());
    return this;
  }

  @Override
  public InfoLogLevel infoLogLevel() {
    assert(isInitialized());
    return InfoLogLevel.getInfoLogLevel(
        infoLogLevel(nativeHandle_));
  }

  @Override
  public DBOptions setMaxOpenFiles(
      final int maxOpenFiles) {
    assert(isInitialized());
    setMaxOpenFiles(nativeHandle_, maxOpenFiles);
    return this;
  }

  @Override
  public int maxOpenFiles() {
    assert(isInitialized());
    return maxOpenFiles(nativeHandle_);
  }

  @Override
  public DBOptions setMaxTotalWalSize(
      final long maxTotalWalSize) {
    assert(isInitialized());
    setMaxTotalWalSize(nativeHandle_, maxTotalWalSize);
    return this;
  }

  @Override
  public long maxTotalWalSize() {
    assert(isInitialized());
    return maxTotalWalSize(nativeHandle_);
  }

  @Override
  public DBOptions createStatistics() {
    assert(isInitialized());
    createStatistics(nativeHandle_);
    return this;
  }

  @Override
  public Statistics statisticsPtr() {
    assert(isInitialized());

    long statsPtr = statisticsPtr(nativeHandle_);
    if(statsPtr == 0) {
      createStatistics();
      statsPtr = statisticsPtr(nativeHandle_);
    }

    return new Statistics(statsPtr);
  }

  @Override
  public DBOptions setDisableDataSync(
      final boolean disableDataSync) {
    assert(isInitialized());
    setDisableDataSync(nativeHandle_, disableDataSync);
    return this;
  }

  @Override
  public boolean disableDataSync() {
    assert(isInitialized());
    return disableDataSync(nativeHandle_);
  }

  @Override
  public DBOptions setUseFsync(
      final boolean useFsync) {
    assert(isInitialized());
    setUseFsync(nativeHandle_, useFsync);
    return this;
  }

  @Override
  public boolean useFsync() {
    assert(isInitialized());
    return useFsync(nativeHandle_);
  }

  @Override
  public DBOptions setDbLogDir(
      final String dbLogDir) {
    assert(isInitialized());
    setDbLogDir(nativeHandle_, dbLogDir);
    return this;
  }

  @Override
  public String dbLogDir() {
    assert(isInitialized());
    return dbLogDir(nativeHandle_);
  }

  @Override
  public DBOptions setWalDir(
      final String walDir) {
    assert(isInitialized());
    setWalDir(nativeHandle_, walDir);
    return this;
  }

  @Override
  public String walDir() {
    assert(isInitialized());
    return walDir(nativeHandle_);
  }

  @Override
  public DBOptions setDeleteObsoleteFilesPeriodMicros(
      final long micros) {
    assert(isInitialized());
    setDeleteObsoleteFilesPeriodMicros(nativeHandle_, micros);
    return this;
  }

  @Override
  public long deleteObsoleteFilesPeriodMicros() {
    assert(isInitialized());
    return deleteObsoleteFilesPeriodMicros(nativeHandle_);
  }

  @Override
  public DBOptions setMaxBackgroundCompactions(
      final int maxBackgroundCompactions) {
    assert(isInitialized());
    setMaxBackgroundCompactions(nativeHandle_, maxBackgroundCompactions);
    return this;
  }

  @Override
  public int maxBackgroundCompactions() {
    assert(isInitialized());
    return maxBackgroundCompactions(nativeHandle_);
  }

  @Override
  public DBOptions setMaxBackgroundFlushes(
      final int maxBackgroundFlushes) {
    assert(isInitialized());
    setMaxBackgroundFlushes(nativeHandle_, maxBackgroundFlushes);
    return this;
  }

  @Override
  public int maxBackgroundFlushes() {
    assert(isInitialized());
    return maxBackgroundFlushes(nativeHandle_);
  }

  @Override
  public DBOptions setMaxLogFileSize(
      final long maxLogFileSize) {
    assert(isInitialized());
    setMaxLogFileSize(nativeHandle_, maxLogFileSize);
    return this;
  }

  @Override
  public long maxLogFileSize() {
    assert(isInitialized());
    return maxLogFileSize(nativeHandle_);
  }

  @Override
  public DBOptions setLogFileTimeToRoll(
      final long logFileTimeToRoll) {
    assert(isInitialized());
    setLogFileTimeToRoll(nativeHandle_, logFileTimeToRoll);
    return this;
  }

  @Override
  public long logFileTimeToRoll() {
    assert(isInitialized());
    return logFileTimeToRoll(nativeHandle_);
  }

  @Override
  public DBOptions setKeepLogFileNum(
      final long keepLogFileNum) {
    assert(isInitialized());
    setKeepLogFileNum(nativeHandle_, keepLogFileNum);
    return this;
  }

  @Override
  public long keepLogFileNum() {
    assert(isInitialized());
    return keepLogFileNum(nativeHandle_);
  }

  @Override
  public DBOptions setMaxManifestFileSize(
      final long maxManifestFileSize) {
    assert(isInitialized());
    setMaxManifestFileSize(nativeHandle_, maxManifestFileSize);
    return this;
  }

  @Override
  public long maxManifestFileSize() {
    assert(isInitialized());
    return maxManifestFileSize(nativeHandle_);
  }

  @Override
  public DBOptions setTableCacheNumshardbits(
      final int tableCacheNumshardbits) {
    assert(isInitialized());
    setTableCacheNumshardbits(nativeHandle_, tableCacheNumshardbits);
    return this;
  }

  @Override
  public int tableCacheNumshardbits() {
    assert(isInitialized());
    return tableCacheNumshardbits(nativeHandle_);
  }

  @Override
  public DBOptions setWalTtlSeconds(
      final long walTtlSeconds) {
    assert(isInitialized());
    setWalTtlSeconds(nativeHandle_, walTtlSeconds);
    return this;
  }

  @Override
  public long walTtlSeconds() {
    assert(isInitialized());
    return walTtlSeconds(nativeHandle_);
  }

  @Override
  public DBOptions setWalSizeLimitMB(
      final long sizeLimitMB) {
    assert(isInitialized());
    setWalSizeLimitMB(nativeHandle_, sizeLimitMB);
    return this;
  }

  @Override
  public long walSizeLimitMB() {
    assert(isInitialized());
    return walSizeLimitMB(nativeHandle_);
  }

  @Override
  public DBOptions setManifestPreallocationSize(
      final long size) {
    assert(isInitialized());
    setManifestPreallocationSize(nativeHandle_, size);
    return this;
  }

  @Override
  public long manifestPreallocationSize() {
    assert(isInitialized());
    return manifestPreallocationSize(nativeHandle_);
  }

  @Override
  public DBOptions setAllowOsBuffer(
      final boolean allowOsBuffer) {
    assert(isInitialized());
    setAllowOsBuffer(nativeHandle_, allowOsBuffer);
    return this;
  }

  @Override
  public boolean allowOsBuffer() {
    assert(isInitialized());
    return allowOsBuffer(nativeHandle_);
  }

  @Override
  public DBOptions setAllowMmapReads(
      final boolean allowMmapReads) {
    assert(isInitialized());
    setAllowMmapReads(nativeHandle_, allowMmapReads);
    return this;
  }

  @Override
  public boolean allowMmapReads() {
    assert(isInitialized());
    return allowMmapReads(nativeHandle_);
  }

  @Override
  public DBOptions setAllowMmapWrites(
      final boolean allowMmapWrites) {
    assert(isInitialized());
    setAllowMmapWrites(nativeHandle_, allowMmapWrites);
    return this;
  }

  @Override
  public boolean allowMmapWrites() {
    assert(isInitialized());
    return allowMmapWrites(nativeHandle_);
  }

  @Override
  public DBOptions setIsFdCloseOnExec(
      final boolean isFdCloseOnExec) {
    assert(isInitialized());
    setIsFdCloseOnExec(nativeHandle_, isFdCloseOnExec);
    return this;
  }

  @Override
  public boolean isFdCloseOnExec() {
    assert(isInitialized());
    return isFdCloseOnExec(nativeHandle_);
  }

  @Override
  public DBOptions setStatsDumpPeriodSec(
      final int statsDumpPeriodSec) {
    assert(isInitialized());
    setStatsDumpPeriodSec(nativeHandle_, statsDumpPeriodSec);
    return this;
  }

  @Override
  public int statsDumpPeriodSec() {
    assert(isInitialized());
    return statsDumpPeriodSec(nativeHandle_);
  }

  @Override
  public DBOptions setAdviseRandomOnOpen(
      final boolean adviseRandomOnOpen) {
    assert(isInitialized());
    setAdviseRandomOnOpen(nativeHandle_, adviseRandomOnOpen);
    return this;
  }

  @Override
  public boolean adviseRandomOnOpen() {
    return adviseRandomOnOpen(nativeHandle_);
  }

  @Override
  public DBOptions setUseAdaptiveMutex(
      final boolean useAdaptiveMutex) {
    assert(isInitialized());
    setUseAdaptiveMutex(nativeHandle_, useAdaptiveMutex);
    return this;
  }

  @Override
  public boolean useAdaptiveMutex() {
    assert(isInitialized());
    return useAdaptiveMutex(nativeHandle_);
  }

  @Override
  public DBOptions setBytesPerSync(
      final long bytesPerSync) {
    assert(isInitialized());
    setBytesPerSync(nativeHandle_, bytesPerSync);
    return this;
  }

  @Override
  public long bytesPerSync() {
    return bytesPerSync(nativeHandle_);
  }

  /**
   * Release the memory allocated for the current instance
   * in the c++ side.
   */
  @Override protected void disposeInternal() {
    assert(isInitialized());
    disposeInternal(nativeHandle_);
  }

  static final int DEFAULT_NUM_SHARD_BITS = -1;

  /**
   * <p>Private constructor to be used by
   * {@link #getDBOptionsFromProps(java.util.Properties)}</p>
   *
   * @param handle native handle to DBOptions instance.
   */
  private DBOptions(final long handle) {
    super();
    nativeHandle_ = handle;
  }

  private static native long getDBOptionsFromProps(
      String optString);

  private native void newDBOptions();
  private native void disposeInternal(long handle);

  private native void setIncreaseParallelism(long handle, int totalThreads);
  private native void setCreateIfMissing(long handle, boolean flag);
  private native boolean createIfMissing(long handle);
  private native void setCreateMissingColumnFamilies(
      long handle, boolean flag);
  private native boolean createMissingColumnFamilies(long handle);
  private native void setErrorIfExists(long handle, boolean errorIfExists);
  private native boolean errorIfExists(long handle);
  private native void setParanoidChecks(
      long handle, boolean paranoidChecks);
  private native boolean paranoidChecks(long handle);
  private native void setRateLimiter(long handle,
      long rateLimiterHandle);
  private native void setLogger(long handle,
      long loggerHandle);
  private native void setInfoLogLevel(long handle, byte logLevel);
  private native byte infoLogLevel(long handle);
  private native void setMaxOpenFiles(long handle, int maxOpenFiles);
  private native int maxOpenFiles(long handle);
  private native void setMaxTotalWalSize(long handle,
      long maxTotalWalSize);
  private native long maxTotalWalSize(long handle);
  private native void createStatistics(long optHandle);
  private native long statisticsPtr(long optHandle);
  private native void setDisableDataSync(long handle, boolean disableDataSync);
  private native boolean disableDataSync(long handle);
  private native boolean useFsync(long handle);
  private native void setUseFsync(long handle, boolean useFsync);
  private native void setDbLogDir(long handle, String dbLogDir);
  private native String dbLogDir(long handle);
  private native void setWalDir(long handle, String walDir);
  private native String walDir(long handle);
  private native void setDeleteObsoleteFilesPeriodMicros(
      long handle, long micros);
  private native long deleteObsoleteFilesPeriodMicros(long handle);
  private native void setMaxBackgroundCompactions(
      long handle, int maxBackgroundCompactions);
  private native int maxBackgroundCompactions(long handle);
  private native void setMaxBackgroundFlushes(
      long handle, int maxBackgroundFlushes);
  private native int maxBackgroundFlushes(long handle);
  private native void setMaxLogFileSize(long handle, long maxLogFileSize)
      throws IllegalArgumentException;
  private native long maxLogFileSize(long handle);
  private native void setLogFileTimeToRoll(
      long handle, long logFileTimeToRoll) throws IllegalArgumentException;
  private native long logFileTimeToRoll(long handle);
  private native void setKeepLogFileNum(long handle, long keepLogFileNum)
      throws IllegalArgumentException;
  private native long keepLogFileNum(long handle);
  private native void setMaxManifestFileSize(
      long handle, long maxManifestFileSize);
  private native long maxManifestFileSize(long handle);
  private native void setTableCacheNumshardbits(
      long handle, int tableCacheNumshardbits);
  private native int tableCacheNumshardbits(long handle);
  private native void setWalTtlSeconds(long handle, long walTtlSeconds);
  private native long walTtlSeconds(long handle);
  private native void setWalSizeLimitMB(long handle, long sizeLimitMB);
  private native long walSizeLimitMB(long handle);
  private native void setManifestPreallocationSize(
      long handle, long size) throws IllegalArgumentException;
  private native long manifestPreallocationSize(long handle);
  private native void setAllowOsBuffer(
      long handle, boolean allowOsBuffer);
  private native boolean allowOsBuffer(long handle);
  private native void setAllowMmapReads(
      long handle, boolean allowMmapReads);
  private native boolean allowMmapReads(long handle);
  private native void setAllowMmapWrites(
      long handle, boolean allowMmapWrites);
  private native boolean allowMmapWrites(long handle);
  private native void setIsFdCloseOnExec(
      long handle, boolean isFdCloseOnExec);
  private native boolean isFdCloseOnExec(long handle);
  private native void setStatsDumpPeriodSec(
      long handle, int statsDumpPeriodSec);
  private native int statsDumpPeriodSec(long handle);
  private native void setAdviseRandomOnOpen(
      long handle, boolean adviseRandomOnOpen);
  private native boolean adviseRandomOnOpen(long handle);
  private native void setUseAdaptiveMutex(
      long handle, boolean useAdaptiveMutex);
  private native boolean useAdaptiveMutex(long handle);
  private native void setBytesPerSync(
      long handle, long bytesPerSync);
  private native long bytesPerSync(long handle);

  int numShardBits_;
  RateLimiterConfig rateLimiterConfig_;
}
