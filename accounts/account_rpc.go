package accounts
import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/logger"
	"time"
)

// AccountService represents a RPC service with support for account specific actions.
type AccountService struct {
	am *Manager
}

// NewAccountService creates a new Account RPC service instance.
func NewAccountService(am *Manager) *AccountService {
	return &AccountService{am: am}
}

// Accounts returns the collection of accounts this node manages
func (s *AccountService) Accounts() ([]Account, error) {
	return s.am.Accounts()
}

// PersonalService represents a RPC service with support for personal methods.
type PersonalService struct {
	am *Manager
}

// NewPersonalService creates a new RPC service with support for personal actions.
func NewPersonalService(am *Manager) *PersonalService {
	return &PersonalService{am}
}

// ListAccounts will return a list of addresses for accounts this node manages.
func (s *PersonalService) ListAccounts(password string) ([]common.Address, error) {
	accounts, err := s.am.Accounts()
	if err != nil {
		return nil, err
	}

	addresses := make([]common.Address, len(accounts))
	for i, acc := range accounts {
		addresses[i] = acc.Address
	}
	return addresses, nil
}

// NewAccount will create a new account and returns the address for the new account.
func (s *PersonalService) NewAccount(password string) (common.Address, error) {
	acc, err := s.am.NewAccount(password)
	if err == nil {
		return acc.Address, nil
	}
	return common.Address{}, err
}

// UnlockAccount will unlock the account associated with the given address with the given password for duration seconds.
// It returns an indication if the action was successful.
func (s *PersonalService) UnlockAccount(addr common.Address, password string, duration int) bool {
	if err := s.am.TimedUnlock(addr, password, time.Duration(duration) * time.Second); err != nil {
		glog.V(logger.Info).Infof("%v\n", err)
		return false
	}
	return true
}

// LockAccount will lock the account associated with the given address when it's unlocked.
func (s *PersonalService) LockAccount(addr common.Address) bool {
	return s.am.Lock(addr) == nil
}