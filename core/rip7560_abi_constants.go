package core

const AcceptAccountMethodSig = uint64(0x1256ebd1)   // acceptAccount(uint256,uint256)
const AcceptPaymasterMethodSig = uint64(0x03be8439) // acceptPaymaster(uint256,uint256,bytes)
const SigFailAccountMethodSig = uint64(0x7715fac2)  // sigFailAccount(uint256,uint256)
const PaymasterMaxContextSize = 65536

const ValidateTransactionAbi = `
[
	{
		"type":"function",
		"name":"validateTransaction",
		"inputs": [
			{"name": "version","type": "uint256"},
			{"name": "txHash","type": "bytes32"},
			{"name": "transaction","type": "bytes"}
		]
	}
]`

const ValidatePaymasterTransactionAbi = `
[
	{
		"type":"function",
		"name":"validatePaymasterTransaction","
		inputs": [
			{"name": "version","type": "uint256"},
			{"name": "txHash","type": "bytes32"},
			{"name": "transaction","type": "bytes"}
		]
	}
]`

const PostPaymasterTransactionAbi = `
[
	{
		"type":"function",
		"name":"postPaymasterTransaction",
		"inputs": [
			{"name": "success","type": "bool"},
			{"name": "actualGasCost","type": "uint256"},
			{"name": "context","type": "bytes"}
		]
	}
]`

// AcceptAccountAbi Note that this is not a true ABI of the "acceptAccount" function.
// This ABI swaps inputs and outputs to simplify the ABI decoding.
const AcceptAccountAbi = `
[
	{
		"type":"function",
		"name":"acceptAccount",
		"outputs": [
			{"name": "validAfter","type": "uint256"},
			{"name": "validUntil","type": "uint256"}
		]
	}
]`

// AcceptPaymasterAbi Note that this is not a true ABI of the "acceptPaymaster" function.
// This ABI swaps inputs and outputs to simplify the ABI decoding.
const AcceptPaymasterAbi = `
[
	{
		"type":"function",
		"name":"acceptPaymaster",
		"outputs": [
			{"name": "validAfter","type": "uint256"},
			{"name": "validUntil","type": "uint256"},
			{"name": "context","type": "bytes"}
		]
	}
]`
