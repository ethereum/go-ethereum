package les

import (
	"bytes"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/les/access"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
)

var (
	testContractCode = common.Hex2Bytes("606060405260cc8060106000396000f360606040526000357c01000000000000000000000000000000000000000000000000000000009004806360cd2685146041578063c16431b914606b57603f565b005b6055600480803590602001909190505060a9565b6040518082815260200191505060405180910390f35b60886004808035906020019091908035906020019091905050608a565b005b80600060005083606481101560025790900160005b50819055505b5050565b6000600060005082606481101560025790900160005b5054905060c7565b91905056")
	testContractAddr common.Address
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
	acc := []common.Address{testBankAddress, acc1Addr, acc2Addr, dummyAddr}

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
	for i := 0; i < 3; i++ {
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
				to:       &testContractAddr,
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
	// Assemble the test environment
	pm, ca := newTestProtocolManagerMust(t, false, 4, testChainGen)
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
