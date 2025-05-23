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
Object.defineProperty(exports, "__esModule", { value: true });
exports.LogsSubscription = exports.ContractLogsSubscription = void 0;
const web3_core_1 = require("web3-core");
const web3_eth_1 = require("web3-eth");
/**
 * ContractLogsSubscription to be used to subscribe to events logs.
 *
 * Following events are supported and can be accessed with either {@link ContractLogsSubscription.once} or ${@link ContractLogsSubscription.on} methods.
 *
 * - **connected**: Emitted when the subscription is connected.
 * - **data**: Fires on each incoming event with the event object as argument.
 * - **changed**: Fires on each event which was removed from the blockchain. The event will have the additional property `removed: true`.
 * - **error**: Fires on each error.
 *
 * ```ts
 * const subscription = await myContract.events.MyEvent({
 *   filter: {myIndexedParam: [20,23], myOtherIndexedParam: '0x123456789...'}, // Using an array means OR: e.g. 20 or 23
 *   fromBlock: 0
 * });
 *
 * subscription.on("connected", function(subscriptionId){
 *   console.log(subscriptionId);
 * });
 *
 * subscription.on('data', function(event){
 *   console.log(event); // same results as the optional callback above
 * });
 *
 * subscription.on('changed', function(event){
 *   // remove event from local database
 * })
 *
 * subscription.on('error', function(error, receipt) { // If the transaction was rejected by the network with a receipt, the second parameter will be the receipt.
 *   ...
 * });
 *
 * // event output example
 * > {
 *   returnValues: {
 *       myIndexedParam: 20,
 *       myOtherIndexedParam: '0x123456789...',
 *       myNonIndexParam: 'My String'
 *   },
 *   raw: {
 *       data: '0x7f9fade1c0d57a7af66ab4ead79fade1c0d57a7af66ab4ead7c2c2eb7b11a91385',
 *       topics: ['0xfd43ade1c09fade1c0d57a7af66ab4ead7c2c2eb7b11a91ffdd57a7af66ab4ead7', '0x7f9fade1c0d57a7af66ab4ead79fade1c0d57a7af66ab4ead7c2c2eb7b11a91385']
 *   },
 *   event: 'MyEvent',
 *   signature: '0xfd43ade1c09fade1c0d57a7af66ab4ead7c2c2eb7b11a91ffdd57a7af66ab4ead7',
 *   logIndex: 0,
 *   transactionIndex: 0,
 *   transactionHash: '0x7f9fade1c0d57a7af66ab4ead79fade1c0d57a7af66ab4ead7c2c2eb7b11a91385',
 *   blockHash: '0xfd43ade1c09fade1c0d57a7af66ab4ead7c2c2eb7b11a91ffdd57a7af66ab4ead7',
 *   blockNumber: 1234,
 *   address: '0xde0B295669a9FD93d5F28D9Ec85E40f4cb697BAe'
 * }
 * ```
 */
class ContractLogsSubscription extends web3_core_1.Web3Subscription {
    constructor(args, options) {
        super(args, options);
        this.address = args.address;
        this.topics = args.topics;
        this.abi = args.abi;
        this.jsonInterface = args.jsonInterface;
    }
    _buildSubscriptionParams() {
        return ['logs', { address: this.address, topics: this.topics }];
    }
    formatSubscriptionResult(data) {
        return (0, web3_eth_1.decodeEventABI)(this.abi, data, this.jsonInterface, super.returnFormat);
    }
}
exports.ContractLogsSubscription = ContractLogsSubscription;
/**
 * @deprecated LogsSubscription is renamed to ContractLogsSubscription in this package to not be confused with LogsSubscription at 'web3-eth'.
 */
exports.LogsSubscription = ContractLogsSubscription;
//# sourceMappingURL=contract_log_subscription.js.map