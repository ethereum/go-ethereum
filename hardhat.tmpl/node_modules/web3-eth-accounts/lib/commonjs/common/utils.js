"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.ecrecover = exports.bigIntToHex = exports.unpadUint8Array = exports.setLengthLeft = exports.zeros = exports.toUint8Array = exports.intToUint8Array = exports.stripHexPrefix = void 0;
exports.parseGethGenesis = parseGethGenesis;
exports.padToEven = padToEven;
exports.uint8ArrayToBigInt = uint8ArrayToBigInt;
exports.bigIntToUint8Array = bigIntToUint8Array;
exports.assertIsUint8Array = assertIsUint8Array;
exports.stripZeros = stripZeros;
exports.bigIntToUnpaddedUint8Array = bigIntToUnpaddedUint8Array;
exports.toType = toType;
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
const web3_validator_1 = require("web3-validator");
const web3_utils_1 = require("web3-utils");
const constants_js_1 = require("../tx/constants.js");
const enums_js_1 = require("./enums.js");
const types_js_1 = require("./types.js");
/**
 * Removes '0x' from a given `String` if present
 * @param str the string value
 * @returns the string without 0x prefix
 */
const stripHexPrefix = (str) => {
    if (typeof str !== 'string')
        throw new Error(`[stripHexPrefix] input must be type 'string', received ${typeof str}`);
    return (0, web3_validator_1.isHexPrefixed)(str) ? str.slice(2) : str;
};
exports.stripHexPrefix = stripHexPrefix;
/**
 * Transforms Geth formatted nonce (i.e. hex string) to 8 byte 0x-prefixed string used internally
 * @param nonce string parsed from the Geth genesis file
 * @returns nonce as a 0x-prefixed 8 byte string
 */
function formatNonce(nonce) {
    if (!nonce || nonce === '0x0') {
        return '0x0000000000000000';
    }
    if ((0, web3_validator_1.isHexPrefixed)(nonce)) {
        return `0x${(0, exports.stripHexPrefix)(nonce).padStart(16, '0')}`;
    }
    return `0x${nonce.padStart(16, '0')}`;
}
/**
 * Converts a `Number` into a hex `String`
 * @param {Number} i
 * @return {String}
 */
const intToHex = function (i) {
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
function parseGethParams(json, mergeForkIdPostMerge = true) {
    var _a, _b;
    // eslint-disable-next-line @typescript-eslint/no-unsafe-assignment
    const { name, config, difficulty, mixHash, gasLimit, coinbase, baseFeePerGas, } = json;
    // eslint-disable-next-line @typescript-eslint/no-unsafe-assignment
    let { extraData, timestamp, nonce } = json;
    const genesisTimestamp = Number(timestamp);
    // eslint-disable-next-line @typescript-eslint/no-unsafe-assignment
    const { chainId } = config;
    // geth is not strictly putting empty fields with a 0x prefix
    if (extraData === '') {
        extraData = '0x';
    }
    // geth may use number for timestamp
    if (!(0, web3_validator_1.isHexPrefixed)(timestamp)) {
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
        throw new Error('EIP155 block number must equal EIP 158 block number since both are part of SpuriousDragon hardfork and the client only supports activating the full hardfork');
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
        hardfork: undefined,
        hardforks: [],
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
                    period: (_a = config.clique.period) !== null && _a !== void 0 ? _a : config.clique.blockperiodseconds,
                    // eslint-disable-next-line @typescript-eslint/no-unsafe-member-access,  @typescript-eslint/no-unsafe-assignment
                    epoch: (_b = config.clique.epoch) !== null && _b !== void 0 ? _b : config.clique.epochlength,
                },
            }
            : {
                type: 'pow',
                algorithm: 'ethash',
                ethash: {},
            },
    };
    const forkMap = {
        [enums_js_1.Hardfork.Homestead]: { name: 'homesteadBlock' },
        [enums_js_1.Hardfork.Dao]: { name: 'daoForkBlock' },
        [enums_js_1.Hardfork.TangerineWhistle]: { name: 'eip150Block' },
        [enums_js_1.Hardfork.SpuriousDragon]: { name: 'eip155Block' },
        [enums_js_1.Hardfork.Byzantium]: { name: 'byzantiumBlock' },
        [enums_js_1.Hardfork.Constantinople]: { name: 'constantinopleBlock' },
        [enums_js_1.Hardfork.Petersburg]: { name: 'petersburgBlock' },
        [enums_js_1.Hardfork.Istanbul]: { name: 'istanbulBlock' },
        [enums_js_1.Hardfork.MuirGlacier]: { name: 'muirGlacierBlock' },
        [enums_js_1.Hardfork.Berlin]: { name: 'berlinBlock' },
        [enums_js_1.Hardfork.London]: { name: 'londonBlock' },
        [enums_js_1.Hardfork.MergeForkIdTransition]: {
            name: 'mergeForkBlock',
            postMerge: mergeForkIdPostMerge,
        },
        [enums_js_1.Hardfork.Shanghai]: { name: 'shanghaiTime', postMerge: true, isTimestamp: true },
        [enums_js_1.Hardfork.ShardingForkDev]: {
            name: 'shardingForkTime',
            postMerge: true,
            isTimestamp: true,
        },
    };
    // forkMapRev is the map from config field name to Hardfork
    const forkMapRev = Object.keys(forkMap).reduce((acc, elem) => {
        acc[forkMap[elem].name] = elem;
        return acc;
    }, {});
    // eslint-disable-next-line @typescript-eslint/no-unsafe-argument
    const configHardforkNames = Object.keys(config).filter(
    // eslint-disable-next-line no-null/no-null, @typescript-eslint/no-unsafe-member-access, @typescript-eslint/prefer-optional-chain
    key => forkMapRev[key] !== undefined && config[key] !== undefined && config[key] !== null);
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
        .filter(fork => fork.block !== null || fork.timestamp !== undefined);
    params.hardforks.sort((a, b) => { var _a, _b; return ((_a = a.block) !== null && _a !== void 0 ? _a : Infinity) - ((_b = b.block) !== null && _b !== void 0 ? _b : Infinity); });
    params.hardforks.sort((a, b) => { var _a, _b; return ((_a = a.timestamp) !== null && _a !== void 0 ? _a : genesisTimestamp) - ((_b = b.timestamp) !== null && _b !== void 0 ? _b : genesisTimestamp); });
    // eslint-disable-next-line @typescript-eslint/no-unsafe-member-access
    if (config.terminalTotalDifficulty !== undefined) {
        // Following points need to be considered for placement of merge hf
        // - Merge hardfork can't be placed at genesis
        // - Place merge hf before any hardforks that require CL participation for e.g. withdrawals
        // - Merge hardfork has to be placed just after genesis if any of the genesis hardforks make CL
        //   necessary for e.g. withdrawals
        const mergeConfig = {
            name: enums_js_1.Hardfork.Merge,
            // eslint-disable-next-line @typescript-eslint/no-unsafe-member-access, @typescript-eslint/no-unsafe-assignment
            ttd: config.terminalTotalDifficulty,
            // eslint-disable-next-line no-null/no-null
            block: null,
        };
        // Merge hardfork has to be placed before first hardfork that is dependent on merge
        const postMergeIndex = params.hardforks.findIndex(
        // eslint-disable-next-line @typescript-eslint/no-unsafe-member-access
        hf => { var _a; return ((_a = forkMap[hf.name]) === null || _a === void 0 ? void 0 : _a.postMerge) === true; });
        if (postMergeIndex !== -1) {
            params.hardforks.splice(postMergeIndex, 0, mergeConfig);
        }
        else {
            params.hardforks.push(mergeConfig);
        }
    }
    const latestHardfork = params.hardforks.length > 0 ? params.hardforks.slice(-1)[0] : undefined;
    params.hardfork = latestHardfork === null || latestHardfork === void 0 ? void 0 : latestHardfork.name;
    params.hardforks.unshift({ name: enums_js_1.Hardfork.Chainstart, block: 0 });
    return params;
}
/**
 * Parses a genesis.json exported from Geth into parameters for Common instance
 * @param json representing the Geth genesis file
 * @param name optional chain name
 * @returns parsed params
 */
// eslint-disable-next-line @typescript-eslint/no-explicit-any
function parseGethGenesis(json, name, mergeForkIdPostMerge) {
    try {
        if (['config', 'difficulty', 'gasLimit', 'alloc'].some(field => !(field in json))) {
            throw new Error('Invalid format, expected geth genesis fields missing');
        }
        if (name !== undefined) {
            // eslint-disable-next-line @typescript-eslint/no-unsafe-member-access, no-param-reassign
            json.name = name;
        }
        return parseGethParams(json, mergeForkIdPostMerge);
    }
    catch (e) {
        // eslint-disable-next-line @typescript-eslint/no-unsafe-member-access, @typescript-eslint/restrict-template-expressions
        throw new Error(`Error parsing parameters file: ${e.message}`);
    }
}
/**
 * Pads a `String` to have an even length
 * @param value
 * @return output
 */
function padToEven(value) {
    let a = value;
    if (typeof a !== 'string') {
        throw new Error(`[padToEven] value must be type 'string', received ${typeof a}`);
    }
    if (a.length % 2)
        a = `0${a}`;
    return a;
}
/**
 * Converts an `Number` to a `Uint8Array`
 * @param {Number} i
 * @return {Uint8Array}
 */
const intToUint8Array = function (i) {
    const hex = intToHex(i);
    return (0, web3_utils_1.hexToBytes)(`0x${padToEven(hex.slice(2))}`);
};
exports.intToUint8Array = intToUint8Array;
/**
 * Attempts to turn a value into a `Uint8Array`.
 * Inputs supported: `Uint8Array` `String` (hex-prefixed), `Number`, null/undefined, `BigInt` and other objects
 * with a `toArray()` or `toUint8Array()` method.
 * @param v the value
 */
const toUint8Array = function (v) {
    var _a;
    // eslint-disable-next-line no-null/no-null
    if (v === null || v === undefined) {
        return new Uint8Array();
    }
    if (v instanceof Uint8Array) {
        return v;
    }
    if (((_a = v === null || v === void 0 ? void 0 : v.constructor) === null || _a === void 0 ? void 0 : _a.name) === 'Uint8Array') {
        return Uint8Array.from(v);
    }
    if (Array.isArray(v)) {
        return Uint8Array.from(v);
    }
    if (typeof v === 'string') {
        if (!(0, web3_validator_1.isHexString)(v)) {
            throw new Error(`Cannot convert string to Uint8Array. only supports 0x-prefixed hex strings and this string was given: ${v}`);
        }
        return (0, web3_utils_1.hexToBytes)(padToEven((0, exports.stripHexPrefix)(v)));
    }
    if (typeof v === 'number') {
        return (0, exports.toUint8Array)((0, web3_utils_1.numberToHex)(v));
    }
    if (typeof v === 'bigint') {
        if (v < BigInt(0)) {
            throw new Error(`Cannot convert negative bigint to Uint8Array. Given: ${v}`);
        }
        let n = v.toString(16);
        if (n.length % 2)
            n = `0${n}`;
        return (0, exports.toUint8Array)(`0x${n}`);
    }
    if (v.toArray) {
        // converts a BN to a Uint8Array
        return Uint8Array.from(v.toArray());
    }
    throw new Error('invalid type');
};
exports.toUint8Array = toUint8Array;
/**
 * Converts a {@link Uint8Array} to a {@link bigint}
 */
function uint8ArrayToBigInt(buf) {
    const hex = (0, web3_utils_1.bytesToHex)(buf);
    if (hex === '0x') {
        return BigInt(0);
    }
    return BigInt(hex);
}
/**
 * Converts a {@link bigint} to a {@link Uint8Array}
 */
function bigIntToUint8Array(num) {
    return (0, exports.toUint8Array)(`0x${num.toString(16)}`);
}
/**
 * Returns a Uint8Array filled with 0s.
 * @param bytes the number of bytes the Uint8Array should be
 */
const zeros = function (bytes) {
    return new Uint8Array(bytes).fill(0);
};
exports.zeros = zeros;
/**
 * Pads a `Uint8Array` with zeros till it has `length` bytes.
 * Truncates the beginning or end of input if its length exceeds `length`.
 * @param msg the value to pad (Uint8Array)
 * @param length the number of bytes the output should be
 * @param right whether to start padding form the left or right
 * @return (Uint8Array)
 */
const setLength = function (msg, length, right) {
    const buf = (0, exports.zeros)(length);
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
function assertIsUint8Array(input) {
    if (!(0, web3_utils_1.isUint8Array)(input)) {
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
const setLengthLeft = function (msg, length) {
    assertIsUint8Array(msg);
    return setLength(msg, length, false);
};
exports.setLengthLeft = setLengthLeft;
/**
 * Trims leading zeros from a `Uint8Array`, `String` or `Number[]`.
 * @param a (Uint8Array|Array|String)
 * @return (Uint8Array|Array|String)
 */
function stripZeros(a) {
    // eslint-disable-next-line @typescript-eslint/no-unsafe-member-access, @typescript-eslint/no-unsafe-assignment
    let first = a[0];
    // eslint-disable-next-line @typescript-eslint/no-unsafe-member-access, @typescript-eslint/no-unsafe-call
    while (a.length > 0 && first.toString() === '0') {
        // eslint-disable-next-line @typescript-eslint/no-unsafe-member-access, @typescript-eslint/no-unsafe-assignment, prefer-destructuring, @typescript-eslint/no-unsafe-call, no-param-reassign
        a = a.slice(1);
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
const unpadUint8Array = function (a) {
    assertIsUint8Array(a);
    return stripZeros(a);
};
exports.unpadUint8Array = unpadUint8Array;
/**
 * Converts a {@link bigint} to a `0x` prefixed hex string
 */
const bigIntToHex = (num) => `0x${num.toString(16)}`;
exports.bigIntToHex = bigIntToHex;
/**
 * Convert value from bigint to an unpadded Uint8Array
 * (useful for RLP transport)
 * @param value value to convert
 */
function bigIntToUnpaddedUint8Array(value) {
    return (0, exports.unpadUint8Array)(bigIntToUint8Array(value));
}
function calculateSigRecovery(v, chainId) {
    if (v === BigInt(0) || v === BigInt(1))
        return v;
    if (chainId === undefined) {
        return v - BigInt(27);
    }
    return v - (chainId * BigInt(2) + BigInt(35));
}
function isValidSigRecovery(recovery) {
    return recovery === BigInt(0) || recovery === BigInt(1);
}
/**
 * ECDSA public key recovery from signature.
 * NOTE: Accepts `v === 0 | v === 1` for EIP1559 transactions
 * @returns Recovered public key
 */
const ecrecover = function (msgHash, v, r, s, chainId) {
    const recovery = calculateSigRecovery(v, chainId);
    if (!isValidSigRecovery(recovery)) {
        throw new Error('Invalid signature v value');
    }
    const senderPubKey = new constants_js_1.secp256k1.Signature(uint8ArrayToBigInt(r), uint8ArrayToBigInt(s))
        .addRecoveryBit(Number(recovery))
        .recoverPublicKey(msgHash)
        .toRawBytes(false);
    return senderPubKey.slice(1);
};
exports.ecrecover = ecrecover;
function toType(input, outputType) {
    // eslint-disable-next-line no-null/no-null
    if (input === null) {
        // eslint-disable-next-line no-null/no-null
        return null;
    }
    if (input === undefined) {
        return undefined;
    }
    if (typeof input === 'string' && !(0, web3_validator_1.isHexString)(input)) {
        throw new Error(`A string must be provided with a 0x-prefix, given: ${input}`);
    }
    else if (typeof input === 'number' && !Number.isSafeInteger(input)) {
        throw new Error('The provided number is greater than MAX_SAFE_INTEGER (please use an alternative input type)');
    }
    const output = (0, exports.toUint8Array)(input);
    switch (outputType) {
        case types_js_1.TypeOutput.Uint8Array:
            return output;
        case types_js_1.TypeOutput.BigInt:
            return uint8ArrayToBigInt(output);
        case types_js_1.TypeOutput.Number: {
            const bigInt = uint8ArrayToBigInt(output);
            if (bigInt > BigInt(Number.MAX_SAFE_INTEGER)) {
                throw new Error('The provided number is greater than MAX_SAFE_INTEGER (please use an alternative output type)');
            }
            return Number(bigInt);
        }
        case types_js_1.TypeOutput.PrefixedHexString:
            return (0, web3_utils_1.bytesToHex)(output);
        default:
            throw new Error('unknown outputType');
    }
}
//# sourceMappingURL=utils.js.map