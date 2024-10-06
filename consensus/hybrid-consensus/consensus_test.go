package hybridconsensus

import (
    "testing"
)

func TestCreateBlock(t *testing.T) {
    validators := []Validator{
        {Address: "0xValidator1", Stake: 100},
        {Address: "0xValidator2", Stake: 200},
    }
    
    hc := NewHybridConsensus(validators)

    block, err := hc.CreateBlock("0xValidator1")
    if err != nil {
        t.Errorf("expected no error, got %v", err)
    }
    
    if block.Number != 1 {
        t.Errorf("expected block number 1, got %d", block.Number)
    }
}

