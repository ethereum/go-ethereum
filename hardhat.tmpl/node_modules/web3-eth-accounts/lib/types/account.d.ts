import { Address, Bytes, CipherOptions, HexString, KeyStore, SignatureObject, SignResult, SignTransactionResult } from 'web3-types';
import type { TypedTransaction, Web3Account } from './types.js';
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
export declare const parseAndValidatePrivateKey: (data: Bytes, ignoreLength?: boolean) => Uint8Array;
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
export declare const hashMessage: (message: string, skipPrefix?: boolean) => string;
/**
 * Takes a hash of a message and a private key, signs the message using the SECP256k1 elliptic curve algorithm, and returns the signature components.
 * @param hash - The hash of the message to be signed, represented as a hexadecimal string.
 * @param privateKey - The private key used to sign the message, represented as a byte array.
 * @returns - The signature Object containing the message, messageHash, signature r, s, v
 */
export declare const signMessageWithPrivateKey: (hash: HexString, privateKey: Bytes) => SignResult;
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
export declare const sign: (data: string, privateKey: Bytes) => SignResult;
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
export declare const signRaw: (data: string, privateKey: Bytes) => SignResult;
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
export declare const signTransaction: (transaction: TypedTransaction, privateKey: HexString) => Promise<SignTransactionResult>;
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
export declare const recoverTransaction: (rawTransaction: HexString) => Address;
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
export declare const recover: (data: string | SignatureObject, signatureOrV?: string, prefixedOrR?: boolean | string, s?: string, prefixed?: boolean) => Address;
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
export declare const privateKeyToAddress: (privateKey: Bytes) => string;
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
export declare const privateKeyToPublicKey: (privateKey: Bytes, isCompressed: boolean) => string;
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
export declare const encrypt: (privateKey: Bytes, password: string | Uint8Array, options?: CipherOptions) => Promise<KeyStore>;
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
export declare const privateKeyToAccount: (privateKey: Bytes, ignoreLength?: boolean) => Web3Account;
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
export declare const create: () => Web3Account;
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
export declare const decrypt: (keystore: KeyStore | string, password: string | Uint8Array, nonStrict?: boolean) => Promise<Web3Account>;
//# sourceMappingURL=account.d.ts.map