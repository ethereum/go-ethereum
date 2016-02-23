# RocksJava Change Log

## 3.13 (8/4/2015)
### New Features
* Exposed BackupEngine API.
* Added CappedPrefixExtractor support.  To use such extractor, simply call useCappedPrefixExtractor in either Options or ColumnFamilyOptions.
* Added RemoveEmptyValueCompactionFilter.

## 3.10.0 (3/24/2015)
### New Features
* Added compression per level API.
* MemEnv is now available in RocksJava via RocksMemEnv class.
* lz4 compression is now included in rocksjava static library when running `make rocksdbjavastatic`.

### Public API Changes
* Overflowing a size_t when setting rocksdb options now throws an IllegalArgumentException, which removes the necessity for a developer to catch these Exceptions explicitly.
* The set and get functions for tableCacheRemoveScanCountLimit are deprecated.


## By 01/31/2015
### New Features
* WriteBatchWithIndex support.
* Iterator support for WriteBatch and WriteBatchWithIndex
* GetUpdatesSince support.
* Snapshots carry now information about the related sequence number.
* TTL DB support.

## By 11/14/2014
### New Features
* Full support for Column Family.
* Slice and Comparator support.
* Default merge operator support.
* RateLimiter support.

## By 06/15/2014
### New Features
* Added basic Java binding for rocksdb::Env such that multiple RocksDB can share the same thread pool and environment.
* Added RestoreBackupableDB

## By 05/30/2014
### Internal Framework Improvement
* Added disOwnNativeHandle to RocksObject, which allows a RocksObject to give-up the ownership of its native handle.  This method is useful when sharing and transferring the ownership of RocksDB C++ resources.

## By 05/15/2014
### New Features
* Added RocksObject --- the base class of all RocksDB classes which holds some RocksDB resources in the C++ side.
* Use environmental variable JAVA_HOME in Makefile for RocksJava
### Public API changes
* Renamed org.rocksdb.Iterator to org.rocksdb.RocksIterator to avoid potential confliction with Java built-in Iterator.

## By 04/30/2014
### New Features
* Added Java binding for MultiGet.
* Added static method RocksDB.loadLibrary(), which loads necessary library files.
* Added Java bindings for 60+ rocksdb::Options.
* Added Java binding for BloomFilter.
* Added Java binding for ReadOptions.
* Added Java binding for memtables.
* Added Java binding for sst formats.
* Added Java binding for RocksDB Iterator which enables sequential scan operation.
* Added Java binding for Statistics
* Added Java binding for BackupableDB.

### DB Benchmark
* Added filluniquerandom, readseq benchmark.
* 70+ command-line options.
* Enabled BloomFilter configuration.

## By 04/15/2014
### New Features
* Added Java binding for WriteOptions.
* Added Java binding for WriteBatch, which enables batch-write.
* Added Java binding for rocksdb::Options.
* Added Java binding for block cache.
* Added Java version DB Benchmark.

### DB Benchmark
* Added readwhilewriting benchmark.

### Internal Framework Improvement
* Avoid a potential byte-array-copy between c++ and Java in RocksDB.get.
* Added SizeUnit in org.rocksdb.util to store consts like KB and GB.

### 03/28/2014
* RocksJava project started.
* Added Java binding for RocksDB, which supports Open, Close, Get and Put.
