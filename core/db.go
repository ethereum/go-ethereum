package core

import (
  	"fmt"
  	"database/sql"
)

var blockExplorerDb *sql.DB

const (
	connStr = "user=postgres dbname=shyftdb sslmode=disable"
	connStrTest =  "user=postgres dbname=shyftdbtest sslmode=disable"
)

func InitDB() (*sql.DB, error){
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		fmt.Println("ERROR OPENING DB, NOT INITIALIZING")
		panic(err)
		return nil, err
	} else {
		blockExplorerDb = db
		return blockExplorerDb, nil
	}
}

func InitDBTest() (*sql.DB, error){
	db, err := sql.Open("postgres", connStrTest)
	if err != nil {
		fmt.Println("ERROR OPENING DB, NOT INITIALIZING")
		panic(err)
		return nil, err
	} else {
		blockExplorerDb = db
		return blockExplorerDb, nil
	}
}

func DBConnection() (*sql.DB, error) {
	if blockExplorerDb == nil {
		_, err := InitDB()
		if err != nil {
			return nil, err
		}
	}
	return blockExplorerDb, nil
}

func ClearTables() {
	sqldb, err := DBConnection()
	if err != nil {
		panic(err)
	}

	sqlStatementTx:= `DELETE FROM txs`
	_, err = sqldb.Exec(sqlStatementTx)
	if err != nil {
		panic(err)
	}

	sqlStatementAcc:= `DELETE FROM accounts`
	_, err = sqldb.Exec(sqlStatementAcc)
	if err != nil {
		panic(err)
	}

	sqlStatement := `DELETE FROM blocks`
	_, err = sqldb.Exec(sqlStatement)
	if err != nil {
		panic(err)
	}
}
