package main

import (
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"gopkg.in/urfave/cli.v1"
)

type outputSign struct {
	Signature string
}

var commandSignMessage = cli.Command{
	Name:      "signmessage",
	Usage:     "sign a message",
	ArgsUsage: "<keyfile> <message/file>",
	Description: `
Sign the message with a keyfile.
It is possible to refer to a file containing the message.`,
	Flags: []cli.Flag{
		passphraseFlag,
		jsonFlag,
	},
	Action: func(ctx *cli.Context) error {
		keyfilepath := ctx.Args().First()
		message := []byte(ctx.Args().Get(1))

		// Load the keyfile.
		keyjson, err := ioutil.ReadFile(keyfilepath)
		if err != nil {
			utils.Fatalf("Failed to read the keyfile at '%s': %v",
				keyfilepath, err)
		}

		// Decrypt key with passphrase.
		passphrase := getPassPhrase(ctx, false)
		key, err := keystore.DecryptKey(keyjson, passphrase)
		if err != nil {
			utils.Fatalf("Error decrypting key: %v", err)
		}

		if len(message) == 0 {
			utils.Fatalf("A message must be provided")
		}
		// Read message if file.
		if _, err := os.Stat(string(message)); err == nil {
			message, err = ioutil.ReadFile(string(message))
			if err != nil {
				utils.Fatalf("Failed to read the message file: %v", err)
			}
		}

		signature, err := crypto.Sign(signHash(message), key.PrivateKey)
		if err != nil {
			utils.Fatalf("Failed to sign message: %v", err)
		}

		out := outputSign{
			Signature: hex.EncodeToString(signature),
		}
		if ctx.Bool(jsonFlag.Name) {
			mustPrintJSON(out)
		} else {
			fmt.Println("Signature: ", out.Signature)
		}
		return nil
	},
}

type outputVerify struct {
	Success            bool
	RecoveredAddress   string
	RecoveredPublicKey string
}

var commandVerifyMessage = cli.Command{
	Name:      "verifymessage",
	Usage:     "verify the signature of a signed message",
	ArgsUsage: "<address> <signature> <message/file>",
	Description: `
Verify the signature of the message.
It is possible to refer to a file containing the message.`,
	Flags: []cli.Flag{
		jsonFlag,
	},
	Action: func(ctx *cli.Context) error {
		addressStr := ctx.Args().First()
		signatureHex := ctx.Args().Get(1)
		message := []byte(ctx.Args().Get(2))

		// Determine whether it is a keyfile, public key or address.
		if !common.IsHexAddress(addressStr) {
			utils.Fatalf("Invalid address: %s", addressStr)
		}
		address := common.HexToAddress(addressStr)

		signature, err := hex.DecodeString(signatureHex)
		if err != nil {
			utils.Fatalf("Signature encoding is not hexadecimal: %v", err)
		}

		if len(message) == 0 {
			utils.Fatalf("A message must be provided")
		}
		// Read message if file.
		if _, err := os.Stat(string(message)); err == nil {
			message, err = ioutil.ReadFile(string(message))
			if err != nil {
				utils.Fatalf("Failed to read the message file: %v", err)
			}
		}

		recoveredPubkey, err := crypto.SigToPub(signHash(message), signature)
		if err != nil || recoveredPubkey == nil {
			utils.Fatalf("Signature verification failed: %v", err)
		}
		recoveredPubkeyBytes := crypto.FromECDSAPub(recoveredPubkey)
		recoveredAddress := crypto.PubkeyToAddress(*recoveredPubkey)

		success := address == recoveredAddress

		out := outputVerify{
			Success:            success,
			RecoveredPublicKey: hex.EncodeToString(recoveredPubkeyBytes),
			RecoveredAddress:   strings.ToLower(recoveredAddress.Hex()),
		}
		if ctx.Bool(jsonFlag.Name) {
			mustPrintJSON(out)
		} else {
			if out.Success {
				fmt.Println("Signature verification successful!")
			} else {
				fmt.Println("Signature verification failed!")
			}
			fmt.Println("Recovered public key: ", out.RecoveredPublicKey)
			fmt.Println("Recovered address: ", out.RecoveredAddress)
		}
		return nil
	},
}
