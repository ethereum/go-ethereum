package shyftdb

import (
	"database/sql"
	"fmt"
)

func InitTestDB() *sql.DB {
	connStr := "user=postgres dbname=shyftdbtest sslmode=disable"
	blockExplorerDbTest, err := sql.Open("postgres", connStr)
	if err != nil {
		fmt.Println(err)
	}

	return blockExplorerDbTest
}

func ClearTables() {
	connStr := "user=postgres dbname=shyftdbtest sslmode=disable"
	blockExplorerDbTest, err := sql.Open("postgres", connStr)
	if err != nil {
		fmt.Println(err)
	}

	sqlStatementTx:= `DELETE FROM txs`
	_, err = blockExplorerDbTest.Exec(sqlStatementTx)
	if err != nil {
		panic(err)
	}

	sqlStatementAcc:= `DELETE FROM accounts`
	_, err = blockExplorerDbTest.Exec(sqlStatementAcc)
	if err != nil {
		panic(err)
	}

	sqlStatement := `DELETE FROM blocks`
	_, err = blockExplorerDbTest.Exec(sqlStatement)
	if err != nil {
		panic(err)
	}
}