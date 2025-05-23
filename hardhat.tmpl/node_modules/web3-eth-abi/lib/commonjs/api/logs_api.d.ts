/**
 *
 *  @module ABI
 */
import { HexString, AbiParameter, DecodedParams } from 'web3-types';
/**
 * Decodes ABI-encoded log data and indexed topic data.
 * @param inputs - A {@link AbiParameter} input array. See the [Solidity documentation](https://docs.soliditylang.org/en/develop/types.html) for a list of types.
 * @param data - The ABI byte code in the `data` field of a log.
 * @param topics - An array with the index parameter topics of the log, without the topic[0] if its a non-anonymous event, otherwise with topic[0]
 * @returns - The result object containing the decoded parameters.
 *
 * @example
 * ```ts
 * let res = web3.eth.abi.decodeLog(
 *    [
 *      {
 *        type: "string",
 *        name: "myString",
 *      },
 *      {
 *        type: "uint256",
 *        name: "myNumber",
 *        indexed: true,
 *      },
 *      {
 *        type: "uint8",
 *        name: "mySmallNumber",
 *        indexed: true,
 *      },
 *    ],
 *    "0x0000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000000748656c6c6f252100000000000000000000000000000000000000000000000000",
 *    [
 *      "0x000000000000000000000000000000000000000000000000000000000000f310",
 *      "0x0000000000000000000000000000000000000000000000000000000000000010",
 *    ]
 *  );
 * > {
 *  '0': 'Hello%!',
 *  '1': 62224n,
 *  '2': 16n,
 *  __length__: 3,
 *  myString: 'Hello%!',
 *  myNumber: 62224n,
 *  mySmallNumber: 16n
 * }
 * ```
 */
export declare const decodeLog: <ReturnType extends DecodedParams>(inputs: Array<AbiParameter> | ReadonlyArray<AbiParameter>, data: HexString, topics: string | string[]) => ReturnType;
