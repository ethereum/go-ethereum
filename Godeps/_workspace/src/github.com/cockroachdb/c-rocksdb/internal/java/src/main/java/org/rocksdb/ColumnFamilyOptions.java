// Copyright (c) 2014, Facebook, Inc.  All rights reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.

package org.rocksdb;

import java.util.ArrayList;
import java.util.List;
import java.util.Properties;

/**
 * ColumnFamilyOptions to control the behavior of a database.  It will be used
 * during the creation of a {@link org.rocksdb.RocksDB} (i.e., RocksDB.open()).
 *
 * If {@link #dispose()} function is not called, then it will be GC'd automatically
 * and native resources will be released as part of the process.
 */
public class ColumnFamilyOptions extends RocksObject
    implements ColumnFamilyOptionsInterface {
  static {
    RocksDB.loadLibrary();
  }

  /**
   * Construct ColumnFamilyOptions.
   *
   * This constructor will create (by allocating a block of memory)
   * an {@code rocksdb::DBOptions} in the c++ side.
   */
  public ColumnFamilyOptions() {
    super();
    newColumnFamilyOptions();
  }

  /**
   * <p>Method to get a options instance by using pre-configured
   * property values. If one or many values are undefined in
   * the context of RocksDB the method will return a null
   * value.</p>
   *
   * <p><strong>Note</strong>: Property keys can be derived from
   * getter methods within the options class. Example: the method
   * {@code writeBufferSize()} has a property key:
   * {@code write_buffer_size}.</p>
   *
   * @param properties {@link java.util.Properties} instance.
   *
   * @return {@link org.rocksdb.ColumnFamilyOptions instance}
   *     or null.
   *
   * @throws java.lang.IllegalArgumentException if null or empty
   *     {@link Properties} instance is passed to the method call.
   */
  public static ColumnFamilyOptions getColumnFamilyOptionsFromProps(
      final Properties properties) {
    if (properties == null || properties.size() == 0) {
      throw new IllegalArgumentException(
          "Properties value must contain at least one value.");
    }
    ColumnFamilyOptions columnFamilyOptions = null;
    StringBuilder stringBuilder = new StringBuilder();
    for (final String name : properties.stringPropertyNames()){
      stringBuilder.append(name);
      stringBuilder.append("=");
      stringBuilder.append(properties.getProperty(name));
      stringBuilder.append(";");
    }
    long handle = getColumnFamilyOptionsFromProps(
        stringBuilder.toString());
    if (handle != 0){
      columnFamilyOptions = new ColumnFamilyOptions(handle);
    }
    return columnFamilyOptions;
  }

  @Override
  public ColumnFamilyOptions optimizeForPointLookup(
      final long blockCacheSizeMb) {
    optimizeForPointLookup(nativeHandle_,
        blockCacheSizeMb);
    return this;
  }

  @Override
  public ColumnFamilyOptions optimizeLevelStyleCompaction() {
    optimizeLevelStyleCompaction(nativeHandle_,
        DEFAULT_COMPACTION_MEMTABLE_MEMORY_BUDGET);
    return this;
  }

  @Override
  public ColumnFamilyOptions optimizeLevelStyleCompaction(
      final long memtableMemoryBudget) {
    optimizeLevelStyleCompaction(nativeHandle_,
        memtableMemoryBudget);
    return this;
  }

  @Override
  public ColumnFamilyOptions optimizeUniversalStyleCompaction() {
    optimizeUniversalStyleCompaction(nativeHandle_,
        DEFAULT_COMPACTION_MEMTABLE_MEMORY_BUDGET);
    return this;
  }

  @Override
  public ColumnFamilyOptions optimizeUniversalStyleCompaction(
      final long memtableMemoryBudget) {
    optimizeUniversalStyleCompaction(nativeHandle_,
        memtableMemoryBudget);
    return this;
  }

  @Override
  public ColumnFamilyOptions setComparator(final BuiltinComparator builtinComparator) {
    assert(isInitialized());
    setComparatorHandle(nativeHandle_, builtinComparator.ordinal());
    return this;
  }

  @Override
  public ColumnFamilyOptions setComparator(
      final AbstractComparator<? extends AbstractSlice<?>> comparator) {
    assert (isInitialized());
    setComparatorHandle(nativeHandle_, comparator.nativeHandle_);
    comparator_ = comparator;
    return this;
  }

  @Override
  public ColumnFamilyOptions setMergeOperatorName(final String name) {
    assert (isInitialized());
    if (name == null) {
      throw new IllegalArgumentException(
          "Merge operator name must not be null.");
    }
    setMergeOperatorName(nativeHandle_, name);
    return this;
  }

  @Override
  public ColumnFamilyOptions setMergeOperator(final MergeOperator mergeOperator) {
    setMergeOperator(nativeHandle_, mergeOperator.newMergeOperatorHandle());
    return this;
  }

  public ColumnFamilyOptions setCompactionFilter(
        final AbstractCompactionFilter<? extends AbstractSlice<?>> compactionFilter) {
    setCompactionFilterHandle(nativeHandle_, compactionFilter.nativeHandle_);
    compactionFilter_ = compactionFilter;
    return this;
  }

  @Override
  public ColumnFamilyOptions setWriteBufferSize(final long writeBufferSize) {
    assert(isInitialized());
    setWriteBufferSize(nativeHandle_, writeBufferSize);
    return this;
  }

  @Override
  public long writeBufferSize()  {
    assert(isInitialized());
    return writeBufferSize(nativeHandle_);
  }

  @Override
  public ColumnFamilyOptions setMaxWriteBufferNumber(
      final int maxWriteBufferNumber) {
    assert(isInitialized());
    setMaxWriteBufferNumber(nativeHandle_, maxWriteBufferNumber);
    return this;
  }

  @Override
  public int maxWriteBufferNumber() {
    assert(isInitialized());
    return maxWriteBufferNumber(nativeHandle_);
  }

  @Override
  public ColumnFamilyOptions setMinWriteBufferNumberToMerge(
      final int minWriteBufferNumberToMerge) {
    setMinWriteBufferNumberToMerge(nativeHandle_, minWriteBufferNumberToMerge);
    return this;
  }

  @Override
  public int minWriteBufferNumberToMerge() {
    return minWriteBufferNumberToMerge(nativeHandle_);
  }

  @Override
  public ColumnFamilyOptions useFixedLengthPrefixExtractor(final int n) {
    assert(isInitialized());
    useFixedLengthPrefixExtractor(nativeHandle_, n);
    return this;
  }

  @Override
  public ColumnFamilyOptions useCappedPrefixExtractor(final int n) {
    assert(isInitialized());
    useCappedPrefixExtractor(nativeHandle_, n);
    return this;
  }

  @Override
  public ColumnFamilyOptions setCompressionType(final CompressionType compressionType) {
    setCompressionType(nativeHandle_, compressionType.getValue());
    return this;
  }

  @Override
  public CompressionType compressionType() {
    return CompressionType.values()[compressionType(nativeHandle_)];
  }

  @Override
  public ColumnFamilyOptions setCompressionPerLevel(
      final List<CompressionType> compressionLevels) {
    final List<Byte> byteCompressionTypes = new ArrayList<>(
        compressionLevels.size());
    for (final CompressionType compressionLevel : compressionLevels) {
      byteCompressionTypes.add(compressionLevel.getValue());
    }
    setCompressionPerLevel(nativeHandle_, byteCompressionTypes);
    return this;
  }

  @Override
  public List<CompressionType> compressionPerLevel() {
    final List<Byte> byteCompressionTypes =
        compressionPerLevel(nativeHandle_);
    final List<CompressionType> compressionLevels = new ArrayList<>();
    for (final Byte byteCompressionType : byteCompressionTypes) {
      compressionLevels.add(CompressionType.getCompressionType(
          byteCompressionType));
    }
    return compressionLevels;
  }

  @Override
  public ColumnFamilyOptions setNumLevels(final int numLevels) {
    setNumLevels(nativeHandle_, numLevels);
    return this;
  }

  @Override
  public int numLevels() {
    return numLevels(nativeHandle_);
  }

  @Override
  public ColumnFamilyOptions setLevelZeroFileNumCompactionTrigger(
      final int numFiles) {
    setLevelZeroFileNumCompactionTrigger(
        nativeHandle_, numFiles);
    return this;
  }

  @Override
  public int levelZeroFileNumCompactionTrigger() {
    return levelZeroFileNumCompactionTrigger(nativeHandle_);
  }

  @Override
  public ColumnFamilyOptions setLevelZeroSlowdownWritesTrigger(
      final int numFiles) {
    setLevelZeroSlowdownWritesTrigger(nativeHandle_, numFiles);
    return this;
  }

  @Override
  public int levelZeroSlowdownWritesTrigger() {
    return levelZeroSlowdownWritesTrigger(nativeHandle_);
  }

  @Override
  public ColumnFamilyOptions setLevelZeroStopWritesTrigger(final int numFiles) {
    setLevelZeroStopWritesTrigger(nativeHandle_, numFiles);
    return this;
  }

  @Override
  public int levelZeroStopWritesTrigger() {
    return levelZeroStopWritesTrigger(nativeHandle_);
  }

  @Override
  public ColumnFamilyOptions setMaxMemCompactionLevel(
      final int maxMemCompactionLevel) {
    return this;
  }

  @Override
  public int maxMemCompactionLevel() {
    return 0;
  }

  @Override
  public ColumnFamilyOptions setTargetFileSizeBase(
      final long targetFileSizeBase) {
    setTargetFileSizeBase(nativeHandle_, targetFileSizeBase);
    return this;
  }

  @Override
  public long targetFileSizeBase() {
    return targetFileSizeBase(nativeHandle_);
  }

  @Override
  public ColumnFamilyOptions setTargetFileSizeMultiplier(
      final int multiplier) {
    setTargetFileSizeMultiplier(nativeHandle_, multiplier);
    return this;
  }

  @Override
  public int targetFileSizeMultiplier() {
    return targetFileSizeMultiplier(nativeHandle_);
  }

  @Override
  public ColumnFamilyOptions setMaxBytesForLevelBase(
      final long maxBytesForLevelBase) {
    setMaxBytesForLevelBase(nativeHandle_, maxBytesForLevelBase);
    return this;
  }

  @Override
  public long maxBytesForLevelBase() {
    return maxBytesForLevelBase(nativeHandle_);
  }

  @Override
  public ColumnFamilyOptions setLevelCompactionDynamicLevelBytes(
      final boolean enableLevelCompactionDynamicLevelBytes) {
    setLevelCompactionDynamicLevelBytes(nativeHandle_,
        enableLevelCompactionDynamicLevelBytes);
    return this;
  }

  @Override
  public boolean levelCompactionDynamicLevelBytes() {
    return levelCompactionDynamicLevelBytes(nativeHandle_);
  }

  @Override
  public ColumnFamilyOptions setMaxBytesForLevelMultiplier(
      final int multiplier) {
    setMaxBytesForLevelMultiplier(nativeHandle_, multiplier);
    return this;
  }

  @Override
  public int maxBytesForLevelMultiplier() {
    return maxBytesForLevelMultiplier(nativeHandle_);
  }

  @Override
  public ColumnFamilyOptions setExpandedCompactionFactor(
      final int expandedCompactionFactor) {
    setExpandedCompactionFactor(nativeHandle_, expandedCompactionFactor);
    return this;
  }

  @Override
  public int expandedCompactionFactor() {
    return expandedCompactionFactor(nativeHandle_);
  }

  @Override
  public ColumnFamilyOptions setSourceCompactionFactor(
      final int sourceCompactionFactor) {
    setSourceCompactionFactor(nativeHandle_, sourceCompactionFactor);
    return this;
  }

  @Override
  public int sourceCompactionFactor() {
    return sourceCompactionFactor(nativeHandle_);
  }

  @Override
  public ColumnFamilyOptions setMaxGrandparentOverlapFactor(
      final int maxGrandparentOverlapFactor) {
    setMaxGrandparentOverlapFactor(nativeHandle_, maxGrandparentOverlapFactor);
    return this;
  }

  @Override
  public int maxGrandparentOverlapFactor() {
    return maxGrandparentOverlapFactor(nativeHandle_);
  }

  @Override
  public ColumnFamilyOptions setSoftRateLimit(
      final double softRateLimit) {
    setSoftRateLimit(nativeHandle_, softRateLimit);
    return this;
  }

  @Override
  public double softRateLimit() {
    return softRateLimit(nativeHandle_);
  }

  @Override
  public ColumnFamilyOptions setHardRateLimit(
      final double hardRateLimit) {
    setHardRateLimit(nativeHandle_, hardRateLimit);
    return this;
  }

  @Override
  public double hardRateLimit() {
    return hardRateLimit(nativeHandle_);
  }

  @Override
  public ColumnFamilyOptions setRateLimitDelayMaxMilliseconds(
      final int rateLimitDelayMaxMilliseconds) {
    setRateLimitDelayMaxMilliseconds(
        nativeHandle_, rateLimitDelayMaxMilliseconds);
    return this;
  }

  @Override
  public int rateLimitDelayMaxMilliseconds() {
    return rateLimitDelayMaxMilliseconds(nativeHandle_);
  }

  @Override
  public ColumnFamilyOptions setArenaBlockSize(
      final long arenaBlockSize) {
    setArenaBlockSize(nativeHandle_, arenaBlockSize);
    return this;
  }

  @Override
  public long arenaBlockSize() {
    return arenaBlockSize(nativeHandle_);
  }

  @Override
  public ColumnFamilyOptions setDisableAutoCompactions(
      final boolean disableAutoCompactions) {
    setDisableAutoCompactions(nativeHandle_, disableAutoCompactions);
    return this;
  }

  @Override
  public boolean disableAutoCompactions() {
    return disableAutoCompactions(nativeHandle_);
  }

  @Override
  public ColumnFamilyOptions setPurgeRedundantKvsWhileFlush(
      final boolean purgeRedundantKvsWhileFlush) {
    setPurgeRedundantKvsWhileFlush(
        nativeHandle_, purgeRedundantKvsWhileFlush);
    return this;
  }

  @Override
  public boolean purgeRedundantKvsWhileFlush() {
    return purgeRedundantKvsWhileFlush(nativeHandle_);
  }

  @Override
  public ColumnFamilyOptions setCompactionStyle(
      final CompactionStyle compactionStyle) {
    setCompactionStyle(nativeHandle_, compactionStyle.getValue());
    return this;
  }

  @Override
  public CompactionStyle compactionStyle() {
    return CompactionStyle.values()[compactionStyle(nativeHandle_)];
  }

  @Override
  public ColumnFamilyOptions setMaxTableFilesSizeFIFO(
      final long maxTableFilesSize) {
    assert(maxTableFilesSize > 0); // unsigned native type
    assert(isInitialized());
    setMaxTableFilesSizeFIFO(nativeHandle_, maxTableFilesSize);
    return this;
  }

  @Override
  public long maxTableFilesSizeFIFO() {
    return maxTableFilesSizeFIFO(nativeHandle_);
  }

  @Override
  public ColumnFamilyOptions setVerifyChecksumsInCompaction(
      final boolean verifyChecksumsInCompaction) {
    setVerifyChecksumsInCompaction(
        nativeHandle_, verifyChecksumsInCompaction);
    return this;
  }

  @Override
  public boolean verifyChecksumsInCompaction() {
    return verifyChecksumsInCompaction(nativeHandle_);
  }

  @Override
  public ColumnFamilyOptions setFilterDeletes(
      final boolean filterDeletes) {
    setFilterDeletes(nativeHandle_, filterDeletes);
    return this;
  }

  @Override
  public boolean filterDeletes() {
    return filterDeletes(nativeHandle_);
  }

  @Override
  public ColumnFamilyOptions setMaxSequentialSkipInIterations(
      final long maxSequentialSkipInIterations) {
    setMaxSequentialSkipInIterations(nativeHandle_, maxSequentialSkipInIterations);
    return this;
  }

  @Override
  public long maxSequentialSkipInIterations() {
    return maxSequentialSkipInIterations(nativeHandle_);
  }

  @Override
  public ColumnFamilyOptions setMemTableConfig(
      final MemTableConfig config) {
    memTableConfig_ = config;
    setMemTableFactory(nativeHandle_, config.newMemTableFactoryHandle());
    return this;
  }

  @Override
  public String memTableFactoryName() {
    assert(isInitialized());
    return memTableFactoryName(nativeHandle_);
  }

  @Override
  public ColumnFamilyOptions setTableFormatConfig(
      final TableFormatConfig config) {
    tableFormatConfig_ = config;
    setTableFactory(nativeHandle_, config.newTableFactoryHandle());
    return this;
  }

  @Override
  public String tableFactoryName() {
    assert(isInitialized());
    return tableFactoryName(nativeHandle_);
  }

  @Override
  public ColumnFamilyOptions setInplaceUpdateSupport(
      final boolean inplaceUpdateSupport) {
    setInplaceUpdateSupport(nativeHandle_, inplaceUpdateSupport);
    return this;
  }

  @Override
  public boolean inplaceUpdateSupport() {
    return inplaceUpdateSupport(nativeHandle_);
  }

  @Override
  public ColumnFamilyOptions setInplaceUpdateNumLocks(
      final long inplaceUpdateNumLocks) {
    setInplaceUpdateNumLocks(nativeHandle_, inplaceUpdateNumLocks);
    return this;
  }

  @Override
  public long inplaceUpdateNumLocks() {
    return inplaceUpdateNumLocks(nativeHandle_);
  }

  @Override
  public ColumnFamilyOptions setMemtablePrefixBloomBits(
      final int memtablePrefixBloomBits) {
    setMemtablePrefixBloomBits(nativeHandle_, memtablePrefixBloomBits);
    return this;
  }

  @Override
  public int memtablePrefixBloomBits() {
    return memtablePrefixBloomBits(nativeHandle_);
  }

  @Override
  public ColumnFamilyOptions setMemtablePrefixBloomProbes(
      final int memtablePrefixBloomProbes) {
    setMemtablePrefixBloomProbes(nativeHandle_, memtablePrefixBloomProbes);
    return this;
  }

  @Override
  public int memtablePrefixBloomProbes() {
    return memtablePrefixBloomProbes(nativeHandle_);
  }

  @Override
  public ColumnFamilyOptions setBloomLocality(int bloomLocality) {
    setBloomLocality(nativeHandle_, bloomLocality);
    return this;
  }

  @Override
  public int bloomLocality() {
    return bloomLocality(nativeHandle_);
  }

  @Override
  public ColumnFamilyOptions setMaxSuccessiveMerges(
      final long maxSuccessiveMerges) {
    setMaxSuccessiveMerges(nativeHandle_, maxSuccessiveMerges);
    return this;
  }

  @Override
  public long maxSuccessiveMerges() {
    return maxSuccessiveMerges(nativeHandle_);
  }

  @Override
  public ColumnFamilyOptions setMinPartialMergeOperands(
      final int minPartialMergeOperands) {
    setMinPartialMergeOperands(nativeHandle_, minPartialMergeOperands);
    return this;
  }

  @Override
  public int minPartialMergeOperands() {
    return minPartialMergeOperands(nativeHandle_);
  }

  @Override
  public ColumnFamilyOptions setOptimizeFiltersForHits(
      final boolean optimizeFiltersForHits) {
    setOptimizeFiltersForHits(nativeHandle_, optimizeFiltersForHits);
    return this;
  }

  @Override
  public boolean optimizeFiltersForHits() {
    return optimizeFiltersForHits(nativeHandle_);
  }

  /**
   * Release the memory allocated for the current instance
   * in the c++ side.
   */
  @Override protected void disposeInternal() {
    assert(isInitialized());
    disposeInternal(nativeHandle_);
  }

  /**
   * <p>Private constructor to be used by
   * {@link #getColumnFamilyOptionsFromProps(java.util.Properties)}</p>
   *
   * @param handle native handle to ColumnFamilyOptions instance.
   */
  private ColumnFamilyOptions(final long handle) {
    super();
    nativeHandle_ = handle;
  }

  private static native long getColumnFamilyOptionsFromProps(
      String optString);

  private native void newColumnFamilyOptions();
  private native void disposeInternal(long handle);

  private native void optimizeForPointLookup(long handle,
      long blockCacheSizeMb);
  private native void optimizeLevelStyleCompaction(long handle,
      long memtableMemoryBudget);
  private native void optimizeUniversalStyleCompaction(long handle,
      long memtableMemoryBudget);
  private native void setComparatorHandle(long handle, int builtinComparator);
  private native void setComparatorHandle(long optHandle, long comparatorHandle);
  private native void setMergeOperatorName(
      long handle, String name);
  private native void setMergeOperator(
      long handle, long mergeOperatorHandle);
  private native void setCompactionFilterHandle(long handle, long compactionFilterHandle);
  private native void setWriteBufferSize(long handle, long writeBufferSize)
      throws IllegalArgumentException;
  private native long writeBufferSize(long handle);
  private native void setMaxWriteBufferNumber(
      long handle, int maxWriteBufferNumber);
  private native int maxWriteBufferNumber(long handle);
  private native void setMinWriteBufferNumberToMerge(
      long handle, int minWriteBufferNumberToMerge);
  private native int minWriteBufferNumberToMerge(long handle);
  private native void setCompressionType(long handle, byte compressionType);
  private native byte compressionType(long handle);
  private native void setCompressionPerLevel(long handle,
      List<Byte> compressionLevels);
  private native List<Byte> compressionPerLevel(long handle);
  private native void useFixedLengthPrefixExtractor(
      long handle, int prefixLength);
  private native void useCappedPrefixExtractor(
      long handle, int prefixLength);
  private native void setNumLevels(
      long handle, int numLevels);
  private native int numLevels(long handle);
  private native void setLevelZeroFileNumCompactionTrigger(
      long handle, int numFiles);
  private native int levelZeroFileNumCompactionTrigger(long handle);
  private native void setLevelZeroSlowdownWritesTrigger(
      long handle, int numFiles);
  private native int levelZeroSlowdownWritesTrigger(long handle);
  private native void setLevelZeroStopWritesTrigger(
      long handle, int numFiles);
  private native int levelZeroStopWritesTrigger(long handle);
  private native void setTargetFileSizeBase(
      long handle, long targetFileSizeBase);
  private native long targetFileSizeBase(long handle);
  private native void setTargetFileSizeMultiplier(
      long handle, int multiplier);
  private native int targetFileSizeMultiplier(long handle);
  private native void setMaxBytesForLevelBase(
      long handle, long maxBytesForLevelBase);
  private native long maxBytesForLevelBase(long handle);
  private native void setLevelCompactionDynamicLevelBytes(
      long handle, boolean enableLevelCompactionDynamicLevelBytes);
  private native boolean levelCompactionDynamicLevelBytes(
      long handle);
  private native void setMaxBytesForLevelMultiplier(
      long handle, int multiplier);
  private native int maxBytesForLevelMultiplier(long handle);
  private native void setExpandedCompactionFactor(
      long handle, int expandedCompactionFactor);
  private native int expandedCompactionFactor(long handle);
  private native void setSourceCompactionFactor(
      long handle, int sourceCompactionFactor);
  private native int sourceCompactionFactor(long handle);
  private native void setMaxGrandparentOverlapFactor(
      long handle, int maxGrandparentOverlapFactor);
  private native int maxGrandparentOverlapFactor(long handle);
  private native void setSoftRateLimit(
      long handle, double softRateLimit);
  private native double softRateLimit(long handle);
  private native void setHardRateLimit(
      long handle, double hardRateLimit);
  private native double hardRateLimit(long handle);
  private native void setRateLimitDelayMaxMilliseconds(
      long handle, int rateLimitDelayMaxMilliseconds);
  private native int rateLimitDelayMaxMilliseconds(long handle);
  private native void setArenaBlockSize(
      long handle, long arenaBlockSize)
      throws IllegalArgumentException;
  private native long arenaBlockSize(long handle);
  private native void setDisableAutoCompactions(
      long handle, boolean disableAutoCompactions);
  private native boolean disableAutoCompactions(long handle);
  private native void setCompactionStyle(long handle, byte compactionStyle);
  private native byte compactionStyle(long handle);
   private native void setMaxTableFilesSizeFIFO(
      long handle, long max_table_files_size);
  private native long maxTableFilesSizeFIFO(long handle);
  private native void setPurgeRedundantKvsWhileFlush(
      long handle, boolean purgeRedundantKvsWhileFlush);
  private native boolean purgeRedundantKvsWhileFlush(long handle);
  private native void setVerifyChecksumsInCompaction(
      long handle, boolean verifyChecksumsInCompaction);
  private native boolean verifyChecksumsInCompaction(long handle);
  private native void setFilterDeletes(
      long handle, boolean filterDeletes);
  private native boolean filterDeletes(long handle);
  private native void setMaxSequentialSkipInIterations(
      long handle, long maxSequentialSkipInIterations);
  private native long maxSequentialSkipInIterations(long handle);
  private native void setMemTableFactory(long handle, long factoryHandle);
  private native String memTableFactoryName(long handle);
  private native void setTableFactory(long handle, long factoryHandle);
  private native String tableFactoryName(long handle);
  private native void setInplaceUpdateSupport(
      long handle, boolean inplaceUpdateSupport);
  private native boolean inplaceUpdateSupport(long handle);
  private native void setInplaceUpdateNumLocks(
      long handle, long inplaceUpdateNumLocks)
      throws IllegalArgumentException;
  private native long inplaceUpdateNumLocks(long handle);
  private native void setMemtablePrefixBloomBits(
      long handle, int memtablePrefixBloomBits);
  private native int memtablePrefixBloomBits(long handle);
  private native void setMemtablePrefixBloomProbes(
      long handle, int memtablePrefixBloomProbes);
  private native int memtablePrefixBloomProbes(long handle);
  private native void setBloomLocality(
      long handle, int bloomLocality);
  private native int bloomLocality(long handle);
  private native void setMaxSuccessiveMerges(
      long handle, long maxSuccessiveMerges)
      throws IllegalArgumentException;
  private native long maxSuccessiveMerges(long handle);
  private native void setMinPartialMergeOperands(
      long handle, int minPartialMergeOperands);
  private native int minPartialMergeOperands(long handle);
  private native void setOptimizeFiltersForHits(long handle,
      boolean optimizeFiltersForHits);
  private native boolean optimizeFiltersForHits(long handle);

  MemTableConfig memTableConfig_;
  TableFormatConfig tableFormatConfig_;
  AbstractComparator<? extends AbstractSlice<?>> comparator_;
  AbstractCompactionFilter<? extends AbstractSlice<?>> compactionFilter_;
}
