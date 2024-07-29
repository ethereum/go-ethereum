package native

import (
	"encoding/json"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/holiman/uint256"
	"math/big"
	"regexp"
	"strconv"
	"strings"
)

func init() {
	tracers.DefaultDirectory.Register("rip7560Validation", newRip7560Tracer, false)
}

/******* taken from ERC-4337 bundler collector tracer  *******/

type partialStack = []*uint256.Int

type lastThreeOpCodesItem struct {
	Opcode    string
	StackTop3 partialStack
}

type contractSizeVal struct {
	ContractSize int    `json:"contractSize"`
	Opcode       string `json:"opcode"`
}

type access struct {
	Reads           map[string]string `json:"reads"`
	Writes          map[string]uint64 `json:"writes"`
	TransientReads  map[string]uint64 `json:"transientReads"`
	TransientWrites map[string]uint64 `json:"transientWrites"`
}

// note - this means an individual 'frame' in 7560 (validate, execute, postOp)
type entryPointCall struct {
	//TopLevelMethodSig     hexutil.Bytes                       `json:"topLevelMethodSig"`
	TopLevelTargetAddress common.Address                      `json:"topLevelTargetAddress"`
	Access                map[common.Address]*access          `json:"access"`
	Opcodes               map[string]uint64                   `json:"opcodes"`
	ExtCodeAccessInfo     map[common.Address]string           `json:"extCodeAccessInfo"`
	ContractSize          map[common.Address]*contractSizeVal `json:"contractSize"`
	OOG                   bool                                `json:"oog"`
}

/******* *******/

const ValidationFramesMaxCount = 3

func newRip7560Tracer(ctx *tracers.Context, cfg json.RawMessage) (*tracers.Tracer, error) {
	var config prestateTracerConfig
	if cfg != nil {
		if err := json.Unmarshal(cfg, &config); err != nil {
			return nil, err
		}
	}
	allowedOpcodeRegex, err := regexp.Compile(
		`^(DUP\d+|PUSH\d+|SWAP\d+|POP|ADD|SUB|MUL|DIV|EQ|LTE?|S?GTE?|SLT|SH[LR]|AND|OR|NOT|ISZERO)$`,
	)
	if err != nil {
		return nil, err
	}
	// TODO FIX mock fields
	t := &rip7560ValidationTracer{
		TraceResults: make([]stateMap, ValidationFramesMaxCount),
		UsedOpcodes:  make([]map[string]bool, ValidationFramesMaxCount),
		Created:      make([]map[common.Address]bool, ValidationFramesMaxCount),
		//Deleted:      make([]map[common.Address]bool, ValidationFramesMaxCount),

		allowedOpcodeRegex: allowedOpcodeRegex,
		lastThreeOpCodes:   make([]*lastThreeOpCodesItem, 0),
		CurrentLevel:       nil,
		lastOp:             "",
		Calls:              make([]*callsItem, 0),
		Keccak:             make([]hexutil.Bytes, 0),
		Logs:               make([]*logsItem, 0),
	}

	return &tracers.Tracer{
		Hooks: &tracing.Hooks{
			OnEnter:   t.OnEnter,
			OnTxStart: t.OnTxStart,
			OnTxEnd:   t.OnTxEnd,
			OnOpcode:  t.OnOpcode,
			OnExit:    t.OnExit,
		},
		GetResult: t.GetResult,
		Stop:      t.Stop,
	}, nil
}

type callsItem struct {
	// Common
	Type string `json:"type"`

	// Enter info
	From   common.Address `json:"from"`
	To     common.Address `json:"to"`
	Method hexutil.Bytes  `json:"method"`
	Value  *hexutil.Big   `json:"value"`
	Gas    uint64         `json:"gas"`

	// Exit info
	GasUsed uint64        `json:"gasUsed"`
	Data    hexutil.Bytes `json:"data"`
}

type logsItem struct {
	Data  hexutil.Bytes   `json:"data"`
	Topic []hexutil.Bytes `json:"topic"`
}

// Array fields contain of all access details of all validation frames
type rip7560ValidationTracer struct {
	//rip7560TxData *types.Rip7560AccountAbstractionTx

	env          *tracing.VMContext
	TraceResults []stateMap                `json:"traceResults"`
	UsedOpcodes  []map[string]bool         `json:"usedOpcodes"`
	Created      []map[common.Address]bool `json:"created"`
	//Deleted      []map[common.Address]bool `json:"deleted"`

	lastThreeOpCodes    []*lastThreeOpCodesItem
	allowedOpcodeRegex  *regexp.Regexp `json:"allowedOpcodeRegex,omitempty"`
	CurrentLevel        *entryPointCall
	lastOp              string
	CallsFromEntryPoint []*entryPointCall `json:"callsFromEntryPoint,omitempty"`
	Keccak              []hexutil.Bytes   `json:"keccak"`
	Calls               []*callsItem      `json:"calls"`
	Logs                []*logsItem       `json:"logs"`

	// todo
	//interrupt atomic.Bool // Atomic flag to signal execution interruption
	//reason    error       // Textual reason for the interruption
}

func (b *rip7560ValidationTracer) OnEnter(depth int, typ byte, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
	if depth == 0 {
		b.createNewTopLevelFrame(to)
	}
	b.Calls = append(b.Calls, &callsItem{
		Type: vm.OpCode(typ).String(),
		From: from,
		To:   to,
		//Method: input[0:10],
		Value: (*hexutil.Big)(value),
		Gas:   gas,
		Data:  input,
	})
}

func (b *rip7560ValidationTracer) OnExit(depth int, output []byte, gasUsed uint64, err error, reverted bool) {
	typ := "RETURN"
	if err != nil {
		typ = "REVERT"
	}
	b.Calls = append(b.Calls, &callsItem{
		Type:    typ,
		GasUsed: gasUsed,
		Data:    output,
	})
}

func (b *rip7560ValidationTracer) OnTxStart(env *tracing.VMContext, tx *types.Transaction, from common.Address) {
	b.env = env
	//b.rip7560TxData = tx.Rip7560TransactionData()
}

func (b *rip7560ValidationTracer) createNewTopLevelFrame(addr common.Address) {
	b.CurrentLevel = &entryPointCall{
		TopLevelTargetAddress: addr,
		Access:                map[common.Address]*access{},
		Opcodes:               map[string]uint64{},
		ExtCodeAccessInfo:     map[common.Address]string{},
		ContractSize:          map[common.Address]*contractSizeVal{},
		OOG:                   false,
	}
	b.CallsFromEntryPoint = append(b.CallsFromEntryPoint, b.CurrentLevel)
	b.lastOp = ""
	return
}

func (b *rip7560ValidationTracer) OnTxEnd(receipt *types.Receipt, err error) {
}

func (b *rip7560ValidationTracer) OnOpcode(pc uint64, op byte, gas, cost uint64, scope tracing.OpContext, rData []byte, depth int, err error) {
	opcode := vm.OpCode(op).String()

	stackSize := len(scope.StackData())
	stackTop3 := partialStack{}
	for i := 0; i < 3 && i < stackSize; i++ {
		stackTop3 = append(stackTop3, StackBack(scope.StackData(), i))
	}
	b.lastThreeOpCodes = append(b.lastThreeOpCodes, &lastThreeOpCodesItem{
		Opcode:    opcode,
		StackTop3: stackTop3,
	})
	if len(b.lastThreeOpCodes) > 3 {
		b.lastThreeOpCodes = b.lastThreeOpCodes[1:]
	}

	if gas < cost || (opcode == "SSTORE" && gas < 2300) {
		b.CurrentLevel.OOG = true
	}

	if opcode == "REVERT" || opcode == "RETURN" {
		// exit() is not called on top-level return/revert, so we reconstruct it from opcode
		if depth == 1 {
			// TODO: uncomment and fix with StackBack
			//ofs := scope.Stack.Back(0).ToBig().Int64()
			//len := scope.Stack.Back(1).ToBig().Int64()
			//data := scope.Memory.GetCopy(ofs, len)
			//b.Calls = append(b.Calls, &callsItem{
			//	Type:    opcode,
			//	GasUsed: 0,
			//	Data:    data,
			//})
		}
		// NOTE: flushing all history after RETURN
		b.lastThreeOpCodes = []*lastThreeOpCodesItem{}
	}

	// not pasting the new "entryPointCall" detection here - not necessary for 7560

	var lastOpInfo *lastThreeOpCodesItem
	if len(b.lastThreeOpCodes) >= 2 {
		lastOpInfo = b.lastThreeOpCodes[len(b.lastThreeOpCodes)-2]
	}
	// store all addresses touched by EXTCODE* opcodes
	if lastOpInfo != nil && strings.HasPrefix(lastOpInfo.Opcode, "EXT") {
		addr := common.HexToAddress(lastOpInfo.StackTop3[0].Hex())
		ops := []string{}
		for _, item := range b.lastThreeOpCodes {
			ops = append(ops, item.Opcode)
		}
		last3OpcodeStr := strings.Join(ops, ",")

		// only store the last EXTCODE* opcode per address - could even be a boolean for our current use-case
		// [OP-051]
		if !strings.Contains(last3OpcodeStr, ",EXTCODESIZE,ISZERO") {
			b.CurrentLevel.ExtCodeAccessInfo[addr] = opcode
		}
	}

	// [OP-041]
	if b.isEXTorCALL(opcode) {
		n := 0
		if !strings.HasPrefix(opcode, "EXT") {
			n = 1
		}
		addr := common.BytesToAddress(StackBack(scope.StackData(), n).Bytes())

		if _, ok := b.CurrentLevel.ContractSize[addr]; !ok && !b.isAllowedPrecompile(addr) {
			b.CurrentLevel.ContractSize[addr] = &contractSizeVal{
				ContractSize: len(b.env.StateDB.GetCode(addr)),
				Opcode:       opcode,
			}
		}
	}

	// [OP-012]
	if b.lastOp == "GAS" && !strings.Contains(opcode, "CALL") {
		b.incrementCount(b.CurrentLevel.Opcodes, "GAS")
	}
	// ignore "unimportant" opcodes
	if opcode != "GAS" && !b.allowedOpcodeRegex.MatchString(opcode) {
		b.incrementCount(b.CurrentLevel.Opcodes, opcode)
	}
	b.lastOp = opcode

	if opcode == "SLOAD" || opcode == "SSTORE" || opcode == "TLOAD" || opcode == "TSTORE" {
		slot := common.BytesToHash(StackBack(scope.StackData(), 0).Bytes())
		slotHex := slot.Hex()
		addr := scope.Address()
		if _, ok := b.CurrentLevel.Access[addr]; !ok {
			b.CurrentLevel.Access[addr] = &access{
				Reads:           map[string]string{},
				Writes:          map[string]uint64{},
				TransientReads:  map[string]uint64{},
				TransientWrites: map[string]uint64{},
			}
		}
		access := *b.CurrentLevel.Access[addr]

		if opcode == "SLOAD" {
			// read slot values before this UserOp was created
			// (so saving it if it was written before the first read)
			_, rOk := access.Reads[slotHex]
			_, wOk := access.Writes[slotHex]
			if !rOk && !wOk {
				access.Reads[slotHex] = b.env.StateDB.GetState(addr, slot).Hex()
			}
		} else if opcode == "SSTORE" {
			b.incrementCount(access.Writes, slotHex)
		} else if opcode == "TLOAD" {
			b.incrementCount(access.TransientReads, slotHex)
		} else if opcode == "TSTORE" {
			b.incrementCount(access.TransientWrites, slotHex)
		}
	}

	if opcode == "KECCAK256" {
		// TODO: uncomment and fix with StackBack
		// collect keccak on 64-byte blocks
		ofs := StackBack(scope.StackData(), 0)
		len := StackBack(scope.StackData(), 1)
		memory := scope.MemoryData()
		//	// currently, solidity uses only 2-word (6-byte) for a key. this might change..still, no need to
		//	// return too much
		if len.Uint64() > 20 && len.Uint64() < 512 {
			keccak := make([]byte, len.Uint64())
			copy(keccak, memory[ofs.Uint64():ofs.Uint64()+len.Uint64()])
			b.Keccak = append(b.Keccak, keccak)
		}
	} else if strings.HasPrefix(opcode, "LOG") {
		count, _ := strconv.Atoi(opcode[3:])
		ofs := StackBack(scope.StackData(), 0)
		len := StackBack(scope.StackData(), 1)
		memory := scope.MemoryData()
		topics := []hexutil.Bytes{}
		for i := 0; i < count; i++ {
			topics = append(topics, StackBack(scope.StackData(), 2+i).Bytes())
			//topics = append(topics, scope.Stack.Back(2+i).Bytes())
		}
		log := make([]byte, len.Uint64())
		copy(log, memory[ofs.Uint64():ofs.Uint64()+len.Uint64()])
		b.Logs = append(b.Logs, &logsItem{
			Data:  log,
			Topic: topics,
		})
	}
}

// StackBack returns the n-th item in stack
func StackBack(stackData []uint256.Int, n int) *uint256.Int {
	return &stackData[len(stackData)-n-1]
}

func (b *rip7560ValidationTracer) isEXTorCALL(opcode string) bool {
	return strings.HasPrefix(opcode, "EXT") ||
		opcode == "CALL" ||
		opcode == "CALLCODE" ||
		opcode == "DELEGATECALL" ||
		opcode == "STATICCALL"
}

// not using 'isPrecompiled' to only allow the ones defined by the ERC-7562 as stateless precompiles
// [OP-062]
func (b *rip7560ValidationTracer) isAllowedPrecompile(addr common.Address) bool {
	addrInt := addr.Big()
	return addrInt.Cmp(big.NewInt(0)) == 1 && addrInt.Cmp(big.NewInt(10)) == -1
}

func (b *rip7560ValidationTracer) incrementCount(m map[string]uint64, k string) {
	if _, ok := m[k]; !ok {
		m[k] = 0
	}
	m[k]++
}

func (b *rip7560ValidationTracer) GetResult() (json.RawMessage, error) {
	jsonResult, err := json.MarshalIndent(*b, "", "    ")
	return jsonResult, err
}

func (b *rip7560ValidationTracer) Stop(err error) {
}
