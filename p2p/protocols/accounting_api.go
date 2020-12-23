package protocols

import (
	"errors"
)

// Textual version number of accounting API
const AccountingVersion = "1.0"

var errNoAccountingMetrics = errors.New("accounting metrics not enabled")

// AccountingApi provides an API to access account related information
type AccountingApi struct {
	metrics *AccountingMetrics
}

// NewAccountingApi creates a new AccountingApi
// m will be used to check if accounting metrics are enabled
func NewAccountingApi(m *AccountingMetrics) *AccountingApi {
	return &AccountingApi{m}
}

// Balance returns local node balance (units credited - units debited)
func (self *AccountingApi) Balance() (int64, error) {
	if self.metrics == nil {
		return 0, errNoAccountingMetrics
	}
	balance := mBalanceCredit.Count() - mBalanceDebit.Count()
	return balance, nil
}

// BalanceCredit returns total amount of units credited by local node
func (self *AccountingApi) BalanceCredit() (int64, error) {
	if self.metrics == nil {
		return 0, errNoAccountingMetrics
	}
	return mBalanceCredit.Count(), nil
}

// BalanceCredit returns total amount of units debited by local node
func (self *AccountingApi) BalanceDebit() (int64, error) {
	if self.metrics == nil {
		return 0, errNoAccountingMetrics
	}
	return mBalanceDebit.Count(), nil
}

// BytesCredit returns total amount of bytes credited by local node
func (self *AccountingApi) BytesCredit() (int64, error) {
	if self.metrics == nil {
		return 0, errNoAccountingMetrics
	}
	return mBytesCredit.Count(), nil
}

// BalanceCredit returns total amount of bytes debited by local node
func (self *AccountingApi) BytesDebit() (int64, error) {
	if self.metrics == nil {
		return 0, errNoAccountingMetrics
	}
	return mBytesDebit.Count(), nil
}

// MsgCredit returns total amount of messages credited by local node
func (self *AccountingApi) MsgCredit() (int64, error) {
	if self.metrics == nil {
		return 0, errNoAccountingMetrics
	}
	return mMsgCredit.Count(), nil
}

// MsgDebit returns total amount of messages debited by local node
func (self *AccountingApi) MsgDebit() (int64, error) {
	if self.metrics == nil {
		return 0, errNoAccountingMetrics
	}
	return mMsgDebit.Count(), nil
}

// PeerDrops returns number of times when local node had to drop remote peers
func (self *AccountingApi) PeerDrops() (int64, error) {
	if self.metrics == nil {
		return 0, errNoAccountingMetrics
	}
	return mPeerDrops.Count(), nil
}

// SelfDrops returns number of times when local node was overdrafted and dropped
func (self *AccountingApi) SelfDrops() (int64, error) {
	if self.metrics == nil {
		return 0, errNoAccountingMetrics
	}
	return mSelfDrops.Count(), nil
}
