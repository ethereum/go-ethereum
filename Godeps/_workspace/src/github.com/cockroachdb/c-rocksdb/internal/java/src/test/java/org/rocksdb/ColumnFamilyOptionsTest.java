// Copyright (c) 2014, Facebook, Inc.  All rights reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.

package org.rocksdb;

import org.junit.ClassRule;
import org.junit.Test;

import java.util.ArrayList;
import java.util.List;
import java.util.Properties;
import java.util.Random;

import static org.assertj.core.api.Assertions.assertThat;

public class ColumnFamilyOptionsTest {

  @ClassRule
  public static final RocksMemoryResource rocksMemoryResource =
      new RocksMemoryResource();

  public static final Random rand = PlatformRandomHelper.
      getPlatformSpecificRandomFactory();

  @Test
  public void getColumnFamilyOptionsFromProps() {
    ColumnFamilyOptions opt = null;
    try {
      // setup sample properties
      Properties properties = new Properties();
      properties.put("write_buffer_size", "112");
      properties.put("max_write_buffer_number", "13");
      opt = ColumnFamilyOptions.
          getColumnFamilyOptionsFromProps(properties);
      assertThat(opt).isNotNull();
      assertThat(String.valueOf(opt.writeBufferSize())).
          isEqualTo(properties.get("write_buffer_size"));
      assertThat(String.valueOf(opt.maxWriteBufferNumber())).
          isEqualTo(properties.get("max_write_buffer_number"));
    } finally {
      if (opt != null) {
        opt.dispose();
      }
    }
  }

  @Test
  public void failColumnFamilyOptionsFromPropsWithIllegalValue() {
    ColumnFamilyOptions opt = null;
    try {
      // setup sample properties
      Properties properties = new Properties();
      properties.put("tomato", "1024");
      properties.put("burger", "2");
      opt = ColumnFamilyOptions.
          getColumnFamilyOptionsFromProps(properties);
      assertThat(opt).isNull();
    } finally {
      if (opt != null) {
        opt.dispose();
      }
    }
  }

  @Test(expected = IllegalArgumentException.class)
  public void failColumnFamilyOptionsFromPropsWithNullValue() {
    ColumnFamilyOptions.getColumnFamilyOptionsFromProps(null);
  }

  @Test(expected = IllegalArgumentException.class)
  public void failColumnFamilyOptionsFromPropsWithEmptyProps() {
    ColumnFamilyOptions.getColumnFamilyOptionsFromProps(
        new Properties());
  }

  @Test
  public void writeBufferSize() throws RocksDBException {
    ColumnFamilyOptions opt = null;
    try {
      opt = new ColumnFamilyOptions();
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
    ColumnFamilyOptions opt = null;
    try {
      opt = new ColumnFamilyOptions();
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
    ColumnFamilyOptions opt = null;
    try {
      opt = new ColumnFamilyOptions();
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
    ColumnFamilyOptions opt = null;
    try {
      opt = new ColumnFamilyOptions();
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
    ColumnFamilyOptions opt = null;
    try {
      opt = new ColumnFamilyOptions();
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
    ColumnFamilyOptions opt = null;
    try {
      opt = new ColumnFamilyOptions();
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
    ColumnFamilyOptions opt = null;
    try {
      opt = new ColumnFamilyOptions();
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
    ColumnFamilyOptions opt = null;
    try {
      opt = new ColumnFamilyOptions();
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
    ColumnFamilyOptions opt = null;
    try {
      opt = new ColumnFamilyOptions();
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
    ColumnFamilyOptions opt = null;
    try {
      opt = new ColumnFamilyOptions();
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
    ColumnFamilyOptions opt = null;
    try {
      opt = new ColumnFamilyOptions();
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
    ColumnFamilyOptions opt = null;
    try {
      opt = new ColumnFamilyOptions();
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
    ColumnFamilyOptions opt = null;
    try {
      opt = new ColumnFamilyOptions();
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
    ColumnFamilyOptions opt = null;
    try {
      opt = new ColumnFamilyOptions();
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
    ColumnFamilyOptions opt = null;
    try {
      opt = new ColumnFamilyOptions();
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
    ColumnFamilyOptions opt = null;
    try {
      opt = new ColumnFamilyOptions();
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
    ColumnFamilyOptions opt = null;
    try {
      opt = new ColumnFamilyOptions();
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
    ColumnFamilyOptions opt = null;
    try {
      opt = new ColumnFamilyOptions();
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
    ColumnFamilyOptions opt = null;
    try {
      opt = new ColumnFamilyOptions();
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
    ColumnFamilyOptions opt = null;
    try {
      opt = new ColumnFamilyOptions();
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
    ColumnFamilyOptions opt = null;
    try {
      opt = new ColumnFamilyOptions();
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
    ColumnFamilyOptions opt = null;
    try {
      opt = new ColumnFamilyOptions();
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
    ColumnFamilyOptions opt = null;
    try {
      opt = new ColumnFamilyOptions();
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
    ColumnFamilyOptions opt = null;
    try {
      opt = new ColumnFamilyOptions();
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
    ColumnFamilyOptions opt = null;
    try {
      opt = new ColumnFamilyOptions();
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
    ColumnFamilyOptions opt = null;
    try {
      opt = new ColumnFamilyOptions();
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
    ColumnFamilyOptions opt = null;
    try {
      opt = new ColumnFamilyOptions();
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
    ColumnFamilyOptions opt = null;
    try {
      int intValue = rand.nextInt();
      opt = new ColumnFamilyOptions();
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
    ColumnFamilyOptions opt = null;
    try {
      int intValue = rand.nextInt();
      opt = new ColumnFamilyOptions();
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
    ColumnFamilyOptions opt = null;
    try {
      long longValue = rand.nextLong();
      opt = new ColumnFamilyOptions();
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
    ColumnFamilyOptions opt = null;
    try {
      int intValue = rand.nextInt();
      opt = new ColumnFamilyOptions();
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
    ColumnFamilyOptions opt = null;
    try {
      boolean aBoolean = rand.nextBoolean();
      opt = new ColumnFamilyOptions();
      opt.setOptimizeFiltersForHits(aBoolean);
      assertThat(opt.optimizeFiltersForHits()).isEqualTo(aBoolean);
    } finally {
      if (opt != null) {
        opt.dispose();
      }
    }
  }

  @Test
  public void memTable() throws RocksDBException {
    ColumnFamilyOptions opt = null;
    try {
      opt = new ColumnFamilyOptions();
      opt.setMemTableConfig(new HashLinkedListMemTableConfig());
      assertThat(opt.memTableFactoryName()).
          isEqualTo("HashLinkedListRepFactory");
    } finally {
      if (opt != null) {
        opt.dispose();
      }
    }
  }

  @Test
  public void comparator() throws RocksDBException {
    ColumnFamilyOptions opt = null;
    try {
      opt = new ColumnFamilyOptions();
      opt.setComparator(BuiltinComparator.BYTEWISE_COMPARATOR);
    } finally {
      if (opt != null) {
        opt.dispose();
      }
    }
  }

  @Test
  public void linkageOfPrepMethods() {
    ColumnFamilyOptions options = null;
    try {
      options = new ColumnFamilyOptions();
      options.optimizeUniversalStyleCompaction();
      options.optimizeUniversalStyleCompaction(4000);
      options.optimizeLevelStyleCompaction();
      options.optimizeLevelStyleCompaction(3000);
      options.optimizeForPointLookup(10);
    } finally {
      if (options != null) {
        options.dispose();
      }
    }
  }

  @Test
  public void shouldSetTestPrefixExtractor() {
    ColumnFamilyOptions options = null;
    try {
      options = new ColumnFamilyOptions();
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
    ColumnFamilyOptions options = null;
    try {
      options = new ColumnFamilyOptions();
      options.useCappedPrefixExtractor(100);
      options.useCappedPrefixExtractor(10);
    } finally {
      if (options != null) {
        options.dispose();
      }
    }
  }

  @Test
  public void compressionTypes() {
    ColumnFamilyOptions columnFamilyOptions = null;
    try {
      columnFamilyOptions = new ColumnFamilyOptions();
      for (CompressionType compressionType :
          CompressionType.values()) {
        columnFamilyOptions.setCompressionType(compressionType);
        assertThat(columnFamilyOptions.compressionType()).
            isEqualTo(compressionType);
        assertThat(CompressionType.valueOf("NO_COMPRESSION")).
            isEqualTo(CompressionType.NO_COMPRESSION);
      }
    } finally {
      if (columnFamilyOptions != null) {
        columnFamilyOptions.dispose();
      }
    }
  }

  @Test
  public void compressionPerLevel() {
    ColumnFamilyOptions columnFamilyOptions = null;
    try {
      columnFamilyOptions = new ColumnFamilyOptions();
      assertThat(columnFamilyOptions.compressionPerLevel()).isEmpty();
      List<CompressionType> compressionTypeList = new ArrayList<>();
      for (int i=0; i < columnFamilyOptions.numLevels(); i++) {
        compressionTypeList.add(CompressionType.NO_COMPRESSION);
      }
      columnFamilyOptions.setCompressionPerLevel(compressionTypeList);
      compressionTypeList = columnFamilyOptions.compressionPerLevel();
      for (CompressionType compressionType : compressionTypeList) {
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
    ColumnFamilyOptions ColumnFamilyOptions = null;
    try {
      ColumnFamilyOptions = new ColumnFamilyOptions();
      for (CompactionStyle compactionStyle :
          CompactionStyle.values()) {
        ColumnFamilyOptions.setCompactionStyle(compactionStyle);
        assertThat(ColumnFamilyOptions.compactionStyle()).
            isEqualTo(compactionStyle);
        assertThat(CompactionStyle.valueOf("FIFO")).
            isEqualTo(CompactionStyle.FIFO);
      }
    } finally {
      if (ColumnFamilyOptions != null) {
        ColumnFamilyOptions.dispose();
      }
    }
  }

  @Test
  public void maxTableFilesSizeFIFO() {
    ColumnFamilyOptions opt = null;
    try {
      opt = new ColumnFamilyOptions();
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
}
