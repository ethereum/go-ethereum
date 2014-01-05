package main

import (
  "math/big"
  "fmt"
  "github.com/obscuren/secp256k1-go"
  _"encoding/hex"
  _"crypto/sha256"
  _ "bytes"
)

/*
Transaction   Contract       Size
-------------------------------------------
sender        sender       20 bytes
recipient     0x0          20 bytes
value         endowment     4 bytes (uint32)
fee           fee           4 bytes (uint32)
d_size        o_size        4 bytes (uint32)
data          ops           *
signature     signature    64 bytes
*/

var StepFee     *big.Int = new(big.Int)
var TxFee       *big.Int = new(big.Int)
var ContractFee *big.Int = new(big.Int)
var MemFee      *big.Int = new(big.Int)
var DataFee     *big.Int = new(big.Int)
var CryptoFee   *big.Int = new(big.Int)
var ExtroFee    *big.Int = new(big.Int)

var Period1Reward *big.Int = new(big.Int)
var Period2Reward *big.Int = new(big.Int)
var Period3Reward *big.Int = new(big.Int)
var Period4Reward *big.Int = new(big.Int)

type Transaction struct {
  nonce       string
  sender      string
  recipient   string
  value       uint64
  fee         uint32
  data        []string
  memory      []int
  lastTx      string
  v           uint32
  r, s        []byte
}

func NewTransaction(to string, value uint64, data []string) *Transaction {
  tx := Transaction{sender: "1234567890", recipient: to, value: value}
  tx.nonce = "0"
  tx.fee = 0//uint32((ContractFee + MemoryFee * float32(len(tx.data))) * 1e8)
  tx.lastTx = "0"

  // Serialize the data
  tx.data = make([]string, len(data))
  for i, val := range data {
    instr, err := CompileInstr(val)
    if err != nil {
      //fmt.Printf("compile error:%d %v\n", i+1, err)
    }

    tx.data[i] = instr
  }

  tx.SetVRS()


  return &tx
}

func (tx *Transaction) Hash() []byte {
  preEnc := []interface{}{
    tx.nonce,
    tx.recipient,
    tx.value,
    tx.fee,
    tx.data,
  }

  return Sha256Bin(Encode(preEnc))
}

func (tx *Transaction) IsContract() bool {
  return tx.recipient == ""
}

func (tx *Transaction) Signature() []byte {
  hash := tx.Hash()
  sec  := Sha256Bin([]byte("myprivkey"))

  sig, _ := secp256k1.Sign(hash, sec)

  return sig
}

func (tx *Transaction) PublicKey() []byte {
  hash := Sha256Bin(tx.MarshalRlp())
  sig  := tx.Signature()

  pubkey, _ := secp256k1.RecoverPubkey(hash, sig)

  return pubkey
}

func (tx *Transaction) Address() []byte {
  pubk := tx.PublicKey()
  // 1 is the marker 04
  key := pubk[1:65]

  return Sha256Bin(key)[12:]
}

func (tx *Transaction) SetVRS() {
  // Add 27 so we get either 27 or 28 (for positive and negative)
  tx.v = uint32(tx.Signature()[64]) + 27

  pubk := tx.PublicKey()[1:65]
  tx.r = pubk[:32]
  tx.s = pubk[32:64]
}

func (tx *Transaction) MarshalRlp() []byte {
  // Prepare the transaction for serialization
  preEnc := []interface{}{
    tx.nonce,
    tx.recipient,
    tx.value,
    tx.fee,
    tx.data,
    tx.v,
    tx.r,
    tx.s,
  }

  return Encode(preEnc)
}

func (tx *Transaction) UnmarshalRlp(data []byte) {
  t, _ := Decode(data,0)
  if slice, ok := t.([]interface{}); ok {
    fmt.Printf("NONCE %T\n", slice[3])
    if nonce, ok := slice[0].(uint8); ok {
      tx.nonce = string(nonce)
    }

    if recipient, ok := slice[1].([]byte); ok {
      tx.recipient = string(recipient)
    }

    // If only I knew of a better way.
    if value, ok := slice[2].(uint8); ok {
      tx.value = uint64(value)
    }
    if value, ok := slice[2].(uint16); ok {
      tx.value = uint64(value)
    }
    if value, ok := slice[2].(uint32); ok {
      tx.value = uint64(value)
    }
    if value, ok := slice[2].(uint64); ok {
      tx.value = uint64(value)
    }
    if fee, ok := slice[3].(uint8); ok {
      tx.fee = uint32(fee)
    }
    if fee, ok := slice[3].(uint16); ok {
      tx.fee = uint32(fee)
    }
    if fee, ok := slice[3].(uint32); ok {
      tx.fee = uint32(fee)
    }
    if fee, ok := slice[3].(uint64); ok {
      tx.fee = uint32(fee)
    }

    // Encode the data/instructions
    if data, ok := slice[4].([]interface{}); ok {
      tx.data = make([]string, len(data))
      for i, d := range data {
        if instr, ok := d.([]byte); ok {
          tx.data[i] = string(instr)
        }
      }
    }

    // vrs
    fmt.Printf("v %T\n", slice[5])
    if v, ok := slice[5].(uint8); ok {
      tx.v = uint32(v)
    }
    if v, ok := slice[5].(uint16); ok {
      tx.v = uint32(v)
    }
    if v, ok := slice[5].(uint32); ok {
      tx.v = uint32(v)
    }
    if v, ok := slice[5].(uint64); ok {
      tx.v = uint32(v)
    }
    if r, ok := slice[6].([]byte); ok {
      tx.r = r
    }
    if s, ok := slice[7].([]byte); ok {
      tx.s = s
    }
  }
}

func InitFees() {
  // Base for 2**60
  b60 := new(big.Int)
  b60.Exp(big.NewInt(2), big.NewInt(64), big.NewInt(0))
  // Base for 2**80
  b80 := new(big.Int)
  b80.Exp(big.NewInt(2), big.NewInt(80), big.NewInt(0))

  StepFee.Exp(big.NewInt(10), big.NewInt(16), big.NewInt(0))
  //StepFee.Div(b60, big.NewInt(64))
  //fmt.Println("StepFee:", StepFee)

  TxFee.Exp(big.NewInt(2), big.NewInt(64), big.NewInt(0))
  //fmt.Println("TxFee:", TxFee)

  ContractFee.Exp(big.NewInt(2), big.NewInt(64), big.NewInt(0))
  //fmt.Println("ContractFee:", ContractFee)

  MemFee.Div(b60, big.NewInt(4))
  //fmt.Println("MemFee:", MemFee)

  DataFee.Div(b60, big.NewInt(16))
  //fmt.Println("DataFee:", DataFee)

  CryptoFee.Div(b60, big.NewInt(16))
  //fmt.Println("CrytoFee:", CryptoFee)

  ExtroFee.Div(b60, big.NewInt(16))
  //fmt.Println("ExtroFee:", ExtroFee)

  Period1Reward.Mul(b80, big.NewInt(1024))
  //fmt.Println("Period1Reward:", Period1Reward)

  Period2Reward.Mul(b80, big.NewInt(512))
  //fmt.Println("Period2Reward:", Period2Reward)

  Period3Reward.Mul(b80, big.NewInt(256))
  //fmt.Println("Period3Reward:", Period3Reward)

  Period4Reward.Mul(b80, big.NewInt(128))
  //fmt.Println("Period4Reward:", Period4Reward)
}
