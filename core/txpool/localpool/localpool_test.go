package localpool

import "errors"

func (l *LocalPool) verifyConsistency() error {
	for _, list := range l.allAccounts {
		for _, tx := range list {
			if _, ok := l.allTxs[tx.Hash()]; !ok {
				return errors.New("tx in nonceOrderedList but not in all txs")
			}
		}
	}
	for _, tx := range l.allTxs {
		found := 0
		for _, list := range l.allAccounts {
			if tx2, ok := list[tx.Nonce()]; ok {
				if tx.Hash() == tx2.Hash() {
					found++
				}
			}
		}
		if found != 1 {
			return errors.New("tx in all txs but not in nonceOrderedList")
		}
	}
	return nil
}
