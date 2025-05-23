import { AbiParameter } from 'web3-types';
import { DecoderResult, EncoderResult } from '../types.js';
export declare function encodeTuple(param: AbiParameter, input: unknown): EncoderResult;
export declare function decodeTuple(param: AbiParameter, bytes: Uint8Array): DecoderResult<{
    [key: string]: unknown;
    __length__: number;
}>;
