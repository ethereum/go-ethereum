# RocksDBLite

RocksDBLite is a project focused on mobile use cases, which don't need a lot of fancy things we've built for server workloads and they are very sensitive to binary size. For that reason, we added a compile flag ROCKSDB_LITE that comments out a lot of the nonessential code and keeps the binary lean.

Some examples of the features disabled by ROCKSDB_LITE:
* compiled-in support for LDB tool
* No backupable DB
* No support for replication (which we provide in form of TrasactionalIterator)
* No advanced monitoring tools
* No special-purpose memtables that are highly optimized for specific use cases
* No Transactions

When adding a new big feature to RocksDB, please add ROCKSDB_LITE compile guard if:
* Nobody from mobile really needs your feature,
* Your feature is adding a lot of weight to the binary.

Don't add ROCKSDB_LITE compile guard if:
* It would introduce a lot of code complexity. Compile guards make code harder to read. It's a trade-off.
* Your feature is not adding a lot of weight.

If unsure, ask. :)
