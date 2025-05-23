import { AbiParameter } from 'web3-types';
import { DecoderResult, EncoderResult } from '../types.js';
export declare function encodeBytes(param: AbiParameter, input: unknown): EncoderResult;
export declare function decodeBytes(param: AbiParameter, bytes: Uint8Array): DecoderResult<string>;
