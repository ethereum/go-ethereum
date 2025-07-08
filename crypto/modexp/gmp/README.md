# GMP Wrapper for go-ethereum

A CGO wrapper for GMP's modular exponentiation functionality.

## Prerequisites

You need to have GMP development libraries installed on your system:

- **Ubuntu/Debian**: `sudo apt-get install libgmp-dev`
- **Fedora/RHEL**: `sudo dnf install gmp-devel`
- **macOS**: `brew install gmp`
- **Windows**: See [GMP Windows builds](https://gmplib.org/)

> TODO: There is an alternative branch, where we compile from source however this is slightly messier (though it allows us to pin to a specific commit like we do for libsecp256k1)

## Usage

This package provides a GMP-backed implementation for modular exponentiation. The API can be expanded, however right now, the main usage is for the modexp precompile.

There are currently two implementations: generic (using Go wrapper types) and cwrapper (direct C calls).

### Byte Array Interface (Recommended)

```go
import "github.com/ethereum/go-ethereum/crypto/modexp/gmp"

base := []byte{0x02}       // 2
exp := []byte{0x0A}        // 10  
mod := []byte{0x03, 0xE8}  // 1000

result, err := gmp.ModExp(base, exp, mod)
if err != nil {
    log.Fatal(err)
}
// result = 24 (2^10 mod 1000)
```

### Direct GMP Interface

```go
// Create numbers
base := gmp.NewInt()
base.SetString("123456789", 10)

exp := gmp.NewInt()
exp.SetString("987654321", 10)

mod := gmp.NewInt()
mod.SetString("1000000007", 10)

// Compute base^exp mod mod
result := gmp.NewInt()
result.ExpMod(base, exp, mod)

fmt.Printf("Result: %s\n", result)
```


## Testing

Run tests from the go-ethereum root directory:

```bash
go test ./crypto/modexp/gmp/...
```

## License

TODO -- whatever go-ethereum does.