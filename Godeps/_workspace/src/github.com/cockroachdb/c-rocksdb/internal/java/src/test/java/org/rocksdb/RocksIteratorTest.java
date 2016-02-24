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

public class RocksIteratorTest {

  @ClassRule
  public static final RocksMemoryResource rocksMemoryResource =
      new RocksMemoryResource();

  @Rule
  public TemporaryFolder dbFolder = new TemporaryFolder();

  @Test
  public void rocksIterator() throws RocksDBException {
    RocksDB db = null;
    Options options = null;
    RocksIterator iterator = null;
    try {
      options = new Options();
      options.setCreateIfMissing(true)
          .setCreateMissingColumnFamilies(true);
      db = RocksDB.open(options,
          dbFolder.getRoot().getAbsolutePath());
      db.put("key1".getBytes(), "value1".getBytes());
      db.put("key2".getBytes(), "value2".getBytes());

      iterator = db.newIterator();

      iterator.seekToFirst();
      assertThat(iterator.isValid()).isTrue();
      assertThat(iterator.key()).isEqualTo("key1".getBytes());
      assertThat(iterator.value()).isEqualTo("value1".getBytes());
      iterator.next();
      assertThat(iterator.isValid()).isTrue();
      assertThat(iterator.key()).isEqualTo("key2".getBytes());
      assertThat(iterator.value()).isEqualTo("value2".getBytes());
      iterator.next();
      assertThat(iterator.isValid()).isFalse();
      iterator.seekToLast();
      iterator.prev();
      assertThat(iterator.isValid()).isTrue();
      assertThat(iterator.key()).isEqualTo("key1".getBytes());
      assertThat(iterator.value()).isEqualTo("value1".getBytes());
      iterator.seekToFirst();
      iterator.seekToLast();
      assertThat(iterator.isValid()).isTrue();
      assertThat(iterator.key()).isEqualTo("key2".getBytes());
      assertThat(iterator.value()).isEqualTo("value2".getBytes());
      iterator.status();
    } finally {
      if (iterator != null) {
        iterator.dispose();
      }
      if (db != null) {
        db.close();
      }
      if (options != null) {
        options.dispose();
      }
    }
  }
}
