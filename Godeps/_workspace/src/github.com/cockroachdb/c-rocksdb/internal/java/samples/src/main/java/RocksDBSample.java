// Copyright (c) 2014, Facebook, Inc.  All rights reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.

import java.lang.IllegalArgumentException;
import java.util.Arrays;
import java.util.List;
import java.util.Map;
import java.util.ArrayList;
import org.rocksdb.*;
import org.rocksdb.util.SizeUnit;
import java.io.IOException;

public class RocksDBSample {
  static {
    RocksDB.loadLibrary();
  }

  public static void main(String[] args) {
    if (args.length < 1) {
      System.out.println("usage: RocksDBSample db_path");
      return;
    }
    String db_path = args[0];
    String db_path_not_found = db_path + "_not_found";

    System.out.println("RocksDBSample");
    RocksDB db = null;
    Options options = new Options();
    try {
      db = RocksDB.open(options, db_path_not_found);
      assert(false);
    } catch (RocksDBException e) {
      System.out.format("caught the expceted exception -- %s\n", e);
      assert(db == null);
    }

    try {
      options.setCreateIfMissing(true)
          .createStatistics()
          .setWriteBufferSize(8 * SizeUnit.KB)
          .setMaxWriteBufferNumber(3)
          .setMaxBackgroundCompactions(10)
          .setCompressionType(CompressionType.SNAPPY_COMPRESSION)
          .setCompactionStyle(CompactionStyle.UNIVERSAL);
    } catch (IllegalArgumentException e) {
      assert(false);
    }

    Statistics stats = options.statisticsPtr();

    assert(options.createIfMissing() == true);
    assert(options.writeBufferSize() == 8 * SizeUnit.KB);
    assert(options.maxWriteBufferNumber() == 3);
    assert(options.maxBackgroundCompactions() == 10);
    assert(options.compressionType() == CompressionType.SNAPPY_COMPRESSION);
    assert(options.compactionStyle() == CompactionStyle.UNIVERSAL);

    assert(options.memTableFactoryName().equals("SkipListFactory"));
    options.setMemTableConfig(
        new HashSkipListMemTableConfig()
            .setHeight(4)
            .setBranchingFactor(4)
            .setBucketCount(2000000));
    assert(options.memTableFactoryName().equals("HashSkipListRepFactory"));

    options.setMemTableConfig(
        new HashLinkedListMemTableConfig()
            .setBucketCount(100000));
    assert(options.memTableFactoryName().equals("HashLinkedListRepFactory"));

    options.setMemTableConfig(
        new VectorMemTableConfig().setReservedSize(10000));
    assert(options.memTableFactoryName().equals("VectorRepFactory"));

    options.setMemTableConfig(new SkipListMemTableConfig());
    assert(options.memTableFactoryName().equals("SkipListFactory"));

    options.setTableFormatConfig(new PlainTableConfig());
    // Plain-Table requires mmap read
    options.setAllowMmapReads(true);
    assert(options.tableFactoryName().equals("PlainTable"));

    options.setRateLimiterConfig(new GenericRateLimiterConfig(10000000,
        10000, 10));
    options.setRateLimiterConfig(new GenericRateLimiterConfig(10000000));


    Filter bloomFilter = new BloomFilter(10);
    BlockBasedTableConfig table_options = new BlockBasedTableConfig();
    table_options.setBlockCacheSize(64 * SizeUnit.KB)
                 .setFilter(bloomFilter)
                 .setCacheNumShardBits(6)
                 .setBlockSizeDeviation(5)
                 .setBlockRestartInterval(10)
                 .setCacheIndexAndFilterBlocks(true)
                 .setHashIndexAllowCollision(false)
                 .setBlockCacheCompressedSize(64 * SizeUnit.KB)
                 .setBlockCacheCompressedNumShardBits(10);

    assert(table_options.blockCacheSize() == 64 * SizeUnit.KB);
    assert(table_options.cacheNumShardBits() == 6);
    assert(table_options.blockSizeDeviation() == 5);
    assert(table_options.blockRestartInterval() == 10);
    assert(table_options.cacheIndexAndFilterBlocks() == true);
    assert(table_options.hashIndexAllowCollision() == false);
    assert(table_options.blockCacheCompressedSize() == 64 * SizeUnit.KB);
    assert(table_options.blockCacheCompressedNumShardBits() == 10);

    options.setTableFormatConfig(table_options);
    assert(options.tableFactoryName().equals("BlockBasedTable"));

    try {
      db = RocksDB.open(options, db_path);
      db.put("hello".getBytes(), "world".getBytes());
      byte[] value = db.get("hello".getBytes());
      assert("world".equals(new String(value)));
      String str = db.getProperty("rocksdb.stats");
      assert(str != null && !str.equals(""));
    } catch (RocksDBException e) {
      System.out.format("[ERROR] caught the unexpceted exception -- %s\n", e);
      assert(db == null);
      assert(false);
    }
    // be sure to release the c++ pointer
    db.close();

    ReadOptions readOptions = new ReadOptions();
    readOptions.setFillCache(false);

    try {
      db = RocksDB.open(options, db_path);
      db.put("hello".getBytes(), "world".getBytes());
      byte[] value = db.get("hello".getBytes());
      System.out.format("Get('hello') = %s\n",
          new String(value));

      for (int i = 1; i <= 9; ++i) {
        for (int j = 1; j <= 9; ++j) {
          db.put(String.format("%dx%d", i, j).getBytes(),
                 String.format("%d", i * j).getBytes());
        }
      }

      for (int i = 1; i <= 9; ++i) {
        for (int j = 1; j <= 9; ++j) {
          System.out.format("%s ", new String(db.get(
              String.format("%dx%d", i, j).getBytes())));
        }
        System.out.println("");
      }

      // write batch test
      WriteOptions writeOpt = new WriteOptions();
      for (int i = 10; i <= 19; ++i) {
        WriteBatch batch = new WriteBatch();
        for (int j = 10; j <= 19; ++j) {
          batch.put(String.format("%dx%d", i, j).getBytes(),
                    String.format("%d", i * j).getBytes());
        }
        db.write(writeOpt, batch);
        batch.dispose();
      }
      for (int i = 10; i <= 19; ++i) {
        for (int j = 10; j <= 19; ++j) {
          assert(new String(
              db.get(String.format("%dx%d", i, j).getBytes())).equals(
                  String.format("%d", i * j)));
          System.out.format("%s ", new String(db.get(
              String.format("%dx%d", i, j).getBytes())));
        }
        System.out.println("");
      }
      writeOpt.dispose();

      value = db.get("1x1".getBytes());
      assert(value != null);
      value = db.get("world".getBytes());
      assert(value == null);
      value = db.get(readOptions, "world".getBytes());
      assert(value == null);

      byte[] testKey = "asdf".getBytes();
      byte[] testValue =
          "asdfghjkl;'?><MNBVCXZQWERTYUIOP{+_)(*&^%$#@".getBytes();
      db.put(testKey, testValue);
      byte[] testResult = db.get(testKey);
      assert(testResult != null);
      assert(Arrays.equals(testValue, testResult));
      assert(new String(testValue).equals(new String(testResult)));
      testResult = db.get(readOptions, testKey);
      assert(testResult != null);
      assert(Arrays.equals(testValue, testResult));
      assert(new String(testValue).equals(new String(testResult)));

      byte[] insufficientArray = new byte[10];
      byte[] enoughArray = new byte[50];
      int len;
      len = db.get(testKey, insufficientArray);
      assert(len > insufficientArray.length);
      len = db.get("asdfjkl;".getBytes(), enoughArray);
      assert(len == RocksDB.NOT_FOUND);
      len = db.get(testKey, enoughArray);
      assert(len == testValue.length);

      len = db.get(readOptions, testKey, insufficientArray);
      assert(len > insufficientArray.length);
      len = db.get(readOptions, "asdfjkl;".getBytes(), enoughArray);
      assert(len == RocksDB.NOT_FOUND);
      len = db.get(readOptions, testKey, enoughArray);
      assert(len == testValue.length);

      db.remove(testKey);
      len = db.get(testKey, enoughArray);
      assert(len == RocksDB.NOT_FOUND);

      // repeat the test with WriteOptions
      WriteOptions writeOpts = new WriteOptions();
      writeOpts.setSync(true);
      writeOpts.setDisableWAL(true);
      db.put(writeOpts, testKey, testValue);
      len = db.get(testKey, enoughArray);
      assert(len == testValue.length);
      assert(new String(testValue).equals(
          new String(enoughArray, 0, len)));
      writeOpts.dispose();

      try {
        for (TickerType statsType : TickerType.values()) {
          stats.getTickerCount(statsType);
        }
        System.out.println("getTickerCount() passed.");
      } catch (Exception e) {
        System.out.println("Failed in call to getTickerCount()");
        assert(false); //Should never reach here.
      }

      try {
        for (HistogramType histogramType : HistogramType.values()) {
          HistogramData data = stats.geHistogramData(histogramType);
        }
        System.out.println("geHistogramData() passed.");
      } catch (Exception e) {
        System.out.println("Failed in call to geHistogramData()");
        assert(false); //Should never reach here.
      }

      RocksIterator iterator = db.newIterator();

      boolean seekToFirstPassed = false;
      for (iterator.seekToFirst(); iterator.isValid(); iterator.next()) {
        iterator.status();
        assert(iterator.key() != null);
        assert(iterator.value() != null);
        seekToFirstPassed = true;
      }
      if(seekToFirstPassed) {
        System.out.println("iterator seekToFirst tests passed.");
      }

      boolean seekToLastPassed = false;
      for (iterator.seekToLast(); iterator.isValid(); iterator.prev()) {
        iterator.status();
        assert(iterator.key() != null);
        assert(iterator.value() != null);
        seekToLastPassed = true;
      }

      if(seekToLastPassed) {
        System.out.println("iterator seekToLastPassed tests passed.");
      }

      iterator.seekToFirst();
      iterator.seek(iterator.key());
      assert(iterator.key() != null);
      assert(iterator.value() != null);

      System.out.println("iterator seek test passed.");

      iterator.dispose();
      System.out.println("iterator tests passed.");

      iterator = db.newIterator();
      List<byte[]> keys = new ArrayList<byte[]>();
      for (iterator.seekToLast(); iterator.isValid(); iterator.prev()) {
        keys.add(iterator.key());
      }
      iterator.dispose();

      Map<byte[], byte[]> values = db.multiGet(keys);
      assert(values.size() == keys.size());
      for(byte[] value1 : values.values()) {
        assert(value1 != null);
      }

      values = db.multiGet(new ReadOptions(), keys);
      assert(values.size() == keys.size());
      for(byte[] value1 : values.values()) {
        assert(value1 != null);
      }
    } catch (RocksDBException e) {
      System.err.println(e);
    }
    if (db != null) {
      db.close();
    }
    // be sure to dispose c++ pointers
    options.dispose();
    readOptions.dispose();
  }
}
