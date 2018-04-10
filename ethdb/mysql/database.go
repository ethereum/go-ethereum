// Copyright 2018 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package mysql

import (
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
)

var (
	ethDB         *gorm.DB // share a MySQL instance to avoid too many connections
	instanceCount int32
)

type database struct {
	db        *gorm.DB
	tableName string
}

type data struct {
	Key   []byte `gorm:"unique_index"`
	Value []byte `gorm:"type:BLOB"`
}

// NewDatabase returns a MySQL wrapped object.
func NewDatabase(name string, cfg *Config) (ethdb.Database, error) {
	if atomic.CompareAndSwapInt32(&instanceCount, 0, 1) {
		// Open db
		connectionString := cfg.String()
		db, err := gorm.Open("mysql", connectionString)
		if err != nil {
			return nil, err
		}
		// Hide the db log
		db.LogMode(false)

		ethDB = db
	} else {
		atomic.AddInt32(&instanceCount, 1)
	}

	if !ethDB.HasTable(name) {
		if err := ethDB.Table(name).CreateTable(&data{}).Error; err != nil {
			if err != nil {
				return nil, err
			}
		}
	}

	return &database{
		db:        ethDB.Table(name),
		tableName: name,
	}, nil
}

// Put puts the given key / value to the queue
func (db *database) Put(key []byte, value []byte) error {
	err := putOnMySQL(db.db, db.tableName, key, value)
	return err
}

func (db *database) Has(key []byte) (bool, error) {
	if err := db.db.Where(&data{
		Key: key,
	}).First(&data{}).Error; gorm.IsRecordNotFoundError(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return true, nil
}

// Get returns the given key if it's present.
func (db *database) Get(key []byte) ([]byte, error) {
	out := &data{}
	err := db.db.Where(&data{
		Key: key,
	}).First(out).Error
	if err != nil {
		return nil, err
	}
	return out.Value, nil
}

// Delete deletes the key from the queue and database
func (db *database) Delete(key []byte) error {
	err := db.db.Where(&data{
		Key: key,
	}).Delete(&data{}).Error
	// Hide not found error
	if gorm.IsRecordNotFoundError(err) {
		return nil
	}
	return err
}

func (db *database) Close() {
	if atomic.CompareAndSwapInt32(&instanceCount, 1, 0) {
		ethDB.Close()
		ethDB = nil
	} else {
		atomic.AddInt32(&instanceCount, -1)
	}
}

// NewBatch create a db transaction to batch insert
func (db *database) NewBatch() ethdb.Batch {
	return &batch{
		database:    db,
		transaction: db.db.Begin(),
	}
}

type batch struct {
	*database
	transaction *gorm.DB
	size        int
	finished    bool
}

func (b *batch) Put(key, value []byte) (err error) {
	defer func() {
		// Update size if success, or rollback it
		if err == nil {
			b.size += len(value)
		} else if !b.finished {
			b.finished = true
			b.transaction.Rollback()
		}
	}()
	return putOnMySQL(b.transaction, b.tableName, key, value)
}

func (b *batch) Write() (err error) {
	// This transaction is finished before. There is no data so ignore it.
	if b.finished {
		return nil
	}
	b.finished = true
	return b.transaction.Commit().Error
}

func (b *batch) ValueSize() int {
	return b.size
}

func (b *batch) Reset() {
	// Rollback previous transaction
	if !b.finished {
		b.transaction.Rollback()
	}
	b.transaction = b.db.Begin()
	b.size = 0
	b.finished = false
}

// putOnMySQL replaces the record if exists, or insert a new one
func putOnMySQL(db *gorm.DB, tableName string, key []byte, value []byte) (err error) {
	for count := 0; ; {
		err := db.Exec(fmt.Sprintf("REPLACE into %s VALUES(?, ?)", tableName), key, value).Error
		if err == nil {
			return nil
		}

		// Retry if it's transaction deadlock error.
		// https://dev.mysql.com/doc/refman/5.7/en/innodb-deadlock-example.html
		if strings.Contains(err.Error(), "Error 1213") {
			if count == 10 {
				return err
			}
			count++
			log.Debug("Failed to put due to transaction deadlock error", "table", tableName, "count", count, "err", err)
			time.Sleep(time.Duration(count*500) * time.Millisecond)
		} else {
			return err
		}
	}
}
