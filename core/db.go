package core

import (
	"database/sql"
	"fmt"
)

var blockExplorerDb *sql.DB

// @NOTE:SHYFT - TODO: Move connection parameters into an env file
const (
	connStr = "user=postgres dbname=shyftdb host=pg password=docker sslmode=disable"
	// connStr     = "postgresql://postgres:docker@pg:5432/shyftdb"
	connStrTest = "user=postgres dbname=shyftdbtest password=docker sslmode=disable"
)

func InitDB() (*sql.DB, error) {
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
