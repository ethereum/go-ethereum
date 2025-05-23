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

/**
 * The `web3-eth` package allows you to interact with an Ethereum blockchain and Ethereum smart contracts.
 *
 * To use this package standalone and use its methods use:
 * ```ts
 * import { Web3Context } from 'web3-core';
 * import { BlockTags } from 'web3-types';
 * import { DEFAULT_RETURN_FORMAT } from 'web3-types';
 * import { getBalance} from 'web3-eth';
 *
 * getBalance(
 *      new Web3Context('http://127.0.0.1:8545'),
 *      '0x407d73d8a49eeb85d32cf465507dd71d507100c1',
 *      BlockTags.LATEST,
 *      DEFAULT_RETURN_FORMAT
 * ).then(console.log);
 * > 1000000000000n
 * ```
 *
 * To use this package within the `web3` object use:
 * ```ts
 * import Web3 from 'web3';
 *
 * const web3 = new Web3(Web3.givenProvider || 'ws://some.local-or-remote.node:8546');
 * web3.eth.getBalance('0x407d73d8a49eeb85d32cf465507dd71d507100c1').then(console.log);
 * > 1000000000000n
 *```
 *
 * With `web3-eth` you can also subscribe (if supported by provider) to events in the Ethereum Blockchain, using the `subscribe` function. See more at the {@link Web3Eth.subscribe} function.
 */
/**
 *
 */
import 'setimmediate';

import { Web3Eth } from './web3_eth.js';

export * from './web3_eth.js';
export * from './utils/decoding.js';
export * from './schemas.js';
export * from './constants.js';
export * from './types.js';
export * from './validation.js';
export * from './rpc_method_wrappers.js';
export * from './utils/format_transaction.js';
export * from './utils/prepare_transaction_for_signing.js';
export * from './web3_subscriptions.js';
export { detectTransactionType } from './utils/detect_transaction_type.js';
export { transactionBuilder, getTransactionFromOrToAttr } from './utils/transaction_builder.js';
export { waitForTransactionReceipt } from './utils/wait_for_transaction_receipt.js';
export { trySendTransaction } from './utils/try_send_transaction.js';
export { SendTxHelper } from './utils/send_tx_helper.js';

export default Web3Eth;
