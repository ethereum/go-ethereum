/**
 * The web3.eth.abi functions let you encode and decode parameters to ABI (Application Binary Interface) for function calls to the EVM (Ethereum Virtual Machine).
 *
 *  For using Web3 ABI functions, first install Web3 package using `npm i web3` or `yarn add web3`.
 * After that, Web3 ABI functions will be available.
 * ```ts
 * import { Web3 } from 'web3';
 *
 * const web3 = new Web3();
 * const encoded = web3.eth.abi.encodeFunctionSignature({
 *     name: 'myMethod',
 *     type: 'function',
 *     inputs: [{
 *         type: 'uint256',
 *         name: 'myNumber'
 *     },{
 *         type: 'string',
 *         name: 'myString'
 *     }]
 * });
 *
 * ```
 *
 * For using individual package install `web3-eth-abi` package using `npm i web3-eth-abi` or `yarn add web3-eth-abi` and only import required functions.
 * This is more efficient approach for building lightweight applications.
 * ```ts
 * import { encodeFunctionSignature } from 'web3-eth-abi';
 *
 * const encoded = encodeFunctionSignature({
 *     name: 'myMethod',
 *     type: 'function',
 *     inputs: [{
 *         type: 'uint256',
 *         name: 'myNumber'
 *     },{
 *         type: 'string',
 *         name: 'myString'
 *     }]
 * });
 *
 * ```
 *
 *  @module ABI
 */
import { Eip712TypedData } from 'web3-types';
/**
 * Get the EIP-191 encoded message to sign, from the typedData object. If `hash` is enabled, the message will be hashed
 * with Keccak256.
 */
export declare const getMessage: (typedData: Eip712TypedData, hash?: boolean) => string;
