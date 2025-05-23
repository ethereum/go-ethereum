/**
 * The `web3.eth.Contract` object makes it easy to interact with smart contracts on the Ethereum blockchain.
 * When you create a new contract object you give it the JSON interface of the respective smart contract and
 * web3 will auto convert all calls into low level ABI calls over RPC for you.
 * This allows you to interact with smart contracts as if they were JavaScript objects.
 *
 * To use it standalone:
 *
 * ```ts
 * const Contract = require('web3-eth-contract');
 *
 * // set provider for all later instances to use
 * Contract.setProvider('ws://localhost:8546');
 *
 * const contract = new Contract(jsonInterface, address);
 *
 * contract.methods.somFunc().send({from: ....})
 * .on('receipt', function(){
 *    ...
 * });
 * ```
 */
/**
 * This comment _supports3_ [Markdown](https://marked.js.org/)
 */
import { Contract } from './contract.js';
import { ContractLogsSubscription } from './contract_log_subscription.js';
/** @deprecated Use `ContractLogsSubscription` instead. */
export type LogsSubscription = ContractLogsSubscription;
export * from './encoding.js';
export * from './contract.js';
export * from './contract-deployer-method-class.js';
export * from './contract_log_subscription.js';
export * from './types.js';
export * from './utils.js';
export default Contract;
