package filters

import (
	"math/big"
	"reflect"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/rlp"
)

var emptyPayload Payload

// processStateChanges builds the state diff Payload from the modified accounts in the StateChangeEvent
func processStateChanges(event core.StateChangeEvent, crit ethereum.FilterQuery) (Payload, error) {
	var accountDiffs []AccountDiff
	block := event.Block
	// Iterate over state changes to build AccountDiffs
	for addr, modifiedAccount := range event.StateChanges {
		if len(crit.Addresses) > 0 && !includes(crit.Addresses, addr) {
			continue
		}

		a, err := buildAccountDiff(addr, modifiedAccount)
		if err != nil {
			return emptyPayload, err
		}

		accountDiffs = append(accountDiffs, a)
	}

	if len(accountDiffs) == 0 {
		return emptyPayload, nil
	}

	stateDiff := StateDiff{
		BlockNumber:     block.Number(),
		BlockHash:       block.Hash(),
		UpdatedAccounts: accountDiffs,
	}

	stateDiffRlp, err := rlp.EncodeToBytes(stateDiff)
	if err != nil {
		return emptyPayload, err
	}
	payload := Payload{
		StateDiffRlp: stateDiffRlp,
	}

	return payload, nil
}

// buildAccountDiff
func buildAccountDiff(addr common.Address, modifiedAccount state.ModifiedAccount) (AccountDiff, error) {
	emptyAccountDiff := AccountDiff{}
	accountBytes, err := rlp.EncodeToBytes(modifiedAccount.StateAccount)
	if err != nil {
		return emptyAccountDiff, err
	}

	var storageDiffs []StorageDiff
	for k, v := range modifiedAccount.Storage {
		// Storage diff value should be an RLP object too
		encodedValueRlp, err := rlp.EncodeToBytes(v[:])
		if err != nil {
			return emptyAccountDiff, err
		}
		storageKey := k
		diff := StorageDiff{
			Key:   storageKey[:],
			Value: encodedValueRlp,
		}
		storageDiffs = append(storageDiffs, diff)
	}

	address := addr
	return AccountDiff{
		Key:     address[:],
		Value:   accountBytes,
		Storage: storageDiffs,
	}, nil
}

func isPayloadEmpty(payload Payload) bool {
	return reflect.DeepEqual(payload, emptyPayload)
}

// Payload packages the data to send to statediff subscriptions
type Payload struct {
	StateDiffRlp []byte `json:"stateDiff"    gencodec:"required"`
}

// StateDiff is the final output structure from the builder
type StateDiff struct {
	BlockNumber     *big.Int      `json:"blockNumber"     gencodec:"required"`
	BlockHash       common.Hash   `json:"blockHash"       gencodec:"required"`
	UpdatedAccounts []AccountDiff `json:"updatedAccounts" gencodec:"required"`
}

// AccountDiff holds the data for a single state diff node
type AccountDiff struct {
	Key     []byte        `json:"key"         gencodec:"required"`
	Value   []byte        `json:"value"       gencodec:"required"`
	Storage []StorageDiff `json:"storage"     gencodec:"required"`
}

// StorageDiff holds the data for a single storage diff node
type StorageDiff struct {
	Key   []byte `json:"key"         gencodec:"required"`
	Value []byte `json:"value"       gencodec:"required"`
}
