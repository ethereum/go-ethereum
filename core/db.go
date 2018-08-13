package core

import (
	"database/sql"
	"fmt"
	"os"
)

var blockExplorerDb *sql.DB

// @NOTE: SHYFT - could be refactored to add a test db environment
const (
	connStrTest = "user=postgres dbname=shyftdbtest password=docker sslmode=disable"
)

var connStr = connectionStr()

// InitDB - initalizes a Postgresql Database for use by the Blockexplorer
func InitDB() (*sql.DB, error) {
	// To set the environment you can run the program with an ENV variable DBENV.
	// DBENV defaults to local for purposes of running the correct local
	// database connection parameters but will use docker connection parameters if DBENV=docker
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

func connectionStr() string {
	dbEnv := os.Getenv("DBENV")
	switch dbEnv {
	default:
		return "user=postgres dbname=shyftdb host=localhost sslmode=disable"
	case "docker":
		return "user=postgres dbname=shyftdb host=pg password=docker sslmode=disable"
	}
}

func InitDBTest() (*sql.DB, error) {
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
