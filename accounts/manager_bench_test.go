package accounts

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

var benchAddresses []common.Address

type fakeWallet struct {
	accounts []Account
}

func (w *fakeWallet) URL() URL {
	return URL{}
}
func (w *fakeWallet) Status() (string, error) {
	return "", nil
}
func (w *fakeWallet) Open(string) error {
	return nil
}
func (w *fakeWallet) Close() error {
	return nil
}
func (w *fakeWallet) Accounts() []Account {
	return w.accounts
}
func (w *fakeWallet) Contains(a Account) bool {
	for _, x := range w.accounts {
		if x.Address == a.Address {
			return true
		}
	}
	return false
}
func (w *fakeWallet) Derive(DerivationPath, bool) (Account, error) {
	return Account{}, nil
}
func (w *fakeWallet) SelfDerive([]DerivationPath, ethereum.ChainStateReader) {}
func (w *fakeWallet) SignData(Account, string, []byte) ([]byte, error) {
	return nil, nil
}
func (w *fakeWallet) SignDataWithPassphrase(Account, string, string, []byte) ([]byte, error) {
	return nil, nil
}
func (w *fakeWallet) SignText(Account, []byte) ([]byte, error) {
	return nil, nil
}
func (w *fakeWallet) SignTextWithPassphrase(Account, string, []byte) ([]byte, error) {
	return nil, nil
}
func (w *fakeWallet) SignTx(Account, *types.Transaction, *big.Int) (*types.Transaction, error) {
	return nil, nil
}
func (w *fakeWallet) SignTxWithPassphrase(Account, string, *types.Transaction, *big.Int) (*types.Transaction, error) {
	return nil, nil
}

func makeWallets(numWallets, accountsPerWallet int) []Wallet {
	wallets := make([]Wallet, numWallets)

	var addr common.Address
	for i := 0; i < numWallets; i++ {
		accs := make([]Account, accountsPerWallet)
		for j := 0; j < accountsPerWallet; j++ {
			addr[19]++
			accs[j] = Account{Address: addr}
		}
		wallets[i] = &fakeWallet{accounts: accs}
	}
	return wallets
}

func BenchmarkManagerAccounts(b *testing.B) {
	cases := []struct {
		name            string
		numWallets      int
		accountsPerWall int
	}{
		{"1x1", 1, 1},
		{"10x10", 10, 10},
		{"10x100", 10, 100},
		{"100x10", 100, 10},
		{"100x100", 100, 100},
	}

	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			am := &Manager{wallets: makeWallets(tc.numWallets, tc.accountsPerWall)}

			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				benchAddresses = am.Accounts()
			}
		})
	}
}
