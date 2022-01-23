package types

import (
	"io"
	"sort"
	"strings"

	"github.com/scroll-tech/go-ethereum/rlp"
)

// BlockResult contains block execution traces and results required for rollers.
type BlockResult struct {
	ExecutionResults []*ExecutionResult `json:"executionResults"`
}

type rlpBlockResult struct {
	ExecutionResults []*ExecutionResult
}

func (b *BlockResult) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, &rlpBlockResult{
		ExecutionResults: b.ExecutionResults,
	})
}

func (b *BlockResult) DecodeRLP(s *rlp.Stream) error {
	var dec rlpBlockResult
	err := s.Decode(&dec)
	if err == nil {
		b.ExecutionResults = dec.ExecutionResults
	}
	return err
}

// ExecutionResult groups all structured logs emitted by the EVM
// while replaying a transaction in debug mode as well as transaction
// execution status, the amount of gas used and the return value
type ExecutionResult struct {
	Gas         uint64         `json:"gas"`
	Failed      bool           `json:"failed"`
	ReturnValue string         `json:"returnValue,omitempty"`
	StructLogs  []StructLogRes `json:"structLogs"`
}

type rlpExecutionResult struct {
	Gas         uint64
	Failed      bool
	ReturnValue string
	StructLogs  []StructLogRes
}

func (e *ExecutionResult) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, rlpExecutionResult{
		Gas:         e.Gas,
		Failed:      e.Failed,
		ReturnValue: e.ReturnValue,
		StructLogs:  e.StructLogs,
	})
}

func (e *ExecutionResult) DecodeRLP(s *rlp.Stream) error {
	var dec rlpExecutionResult
	err := s.Decode(&dec)
	if err == nil {
		e.Gas, e.Failed, e.ReturnValue, e.StructLogs = dec.Gas, dec.Failed, dec.ReturnValue, dec.StructLogs
	}
	return err
}

// StructLogRes stores a structured log emitted by the EVM while replaying a
// transaction in debug mode
type StructLogRes struct {
	Pc      uint64             `json:"pc"`
	Op      string             `json:"op"`
	Gas     uint64             `json:"gas"`
	GasCost uint64             `json:"gasCost"`
	Depth   int                `json:"depth"`
	Error   string             `json:"error,omitempty"`
	Stack   *[]string          `json:"stack,omitempty"`
	Memory  *[]string          `json:"memory,omitempty"`
	Storage *map[string]string `json:"storage,omitempty"`
}

type rlpStructLogRes struct {
	Pc      uint64
	Op      string
	Gas     uint64
	GasCost uint64
	Depth   uint
	Error   string
	Stack   []string
	Memory  []string
	Storage []string
}

// EncodeRLP implements rlp.Encoder.
func (r *StructLogRes) EncodeRLP(w io.Writer) error {
	data := rlpStructLogRes{
		Pc:      r.Pc,
		Op:      r.Op,
		Gas:     r.Gas,
		GasCost: r.GasCost,
		Depth:   uint(r.Depth),
		Error:   r.Error,
	}
	if r.Stack != nil {
		data.Stack = make([]string, len(*r.Stack))
		for i, val := range *r.Stack {
			data.Stack[i] = val
		}
	}
	if r.Memory != nil {
		data.Memory = make([]string, len(*r.Memory))
		for i, val := range *r.Memory {
			data.Memory[i] = val
		}
	}
	if r.Storage != nil {
		keys := make([]string, 0, len(*r.Storage))
		for key := range *r.Storage {
			keys = append(keys, key)
		}
		sort.Slice(keys, func(i, j int) bool {
			return strings.Compare(keys[i], keys[j]) >= 0
		})
		data.Storage = make([]string, 0, len(*r.Storage)*2)
		for _, key := range keys {
			data.Storage = append(data.Storage, []string{key, (*r.Storage)[key]}...)
		}
	}
	return rlp.Encode(w, data)
}

// DecodeRLP implements rlp.Decoder.
func (r *StructLogRes) DecodeRLP(s *rlp.Stream) error {
	var dec rlpStructLogRes
	err := s.Decode(&dec)
	if err != nil {
		return err
	}
	r.Pc, r.Op, r.Gas, r.GasCost, r.Depth, r.Error = dec.Pc, dec.Op, dec.Gas, dec.GasCost, int(dec.Depth), dec.Error
	if len(dec.Stack) != 0 {
		stack := make([]string, len(dec.Stack))
		for i, val := range dec.Stack {
			stack[i] = val
		}
		r.Stack = &stack
	}
	if len(dec.Memory) != 0 {
		memory := make([]string, len(dec.Memory))
		for i, val := range dec.Memory {
			memory[i] = val
		}
		r.Memory = &memory
	}
	if len(dec.Storage) != 0 {
		storage := make(map[string]string, len(dec.Storage)*2)
		for i := 0; i < len(dec.Storage); i += 2 {
			key, val := dec.Storage[i], dec.Storage[i+1]
			storage[key] = val
		}
		r.Storage = &storage
	}
	return nil
}
