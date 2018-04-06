/*
 * Copyright 2017 Dgraph Labs, Inc. and Contributors
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

package badger

// ManagedDB allows end users to manage the transactions themselves. Transaction
// start and commit timestamps are set by end-user.
//
// This is only useful for databases built on top of Badger (like Dgraph), and
// can be ignored by most users.
//
// WARNING: This is an experimental feature and may be changed significantly in
// a future release. So please proceed with caution.
type ManagedDB struct {
	*DB
}

// OpenManaged returns a new ManagedDB, which allows more control over setting
// transaction timestamps.
//
// This is only useful for databases built on top of Badger (like Dgraph), and
// can be ignored by most users.
func OpenManaged(opts Options) (*ManagedDB, error) {
	opts.managedTxns = true
	db, err := Open(opts)
	if err != nil {
		return nil, err
	}
	return &ManagedDB{db}, nil
}

// NewTransaction overrides DB.NewTransaction() and panics when invoked. Use
// NewTransactionAt() instead.
func (db *ManagedDB) NewTransaction(update bool) {
	panic("Cannot use NewTransaction() for ManagedDB. Use NewTransactionAt() instead.")
}

// NewTransactionAt follows the same logic as DB.NewTransaction(), but uses the
// provided read timestamp.
//
// This is only useful for databases built on top of Badger (like Dgraph), and
// can be ignored by most users.
func (db *ManagedDB) NewTransactionAt(readTs uint64, update bool) *Txn {
	txn := db.DB.NewTransaction(update)
	txn.readTs = readTs
	return txn
}

// CommitAt commits the transaction, following the same logic as Commit(), but
// at the given commit timestamp. This will panic if not used with ManagedDB.
//
// This is only useful for databases built on top of Badger (like Dgraph), and
// can be ignored by most users.
func (txn *Txn) CommitAt(commitTs uint64, callback func(error)) error {
	if !txn.db.opt.managedTxns {
		return ErrManagedTxn
	}
	txn.commitTs = commitTs
	return txn.Commit(callback)
}

// PurgeVersionsBelow will delete all versions of a key below the specified version
func (db *ManagedDB) PurgeVersionsBelow(key []byte, ts uint64) error {
	txn := db.NewTransactionAt(ts, false)
	defer txn.Discard()
	return db.purgeVersionsBelow(txn, key, ts)
}

// GetSequence is not supported on ManagedDB. Calling this would result
// in a panic.
func (db *ManagedDB) GetSequence(_ []byte, _ uint64) (*Sequence, error) {
	panic("Cannot use GetSequence for ManagedDB.")
}
