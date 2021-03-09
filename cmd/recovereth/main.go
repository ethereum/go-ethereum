// Copyright 2018 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"golang.org/x/crypto/pbkdf2"
	"golang.org/x/crypto/scrypt"
	"gopkg.in/urfave/cli.v1"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"bufio"
)

var (
	app            = cli.NewApp()
	passphraseFlag = cli.StringFlag{
		Name:  "passwordfile",
		Usage: "the file that contains the passphrase for the keyfile",
	}
)

func init() {
	app.Name = "recovereth"
	app.Usage = "Attempts to recover wallets"
	app.Flags = []cli.Flag{
		passphraseFlag,
	}
	app.Action = recover
	app.Commands = []cli.Command{}
}

type encryptedKeyJSONV3 struct {
	Address string     `json:"address"`
	Crypto  cryptoJSON `json:"crypto"`
	Id      string     `json:"id"`
	Version int        `json:"version"`
}

type cryptoJSON struct {
	Cipher       string                 `json:"cipher"`
	CipherText   string                 `json:"ciphertext"`
	CipherParams cipherparamsJSON       `json:"cipherparams"`
	KDF          string                 `json:"kdf"`
	KDFParams    map[string]interface{} `json:"kdfparams"`
	MAC          string                 `json:"mac"`
}
type cipherparamsJSON struct {
	IV string `json:"iv"`
}

// flipBitInString flips bit i in the hex-string
func flipBitInString(bit uint, text string) string {
	bytearray := common.Hex2Bytes(text)
	b := bytearray[bit/8]
	b ^= 0x80 >> (bit % 8)
	bytearray[bit/8] = b
	return hex.EncodeToString(bytearray)
}

// flipBit flips bit i in the given cryptoJSON
func flipBit(j cryptoJSON, i uint) (cryptoJSON, error) {

	if i/8 < uint(len(j.CipherText))/2 {
		newText := flipBitInString(i, j.CipherText)
		j.CipherText = newText
		return j, nil
	}
	i -= uint(len(j.CipherText)) * 4

	if i/8 < uint(len(j.CipherParams.IV))/2 {
		newIV := flipBitInString(i, j.CipherParams.IV)
		j.CipherParams.IV = newIV
		return j, nil
	}

	i -= uint(len(j.CipherParams.IV)) * 4

	// salt 32 bytes
	salt, _ := j.KDFParams["salt"].(string)
	if i/8 < uint(len(salt))/2 {
		newSalt := flipBitInString(i, salt)
		j.KDFParams["salt"] = newSalt
		return j, nil
	}
	return j, fmt.Errorf("Exhausted")
}

func ensureInt(x interface{}) int {
	res, ok := x.(int)
	if !ok {
		res = int(x.(float64))
	}
	return res
}


func getKDFKey(cryptoJSON cryptoJSON, auth string) ([]byte, error) {
	authArray := []byte(auth)
	salt, err := hex.DecodeString(cryptoJSON.KDFParams["salt"].(string))
	if err != nil {
		return nil, err
	}
	dkLen := ensureInt(cryptoJSON.KDFParams["dklen"])
	if cryptoJSON.KDF == "scrypt" {
		n := ensureInt(cryptoJSON.KDFParams["n"])
		r := ensureInt(cryptoJSON.KDFParams["r"])
		p := ensureInt(cryptoJSON.KDFParams["p"])
		return scrypt.Key(authArray, salt, n, r, p, dkLen)

	} else if cryptoJSON.KDF == "pbkdf2" {
		c := ensureInt(cryptoJSON.KDFParams["c"])
		prf := cryptoJSON.KDFParams["prf"].(string)
		if prf != "hmac-sha256" {
			return nil, fmt.Errorf("Unsupported PBKDF2 PRF: %s", prf)
		}
		key := pbkdf2.Key(authArray, salt, c, dkLen, sha256.New)
		return key, nil
	}
	return nil, fmt.Errorf("Unsupported KDF: %s", cryptoJSON.KDF)
}

func aesCTRXOR(key, inText, iv []byte) ([]byte, error) {
	aesBlock, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	stream := cipher.NewCTR(aesBlock, iv)
	outText := make([]byte, len(inText))
	stream.XORKeyStream(outText, inText)
	return outText, err
}

// decryptKeyV3x decrypts does not validate MAC during decryption
func decryptKeyV3x(crypto cryptoJSON, auth string) (keyBytes []byte, err error) {
	//keyId = uuid.Parse(keyProtected.Id)

	iv, err := hex.DecodeString(crypto.CipherParams.IV)
	if err != nil {
		return nil, err
	}

	cipherText, err := hex.DecodeString(crypto.CipherText)
	if err != nil {
		return nil, err
	}

	derivedKey, err := getKDFKey(crypto, auth)
	if err != nil {
		return nil, err
	}

	plainText, err := aesCTRXOR(derivedKey[:16], cipherText, iv)
	if err != nil {
		return nil, err
	}
	return plainText, err
}

func testPasswordPrefixes(keyjson, auth string) error {
	l := []string{"", "\n", "\r\n", " ", "\r"}
	k := new(encryptedKeyJSONV3)
	if err := json.Unmarshal([]byte(keyjson), k); err != nil {
		log.Fatal(err)
	}
	addr := common.HexToAddress(k.Address)
	for _, prefix := range l {
		for _, suffix := range l {
			pw := fmt.Sprintf("%v%v%v", prefix, auth, suffix)
			keyBytes, err := decryptKeyV3x(k.Crypto, pw)
			fmt.Printf("auth: %x\n", pw)
			if err != nil {
				return err
			}
			key := crypto.ToECDSAUnsafe(keyBytes)
			if err != nil {
				return err
			}
			recovered := crypto.PubkeyToAddress(key.PublicKey)
			fmt.Printf("found address %s (looking for %s)\n", recovered.Hex(), addr.Hex())
			if recovered == addr {
				fmt.Printf("Recovery successfull!\n")
				fmt.Printf("Recovered password: \n------------\n%v\nHex: 0x%x\n------------\n", pw, pw)
				fmt.Printf("Please report to the go-ethereum repository "+
					"that you were able to recover a wallet with prefix 0x%x and suffix 0x%x.", prefix, suffix)
				return nil
			}
		}
	}
	return errors.New("Exhausted")
}
func testPasswordCrops(keyjson, auth string) error {
	k := new(encryptedKeyJSONV3)
	if err := json.Unmarshal([]byte(keyjson), k); err != nil {
		log.Fatal(err)
	}
	addr := common.HexToAddress(k.Address)
	for i:= 0 ; i < len(auth); i++ {
		pw := auth[:i]
		keyBytes, err := decryptKeyV3x(k.Crypto, pw)
		fmt.Printf("auth: %x\n", pw)
		if err != nil {
			return err
		}
		key := crypto.ToECDSAUnsafe(keyBytes)
		if err != nil {
			return err
		}
		recovered := crypto.PubkeyToAddress(key.PublicKey)
		fmt.Printf("found address %s (looking for %s)\n", recovered.Hex(), addr.Hex())
		if recovered == addr {
			fmt.Printf("Recovery successfull!\n")
			fmt.Printf("Recovered password: \n------------\n%v\nHex: 0x%x\n------------\n", pw, pw)
			fmt.Printf("Please report to the go-ethereum repository "+
				"that you were able to recover a wallet by cropping %d-length password to length %d ", len(auth), len(pw))
			return nil
		}
	}
	return errors.New("Exhausted")
}

func testRecoveryBitFlip(keyjson, auth string, i uint) error {
	// i = 0 starts on the Ciphertext
	// i = 256 starts on the IV
	// i = 384 starts on the SALT
	for ; ; i++ {
		k := new(encryptedKeyJSONV3)

		if err := json.Unmarshal([]byte(keyjson), k); err != nil {
			return err
		}
		addr := common.HexToAddress(k.Address)

		k2, err := flipBit(k.Crypto, i)
		if err != nil {
			return err
		}

		fmt.Printf("C  %s -> %s \n", k.Crypto.CipherText, k2.CipherText)
		fmt.Printf("IV %s -> %s \n", k.Crypto.CipherParams.IV, k2.CipherParams.IV)
		fmt.Printf("Sa %s -> %s \n", k.Crypto.KDFParams["salt"].(string), k2.KDFParams["salt"].(string))

		keyBytes, err := decryptKeyV3x(k2, auth)
		if err != nil {
			return err
		}
		key := crypto.ToECDSAUnsafe(keyBytes)
		if err != nil {
			return err
		}
		recovered := crypto.PubkeyToAddress(key.PublicKey)
		fmt.Printf("found address %s (looking for %s)\n", recovered.Hex(), addr.Hex())
		if recovered == addr {
			fmt.Printf("Recovery successfull!\n")
			json.Marshal(k2)
			json, _ := json.MarshalIndent(k2, "", "    ")
			fmt.Printf("Recovered json-file: \n------------\n%v\n------------\n", string(json))
			fmt.Printf("Please report to the go-ethereum repository "+
				"that you were able to recover a wallet by 'bitflip'-manipulation at bit %d.", i)
			return nil
		}
	}
}

func testDerivedKeyBitflip(keyjson, auth string) error {
	k := new(encryptedKeyJSONV3)
	if err := json.Unmarshal([]byte(keyjson), k); err != nil {
		log.Fatal(err)
	}
	addr := common.HexToAddress(k.Address)
	iv, _ := hex.DecodeString(k.Crypto.CipherParams.IV)
	cipherText, _ := hex.DecodeString(k.Crypto.CipherText)
	derivedKey, _ := getKDFKey(k.Crypto, auth)

	derivedKeyHex := common.Bytes2Hex(derivedKey)
	fmt.Printf("Derived key %s\n", derivedKeyHex)
	var i uint
	for i = 0; i < 256; i++ {
		corruptedKeyHex := flipBitInString(i, derivedKeyHex)
		corruptedKey := common.Hex2Bytes(corruptedKeyHex)
		plainText, err := aesCTRXOR(corruptedKey[:16], cipherText, iv)
		key := crypto.ToECDSAUnsafe(plainText)

		if err != nil {
			return err
		}
		recovered := crypto.PubkeyToAddress(key.PublicKey)
		fmt.Printf("found address %s (looking for %s)\n", recovered.Hex(), addr.Hex())
		if recovered == addr {
			fmt.Printf("Recovery successfull!\n")
			fmt.Printf("Recovered privatekey: \n------------\n0x%x\n------------\n", plainText)
			fmt.Printf("Please report to the go-ethereum repository "+
				"that you were able to recover a wallet by 'derived-key bitflip-manipulation' at bit %d.", i)
			return nil
		}
	}
	return errors.New("Exhausted")
}

func main() {
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// confirm displays a text and asks for user confirmation
func confirm(text string) bool {
	fmt.Printf(text)
	fmt.Printf("\nEnter 'ok' to proceed:\n>")

	text, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		log.Fatalf("Failed to read user input", "err", err)
	}

	if text := strings.TrimSpace(text); text == "ok" {
		return true
	}
	return false
}


func recover(ctx *cli.Context) error {

	infotext := `This software comes without any guarantees. Warning: This tool will print out sensitive data on the terminal output. Do not copy-paste information from the terminal into 
e.g. github tickets, unless you know what you are doing

This is not a brute-forcer optimized for cracking wallets. In fact, it is actually slower than geth, since this tool does not 
exit early on MAC errors. It does some basic checks for typical errors, and also checks for bitflip permutations on the wallet fields.

The usecase for this tool is when you are quite certain already that you know the password, and you believe that the wallet itself is corrupt.`

	if ok:=	confirm(infotext); !ok{
		return errors.New("User exited")
	}

	keyfilepath := ctx.Args().First()

	// Read key from file.
	keyjson, err := ioutil.ReadFile(keyfilepath)
	if err != nil {
		utils.Fatalf("Failed to read the keyfile at '%s': %v", keyfilepath, err)
	}

	passphraseFile := ctx.String(passphraseFlag.Name)
	if passphraseFile == "" {
		utils.Fatalf("Failed to read passphrase file '%s': %v", passphraseFile, err)
	}
	content, err := ioutil.ReadFile(passphraseFile)
	if err != nil {
		utils.Fatalf("Failed to read passphrase file '%s': %v", passphraseFile, err)
	}
	pw := strings.TrimRight(string(content), "\r\n")
	if err := testDerivedKeyBitflip(string(keyjson), pw); err == nil{
		return nil
	}
	if err := testPasswordPrefixes(string(keyjson), pw); err == nil{
		return nil
	}
	if err := testPasswordCrops(string(keyjson), pw); err == nil{
		return nil
	}
	if err := testRecoveryBitFlip(string(keyjson), pw,0); err == nil{
		return nil
	}

	fmt.Printf("\n\nRecovery failed.\n")

	return nil
}
