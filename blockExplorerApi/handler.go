package main

///@NOTE Shyft handler functions when endpoints are hit
import (
	"database/sql"
	"fmt"
	"net/http"

	_ "github.com/lib/pq"

	"github.com/ethereum/go-ethereum/shyftdb"
	"github.com/gorilla/mux"
)

// GetTransaction gets txs
func GetTransaction(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	txHash := vars["txHash"]
	connStr := "user=postgres dbname=shyftdb sslmode=disable"
	blockExplorerDb, err := sql.Open("postgres", connStr)
	if err != nil {
		return
	}

	getTxResponse := shyftdb.GetTransaction(blockExplorerDb, txHash)

	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)

	fmt.Fprintln(w, getTxResponse)
}

// GetAllTransactions gets txs
func GetAllTransactions(w http.ResponseWriter, r *http.Request) {
	connStr := "user=postgres dbname=shyftdb sslmode=disable"
	blockExplorerDb, err := sql.Open("postgres", connStr)
	if err != nil {
		return
	}

	txs := shyftdb.GetAllTransactions(blockExplorerDb)

	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}


	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)

	fmt.Fprintln(w, txs)
}

// GetAccount gets balance
func GetAccount(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	address := vars["address"]
	connStr := "user=postgres dbname=shyftdb sslmode=disable"
	blockExplorerDb, err := sql.Open("postgres", connStr)
	if err != nil {
		return
	}

	getAccountBalance := shyftdb.GetAccount(blockExplorerDb, address)

	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)

	fmt.Fprintln(w, getAccountBalance)
}

// GetAccount gets balance
func GetAccountTxs(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	address := vars["address"]
	connStr := "user=postgres dbname=shyftdb sslmode=disable"
	blockExplorerDb, err := sql.Open("postgres", connStr)
	if err != nil {
		return
	}

	getAccountTxs := shyftdb.GetAccountTxs(blockExplorerDb, address)

	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)

	fmt.Fprintln(w, getAccountTxs)
}

// GetAllAccounts gets balances
func GetAllAccounts(w http.ResponseWriter, r *http.Request) {
	connStr := "user=postgres dbname=shyftdb sslmode=disable"
	blockExplorerDb, err := sql.Open("postgres", connStr)
	if err != nil {
		return
	}
	allAccounts := shyftdb.GetAllAccounts(blockExplorerDb)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)

	fmt.Fprintln(w, allAccounts)
}

//GetBlock returns block json
func GetBlock(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	blockNumber := vars["blockNumber"]
	connStr := "user=postgres dbname=shyftdb sslmode=disable"
	blockExplorerDb, err := sql.Open("postgres", connStr)
	if err != nil {
		return
	}

	getBlockResponse := shyftdb.GetBlock(blockExplorerDb, blockNumber)

	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)

	fmt.Fprintln(w, getBlockResponse)
}

// GetAllBlocks response
func GetAllBlocks(w http.ResponseWriter, r *http.Request) {
	connStr := "user=postgres dbname=shyftdb sslmode=disable"
	blockExplorerDb, err := sql.Open("postgres", connStr)
	if err != nil {
		return
	}
	block3 := shyftdb.GetAllBlocks(blockExplorerDb)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, block3)
}

//func GetRecentBlock(w http.ResponseWriter, r *http.Request) {
//	connStr := "user=postgres dbname=shyftdb sslmode=disable"
//	blockExplorerDb, err := sql.Open("postgres", connStr)
//	if err != nil {
//		return
//	}
//
//
//}

//GetInternalTransactions gets internal txs
func GetInternalTransactions(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	address := vars["address"]

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)

	fmt.Fprintln(w, "Get InternalTransactions", address)
}

//GetInternalTransactionsHash gets internal txs hash
func GetInternalTransactionsHash(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	transactionHash := vars["transaction_hash"]

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)

	fmt.Fprintln(w, "Get Internal Transaction Hash", transactionHash)
}
