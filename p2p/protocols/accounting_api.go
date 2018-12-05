package protocols

import (
	"errors"
)

const AccountingVersion = "1.0"

var errNoAccountingMetrics = errors.New("no accounting metrics")

type AccountingApi struct {
	metrics *AccountingMetrics
}

func NewAccountingApi(m *AccountingMetrics) *AccountingApi {
	return &AccountingApi{m}
}

func (self *AccountingApi) BalanceCredit() (int64, error) {
	if self.metrics == nil {
		return 0, errNoAccountingMetrics
	}
	return mBalanceCredit.Count(), nil
}

func (self *AccountingApi) BalanceDebit() (int64, error) {
	if self.metrics == nil {
		return 0, errNoAccountingMetrics
	}
	return mBalanceDebit.Count(), nil
}

func (self *AccountingApi) BytesCredit() (int64, error) {
	if self.metrics == nil {
		return 0, errNoAccountingMetrics
	}
	return mBytesCredit.Count(), nil
}

func (self *AccountingApi) BytesDebit() (int64, error) {
	if self.metrics == nil {
		return 0, errNoAccountingMetrics
	}
	return mBytesDebit.Count(), nil
}

func (self *AccountingApi) MsgCredit() (int64, error) {
	if self.metrics == nil {
		return 0, errNoAccountingMetrics
	}
	return mMsgCredit.Count(), nil
}

func (self *AccountingApi) MsgDebit() (int64, error) {
	if self.metrics == nil {
		return 0, errNoAccountingMetrics
	}
	return mMsgDebit.Count(), nil
}

func (self *AccountingApi) PeerDrops() (int64, error) {
	if self.metrics == nil {
		return 0, errNoAccountingMetrics
	}
	return mPeerDrops.Count(), nil
}

func (self *AccountingApi) SelfDrops() (int64, error) {
	if self.metrics == nil {
		return 0, errNoAccountingMetrics
	}
	return mSelfDrops.Count(), nil
}