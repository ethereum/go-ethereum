package vm

import "math/big"

type ruleSet struct {
	hs *big.Int
}

func (r ruleSet) IsHomestead(n *big.Int) bool { return n.Cmp(r.hs) >= 0 }
