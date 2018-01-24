# go-base58

I extracted this package from https://github.com/conformal/btcutil to provide a simple base58 package that
- defaults to base58-check (btc)
- and allows using different alphabets.

## Usage

```go
package main

import (
  "fmt"
  b58 "github.com/jbenet/go-base58"
)

func main() {
  buf := []byte{255, 254, 253, 252}
  fmt.Printf("buffer: %v\n", buf)

  str := b58.Encode(buf)
  fmt.Printf("encoded: %s\n", str)

  buf2 := b58.Decode(str)
  fmt.Printf("decoded: %v\n", buf2)
}
```

### Another alphabet

```go
package main

import (
  "fmt"
  b58 "github.com/jbenet/go-base58"
)

const BogusAlphabet = "ZYXWVUTSRQPNMLKJHGFEDCBAzyxwvutsrqponmkjihgfedcba987654321"


func encdec(alphabet string) {
  fmt.Printf("using: %s\n", alphabet)

  buf := []byte{255, 254, 253, 252}
  fmt.Printf("buffer: %v\n", buf)

  str := b58.EncodeAlphabet(buf, alphabet)
  fmt.Printf("encoded: %s\n", str)

  buf2 := b58.DecodeAlphabet(str, alphabet)
  fmt.Printf("decoded: %v\n\n", buf2)
}


func main() {
  encdec(b58.BTCAlphabet)
  encdec(b58.FlickrAlphabet)
  encdec(BogusAlphabet)
}
```


## License

Package base58 (and the original btcutil) are licensed under the ISC License.
