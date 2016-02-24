// Copyright (c) 2014, Facebook, Inc.  All rights reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.

package org.rocksdb;

import java.util.Collections;

import org.junit.ClassRule;
import org.junit.Rule;
import org.junit.Test;
import org.junit.rules.TemporaryFolder;

import static org.assertj.core.api.Assertions.assertThat;

public class StatisticsCollectorTest {

  @ClassRule
  public static final RocksMemoryResource rocksMemoryResource =
      new RocksMemoryResource();

  @Rule
  public TemporaryFolder dbFolder = new TemporaryFolder();

  @Test
  public void statisticsCollector()
      throws InterruptedException, RocksDBException {
    Options opt = null;
    RocksDB db = null;
    try {
      opt = new Options().createStatistics().setCreateIfMissing(true);
      Statistics stats = opt.statisticsPtr();

      db = RocksDB.open(opt,
          dbFolder.getRoot().getAbsolutePath());

      StatsCallbackMock callback = new StatsCallbackMock();
      StatsCollectorInput statsInput = new StatsCollectorInput(stats, callback);

      StatisticsCollector statsCollector = new StatisticsCollector(
          Collections.singletonList(statsInput), 100);
      statsCollector.start();

      Thread.sleep(1000);

      assertThat(callback.tickerCallbackCount).isGreaterThan(0);
      assertThat(callback.histCallbackCount).isGreaterThan(0);

      statsCollector.shutDown(1000);
    } finally {
      if (db != null) {
        db.close();
      }
      if (opt != null) {
        opt.dispose();
      }
    }
  }
}
