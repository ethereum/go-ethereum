// Copyright (c) 2014, Facebook, Inc.  All rights reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.
/**
 * Copyright (C) 2011 the original author or authors.
 * See the notice.md file distributed with this work for additional
 * information regarding copyright ownership.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
package org.rocksdb.benchmark;

import java.lang.Runnable;
import java.lang.Math;
import java.io.File;
import java.nio.ByteBuffer;
import java.util.Collection;
import java.util.Date;
import java.util.EnumMap;
import java.util.List;
import java.util.Map;
import java.util.Random;
import java.util.concurrent.TimeUnit;
import java.util.Arrays;
import java.util.ArrayList;
import java.util.concurrent.Callable;
import java.util.concurrent.Executors;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Future;
import java.util.concurrent.TimeUnit;
import org.rocksdb.*;
import org.rocksdb.RocksMemEnv;
import org.rocksdb.util.SizeUnit;

class Stats {
  int id_;
  long start_;
  long finish_;
  double seconds_;
  long done_;
  long found_;
  long lastOpTime_;
  long nextReport_;
  long bytes_;
  StringBuilder message_;
  boolean excludeFromMerge_;

  // TODO(yhchiang): use the following arguments:
  //   (Long)Flag.stats_interval
  //   (Integer)Flag.stats_per_interval

  Stats(int id) {
    id_ = id;
    nextReport_ = 100;
    done_ = 0;
    bytes_ = 0;
    seconds_ = 0;
    start_ = System.nanoTime();
    lastOpTime_ = start_;
    finish_ = start_;
    found_ = 0;
    message_ = new StringBuilder("");
    excludeFromMerge_ = false;
  }

  void merge(final Stats other) {
    if (other.excludeFromMerge_) {
      return;
    }

    done_ += other.done_;
    found_ += other.found_;
    bytes_ += other.bytes_;
    seconds_ += other.seconds_;
    if (other.start_ < start_) start_ = other.start_;
    if (other.finish_ > finish_) finish_ = other.finish_;

    // Just keep the messages from one thread
    if (message_.length() == 0) {
      message_ = other.message_;
    }
  }

  void stop() {
    finish_ = System.nanoTime();
    seconds_ = (double) (finish_ - start_) * 1e-9;
  }

  void addMessage(String msg) {
    if (message_.length() > 0) {
      message_.append(" ");
    }
    message_.append(msg);
  }

  void setId(int id) { id_ = id; }
  void setExcludeFromMerge() { excludeFromMerge_ = true; }

  void finishedSingleOp(int bytes) {
    done_++;
    lastOpTime_ = System.nanoTime();
    bytes_ += bytes;
    if (done_ >= nextReport_) {
      if (nextReport_ < 1000) {
        nextReport_ += 100;
      } else if (nextReport_ < 5000) {
        nextReport_ += 500;
      } else if (nextReport_ < 10000) {
        nextReport_ += 1000;
      } else if (nextReport_ < 50000) {
        nextReport_ += 5000;
      } else if (nextReport_ < 100000) {
        nextReport_ += 10000;
      } else if (nextReport_ < 500000) {
        nextReport_ += 50000;
      } else {
        nextReport_ += 100000;
      }
      System.err.printf("... Task %s finished %d ops%30s\r", id_, done_, "");
    }
  }

  void report(String name) {
    // Pretend at least one op was done in case we are running a benchmark
    // that does not call FinishedSingleOp().
    if (done_ < 1) done_ = 1;

    StringBuilder extra = new StringBuilder("");
    if (bytes_ > 0) {
      // Rate is computed on actual elapsed time, not the sum of per-thread
      // elapsed times.
      double elapsed = (finish_ - start_) * 1e-9;
      extra.append(String.format("%6.1f MB/s", (bytes_ / 1048576.0) / elapsed));
    }
    extra.append(message_.toString());
    double elapsed = (finish_ - start_);
    double throughput = (double) done_ / (elapsed * 1e-9);

    System.out.format("%-12s : %11.3f micros/op %d ops/sec;%s%s\n",
            name, (elapsed * 1e-6) / done_,
            (long) throughput, (extra.length() == 0 ? "" : " "), extra.toString());
  }
}

public class DbBenchmark {
  enum Order {
    SEQUENTIAL,
    RANDOM
  }

  enum DBState {
    FRESH,
    EXISTING
  }

  static {
    RocksDB.loadLibrary();
  }

  abstract class BenchmarkTask implements Callable<Stats> {
    // TODO(yhchiang): use (Integer)Flag.perf_level.
    public BenchmarkTask(
        int tid, long randSeed, long numEntries, long keyRange) {
      tid_ = tid;
      rand_ = new Random(randSeed + tid * 1000);
      numEntries_ = numEntries;
      keyRange_ = keyRange;
      stats_ = new Stats(tid);
    }

    @Override public Stats call() throws RocksDBException {
      stats_.start_ = System.nanoTime();
      runTask();
      stats_.finish_ = System.nanoTime();
      return stats_;
    }

    abstract protected void runTask() throws RocksDBException;

    protected int tid_;
    protected Random rand_;
    protected long numEntries_;
    protected long keyRange_;
    protected Stats stats_;

    protected void getFixedKey(byte[] key, long sn) {
      generateKeyFromLong(key, sn);
    }

    protected void getRandomKey(byte[] key, long range) {
      generateKeyFromLong(key, Math.abs(rand_.nextLong() % range));
    }
  }

  abstract class WriteTask extends BenchmarkTask {
    public WriteTask(
        int tid, long randSeed, long numEntries, long keyRange,
        WriteOptions writeOpt, long entriesPerBatch) {
      super(tid, randSeed, numEntries, keyRange);
      writeOpt_ = writeOpt;
      entriesPerBatch_ = entriesPerBatch;
      maxWritesPerSecond_ = -1;
    }

    public WriteTask(
        int tid, long randSeed, long numEntries, long keyRange,
        WriteOptions writeOpt, long entriesPerBatch, long maxWritesPerSecond) {
      super(tid, randSeed, numEntries, keyRange);
      writeOpt_ = writeOpt;
      entriesPerBatch_ = entriesPerBatch;
      maxWritesPerSecond_ = maxWritesPerSecond;
    }

    @Override public void runTask() throws RocksDBException {
      if (numEntries_ != DbBenchmark.this.num_) {
        stats_.message_.append(String.format(" (%d ops)", numEntries_));
      }
      byte[] key = new byte[keySize_];
      byte[] value = new byte[valueSize_];

      try {
        if (entriesPerBatch_ == 1) {
          for (long i = 0; i < numEntries_; ++i) {
            getKey(key, i, keyRange_);
            DbBenchmark.this.gen_.generate(value);
            db_.put(writeOpt_, key, value);
            stats_.finishedSingleOp(keySize_ + valueSize_);
            writeRateControl(i);
            if (isFinished()) {
              return;
            }
          }
        } else {
          for (long i = 0; i < numEntries_; i += entriesPerBatch_) {
            WriteBatch batch = new WriteBatch();
            for (long j = 0; j < entriesPerBatch_; j++) {
              getKey(key, i + j, keyRange_);
              DbBenchmark.this.gen_.generate(value);
              batch.put(key, value);
              stats_.finishedSingleOp(keySize_ + valueSize_);
            }
            db_.write(writeOpt_, batch);
            batch.dispose();
            writeRateControl(i);
            if (isFinished()) {
              return;
            }
          }
        }
      } catch (InterruptedException e) {
        // thread has been terminated.
      }
    }

    protected void writeRateControl(long writeCount)
        throws InterruptedException {
      if (maxWritesPerSecond_ <= 0) return;
      long minInterval =
          writeCount * TimeUnit.SECONDS.toNanos(1) / maxWritesPerSecond_;
      long interval = System.nanoTime() - stats_.start_;
      if (minInterval - interval > TimeUnit.MILLISECONDS.toNanos(1)) {
        TimeUnit.NANOSECONDS.sleep(minInterval - interval);
      }
    }

    abstract protected void getKey(byte[] key, long id, long range);
    protected WriteOptions writeOpt_;
    protected long entriesPerBatch_;
    protected long maxWritesPerSecond_;
  }

  class WriteSequentialTask extends WriteTask {
    public WriteSequentialTask(
        int tid, long randSeed, long numEntries, long keyRange,
        WriteOptions writeOpt, long entriesPerBatch) {
      super(tid, randSeed, numEntries, keyRange,
            writeOpt, entriesPerBatch);
    }
    public WriteSequentialTask(
        int tid, long randSeed, long numEntries, long keyRange,
        WriteOptions writeOpt, long entriesPerBatch,
        long maxWritesPerSecond) {
      super(tid, randSeed, numEntries, keyRange,
            writeOpt, entriesPerBatch,
            maxWritesPerSecond);
    }
    @Override protected void getKey(byte[] key, long id, long range) {
      getFixedKey(key, id);
    }
  }

  class WriteRandomTask extends WriteTask {
    public WriteRandomTask(
        int tid, long randSeed, long numEntries, long keyRange,
        WriteOptions writeOpt, long entriesPerBatch) {
      super(tid, randSeed, numEntries, keyRange,
            writeOpt, entriesPerBatch);
    }
    public WriteRandomTask(
        int tid, long randSeed, long numEntries, long keyRange,
        WriteOptions writeOpt, long entriesPerBatch,
        long maxWritesPerSecond) {
      super(tid, randSeed, numEntries, keyRange,
            writeOpt, entriesPerBatch,
            maxWritesPerSecond);
    }
    @Override protected void getKey(byte[] key, long id, long range) {
      getRandomKey(key, range);
    }
  }

  class WriteUniqueRandomTask extends WriteTask {
    static final int MAX_BUFFER_SIZE = 10000000;
    public WriteUniqueRandomTask(
        int tid, long randSeed, long numEntries, long keyRange,
        WriteOptions writeOpt, long entriesPerBatch) {
      super(tid, randSeed, numEntries, keyRange,
            writeOpt, entriesPerBatch);
      initRandomKeySequence();
    }
    public WriteUniqueRandomTask(
        int tid, long randSeed, long numEntries, long keyRange,
        WriteOptions writeOpt, long entriesPerBatch,
        long maxWritesPerSecond) {
      super(tid, randSeed, numEntries, keyRange,
            writeOpt, entriesPerBatch,
            maxWritesPerSecond);
      initRandomKeySequence();
    }
    @Override protected void getKey(byte[] key, long id, long range) {
      generateKeyFromLong(key, nextUniqueRandom());
    }

    protected void initRandomKeySequence() {
      bufferSize_ = MAX_BUFFER_SIZE;
      if (bufferSize_ > keyRange_) {
        bufferSize_ = (int) keyRange_;
      }
      currentKeyCount_ = bufferSize_;
      keyBuffer_ = new long[MAX_BUFFER_SIZE];
      for (int k = 0; k < bufferSize_; ++k) {
        keyBuffer_[k] = k;
      }
    }

    /**
     * Semi-randomly return the next unique key.  It is guaranteed to be
     * fully random if keyRange_ <= MAX_BUFFER_SIZE.
     */
    long nextUniqueRandom() {
      if (bufferSize_ == 0) {
        System.err.println("bufferSize_ == 0.");
        return 0;
      }
      int r = rand_.nextInt(bufferSize_);
      // randomly pick one from the keyBuffer
      long randKey = keyBuffer_[r];
      if (currentKeyCount_ < keyRange_) {
        // if we have not yet inserted all keys, insert next new key to [r].
        keyBuffer_[r] = currentKeyCount_++;
      } else {
        // move the last element to [r] and decrease the size by 1.
        keyBuffer_[r] = keyBuffer_[--bufferSize_];
      }
      return randKey;
    }

    int bufferSize_;
    long currentKeyCount_;
    long[] keyBuffer_;
  }

  class ReadRandomTask extends BenchmarkTask {
    public ReadRandomTask(
        int tid, long randSeed, long numEntries, long keyRange) {
      super(tid, randSeed, numEntries, keyRange);
    }
    @Override public void runTask() throws RocksDBException {
      byte[] key = new byte[keySize_];
      byte[] value = new byte[valueSize_];
      for (long i = 0; i < numEntries_; i++) {
        getRandomKey(key, keyRange_);
        int len = db_.get(key, value);
        if (len != RocksDB.NOT_FOUND) {
          stats_.found_++;
          stats_.finishedSingleOp(keySize_ + valueSize_);
        } else {
          stats_.finishedSingleOp(keySize_);
        }
        if (isFinished()) {
          return;
        }
      }
    }
  }

  class ReadSequentialTask extends BenchmarkTask {
    public ReadSequentialTask(
        int tid, long randSeed, long numEntries, long keyRange) {
      super(tid, randSeed, numEntries, keyRange);
    }
    @Override public void runTask() throws RocksDBException {
      RocksIterator iter = db_.newIterator();
      long i;
      for (iter.seekToFirst(), i = 0;
           iter.isValid() && i < numEntries_;
           iter.next(), ++i) {
        stats_.found_++;
        stats_.finishedSingleOp(iter.key().length + iter.value().length);
        if (isFinished()) {
          iter.dispose();
          return;
        }
      }
      iter.dispose();
    }
  }

  public DbBenchmark(Map<Flag, Object> flags) throws Exception {
    benchmarks_ = (List<String>) flags.get(Flag.benchmarks);
    num_ = (Integer) flags.get(Flag.num);
    threadNum_ = (Integer) flags.get(Flag.threads);
    reads_ = (Integer) (flags.get(Flag.reads) == null ?
        flags.get(Flag.num) : flags.get(Flag.reads));
    keySize_ = (Integer) flags.get(Flag.key_size);
    valueSize_ = (Integer) flags.get(Flag.value_size);
    compressionRatio_ = (Double) flags.get(Flag.compression_ratio);
    useExisting_ = (Boolean) flags.get(Flag.use_existing_db);
    randSeed_ = (Long) flags.get(Flag.seed);
    databaseDir_ = (String) flags.get(Flag.db);
    writesPerSeconds_ = (Integer) flags.get(Flag.writes_per_second);
    memtable_ = (String) flags.get(Flag.memtablerep);
    maxWriteBufferNumber_ = (Integer) flags.get(Flag.max_write_buffer_number);
    prefixSize_ = (Integer) flags.get(Flag.prefix_size);
    keysPerPrefix_ = (Integer) flags.get(Flag.keys_per_prefix);
    hashBucketCount_ = (Long) flags.get(Flag.hash_bucket_count);
    usePlainTable_ = (Boolean) flags.get(Flag.use_plain_table);
    useMemenv_ = (Boolean) flags.get(Flag.use_mem_env);
    flags_ = flags;
    finishLock_ = new Object();
    // options.setPrefixSize((Integer)flags_.get(Flag.prefix_size));
    // options.setKeysPerPrefix((Long)flags_.get(Flag.keys_per_prefix));
    compressionType_ = (String) flags.get(Flag.compression_type);
    compression_ = CompressionType.NO_COMPRESSION;
    try {
      if (compressionType_!=null) {
          final CompressionType compressionType =
              CompressionType.getCompressionType(compressionType_);
          if (compressionType != null &&
              compressionType != CompressionType.NO_COMPRESSION) {
            System.loadLibrary(compressionType.getLibraryName());
          }

      }
    } catch (UnsatisfiedLinkError e) {
      System.err.format("Unable to load %s library:%s%n" +
                        "No compression is used.%n",
          compressionType_, e.toString());
      compressionType_ = "none";
    }
    gen_ = new RandomGenerator(randSeed_, compressionRatio_);
  }

  private void prepareReadOptions(ReadOptions options) {
    options.setVerifyChecksums((Boolean)flags_.get(Flag.verify_checksum));
    options.setTailing((Boolean)flags_.get(Flag.use_tailing_iterator));
  }

  private void prepareWriteOptions(WriteOptions options) {
    options.setSync((Boolean)flags_.get(Flag.sync));
    options.setDisableWAL((Boolean)flags_.get(Flag.disable_wal));
  }

  private void prepareOptions(Options options) throws RocksDBException {
    if (!useExisting_) {
      options.setCreateIfMissing(true);
    } else {
      options.setCreateIfMissing(false);
    }
    if (useMemenv_) {
      options.setEnv(new RocksMemEnv());
    }
    switch (memtable_) {
      case "skip_list":
        options.setMemTableConfig(new SkipListMemTableConfig());
        break;
      case "vector":
        options.setMemTableConfig(new VectorMemTableConfig());
        break;
      case "hash_linkedlist":
        options.setMemTableConfig(
            new HashLinkedListMemTableConfig()
                .setBucketCount(hashBucketCount_));
        options.useFixedLengthPrefixExtractor(prefixSize_);
        break;
      case "hash_skiplist":
      case "prefix_hash":
        options.setMemTableConfig(
            new HashSkipListMemTableConfig()
                .setBucketCount(hashBucketCount_));
        options.useFixedLengthPrefixExtractor(prefixSize_);
        break;
      default:
        System.err.format(
            "unable to detect the specified memtable, " +
                "use the default memtable factory %s%n",
            options.memTableFactoryName());
        break;
    }
    if (usePlainTable_) {
      options.setTableFormatConfig(
          new PlainTableConfig().setKeySize(keySize_));
    } else {
      BlockBasedTableConfig table_options = new BlockBasedTableConfig();
      table_options.setBlockSize((Long)flags_.get(Flag.block_size))
                   .setBlockCacheSize((Long)flags_.get(Flag.cache_size))
                   .setCacheNumShardBits(
                      (Integer)flags_.get(Flag.cache_numshardbits));
      options.setTableFormatConfig(table_options);
    }
    options.setWriteBufferSize(
        (Long)flags_.get(Flag.write_buffer_size));
    options.setMaxWriteBufferNumber(
        (Integer)flags_.get(Flag.max_write_buffer_number));
    options.setMaxBackgroundCompactions(
        (Integer)flags_.get(Flag.max_background_compactions));
    options.getEnv().setBackgroundThreads(
        (Integer)flags_.get(Flag.max_background_compactions));
    options.setMaxBackgroundFlushes(
        (Integer)flags_.get(Flag.max_background_flushes));
    options.setMaxOpenFiles(
        (Integer)flags_.get(Flag.open_files));
    options.setDisableDataSync(
        (Boolean)flags_.get(Flag.disable_data_sync));
    options.setUseFsync(
        (Boolean)flags_.get(Flag.use_fsync));
    options.setWalDir(
        (String)flags_.get(Flag.wal_dir));
    options.setDeleteObsoleteFilesPeriodMicros(
        (Integer)flags_.get(Flag.delete_obsolete_files_period_micros));
    options.setTableCacheNumshardbits(
        (Integer)flags_.get(Flag.table_cache_numshardbits));
    options.setAllowMmapReads(
        (Boolean)flags_.get(Flag.mmap_read));
    options.setAllowMmapWrites(
        (Boolean)flags_.get(Flag.mmap_write));
    options.setAdviseRandomOnOpen(
        (Boolean)flags_.get(Flag.advise_random_on_open));
    options.setUseAdaptiveMutex(
        (Boolean)flags_.get(Flag.use_adaptive_mutex));
    options.setBytesPerSync(
        (Long)flags_.get(Flag.bytes_per_sync));
    options.setBloomLocality(
        (Integer)flags_.get(Flag.bloom_locality));
    options.setMinWriteBufferNumberToMerge(
        (Integer)flags_.get(Flag.min_write_buffer_number_to_merge));
    options.setMemtablePrefixBloomBits(
        (Integer)flags_.get(Flag.memtable_bloom_bits));
    options.setNumLevels(
        (Integer)flags_.get(Flag.num_levels));
    options.setTargetFileSizeBase(
        (Integer)flags_.get(Flag.target_file_size_base));
    options.setTargetFileSizeMultiplier(
        (Integer)flags_.get(Flag.target_file_size_multiplier));
    options.setMaxBytesForLevelBase(
        (Integer)flags_.get(Flag.max_bytes_for_level_base));
    options.setMaxBytesForLevelMultiplier(
        (Integer)flags_.get(Flag.max_bytes_for_level_multiplier));
    options.setLevelZeroStopWritesTrigger(
        (Integer)flags_.get(Flag.level0_stop_writes_trigger));
    options.setLevelZeroSlowdownWritesTrigger(
        (Integer)flags_.get(Flag.level0_slowdown_writes_trigger));
    options.setLevelZeroFileNumCompactionTrigger(
        (Integer)flags_.get(Flag.level0_file_num_compaction_trigger));
    options.setSoftRateLimit(
        (Double)flags_.get(Flag.soft_rate_limit));
    options.setHardRateLimit(
        (Double)flags_.get(Flag.hard_rate_limit));
    options.setRateLimitDelayMaxMilliseconds(
        (Integer)flags_.get(Flag.rate_limit_delay_max_milliseconds));
    options.setMaxGrandparentOverlapFactor(
        (Integer)flags_.get(Flag.max_grandparent_overlap_factor));
    options.setDisableAutoCompactions(
        (Boolean)flags_.get(Flag.disable_auto_compactions));
    options.setSourceCompactionFactor(
        (Integer)flags_.get(Flag.source_compaction_factor));
    options.setFilterDeletes(
        (Boolean)flags_.get(Flag.filter_deletes));
    options.setMaxSuccessiveMerges(
        (Integer)flags_.get(Flag.max_successive_merges));
    options.setWalTtlSeconds((Long)flags_.get(Flag.wal_ttl_seconds));
    options.setWalSizeLimitMB((Long)flags_.get(Flag.wal_size_limit_MB));
    /* TODO(yhchiang): enable the following parameters
    options.setCompressionType((String)flags_.get(Flag.compression_type));
    options.setCompressionLevel((Integer)flags_.get(Flag.compression_level));
    options.setMinLevelToCompress((Integer)flags_.get(Flag.min_level_to_compress));
    options.setHdfs((String)flags_.get(Flag.hdfs)); // env
    options.setStatistics((Boolean)flags_.get(Flag.statistics));
    options.setUniversalSizeRatio(
        (Integer)flags_.get(Flag.universal_size_ratio));
    options.setUniversalMinMergeWidth(
        (Integer)flags_.get(Flag.universal_min_merge_width));
    options.setUniversalMaxMergeWidth(
        (Integer)flags_.get(Flag.universal_max_merge_width));
    options.setUniversalMaxSizeAmplificationPercent(
        (Integer)flags_.get(Flag.universal_max_size_amplification_percent));
    options.setUniversalCompressionSizePercent(
        (Integer)flags_.get(Flag.universal_compression_size_percent));
    // TODO(yhchiang): add RocksDB.openForReadOnly() to enable Flag.readonly
    // TODO(yhchiang): enable Flag.merge_operator by switch
    options.setAccessHintOnCompactionStart(
        (String)flags_.get(Flag.compaction_fadvice));
    // available values of fadvice are "NONE", "NORMAL", "SEQUENTIAL", "WILLNEED" for fadvice
    */
  }

  private void run() throws RocksDBException {
    if (!useExisting_) {
      destroyDb();
    }
    Options options = new Options();
    prepareOptions(options);
    open(options);

    printHeader(options);

    for (String benchmark : benchmarks_) {
      List<Callable<Stats>> tasks = new ArrayList<Callable<Stats>>();
      List<Callable<Stats>> bgTasks = new ArrayList<Callable<Stats>>();
      WriteOptions writeOpt = new WriteOptions();
      prepareWriteOptions(writeOpt);
      ReadOptions readOpt = new ReadOptions();
      prepareReadOptions(readOpt);
      int currentTaskId = 0;
      boolean known = true;

      switch (benchmark) {
        case "fillseq":
          tasks.add(new WriteSequentialTask(
              currentTaskId++, randSeed_, num_, num_, writeOpt, 1));
          break;
        case "fillbatch":
          tasks.add(new WriteRandomTask(
              currentTaskId++, randSeed_, num_ / 1000, num_, writeOpt, 1000));
          break;
        case "fillrandom":
          tasks.add(new WriteRandomTask(
              currentTaskId++, randSeed_, num_, num_, writeOpt, 1));
          break;
        case "filluniquerandom":
          tasks.add(new WriteUniqueRandomTask(
              currentTaskId++, randSeed_, num_, num_, writeOpt, 1));
          break;
        case "fillsync":
          writeOpt.setSync(true);
          tasks.add(new WriteRandomTask(
              currentTaskId++, randSeed_, num_ / 1000, num_ / 1000,
              writeOpt, 1));
          break;
        case "readseq":
          for (int t = 0; t < threadNum_; ++t) {
            tasks.add(new ReadSequentialTask(
                currentTaskId++, randSeed_, reads_ / threadNum_, num_));
          }
          break;
        case "readrandom":
          for (int t = 0; t < threadNum_; ++t) {
            tasks.add(new ReadRandomTask(
                currentTaskId++, randSeed_, reads_ / threadNum_, num_));
          }
          break;
        case "readwhilewriting":
          WriteTask writeTask = new WriteRandomTask(
              -1, randSeed_, Long.MAX_VALUE, num_, writeOpt, 1, writesPerSeconds_);
          writeTask.stats_.setExcludeFromMerge();
          bgTasks.add(writeTask);
          for (int t = 0; t < threadNum_; ++t) {
            tasks.add(new ReadRandomTask(
                currentTaskId++, randSeed_, reads_ / threadNum_, num_));
          }
          break;
        case "readhot":
          for (int t = 0; t < threadNum_; ++t) {
            tasks.add(new ReadRandomTask(
                currentTaskId++, randSeed_, reads_ / threadNum_, num_ / 100));
          }
          break;
        case "delete":
          destroyDb();
          open(options);
          break;
        default:
          known = false;
          System.err.println("Unknown benchmark: " + benchmark);
          break;
      }
      if (known) {
        ExecutorService executor = Executors.newCachedThreadPool();
        ExecutorService bgExecutor = Executors.newCachedThreadPool();
        try {
          // measure only the main executor time
          List<Future<Stats>> bgResults = new ArrayList<Future<Stats>>();
          for (Callable bgTask : bgTasks) {
            bgResults.add(bgExecutor.submit(bgTask));
          }
          start();
          List<Future<Stats>> results = executor.invokeAll(tasks);
          executor.shutdown();
          boolean finished = executor.awaitTermination(10, TimeUnit.SECONDS);
          if (!finished) {
            System.out.format(
                "Benchmark %s was not finished before timeout.",
                benchmark);
            executor.shutdownNow();
          }
          setFinished(true);
          bgExecutor.shutdown();
          finished = bgExecutor.awaitTermination(10, TimeUnit.SECONDS);
          if (!finished) {
            System.out.format(
                "Benchmark %s was not finished before timeout.",
                benchmark);
            bgExecutor.shutdownNow();
          }

          stop(benchmark, results, currentTaskId);
        } catch (InterruptedException e) {
          System.err.println(e);
        }
      }
      writeOpt.dispose();
      readOpt.dispose();
    }
    options.dispose();
    db_.close();
  }

  private void printHeader(Options options) {
    int kKeySize = 16;
    System.out.printf("Keys:     %d bytes each\n", kKeySize);
    System.out.printf("Values:   %d bytes each (%d bytes after compression)\n",
        valueSize_,
        (int) (valueSize_ * compressionRatio_ + 0.5));
    System.out.printf("Entries:  %d\n", num_);
    System.out.printf("RawSize:  %.1f MB (estimated)\n",
        ((double)(kKeySize + valueSize_) * num_) / SizeUnit.MB);
    System.out.printf("FileSize:   %.1f MB (estimated)\n",
        (((kKeySize + valueSize_ * compressionRatio_) * num_) / SizeUnit.MB));
    System.out.format("Memtable Factory: %s%n", options.memTableFactoryName());
    System.out.format("Prefix:   %d bytes%n", prefixSize_);
    System.out.format("Compression: %s%n", compressionType_);
    printWarnings();
    System.out.printf("------------------------------------------------\n");
  }

  void printWarnings() {
    boolean assertsEnabled = false;
    assert assertsEnabled = true; // Intentional side effect!!!
    if (assertsEnabled) {
      System.out.printf(
          "WARNING: Assertions are enabled; benchmarks unnecessarily slow\n");
    }
  }

  private void open(Options options) throws RocksDBException {
    db_ = RocksDB.open(options, databaseDir_);
  }

  private void start() {
    setFinished(false);
    startTime_ = System.nanoTime();
  }

  private void stop(
      String benchmark, List<Future<Stats>> results, int concurrentThreads) {
    long endTime = System.nanoTime();
    double elapsedSeconds =
        1.0d * (endTime - startTime_) / TimeUnit.SECONDS.toNanos(1);

    Stats stats = new Stats(-1);
    int taskFinishedCount = 0;
    for (Future<Stats> result : results) {
      if (result.isDone()) {
        try {
          Stats taskStats = result.get(3, TimeUnit.SECONDS);
          if (!result.isCancelled()) {
            taskFinishedCount++;
          }
          stats.merge(taskStats);
        } catch (Exception e) {
          // then it's not successful, the output will indicate this
        }
      }
    }
    String extra = "";
    if (benchmark.indexOf("read") >= 0) {
      extra = String.format(" %d / %d found; ", stats.found_, stats.done_);
    } else {
      extra = String.format(" %d ops done; ", stats.done_);
    }

    System.out.printf(
        "%-16s : %11.5f micros/op; %6.1f MB/s;%s %d / %d task(s) finished.\n",
        benchmark, elapsedSeconds / stats.done_ * 1e6,
        (stats.bytes_ / 1048576.0) / elapsedSeconds, extra,
        taskFinishedCount, concurrentThreads);
  }

  public void generateKeyFromLong(byte[] slice, long n) {
    assert(n >= 0);
    int startPos = 0;

    if (keysPerPrefix_ > 0) {
      long numPrefix = (num_ + keysPerPrefix_ - 1) / keysPerPrefix_;
      long prefix = n % numPrefix;
      int bytesToFill = Math.min(prefixSize_, 8);
      for (int i = 0; i < bytesToFill; ++i) {
        slice[i] = (byte) (prefix % 256);
        prefix /= 256;
      }
      for (int i = 8; i < bytesToFill; ++i) {
        slice[i] = '0';
      }
      startPos = bytesToFill;
    }

    for (int i = slice.length - 1; i >= startPos; --i) {
      slice[i] = (byte) ('0' + (n % 10));
      n /= 10;
    }
  }

  private void destroyDb() {
    if (db_ != null) {
      db_.close();
    }
    // TODO(yhchiang): develop our own FileUtil
    // FileUtil.deleteDir(databaseDir_);
  }

  private void printStats() {
  }

  static void printHelp() {
    System.out.println("usage:");
    for (Flag flag : Flag.values()) {
      System.out.format("  --%s%n\t%s%n",
          flag.name(),
          flag.desc());
      if (flag.getDefaultValue() != null) {
        System.out.format("\tDEFAULT: %s%n",
            flag.getDefaultValue().toString());
      }
    }
  }

  public static void main(String[] args) throws Exception {
    Map<Flag, Object> flags = new EnumMap<Flag, Object>(Flag.class);
    for (Flag flag : Flag.values()) {
      if (flag.getDefaultValue() != null) {
        flags.put(flag, flag.getDefaultValue());
      }
    }
    for (String arg : args) {
      boolean valid = false;
      if (arg.equals("--help") || arg.equals("-h")) {
        printHelp();
        System.exit(0);
      }
      if (arg.startsWith("--")) {
        try {
          String[] parts = arg.substring(2).split("=");
          if (parts.length >= 1) {
            Flag key = Flag.valueOf(parts[0]);
            if (key != null) {
              Object value = null;
              if (parts.length >= 2) {
                value = key.parseValue(parts[1]);
              }
              flags.put(key, value);
              valid = true;
            }
          }
        }
        catch (Exception e) {
        }
      }
      if (!valid) {
        System.err.println("Invalid argument " + arg);
        System.exit(1);
      }
    }
    new DbBenchmark(flags).run();
  }

  private enum Flag {
    benchmarks(
        Arrays.asList(
            "fillseq",
            "readrandom",
            "fillrandom"),
        "Comma-separated list of operations to run in the specified order\n" +
        "\tActual benchmarks:\n" +
        "\t\tfillseq          -- write N values in sequential key order in async mode.\n" +
        "\t\tfillrandom       -- write N values in random key order in async mode.\n" +
        "\t\tfillbatch        -- write N/1000 batch where each batch has 1000 values\n" +
        "\t\t                   in random key order in sync mode.\n" +
        "\t\tfillsync         -- write N/100 values in random key order in sync mode.\n" +
        "\t\tfill100K         -- write N/1000 100K values in random order in async mode.\n" +
        "\t\treadseq          -- read N times sequentially.\n" +
        "\t\treadrandom       -- read N times in random order.\n" +
        "\t\treadhot          -- read N times in random order from 1% section of DB.\n" +
        "\t\treadwhilewriting -- measure the read performance of multiple readers\n" +
        "\t\t                   with a bg single writer.  The write rate of the bg\n" +
        "\t\t                   is capped by --writes_per_second.\n" +
        "\tMeta Operations:\n" +
        "\t\tdelete            -- delete DB") {
      @Override public Object parseValue(String value) {
        return new ArrayList<String>(Arrays.asList(value.split(",")));
      }
    },
    compression_ratio(0.5d,
        "Arrange to generate values that shrink to this fraction of\n" +
        "\ttheir original size after compression.") {
      @Override public Object parseValue(String value) {
        return Double.parseDouble(value);
      }
    },
    use_existing_db(false,
        "If true, do not destroy the existing database.  If you set this\n" +
        "\tflag and also specify a benchmark that wants a fresh database,\n" +
        "\tthat benchmark will fail.") {
      @Override public Object parseValue(String value) {
        return parseBoolean(value);
      }
    },
    num(1000000,
        "Number of key/values to place in database.") {
      @Override public Object parseValue(String value) {
        return Integer.parseInt(value);
      }
    },
    threads(1,
        "Number of concurrent threads to run.") {
      @Override public Object parseValue(String value) {
        return Integer.parseInt(value);
      }
    },
    reads(null,
        "Number of read operations to do.  If negative, do --nums reads.") {
      @Override public Object parseValue(String value) {
        return Integer.parseInt(value);
      }
    },
    key_size(16,
        "The size of each key in bytes.") {
      @Override public Object parseValue(String value) {
        return Integer.parseInt(value);
      }
    },
    value_size(100,
        "The size of each value in bytes.") {
      @Override public Object parseValue(String value) {
        return Integer.parseInt(value);
      }
    },
    write_buffer_size(4 * SizeUnit.MB,
        "Number of bytes to buffer in memtable before compacting\n" +
        "\t(initialized to default value by 'main'.)") {
      @Override public Object parseValue(String value) {
        return Long.parseLong(value);
      }
    },
    max_write_buffer_number(2,
             "The number of in-memory memtables. Each memtable is of size\n" +
             "\twrite_buffer_size.") {
      @Override public Object parseValue(String value) {
        return Integer.parseInt(value);
      }
    },
    prefix_size(0, "Controls the prefix size for HashSkipList, HashLinkedList,\n" +
                   "\tand plain table.") {
      @Override public Object parseValue(String value) {
        return Integer.parseInt(value);
      }
    },
    keys_per_prefix(0, "Controls the average number of keys generated\n" +
             "\tper prefix, 0 means no special handling of the prefix,\n" +
             "\ti.e. use the prefix comes with the generated random number.") {
      @Override public Object parseValue(String value) {
        return Integer.parseInt(value);
      }
    },
    memtablerep("skip_list",
        "The memtable format.  Available options are\n" +
        "\tskip_list,\n" +
        "\tvector,\n" +
        "\thash_linkedlist,\n" +
        "\thash_skiplist (prefix_hash.)") {
      @Override public Object parseValue(String value) {
        return value;
      }
    },
    hash_bucket_count(SizeUnit.MB,
        "The number of hash buckets used in the hash-bucket-based\n" +
        "\tmemtables.  Memtables that currently support this argument are\n" +
        "\thash_linkedlist and hash_skiplist.") {
      @Override public Object parseValue(String value) {
        return Long.parseLong(value);
      }
    },
    writes_per_second(10000,
        "The write-rate of the background writer used in the\n" +
        "\t`readwhilewriting` benchmark.  Non-positive number indicates\n" +
        "\tusing an unbounded write-rate in `readwhilewriting` benchmark.") {
      @Override public Object parseValue(String value) {
        return Integer.parseInt(value);
      }
    },
    use_plain_table(false,
        "Use plain-table sst format.") {
      @Override public Object parseValue(String value) {
        return parseBoolean(value);
      }
    },
    cache_size(-1L,
        "Number of bytes to use as a cache of uncompressed data.\n" +
        "\tNegative means use default settings.") {
      @Override public Object parseValue(String value) {
        return Long.parseLong(value);
      }
    },
    seed(0L,
        "Seed base for random number generators.") {
      @Override public Object parseValue(String value) {
        return Long.parseLong(value);
      }
    },
    num_levels(7,
        "The total number of levels.") {
      @Override public Object parseValue(String value) {
        return Integer.parseInt(value);
      }
    },
    numdistinct(1000,
        "Number of distinct keys to use. Used in RandomWithVerify to\n" +
        "\tread/write on fewer keys so that gets are more likely to find the\n" +
        "\tkey and puts are more likely to update the same key.") {
      @Override public Object parseValue(String value) {
        return Long.parseLong(value);
      }
    },
    merge_keys(-1,
        "Number of distinct keys to use for MergeRandom and\n" +
        "\tReadRandomMergeRandom.\n" +
        "\tIf negative, there will be FLAGS_num keys.") {
      @Override public Object parseValue(String value) {
        return Long.parseLong(value);
      }
    },
    bloom_locality(0,"Control bloom filter probes locality.") {
      @Override public Object parseValue(String value) {
        return Integer.parseInt(value);
      }
    },
    duration(0,"Time in seconds for the random-ops tests to run.\n" +
        "\tWhen 0 then num & reads determine the test duration.") {
      @Override public Object parseValue(String value) {
        return Integer.parseInt(value);
      }
    },
    num_multi_db(0,
        "Number of DBs used in the benchmark. 0 means single DB.") {
      @Override public Object parseValue(String value) {
        return Integer.parseInt(value);
      }
    },
    histogram(false,"Print histogram of operation timings.") {
      @Override public Object parseValue(String value) {
        return parseBoolean(value);
      }
    },
    min_write_buffer_number_to_merge(
        defaultOptions_.minWriteBufferNumberToMerge(),
        "The minimum number of write buffers that will be merged together\n" +
        "\tbefore writing to storage. This is cheap because it is an\n" +
        "\tin-memory merge. If this feature is not enabled, then all these\n" +
        "\twrite buffers are flushed to L0 as separate files and this\n" +
        "\tincreases read amplification because a get request has to check\n" +
        "\tin all of these files. Also, an in-memory merge may result in\n" +
        "\twriting less data to storage if there are duplicate records\n" +
        "\tin each of these individual write buffers.") {
      @Override public Object parseValue(String value) {
        return Integer.parseInt(value);
      }
    },
    max_background_compactions(
        defaultOptions_.maxBackgroundCompactions(),
        "The maximum number of concurrent background compactions\n" +
        "\tthat can occur in parallel.") {
      @Override public Object parseValue(String value) {
        return Integer.parseInt(value);
      }
    },
    max_background_flushes(
        defaultOptions_.maxBackgroundFlushes(),
        "The maximum number of concurrent background flushes\n" +
        "\tthat can occur in parallel.") {
      @Override public Object parseValue(String value) {
        return Integer.parseInt(value);
      }
    },
    /* TODO(yhchiang): enable the following
    compaction_style((int32_t) defaultOptions_.compactionStyle(),
        "style of compaction: level-based vs universal.") {
      @Override public Object parseValue(String value) {
        return Integer.parseInt(value);
      }
    },*/
    universal_size_ratio(0,
        "Percentage flexibility while comparing file size\n" +
        "\t(for universal compaction only).") {
      @Override public Object parseValue(String value) {
        return Integer.parseInt(value);
      }
    },
    universal_min_merge_width(0,"The minimum number of files in a\n" +
        "\tsingle compaction run (for universal compaction only).") {
      @Override public Object parseValue(String value) {
        return Integer.parseInt(value);
      }
    },
    universal_max_merge_width(0,"The max number of files to compact\n" +
        "\tin universal style compaction.") {
      @Override public Object parseValue(String value) {
        return Integer.parseInt(value);
      }
    },
    universal_max_size_amplification_percent(0,
        "The max size amplification for universal style compaction.") {
      @Override public Object parseValue(String value) {
        return Integer.parseInt(value);
      }
    },
    universal_compression_size_percent(-1,
        "The percentage of the database to compress for universal\n" +
        "\tcompaction. -1 means compress everything.") {
      @Override public Object parseValue(String value) {
        return Integer.parseInt(value);
      }
    },
    block_size(defaultBlockBasedTableOptions_.blockSize(),
        "Number of bytes in a block.") {
      @Override public Object parseValue(String value) {
        return Long.parseLong(value);
      }
    },
    compressed_cache_size(-1,
        "Number of bytes to use as a cache of compressed data.") {
      @Override public Object parseValue(String value) {
        return Long.parseLong(value);
      }
    },
    open_files(defaultOptions_.maxOpenFiles(),
        "Maximum number of files to keep open at the same time\n" +
        "\t(use default if == 0)") {
      @Override public Object parseValue(String value) {
        return Integer.parseInt(value);
      }
    },
    bloom_bits(-1,"Bloom filter bits per key. Negative means\n" +
        "\tuse default settings.") {
      @Override public Object parseValue(String value) {
        return Integer.parseInt(value);
      }
    },
    memtable_bloom_bits(0,"Bloom filter bits per key for memtable.\n" +
        "\tNegative means no bloom filter.") {
      @Override public Object parseValue(String value) {
        return Integer.parseInt(value);
      }
    },
    cache_numshardbits(-1,"Number of shards for the block cache\n" +
        "\tis 2 ** cache_numshardbits. Negative means use default settings.\n" +
        "\tThis is applied only if FLAGS_cache_size is non-negative.") {
      @Override public Object parseValue(String value) {
        return Integer.parseInt(value);
      }
    },
    verify_checksum(false,"Verify checksum for every block read\n" +
        "\tfrom storage.") {
      @Override public Object parseValue(String value) {
        return parseBoolean(value);
      }
    },
    statistics(false,"Database statistics.") {
      @Override public Object parseValue(String value) {
        return parseBoolean(value);
      }
    },
    writes(-1,"Number of write operations to do. If negative, do\n" +
        "\t--num reads.") {
      @Override public Object parseValue(String value) {
        return Long.parseLong(value);
      }
    },
    sync(false,"Sync all writes to disk.") {
      @Override public Object parseValue(String value) {
        return parseBoolean(value);
      }
    },
    disable_data_sync(false,"If true, do not wait until data is\n" +
        "\tsynced to disk.") {
      @Override public Object parseValue(String value) {
        return parseBoolean(value);
      }
    },
    use_fsync(false,"If true, issue fsync instead of fdatasync.") {
      @Override public Object parseValue(String value) {
        return parseBoolean(value);
      }
    },
    disable_wal(false,"If true, do not write WAL for write.") {
      @Override public Object parseValue(String value) {
        return parseBoolean(value);
      }
    },
    wal_dir("", "If not empty, use the given dir for WAL.") {
      @Override public Object parseValue(String value) {
        return value;
      }
    },
    target_file_size_base(2 * 1048576,"Target file size at level-1") {
      @Override public Object parseValue(String value) {
        return Integer.parseInt(value);
      }
    },
    target_file_size_multiplier(1,
        "A multiplier to compute target level-N file size (N >= 2)") {
      @Override public Object parseValue(String value) {
        return Integer.parseInt(value);
      }
    },
    max_bytes_for_level_base(10 * 1048576,
      "Max bytes for level-1") {
      @Override public Object parseValue(String value) {
        return Integer.parseInt(value);
      }
    },
    max_bytes_for_level_multiplier(10,
        "A multiplier to compute max bytes for level-N (N >= 2)") {
      @Override public Object parseValue(String value) {
        return Integer.parseInt(value);
      }
    },
    level0_stop_writes_trigger(12,"Number of files in level-0\n" +
        "\tthat will trigger put stop.") {
      @Override public Object parseValue(String value) {
        return Integer.parseInt(value);
      }
    },
    level0_slowdown_writes_trigger(8,"Number of files in level-0\n" +
        "\tthat will slow down writes.") {
      @Override public Object parseValue(String value) {
        return Integer.parseInt(value);
      }
    },
    level0_file_num_compaction_trigger(4,"Number of files in level-0\n" +
        "\twhen compactions start.") {
      @Override public Object parseValue(String value) {
        return Integer.parseInt(value);
      }
    },
    readwritepercent(90,"Ratio of reads to reads/writes (expressed\n" +
        "\tas percentage) for the ReadRandomWriteRandom workload. The\n" +
        "\tdefault value 90 means 90% operations out of all reads and writes\n" +
        "\toperations are reads. In other words, 9 gets for every 1 put.") {
      @Override public Object parseValue(String value) {
        return Integer.parseInt(value);
      }
    },
    mergereadpercent(70,"Ratio of merges to merges&reads (expressed\n" +
        "\tas percentage) for the ReadRandomMergeRandom workload. The\n" +
        "\tdefault value 70 means 70% out of all read and merge operations\n" +
        "\tare merges. In other words, 7 merges for every 3 gets.") {
      @Override public Object parseValue(String value) {
        return Integer.parseInt(value);
      }
    },
    deletepercent(2,"Percentage of deletes out of reads/writes/\n" +
        "\tdeletes (used in RandomWithVerify only). RandomWithVerify\n" +
        "\tcalculates writepercent as (100 - FLAGS_readwritepercent -\n" +
        "\tdeletepercent), so deletepercent must be smaller than (100 -\n" +
        "\tFLAGS_readwritepercent)") {
      @Override public Object parseValue(String value) {
        return Integer.parseInt(value);
      }
    },
    delete_obsolete_files_period_micros(0,"Option to delete\n" +
        "\tobsolete files periodically. 0 means that obsolete files are\n" +
        "\tdeleted after every compaction run.") {
      @Override public Object parseValue(String value) {
        return Integer.parseInt(value);
      }
    },
    compression_type("snappy",
        "Algorithm used to compress the database.") {
      @Override public Object parseValue(String value) {
        return value;
      }
    },
    compression_level(-1,
        "Compression level. For zlib this should be -1 for the\n" +
        "\tdefault level, or between 0 and 9.") {
      @Override public Object parseValue(String value) {
        return Integer.parseInt(value);
      }
    },
    min_level_to_compress(-1,"If non-negative, compression starts\n" +
        "\tfrom this level. Levels with number < min_level_to_compress are\n" +
        "\tnot compressed. Otherwise, apply compression_type to\n" +
        "\tall levels.") {
      @Override public Object parseValue(String value) {
        return Integer.parseInt(value);
      }
    },
    table_cache_numshardbits(4,"") {
      @Override public Object parseValue(String value) {
        return Integer.parseInt(value);
      }
    },
    stats_interval(0,"Stats are reported every N operations when\n" +
        "\tthis is greater than zero. When 0 the interval grows over time.") {
      @Override public Object parseValue(String value) {
        return Long.parseLong(value);
      }
    },
    stats_per_interval(0,"Reports additional stats per interval when\n" +
        "\tthis is greater than 0.") {
      @Override public Object parseValue(String value) {
        return Integer.parseInt(value);
      }
    },
    perf_level(0,"Level of perf collection.") {
      @Override public Object parseValue(String value) {
        return Integer.parseInt(value);
      }
    },
    soft_rate_limit(0.0,"") {
      @Override public Object parseValue(String value) {
        return Double.parseDouble(value);
      }
    },
    hard_rate_limit(0.0,"When not equal to 0 this make threads\n" +
        "\tsleep at each stats reporting interval until the compaction\n" +
        "\tscore for all levels is less than or equal to this value.") {
      @Override public Object parseValue(String value) {
        return Double.parseDouble(value);
      }
    },
    rate_limit_delay_max_milliseconds(1000,
        "When hard_rate_limit is set then this is the max time a put will\n" +
        "\tbe stalled.") {
      @Override public Object parseValue(String value) {
        return Integer.parseInt(value);
      }
    },
    max_grandparent_overlap_factor(10,"Control maximum bytes of\n" +
        "\toverlaps in grandparent (i.e., level+2) before we stop building a\n" +
        "\tsingle file in a level->level+1 compaction.") {
      @Override public Object parseValue(String value) {
        return Integer.parseInt(value);
      }
    },
    readonly(false,"Run read only benchmarks.") {
      @Override public Object parseValue(String value) {
        return parseBoolean(value);
      }
    },
    disable_auto_compactions(false,"Do not auto trigger compactions.") {
      @Override public Object parseValue(String value) {
        return parseBoolean(value);
      }
    },
    source_compaction_factor(1,"Cap the size of data in level-K for\n" +
        "\ta compaction run that compacts Level-K with Level-(K+1) (for\n" +
        "\tK >= 1)") {
      @Override public Object parseValue(String value) {
        return Integer.parseInt(value);
      }
    },
    wal_ttl_seconds(0L,"Set the TTL for the WAL Files in seconds.") {
      @Override public Object parseValue(String value) {
        return Long.parseLong(value);
      }
    },
    wal_size_limit_MB(0L,"Set the size limit for the WAL Files\n" +
        "\tin MB.") {
      @Override public Object parseValue(String value) {
        return Long.parseLong(value);
      }
    },
    /* TODO(yhchiang): enable the following
    bufferedio(rocksdb::EnvOptions().use_os_buffer,
        "Allow buffered io using OS buffers.") {
      @Override public Object parseValue(String value) {
        return parseBoolean(value);
      }
    },
    */
    mmap_read(false,
        "Allow reads to occur via mmap-ing files.") {
      @Override public Object parseValue(String value) {
        return parseBoolean(value);
      }
    },
    mmap_write(false,
        "Allow writes to occur via mmap-ing files.") {
      @Override public Object parseValue(String value) {
        return parseBoolean(value);
      }
    },
    advise_random_on_open(defaultOptions_.adviseRandomOnOpen(),
        "Advise random access on table file open.") {
      @Override public Object parseValue(String value) {
        return parseBoolean(value);
      }
    },
    compaction_fadvice("NORMAL",
      "Access pattern advice when a file is compacted.") {
      @Override public Object parseValue(String value) {
        return value;
      }
    },
    use_tailing_iterator(false,
        "Use tailing iterator to access a series of keys instead of get.") {
      @Override public Object parseValue(String value) {
        return parseBoolean(value);
      }
    },
    use_adaptive_mutex(defaultOptions_.useAdaptiveMutex(),
        "Use adaptive mutex.") {
      @Override public Object parseValue(String value) {
        return parseBoolean(value);
      }
    },
    bytes_per_sync(defaultOptions_.bytesPerSync(),
        "Allows OS to incrementally sync files to disk while they are\n" +
        "\tbeing written, in the background. Issue one request for every\n" +
        "\tbytes_per_sync written. 0 turns it off.") {
      @Override public Object parseValue(String value) {
        return Long.parseLong(value);
      }
    },
    filter_deletes(false," On true, deletes use bloom-filter and drop\n" +
        "\tthe delete if key not present.") {
      @Override public Object parseValue(String value) {
        return parseBoolean(value);
      }
    },
    max_successive_merges(0,"Maximum number of successive merge\n" +
        "\toperations on a key in the memtable.") {
      @Override public Object parseValue(String value) {
        return Integer.parseInt(value);
      }
    },
    db("/tmp/rocksdbjni-bench",
       "Use the db with the following name.") {
      @Override public Object parseValue(String value) {
        return value;
      }
    },
    use_mem_env(false, "Use RocksMemEnv instead of default filesystem based\n" +
        "environment.") {
      @Override public Object parseValue(String value) {
        return parseBoolean(value);
      }
    };

    private Flag(Object defaultValue, String desc) {
      defaultValue_ = defaultValue;
      desc_ = desc;
    }

    public Object getDefaultValue() {
      return defaultValue_;
    }

    public String desc() {
      return desc_;
    }

    public boolean parseBoolean(String value) {
      if (value.equals("1")) {
        return true;
      } else if (value.equals("0")) {
        return false;
      }
      return Boolean.parseBoolean(value);
    }

    protected abstract Object parseValue(String value);

    private final Object defaultValue_;
    private final String desc_;
  }

  private static class RandomGenerator {
    private final byte[] data_;
    private int dataLength_;
    private int position_;
    private double compressionRatio_;
    Random rand_;

    private RandomGenerator(long seed, double compressionRatio) {
      // We use a limited amount of data over and over again and ensure
      // that it is larger than the compression window (32KB), and also
      byte[] value = new byte[100];
      // large enough to serve all typical value sizes we want to write.
      rand_ = new Random(seed);
      dataLength_ = value.length * 10000;
      data_ = new byte[dataLength_];
      compressionRatio_ = compressionRatio;
      int pos = 0;
      while (pos < dataLength_) {
        compressibleBytes(value);
        System.arraycopy(value, 0, data_, pos,
                         Math.min(value.length, dataLength_ - pos));
        pos += value.length;
      }
    }

    private void compressibleBytes(byte[] value) {
      int baseLength = value.length;
      if (compressionRatio_ < 1.0d) {
        baseLength = (int) (compressionRatio_ * value.length + 0.5);
      }
      if (baseLength <= 0) {
        baseLength = 1;
      }
      int pos;
      for (pos = 0; pos < baseLength; ++pos) {
        value[pos] = (byte) (' ' + rand_.nextInt(95));  // ' ' .. '~'
      }
      while (pos < value.length) {
        System.arraycopy(value, 0, value, pos,
                         Math.min(baseLength, value.length - pos));
        pos += baseLength;
      }
    }

    private void generate(byte[] value) {
      if (position_ + value.length > data_.length) {
        position_ = 0;
        assert(value.length <= data_.length);
      }
      position_ += value.length;
      System.arraycopy(data_, position_ - value.length,
                       value, 0, value.length);
    }
  }

  boolean isFinished() {
    synchronized(finishLock_) {
      return isFinished_;
    }
  }

  void setFinished(boolean flag) {
    synchronized(finishLock_) {
      isFinished_ = flag;
    }
  }

  RocksDB db_;
  final List<String> benchmarks_;
  final int num_;
  final int reads_;
  final int keySize_;
  final int valueSize_;
  final int threadNum_;
  final int writesPerSeconds_;
  final long randSeed_;
  final boolean useExisting_;
  final String databaseDir_;
  double compressionRatio_;
  RandomGenerator gen_;
  long startTime_;

  // env
  boolean useMemenv_;

  // memtable related
  final int maxWriteBufferNumber_;
  final int prefixSize_;
  final int keysPerPrefix_;
  final String memtable_;
  final long hashBucketCount_;

  // sst format related
  boolean usePlainTable_;

  Object finishLock_;
  boolean isFinished_;
  Map<Flag, Object> flags_;
  // as the scope of a static member equals to the scope of the problem,
  // we let its c++ pointer to be disposed in its finalizer.
  static Options defaultOptions_ = new Options();
  static BlockBasedTableConfig defaultBlockBasedTableOptions_ =
    new BlockBasedTableConfig();
  String compressionType_;
  CompressionType compression_;
}
