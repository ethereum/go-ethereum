/*
	This file is part of go-ethereum

	go-ethereum is free software: you can redistribute it and/or modify
	it under the terms of the GNU Lesser General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	go-ethereum is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU Lesser General Public License
	along with go-ethereum.  If not, see <http://www.gnu.org/licenses/>.
*/
/**
 * @authors
 * 	Gustav Simonsson <gustav.simonsson@gmail.com>
 * @date 2015
 *
 */

package crypto

import (
	"bytes"
	"code.google.com/p/go-uuid/uuid"
	"crypto/ecdsa"
	"crypto/elliptic"
	crand "crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"time"
)

type Key struct {
	Id    *uuid.UUID // Version 4 "random" for unique id not derived from key data
	Flags [4]byte    // RFU
	// we only store privkey as pubkey/address can be derived from it
	// privkey in this struct is always in plaintext
	PrivateKey *ecdsa.PrivateKey
}

type KeyPlainJSON struct {
	Id         string
	Flags      string
	PrivateKey string
}

type CipherJSON struct {
	Salt       string
	IV         string
	CipherText string
}

type KeyProtectedJSON struct {
	Id     string
	Flags  string
	Crypto CipherJSON
}

func (k *Key) Address() []byte {
	pubBytes := FromECDSAPub(&k.PrivateKey.PublicKey)
	return Sha3(pubBytes)[12:]
}

func (k *Key) MarshalJSON() (j []byte, err error) {
	stringStruct := KeyPlainJSON{
		k.Id.String(),
		hex.EncodeToString(k.Flags[:]),
		hex.EncodeToString(FromECDSA(k.PrivateKey)),
	}
	j, _ = json.Marshal(stringStruct)
	return
}

func (k *Key) UnmarshalJSON(j []byte) (err error) {
	keyJSON := new(KeyPlainJSON)
	err = json.Unmarshal(j, &keyJSON)
	if err != nil {
		return
	}

	u := new(uuid.UUID)
	*u = uuid.Parse(keyJSON.Id)
	if *u == nil {
		err = errors.New("UUID parsing failed")
		return
	}
	k.Id = u

	flagsBytes, err := hex.DecodeString(keyJSON.Flags)
	if err != nil {
		return
	}

	PrivateKeyBytes, err := hex.DecodeString(keyJSON.PrivateKey)
	if err != nil {
		return
	}

	copy(k.Flags[:], flagsBytes[0:4])
	k.PrivateKey = ToECDSA(PrivateKeyBytes)

	return
}

func NewKey() *Key {
	randBytes := GetEntropyCSPRNG(32)
	reader := bytes.NewReader(randBytes)
	_, x, y, err := elliptic.GenerateKey(S256(), reader)
	if err != nil {
		panic("key generation: elliptic.GenerateKey failed: " + err.Error())
	}
	privateKeyMarshalled := elliptic.Marshal(S256(), x, y)
	privateKeyECDSA := ToECDSA(privateKeyMarshalled)

	key := new(Key)
	id := uuid.NewRandom()
	key.Id = &id
	// flags := new([4]byte)
	// key.Flags = flags
	key.PrivateKey = privateKeyECDSA
	return key
}

// plain crypto/rand. this is /dev/urandom on Unix-like systems.
func GetEntropyCSPRNG(n int) []byte {
	mainBuff := make([]byte, n)
	_, err := io.ReadFull(crand.Reader, mainBuff)
	if err != nil {
		panic("key generation: reading from crypto/rand failed: " + err.Error())
	}
	return mainBuff
}

// TODO: verify. Do not use until properly discussed.
// we start with crypt/rand, then mix in additional sources of entropy.
// These sources are from three types: OS, go runtime and ethereum client state.
func GetEntropyTinFoilHat() []byte {
	startTime := time.Now().UnixNano()
	// for each source, we XOR in it's SHA3 hash.
	mainBuff := GetEntropyCSPRNG(32)
	// 1. OS entropy sources
	startTimeBytes := make([]byte, 32)
	binary.PutVarint(startTimeBytes, startTime)
	startTimeHash := Sha3(startTimeBytes)
	mix32Byte(mainBuff, startTimeHash)

	pid := os.Getpid()
	pidBytes := make([]byte, 32)
	binary.PutUvarint(pidBytes, uint64(pid))
	pidHash := Sha3(pidBytes)
	mix32Byte(mainBuff, pidHash)

	osEnv := os.Environ()
	osEnvBytes := []byte(strings.Join(osEnv, ""))
	osEnvHash := Sha3(osEnvBytes)
	mix32Byte(mainBuff, osEnvHash)

	// not all OS have hostname in env variables
	osHostName, err := os.Hostname()
	if err != nil {
		osHostNameBytes := []byte(osHostName)
		osHostNameHash := Sha3(osHostNameBytes)
		mix32Byte(mainBuff, osHostNameHash)
	}

	// 2. go runtime entropy sources
	memStats := new(runtime.MemStats)
	runtime.ReadMemStats(memStats)
	memStatsBytes := []byte(fmt.Sprintf("%v", memStats))
	memStatsHash := Sha3(memStatsBytes)
	mix32Byte(mainBuff, memStatsHash)

	// 3. Mix in ethereum / client state
	// TODO: list of network peers structs (IP, port, etc)
	// TODO: merkle patricia tree root hash for world state and tx list

	// 4. Yo dawg we heard you like entropy so we'll grab some entropy from how
	//    long it took to grab the above entropy. And a yield, for good measure.
	runtime.Gosched()
	diffTime := time.Now().UnixNano() - startTime
	diffTimeBytes := make([]byte, 32)
	binary.PutVarint(diffTimeBytes, diffTime)
	diffTimeHash := Sha3(diffTimeBytes)
	mix32Byte(mainBuff, diffTimeHash)

	return mainBuff
}

func mix32Byte(buff []byte, mixBuff []byte) []byte {
	for i := 0; i < 32; i++ {
		buff[i] ^= mixBuff[i]
	}
	return buff
}
