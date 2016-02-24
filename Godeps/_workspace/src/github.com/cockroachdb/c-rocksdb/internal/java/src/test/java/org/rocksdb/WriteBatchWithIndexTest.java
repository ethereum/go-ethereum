//  Copyright (c) 2014, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.
//
// Copyright (c) 2011 The LevelDB Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file. See the AUTHORS file for names of contributors.

package org.rocksdb;

import org.junit.ClassRule;
import org.junit.Rule;
import org.junit.Test;
import org.junit.rules.TemporaryFolder;

import java.nio.ByteBuffer;
import java.util.ArrayDeque;
import java.util.Deque;

import static org.assertj.core.api.Assertions.assertThat;


public class WriteBatchWithIndexTest {

  @ClassRule
  public static final RocksMemoryResource rocksMemoryResource =
      new RocksMemoryResource();

  @Rule
  public TemporaryFolder dbFolder = new TemporaryFolder();

  @Test
  public void readYourOwnWrites() throws RocksDBException {
    RocksDB db = null;
    Options options = null;
    try {
      options = new Options();
      // Setup options
      options.setCreateIfMissing(true);
      db = RocksDB.open(options, dbFolder.getRoot().getAbsolutePath());

      final byte[] k1 = "key1".getBytes();
      final byte[] v1 = "value1".getBytes();
      final byte[] k2 = "key2".getBytes();
      final byte[] v2 = "value2".getBytes();

      db.put(k1, v1);
      db.put(k2, v2);

      final WriteBatchWithIndex wbwi = new WriteBatchWithIndex(true);

      RocksIterator base = null;
      RocksIterator it = null;
      try {
        base = db.newIterator();
        it = wbwi.newIteratorWithBase(base);

        it.seek(k1);
        assertThat(it.isValid()).isTrue();
        assertThat(it.key()).isEqualTo(k1);
        assertThat(it.value()).isEqualTo(v1);

        it.seek(k2);
        assertThat(it.isValid()).isTrue();
        assertThat(it.key()).isEqualTo(k2);
        assertThat(it.value()).isEqualTo(v2);

        //put data to the write batch and make sure we can read it.
        final byte[] k3 = "key3".getBytes();
        final byte[] v3 = "value3".getBytes();
        wbwi.put(k3, v3);
        it.seek(k3);
        assertThat(it.isValid()).isTrue();
        assertThat(it.key()).isEqualTo(k3);
        assertThat(it.value()).isEqualTo(v3);

        //update k2 in the write batch and check the value
        final byte[] v2Other = "otherValue2".getBytes();
        wbwi.put(k2, v2Other);
        it.seek(k2);
        assertThat(it.isValid()).isTrue();
        assertThat(it.key()).isEqualTo(k2);
        assertThat(it.value()).isEqualTo(v2Other);

        //remove k1 and make sure we can read back the write
        wbwi.remove(k1);
        it.seek(k1);
        assertThat(it.key()).isNotEqualTo(k1);

        //reinsert k1 and make sure we see the new value
        final byte[] v1Other = "otherValue1".getBytes();
        wbwi.put(k1, v1Other);
        it.seek(k1);
        assertThat(it.isValid()).isTrue();
        assertThat(it.key()).isEqualTo(k1);
        assertThat(it.value()).isEqualTo(v1Other);
      } finally {
        if (it != null) {
          it.dispose();
        }
        if (base != null) {
          base.dispose();
        }
      }

    } finally {
      if (db != null) {
        db.close();
      }
      if (options != null) {
        options.dispose();
      }
    }
  }

  @Test
  public void write_writeBatchWithIndex() throws RocksDBException {
    RocksDB db = null;
    Options options = null;
    try {
      options = new Options();
      // Setup options
      options.setCreateIfMissing(true);
      db = RocksDB.open(options, dbFolder.getRoot().getAbsolutePath());

      final byte[] k1 = "key1".getBytes();
      final byte[] v1 = "value1".getBytes();
      final byte[] k2 = "key2".getBytes();
      final byte[] v2 = "value2".getBytes();

      WriteBatchWithIndex wbwi = null;

      try {
        wbwi = new WriteBatchWithIndex();


        wbwi.put(k1, v1);
        wbwi.put(k2, v2);

        db.write(new WriteOptions(), wbwi);
      } finally {
        if(wbwi != null) {
          wbwi.dispose();
        }
      }

      assertThat(db.get(k1)).isEqualTo(v1);
      assertThat(db.get(k2)).isEqualTo(v2);

    } finally {
      if (db != null) {
        db.close();
      }
      if (options != null) {
        options.dispose();
      }
    }
  }

  @Test
  public void iterator() throws RocksDBException {
    final WriteBatchWithIndex wbwi = new WriteBatchWithIndex(true);

    final String k1 = "key1";
    final String v1 = "value1";
    final String k2 = "key2";
    final String v2 = "value2";
    final String k3 = "key3";
    final String v3 = "value3";
    final byte[] k1b = k1.getBytes();
    final byte[] v1b = v1.getBytes();
    final byte[] k2b = k2.getBytes();
    final byte[] v2b = v2.getBytes();
    final byte[] k3b = k3.getBytes();
    final byte[] v3b = v3.getBytes();

    //add put records
    wbwi.put(k1b, v1b);
    wbwi.put(k2b, v2b);
    wbwi.put(k3b, v3b);

    //add a deletion record
    final String k4 = "key4";
    final byte[] k4b = k4.getBytes();
    wbwi.remove(k4b);

    WBWIRocksIterator.WriteEntry[] expected = {
        new WBWIRocksIterator.WriteEntry(WBWIRocksIterator.WriteType.PUT,
            new DirectSlice(k1), new DirectSlice(v1)),
        new WBWIRocksIterator.WriteEntry(WBWIRocksIterator.WriteType.PUT,
            new DirectSlice(k2), new DirectSlice(v2)),
        new WBWIRocksIterator.WriteEntry(WBWIRocksIterator.WriteType.PUT,
            new DirectSlice(k3), new DirectSlice(v3)),
        new WBWIRocksIterator.WriteEntry(WBWIRocksIterator.WriteType.DELETE,
            new DirectSlice(k4), DirectSlice.NONE)
    };

    WBWIRocksIterator it = null;
    try {
      it = wbwi.newIterator();

      //direct access - seek to key offsets
      final int[] testOffsets = {2, 0, 1, 3};

      for(int i = 0; i < testOffsets.length; i++) {
        final int testOffset = testOffsets[i];
        final byte[] key = toArray(expected[testOffset].getKey().data());

        it.seek(key);
        assertThat(it.isValid()).isTrue();
        assertThat(it.entry().equals(expected[testOffset])).isTrue();
      }

      //forward iterative access
      int i = 0;
      for(it.seekToFirst(); it.isValid(); it.next()) {
        assertThat(it.entry().equals(expected[i++])).isTrue();
      }

      //reverse iterative access
      i = expected.length - 1;
      for(it.seekToLast(); it.isValid(); it.prev()) {
        assertThat(it.entry().equals(expected[i--])).isTrue();
      }

    } finally {
      if(it != null) {
        it.dispose();
      }
    }
  }

  @Test
  public void zeroByteTests() {
    final WriteBatchWithIndex wbwi = new WriteBatchWithIndex(true);
    byte[] zeroByteValue = new byte[] { 0, 0 };

    //add zero byte value
    wbwi.put(zeroByteValue, zeroByteValue);

    ByteBuffer buffer = ByteBuffer.allocateDirect(zeroByteValue.length);
    buffer.put(zeroByteValue);

    WBWIRocksIterator.WriteEntry[] expected = {
        new WBWIRocksIterator.WriteEntry(WBWIRocksIterator.WriteType.PUT,
            new DirectSlice(buffer, zeroByteValue.length),
            new DirectSlice(buffer, zeroByteValue.length))
    };
    WBWIRocksIterator it = null;
    try {
      it = wbwi.newIterator();
      it.seekToFirst();
      assertThat(it.entry().equals(expected[0])).isTrue();
      assertThat(it.entry().hashCode() == expected[0].hashCode()).isTrue();
    } finally {
      if(it != null) {
        it.dispose();
      }
    }
  }

  private byte[] toArray(final ByteBuffer buf) {
    final byte[] ary = new byte[buf.remaining()];
    buf.get(ary);
    return ary;
  }
}
