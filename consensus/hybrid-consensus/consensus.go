package hybridconsensus

import (
    "errors"
    "time"
)

type Block struct {
    Number    uint64
    Validator string
    Timestamp time.Time
    PrevHash  string
}

type Validator struct {
    Address string
    Stake   uint64
}

type HybridConsensus struct {
    validators []Validator
    blocks     []Block
}

func NewHybridConsensus(validators []Validator) *HybridConsensus {
    return &HybridConsensus{
        validators: validators,
        blocks:     []Block{},
    }
}

func (hc *HybridConsensus) CreateBlock(validator string) (*Block, error) {
    if !hc.IsValidator(validator) {
        return nil, errors.New("validator not authorized")
    }

    newBlock := Block{
        Number:    uint64(len(hc.blocks) + 1),
        Validator: validator,
        Timestamp: time.Now(),
        PrevHash:  hc.GetLatestBlockHash(),
    }

    hc.blocks = append(hc.blocks, newBlock)
    return &newBlock, nil
}

func (hc *HybridConsensus) GetLatestBlockHash() string {
    if len(hc.blocks) == 0 {
        return ""
    }
    latestBlock := hc.blocks[len(hc.blocks)-1]
    return string(latestBlock.Number)
}

func (hc *HybridConsensus) IsValidator(address string) bool {
    for _, validator := range hc.validators {
        if validator.Address == address {
            return true
        }
    }
    return false
}

