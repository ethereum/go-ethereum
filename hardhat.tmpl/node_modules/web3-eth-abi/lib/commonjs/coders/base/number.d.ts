import type { AbiParameter } from 'web3-types';
import { DecoderResult, EncoderResult } from '../types.js';
export declare function encodeNumber(param: AbiParameter, input: unknown): EncoderResult;
export declare function decodeNumber(param: AbiParameter, bytes: Uint8Array): DecoderResult<bigint>;
