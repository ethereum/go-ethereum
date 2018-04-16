package main

///@NOTE Shyft handler functions when endpoints are hit
import (
	"encoding/json"
	"fmt"
	"net/http"
	logger "log"
	"github.com/gorilla/mux"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/ethereum/go-ethereum/shyftdb"
)

// GetAllTransactions gets txs
func GetAllTransactions(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	address := vars["address"]
	// addressBytes := []byte(address)

	blockExplorerDb, err := leveldb.OpenFile(`../shyftData/geth/blockExplorerDb/`, nil)
	if err != nil {
		return
	}

	data, err := blockExplorerDb.Get([]byte(address), nil)
	bodyString := string(data)

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(bodyString)
}

// GetBalance gets balance
func GetBalance(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	address := vars["address"]
	//addressBytes := []byte(address)

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(address)
}

// GetBalances gets balances
func GetBalances(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	addresses := vars["addresses"]

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)

	fmt.Fprintln(w, "Get Balances", addresses)
}

// GetBlocksMined get blocks mined
func GetBlocksMined(w http.ResponseWriter, r *http.Request) {
	blockExplorerDb, err := leveldb.OpenFile("./shyftData/geth/blockExplorerDb/", &opt.Options{
		ErrorIfMissing: true,
		ReadOnly: true,
	})
	if err != nil {
		logger.Print(err)
		return
	}
	blocks := shyftdb.GetAllBlocks(blockExplorerDb)
	
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)

	fmt.Fprintln(w, "Get Blocks Mined", blocks)
}

// GetTransactions gets txs
func GetTransactions(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	address := vars["address"]

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)

	fmt.Fprintln(w, "Get Transactions", address)
}

//GetInternalTransactions gets internal txs
func GetInternalTransactions(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	address := vars["address"]

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)

	fmt.Fprintln(w, "GetInternalTransactions", address)
}

//GetInternalTransactionsHash gets internal txs hash
func GetInternalTransactionsHash(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	transactionHash := vars["transaction_hash"]

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)

	fmt.Fprintln(w, "Get Internal Transaction Hash", transactionHash)
}
