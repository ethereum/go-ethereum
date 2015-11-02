package les

import (
	"bytes"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/access"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
)

var (
	acc1Key, _ = crypto.HexToECDSA("8a1f9a8f95be41cd7ccb6168179afb4504aefe388d1e14474d32c45c72ce7b7a")
	acc2Key, _ = crypto.HexToECDSA("49a7b37aa6f6645917e7b807e9d1c00d4fa71f18343b0d4122a4d2df64dd6fee")
	acc1Addr = crypto.PubkeyToAddress(acc1Key.PublicKey)
	acc2Addr = crypto.PubkeyToAddress(acc2Key.PublicKey)

	testContractCode = common.Hex2Bytes("606060405260cc8060106000396000f360606040526000357c01000000000000000000000000000000000000000000000000000000009004806360cd2685146041578063c16431b914606b57603f565b005b6055600480803590602001909190505060a9565b6040518082815260200191505060405180910390f35b60886004808035906020019091908035906020019091905050608a565b005b80600060005083606481101560025790900160005b50819055505b5050565b6000600060005082606481101560025790900160005b5054905060c7565b91905056")
	testContractAddr	common.Address
)
/*
contract test {

    uint256[100] data;

    function Put(uint256 addr, uint256 value) {
        data[addr] = value;
    }

    function Get(uint256 addr) constant returns (uint256 value) {
        return data[addr];
    }
}
*/

type odrTestFn func(ca *access.ChainAccess, bc *core.BlockChain, ctx *access.OdrContext, bhash common.Hash) []byte

func TestOdrGetBlockLes1(t *testing.T) { testOdr(t, 1, 1, odrGetBlock) }

func odrGetBlock(ca *access.ChainAccess, bc *core.BlockChain, ctx *access.OdrContext, bhash common.Hash) []byte {
	block := bc.GetBlock(bhash, ctx)
	if block == nil {
		return nil
	}
	rlp, _ := rlp.EncodeToBytes(block)
	return rlp
}

func TestOdrGetReceiptsLes1(t *testing.T) { testOdr(t, 1, 1, odrGetReceipts) }

func odrGetReceipts(ca *access.ChainAccess, bc *core.BlockChain, ctx *access.OdrContext, bhash common.Hash) []byte {
	receipts := core.GetBlockReceipts(ca, bhash, ctx)
	if receipts == nil {
		return nil
	}
	rlp, _ := rlp.EncodeToBytes(receipts)
	return rlp
}

func TestOdrAccountsLes1(t *testing.T) { testOdr(t, 1, 1, odrAccounts) }

func odrAccounts(ca *access.ChainAccess, bc *core.BlockChain, ctx *access.OdrContext, bhash common.Hash) []byte {
	dummyAddr := common.HexToAddress("1234567812345678123456781234567812345678")
	acc := []common.Address{ testBankAddress, acc1Addr, acc2Addr, dummyAddr}

	trie.ClearGlobalCache()
	
	var res []byte
	for _, addr := range acc {
		header := bc.GetHeader(bhash)
		st, err := state.New(header.Root, ca, ctx)
		if err == nil {
			bal := st.GetBalance(addr)
			rlp, _ := rlp.EncodeToBytes(bal)
			res = append(res, rlp...)
		}
	}
	
	return res
}

func TestOdrContractCallLes1(t *testing.T) { testOdr(t, 1, 2, odrContractCall) }

// callmsg is the message type used for call transations.
type callmsg struct {
	from          *state.StateObject
	to            *common.Address
	gas, gasPrice *big.Int
	value         *big.Int
	data          []byte
}

// accessor boilerplate to implement core.Message
func (m callmsg) From() (common.Address, error) { return m.from.Address(), nil }
func (m callmsg) Nonce() uint64                 { return m.from.Nonce() }
func (m callmsg) To() *common.Address           { return m.to }
func (m callmsg) GasPrice() *big.Int            { return m.gasPrice }
func (m callmsg) Gas() *big.Int                 { return m.gas }
func (m callmsg) Value() *big.Int               { return m.value }
func (m callmsg) Data() []byte                  { return m.data }

func odrContractCall(ca *access.ChainAccess, bc *core.BlockChain, ctx *access.OdrContext, bhash common.Hash) []byte {
	data := common.Hex2Bytes("60CD26850000000000000000000000000000000000000000000000000000000000000000")

	var res []byte
	for i:=0; i<3; i++ {
		data[35] = byte(i)
		header := bc.GetHeader(bhash)
		statedb, err := state.New(header.Root, ca, ctx)
		if err == nil {
			from := statedb.GetOrNewStateObject(testBankAddress)
			from.SetBalance(common.MaxBig)

			msg := callmsg{
				from:     from,
				gas:      big.NewInt(100000),
				gasPrice: big.NewInt(0),
				value:    big.NewInt(0),
				data:     data,
				to:		  &testContractAddr,
			}

			vmenv := core.NewEnv(statedb, bc, msg, header)
			gp := new(core.GasPool).AddGas(common.MaxBig)
			ret, _, _ := core.ApplyMessage(vmenv, msg, gp)
			res = append(res, ret...)
		}
	}
	return res
}

func testOdr(t *testing.T, protocol int, expFail uint64, fn odrTestFn) {
	// Create a chain generator with some simple transactions (blatantly stolen from @fjl/chain_makerts_test)
	generator := func(i int, block *core.BlockGen) {
		switch i {
		case 0:
			// In block 1, the test bank sends account #1 some ether.
			tx, _ := types.NewTransaction(block.TxNonce(testBankAddress), acc1Addr, big.NewInt(10000), params.TxGas, nil, nil).SignECDSA(testBankKey)
			block.AddTx(tx)
		case 1:
			// In block 2, the test bank sends some more ether to account #1.
			// acc1Addr passes it on to account #2.
			// acc1Addr creates a test contract.
			tx1, _ := types.NewTransaction(block.TxNonce(testBankAddress), acc1Addr, big.NewInt(1000), params.TxGas, nil, nil).SignECDSA(testBankKey)
			nonce := block.TxNonce(acc1Addr)
			tx2, _ := types.NewTransaction(nonce, acc2Addr, big.NewInt(1000), params.TxGas, nil, nil).SignECDSA(acc1Key)
			nonce++
			tx3, _ := types.NewContractCreation(nonce, big.NewInt(0), big.NewInt(100000), big.NewInt(0), testContractCode).SignECDSA(acc1Key)
			testContractAddr = crypto.CreateAddress(acc1Addr, nonce)
			block.AddTx(tx1)
			block.AddTx(tx2)
			block.AddTx(tx3)
		case 2:
			// Block 3 is empty but was mined by account #2.
			block.SetCoinbase(acc2Addr)
			block.SetExtra([]byte("yeehaw"))
			data := common.Hex2Bytes("C16431B900000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000001")
			tx, _ := types.NewTransaction(block.TxNonce(testBankAddress), testContractAddr, big.NewInt(0), big.NewInt(100000), nil, data).SignECDSA(testBankKey)
			block.AddTx(tx)
		case 3:
			// Block 4 includes blocks 2 and 3 as uncle headers (with modified extra data).
			b2 := block.PrevBlock(1).Header()
			b2.Extra = []byte("foo")
			block.AddUncle(b2)
			b3 := block.PrevBlock(2).Header()
			b3.Extra = []byte("foo")
			block.AddUncle(b3)
			data := common.Hex2Bytes("C16431B900000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000002")
			tx, _ := types.NewTransaction(block.TxNonce(testBankAddress), testContractAddr, big.NewInt(0), big.NewInt(100000), nil, data).SignECDSA(testBankKey)
			block.AddTx(tx)
		}
	}
	// Assemble the test environment
	pm, ca := newTestProtocolManagerMust(t, false, 4, generator)
	lpm, lca := newTestProtocolManagerMust(t, true, 0, nil)
	_, _, lpeer, _ := newTestPeerPair("peer", protocol, pm, lpm)
	time.Sleep(time.Millisecond * 100)
	lpm.synchronise(lpeer)

	cid := access.NewChannelID(time.Millisecond * 200)

	test := func(expFail uint64) {
		for i := uint64(0); i <= pm.blockchain.CurrentBlock().NumberU64(); i++ {
			bhash := core.GetCanonicalHash(ca.Db(), i)
			b1 := fn(ca, pm.blockchain, access.NoOdr, bhash)
			b2 := fn(lca, lpm.blockchain, access.NewContext(cid), bhash)
			eq := bytes.Equal(b1, b2)
			exp := i < expFail
			if exp && !eq {		
				t.Errorf("odr mismatch")
			}
			if !exp && eq {		
				t.Errorf("unexpected odr match")
			}
		}
	}

	// temporarily remove peer to test odr fails
	lca.UnregisterPeer(lpeer.id)
	// expect retrievals to fail (except genesis block) without a les peer
	test(expFail)
	lca.RegisterPeer(lpeer.id, lpeer.version, lpeer.Head(), lpeer.RequestBodies, lpeer.RequestNodeData, lpeer.RequestReceipts, lpeer.RequestProofs)
	// expect all retrievals to pass
	test(5)
	lca.UnregisterPeer(lpeer.id)
	// still expect all retrievals to pass, now data should be cached locally
	test(5)
}