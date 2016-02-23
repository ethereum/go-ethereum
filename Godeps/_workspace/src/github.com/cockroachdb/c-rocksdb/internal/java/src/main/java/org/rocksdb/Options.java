// Copyright (c) 2014, Facebook, Inc.  All rights reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.

package org.rocksdb;

import java.util.ArrayList;
import java.util.List;

/**
 * Options to control the behavior of a database.  It will be used
 * during the creation of a {@link org.rocksdb.RocksDB} (i.e., RocksDB.open()).
 *
 * If {@link #dispose()} function is not called, then it will be GC'd automatically
 * and native resources will be released as part of the process.
 */
public class Options extends RocksObject
    implements DBOptionsInterface, ColumnFamilyOptionsInterface {
  static {
    RocksDB.loadLibrary();
  }
  /**
   * Construct options for opening a RocksDB.
   *
   * This constructor will create (by allocating a block of memory)
   * an {@code rocksdb::Options} in the c++ side.
   */
  public Options() {
    super();
    newOptions();
    env_ = Env.getDefault();
  }

  /**
   * Construct options for opening a RocksDB. Reusing database options
   * and column family options.
   *
   * @param dbOptions {@link org.rocksdb.DBOptions} instance
   * @param columnFamilyOptions {@link org.rocksdb.ColumnFamilyOptions}
   *     instance
   */
  public Options(final DBOptions dbOptions,
      final ColumnFamilyOptions columnFamilyOptions) {
    super();
    newOptions(dbOptions.nativeHandle_, columnFamilyOptions.nativeHandle_);
    env_ = Env.getDefault();
  }

  @Override
  public Options setIncreaseParallelism(final int totalThreads) {
    assert(isInitialized());
    setIncreaseParallelism(nativeHandle_, totalThreads);
    return this;
  }

  @Override
  public Options setCreateIfMissing(final boolean flag) {
    assert(isInitialized());
    setCreateIfMissing(nativeHandle_, flag);
    return this;
  }

  @Override
  public Options setCreateMissingColumnFamilies(final boolean flag) {
    assert(isInitialized());
    setCreateMissingColumnFamilies(nativeHandle_, flag);
    return this;
  }

  /**
   * Use the specified object to interact with the environment,
   * e.g. to read/write files, schedule background work, etc.
   * Default: {@link Env#getDefault()}
   *
   * @param env {@link Env} instance.
   * @return the instance of the current Options.
   */
  public Options setEnv(final Env env) {
    assert(isInitialized());
    setEnv(nativeHandle_, env.nativeHandle_);
    env_ = env;
    return this;
  }

  /**
   * Returns the set RocksEnv instance.
   *
   * @return {@link RocksEnv} instance set in the Options.
   */
  public Env getEnv() {
    return env_;
  }

  /**
   * <p>Set appropriate parameters for bulk loading.
   * The reason that this is a function that returns "this" instead of a
   * constructor is to enable chaining of multiple similar calls in the future.
   * </p>
   *
   * <p>All data will be in level 0 without any automatic compaction.
   * It's recommended to manually call CompactRange(NULL, NULL) before reading
   * from the database, because otherwise the read can be very slow.</p>
   *
   * @return the instance of the current Options.
   */
  public Options prepareForBulkLoad() {
    prepareForBulkLoad(nativeHandle_);
    return this;
  }

  @Override
  public boolean createIfMissing() {
    assert(isInitialized());
    return createIfMissing(nativeHandle_);
  }

  @Override
  public boolean createMissingColumnFamilies() {
    assert(isInitialized());
    return createMissingColumnFamilies(nativeHandle_);
  }

  @Override
  public Options optimizeForPointLookup(
      long blockCacheSizeMb) {
    optimizeForPointLookup(nativeHandle_,
        blockCacheSizeMb);
    return this;
  }

  @Override
  public Options optimizeLevelStyleCompaction() {
    optimizeLevelStyleCompaction(nativeHandle_,
        DEFAULT_COMPACTION_MEMTABLE_MEMORY_BUDGET);
    return this;
  }

  @Override
  public Options optimizeLevelStyleCompaction(
      long memtableMemoryBudget) {
    optimizeLevelStyleCompaction(nativeHandle_,
        memtableMemoryBudget);
    return this;
  }

  @Override
  public Options optimizeUniversalStyleCompaction() {
    optimizeUniversalStyleCompaction(nativeHandle_,
        DEFAULT_COMPACTION_MEMTABLE_MEMORY_BUDGET);
    return this;
  }

  @Override
  public Options optimizeUniversalStyleCompaction(
      final long memtableMemoryBudget) {
    optimizeUniversalStyleCompaction(nativeHandle_,
        memtableMemoryBudget);
    return this;
  }

  @Override
  public Options setComparator(final BuiltinComparator builtinComparator) {
    assert(isInitialized());
    setComparatorHandle(nativeHandle_, builtinComparator.ordinal());
    return this;
  }

  @Override
  public Options setComparator(
      final AbstractComparator<? extends AbstractSlice<?>> comparator) {
    assert (isInitialized());
    setComparatorHandle(nativeHandle_, comparator.nativeHandle_);
    comparator_ = comparator;
    return this;
  }

  @Override
  public Options setMergeOperatorName(final String name) {
    assert (isInitialized());
    if (name == null) {
      throw new IllegalArgumentException(
          "Merge operator name must not be null.");
    }
    setMergeOperatorName(nativeHandle_, name);
    return this;
  }

  @Override
  public Options setMergeOperator(final MergeOperator mergeOperator) {
    setMergeOperator(nativeHandle_, mergeOperator.newMergeOperatorHandle());
    return this;
  }

  @Override
  public Options setWriteBufferSize(final long writeBufferSize) {
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
  public Options setMaxWriteBufferNumber(final int maxWriteBufferNumber) {
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
  public boolean errorIfExists() {
    assert(isInitialized());
    return errorIfExists(nativeHandle_);
  }

  @Override
  public Options setErrorIfExists(final boolean errorIfExists) {
    assert(isInitialized());
    setErrorIfExists(nativeHandle_, errorIfExists);
    return this;
  }

  @Override
  public boolean paranoidChecks() {
    assert(isInitialized());
    return paranoidChecks(nativeHandle_);
  }

  @Override
  public Options setParanoidChecks(final boolean paranoidChecks) {
    assert(isInitialized());
    setParanoidChecks(nativeHandle_, paranoidChecks);
    return this;
  }

  @Override
  public int maxOpenFiles() {
    assert(isInitialized());
    return maxOpenFiles(nativeHandle_);
  }

  @Override
  public Options setMaxTotalWalSize(final long maxTotalWalSize) {
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
  public Options setMaxOpenFiles(final int maxOpenFiles) {
    assert(isInitialized());
    setMaxOpenFiles(nativeHandle_, maxOpenFiles);
    return this;
  }

  @Override
  public boolean disableDataSync() {
    assert(isInitialized());
    return disableDataSync(nativeHandle_);
  }

  @Override
  public Options setDisableDataSync(final boolean disableDataSync) {
    assert(isInitialized());
    setDisableDataSync(nativeHandle_, disableDataSync);
    return this;
  }

  @Override
  public boolean useFsync() {
    assert(isInitialized());
    return useFsync(nativeHandle_);
  }

  @Override
  public Options setUseFsync(final boolean useFsync) {
    assert(isInitialized());
    setUseFsync(nativeHandle_, useFsync);
    return this;
  }

  @Override
  public String dbLogDir() {
    assert(isInitialized());
    return dbLogDir(nativeHandle_);
  }

  @Override
  public Options setDbLogDir(final String dbLogDir) {
    assert(isInitialized());
    setDbLogDir(nativeHandle_, dbLogDir);
    return this;
  }

  @Override
  public String walDir() {
    assert(isInitialized());
    return walDir(nativeHandle_);
  }

  @Override
  public Options setWalDir(final String walDir) {
    assert(isInitialized());
    setWalDir(nativeHandle_, walDir);
    return this;
  }

  @Override
  public long deleteObsoleteFilesPeriodMicros() {
    assert(isInitialized());
    return deleteObsoleteFilesPeriodMicros(nativeHandle_);
  }

  @Override
  public Options setDeleteObsoleteFilesPeriodMicros(
      final long micros) {
    assert(isInitialized());
    setDeleteObsoleteFilesPeriodMicros(nativeHandle_, micros);
    return this;
  }

  @Override
  public int maxBackgroundCompactions() {
    assert(isInitialized());
    return maxBackgroundCompactions(nativeHandle_);
  }

  @Override
  public Options createStatistics() {
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
  public Options setMaxBackgroundCompactions(
      final int maxBackgroundCompactions) {
    assert(isInitialized());
    setMaxBackgroundCompactions(nativeHandle_, maxBackgroundCompactions);
    return this;
  }

  @Override
  public int maxBackgroundFlushes() {
    assert(isInitialized());
    return maxBackgroundFlushes(nativeHandle_);
  }

  @Override
  public Options setMaxBackgroundFlushes(
      final int maxBackgroundFlushes) {
    assert(isInitialized());
    setMaxBackgroundFlushes(nativeHandle_, maxBackgroundFlushes);
    return this;
  }

  @Override
  public long maxLogFileSize() {
    assert(isInitialized());
    return maxLogFileSize(nativeHandle_);
  }

  @Override
  public Options setMaxLogFileSize(final long maxLogFileSize) {
    assert(isInitialized());
    setMaxLogFileSize(nativeHandle_, maxLogFileSize);
    return this;
  }

  @Override
  public long logFileTimeToRoll() {
    assert(isInitialized());
    return logFileTimeToRoll(nativeHandle_);
  }

  @Override
  public Options setLogFileTimeToRoll(final long logFileTimeToRoll) {
    assert(isInitialized());
    setLogFileTimeToRoll(nativeHandle_, logFileTimeToRoll);
    return this;
  }

  @Override
  public long keepLogFileNum() {
    assert(isInitialized());
    return keepLogFileNum(nativeHandle_);
  }

  @Override
  public Options setKeepLogFileNum(final long keepLogFileNum) {
    assert(isInitialized());
    setKeepLogFileNum(nativeHandle_, keepLogFileNum);
    return this;
  }

  @Override
  public long maxManifestFileSize() {
    assert(isInitialized());
    return maxManifestFileSize(nativeHandle_);
  }

  @Override
  public Options setMaxManifestFileSize(
      final long maxManifestFileSize) {
    assert(isInitialized());
    setMaxManifestFileSize(nativeHandle_, maxManifestFileSize);
    return this;
  }

  @Override
  public Options setMaxTableFilesSizeFIFO(
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
  public int tableCacheNumshardbits() {
    assert(isInitialized());
    return tableCacheNumshardbits(nativeHandle_);
  }

  @Override
  public Options setTableCacheNumshardbits(
      final int tableCacheNumshardbits) {
    assert(isInitialized());
    setTableCacheNumshardbits(nativeHandle_, tableCacheNumshardbits);
    return this;
  }

  @Override
  public long walTtlSeconds() {
    assert(isInitialized());
    return walTtlSeconds(nativeHandle_);
  }

  @Override
  public Options setWalTtlSeconds(final long walTtlSeconds) {
    assert(isInitialized());
    setWalTtlSeconds(nativeHandle_, walTtlSeconds);
    return this;
  }

  @Override
  public long walSizeLimitMB() {
    assert(isInitialized());
    return walSizeLimitMB(nativeHandle_);
  }

  @Override
  public Options setWalSizeLimitMB(final long sizeLimitMB) {
    assert(isInitialized());
    setWalSizeLimitMB(nativeHandle_, sizeLimitMB);
    return this;
  }

  @Override
  public long manifestPreallocationSize() {
    assert(isInitialized());
    return manifestPreallocationSize(nativeHandle_);
  }

  @Override
  public Options setManifestPreallocationSize(final long size) {
    assert(isInitialized());
    setManifestPreallocationSize(nativeHandle_, size);
    return this;
  }

  @Override
  public boolean allowOsBuffer() {
    assert(isInitialized());
    return allowOsBuffer(nativeHandle_);
  }

  @Override
  public Options setAllowOsBuffer(final boolean allowOsBuffer) {
    assert(isInitialized());
    setAllowOsBuffer(nativeHandle_, allowOsBuffer);
    return this;
  }

  @Override
  public boolean allowMmapReads() {
    assert(isInitialized());
    return allowMmapReads(nativeHandle_);
  }

  @Override
  public Options setAllowMmapReads(final boolean allowMmapReads) {
    assert(isInitialized());
    setAllowMmapReads(nativeHandle_, allowMmapReads);
    return this;
  }

  @Override
  public boolean allowMmapWrites() {
    assert(isInitialized());
    return allowMmapWrites(nativeHandle_);
  }

  @Override
  public Options setAllowMmapWrites(final boolean allowMmapWrites) {
    assert(isInitialized());
    setAllowMmapWrites(nativeHandle_, allowMmapWrites);
    return this;
  }

  @Override
  public boolean isFdCloseOnExec() {
    assert(isInitialized());
    return isFdCloseOnExec(nativeHandle_);
  }

  @Override
  public Options setIsFdCloseOnExec(final boolean isFdCloseOnExec) {
    assert(isInitialized());
    setIsFdCloseOnExec(nativeHandle_, isFdCloseOnExec);
    return this;
  }

  @Override
  public int statsDumpPeriodSec() {
    assert(isInitialized());
    return statsDumpPeriodSec(nativeHandle_);
  }

  @Override
  public Options setStatsDumpPeriodSec(final int statsDumpPeriodSec) {
    assert(isInitialized());
    setStatsDumpPeriodSec(nativeHandle_, statsDumpPeriodSec);
    return this;
  }

  @Override
  public boolean adviseRandomOnOpen() {
    return adviseRandomOnOpen(nativeHandle_);
  }

  @Override
  public Options setAdviseRandomOnOpen(final boolean adviseRandomOnOpen) {
    assert(isInitialized());
    setAdviseRandomOnOpen(nativeHandle_, adviseRandomOnOpen);
    return this;
  }

  @Override
  public boolean useAdaptiveMutex() {
    assert(isInitialized());
    return useAdaptiveMutex(nativeHandle_);
  }

  @Override
  public Options setUseAdaptiveMutex(final boolean useAdaptiveMutex) {
    assert(isInitialized());
    setUseAdaptiveMutex(nativeHandle_, useAdaptiveMutex);
    return this;
  }

  @Override
  public long bytesPerSync() {
    return bytesPerSync(nativeHandle_);
  }

  @Override
  public Options setBytesPerSync(final long bytesPerSync) {
    assert(isInitialized());
    setBytesPerSync(nativeHandle_, bytesPerSync);
    return this;
  }

  @Override
  public Options setMemTableConfig(final MemTableConfig config) {
    memTableConfig_ = config;
    setMemTableFactory(nativeHandle_, config.newMemTableFactoryHandle());
    return this;
  }

  @Override
  public Options setRateLimiterConfig(final RateLimiterConfig config) {
    rateLimiterConfig_ = config;
    setRateLimiter(nativeHandle_, config.newRateLimiterHandle());
    return this;
  }

  @Override
  public Options setLogger(final Logger logger) {
    assert(isInitialized());
    setLogger(nativeHandle_, logger.nativeHandle_);
    return this;
  }

  @Override
  public Options setInfoLogLevel(final InfoLogLevel infoLogLevel) {
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
  public String memTableFactoryName() {
    assert(isInitialized());
    return memTableFactoryName(nativeHandle_);
  }

  @Override
  public Options setTableFormatConfig(final TableFormatConfig config) {
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
  public Options useFixedLengthPrefixExtractor(final int n) {
    assert(isInitialized());
    useFixedLengthPrefixExtractor(nativeHandle_, n);
    return this;
  }

  @Override
  public Options useCappedPrefixExtractor(final int n) {
    assert(isInitialized());
    useCappedPrefixExtractor(nativeHandle_, n);
    return this;
  }

  @Override
  public CompressionType compressionType() {
    return CompressionType.values()[compressionType(nativeHandle_)];
  }

  @Override
  public Options setCompressionPerLevel(final List<CompressionType> compressionLevels) {
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
  public Options setCompressionType(CompressionType compressionType) {
    setCompressionType(nativeHandle_, compressionType.getValue());
    return this;
  }

  @Override
  public CompactionStyle compactionStyle() {
    return CompactionStyle.values()[compactionStyle(nativeHandle_)];
  }

  @Override
  public Options setCompactionStyle(
      final CompactionStyle compactionStyle) {
    setCompactionStyle(nativeHandle_, compactionStyle.getValue());
    return this;
  }

  @Override
  public int numLevels() {
    return numLevels(nativeHandle_);
  }

  @Override
  public Options setNumLevels(int numLevels) {
    setNumLevels(nativeHandle_, numLevels);
    return this;
  }

  @Override
  public int levelZeroFileNumCompactionTrigger() {
    return levelZeroFileNumCompactionTrigger(nativeHandle_);
  }

  @Override
  public Options setLevelZeroFileNumCompactionTrigger(
      final int numFiles) {
    setLevelZeroFileNumCompactionTrigger(
        nativeHandle_, numFiles);
    return this;
  }

  @Override
  public int levelZeroSlowdownWritesTrigger() {
    return levelZeroSlowdownWritesTrigger(nativeHandle_);
  }

  @Override
  public Options setLevelZeroSlowdownWritesTrigger(
      final int numFiles) {
    setLevelZeroSlowdownWritesTrigger(nativeHandle_, numFiles);
    return this;
  }

  @Override
  public int levelZeroStopWritesTrigger() {
    return levelZeroStopWritesTrigger(nativeHandle_);
  }

  @Override
  public Options setLevelZeroStopWritesTrigger(
      final int numFiles) {
    setLevelZeroStopWritesTrigger(nativeHandle_, numFiles);
    return this;
  }

  @Override
  public int maxMemCompactionLevel() {
    return 0;
  }

  @Override
  public Options setMaxMemCompactionLevel(
      final int maxMemCompactionLevel) {
    return this;
  }

  @Override
  public long targetFileSizeBase() {
    return targetFileSizeBase(nativeHandle_);
  }

  @Override
  public Options setTargetFileSizeBase(long targetFileSizeBase) {
    setTargetFileSizeBase(nativeHandle_, targetFileSizeBase);
    return this;
  }

  @Override
  public int targetFileSizeMultiplier() {
    return targetFileSizeMultiplier(nativeHandle_);
  }

  @Override
  public Options setTargetFileSizeMultiplier(int multiplier) {
    setTargetFileSizeMultiplier(nativeHandle_, multiplier);
    return this;
  }

  @Override
  public Options setMaxBytesForLevelBase(final long maxBytesForLevelBase) {
    setMaxBytesForLevelBase(nativeHandle_, maxBytesForLevelBase);
    return this;
  }

  @Override
  public long maxBytesForLevelBase() {
    return maxBytesForLevelBase(nativeHandle_);
  }

  @Override
  public Options setLevelCompactionDynamicLevelBytes(
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
  public int maxBytesForLevelMultiplier() {
    return maxBytesForLevelMultiplier(nativeHandle_);
  }

  @Override
  public Options setMaxBytesForLevelMultiplier(final int multiplier) {
    setMaxBytesForLevelMultiplier(nativeHandle_, multiplier);
    return this;
  }

  @Override
  public int expandedCompactionFactor() {
    return expandedCompactionFactor(nativeHandle_);
  }

  @Override
  public Options setExpandedCompactionFactor(
      final int expandedCompactionFactor) {
    setExpandedCompactionFactor(nativeHandle_, expandedCompactionFactor);
    return this;
  }

  @Override
  public int sourceCompactionFactor() {
    return sourceCompactionFactor(nativeHandle_);
  }

  @Override
  public Options setSourceCompactionFactor(int sourceCompactionFactor) {
    setSourceCompactionFactor(nativeHandle_, sourceCompactionFactor);
    return this;
  }

  @Override
  public int maxGrandparentOverlapFactor() {
    return maxGrandparentOverlapFactor(nativeHandle_);
  }

  @Override
  public Options setMaxGrandparentOverlapFactor(
      final int maxGrandparentOverlapFactor) {
    setMaxGrandparentOverlapFactor(nativeHandle_, maxGrandparentOverlapFactor);
    return this;
  }

  @Override
  public double softRateLimit() {
    return softRateLimit(nativeHandle_);
  }

  @Override
  public Options setSoftRateLimit(final double softRateLimit) {
    setSoftRateLimit(nativeHandle_, softRateLimit);
    return this;
  }

  @Override
  public double hardRateLimit() {
    return hardRateLimit(nativeHandle_);
  }

  @Override
  public Options setHardRateLimit(double hardRateLimit) {
    setHardRateLimit(nativeHandle_, hardRateLimit);
    return this;
  }

  @Override
  public int rateLimitDelayMaxMilliseconds() {
    return rateLimitDelayMaxMilliseconds(nativeHandle_);
  }

  @Override
  public Options setRateLimitDelayMaxMilliseconds(
      final int rateLimitDelayMaxMilliseconds) {
    setRateLimitDelayMaxMilliseconds(
        nativeHandle_, rateLimitDelayMaxMilliseconds);
    return this;
  }

  @Override
  public long arenaBlockSize() {
    return arenaBlockSize(nativeHandle_);
  }

  @Override
  public Options setArenaBlockSize(final long arenaBlockSize) {
    setArenaBlockSize(nativeHandle_, arenaBlockSize);
    return this;
  }

  @Override
  public boolean disableAutoCompactions() {
    return disableAutoCompactions(nativeHandle_);
  }

  @Override
  public Options setDisableAutoCompactions(
      final boolean disableAutoCompactions) {
    setDisableAutoCompactions(nativeHandle_, disableAutoCompactions);
    return this;
  }

  @Override
  public boolean purgeRedundantKvsWhileFlush() {
    return purgeRedundantKvsWhileFlush(nativeHandle_);
  }

  @Override
  public Options setPurgeRedundantKvsWhileFlush(
      final boolean purgeRedundantKvsWhileFlush) {
    setPurgeRedundantKvsWhileFlush(
        nativeHandle_, purgeRedundantKvsWhileFlush);
    return this;
  }

  @Override
  public boolean verifyChecksumsInCompaction() {
    return verifyChecksumsInCompaction(nativeHandle_);
  }

  @Override
  public Options setVerifyChecksumsInCompaction(
      final boolean verifyChecksumsInCompaction) {
    setVerifyChecksumsInCompaction(
        nativeHandle_, verifyChecksumsInCompaction);
    return this;
  }

  @Override
  public boolean filterDeletes() {
    return filterDeletes(nativeHandle_);
  }

  @Override
  public Options setFilterDeletes(
      final boolean filterDeletes) {
    setFilterDeletes(nativeHandle_, filterDeletes);
    return this;
  }

  @Override
  public long maxSequentialSkipInIterations() {
    return maxSequentialSkipInIterations(nativeHandle_);
  }

  @Override
  public Options setMaxSequentialSkipInIterations(
      final long maxSequentialSkipInIterations) {
    setMaxSequentialSkipInIterations(nativeHandle_, maxSequentialSkipInIterations);
    return this;
  }

  @Override
  public boolean inplaceUpdateSupport() {
    return inplaceUpdateSupport(nativeHandle_);
  }

  @Override
  public Options setInplaceUpdateSupport(
      final boolean inplaceUpdateSupport) {
    setInplaceUpdateSupport(nativeHandle_, inplaceUpdateSupport);
    return this;
  }

  @Override
  public long inplaceUpdateNumLocks() {
    return inplaceUpdateNumLocks(nativeHandle_);
  }

  @Override
  public Options setInplaceUpdateNumLocks(
      final long inplaceUpdateNumLocks) {
    setInplaceUpdateNumLocks(nativeHandle_, inplaceUpdateNumLocks);
    return this;
  }

  @Override
  public int memtablePrefixBloomBits() {
    return memtablePrefixBloomBits(nativeHandle_);
  }

  @Override
  public Options setMemtablePrefixBloomBits(
      final int memtablePrefixBloomBits) {
    setMemtablePrefixBloomBits(nativeHandle_, memtablePrefixBloomBits);
    return this;
  }

  @Override
  public int memtablePrefixBloomProbes() {
    return memtablePrefixBloomProbes(nativeHandle_);
  }

  @Override
  public Options setMemtablePrefixBloomProbes(
      final int memtablePrefixBloomProbes) {
    setMemtablePrefixBloomProbes(nativeHandle_, memtablePrefixBloomProbes);
    return this;
  }

  @Override
  public int bloomLocality() {
    return bloomLocality(nativeHandle_);
  }

  @Override
  public Options setBloomLocality(final int bloomLocality) {
    setBloomLocality(nativeHandle_, bloomLocality);
    return this;
  }

  @Override
  public long maxSuccessiveMerges() {
    return maxSuccessiveMerges(nativeHandle_);
  }

  @Override
  public Options setMaxSuccessiveMerges(long maxSuccessiveMerges) {
    setMaxSuccessiveMerges(nativeHandle_, maxSuccessiveMerges);
    return this;
  }

  @Override
  public int minWriteBufferNumberToMerge() {
    return minWriteBufferNumberToMerge(nativeHandle_);
  }

  @Override
  public Options setMinWriteBufferNumberToMerge(
      final int minWriteBufferNumberToMerge) {
    setMinWriteBufferNumberToMerge(nativeHandle_, minWriteBufferNumberToMerge);
    return this;
  }

  @Override
  public int minPartialMergeOperands() {
    return minPartialMergeOperands(nativeHandle_);
  }

  @Override
  public Options setMinPartialMergeOperands(
      final int minPartialMergeOperands) {
    setMinPartialMergeOperands(nativeHandle_, minPartialMergeOperands);
    return this;
  }

  @Override
  public Options setOptimizeFiltersForHits(
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

  private native void newOptions();
  private native void newOptions(long dbOptHandle,
      long cfOptHandle);
  private native void disposeInternal(long handle);
  private native void setEnv(long optHandle, long envHandle);
  private native void prepareForBulkLoad(long handle);

  // DB native handles
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
  private native void setMaxTableFilesSizeFIFO(
      long handle, long maxTableFilesSize);
  private native long maxTableFilesSizeFIFO(long handle);
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
  // CF native handles
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
      long handle, long arenaBlockSize) throws IllegalArgumentException;
  private native long arenaBlockSize(long handle);
  private native void setDisableAutoCompactions(
      long handle, boolean disableAutoCompactions);
  private native boolean disableAutoCompactions(long handle);
  private native void setCompactionStyle(long handle, byte compactionStyle);
  private native byte compactionStyle(long handle);
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
  // instance variables
  Env env_;
  MemTableConfig memTableConfig_;
  TableFormatConfig tableFormatConfig_;
  RateLimiterConfig rateLimiterConfig_;
  AbstractComparator<? extends AbstractSlice<?>> comparator_;
}
