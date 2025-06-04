/// <reference types="node" />
import { Decoded, Input, List } from './types';
export { Decoded, Input, List };
/**
 * RLP Encoding based on: https://github.com/ethereum/wiki/wiki/%5BEnglish%5D-RLP
 * This function takes in a data, convert it to buffer if not, and a length for recursion
 * @param input - will be converted to buffer
 * @returns returns buffer of encoded data
 **/
export declare function encode(input: Input): Buffer;
/**
 * RLP Decoding based on: {@link https://github.com/ethereum/wiki/wiki/%5BEnglish%5D-RLP|RLP}
 * @param input - will be converted to buffer
 * @param stream - Is the input a stream (false by default)
 * @returns - returns decode Array of Buffers containg the original message
 **/
export declare function decode(input: Buffer, stream?: boolean): Buffer;
export declare function decode(input: Buffer[], stream?: boolean): Buffer[];
export declare function decode(input: Input, stream?: boolean): Buffer[] | Buffer | Decoded;
/**
 * Get the length of the RLP input
 * @param input
 * @returns The length of the input or an empty Buffer if no input
 */
export declare function getLength(input: Input): Buffer | number;
