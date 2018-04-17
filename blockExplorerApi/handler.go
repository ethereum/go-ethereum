package main

///@NOTE Shyft handler functions when endpoints are hit
import (
	"encoding/json"
	"fmt"
	logger "log"
	"net/http"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/shyftdb"
	"github.com/gorilla/mux"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/util"
)

// GetAllTransactions gets txs
func GetAllTransactions(w http.ResponseWriter, r *http.Request) {
	db, err := leveldb.OpenFile(`../shyftData/geth/blockExplorerDb/`, nil)
	if err != nil {
		return
	}
	var data []byte
	iter := db.NewIterator(util.BytesPrefix([]byte("tx-")), nil)
	for iter.Next() {

		logger.Printf("%s\t%s", "\nALL TX VALUE: ", string(iter.Value()))
		data = append(iter.Value())
	}

	logger.Print("outside loop", data)
	iter.Release()
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(data)
}

// GetBalance gets balance
func GetBalance(w http.ResponseWriter, r *http.Request) {
	// vars := mux.Vars(r)
	// address := vars["address"]
	address := "0x43ec6d0942f7faef069f7f63d0384a27f529b062"
	addressBytes := []byte(address)

	db, err := leveldb.OpenFile(`../shyftData/geth/blockExplorerDb/`, nil)
	if err != nil {
		return
	}

	// if err := db.Put(addressBytes, []byte("Golang BABY WOOOOOOOOO! BARBADOS"), nil); err != nil {
	// 	log.Crit("Failed to store TX", "err", err)
	// }

	data, err := db.Get(addressBytes, nil)
	if err != nil {
		log.Crit("Could not retrieve block", "err", err)
	}

	bodyString := string(data)

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(bodyString)
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
	logger.Print("Check logs")
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
