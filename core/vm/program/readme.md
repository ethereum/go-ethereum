### What is this

In many cases, we have a need to create somewhat nontrivial bytecode, for testing various
quirks related to state transition or evm execution.

For example, we want to have a `CREATE2`- op create a contract, which is then invoked, and when invoked does a selfdestruct-to-self.

It is overkill to go full solidity, but it is also a bit tricky do assemble this by concatenating bytes.

This utility takes an approach from [goevmlab](https://github.com/holiman/goevmlab/) where it has been used for several years,
a go-lang utility to assemble evm bytecode.

Using this utility, the case above can be expressed as:
```golang
	// Some runtime code
	runtime := program.New().Ops(vm.ADDRESS, vm.SELFDESTRUCT).Bytecode()
	// A constructor returning the runtime code
	initcode := program.New().ReturnData(runtime).Bytecode()
	// A factory invoking the constructor
	outer := program.New().Create2AndCall(initcode, nil).Bytecode()
```

### Warning

This package is a utility for testing, _not_ for production. As such:

- There are not package guarantees. We might iterate heavily on this package, and do backwards-incompatible changes without warning
- There are no quality-guarantees. These utilities may produce evm-code that is non-functional. YMMV.
- There are no stability-guarantees. The utility will `panic` if the inputs do not align / make sense.

