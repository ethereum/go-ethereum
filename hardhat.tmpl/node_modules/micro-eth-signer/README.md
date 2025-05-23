# micro-eth-signer

Minimal library for Ethereum transactions, addresses and smart contracts.

- 🔓 Secure: audited [noble](https://paulmillr.com/noble/) cryptography, no network code, [hedged signatures](#transactions-create-sign)
- 🔻 Tree-shakeable: unused code is excluded from your builds
- 🔍 Reliable: 150MB of test vectors from EIPs, ethers and viem
- ✍️ Core: transactions, addresses, messages
- 🌍 Network-related: execute Uniswap & Chainlink, fetch tx history
- 🦺 Advanced: type-safe ABI parsing, RLP, SSZ, KZG, Verkle
- 🪶 29KB gzipped (1300 lines) for core, just 3 deps

_Check out all web3 utility libraries:_ [ETH](https://github.com/paulmillr/micro-eth-signer), [BTC](https://github.com/paulmillr/scure-btc-signer), [SOL](https://github.com/paulmillr/micro-sol-signer)

## Usage

> `npm install micro-eth-signer`

> `jsr add jsr:@paulmillr/micro-eth-signer`

We support all major platforms and runtimes.
For React Native, you may need a [polyfill for getRandomValues](https://github.com/LinusU/react-native-get-random-values).
If you don't like NPM, a standalone [eth-signer.js](https://github.com/paulmillr/micro-eth-signer/releases) is also available.

- Core
  - [Create random wallet](#create-random-wallet)
  - [Transactions: create, sign](#transactions-create-sign)
  - [Addresses: create, checksum](#addresses-create-checksum)
  - [Messages: sign, verify](#messages-sign-verify)
- Network-related
  - [Init network](#init-network)
  - [Fetch balances and history](#fetch-balances-and-history-from-an-archive-node)
  - [Fetch Chainlink oracle prices](#fetch-chainlink-oracle-prices)
  - [Resolve ENS address](#resolve-ens-address)
  - [Swap tokens with Uniswap](#swap-tokens-with-uniswap)
- Advanced
  - [Type-safe ABI parsing](#type-safe-abi-parsing)
  - [Human-readable transaction hints](#human-readable-transaction-hints)
  - [Human-readable event hints](#human-readable-event-hints)
  - [RLP & SSZ](#rlp--ssz)
  - [KZG & Verkle](#kzg--verkle)
- [Security](#security)
- [Performance](#performance)
- [License](#license)

## Core

### Create random wallet

```ts
import { addr } from 'micro-eth-signer';
const random = addr.random(); // Secure: uses CSPRNG
console.log(random.privateKey, random.address);
// '0x17ed046e6c4c21df770547fad9a157fd17b48b35fe9984f2ff1e3c6a62700bae'
// '0x26d930712fd2f612a107A70fd0Ad79b777cD87f6'
```

### Transactions: create, sign

```ts
import { Transaction, weigwei, weieth } from 'micro-eth-signer';
const tx = Transaction.prepare({
  to: '0xdf90dea0e0bf5ca6d2a7f0cb86874ba6714f463e',
  value: weieth.decode('1.1'), // 1.1eth in wei
  maxFeePerGas: weigwei.decode('100'), // 100gwei in wei (priority fee is 1 gwei)
  nonce: 0n,
});
// Uses `random` from example above. Alternatively, pass 0x hex string or Uint8Array
const signedTx = tx.signBy(random.privateKey);
console.log('signed tx', signedTx, signedTx.toHex());
console.log('fee', signedTx.fee);

// Hedged signatures, with extra noise / security
const signedTx2 = tx.signBy(random.privateKey, { extraEntropy: true });

// Send whole account balance. See Security section for caveats
const CURRENT_BALANCE = '1.7182050000017'; // in eth
const txSendingWholeBalance = unsignedTx.setWholeAmount(weieth.decode(CURRENT_BALANCE));
```

We support legacy, EIP2930, EIP1559, EIP4844 and EIP7702 transactions.

Signing is done with [noble-curves](https://github.com/paulmillr/noble-curves), using RFC 6979.
Hedged signatures are also supported - check out the blog post
[Deterministic signatures are not your friends](https://paulmillr.com/posts/deterministic-signatures/).

### Addresses: create, checksum

```ts
import { addr } from 'micro-eth-signer';
const priv = '0x0687640ee33ef844baba3329db9e16130bd1735cbae3657bd64aed25e9a5c377';
const pub = '030fba7ba5cfbf8b00dd6f3024153fc44ddda93727da58c99326eb0edd08195cdb';
const nonChecksummedAddress = '0x0089d53f703f7e0843953d48133f74ce247184c2';
const checksummedAddress = addr.addChecksum(nonChecksummedAddress);
console.log(
  checksummedAddress, // 0x0089d53F703f7E0843953D48133f74cE247184c2
  addr.isValid(checksummedAddress), // true
  addr.isValid(nonChecksummedAddress), // also true
  addr.fromPrivateKey(priv),
  addr.fromPublicKey(pub)
);
```

### Messages: sign, verify

There are two messaging standards: [EIP-191](https://eips.ethereum.org/EIPS/eip-191) & [EIP-712](https://eips.ethereum.org/EIPS/eip-712).

#### EIP-191

```ts
import * as typed from 'micro-eth-signer/typed-data';

// Example message
const message = 'Hello, Ethereum!';
const privateKey = '0x4c0883a69102937d6231471b5dbb6204fe512961708279f1d7b1b8e7e8b1b1e1';

// Sign the message
const signature = typed.personal.sign(message, privateKey);
console.log('Signature:', signature);

// Verify the signature
const address = '0xYourEthereumAddress';
const isValid = typed.personal.verify(signature, message, address);
console.log('Is valid:', isValid);
```

#### EIP-712

```ts
import * as typed from 'micro-eth-signer/typed-data';

const types = {
  Person: [
    { name: 'name', type: 'string' },
    { name: 'wallet', type: 'address' },
  ],
  Mail: [
    { name: 'from', type: 'Person' },
    { name: 'to', type: 'Person' },
    { name: 'contents', type: 'string' },
  ],
};

// Define the domain
const domain: typed.EIP712Domain = {
  name: 'Ether Mail',
  version: '1',
  chainId: 1,
  verifyingContract: '0xCcCCccccCCCCcCCCCCCcCcCccCcCCCcCcccccccC',
  salt: '0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef',
};

// Define the message
const message = {
  from: {
    name: 'Alice',
    wallet: '0xCcCCccccCCCCcCCCCCCcCcCccCcCCCcCcccccccC',
  },
  to: {
    name: 'Bob',
    wallet: '0xbBbBBBBbbBBBbbbBbbBbbbbBBbBbbbbBbBbbBBbB',
  },
  contents: 'Hello, Bob!',
};

// Create the typed data
const typedData: typed.TypedData<typeof types, 'Mail'> = {
  types,
  primaryType: 'Mail',
  domain,
  message,
};

// Sign the typed data
const privateKey = '0x4c0883a69102937d6231471b5dbb6204fe512961708279f1d7b1b8e7e8b1b1e1';
const signature = typed.signTyped(typedData, privateKey);
console.log('Signature:', signature);

// Verify the signature
const address = '0xYourEthereumAddress';
const isValid = typed.verifyTyped(signature, typedData, address);

// Recover the public key
const publicKey = typed.recoverPublicKeyTyped(signature, typedData);
```

## Network-related

### Init network

eth-signer is network-free and makes it easy to audit network-related code:
all requests are done with user-provided function, conforming to built-in `fetch()`.
We recommend using [micro-ftch](https://github.com/paulmillr/micro-ftch),
which implements kill-switch, logging, batching / concurrency and other features.

Most APIs (chainlink, uniswap) expect instance of Web3Provider.
The call stack would look like this:

- `Chainlink` => `Web3Provider` => `jsonrpc` => `fetch`

To initialize Web3Provider, do the following:

```js
// Requests are made with fetch(), a built-in method
import { jsonrpc } from 'micro-ftch';
import { Web3Provider } from 'micro-eth-signer/net';
const RPC_URL = 'http://localhost:8545';
const prov = new Web3Provider(jsonrpc(fetch, RPC_URL));

// Example using mewapi RPC
const RPC_URL_2 = 'https://nodes.mewapi.io/rpc/eth';
const prov2 = new Web3Provider(
  jsonrpc(fetch, RPC_URL_2, { Origin: 'https://www.myetherwallet.com' })
);
```

### Fetch balances & history

> [!NOTE]
> Basic data can be fetched from any node.
> Uses `trace_filter` & requires [Erigon](https://erigon.tech), others are too slow.

```ts
const addr = '0xd8da6bf26964af9d7eed9e03e53415d37aa96045';
const block = await prov.blockInfo(await prov.height());
console.log('current block', block.number, block.timestamp, block.baseFeePerGas);
console.log('info for addr', addr, await prov.unspent(addr));

// Other methods of Web3Provider:
// blockInfo(block: number): Promise<BlockInfo>; // {baseFeePerGas, hash, timestamp...}
// height(): Promise<number>;
// internalTransactions(address: string, opts?: TraceOpts): Promise<any[]>;
// ethLogsSingle(topics: Topics, opts: LogOpts): Promise<Log[]>;
// ethLogs(topics: Topics, opts?: LogOpts): Promise<Log[]>;
// tokenTransfers(address: string, opts?: LogOpts): Promise<[Log[], Log[]]>;
// wethTransfers(address: string, opts?: LogOpts): Promise<[Log[]]>;
// txInfo(txHash: string, opts?: TxInfoOpts): Promise<{
//   type: "legacy" | "eip2930" | "eip1559" | "eip4844"; info: any; receipt: any; raw: string | undefined;
// }>;
// tokenInfo(address: string): Promise<TokenInfo | undefined>;
// transfers(address: string, opts?: TraceOpts & LogOpts): Promise<TxTransfers[]>;
// allowances(address: string, opts?: LogOpts): Promise<TxAllowances>;
// tokenBalances(address: string, tokens: string[]): Promise<Record<string, bigint>>;
```

### Fetch Chainlink oracle prices

```ts
import { Chainlink } from 'micro-eth-signer/net';
const link = new Chainlink(prov);
const btc = await link.coinPrice('BTC');
const bat = await link.tokenPrice('BAT');
console.log({ btc, bat }); // BTC 19188.68870991, BAT 0.39728989 in USD
```

### Resolve ENS address

```ts
import { ENS } from 'micro-eth-signer/net';
const ens = new ENS(prov);
const vitalikAddr = await ens.nameToAddress('vitalik.eth');
```

### Swap tokens with Uniswap

> Btw cool tool, glad you built it!

_Uniswap Founder_

Swap 12.12 USDT to BAT with uniswap V3 defaults of 0.5% slippage, 30 min expiration.

```ts
import { tokenFromSymbol } from 'micro-eth-signer/abi';
import { UniswapV3 } from 'micro-eth-signer/net'; // or UniswapV2

const USDT = tokenFromSymbol('USDT');
const BAT = tokenFromSymbol('BAT');
const u3 = new UniswapV3(prov); // or new UniswapV2(provider)
const fromAddress = '0xd8da6bf26964af9d7eed9e03e53415d37aa96045';
const toAddress = '0xd8da6bf26964af9d7eed9e03e53415d37aa96045';
const swap = await u3.swap(USDT, BAT, '12.12', { slippagePercent: 0.5, ttl: 30 * 60 });
const swapData = await swap.tx(fromAddress, toAddress);
console.log(swapData.amount, swapData.expectedAmount, swapData.allowance);
```

## Advanced

### Type-safe ABI parsing

The ABI is type-safe when `as const` is specified:

```ts
import { createContract } from 'micro-eth-signer/abi';
const PAIR_CONTRACT = [
  {
    type: 'function',
    name: 'getReserves',
    outputs: [
      { name: 'reserve0', type: 'uint112' },
      { name: 'reserve1', type: 'uint112' },
      { name: 'blockTimestampLast', type: 'uint32' },
    ],
  },
] as const;

const contract = createContract(PAIR_CONTRACT);
// Would create following typescript type:
{
  getReserves: {
    encodeInput: () => Uint8Array;
    decodeOutput: (b: Uint8Array) => {
      reserve0: bigint;
      reserve1: bigint;
      blockTimestampLast: bigint;
    };
  }
}
```

We're parsing values as:

```js
// no inputs
{} -> encodeInput();
// single input
{inputs: [{type: 'uint'}]} -> encodeInput(bigint);
// all inputs named
{inputs: [{type: 'uint', name: 'lol'}, {type: 'address', name: 'wut'}]} -> encodeInput({lol: bigint, wut: string})
// at least one input is unnamed
{inputs: [{type: 'uint', name: 'lol'}, {type: 'address'}]} -> encodeInput([bigint, string])
// Same applies for output!
```

There are following limitations:

- Fixed size arrays can have 999 elements at max: string[], string[1], ..., string[999]
- Fixed size 2d arrays can have 39 elements at max: string[][], string[][1], ..., string[39][39]
- Which is enough for almost all cases
- ABI must be described as constant value: `[...] as const`
- We're not able to handle contracts with method overload (same function names with different args) — the code will still work, but not types

Check out [`src/net/ens.ts`](./src/net/ens.ts) for type-safe contract execution example.

### Human-readable transaction hints

The transaction sent ERC-20 USDT token between addresses. The library produces a following hint:

> Transfer 22588 USDT to 0xdac17f958d2ee523a2206206994597c13d831ec7

```ts
import { decodeTx } from 'micro-eth-signer/abi';

const tx =
  '0xf8a901851d1a94a20082c12a94dac17f958d2ee523a2206206994597c13d831ec780b844a9059cbb000000000000000000000000dac17f958d2ee523a2206206994597c13d831ec7000000000000000000000000000000000000000000000000000000054259870025a066fcb560b50e577f6dc8c8b2e3019f760da78b4c04021382ba490c572a303a42a0078f5af8ac7e11caba9b7dc7a64f7bdc3b4ce1a6ab0a1246771d7cc3524a7200';
// Decode tx information
deepStrictEqual(decodeTx(tx), {
  name: 'transfer',
  signature: 'transfer(address,uint256)',
  value: {
    to: '0xdac17f958d2ee523a2206206994597c13d831ec7',
    value: 22588000000n,
  },
  hint: 'Transfer 22588 USDT to 0xdac17f958d2ee523a2206206994597c13d831ec7',
});
```

Or if you have already decoded tx:

```ts
import { decodeData } from 'micro-eth-signer/abi';

const to = '0x7a250d5630b4cf539739df2c5dacb4c659f2488d';
const data =
  '7ff36ab5000000000000000000000000000000000000000000000000ab54a98ceb1f0ad30000000000000000000000000000000000000000000000000000000000000080000000000000000000000000d8da6bf26964af9d7eed9e03e53415d37aa96045000000000000000000000000000000000000000000000000000000006fd9c6ea0000000000000000000000000000000000000000000000000000000000000002000000000000000000000000c02aaa39b223fe8d0a0e5c4f27ead9083c756cc2000000000000000000000000106d3c66d22d2dd0446df23d7f5960752994d600';
const value = 100000000000000000n;

deepStrictEqual(decodeData(to, data, value, { customContracts }), {
  name: 'swapExactETHForTokens',
  signature: 'swapExactETHForTokens(uint256,address[],address,uint256)',
  value: {
    amountOutMin: 12345678901234567891n,
    path: [
      '0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2',
      '0x106d3c66d22d2dd0446df23d7f5960752994d600',
    ],
    to: '0xd8da6bf26964af9d7eed9e03e53415d37aa96045',
    deadline: 1876543210n,
  },
});

// With custom tokens/contracts
const customContracts = {
  '0x106d3c66d22d2dd0446df23d7f5960752994d600': { abi: 'ERC20', symbol: 'LABRA', decimals: 9 },
};
deepStrictEqual(decodeData(to, data, value, { customContracts }), {
  name: 'swapExactETHForTokens',
  signature: 'swapExactETHForTokens(uint256,address[],address,uint256)',
  value: {
    amountOutMin: 12345678901234567891n,
    path: [
      '0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2',
      '0x106d3c66d22d2dd0446df23d7f5960752994d600',
    ],
    to: '0xd8da6bf26964af9d7eed9e03e53415d37aa96045',
    deadline: 1876543210n,
  },
  hint: 'Swap 0.1 ETH for at least 12345678901.234567891 LABRA. Expires at Tue, 19 Jun 2029 06:00:10 GMT',
});
```

### Human-readable event hints

Decoding the event produces the following hint:

> Allow 0xe592427a0aece92de3edee1f18e0157c05861564 spending up to 1000 BAT from 0xd8da6bf26964af9d7eed9e03e53415d37aa96045

```ts
import { decodeEvent } from 'micro-eth-signer/abi';

const to = '0x0d8775f648430679a709e98d2b0cb6250d2887ef';
const topics = [
  '0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925',
  '0x000000000000000000000000d8da6bf26964af9d7eed9e03e53415d37aa96045',
  '0x000000000000000000000000e592427a0aece92de3edee1f18e0157c05861564',
];
const data = '0x00000000000000000000000000000000000000000000003635c9adc5dea00000';
const einfo = decodeEvent(to, topics, data);
console.log(einfo);
```

### RLP & SSZ

[packed](https://github.com/paulmillr/micro-packed) allows us to implement
RLP in just 100 lines of code, and SSZ in 1500 lines.

SSZ includes [EIP-7495](https://eips.ethereum.org/EIPS/eip-7495) stable containers.

```ts
import { RLP } from 'micro-eth-signer/rlp';
// More RLP examples in test/rlp.test.js
RLP.decode(RLP.encode('dog'));
```

```ts
import * as ssz from 'micro-eth-signer/ssz';
// More SSZ examples in test/ssz.test.js
```

### KZG & Verkle

Allows to create & verify KZG EIP-4844 proofs.

```ts
import * as verkle from 'micro-eth-signer/verkle';

import { KZG } from 'micro-eth-signer/kzg';
// 400kb, 4-sec init
import { trustedSetup } from '@paulmillr/trusted-setups';
// 800kb, instant init
import { trustedSetup as fastSetup } from '@paulmillr/trusted-setups/fast.js';

// More KZG & Verkle examples in
// https://github.com/ethereumjs/ethereumjs-monorepo

const kzg = new KZG(trustedSetup);

// Example blob and scalar
const blob = '0x1234567890abcdef'; // Add actual blob data
const z = '0x1'; // Add actual scalar

// Compute and verify proof
const [proof, y] = kzg.computeProof(blob, z);
console.log('Proof:', proof);
console.log('Y:', y);
const commitment = '0x1234567890abcdef'; // Add actual commitment
const z = '0x1'; // Add actual scalar
// const y = '0x2'; // Add actual y value
const proof = '0x3'; // Add actual proof
const isValid = kzg.verifyProof(commitment, z, y, proof);
console.log('Is valid:', isValid);

// Compute and verify blob proof
const blob = '0x1234567890abcdef'; // Add actual blob data
const commitment = '0x1'; // Add actual commitment
const proof = kzg.computeBlobProof(blob, commitment);
console.log('Blob proof:', proof);
const isValidB = kzg.verifyBlobProof(blob, commitment, proof);
```

## Security

Main points to consider when auditing the library:

- ABI correctness
  - All ABI JSON should be compared to some external source
  - There are different databases of ABI: one is hosted by Etherscan, when you open contract page
- Network access
  - There must be no network calls in the library
  - Some functionality requires network: these need external network interface, conforming to `Web3Provider`
  - `createContract(abi)` should create purely offline contract
  - `createContract(abi, net)` would create contract that calls network using `net`, using external interface
- Skipped test vectors
  - There is `SKIPPED_ERRORS`, which contains list of test vectors from other libs that we skip
  - They are skipped because we consider them invalid, or so
  - If you believe they're skipped for wrong reasons, investigate and report

The library is cross-tested against other libraries (last update on 25 Feb 2024):

- ethereum-tests v13.1
- ethers 6.11.1
- viem v2.7.13

Check out article [ZSTs, ABIs, stolen keys and broken legs](https://github.com/paulmillr/micro-eth-signer/discussions/20) about caveats of secure ABI parsing found during development of the library.

### Privacy considerations

Default priority fee is 1 gwei, which matches what other wallets have.
However, it's recommended to fetch recommended priority fee from a node.

### Sending whole balance

There is a method `setWholeAmount` which allows to send whole account balance:

```ts
const CURRENT_BALANCE = '1.7182050000017'; // in eth
const txSendingWholeBalance = unsignedTx.setWholeAmount(weieth.decode(CURRENT_BALANCE));
```

It does two things:

1. `amount = accountBalance - maxFeePerGas * gasLimit`
2. `maxPriorityFeePerGas = maxFeePerGas`

Every eth block sets a fee for all its transactions, called base fee.
maxFeePerGas indicates how much gas user is able to spend in the worst case.
If the block's base fee is 5 gwei, while user is able to spend 10 gwei in maxFeePerGas,
the transaction would only consume 5 gwei. That means, base fee is unknown
before the transaction is included in a block.

By setting priorityFee to maxFee, we make the process deterministic:
`maxFee = 10, maxPriority = 10, baseFee = 5` would always spend 10 gwei.
In the end, the balance would become 0.

> [!WARNING]
> Using the method would decrease privacy of a transfer, because
> payments for services have specific amounts, and not _the whole amount_.

## Performance

Transaction signature matches `noble-curves` `sign()` speed,
which means over 4000 times per second on an M2 mac.

The first call of `sign` will take 20ms+ due to noble-curves secp256k1 `utils.precompute`.

To run benchmarks, execute `npm run bench`.

## Contributing

Make sure to use recursive cloning for the [eth-vectors](https://github.com/paulmillr/eth-vectors) submodule:

    git clone --recursive https://github.com/paulmillr/micro-eth-signer.git

## License

MIT License

Copyright (c) 2021 Paul Miller (https://paulmillr.com)
