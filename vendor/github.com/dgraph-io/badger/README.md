# BadgerDB [![GoDoc](https://godoc.org/github.com/dgraph-io/badger?status.svg)](https://godoc.org/github.com/dgraph-io/badger) [![Go Report Card](https://goreportcard.com/badge/github.com/dgraph-io/badger)](https://goreportcard.com/report/github.com/dgraph-io/badger) [![Build Status](https://teamcity.dgraph.io/guestAuth/app/rest/builds/buildType:(id:Badger_UnitTests)/statusIcon.svg)](https://teamcity.dgraph.io/viewLog.html?buildTypeId=Badger_UnitTests&buildId=lastFinished&guest=1) ![Appveyor](https://ci.appveyor.com/api/projects/status/github/dgraph-io/badger?branch=master&svg=true) [![Coverage Status](https://coveralls.io/repos/github/dgraph-io/badger/badge.svg?branch=master)](https://coveralls.io/github/dgraph-io/badger?branch=master)

![Badger mascot](images/diggy-shadow.png)

BadgerDB is an embeddable, persistent, simple and fast key-value (KV) database
written in pure Go. It's meant to be a performant alternative to non-Go-based
key-value stores like [RocksDB](https://github.com/facebook/rocksdb).

## Project Status
Badger v1.0 was released in Nov 2017. Check the [Changelog] for the full details.

[Changelog]:https://github.com/dgraph-io/badger/blob/master/CHANGELOG.md

We introduced transactions in [v0.9.0] which involved a major API change. If you have a Badger 
datastore prior to that, please use [v0.8.1], but we strongly urge you to upgrade. Upgrading from
both v0.8 and v0.9 will require you to [take backups](#database-backup) and restore using the new
version.

[v1.0.1]: //github.com/dgraph-io/badger/tree/v1.0.1
[v0.8.1]: //github.com/dgraph-io/badger/tree/v0.8.1
[v0.9.0]: //github.com/dgraph-io/badger/tree/v0.9.0

## Table of Contents
 * [Getting Started](#getting-started)
    + [Installing](#installing)
    + [Opening a database](#opening-a-database)
    + [Transactions](#transactions)
      - [Read-only transactions](#read-only-transactions)
      - [Read-write transactions](#read-write-transactions)
      - [Managing transactions manually](#managing-transactions-manually)
    + [Using key/value pairs](#using-keyvalue-pairs)
    + [Monotonically increasing integers](#monotonically-increasing-integers)
    * [Merge Operations](#merge-operations)
    + [Setting Time To Live(TTL) and User Metadata on Keys](#setting-time-to-livettl-and-user-metadata-on-keys)
    + [Iterating over keys](#iterating-over-keys)
      - [Prefix scans](#prefix-scans)
      - [Key-only iteration](#key-only-iteration)
    + [Garbage Collection](#garbage-collection)
    + [Database backup](#database-backup)
    + [Memory usage](#memory-usage)
    + [Statistics](#statistics)
  * [Resources](#resources)
    + [Blog Posts](#blog-posts)
  * [Contact](#contact)
  * [Design](#design)
    + [Comparisons](#comparisons)
    + [Benchmarks](#benchmarks)
  * [Other Projects Using Badger](#other-projects-using-badger)
  * [Frequently Asked Questions](#frequently-asked-questions)

## Getting Started

### Installing
To start using Badger, install Go 1.8 or above and run `go get`:

```sh
$ go get github.com/dgraph-io/badger/...
```

This will retrieve the library and install the `badger_info` command line
utility into your `$GOBIN` path.


### Opening a database
The top-level object in Badger is a `DB`. It represents multiple files on disk
in specific directories, which contain the data for a single database.

To open your database, use the `badger.Open()` function, with the appropriate
options. The `Dir` and `ValueDir` options are mandatory and must be
specified by the client. They can be set to the same value to simplify things.

```go
package main

import (
	"log"

	"github.com/dgraph-io/badger"
)

func main() {
  // Open the Badger database located in the /tmp/badger directory.
  // It will be created if it doesn't exist.
  opts := badger.DefaultOptions
  opts.Dir = "/tmp/badger"
  opts.ValueDir = "/tmp/badger"
  db, err := badger.Open(opts)
  if err != nil {
	  log.Fatal(err)
  }
  defer db.Close()
  // Your code here…
}
```

Please note that Badger obtains a lock on the directories so multiple processes
cannot open the same database at the same time.

### Transactions

#### Read-only transactions
To start a read-only transaction, you can use the `DB.View()` method:

```go
err := db.View(func(txn *badger.Txn) error {
  // Your code here…
  return nil
})
```

You cannot perform any writes or deletes within this transaction. Badger
ensures that you get a consistent view of the database within this closure. Any
writes that happen elsewhere after the transaction has started, will not be
seen by calls made within the closure.

#### Read-write transactions
To start a read-write transaction, you can use the `DB.Update()` method:

```go
err := db.Update(func(txn *badger.Txn) error {
  // Your code here…
  return nil
})
```

All database operations are allowed inside a read-write transaction.

Always check the returned error value. If you return an error
within your closure it will be passed through.

An `ErrConflict` error will be reported in case of a conflict. Depending on the state 
of your application, you have the option to retry the operation if you receive 
this error.

An `ErrTxnTooBig` will be reported in case the number of pending writes/deletes in
the transaction exceed a certain limit. In that case, it is best to commit the
transaction and start a new transaction immediately. Here is an example (we are
not checking for errors in some places for simplicity):

```go
updates := make(map[string]string)
txn := db.NewTransaction(true)
for k,v := range updates {
  if err := txn.Set(byte[](k),byte[](v)); err == ErrTxnTooBig {
    _ = txn.Commit()
    txn = db.NewTransaction(..)
    _ = txn.Set(k,v) 
  }
}
_ = txn.Commit()
```

#### Managing transactions manually
The `DB.View()` and `DB.Update()` methods are wrappers around the
`DB.NewTransaction()` and `Txn.Commit()` methods (or `Txn.Discard()` in case of
read-only transactions). These helper methods will start the transaction,
execute a function, and then safely discard your transaction if an error is
returned. This is the recommended way to use Badger transactions.

However, sometimes you may want to manually create and commit your
transactions. You can use the `DB.NewTransaction()` function directly, which
takes in a boolean argument to specify whether a read-write transaction is
required. For read-write transactions, it is necessary to call `Txn.Commit()`
to ensure the transaction is committed. For read-only transactions, calling
`Txn.Discard()` is sufficient. `Txn.Commit()` also calls `Txn.Discard()`
internally to cleanup the transaction, so just calling `Txn.Commit()` is
sufficient for read-write transaction. However, if your code doesn’t call
`Txn.Commit()` for some reason (for e.g it returns prematurely with an error),
then please make sure you call `Txn.Discard()` in a `defer` block. Refer to the
code below.

```go
// Start a writable transaction.
txn, err := db.NewTransaction(true)
if err != nil {
    return err
}
defer txn.Discard()

// Use the transaction...
err := txn.Set([]byte("answer"), []byte("42"))
if err != nil {
    return err
}

// Commit the transaction and check for error.
if err := txn.Commit(nil); err != nil {
    return err
}
```

The first argument to `DB.NewTransaction()` is a boolean stating if the transaction
should be writable.

Badger allows an optional callback to the `Txn.Commit()` method. Normally, the
callback can be set to `nil`, and the method will return after all the writes
have succeeded. However, if this callback is provided, the `Txn.Commit()`
method returns as soon as it has checked for any conflicts. The actual writing
to the disk happens asynchronously, and the callback is invoked once the
writing has finished, or an error has occurred. This can improve the throughput
of the application in some cases. But it also means that a transaction is not
durable until the callback has been invoked with a `nil` error value.

### Using key/value pairs
To save a key/value pair, use the `Txn.Set()` method:

```go
err := db.Update(func(txn *badger.Txn) error {
  err := txn.Set([]byte("answer"), []byte("42"))
  return err
})
```

This will set the value of the `"answer"` key to `"42"`. To retrieve this
value, we can use the `Txn.Get()` method:

```go
err := db.View(func(txn *badger.Txn) error {
  item, err := txn.Get([]byte("answer"))
  if err != nil {
    return err
  }
  val, err := item.Value()
  if err != nil {
    return err
  }
  fmt.Printf("The answer is: %s\n", val)
  return nil
})
```

`Txn.Get()` returns `ErrKeyNotFound` if the value is not found.

Please note that values returned from `Get()` are only valid while the
transaction is open. If you need to use a value outside of the transaction
then you must use `copy()` to copy it to another byte slice.

Use the `Txn.Delete()` method to delete a key.

### Monotonically increasing integers

To get unique monotonically increasing integers with strong durability, you can
use the `DB.GetSequence` method. This method returns a `Sequence` object, which
is thread-safe and can be used concurrently via various goroutines.

Badger would lease a range of integers to hand out from memory, with the
bandwidth provided to `DB.GetSequence`. The frequency at which disk writes are
done is determined by this lease bandwidth and the frequency of `Next`
invocations. Setting a bandwith too low would do more disk writes, setting it
too high would result in wasted integers if Badger is closed or crashes.
To avoid wasted integers, call `Release` before closing Badger.

```go
seq, err := db.GetSequence(key, 1000)
defer seq.Release()
for {
  num, err := seq.Next()
}
```

### Merge Operations
Badger provides support for unordered merge operations. You can define a func
of type `MergeFunc` which takes in an existing value, and a value to be
_merged_ with it. It returns a new value which is the result of the _merge_
operation. All values are specified in byte arrays. For e.g., here is a merge
function (`add`) which adds a `uint64` value to an existing `uint64` value.

```Go
uint64ToBytes(i uint64) []byte {
  var buf [8]byte
  binary.BigEndian.PutUint64(buf[:], i)
  return buf[:]
}

func bytesToUint64(b []byte) uint64 {
  return binary.BigEndian.Uint64(b)
}

// Merge function to add two uint64 numbers
func add(existing, new []byte) []byte {
  return uint64ToBytes(bytesToUint64(existing) + bytesToUint64(new))
}
```

This function can then be passed to the `DB.GetMergeOperator()` method, along
with a key, and a duration value. The duration specifies how often the merge
function is run on values that have been added using the `MergeOperator.Add()`
method.

`MergeOperator.Get()` method can be used to retrieve the cumulative value of the key
associated with the merge operation.

```Go
key := []byte("merge")
m := db.GetMergeOperator(key, add, 200*time.Millisecond)
defer m.Stop()

m.Add(uint64ToBytes(1))
m.Add(uint64ToBytes(2))
m.Add(uint64ToBytes(3))

res, err := m.Get() // res should have value 6 encoded
fmt.Println(bytesToUint64(res))
```

### Setting Time To Live(TTL) and User Metadata on Keys
Badger allows setting an optional Time to Live (TTL) value on keys. Once the TTL has
elapsed, the key will no longer be retrievable and will be eligible for garbage
collection. A TTL can be set as a `time.Duration` value using the `Txn.SetWithTTL()`
API method.

An optional user metadata value can be set on each key. A user metadata value
is represented by a single byte. It can be used to set certain bits along
with the key to aid in interpreting or decoding the key-value pair. User
metadata can be set using the `Txn.SetWithMeta()` API method.

`Txn.SetEntry()` can be used to set the key, value, user metatadata and TTL,
all at once.

### Iterating over keys
To iterate over keys, we can use an `Iterator`, which can be obtained using the
`Txn.NewIterator()` method. Iteration happens in byte-wise lexicographical sorting
order.


```go
err := db.View(func(txn *badger.Txn) error {
  opts := badger.DefaultIteratorOptions
  opts.PrefetchSize = 10
  it := txn.NewIterator(opts)
  for it.Rewind(); it.Valid(); it.Next() {
    item := it.Item()
    k := item.Key()
    v, err := item.Value()
    if err != nil {
      return err
    }
    fmt.Printf("key=%s, value=%s\n", k, v)
  }
  return nil
})
```

The iterator allows you to move to a specific point in the list of keys and move
forward or backward through the keys one at a time.

By default, Badger prefetches the values of the next 100 items. You can adjust
that with the `IteratorOptions.PrefetchSize` field. However, setting it to
a value higher than GOMAXPROCS (which we recommend to be 128 or higher)
shouldn’t give any additional benefits. You can also turn off the fetching of
values altogether. See section below on key-only iteration.

#### Prefix scans
To iterate over a key prefix, you can combine `Seek()` and `ValidForPrefix()`:

```go
db.View(func(txn *badger.Txn) error {
  it := txn.NewIterator(badger.DefaultIteratorOptions)
  prefix := []byte("1234")
  for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
    item := it.Item()
    k := item.Key()
    v, err := item.Value()
    if err != nil {
      return err
    }
    fmt.Printf("key=%s, value=%s\n", k, v)
  }
  return nil
})
```

#### Key-only iteration
Badger supports a unique mode of iteration called _key-only_ iteration. It is
several order of magnitudes faster than regular iteration, because it involves
access to the LSM-tree only, which is usually resident entirely in RAM. To
enable key-only iteration, you need to set the `IteratorOptions.PrefetchValues`
field to `false`. This can also be used to do sparse reads for selected keys
during an iteration, by calling `item.Value()` only when required.

```go
err := db.View(func(txn *badger.Txn) error {
  opts := badger.DefaultIteratorOptions
  opts.PrefetchValues = false
  it := txn.NewIterator(opts)
  for it.Rewind(); it.Valid(); it.Next() {
    item := it.Item()
    k := item.Key()
    fmt.Printf("key=%s\n", k)
  }
  return nil
})
```

### Garbage Collection
Badger values need to be garbage collected, because of two reasons:

* Badger keeps values separately from the LSM tree. This means that the compaction operations
that clean up the LSM tree do not touch the values at all. Values need to be cleaned up
separately.

* Concurrent read/write transactions could leave behind multiple values for a single key, because they
are stored with different versions. These could accumulate, and take up unneeded space beyond the
time these older versions are needed.

Badger relies on the client to perform garbage collection at a time of their choosing. It provides
the following methods, which can be invoked at an appropriate time:

* `DB.PurgeOlderVersions()`: This method iterates over the database, and cleans up all but the latest
versions of the key-value pairs. It marks the older versions as deleted, which makes them eligible for
garbage collection.
* `DB.PurgeVersionsBelow(key, ts)`: This method is useful to do a more targeted clean up of older versions
of key-value pairs. You can specify a key, and a timestamp. All versions of the key older than the timestamp
are marked as deleted, making them eligible for garbage collection.
* `DB.RunValueLogGC()`: This method is designed to do garbage collection while
  Badger is online. Please ensure that you call the `DB.Purge…()` methods first
  before invoking this method. It uses any statistics generated by the
  `DB.Purge(…)` methods to pick files that are likely to lead to maximum space
  reclamation. It loops until it encounters a file which does not lead to any
  garbage collection.

  It could lead to increased I/O if `DB.RunValueLogGC()` hasn’t been called for
  a long time, and many deletes have happened in the meanwhile. So it is recommended
  that this method be called regularly.

### Database backup
There are two public API methods `DB.Backup()` and `DB.Load()` which can be
used to do online backups and restores. Badger v0.9 provides a CLI tool
`badger`, which can do offline backup/restore. Make sure you have `$GOPATH/bin`
in your PATH to use this tool.

The command below will create a version-agnostic backup of the database, to a
file `badger.bak` in the current working directory

```
badger backup --dir <path/to/badgerdb>
```

To restore `badger.bak` in the current working directory to a new database:

```
badger restore --dir <path/to/badgerdb>
```

See `badger --help` for more details.

If you have a Badger database that was created using v0.8 (or below), you can
use the `badger_backup` tool provided in v0.8.1, and then restore it using the
command above to upgrade your database to work with the latest version.

```
badger_backup --dir <path/to/badgerdb> --backup-file badger.bak
```

### Memory usage
Badger's memory usage can be managed by tweaking several options available in
the `Options` struct that is passed in when opening the database using
`DB.Open`.

- `Options.ValueLogLoadingMode` can be set to `options.FileIO` (instead of the
  default `options.MemoryMap`) to avoid memory-mapping log files. This can be
  useful in environments with low RAM.
- Number of memtables (`Options.NumMemtables`)
  - If you modify `Options.NumMemtables`, also adjust `Options.NumLevelZeroTables` and
   `Options.NumLevelZeroTablesStall` accordingly.
- Number of concurrent compactions (`Options.NumCompactors`)
- Mode in which LSM tree is loaded (`Options.TableLoadingMode`)
- Size of table (`Options.MaxTableSize`)
- Size of value log file (`Options.ValueLogFileSize`)

If you want to decrease the memory usage of Badger instance, tweak these
options (ideally one at a time) until you achieve the desired
memory usage.

### Statistics
Badger records metrics using the [expvar] package, which is included in the Go
standard library. All the metrics are documented in [y/metrics.go][metrics]
file.

`expvar` package adds a handler in to the default HTTP server (which has to be
started explicitly), and serves up the metrics at the `/debug/vars` endpoint.
These metrics can then be collected by a system like [Prometheus], to get
better visibility into what Badger is doing.

[expvar]: https://golang.org/pkg/expvar/
[metrics]: https://github.com/dgraph-io/badger/blob/master/y/metrics.go
[Prometheus]: https://prometheus.io/

## Resources

### Blog Posts
1. [Introducing Badger: A fast key-value store written natively in
Go](https://open.dgraph.io/post/badger/)
2. [Make Badger crash resilient with ALICE](https://blog.dgraph.io/post/alice/)
3. [Badger vs LMDB vs BoltDB: Benchmarking key-value databases in Go](https://blog.dgraph.io/post/badger-lmdb-boltdb/)
4. [Concurrent ACID Transactions in Badger](https://blog.dgraph.io/post/badger-txn/)

## Design
Badger was written with these design goals in mind:

- Write a key-value database in pure Go.
- Use latest research to build the fastest KV database for data sets spanning terabytes.
- Optimize for SSDs.

Badger’s design is based on a paper titled _[WiscKey: Separating Keys from
Values in SSD-conscious Storage][wisckey]_.

[wisckey]: https://www.usenix.org/system/files/conference/fast16/fast16-papers-lu.pdf

### Comparisons
| Feature             | Badger                                       | RocksDB                       | BoltDB    |
| -------             | ------                                       | -------                       | ------    |
| Design              | LSM tree with value log                      | LSM tree only                 | B+ tree   |
| High Read throughput | Yes                                          | No                           | Yes        |
| High Write throughput | Yes                                          | Yes                           | No        |
| Designed for SSDs   | Yes (with latest research <sup>1</sup>)      | Not specifically <sup>2</sup> | No        |
| Embeddable          | Yes                                          | Yes                           | Yes       |
| Sorted KV access    | Yes                                          | Yes                           | Yes       |
| Pure Go (no Cgo)    | Yes                                          | No                            | Yes       |
| Transactions        | Yes, ACID, concurrent with SSI<sup>3</sup> | Yes (but non-ACID)            | Yes, ACID |
| Snapshots           | Yes                                           | Yes                           | Yes       |
| TTL support         | Yes                                           | Yes                           | No       |

<sup>1</sup> The [WISCKEY paper][wisckey] (on which Badger is based) saw big
wins with separating values from keys, significantly reducing the write
amplification compared to a typical LSM tree.

<sup>2</sup> RocksDB is an SSD optimized version of LevelDB, which was designed specifically for rotating disks.
As such RocksDB's design isn't aimed at SSDs.

<sup>3</sup> SSI: Serializable Snapshot Isolation. For more details, see the blog post [Concurrent ACID Transactions in Badger](https://blog.dgraph.io/post/badger-txn/)

### Benchmarks
We have run comprehensive benchmarks against RocksDB, Bolt and LMDB. The
benchmarking code, and the detailed logs for the benchmarks can be found in the
[badger-bench] repo. More explanation, including graphs can be found the blog posts (linked
above).

[badger-bench]: https://github.com/dgraph-io/badger-bench

## Other Projects Using Badger
Below is a list of public, open source projects that use Badger:

* [Dgraph](https://github.com/dgraph-io/dgraph) - Distributed graph database.
* [go-ipfs](https://github.com/ipfs/go-ipfs) - Go client for the InterPlanetary File System (IPFS), a new hypermedia distribution protocol.
* [0-stor](https://github.com/zero-os/0-stor) - Single device object store.
* [Sandglass](https://github.com/celrenheit/sandglass) - distributed, horizontally scalable, persistent, time sorted message queue.

If you are using Badger in a project please send a pull request to add it to the list.

## Frequently Asked Questions
- **My writes are getting stuck. Why?**

This can happen if a long running iteration with `Prefetch` is set to false, but
a `Item::Value` call is made internally in the loop. That causes Badger to
acquire read locks over the value log files to avoid value log GC removing the
file from underneath. As a side effect, this also blocks a new value log GC
file from being created, when the value log file boundary is hit.

Please see Github issues [#293](https://github.com/dgraph-io/badger/issues/293)
and [#315](https://github.com/dgraph-io/badger/issues/315).

There are multiple workarounds during iteration:

1. Use `Item::ValueCopy` instead of `Item::Value` when retrieving value.
1. Set `Prefetch` to true. Badger would then copy over the value and release the
   file lock immediately.
1. When `Prefetch` is false, don't call `Item::Value` and do a pure key-only
   iteration. This might be useful if you just want to delete a lot of keys.
1. Do the writes in a separate transaction after the reads.

- **My writes are really slow. Why?**

Are you creating a new transaction for every single key update? This will lead
to very low throughput. To get best write performance, batch up multiple writes
inside a transaction using single `DB.Update()` call. You could also have
multiple such `DB.Update()` calls being made concurrently from multiple
goroutines.

- **I don't see any disk write. Why?**

If you're using Badger with `SyncWrites=false`, then your writes might not be written to value log
and won't get synced to disk immediately. Writes to LSM tree are done inmemory first, before they
get compacted to disk. The compaction would only happen once `MaxTableSize` has been reached. So, if
you're doing a few writes and then checking, you might not see anything on disk. Once you `Close`
the database, you'll see these writes on disk.

- **Which instances should I use for Badger?**

We recommend using instances which provide local SSD storage, without any limit
on the maximum IOPS. In AWS, these are storage optimized instances like i3. They
provide local SSDs which clock 100K IOPS over 4KB blocks easily.

- **Are there any Go specific settings that I should use?**

We *highly* recommend setting a high number for GOMAXPROCS, which allows Go to
observe the full IOPS throughput provided by modern SSDs. In Dgraph, we have set
it to 128. For more details, [see this
thread](https://groups.google.com/d/topic/golang-nuts/jPb_h3TvlKE/discussion).

## Contact
- Please use [discuss.dgraph.io](https://discuss.dgraph.io) for questions, feature requests and discussions.
- Please use [Github issue tracker](https://github.com/dgraph-io/badger/issues) for filing bugs or feature requests.
- Join [![Slack Status](http://slack.dgraph.io/badge.svg)](http://slack.dgraph.io).
- Follow us on Twitter [@dgraphlabs](https://twitter.com/dgraphlabs).

