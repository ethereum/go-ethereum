/**
 * The web3.eth.accounts contains functions to generate Ethereum accounts and sign transactions and data.
 *
 * **_NOTE:_** This package has NOT been audited and might potentially be unsafe. Take precautions to clear memory properly, store the private keys safely, and test transaction receiving and sending functionality properly before using in production!
 *
 *
 * To use this package standalone and use its methods use:
 * ```ts
 * import { create, decrypt } from 'web3-eth-accounts'; // ....
 * ```
 *
 * To use this package within the web3 object use:
 *
 * ```ts
 * import Web3 from 'web3';
 *
 * const web3 = new Web3(Web3.givenProvider || 'ws://some.local-or-remote.node:8546');
 * // now you have access to the accounts class
 * web3.eth.accounts.create();
 * ```
 */
export * from './wallet.js';
export * from './account.js';
export * from './types.js';
export * from './schemas.js';
export * from './common/index.js';
export * from './tx/index.js';
