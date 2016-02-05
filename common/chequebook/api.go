package chequebook

import (
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

const Version = "1.0"

var errNoChequebook = errors.New("no chequebook")

type Api struct {
	chequebookf func() *Chequebook
}

func NewApi(ch func() *Chequebook) *Api {
	return &Api{ch}
}

func (self *Api) Balance() (string, error) {
	ch := self.chequebookf()
	if ch == nil {
		return "", errNoChequebook
	}
	return ch.Balance().String(), nil
}

func (self *Api) Issue(beneficiary common.Address, amount *big.Int) (cheque *Cheque, err error) {
	ch := self.chequebookf()
	if ch == nil {
		return nil, errNoChequebook
	}
	return ch.Issue(beneficiary, amount)
}

func (self *Api) Cash(cheque *Cheque) (txhash string, err error) {
	ch := self.chequebookf()
	if ch == nil {
		return "", errNoChequebook
	}
	return ch.Cash(cheque)
}

func (self *Api) Deposit(amount *big.Int) (txhash string, err error) {
	ch := self.chequebookf()
	if ch == nil {
		return "", errNoChequebook
	}
	return ch.Deposit(amount)
}
