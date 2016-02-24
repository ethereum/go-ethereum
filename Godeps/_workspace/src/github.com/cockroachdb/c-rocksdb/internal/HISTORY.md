# Rocksdb Change Log

## 4.0.0 (9/9/2015)
### New Features
* Added support for transactions.  See include/rocksdb/utilities/transaction.h for more info.
* DB::GetProperty() now accept "rocksdb.aggregated-table-properties" and "rocksdb.aggregated-table-properties-at-levelN", in which case it returns aggregated table properties of the target column family, or the aggregated table properties of the specified level N if the "at-level" version is used.
* Add compression option kZSTDNotFinalCompression for people to experiment ZSTD although its format is not finalized.
* We removed the need for LATEST_BACKUP file in BackupEngine. We still keep writing it when we create new backups (because of backward compatibility), but we don't read it anymore.

### Public API Changes
* Removed class Env::RandomRWFile and Env::NewRandomRWFile().
* Renamed DBOptions.num_subcompactions to DBOptions.max_subcompactions to make the name better match the actual functionality of the option.
* Added Equal() method to the Comparator interface that can optionally be overwritten in cases where equality comparisons can be done more efficiently than three-way comparisons.
* Previous 'experimental' OptimisticTransaction class has been replaced by Transaction class.

## 3.13.0 (8/6/2015)
### New Features
* RollbackToSavePoint() in WriteBatch/WriteBatchWithIndex
* Add NewCompactOnDeletionCollectorFactory() in utilities/table_properties_collectors, which allows rocksdb to mark a SST file as need-compaction when it observes at least D deletion entries in any N consecutive entries in that SST file.  Note that this feature depends on an experimental NeedCompact() API --- the result of this API will not persist after DB restart.
* Add DBOptions::delete_scheduler. Use NewDeleteScheduler() in include/rocksdb/delete_scheduler.h to create a DeleteScheduler that can be shared among multiple RocksDB instances to control the file deletion rate of SST files that exist in the first db_path.

### Public API Changes
* Deprecated WriteOptions::timeout_hint_us. We no longer support write timeout. If you really need this option, talk to us and we might consider returning it.
* Deprecated purge_redundant_kvs_while_flush option.
* Removed BackupEngine::NewBackupEngine() and NewReadOnlyBackupEngine() that were deprecated in RocksDB 3.8. Please use BackupEngine::Open() instead.
* Deprecated Compaction Filter V2. We are not aware of any existing use-cases. If you use this filter, your compile will break with RocksDB 3.13. Please let us know if you use it and we'll put it back in RocksDB 3.14.
* Env::FileExists now returns a Status instead of a boolean
* Add statistics::getHistogramString() to print detailed distribution of a histogram metric.
* Add DBOptions::skip_stats_update_on_db_open.  When it is on, DB::Open() will run faster as it skips the random reads required for loading necessary stats from SST files to optimize compaction.

## 3.12.0 (7/2/2015)
### New Features
* Added experimental support for optimistic transactions.  See include/rocksdb/utilities/optimistic_transaction.h for more info.
* Added a new way to report QPS from db_bench (check out --report_file and --report_interval_seconds)
* Added a cache for individual rows. See DBOptions::row_cache for more info.
* Several new features on EventListener (see include/rocksdb/listener.h):
 - OnCompationCompleted() now returns per-compaciton job statistics, defined in include/rocksdb/compaction_job_stats.h.
 - Added OnTableFileCreated() and OnTableFileDeleted().
* Add compaction_options_universal.enable_trivial_move to true, to allow trivial move while performing universal compaction. Trivial move will happen only when all the input files are non overlapping.

### Public API changes
* EventListener::OnFlushCompleted() now passes FlushJobInfo instead of a list of parameters.
* DB::GetDbIdentity() is now a const function.  If this function is overridden in your application, be sure to also make GetDbIdentity() const to avoid compile error.
* Move listeners from ColumnFamilyOptions to DBOptions.
* Add max_write_buffer_number_to_maintain option
* DB::CompactRange()'s parameter reduce_level is changed to change_level, to allow users to move levels to lower levels if allowed. It can be used to migrate a DB from options.level_compaction_dynamic_level_bytes=false to options.level_compaction_dynamic_level_bytes.true.
* Change default value for options.compaction_filter_factory and options.compaction_filter_factory_v2 to nullptr instead of DefaultCompactionFilterFactory and DefaultCompactionFilterFactoryV2.
* If CancelAllBackgroundWork is called without doing a flush after doing loads with WAL disabled, the changes which haven't been flushed before the call to CancelAllBackgroundWork will be lost.
* WBWIIterator::Entry() now returns WriteEntry instead of `const WriteEntry&`
* options.hard_rate_limit is deprecated.
* When options.soft_rate_limit or options.level0_slowdown_writes_trigger is triggered, the way to slow down writes is changed to: write rate to DB is limited to to options.delayed_write_rate.
* DB::GetApproximateSizes() adds a parameter to allow the estimation to include data in mem table, with default to be not to include. It is now only supported in skip list mem table.
* DB::CompactRange() now accept CompactRangeOptions instead of multiple paramters. CompactRangeOptions is defined in include/rocksdb/options.h.
* CompactRange() will now skip bottommost level compaction for level based compaction if there is no compaction filter, bottommost_level_compaction is introduced in CompactRangeOptions to control when it's possbile to skip bottommost level compaction. This mean that if you want the compaction to produce a single file you need to set bottommost_level_compaction to BottommostLevelCompaction::kForce.
* Add Cache.GetPinnedUsage() to get the size of memory occupied by entries that are in use by the system.
* DB:Open() will fail if the compression specified in Options is not linked with the binary. If you see this failure, recompile RocksDB with compression libraries present on your system. Also, previously our default compression was snappy. This behavior is now changed. Now, the default compression is snappy only if it's available on the system. If it isn't we change the default to kNoCompression.
* We changed how we account for memory used in block cache. Previously, we only counted the sum of block sizes currently present in block cache. Now, we count the actual memory usage of the blocks. For example, a block of size 4.5KB will use 8KB memory with jemalloc. This might decrease your memory usage and possibly decrease performance. Increase block cache size if you see this happening after an upgrade.
* Add BackupEngineImpl.options_.max_background_operations to specify the maximum number of operations that may be performed in parallel. Add support for parallelized backup and restore.
* Add DB::SyncWAL() that does a WAL sync without blocking writers.

## 3.11.0 (5/19/2015)
### New Features
* Added a new API Cache::SetCapacity(size_t capacity) to dynamically change the maximum configured capacity of the cache. If the new capacity is less than the existing cache usage, the implementation will try to lower the usage by evicting the necessary number of elements following a strict LRU policy.
* Added an experimental API for handling flashcache devices (blacklists background threads from caching their reads) -- NewFlashcacheAwareEnv
* If universal compaction is used and options.num_levels > 1, compact files are tried to be stored in none-L0 with smaller files based on options.target_file_size_base. The limitation of DB size when using universal compaction is greatly mitigated by using more levels. You can set num_levels = 1 to make universal compaction behave as before. If you set num_levels > 1 and want to roll back to a previous version, you need to compact all files to a big file in level 0 (by setting target_file_size_base to be large and CompactRange(<cf_handle>, nullptr, nullptr, true, 0) and reopen the DB with the same version to rewrite the manifest, and then you can open it using previous releases.
* More information about rocksdb background threads are available in Env::GetThreadList(), including the number of bytes read / written by a compaction job, mem-table size and current number of bytes written by a flush job and many more.  Check include/rocksdb/thread_status.h for more detail.

### Public API changes
* TablePropertiesCollector::AddUserKey() is added to replace TablePropertiesCollector::Add(). AddUserKey() exposes key type, sequence number and file size up to now to users.
* DBOptions::bytes_per_sync used to apply to both WAL and table files. As of 3.11 it applies only to table files. If you want to use this option to sync WAL in the background, please use wal_bytes_per_sync

## 3.10.0 (3/24/2015)
### New Features
* GetThreadStatus() is now able to report detailed thread status, including:
 - Thread Operation including flush and compaction.
 - The stage of the current thread operation.
 - The elapsed time in micros since the current thread operation started.
 More information can be found in include/rocksdb/thread_status.h.  In addition, when running db_bench with --thread_status_per_interval, db_bench will also report thread status periodically.
* Changed the LRU caching algorithm so that referenced blocks (by iterators) are never evicted. This change made parameter removeScanCountLimit obsolete. Because of that NewLRUCache doesn't take three arguments anymore. table_cache_remove_scan_limit option is also removed
* By default we now optimize the compilation for the compilation platform (using -march=native). If you want to build portable binary, use 'PORTABLE=1' before the make command.
* We now allow level-compaction to place files in different paths by
  specifying them in db_paths along with the target_size.
  Lower numbered levels will be placed earlier in the db_paths and higher
  numbered levels will be placed later in the db_paths vector.
* Potentially big performance improvements if you're using RocksDB with lots of column families (100-1000)
* Added BlockBasedTableOptions.format_version option, which allows user to specify which version of block based table he wants. As a general guidline, newer versions have more features, but might not be readable by older versions of RocksDB.
* Added new block based table format (version 2), which you can enable by setting BlockBasedTableOptions.format_version = 2. This format changes how we encode size information in compressed blocks and should help with memory allocations if you're using Zlib or BZip2 compressions.
* MemEnv (env that stores data in memory) is now available in default library build. You can create it by calling NewMemEnv().
* Add SliceTransform.SameResultWhenAppended() to help users determine it is safe to apply prefix bloom/hash.
* Block based table now makes use of prefix bloom filter if it is a full fulter.
* Block based table remembers whether a whole key or prefix based bloom filter is supported in SST files. Do a sanity check when reading the file with users' configuration.
* Fixed a bug in ReadOnlyBackupEngine that deleted corrupted backups in some cases, even though the engine was ReadOnly
* options.level_compaction_dynamic_level_bytes, a feature to allow RocksDB to pick dynamic base of bytes for levels. With this feature turned on, we will automatically adjust max bytes for each level. The goal of this feature is to have lower bound on size amplification. For more details, see comments in options.h.
* Added an abstract base class WriteBatchBase for write batches
* Fixed a bug where we start deleting files of a dropped column families even if there are still live references to it

### Public API changes
* Deprecated skip_log_error_on_recovery and table_cache_remove_scan_count_limit options.
* Logger method logv with log level parameter is now virtual

### RocksJava
* Added compression per level API.
* MemEnv is now available in RocksJava via RocksMemEnv class.
* lz4 compression is now included in rocksjava static library when running `make rocksdbjavastatic`.
* Overflowing a size_t when setting rocksdb options now throws an IllegalArgumentException, which removes the necessity for a developer to catch these Exceptions explicitly.

## 3.9.0 (12/8/2014)

### New Features
* Add rocksdb::GetThreadList(), which in the future will return the current status of all
  rocksdb-related threads.  We will have more code instruments in the following RocksDB
  releases.
* Change convert function in rocksdb/utilities/convenience.h to return Status instead of boolean.
  Also add support for nested options in convert function

### Public API changes
* New API to create a checkpoint added. Given a directory name, creates a new
  database which is an image of the existing database.
* New API LinkFile added to Env. If you implement your own Env class, an
  implementation of the API LinkFile will have to be provided.
* MemTableRep takes MemTableAllocator instead of Arena

### Improvements
* RocksDBLite library now becomes smaller and will be compiled with -fno-exceptions flag.

## 3.8.0 (11/14/2014)

### Public API changes
* BackupEngine::NewBackupEngine() was deprecated; please use BackupEngine::Open() from now on.
* BackupableDB/RestoreBackupableDB have new GarbageCollect() methods, which will clean up files from corrupt and obsolete backups.
* BackupableDB/RestoreBackupableDB have new GetCorruptedBackups() methods which list corrupt backups.

### Cleanup
* Bunch of code cleanup, some extra warnings turned on (-Wshadow, -Wshorten-64-to-32, -Wnon-virtual-dtor)

### New features
* CompactFiles and EventListener, although they are still in experimental state
* Full ColumnFamily support in RocksJava.

## 3.7.0 (11/6/2014)
### Public API changes
* Introduce SetOptions() API to allow adjusting a subset of options dynamically online
* Introduce 4 new convenient functions for converting Options from string: GetColumnFamilyOptionsFromMap(), GetColumnFamilyOptionsFromString(), GetDBOptionsFromMap(), GetDBOptionsFromString()
* Remove WriteBatchWithIndex.Delete() overloads using SliceParts
* When opening a DB, if options.max_background_compactions is larger than the existing low pri pool of options.env, it will enlarge it. Similarly, options.max_background_flushes is larger than the existing high pri pool of options.env, it will enlarge it.

## 3.6.0 (10/7/2014)
### Disk format changes
* If you're using RocksDB on ARM platforms and you're using default bloom filter, there is a disk format change you need to be aware of. There are three steps you need to do when you convert to new release: 1. turn off filter policy, 2. compact the whole database, 3. turn on filter policy

### Behavior changes
* We have refactored our system of stalling writes.  Any stall-related statistics' meanings are changed. Instead of per-write stall counts, we now count stalls per-epoch, where epochs are periods between flushes and compactions. You'll find more information in our Tuning Perf Guide once we release RocksDB 3.6.
* When disableDataSync=true, we no longer sync the MANIFEST file.
* Add identity_as_first_hash property to CuckooTable. SST file needs to be rebuilt to be opened by reader properly.

### Public API changes
* Change target_file_size_base type to uint64_t from int.
* Remove allow_thread_local. This feature was proved to be stable, so we are turning it always-on.

## 3.5.0 (9/3/2014)
### New Features
* Add include/utilities/write_batch_with_index.h, providing a utilitiy class to query data out of WriteBatch when building it.
* Move BlockBasedTable related options to BlockBasedTableOptions from Options. Change corresponding JNI interface. Options affected include:
  no_block_cache, block_cache, block_cache_compressed, block_size, block_size_deviation, block_restart_interval, filter_policy, whole_key_filtering. filter_policy is changed to shared_ptr from a raw pointer.
* Remove deprecated options: disable_seek_compaction and db_stats_log_interval
* OptimizeForPointLookup() takes one parameter for block cache size. It now builds hash index, bloom filter, and block cache.

### Public API changes
* The Prefix Extractor used with V2 compaction filters is now passed user key to SliceTransform::Transform instead of unparsed RocksDB key.

## 3.4.0 (8/18/2014)
### New Features
* Support Multiple DB paths in universal style compactions
* Add feature of storing plain table index and bloom filter in SST file.
* CompactRange() will never output compacted files to level 0. This used to be the case when all the compaction input files were at level 0.
* Added iterate_upper_bound to define the extent upto which the forward iterator will return entries. This will prevent iterating over delete markers and overwritten entries for edge cases where you want to break out the iterator anyways. This may improve perfomance in case there are a large number of delete markers or overwritten entries.

### Public API changes
* DBOptions.db_paths now is a vector of a DBPath structure which indicates both of path and target size
* NewPlainTableFactory instead of bunch of parameters now accepts PlainTableOptions, which is defined in include/rocksdb/table.h
* Moved include/utilities/*.h to include/rocksdb/utilities/*.h
* Statistics APIs now take uint32_t as type instead of Tickers. Also make two access functions getTickerCount and histogramData const
* Add DB property rocksdb.estimate-num-keys, estimated number of live keys in DB.
* Add DB::GetIntProperty(), which returns DB properties that are integer as uint64_t.
* The Prefix Extractor used with V2 compaction filters is now passed user key to SliceTransform::Transform instead of unparsed RocksDB key.

## 3.3.0 (7/10/2014)
### New Features
* Added JSON API prototype.
* HashLinklist reduces performance outlier caused by skewed bucket by switching data in the bucket from linked list to skip list. Add parameter threshold_use_skiplist in NewHashLinkListRepFactory().
* RocksDB is now able to reclaim storage space more effectively during the compaction process.  This is done by compensating the size of each deletion entry by the 2X average value size, which makes compaction to be triggerred by deletion entries more easily.
* Add TimeOut API to write.  Now WriteOptions have a variable called timeout_hint_us.  With timeout_hint_us set to non-zero, any write associated with this timeout_hint_us may be aborted when it runs longer than the specified timeout_hint_us, and it is guaranteed that any write completes earlier than the specified time-out will not be aborted due to the time-out condition.
* Add a rate_limiter option, which controls total throughput of flush and compaction. The throughput is specified in bytes/sec. Flush always has precedence over compaction when available bandwidth is constrained.

### Public API changes
* Removed NewTotalOrderPlainTableFactory because it is not used and implemented semantically incorrect.

## 3.2.0 (06/20/2014)

### Public API changes
* We removed seek compaction as a concept from RocksDB because:
1) It makes more sense for spinning disk workloads, while RocksDB is primarily designed for flash and memory,
2) It added some complexity to the important code-paths,
3) None of our internal customers were really using it.
Because of that, Options::disable_seek_compaction is now obsolete. It is still a parameter in Options, so it does not break the build, but it does not have any effect. We plan to completely remove it at some point, so we ask users to please remove this option from your code base.
* Add two paramters to NewHashLinkListRepFactory() for logging on too many entries in a hash bucket when flushing.
* Added new option BlockBasedTableOptions::hash_index_allow_collision. When enabled, prefix hash index for block-based table will not store prefix and allow hash collision, reducing memory consumption.

### New Features
* PlainTable now supports a new key encoding: for keys of the same prefix, the prefix is only written once. It can be enabled through encoding_type paramter of NewPlainTableFactory()
* Add AdaptiveTableFactory, which is used to convert from a DB of PlainTable to BlockBasedTabe, or vise versa. It can be created using NewAdaptiveTableFactory()

### Performance Improvements
* Tailing Iterator re-implemeted with ForwardIterator + Cascading Search Hint , see ~20% throughput improvement.

## 3.1.0 (05/21/2014)

### Public API changes
* Replaced ColumnFamilyOptions::table_properties_collectors with ColumnFamilyOptions::table_properties_collector_factories

### New Features
* Hash index for block-based table will be materialized and reconstructed more efficiently. Previously hash index is constructed by scanning the whole table during every table open.
* FIFO compaction style

## 3.0.0 (05/05/2014)

### Public API changes
* Added _LEVEL to all InfoLogLevel enums
* Deprecated ReadOptions.prefix and ReadOptions.prefix_seek. Seek() defaults to prefix-based seek when Options.prefix_extractor is supplied. More detail is documented in https://github.com/facebook/rocksdb/wiki/Prefix-Seek-API-Changes
* MemTableRepFactory::CreateMemTableRep() takes info logger as an extra parameter.

### New Features
* Column family support
* Added an option to use different checksum functions in BlockBasedTableOptions
* Added ApplyToAllCacheEntries() function to Cache

## 2.8.0 (04/04/2014)

* Removed arena.h from public header files.
* By default, checksums are verified on every read from database
* Change default value of several options, including: paranoid_checks=true, max_open_files=5000, level0_slowdown_writes_trigger=20, level0_stop_writes_trigger=24, disable_seek_compaction=true, max_background_flushes=1 and allow_mmap_writes=false
* Added is_manual_compaction to CompactionFilter::Context
* Added "virtual void WaitForJoin()" in class Env. Default operation is no-op.
* Removed BackupEngine::DeleteBackupsNewerThan() function
* Added new option -- verify_checksums_in_compaction
* Changed Options.prefix_extractor from raw pointer to shared_ptr (take ownership)
  Changed HashSkipListRepFactory and HashLinkListRepFactory constructor to not take SliceTransform object (use Options.prefix_extractor implicitly)
* Added Env::GetThreadPoolQueueLen(), which returns the waiting queue length of thread pools
* Added a command "checkconsistency" in ldb tool, which checks
  if file system state matches DB state (file existence and file sizes)
* Separate options related to block based table to a new struct BlockBasedTableOptions.
* WriteBatch has a new function Count() to return total size in the batch, and Data() now returns a reference instead of a copy
* Add more counters to perf context.
* Supports several more DB properties: compaction-pending, background-errors and cur-size-active-mem-table.

### New Features
* If we find one truncated record at the end of the MANIFEST or WAL files,
  we will ignore it. We assume that writers of these records were interrupted
  and that we can safely ignore it.
* A new SST format "PlainTable" is added, which is optimized for memory-only workloads. It can be created through NewPlainTableFactory() or NewTotalOrderPlainTableFactory().
* A new mem table implementation hash linked list optimizing for the case that there are only few keys for each prefix, which can be created through NewHashLinkListRepFactory().
* Merge operator supports a new function PartialMergeMulti() to allow users to do partial merges against multiple operands.
* Now compaction filter has a V2 interface. It buffers the kv-pairs sharing the same key prefix, process them in batches, and return the batched results back to DB. The new interface uses a new structure CompactionFilterContext for the same purpose as CompactionFilter::Context in V1.
* Geo-spatial support for locations and radial-search.

## 2.7.0 (01/28/2014)

### Public API changes

* Renamed `StackableDB::GetRawDB()` to `StackableDB::GetBaseDB()`.
* Renamed `WriteBatch::Data()` `const std::string& Data() const`.
* Renamed class `TableStats` to `TableProperties`.
* Deleted class `PrefixHashRepFactory`. Please use `NewHashSkipListRepFactory()` instead.
* Supported multi-threaded `EnableFileDeletions()` and `DisableFileDeletions()`.
* Added `DB::GetOptions()`.
* Added `DB::GetDbIdentity()`.

### New Features

* Added [BackupableDB](https://github.com/facebook/rocksdb/wiki/How-to-backup-RocksDB%3F)
* Implemented [TailingIterator](https://github.com/facebook/rocksdb/wiki/Tailing-Iterator), a special type of iterator that
  doesn't create a snapshot (can be used to read newly inserted data)
  and is optimized for doing sequential reads.
* Added property block for table, which allows (1) a table to store
  its metadata and (2) end user to collect and store properties they
  are interested in.
* Enabled caching index and filter block in block cache (turned off by default).
* Supported error report when doing manual compaction.
* Supported additional Linux platform flavors and Mac OS.
* Put with `SliceParts` - Variant of `Put()` that gathers output like `writev(2)`
* Bug fixes and code refactor for compatibility with upcoming Column
  Family feature.

### Performance Improvements

* Huge benchmark performance improvements by multiple efforts. For example, increase in readonly QPS from about 530k in 2.6 release to 1.1 million in 2.7 [1]
* Speeding up a way RocksDB deleted obsolete files - no longer listing the whole directory under a lock -- decrease in p99
* Use raw pointer instead of shared pointer for statistics: [5b825d](https://github.com/facebook/rocksdb/commit/5b825d6964e26ec3b4bb6faa708ebb1787f1d7bd) -- huge increase in performance -- shared pointers are slow
* Optimized locking for `Get()` -- [1fdb3f](https://github.com/facebook/rocksdb/commit/1fdb3f7dc60e96394e3e5b69a46ede5d67fb976c) -- 1.5x QPS increase for some workloads
* Cache speedup - [e8d40c3](https://github.com/facebook/rocksdb/commit/e8d40c31b3cca0c3e1ae9abe9b9003b1288026a9)
* Implemented autovector, which allocates first N elements on stack. Most of vectors in RocksDB are small. Also, we never want to allocate heap objects while holding a mutex. -- [c01676e4](https://github.com/facebook/rocksdb/commit/c01676e46d3be08c3c140361ef1f5884f47d3b3c)
* Lots of efforts to move malloc, memcpy and IO outside of locks
