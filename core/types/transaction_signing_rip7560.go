package types

import (
	"github.com/ethereum/go-ethereum/common"
	"math/big"
)

type rip7560Signer struct{ londonSigner }

func NewRIP7560Signer(chainId *big.Int) Signer {
	return rip7560Signer{londonSigner{eip2930Signer{NewEIP155Signer(chainId)}}}
}

func (s rip7560Signer) Sender(tx *Transaction) (common.Address, error) {
	if tx.Type() != Rip7560Type {
		return s.londonSigner.Sender(tx)
	}
	return [20]byte{}, nil
}

// Hash returns the hash to be signed by the sender.
// It does not uniquely identify the transaction.
func (s rip7560Signer) Hash(tx *Transaction) common.Hash {
	if tx.Type() != Rip7560Type {
		return s.londonSigner.Hash(tx)
	}
	aatx := tx.Rip7560TransactionData()
	return prefixedRlpHash(
		tx.Type(),
		[]interface{}{
			s.chainId,
			tx.GasTipCap(),
			tx.GasFeeCap(),
			tx.Gas(),
			//tx.To(),
			tx.Data(),
			tx.AccessList(),

			aatx.Sender,
			aatx.PaymasterData,
			aatx.DeployerData,
			aatx.BuilderFee,
			aatx.ValidationGas,
			aatx.PaymasterGas,
		})
}
