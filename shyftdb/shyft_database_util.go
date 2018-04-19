package shyftdb

import (
    "fmt"
    "bytes"
    "encoding/gob"
	"math/big"

    "github.com/syndtr/goleveldb/leveldb"
    "github.com/syndtr/goleveldb/leveldb/util"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	
	"database/sql"
	_ "github.com/lib/pq"
)


type SBlock struct {
	hash string
	txes []string
}

type ShyftTxEntry struct {
	TxHash    common.Hash
	To   	  *common.Address
	From 	  *common.Address
	BlockHash common.Hash
	Amount 	  *big.Int
	GasPrice  *big.Int
	Gas 	  uint64
	Nonce     uint64
	Data      []byte
}

type ShyftTxEntryPretty struct {
	TxHash    string
	To   	  string
	From 	  string
	BlockHash string
	Amount 	  *big.Int
	GasPrice  *big.Int
	Gas 	  uint64
	Nonce     uint64
	Data      []byte
}

type ShyftAccountEntry struct {
	Balance *big.Int
	Txs     []string
}

//func WriteBlock(db *leveldb.DB, block *types.Block) error {
//	fmt.Println("+++++++++++++++++++++++++++ BLOCK NUMBER", block.Number())
//	fmt.Println("+++++++++++++++++++++++++++ # of TX", len(block.Transactions()))
//	leng := block.Transactions().Len()
//	var tx_strs = make([]string, leng)
//	hash := block.Header().Hash().Bytes()
//
//	buf := &bytes.Buffer{}
//	gob.NewEncoder(buf).Encode(tx_strs)
//	bs := buf.Bytes()
//
//    key := append([]byte("bk-")[:], hash[:]...)
//	if err := db.Put(key, bs, nil); err != nil {
//		log.Crit("Failed to store block", "err", err)
//		return nil // Do we want to force an exit here?
//	}
//	WriteMinerReward(db, block)
//
//	if block.Transactions().Len() > 0 {
//		for i, tx := range block.Transactions() {
//			tx_strs[i] = WriteTransactions(db, tx, block.Header().Hash())
//		}
//	}

func WriteBlock(sqldb *sql.DB, block *types.Block) error {

    //hash := block.Header().Hash().Bytes()
    coinbase := block.Header().Coinbase.String()
    number := block.Header().Number.String()
    
// 	if block.Transactions().Len() > 0 {
//		for i, tx := range block.Transactions() {
// 			tx_strs[i] = WriteTransactions(db, tx, block.Header().Hash())
// 			//tx_bytes[i] = tx.Hash().Bytes()
//  		}
// 	}
	
	//connStr := "user=postgres dbname=shyftdb sslmode=disable"
	//sqldb, err := sql.Open("postgres", connStr)

	//if merr := sqldb.Ping(); merr != nil {
	//    fmt.Println("ping ERROR")
  	//	fmt.Println(merr)
	//}

    //sqldb.Exec("INSERT INTO block(hash, miner) VALUES ($1)", block)
  	//qerr := sqldb.QueryRow(`INSERT INTO block(hash, miner) VALUES('bark', 'willow')`).Scan(&fun)
  	res, qerr := sqldb.Exec(`INSERT INTO blocks(hash, coinbase, number) VALUES(($1), ($2), ($3))`, block.Header().Hash().Hex(), coinbase, number) //.Scan(&fun)
  	fmt.Println("insert ERROR")
  	fmt.Println(qerr)
  	fmt.Println(res)
  	fmt.Println(number)

	return nil
}

func WriteTransactions(db *leveldb.DB, tx *types.Transaction, blockHash common.Hash) string {
	txData := ShyftTxEntry{
		TxHash:    tx.Hash(),
		To:   	   tx.To(),
		From: 	   tx.From(),
		BlockHash: blockHash,
		Amount:    tx.Value(),
		GasPrice:  tx.GasPrice(),
		Gas:   	   tx.Gas(),
		Nonce:     tx.Nonce(),
		Data:      tx.Data(),
	}
	var encodedData bytes.Buffer
	encoder := gob.NewEncoder(&encodedData)
	if err := encoder.Encode(txData); err != nil {
		log.Crit("Faild to encode TX data", "err", err)
	}
	key := append([]byte("tx-")[:], tx.Hash().Bytes()[:]...)
	if err := db.Put(key, encodedData.Bytes(), nil); err != nil {
		log.Crit("Failed to store TX", "err", err)
	}
	//WriteAccountBalances(db, tx)
	return tx.Hash().String()
}

func WriteFromBalance(db *leveldb.DB, tx *types.Transaction) {
	key := append([]byte("acc-")[:], tx.From().Hash().Bytes()[:]...)
	// The from (sender) addr must have balance. If it fails to retrieve there is a bigger issue.
	retrievedData, err := db.Get(key, nil)
	if err != nil {
		log.Crit("From MUST have eth and no record found", "err", err)
	}
	var decodedData ShyftAccountEntry
	d := gob.NewDecoder(bytes.NewBuffer(retrievedData))
	if err := d.Decode(&decodedData); err != nil {
		log.Crit("Failed to decode From data:", "err", err)
	}
	decodedData.Balance.Sub(decodedData.Balance, tx.Value())
	decodedData.Txs = append(decodedData.Txs, tx.Hash().String())
	// Encode updated data
	var encodedData bytes.Buffer
	encoder := gob.NewEncoder(&encodedData)
	if err := encoder.Encode(decodedData); err != nil {
		log.Crit("Faild to encode From Account data", "err", err)
	}
	if err := db.Put(key, encodedData.Bytes(), nil); err != nil {
		log.Crit("Could not write the From account data", "err", err)
	}
}

func WriteToBalance(db *leveldb.DB, tx *types.Transaction) {
	key := append([]byte("acc-")[:], tx.To().Hash().Bytes()[:]...)
	var txs []string

	retrievedData, err := db.Get(key, nil)
	if err != nil {
		accData := ShyftAccountEntry{
			Balance: tx.Value(),
			Txs: append(txs, tx.Hash().String()),
		}
		var encodedData bytes.Buffer
		encoder := gob.NewEncoder(&encodedData)
		if err := encoder.Encode(accData); err != nil {
			log.Crit("Faild to encode To Account data", "err", err)
		}
		if err := db.Put(key, encodedData.Bytes(), nil); err != nil {
			log.Crit("Could not write the TO account's first tx", "err", err)
		}
	}
	var decodedData ShyftAccountEntry
	d := gob.NewDecoder(bytes.NewBuffer(retrievedData))
	if err := d.Decode(&decodedData); err != nil {
		log.Crit("Failed to decode To account data:", "err", err)
	}
	decodedData.Balance.Add(decodedData.Balance, tx.Value())
	decodedData.Txs = append(decodedData.Txs, tx.Hash().String())
	// Encode updated data
	var encodedData bytes.Buffer
	encoder := gob.NewEncoder(&encodedData)
	if err := encoder.Encode(decodedData); err != nil {
		log.Crit("Faild to encode To Account data", "err", err)
	}
	if err := db.Put(key, encodedData.Bytes(), nil); err != nil {
		log.Crit("Could not write the To account data", "err", err)
	}
}

// @NOTE: This function is extremely complex and requires heavy testing and knowdlege of edge cases:
// uncle blocks, account balance updates based on reorgs, diverges that get dropped.
// Reason for this is because the accounts are not deterministic like the block and tx hashes.
// @TODO: Calculate reward if there are uncles
// @TODO: Calculate mining reward (most likely retrieve higher up in the operations)
// @TODO: Calculate reorg
func WriteMinerReward(db *leveldb.DB, block *types.Block)  {
	var totalGas *big.Int
	var txs []string
	key := append([]byte("acc-")[:], block.Coinbase().Hash().Bytes()[:]...)
	for _, tx := range block.Transactions() {
		totalGas.Add(totalGas, new(big.Int).Mul(tx.GasPrice(), new(big.Int).SetUint64(tx.Gas())))
	}
	retrievedData, err := db.Get(key, nil)
	if err != nil {
		// Assume time this account has had a tx
		// Balacne is exclusively minerreward + total gas from the block b/c no prior evm activity
		// Txs would be empty because they have not had any transactions on the EVM
		// @TODO: Calc mining reward
		//balance := totalGas.Add(totalGas, MINING_REWARD)
		balance := totalGas
		accData := ShyftAccountEntry{
			Balance: balance,
			Txs: txs,
		}
		var encodedData bytes.Buffer
		encoder := gob.NewEncoder(&encodedData)
		if err := encoder.Encode(accData); err != nil {
			log.Crit("Faild to encode Miner Account data", "err", err)
		}
		if err := db.Put(key, encodedData.Bytes(), nil); err != nil {
			log.Crit("Could not write the miner's first tx", "err", err)
		}
	} else {
		// The account has already have previous data stored due to activity in the EVM
		// Decode the data to update balance
		var decodedData ShyftAccountEntry
		d := gob.NewDecoder(bytes.NewBuffer(retrievedData))
		if err := d.Decode(&decodedData); err != nil {
			log.Crit("Failed to decode miner data:", "err", err)
		}
		// Write new balance
		// @TODO: Calc mining reward
		// decodedData.Balance.Add(decodedData.Balance, totalGas.Add(totalGas, MINING_REWARD)))
		decodedData.Balance.Add(decodedData.Balance, totalGas)
		// Encode the data to be written back to the db
		var encodedData bytes.Buffer
		encoder := gob.NewEncoder(&encodedData)
		if err := encoder.Encode(decodedData); err != nil {
			log.Crit("Faild to encode Miner Account data", "err", err)
		}
		// Write newly encoded data back to the db
		if err := db.Put(key, encodedData.Bytes(), nil); err != nil {
			log.Crit("Could not update miner account data", "err", err)
		}
	}
}

///////////
// Getters
//////////

func GetAllBlocks(db *leveldb.DB) []SBlock{
	var arr []SBlock
	iter := db.NewIterator(util.BytesPrefix([]byte("bk-")), nil)
	for iter.Next() {
	    result := iter.Value()
	    buf := bytes.NewBuffer(result)
		strs2 := []string{}
		gob.NewDecoder(buf).Decode(&strs2)
		//fmt.Println("the key is")
		hash := common.BytesToHash(iter.Key())
		hex := hash.Hex()
		//fmt.Println(hex)
		sblock := SBlock{hex, strs2}
		arr = append(arr, sblock)

		//fmt.Println("\n ALL BK BK VALUE" + string(result))
	}
	
	iter.Release()
	return arr
}

func GetBlock(db *leveldb.DB, block *types.Block) []byte {
	hash := block.Header().Hash().Bytes()
	key := append([]byte("bk-")[:], hash[:]...)
	data, err := db.Get(key, nil)
	if err != nil {
		log.Crit("Could not retrieve block", "err", err)
	}
	fmt.Println("\nBLOCK Value: " + string(data))
	return data
}

func GetAllTransactions(db *leveldb.DB) []ShyftTxEntryPretty {
	var txs []ShyftTxEntryPretty
	iter := db.NewIterator(util.BytesPrefix([]byte("tx-")), nil)
	for iter.Next() {
		var txData ShyftTxEntry
		d := gob.NewDecoder(bytes.NewBuffer(iter.Value()))
		if err := d.Decode(&txData); err != nil {
			log.Crit("Failed to decode tx:", "err", err)
		}
		prettyFormat := ShyftTxEntryPretty{
			TxHash:    txData.TxHash.Hex(),
			From:  	   txData.From.Hex(),
			To:  	   txData.To.Hex(),
			BlockHash: txData.BlockHash.Hex(),
			Amount:    txData.Amount,
			Gas:       txData.Gas,
			GasPrice:  txData.GasPrice,
			Nonce:     txData.Nonce,
			Data:      txData.Data,
		}
		txs = append(txs, prettyFormat)
		// Uncomment to view "pretty" version of data
		//fmt.Println("DECODED TX")
		//fmt.Println("Tx Hash: ", txData.TxHash.Hex())
		//fmt.Println("From: ", txData.From.Hex())
		//fmt.Println("To: ", txData.To.Hex())
		//fmt.Println("BlockHash: ", txData.BlockHash.Hex())
		//fmt.Println("Amount: ", txData.Amount)
		//fmt.Println("Gas: ", txData.Gas)
		//fmt.Println("GasPrice: ", txData.GasPrice)
		//fmt.Println("Nonce: ", txData.Nonce)
		//fmt.Println("Data: ", txData.Data)
	}
	iter.Release()
	return txs
}

func GetTransaction (db *leveldb.DB, tx *types.Transaction) ShyftTxEntryPretty {
	var prettyFormat ShyftTxEntryPretty
	key := append([]byte("tx-")[:], tx.Hash().Bytes()[:]...)
	data, err := db.Get(key, nil)
	if err != nil {
		log.Crit("Could not retrieve TX", "err", err)
	}
	if len(data) > 0 {
		var txData ShyftTxEntry
		d := gob.NewDecoder(bytes.NewBuffer(data))
		if err := d.Decode(&txData); err != nil {
			log.Crit("Failed to decode tx:", "err", err)
		}
		prettyFormat = ShyftTxEntryPretty{
			TxHash:    txData.TxHash.Hex(),
			From:  	   txData.From.Hex(),
			To:  	   txData.To.Hex(),
			BlockHash: txData.BlockHash.Hex(),
			Amount:    txData.Amount,
			Gas:       txData.Gas,
			GasPrice:  txData.GasPrice,
			Nonce:     txData.Nonce,
			Data:      txData.Data,
		}
		// Uncomment to view "pretty" version of data
		//fmt.Println("DECODED TX")
		//fmt.Println("Tx Hash: ", txData.TxHash.Hex())
		//fmt.Println("From: ", txData.From.Hex())
		//fmt.Println("To: ", txData.To.Hex())
		//fmt.Println("BlockHash: ", txData.BlockHash.Hex())
		//fmt.Println("Amount: ", txData.Amount)
		//fmt.Println("Gas: ", txData.Gas)
		//fmt.Println("GasPrice: ", txData.GasPrice)
		//fmt.Println("Nonce: ", txData.Nonce)
		//fmt.Println("Data: ", txData.Data)
	}
	return prettyFormat
}
