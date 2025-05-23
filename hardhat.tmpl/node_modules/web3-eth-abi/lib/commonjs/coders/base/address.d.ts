import { AbiParameter } from 'web3-types';
import { DecoderResult, EncoderResult } from '../types.js';
export declare function encodeAddress(param: AbiParameter, input: unknown): EncoderResult;
export declare function decodeAddress(_param: AbiParameter, bytes: Uint8Array): DecoderResult<string>;
