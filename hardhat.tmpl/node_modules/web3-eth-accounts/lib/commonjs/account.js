"use strict";
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
var __awaiter = (this && this.__awaiter) || function (thisArg, _arguments, P, generator) {
    function adopt(value) { return value instanceof P ? value : new P(function (resolve) { resolve(value); }); }
    return new (P || (P = Promise))(function (resolve, reject) {
        function fulfilled(value) { try { step(generator.next(value)); } catch (e) { reject(e); } }
        function rejected(value) { try { step(generator["throw"](value)); } catch (e) { reject(e); } }
        function step(result) { result.done ? resolve(result.value) : adopt(result.value).then(fulfilled, rejected); }
        step((generator = generator.apply(thisArg, _arguments || [])).next());
    });
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.decrypt = exports.create = exports.privateKeyToAccount = exports.encrypt = exports.privateKeyToPublicKey = exports.privateKeyToAddress = exports.recover = exports.recoverTransaction = exports.signTransaction = exports.signRaw = exports.sign = exports.signMessageWithPrivateKey = exports.hashMessage = exports.parseAndValidatePrivateKey = void 0;
/**
 * The web3 accounts package contains functions to generate Ethereum accounts and sign transactions & data.
 *
 * For using accounts functions, first install Web3 package using `npm i web3` or `yarn add web3` based on your package manager usage.
 * After that, Accounts functions will be available as mentioned in following snippet.
 * ```ts
 * import {Web3} from 'web3';
 *
 * const web3 = new Web3();
 * const account = web3.eth.accounts.create();
 * const result = web3.eth.accounts.hashMessage("Test Message");
 *
 * ```
 *
 * For using individual package install `web3-eth-accounts` package using `npm i web3-eth-accounts` or `yarn add web3-eth-accounts` and only import required functions.
 * This is more efficient approach for building lightweight applications.
 * ```ts
 * import {create,hashMessage} from 'web3-eth-accounts';
 *
 * const account = create();
 * const result = hashMessage("Test Message");
 *
 * ```
 * @module Accounts
 *
 */
const aes_js_1 = require("ethereum-cryptography/aes.js");
const pbkdf2_js_1 = require("ethereum-cryptography/pbkdf2.js");
const scrypt_js_1 = require("ethereum-cryptography/scrypt.js");
const web3_errors_1 = require("web3-errors");
const web3_utils_1 = require("web3-utils");
const web3_validator_1 = require("web3-validator");
const constants_js_1 = require("./tx/constants.js");
const schemas_js_1 = require("./schemas.js");
const transactionFactory_js_1 = require("./tx/transactionFactory.js");
/**
 * Get the private key Uint8Array after the validation.
 * Note: This function is not exported through main web3 package, so for using it directly import from accounts package.
 * @param data - Private key
 * @param ignoreLength - Optional, ignore length check during validation
 * @returns The Uint8Array private key
 *
 * ```ts
 * parseAndValidatePrivateKey("0x08c673022000ece7964ea4db2d9369c50442b2869cbd8fc21baaca59e18f642c")
 *
 * > Uint8Array(32) [
 * 186,  26, 143, 168, 235, 179,  90,  75,
 * 101,  63,  84, 221, 152, 150,  30, 203,
 *   8, 113,  94, 226,  53, 213, 216,   5,
 * 194, 159,  17,  53, 219,  97, 121, 248
 * ]
 *
 * ```
 */
const parseAndValidatePrivateKey = (data, ignoreLength) => {
    let privateKeyUint8Array;
    // To avoid the case of 1 character less in a hex string which is prefixed with '0' by using 'bytesToUint8Array'
    if (!ignoreLength && typeof data === 'string' && (0, web3_validator_1.isHexStrict)(data) && data.length !== 66) {
        throw new web3_errors_1.PrivateKeyLengthError();
    }
    try {
        privateKeyUint8Array = (0, web3_utils_1.isUint8Array)(data) ? data : (0, web3_utils_1.bytesToUint8Array)(data);
    }
    catch (_a) {
        throw new web3_errors_1.InvalidPrivateKeyError();
    }
    if (!ignoreLength && privateKeyUint8Array.byteLength !== 32) {
        throw new web3_errors_1.PrivateKeyLengthError();
    }
    return privateKeyUint8Array;
};
exports.parseAndValidatePrivateKey = parseAndValidatePrivateKey;
/**
 *
 * Hashes the given message. The data will be `UTF-8 HEX` decoded and enveloped as follows:
 * `"\x19Ethereum Signed Message:\n" + message.length + message` and hashed using keccak256.
 *
 * @param message - A message to hash, if its HEX it will be UTF8 decoded.
 * @param skipPrefix - (default: false) If true, the message will be not prefixed with "\x19Ethereum Signed Message:\n" + message.length
 * @returns The hashed message
 *
 * ```ts
 * web3.eth.accounts.hashMessage("Hello world")
 *
 * > "0x8144a6fa26be252b86456491fbcd43c1de7e022241845ffea1c3df066f7cfede"
 *
 * web3.eth.accounts.hashMessage(web3.utils.utf8ToHex("Hello world")) // Will be hex decoded in hashMessage
 *
 * > "0x8144a6fa26be252b86456491fbcd43c1de7e022241845ffea1c3df066f7cfede"
 *
 * web3.eth.accounts.hashMessage("Hello world", true)
 *
 * > "0xed6c11b0b5b808960df26f5bfc471d04c1995b0ffd2055925ad1be28d6baadfd"
 * ```
 */
const hashMessage = (message, skipPrefix = false) => {
    const messageHex = (0, web3_validator_1.isHexStrict)(message) ? message : (0, web3_utils_1.utf8ToHex)(message);
    const messageBytes = (0, web3_utils_1.hexToBytes)(messageHex);
    const preamble = (0, web3_utils_1.hexToBytes)((0, web3_utils_1.fromUtf8)(`\x19Ethereum Signed Message:\n${messageBytes.byteLength}`));
    const ethMessage = skipPrefix ? messageBytes : (0, web3_utils_1.uint8ArrayConcat)(preamble, messageBytes);
    return (0, web3_utils_1.sha3Raw)(ethMessage); // using keccak in web3-utils.sha3Raw instead of SHA3 (NIST Standard) as both are different
};
exports.hashMessage = hashMessage;
/**
 * Takes a hash of a message and a private key, signs the message using the SECP256k1 elliptic curve algorithm, and returns the signature components.
 * @param hash - The hash of the message to be signed, represented as a hexadecimal string.
 * @param privateKey - The private key used to sign the message, represented as a byte array.
 * @returns - The signature Object containing the message, messageHash, signature r, s, v
 */
const signMessageWithPrivateKey = (hash, privateKey) => {
    const privateKeyUint8Array = (0, exports.parseAndValidatePrivateKey)(privateKey);
    const signature = constants_js_1.secp256k1.sign(hash.substring(2), privateKeyUint8Array);
    const signatureBytes = signature.toCompactRawBytes();
    const r = signature.r.toString(16).padStart(64, '0');
    const s = signature.s.toString(16).padStart(64, '0');
    const v = signature.recovery + 27;
    return {
        messageHash: hash,
        v: (0, web3_utils_1.numberToHex)(v),
        r: `0x${r}`,
        s: `0x${s}`,
        signature: `${(0, web3_utils_1.bytesToHex)(signatureBytes)}${v.toString(16)}`,
    };
};
exports.signMessageWithPrivateKey = signMessageWithPrivateKey;
/**
 * Signs arbitrary data with a given private key.
 * :::info
 * The value passed as the data parameter will be UTF-8 HEX decoded and wrapped as follows: "\\x19Ethereum Signed Message:\\n" + message.length + message
 * :::

 * @param data - The data to sign
 * @param privateKey - The 32 byte private key to sign with
 * @returns The signature Object containing the message, messageHash, signature r, s, v
 *
 * ```ts
 * web3.eth.accounts.sign('Some data', '0x4c0883a69102937d6231471b5dbb6204fe5129617082792ae468d01a3f362318')
 * > {
 * message: 'Some data',
 * messageHash: '0x1da44b586eb0729ff70a73c326926f6ed5a25f5b056e7f47fbc6e58d86871655',
 * v: '0x1c',
 * r: '0xb91467e570a6466aa9e9876cbcd013baba02900b8979d43fe208a4a4f339f5fd',
 * s: '0x6007e74cd82e037b800186422fc2da167c747ef045e5d18a5f5d4300f8e1a029',
 * signature: '0xb91467e570a6466aa9e9876cbcd013baba02900b8979d43fe208a4a4f339f5fd6007e74cd82e037b800186422fc2da167c747ef045e5d18a5f5d4300f8e1a0291c'
 * }
 * ```
 */
const sign = (data, privateKey) => {
    const hash = (0, exports.hashMessage)(data);
    const { messageHash, v, r, s, signature } = (0, exports.signMessageWithPrivateKey)(hash, privateKey);
    return {
        message: data,
        messageHash,
        v,
        r,
        s,
        signature,
    };
};
exports.sign = sign;
/**
 * Signs raw data with a given private key without adding the Ethereum-specific prefix.
 *
 * @param data - The raw data to sign. If it's a hex string, it will be used as-is. Otherwise, it will be UTF-8 encoded.
 * @param privateKey - The 32 byte private key to sign with
 * @returns The signature Object containing the message, messageHash, signature r, s, v
 *
 * ```ts
 * web3.eth.accounts.signRaw('Some data', '0x4c0883a69102937d6231471b5dbb6204fe5129617082792ae468d01a3f362318')
 * > {
 *   message: 'Some data',
 *   messageHash: '0x43a26051362b8040b289abe93334a5e3662751aa691185ae9e9a2e1e0c169350',
 *   v: '0x1b',
 *   r: '0x93da7e2ddd6b2ff1f5af0c752f052ed0d7d5bff19257db547a69cd9a879b37d4',
 *   s: '0x334485e42b33815fd2cf8a245a5393b282214060844a9681495df2257140e75c',
 *   signature: '0x93da7e2ddd6b2ff1f5af0c752f052ed0d7d5bff19257db547a69cd9a879b37d4334485e42b33815fd2cf8a245a5393b282214060844a9681495df2257140e75c1b'
 * }
 * ```
 */
const signRaw = (data, privateKey) => {
    // Hash the message without the Ethereum-specific prefix
    const hash = (0, exports.hashMessage)(data, true);
    // Sign the hash with the private key
    const { messageHash, v, r, s, signature } = (0, exports.signMessageWithPrivateKey)(hash, privateKey);
    return {
        message: data,
        messageHash,
        v,
        r,
        s,
        signature,
    };
};
exports.signRaw = signRaw;
/**
 * Signs an Ethereum transaction with a given private key.
 *
 * @param transaction - The transaction, must be a legacy, EIP2930 or EIP 1559 transaction type
 * @param privateKey -  The private key to import. This is 32 bytes of random data.
 * @returns A signTransactionResult object that contains message hash, r, s, v, transaction hash and raw transaction.
 *
 * This function is not stateful here. We need network access to get the account `nonce` and `chainId` to sign the transaction.
 * This function will rely on user to provide the full transaction to be signed. If you want to sign a partial transaction object
 * Use {@link Web3.eth.accounts.sign} instead.
 *
 * Signing a legacy transaction
 * ```ts
 * import {signTransaction, Transaction} from 'web3-eth-accounts';
 *
 * signTransaction(new Transaction({
 *	to: '0x118C2E5F57FD62C2B5b46a5ae9216F4FF4011a07',
 *	value: '0x186A0',
 *	gasLimit: '0x520812',
 *	gasPrice: '0x09184e72a000',
 *	data: '',
 *	chainId: 1,
 *	nonce: 0 }),
 * '0x4c0883a69102937d6231471b5dbb6204fe5129617082792ae468d01a3f362318')
 *
 * > {
 * messageHash: '0x28b7b75f7ba48d588a902c1ff4d5d13cc0ca9ac0aaa39562368146923fb853bf',
 * v: '0x25',
 * r: '0x601b0017b0e20dd0eeda4b895fbc1a9e8968990953482214f880bae593e71b5',
 * s: '0x690d984493560552e3ebdcc19a65b9c301ea9ddc82d3ab8cfde60485fd5722ce',
 * rawTransaction: '0xf869808609184e72a0008352081294118c2e5f57fd62c2b5b46a5ae9216f4ff4011a07830186a08025a00601b0017b0e20dd0eeda4b895fbc1a9e8968990953482214f880bae593e71b5a0690d984493560552e3ebdcc19a65b9c301ea9ddc82d3ab8cfde60485fd5722ce',
 * transactionHash: '0xc5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470'
 * ```
 *
 * Signing an eip 1559 transaction
 * ```ts
 * import {signTransaction, Transaction} from 'web3-eth-accounts';
 *
 * signTransaction(new Transaction({
 *	to: '0xF0109fC8DF283027b6285cc889F5aA624EaC1F55',
 *	maxPriorityFeePerGas: '0x3B9ACA00',
 *	maxFeePerGas: '0xB2D05E00',
 *	gasLimit: '0x6A4012',
 *	value: '0x186A0',
 *	data: '',
 *	chainId: 1,
 *	nonce: 0}),
 * "0x4c0883a69102937d6231471b5dbb6204fe5129617082792ae468d01a3f362318")
 * > {
 *  messageHash: '0x5744f24d5f0aff6c70487c8e85adf07d8564e50b08558788f00479611d7bae5f',
 * v: '0x25',
 * r: '0x78a5a6b2876c3985f90f82073d18d57ac299b608cc76a4ba697b8bb085048347',
 * s: '0x9cfcb40cc7d505ed17ff2d3337b51b066648f10c6b7e746117de69b2eb6358d',
 * rawTransaction: '0xf8638080836a401294f0109fc8df283027b6285cc889f5aa624eac1f55830186a08025a078a5a6b2876c3985f90f82073d18d57ac299b608cc76a4ba697b8bb085048347a009cfcb40cc7d505ed17ff2d3337b51b066648f10c6b7e746117de69b2eb6358d',
 * transactionHash: '0xc5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470'
 * }
 * ```
 *
 * Signing an eip 2930 transaction
 * ```ts
 * import {signTransaction, Transaction} from 'web3-eth-accounts';
 *
 * signTransaction(new Transaction ({
 *	chainId: 1,
 *	nonce: 0,
 *	gasPrice: '0x09184e72a000',
 *	gasLimit: '0x2710321',
 *	to: '0xF0109fC8DF283027b6285cc889F5aA624EaC1F55',
 *	value: '0x186A0',
 *	data: '',
 *	accessList: [
 *		{
 *			address: '0x0000000000000000000000000000000000000101',
 *			storageKeys: [
 *				'0x0000000000000000000000000000000000000000000000000000000000000000',
 *				'0x00000000000000000000000000000000000000000000000000000000000060a7',
 *			],
 *		},
 *	],
 * }),"0x4c0883a69102937d6231471b5dbb6204fe5129617082792ae468d01a3f362318")
 *
 * > {
 * messageHash: '0xc55ea24bdb4c379550a7c9a6818ac39ca33e75bc78ddb862bd82c31cc1c7a073',
 * v: '0x26',
 * r: '0x27344e77871c8b2068bc998bf28e0b5f9920867a69c455b2ed0c1c150fec098e',
 * s: '0x519f0130a1d662841d4a28082e9c9bb0a15e0e59bb46cfc39a52f0e285dec6b9',
 * rawTransaction: '0xf86a808609184e72a000840271032194f0109fc8df283027b6285cc889f5aa624eac1f55830186a08026a027344e77871c8b2068bc998bf28e0b5f9920867a69c455b2ed0c1c150fec098ea0519f0130a1d662841d4a28082e9c9bb0a15e0e59bb46cfc39a52f0e285dec6b9',
 * transactionHash: '0xc5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470'
 * }
 * ```
 */
const signTransaction = (transaction, privateKey) => __awaiter(void 0, void 0, void 0, function* () {
    const signedTx = transaction.sign((0, web3_utils_1.hexToBytes)(privateKey));
    if ((0, web3_validator_1.isNullish)(signedTx.v) || (0, web3_validator_1.isNullish)(signedTx.r) || (0, web3_validator_1.isNullish)(signedTx.s))
        throw new web3_errors_1.TransactionSigningError('Signer Error');
    const validationErrors = signedTx.validate(true);
    if (validationErrors.length > 0) {
        let errorString = 'Signer Error ';
        for (const validationError of validationErrors) {
            errorString += `${errorString} ${validationError}.`;
        }
        throw new web3_errors_1.TransactionSigningError(errorString);
    }
    const rawTx = (0, web3_utils_1.bytesToHex)(signedTx.serialize());
    const txHash = (0, web3_utils_1.sha3Raw)(rawTx); // using keccak in web3-utils.sha3Raw instead of SHA3 (NIST Standard) as both are different
    return {
        messageHash: (0, web3_utils_1.bytesToHex)(signedTx.getMessageToSign(true)),
        v: `0x${signedTx.v.toString(16)}`,
        r: `0x${signedTx.r.toString(16).padStart(64, '0')}`,
        s: `0x${signedTx.s.toString(16).padStart(64, '0')}`,
        rawTransaction: rawTx,
        transactionHash: (0, web3_utils_1.bytesToHex)(txHash),
    };
});
exports.signTransaction = signTransaction;
/**
 * Recovers the Ethereum address which was used to sign the given RLP encoded transaction.
 *
 * @param rawTransaction - The hex string having RLP encoded transaction
 * @returns The Ethereum address used to sign this transaction
 * ```ts
 * web3.eth.accounts.recoverTransaction('0xf869808504e3b29200831e848094f0109fc8df283027b6285cc889f5aa624eac1f55843b9aca008025a0c9cf86333bcb065d140032ecaab5d9281bde80f21b9687b3e94161de42d51895a0727a108a0b8d101465414033c3f705a9c7b826e596766046ee1183dbc8aeaa68');
 * > "0x2c7536E3605D9C16a7a3D7b1898e529396a65c23"
 * ```
 */
const recoverTransaction = (rawTransaction) => {
    if ((0, web3_validator_1.isNullish)(rawTransaction))
        throw new web3_errors_1.UndefinedRawTransactionError();
    const tx = transactionFactory_js_1.TransactionFactory.fromSerializedData((0, web3_utils_1.hexToBytes)(rawTransaction));
    return (0, web3_utils_1.toChecksumAddress)(tx.getSenderAddress().toString());
};
exports.recoverTransaction = recoverTransaction;
/**
 * Recovers the Ethereum address which was used to sign the given data
 *
 * @param data - Either a signed message, hash, or the {@link signatureObject}
 * @param signature - The raw RLP encoded signature
 * @param signatureOrV - signature or V
 * @param prefixedOrR - prefixed or R
 * @param s - S value in signature
 * @param prefixed - (default: false) If the last parameter is true, the given message will NOT automatically be prefixed with `"\\x19Ethereum Signed Message:\\n" + message.length + message`, and assumed to be already prefixed and hashed.
 * @returns The Ethereum address used to sign this data
 *
 * ```ts
 * const data = 'Some data';
 * const sigObj = web3.eth.accounts.sign(data, '0xbe6383dad004f233317e46ddb46ad31b16064d14447a95cc1d8c8d4bc61c3728')
 *
 * > {
 *   message: 'Some data',
 *   messageHash: '0x1da44b586eb0729ff70a73c326926f6ed5a25f5b056e7f47fbc6e58d86871655',
 *   v: '0x1b',
 *   r: '0xa8037a6116c176a25e6fc224947fde9e79a2deaa0dd8b67b366fbdfdbffc01f9',
 *   s: '0x53e41351267b20d4a89ebfe9c8f03c04de9b345add4a52f15bd026b63c8fb150',
 *   signature: '0xa8037a6116c176a25e6fc224947fde9e79a2deaa0dd8b67b366fbdfdbffc01f953e41351267b20d4a89ebfe9c8f03c04de9b345add4a52f15bd026b63c8fb1501b'
 * }
 *
 * // now recover
 * web3.eth.accounts.recover(data, sigObj.v, sigObj.r, sigObj.s)
 *
 * > 0xEB014f8c8B418Db6b45774c326A0E64C78914dC0
 * ```
 */
const recover = (data, signatureOrV, prefixedOrR, s, prefixed) => {
    if (typeof data === 'object') {
        const signatureStr = `${data.r}${data.s.slice(2)}${data.v.slice(2)}`;
        return (0, exports.recover)(data.messageHash, signatureStr, prefixedOrR);
    }
    if (typeof signatureOrV === 'string' && typeof prefixedOrR === 'string' && !(0, web3_validator_1.isNullish)(s)) {
        const signatureStr = `${prefixedOrR}${s.slice(2)}${signatureOrV.slice(2)}`;
        return (0, exports.recover)(data, signatureStr, prefixed);
    }
    if ((0, web3_validator_1.isNullish)(signatureOrV))
        throw new web3_errors_1.InvalidSignatureError('signature string undefined');
    const V_INDEX = 130; // r = first 32 bytes, s = second 32 bytes, v = last byte of signature
    const hashedMessage = prefixedOrR ? data : (0, exports.hashMessage)(data);
    let v = parseInt(signatureOrV.substring(V_INDEX), 16); // 0x + r + s + v
    if (v > 26) {
        v -= 27;
    }
    const ecPublicKey = constants_js_1.secp256k1.Signature.fromCompact(signatureOrV.slice(2, V_INDEX))
        .addRecoveryBit(v)
        .recoverPublicKey(hashedMessage.replace('0x', ''))
        .toRawBytes(false);
    const publicHash = (0, web3_utils_1.sha3Raw)(ecPublicKey.subarray(1));
    const address = (0, web3_utils_1.toChecksumAddress)(`0x${publicHash.slice(-40)}`);
    return address;
};
exports.recover = recover;
/**
 * Get the ethereum Address from a private key
 *
 * @param privateKey - String or Uint8Array of 32 bytes
 * @param ignoreLength - if true, will not error check length
 * @returns The Ethereum address
 * @example
 * ```ts
 * web3.eth.accounts.privateKeyToAddress("0xbe6383dad004f233317e46ddb46ad31b16064d14447a95cc1d8c8d4bc61c3728")
 *
 * > "0xEB014f8c8B418Db6b45774c326A0E64C78914dC0"
 * ```
 */
const privateKeyToAddress = (privateKey) => {
    const privateKeyUint8Array = (0, exports.parseAndValidatePrivateKey)(privateKey);
    // Get public key from private key in compressed format
    const publicKey = constants_js_1.secp256k1.getPublicKey(privateKeyUint8Array, false);
    // Uncompressed ECDSA public key contains the prefix `0x04` which is not used in the Ethereum public key
    const publicKeyHash = (0, web3_utils_1.sha3Raw)(publicKey.slice(1));
    // The hash is returned as 256 bits (32 bytes) or 64 hex characters
    // To get the address, take the last 20 bytes of the public hash
    const address = publicKeyHash.slice(-40);
    return (0, web3_utils_1.toChecksumAddress)(`0x${address}`);
};
exports.privateKeyToAddress = privateKeyToAddress;
/**
 * Get the public key from a private key
 *
 * @param privateKey - String or Uint8Array of 32 bytes
 * @param isCompressed - if true, will generate a 33 byte compressed public key instead of a 65 byte public key
 * @returns The public key
 * @example
 * ```ts
 * web3.eth.accounts.privateKeyToPublicKey("0x1e046a882bb38236b646c9f135cf90ad90a140810f439875f2a6dd8e50fa261f", true)
 *
 * > "0x42beb65f179720abaa3ec9a70a539629cbbc5ec65bb57e7fc78977796837e537662dd17042e6449dc843c281067a4d6d8d1a1775a13c41901670d5de7ee6503a" // uncompressed public key
 * ```
 */
const privateKeyToPublicKey = (privateKey, isCompressed) => {
    const privateKeyUint8Array = (0, exports.parseAndValidatePrivateKey)(privateKey);
    // Get public key from private key in compressed format
    return `0x${(0, web3_utils_1.bytesToHex)(constants_js_1.secp256k1.getPublicKey(privateKeyUint8Array, isCompressed)).slice(4)}`; // 0x and removing compression byte
};
exports.privateKeyToPublicKey = privateKeyToPublicKey;
/**
 * encrypt a private key with a password, returns a V3 JSON Keystore
 *
 * Read more: https://github.com/ethereum/wiki/wiki/Web3-Secret-Storage-Definition
 *
 * @param privateKey - The private key to encrypt, 32 bytes.
 * @param password - The password used for encryption.
 * @param options - Options to configure to encrypt the keystore either scrypt or pbkdf2
 * @returns Returns a V3 JSON Keystore
 *
 * Encrypt using scrypt options:
 * ```ts
 *
 * web3.eth.accounts.encrypt(
 *    '0x67f476289210e3bef3c1c75e4de993ff0a00663df00def84e73aa7411eac18a6',
 *    '123',
 *    {
 *        n: 8192,
 *	    iv: web3.utils.hexToBytes('0xbfb43120ae00e9de110f8325143a2709'),
 *	    salt: web3.utils.hexToBytes('0x210d0ec956787d865358ac45716e6dd42e68d48e346d795746509523aeb477dd'),
 *	}).then(console.log)
 *
 * > {
 * version: 3,
 * id: 'c0cb0a94-4702-4492-b6e6-eb2ac404344a',
 * address: 'cda9a91875fc35c8ac1320e098e584495d66e47c',
 * crypto: {
 *   ciphertext: 'cb3e13e3281ff3861a3f0257fad4c9a51b0eb046f9c7821825c46b210f040b8f',
 *   cipherparams: { iv: 'bfb43120ae00e9de110f8325143a2709' },
 *   cipher: 'aes-128-ctr',
 *   kdf: 'scrypt',
 *   kdfparams: {
 *     n: 8192,
 *     r: 8,
 *     p: 1,
 *     dklen: 32,
 *     salt: '210d0ec956787d865358ac45716e6dd42e68d48e346d795746509523aeb477dd'
 *   },
 *   mac: 'efbf6d3409f37c0084a79d5fdf9a6f5d97d11447517ef1ea8374f51e581b7efd'
 * }
 *}
 *```
 *
 * Encrypting using pbkdf2 options:
 * ```ts
 * web3.eth.accounts.encrypt('0x348ce564d427a3311b6536bbcff9390d69395b06ed6c486954e971d960fe8709',
 *'123',
 *{
 *	iv: 'bfb43120ae00e9de110f8325143a2709',
 *	salt: '210d0ec956787d865358ac45716e6dd42e68d48e346d795746509523aeb477dd',
 *	c: 262144,
 *	kdf: 'pbkdf2',
 *}).then(console.log)
 *
 * >
 * {
 *   version: 3,
 *   id: '77381417-0973-4e4b-b590-8eb3ace0fe2d',
 *   address: 'b8ce9ab6943e0eced004cde8e3bbed6568b2fa01',
 *   crypto: {
 *     ciphertext: '76512156a34105fa6473ad040c666ae7b917d14c06543accc0d2dc28e6073b12',
 *     cipherparams: { iv: 'bfb43120ae00e9de110f8325143a2709' },
 *     cipher: 'aes-128-ctr',
 *     kdf: 'pbkdf2',
 *     kdfparams: {
 *       dklen: 32,
 *       salt: '210d0ec956787d865358ac45716e6dd42e68d48e346d795746509523aeb477dd',
 *       c: 262144,
 *       prf: 'hmac-sha256'
 *     },
 *   mac: '46eb4884e82dc43b5aa415faba53cc653b7038e9d61cc32fd643cf8c396189b7'
 *   }
 * }
 *```
 */
const encrypt = (privateKey, password, options) => __awaiter(void 0, void 0, void 0, function* () {
    var _a, _b, _c, _d, _e, _f, _g;
    const privateKeyUint8Array = (0, exports.parseAndValidatePrivateKey)(privateKey);
    // if given salt or iv is a string, convert it to a Uint8Array
    let salt;
    if (options === null || options === void 0 ? void 0 : options.salt) {
        salt = typeof options.salt === 'string' ? (0, web3_utils_1.hexToBytes)(options.salt) : options.salt;
    }
    else {
        salt = (0, web3_utils_1.randomBytes)(32);
    }
    if (!((0, web3_validator_1.isString)(password) || (0, web3_utils_1.isUint8Array)(password))) {
        throw new web3_errors_1.InvalidPasswordError();
    }
    const uint8ArrayPassword = typeof password === 'string' ? (0, web3_utils_1.hexToBytes)((0, web3_utils_1.utf8ToHex)(password)) : password;
    let initializationVector;
    if (options === null || options === void 0 ? void 0 : options.iv) {
        initializationVector = typeof options.iv === 'string' ? (0, web3_utils_1.hexToBytes)(options.iv) : options.iv;
        if (initializationVector.length !== 16) {
            throw new web3_errors_1.IVLengthError();
        }
    }
    else {
        initializationVector = (0, web3_utils_1.randomBytes)(16);
    }
    const kdf = (_a = options === null || options === void 0 ? void 0 : options.kdf) !== null && _a !== void 0 ? _a : 'scrypt';
    let derivedKey;
    let kdfparams;
    // derive key from key derivation function
    if (kdf === 'pbkdf2') {
        kdfparams = {
            dklen: (_b = options === null || options === void 0 ? void 0 : options.dklen) !== null && _b !== void 0 ? _b : 32,
            salt: (0, web3_utils_1.bytesToHex)(salt).replace('0x', ''),
            c: (_c = options === null || options === void 0 ? void 0 : options.c) !== null && _c !== void 0 ? _c : 262144,
            prf: 'hmac-sha256',
        };
        if (kdfparams.c < 1000) {
            // error when c < 1000, pbkdf2 is less secure with less iterations
            throw new web3_errors_1.PBKDF2IterationsError();
        }
        derivedKey = (0, pbkdf2_js_1.pbkdf2Sync)(uint8ArrayPassword, salt, kdfparams.c, kdfparams.dklen, 'sha256');
    }
    else if (kdf === 'scrypt') {
        kdfparams = {
            n: (_d = options === null || options === void 0 ? void 0 : options.n) !== null && _d !== void 0 ? _d : 8192,
            r: (_e = options === null || options === void 0 ? void 0 : options.r) !== null && _e !== void 0 ? _e : 8,
            p: (_f = options === null || options === void 0 ? void 0 : options.p) !== null && _f !== void 0 ? _f : 1,
            dklen: (_g = options === null || options === void 0 ? void 0 : options.dklen) !== null && _g !== void 0 ? _g : 32,
            salt: (0, web3_utils_1.bytesToHex)(salt).replace('0x', ''),
        };
        derivedKey = (0, scrypt_js_1.scryptSync)(uint8ArrayPassword, salt, kdfparams.n, kdfparams.p, kdfparams.r, kdfparams.dklen);
    }
    else {
        throw new web3_errors_1.InvalidKdfError();
    }
    const cipher = yield (0, aes_js_1.encrypt)(privateKeyUint8Array, derivedKey.slice(0, 16), initializationVector, 'aes-128-ctr');
    const ciphertext = (0, web3_utils_1.bytesToHex)(cipher).slice(2);
    const mac = (0, web3_utils_1.sha3Raw)((0, web3_utils_1.uint8ArrayConcat)(derivedKey.slice(16, 32), cipher)).replace('0x', '');
    return {
        version: 3,
        id: (0, web3_utils_1.uuidV4)(),
        address: (0, exports.privateKeyToAddress)(privateKeyUint8Array).toLowerCase().replace('0x', ''),
        crypto: {
            ciphertext,
            cipherparams: {
                iv: (0, web3_utils_1.bytesToHex)(initializationVector).replace('0x', ''),
            },
            cipher: 'aes-128-ctr',
            kdf,
            kdfparams,
            mac,
        },
    };
});
exports.encrypt = encrypt;
/**
 * Get an Account object from the privateKey
 *
 * @param privateKey - String or Uint8Array of 32 bytes
 * @param ignoreLength - if true, will not error check length
 * @returns A Web3Account object
 *
 * :::info
 * The `Web3Account.signTransaction` is not stateful if directly imported from accounts package and used. Network access is required to get the account `nonce` and `chainId` to sign the transaction, so use {@link Web3.eth.accounts.signTransaction} for signing transactions.
 * ::::
 *
 * ```ts
 * web3.eth.accounts.privateKeyToAccount("0x348ce564d427a3311b6536bbcff9390d69395b06ed6c486954e971d960fe8709");
 *
 * >    {
 * 			address: '0xb8CE9ab6943e0eCED004cDe8e3bBed6568B2Fa01',
 * 			privateKey: '0x348ce564d427a3311b6536bbcff9390d69395b06ed6c486954e971d960fe8709',
 * 			sign,
 * 			signTransaction,
 * 			encrypt,
 * 	}
 * ```
 */
const privateKeyToAccount = (privateKey, ignoreLength) => {
    const privateKeyUint8Array = (0, exports.parseAndValidatePrivateKey)(privateKey, ignoreLength);
    return {
        address: (0, exports.privateKeyToAddress)(privateKeyUint8Array),
        privateKey: (0, web3_utils_1.bytesToHex)(privateKeyUint8Array),
        // eslint-disable-next-line @typescript-eslint/no-unused-vars
        signTransaction: (_tx) => {
            throw new web3_errors_1.TransactionSigningError('Do not have network access to sign the transaction');
        },
        sign: (data) => (0, exports.sign)(typeof data === 'string' ? data : JSON.stringify(data), privateKeyUint8Array),
        encrypt: (password, options) => __awaiter(void 0, void 0, void 0, function* () { return (0, exports.encrypt)(privateKeyUint8Array, password, options); }),
    };
};
exports.privateKeyToAccount = privateKeyToAccount;
/**
 *
 * Generates and returns a Web3Account object that includes the private and public key
 * For creation of private key, it uses an audited package ethereum-cryptography/secp256k1
 * that is cryptographically secure random number with certain characteristics.
 * Read more: https://www.npmjs.com/package/ethereum-cryptography#secp256k1-curve
 *
 * @returns A Web3Account object
 * ```ts
 * web3.eth.accounts.create();
 * {
 * address: '0xbD504f977021b5E5DdccD8741A368b147B3B38bB',
 * privateKey: '0x964ced1c69ad27a311c432fdc0d8211e987595f7eb34ab405a5f16bdc9563ec5',
 * signTransaction: [Function: signTransaction],
 * sign: [Function: sign],
 * encrypt: [AsyncFunction: encrypt]
 * }
 * ```
 */
const create = () => {
    const privateKey = constants_js_1.secp256k1.utils.randomPrivateKey();
    return (0, exports.privateKeyToAccount)(`${(0, web3_utils_1.bytesToHex)(privateKey)}`);
};
exports.create = create;
/**
 * Decrypts a v3 keystore JSON, and creates the account.
 *
 * @param keystore - the encrypted Keystore object or string to decrypt
 * @param password - The password that was used for encryption
 * @param nonStrict - if true and given a json string, the keystore will be parsed as lowercase.
 * @returns Returns the decrypted Web3Account object
 * Decrypting scrypt
 *
 * ```ts
 * web3.eth.accounts.decrypt({
 *   version: 3,
 *   id: 'c0cb0a94-4702-4492-b6e6-eb2ac404344a',
 *   address: 'cda9a91875fc35c8ac1320e098e584495d66e47c',
 *   crypto: {
 *   ciphertext: 'cb3e13e3281ff3861a3f0257fad4c9a51b0eb046f9c7821825c46b210f040b8f',
 *      cipherparams: { iv: 'bfb43120ae00e9de110f8325143a2709' },
 *      cipher: 'aes-128-ctr',
 *      kdf: 'scrypt',
 *      kdfparams: {
 *        n: 8192,
 *        r: 8,
 *        p: 1,
 *        dklen: 32,
 *        salt: '210d0ec956787d865358ac45716e6dd42e68d48e346d795746509523aeb477dd'
 *      },
 *      mac: 'efbf6d3409f37c0084a79d5fdf9a6f5d97d11447517ef1ea8374f51e581b7efd'
 *    }
 *   }, '123').then(console.log);
 *
 *
 * > {
 * address: '0xcdA9A91875fc35c8Ac1320E098e584495d66e47c',
 * privateKey: '67f476289210e3bef3c1c75e4de993ff0a00663df00def84e73aa7411eac18a6',
 * signTransaction: [Function: signTransaction],
 * sign: [Function: sign],
 * encrypt: [AsyncFunction: encrypt]
 * }
 * ```
 */
const decrypt = (keystore, password, nonStrict) => __awaiter(void 0, void 0, void 0, function* () {
    const json = typeof keystore === 'object'
        ? keystore
        : JSON.parse(nonStrict ? keystore.toLowerCase() : keystore);
    web3_validator_1.validator.validateJSONSchema(schemas_js_1.keyStoreSchema, json);
    if (json.version !== 3)
        throw new web3_errors_1.KeyStoreVersionError();
    const uint8ArrayPassword = typeof password === 'string' ? (0, web3_utils_1.hexToBytes)((0, web3_utils_1.utf8ToHex)(password)) : password;
    web3_validator_1.validator.validate(['bytes'], [uint8ArrayPassword]);
    let derivedKey;
    if (json.crypto.kdf === 'scrypt') {
        const kdfparams = json.crypto.kdfparams;
        const uint8ArraySalt = typeof kdfparams.salt === 'string' ? (0, web3_utils_1.hexToBytes)(kdfparams.salt) : kdfparams.salt;
        derivedKey = (0, scrypt_js_1.scryptSync)(uint8ArrayPassword, uint8ArraySalt, kdfparams.n, kdfparams.p, kdfparams.r, kdfparams.dklen);
    }
    else if (json.crypto.kdf === 'pbkdf2') {
        const kdfparams = json.crypto.kdfparams;
        const uint8ArraySalt = typeof kdfparams.salt === 'string' ? (0, web3_utils_1.hexToBytes)(kdfparams.salt) : kdfparams.salt;
        derivedKey = (0, pbkdf2_js_1.pbkdf2Sync)(uint8ArrayPassword, uint8ArraySalt, kdfparams.c, kdfparams.dklen, 'sha256');
    }
    else {
        throw new web3_errors_1.InvalidKdfError();
    }
    const ciphertext = (0, web3_utils_1.hexToBytes)(json.crypto.ciphertext);
    const mac = (0, web3_utils_1.sha3Raw)((0, web3_utils_1.uint8ArrayConcat)(derivedKey.slice(16, 32), ciphertext)).replace('0x', '');
    if (mac !== json.crypto.mac) {
        throw new web3_errors_1.KeyDerivationError();
    }
    const seed = yield (0, aes_js_1.decrypt)((0, web3_utils_1.hexToBytes)(json.crypto.ciphertext), derivedKey.slice(0, 16), (0, web3_utils_1.hexToBytes)(json.crypto.cipherparams.iv));
    return (0, exports.privateKeyToAccount)(seed);
});
exports.decrypt = decrypt;
//# sourceMappingURL=account.js.map