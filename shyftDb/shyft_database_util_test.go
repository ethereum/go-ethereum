package shyftdb

import (
	"encoding/json"
	"math/big"
	"strconv"
	"strings"
	"testing"

	"github.com/ShyftNetwork/go-empyrean/common"
	"github.com/ShyftNetwork/go-empyrean/consensus/ethash"
	"github.com/ShyftNetwork/go-empyrean/core"
	"github.com/ShyftNetwork/go-empyrean/core/types"
	"github.com/ShyftNetwork/go-empyrean/crypto"
	"github.com/ShyftNetwork/go-empyrean/eth"
)

type ShyftTracer struct{}

const (
	testAddress = "0x8605cdbbdb6d264aa742e77020dcbc58fcdce182"
)

func TestBlock(t *testing.T) {
	//SET UP FOR TEST FUNCTIONS
	eth.NewShyftTestLDB()
	core.InitDBTest()
	shyftTracer := new(eth.ShyftTracer)
	core.SetIShyftTracer(shyftTracer)

	ethConf := &eth.Config{
		Genesis:   core.DeveloperGenesisBlock(15, common.Address{}),
		Etherbase: common.HexToAddress(testAddress),
		Ethash: ethash.Config{
			PowMode: ethash.ModeTest,
		},
	}

	eth.SetGlobalConfig(ethConf)

	eth.InitTracerEnv()
	core.ClearTables()

	key, _ := crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	signer := types.NewEIP155Signer(big.NewInt(2147483647))

	//Nonce, To Address,Value, GasLimit, Gasprice, data
	tx1 := types.NewTransaction(1, common.BytesToAddress([]byte{0x11}), big.NewInt(5), 1111, big.NewInt(11111), []byte{0x11, 0x11, 0x11})
	mytx1, _ := types.SignTx(tx1, signer, key)
	tx2 := types.NewTransaction(2, common.BytesToAddress([]byte{0x22}), big.NewInt(5), 2222, big.NewInt(22222), []byte{0x22, 0x22, 0x22})
	mytx2, _ := types.SignTx(tx2, signer, key)
	tx3 := types.NewTransaction(3, common.BytesToAddress([]byte{0x33}), big.NewInt(5), 3333, big.NewInt(33333), []byte{0x33, 0x33, 0x33})
	mytx3, _ := types.SignTx(tx3, signer, key)
	txs := []*types.Transaction{mytx1, mytx2}
	txs1 := []*types.Transaction{mytx3}

	//Nonce,Value, GasLimit, Gasprice, data
	contractCreation := types.NewContractCreation(1, big.NewInt(111), 1111, big.NewInt(11111), []byte{0x11, 0x11, 0x11})
	mytx4, _ := types.SignTx(contractCreation, signer, key)
	txs2 := []*types.Transaction{mytx4}

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

	block1 := types.NewBlock(&types.Header{Number: big.NewInt(323)}, txs, nil, receipts)
	block2 := types.NewBlock(&types.Header{Number: big.NewInt(320)}, txs1, nil, receipts)
	block3 := types.NewBlock(&types.Header{Number: big.NewInt(322)}, txs2, nil, receipts)
	blocks := []*types.Block{block1, block2, block3}

	sqldb, err := core.DBConnection()
	if err != nil {
		panic(err)
	}

	fromAddr := "0x71562b71999873db5b286df957af199ec94617f7"
	fromAddrEndBalance := "75"
	fromAddrEndNonce := "5"
	toAddr := common.BytesToAddress([]byte{0x11})
	core.CreateAccount(sqldb, fromAddr, "201", "1")

	t.Run("TestBlockToReturnBlock", func(t *testing.T) {
		for _, bc := range blocks {
			// Write and verify the block in the database
			if err := core.SWriteBlock(bc, receipts); err != nil {
				t.Fatalf("Failed to write block into database: %v", err)
			}
		}

		entry := core.SGetBlock(sqldb, block1.Number().String())
		byt := []byte(entry)
		var data core.SBlock
		json.Unmarshal(byt, &data)

		//TODO Difficulty, rewards, age
		if block1.Hash().String() != data.Hash {
			t.Fatalf("Block Hash [%v]: Block hash not found", block1.Hash().String())
		}
		if block1.Coinbase().String() != data.Coinbase {
			t.Fatalf("Block coinbase [%v]: Block coinbase not found", block1.Coinbase().String())
		}
		if block1.Number().String() != data.Number {
			t.Fatalf("Block number [%v]: Block number not found", block1.Number().String())
		}
		if block1.GasUsed() != data.GasUsed {
			t.Fatalf("Gas Used [%v]: Gas used not found", block1.GasUsed())
		}
		if block1.GasLimit() != data.GasLimit {
			t.Fatalf("Gas Limit [%v]: Gas limit not found", block1.GasLimit())
		}
		if block1.Transactions().Len() != data.TxCount {
			t.Fatalf("Tx Count [%v]: Tx Count not found", block1.Transactions().Len())
		}
		if len(block1.Uncles()) != data.UncleCount {
			t.Fatalf("Uncle count [%v]: Uncle count not found", len(block1.Uncles()))
		}
		if block1.ParentHash().String() != data.ParentHash {
			t.Fatalf("Parent hash [%v]: Parent hash not found", block1.ParentHash().String())
		}
		if block1.UncleHash().String() != data.UncleHash {
			t.Fatalf("Uncle hash [%v]: Uncle hash not found", block1.UncleHash().String())
		}
		if block1.Size().String() != data.Size {
			t.Fatalf("Size [%v]: Size not found", block1.Size().String())
		}
		if block1.Nonce() != data.Nonce {
			t.Fatalf("Block nonce [%v]: Block nonce not found", block1.Nonce())
		}

		if getAllBlocks := core.SGetAllBlocks(sqldb); len(getAllBlocks) == 0 {
			t.Fatalf("GetAllBlocks [%v]: GetAllBlocks did not return correctly", getAllBlocks)
		}

		if getAllBlocksMinedByAddress := core.SGetAllBlocksMinedByAddress(sqldb, block1.Coinbase().String()); len(getAllBlocksMinedByAddress) == 0 {
			t.Fatalf("GetAllBlocksMinedByAddress [%v]: GetAllBlocksMinedByAddress did not return correctly", getAllBlocksMinedByAddress)
		}
	})

	t.Run("TestGetRecentBlock", func(t *testing.T) {
		response := core.SGetRecentBlock(sqldb)
		byteRes := []byte(response)
		var recentBlock core.SBlock
		json.Unmarshal(byteRes, &recentBlock)

		if block1.Hash().String() != recentBlock.Hash {
			t.Fatalf("Block Hash [%v]: Block hash not found", block1.Hash().String())
		}
		if block1.Coinbase().String() != recentBlock.Coinbase {
			t.Fatalf("Block coinbase [%v]: Block coinbase not found", block1.Coinbase().String())
		}
		if block1.Number().String() != recentBlock.Number {
			t.Fatalf("Block number [%v]: Block number not found", block1.Number().String())
		}
		if block1.GasUsed() != recentBlock.GasUsed {
			t.Fatalf("Gas Used [%v]: Gas used not found", block1.GasUsed())
		}
		if block1.GasLimit() != recentBlock.GasLimit {
			t.Fatalf("Gas Limit [%v]: Gas limit not found", block1.GasLimit())
		}
		if block1.Transactions().Len() != recentBlock.TxCount {
			t.Fatalf("Tx Count [%v]: Tx Count not found", block1.Transactions().Len())
		}
		if len(block1.Uncles()) != recentBlock.UncleCount {
			t.Fatalf("Uncle count [%v]: Uncle count not found", len(block1.Uncles()))
		}
		if block1.ParentHash().String() != recentBlock.ParentHash {
			t.Fatalf("Parent hash [%v]: Parent hash not found", block1.ParentHash().String())
		}
		if block1.UncleHash().String() != recentBlock.UncleHash {
			t.Fatalf("Uncle hash [%v]: Uncle hash not found", block1.UncleHash().String())
		}
		if block1.Size().String() != recentBlock.Size {
			t.Fatalf("Size [%v]: Size not found", block1.Size().String())
		}
		if block1.Nonce() != recentBlock.Nonce {
			t.Fatalf("Block nonce [%v]: Block nonce not found", block1.Nonce())
		}

		if allTxsFromBlock := core.SGetAllTransactionsFromBlock(sqldb, block2.Number().String()); len(allTxsFromBlock) == 0 {
			t.Fatalf("GetAllTransactionsFromBlock [%v]: GetAllTransactionsFromBlock did not return correctly", allTxsFromBlock)
		}
	})

	t.Run("TestContractCreationTx", func(t *testing.T) {
		var contractAddressFromReciept string
		for _, receipt := range receipts {
			contractAddressFromReciept = (*types.ReceiptForStorage)(receipt).ContractAddress.String()
		}

		for _, tx := range txs2 {
			txn := core.SGetTransaction(sqldb, tx.Hash().String())
			byt := []byte(txn)
			var data core.ShyftTxEntryPretty
			json.Unmarshal(byt, &data)

			if tx.Hash().String() != data.TxHash {
				t.Fatalf("txHash [%v]: tx Hash not found", tx.Hash().String())
			}
			if contractAddressFromReciept != data.ToGet {
				t.Fatalf("Contract Addr [%v]: Contract addr not found", contractAddressFromReciept)
			}
			if strings.ToLower(tx.From().String()) != data.From {
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
			if block1.GasLimit() != data.GasLimit {
				t.Fatalf("Gas Limit [%v]: Gas limit not found", block1.GasLimit())
			}
			if block3.Hash().String() != data.BlockHash {
				t.Fatalf("Block Hash [%v]: Block hash not found", block1.Hash().String())
			}
			if block3.Number().String() != data.BlockNumber {
				t.Fatalf("Block Number [%v]: Block number not found", block1.Number().String())
			}
			if tx.Value().String() != data.Amount {
				t.Fatalf("Amount [%v]: Amount not found", tx.Value().String())
			}
			if tx.Cost().Uint64() != data.Cost {
				t.Fatalf("Cost [%v]: Cost not found", tx.Cost().String())
			}
			var status string
			if receipt.Status == 1 {
				status = "SUCCESS"
			}
			if receipt.Status == 0 {
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
	})

	t.Run("TestTransactionsToReturnTransactions", func(t *testing.T) {
		for _, tx := range txs {
			txn := core.SGetTransaction(sqldb, tx.Hash().String())
			byt := []byte(txn)
			var data core.ShyftTxEntryPretty
			json.Unmarshal(byt, &data)

			//TODO age, data
			if strings.ToLower(tx.Hash().String()) != data.TxHash {
				t.Fatalf("txHash [%v]: tx Hash not found", tx.Hash().String())
			}
			if strings.ToLower(tx.From().String()) != data.From {
				t.Fatalf("From Addr [%v]: From addr not found", tx.From().String())
			}
			if strings.ToLower(tx.To().String()) != data.ToGet {
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
			if block1.GasLimit() != data.GasLimit {
				t.Fatalf("Gas Limit [%v]: Gas limit not found", block1.GasLimit())
			}
			if block1.Hash().String() != data.BlockHash {
				t.Fatalf("Block Hash [%v]: Block hash not found", block1.Hash().String())
			}
			if block1.Number().String() != data.BlockNumber {
				t.Fatalf("Block Number [%v]: Block number not found", block1.Number().String())
			}
			if tx.Value().String() != data.Amount {
				t.Fatalf("Amount [%v]: Amount not found", tx.Value().String())
			}
			if tx.Cost().Uint64() != data.Cost {
				t.Fatalf("Cost [%v]: Cost not found", tx.Cost().String())
			}
			var status string
			if receipt.Status == 1 {
				status = "SUCCESS"
			}
			if receipt.Status == 0 {
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

		if getAllTx := core.SGetAllTransactions(sqldb); len(getAllTx) == 0 {
			t.Fatalf("GetAllTransactions [%v]: GetAllTransactions did not return correctly", getAllTx)
		}
	})
	t.Run("TestAccountsToReturnAccounts", func(t *testing.T) {
		for _, tx := range txs {
			accountAddrTo := core.SGetAccount(sqldb, tx.To().String())
			byts := []byte(accountAddrTo)
			var accountDataTo core.SAccounts
			json.Unmarshal(byts, &accountDataTo)

			if strings.ToLower(tx.To().String()) != accountDataTo.Addr {
				t.Fatalf("To address [%v]: To address not found", accountDataTo.Addr)
			}
			if tx.Value().String() != accountDataTo.Balance {
				t.Fatalf("To address balance [%v]: To address balance not found", accountDataTo.Balance)
			}
			if strconv.FormatUint(tx.Nonce(), 10) != accountDataTo.AccountNonce {
				t.Fatalf("To account nonce [%v]: To account nonce not found", accountDataTo.AccountNonce)
			}
		}
		accountAddrFrom := core.SGetAccount(sqldb, fromAddr)
		byts := []byte(accountAddrFrom)
		var accountDataFrom core.SAccounts
		json.Unmarshal(byts, &accountDataFrom)

		if fromAddr != accountDataFrom.Addr {
			t.Fatalf("To address [%v]: To address not found", accountDataFrom.Addr)
		}
		if fromAddrEndBalance != accountDataFrom.Balance {
			t.Fatalf("To address balance [%v]: To address balance not found", accountDataFrom.Balance)
		}
		if fromAddrEndNonce != accountDataFrom.AccountNonce {
			t.Fatalf("To account nonce [%v]: To account nonce not found", accountDataFrom.AccountNonce)
		}
		if getAllAccountTxs := core.SGetAccountTxs(sqldb, toAddr.String()); len(getAllAccountTxs) == 0 {
			t.Fatalf("GetAccountTxs [%v]: GetAccountTxs did not return correctly", getAllAccountTxs)
		}
		if getAllAccounts := core.SGetAllAccounts(sqldb); len(getAllAccounts) == 0 {
			t.Fatalf("GetAllAccounts [%v]: GetAllAccounts did not return correctly", getAllAccounts)
		}
	})
}

//Genesis.go functions
// WriteBlockZero
// WriteShyftGen
