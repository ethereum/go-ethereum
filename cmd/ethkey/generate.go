package main

import (
	"crypto/ecdsa"
	"crypto/rand"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/pborman/uuid"
	"gopkg.in/urfave/cli.v1"
)

type outputGenerate struct {
	Address      string
	AddressEIP55 string
}

var commandGenerate = cli.Command{
	Name:      "generate",
	Usage:     "generate new keyfile",
	ArgsUsage: "[ <keyfile> ]",
	Description: `
Generate a new keyfile.
If you want to use an existing private key to use in the keyfile, it can be 
specified by setting --privatekey with the location of the file containing the 
private key.`,
	Flags: []cli.Flag{
		passphraseFlag,
		jsonFlag,
		cli.StringFlag{
			Name: "privatekey",
			Usage: "the file from where to read the private key to " +
				"generate a keyfile for",
		},
	},
	Action: func(ctx *cli.Context) error {
		// Check if keyfile path given and make sure it doesn't already exist.
		keyfilepath := ctx.Args().First()
		if keyfilepath == "" {
			keyfilepath = defaultKeyfileName
		}
		if _, err := os.Stat(keyfilepath); err == nil {
			utils.Fatalf("Keyfile already exists at %s.", keyfilepath)
		} else if !os.IsNotExist(err) {
			utils.Fatalf("Error checking if keyfile exists: %v", err)
		}

		var privateKey *ecdsa.PrivateKey

		// First check if a private key file is provided.
		privateKeyFile := ctx.String("privatekey")
		if privateKeyFile != "" {
			privateKeyBytes, err := ioutil.ReadFile(privateKeyFile)
			if err != nil {
				utils.Fatalf("Failed to read the private key file '%s': %v",
					privateKeyFile, err)
			}

			pk, err := crypto.HexToECDSA(string(privateKeyBytes))
			if err != nil {
				utils.Fatalf(
					"Could not construct ECDSA private key from file content: %v",
					err)
			}
			privateKey = pk
		}

		// If not loaded, generate random.
		if privateKey == nil {
			pk, err := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
			if err != nil {
				utils.Fatalf("Failed to generate random private key: %v", err)
			}
			privateKey = pk
		}

		// Create the keyfile object with a random UUID.
		id := uuid.NewRandom()
		key := &keystore.Key{
			Id:         id,
			Address:    crypto.PubkeyToAddress(privateKey.PublicKey),
			PrivateKey: privateKey,
		}

		// Encrypt key with passphrase.
		passphrase := getPassPhrase(ctx, true)
		keyjson, err := keystore.EncryptKey(key, passphrase,
			keystore.StandardScryptN, keystore.StandardScryptP)
		if err != nil {
			utils.Fatalf("Error encrypting key: %v", err)
		}

		// Store the file to disk.
		if err := os.MkdirAll(filepath.Dir(keyfilepath), 0700); err != nil {
			utils.Fatalf("Could not create directory %s", filepath.Dir(keyfilepath))
		}
		if err := ioutil.WriteFile(keyfilepath, keyjson, 0600); err != nil {
			utils.Fatalf("Failed to write keyfile to %s: %v", keyfilepath, err)
		}

		// Output some information.
		out := outputGenerate{
			Address: key.Address.Hex(),
		}
		if ctx.Bool(jsonFlag.Name) {
			mustPrintJSON(out)
		} else {
			fmt.Println("Address:       ", out.Address)
		}
		return nil
	},
}
