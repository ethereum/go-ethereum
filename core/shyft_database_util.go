package core

import (
	"encoding/json"
	"math/big"
	"time"
	"strconv"
	"database/sql"
	"log"
	_ "github.com/lib/pq"
	"fmt"
	"github.com/ShyftNetwork/go-empyrean/common"
	"github.com/ShyftNetwork/go-empyrean/core/types"
	Rewards "github.com/ShyftNetwork/go-empyrean/consensus/ethash"
	"github.com/ShyftNetwork/go-empyrean/shyfttracerinterface"
)

var IShyftTracer shyfttracerinterface.IShyftTracer

func SetIShyftTracer(st shyfttracerinterface.IShyftTracer) {
	IShyftTracer = st
}

//SBlock type
type SBlock struct {
	Hash     	string
	Coinbase 	string
	Age        	string
	ParentHash 	string
	UncleHash 	string
	Difficulty 	string
	Size 		string
	Rewards 	string
	Number   	string
	GasUsed	 	uint64
	GasLimit 	uint64
	Nonce 		uint64
	TxCount  	int
	UncleCount 	int
	Blocks 		[]SBlock
}

//blockRes struct
type blockRes struct {
	hash     string
	coinbase string
	number   string
	Blocks   []SBlock
}

type SAccounts struct {
	Addr    		string
	Balance 		string
	AccountNonce	string
}

type accountRes struct {
	addr        string
	balance     string
	AllAccounts []SAccounts
}

type txRes struct {
	TxEntry []ShyftTxEntryPretty
}

type ShyftTxEntryPretty struct {
	TxHash    		string
	To        		string
	From      		string
	BlockHash 		string
	BlockNumber 	string
	Amount   		string
	GasPrice  		uint64
	Gas       		uint64
	GasLimit  		uint64
	Cost	  		uint64
	Nonce     		uint64
	Status	  		string
	IsContract 		bool
	Age        		time.Time
	Data      		[]byte
}

type SendAndReceive struct {
	To        	 string
	From      	 string
	Amount    	 string
	Address   	 string
	Balance   	 string
	AccountNonce uint64 `json:",string"`
}

//WriteBlock writes to block info to sql db
func SWriteBlock(block *types.Block, receipts []*types.Receipt) error {
	sqldb, err := DBConnection()
	if err != nil {
		panic(err)
	}

	rewards := swriteMinerRewards(sqldb,block)

	blockData := SBlock{
		Hash: 			block.Header().Hash().Hex(),
		Coinbase: 		block.Header().Coinbase.String(),
		Number: 		block.Header().Number.String(),
		GasUsed: 		block.Header().GasUsed,
		GasLimit: 		block.Header().GasLimit,
		TxCount: 		block.Transactions().Len(),
		UncleCount: 	len(block.Uncles()),
		ParentHash: 	block.ParentHash().String(),
		UncleHash: 		block.UncleHash().String(),
		Difficulty: 	block.Difficulty().String(),
		Size: 			block.Size().String(),
		Nonce: 			block.Nonce(),
	}

	i, err := strconv.ParseInt(block.Time().String(), 10, 64)
	if err != nil {
		panic(err)
	}
	age := time.Unix(i, 0)

	sqlStatement := `INSERT INTO blocks(hash, coinbase, number, gasUsed, gasLimit, txCount, uncleCount, age, parentHash, uncleHash, difficulty, size, rewards, nonce) VALUES(($1), ($2), ($3), ($4), ($5), ($6), ($7), ($8), ($9), ($10), ($11), ($12),($13), ($14)) RETURNING number`
	qerr := sqldb.QueryRow(sqlStatement, blockData.Hash, blockData.Coinbase, blockData.Number, blockData.GasUsed, blockData.GasLimit, blockData.TxCount, blockData.UncleCount, age, blockData.ParentHash, blockData.UncleHash, blockData.Difficulty, blockData.Size, rewards, blockData.Nonce).Scan(&blockData.Number)
	if qerr != nil {
		panic(qerr)
	}

	if block.Transactions().Len() > 0 {
		for _, tx := range block.Transactions() {
			swriteTransactions(sqldb, tx, block.Header().Hash(), blockData.Number, receipts, age, blockData.GasLimit)
			if block.Transactions()[0].To() != nil {
				swriteFromBalance(sqldb, tx)
			}
			if block.Transactions()[0].To() == nil {
				swriteContractBalance(sqldb, tx)
			}
		}
	}
	return nil
}

//swriteTransactions writes to sqldb, a SHYFT postgres instance
func swriteTransactions(sqldb *sql.DB, tx *types.Transaction, blockHash common.Hash, blockNumber string,  receipts []*types.Receipt, age time.Time, gasLimit uint64) error {
	var isContract bool
	var statusFromReciept, contractAddressFromReciept, retNonce string

	txData := ShyftTxEntryPretty{
		TxHash:    tx.Hash().Hex(),
		From:      tx.From().Hex(),
		To:        tx.To().Hex(),
		BlockHash: blockHash.Hex(),
		Amount:    tx.Value().String(),
		Cost:	   tx.Cost().Uint64(),
		GasPrice:  tx.GasPrice().Uint64(),
		Gas:       tx.Gas(),
		Nonce:     tx.Nonce(),
		Data:      tx.Data(),
	}

	if tx.To() == nil {
		for _, receipt := range receipts {
			statusReciept := (*types.ReceiptForStorage)(receipt).Status
			contractAddressFromReciept = (*types.ReceiptForStorage)(receipt).ContractAddress.String()
			switch {
			case statusReciept == 0:
				statusFromReciept = "FAIL"
			case statusReciept == 1:
				statusFromReciept = "SUCCESS"
			}
		}
		isContract = true
		sqlStatement := `INSERT INTO txs(txhash, from_addr, to_addr, blockhash, blockNumber, amount, gasprice, gas, gasLimit, txfee, nonce, isContract, txStatus, age, data) VALUES(($1), ($2), ($3), ($4), ($5), ($6), ($7), ($8), ($9), ($10), ($11), ($12), ($13), ($14), ($15)) RETURNING nonce`
		err := sqldb.QueryRow(sqlStatement, txData.TxHash, txData.From, contractAddressFromReciept, txData.BlockHash, blockNumber, txData.Amount, txData.GasPrice, txData.Gas, gasLimit, txData.Cost, txData.Nonce, isContract, statusFromReciept, age, txData.Data).Scan(&retNonce)
		if err != nil {
			panic(err)
		}
	} else {
		isContract = false
		for _, receipt := range receipts {
			statusReciept := (*types.ReceiptForStorage)(receipt).Status
			switch {
			case statusReciept == 0:
				statusFromReciept = "FAIL"
			case statusReciept == 1:
				statusFromReciept = "SUCCESS"
			}
		}
		sqlStatement := `INSERT INTO txs(txhash, from_addr, to_addr, blockhash, blockNumber, amount, gasprice, gas, gasLimit, txfee, nonce, isContract, txStatus, age, data) VALUES(($1), ($2), ($3), ($4), ($5), ($6), ($7), ($8), ($9), ($10), ($11), ($12), ($13), ($14), ($15)) RETURNING nonce`
		err := sqldb.QueryRow(sqlStatement, txData.TxHash, txData.From, txData.To, txData.BlockHash, blockNumber, txData.Amount, txData.GasPrice, txData.Gas, gasLimit, txData.Cost, txData.Nonce, isContract, statusFromReciept, age, txData.Data).Scan(&retNonce)
		if err != nil {
			panic(err)
		}
	}
	//Runs necessary functions for tracing internal transactions through tracers.go
	IShyftTracer.GetTracerToRun(tx.Hash())

	return nil
}

func swriteContractBalance(sqldb *sql.DB, tx *types.Transaction) error {
	sendAndReceiveData := SendAndReceive{
		From:   		tx.From().Hex(),
		Amount: 		tx.Value().String(),
		AccountNonce: 	tx.Nonce(),
	}

	var response string
	sqlExistsStatement := `SELECT balance from accounts WHERE addr = ($1)`
	err := sqldb.QueryRow(sqlExistsStatement, sendAndReceiveData.From).Scan(&response)

	switch {
	case err == sql.ErrNoRows:
		sqlStatement := `INSERT INTO accounts(addr, balance, accountNonce) VALUES(($1), ($2), ($3)) RETURNING addr`
		insertErr := sqldb.QueryRow(sqlStatement, sendAndReceiveData.From, sendAndReceiveData.Amount, sendAndReceiveData.AccountNonce).Scan(&sendAndReceiveData.From)
		if insertErr != nil {
			panic(insertErr)
		}
	default:
		getAccountBalanceSender:= SGetAccount(sqldb, sendAndReceiveData.From)
		var newBalanceSender,newAccountNonceSender  big.Int
		var nonceIncrement = big.NewInt(1)

		var senderBalance SendAndReceive
		if err := json.Unmarshal([]byte(getAccountBalanceSender), &senderBalance); err != nil {
			log.Fatal(err)
		}

		//Converts string to UINT64 > Big.Int for adding and subtraction
		balanceSender, _ := strconv.ParseUint(senderBalance.Balance, 10, 64)
		balanceSen := new(big.Int).SetUint64(balanceSender)
		senderAccountNonce := new(big.Int).SetUint64(sendAndReceiveData.AccountNonce)

		newBalanceSender.Sub(balanceSen, tx.Value())
		newAccountNonceSender.Add(senderAccountNonce, nonceIncrement)

		updateSQLStatement := `UPDATE accounts SET balance = ($2), accountNonce = ($3) WHERE addr = ($1)`
		_, err := sqldb.Exec(updateSQLStatement, sendAndReceiveData.From, newBalanceSender.String(), newAccountNonceSender.String())
		if err != nil {
			panic(err)
		}
	}
	return nil
}

//writeFromBalance writes senders balance to accounts db
func swriteFromBalance(sqldb *sql.DB, tx *types.Transaction) error {
	sendAndReceiveData, balanceRec, balanceSen, accountNonceRec, accountNonceSen := swriteBalanceHelper(sqldb, tx)

	var response string
	sqlExistsStatement := `SELECT balance from accounts WHERE addr = ($1)`
	err := sqldb.QueryRow(sqlExistsStatement, sendAndReceiveData.To).Scan(&response)

	switch {
	case err == sql.ErrNoRows:
		accountNonce := strconv.FormatUint(tx.Nonce(), 10)
		sqlStatement := `INSERT INTO accounts(addr, balance, accountNonce) VALUES(($1), ($2), ($3)) RETURNING addr`
		insertErr := sqldb.QueryRow(sqlStatement, sendAndReceiveData.To, sendAndReceiveData.Amount, accountNonce).Scan(&sendAndReceiveData.To)
		if insertErr != nil {
			panic(insertErr)
		}
	case err != nil:
		log.Fatal(err)
	default:
		var newBalanceReceiver, newBalanceSender, newAccountNonceReceiver, newAccountNonceSender  big.Int
		var nonceIncrement = big.NewInt(1)

		//Convert UINT64 to BIG.INT in order to add and subtract nonces & balances
		balanceR := new(big.Int).SetUint64(balanceRec)
		balanceS := new(big.Int).SetUint64(balanceSen)

		accountR := new(big.Int).SetUint64(accountNonceRec)
		accountS := new(big.Int).SetUint64(accountNonceSen)

		newBalanceReceiver.Add(balanceR, tx.Value())
		newBalanceSender.Sub(balanceS, tx.Value())

		newAccountNonceReceiver.Add(accountR, nonceIncrement)
		newAccountNonceSender.Add(accountS, nonceIncrement)

		updateSQLStatement := `UPDATE accounts SET balance = ($2), accountNonce = ($3) WHERE addr = ($1)`
		_, err = sqldb.Exec(updateSQLStatement, sendAndReceiveData.To, newBalanceReceiver.String(), newAccountNonceReceiver.String())
		if err != nil {
			panic(err)
		}

		_, err = sqldb.Exec(updateSQLStatement, sendAndReceiveData.From, newBalanceSender.String(), newAccountNonceSender.String())
		if err != nil {
			panic(err)
		}
	}
	return nil
}

func swriteBalanceHelper(sqldb *sql.DB, tx *types.Transaction) (SendAndReceive, uint64, uint64, uint64, uint64) {
	sendAndReceiveData := SendAndReceive{
		To: tx.To().Hex(),
		From: tx.From().Hex(),
		Amount: tx.Value().String(),
	}

	fmt.Println("TO", sendAndReceiveData.To)
	fmt.Println("FROM", sendAndReceiveData.From)
	getAccountBalanceReceiver := SGetAccount(sqldb, sendAndReceiveData.To)
	getAccountBalanceSender:= SGetAccount(sqldb, sendAndReceiveData.From)
	fmt.Println("HERE", getAccountBalanceReceiver)
	var receiverData SendAndReceive
	if err := json.Unmarshal([]byte(getAccountBalanceReceiver), &receiverData); err != nil {
		log.Fatal(err)
	}

	var senderData SendAndReceive
	if err := json.Unmarshal([]byte(getAccountBalanceSender), &senderData); err != nil {
		log.Fatal(err)
	}

	balanceReceiver, _ := strconv.ParseUint(receiverData.Balance, 10, 64)
	balanceSender, _ := strconv.ParseUint(senderData.Balance, 10, 64)

	return sendAndReceiveData, balanceReceiver, balanceSender, receiverData.AccountNonce, senderData.AccountNonce
}

// @NOTE: This function is extremely complex and requires heavy testing and knowdlege of edge cases:
// uncle blocks, account balance updates based on reorgs, diverges that get dropped.
// Reason for this is because the accounts are not deterministic like the block and tx hashes.
// @TODO: Calculate reorg
func swriteMinerRewards(sqldb *sql.DB, block *types.Block) string {
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

	sstoreReward(sqldb, minerAddr, totalMinerReward)
	sstoreReward(sqldb, shyftConduitAddress, Rewards.ShyftNetworkBlockReward)
	var uncRewards = new(big.Int)
	for i := 0; i < len(uncleAddrs); i++ {
		_ = uncleRewards[i]
		sstoreReward(sqldb, uncleAddrs[i], uncleRewards[i])
	}

	fullRewardValue := new(big.Int)
	fullRewardValue.Add(totalMinerReward, Rewards.ShyftNetworkBlockReward)
	fullRewardValue.Add(fullRewardValue, uncRewards)

	return fullRewardValue.String()
}

func sstoreReward(sqldb *sql.DB, address string, reward *big.Int) {
	// Check if address exists
	var addressBalance string
	addressExistsStatement := `SELECT balance from accounts WHERE addr = ($1)`
	err := sqldb.QueryRow(addressExistsStatement, address).Scan(&addressBalance)

	if err == sql.ErrNoRows {
		// Addr does not exist, thus create new entry
		createAddressSqlStatement := `INSERT INTO accounts(addr, balance, accountNonce) VALUES(($1), ($2), ($3)) RETURNING addr`

		// We convert totalReward into a string and postgres converts into number
		_, insertErr := sqldb.Exec(createAddressSqlStatement, address, reward.String(), 1)
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


func CreateAccount (sqldb *sql.DB, addr string, amount uint64, accountNonce uint64) {

	sqlStatement := `INSERT INTO accounts(addr, balance, accountNonce) VALUES(($1), ($2), ($3)) RETURNING addr`
	insertErr := sqldb.QueryRow(sqlStatement, addr, amount, accountNonce).Scan(&addr)
	if insertErr != nil {
		panic(insertErr)
	}

}


