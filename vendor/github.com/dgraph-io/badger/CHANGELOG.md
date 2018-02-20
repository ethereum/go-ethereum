# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/en/1.0.0/)
and this project adheres to [Semantic Versioning](http://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.3.0] - 2017-12-12
* Add `DB.NextSequence()` method to generate monotonically increasing integer
  sequences.
* Add `DB.Size()` method to return the size of LSM and value log files.
* Tweaked mmap code to make Windows 32-bit builds work.
* Tweaked build tags on some files to make iOS builds work.
* Fix `DB.PurgeOlderVersions()` to not violate some constraints.

## [1.2.0] - 2017-11-30
* Expose a `Txn.SetEntry()` method to allow setting the key-value pair
  and all the metadata at the same time.

## [1.1.1] - 2017-11-28
* Fix bug where txn.Get was returing key deleted in same transaction.
* Fix race condition while decrementing reference in oracle.
* Update doneCommit in the callback for CommitAsync.
* Iterator see writes of current txn.

## [1.1.0] - 2017-11-13
* Create Badger directory if it does not exist when `badger.Open` is called.
* Added `Item.ValueCopy()` to avoid deadlocks in long-running iterations
* Fixed 64-bit alignment issues to make Badger run on Arm v7

## [1.0.1] - 2017-11-06
* Fix an uint16 overflow when resizing key slice

[Unreleased]: https://github.com/dgraph-io/badger/compare/v1.3.0...HEAD
[1.3.0]: https://github.com/dgraph-io/badger/compare/v1.2.0...v1.3.0
[1.2.0]: https://github.com/dgraph-io/badger/compare/v1.1.1...v1.2.0
[1.1.1]: https://github.com/dgraph-io/badger/compare/v1.1.0...v1.1.1
[1.1.0]: https://github.com/dgraph-io/badger/compare/v1.0.1...v1.1.0
[1.0.1]: https://github.com/dgraph-io/badger/compare/v1.0.0...v1.0.1
