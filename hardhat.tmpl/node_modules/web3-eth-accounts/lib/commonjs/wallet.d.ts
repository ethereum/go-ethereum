import { Web3BaseWallet, Web3BaseWalletAccount, KeyStore } from 'web3-types';
import { WebStorage } from './types.js';
/**
 * Wallet is an in memory `wallet` that can hold multiple accounts.
 * These accounts can be used when using web3.eth.sendTransaction() or web3.eth.contract.methods.contractfunction().send();
 *
 * For using Wallet functionality, install Web3 package using `npm i web3` or `yarn add web3`.
 * After that, Wallet functionality will be available as mentioned below.
 *
 * ```ts
 * import { Web3 } from 'web3';
 * const web3 = new Web3('http://127.0.0.1:7545');
 *
 * const wallet = await web3.eth.accounts.wallet.create(2);
 *
 * const signature = wallet.at(0).sign("Test Data"); // use wallet
 *
 * // fund account before sending following transaction ...
 *
 * const receipt = await web3.eth.sendTransaction({ // internally sign transaction using wallet
 *    from: wallet.at(0).address,
 *    to: "0xdAC17F958D2ee523a2206206994597C13D831ec7",
 *    value: 1
 *    //....
 * });
 * ```
 */
export declare class Wallet<T extends Web3BaseWalletAccount = Web3BaseWalletAccount> extends Web3BaseWallet<T> {
    private readonly _addressMap;
    private readonly _defaultKeyName;
    /**
     * Get the storage object of the browser
     *
     * @returns the storage
     */
    static getStorage(): WebStorage | undefined;
    /**
     * Generates one or more accounts in the wallet. If wallets already exist they will not be overridden.
     *
     * @param numberOfAccounts - Number of accounts to create. Leave empty to create an empty wallet.
     * @returns The wallet
     * ```ts
     * web3.eth.accounts.wallet.create(2)
     * > Wallet(2) [
     *   {
     *     address: '0xde38310a42B751AE57d30cFFF4a0A3c52A442fCE',
     *     privateKey: '0x6422c9d28efdcbee93c1d32a5fc6fd6fa081b985487885296cf8c9bbb5872600',
     *     signTransaction: [Function: signTransaction],
     *     sign: [Function: sign],
     *     encrypt: [Function: encrypt]
     *   },
     *   {
     *     address: '0x766BF755246d924B1d017Fdb5390f38a60166691',
     *     privateKey: '0x756530f13c0eb636ebdda655335f5dea9921e3362e2e588b0ad59e556f7751f0',
     *     signTransaction: [Function: signTransaction],
     *     sign: [Function: sign],
     *     encrypt: [Function: encrypt]
     *   },
     *   _accountProvider: {
     *     create: [Function: create],
     *     privateKeyToAccount: [Function: privateKeyToAccount],
     *     decrypt: [Function: decrypt]
     *   },
     *   _addressMap: Map(2) {
     *     '0xde38310a42b751ae57d30cfff4a0a3c52a442fce' => 0,
     *     '0x766bf755246d924b1d017fdb5390f38a60166691' => 1
     *   },
     *   _defaultKeyName: 'web3js_wallet'
     * ]
     *
     * ```
     */
    create(numberOfAccounts: number): this;
    /**
     * Adds an account using a private key or account object to the wallet.
     *
     * @param account - A private key or account object
     * @returns The wallet
     *
     * ```ts
     * web3.eth.accounts.wallet.add('0xbce9b59981303e76c4878b1a6d7b088ec6b9dd5c966b7d5f54d7a749ff683387');
     * > Wallet(1) [
     *   {
     *     address: '0x85D70633b90e03e0276B98880286D0D055685ed7',
     *     privateKey: '0xbce9b59981303e76c4878b1a6d7b088ec6b9dd5c966b7d5f54d7a749ff683387',
     *     signTransaction: [Function: signTransaction],
     *     sign: [Function: sign],
     *     encrypt: [Function: encrypt]
     *   },
     *   _accountProvider: {
     *     create: [Function: create],
     *     privateKeyToAccount: [Function: privateKeyToAccount],
     *     decrypt: [Function: decrypt]
     *   },
     *   _addressMap: Map(1) { '0x85d70633b90e03e0276b98880286d0d055685ed7' => 0 },
     *   _defaultKeyName: 'web3js_wallet'
     * ]
     * ```
     */
    add(account: T | string): this;
    /**
     * Get the account of the wallet with either the index or public address.
     *
     * @param addressOrIndex - A string of the address or number index within the wallet.
     * @returns The account object or undefined if the account doesn't exist
     */
    get(addressOrIndex: string | number): T | undefined;
    /**
     * Removes an account from the wallet.
     *
     * @param addressOrIndex - The account address, or index in the wallet.
     * @returns true if the wallet was removed. false if it couldn't be found.
     * ```ts
     * web3.eth.accounts.wallet.add('0xbce9b59981303e76c4878b1a6d7b088ec6b9dd5c966b7d5f54d7a749ff683387');
     *
     * web3.eth.accounts.wallet.remove('0x85D70633b90e03e0276B98880286D0D055685ed7');
     * > true
     * web3.eth.accounts.wallet
     * > Wallet(0) [
     * _accountProvider: {
     *   create: [Function: create],
     *   privateKeyToAccount: [Function: privateKeyToAccount],
     *   decrypt: [Function: decrypt]
     * },
     * _addressMap: Map(0) {},
     * _defaultKeyName: 'web3js_wallet'
     * ]
     * ```
     */
    remove(addressOrIndex: string | number): boolean;
    /**
     * Securely empties the wallet and removes all its accounts.
     * Use this with *caution as it will remove all accounts stored in local wallet.
     *
     * @returns The wallet object
     * ```ts
     *
     * web3.eth.accounts.wallet.clear();
     * > Wallet(0) [
     * _accountProvider: {
     *   create: [Function: create],
     *   privateKeyToAccount: [Function: privateKeyToAccount],
     *   decrypt: [Function: decrypt]
     * },
     * _addressMap: Map(0) {},
     * _defaultKeyName: 'web3js_wallet'
     * ]
     * ```
     */
    clear(): this;
    /**
     * Encrypts all wallet accounts to an array of encrypted keystore v3 objects.
     *
     * @param password - The password which will be used for encryption
     * @param options - encryption options
     * @returns An array of the encrypted keystore v3.
     *
     * ```ts
     * web3.eth.accounts.wallet.create(1)
     * web3.eth.accounts.wallet.encrypt("abc").then(console.log);
     * > [
     * '{"version":3,"id":"fa46e213-a7c3-4844-b903-dd14d39cc7db",
     * "address":"fa3e41a401609103c241431cbdee8623ae2a321a","crypto":
     * {"ciphertext":"8d179a911d6146ad2924e86bf493ed89b8ff3596ffec0816e761c542016ab13c",
     * "cipherparams":{"iv":"acc888c6cf4a19b86846cef0185a7164"},"cipher":"aes-128-ctr",
     * "kdf":"scrypt","kdfparams":{"n":8192,"r":8,"p":1,"dklen":32,"salt":"6a743c9b367d15f4758e4f3f3378ff0fd443708d1c64854e07588ea5331823ae"},
     * "mac":"410544c8307e3691fda305eb3722d82c3431f212a87daa119a21587d96698b57"}}'
     * ]
     * ```
     */
    encrypt(password: string, options?: Record<string, unknown> | undefined): Promise<KeyStore[]>;
    /**
     * Decrypts keystore v3 objects.
     *
     * @param encryptedWallets - An array of encrypted keystore v3 objects to decrypt
     * @param password - The password to encrypt with
     * @param options - decrypt options for the wallets
     * @returns The decrypted wallet object
     *
     * ```ts
     * web3.eth.accounts.wallet.decrypt([
     * { version: 3,
     * id: '83191a81-aaca-451f-b63d-0c5f3b849289',
     * address: '06f702337909c06c82b09b7a22f0a2f0855d1f68',
     * crypto:
     * { ciphertext: '7d34deae112841fba86e3e6cf08f5398dda323a8e4d29332621534e2c4069e8d',
     *   cipherparams: { iv: '497f4d26997a84d570778eae874b2333' },
     *   cipher: 'aes-128-ctr',
     *   kdf: 'scrypt',
     *   kdfparams:
     *    { dklen: 32,
     *      salt: '208dd732a27aa4803bb760228dff18515d5313fd085bbce60594a3919ae2d88d',
     *      n: 262144,
     *      r: 8,
     *      p: 1 },
     *   mac: '0062a853de302513c57bfe3108ab493733034bf3cb313326f42cf26ea2619cf9' } },
     * { version: 3,
     * id: '7d6b91fa-3611-407b-b16b-396efb28f97e',
     * address: 'b5d89661b59a9af0b34f58d19138baa2de48baaf',
     * crypto:
     * { ciphertext: 'cb9712d1982ff89f571fa5dbef447f14b7e5f142232bd2a913aac833730eeb43',
     *   cipherparams: { iv: '8cccb91cb84e435437f7282ec2ffd2db' },
     *   cipher: 'aes-128-ctr',
     *   kdf: 'scrypt',
     *   kdfparams:
     *    { dklen: 32,
     *      salt: '08ba6736363c5586434cd5b895e6fe41ea7db4785bd9b901dedce77a1514e8b8',
     *      n: 262144,
     *      r: 8,
     *      p: 1 },
     *   mac: 'd2eb068b37e2df55f56fa97a2bf4f55e072bef0dd703bfd917717d9dc54510f0' } }
     * ], 'test').then(console.log)
     * > Wallet {
     *   _accountProvider: {
     *     create: [Function: create],
     *     privateKeyToAccount: [Function: privateKeyToAccount],
     *     decrypt: [Function: decrypt]
     *   },
     *   _defaultKeyName: 'web3js_wallet',
     *   _accounts: {
     *     '0x85d70633b90e03e0276b98880286d0d055685ed7': {
     *       address: '0x85D70633b90e03e0276B98880286D0D055685ed7',
     *       privateKey: '0xbce9b59981303e76c4878b1a6d7b088ec6b9dd5c966b7d5f54d7a749ff683387',
     *       signTransaction: [Function: signTransaction],
     *       sign: [Function: sign],
     *       encrypt: [Function: encrypt]
     *     },
     *     '0x06f702337909c06c82b09b7a22f0a2f0855d1f68': {
     *       address: '0x06F702337909C06C82B09B7A22F0a2f0855d1F68',
     *       privateKey: '87a51da18900da7398b3bab03996833138f269f8f66dd1237b98df6b9ce14573',
     *       signTransaction: [Function: signTransaction],
     *       sign: [Function: sign],
     *       encrypt: [Function: encrypt]
     *     },
     *     '0xb5d89661b59a9af0b34f58d19138baa2de48baaf': {
     *       address: '0xB5d89661B59a9aF0b34f58D19138bAa2de48BAaf',
     *       privateKey: '7ee61c5282979aae9dd795bb6a54e8bdc2bfe009acb64eb9a67322eec3b3da6e',
     *       signTransaction: [Function: signTransaction],
     *       sign: [Function: sign],
     *       encrypt: [Function: encrypt]
     *     }
     *   }
     * }
     * ```
     */
    decrypt(encryptedWallets: KeyStore[], password: string, options?: Record<string, unknown> | undefined): Promise<this>;
    /**
     * Stores the wallet encrypted and as string in local storage.
     * **__NOTE:__** Browser only
     *
     * @param password - The password to encrypt the wallet
     * @param keyName - (optional) The key used for the local storage position, defaults to `"web3js_wallet"`.
     * @returns Will return boolean value true if saved properly
     * ```ts
     * web3.eth.accounts.wallet.save('test#!$');
     * >true
     * ```
     */
    save(password: string, keyName?: string): Promise<boolean>;
    /**
     * Loads a wallet from local storage and decrypts it.
     * **__NOTE:__** Browser only
     *
     * @param password - The password to decrypt the wallet.
     * @param keyName - (optional)The key used for local storage position, defaults to `web3js_wallet"`
     * @returns Returns the wallet object
     *
     * ```ts
     * web3.eth.accounts.wallet.save('test#!$');
     * > true
     * web3.eth.accounts.wallet.load('test#!$');
     * { defaultKeyName: "web3js_wallet",
     *   length: 0,
     *   _accounts: Accounts {_requestManager: RequestManager, givenProvider: Proxy, providers: {…}, _provider: WebsocketProvider, …},
     *   [[Prototype]]: Object
     * }
     * ```
     */
    load(password: string, keyName?: string): Promise<this>;
}
