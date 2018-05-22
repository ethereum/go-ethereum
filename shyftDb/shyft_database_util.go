package shyftdb

import (
	"encoding/json"
	"fmt"
	"math/big"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"time"
	"strconv"
	"database/sql"
	"log"

	_ "github.com/lib/pq"
	Rewards "github.com/ethereum/go-ethereum/consensus/ethash"
)

//SBlock type
type SBlock struct {
	Hash     string
	Coinbase string
	Number   string
	GasUsed	 string
	GasLimit string
	TxCount  string
	UncleCount string
	Age        string
	ParentHash string
	UncleHash string
	Difficulty string
	Size string
	Nonce string
}

//blockRes struct
type blockRes struct {
	hash     string
	coinbase string
	number   string
	Blocks   []SBlock
}

type SAccounts struct {
	Addr    string
	Balance string
	TxCountAccount string
}

type accountRes struct {
	addr        string
	balance     string
	AllAccounts []SAccounts
}

//ShyftTxEntry structure
type ShyftTxEntry struct {
	TxHash    common.Hash
	To        *common.Address
	From      *common.Address
	BlockHash string
	Amount    *big.Int
	GasPrice  *big.Int
	Gas       uint64
	Cost	  *big.Int
	Nonce     uint64
	Data      []byte
}

type txRes struct {
	TxEntry []ShyftTxEntryPretty
}

type ShyftTxEntryPretty struct {
	TxHash    string
	To        string
	From      string
	BlockHash string
	BlockNumber string
	Amount    uint64
	GasPrice  uint64
	Gas       uint64
	GasLimit  uint64
	Cost	  uint64
	Nonce     uint64
	Status	  string
	IsContract bool
	Age        time.Time
	Data      []byte
}

type ShyftAccountEntry struct {
	Balance string
	Txs     []string
}

type SendAndReceive struct {
	To        string
	From      string
	Amount    string
	Address   string
	Balance   string
	TxCountAccount string
}

//WriteBlock writes to block info to sql db
func WriteBlock(sqldb *sql.DB, block *types.Block, receipts []*types.Receipt) error {
	//Need to create field in postgres db isContract : True || False
	//Need to fix nonce numeric issue (attempt to reproduce and record)
	//Need to update tx To Field where null with Contract Address
	//Need to update account table with that Contract Address
	//Need to update AccountNonce and Balance of Tx To field || Contract Address
	WriteMinerRewards(sqldb,block)
	coinbase := block.Header().Coinbase.String()
	number := block.Header().Number.String()
	gasUsed := block.Header().GasUsed
	gasLimit := block.Header().GasLimit
	txCount := block.Transactions().Len()
	uncleCount := len(block.Uncles())
	parentHash := block.ParentHash().String()
	uncleHash := block.UncleHash().String()
	blockDifficulty := block.Difficulty().String()
	blockSize := block.Size().String()
	blockNonce := block.Nonce()

	i, err := strconv.ParseInt(block.Time().String(), 10, 64)
	if err != nil {
		panic(err)
	}
	age := time.Unix(i, 0)

	sqlStatement := `INSERT INTO blocks(hash, coinbase, number, gasUsed, gasLimit, txCount, uncleCount, age, parentHash, uncleHash, difficulty, size, nonce) VALUES(($1), ($2), ($3), ($4), ($5), ($6), ($7), ($8), ($9), ($10), ($11), ($12),($13)) RETURNING number`
	qerr := sqldb.QueryRow(sqlStatement, block.Header().Hash().Hex(), coinbase, number, gasUsed, gasLimit, txCount, uncleCount, age, parentHash, uncleHash, blockDifficulty, blockSize, blockNonce).Scan(&number)
	if qerr != nil {
		panic(qerr)
	}

	if block.Transactions().Len() > 0 {
		for _, tx := range block.Transactions() {
			//WriteMinerRewards(sqldb, block)
			WriteTransactions(sqldb, tx, block.Header().Hash(), block.Header().Number.String(), receipts, age, gasLimit)
			if block.Transactions()[0].To() != nil {
				WriteFromBalance(sqldb, tx)
			}
			if block.Transactions()[0].To() == nil {
				WriteContractBalance(sqldb, tx)
				WriteContractsTxHashReferences(sqldb, tx)
			}
		}
	}
	return nil
}

//WriteTransactions writes to sqldb
func WriteTransactions(sqldb *sql.DB, tx *types.Transaction, blockHash common.Hash, blockNumber string,  receipts []*types.Receipt, age time.Time, gasLimit uint64) error {
	txData := ShyftTxEntry{
		TxHash:    tx.Hash(),
		From:      tx.From(),
		To:        tx.To(),
		BlockHash: blockHash.Hex(),
		Amount:    tx.Value(),
		Cost:	   tx.Cost(),
		GasPrice:  tx.GasPrice(),
		Gas:       tx.Gas(),
		Nonce:     tx.Nonce(),
		Data:      tx.Data(),
	}
	txHash := txData.TxHash.Hex()
	from := txData.From.Hex()
	blockHasher := txData.BlockHash
	amount := txData.Amount.String()
	gasPrice := txData.GasPrice.String()
	txFee := txData.Cost.String()
	nonce := txData.Nonce
	gas := txData.Gas
	data := txData.Data
	to := txData.To
	var isContract bool
	var statusFromReciept string

	if (to == nil){
		var contractAddressFromReciept string
		for _, receipt := range receipts {
			statusReciept := (*types.ReceiptForStorage)(receipt).Status
			contractAddressFromReciept = (*types.ReceiptForStorage)(receipt).ContractAddress.String()
			if statusReciept == 1 {
				statusFromReciept = "SUCCESS"
			}
			if statusReciept == 0 {
				statusFromReciept = "FAIL"
			}
		}

		var retNonce string
		isContract = true
		sqlStatement := `INSERT INTO txs(txhash, from_addr, to_addr, blockhash, blockNumber, amount, gasprice, gas, gasLimit,txfee, nonce, isContract, txStatus, age, data) VALUES(($1), ($2), ($3), ($4), ($5), ($6), ($7), ($8), ($9), ($10), ($11), ($12), ($13), ($14), ($15)) RETURNING nonce`
		qerr := sqldb.QueryRow(sqlStatement, txHash, from, contractAddressFromReciept, blockHasher, blockNumber, amount, gasPrice, gas, gasLimit, txFee, nonce, isContract, statusFromReciept, age,data).Scan(&retNonce)

		if qerr != nil {
			panic(qerr)
		}
	} else {
		var retNonce string
		isContract = false
		for _, receipt := range receipts {
			statusReciept := (*types.ReceiptForStorage)(receipt).Status
			if statusReciept == 1 {
				statusFromReciept = "SUCCESS"
			}
			if statusReciept == 0 {
				statusFromReciept = "FAIL"
			}
		}

		sqlStatement := `INSERT INTO txs(txhash, from_addr, to_addr, blockhash, blockNumber, amount, gasprice, gas, gasLimit, txfee, nonce, isContract, txStatus, age, data) VALUES(($1), ($2), ($3), ($4), ($5), ($6), ($7), ($8), ($9), ($10), ($11), ($12), ($13), ($14), ($15)) RETURNING nonce`
		qerr := sqldb.QueryRow(sqlStatement, txHash, from, to.Hex(), blockHasher, blockNumber, amount, gasPrice, gas, gasLimit, txFee, nonce, isContract, statusFromReciept, age,data).Scan(&retNonce)

		if qerr != nil {
			panic(qerr)
		}
	}

	return nil
}

func WriteContractsTxHashReferences(sqldb *sql.DB, tx *types.Transaction) error {
	txHash := tx.Hash().Hex()

	sqlStatement := `INSERT INTO contracts(txHash) VALUES(($1)) RETURNING txHash`
	insertErr := sqldb.QueryRow(sqlStatement, txHash).Scan(&txHash)
	if insertErr != nil {
		panic(insertErr)
	}
	return nil
}

func WriteContractBalance(sqldb *sql.DB, tx *types.Transaction) error {
	sendAndReceiveData,balanceSen,accountNonceSen := WriteContractBalanceHelper(sqldb, tx)
	fromAddr := sendAndReceiveData.From
	amount := sendAndReceiveData.Amount
	balanceSender := balanceSen

	var response string
	sqlExistsStatement := `SELECT balance from accounts WHERE addr = ($1)`
	err := sqldb.QueryRow(sqlExistsStatement, fromAddr).Scan(&response)
	switch {
	case err == sql.ErrNoRows:
		sqlStatement := `INSERT INTO accounts(addr, balance, txCountAccount) VALUES(($1), ($2), ($3)) RETURNING addr`
		insertErr := sqldb.QueryRow(sqlStatement, fromAddr, amount, accountNonceSen).Scan(&fromAddr)
		if insertErr != nil {
			panic(insertErr)
		}
	case err != nil:
		log.Fatal(err)
	default:
		var newBalanceSender big.Int
		var newAccountNonceSender big.Int
		var nonceIncrement = big.NewInt(1)
		updateSQLStatement := `UPDATE accounts SET balance = ($2), txCountAccount = ($3) WHERE addr = ($1)`

		s := new(big.Int)
		_, error := fmt.Sscan(balanceSender, s)
		if error != nil {
			log.Println("error scanning value:", error)
		}

		accountS := new(big.Int)
		_, errors := fmt.Sscan(accountNonceSen, accountS)
		if errors != nil {
			log.Println("error scanning value:", error)
		}

		newBalanceSender.Sub(s, tx.Value())
		newAccountNonceSender.Add(accountS, nonceIncrement)

		_, err = sqldb.Exec(updateSQLStatement, fromAddr, newBalanceSender.String(), newAccountNonceSender.String())
		if err != nil {
			panic(err)
		}
	}
	return nil
}

func WriteContractBalanceHelper(sqldb *sql.DB, tx *types.Transaction) (SendAndReceive, string, string) {
	sendAndReceiveData := SendAndReceive{
		From: tx.From().Hex(),
		Amount: tx.Value().String(),
	}

	fromAddr := sendAndReceiveData.From
	getAccountBalanceSender:= GetAccount(sqldb, fromAddr)

	var senderBalance SendAndReceive
	if err := json.Unmarshal([]byte(getAccountBalanceSender), &senderBalance); err != nil {
		log.Fatal(err)
	}
	balanceSender := senderBalance.Balance
	accountNonceSender := senderBalance.TxCountAccount

	return sendAndReceiveData, balanceSender, accountNonceSender
}

//WriteFromBalance writes senders balance to accounts db
func WriteFromBalance(sqldb *sql.DB, tx *types.Transaction) error {
	sendAndReceiveData, balanceRec, balanceSen, accountNonceRec, accountNonceSen := WriteBalanceHelper(sqldb, tx)
	toAddr := sendAndReceiveData.To
	fromAddr := sendAndReceiveData.From
	amount := sendAndReceiveData.Amount
	balanceReceiver := balanceRec
	balanceSender := balanceSen

	var response string
	sqlExistsStatement := `SELECT balance from accounts WHERE addr = ($1)`
	err := sqldb.QueryRow(sqlExistsStatement, toAddr).Scan(&response)
	switch {
	case err == sql.ErrNoRows:
		i, err := strconv.Atoi(accountNonceRec)
		if err !=  nil {
			fmt.Println(err)
		}
		sqlStatement := `INSERT INTO accounts(addr, balance, txCountAccount) VALUES(($1), ($2), ($3)) RETURNING addr`
		insertErr := sqldb.QueryRow(sqlStatement, toAddr, amount, i).Scan(&toAddr)
		if insertErr != nil {
			panic(insertErr)
		}
	case err != nil:
		log.Fatal(err)
	default:
		var newBalanceReceiver big.Int
		var newBalanceSender big.Int
		var newAccountNonceReceiver big.Int
		var newAccountNonceSender big.Int
		var nonceIncrement = big.NewInt(1)
		updateSQLStatement := `UPDATE accounts SET balance = ($2), txCountAccount = ($3) WHERE addr = ($1)`

		r := new(big.Int)
		_, err := fmt.Sscan(balanceReceiver, r)
		if err != nil {
			log.Println("error scanning value:", err)
		}

		s := new(big.Int)
		_, error := fmt.Sscan(balanceSender, s)
		if error != nil {
			log.Println("error scanning value:", error)
		}

		accountR := new(big.Int)
		_, er := fmt.Sscan(accountNonceRec, accountR)
		if er != nil {
			log.Println("error scanning value:", er)
		}

		accountS := new(big.Int)
		_, errors := fmt.Sscan(accountNonceSen, accountS)
		if errors != nil {
			log.Println("error scanning value:", error)
		}

		newBalanceReceiver.Add(r, tx.Value())
		newBalanceSender.Sub(s, tx.Value())

		newAccountNonceReceiver.Add(accountR, nonceIncrement)
		newAccountNonceSender.Add(accountS, nonceIncrement)

		_, err = sqldb.Exec(updateSQLStatement, toAddr, newBalanceReceiver.String(), newAccountNonceReceiver.String())
		if err != nil {
			panic(err)
		}

		_, err = sqldb.Exec(updateSQLStatement, fromAddr, newBalanceSender.String(), newAccountNonceSender.String())
		if err != nil {
			panic(err)
		}
	}
	return nil
}

func WriteBalanceHelper(sqldb *sql.DB, tx *types.Transaction) (SendAndReceive, string, string, string, string) {
	sendAndReceiveData := SendAndReceive{
		To: tx.To().Hex(),
		From: tx.From().Hex(),
		Amount: tx.Value().String(),
	}

	toAddr := sendAndReceiveData.To
	fromAddr := sendAndReceiveData.From

	getAccountBalanceReceiver := GetAccount(sqldb, toAddr)
	getAccountBalanceSender:= GetAccount(sqldb, fromAddr)

	var receiverBalance SendAndReceive
	if err := json.Unmarshal([]byte(getAccountBalanceReceiver), &receiverBalance); err != nil {
		log.Fatal(err)
	}

	var senderBalance SendAndReceive
	if err := json.Unmarshal([]byte(getAccountBalanceSender), &senderBalance); err != nil {
		log.Fatal(err)
	}

	balanceReceiver := receiverBalance.Balance
	balanceSender := senderBalance.Balance

	accountNonceReceiver := receiverBalance.TxCountAccount
	accountNonceSender := senderBalance.TxCountAccount

	return sendAndReceiveData, balanceReceiver, balanceSender, accountNonceReceiver, accountNonceSender
}

// @NOTE: This function is extremely complex and requires heavy testing and knowdlege of edge cases:
// uncle blocks, account balance updates based on reorgs, diverges that get dropped.
// Reason for this is because the accounts are not deterministic like the block and tx hashes.
// @TODO: Calculate reorg
func WriteMinerRewards(sqldb *sql.DB, block *types.Block) {
	minerAddr := block.Coinbase().String()
	shyftConduitAddress := Rewards.ShyftNetworkConduitAddress.String()
	// Calculate the total gas used in the block
	totalGas := new(big.Int)
	for _, tx := range block.Transactions() {
		totalGas.Add(totalGas, new(big.Int).Mul(tx.GasPrice(), new(big.Int).SetUint64(tx.Gas())))
	}

	totalMinerReward := totalGas.Add(totalGas, Rewards.ShyftMinerBlockReward)

	// References:
	// https://ethereum.stackexchange.com/questions/27172/different-uncles-reward
	// line 551 in consensus.go (shyft_go-ethereum/consensus/ethash/consensus.go)
	// Some weird constants to avoid constant memory allocs for them.
	var big8   = big.NewInt(8)
	var uncleRewards []*big.Int
	var uncleAddrs []string

	// uncleReward is overwritten after each iteration
	uncleReward := new(big.Int)
	for _, uncle := range block.Uncles() {
		uncleReward.Add(uncle.Number, big8)
		uncleReward.Sub(uncleReward, block.Number())
		uncleReward.Mul(uncleReward, Rewards.ShyftMinerBlockReward)
		uncleReward.Div(uncleReward, big8)
		uncleRewards = append(uncleRewards, uncleReward)
		uncleAddrs = append(uncleAddrs, uncle.Coinbase.String())
	}

	StoreReward(sqldb, minerAddr, totalMinerReward)
	StoreReward(sqldb, shyftConduitAddress, Rewards.ShyftNetworkBlockReward)
	for i := 0; i < len(uncleAddrs); i++ {
		StoreReward(sqldb, uncleAddrs[i], uncleRewards[i])
	}
}

func StoreReward(sqldb *sql.DB, address string, reward *big.Int) {
	// Check if address exists
	var addressBalance string
	addressExistsStatement := `SELECT balance from accounts WHERE addr = ($1)`
	err := sqldb.QueryRow(addressExistsStatement, address).Scan(&addressBalance)

	if err == sql.ErrNoRows {
		// Addr does not exist, thus create new entry
		createAddressSqlStatement := `INSERT INTO accounts(addr, balance, txCountAccount) VALUES(($1), ($2), ($3)) RETURNING addr`

		// We convert totalReward into a string and postgres converts into number
		_, insertErr := sqldb.Exec(createAddressSqlStatement, address, reward.String(), 0)
		if insertErr != nil {
			panic(insertErr)
		}
		return
	} else if err != nil {
		// Something went wrong panic
		panic(err)
	} else {
		// Addr exists, update existing balance
		bigBalance := new(big.Int)
		bigBalance, err := bigBalance.SetString(addressBalance, 0)
		if !err {
			panic(err)
		}
		newBalance := new(big.Int)
		newBalance.Add(newBalance, bigBalance)
		newBalance.Add(newBalance, reward)

		updateAddressSQLStatement := `UPDATE accounts SET balance = ($1) WHERE addr = ($2)`

		// We convert totalReward into a string and postgres converts into number
		_, updateErr := sqldb.Exec(updateAddressSQLStatement, newBalance.String(), address)
		if updateErr != nil {
			panic(updateErr)
		}
		return
	}
}

///////////
// Getters
//////////
//GetAllBlocks returns []SBlock blocks for API
func GetAllBlocks(sqldb *sql.DB) string {
	var arr blockRes
	var blockArr string
	rows, err := sqldb.Query(`
		SELECT
			hash,
			coinbase,
			gasused,
			gaslimit,
			txcount,
			unclecount,
			age,
			number
		FROM blocks`)
	if err != nil {
		fmt.Println("err")
	}
	defer rows.Close()

	for rows.Next() {
		var hash string
		var coinbase string
		var gasUsed string
		var gasLimit string
		var txCount string
		var uncleCount string
		var age string
		var num string

		err = rows.Scan(
			&hash,
			&coinbase,
			&gasUsed,
			&gasLimit,
			&txCount,
			&uncleCount,
			&age,
			&num,
		)

		arr.Blocks = append(arr.Blocks, SBlock{
			Hash:     hash,
			Coinbase: coinbase,
			GasUsed: gasUsed,
			GasLimit: gasLimit,
			TxCount: txCount,
			UncleCount: uncleCount,
			Age: age,
			Number:   num,
		})

		blocks, _ := json.Marshal(arr.Blocks)
		blocksFmt := string(blocks)
		blockArr = blocksFmt
	}
	return blockArr
}

//GetBlock queries to send single block info
//TODO provide blockHash arg passed from handler.go
func GetBlock(sqldb *sql.DB, blockNumber string) string {
	sqlStatement := `SELECT * FROM blocks WHERE number=$1;`
	row := sqldb.QueryRow(sqlStatement, blockNumber)
	var hash string
	var coinbase string
	var gasUsed string
	var gasLimit string
	var txCount string
	var uncleCount string
	var age string
	var parentHash string
	var uncleHash string
	var difficulty string
	var size string
	var nonce string
	var num string
	row.Scan(
		&hash,
		&coinbase,
		&gasUsed,
		&gasLimit,
		&txCount,
		&uncleCount,
		&age,
		&parentHash,
		&uncleHash,
		&difficulty,
		&size,
		&nonce,
		&num,)

	block := SBlock{
		Hash:     hash,
		Coinbase: coinbase,
		GasUsed: gasUsed,
		GasLimit: gasLimit,
		TxCount: txCount,
		UncleCount: uncleCount,
		Age: age,
		ParentHash:parentHash,
		UncleHash:uncleHash,
		Difficulty:difficulty,
		Size: size,
		Nonce:nonce,
		Number:   num,
	}
	json, _ := json.Marshal(block)
	return string(json)
}

//GetAllTransactions getter fn for API
func GetAllTransactions(sqldb *sql.DB) string {
	var arr txRes
	var txx string
	rows, err := sqldb.Query(`
		SELECT * FROM txs`)
	if err != nil {
		fmt.Println("err")
	}
	defer rows.Close()
	for rows.Next() {
		var txhash string
		var to_addr string
		var from_addr string
		var blockhash string
		var blocknumber string
		var amount uint64
		var gasprice uint64
		var gas uint64
		var gasLimit uint64
		var txfee uint64
		var nonce uint64
		var status string
		var isContract bool
		var age time.Time
		var data []byte
		err = rows.Scan(
			&txhash,
			&to_addr,
			&from_addr,
			&blockhash,
			&blocknumber,
			&amount,
			&gasprice,
			&gas,
			&gasLimit,
			&txfee,
			&nonce,
			&status,
			&isContract,
			&age,
			&data,
		)

		arr.TxEntry = append(arr.TxEntry, ShyftTxEntryPretty{
			TxHash:    txhash,
			To:        to_addr,
			From:      from_addr,
			BlockHash: blockhash,
			BlockNumber: blocknumber,
			Amount:    amount,
			GasPrice:  gasprice,
			Gas:       gas,
			GasLimit: gasLimit,
			Cost:      txfee,
			Nonce:     nonce,
			Status:    status,
			IsContract: isContract,
			Age:		age,
			Data: 		data,
		})

		tx, _ := json.Marshal(arr.TxEntry)
		newtx := string(tx)
		txx = newtx
	}
	return txx
}

//GetTransaction fn returns single tx
func GetTransaction(sqldb *sql.DB, txHash string) string {
	sqlStatement := `SELECT * FROM txs WHERE txhash=$1;`
	row := sqldb.QueryRow(sqlStatement, txHash)
	var txhash string
	var to_addr string
	var from_addr string
	var blockhash string
	var blocknumber string
	var amount uint64
	var gasprice uint64
	var gas uint64
	var gasLimit uint64
	var txfee uint64
	var nonce uint64
	var status string
	var isContract bool
	var age time.Time
	var data []byte
	row.Scan(
		&txhash,
		&to_addr,
		&from_addr,
		&blockhash,
		&blocknumber,
		&amount,
		&gasprice,
		&gas,
		&gasLimit,
		&txfee,
		&nonce,
		&status,
		&isContract,
		&age,
		&data)

	tx := ShyftTxEntryPretty{
		TxHash:    txhash,
		To:        to_addr,
		From:      from_addr,
		BlockHash: blockhash,
		BlockNumber: blocknumber,
		Amount:    amount,
		GasPrice:  gasprice,
		Gas:       gas,
		GasLimit:	gasLimit,
		Cost:      txfee,
		Nonce:     nonce,
		Status:    status,
		IsContract: isContract,
		Age:		age,
		Data:	   data,
	}
	json, _ := json.Marshal(tx)

	return string(json)
}

//GetAccount returns account balances
func GetAccount(sqldb *sql.DB, address string) string {
	sqlStatement := `SELECT * FROM accounts WHERE addr=$1;`
	row := sqldb.QueryRow(sqlStatement, address)
	var addr string
	var balance string
	var txCountAccount string
	row.Scan(
		&addr,
		&balance,
		&txCountAccount)

	account := SAccounts{
		Addr:    addr,
		Balance: balance,
		TxCountAccount: txCountAccount,
	}
	json, _ := json.Marshal(account)
	return string(json)
}

//GetAllAccounts returns all accounts and balances
func GetAllAccounts(sqldb *sql.DB) string {
	var array accountRes
	var accountsArr string
	var txCountAccount string
	accs, err := sqldb.Query(`
		SELECT
			addr,
			balance,
			txCountAccount
		FROM accounts`)
	if err != nil {
		fmt.Println(err)
	}

	defer accs.Close()

	for accs.Next() {
		var addr string
		var balance string
		err = accs.Scan(
			&addr,
			&balance,
			&txCountAccount,
		)

		array.AllAccounts = append(array.AllAccounts, SAccounts{
			Addr:    addr,
			Balance: balance,
			TxCountAccount: txCountAccount,
		})

		accounts, _ := json.Marshal(array.AllAccounts)
		accountsFmt := string(accounts)
		accountsArr = accountsFmt
	}
	return accountsArr
}