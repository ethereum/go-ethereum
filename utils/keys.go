package utils

import (
	"fmt"
	"github.com/ethereum/eth-go/ethutil"
	"github.com/obscuren/secp256k1-go"
)

func CreateKeyPair(force bool) {
	data, _ := ethutil.Config.Db.Get([]byte("KeyRing"))
	if len(data) == 0 || force {
		pub, prv := secp256k1.GenerateKeyPair()
		pair := &ethutil.Key{PrivateKey: prv, PublicKey: pub}
		ethutil.Config.Db.Put([]byte("KeyRing"), pair.RlpEncode())
		mne := ethutil.MnemonicEncode(ethutil.Hex(prv))

		fmt.Printf(`
Generating new address and keypair.
Please keep your keys somewhere save.

++++++++++++++++ KeyRing +++++++++++++++++++
addr: %x
prvk: %x
pubk: %x
++++++++++++++++++++++++++++++++++++++++++++
save these words so you can restore your account later: %s
`, pair.Address(), prv, pub, mne)

	}
}

func ImportPrivateKey(prvKey string) {
	key := ethutil.FromHex(prvKey)
	msg := []byte("tmp")
	// Couldn't think of a better way to get the pub key
	sig, _ := secp256k1.Sign(msg, key)
	pub, _ := secp256k1.RecoverPubkey(msg, sig)
	pair := &ethutil.Key{PrivateKey: key, PublicKey: pub}
	ethutil.Config.Db.Put([]byte("KeyRing"), pair.RlpEncode())

	fmt.Printf(`
Importing private key

++++++++++++++++ KeyRing +++++++++++++++++++
addr: %x
prvk: %x
pubk: %x
++++++++++++++++++++++++++++++++++++++++++++

`, pair.Address(), key, pub)
}
