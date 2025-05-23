"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.SyncingSubscription = exports.NewHeadsSubscription = exports.NewPendingTransactionsSubscription = exports.LogsSubscription = void 0;
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
/* eslint-disable-next-line max-classes-per-file */
const web3_utils_1 = require("web3-utils");
const web3_core_1 = require("web3-core");
const schemas_js_1 = require("./schemas.js");
/**
 * ## subscribe('logs')
 * Subscribes to incoming logs, filtered by the given options. If a valid numerical fromBlock options property is set, web3.js will retrieve logs beginning from this point, backfilling the response as necessary.
 *
 * You can subscribe to logs matching a given filter object, which can take the following parameters:
 * - `fromBlock`: (optional, default: 'latest') Integer block number, or `'latest'` for the last mined block or `'pending'`, `'earliest'` for not yet mined transactions.
 * - `address`: (optional) Contract address or a list of addresses from which logs should originate.
 * - `topics`: (optional) Array of 32 Bytes DATA topics. Topics are order-dependent. Each topic can also be an array of DATA with `or` options.
 *
 */
class LogsSubscription extends web3_core_1.Web3Subscription {
    _buildSubscriptionParams() {
        return ['logs', this.args];
    }
    formatSubscriptionResult(data) {
        return (0, web3_utils_1.format)(schemas_js_1.logSchema, data, super.returnFormat);
    }
}
exports.LogsSubscription = LogsSubscription;
/**
 * ## subscribe('pendingTransactions')
 * Subscribes to incoming pending transactions.
 *
 * You can subscribe to pending transactions by calling web3.eth.subscribe('pendingTransactions').
 * @example
 * ```ts
 * (await web3.eth.subscribe('pendingTransactions')).on('data', console.log);
 * ```
 */
class NewPendingTransactionsSubscription extends web3_core_1.Web3Subscription {
    // eslint-disable-next-line
    _buildSubscriptionParams() {
        return ['newPendingTransactions'];
    }
    formatSubscriptionResult(data) {
        return (0, web3_utils_1.format)({ format: 'string' }, data, super.returnFormat);
    }
}
exports.NewPendingTransactionsSubscription = NewPendingTransactionsSubscription;
/**
 * ## subscribe('newHeads') ( same as subscribe('newBlockHeaders'))
 *
 * Subscribes to incoming block headers. This can be used as timer to check for changes on the blockchain.
 *
 * The structure of a returned block header is {@link BlockHeaderOutput}:
 * @example
 * ```ts
 * (await web3.eth.subscribe('newHeads')).on( // 'newBlockHeaders' would work as well
 *  'data',
 * console.log
 * );
 * >{
 * parentHash: '0x9e746a1d906b299def98c75b06f714d62dacadd567c7515d76eeaa8c8074c738',
 * sha3Uncles: '0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347',
 * miner: '0x0000000000000000000000000000000000000000',
 * stateRoot: '0xe0f04b04861ecfa95e82a9310d6a7ef7aef8d7417f5209c182582bfb98a8e307',
 * transactionsRoot: '0x31ab4ea571a9e10d3a19aaed07d190595b1dfa34e03960c04293fec565dea536',
 * logsBloom: '0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000',
 * difficulty: 2n,
 * number: 21n,
 * gasLimit: 11738125n,
 * gasUsed: 830006n,
 * timestamp: 1678797237n,
 * extraData: '0xd883010b02846765746888676f312e32302e31856c696e757800000000000000e0a6e93cf40e2e71a72e493272210c3f43738ccc7e7d7b14ffd51833797d896c09117e8dc4fbcbc969bd21b42e5af3e276a911524038c001b2109b63b8e0352601',
 * nonce: 0n
 * }
 * ```
 */
class NewHeadsSubscription extends web3_core_1.Web3Subscription {
    // eslint-disable-next-line
    _buildSubscriptionParams() {
        return ['newHeads'];
    }
    formatSubscriptionResult(data) {
        return (0, web3_utils_1.format)(schemas_js_1.blockHeaderSchema, data, super.returnFormat);
    }
}
exports.NewHeadsSubscription = NewHeadsSubscription;
/**
 * ## subscribe('syncing')
 *
 * Subscribe to syncing events. This will return `true` when the node is syncing and when it’s finished syncing will return `false`, for the `changed` event.
 * @example
 * ```ts
 * (await web3.eth.subscribe('syncing')).on('changed', console.log);
 * > `true` // when syncing
 *
 * (await web3.eth.subscribe('syncing')).on('data', console.log);
 * > {
 *      startingBlock: 0,
 *      currentBlock: 0,
 *      highestBlock: 0,
 *      pulledStates: 0,
 *      knownStates: 0
 *   }
 * ```
 */
class SyncingSubscription extends web3_core_1.Web3Subscription {
    // eslint-disable-next-line
    _buildSubscriptionParams() {
        return ['syncing'];
    }
    _processSubscriptionResult(data) {
        if (typeof data === 'boolean') {
            this.emit('changed', data);
        }
        else {
            const mappedData = Object.fromEntries(Object.entries((data === null || data === void 0 ? void 0 : data.status) || data).map(([key, value]) => [
                key.charAt(0).toLowerCase() + key.substring(1),
                value,
            ]));
            this.emit('changed', data.syncing);
            this.emit('data', (0, web3_utils_1.format)(schemas_js_1.syncSchema, mappedData, super.returnFormat));
        }
    }
}
exports.SyncingSubscription = SyncingSubscription;
//# sourceMappingURL=web3_subscriptions.js.map