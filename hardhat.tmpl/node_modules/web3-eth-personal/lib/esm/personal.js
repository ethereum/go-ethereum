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
import { Web3Context } from 'web3-core';
import * as rpcWrappers from './rpc_method_wrappers.js';
/**
 * Eth Personal allows you to interact with the Ethereum node’s accounts.
 * For using Eth Personal package, first install Web3 package using: `npm i web3` or `yarn add web3` based on your package manager.
 * ```ts
 *
 *import { Web3 } from 'web3';
 *  const web3 = new Web3('http://127.0.0.1:7545');
 *
 *  console.log(await web3.eth.personal.getAccounts());
 *
 * ```
 * For using individual package install `web3-eth-personal` packages using: `npm i web3-eth-personal` or `yarn add web3-eth-personal`.
 *
 * ```ts
 * import {Personal} from 'web3-eth-personal';
 *
 * const personal = new Personal('http://127.0.0.1:7545');
 * console.log(await personal.getAccounts());
 * ```
 */
export class Personal extends Web3Context {
    /**
     *Returns a list of accounts the node controls by using the provider and calling the RPC method personal_listAccounts. Using `web3.eth.accounts.create()` will not add accounts into this list. For that use `web3.eth.personal.newAccount()`.
     * @returns - An array of addresses controlled by the node.
     * @example
     * ```ts
     *  const accounts = await personal.getAccounts();
     * console.log(accounts);
     * >
     * [
     * 	'0x79D7BbaC53C9aF700d0B250e9AE789E503Fcd6AE',
     * 	'0xe2597eB05CF9a87eB1309e86750C903EC38E527e',
     * 	'0x7eD0e85B8E1E925600B4373e6d108F34AB38a401',
     * 	'0xE4bEEf667408b99053dC147Ed19592aDa0d77F59',
     * 	'0x7AB80aeB6bb488B7f6c41c58e83Ef248eB39c882',
     * 	'0x12B1D9d74d73b1C3A245B19C1C5501c653aF1af9',
     * 	'0x1a6075A263Ee140e00Dbf8E374Fc5A443d097894',
     * 	'0x4FEC0A51024B13030D26E70904B066C6d41157A5',
     * 	'0x03095dc4857BB26f3a4550c5651Df8b7f6b6B1Ef',
     * 	'0xac0B9b6e8A17991cb172B2ABAF45Fb5eb769E540'
     * ]
     * ```
     */
    getAccounts() {
        return __awaiter(this, void 0, void 0, function* () {
            return rpcWrappers.getAccounts(this.requestManager);
        });
    }
    /**
     * Creates a new account and returns its address.
     * **_NOTE:_**  This function sends a sensitive information like password. Never call this function over a unsecured Websocket or HTTP provider, as your password will be sent in plain text!
     * @param password - The password to encrypt the account with.
     * @returns - The address of the new account.
     * @example
     * ```ts
     * const addr = await web3.eth.personal.newAccount('password');
     * console.log(addr);
     * > '0x1234567891011121314151617181920212223456'
     * ```
     */
    newAccount(password) {
        return __awaiter(this, void 0, void 0, function* () {
            return rpcWrappers.newAccount(this.requestManager, password);
        });
    }
    /**
     * Unlocks an account for a given duration.
     * @param address - The address of the account to unlock.
     * @param password - The password of the account to unlock.
     * @param unlockDuration - The duration in seconds to unlock the account for.
     * @example
     * ```ts
     * await personal.unlockAccount(
     * 	"0x0d4aa485ecbc499c70860feb7e5aaeaf5fd8172e",
     * 	"123456",
     * 	600
     * );
     * ```
     */
    unlockAccount(address, password, unlockDuration) {
        return __awaiter(this, void 0, void 0, function* () {
            return rpcWrappers.unlockAccount(this.requestManager, address, password, unlockDuration);
        });
    }
    /**
     * Locks the given account
     * @param address - The address of the account to lock.
     * @returns - `true` if the account was locked, otherwise `false`.
     * @example
     * ```ts
     * await personal.lockAccount(
     * 	"0x0d4aa485ecbc499c70860feb7e5aaeaf5fd8172e"
     * );
     * ```
     */
    lockAccount(address) {
        return __awaiter(this, void 0, void 0, function* () {
            return rpcWrappers.lockAccount(this.requestManager, address);
        });
    }
    /**
     * Imports the given private key into the key store, encrypting it with the passphrase.
     * @param keyData - An unencrypted private key (hex string).
     * @param passphrase  - The password of the account
     * @returns - The address of the new account.
     * @example
     * ```ts
     * const accountAddress = await personal.importRawKey(
     * 	"abe40cb08850da918ee951b237fa87946499b2d8643e4aa12b0610b050c731f6",
     * 	"123456"
     * );
     *
     * console.log(unlockTx);
     * > 0x8727a8b34ec833154b72b62cac05d69f86eb6556
     * ```
     */
    importRawKey(keyData, passphrase) {
        return __awaiter(this, void 0, void 0, function* () {
            return rpcWrappers.importRawKey(this.requestManager, keyData, passphrase);
        });
    }
    /**
     * This method sends a transaction over the management API.
     * **_NOTE:_** Sending your account password over an unsecured HTTP RPC connection is highly unsecure.
     * @param tx - The transaction options
     * @param passphrase - The passphrase of the current account
     * @returns - The transaction hash
     * @example
     * ```ts
     * const txHash = personal
     * .sendTransaction({
     *  	from: "0x0d4aa485ecbc499c70860feb7e5aaeaf5fd8172e",
     * 	gasPrice: "20000000000",
     * 	gas: "21000",
     * 	to: "0x3535353535353535353535353535353535353535",
     * 	value: "1000000",
     * 	data: "",
     * 	nonce: 0,
     * },
     * "123456");
     *
     * console.log(txHash);
     * > 0x9445325c3c5638c9fe425b003b8c32f03e9f99d409555a650a6838ba712bb51b
     * ```
     */
    sendTransaction(tx, passphrase) {
        return __awaiter(this, void 0, void 0, function* () {
            return rpcWrappers.sendTransaction(this.requestManager, tx, passphrase, this.config);
        });
    }
    /**
     * Signs a transaction. This account needs to be unlocked.
     * **_NOTE:_** Sending your account password over an unsecured HTTP RPC connection is highly unsecure.
     * @param tx - The transaction data to sign. See sendTransaction  for more information.
     * @param passphrase - The password of the `from` account, to sign the transaction with.
     * @returns - The RLP encoded transaction. The `raw` property can be used to send the transaction using  sendSignedTransaction.
     * @example
     * ```ts
     * const tx = personal
     * .signTransaction({
     * 	from: "0x0d4aa485ecbc499c70860feb7e5aaeaf5fd8172e",
     * 	gasPrice: "20000000000",
     * 	gas: "21000",
     * 	to: "0x3535353535353535353535353535353535353535",
     * 	value: "1000000000000000000",
     * 	data: "",
     * 	nonce: 0,
     * },
     * "123456");
     *
     * console.log(tx);
     *
     * > {
     * 	raw: '0xf86e808504a817c800825208943535353535353535353535353535353535353535880de0b6b3a764000080820a95a0c951c03238fe930e6e69ab9d6af9f29248a514048e44884f0e60c4de40de3526a038b71399bf0c8925749ab79e91ce6cd2fc068c84c18ff6a197b48c4cbef01e00',
     * 	tx: {
     * 	type: '0x0',
     * 	nonce: '0x0',
     * 	gasPrice: '0x4a817c800',
     * 	maxPriorityFeePerGas: null,
     * 	maxFeePerGas: null,
     * 	gas: '0x5208',
     * 	value: '0xde0b6b3a7640000',
     * 	input: '0x',
     * 	v: '0xa95',
     * 	r: '0xc951c03238fe930e6e69ab9d6af9f29248a514048e44884f0e60c4de40de3526',
     * 	s: '0x38b71399bf0c8925749ab79e91ce6cd2fc068c84c18ff6a197b48c4cbef01e00',
     * 	to: '0x3535353535353535353535353535353535353535',
     * 	hash: '0x65e3df790ab2a32068b13cff970b26445b8995229ae4abbed61bd996f09fce69'
     * 	}
     * }
     * ```
     */
    signTransaction(tx, passphrase) {
        return __awaiter(this, void 0, void 0, function* () {
            return rpcWrappers.signTransaction(this.requestManager, tx, passphrase, this.config);
        });
    }
    /**
     * Calculates an Ethereum specific signature with:
     * sign(keccak256("\x19Ethereum Signed Message:\n" + dataToSign.length + dataToSign)))
     * Adding a prefix to the message makes the calculated signature recognisable as an Ethereum specific signature.
     *
     * If you have the original message and the signed message, you can discover the signing account address using web3.eth.personal.ecRecover
     * **_NOTE:_** Sending your account password over an unsecured HTTP RPC connection is highly unsecure.
     * @param data - The data to sign.
     * @param address - The address to sign with.
     * @param passphrase - The passphrase to decrypt the account with.
     * @returns - The signature.
     * @example
     * ```ts
     * const sig = await personal.sign("Hello world", "0x0D4Aa485ECbC499c70860fEb7e5AaeAf5fd8172E", "123456")
     * console.log(sig)
     * > 0x5d21d01b3198ac34d0585a9d76c4d1c8123e5e06746c8962318a1c08ffb207596e6fce4a6f377b7c0fc98c5f646cd73438c80e8a1a95cbec55a84c2889dca0301b
     * ```
     */
    sign(data, address, passphrase) {
        return __awaiter(this, void 0, void 0, function* () {
            return rpcWrappers.sign(this.requestManager, data, address, passphrase);
        });
    }
    /**
     * Recovers the account that signed the data.
     * @param signedData - Data that was signed. If String it will be converted using {@link utf8ToHex}
     * @param signature - The signature
     * @returns - The address of the account that signed the data.
     * @example
     * ```ts
     *  const address = await personal.ecRecover(
     * 	"Hello world",
     * 	"0x5d21d01b3198ac34d0585a9d76c4d1c8123e5e06746c8962318a1c08ffb207596e6fce4a6f377b7c0fc98c5f646cd73438c80e8a1a95cbec55a84c2889dca0301b"
     * );
     * console.log(address);
     * > 0x0d4aa485ecbc499c70860feb7e5aaeaf5fd8172e
     * ```
     */
    ecRecover(signedData, signature) {
        return __awaiter(this, void 0, void 0, function* () {
            return rpcWrappers.ecRecover(this.requestManager, signedData, signature);
        });
    }
}
//# sourceMappingURL=personal.js.map