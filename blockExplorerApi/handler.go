package main

///@NOTE Shyft handler functions when endpoints are hit
import (
	"database/sql"
	"fmt"
	logger "log"
	"net/http"

	_ "github.com/lib/pq"

	shyftdb "github.com/ethereum/shyft_go-ethereum/shyftdb"
	"github.com/gorilla/mux"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
)

// GetTransaction gets txs
func GetTransaction(w http.ResponseWriter, r *http.Request) {
	// vars := mux.Vars(r)
	// address := vars["address"]

	//tx := shyftdb.GetTransaction()
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)

	//fmt.Fprintln(w, "Get All Transactions", addresses)
}

// GetAllTransactions gets txs
func GetAllTransactions(w http.ResponseWriter, r *http.Request) {
	//txs := shyftdb.GetAllTransactions()

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)

	//fmt.Fprintln(w, "Get All Transactions", address)
}

// GetBalance gets balance
func GetBalance(w http.ResponseWriter, r *http.Request) {
	// vars := mux.Vars(r)
	// address := vars["address"]
	//addressBytes := []byte(address)

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)

	//fmt.Fprintln(w, "Get Balances", addresses)
}

// GetBalances gets balances
func GetBalances(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)

	//fmt.Fprintln(w, "Get Balances", addresses)
}

//GetBlock returns block json
func GetBlock(w http.ResponseWriter, r *http.Request) {

	connStr := "user=postgres dbname=shyftdb sslmode=disable"
	blockExplorerDb, err := sql.Open("postgres", connStr)
	if err != nil {
		return
	}

	block3 := shyftdb.GetBlock(blockExplorerDb)

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)

	fmt.Fprintln(w, "block", block3)
}

// GetAllBlocks response
func GetAllBlocks(w http.ResponseWriter, r *http.Request) {
	blockExplorerDb, err := leveldb.OpenFile("./shyftData/geth/blockExplorerDb/", &opt.Options{
		ErrorIfMissing: true,
		ReadOnly:       true,
	})
	if err != nil {
		logger.Print(err)
		return
	}
	blocks := shyftdb.GetAllBlocks(blockExplorerDb)

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)

	fmt.Fprintln(w, "blocks", blocks)
}

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
