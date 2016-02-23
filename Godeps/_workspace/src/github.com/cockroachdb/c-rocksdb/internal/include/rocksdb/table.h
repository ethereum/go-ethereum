// Copyright (c) 2011 The LevelDB Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file. See the AUTHORS file for names of contributors.
//
// Currently we support two types of tables: plain table and block-based table.
//   1. Block-based table: this is the default table type that we inherited from
//      LevelDB, which was designed for storing data in hard disk or flash
//      device.
//   2. Plain table: it is one of RocksDB's SST file format optimized
//      for low query latency on pure-memory or really low-latency media.
//
// A tutorial of rocksdb table formats is available here:
//   https://github.com/facebook/rocksdb/wiki/A-Tutorial-of-RocksDB-SST-formats
//
// Example code is also available
//   https://github.com/facebook/rocksdb/wiki/A-Tutorial-of-RocksDB-SST-formats#wiki-examples

#pragma once
#include <memory>
#include <string>
#include <unordered_map>

#include "rocksdb/env.h"
#include "rocksdb/iterator.h"
#include "rocksdb/options.h"
#include "rocksdb/immutable_options.h"
#include "rocksdb/status.h"

namespace rocksdb {

// -- Block-based Table
class FlushBlockPolicyFactory;
class RandomAccessFile;
struct TableBuilderOptions;
class TableBuilder;
class TableReader;
class WritableFileWriter;
struct EnvOptions;
struct Options;

using std::unique_ptr;

enum ChecksumType : char {
  kNoChecksum = 0x0,  // not yet supported. Will fail
  kCRC32c = 0x1,
  kxxHash = 0x2,
};

// For advanced user only
struct BlockBasedTableOptions {
  // @flush_block_policy_factory creates the instances of flush block policy.
  // which provides a configurable way to determine when to flush a block in
  // the block based tables.  If not set, table builder will use the default
  // block flush policy, which cut blocks by block size (please refer to
  // `FlushBlockBySizePolicy`).
  std::shared_ptr<FlushBlockPolicyFactory> flush_block_policy_factory;

  // TODO(kailiu) Temporarily disable this feature by making the default value
  // to be false.
  //
  // Indicating if we'd put index/filter blocks to the block cache.
  // If not specified, each "table reader" object will pre-load index/filter
  // block during table initialization.
  bool cache_index_and_filter_blocks = false;

  // The index type that will be used for this table.
  enum IndexType : char {
    // A space efficient index block that is optimized for
    // binary-search-based index.
    kBinarySearch,

    // The hash index, if enabled, will do the hash lookup when
    // `Options.prefix_extractor` is provided.
    kHashSearch,
  };

  IndexType index_type = kBinarySearch;

  // Influence the behavior when kHashSearch is used.
  // if false, stores a precise prefix to block range mapping
  // if true, does not store prefix and allows prefix hash collision
  // (less memory consumption)
  bool hash_index_allow_collision = true;

  // Use the specified checksum type. Newly created table files will be
  // protected with this checksum type. Old table files will still be readable,
  // even though they have different checksum type.
  ChecksumType checksum = kCRC32c;

  // Disable block cache. If this is set to true,
  // then no block cache should be used, and the block_cache should
  // point to a nullptr object.
  bool no_block_cache = false;

  // If non-NULL use the specified cache for blocks.
  // If NULL, rocksdb will automatically create and use an 8MB internal cache.
  std::shared_ptr<Cache> block_cache = nullptr;

  // If non-NULL use the specified cache for compressed blocks.
  // If NULL, rocksdb will not use a compressed block cache.
  std::shared_ptr<Cache> block_cache_compressed = nullptr;

  // Approximate size of user data packed per block.  Note that the
  // block size specified here corresponds to uncompressed data.  The
  // actual size of the unit read from disk may be smaller if
  // compression is enabled.  This parameter can be changed dynamically.
  size_t block_size = 4 * 1024;

  // This is used to close a block before it reaches the configured
  // 'block_size'. If the percentage of free space in the current block is less
  // than this specified number and adding a new record to the block will
  // exceed the configured block size, then this block will be closed and the
  // new record will be written to the next block.
  int block_size_deviation = 10;

  // Number of keys between restart points for delta encoding of keys.
  // This parameter can be changed dynamically.  Most clients should
  // leave this parameter alone.
  int block_restart_interval = 16;

  // If non-nullptr, use the specified filter policy to reduce disk reads.
  // Many applications will benefit from passing the result of
  // NewBloomFilterPolicy() here.
  std::shared_ptr<const FilterPolicy> filter_policy = nullptr;

  // If true, place whole keys in the filter (not just prefixes).
  // This must generally be true for gets to be efficient.
  bool whole_key_filtering = true;

  // We currently have three versions:
  // 0 -- This version is currently written out by all RocksDB's versions by
  // default.  Can be read by really old RocksDB's. Doesn't support changing
  // checksum (default is CRC32).
  // 1 -- Can be read by RocksDB's versions since 3.0. Supports non-default
  // checksum, like xxHash. It is written by RocksDB when
  // BlockBasedTableOptions::checksum is something other than kCRC32c. (version
  // 0 is silently upconverted)
  // 2 -- Can be read by RocksDB's versions since 3.10. Changes the way we
  // encode compressed blocks with LZ4, BZip2 and Zlib compression. If you
  // don't plan to run RocksDB before version 3.10, you should probably use
  // this.
  // This option only affects newly written tables. When reading exising tables,
  // the information about version is read from the footer.
  uint32_t format_version = 0;
};

// Table Properties that are specific to block-based table properties.
struct BlockBasedTablePropertyNames {
  // value of this propertis is a fixed int32 number.
  static const std::string kIndexType;
  // value is "1" for true and "0" for false.
  static const std::string kWholeKeyFiltering;
  // value is "1" for true and "0" for false.
  static const std::string kPrefixFiltering;
};

// Create default block based table factory.
extern TableFactory* NewBlockBasedTableFactory(
    const BlockBasedTableOptions& table_options = BlockBasedTableOptions());

#ifndef ROCKSDB_LITE

enum EncodingType : char {
  // Always write full keys without any special encoding.
  kPlain,
  // Find opportunity to write the same prefix once for multiple rows.
  // In some cases, when a key follows a previous key with the same prefix,
  // instead of writing out the full key, it just writes out the size of the
  // shared prefix, as well as other bytes, to save some bytes.
  //
  // When using this option, the user is required to use the same prefix
  // extractor to make sure the same prefix will be extracted from the same key.
  // The Name() value of the prefix extractor will be stored in the file. When
  // reopening the file, the name of the options.prefix_extractor given will be
  // bitwise compared to the prefix extractors stored in the file. An error
  // will be returned if the two don't match.
  kPrefix,
};

// Table Properties that are specific to plain table properties.
struct PlainTablePropertyNames {
  static const std::string kPrefixExtractorName;
  static const std::string kEncodingType;
  static const std::string kBloomVersion;
  static const std::string kNumBloomBlocks;
};

const uint32_t kPlainTableVariableLength = 0;

struct PlainTableOptions {
  // @user_key_len: plain table has optimization for fix-sized keys, which can
  //                be specified via user_key_len.  Alternatively, you can pass
  //                `kPlainTableVariableLength` if your keys have variable
  //                lengths.
  uint32_t user_key_len = kPlainTableVariableLength;

  // @bloom_bits_per_key: the number of bits used for bloom filer per prefix.
  //                      You may disable it by passing a zero.
  int bloom_bits_per_key = 10;

  // @hash_table_ratio: the desired utilization of the hash table used for
  //                    prefix hashing.
  //                    hash_table_ratio = number of prefixes / #buckets in the
  //                    hash table
  double hash_table_ratio = 0.75;

  // @index_sparseness: inside each prefix, need to build one index record for
  //                    how many keys for binary search inside each hash bucket.
  //                    For encoding type kPrefix, the value will be used when
  //                    writing to determine an interval to rewrite the full
  //                    key. It will also be used as a suggestion and satisfied
  //                    when possible.
  size_t index_sparseness = 16;

  // @huge_page_tlb_size: if <=0, allocate hash indexes and blooms from malloc.
  //                      Otherwise from huge page TLB. The user needs to
  //                      reserve huge pages for it to be allocated, like:
  //                          sysctl -w vm.nr_hugepages=20
  //                      See linux doc Documentation/vm/hugetlbpage.txt
  size_t huge_page_tlb_size = 0;

  // @encoding_type: how to encode the keys. See enum EncodingType above for
  //                 the choices. The value will determine how to encode keys
  //                 when writing to a new SST file. This value will be stored
  //                 inside the SST file which will be used when reading from
  //                 the file, which makes it possible for users to choose
  //                 different encoding type when reopening a DB. Files with
  //                 different encoding types can co-exist in the same DB and
  //                 can be read.
  EncodingType encoding_type = kPlain;

  // @full_scan_mode: mode for reading the whole file one record by one without
  //                  using the index.
  bool full_scan_mode = false;

  // @store_index_in_file: compute plain table index and bloom filter during
  //                       file building and store it in file. When reading
  //                       file, index will be mmaped instead of recomputation.
  bool store_index_in_file = false;
};

// -- Plain Table with prefix-only seek
// For this factory, you need to set Options.prefix_extrator properly to make it
// work. Look-up will starts with prefix hash lookup for key prefix. Inside the
// hash bucket found, a binary search is executed for hash conflicts. Finally,
// a linear search is used.

extern TableFactory* NewPlainTableFactory(const PlainTableOptions& options =
                                              PlainTableOptions());

struct CuckooTablePropertyNames {
  // The key that is used to fill empty buckets.
  static const std::string kEmptyKey;
  // Fixed length of value.
  static const std::string kValueLength;
  // Number of hash functions used in Cuckoo Hash.
  static const std::string kNumHashFunc;
  // It denotes the number of buckets in a Cuckoo Block. Given a key and a
  // particular hash function, a Cuckoo Block is a set of consecutive buckets,
  // where starting bucket id is given by the hash function on the key. In case
  // of a collision during inserting the key, the builder tries to insert the
  // key in other locations of the cuckoo block before using the next hash
  // function. This reduces cache miss during read operation in case of
  // collision.
  static const std::string kCuckooBlockSize;
  // Size of the hash table. Use this number to compute the modulo of hash
  // function. The actual number of buckets will be kMaxHashTableSize +
  // kCuckooBlockSize - 1. The last kCuckooBlockSize-1 buckets are used to
  // accommodate the Cuckoo Block from end of hash table, due to cache friendly
  // implementation.
  static const std::string kHashTableSize;
  // Denotes if the key sorted in the file is Internal Key (if false)
  // or User Key only (if true).
  static const std::string kIsLastLevel;
  // Indicate if using identity function for the first hash function.
  static const std::string kIdentityAsFirstHash;
  // Indicate if using module or bit and to calculate hash value
  static const std::string kUseModuleHash;
  // Fixed user key length
  static const std::string kUserKeyLength;
};

struct CuckooTableOptions {
  // Determines the utilization of hash tables. Smaller values
  // result in larger hash tables with fewer collisions.
  double hash_table_ratio = 0.9;
  // A property used by builder to determine the depth to go to
  // to search for a path to displace elements in case of
  // collision. See Builder.MakeSpaceForKey method. Higher
  // values result in more efficient hash tables with fewer
  // lookups but take more time to build.
  uint32_t max_search_depth = 100;
  // In case of collision while inserting, the builder
  // attempts to insert in the next cuckoo_block_size
  // locations before skipping over to the next Cuckoo hash
  // function. This makes lookups more cache friendly in case
  // of collisions.
  uint32_t cuckoo_block_size = 5;
  // If this option is enabled, user key is treated as uint64_t and its value
  // is used as hash value directly. This option changes builder's behavior.
  // Reader ignore this option and behave according to what specified in table
  // property.
  bool identity_as_first_hash = false;
  // If this option is set to true, module is used during hash calculation.
  // This often yields better space efficiency at the cost of performance.
  // If this optino is set to false, # of entries in table is constrained to be
  // power of two, and bit and is used to calculate hash, which is faster in
  // general.
  bool use_module_hash = true;
};

// Cuckoo Table Factory for SST table format using Cache Friendly Cuckoo Hashing
extern TableFactory* NewCuckooTableFactory(
    const CuckooTableOptions& table_options = CuckooTableOptions());

#endif  // ROCKSDB_LITE

class RandomAccessFileReader;

// A base class for table factories.
class TableFactory {
 public:
  virtual ~TableFactory() {}

  // The type of the table.
  //
  // The client of this package should switch to a new name whenever
  // the table format implementation changes.
  //
  // Names starting with "rocksdb." are reserved and should not be used
  // by any clients of this package.
  virtual const char* Name() const = 0;

  // Returns a Table object table that can fetch data from file specified
  // in parameter file. It's the caller's responsibility to make sure
  // file is in the correct format.
  //
  // NewTableReader() is called in two places:
  // (1) TableCache::FindTable() calls the function when table cache miss
  //     and cache the table object returned.
  // (1) SstFileReader (for SST Dump) opens the table and dump the table
  //     contents using the interator of the table.
  // ImmutableCFOptions is a subset of Options that can not be altered.
  // EnvOptions is a subset of Options that will be used by Env.
  // Multiple configured can be accessed from there, including and not
  // limited to block cache and key comparators.
  // file is a file handler to handle the file for the table
  // file_size is the physical file size of the file
  // table_reader is the output table reader
  virtual Status NewTableReader(
      const ImmutableCFOptions& ioptions, const EnvOptions& env_options,
      const InternalKeyComparator& internal_comparator,
      unique_ptr<RandomAccessFileReader>&& file, uint64_t file_size,
      unique_ptr<TableReader>* table_reader) const = 0;

  // Return a table builder to write to a file for this table type.
  //
  // It is called in several places:
  // (1) When flushing memtable to a level-0 output file, it creates a table
  //     builder (In DBImpl::WriteLevel0Table(), by calling BuildTable())
  // (2) During compaction, it gets the builder for writing compaction output
  //     files in DBImpl::OpenCompactionOutputFile().
  // (3) When recovering from transaction logs, it creates a table builder to
  //     write to a level-0 output file (In DBImpl::WriteLevel0TableForRecovery,
  //     by calling BuildTable())
  // (4) When running Repairer, it creates a table builder to convert logs to
  //     SST files (In Repairer::ConvertLogToTable() by calling BuildTable())
  //
  // ImmutableCFOptions is a subset of Options that can not be altered.
  // Multiple configured can be acceseed from there, including and not limited
  // to compression options. file is a handle of a writable file.
  // It is the caller's responsibility to keep the file open and close the file
  // after closing the table builder. compression_type is the compression type
  // to use in this table.
  virtual TableBuilder* NewTableBuilder(
      const TableBuilderOptions& table_builder_options,
      WritableFileWriter* file) const = 0;

  // Sanitizes the specified DB Options and ColumnFamilyOptions.
  //
  // If the function cannot find a way to sanitize the input DB Options,
  // a non-ok Status will be returned.
  virtual Status SanitizeOptions(
      const DBOptions& db_opts,
      const ColumnFamilyOptions& cf_opts) const = 0;

  // Return a string that contains printable format of table configurations.
  // RocksDB prints configurations at DB Open().
  virtual std::string GetPrintableTableOptions() const = 0;
};

#ifndef ROCKSDB_LITE
// Create a special table factory that can open either of the supported
// table formats, based on setting inside the SST files. It should be used to
// convert a DB from one table format to another.
// @table_factory_to_write: the table factory used when writing to new files.
// @block_based_table_factory:  block based table factory to use. If NULL, use
//                              a default one.
// @plain_table_factory: plain table factory to use. If NULL, use a default one.
// @cuckoo_table_factory: cuckoo table factory to use. If NULL, use a default one.
extern TableFactory* NewAdaptiveTableFactory(
    std::shared_ptr<TableFactory> table_factory_to_write = nullptr,
    std::shared_ptr<TableFactory> block_based_table_factory = nullptr,
    std::shared_ptr<TableFactory> plain_table_factory = nullptr,
    std::shared_ptr<TableFactory> cuckoo_table_factory = nullptr);

#endif  // ROCKSDB_LITE

}  // namespace rocksdb
