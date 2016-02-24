//  Copyright (c) 2014, Facebook, Inc.  All rights reserved.
//  This source code is licensed under the BSD-style license found in the
//  LICENSE file in the root directory of this source tree. An additional grant
//  of patent rights can be found in the PATENTS file in the same directory.
//
// Copyright (c) 2011 The LevelDB Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file. See the AUTHORS file for names of contributors.

#define __STDC_FORMAT_MACROS

#ifndef GFLAGS
#include <cstdio>
int main() {
  fprintf(stderr, "Please install gflags to run rocksdb tools\n");
  return 1;
}
#else

#include <gflags/gflags.h>

#include <atomic>
#include <iostream>
#include <memory>
#include <thread>
#include <type_traits>
#include <vector>

#include "db/dbformat.h"
#include "db/memtable.h"
#include "db/writebuffer.h"
#include "port/port.h"
#include "port/stack_trace.h"
#include "rocksdb/comparator.h"
#include "rocksdb/memtablerep.h"
#include "rocksdb/options.h"
#include "rocksdb/slice_transform.h"
#include "util/arena.h"
#include "util/mutexlock.h"
#include "util/stop_watch.h"
#include "util/testutil.h"

using GFLAGS::ParseCommandLineFlags;
using GFLAGS::RegisterFlagValidator;
using GFLAGS::SetUsageMessage;

DEFINE_string(benchmarks, "fillrandom",
              "Comma-separated list of benchmarks to run. Options:\n"
              "\tfillrandom             -- write N random values\n"
              "\tfillseq                -- write N values in sequential order\n"
              "\treadrandom             -- read N values in random order\n"
              "\treadseq                -- scan the DB\n"
              "\treadwrite              -- 1 thread writes while N - 1 threads "
              "do random\n"
              "\t                          reads\n"
              "\tseqreadwrite           -- 1 thread writes while N - 1 threads "
              "do scans\n");

DEFINE_string(memtablerep, "skiplist",
              "Which implementation of memtablerep to use. See "
              "include/memtablerep.h for\n"
              "  more details. Options:\n"
              "\tskiplist            -- backed by a skiplist\n"
              "\tvector              -- backed by an std::vector\n"
              "\thashskiplist        -- backed by a hash skip list\n"
              "\thashlinklist        -- backed by a hash linked list\n"
              "\tcuckoo              -- backed by a cuckoo hash table");

DEFINE_int64(bucket_count, 1000000,
             "bucket_count parameter to pass into NewHashSkiplistRepFactory or "
             "NewHashLinkListRepFactory");

DEFINE_int32(
    hashskiplist_height, 4,
    "skiplist_height parameter to pass into NewHashSkiplistRepFactory");

DEFINE_int32(
    hashskiplist_branching_factor, 4,
    "branching_factor parameter to pass into NewHashSkiplistRepFactory");

DEFINE_int32(
    huge_page_tlb_size, 0,
    "huge_page_tlb_size parameter to pass into NewHashLinkListRepFactory");

DEFINE_int32(bucket_entries_logging_threshold, 4096,
             "bucket_entries_logging_threshold parameter to pass into "
             "NewHashLinkListRepFactory");

DEFINE_bool(if_log_bucket_dist_when_flash, true,
            "if_log_bucket_dist_when_flash parameter to pass into "
            "NewHashLinkListRepFactory");

DEFINE_int32(
    threshold_use_skiplist, 256,
    "threshold_use_skiplist parameter to pass into NewHashLinkListRepFactory");

DEFINE_int64(
    write_buffer_size, 256,
    "write_buffer_size parameter to pass into NewHashCuckooRepFactory");

DEFINE_int64(
    average_data_size, 64,
    "average_data_size parameter to pass into NewHashCuckooRepFactory");

DEFINE_int64(
    hash_function_count, 4,
    "hash_function_count parameter to pass into NewHashCuckooRepFactory");

DEFINE_int32(
    num_threads, 1,
    "Number of concurrent threads to run. If the benchmark includes writes,\n"
    "then at most one thread will be a writer");

DEFINE_int32(num_operations, 1000000,
             "Number of operations to do for write and random read benchmarks");

DEFINE_int32(num_scans, 10,
             "Number of times for each thread to scan the memtablerep for "
             "sequential read "
             "benchmarks");

DEFINE_int32(item_size, 100, "Number of bytes each item should be");

DEFINE_int32(prefix_length, 8,
             "Prefix length to pass into NewFixedPrefixTransform");

/* VectorRep settings */
DEFINE_int64(vectorrep_count, 0,
             "Number of entries to reserve on VectorRep initialization");

DEFINE_int64(seed, 0,
             "Seed base for random number generators. "
             "When 0 it is deterministic.");

static rocksdb::Env* FLAGS_env = rocksdb::Env::Default();

namespace rocksdb {

namespace {
struct CallbackVerifyArgs {
  bool found;
  LookupKey* key;
  MemTableRep* table;
  InternalKeyComparator* comparator;
};
}  // namespace

// Helper for quickly generating random data.
class RandomGenerator {
 private:
  std::string data_;
  unsigned int pos_;

 public:
  RandomGenerator() {
    Random rnd(301);
    auto size = (unsigned)std::max(1048576, FLAGS_item_size);
    test::RandomString(&rnd, size, &data_);
    pos_ = 0;
  }

  Slice Generate(unsigned int len) {
    assert(len <= data_.size());
    if (pos_ + len > data_.size()) {
      pos_ = 0;
    }
    pos_ += len;
    return Slice(data_.data() + pos_ - len, len);
  }
};

enum WriteMode { SEQUENTIAL, RANDOM, UNIQUE_RANDOM };

class KeyGenerator {
 public:
  KeyGenerator(Random64* rand, WriteMode mode, uint64_t num)
      : rand_(rand), mode_(mode), num_(num), next_(0) {
    if (mode_ == UNIQUE_RANDOM) {
      // NOTE: if memory consumption of this approach becomes a concern,
      // we can either break it into pieces and only random shuffle a section
      // each time. Alternatively, use a bit map implementation
      // (https://reviews.facebook.net/differential/diff/54627/)
      values_.resize(num_);
      for (uint64_t i = 0; i < num_; ++i) {
        values_[i] = i;
      }
      std::shuffle(
          values_.begin(), values_.end(),
          std::default_random_engine(static_cast<unsigned int>(FLAGS_seed)));
    }
  }

  uint64_t Next() {
    switch (mode_) {
      case SEQUENTIAL:
        return next_++;
      case RANDOM:
        return rand_->Next() % num_;
      case UNIQUE_RANDOM:
        return values_[next_++];
    }
    assert(false);
    return std::numeric_limits<uint64_t>::max();
  }

 private:
  Random64* rand_;
  WriteMode mode_;
  const uint64_t num_;
  uint64_t next_;
  std::vector<uint64_t> values_;
};

class BenchmarkThread {
 public:
  explicit BenchmarkThread(MemTableRep* table, KeyGenerator* key_gen,
                           uint64_t* bytes_written, uint64_t* bytes_read,
                           uint64_t* sequence, uint64_t num_ops,
                           uint64_t* read_hits)
      : table_(table),
        key_gen_(key_gen),
        bytes_written_(bytes_written),
        bytes_read_(bytes_read),
        sequence_(sequence),
        num_ops_(num_ops),
        read_hits_(read_hits) {}

  virtual void operator()() = 0;
  virtual ~BenchmarkThread() {}

 protected:
  MemTableRep* table_;
  KeyGenerator* key_gen_;
  uint64_t* bytes_written_;
  uint64_t* bytes_read_;
  uint64_t* sequence_;
  uint64_t num_ops_;
  uint64_t* read_hits_;
  RandomGenerator generator_;
};

class FillBenchmarkThread : public BenchmarkThread {
 public:
  FillBenchmarkThread(MemTableRep* table, KeyGenerator* key_gen,
                      uint64_t* bytes_written, uint64_t* bytes_read,
                      uint64_t* sequence, uint64_t num_ops, uint64_t* read_hits)
      : BenchmarkThread(table, key_gen, bytes_written, bytes_read, sequence,
                        num_ops, read_hits) {}

  void FillOne() {
    char* buf = nullptr;
    auto internal_key_size = 16;
    auto encoded_len =
        FLAGS_item_size + VarintLength(internal_key_size) + internal_key_size;
    KeyHandle handle = table_->Allocate(encoded_len, &buf);
    assert(buf != nullptr);
    char* p = EncodeVarint32(buf, internal_key_size);
    auto key = key_gen_->Next();
    EncodeFixed64(p, key);
    p += 8;
    EncodeFixed64(p, ++(*sequence_));
    p += 8;
    Slice bytes = generator_.Generate(FLAGS_item_size);
    memcpy(p, bytes.data(), FLAGS_item_size);
    p += FLAGS_item_size;
    assert(p == buf + encoded_len);
    table_->Insert(handle);
    *bytes_written_ += encoded_len;
  }

  void operator()() override {
    for (unsigned int i = 0; i < num_ops_; ++i) {
      FillOne();
    }
  }
};

class ConcurrentFillBenchmarkThread : public FillBenchmarkThread {
 public:
  ConcurrentFillBenchmarkThread(MemTableRep* table, KeyGenerator* key_gen,
                                uint64_t* bytes_written, uint64_t* bytes_read,
                                uint64_t* sequence, uint64_t num_ops,
                                uint64_t* read_hits,
                                std::atomic_int* threads_done)
      : FillBenchmarkThread(table, key_gen, bytes_written, bytes_read, sequence,
                            num_ops, read_hits) {
    threads_done_ = threads_done;
  }

  void operator()() override {
    // # of read threads will be total threads - write threads (always 1). Loop
    // while all reads complete.
    while ((*threads_done_).load() < (FLAGS_num_threads - 1)) {
      FillOne();
    }
  }

 private:
  std::atomic_int* threads_done_;
};

class ReadBenchmarkThread : public BenchmarkThread {
 public:
  ReadBenchmarkThread(MemTableRep* table, KeyGenerator* key_gen,
                      uint64_t* bytes_written, uint64_t* bytes_read,
                      uint64_t* sequence, uint64_t num_ops, uint64_t* read_hits)
      : BenchmarkThread(table, key_gen, bytes_written, bytes_read, sequence,
                        num_ops, read_hits) {}

  static bool callback(void* arg, const char* entry) {
    CallbackVerifyArgs* callback_args = static_cast<CallbackVerifyArgs*>(arg);
    assert(callback_args != nullptr);
    uint32_t key_length;
    const char* key_ptr = GetVarint32Ptr(entry, entry + 5, &key_length);
    if ((callback_args->comparator)
            ->user_comparator()
            ->Equal(Slice(key_ptr, key_length - 8),
                    callback_args->key->user_key())) {
      callback_args->found = true;
    }
    return false;
  }

  void ReadOne() {
    std::string user_key;
    auto key = key_gen_->Next();
    PutFixed64(&user_key, key);
    LookupKey lookup_key(user_key, *sequence_);
    InternalKeyComparator internal_key_comp(BytewiseComparator());
    CallbackVerifyArgs verify_args;
    verify_args.found = false;
    verify_args.key = &lookup_key;
    verify_args.table = table_;
    verify_args.comparator = &internal_key_comp;
    table_->Get(lookup_key, &verify_args, callback);
    if (verify_args.found) {
      *bytes_read_ += VarintLength(16) + 16 + FLAGS_item_size;
      ++*read_hits_;
    }
  }
  void operator()() override {
    for (unsigned int i = 0; i < num_ops_; ++i) {
      ReadOne();
    }
  }
};

class SeqReadBenchmarkThread : public BenchmarkThread {
 public:
  SeqReadBenchmarkThread(MemTableRep* table, KeyGenerator* key_gen,
                         uint64_t* bytes_written, uint64_t* bytes_read,
                         uint64_t* sequence, uint64_t num_ops,
                         uint64_t* read_hits)
      : BenchmarkThread(table, key_gen, bytes_written, bytes_read, sequence,
                        num_ops, read_hits) {}

  void ReadOneSeq() {
    std::unique_ptr<MemTableRep::Iterator> iter(table_->GetIterator());
    for (iter->SeekToFirst(); iter->Valid(); iter->Next()) {
      // pretend to read the value
      *bytes_read_ += VarintLength(16) + 16 + FLAGS_item_size;
    }
    ++*read_hits_;
  }

  void operator()() override {
    for (unsigned int i = 0; i < num_ops_; ++i) {
      { ReadOneSeq(); }
    }
  }
};

class ConcurrentReadBenchmarkThread : public ReadBenchmarkThread {
 public:
  ConcurrentReadBenchmarkThread(MemTableRep* table, KeyGenerator* key_gen,
                                uint64_t* bytes_written, uint64_t* bytes_read,
                                uint64_t* sequence, uint64_t num_ops,
                                uint64_t* read_hits,
                                std::atomic_int* threads_done)
      : ReadBenchmarkThread(table, key_gen, bytes_written, bytes_read, sequence,
                            num_ops, read_hits) {
    threads_done_ = threads_done;
  }

  void operator()() override {
    for (unsigned int i = 0; i < num_ops_; ++i) {
      ReadOne();
    }
    ++*threads_done_;
  }

 private:
  std::atomic_int* threads_done_;
};

class SeqConcurrentReadBenchmarkThread : public SeqReadBenchmarkThread {
 public:
  SeqConcurrentReadBenchmarkThread(MemTableRep* table, KeyGenerator* key_gen,
                                   uint64_t* bytes_written,
                                   uint64_t* bytes_read, uint64_t* sequence,
                                   uint64_t num_ops, uint64_t* read_hits,
                                   std::atomic_int* threads_done)
      : SeqReadBenchmarkThread(table, key_gen, bytes_written, bytes_read,
                               sequence, num_ops, read_hits) {
    threads_done_ = threads_done;
  }

  void operator()() override {
    for (unsigned int i = 0; i < num_ops_; ++i) {
      ReadOneSeq();
    }
    ++*threads_done_;
  }

 private:
  std::atomic_int* threads_done_;
};

class Benchmark {
 public:
  explicit Benchmark(MemTableRep* table, KeyGenerator* key_gen,
                     uint64_t* sequence, uint32_t num_threads)
      : table_(table),
        key_gen_(key_gen),
        sequence_(sequence),
        num_threads_(num_threads) {}

  virtual ~Benchmark() {}
  virtual void Run() {
    std::cout << "Number of threads: " << num_threads_ << std::endl;
    std::vector<std::thread> threads;
    uint64_t bytes_written = 0;
    uint64_t bytes_read = 0;
    uint64_t read_hits = 0;
    StopWatchNano timer(Env::Default(), true);
    RunThreads(&threads, &bytes_written, &bytes_read, true, &read_hits);
    auto elapsed_time = static_cast<double>(timer.ElapsedNanos() / 1000);
    std::cout << "Elapsed time: " << static_cast<int>(elapsed_time) << " us"
              << std::endl;

    if (bytes_written > 0) {
      auto MiB_written = static_cast<double>(bytes_written) / (1 << 20);
      auto write_throughput = MiB_written / (elapsed_time / 1000000);
      std::cout << "Total bytes written: " << MiB_written << " MiB"
                << std::endl;
      std::cout << "Write throughput: " << write_throughput << " MiB/s"
                << std::endl;
      auto us_per_op = elapsed_time / num_write_ops_per_thread_;
      std::cout << "write us/op: " << us_per_op << std::endl;
    }
    if (bytes_read > 0) {
      auto MiB_read = static_cast<double>(bytes_read) / (1 << 20);
      auto read_throughput = MiB_read / (elapsed_time / 1000000);
      std::cout << "Total bytes read: " << MiB_read << " MiB" << std::endl;
      std::cout << "Read throughput: " << read_throughput << " MiB/s"
                << std::endl;
      auto us_per_op = elapsed_time / num_read_ops_per_thread_;
      std::cout << "read us/op: " << us_per_op << std::endl;
    }
  }

  virtual void RunThreads(std::vector<std::thread>* threads,
                          uint64_t* bytes_written, uint64_t* bytes_read,
                          bool write, uint64_t* read_hits) = 0;

 protected:
  MemTableRep* table_;
  KeyGenerator* key_gen_;
  uint64_t* sequence_;
  uint64_t num_write_ops_per_thread_;
  uint64_t num_read_ops_per_thread_;
  const uint32_t num_threads_;
};

class FillBenchmark : public Benchmark {
 public:
  explicit FillBenchmark(MemTableRep* table, KeyGenerator* key_gen,
                         uint64_t* sequence)
      : Benchmark(table, key_gen, sequence, 1) {
    num_write_ops_per_thread_ = FLAGS_num_operations;
  }

  void RunThreads(std::vector<std::thread>* threads, uint64_t* bytes_written,
                  uint64_t* bytes_read, bool write,
                  uint64_t* read_hits) override {
    FillBenchmarkThread(table_, key_gen_, bytes_written, bytes_read, sequence_,
                        num_write_ops_per_thread_, read_hits)();
  }
};

class ReadBenchmark : public Benchmark {
 public:
  explicit ReadBenchmark(MemTableRep* table, KeyGenerator* key_gen,
                         uint64_t* sequence)
      : Benchmark(table, key_gen, sequence, FLAGS_num_threads) {
    num_read_ops_per_thread_ = FLAGS_num_operations / FLAGS_num_threads;
  }

  void RunThreads(std::vector<std::thread>* threads, uint64_t* bytes_written,
                  uint64_t* bytes_read, bool write,
                  uint64_t* read_hits) override {
    for (int i = 0; i < FLAGS_num_threads; ++i) {
      threads->emplace_back(
          ReadBenchmarkThread(table_, key_gen_, bytes_written, bytes_read,
                              sequence_, num_read_ops_per_thread_, read_hits));
    }
    for (auto& thread : *threads) {
      thread.join();
    }
    std::cout << "read hit%: "
              << (static_cast<double>(*read_hits) / FLAGS_num_operations) * 100
              << std::endl;
  }
};

class SeqReadBenchmark : public Benchmark {
 public:
  explicit SeqReadBenchmark(MemTableRep* table, uint64_t* sequence)
      : Benchmark(table, nullptr, sequence, FLAGS_num_threads) {
    num_read_ops_per_thread_ = FLAGS_num_scans;
  }

  void RunThreads(std::vector<std::thread>* threads, uint64_t* bytes_written,
                  uint64_t* bytes_read, bool write,
                  uint64_t* read_hits) override {
    for (int i = 0; i < FLAGS_num_threads; ++i) {
      threads->emplace_back(SeqReadBenchmarkThread(
          table_, key_gen_, bytes_written, bytes_read, sequence_,
          num_read_ops_per_thread_, read_hits));
    }
    for (auto& thread : *threads) {
      thread.join();
    }
  }
};

template <class ReadThreadType>
class ReadWriteBenchmark : public Benchmark {
 public:
  explicit ReadWriteBenchmark(MemTableRep* table, KeyGenerator* key_gen,
                              uint64_t* sequence)
      : Benchmark(table, key_gen, sequence, FLAGS_num_threads) {
    num_read_ops_per_thread_ =
        FLAGS_num_threads <= 1
            ? 0
            : (FLAGS_num_operations / (FLAGS_num_threads - 1));
    num_write_ops_per_thread_ = FLAGS_num_operations;
  }

  void RunThreads(std::vector<std::thread>* threads, uint64_t* bytes_written,
                  uint64_t* bytes_read, bool write,
                  uint64_t* read_hits) override {
    std::atomic_int threads_done;
    threads_done.store(0);
    threads->emplace_back(ConcurrentFillBenchmarkThread(
        table_, key_gen_, bytes_written, bytes_read, sequence_,
        num_write_ops_per_thread_, read_hits, &threads_done));
    for (int i = 1; i < FLAGS_num_threads; ++i) {
      threads->emplace_back(
          ReadThreadType(table_, key_gen_, bytes_written, bytes_read, sequence_,
                         num_read_ops_per_thread_, read_hits, &threads_done));
    }
    for (auto& thread : *threads) {
      thread.join();
    }
  }
};

}  // namespace rocksdb

void PrintWarnings() {
#if defined(__GNUC__) && !defined(__OPTIMIZE__)
  fprintf(stdout,
          "WARNING: Optimization is disabled: benchmarks unnecessarily slow\n");
#endif
#ifndef NDEBUG
  fprintf(stdout,
          "WARNING: Assertions are enabled; benchmarks unnecessarily slow\n");
#endif
}

int main(int argc, char** argv) {
  rocksdb::port::InstallStackTraceHandler();
  SetUsageMessage(std::string("\nUSAGE:\n") + std::string(argv[0]) +
                  " [OPTIONS]...");
  ParseCommandLineFlags(&argc, &argv, true);

  PrintWarnings();

  rocksdb::Options options;

  std::unique_ptr<rocksdb::MemTableRepFactory> factory;
  if (FLAGS_memtablerep == "skiplist") {
    factory.reset(new rocksdb::SkipListFactory);
  } else if (FLAGS_memtablerep == "vector") {
    factory.reset(new rocksdb::VectorRepFactory);
  } else if (FLAGS_memtablerep == "hashskiplist") {
    factory.reset(rocksdb::NewHashSkipListRepFactory(
        FLAGS_bucket_count, FLAGS_hashskiplist_height,
        FLAGS_hashskiplist_branching_factor));
    options.prefix_extractor.reset(
        rocksdb::NewFixedPrefixTransform(FLAGS_prefix_length));
  } else if (FLAGS_memtablerep == "hashlinklist") {
    factory.reset(rocksdb::NewHashLinkListRepFactory(
        FLAGS_bucket_count, FLAGS_huge_page_tlb_size,
        FLAGS_bucket_entries_logging_threshold,
        FLAGS_if_log_bucket_dist_when_flash, FLAGS_threshold_use_skiplist));
    options.prefix_extractor.reset(
        rocksdb::NewFixedPrefixTransform(FLAGS_prefix_length));
  } else if (FLAGS_memtablerep == "cuckoo") {
    factory.reset(rocksdb::NewHashCuckooRepFactory(
        FLAGS_write_buffer_size, FLAGS_average_data_size,
        static_cast<uint32_t>(FLAGS_hash_function_count)));
    options.prefix_extractor.reset(
        rocksdb::NewFixedPrefixTransform(FLAGS_prefix_length));
  } else {
    fprintf(stdout, "Unknown memtablerep: %s\n", FLAGS_memtablerep.c_str());
    exit(1);
  }

  rocksdb::InternalKeyComparator internal_key_comp(
      rocksdb::BytewiseComparator());
  rocksdb::MemTable::KeyComparator key_comp(internal_key_comp);
  rocksdb::Arena arena;
  rocksdb::WriteBuffer wb(FLAGS_write_buffer_size);
  rocksdb::MemTableAllocator memtable_allocator(&arena, &wb);
  uint64_t sequence;
  auto createMemtableRep = [&] {
    sequence = 0;
    return factory->CreateMemTableRep(key_comp, &memtable_allocator,
                                      options.prefix_extractor.get(),
                                      options.info_log.get());
  };
  std::unique_ptr<rocksdb::MemTableRep> memtablerep;
  rocksdb::Random64 rng(FLAGS_seed);
  const char* benchmarks = FLAGS_benchmarks.c_str();
  while (benchmarks != nullptr) {
    std::unique_ptr<rocksdb::KeyGenerator> key_gen;
    const char* sep = strchr(benchmarks, ',');
    rocksdb::Slice name;
    if (sep == nullptr) {
      name = benchmarks;
      benchmarks = nullptr;
    } else {
      name = rocksdb::Slice(benchmarks, sep - benchmarks);
      benchmarks = sep + 1;
    }
    std::unique_ptr<rocksdb::Benchmark> benchmark;
    if (name == rocksdb::Slice("fillseq")) {
      memtablerep.reset(createMemtableRep());
      key_gen.reset(new rocksdb::KeyGenerator(&rng, rocksdb::SEQUENTIAL,
                                              FLAGS_num_operations));
      benchmark.reset(new rocksdb::FillBenchmark(memtablerep.get(),
                                                 key_gen.get(), &sequence));
    } else if (name == rocksdb::Slice("fillrandom")) {
      memtablerep.reset(createMemtableRep());
      key_gen.reset(new rocksdb::KeyGenerator(&rng, rocksdb::UNIQUE_RANDOM,
                                              FLAGS_num_operations));
      benchmark.reset(new rocksdb::FillBenchmark(memtablerep.get(),
                                                 key_gen.get(), &sequence));
    } else if (name == rocksdb::Slice("readrandom")) {
      key_gen.reset(new rocksdb::KeyGenerator(&rng, rocksdb::RANDOM,
                                              FLAGS_num_operations));
      benchmark.reset(new rocksdb::ReadBenchmark(memtablerep.get(),
                                                 key_gen.get(), &sequence));
    } else if (name == rocksdb::Slice("readseq")) {
      key_gen.reset(new rocksdb::KeyGenerator(&rng, rocksdb::SEQUENTIAL,
                                              FLAGS_num_operations));
      benchmark.reset(
          new rocksdb::SeqReadBenchmark(memtablerep.get(), &sequence));
    } else if (name == rocksdb::Slice("readwrite")) {
      memtablerep.reset(createMemtableRep());
      key_gen.reset(new rocksdb::KeyGenerator(&rng, rocksdb::RANDOM,
                                              FLAGS_num_operations));
      benchmark.reset(new rocksdb::ReadWriteBenchmark<
          rocksdb::ConcurrentReadBenchmarkThread>(memtablerep.get(),
                                                  key_gen.get(), &sequence));
    } else if (name == rocksdb::Slice("seqreadwrite")) {
      memtablerep.reset(createMemtableRep());
      key_gen.reset(new rocksdb::KeyGenerator(&rng, rocksdb::RANDOM,
                                              FLAGS_num_operations));
      benchmark.reset(new rocksdb::ReadWriteBenchmark<
          rocksdb::SeqConcurrentReadBenchmarkThread>(memtablerep.get(),
                                                     key_gen.get(), &sequence));
    } else {
      std::cout << "WARNING: skipping unknown benchmark '" << name.ToString()
                << std::endl;
      continue;
    }
    std::cout << "Running " << name.ToString() << std::endl;
    benchmark->Run();
  }

  return 0;
}

#endif  // GFLAGS
