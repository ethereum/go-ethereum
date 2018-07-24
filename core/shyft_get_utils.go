package core

import (
	"encoding/json"
	"database/sql"
	"fmt"
	"time"
)

///////////
// Getters
//////////
func SGetAllBlocks(sqldb *sql.DB) string {
	var arr blockRes
	var blockArr string
	rows, err := sqldb.Query(`SELECT * FROM blocks`)
	if err != nil {
		fmt.Println("err")
	}
	defer rows.Close()

	for rows.Next() {
		var hash, coinbase, age, parentHash, uncleHash, difficulty, size, rewards, num string
		var gasUsed, gasLimit, nonce uint64
		var txCount, uncleCount int

		err = rows.Scan(
			&hash, &coinbase, &gasUsed, &gasLimit, &txCount, &uncleCount, &age, &parentHash, &uncleHash, &difficulty, &size, &nonce, &rewards, &num,)

		arr.Blocks = append(arr.Blocks, SBlock{
			Hash:     		hash,
			Coinbase: 		coinbase,
			GasUsed: 		gasUsed,
			GasLimit: 		gasLimit,
			TxCount: 		txCount,
			UncleCount: 	uncleCount,
			Age: 			age,
			ParentHash:		parentHash,
			UncleHash:		uncleHash,
			Difficulty:		difficulty,
			Size: 			size,
			Nonce:			nonce,
			Rewards: 		rewards,
			Number:   		num,
		})

		blocks, _ := json.Marshal(arr.Blocks)
		blocksFmt := string(blocks)
		blockArr = blocksFmt
	}
	return blockArr
}

//GetBlock queries to send single block info
//TODO provide blockHash arg passed from handler.go
func SGetBlock(sqldb *sql.DB, blockNumber string) string {
	sqlStatement := `SELECT * FROM blocks WHERE number=$1;`
	row := sqldb.QueryRow(sqlStatement, blockNumber)
	var hash, coinbase, age, parentHash, uncleHash, difficulty, size, rewards, num string
	var gasUsed, gasLimit, nonce uint64
	var txCount, uncleCount int

	row.Scan(
		&hash, &coinbase, &gasUsed, &gasLimit, &txCount, &uncleCount, &age, &parentHash, &uncleHash, &difficulty, &size, &nonce, &rewards, &num,)

	block := SBlock{
		Hash:     	hash,
		Coinbase: 	coinbase,
		GasUsed: 	gasUsed,
		GasLimit: 	gasLimit,
		TxCount: 	txCount,
		UncleCount: uncleCount,
		Age: 		age,
		ParentHash:	parentHash,
		UncleHash:	uncleHash,
		Difficulty:	difficulty,
		Size: 		size,
		Nonce:		nonce,
		Rewards: 	rewards,
		Number:   	num,
	}
	json, _ := json.Marshal(block)
	return string(json)
}

func SGetRecentBlock(sqldb *sql.DB) string {
	sqlStatement := `SELECT * FROM blocks WHERE number=(SELECT MAX(number) FROM blocks);`
	row := sqldb.QueryRow(sqlStatement)
	var hash, coinbase, age, parentHash, uncleHash, difficulty, size, rewards, num string
	var gasUsed, gasLimit, nonce uint64
	var txCount, uncleCount int

	row.Scan(
		&hash, &coinbase, &gasUsed, &gasLimit, &txCount, &uncleCount, &age, &parentHash, &uncleHash, &difficulty, &size, &nonce, &rewards, &num,)

	block := SBlock{
		Hash:     	hash,
		Coinbase: 	coinbase,
		GasUsed: 	gasUsed,
		GasLimit: 	gasLimit,
		TxCount: 	txCount,
		UncleCount: uncleCount,
		Age: 		age,
		ParentHash:	parentHash,
		UncleHash:	uncleHash,
		Difficulty:	difficulty,
		Size: 		size,
		Nonce:		nonce,
		Rewards: 	rewards,
		Number:   	num,
	}
	json, _ := json.Marshal(block)
	return string(json)
}

func SGetAllTransactionsFromBlock(sqldb *sql.DB, blockNumber string) string {
	var arr txRes
	var txx string
	sqlStatement := `SELECT * FROM txs WHERE blocknumber=$1`
	rows, err := sqldb.Query(sqlStatement, blockNumber)
	if err != nil {
		fmt.Println("err")
	}
	defer rows.Close()
	for rows.Next() {
		var txhash, to_addr, from_addr, blockhash, blocknumber, amount, status string
		var gasprice, gas, gasLimit, txfee, nonce uint64
		var isContract bool
		var age time.Time
		var data []byte

		err = rows.Scan(
			&txhash, &to_addr, &from_addr, &blockhash, &blocknumber, &amount, &gasprice, &gas, &gasLimit, &txfee, &nonce, &status, &isContract, &age, &data,
		)

		arr.TxEntry = append(arr.TxEntry, ShyftTxEntryPretty{
			TxHash:    	 txhash,
			To:        	 to_addr,
			From:      	 from_addr,
			BlockHash: 	 blockhash,
			BlockNumber: blocknumber,
			Amount:    	 amount,
			GasPrice:  	 gasprice,
			Gas:       	 gas,
			GasLimit: 	 gasLimit,
			Cost:      	 txfee,
			Nonce:     	 nonce,
			Status:    	 status,
			IsContract:  isContract,
			Age:		 age,
			Data: 		 data,
		})

		tx, _ := json.Marshal(arr.TxEntry)
		newtx := string(tx)
		txx = newtx
	}
	return txx
}

func SGetAllBlocksMinedByAddress(sqldb *sql.DB, coinbase string) string {
	var arr blockRes
	var blockArr string
	sqlStatement := `SELECT * FROM blocks WHERE coinbase=$1`
	rows, err := sqldb.Query(sqlStatement, coinbase)
	if err != nil {
		fmt.Println("err")
	}
	defer rows.Close()

	for rows.Next() {
		var hash, coinbase, age, parentHash, uncleHash, difficulty, size, rewards, num string
		var gasUsed, gasLimit, nonce uint64
		var txCount, uncleCount int

		err = rows.Scan(
			&hash, &coinbase, &gasUsed, &gasLimit, &txCount, &uncleCount, &age, &parentHash, &uncleHash, &difficulty, &size, &nonce, &rewards, &num,)

		arr.Blocks = append(arr.Blocks, SBlock{
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
			Rewards: rewards,
			Number:   num,
		})

		blocks, _ := json.Marshal(arr.Blocks)
		blocksFmt := string(blocks)
		blockArr = blocksFmt
	}
	return blockArr
}

//GetAllTransactions getter fn for API
func SGetAllTransactions(sqldb *sql.DB) string {
	var arr txRes
	var txx string
	rows, err := sqldb.Query(`SELECT * FROM txs`)
	if err != nil {
		fmt.Println("err")
	}
	defer rows.Close()
	for rows.Next() {
		var txhash, to_addr, from_addr, blockhash, blocknumber, amount, status string
		var gasprice, gas, gasLimit, txfee, nonce uint64
		var isContract bool
		var age time.Time
		var data []byte

		err = rows.Scan(
			&txhash, &to_addr, &from_addr, &blockhash, &blocknumber, &amount, &gasprice, &gas, &gasLimit, &txfee, &nonce, &status, &isContract, &age, &data,
		)

		arr.TxEntry = append(arr.TxEntry, ShyftTxEntryPretty{
			TxHash:    	 txhash,
			To:        	 to_addr,
			From:      	 from_addr,
			BlockHash: 	 blockhash,
			BlockNumber: blocknumber,
			Amount:    	 amount,
			GasPrice:  	 gasprice,
			Gas:       	 gas,
			GasLimit: 	 gasLimit,
			Cost:      	 txfee,
			Nonce:     	 nonce,
			Status:    	 status,
			IsContract:  isContract,
			Age:		 age,
			Data: 		 data,
		})

		tx, _ := json.Marshal(arr.TxEntry)
		newtx := string(tx)
		txx = newtx
	}
	return txx
}

//GetTransaction fn returns single tx
func SGetTransaction(sqldb *sql.DB, txHash string) string {
	sqlStatement := `SELECT * FROM txs WHERE txhash=$1;`
	row := sqldb.QueryRow(sqlStatement, txHash)
	var txhash, to_addr, from_addr, blockhash, blocknumber, amount, status string
	var gasprice, gas, gasLimit, txfee, nonce uint64
	var isContract bool
	var age time.Time
	var data []byte

	row.Scan(
		&txhash, &to_addr, &from_addr, &blockhash, &blocknumber, &amount, &gasprice, &gas, &gasLimit, &txfee, &nonce, &status, &isContract, &age, &data)

	tx := ShyftTxEntryPretty{
		TxHash:      txhash,
		To:        	 to_addr,
		From:      	 from_addr,
		BlockHash: 	 blockhash,
		BlockNumber: blocknumber,
		Amount:      amount,
		GasPrice:    gasprice,
		Gas:         gas,
		GasLimit:	 gasLimit,
		Cost:      	 txfee,
		Nonce:     	 nonce,
		Status:    	 status,
		IsContract:  isContract,
		Age:		 age,
		Data:	     data,
	}
	json, _ := json.Marshal(tx)

	return string(json)
}

func InnerSGetAccount(sqldb *sql.DB, address string) (SAccounts, bool) {
	sqlStatement := `SELECT * FROM accounts WHERE addr=$1;`
	var addr, balance, accountNonce string
	err := sqldb.QueryRow(sqlStatement, address).Scan(&addr, &balance, &accountNonce)
	if err == sql.ErrNoRows {
		return SAccounts{}, false
	} else {
		account := SAccounts{
			Addr:    		addr,
			Balance: 		balance,
			AccountNonce: 	accountNonce,
		}
		return account, true
	}
}

//GetAccount returns account balances
func SGetAccount(sqldb *sql.DB, address string) string {
	var account, _ = InnerSGetAccount(sqldb, address)
	json, _ := json.Marshal(account)
	return string(json)
}

//GetAllAccounts returns all accounts and balances
func SGetAllAccounts(sqldb *sql.DB) string {
	var array accountRes
	var accountsArr, accountNonce string

	accs, err := sqldb.Query(`
		SELECT
			addr,
			balance,
			accountNonce
		FROM accounts`)
	if err != nil {
		fmt.Println(err)
	}

	defer accs.Close()

	for accs.Next() {
		var addr, balance string
		err = accs.Scan(
			&addr, &balance, &accountNonce,
		)

		array.AllAccounts = append(array.AllAccounts, SAccounts{
			Addr:    		addr,
			Balance: 		balance,
			AccountNonce: 	accountNonce,
		})

		accounts, _ := json.Marshal(array.AllAccounts)
		accountsFmt := string(accounts)
		accountsArr = accountsFmt
	}
	return accountsArr
}

//GetAccount returns account balances
func SGetAccountTxs(sqldb *sql.DB, address string) string {
	var arr txRes
	var txx string
	sqlStatement := `SELECT * FROM txs WHERE to_addr=$1 OR from_addr=$1;`
	rows, err := sqldb.Query(sqlStatement, address)
	if err != nil {
		fmt.Println("err", err)
	}
	defer rows.Close()
	for rows.Next() {
		var txhash, to_addr, from_addr, blockhash, blocknumber, amount, status string
		var gasprice, gas, gasLimit, txfee, nonce uint64
		var isContract bool
		var age time.Time
		var data []byte

		err = rows.Scan(
			&txhash, &to_addr, &from_addr, &blockhash, &blocknumber, &amount, &gasprice, &gas, &gasLimit, &txfee, &nonce, &status, &isContract, &age, &data,
		)

		arr.TxEntry = append(arr.TxEntry, ShyftTxEntryPretty{
			TxHash:    	 txhash,
			To:        	 to_addr,
			From:      	 from_addr,
			BlockHash:   blockhash,
			BlockNumber: blocknumber,
			Amount:    	 amount,
			GasPrice:  	 gasprice,
			Gas:       	 gas,
			GasLimit: 	 gasLimit,
			Cost:      	 txfee,
			Nonce:     	 nonce,
			Status:    	 status,
			IsContract:  isContract,
			Age:		 age,
			Data: 		 data,
		})

		tx, _ := json.Marshal(arr.TxEntry)
		newtx := string(tx)
		txx = newtx
	}
	return txx
}
