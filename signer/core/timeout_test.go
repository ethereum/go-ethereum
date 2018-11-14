package core

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/internal/ethapi"
)

type nonResponsiveHander struct {
	sleepTime time.Duration
}

func (n nonResponsiveHander) List(ctx context.Context) ([]common.Address, error) {
	time.Sleep(n.sleepTime)
	return []common.Address{common.HexToAddress("0xdeadbeef")}, nil
}

func (nonResponsiveHander) New(ctx context.Context) (accounts.Account, error) {
	panic("implement me")
}

func (nonResponsiveHander) SignTransaction(ctx context.Context, args SendTxArgs, methodSelector *string) (*ethapi.SignTransactionResult, error) {
	panic("implement me")
}

func (nonResponsiveHander) Sign(ctx context.Context, addr common.MixedcaseAddress, data hexutil.Bytes) (hexutil.Bytes, error) {
	panic("implement me")
}

func (nonResponsiveHander) Export(ctx context.Context, addr common.Address) (json.RawMessage, error) {
	panic("implement me")
}

// TestTimeout checks that it does timeout
func TestTimeout(t *testing.T) {
	a := &nonResponsiveHander{1 * time.Second}
	api := NewTimedExternalAPI(a, 1*time.Millisecond)
	addr, err := api.List(nil)
	if len(addr) > 0 {
		t.Errorf("expected no accounts, got %d", len(addr))
	}
	if err == nil {
		t.Errorf("expected err")

	}
}

// TestTimeout checks that it does timeout
func TestNotTimeout(t *testing.T) {
	a := &nonResponsiveHander{1 * time.Millisecond}
	api := NewTimedExternalAPI(a, 1*time.Second)
	addr, err := api.List(nil)
	if len(addr) != 1 {
		t.Errorf("expected 1 accounts, got %d", len(addr))
	}
	if err != nil {
		t.Errorf("expected no err, got %v", err)
	}
}
