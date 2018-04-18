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
	return tx.Hash().String()
}

// Meant for internal tests
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