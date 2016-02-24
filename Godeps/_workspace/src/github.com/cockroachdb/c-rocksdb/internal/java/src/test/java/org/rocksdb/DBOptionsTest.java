// Copyright (c) 2014, Facebook, Inc.  All rights reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.

package org.rocksdb;

import org.junit.ClassRule;
import org.junit.Test;

import java.util.Properties;
import java.util.Random;

import static org.assertj.core.api.Assertions.assertThat;

public class DBOptionsTest {

  @ClassRule
  public static final RocksMemoryResource rocksMemoryResource =
      new RocksMemoryResource();

  public static final Random rand = PlatformRandomHelper.
      getPlatformSpecificRandomFactory();

  @Test
  public void getDBOptionsFromProps() {
    DBOptions opt = null;
    try {
      // setup sample properties
      Properties properties = new Properties();
      properties.put("allow_mmap_reads", "true");
      properties.put("bytes_per_sync", "13");
      opt = DBOptions.getDBOptionsFromProps(properties);
      assertThat(opt).isNotNull();
      assertThat(String.valueOf(opt.allowMmapReads())).
          isEqualTo(properties.get("allow_mmap_reads"));
      assertThat(String.valueOf(opt.bytesPerSync())).
          isEqualTo(properties.get("bytes_per_sync"));
    } finally {
      if (opt != null) {
        opt.dispose();
      }
    }
  }

  @Test
  public void failDBOptionsFromPropsWithIllegalValue() {
    DBOptions opt = null;
    try {
      // setup sample properties
      Properties properties = new Properties();
      properties.put("tomato", "1024");
      properties.put("burger", "2");
      opt = DBOptions.
          getDBOptionsFromProps(properties);
      assertThat(opt).isNull();
    } finally {
      if (opt != null) {
        opt.dispose();
      }
    }
  }

  @Test(expected = IllegalArgumentException.class)
  public void failDBOptionsFromPropsWithNullValue() {
    DBOptions.getDBOptionsFromProps(null);
  }

  @Test(expected = IllegalArgumentException.class)
  public void failDBOptionsFromPropsWithEmptyProps() {
    DBOptions.getDBOptionsFromProps(
        new Properties());
  }

  @Test
  public void setIncreaseParallelism() {
    DBOptions opt = null;
    try {
      opt = new DBOptions();
      final int threads = Runtime.getRuntime().availableProcessors() * 2;
      opt.setIncreaseParallelism(threads);
    } finally {
      if (opt != null) {
        opt.dispose();
      }
    }
  }

  @Test
  public void createIfMissing() {
    DBOptions opt = null;
    try {
      opt = new DBOptions();
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
    DBOptions opt = null;
    try {
      opt = new DBOptions();
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
    DBOptions opt = null;
    try {
      opt = new DBOptions();
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
    DBOptions opt = null;
    try {
      opt = new DBOptions();
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
    DBOptions opt = null;
    try {
      opt = new DBOptions();
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
    DBOptions opt = null;
    try {
      opt = new DBOptions();
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
    DBOptions opt = null;
    try {
      opt = new DBOptions();
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
    DBOptions opt = null;
    try {
      opt = new DBOptions();
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
    DBOptions opt = null;
    try {
      opt = new DBOptions();
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
    DBOptions opt = null;
    try {
      opt = new DBOptions();
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
    DBOptions opt = null;
    try {
      opt = new DBOptions();
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
    DBOptions opt = null;
    try {
      opt = new DBOptions();
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
    DBOptions opt = null;
    try {
      opt = new DBOptions();
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
    DBOptions opt = null;
    try {
      opt = new DBOptions();
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
    DBOptions opt = null;
    try {
      opt = new DBOptions();
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
    DBOptions opt = null;
    try {
      opt = new DBOptions();
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
    DBOptions opt = null;
    try {
      opt = new DBOptions();
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
    DBOptions opt = null;
    try {
      opt = new DBOptions();
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
    DBOptions opt = null;
    try {
      opt = new DBOptions();
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
    DBOptions opt = null;
    try {
      opt = new DBOptions();
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
    DBOptions opt = null;
    try {
      opt = new DBOptions();
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
    DBOptions opt = null;
    try {
      opt = new DBOptions();
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
    DBOptions opt = null;
    try {
      opt = new DBOptions();
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
    DBOptions opt = null;
    try {
      opt = new DBOptions();
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
    DBOptions opt = null;
    try {
      opt = new DBOptions();
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
    DBOptions opt = null;
    try {
      opt = new DBOptions();
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
    DBOptions opt = null;
    try {
      opt = new DBOptions();
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
    DBOptions opt = null;
    try {
      opt = new DBOptions();
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
    DBOptions opt = null;
    try {
      opt = new DBOptions();
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
  public void rateLimiterConfig() {
    DBOptions options = null;
    DBOptions anotherOptions = null;
    try {
      options = new DBOptions();
      RateLimiterConfig rateLimiterConfig =
          new GenericRateLimiterConfig(1000, 100 * 1000, 1);
      options.setRateLimiterConfig(rateLimiterConfig);
      // Test with parameter initialization
      anotherOptions = new DBOptions();
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
  public void statistics() {
    DBOptions options = new DBOptions();
    Statistics statistics = options.createStatistics().
        statisticsPtr();
    assertThat(statistics).isNotNull();

    DBOptions anotherOptions = new DBOptions();
    statistics = anotherOptions.statisticsPtr();
    assertThat(statistics).isNotNull();
  }
}
