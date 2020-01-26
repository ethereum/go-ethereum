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

package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/log"

	// here we are adding multiple default supported db drivers
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

// DBStorage is a storage type which is backed by a general purpose database
type DBStorage struct {
	driverName     string
	dataSourceName string
	tableName      string
	db             *sql.DB
	key            []byte
}

// DBRow is the structure to hold a row of our configuration database
// table schemas for all three tables (kps, js, config) are the same
type DBRow struct {
	id  int
	key string
	val string
}

// Default table name for storages
const (
	PasswordTable = "kps"
	ConfigTable   = "config"
	JsTable       = "js"
)

// NewDBStorage create new database backed storage
func NewDBStorage(key []byte, driverName, dataSourceName, tableName string) (*DBStorage, error) {
	// sql.Open only validates the input, but didn't create a connection
	db, err := sql.Open(driverName, dataSourceName)
	if err != nil {
		log.Error("failed to validate driver: #{driverName}, #{dataSourceName}")
		db.Close()
		return nil, err
	}

	// Connects to the database and make sure it is ok, connection will be closed shortly since default MaxIdle is 0
	err = db.Ping()
	if err != nil {
		log.Error("failed to connect to database: #{dataSourceName}")
		db.Close()
		return nil, err
	}

	// set connection limits
	db.SetMaxOpenConns(5)

	// init table
	initTable(driverName, tableName, db)

	return &DBStorage{
		driverName:     driverName,
		dataSourceName: dataSourceName,
		tableName:      tableName,
		db:             db,
		key:            key,
	}, nil
}

func initTable(driverName, tableName string, db *sql.DB) error {
	var err error
	switch driverName {
	case "postgres":
		_, err = db.Exec(fmt.Sprintf(`
CREATE TABLE IF NOT EXISTS %s (
	id SERIAL PRIMARY KEY,
	k VARCHAR(255) UNIQUE NOT NULL,
	v TEXT NOT NULL
)
		`, tableName))
	case "mysql":
		_, err = db.Exec(fmt.Sprintf(`
CREATE TABLE IF NOT EXISTS %s (
	id INT AUTO_INCREMENT PRIMARY KEY,
	k VARCHAR(255) UNIQUE NOT NULL, 
	v TEXT NOT NULL
)
		`, tableName))
	case "sqlite3":
		_, err = db.Exec(fmt.Sprintf(`
CREATE TABLE IF NOT EXISTS %s (
	id INTEGER PRIMARY KEY, 
	k TEXT, 
	v TEXT
)
		`, tableName))
	}

	return err
}

// Put stores a value by key. 0-length keys results in noop.
func (s *DBStorage) Put(key, value string) {
	if len(key) == 0 {
		return
	}
	ciphertext, iv, err := Encrypt(s.key, []byte(value), []byte(key))
	if err != nil {
		log.Warn("Failed to encrypt entry", "err", err)
		return
	}

	creds := StoredCredential{Iv: iv, CipherText: ciphertext}
	sql := s.formatSQL(getSQL)
	_, exist, err := s.queryRow(sql, key)
	if err != nil {
		log.Warn("Failed to execute SQL", "err", err)
		return
	}

	raw, err := json.Marshal(creds)
	if err != nil {
		log.Warn("Failed to marshal StoredCredential data")
		return
	}

	if !exist {
		sql = s.formatSQL(insertSQL)
		s.exec(sql, key, raw)
	} else {
		sql = s.formatSQL(updateSQL)
		s.exec(sql, raw, key)
	}
}

// Get returns the previously stored value, or an error if it does not exist or
// key is of 0-length.
func (s *DBStorage) Get(key string) (string, error) {
	sql := s.formatSQL(getSQL)
	row, exist, err := s.queryRow(sql, key)
	if err != nil {
		log.Warn("Failed to execute SQL", "err", err)
		return "", err
	}
	if !exist {
		log.Warn("Key does not exist", "key", key)
		return "", ErrNotFound
	}

	cred := StoredCredential{}
	if err = json.Unmarshal([]byte(row.val), &cred); err != nil {
		log.Warn("Failed to unmarshall stored json", "err", err)
		return "", err
	}

	entry, err := Decrypt(s.key, cred.Iv, cred.CipherText, []byte(key))
	if err != nil {
		log.Warn("Failed to decrypt key", "key", key)
		return "", err
	}

	return string(entry), nil
}

// Del removes a key-value pair. If the key doesn't exist, the method is a noop.
func (s *DBStorage) Del(key string) {
	sql := s.formatSQL(deleteSQL)
	s.exec(sql, key)
}

func (s *DBStorage) exec(query string, args ...interface{}) {
	_, err := s.db.Exec(query, args...)
	if err != nil {
		log.Warn("Failed to execute sql", query, args)
	}
}

func (s *DBStorage) queryRow(query string, args ...interface{}) (*DBRow, bool, error) {
	row := DBRow{}
	err := s.db.QueryRow(query, args...).Scan(&row.id, &row.key, &row.val)
	if err != nil && err != sql.ErrNoRows {
		return nil, false, err
	}

	if row.id == 0 {
		return nil, false, nil
	}
	return &row, true, nil
}

var (
	getSQL    string = "SELECT * FROM tableName WHERE k = ?"
	updateSQL string = "UPDATE tableName SET v = ? WHERE k = ?"
	insertSQL string = "INSERT INTO tableName (k, v) VALUES (?, ?)"
	deleteSQL string = "DELETE FROM tableName WHERE k = ?"
)

func (s *DBStorage) formatSQL(sql string) string {
	switch s.driverName {
	case "postgres":
		params := strings.Count(sql, "?")
		for i := 1; i <= params; i++ {
			sql = strings.Replace(sql, "?", fmt.Sprintf("$%d", i), 1)
		}
	default:
		// for MS SQL Server / MySQL / SQLite
		// since they're already using ? as placeholder, do nothing
	}

	return strings.ReplaceAll(sql, "tableName", s.tableName)
}

// Close sql.DB
func (s *DBStorage) Close() {
	s.db.Close()
}
