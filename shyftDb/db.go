package shyftdb

import (
  	"fmt"
  	"database/sql"
)

var blockExplorerDb *sql.DB

func InitDB() (*sql.DB, error){
	connStr := "user=postgres dbname=shyftdb sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		fmt.Println("ERROR OPENING DB, NOT INITIALIZING")
		fmt.Println(err)
		return nil, err
	} else {
		blockExplorerDb = db
		return blockExplorerDb, nil
	}
}

func DBConnection() (*sql.DB, error) {
	if (blockExplorerDb == nil) {
		_, err := InitDB()
		if(err != nil) {
			return nil, err
		}
	}
	return blockExplorerDb, nil
}