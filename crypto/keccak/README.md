This is a vendored and modified copy of golang.org/x/crypto/sha3, with an assembly
implementation of keccak256. We wish to retain the assembly implementation,
which was removed in v0.44.0.

Ethereum uses a 'legacy' variant of Keccak, which was defined before it became SHA3. As
such, we cannot use the standard library crypto/sha3 package.
