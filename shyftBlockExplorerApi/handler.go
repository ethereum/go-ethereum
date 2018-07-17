package main

///@NOTE Shyft handler functions when endpoints are hit
import (
	"database/sql"
	"fmt"
	"net/http"

	_ "github.com/lib/pq"

	"github.com/empyrean/go-ethereum/shyftdb"
	"github.com/gorilla/mux"
	"bytes"
	"io/ioutil"
	"encoding/json"
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

// GetAllTransactions gets txs
func GetAllTransactionsFromBlock(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	blockNumber := vars["blockNumber"]
	connStr := "user=postgres dbname=shyftdb sslmode=disable"
	blockExplorerDb, err := sql.Open("postgres", connStr)
	if err != nil {
		return
	}

	txsFromBlock := shyftdb.GetAllTransactionsFromBlock(blockExplorerDb, blockNumber)

	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)

	fmt.Fprintln(w, txsFromBlock)
}

func GetAllBlocksMinedByAddress(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	coinbase := vars["coinbase"]
	connStr := "user=postgres dbname=shyftdb sslmode=disable"
	blockExplorerDb, err := sql.Open("postgres", connStr)
	if err != nil {
		return
	}

	blocksMined := shyftdb.GetAllBlocksMinedByAddress(blockExplorerDb, coinbase)

	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)

	fmt.Fprintln(w, blocksMined)
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

func GetRecentBlock(w http.ResponseWriter, r *http.Request) {
	connStr := "user=postgres dbname=shyftdb sslmode=disable"
	blockExplorerDb, err := sql.Open("postgres", connStr)
	if err != nil {
		return
	}
	mostRecentBlock := shyftdb.GetRecentBlock(blockExplorerDb)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, mostRecentBlock)

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

func BroadcastTx(w http.ResponseWriter, r *http.Request) {
	// Example return result (returns tx hash):
	// {"jsonrpc":"2.0","id":1,"result":"0xafa4c62f29dbf16bbfac4eea7cbd001a9aa95c59974043a17f863172f8208029"}

	// http params
	vars := mux.Vars(r)
	transactionHash := vars["transaction_hash"]

	// format the transactionHash into a proper sendRawTransaction jsonrpc request
	formatted_json := []byte(fmt.Sprintf(`{"jsonrpc":"2.0","method":"eth_sendRawTransaction","params":["%s"],"id":0}`, transactionHash))

	// send json rpc request
	resp, _ := http.Post("http://localhost:8545", "application/json", bytes.NewBuffer(formatted_json))
	body, _ := ioutil.ReadAll(resp.Body)
	byt := []byte(string(body))

	// read json and return result as http response, be it an error or tx hash
	var dat map[string]interface{}
	if err := json.Unmarshal(byt, &dat); err != nil {
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "ERROR parsing json")
	}
	tx_hash := dat["result"]
	if(tx_hash == nil) {
		errMap := dat["error"].(map[string]interface{})
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "ERROR:", errMap["message"])
	} else {
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "Transaction Hash:", tx_hash)
	}
}