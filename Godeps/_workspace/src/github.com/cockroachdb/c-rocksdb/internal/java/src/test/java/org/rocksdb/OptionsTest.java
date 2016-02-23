// Copyright (c) 2014, Facebook, Inc.  All rights reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.

package org.rocksdb;

import java.util.ArrayList;
import java.util.List;
import java.util.Random;
import org.junit.ClassRule;
import org.junit.Test;

import static org.assertj.core.api.Assertions.assertThat;


public class OptionsTest {

  @ClassRule
  public static final RocksMemoryResource rocksMemoryResource =
      new RocksMemoryResource();

  public static final Random rand = PlatformRandomHelper.
      getPlatformSpecificRandomFactory();

  @Test
  public void setIncreaseParallelism() {
    Options opt = null;
    try {
      opt = new Options();
      final int threads = Runtime.getRuntime().availableProcessors() * 2;
      opt.setIncreaseParallelism(threads);
    } finally {
      if (opt != null) {
        opt.dispose();
      }
    }
  }

  @Test
  public void writeBufferSize() throws RocksDBException {
    Options opt = null;
    try {
      opt = new Options();
      long longValue = rand.nextLong();
      opt.setWriteBufferSize(longValue);
      assertThat(opt.writeBufferSize()).isEqualTo(longValue);
    } finally {
      if (opt != null) {
        opt.dispose();
      }
    }
  }

  @Test
  public void maxWriteBufferNumber() {
    Options opt = null;
    try {
      opt = new Options();
      int intValue = rand.nextInt();
      opt.setMaxWriteBufferNumber(intValue);
      assertThat(opt.maxWriteBufferNumber()).isEqualTo(intValue);
    } finally {
      if (opt != null) {
        opt.dispose();
      }
    }
  }

  @Test
  public void minWriteBufferNumberToMerge() {
    Options opt = null;
    try {
      opt = new Options();
      int intValue = rand.nextInt();
      opt.setMinWriteBufferNumberToMerge(intValue);
      assertThat(opt.minWriteBufferNumberToMerge()).isEqualTo(intValue);
    } finally {
      if (opt != null) {
        opt.dispose();
      }
    }
  }

  @Test
  public void numLevels() {
    Options opt = null;
    try {
      opt = new Options();
      int intValue = rand.nextInt();
      opt.setNumLevels(intValue);
      assertThat(opt.numLevels()).isEqualTo(intValue);
    } finally {
      if (opt != null) {
        opt.dispose();
      }
    }
  }

  @Test
  public void levelZeroFileNumCompactionTrigger() {
    Options opt = null;
    try {
      opt = new Options();
      int intValue = rand.nextInt();
      opt.setLevelZeroFileNumCompactionTrigger(intValue);
      assertThat(opt.levelZeroFileNumCompactionTrigger()).isEqualTo(intValue);
    } finally {
      if (opt != null) {
        opt.dispose();
      }
    }
  }

  @Test
  public void levelZeroSlowdownWritesTrigger() {
    Options opt = null;
    try {
      opt = new Options();
      int intValue = rand.nextInt();
      opt.setLevelZeroSlowdownWritesTrigger(intValue);
      assertThat(opt.levelZeroSlowdownWritesTrigger()).isEqualTo(intValue);
    } finally {
      if (opt != null) {
        opt.dispose();
      }
    }
  }

  @Test
  public void levelZeroStopWritesTrigger() {
    Options opt = null;
    try {
      opt = new Options();
      int intValue = rand.nextInt();
      opt.setLevelZeroStopWritesTrigger(intValue);
      assertThat(opt.levelZeroStopWritesTrigger()).isEqualTo(intValue);
    } finally {
      if (opt != null) {
        opt.dispose();
      }
    }
  }

  @Test
  public void targetFileSizeBase() {
    Options opt = null;
    try {
      opt = new Options();
      long longValue = rand.nextLong();
      opt.setTargetFileSizeBase(longValue);
      assertThat(opt.targetFileSizeBase()).isEqualTo(longValue);
    } finally {
      if (opt != null) {
        opt.dispose();
      }
    }
  }

  @Test
  public void targetFileSizeMultiplier() {
    Options opt = null;
    try {
      opt = new Options();
      int intValue = rand.nextInt();
      opt.setTargetFileSizeMultiplier(intValue);
      assertThat(opt.targetFileSizeMultiplier()).isEqualTo(intValue);
    } finally {
      if (opt != null) {
        opt.dispose();
      }
    }
  }

  @Test
  public void maxBytesForLevelBase() {
    Options opt = null;
    try {
      opt = new Options();
      long longValue = rand.nextLong();
      opt.setMaxBytesForLevelBase(longValue);
      assertThat(opt.maxBytesForLevelBase()).isEqualTo(longValue);
    } finally {
      if (opt != null) {
        opt.dispose();
      }
    }
  }

  @Test
  public void levelCompactionDynamicLevelBytes() {
    Options opt = null;
    try {
      opt = new Options();
      final boolean boolValue = rand.nextBoolean();
      opt.setLevelCompactionDynamicLevelBytes(boolValue);
      assertThat(opt.levelCompactionDynamicLevelBytes())
          .isEqualTo(boolValue);
    } finally {
      if (opt != null) {
        opt.dispose();
      }
    }
  }

  @Test
  public void maxBytesForLevelMultiplier() {
    Options opt = null;
    try {
      opt = new Options();
      int intValue = rand.nextInt();
      opt.setMaxBytesForLevelMultiplier(intValue);
      assertThat(opt.maxBytesForLevelMultiplier()).isEqualTo(intValue);
    } finally {
      if (opt != null) {
        opt.dispose();
      }
    }
  }

  @Test
  public void expandedCompactionFactor() {
    Options opt = null;
    try {
      opt = new Options();
      int intValue = rand.nextInt();
      opt.setExpandedCompactionFactor(intValue);
      assertThat(opt.expandedCompactionFactor()).isEqualTo(intValue);
    } finally {
      if (opt != null) {
        opt.dispose();
      }
    }
  }

  @Test
  public void sourceCompactionFactor() {
    Options opt = null;
    try {
      opt = new Options();
      int intValue = rand.nextInt();
      opt.setSourceCompactionFactor(intValue);
      assertThat(opt.sourceCompactionFactor()).isEqualTo(intValue);
    } finally {
      if (opt != null) {
        opt.dispose();
      }
    }
  }

  @Test
  public void maxGrandparentOverlapFactor() {
    Options opt = null;
    try {
      opt = new Options();
      int intValue = rand.nextInt();
      opt.setMaxGrandparentOverlapFactor(intValue);
      assertThat(opt.maxGrandparentOverlapFactor()).isEqualTo(intValue);
    } finally {
      if (opt != null) {
        opt.dispose();
      }
    }
  }

  @Test
  public void softRateLimit() {
    Options opt = null;
    try {
      opt = new Options();
      double doubleValue = rand.nextDouble();
      opt.setSoftRateLimit(doubleValue);
      assertThat(opt.softRateLimit()).isEqualTo(doubleValue);
    } finally {
      if (opt != null) {
        opt.dispose();
      }
    }
  }

  @Test
  public void hardRateLimit() {
    Options opt = null;
    try {
      opt = new Options();
      double doubleValue = rand.nextDouble();
      opt.setHardRateLimit(doubleValue);
      assertThat(opt.hardRateLimit()).isEqualTo(doubleValue);
    } finally {
      if (opt != null) {
        opt.dispose();
      }
    }
  }

  @Test
  public void rateLimitDelayMaxMilliseconds() {
    Options opt = null;
    try {
      opt = new Options();
      int intValue = rand.nextInt();
      opt.setRateLimitDelayMaxMilliseconds(intValue);
      assertThat(opt.rateLimitDelayMaxMilliseconds()).isEqualTo(intValue);
    } finally {
      if (opt != null) {
        opt.dispose();
      }
    }
  }

  @Test
  public void arenaBlockSize() throws RocksDBException {
    Options opt = null;
    try {
      opt = new Options();
      long longValue = rand.nextLong();
      opt.setArenaBlockSize(longValue);
      assertThat(opt.arenaBlockSize()).isEqualTo(longValue);
    } finally {
      if (opt != null) {
        opt.dispose();
      }
    }
  }

  @Test
  public void disableAutoCompactions() {
    Options opt = null;
    try {
      opt = new Options();
      boolean boolValue = rand.nextBoolean();
      opt.setDisableAutoCompactions(boolValue);
      assertThat(opt.disableAutoCompactions()).isEqualTo(boolValue);
    } finally {
      if (opt != null) {
        opt.dispose();
      }
    }
  }

  @Test
  public void purgeRedundantKvsWhileFlush() {
    Options opt = null;
    try {
      opt = new Options();
      boolean boolValue = rand.nextBoolean();
      opt.setPurgeRedundantKvsWhileFlush(boolValue);
      assertThat(opt.purgeRedundantKvsWhileFlush()).isEqualTo(boolValue);
    } finally {
      if (opt != null) {
        opt.dispose();
      }
    }
  }

  @Test
  public void verifyChecksumsInCompaction() {
    Options opt = null;
    try {
      opt = new Options();
      boolean boolValue = rand.nextBoolean();
      opt.setVerifyChecksumsInCompaction(boolValue);
      assertThat(opt.verifyChecksumsInCompaction()).isEqualTo(boolValue);
    } finally {
      if (opt != null) {
        opt.dispose();
      }
    }
  }

  @Test
  public void filterDeletes() {
    Options opt = null;
    try {
      opt = new Options();
      boolean boolValue = rand.nextBoolean();
      opt.setFilterDeletes(boolValue);
      assertThat(opt.filterDeletes()).isEqualTo(boolValue);
    } finally {
      if (opt != null) {
        opt.dispose();
      }
    }
  }

  @Test
  public void maxSequentialSkipInIterations() {
    Options opt = null;
    try {
      opt = new Options();
      long longValue = rand.nextLong();
      opt.setMaxSequentialSkipInIterations(longValue);
      assertThat(opt.maxSequentialSkipInIterations()).isEqualTo(longValue);
    } finally {
      if (opt != null) {
        opt.dispose();
      }
    }
  }

  @Test
  public void inplaceUpdateSupport() {
    Options opt = null;
    try {
      opt = new Options();
      boolean boolValue = rand.nextBoolean();
      opt.setInplaceUpdateSupport(boolValue);
      assertThat(opt.inplaceUpdateSupport()).isEqualTo(boolValue);
    } finally {
      if (opt != null) {
        opt.dispose();
      }
    }
  }

  @Test
  public void inplaceUpdateNumLocks() throws RocksDBException {
    Options opt = null;
    try {
      opt = new Options();
      long longValue = rand.nextLong();
      opt.setInplaceUpdateNumLocks(longValue);
      assertThat(opt.inplaceUpdateNumLocks()).isEqualTo(longValue);
    } finally {
      if (opt != null) {
        opt.dispose();
      }
    }
  }

  @Test
  public void memtablePrefixBloomBits() {
    Options opt = null;
    try {
      opt = new Options();
      int intValue = rand.nextInt();
      opt.setMemtablePrefixBloomBits(intValue);
      assertThat(opt.memtablePrefixBloomBits()).isEqualTo(intValue);
    } finally {
      if (opt != null) {
        opt.dispose();
      }
    }
  }

  @Test
  public void memtablePrefixBloomProbes() {
    Options opt = null;
    try {
      int intValue = rand.nextInt();
      opt = new Options();
      opt.setMemtablePrefixBloomProbes(intValue);
      assertThat(opt.memtablePrefixBloomProbes()).isEqualTo(intValue);
    } finally {
      if (opt != null) {
        opt.dispose();
      }
    }
  }

  @Test
  public void bloomLocality() {
    Options opt = null;
    try {
      int intValue = rand.nextInt();
      opt = new Options();
      opt.setBloomLocality(intValue);
      assertThat(opt.bloomLocality()).isEqualTo(intValue);
    } finally {
      if (opt != null) {
        opt.dispose();
      }
    }
  }

  @Test
  public void maxSuccessiveMerges() throws RocksDBException {
    Options opt = null;
    try {
      long longValue = rand.nextLong();
      opt = new Options();
      opt.setMaxSuccessiveMerges(longValue);
      assertThat(opt.maxSuccessiveMerges()).isEqualTo(longValue);
    } finally {
      if (opt != null) {
        opt.dispose();
      }
    }
  }

  @Test
  public void minPartialMergeOperands() {
    Options opt = null;
    try {
      int intValue = rand.nextInt();
      opt = new Options();
      opt.setMinPartialMergeOperands(intValue);
      assertThat(opt.minPartialMergeOperands()).isEqualTo(intValue);
    } finally {
      if (opt != null) {
        opt.dispose();
      }
    }
  }

  @Test
  public void optimizeFiltersForHits() {
    Options opt = null;
    try {
      boolean aBoolean = rand.nextBoolean();
      opt = new Options();
      opt.setOptimizeFiltersForHits(aBoolean);
      assertThat(opt.optimizeFiltersForHits()).isEqualTo(aBoolean);
    } finally {
      if (opt != null) {
        opt.dispose();
      }
    }
  }

  @Test
  public void createIfMissing() {
    Options opt = null;
    try {
      opt = new Options();
      boolean boolValue = rand.nextBoolean();
      opt.setCreateIfMissing(boolValue);
      assertThat(opt.createIfMissing()).
          isEqualTo(boolValue);
    } finally {
      if (opt != null) {
        opt.dispose();
      }
    }
  }

  @Test
  public void createMissingColumnFamilies() {
    Options opt = null;
    try {
      opt = new Options();
      boolean boolValue = rand.nextBoolean();
      opt.setCreateMissingColumnFamilies(boolValue);
      assertThat(opt.createMissingColumnFamilies()).
          isEqualTo(boolValue);
    } finally {
      if (opt != null) {
        opt.dispose();
      }
    }
  }

  @Test
  public void errorIfExists() {
    Options opt = null;
    try {
      opt = new Options();
      boolean boolValue = rand.nextBoolean();
      opt.setErrorIfExists(boolValue);
      assertThat(opt.errorIfExists()).isEqualTo(boolValue);
    } finally {
      if (opt != null) {
        opt.dispose();
      }
    }
  }

  @Test
  public void paranoidChecks() {
    Options opt = null;
    try {
      opt = new Options();
      boolean boolValue = rand.nextBoolean();
      opt.setParanoidChecks(boolValue);
      assertThat(opt.paranoidChecks()).
          isEqualTo(boolValue);
    } finally {
      if (opt != null) {
        opt.dispose();
      }
    }
  }

  @Test
  public void maxTotalWalSize() {
    Options opt = null;
    try {
      opt = new Options();
      long longValue = rand.nextLong();
      opt.setMaxTotalWalSize(longValue);
      assertThat(opt.maxTotalWalSize()).
          isEqualTo(longValue);
    } finally {
      if (opt != null) {
        opt.dispose();
      }
    }
  }

  @Test
  public void maxOpenFiles() {
    Options opt = null;
    try {
      opt = new Options();
      int intValue = rand.nextInt();
      opt.setMaxOpenFiles(intValue);
      assertThat(opt.maxOpenFiles()).isEqualTo(intValue);
    } finally {
      if (opt != null) {
        opt.dispose();
      }
    }
  }

  @Test
  public void disableDataSync() {
    Options opt = null;
    try {
      opt = new Options();
      boolean boolValue = rand.nextBoolean();
      opt.setDisableDataSync(boolValue);
      assertThat(opt.disableDataSync()).
          isEqualTo(boolValue);
    } finally {
      if (opt != null) {
        opt.dispose();
      }
    }
  }

  @Test
  public void useFsync() {
    Options opt = null;
    try {
      opt = new Options();
      boolean boolValue = rand.nextBoolean();
      opt.setUseFsync(boolValue);
      assertThat(opt.useFsync()).isEqualTo(boolValue);
    } finally {
      if (opt != null) {
        opt.dispose();
      }
    }
  }

  @Test
  public void dbLogDir() {
    Options opt = null;
    try {
      opt = new Options();
      String str = "path/to/DbLogDir";
      opt.setDbLogDir(str);
      assertThat(opt.dbLogDir()).isEqualTo(str);
    } finally {
      if (opt != null) {
        opt.dispose();
      }
    }
  }

  @Test
  public void walDir() {
    Options opt = null;
    try {
      opt = new Options();
      String str = "path/to/WalDir";
      opt.setWalDir(str);
      assertThat(opt.walDir()).isEqualTo(str);
    } finally {
      if (opt != null) {
        opt.dispose();
      }
    }
  }

  @Test
  public void deleteObsoleteFilesPeriodMicros() {
    Options opt = null;
    try {
      opt = new Options();
      long longValue = rand.nextLong();
      opt.setDeleteObsoleteFilesPeriodMicros(longValue);
      assertThat(opt.deleteObsoleteFilesPeriodMicros()).
          isEqualTo(longValue);
    } finally {
      if (opt != null) {
        opt.dispose();
      }
    }
  }

  @Test
  public void maxBackgroundCompactions() {
    Options opt = null;
    try {
      opt = new Options();
      int intValue = rand.nextInt();
      opt.setMaxBackgroundCompactions(intValue);
      assertThat(opt.maxBackgroundCompactions()).
          isEqualTo(intValue);
    } finally {
      if (opt != null) {
        opt.dispose();
      }
    }
  }

  @Test
  public void maxBackgroundFlushes() {
    Options opt = null;
    try {
      opt = new Options();
      int intValue = rand.nextInt();
      opt.setMaxBackgroundFlushes(intValue);
      assertThat(opt.maxBackgroundFlushes()).
          isEqualTo(intValue);
    } finally {
      if (opt != null) {
        opt.dispose();
      }
    }
  }

  @Test
  public void maxLogFileSize() throws RocksDBException {
    Options opt = null;
    try {
      opt = new Options();
      long longValue = rand.nextLong();
      opt.setMaxLogFileSize(longValue);
      assertThat(opt.maxLogFileSize()).isEqualTo(longValue);
    } finally {
      if (opt != null) {
        opt.dispose();
      }
    }
  }

  @Test
  public void logFileTimeToRoll() throws RocksDBException {
    Options opt = null;
    try {
      opt = new Options();
      long longValue = rand.nextLong();
      opt.setLogFileTimeToRoll(longValue);
      assertThat(opt.logFileTimeToRoll()).
          isEqualTo(longValue);
    } finally {
      if (opt != null) {
        opt.dispose();
      }
    }
  }

  @Test
  public void keepLogFileNum() throws RocksDBException {
    Options opt = null;
    try {
      opt = new Options();
      long longValue = rand.nextLong();
      opt.setKeepLogFileNum(longValue);
      assertThat(opt.keepLogFileNum()).isEqualTo(longValue);
    } finally {
      if (opt != null) {
        opt.dispose();
      }
    }
  }

  @Test
  public void maxManifestFileSize() {
    Options opt = null;
    try {
      opt = new Options();
      long longValue = rand.nextLong();
      opt.setMaxManifestFileSize(longValue);
      assertThat(opt.maxManifestFileSize()).
          isEqualTo(longValue);
    } finally {
      if (opt != null) {
        opt.dispose();
      }
    }
  }

  @Test
  public void tableCacheNumshardbits() {
    Options opt = null;
    try {
      opt = new Options();
      int intValue = rand.nextInt();
      opt.setTableCacheNumshardbits(intValue);
      assertThat(opt.tableCacheNumshardbits()).
          isEqualTo(intValue);
    } finally {
      if (opt != null) {
        opt.dispose();
      }
    }
  }

  @Test
  public void walSizeLimitMB() {
    Options opt = null;
    try {
      opt = new Options();
      long longValue = rand.nextLong();
      opt.setWalSizeLimitMB(longValue);
      assertThat(opt.walSizeLimitMB()).isEqualTo(longValue);
    } finally {
      if (opt != null) {
        opt.dispose();
      }
    }
  }

  @Test
  public void walTtlSeconds() {
    Options opt = null;
    try {
      opt = new Options();
      long longValue = rand.nextLong();
      opt.setWalTtlSeconds(longValue);
      assertThat(opt.walTtlSeconds()).isEqualTo(longValue);
    } finally {
      if (opt != null) {
        opt.dispose();
      }
    }
  }

  @Test
  public void manifestPreallocationSize() throws RocksDBException {
    Options opt = null;
    try {
      opt = new Options();
      long longValue = rand.nextLong();
      opt.setManifestPreallocationSize(longValue);
      assertThat(opt.manifestPreallocationSize()).
          isEqualTo(longValue);
    } finally {
      if (opt != null) {
        opt.dispose();
      }
    }
  }

  @Test
  public void allowOsBuffer() {
    Options opt = null;
    try {
      opt = new Options();
      boolean boolValue = rand.nextBoolean();
      opt.setAllowOsBuffer(boolValue);
      assertThat(opt.allowOsBuffer()).isEqualTo(boolValue);
    } finally {
      if (opt != null) {
        opt.dispose();
      }
    }
  }

  @Test
  public void allowMmapReads() {
    Options opt = null;
    try {
      opt = new Options();
      boolean boolValue = rand.nextBoolean();
      opt.setAllowMmapReads(boolValue);
      assertThat(opt.allowMmapReads()).isEqualTo(boolValue);
    } finally {
      if (opt != null) {
        opt.dispose();
      }
    }
  }

  @Test
  public void allowMmapWrites() {
    Options opt = null;
    try {
      opt = new Options();
      boolean boolValue = rand.nextBoolean();
      opt.setAllowMmapWrites(boolValue);
      assertThat(opt.allowMmapWrites()).isEqualTo(boolValue);
    } finally {
      if (opt != null) {
        opt.dispose();
      }
    }
  }

  @Test
  public void isFdCloseOnExec() {
    Options opt = null;
    try {
      opt = new Options();
      boolean boolValue = rand.nextBoolean();
      opt.setIsFdCloseOnExec(boolValue);
      assertThat(opt.isFdCloseOnExec()).isEqualTo(boolValue);
    } finally {
      if (opt != null) {
        opt.dispose();
      }
    }
  }

  @Test
  public void statsDumpPeriodSec() {
    Options opt = null;
    try {
      opt = new Options();
      int intValue = rand.nextInt();
      opt.setStatsDumpPeriodSec(intValue);
      assertThat(opt.statsDumpPeriodSec()).isEqualTo(intValue);
    } finally {
      if (opt != null) {
        opt.dispose();
      }
    }
  }

  @Test
  public void adviseRandomOnOpen() {
    Options opt = null;
    try {
      opt = new Options();
      boolean boolValue = rand.nextBoolean();
      opt.setAdviseRandomOnOpen(boolValue);
      assertThat(opt.adviseRandomOnOpen()).isEqualTo(boolValue);
    } finally {
      if (opt != null) {
        opt.dispose();
      }
    }
  }

  @Test
  public void useAdaptiveMutex() {
    Options opt = null;
    try {
      opt = new Options();
      boolean boolValue = rand.nextBoolean();
      opt.setUseAdaptiveMutex(boolValue);
      assertThat(opt.useAdaptiveMutex()).isEqualTo(boolValue);
    } finally {
      if (opt != null) {
        opt.dispose();
      }
    }
  }

  @Test
  public void bytesPerSync() {
    Options opt = null;
    try {
      opt = new Options();
      long longValue = rand.nextLong();
      opt.setBytesPerSync(longValue);
      assertThat(opt.bytesPerSync()).isEqualTo(longValue);
    } finally {
      if (opt != null) {
        opt.dispose();
      }
    }
  }

  @Test
  public void env() {
    Options options = null;
    try {
      options = new Options();
      Env env = Env.getDefault();
      options.setEnv(env);
      assertThat(options.getEnv()).isSameAs(env);
    } finally {
      if (options != null) {
        options.dispose();
      }
    }
  }

  @Test
  public void linkageOfPrepMethods() {
    Options options = null;
    try {
      options = new Options();
      options.optimizeUniversalStyleCompaction();
      options.optimizeUniversalStyleCompaction(4000);
      options.optimizeLevelStyleCompaction();
      options.optimizeLevelStyleCompaction(3000);
      options.optimizeForPointLookup(10);
      options.prepareForBulkLoad();
    } finally {
      if (options != null) {
        options.dispose();
      }
    }
  }

  @Test
  public void compressionTypes() {
    Options options = null;
    try {
      options = new Options();
      for (CompressionType compressionType :
          CompressionType.values()) {
        options.setCompressionType(compressionType);
        assertThat(options.compressionType()).
            isEqualTo(compressionType);
        assertThat(CompressionType.valueOf("NO_COMPRESSION")).
            isEqualTo(CompressionType.NO_COMPRESSION);
      }
    } finally {
      if (options != null) {
        options.dispose();
      }
    }
  }

  @Test
  public void compressionPerLevel() {
    ColumnFamilyOptions columnFamilyOptions = null;
    try {
      columnFamilyOptions = new ColumnFamilyOptions();
      assertThat(columnFamilyOptions.compressionPerLevel()).isEmpty();
      List<CompressionType> compressionTypeList =
          new ArrayList<>();
      for (int i=0; i < columnFamilyOptions.numLevels(); i++) {
        compressionTypeList.add(CompressionType.NO_COMPRESSION);
      }
      columnFamilyOptions.setCompressionPerLevel(compressionTypeList);
      compressionTypeList = columnFamilyOptions.compressionPerLevel();
      for (final CompressionType compressionType : compressionTypeList) {
        assertThat(compressionType).isEqualTo(
            CompressionType.NO_COMPRESSION);
      }
    } finally {
      if (columnFamilyOptions != null) {
        columnFamilyOptions.dispose();
      }
    }
  }

  @Test
  public void differentCompressionsPerLevel() {
    ColumnFamilyOptions columnFamilyOptions = null;
    try {
      columnFamilyOptions = new ColumnFamilyOptions();
      columnFamilyOptions.setNumLevels(3);

      assertThat(columnFamilyOptions.compressionPerLevel()).isEmpty();
      List<CompressionType> compressionTypeList = new ArrayList<>();

      compressionTypeList.add(CompressionType.BZLIB2_COMPRESSION);
      compressionTypeList.add(CompressionType.SNAPPY_COMPRESSION);
      compressionTypeList.add(CompressionType.LZ4_COMPRESSION);

      columnFamilyOptions.setCompressionPerLevel(compressionTypeList);
      compressionTypeList = columnFamilyOptions.compressionPerLevel();

      assertThat(compressionTypeList.size()).isEqualTo(3);
      assertThat(compressionTypeList).
          containsExactly(
              CompressionType.BZLIB2_COMPRESSION,
              CompressionType.SNAPPY_COMPRESSION,
              CompressionType.LZ4_COMPRESSION);

    } finally {
      if (columnFamilyOptions != null) {
        columnFamilyOptions.dispose();
      }
    }
  }

  @Test
  public void compactionStyles() {
    Options options = null;
    try {
      options = new Options();
      for (CompactionStyle compactionStyle :
          CompactionStyle.values()) {
        options.setCompactionStyle(compactionStyle);
        assertThat(options.compactionStyle()).
            isEqualTo(compactionStyle);
        assertThat(CompactionStyle.valueOf("FIFO")).
            isEqualTo(CompactionStyle.FIFO);
      }
    } finally {
      if (options != null) {
        options.dispose();
      }
    }
  }

  @Test
  public void maxTableFilesSizeFIFO() {
    Options opt = null;
    try {
      opt = new Options();
      long longValue = rand.nextLong();
      // Size has to be positive
      longValue = (longValue < 0) ? -longValue : longValue;
      longValue = (longValue == 0) ? longValue + 1 : longValue;
      opt.setMaxTableFilesSizeFIFO(longValue);
      assertThat(opt.maxTableFilesSizeFIFO()).
          isEqualTo(longValue);
    } finally {
      if (opt != null) {
        opt.dispose();
      }
    }
  }

  @Test
  public void rateLimiterConfig() {
    Options options = null;
    Options anotherOptions = null;
    RateLimiterConfig rateLimiterConfig;
    try {
      options = new Options();
      rateLimiterConfig = new GenericRateLimiterConfig(1000, 100 * 1000, 1);
      options.setRateLimiterConfig(rateLimiterConfig);
      // Test with parameter initialization
      anotherOptions = new Options();
      anotherOptions.setRateLimiterConfig(
          new GenericRateLimiterConfig(1000));
    } finally {
      if (options != null) {
        options.dispose();
      }
      if (anotherOptions != null) {
        anotherOptions.dispose();
      }
    }
  }

  @Test
  public void shouldSetTestPrefixExtractor() {
    Options options = null;
    try {
      options = new Options();
      options.useFixedLengthPrefixExtractor(100);
      options.useFixedLengthPrefixExtractor(10);
    } finally {
      if (options != null) {
        options.dispose();
      }
    }
  }

  @Test
  public void shouldSetTestCappedPrefixExtractor() {
    Options options = null;
    try {
      options = new Options();
      options.useCappedPrefixExtractor(100);
      options.useCappedPrefixExtractor(10);
    } finally {
      if (options != null) {
        options.dispose();
      }
    }
  }


  @Test
  public void shouldTestMemTableFactoryName()
      throws RocksDBException {
    Options options = null;
    try {
      options = new Options();
      options.setMemTableConfig(new VectorMemTableConfig());
      assertThat(options.memTableFactoryName()).
          isEqualTo("VectorRepFactory");
      options.setMemTableConfig(
          new HashLinkedListMemTableConfig());
      assertThat(options.memTableFactoryName()).
          isEqualTo("HashLinkedListRepFactory");
    } finally {
      if (options != null) {
        options.dispose();
      }
    }
  }

  @Test
  public void statistics() {
    Options options = null;
    Options anotherOptions = null;
    try {
      options = new Options();
      Statistics statistics = options.createStatistics().
          statisticsPtr();
      assertThat(statistics).isNotNull();
      anotherOptions = new Options();
      statistics = anotherOptions.statisticsPtr();
      assertThat(statistics).isNotNull();
    } finally {
      if (options != null) {
        options.dispose();
      }
      if (anotherOptions != null) {
        anotherOptions.dispose();
      }
    }
  }
}
