package shyftdb

import (
	"github.com/empyrean/go-ethereum/common"
	"github.com/empyrean/go-ethereum/core/types"
	"math/big"
	"testing"
	//"time"
	"encoding/json"
	"github.com/empyrean/go-ethereum/crypto"
	"strconv"
)

func TestBlockToReturnBlock(t *testing.T) {
	key, _ := crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	signer := types.NewEIP155Signer(big.NewInt(2147483647))

	//Nonce, To Address,Value, GasLimit, Gasprice, data
	tx1 := types.NewTransaction(1, common.BytesToAddress([]byte{0x11}), big.NewInt(111), 1111, big.NewInt(11111), []byte{0x11, 0x11, 0x11})
	mytx, _ := types.SignTx(tx1, signer, key)
	tx2 := types.NewTransaction(2, common.BytesToAddress([]byte{0x22}), big.NewInt(222), 2222, big.NewInt(22222), []byte{0x22, 0x22, 0x22})
	mytx2, _ := types.SignTx(tx2, signer, key)
	tx3 := types.NewTransaction(3, common.BytesToAddress([]byte{0x33}), big.NewInt(333), 3333, big.NewInt(33333), []byte{0x33, 0x33, 0x33})
	mytx3, _ := types.SignTx(tx3, signer, key)
	txs := []*types.Transaction{mytx, mytx2, mytx3}

	receipt := &types.Receipt{
		Status:            types.ReceiptStatusSuccessful,
		CumulativeGasUsed: 1,
		Logs: []*types.Log{
			{Address: common.BytesToAddress([]byte{0x11})},
			{Address: common.BytesToAddress([]byte{0x01, 0x11})},
		},
		TxHash:          common.BytesToHash([]byte{0x11, 0x11}),
		ContractAddress: common.BytesToAddress([]byte{0x01, 0x11, 0x11}),
		GasUsed:         111111,
	}

	receipts := []*types.Receipt{receipt}
	block := types.NewBlock(&types.Header{Number: big.NewInt(315)}, txs, nil, receipts)

	// Write and verify the block in the database
	if err := WriteBlock(block, receipts); err != nil {
		t.Fatalf("Failed to write block into database: %v", err)
	}

	sqldb, err := DBConnection()
	if err != nil {
		panic(err)
	}

	entry := GetBlock(sqldb, block.Number().String())
	byt := []byte(entry)
	var data SBlock
	json.Unmarshal(byt, &data)

	//TODO Difficulty, rewards, age
	if block.Hash().String() != data.Hash {
		t.Fatalf("Block Hash [%v]: Block hash not found", block.Hash().String())
	}
	if block.Coinbase().String() != data.Coinbase {
		t.Fatalf("Block coinbase [%v]: Block coinbase not found", block.Coinbase().String())
	}
	if block.Number().String() != data.Number {
		t.Fatalf("Block number [%v]: Block number not found", block.Number().String())
	}
	if block.GasUsed() != data.GasUsed {
		t.Fatalf("Gas Used [%v]: Gas used not found", block.GasUsed())
	}
	if block.GasLimit() != data.GasLimit {
		t.Fatalf("Gas Limit [%v]: Gas limit not found", block.GasLimit())
	}
	if block.Transactions().Len() != data.TxCount {
		t.Fatalf("Tx Count [%v]: Tx Count not found", block.Transactions().Len())
	}
	if len(block.Uncles()) != data.UncleCount {
		t.Fatalf("Uncle count [%v]: Uncle count not found", len(block.Uncles()))
	}
	if block.ParentHash().String() != data.ParentHash {
		t.Fatalf("Parent hash [%v]: Parent hash not found", block.ParentHash().String())
	}
	if block.UncleHash().String() != data.UncleHash {
		t.Fatalf("Uncle hash [%v]: Uncle hash not found", block.UncleHash().String())
	}
	if block.Size().String() != data.Size {
		t.Fatalf("Size [%v]: Size not found", block.Size().String())
	}
	if block.Nonce() != data.Nonce {
		t.Fatalf("Block nonce [%v]: Block nonce not found", block.Nonce())
	}

	if getAllBlocks := GetAllBlocks(sqldb); len(getAllBlocks) == 0 {
		t.Fatalf("GetAllBlocks [%v]: GetAllBlocks did not return correctly", getAllBlocks)
	}

	if getAllBlocksMinedByAddress := GetAllBlocksMinedByAddress(sqldb, block.Coinbase().String()); len(getAllBlocksMinedByAddress) == 0 {
		t.Fatalf("GetAllBlocksMinedByAddress [%v]: GetAllBlocksMinedByAddress did not return correctly", getAllBlocksMinedByAddress)
	}

	ClearTables()
}

func TestGetRecentBlock(t *testing.T) {
	key, _ := crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	signer := types.NewEIP155Signer(big.NewInt(2147483647))

	//Nonce, To Address,Value, GasLimit, Gasprice, data
	tx1 := types.NewTransaction(1, common.BytesToAddress([]byte{0x11}), big.NewInt(111), 1111, big.NewInt(11111), []byte{0x11, 0x11, 0x11})
	mytx, _ := types.SignTx(tx1, signer, key)
	tx2 := types.NewTransaction(2, common.BytesToAddress([]byte{0x22}), big.NewInt(222), 2222, big.NewInt(22222), []byte{0x22, 0x22, 0x22})
	mytx2, _ := types.SignTx(tx2, signer, key)
	tx3 := types.NewTransaction(3, common.BytesToAddress([]byte{0x33}), big.NewInt(333), 3333, big.NewInt(33333), []byte{0x33, 0x33, 0x33})
	mytx3, _ := types.SignTx(tx3, signer, key)
	txs := []*types.Transaction{mytx, mytx2}
	txs1 := []*types.Transaction{mytx3}

	receipt1 := &types.Receipt{
		Status:            types.ReceiptStatusSuccessful,
		CumulativeGasUsed: 1,
		Logs: []*types.Log{
			{Address: common.BytesToAddress([]byte{0x11})},
			{Address: common.BytesToAddress([]byte{0x01, 0x11})},
		},
		TxHash:          common.BytesToHash([]byte{0x11, 0x11}),
		ContractAddress: common.BytesToAddress([]byte{0x01, 0x11, 0x11}),
		GasUsed:         111111,
	}

	receipts := []*types.Receipt{receipt1}
	block := types.NewBlock(&types.Header{Number: big.NewInt(322)}, txs, nil, receipts)
	block2 := types.NewBlock(&types.Header{Number: big.NewInt(320)}, txs1, nil, receipts)
	blocks := []*types.Block{block, block2}

	for _, bc := range blocks {
		// Write and verify the block in the database
		if err := WriteBlock(bc, receipts); err != nil {
			t.Fatalf("Failed to write block into database: %v", err)
		}
	}

	sqldb, err := DBConnection()
	if err != nil {
		panic(err)
	}

	response := GetRecentBlock(sqldb)
	byteRes := []byte(response)
	var recentBlock SBlock
	json.Unmarshal(byteRes, &recentBlock)

	if block.Hash().String() != recentBlock.Hash {
		t.Fatalf("Block Hash [%v]: Block hash not found", block.Hash().String())
	}
	if block.Coinbase().String() != recentBlock.Coinbase {
		t.Fatalf("Block coinbase [%v]: Block coinbase not found", block.Coinbase().String())
	}
	if block.Number().String() != recentBlock.Number {
		t.Fatalf("Block number [%v]: Block number not found", block.Number().String())
	}
	if block.GasUsed() != recentBlock.GasUsed {
		t.Fatalf("Gas Used [%v]: Gas used not found", block.GasUsed())
	}
	if block.GasLimit() != recentBlock.GasLimit {
		t.Fatalf("Gas Limit [%v]: Gas limit not found", block.GasLimit())
	}
	if block.Transactions().Len() != recentBlock.TxCount {
		t.Fatalf("Tx Count [%v]: Tx Count not found", block.Transactions().Len())
	}
	if len(block.Uncles()) != recentBlock.UncleCount {
		t.Fatalf("Uncle count [%v]: Uncle count not found", len(block.Uncles()))
	}
	if block.ParentHash().String() != recentBlock.ParentHash {
		t.Fatalf("Parent hash [%v]: Parent hash not found", block.ParentHash().String())
	}
	if block.UncleHash().String() != recentBlock.UncleHash {
		t.Fatalf("Uncle hash [%v]: Uncle hash not found", block.UncleHash().String())
	}
	if block.Size().String() != recentBlock.Size {
		t.Fatalf("Size [%v]: Size not found", block.Size().String())
	}
	if block.Nonce() != recentBlock.Nonce {
		t.Fatalf("Block nonce [%v]: Block nonce not found", block.Nonce())
	}

	if allTxsFromBlock := GetAllTransactionsFromBlock(sqldb, block2.Number().String()); len(allTxsFromBlock) == 0 {
		t.Fatalf("GetAllTransactionsFromBlock [%v]: GetAllTransactionsFromBlock did not return correctly", allTxsFromBlock)
	}
	ClearTables()
}

func TestContractCreationTx(t *testing.T) {
	key, _ := crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	signer := types.NewEIP155Signer(big.NewInt(2147483647))

	//Nonce,Value, GasLimit, Gasprice, data
	contractCreation := types.NewContractCreation(1, big.NewInt(111), 1111, big.NewInt(11111), []byte{0x11, 0x11, 0x11})
	mytx, _ := types.SignTx(contractCreation, signer, key)
	txs := []*types.Transaction{mytx}

	receipt2 := &types.Receipt{
		Status:            types.ReceiptStatusSuccessful,
		CumulativeGasUsed: 1,
		Logs: []*types.Log{
			{Address: common.BytesToAddress([]byte{0x11})},
			{Address: common.BytesToAddress([]byte{0x01, 0x11})},
		},
		TxHash:          common.BytesToHash([]byte{0x11, 0x11}),
		ContractAddress: common.BytesToAddress([]byte{0x01, 0x11, 0x11}),
		GasUsed:         111111,
	}

	receipts := []*types.Receipt{receipt2}
	block := types.NewBlock(&types.Header{Number: big.NewInt(314)}, txs, nil, receipts)

	if err := WriteBlock(block, receipts); err != nil {
		t.Fatalf("Failed to write block into database: %v", err)
	}

	var contractAddressFromReciept string
	for _, receipt := range receipts {
		contractAddressFromReciept = (*types.ReceiptForStorage)(receipt).ContractAddress.String()
	}

	sqldb, err := DBConnection()
	if err != nil {
		panic(err)
	}

	for _, tx := range txs {
		txn := GetTransaction(sqldb, tx.Hash().String())
		byt := []byte(txn)
		var data ShyftTxEntryPretty
		json.Unmarshal(byt, &data)

		if tx.Hash().String() != data.TxHash {
			t.Fatalf("txHash [%v]: tx Hash not found", tx.Hash().String())
		}
		if contractAddressFromReciept != data.To {
			t.Fatalf("Contract Addr [%v]: Contract addr not found", contractAddressFromReciept)
		}
		if tx.From().String() != data.From {
			t.Fatalf("From Addr [%v]: From addr not found", tx.From().String())
		}
		if tx.Nonce() != data.Nonce {
			t.Fatalf("Nonce [%v]: Nonce not found", tx.Nonce())
		}
		if tx.Gas() != data.Gas {
			t.Fatalf("Gas [%v]: Gas not found", tx.Gas())
		}
		if tx.GasPrice().Uint64() != data.GasPrice {
			t.Fatalf("Gas Price [%v]: Gas price not found", tx.GasPrice().String())
		}
		if block.GasLimit() != data.GasLimit {
			t.Fatalf("Gas Limit [%v]: Gas limit not found", block.GasLimit())
		}
		if block.Hash().String() != data.BlockHash {
			t.Fatalf("Block Hash [%v]: Block hash not found", block.Hash().String())
		}
		if block.Number().String() != data.BlockNumber {
			t.Fatalf("Block Number [%v]: Block number not found", block.Number().String())
		}
		if tx.Value().String() != data.Amount {
			t.Fatalf("Amount [%v]: Amount not found", tx.Value().String())
		}
		if tx.Cost().Uint64() != data.Cost {
			t.Fatalf("Cost [%v]: Cost not found", tx.Cost().String())
		}
		var status string
		if receipt2.Status == 1 {
			status = "SUCCESS"
		}
		if receipt2.Status == 0 {
			status = "FAIL"
		}
		if status != data.Status {
			t.Fatalf("Receipt status [%v]: Receipt status not found", status)
		}
		var isContract bool
		if tx.To() != nil {
			isContract = false
		} else {
			isContract = true
		}
		if isContract != data.IsContract {
			t.Fatalf("isContract [%v]: isContract bool is incorrect", isContract)
		}
	}
	ClearTables()
}

func TestTransactionsToReturnTransactions(t *testing.T) {
	key, _ := crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	signer := types.NewEIP155Signer(big.NewInt(2147483647))

	//Nonce, To Address,Value, GasLimit, Gasprice, data
	tx1 := types.NewTransaction(1, common.BytesToAddress([]byte{0x11}), big.NewInt(111), 1111, big.NewInt(11111), []byte{0x11, 0x11, 0x11})
	mytx, _ := types.SignTx(tx1, signer, key)
	tx2 := types.NewTransaction(2, common.BytesToAddress([]byte{0x22}), big.NewInt(222), 2222, big.NewInt(22222), []byte{0x22, 0x22, 0x22})
	mytx2, _ := types.SignTx(tx2, signer, key)
	tx3 := types.NewTransaction(3, common.BytesToAddress([]byte{0x33}), big.NewInt(333), 3333, big.NewInt(33333), []byte{0x33, 0x33, 0x33})
	mytx3, _ := types.SignTx(tx3, signer, key)
	txs := []*types.Transaction{mytx, mytx2, mytx3}

	receipt1 := &types.Receipt{
		Status:            types.ReceiptStatusSuccessful,
		CumulativeGasUsed: 1,
		Logs: []*types.Log{
			{Address: common.BytesToAddress([]byte{0x11})},
			{Address: common.BytesToAddress([]byte{0x01, 0x11})},
		},
		TxHash:          common.BytesToHash([]byte{0x11, 0x11}),
		ContractAddress: common.BytesToAddress([]byte{0x01, 0x11, 0x11}),
		GasUsed:         111111,
	}

	receipts := []*types.Receipt{receipt1}
	block := types.NewBlock(&types.Header{Number: big.NewInt(314)}, txs, nil, receipts)

	if err := WriteBlock(block, receipts); err != nil {
		t.Fatalf("Failed to write block into database: %v", err)
	}
	sqldb, err := DBConnection()
	if err != nil {
		panic(err)
	}

	for _, tx := range txs {
		txn := GetTransaction(sqldb, tx.Hash().String())
		byt := []byte(txn)
		var data ShyftTxEntryPretty
		json.Unmarshal(byt, &data)

		//TODO age, data
		if tx.Hash().String() != data.TxHash {
			t.Fatalf("txHash [%v]: tx Hash not found", tx.Hash().String())
		}
		if tx.From().String() != data.From {
			t.Fatalf("From Addr [%v]: From addr not found", tx.From().String())
		}
		if tx.To().String() != data.To {
			t.Fatalf("To Addr [%v]: To addr not found", tx.To().String())
		}
		if tx.Nonce() != data.Nonce {
			t.Fatalf("Nonce [%v]: Nonce not found", tx.Nonce())
		}
		if tx.Gas() != data.Gas {
			t.Fatalf("Gas [%v]: Gas not found", tx.Gas())
		}
		if tx.GasPrice().Uint64() != data.GasPrice {
			t.Fatalf("Gas Price [%v]: Gas price not found", tx.GasPrice().String())
		}
		if block.GasLimit() != data.GasLimit {
			t.Fatalf("Gas Limit [%v]: Gas limit not found", block.GasLimit())
		}
		if block.Hash().String() != data.BlockHash {
			t.Fatalf("Block Hash [%v]: Block hash not found", block.Hash().String())
		}
		if block.Number().String() != data.BlockNumber {
			t.Fatalf("Block Number [%v]: Block number not found", block.Number().String())
		}
		if tx.Value().String() != data.Amount {
			t.Fatalf("Amount [%v]: Amount not found", tx.Value().String())
		}
		if tx.Cost().Uint64() != data.Cost {
			t.Fatalf("Cost [%v]: Cost not found", tx.Cost().String())
		}
		var status string
		if receipt1.Status == 1 {
			status = "SUCCESS"
		}
		if receipt1.Status == 0 {
			status = "FAIL"
		}
		if status != data.Status {
			t.Fatalf("Receipt status [%v]: Receipt status not found", status)
		}
		var isContract bool
		if tx.To() != nil {
			isContract = false
		} else {
			isContract = true
		}
		if isContract != data.IsContract {
			t.Fatalf("isContract [%v]: isContract bool is incorrect", isContract)
		}
	}

	if getAllTx := GetAllTransactions(sqldb); len(getAllTx) == 0 {
		t.Fatalf("GetAllTransactions [%v]: GetAllTransactions did not return correctly", getAllTx)
	}
	ClearTables()
}

func TestAccountsToReturnAccounts(t *testing.T) {
	key, _ := crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	signer := types.NewEIP155Signer(big.NewInt(2147483647))

	//Nonce, To Address,Value, GasLimit, Gasprice, data
	tx1 := types.NewTransaction(1, common.BytesToAddress([]byte{0x11}), big.NewInt(111), 1111, big.NewInt(11111), []byte{0x11, 0x11, 0x11})
	mytx, _ := types.SignTx(tx1, signer, key)
	tx2 := types.NewTransaction(2, common.BytesToAddress([]byte{0x22}), big.NewInt(222), 2222, big.NewInt(22222), []byte{0x22, 0x22, 0x22})
	mytx2, _ := types.SignTx(tx2, signer, key)
	tx3 := types.NewTransaction(3, common.BytesToAddress([]byte{0x33}), big.NewInt(333), 3333, big.NewInt(33333), []byte{0x33, 0x33, 0x33})
	mytx3, _ := types.SignTx(tx3, signer, key)
	txs := []*types.Transaction{mytx, mytx2, mytx3}

	receipt1 := &types.Receipt{
		Status:            types.ReceiptStatusSuccessful,
		CumulativeGasUsed: 1,
		Logs: []*types.Log{
			{Address: common.BytesToAddress([]byte{0x11})},
			{Address: common.BytesToAddress([]byte{0x01, 0x11})},
		},
		TxHash:          common.BytesToHash([]byte{0x11, 0x11}),
		ContractAddress: common.BytesToAddress([]byte{0x01, 0x11, 0x11}),
		GasUsed:         111111,
	}

	receipts := []*types.Receipt{receipt1}
	block := types.NewBlock(&types.Header{Number: big.NewInt(315)}, txs, nil, receipts)
	if err := WriteBlock(block, receipts); err != nil {
		t.Fatalf("Failed to write block into database: %v", err)
	}

	sqldb, err := DBConnection()
	if err != nil {
		panic(err)
	}

	for _, tx := range txs {
		accountAddrTo := GetAccount(sqldb, tx.To().String())
		byts := []byte(accountAddrTo)
		var accountDataTo SAccounts
		json.Unmarshal(byts, &accountDataTo)

		if tx.To().String() != accountDataTo.Addr {
			t.Fatalf("To address [%v]: To address not found", accountDataTo.Addr)
		}
		if tx.Value().String() != accountDataTo.Balance {
			t.Fatalf("To address balance [%v]: To address balance not found", accountDataTo.Balance)
		}
		if strconv.FormatUint(tx.Nonce(), 10) != accountDataTo.TxCountAccount {
			t.Fatalf("To account nonce [%v]: To account nonce not found", accountDataTo.TxCountAccount)
		}
		if getAllAccountTxs := GetAccountTxs(sqldb, tx.To().String()); len(getAllAccountTxs) == 0 {
			t.Fatalf("GetAccountTxs [%v]: GetAccountTxs did not return correctly", getAllAccountTxs)
		}
	}

	if getAllAccounts := GetAllAccounts(sqldb); len(getAllAccounts) == 0 {
		t.Fatalf("GetAllAccounts [%v]: GetAllAccounts did not return correctly", getAllAccounts)
	}
	ClearTables()
}
