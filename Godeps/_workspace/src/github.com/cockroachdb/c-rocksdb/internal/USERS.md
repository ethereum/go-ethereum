This document lists users of RocksDB and their use cases. If you are using RocksDB, please open a pull request and add yourself to the list.

## Facebook
At Facebook, we use RocksDB as a backend for many different stateful services. We're also experimenting with running RocksDB as a storage engine for two databases:

1. MyRocks -- https://github.com/MySQLOnRocksDB/mysql-5.6
2. MongoRocks -- https://github.com/mongodb-partners/mongo-rocks

## LinkedIn
Two different use cases at Linkedin are using RocksDB as a storage engine:

1. LinkedIn's follow feed for storing user's activities
2. Apache Samza, open source framework for stream processing

Learn more about those use cases in a Tech Talk by Ankit Gupta and Naveen Somasundaram: http://www.youtube.com/watch?v=plqVp_OnSzg

## Yahoo
Yahoo is using RocksDB as a storage engine for their biggest distributed data store Sherpa. Learn more about it here: http://yahooeng.tumblr.com/post/120730204806/sherpa-scales-new-heights

## CockroachDB
CockroachDB is an open-source geo-replicated transactional database (still in development). They are using RocksDB as their storage engine. Check out their github: https://github.com/cockroachdb/cockroach

## DNANexus
DNANexus is using RocksDB to speed up processing of genomics data.
You can learn more from this great blog post by Mike Lin: http://devblog.dnanexus.com/faster-bam-sorting-with-samtools-and-rocksdb/

## Iron.io
Iron.io is using RocksDB as a storage engine for their distributed queueing system.
Learn more from Tech Talk by Reed Allman: http://www.youtube.com/watch?v=HTjt6oj-RL4

## Tango Me
Tango is using RocksDB as a graph storage to store all users' connection data and other social activity data.

## Turn
Turn is using RocksDB as a storage layer for their key/value store, serving at peak 2.4MM QPS out of different datacenters.
Check out our RocksDB Protobuf merge operator at: https://github.com/vladb38/rocksdb_protobuf

## Santanader UK/Cloudera Profession Services
Check out their blog post: http://blog.cloudera.com/blog/2015/08/inside-santanders-near-real-time-data-ingest-architecture/
