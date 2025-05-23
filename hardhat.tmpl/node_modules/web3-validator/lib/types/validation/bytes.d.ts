import { ValidInputTypes } from '../types.js';
/**
 * checks input if typeof data is valid Uint8Array input
 */
export declare const isUint8Array: (data: ValidInputTypes) => data is Uint8Array;
export declare const isBytes: (value: ValidInputTypes | Uint8Array | number[], options?: {
    abiType: string;
    size?: never;
} | {
    size: number;
    abiType?: never;
}) => boolean;
//# sourceMappingURL=bytes.d.ts.map