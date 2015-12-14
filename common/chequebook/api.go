package chequebook

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

const Version = "1.0"

type Api struct {
	ch *Chequebook
}

func NewApi(ch *Chequebook) *Api {
	return &Api{ch}
}

func (self *Api) Issue(beneficiary common.Address, amount *big.Int) (cheque *Cheque, err error) {
	return self.ch.Issue(beneficiary, amount)
}

func (self *Api) Cash(cheque *Cheque) (txhash string, err error) {
	return self.ch.Cash(cheque)
}

func (self *Api) Deposit(amount *big.Int) (txhash string, err error) {
	return self.ch.Deposit(amount)
}
