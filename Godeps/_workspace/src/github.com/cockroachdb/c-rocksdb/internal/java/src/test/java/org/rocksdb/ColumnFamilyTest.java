// Copyright (c) 2014, Facebook, Inc.  All rights reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.

package org.rocksdb;

import java.util.*;

import org.junit.ClassRule;
import org.junit.Rule;
import org.junit.Test;
import org.junit.rules.TemporaryFolder;

import static org.assertj.core.api.Assertions.assertThat;

public class ColumnFamilyTest {

  @ClassRule
  public static final RocksMemoryResource rocksMemoryResource =
      new RocksMemoryResource();

  @Rule
  public TemporaryFolder dbFolder = new TemporaryFolder();

  @Test
  public void listColumnFamilies() throws RocksDBException {
    RocksDB db = null;
    Options options = null;
    try {
      options = new Options();
      options.setCreateIfMissing(true);

      DBOptions dbOptions = new DBOptions();
      dbOptions.setCreateIfMissing(true);

      db = RocksDB.open(options, dbFolder.getRoot().getAbsolutePath());
      // Test listColumnFamilies
      List<byte[]> columnFamilyNames;
      columnFamilyNames = RocksDB.listColumnFamilies(options, dbFolder.getRoot().getAbsolutePath());
      assertThat(columnFamilyNames).isNotNull();
      assertThat(columnFamilyNames.size()).isGreaterThan(0);
      assertThat(columnFamilyNames.size()).isEqualTo(1);
      assertThat(new String(columnFamilyNames.get(0))).isEqualTo("default");
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
  public void defaultColumnFamily() throws RocksDBException {
    RocksDB db = null;
    Options options = null;
    ColumnFamilyHandle cfh;
    try {
      options = new Options().setCreateIfMissing(true);

      db = RocksDB.open(options, dbFolder.getRoot().getAbsolutePath());
      cfh = db.getDefaultColumnFamily();
      assertThat(cfh).isNotNull();

      final byte[] key = "key".getBytes();
      final byte[] value = "value".getBytes();

      db.put(cfh, key, value);

      final byte[] actualValue = db.get(cfh, key);

      assertThat(cfh).isNotNull();
      assertThat(actualValue).isEqualTo(value);
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
  public void createColumnFamily() throws RocksDBException {
    RocksDB db = null;
    Options options = null;
    ColumnFamilyHandle columnFamilyHandle = null;
    try {
      options = new Options();
      options.setCreateIfMissing(true);

      db = RocksDB.open(options, dbFolder.getRoot().getAbsolutePath());
      columnFamilyHandle = db.createColumnFamily(
          new ColumnFamilyDescriptor("new_cf".getBytes(), new ColumnFamilyOptions()));

      List<byte[]> columnFamilyNames;
      columnFamilyNames = RocksDB.listColumnFamilies(options, dbFolder.getRoot().getAbsolutePath());
      assertThat(columnFamilyNames).isNotNull();
      assertThat(columnFamilyNames.size()).isGreaterThan(0);
      assertThat(columnFamilyNames.size()).isEqualTo(2);
      assertThat(new String(columnFamilyNames.get(0))).isEqualTo("default");
      assertThat(new String(columnFamilyNames.get(1))).isEqualTo("new_cf");
    } finally {
      if (columnFamilyHandle != null) {
        columnFamilyHandle.dispose();
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
  public void openWithColumnFamilies() throws RocksDBException {
    RocksDB db = null;
    DBOptions options = null;
    List<ColumnFamilyDescriptor> cfNames =
        new ArrayList<>();
    List<ColumnFamilyHandle> columnFamilyHandleList =
        new ArrayList<>();
    try {
      options = new DBOptions();
      options.setCreateIfMissing(true);
      options.setCreateMissingColumnFamilies(true);
      // Test open database with column family names
      cfNames.add(new ColumnFamilyDescriptor(RocksDB.DEFAULT_COLUMN_FAMILY));
      cfNames.add(new ColumnFamilyDescriptor("new_cf".getBytes()));

      db = RocksDB.open(options, dbFolder.getRoot().getAbsolutePath(),
          cfNames, columnFamilyHandleList);
      assertThat(columnFamilyHandleList.size()).isEqualTo(2);
      db.put("dfkey1".getBytes(), "dfvalue".getBytes());
      db.put(columnFamilyHandleList.get(0), "dfkey2".getBytes(),
          "dfvalue".getBytes());
      db.put(columnFamilyHandleList.get(1), "newcfkey1".getBytes(),
          "newcfvalue".getBytes());

      String retVal = new String(db.get(columnFamilyHandleList.get(1),
          "newcfkey1".getBytes()));
      assertThat(retVal).isEqualTo("newcfvalue");
      assertThat((db.get(columnFamilyHandleList.get(1),
          "dfkey1".getBytes()))).isNull();
      db.remove(columnFamilyHandleList.get(1), "newcfkey1".getBytes());
      assertThat((db.get(columnFamilyHandleList.get(1),
          "newcfkey1".getBytes()))).isNull();
      db.remove(columnFamilyHandleList.get(0), new WriteOptions(),
          "dfkey2".getBytes());
      assertThat(db.get(columnFamilyHandleList.get(0), new ReadOptions(),
          "dfkey2".getBytes())).isNull();
    } finally {
      for (ColumnFamilyHandle columnFamilyHandle : columnFamilyHandleList) {
        columnFamilyHandle.dispose();
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
  public void getWithOutValueAndCf() throws RocksDBException {
    RocksDB db = null;
    DBOptions options = null;
    List<ColumnFamilyDescriptor> cfDescriptors =
        new ArrayList<>();
    List<ColumnFamilyHandle> columnFamilyHandleList =
        new ArrayList<>();
    try {
      options = new DBOptions();
      options.setCreateIfMissing(true);
      options.setCreateMissingColumnFamilies(true);
      // Test open database with column family names
      cfDescriptors.add(new ColumnFamilyDescriptor(RocksDB.DEFAULT_COLUMN_FAMILY));
      db = RocksDB.open(options, dbFolder.getRoot().getAbsolutePath(),
          cfDescriptors, columnFamilyHandleList);
      db.put(columnFamilyHandleList.get(0), new WriteOptions(),
          "key1".getBytes(), "value".getBytes());
      db.put("key2".getBytes(), "12345678".getBytes());
      byte[] outValue = new byte[5];
      // not found value
      int getResult = db.get("keyNotFound".getBytes(), outValue);
      assertThat(getResult).isEqualTo(RocksDB.NOT_FOUND);
      // found value which fits in outValue
      getResult = db.get(columnFamilyHandleList.get(0), "key1".getBytes(), outValue);
      assertThat(getResult).isNotEqualTo(RocksDB.NOT_FOUND);
      assertThat(outValue).isEqualTo("value".getBytes());
      // found value which fits partially
      getResult = db.get(columnFamilyHandleList.get(0), new ReadOptions(),
          "key2".getBytes(), outValue);
      assertThat(getResult).isNotEqualTo(RocksDB.NOT_FOUND);
      assertThat(outValue).isEqualTo("12345".getBytes());
    } finally {
      for (ColumnFamilyHandle columnFamilyHandle : columnFamilyHandleList) {
        columnFamilyHandle.dispose();
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
  public void createWriteDropColumnFamily() throws RocksDBException {
    RocksDB db = null;
    DBOptions opt = null;
    ColumnFamilyHandle tmpColumnFamilyHandle = null;
    List<ColumnFamilyDescriptor> cfNames =
        new ArrayList<>();
    List<ColumnFamilyHandle> columnFamilyHandleList =
        new ArrayList<>();
    try {
      opt = new DBOptions();
      opt.setCreateIfMissing(true);
      opt.setCreateMissingColumnFamilies(true);
      cfNames.add(new ColumnFamilyDescriptor(RocksDB.DEFAULT_COLUMN_FAMILY));
      cfNames.add(new ColumnFamilyDescriptor("new_cf".getBytes()));

      db = RocksDB.open(opt, dbFolder.getRoot().getAbsolutePath(),
          cfNames, columnFamilyHandleList);
      tmpColumnFamilyHandle = db.createColumnFamily(
          new ColumnFamilyDescriptor("tmpCF".getBytes(), new ColumnFamilyOptions()));
      db.put(tmpColumnFamilyHandle, "key".getBytes(), "value".getBytes());
      db.dropColumnFamily(tmpColumnFamilyHandle);
      tmpColumnFamilyHandle.dispose();
    } finally {
      for (ColumnFamilyHandle columnFamilyHandle : columnFamilyHandleList) {
        columnFamilyHandle.dispose();
      }
      if (tmpColumnFamilyHandle != null) {
        tmpColumnFamilyHandle.dispose();
      }
      if (db != null) {
        db.close();
      }
      if (opt != null) {
        opt.dispose();
      }
    }
  }

  @Test
  public void writeBatch() throws RocksDBException {
    RocksDB db = null;
    DBOptions opt = null;
    List<ColumnFamilyDescriptor> cfNames =
        new ArrayList<>();
    List<ColumnFamilyHandle> columnFamilyHandleList =
        new ArrayList<>();
    try {
      opt = new DBOptions();
      opt.setCreateIfMissing(true);
      opt.setCreateMissingColumnFamilies(true);

      cfNames.add(new ColumnFamilyDescriptor(RocksDB.DEFAULT_COLUMN_FAMILY,
          new ColumnFamilyOptions().setMergeOperator(new StringAppendOperator())));
      cfNames.add(new ColumnFamilyDescriptor("new_cf".getBytes()));

      db = RocksDB.open(opt, dbFolder.getRoot().getAbsolutePath(),
          cfNames, columnFamilyHandleList);

      WriteBatch writeBatch = new WriteBatch();
      WriteOptions writeOpt = new WriteOptions();
      writeBatch.put("key".getBytes(), "value".getBytes());
      writeBatch.put(db.getDefaultColumnFamily(),
          "mergeKey".getBytes(), "merge".getBytes());
      writeBatch.merge(db.getDefaultColumnFamily(), "mergeKey".getBytes(),
          "merge".getBytes());
      writeBatch.put(columnFamilyHandleList.get(1), "newcfkey".getBytes(),
          "value".getBytes());
      writeBatch.put(columnFamilyHandleList.get(1), "newcfkey2".getBytes(),
          "value2".getBytes());
      writeBatch.remove("xyz".getBytes());
      writeBatch.remove(columnFamilyHandleList.get(1), "xyz".getBytes());
      db.write(writeOpt, writeBatch);
      writeBatch.dispose();
      assertThat(db.get(columnFamilyHandleList.get(1),
          "xyz".getBytes()) == null);
      assertThat(new String(db.get(columnFamilyHandleList.get(1),
          "newcfkey".getBytes()))).isEqualTo("value");
      assertThat(new String(db.get(columnFamilyHandleList.get(1),
          "newcfkey2".getBytes()))).isEqualTo("value2");
      assertThat(new String(db.get("key".getBytes()))).isEqualTo("value");
      // check if key is merged
      assertThat(new String(db.get(db.getDefaultColumnFamily(),
          "mergeKey".getBytes()))).isEqualTo("merge,merge");
    } finally {
      for (ColumnFamilyHandle columnFamilyHandle : columnFamilyHandleList) {
        columnFamilyHandle.dispose();
      }
      if (db != null) {
        db.close();
      }
      if (opt != null) {
        opt.dispose();
      }
    }
  }

  @Test
  public void iteratorOnColumnFamily() throws RocksDBException {
    RocksDB db = null;
    DBOptions options = null;
    RocksIterator rocksIterator = null;
    List<ColumnFamilyDescriptor> cfNames =
        new ArrayList<>();
    List<ColumnFamilyHandle> columnFamilyHandleList =
        new ArrayList<>();
    try {
      options = new DBOptions();
      options.setCreateIfMissing(true);
      options.setCreateMissingColumnFamilies(true);

      cfNames.add(new ColumnFamilyDescriptor(RocksDB.DEFAULT_COLUMN_FAMILY));
      cfNames.add(new ColumnFamilyDescriptor("new_cf".getBytes()));

      db = RocksDB.open(options, dbFolder.getRoot().getAbsolutePath(),
          cfNames, columnFamilyHandleList);
      db.put(columnFamilyHandleList.get(1), "newcfkey".getBytes(),
          "value".getBytes());
      db.put(columnFamilyHandleList.get(1), "newcfkey2".getBytes(),
          "value2".getBytes());
      rocksIterator = db.newIterator(
          columnFamilyHandleList.get(1));
      rocksIterator.seekToFirst();
      Map<String, String> refMap = new HashMap<>();
      refMap.put("newcfkey", "value");
      refMap.put("newcfkey2", "value2");
      int i = 0;
      while (rocksIterator.isValid()) {
        i++;
        assertThat(refMap.get(new String(rocksIterator.key()))).
            isEqualTo(new String(rocksIterator.value()));
        rocksIterator.next();
      }
      assertThat(i).isEqualTo(2);
      rocksIterator.dispose();
    } finally {
      if (rocksIterator != null) {
        rocksIterator.dispose();
      }
      for (ColumnFamilyHandle columnFamilyHandle : columnFamilyHandleList) {
        columnFamilyHandle.dispose();
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
  public void multiGet() throws RocksDBException {
    RocksDB db = null;
    DBOptions options = null;
    List<ColumnFamilyDescriptor> cfDescriptors =
        new ArrayList<>();
    List<ColumnFamilyHandle> columnFamilyHandleList =
        new ArrayList<>();
    try {
      options = new DBOptions();
      options.setCreateIfMissing(true);
      options.setCreateMissingColumnFamilies(true);

      cfDescriptors.add(new ColumnFamilyDescriptor(RocksDB.DEFAULT_COLUMN_FAMILY));
      cfDescriptors.add(new ColumnFamilyDescriptor("new_cf".getBytes()));

      db = RocksDB.open(options, dbFolder.getRoot().getAbsolutePath(),
          cfDescriptors, columnFamilyHandleList);
      db.put(columnFamilyHandleList.get(0), "key".getBytes(), "value".getBytes());
      db.put(columnFamilyHandleList.get(1), "newcfkey".getBytes(), "value".getBytes());

      List<byte[]> keys = new ArrayList<>();
      keys.add("key".getBytes());
      keys.add("newcfkey".getBytes());
      Map<byte[], byte[]> retValues = db.multiGet(columnFamilyHandleList, keys);
      assertThat(retValues.size()).isEqualTo(2);
      assertThat(new String(retValues.get(keys.get(0))))
          .isEqualTo("value");
      assertThat(new String(retValues.get(keys.get(1))))
          .isEqualTo("value");
      retValues = db.multiGet(new ReadOptions(), columnFamilyHandleList, keys);
      assertThat(retValues.size()).isEqualTo(2);
      assertThat(new String(retValues.get(keys.get(0))))
          .isEqualTo("value");
      assertThat(new String(retValues.get(keys.get(1))))
          .isEqualTo("value");
    } finally {
      for (ColumnFamilyHandle columnFamilyHandle : columnFamilyHandleList) {
        columnFamilyHandle.dispose();
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
  public void properties() throws RocksDBException {
    RocksDB db = null;
    DBOptions options = null;
    List<ColumnFamilyDescriptor> cfNames =
        new ArrayList<>();
    List<ColumnFamilyHandle> columnFamilyHandleList =
        new ArrayList<>();
    try {
      options = new DBOptions();
      options.setCreateIfMissing(true);
      options.setCreateMissingColumnFamilies(true);

      cfNames.add(new ColumnFamilyDescriptor(RocksDB.DEFAULT_COLUMN_FAMILY));
      cfNames.add(new ColumnFamilyDescriptor("new_cf".getBytes()));

      db = RocksDB.open(options, dbFolder.getRoot().getAbsolutePath(),
          cfNames, columnFamilyHandleList);
      assertThat(db.getProperty("rocksdb.estimate-num-keys")).
          isNotNull();
      assertThat(db.getLongProperty(columnFamilyHandleList.get(0),
          "rocksdb.estimate-num-keys")).isGreaterThanOrEqualTo(0);
      assertThat(db.getProperty("rocksdb.stats")).isNotNull();
      assertThat(db.getProperty(columnFamilyHandleList.get(0),
          "rocksdb.sstables")).isNotNull();
      assertThat(db.getProperty(columnFamilyHandleList.get(1),
          "rocksdb.estimate-num-keys")).isNotNull();
      assertThat(db.getProperty(columnFamilyHandleList.get(1),
          "rocksdb.stats")).isNotNull();
      assertThat(db.getProperty(columnFamilyHandleList.get(1),
          "rocksdb.sstables")).isNotNull();
    } finally {
      for (ColumnFamilyHandle columnFamilyHandle : columnFamilyHandleList) {
        columnFamilyHandle.dispose();
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
  public void iterators() throws RocksDBException {
    RocksDB db = null;
    DBOptions options = null;
    List<ColumnFamilyDescriptor> cfNames =
        new ArrayList<>();
    List<ColumnFamilyHandle> columnFamilyHandleList =
        new ArrayList<>();
    List<RocksIterator> iterators = null;
    try {
      options = new DBOptions();
      options.setCreateIfMissing(true);
      options.setCreateMissingColumnFamilies(true);

      cfNames.add(new ColumnFamilyDescriptor(RocksDB.DEFAULT_COLUMN_FAMILY));
      cfNames.add(new ColumnFamilyDescriptor("new_cf".getBytes()));

      db = RocksDB.open(options, dbFolder.getRoot().getAbsolutePath(),
          cfNames, columnFamilyHandleList);
      iterators = db.newIterators(columnFamilyHandleList);
      assertThat(iterators.size()).isEqualTo(2);
      RocksIterator iter = iterators.get(0);
      iter.seekToFirst();
      Map<String, String> defRefMap = new HashMap<>();
      defRefMap.put("dfkey1", "dfvalue");
      defRefMap.put("key", "value");
      while (iter.isValid()) {
        assertThat(defRefMap.get(new String(iter.key()))).
            isEqualTo(new String(iter.value()));
        iter.next();
      }
      // iterate over new_cf key/value pairs
      Map<String, String> cfRefMap = new HashMap<>();
      cfRefMap.put("newcfkey", "value");
      cfRefMap.put("newcfkey2", "value2");
      iter = iterators.get(1);
      iter.seekToFirst();
      while (iter.isValid()) {
        assertThat(cfRefMap.get(new String(iter.key()))).
            isEqualTo(new String(iter.value()));
        iter.next();
      }
    } finally {
      if (iterators != null) {
        for (RocksIterator rocksIterator : iterators) {
          rocksIterator.dispose();
        }
      }
      for (ColumnFamilyHandle columnFamilyHandle : columnFamilyHandleList) {
        columnFamilyHandle.dispose();
      }
      if (db != null) {
        db.close();
      }
      if (options != null) {
        options.dispose();
      }
    }
  }

  @Test(expected = RocksDBException.class)
  public void failPutDisposedCF() throws RocksDBException {
    RocksDB db = null;
    DBOptions options = null;
    List<ColumnFamilyDescriptor> cfNames =
        new ArrayList<>();
    List<ColumnFamilyHandle> columnFamilyHandleList =
        new ArrayList<>();
    try {
      options = new DBOptions();
      options.setCreateIfMissing(true);

      cfNames.add(new ColumnFamilyDescriptor(RocksDB.DEFAULT_COLUMN_FAMILY));
      cfNames.add(new ColumnFamilyDescriptor("new_cf".getBytes()));

      db = RocksDB.open(options, dbFolder.getRoot().getAbsolutePath(),
          cfNames, columnFamilyHandleList);
      db.dropColumnFamily(columnFamilyHandleList.get(1));
      db.put(columnFamilyHandleList.get(1), "key".getBytes(), "value".getBytes());
    } finally {
      for (ColumnFamilyHandle columnFamilyHandle : columnFamilyHandleList) {
        columnFamilyHandle.dispose();
      }
      if (db != null) {
        db.close();
      }
      if (options != null) {
        options.dispose();
      }
    }
  }

  @Test(expected = RocksDBException.class)
  public void failRemoveDisposedCF() throws RocksDBException {
    RocksDB db = null;
    DBOptions options = null;
    List<ColumnFamilyDescriptor> cfNames =
        new ArrayList<>();
    List<ColumnFamilyHandle> columnFamilyHandleList =
        new ArrayList<>();
    try {
      options = new DBOptions();
      options.setCreateIfMissing(true);

      cfNames.add(new ColumnFamilyDescriptor(RocksDB.DEFAULT_COLUMN_FAMILY));
      cfNames.add(new ColumnFamilyDescriptor("new_cf".getBytes()));

      db = RocksDB.open(options, dbFolder.getRoot().getAbsolutePath(),
          cfNames, columnFamilyHandleList);
      db.dropColumnFamily(columnFamilyHandleList.get(1));
      db.remove(columnFamilyHandleList.get(1), "key".getBytes());
    } finally {
      for (ColumnFamilyHandle columnFamilyHandle : columnFamilyHandleList) {
        columnFamilyHandle.dispose();
      }
      if (db != null) {
        db.close();
      }
      if (options != null) {
        options.dispose();
      }
    }
  }

  @Test(expected = RocksDBException.class)
  public void failGetDisposedCF() throws RocksDBException {
    RocksDB db = null;
    DBOptions options = null;
    List<ColumnFamilyDescriptor> cfNames =
        new ArrayList<>();
    List<ColumnFamilyHandle> columnFamilyHandleList =
        new ArrayList<>();
    try {
      options = new DBOptions();
      options.setCreateIfMissing(true);

      cfNames.add(new ColumnFamilyDescriptor(RocksDB.DEFAULT_COLUMN_FAMILY));
      cfNames.add(new ColumnFamilyDescriptor("new_cf".getBytes()));

      db = RocksDB.open(options, dbFolder.getRoot().getAbsolutePath(),
          cfNames, columnFamilyHandleList);
      db.dropColumnFamily(columnFamilyHandleList.get(1));
      db.get(columnFamilyHandleList.get(1), "key".getBytes());
    } finally {
      for (ColumnFamilyHandle columnFamilyHandle : columnFamilyHandleList) {
        columnFamilyHandle.dispose();
      }
      if (db != null) {
        db.close();
      }
      if (options != null) {
        options.dispose();
      }
    }
  }

  @Test(expected = RocksDBException.class)
  public void failMultiGetWithoutCorrectNumberOfCF() throws RocksDBException {
    RocksDB db = null;
    DBOptions options = null;
    List<ColumnFamilyDescriptor> cfNames =
        new ArrayList<>();
    List<ColumnFamilyHandle> columnFamilyHandleList =
        new ArrayList<>();
    try {
      options = new DBOptions();
      options.setCreateIfMissing(true);

      cfNames.add(new ColumnFamilyDescriptor(RocksDB.DEFAULT_COLUMN_FAMILY));
      cfNames.add(new ColumnFamilyDescriptor("new_cf".getBytes()));

      db = RocksDB.open(options, dbFolder.getRoot().getAbsolutePath(),
          cfNames, columnFamilyHandleList);
      List<byte[]> keys = new ArrayList<>();
      keys.add("key".getBytes());
      keys.add("newcfkey".getBytes());
      List<ColumnFamilyHandle> cfCustomList = new ArrayList<>();
      db.multiGet(cfCustomList, keys);

    } finally {
      for (ColumnFamilyHandle columnFamilyHandle : columnFamilyHandleList) {
        columnFamilyHandle.dispose();
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
  public void testByteCreateFolumnFamily() throws RocksDBException {
    RocksDB db = null;
    Options options = null;
    ColumnFamilyHandle cf1 = null, cf2 = null, cf3 = null;
    try {
      options = new Options().setCreateIfMissing(true);
      db = RocksDB.open(options, dbFolder.getRoot().getAbsolutePath());

      byte[] b0 = new byte[] { (byte)0x00 };
      byte[] b1 = new byte[] { (byte)0x01 };
      byte[] b2 = new byte[] { (byte)0x02 };
      cf1 = db.createColumnFamily(new ColumnFamilyDescriptor(b0));
      cf2 = db.createColumnFamily(new ColumnFamilyDescriptor(b1));
      List<byte[]> families = RocksDB.listColumnFamilies(options, dbFolder.getRoot().getAbsolutePath());
      assertThat(families).contains("default".getBytes(), b0, b1);
      cf3 = db.createColumnFamily(new ColumnFamilyDescriptor(b2));
    } finally {
      if (cf1 != null) {
        cf1.dispose();
      }
      if (cf2 != null) {
        cf2.dispose();
      }
      if (cf3 != null) {
        cf3.dispose();
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
  public void testCFNamesWithZeroBytes() throws RocksDBException {
    RocksDB db = null;
    Options options = null;
    ColumnFamilyHandle cf1 = null, cf2 = null;
    try {
      options = new Options().setCreateIfMissing(true);
      db = RocksDB.open(options, dbFolder.getRoot().getAbsolutePath());

      byte[] b0 = new byte[] { 0, 0 };
      byte[] b1 = new byte[] { 0, 1 };
      cf1 = db.createColumnFamily(new ColumnFamilyDescriptor(b0));
      cf2 = db.createColumnFamily(new ColumnFamilyDescriptor(b1));
      List<byte[]> families = RocksDB.listColumnFamilies(options, dbFolder.getRoot().getAbsolutePath());
      assertThat(families).contains("default".getBytes(), b0, b1);
    } finally {
      if (cf1 != null) {
        cf1.dispose();
      }
      if (cf2 != null) {
        cf2.dispose();
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
  public void testCFNameSimplifiedChinese() throws RocksDBException {
    RocksDB db = null;
    Options options = null;
    ColumnFamilyHandle columnFamilyHandle = null;
    try {
      options = new Options().setCreateIfMissing(true);
      db = RocksDB.open(options, dbFolder.getRoot().getAbsolutePath());
      final String simplifiedChinese = "\u7b80\u4f53\u5b57";
      columnFamilyHandle = db.createColumnFamily(
          new ColumnFamilyDescriptor(simplifiedChinese.getBytes()));

      List<byte[]> families = RocksDB.listColumnFamilies(options, dbFolder.getRoot().getAbsolutePath());
      assertThat(families).contains("default".getBytes(), simplifiedChinese.getBytes());
    } finally {
      if (columnFamilyHandle != null) {
        columnFamilyHandle.dispose();
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
