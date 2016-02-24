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

public class SnapshotTest {

  @ClassRule
  public static final RocksMemoryResource rocksMemoryResource =
      new RocksMemoryResource();

  @Rule
  public TemporaryFolder dbFolder = new TemporaryFolder();

  @Test
  public void snapshots() throws RocksDBException {
    RocksDB db = null;
    Options options = null;
    ReadOptions readOptions = null;
    try {

      options = new Options();
      options.setCreateIfMissing(true);

      db = RocksDB.open(options, dbFolder.getRoot().getAbsolutePath());
      db.put("key".getBytes(), "value".getBytes());
      // Get new Snapshot of database
      Snapshot snapshot = db.getSnapshot();
      assertThat(snapshot.getSequenceNumber()).isGreaterThan(0);
      assertThat(snapshot.getSequenceNumber()).isEqualTo(1);
      readOptions = new ReadOptions();
      // set snapshot in ReadOptions
      readOptions.setSnapshot(snapshot);
      // retrieve key value pair
      assertThat(new String(db.get("key".getBytes()))).
          isEqualTo("value");
      // retrieve key value pair created before
      // the snapshot was made
      assertThat(new String(db.get(readOptions,
          "key".getBytes()))).isEqualTo("value");
      // add new key/value pair
      db.put("newkey".getBytes(), "newvalue".getBytes());
      // using no snapshot the latest db entries
      // will be taken into account
      assertThat(new String(db.get("newkey".getBytes()))).
          isEqualTo("newvalue");
      // snapshopot was created before newkey
      assertThat(db.get(readOptions, "newkey".getBytes())).
          isNull();
      // Retrieve snapshot from read options
      Snapshot sameSnapshot = readOptions.snapshot();
      readOptions.setSnapshot(sameSnapshot);
      // results must be the same with new Snapshot
      // instance using the same native pointer
      assertThat(new String(db.get(readOptions,
          "key".getBytes()))).isEqualTo("value");
      // update key value pair to newvalue
      db.put("key".getBytes(), "newvalue".getBytes());
      // read with previously created snapshot will
      // read previous version of key value pair
      assertThat(new String(db.get(readOptions,
          "key".getBytes()))).isEqualTo("value");
      // read for newkey using the snapshot must be
      // null
      assertThat(db.get(readOptions, "newkey".getBytes())).
          isNull();
      // setting null to snapshot in ReadOptions leads
      // to no Snapshot being used.
      readOptions.setSnapshot(null);
      assertThat(new String(db.get(readOptions,
          "newkey".getBytes()))).isEqualTo("newvalue");
      // release Snapshot
      db.releaseSnapshot(snapshot);
    } finally {
      if (db != null) {
        db.close();
      }
      if (options != null) {
        options.dispose();
      }
      if (readOptions != null) {
        readOptions.dispose();
      }
    }
  }

  @Test
  public void iteratorWithSnapshot() throws RocksDBException {
    RocksDB db = null;
    Options options = null;
    ReadOptions readOptions = null;
    RocksIterator iterator = null;
    RocksIterator snapshotIterator = null;
    try {

      options = new Options();
      options.setCreateIfMissing(true);

      db = RocksDB.open(options, dbFolder.getRoot().getAbsolutePath());
      db.put("key".getBytes(), "value".getBytes());
      // Get new Snapshot of database
      Snapshot snapshot = db.getSnapshot();
      readOptions = new ReadOptions();
      // set snapshot in ReadOptions
      readOptions.setSnapshot(snapshot);
      db.put("key2".getBytes(), "value2".getBytes());

      // iterate over current state of db
      iterator = db.newIterator();
      iterator.seekToFirst();
      assertThat(iterator.isValid()).isTrue();
      assertThat(iterator.key()).isEqualTo("key".getBytes());
      iterator.next();
      assertThat(iterator.isValid()).isTrue();
      assertThat(iterator.key()).isEqualTo("key2".getBytes());
      iterator.next();
      assertThat(iterator.isValid()).isFalse();

      // iterate using a snapshot
      snapshotIterator = db.newIterator(readOptions);
      snapshotIterator.seekToFirst();
      assertThat(snapshotIterator.isValid()).isTrue();
      assertThat(snapshotIterator.key()).isEqualTo("key".getBytes());
      snapshotIterator.next();
      assertThat(snapshotIterator.isValid()).isFalse();

      // release Snapshot
      db.releaseSnapshot(snapshot);
    } finally {
      if (iterator != null) {
        iterator.dispose();
      }
      if (snapshotIterator != null) {
        snapshotIterator.dispose();
      }
      if (db != null) {
        db.close();
      }
      if (options != null) {
        options.dispose();
      }
      if (readOptions != null) {
        readOptions.dispose();
      }
    }
  }

  @Test
  public void iteratorWithSnapshotOnColumnFamily() throws RocksDBException {
    RocksDB db = null;
    Options options = null;
    ReadOptions readOptions = null;
    RocksIterator iterator = null;
    RocksIterator snapshotIterator = null;
    try {

      options = new Options();
      options.setCreateIfMissing(true);

      db = RocksDB.open(options, dbFolder.getRoot().getAbsolutePath());
      db.put("key".getBytes(), "value".getBytes());
      // Get new Snapshot of database
      Snapshot snapshot = db.getSnapshot();
      readOptions = new ReadOptions();
      // set snapshot in ReadOptions
      readOptions.setSnapshot(snapshot);
      db.put("key2".getBytes(), "value2".getBytes());

      // iterate over current state of column family
      iterator = db.newIterator(db.getDefaultColumnFamily());
      iterator.seekToFirst();
      assertThat(iterator.isValid()).isTrue();
      assertThat(iterator.key()).isEqualTo("key".getBytes());
      iterator.next();
      assertThat(iterator.isValid()).isTrue();
      assertThat(iterator.key()).isEqualTo("key2".getBytes());
      iterator.next();
      assertThat(iterator.isValid()).isFalse();

      // iterate using a snapshot on default column family
      snapshotIterator = db.newIterator(db.getDefaultColumnFamily(),
          readOptions);
      snapshotIterator.seekToFirst();
      assertThat(snapshotIterator.isValid()).isTrue();
      assertThat(snapshotIterator.key()).isEqualTo("key".getBytes());
      snapshotIterator.next();
      assertThat(snapshotIterator.isValid()).isFalse();

      // release Snapshot
      db.releaseSnapshot(snapshot);
    } finally {
      if (iterator != null) {
        iterator.dispose();
      }
      if (snapshotIterator != null) {
        snapshotIterator.dispose();
      }
      if (db != null) {
        db.close();
      }
      if (options != null) {
        options.dispose();
      }
      if (readOptions != null) {
        readOptions.dispose();
      }
    }
  }
}
