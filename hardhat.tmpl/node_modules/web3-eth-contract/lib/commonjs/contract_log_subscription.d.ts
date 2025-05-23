import { AbiEventFragment, HexString, Topic, DataFormat, EventLog, ContractAbiWithSignature } from 'web3-types';
import { Web3RequestManager, Web3Subscription, Web3SubscriptionManager } from 'web3-core';
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
export declare class ContractLogsSubscription extends Web3Subscription<{
    data: EventLog;
    changed: EventLog & {
        removed: true;
    };
}, {
    address?: HexString;
    topics?: (Topic | Topic[] | null)[];
    abi: AbiEventFragment;
}> {
    /**
     * Address of tye contract
     */
    readonly address?: HexString;
    /**
     * The list of topics subscribed
     */
    readonly topics?: (Topic | Topic[] | null)[];
    /**
     * The {@doclink glossary#json-interface-abi | JSON Interface} of the event.
     */
    readonly abi: AbiEventFragment & {
        signature: HexString;
    };
    readonly jsonInterface: ContractAbiWithSignature;
    constructor(args: {
        address?: HexString;
        topics?: (Topic | Topic[] | null)[];
        abi: AbiEventFragment & {
            signature: HexString;
        };
        jsonInterface: ContractAbiWithSignature;
    }, options: {
        subscriptionManager: Web3SubscriptionManager;
        returnFormat?: DataFormat;
    });
    /**
     * @deprecated This constructor overloading should not be used
     */
    constructor(args: {
        address?: HexString;
        topics?: (Topic | Topic[] | null)[];
        abi: AbiEventFragment & {
            signature: HexString;
        };
        jsonInterface: ContractAbiWithSignature;
    }, options: {
        requestManager: Web3RequestManager;
        returnFormat?: DataFormat;
    });
    protected _buildSubscriptionParams(): (string | {
        address: string | undefined;
        topics: (string | string[] | null)[] | undefined;
    })[];
    protected formatSubscriptionResult(data: EventLog): EventLog;
}
/**
 * @deprecated LogsSubscription is renamed to ContractLogsSubscription in this package to not be confused with LogsSubscription at 'web3-eth'.
 */
export declare const LogsSubscription: typeof ContractLogsSubscription;
