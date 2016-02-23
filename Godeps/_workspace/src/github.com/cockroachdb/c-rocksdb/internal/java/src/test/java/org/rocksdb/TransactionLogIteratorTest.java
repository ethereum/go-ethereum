package org.rocksdb;

import org.junit.ClassRule;
import org.junit.Rule;
import org.junit.Test;
import org.junit.rules.TemporaryFolder;

import static org.assertj.core.api.Assertions.assertThat;

public class TransactionLogIteratorTest {
  @ClassRule
  public static final RocksMemoryResource rocksMemoryResource =
      new RocksMemoryResource();

  @Rule
  public TemporaryFolder dbFolder = new TemporaryFolder();

  @Test
  public void transactionLogIterator() throws RocksDBException {
    RocksDB db = null;
    Options options = null;
    TransactionLogIterator transactionLogIterator = null;
    try {
      options = new Options().
          setCreateIfMissing(true);
      db = RocksDB.open(options, dbFolder.getRoot().getAbsolutePath());
      transactionLogIterator = db.getUpdatesSince(0);
    } finally {
      if (transactionLogIterator != null) {
        transactionLogIterator.dispose();
      }
      if (db != null) {
        db.close();
      }
      if (options != null) {
        options.dispose();
      }
    }
  }

  @Test
  public void getBatch() throws RocksDBException {
    final int numberOfPuts = 5;
    RocksDB db = null;
    Options options = null;
    ColumnFamilyHandle cfHandle = null;
    TransactionLogIterator transactionLogIterator = null;
    try {
      options = new Options().
          setCreateIfMissing(true).
          setWalTtlSeconds(1000).
          setWalSizeLimitMB(10);

      db = RocksDB.open(options, dbFolder.getRoot().getAbsolutePath());

      for (int i = 0; i < numberOfPuts; i++){
        db.put(String.valueOf(i).getBytes(),
            String.valueOf(i).getBytes());
      }
      db.flush(new FlushOptions().setWaitForFlush(true));

      // the latest sequence number is 5 because 5 puts
      // were written beforehand
      assertThat(db.getLatestSequenceNumber()).
          isEqualTo(numberOfPuts);

      // insert 5 writes into a cf
      cfHandle = db.createColumnFamily(
          new ColumnFamilyDescriptor("new_cf".getBytes()));

      for (int i = 0; i < numberOfPuts; i++){
        db.put(cfHandle, String.valueOf(i).getBytes(),
            String.valueOf(i).getBytes());
      }
      // the latest sequence number is 10 because
      // (5 + 5) puts were written beforehand
      assertThat(db.getLatestSequenceNumber()).
          isEqualTo(numberOfPuts + numberOfPuts);

      // Get updates since the beginning
      transactionLogIterator = db.getUpdatesSince(0);
      assertThat(transactionLogIterator.isValid()).isTrue();
      transactionLogIterator.status();

      // The first sequence number is 1
      TransactionLogIterator.BatchResult batchResult =
          transactionLogIterator.getBatch();
      assertThat(batchResult.sequenceNumber()).isEqualTo(1);
    } finally {
      if (transactionLogIterator != null) {
        transactionLogIterator.dispose();
      }
      if (cfHandle != null) {
        cfHandle.dispose();
      }
      if (db != null) {
        db.close();
      }
      if (options != null) {
        options.dispose();
      }
    }
  }

  @Test
  public void transactionLogIteratorStallAtLastRecord() throws RocksDBException {
    RocksDB db = null;
    Options options = null;
    TransactionLogIterator transactionLogIterator = null;
    try {
      options = new Options().
          setCreateIfMissing(true).
          setWalTtlSeconds(1000).
          setWalSizeLimitMB(10);

      db = RocksDB.open(options, dbFolder.getRoot().getAbsolutePath());
      db.put("key1".getBytes(), "value1".getBytes());
      // Get updates since the beginning
      transactionLogIterator = db.getUpdatesSince(0);
      transactionLogIterator.status();
      assertThat(transactionLogIterator.isValid()).isTrue();
      transactionLogIterator.next();
      assertThat(transactionLogIterator.isValid()).isFalse();
      transactionLogIterator.status();
      db.put("key2".getBytes(), "value2".getBytes());
      transactionLogIterator.next();
      transactionLogIterator.status();
      assertThat(transactionLogIterator.isValid()).isTrue();

    } finally {
      if (transactionLogIterator != null) {
        transactionLogIterator.dispose();
      }
      if (db != null) {
        db.close();
      }
      if (options != null) {
        options.dispose();
      }
    }
  }

  @Test
  public void transactionLogIteratorCheckAfterRestart() throws RocksDBException {
    final int numberOfKeys = 2;
    RocksDB db = null;
    Options options = null;
    TransactionLogIterator transactionLogIterator = null;
    try {
      options = new Options().
          setCreateIfMissing(true).
          setWalTtlSeconds(1000).
          setWalSizeLimitMB(10);

      db = RocksDB.open(options, dbFolder.getRoot().getAbsolutePath());
      db.put("key1".getBytes(), "value1".getBytes());
      db.put("key2".getBytes(), "value2".getBytes());
      db.flush(new FlushOptions().setWaitForFlush(true));
      // reopen
      db.close();
      db = RocksDB.open(options, dbFolder.getRoot().getAbsolutePath());
      assertThat(db.getLatestSequenceNumber()).isEqualTo(numberOfKeys);

      transactionLogIterator = db.getUpdatesSince(0);
      for (int i = 0; i < numberOfKeys; i++) {
        transactionLogIterator.status();
        assertThat(transactionLogIterator.isValid()).isTrue();
        transactionLogIterator.next();
      }
    } finally {
      if (transactionLogIterator != null) {
        transactionLogIterator.dispose();
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
