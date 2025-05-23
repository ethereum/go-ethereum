import { AbiParameter } from 'web3-types';
import { DecoderResult, EncoderResult } from '../types.js';
export declare function encodeArray(param: AbiParameter, values: unknown): EncoderResult;
export declare function decodeArray(param: AbiParameter, bytes: Uint8Array): DecoderResult<unknown[]>;
//# sourceMappingURL=array.d.ts.map