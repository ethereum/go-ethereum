/*
This file is part of web3.js.

web3.js is free software: you can redistribute it and/or modify
it under the terms of the GNU Lesser General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

web3.js is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Lesser General Public License for more details.

You should have received a copy of the GNU Lesser General Public License
along with web3.js.  If not, see <http://www.gnu.org/licenses/>.
*/
import { isHexPrefixed, isHexString } from 'web3-validator';
import { bytesToHex, hexToBytes, isUint8Array, numberToHex } from 'web3-utils';
import { secp256k1 } from '../tx/constants.js';
import { Hardfork } from './enums.js';
import { ToBytesInputTypes, TypeOutput, TypeOutputReturnType } from './types.js';

type ConfigHardfork =
	// eslint-disable-next-line @typescript-eslint/ban-types
	| { name: string; block: null; timestamp: number }
	| { name: string; block: number; timestamp?: number };

/**
 * Removes '0x' from a given `String` if present
 * @param str the string value
 * @returns the string without 0x prefix
 */
export const stripHexPrefix = (str: string): string => {
	if (typeof str !== 'string')
		throw new Error(`[stripHexPrefix] input must be type 'string', received ${typeof str}`);

	return isHexPrefixed(str) ? str.slice(2) : str;
};
/**
 * Transforms Geth formatted nonce (i.e. hex string) to 8 byte 0x-prefixed string used internally
 * @param nonce string parsed from the Geth genesis file
 * @returns nonce as a 0x-prefixed 8 byte string
 */
function formatNonce(nonce: string): string {
	if (!nonce || nonce === '0x0') {
		return '0x0000000000000000';
	}
	if (isHexPrefixed(nonce)) {
		return `0x${stripHexPrefix(nonce).padStart(16, '0')}`;
	}
	return `0x${nonce.padStart(16, '0')}`;
}

/**
 * Converts a `Number` into a hex `String`
 * @param {Number} i
 * @return {String}
 */
const intToHex = function (i: number) {
	if (!Number.isSafeInteger(i) || i < 0) {
		throw new Error(`Received an invalid integer type: ${i}`);
	}
	return `0x${i.toString(16)}`;
};

/**
 * Converts Geth genesis parameters to an EthereumJS compatible `CommonOpts` object
 * @param json object representing the Geth genesis file
 * @param optional mergeForkIdPostMerge which clarifies the placement of MergeForkIdTransition
 * hardfork, which by default is post merge as with the merged eth networks but could also come
 * before merge like in kiln genesis
 * @returns genesis parameters in a `CommonOpts` compliant object
 */
// eslint-disable-next-line @typescript-eslint/no-explicit-any
function parseGethParams(json: any, mergeForkIdPostMerge = true) {
	// eslint-disable-next-line @typescript-eslint/no-unsafe-assignment
	const {
		name,
		config,
		difficulty,
		mixHash,
		gasLimit,
		coinbase,
		baseFeePerGas,
	}: {
		name: string;
		// eslint-disable-next-line @typescript-eslint/no-explicit-any
		config: any;
		difficulty: string;
		mixHash: string;
		gasLimit: string;
		coinbase: string;
		baseFeePerGas: string;
	} = json;
	// eslint-disable-next-line @typescript-eslint/no-unsafe-assignment
	let { extraData, timestamp, nonce }: { extraData: string; timestamp: string; nonce: string } =
		json;
	const genesisTimestamp = Number(timestamp);
	// eslint-disable-next-line @typescript-eslint/no-unsafe-assignment
	const { chainId }: { chainId: number } = config;

	// geth is not strictly putting empty fields with a 0x prefix
	if (extraData === '') {
		extraData = '0x';
	}
	// geth may use number for timestamp
	if (!isHexPrefixed(timestamp)) {
		// eslint-disable-next-line radix
		timestamp = intToHex(parseInt(timestamp));
	}
	// geth may not give us a nonce strictly formatted to an 8 byte hex string
	if (nonce.length !== 18) {
		nonce = formatNonce(nonce);
	}

	// EIP155 and EIP158 are both part of Spurious Dragon hardfork and must occur at the same time
	// but have different configuration parameters in geth genesis parameters
	// eslint-disable-next-line @typescript-eslint/no-unsafe-member-access
	if (config.eip155Block !== config.eip158Block) {
		throw new Error(
			'EIP155 block number must equal EIP 158 block number since both are part of SpuriousDragon hardfork and the client only supports activating the full hardfork',
		);
	}

	const params = {
		name,
		chainId,
		networkId: chainId,
		genesis: {
			timestamp,
			// eslint-disable-next-line radix
			gasLimit: parseInt(gasLimit), // geth gasLimit and difficulty are hex strings while ours are `number`s
			// eslint-disable-next-line radix
			difficulty: parseInt(difficulty),
			nonce,
			extraData,
			mixHash,
			coinbase,
			baseFeePerGas,
		},
		hardfork: undefined as string | undefined,
		hardforks: [] as ConfigHardfork[],
		bootstrapNodes: [],
		consensus:
			// eslint-disable-next-line @typescript-eslint/no-unsafe-member-access
			config.clique !== undefined
				? {
						type: 'poa',
						algorithm: 'clique',
						clique: {
							// The recent geth genesis seems to be using blockperiodseconds
							// and epochlength for clique specification
							// see: https://hackmd.io/PqZgMpnkSWCWv5joJoFymQ
							// eslint-disable-next-line @typescript-eslint/no-unsafe-member-access, @typescript-eslint/no-unsafe-assignment
							period: config.clique.period ?? config.clique.blockperiodseconds,
							// eslint-disable-next-line @typescript-eslint/no-unsafe-member-access,  @typescript-eslint/no-unsafe-assignment
							epoch: config.clique.epoch ?? config.clique.epochlength,
						},
				  }
				: {
						type: 'pow',
						algorithm: 'ethash',
						ethash: {},
				  },
	};

	const forkMap: { [key: string]: { name: string; postMerge?: boolean; isTimestamp?: boolean } } =
		{
			[Hardfork.Homestead]: { name: 'homesteadBlock' },
			[Hardfork.Dao]: { name: 'daoForkBlock' },
			[Hardfork.TangerineWhistle]: { name: 'eip150Block' },
			[Hardfork.SpuriousDragon]: { name: 'eip155Block' },
			[Hardfork.Byzantium]: { name: 'byzantiumBlock' },
			[Hardfork.Constantinople]: { name: 'constantinopleBlock' },
			[Hardfork.Petersburg]: { name: 'petersburgBlock' },
			[Hardfork.Istanbul]: { name: 'istanbulBlock' },
			[Hardfork.MuirGlacier]: { name: 'muirGlacierBlock' },
			[Hardfork.Berlin]: { name: 'berlinBlock' },
			[Hardfork.London]: { name: 'londonBlock' },
			[Hardfork.MergeForkIdTransition]: {
				name: 'mergeForkBlock',
				postMerge: mergeForkIdPostMerge,
			},
			[Hardfork.Shanghai]: { name: 'shanghaiTime', postMerge: true, isTimestamp: true },
			[Hardfork.ShardingForkDev]: {
				name: 'shardingForkTime',
				postMerge: true,
				isTimestamp: true,
			},
		};

	// forkMapRev is the map from config field name to Hardfork
	const forkMapRev = Object.keys(forkMap).reduce<{ [key: string]: string }>((acc, elem) => {
		acc[forkMap[elem].name] = elem;
		return acc;
	}, {});
	// eslint-disable-next-line @typescript-eslint/no-unsafe-argument
	const configHardforkNames = Object.keys(config).filter(
		// eslint-disable-next-line no-null/no-null, @typescript-eslint/no-unsafe-member-access, @typescript-eslint/prefer-optional-chain
		key => forkMapRev[key] !== undefined && config[key] !== undefined && config[key] !== null,
	);

	params.hardforks = configHardforkNames
		.map(nameBlock => ({
			name: forkMapRev[nameBlock],
			// eslint-disable-next-line @typescript-eslint/no-unsafe-assignment
			block:
				// eslint-disable-next-line @typescript-eslint/no-unsafe-member-access
				forkMap[forkMapRev[nameBlock]].isTimestamp === true ||
				// eslint-disable-next-line @typescript-eslint/no-unsafe-member-access
				typeof config[nameBlock] !== 'number'
					? // eslint-disable-next-line no-null/no-null
					  null
					: // eslint-disable-next-line @typescript-eslint/no-unsafe-member-access
					  config[nameBlock],
			// eslint-disable-next-line @typescript-eslint/no-unsafe-assignment
			timestamp:
				// eslint-disable-next-line @typescript-eslint/no-unsafe-member-access
				forkMap[forkMapRev[nameBlock]].isTimestamp === true &&
				// eslint-disable-next-line @typescript-eslint/no-unsafe-member-access
				typeof config[nameBlock] === 'number'
					? // eslint-disable-next-line @typescript-eslint/no-unsafe-member-access
					  config[nameBlock]
					: undefined,
		}))
		// eslint-disable-next-line no-null/no-null
		.filter(fork => fork.block !== null || fork.timestamp !== undefined) as ConfigHardfork[];

	params.hardforks.sort(
		(a: ConfigHardfork, b: ConfigHardfork) => (a.block ?? Infinity) - (b.block ?? Infinity),
	);

	params.hardforks.sort(
		(a: ConfigHardfork, b: ConfigHardfork) =>
			(a.timestamp ?? genesisTimestamp) - (b.timestamp ?? genesisTimestamp),
	);
	// eslint-disable-next-line @typescript-eslint/no-unsafe-member-access
	if (config.terminalTotalDifficulty !== undefined) {
		// Following points need to be considered for placement of merge hf
		// - Merge hardfork can't be placed at genesis
		// - Place merge hf before any hardforks that require CL participation for e.g. withdrawals
		// - Merge hardfork has to be placed just after genesis if any of the genesis hardforks make CL
		//   necessary for e.g. withdrawals
		const mergeConfig = {
			name: Hardfork.Merge,
			// eslint-disable-next-line @typescript-eslint/no-unsafe-member-access, @typescript-eslint/no-unsafe-assignment
			ttd: config.terminalTotalDifficulty,
			// eslint-disable-next-line no-null/no-null
			block: null,
		};

		// Merge hardfork has to be placed before first hardfork that is dependent on merge
		const postMergeIndex = params.hardforks.findIndex(
			// eslint-disable-next-line @typescript-eslint/no-unsafe-member-access
			hf => forkMap[hf.name]?.postMerge === true,
		);
		if (postMergeIndex !== -1) {
			params.hardforks.splice(postMergeIndex, 0, mergeConfig as unknown as ConfigHardfork);
		} else {
			params.hardforks.push(mergeConfig as unknown as ConfigHardfork);
		}
	}

	const latestHardfork = params.hardforks.length > 0 ? params.hardforks.slice(-1)[0] : undefined;
	params.hardfork = latestHardfork?.name;
	params.hardforks.unshift({ name: Hardfork.Chainstart, block: 0 });

	return params;
}

/**
 * Parses a genesis.json exported from Geth into parameters for Common instance
 * @param json representing the Geth genesis file
 * @param name optional chain name
 * @returns parsed params
 */
// eslint-disable-next-line @typescript-eslint/no-explicit-any
export function parseGethGenesis(json: any, name?: string, mergeForkIdPostMerge?: boolean) {
	try {
		if (['config', 'difficulty', 'gasLimit', 'alloc'].some(field => !(field in json))) {
			throw new Error('Invalid format, expected geth genesis fields missing');
		}
		if (name !== undefined) {
			// eslint-disable-next-line @typescript-eslint/no-unsafe-member-access, no-param-reassign
			json.name = name;
		}
		return parseGethParams(json, mergeForkIdPostMerge);
	} catch (e) {
		// eslint-disable-next-line @typescript-eslint/no-unsafe-member-access, @typescript-eslint/restrict-template-expressions
		throw new Error(`Error parsing parameters file: ${(e as { message: string }).message}`);
	}
}

/**
 * Pads a `String` to have an even length
 * @param value
 * @return output
 */
export function padToEven(value: string): string {
	let a = value;

	if (typeof a !== 'string') {
		throw new Error(`[padToEven] value must be type 'string', received ${typeof a}`);
	}

	if (a.length % 2) a = `0${a}`;

	return a;
}

/**
 * Converts an `Number` to a `Uint8Array`
 * @param {Number} i
 * @return {Uint8Array}
 */
export const intToUint8Array = function (i: number) {
	const hex = intToHex(i);
	return hexToBytes(`0x${padToEven(hex.slice(2))}`);
};

/**
 * Attempts to turn a value into a `Uint8Array`.
 * Inputs supported: `Uint8Array` `String` (hex-prefixed), `Number`, null/undefined, `BigInt` and other objects
 * with a `toArray()` or `toUint8Array()` method.
 * @param v the value
 */
export const toUint8Array = function (v: ToBytesInputTypes): Uint8Array {
	// eslint-disable-next-line no-null/no-null
	if (v === null || v === undefined) {
		return new Uint8Array();
	}

	if (v instanceof Uint8Array) {
		return v;
	}

	if (v?.constructor?.name === 'Uint8Array') {
		return Uint8Array.from(v as unknown as Uint8Array);
	}

	if (Array.isArray(v)) {
		return Uint8Array.from(v);
	}

	if (typeof v === 'string') {
		if (!isHexString(v)) {
			throw new Error(
				`Cannot convert string to Uint8Array. only supports 0x-prefixed hex strings and this string was given: ${v}`,
			);
		}
		return hexToBytes(padToEven(stripHexPrefix(v)));
	}

	if (typeof v === 'number') {
		return toUint8Array(numberToHex(v));
	}

	if (typeof v === 'bigint') {
		if (v < BigInt(0)) {
			throw new Error(`Cannot convert negative bigint to Uint8Array. Given: ${v}`);
		}
		let n = v.toString(16);
		if (n.length % 2) n = `0${n}`;
		return toUint8Array(`0x${n}`);
	}

	if (v.toArray) {
		// converts a BN to a Uint8Array
		return Uint8Array.from(v.toArray());
	}

	throw new Error('invalid type');
};

/**
 * Converts a {@link Uint8Array} to a {@link bigint}
 */
export function uint8ArrayToBigInt(buf: Uint8Array) {
	const hex = bytesToHex(buf);
	if (hex === '0x') {
		return BigInt(0);
	}
	return BigInt(hex);
}

/**
 * Converts a {@link bigint} to a {@link Uint8Array}
 */
export function bigIntToUint8Array(num: bigint) {
	return toUint8Array(`0x${num.toString(16)}`);
}

/**
 * Returns a Uint8Array filled with 0s.
 * @param bytes the number of bytes the Uint8Array should be
 */
export const zeros = function (bytes: number): Uint8Array {
	return new Uint8Array(bytes).fill(0);
};

/**
 * Pads a `Uint8Array` with zeros till it has `length` bytes.
 * Truncates the beginning or end of input if its length exceeds `length`.
 * @param msg the value to pad (Uint8Array)
 * @param length the number of bytes the output should be
 * @param right whether to start padding form the left or right
 * @return (Uint8Array)
 */
const setLength = function (msg: Uint8Array, length: number, right: boolean) {
	const buf = zeros(length);
	if (right) {
		if (msg.length < length) {
			buf.set(msg);
			return buf;
		}
		return msg.subarray(0, length);
	}
	if (msg.length < length) {
		buf.set(msg, length - msg.length);
		return buf;
	}
	return msg.subarray(-length);
};

/**
 * Throws if input is not a Uint8Array
 * @param {Uint8Array} input value to check
 */
export function assertIsUint8Array(input: unknown): asserts input is Uint8Array {
	if (!isUint8Array(input)) {
		// eslint-disable-next-line @typescript-eslint/restrict-template-expressions
		const msg = `This method only supports Uint8Array but input was: ${input}`;
		throw new Error(msg);
	}
}
/**
 * Left Pads a `Uint8Array` with leading zeros till it has `length` bytes.
 * Or it truncates the beginning if it exceeds.
 * @param msg the value to pad (Uint8Array)
 * @param length the number of bytes the output should be
 * @return (Uint8Array)
 */
export const setLengthLeft = function (msg: Uint8Array, length: number) {
	assertIsUint8Array(msg);
	return setLength(msg, length, false);
};

/**
 * Trims leading zeros from a `Uint8Array`, `String` or `Number[]`.
 * @param a (Uint8Array|Array|String)
 * @return (Uint8Array|Array|String)
 */
export function stripZeros<T extends Uint8Array | number[] | string>(a: T): T {
	// eslint-disable-next-line @typescript-eslint/no-unsafe-member-access, @typescript-eslint/no-unsafe-assignment
	let first = a[0];
	// eslint-disable-next-line @typescript-eslint/no-unsafe-member-access, @typescript-eslint/no-unsafe-call
	while (a.length > 0 && first.toString() === '0') {
		// eslint-disable-next-line @typescript-eslint/no-unsafe-member-access, @typescript-eslint/no-unsafe-assignment, prefer-destructuring, @typescript-eslint/no-unsafe-call, no-param-reassign
		a = a.slice(1) as T;
		// eslint-disable-next-line @typescript-eslint/no-unsafe-assignment, prefer-destructuring, @typescript-eslint/no-unsafe-member-access
		first = a[0];
	}
	// eslint-disable-next-line @typescript-eslint/no-unsafe-return
	return a;
}

/**
 * Trims leading zeros from a `Uint8Array`.
 * @param a (Uint8Array)
 * @return (Uint8Array)
 */
export const unpadUint8Array = function (a: Uint8Array): Uint8Array {
	assertIsUint8Array(a);
	return stripZeros(a);
};

/**
 * Converts a {@link bigint} to a `0x` prefixed hex string
 */
export const bigIntToHex = (num: bigint) => `0x${num.toString(16)}`;

/**
 * Convert value from bigint to an unpadded Uint8Array
 * (useful for RLP transport)
 * @param value value to convert
 */
export function bigIntToUnpaddedUint8Array(value: bigint): Uint8Array {
	return unpadUint8Array(bigIntToUint8Array(value));
}

function calculateSigRecovery(v: bigint, chainId?: bigint): bigint {
	if (v === BigInt(0) || v === BigInt(1)) return v;

	if (chainId === undefined) {
		return v - BigInt(27);
	}
	return v - (chainId * BigInt(2) + BigInt(35));
}

function isValidSigRecovery(recovery: bigint): boolean {
	return recovery === BigInt(0) || recovery === BigInt(1);
}

/**
 * ECDSA public key recovery from signature.
 * NOTE: Accepts `v === 0 | v === 1` for EIP1559 transactions
 * @returns Recovered public key
 */
export const ecrecover = function (
	msgHash: Uint8Array,
	v: bigint,
	r: Uint8Array,
	s: Uint8Array,
	chainId?: bigint,
): Uint8Array {
	const recovery = calculateSigRecovery(v, chainId);
	if (!isValidSigRecovery(recovery)) {
		throw new Error('Invalid signature v value');
	}

	const senderPubKey = new secp256k1.Signature(uint8ArrayToBigInt(r), uint8ArrayToBigInt(s))
		.addRecoveryBit(Number(recovery))
		.recoverPublicKey(msgHash)
		.toRawBytes(false);
	return senderPubKey.slice(1);
};

/**
 * Convert an input to a specified type.
 * Input of null/undefined returns null/undefined regardless of the output type.
 * @param input value to convert
 * @param outputType type to output
 */
// eslint-disable-next-line @typescript-eslint/ban-types
export function toType<T extends TypeOutput>(input: null, outputType: T): null;
export function toType<T extends TypeOutput>(input: undefined, outputType: T): undefined;
export function toType<T extends TypeOutput>(
	input: ToBytesInputTypes,
	outputType: T,
): TypeOutputReturnType[T];
export function toType<T extends TypeOutput>(
	input: ToBytesInputTypes,
	outputType: T,
	// eslint-disable-next-line @typescript-eslint/ban-types
): TypeOutputReturnType[T] | undefined | null {
	// eslint-disable-next-line no-null/no-null
	if (input === null) {
		// eslint-disable-next-line no-null/no-null
		return null;
	}
	if (input === undefined) {
		return undefined;
	}

	if (typeof input === 'string' && !isHexString(input)) {
		throw new Error(`A string must be provided with a 0x-prefix, given: ${input}`);
	} else if (typeof input === 'number' && !Number.isSafeInteger(input)) {
		throw new Error(
			'The provided number is greater than MAX_SAFE_INTEGER (please use an alternative input type)',
		);
	}

	const output = toUint8Array(input);

	switch (outputType) {
		case TypeOutput.Uint8Array:
			return output as TypeOutputReturnType[T];
		case TypeOutput.BigInt:
			return uint8ArrayToBigInt(output) as TypeOutputReturnType[T];
		case TypeOutput.Number: {
			const bigInt = uint8ArrayToBigInt(output);
			if (bigInt > BigInt(Number.MAX_SAFE_INTEGER)) {
				throw new Error(
					'The provided number is greater than MAX_SAFE_INTEGER (please use an alternative output type)',
				);
			}
			return Number(bigInt) as TypeOutputReturnType[T];
		}
		case TypeOutput.PrefixedHexString:
			return bytesToHex(output) as TypeOutputReturnType[T];
		default:
			throw new Error('unknown outputType');
	}
}
