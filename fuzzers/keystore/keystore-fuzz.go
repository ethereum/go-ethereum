package keystore

import (
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"os"
)

func Fuzz(input []byte) int {

	ks := keystore.NewKeyStore("/tmp/ks", keystore.LightScryptN, keystore.LightScryptP)

	a, err := ks.NewAccount(string(input))
	if err != nil {
		panic(err)
	}
	if err := ks.Unlock(a, string(input)); err != nil {
		panic(err)
	}
	os.Remove(a.URL.Path)
	return 0
}
