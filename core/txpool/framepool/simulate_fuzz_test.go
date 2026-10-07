// Copyright 2026 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package framepool

import (
	"crypto/ecdsa"
	"encoding/binary"
	"fmt"
	"maps"
	"math"
	"math/big"
	"math/rand/v2"
	"slices"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/core/vm/program"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

// The fuzzers below decode a frame transaction and a head state from the raw
// input and check the properties the pool relies on for every accepted prefix:
//
//   - dependency completeness: changing any state outside the recorded
//     dependency keys leaves the simulation result unchanged;
//   - the balance-only premise: changing balances without flipping EIP-161
//     emptiness, keeping the payer solvent, changes only the payer balance;
//   - soundness: full block execution at the same head accepts the transaction.
//
// Generated code is biased towards the trace-rule edges: sender storage, helper
// calls of every kind, EXTCODE* of precompiles, empty and delegated accounts,
// GAS before calls, introspection, banned opcodes and APPROVE scopes. Every
// program starts with a fixed prologue so statements can branch on state.

// Prologue jump destinations shared by all generated programs.
const (
	fuzzLabelFail    = 3 // REVERT
	fuzzLabelStop    = 7 // STOP, success without approval
	fuzzLabelApprove = 9 // APPROVE with the program's scope
)

const fuzzMaxStatements = 6

var (
	fuzzContractSender = common.HexToAddress("0x5e5e000000000000000000000000000000000001")
	fuzzPaymaster      = common.HexToAddress("0x9a9a000000000000000000000000000000000001")
	fuzzFactory        = common.HexToAddress("0xfafa000000000000000000000000000000000001")
	fuzzDelegate       = common.HexToAddress("0xdede000000000000000000000000000000000001")
	fuzzUnused         = common.HexToAddress("0xeeee000000000000000000000000000000000001")
	fuzzHelpers        = []common.Address{
		common.HexToAddress("0x4e4e000000000000000000000000000000000001"),
		common.HexToAddress("0x4e4e000000000000000000000000000000000002"),
		common.HexToAddress("0x4e4e000000000000000000000000000000000003"),
		common.HexToAddress("0x4e4e000000000000000000000000000000000004"),
	}
	// A mix of old and recent precompiles; each may or may not exist.
	fuzzPrecompiles = []common.Address{
		common.BytesToAddress([]byte{0x01}),
		common.BytesToAddress([]byte{0x02}),
		common.BytesToAddress([]byte{0x04}),
		common.BytesToAddress([]byte{0x0b}),
		common.BytesToAddress([]byte{0x01, 0x00}),
	}
	fuzzSlots = []common.Hash{
		{},
		common.BigToHash(big.NewInt(1)),
		common.BigToHash(big.NewInt(2)),
		common.HexToHash("0x42"),
		crypto.Keccak256Hash([]byte("framepool fuzz slot")),
	}
	fuzzEther = new(uint256.Int).Exp(uint256.NewInt(10), uint256.NewInt(18))
)

// fuzzInput hands out fuzzer-controlled choices. An exhausted input yields
// zero, so every choice lists its most permissive option first.
type fuzzInput struct{ data []byte }

func (in *fuzzInput) byte() byte {
	if len(in.data) == 0 {
		return 0
	}
	b := in.data[0]
	in.data = in.data[1:]
	return b
}

func (in *fuzzInput) intn(n int) int { return int(in.byte()) % n }

// oneIn is false on exhausted input.
func (in *fuzzInput) oneIn(n int) bool { return in.intn(n) == n-1 }

func (in *fuzzInput) bytes(n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = in.byte()
	}
	return b
}

func fuzzPick[T any](in *fuzzInput, options ...T) T {
	return options[in.intn(len(options))]
}

type fuzzAccount struct {
	nonce   uint64
	balance uint256.Int
	code    []byte
	storage map[common.Hash]common.Hash
}

func (a *fuzzAccount) empty() bool {
	return a == nil || (a.nonce == 0 && a.balance.IsZero() && len(a.code) == 0)
}

// fuzzWorld describes a head state. Empty accounts are deleted on commit, as
// EIP-161 guarantees for any real head.
type fuzzWorld map[common.Address]*fuzzAccount

func (w fuzzWorld) account(addr common.Address) *fuzzAccount {
	if w[addr] == nil {
		w[addr] = &fuzzAccount{storage: make(map[common.Hash]common.Hash)}
	}
	return w[addr]
}

func (w fuzzWorld) balance(addr common.Address) *uint256.Int {
	if acc := w[addr]; acc != nil {
		return &acc.balance
	}
	return new(uint256.Int)
}

// with returns a copy of the world with the mutations applied in order.
func (w fuzzWorld) with(muts ...*fuzzMutation) fuzzWorld {
	cpy := make(fuzzWorld, len(w))
	for addr, acc := range w {
		cpy[addr] = &fuzzAccount{nonce: acc.nonce, balance: acc.balance, code: acc.code, storage: maps.Clone(acc.storage)}
	}
	for _, m := range muts {
		m.apply(cpy)
	}
	return cpy
}

func (w fuzzWorld) state(t *testing.T, rules params.Rules) *state.StateDB {
	db := state.NewDatabaseForTesting()
	sdb, err := state.New(types.EmptyRootHash, db)
	if err != nil {
		t.Fatal(err)
	}
	for addr, acc := range w {
		sdb.SetNonce(addr, acc.nonce, tracing.NonceChangeUnspecified)
		sdb.SetBalance(addr, &acc.balance, tracing.BalanceChangeUnspecified)
		if len(acc.code) != 0 {
			sdb.SetCode(addr, acc.code, tracing.CodeChangeUnspecified)
		}
		for key, value := range acc.storage {
			sdb.SetState(addr, key, value)
		}
	}
	root, err := sdb.Commit(rules, 0)
	if err != nil {
		t.Fatal(err)
	}
	if sdb, err = state.New(root, db); err != nil {
		t.Fatal(err)
	}
	return sdb
}

type fuzzEnv struct {
	config   *params.ChainConfig
	keys     []*ecdsa.PrivateKey
	eoas     []common.Address // eoas[0] signs for the sender, eoas[1] for a sponsor
	targets  []common.Address // literal CALL* and EXTCODE* targets
	universe []common.Address // accounts the state mutators may change
}

func newFuzzEnv(tb testing.TB) *fuzzEnv {
	config := *params.AllDevChainProtocolChanges
	config.AmsterdamTime = new(uint64)
	env := &fuzzEnv{config: &config}
	for id := 1; id <= 2; id++ {
		key, err := crypto.HexToECDSA(fmt.Sprintf("%064x", id))
		if err != nil {
			tb.Fatal(err)
		}
		env.keys = append(env.keys, key)
		env.eoas = append(env.eoas, crypto.PubkeyToAddress(key.PublicKey))
	}
	others := []common.Address{fuzzContractSender, fuzzPaymaster, fuzzFactory, fuzzDelegate, fuzzUnused, params.FrameTxExpiryVerifier, params.FrameTxEntryPoint, {}}
	env.universe = slices.Concat(env.eoas, fuzzHelpers, fuzzPrecompiles, others)
	env.targets = slices.Concat(fuzzHelpers, fuzzHelpers, fuzzPrecompiles, env.eoas, others)
	return env
}

// fuzzCodeGen emits stack-balanced statements from the fuzz input.
type fuzzCodeGen struct {
	in      *fuzzInput
	targets []common.Address
	frames  int
	sigs    int
}

func fuzzPrologue(scope byte) []byte {
	return []byte{
		byte(vm.PUSH1), 15, byte(vm.JUMP),
		byte(vm.JUMPDEST), byte(vm.PUSH0), byte(vm.PUSH0), byte(vm.REVERT),
		byte(vm.JUMPDEST), byte(vm.STOP),
		byte(vm.JUMPDEST), byte(vm.PUSH1), scope, byte(vm.PUSH0), byte(vm.PUSH0), byte(vm.APPROVE),
		byte(vm.JUMPDEST),
	}
}

// verifier returns runtime code which normally ends by approving scope.
func (g *fuzzCodeGen) verifier(scope byte) []byte {
	p := program.New().Append(fuzzPrologue(scope))
	g.statements(p)
	switch g.in.intn(8) {
	case 6:
		p.Push(fuzzPick(g.in, 0, 1, 2, 3, 4, 256)).Push(0).Push(0).Op(vm.APPROVE)
	case 7:
		p.Op(vm.STOP)
	default:
		p.Push(fuzzLabelApprove).Op(vm.JUMP)
	}
	return p.Bytes()
}

// callee returns code for helpers reached through CALL* and for post-ops.
func (g *fuzzCodeGen) callee() []byte {
	// DELEGATECALLed from a verifier, a callee may approve in its place.
	p := program.New().Append(fuzzPrologue(byte(fuzzPick(g.in, types.ApproveExecutionAndPayment, types.ApproveExecution, types.ApprovePayment))))
	g.statements(p)
	switch g.in.intn(4) {
	case 2:
		p.Op(vm.STOP)
	case 3:
		p.Push(fuzzLabelFail).Op(vm.JUMP)
	default:
		p.Return(0x80, 32)
	}
	return p.Bytes()
}

func (g *fuzzCodeGen) statements(p *program.Program) {
	for range g.in.intn(fuzzMaxStatements + 1) {
		g.statement(p)
	}
}

var (
	fuzzBannedOps = []vm.OpCode{vm.GASPRICE, vm.COINBASE, vm.TIMESTAMP, vm.NUMBER, vm.PREVRANDAO, vm.GASLIMIT, vm.BASEFEE, vm.BLOBBASEFEE, vm.SLOTNUM, vm.SELFBALANCE, vm.GAS}
	fuzzEnvOps    = []vm.OpCode{vm.ADDRESS, vm.CALLER, vm.ORIGIN, vm.CALLVALUE, vm.CALLDATASIZE, vm.CODESIZE, vm.CHAINID, vm.MSIZE, vm.RETURNDATASIZE, vm.PC}
	fuzzBinaryOps = []vm.OpCode{vm.ADD, vm.SUB, vm.MUL, vm.DIV, vm.MOD, vm.AND, vm.OR, vm.XOR, vm.EQ, vm.LT, vm.GT, vm.SHR, vm.BYTE}
	fuzzCallOps   = []vm.OpCode{vm.CALL, vm.STATICCALL, vm.DELEGATECALL, vm.CALLCODE}
)

func (g *fuzzCodeGen) slot() common.Hash { return fuzzPick(g.in, fuzzSlots...) }

func (g *fuzzCodeGen) statement(p *program.Program) {
	switch g.in.intn(20) {
	case 0, 1, 2, 3: // Abort on a state-derived condition.
		g.expr(p, 0)
		if g.in.intn(2) == 1 {
			p.Op(vm.ISZERO)
		}
		p.Push(fuzzLabelFail).Op(vm.JUMPI)
	case 4: // Approve early on a condition.
		g.expr(p, 0)
		p.Push(fuzzLabelApprove).Op(vm.JUMPI)
	case 5: // Succeed without approval on a condition.
		g.expr(p, 0)
		p.Push(fuzzLabelStop).Op(vm.JUMPI)
	case 6, 7:
		g.expr(p, 0)
		p.Op(vm.POP)
	case 8:
		g.expr(p, 0)
		p.Push(fuzzPick(g.in, 0, 32, 0x80, 0x200)).Op(vm.MSTORE)
	case 9: // Precompile input material.
		p.Push(g.in.bytes(32)).Push(fuzzPick(g.in, 0, 32, 64, 96)).Op(vm.MSTORE)
	case 10:
		g.expr(p, 0)
		p.Push(g.slot()).Op(vm.SSTORE)
	case 11:
		g.expr(p, 0)
		p.Push(g.slot()).Op(vm.TSTORE)
	case 12, 13:
		g.call(p)
		if g.in.intn(2) == 0 {
			p.Op(vm.POP)
		} else {
			p.Op(vm.ISZERO).Push(fuzzLabelFail).Op(vm.JUMPI)
		}
	case 14: // A nested creation, legal only inside the deploy frame.
		initcode := []byte{byte(vm.PUSH0), byte(vm.PUSH0), byte(vm.RETURN)}
		p.Push(initcode).Push(0).Op(vm.MSTORE)
		if g.in.intn(2) == 0 {
			p.Push(len(initcode)).Push(32 - len(initcode))
			g.value(p)
			p.Op(vm.CREATE)
		} else {
			p.Push(g.in.byte()).Push(len(initcode)).Push(32 - len(initcode))
			g.value(p)
			p.Op(vm.CREATE2)
		}
		p.Op(vm.POP)
	case 15:
		p.Push(32).Push(0).Op(vm.LOG0)
	case 16:
		switch k := g.in.intn(len(fuzzBannedOps) + 4); {
		case k < len(fuzzBannedOps):
			p.Op(fuzzBannedOps[k], vm.POP)
		case k == len(fuzzBannedOps):
			p.Push(0).Op(vm.BLOCKHASH, vm.POP)
		case k == len(fuzzBannedOps)+1:
			g.address(p)
			p.Op(vm.BALANCE, vm.POP)
		case k == len(fuzzBannedOps)+2:
			p.Op(vm.INVALID)
		default:
			g.address(p)
			p.Op(vm.SELFDESTRUCT)
		}
	case 17:
		p.Push(fuzzPick(g.in, 0, 1, 2, 3, 4)).Push(0).Push(0).Op(vm.APPROVE)
	case 18: // Burn gas in a short loop.
		p.Push(1 + g.in.intn(64))
		_, loop := p.Jumpdest()
		p.Push(1).Op(vm.SWAP1, vm.SUB, vm.DUP1).Push(loop).Op(vm.JUMPI, vm.POP)
	default: // Memory copies from code, frames and return data.
		switch g.in.intn(4) {
		case 0:
			p.Push(32).Push(g.in.intn(8)).Push(0x200)
			g.address(p)
			p.Op(vm.EXTCODECOPY)
		case 1:
			p.Push(g.in.intn(g.frames + 1)).Push(32).Push(g.in.intn(8)).Push(0x200).Op(vm.FRAMEDATACOPY)
		case 2:
			p.Push(32).Push(0).Push(0x200).Op(vm.CALLDATACOPY)
		default:
			p.Push(g.in.intn(33)).Push(0).Push(0x200).Op(vm.RETURNDATACOPY)
		}
	}
}

// expr pushes exactly one value.
func (g *fuzzCodeGen) expr(p *program.Program, depth int) {
	choices := 22
	if depth >= 2 {
		choices = 19 // Leaves only.
	}
	switch g.in.intn(choices) {
	case 0:
		p.Push(fuzzPick(g.in, uint64(0), 1, 2, 31, 32, 0xff, 1<<40, 1000, 3000, 10_000))
	case 1, 2:
		p.Push(g.slot()).Op(vm.SLOAD)
	case 3:
		g.address(p)
		p.Op(vm.EXTCODESIZE)
	case 4, 5:
		g.address(p)
		p.Op(vm.EXTCODEHASH)
	case 6:
		p.Push(32).Push(g.in.intn(8)).Push(0x200)
		g.address(p)
		p.Op(vm.EXTCODECOPY).Push(0x200).Op(vm.MLOAD)
	case 7, 8:
		g.call(p)
	case 9:
		g.call(p)
		p.Op(vm.POP, vm.RETURNDATASIZE)
	case 10:
		g.call(p)
		p.Op(vm.POP).Push(0x80).Op(vm.MLOAD)
	case 11:
		p.Push(g.in.intn(14)).Op(vm.TXPARAM)
	case 12:
		if g.in.intn(2) == 0 {
			// A bit of an earlier frame's gas usage, which flips when an
			// account access turns warm.
			p.Push(1).Push(fuzzPick(g.in, 1000, 3000))
			p.Push(frameParamGasUsed).Push(g.in.intn(g.frames)).Op(vm.FRAMEPARAM, vm.DIV, vm.AND)
		} else {
			p.Push(g.in.intn(13)).Push(g.in.intn(g.frames + 1)).Op(vm.FRAMEPARAM)
		}
	case 13:
		p.Push(g.in.intn(g.frames + 1)).Push(g.in.intn(40)).Op(vm.FRAMEDATALOAD)
	case 14:
		p.Push(g.in.intn(5)).Push(g.in.intn(g.sigs + 1)).Op(vm.SIGPARAM)
	case 15:
		p.Op(fuzzPick(g.in, fuzzEnvOps...))
	case 16:
		p.Push(g.in.intn(40)).Op(vm.CALLDATALOAD)
	case 17:
		p.Push(g.slot()).Op(vm.TLOAD)
	case 18:
		p.Push(fuzzPick(g.in, 0, 32, 0x80, 0x200)).Op(vm.MLOAD)
	case 19, 20:
		g.expr(p, depth+1)
		g.expr(p, depth+1)
		p.Op(fuzzPick(g.in, fuzzBinaryOps...))
	default:
		g.expr(p, depth+1)
		p.Op(fuzzPick(g.in, vm.ISZERO, vm.NOT, vm.CLZ))
	}
}

func (g *fuzzCodeGen) address(p *program.Program) {
	switch g.in.intn(8) {
	case 5:
		p.Push(2).Op(vm.TXPARAM) // tx.sender
	case 6:
		p.Op(fuzzPick(g.in, vm.ADDRESS, vm.CALLER, vm.ORIGIN))
	case 7:
		p.Push(0).Push(g.in.intn(g.frames)).Op(vm.FRAMEPARAM) // A frame's resolved target.
	default:
		p.Push(fuzzPick(g.in, g.targets...))
	}
}

// value pushes a call or creation value, usually zero. The caller may not
// afford it.
func (g *fuzzCodeGen) value(p *program.Program) {
	if g.in.oneIn(8) {
		p.Push(fuzzPick(g.in, uint64(1), 1e18))
	} else {
		p.Push(0)
	}
}

// call pushes the success flag of a CALL* to a generated target. GAS is
// usually the instruction immediately preceding the call.
func (g *fuzzCodeGen) call(p *program.Program) {
	op := fuzzPick(g.in, fuzzCallOps...)
	p.Push(32).Push(0x80).Push(fuzzPick(g.in, 0, 32, 128, 192)).Push(0)
	if op == vm.CALL || op == vm.CALLCODE {
		g.value(p)
	}
	g.address(p)
	if g.in.intn(3) == 2 {
		p.Push(fuzzPick(g.in, 0, 50, 700, 3000, 10_000, 0xffffff))
	} else {
		p.Op(vm.GAS)
	}
	p.Op(op)
}

type fuzzCase struct {
	env     *fuzzEnv
	rules   params.Rules
	head    *types.Header
	world   fuzzWorld
	tx      *types.Transaction
	prefix  Prefix
	sender  common.Address
	payer   common.Address // Expected payer.
	maxCost *uint256.Int
	rest    *fuzzInput // Remaining input drives the state mutations.
}

// generate decodes a transaction and head state. It returns nil for inputs the
// pool rejects before simulation: static, signature or prefix-shape errors.
func (env *fuzzEnv) generate(data []byte) *fuzzCase {
	in := &fuzzInput{data: data}
	world := make(fuzzWorld)
	shape := in.byte()
	var (
		sponsored = shape&1 != 0
		deploy    = shape&6 == 6
		expiry    = shape&0x18 == 0x18
		postOps   = int(shape>>5) % 3
		extraSig  = in.oneIn(8)
	)
	frameCount, sigCount := 1+postOps, 1
	if sponsored {
		frameCount++
		sigCount++
	}
	if deploy {
		frameCount++
	}
	if expiry {
		frameCount++
	}
	if extraSig {
		sigCount++
	}
	g := &fuzzCodeGen{in: in, targets: env.targets, frames: frameCount, sigs: sigCount}

	for _, helper := range fuzzHelpers {
		switch in.intn(6) {
		case 0, 1:
			world.account(helper).code = g.callee()
		case 2: // Absent.
		case 3:
			world.account(helper).balance.SetUint64(1)
		case 4:
			world.account(helper).code = types.AddressToDelegation(fuzzPick(in, env.targets...))
		default:
			world.account(helper).code = []byte{byte(vm.STOP)}
		}
	}
	for _, precompile := range fuzzPrecompiles {
		if in.intn(2) == 1 {
			world.account(precompile).balance.SetUint64(1)
		}
	}
	if !in.oneIn(16) {
		world.account(params.FrameTxExpiryVerifier).code = params.FrameTxExpiryVerifierCode
	} else {
		world.account(params.FrameTxExpiryVerifier).code = []byte{byte(vm.STOP)}
	}

	verifyFlags := uint64(types.ApproveExecutionAndPayment)
	if sponsored {
		verifyFlags = types.ApproveExecution
	}
	if in.oneIn(16) {
		verifyFlags = uint64(in.intn(4))
	}
	var sender common.Address
	if deploy {
		runtime := g.verifier(byte(verifyFlags))
		initcode := program.New().Append(fuzzPrologue(0))
		for range in.intn(fuzzMaxStatements + 1) {
			if in.intn(2) == 0 { // Sender storage writes are legal here.
				g.expr(initcode, 1)
				initcode.Push(g.slot()).Op(vm.SSTORE)
			} else {
				g.statement(initcode)
			}
		}
		initcode.ReturnViaCodeCopy(runtime)
		code := initcode.Bytes()

		factory := program.New().Append(fuzzPrologue(0))
		g.statements(factory)
		nonce := fuzzPick(in, uint64(0), 1, 2, math.MaxUint64-1, math.MaxUint64-2, math.MaxUint64)
		if in.oneIn(4) { // An endowed creation the factory may not afford.
			factory.Mstore(code, 0)
			if in.intn(2) == 0 {
				factory.Push(in.byte()).Push(len(code)).Push(0).Push(fuzzPick(in, uint64(1), 1e18)).Op(vm.CREATE2, vm.POP)
			} else {
				factory.Push(len(code)).Push(0).Push(fuzzPick(in, uint64(1), 1e18)).Op(vm.CREATE, vm.POP)
			}
		}
		create := in.intn(4)
		if create == 2 {
			factory.Mstore(code, 0).Push(len(code)).Push(0).Push(0).Op(vm.CREATE)
			sender = crypto.CreateAddress(fuzzFactory, nonce)
		} else {
			salt := common.BytesToHash(in.bytes(2))
			factory.Create2(code, salt)
			sender = crypto.CreateAddress2(fuzzFactory, salt, crypto.Keccak256(code))
		}
		if create == 3 {
			sender = env.eoas[0] // The creation lands elsewhere.
		}
		if in.oneIn(4) {
			factory.Op(vm.POP)
		} else {
			factory.Op(vm.ISZERO).Push(fuzzLabelFail).Op(vm.JUMPI)
		}
		g.statements(factory)
		factory.Op(vm.STOP)
		if in.oneIn(8) { // A delegated factory runs its delegate's code.
			world.account(fuzzFactory).code = types.AddressToDelegation(fuzzDelegate)
			world.account(fuzzDelegate).code = factory.Bytes()
		} else {
			world.account(fuzzFactory).code = factory.Bytes()
		}
		world.account(fuzzFactory).nonce = nonce
		world.account(fuzzFactory).balance = *fuzzPick(in, new(uint256.Int), fuzzEther)
	} else {
		world.account(fuzzFactory).code = g.callee()
		switch in.intn(4) {
		case 0:
			sender = env.eoas[0]
		case 1:
			sender = env.eoas[0]
			world.account(sender).code = types.AddressToDelegation(fuzzDelegate)
			world.account(fuzzDelegate).code = g.verifier(byte(verifyFlags))
		case 2:
			sender = fuzzContractSender
			world.account(sender).code = g.verifier(byte(verifyFlags))
		default:
			sender = env.eoas[0]
			world.account(sender).code = types.AddressToDelegation(fuzzPick(in, fuzzHelpers...))
		}
	}
	senderAccount := world.account(sender)
	senderAccount.nonce = fuzzPick(in, uint64(0), 1, 7)
	senderAccount.balance = *fuzzPick(in, fuzzEther, new(uint256.Int), uint256.NewInt(1))
	for _, slot := range fuzzSlots {
		switch in.intn(4) {
		case 1:
			senderAccount.storage[slot] = common.BytesToHash([]byte{1})
		case 2:
			senderAccount.storage[slot] = common.BytesToHash([]byte{2})
		case 3:
			senderAccount.storage[slot] = common.BytesToHash(append([]byte{1}, in.bytes(31)...))
		}
	}
	txNonce := senderAccount.nonce
	if in.oneIn(16) {
		txNonce++
	}

	payer := sender
	if sponsored {
		switch in.intn(4) {
		case 0:
			payer = env.eoas[1]
		case 1:
			payer = fuzzPaymaster
			world.account(payer).code = g.verifier(byte(types.ApprovePayment))
		case 2:
			payer = env.eoas[1]
			world.account(payer).code = types.AddressToDelegation(fuzzPaymaster)
			world.account(fuzzPaymaster).code = g.verifier(byte(types.ApprovePayment))
		default:
			payer = fuzzPick(in, env.universe...)
		}
		if payer != sender {
			world.account(payer).nonce = fuzzPick(in, uint64(0), 1)
		}
	}
	for _, addr := range []common.Address{params.FrameTxEntryPoint, {}, fuzzUnused} {
		if in.oneIn(8) {
			world.account(addr).balance.SetUint64(1)
		}
	}

	// Frames, with prefix budgets clamped to the policy caps.
	executionLeft, stateLeft := MaxVerifyGas-uint64(sigCount)*params.FrameTxSecp256k1SigGas, MaxVerifyStateGas
	limits := func(execution, state uint64) types.Limits {
		execution, state = min(execution, executionLeft), min(state, stateLeft)
		executionLeft, stateLeft = executionLeft-execution, stateLeft-state
		return types.Limits{Execution: execution, State: state}
	}
	creation := uint64(params.AccountCreationSize * params.CostPerStateByte)
	frameData := func() []byte { return in.bytes(fuzzPick(in, 0, 4, 32, 33)) }
	var frames []types.Frame
	if expiry {
		deadline := make([]byte, params.FrameTxExpiryDataLen)
		binary.BigEndian.PutUint64(deadline, fuzzPick(in, uint64(1000), 1001, 4600, 999))
		target := params.FrameTxExpiryVerifier
		frames = append(frames, types.Frame{Mode: types.ModeVerify, Target: &target, Data: deadline, GasLimits: limits(fuzzPick(in, uint64(10_000), 3_000, 100), 0), Value: new(uint256.Int)})
	}
	if deploy {
		target := fuzzFactory
		state := fuzzPick(in, uint64(300_000), MaxVerifyStateGas, creation, 0, 100_000, 200_000, 250_000, 350_000, 400_000)
		frames = append(frames, types.Frame{Mode: types.ModeDefault, Target: &target, Data: frameData(), GasLimits: limits(fuzzPick(in, uint64(60_000), 30_000, 90_000, 5_000), state), Value: new(uint256.Int)})
	}
	verify := types.Frame{Mode: types.ModeVerify, Flags: verifyFlags, Data: frameData(), Value: new(uint256.Int)}
	verify.GasLimits = limits(fuzzPick(in, uint64(30_000), 10_000, 60_000, 3_000, 200), fuzzPick(in, uint64(0), creation))
	if in.oneIn(4) {
		target := sender
		verify.Target = &target
	}
	frames = append(frames, verify)
	if sponsored {
		target := payer
		frames = append(frames, types.Frame{Mode: types.ModeVerify, Flags: types.ApprovePayment, Target: &target, Data: frameData(), GasLimits: limits(fuzzPick(in, uint64(30_000), 10_000, 60_000, 3_000), fuzzPick(in, creation, 0)), Value: new(uint256.Int)})
	}
	for range postOps {
		frame := types.Frame{Mode: types.ModeSender, Data: frameData(), GasLimits: types.Limits{Execution: 50_000}, Value: new(uint256.Int)}
		if in.intn(2) == 1 {
			frame.Mode = types.ModeDefault
		}
		if in.intn(2) == 1 {
			target := fuzzPick(in, fuzzHelpers...)
			frame.Target = &target
		}
		if frame.Mode == types.ModeSender && in.oneIn(4) {
			frame.Value.SetUint64(1)
		}
		frames = append(frames, frame)
	}

	// Signatures: protocol entries signed with real keys, ARBITRARY otherwise.
	var (
		signatures = make(types.SignatureList, sigCount)
		signers    = make([]*ecdsa.PrivateKey, sigCount)
	)
	arbitrary := func(i int) {
		signatures[i] = types.SignatureEntry{Scheme: types.FrameTxSchemeArbitrary, Signature: in.bytes(in.intn(66))}
	}
	secp := func(i, key int, explicit bool) {
		signatures[i] = types.SignatureEntry{Scheme: types.FrameTxSchemeSecp256k1}
		if explicit {
			signatures[i].Signer = env.eoas[key].Bytes()
		}
		if in.oneIn(16) {
			signatures[i].Msg = crypto.Keccak256(in.bytes(4))
		}
		signers[i] = env.keys[key]
	}
	if sender == env.eoas[0] && !in.oneIn(4) {
		secp(0, 0, in.intn(2) == 1)
	} else if in.intn(2) == 1 {
		secp(0, 0, true)
	} else {
		arbitrary(0)
	}
	if sponsored {
		if payer == env.eoas[1] && !in.oneIn(4) || in.intn(2) == 1 {
			secp(1, 1, true)
		} else {
			arbitrary(1)
		}
	}
	if extraSig {
		arbitrary(sigCount - 1)
	}

	maxFee := fuzzPick(in, uint64(10), 1, 1000, 1_000_000_000)
	inner := &types.FrameTx{
		ChainID: uint256.MustFromBig(env.config.ChainID), Nonce: txNonce, Sender: sender, Frames: frames, Signatures: signatures,
		Fees: types.Fees{MaxFeePerGas: uint256.NewInt(maxFee), MaxPriorityFeePerGas: uint256.NewInt(min(maxFee, fuzzPick(in, uint64(1), 0, maxFee))), MaxFeePerBlobGas: new(uint256.Int)},
	}
	head := &types.Header{
		Number: big.NewInt(1), Time: 1000, Difficulty: new(big.Int), GasLimit: 30_000_000,
		BaseFee: new(big.Int).SetUint64(fuzzPick(in, maxFee, 1, maxFee/2)), ExcessBlobGas: new(uint64), BlobGasUsed: new(uint64),
	}
	signer := types.MakeSigner(env.config, head.Number, head.Time)
	sign := func(i int, hash []byte) {
		sig, err := crypto.Sign(hash, signers[i])
		if err != nil {
			panic(err)
		}
		signatures[i].Signature = append([]byte{sig[64]}, sig[:64]...)
	}
	// Signatures over an explicit message are part of the canonical hash.
	for i := range signatures {
		if signers[i] != nil && len(signatures[i].Msg) != 0 {
			sign(i, signatures[i].Msg)
		}
	}
	hash := signer.Hash(types.NewTx(inner))
	for i := range signatures {
		if signers[i] != nil && len(signatures[i].Msg) == 0 {
			sign(i, hash[:])
		}
	}
	tx := types.NewTx(inner)
	if tx.FrameTxValidateStatic() != nil || types.ValidateFrameTxSignatures(signatures, sender, signer.Hash(tx)) != nil {
		return nil
	}
	prefix, err := ClassifyPrefix(frames, sender, signatures)
	if err != nil {
		return nil
	}
	maxCost := new(uint256.Int).Mul(uint256.NewInt(tx.Gas()), uint256.NewInt(maxFee))
	switch in.intn(5) {
	case 0:
		world.account(payer).balance = *fuzzEther
	case 1:
		world.account(payer).balance = *maxCost
	case 2:
		world.account(payer).balance.SubUint64(maxCost, 1)
	case 3:
		world.account(payer).balance.SetUint64(1)
	}
	return &fuzzCase{
		env: env, rules: env.config.Rules(head.Number, true, head.Time), head: head, world: world, tx: tx, prefix: prefix,
		sender: sender, payer: frames[prefix.End].ResolvedTarget(sender), maxCost: maxCost, rest: in,
	}
}

func (c *fuzzCase) simulate(t *testing.T, world fuzzWorld) (*simResult, error) {
	return simulate(c.env.config, c.head, world.state(t, c.rules), c.tx, c.prefix)
}

// universe lists every account a mutator may change, including the sender.
func (c *fuzzCase) universe() []common.Address {
	if slices.Contains(c.env.universe, c.sender) {
		return c.env.universe
	}
	return append(slices.Clone(c.env.universe), c.sender)
}

func (c *fuzzCase) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "sender %v nonce %d payer %v maxCost %v prefix %+v\n", c.sender, c.tx.Nonce(), c.payer, c.maxCost, c.prefix)
	for i, frame := range c.tx.Frames() {
		fmt.Fprintf(&b, "frame %d: mode %d flags %d target %v data %x limits %+v value %v\n", i, frame.Mode, frame.Flags, frame.ResolvedTarget(c.sender), frame.Data, frame.GasLimits, frame.Value)
	}
	for i, sig := range c.tx.FrameSignatures() {
		fmt.Fprintf(&b, "signature %d: scheme %d signer %x msg %x\n", i, sig.Scheme, sig.Signer, sig.Msg)
	}
	for _, addr := range slices.SortedFunc(maps.Keys(c.world), common.Address.Cmp) {
		acc := c.world[addr]
		fmt.Fprintf(&b, "account %v: nonce %d balance %v code %x storage %v\n", addr, acc.nonce, &acc.balance, acc.code, acc.storage)
	}
	return b.String()
}

// diffSimResults describes the first difference between two accepted results.
func diffSimResults(want, got *simResult, payerBalance bool) string {
	switch {
	case want.payer != got.payer:
		return fmt.Sprintf("payer %v, want %v", got.payer, want.payer)
	case want.codedPaymaster != got.codedPaymaster:
		return fmt.Sprintf("coded paymaster %v, want %v", got.codedPaymaster, want.codedPaymaster)
	case (want.expiryDeadline == nil) != (got.expiryDeadline == nil) || (want.expiryDeadline != nil && *want.expiryDeadline != *got.expiryDeadline):
		return fmt.Sprintf("expiry %v, want %v", got.expiryDeadline, want.expiryDeadline)
	case !maps.Equal(want.dependencies.accounts, got.dependencies.accounts):
		return fmt.Sprintf("account dependencies %v, want %v", got.dependencies.accounts, want.dependencies.accounts)
	case !maps.Equal(want.dependencies.slots, got.dependencies.slots):
		return fmt.Sprintf("slot dependencies %v, want %v", slices.Collect(maps.Keys(got.dependencies.slots)), slices.Collect(maps.Keys(want.dependencies.slots)))
	case payerBalance && want.payerBalance.Cmp(got.payerBalance) != 0:
		return fmt.Sprintf("payer balance %v, want %v", got.payerBalance, want.payerBalance)
	}
	return ""
}

// fuzzStorage marks storage mutations next to the account dependency fields.
const fuzzStorage accountFields = 1 << 7

type fuzzMutation struct {
	addr  common.Address
	field accountFields
	slot  common.Hash
	nonce uint64
	value uint256.Int // Balance or storage value.
	code  []byte
}

func (m *fuzzMutation) apply(world fuzzWorld) {
	acc := world.account(m.addr)
	switch m.field {
	case dependencyNonce:
		acc.nonce = m.nonce
	case dependencyBalance:
		acc.balance = m.value
	case dependencyCode:
		acc.code = m.code
	default:
		if m.value.IsZero() {
			delete(acc.storage, m.slot)
		} else {
			acc.storage[m.slot] = m.value.Bytes32()
		}
	}
}

func (m *fuzzMutation) String() string {
	switch m.field {
	case dependencyNonce:
		return fmt.Sprintf("%v nonce := %d", m.addr, m.nonce)
	case dependencyBalance:
		return fmt.Sprintf("%v balance := %v", m.addr, &m.value)
	case dependencyCode:
		return fmt.Sprintf("%v code := %x", m.addr, m.code)
	default:
		return fmt.Sprintf("%v storage[%v] := %v", m.addr, m.slot, &m.value)
	}
}

// mutate returns a value change for the field. Nonces only grow, as in a block
// on top of the head, by less than creatorNonceMargin.
func (c *fuzzCase) mutate(in *fuzzInput, addr common.Address, field accountFields, slot common.Hash) *fuzzMutation {
	m := &fuzzMutation{addr: addr, field: field, slot: slot}
	acc := c.world[addr]
	if acc == nil {
		acc = &fuzzAccount{}
	}
	switch field {
	case dependencyNonce:
		m.nonce = acc.nonce + min(fuzzPick(in, uint64(1), 2, 3, 1<<16), math.MaxUint64-acc.nonce)
	case dependencyBalance:
		if acc.balance.IsZero() {
			m.value = *fuzzPick(in, uint256.NewInt(1), fuzzEther)
		} else {
			m.value = *fuzzPick(in, new(uint256.Int), new(uint256.Int).AddUint64(&acc.balance, 1), uint256.NewInt(1))
		}
	case dependencyCode:
		returnOne := program.New().Push(1).Push(0).Op(vm.MSTORE).Return(0, 32).Bytes()
		if len(acc.code) == 0 {
			m.code = fuzzPick(in, []byte{byte(vm.STOP)}, returnOne, types.AddressToDelegation(fuzzHelpers[0]), approveCode(program.New(), types.ApproveExecutionAndPayment))
		} else {
			m.code = fuzzPick(in, nil, []byte{byte(vm.STOP)}, returnOne, append(slices.Clone(acc.code), byte(vm.STOP)))
		}
	default:
		current := new(uint256.Int).SetBytes(acc.storage[slot].Bytes())
		if current.IsZero() {
			if in.intn(2) == 1 {
				m.value.SetBytes(in.bytes(32))
			}
			if m.value.IsZero() {
				m.value.SetOne()
			}
		} else {
			m.value = *fuzzPick(in, new(uint256.Int), new(uint256.Int).AddUint64(current, 1))
		}
	}
	return m
}

// mutations selects changes of the fields for which covered is false.
func (c *fuzzCase) mutations(in *fuzzInput, covered func(common.Address, accountFields, common.Hash) bool) []*fuzzMutation {
	var muts []*fuzzMutation
	for _, addr := range c.universe() {
		for _, field := range []accountFields{dependencyNonce, dependencyBalance, dependencyCode} {
			if !covered(addr, field, common.Hash{}) && in.intn(2) == 0 {
				muts = append(muts, c.mutate(in, addr, field, common.Hash{}))
			}
		}
		for _, slot := range fuzzSlots {
			if !covered(addr, fuzzStorage, slot) && in.intn(2) == 0 {
				muts = append(muts, c.mutate(in, addr, fuzzStorage, slot))
			}
		}
	}
	return muts
}

// covers reports whether result's dependencies include the field.
func covers(result *simResult, sender, addr common.Address, field accountFields, slot common.Hash) bool {
	if field == fuzzStorage {
		_, ok := result.dependencies.slots[slot]
		return addr == sender && ok
	}
	return result.dependencies.accounts[addr]&field != 0
}

// restore returns the mutation undoing m on the case's head state.
func (c *fuzzCase) restore(m *fuzzMutation) *fuzzMutation {
	r := &fuzzMutation{addr: m.addr, field: m.field, slot: m.slot}
	if acc := c.world[m.addr]; acc != nil {
		r.nonce, r.value, r.code = acc.nonce, acc.balance, acc.code
		if m.field == fuzzStorage {
			r.value.SetBytes(acc.storage[m.slot].Bytes())
		}
	}
	return r
}

// checkUnchanged fails the test when simulating world with muts applied
// differs from want, naming a single responsible mutation when one exists. It
// returns the result of the mutated simulation.
func (c *fuzzCase) checkUnchanged(t *testing.T, invariant string, want *simResult, world fuzzWorld, muts []*fuzzMutation, payerBalance bool) *simResult {
	t.Helper()
	differs := func(muts []*fuzzMutation) (*simResult, string) {
		got, err := c.simulate(t, world.with(muts...))
		if err != nil {
			return nil, fmt.Sprintf("rejected: %v", err)
		}
		return got, diffSimResults(want, got, payerBalance)
	}
	got, diff := differs(muts)
	if diff == "" {
		return got
	}
	slots := slices.Collect(maps.Keys(want.dependencies.slots))
	for _, m := range muts {
		if _, single := differs([]*fuzzMutation{m}); single != "" {
			t.Fatalf("%s violated by %v: %s\ndependencies %v slots %v\n%v", invariant, m, single, want.dependencies.accounts, slots, c)
		}
	}
	t.Fatalf("%s violated by %v: %s\ndependencies %v slots %v\n%v", invariant, muts, diff, want.dependencies.accounts, slots, c)
	return nil
}

// fuzzSeeds adds deterministic inputs covering every prefix shape.
func fuzzSeeds(f *testing.F) {
	rng := rand.New(rand.NewPCG(8141, 1))
	for range 48 {
		seed := make([]byte, 16+rng.IntN(600))
		for i := range seed {
			seed[i] = byte(rng.Uint32())
		}
		f.Add(seed)
	}
	for shape := range 0x20 {
		f.Add([]byte{byte(shape)})
	}
}

// FuzzSimulationDependencies checks dependency completeness and the
// balance-only premise for every accepted prefix.
func FuzzSimulationDependencies(f *testing.F) {
	env := newFuzzEnv(f)
	fuzzSeeds(f)
	f.Fuzz(func(t *testing.T, data []byte) {
		c := env.generate(data)
		if c == nil {
			return
		}
		base, err := c.simulate(t, c.world)
		if err != nil {
			// A rejection may turn into an acceptance through a few changes,
			// but the accepted state then satisfies dependency completeness:
			// undoing every change outside its dependencies keeps the result.
			in := c.rest
			if len(in.data) < 8 { // Short inputs still pick varied changes.
				in = &fuzzInput{data: slices.Concat(in.data, crypto.Keccak256(data))}
			}
			var muts []*fuzzMutation
			for range 1 + in.intn(3) {
				addr := fuzzPick(in, c.universe()...)
				field := fuzzPick(in, dependencyNonce, dependencyBalance, dependencyCode, fuzzStorage)
				var slot common.Hash
				if field == fuzzStorage {
					slot = fuzzPick(in, fuzzSlots...)
				}
				muts = append(muts, c.mutate(in, addr, field, slot))
			}
			flipped := c.world.with(muts...)
			accepted, err := c.simulate(t, flipped)
			if err != nil {
				return
			}
			var undo []*fuzzMutation
			for _, m := range muts {
				// Restoring a grown nonce is no reachable change.
				if m.field != dependencyNonce && !covers(accepted, c.sender, m.addr, m.field, m.slot) {
					undo = append(undo, c.restore(m))
				}
			}
			c.checkUnchanged(t, "dependency completeness after acceptance", accepted, flipped, undo, true)
			return
		}
		if base.payer != c.payer || base.payerBalance.Cmp(c.world.balance(c.payer)) != 0 {
			t.Fatalf("accepted payer %v balance %v, want %v balance %v", base.payer, base.payerBalance, c.payer, c.world.balance(c.payer))
		}
		// Dependency completeness.
		c.checkUnchanged(t, "dependency completeness", base, c.world, c.mutations(c.rest, func(addr common.Address, field accountFields, slot common.Hash) bool {
			return covers(base, c.sender, addr, field, slot)
		}), true)

		// Balance-only premise: emptiness and payer solvency are preserved.
		var muts []*fuzzMutation
		for _, addr := range c.universe() {
			acc := c.world[addr]
			if acc.empty() || c.rest.intn(2) == 1 {
				continue
			}
			m := &fuzzMutation{addr: addr, field: dependencyBalance}
			switch c.rest.intn(4) {
			case 0:
				m.value.AddUint64(&acc.balance, 1)
			case 1:
				m.value = *fuzzEther
			case 2:
				m.value.SetUint64(1)
			default: // Possibly zero, legal only for accounts kept alive by nonce or code.
				if acc.balance.IsZero() {
					continue
				}
				m.value.SubUint64(&acc.balance, 1)
			}
			if (acc.nonce == 0 && len(acc.code) == 0 && m.value.IsZero()) || (addr == c.payer && m.value.Lt(c.maxCost)) {
				continue
			}
			muts = append(muts, m)
		}
		got := c.checkUnchanged(t, "balance-only premise", base, c.world, muts, false)
		if want := c.world.with(muts...).balance(c.payer); got.payerBalance.Cmp(want) != 0 {
			t.Fatalf("payer balance %v, want %v", got.payerBalance, want)
		}
	})
}

// fuzzChain serves the block context of the fuzzed head. Head number 1 never
// asks the chain for ancestors.
type fuzzChain struct{ config *params.ChainConfig }

func (c fuzzChain) Config() *params.ChainConfig               { return c.config }
func (fuzzChain) CurrentHeader() *types.Header                { return nil }
func (fuzzChain) GetHeader(common.Hash, uint64) *types.Header { return nil }
func (fuzzChain) GetHeaderByNumber(uint64) *types.Header      { return nil }
func (fuzzChain) GetHeaderByHash(common.Hash) *types.Header   { return nil }
func (fuzzChain) Engine() consensus.Engine                    { return nil }

// execute applies the transaction in full at the head state, as included by
// author, and checks that it pays as simulated after a successful prefix.
func (c *fuzzCase) execute(t *testing.T, want *simResult, author common.Address, hooks *tracing.Hooks) {
	t.Helper()
	msg, err := core.TransactionToMessage(c.tx, types.MakeSigner(c.env.config, c.head.Number, c.head.Time), c.head.BaseFee)
	if err != nil {
		t.Fatal(err)
	}
	evm := vm.NewEVM(core.NewEVMBlockContext(c.head, fuzzChain{c.env.config}, &author), c.world.state(t, c.rules), c.env.config, vm.Config{Tracer: hooks})
	result, err := core.ApplyMessage(evm, msg, nil)
	if err != nil {
		t.Fatalf("accepted prefix fails execution by %v: %v\n%v", author, err, c)
	}
	if result.FramePayer == nil || *result.FramePayer != want.payer {
		t.Fatalf("execution by %v payer %v, simulated %v\n%v", author, result.FramePayer, want.payer, c)
	}
	for i := 0; i <= c.prefix.End; i++ {
		if result.FrameReceipts[i].Status != types.ReceiptStatusSuccessful {
			t.Fatalf("execution by %v prefix frame %d status %d\n%v", author, i, result.FrameReceipts[i].Status, c)
		}
	}
}

// FuzzSimulationSoundness checks that every accepted prefix also passes full
// transaction execution against the same head state, whoever authors the block.
func FuzzSimulationSoundness(f *testing.F) {
	env := newFuzzEnv(f)
	fuzzSeeds(f)
	f.Fuzz(func(t *testing.T, data []byte) {
		c := env.generate(data)
		if c == nil {
			return
		}
		base, err := c.simulate(t, c.world)
		if err != nil {
			return
		}
		// Inclusion warms the author's account (EIP-3651), which simulation
		// cannot know. Authorless execution lists the accounts the transaction
		// reaches, whose warmth matters, to prefer them as authors.
		touched := make(map[common.Address]bool)
		for i := 0; i <= c.prefix.End; i++ {
			touched[c.tx.Frames()[i].ResolvedTarget(c.sender)] = true
		}
		c.execute(t, base, common.Address{}, &tracing.Hooks{
			OnEnter: func(_ int, _ byte, _, to common.Address, _ []byte, _ uint64, _ *big.Int) {
				touched[to] = true
			},
			OnOpcode: func(_ uint64, op byte, _, _ uint64, scope tracing.OpContext, _ []byte, _ int, _ error) {
				switch vm.OpCode(op) {
				case vm.EXTCODESIZE, vm.EXTCODECOPY, vm.EXTCODEHASH:
					if stack := scope.StackData(); len(stack) > 0 {
						touched[common.Address(stack[len(stack)-1].Bytes20())] = true
					}
				}
			},
		})
		authors := slices.DeleteFunc(slices.Clone(c.universe()), func(addr common.Address) bool {
			if touched[addr] {
				return false
			}
			// Resolving a delegation loads the delegate's account.
			for account := range touched {
				if acc := c.world[account]; acc != nil {
					if delegate, ok := types.ParseDelegation(acc.code); ok && delegate == addr {
						return false
					}
				}
			}
			return true
		})
		if len(authors) == 0 || c.rest.oneIn(4) {
			authors = c.universe()
		}
		c.execute(t, base, fuzzPick(c.rest, authors...), nil)
	})
}
