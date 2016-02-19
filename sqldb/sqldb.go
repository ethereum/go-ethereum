// Copyright 2015 The chatty Developers
// This file is part of the chatty library.
//
// The chatty library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The chatty library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package sqldb

import (
  "os"
  "errors"
  "strings"
  "regexp"
  "database/sql"
  "sync"
  "fmt"

	"github.com/chattynet/chatty/logger"
	"github.com/chattynet/chatty/logger/glog"
  "github.com/chattynet/chatty/core"
  "github.com/chattynet/chatty/core/types"
  _ "github.com/mattn/go-sqlite3"
)

var db_version uint64 = 1

type SQLDB struct {
	fn string      // filename for reporting
	db *sql.DB // sqlite3 instance

	quitLock sync.Mutex      // Mutex protecting the quit channel access
	quitChan chan chan error // Quit channel to stop the metrics collection before closing the database
}

type SQL_Transaction struct {
  Hash          string
  BlockNumber   uint64
}

type SQL_Transactions []*SQL_Transaction

func NewTransaction(hash string, blocknumber uint64) *SQL_Transaction {
	return &SQL_Transaction{Hash: hash, BlockNumber: blocknumber}
}

func (self *SQL_Transaction) MarshalJSON() ([]byte, error) {
  result := fmt.Sprintf(`{ "hash": "%s", "blockNumber": "0x%x" }`, self.Hash, self.BlockNumber);
  return []byte(result), nil
}

func checkExists(db *sql.DB, query string) (bool, error) {
  rows, err := db.Query(query)
  if err != nil {
    glog.V(logger.Error).Infoln("Error checking existence", err, query)
    return false, err
  }
  defer rows.Close()

  return rows.Next(), nil;
}

func getLastBlockNumber(db *sql.DB) (uint64, error) {
  query := `SELECT number FROM chatty_blocks ORDER BY number DESC LIMIT 1`;
  rows, err := db.Query(query)
  if err != nil {
    glog.V(logger.Error).Infoln("Error getting last block", err, query)
    return 0, err
  }
  defer rows.Close()

  if rows.Next() {
    var lb uint64
    rows.Scan(&lb)
    return lb, nil
  }

  return 0, nil
}

func checkTables(db *sql.DB) (bool, error) {
  return checkExists(db, `SELECT name FROM sqlite_master WHERE type='table' AND name='chatty_status'`)
}

func checkVersion(db *sql.DB) (bool, error) {
  query := `SELECT version FROM chatty_status ORDER BY created DESC LIMIT 1`;
  rows, err := db.Query(query)
  if err != nil {
    glog.V(logger.Error).Infoln("Error checking version", err, query)
    return false, err
  }
  defer rows.Close()

  if rows.Next() {
    var v uint64
    rows.Scan(&v)
    return (v == db_version), nil
  }

  return false, nil
}

func Init(file string) (*sql.DB, error) {
  // Open the db
  db, err := sql.Open("sqlite3", file)
  // (Re)check for errors and abort if opening of the db failed
  if err != nil {
    return nil, err
  }

  var cv bool = false
  var ct bool = false
  ct, err = checkTables(db)
  if err != nil {
    return nil, err
  }

  if ct {
    cv, err = checkVersion(db)
    if err != nil {
      return nil, err
    }
  }

  // tables exist with version mismatch, drop and recreate
  if ct && !cv {
    glog.V(logger.Info).Infoln("Dropping old SQL DB", file)
    db.Close()
    os.Remove(file)

    // Open the db
    db, err = sql.Open("sqlite3", file)
    // (Re)check for errors and abort if opening of the db failed
    if err != nil {
      return nil, err
    }
    ct = false
  }

  if !ct {
    glog.V(logger.Info).Infoln("Creating new SQL DB", file)
    // create the tables if they doesn't exist
    sqlStmt := `
      CREATE TABLE chatty_status (
        created TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
        version INT NOT NULL
      );
      INSERT INTO chatty_status(version)
      VALUES (?);

      CREATE TABLE chatty_blocks (
        number UNSIGNED BIG INT NOT NULL PRIMARY KEY,
        hash TEXT
      );

      CREATE TABLE chatty_transactions (
        hash TEXT NOT NULL PRIMARY KEY,
        blocknumber UNSIGNED BIG INT NOT NULL,
        sender TEXT NOT NULL,
        receiver TEXT NOT NULL
      );
    `

    _, err = db.Exec(sqlStmt, db_version)
    if err != nil {
      glog.V(logger.Error).Infoln("Error creating SQL tables", err, sqlStmt)
      return nil, err
    }
  } else {
    glog.V(logger.Info).Infoln("Loading existing SQL DB", file)
  }

  return db, nil;
}

// NewSQLiteDatabase returns a sqlite3 wrapped object. sqlite3 does not persist data by
// it self but requires a background poller which syncs every X. `Flush` should be called
// when data needs to be stored and written to disk.
func NewSQLiteDatabase(file string) (*SQLDB, error) {
  db, err :=  Init(file)

  if err != nil {
    return nil, err
  }

	return &SQLDB{
		fn: file,
		db: db,
	}, nil
}

func (self *SQLDB) Refresh(chainManager *core.ChainManager) {
  fromBlock, err := getLastBlockNumber(self.db)
  if err != nil {
    glog.V(logger.Error).Infoln("Error fetching last SQL block number", err)
    return
  }

  toBlock := chainManager.CurrentBlock().Number().Uint64()

  if fromBlock >= toBlock {
    // sanity check TODO: redo the whole SQL DB in this case!
    if fromBlock > toBlock {
      glog.V(logger.Error).Infoln("SQL DB ahead of chain, recreating")
      self.Close()
      os.Remove(self.fn)
      self.db, err = Init(self.fn)
      if err != nil {
        glog.V(logger.Error).Infoln("SQL DB Reinit:", err)
        return
      }
    } else {
      return
    }
  }

  tx, err := self.db.Begin()
  if err != nil {
    glog.V(logger.Error).Infoln("SQL DB Begin:", err)
    return
	}

  stmtBlock, err := tx.Prepare(`insert or replace into chatty_blocks(number, hash) values(?, ?)`)
  if err != nil {
    glog.V(logger.Error).Infoln("SQL DB:", err)
    return
	}
	defer stmtBlock.Close()

  stmtTrans, err := tx.Prepare(`insert or replace into chatty_transactions(hash, blocknumber, sender, receiver) values(?, ?, ?, ?)`)
  if err != nil {
    glog.V(logger.Error).Infoln("SQL DB:", err)
    return
	}
  defer stmtTrans.Close()

  glog.V(logger.Info).Infoln("SQL DB refreshing between blocks:", fromBlock, toBlock)
  for i := fromBlock + 1; i <= toBlock; i++ {
    block := chainManager.GetBlockByNumber(i)
    // block
    _, err = stmtBlock.Exec(i, block.Hash().Hex())
		if err != nil {
      glog.V(logger.Error).Infoln("SQL DB:", err)
      tx.Rollback()
      return
		}
    // transactions

    for _, trans := range block.Transactions() {
      sender, err := trans.From()
      if err != nil {
        glog.V(logger.Error).Infoln("SQL DB:", err)
        continue
      }
      senderHex := sender.Hex()
      receiver := trans.To()
      receiverHex := ""
      if ( receiver != nil ) {
        receiverHex = receiver.Hex()
      }
      _, err = stmtTrans.Exec(trans.Hash().Hex(), i, senderHex, receiverHex)
  		if err != nil {
        glog.V(logger.Error).Infoln("SQL DB:", err)
        tx.Rollback()
        return
  		}
    }
  }
  tx.Commit()
}

func (self *SQLDB) InsertBlock(block *types.Block) {
  tx, err := self.db.Begin()
  if err != nil {
    glog.V(logger.Error).Infoln("SQL DB Begin:", err)
    return
  }

  stmtBlock, err := tx.Prepare(`insert or replace into chatty_blocks(number, hash) values(?, ?)`)
  if err != nil {
    glog.V(logger.Error).Infoln("SQL DB:", err)
    return
  }
  defer stmtBlock.Close()

  stmtTrans, err := tx.Prepare(`insert or replace into chatty_transactions(hash, blocknumber, sender, receiver) values(?, ?, ?, ?)`)
  if err != nil {
    glog.V(logger.Error).Infoln("SQL DB:", err)
    return
  }
  defer stmtTrans.Close()

  // block
  _, err = stmtBlock.Exec(block.Number().Uint64(), block.Hash().Hex())
  if err != nil {
    glog.V(logger.Error).Infoln("SQL DB:", err)
    tx.Rollback()
    return
  }
  // transactions

  for _, trans := range block.Transactions() {
    sender, err := trans.From()
    if err != nil {
      glog.V(logger.Error).Infoln("SQL DB:", err)
      continue
    }
    senderHex := sender.Hex()
    receiver := trans.To()
    receiverHex := ""
    if ( receiver != nil ) {
      receiverHex = receiver.Hex()
    }

    _, err = stmtTrans.Exec(trans.Hash().Hex(), block.Number().Uint64(), senderHex, receiverHex)
    if err != nil {
      glog.V(logger.Error).Infoln("SQL DB:", err)
      tx.Rollback()
      return
    }
  }

  tx.Commit()
}

func (self *SQLDB) DeleteBlock(block *types.Block) {
  query := `DELETE FROM blocks WHERE number = ?`
  _, err := self.db.Exec(query, block.Number().Uint64())
  if err != nil {
    glog.V(logger.Error).Infoln("Error creating SQL tables", err, query)
  }
}

func (self *SQLDB) Close() {
	// Stop the metrics collection to avoid internal database races
	self.quitLock.Lock()
	defer self.quitLock.Unlock()

	if self.quitChan != nil {
		errc := make(chan error)
		self.quitChan <- errc
		if err := <-errc; err != nil {
			glog.V(logger.Error).Infof("metrics failure in '%s': %v\n", self.fn, err)
		}
	}
	// Commit and close the database
	/*if err := self.Commit(); err != nil {
		glog.V(logger.Error).Infof("commit '%s' failed: %v\n", self.fn, err)
	}*/

	self.db.Close()
	glog.V(logger.Error).Infoln("Closed SQL DB:", self.fn)
}

func (self *SQLDB) SelectTransactionsForAccounts(accounts []string) (trans SQL_Transactions, err error) {
  if len(accounts) <= 0 {
    return nil, errors.New("Input accounts required")
  }

  // regexp check for SQL safety
  for _, acct := range accounts {
  	matched, err := regexp.MatchString("^0x[0-9,a-f]{40}$", acct)
    if err != nil {
      return nil, err
    }
    if !matched {
      return nil, errors.New("Input account error")
    }
  }

  acctSQL := strings.Join(accounts, "','")
  query := `
    SELECT hash, blocknumber
    FROM chatty_transactions
    WHERE sender IN ('`
  query += acctSQL
  query += `')
    OR receiver IN ('`
  query += acctSQL
  query += `')`

  rows, err := self.db.Query(query)
  if err != nil {
    return nil, err
  }

  for rows.Next() {
    var hash string
    var bn uint64
    rows.Scan(&hash, &bn)
    trans = append(trans, NewTransaction(hash, bn))
  }
  rows.Close()

  return trans, nil
}

func (self *SQLDB) DB() *sql.DB {
	return self.db
}
