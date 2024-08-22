module libevm/examples

go 1.22.4

replace github.com/ethereum/go-ethereum => ../../

require github.com/ethereum/go-ethereum v0.0.0-00010101000000-000000000000

require (
	github.com/holiman/uint256 v1.3.1 // indirect
	golang.org/x/crypto v0.22.0 // indirect
	golang.org/x/sys v0.22.0 // indirect
)
