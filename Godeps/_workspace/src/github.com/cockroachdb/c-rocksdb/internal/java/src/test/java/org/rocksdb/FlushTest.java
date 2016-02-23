// Copyright (c) 2014, Facebook, Inc.  All rights reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.
package org.rocksdb;

import org.junit.ClassRule;
import org.junit.Rule;
import org.junit.Test;
import org.junit.rules.TemporaryFolder;

import static org.assertj.core.api.Assertions.assertThat;

public class FlushTest {

  @ClassRule
  public static final RocksMemoryResource rocksMemoryResource =
      new RocksMemoryResource();

  @Rule
  public TemporaryFolder dbFolder = new TemporaryFolder();

  @Test
  public void flush() throws RocksDBException {
    RocksDB db = null;
    Options options = null;
    WriteOptions wOpt = null;
    FlushOptions flushOptions = null;
    try {
      options = new Options();
      // Setup options
      options.setCreateIfMissing(true);
      options.setMaxWriteBufferNumber(10);
      options.setMinWriteBufferNumberToMerge(10);
      wOpt = new WriteOptions();
      flushOptions = new FlushOptions();
      flushOptions.setWaitForFlush(true);
      assertThat(flushOptions.waitForFlush()).isTrue();
      wOpt.setDisableWAL(true);
      db = RocksDB.open(options, dbFolder.getRoot().getAbsolutePath());
      db.put(wOpt, "key1".getBytes(), "value1".getBytes());
      db.put(wOpt, "key2".getBytes(), "value2".getBytes());
      db.put(wOpt, "key3".getBytes(), "value3".getBytes());
      db.put(wOpt, "key4".getBytes(), "value4".getBytes());
      assertThat(db.getProperty("rocksdb.num-entries-active-mem-table")).isEqualTo("4");
      db.flush(flushOptions);
      assertThat(db.getProperty("rocksdb.num-entries-active-mem-table")).
          isEqualTo("0");
    } finally {
      if (flushOptions != null) {
        flushOptions.dispose();
      }
      if (db != null) {
        db.close();
      }
      if (options != null) {
        options.dispose();
      }
      if (wOpt != null) {
        wOpt.dispose();
      }

    }
  }
}
