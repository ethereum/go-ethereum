//nolint: unused
package blockstm

import "encoding/base64"

const (
	ReadKindMap     = 0
	ReadKindStorage = 1
)

type ReadDescriptor struct {
	Path []byte
	Kind int
	V    Version
}

type WriteDescriptor struct {
	Path []byte
	V    Version
	Val  interface{}
}

type TxnInput []ReadDescriptor
type TxnOutput []WriteDescriptor

// hasNewWrite: returns true if the current set has a new write compared to the input
func (txo TxnOutput) hasNewWrite(cmpSet []WriteDescriptor) bool {
	if len(txo) == 0 {
		return false
	} else if len(cmpSet) == 0 || len(txo) > len(cmpSet) {
		return true
	}

	cmpMap := map[string]bool{base64.StdEncoding.EncodeToString(cmpSet[0].Path): true}

	for i := 1; i < len(cmpSet); i++ {
		cmpMap[base64.StdEncoding.EncodeToString(cmpSet[i].Path)] = true
	}

	for _, v := range txo {
		if !cmpMap[base64.StdEncoding.EncodeToString(v.Path)] {
			return true
		}
	}

	return false
}

type TxnInputOutput struct {
	inputs  []TxnInput
	outputs []TxnOutput
}

func (io *TxnInputOutput) readSet(txnIdx int) []ReadDescriptor {
	return io.inputs[txnIdx]
}

func (io *TxnInputOutput) writeSet(txnIdx int) []WriteDescriptor {
	return io.outputs[txnIdx]
}

func MakeTxnInputOutput(numTx int) *TxnInputOutput {
	return &TxnInputOutput{
		inputs:  make([]TxnInput, numTx),
		outputs: make([]TxnOutput, numTx),
	}
}

func (io *TxnInputOutput) recordRead(txId int, input []ReadDescriptor) {
	io.inputs[txId] = input
}

func (io *TxnInputOutput) recordWrite(txId int, output []WriteDescriptor) {
	io.outputs[txId] = output
}
