package main

import (
  "math/big"
  "fmt"
  "math/rand"
  "time"
  "github.com/obscuren/sha3"
)

type Dagger struct {
  hash *big.Int
  xn *big.Int
}

func (dag *Dagger) Search(diff *big.Int) *big.Int {
  dag.hash = big.NewInt(0)

  obj := BigPow(2, 256)
  obj = obj.Div(obj, diff)

  fmt.Println("diff", diff, "< objective", obj)

  r := rand.New(rand.NewSource(time.Now().UnixNano()))
  rnd := big.NewInt(r.Int63())
  fmt.Println("init rnd =", rnd)

  for i := 0; i < 1000; i++ {
    if dag.Eval(rnd).Cmp(obj) < 0 {
      fmt.Println("Found result! i = ", i)
      return rnd
    }

    rnd = rnd.Add(rnd, big.NewInt(1))
  }

  return big.NewInt(0)
}

func (dag *Dagger) Node(L uint64, i uint64) *big.Int {
  if L == i {
    return dag.hash
  }

  var m *big.Int
  if L == 9 {
    m = big.NewInt(16)
  } else {
    m = big.NewInt(3)
  }

  sha := sha3.NewKeccak224()
  sha.Reset()
  d := sha3.NewKeccak224()
  b := new(big.Int)
  ret := new(big.Int)

  for k := 0; k < int(m.Uint64()); k++ {
    d.Reset()
    d.Write(dag.hash.Bytes())
    d.Write(dag.xn.Bytes())
    d.Write(big.NewInt(int64(L)).Bytes())
    d.Write(big.NewInt(int64(i)).Bytes())
    d.Write(big.NewInt(int64(k)).Bytes())

    b.SetBytes(d.Sum(nil))
    pk := b.Uint64() & ((1 << ((L - 1) * 3)) - 1)
    sha.Write(dag.Node(L - 1, pk).Bytes())
  }

  ret.SetBytes(sha.Sum(nil))

  return ret
}

func (dag *Dagger) Eval(N *big.Int) *big.Int {
  pow := BigPow(2, 26)
  dag.xn = N.Div(N, pow)

  sha := sha3.NewKeccak224()
  sha.Reset()
  ret := new(big.Int)

  doneChan := make(chan bool, 3)

  for k := 0; k < 4; k++ {
    go func(_k int) {
      d := sha3.NewKeccak224()
      b := new(big.Int)

      d.Reset()
      d.Write(dag.hash.Bytes())
      d.Write(dag.xn.Bytes())
      d.Write(N.Bytes())
      d.Write(big.NewInt(int64(_k)).Bytes())

      b.SetBytes(d.Sum(nil))
      pk := (b.Uint64() & 0x1ffffff)

      sha.Write(dag.Node(9, pk).Bytes())
      doneChan <- true
    }(k)
  }

  for k := 0; k < 4; k++ {
    <- doneChan
  }

  return ret.SetBytes(sha.Sum(nil))
}
