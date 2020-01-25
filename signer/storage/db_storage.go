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

	"github.com/ethereum/go-ethereum/log"

	// here we are adding multiple default supported db drivers
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

// DBStorage is a storage type which is backed by a general purpose database
type DBStorage struct {
	driverName     string
	dataSourceName string
	db             *sql.DB
	key            []byte
}

// KPS is the structure to hold a row of our credentials database
type KPS struct {
	id      int
	address string
	json    string
}

// NewDBStorage create new database backed storage
func NewDBStorage(key []byte, driverName, dataSourceName string) (*DBStorage, error) {
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

	return &DBStorage{
		driverName:     driverName,
		dataSourceName: dataSourceName,
		db:             db,
		key:            key,
	}, nil
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
	_, exist, err := s.queryRow("SELECT * FROM kps WHERE address = ?", key)
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
		s.exec("INSERT INTO kps (address, json) VALUES (?, ?)", key, raw)
	} else {
		s.exec("UPDATE kps SET json = ? WHERE address = ?", raw, key)
	}
}

// Get returns the previously stored value, or an error if it does not exist or
// key is of 0-length.
func (s *DBStorage) Get(key string) (string, error) {
	kps, exist, err := s.queryRow("SELECT * FROM kps WHERE address = ?", key)
	if err != nil {
		log.Warn("Failed to execute SQL", "err", err)
		return "", err
	}
	if !exist {
		log.Warn("Key does not exist", "key", key)
		return "", ErrNotFound
	}

	cred := StoredCredential{}
	if err = json.Unmarshal([]byte(kps.json), &cred); err != nil {
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
	s.exec("DELETE FROM kps WHERE address = ?", key)
}

func (s *DBStorage) exec(query string, args ...interface{}) {
	_, err := s.db.Exec(query, args...)
	if err != nil {
		log.Warn("Failed to execute sql", query, args)
	}
}

func (s *DBStorage) queryRow(query string, args ...interface{}) (*KPS, bool, error) {
	kps := KPS{}
	err := s.db.QueryRow(query, args...).Scan(&kps.id, &kps.address, &kps.json)
	if err != nil && err != sql.ErrNoRows {
		return nil, false, err
	}

	if kps.id == 0 {
		return nil, false, nil
	}
	return &kps, true, nil
}

// Close sql.DB
func (s *DBStorage) Close() {
	s.db.Close()
}

func main() {
	// "root:900406.mysql@tcp(localhost:3306)/adv_database"
	db, err := sql.Open("mysql", "server=localhost;user id=root;password=900406.mysql;port=3306;database=adv_database")
	if err != nil {
		fmt.Println("failure")
		panic(err)
	}
	fmt.Println("success")
	defer db.Close()
}
