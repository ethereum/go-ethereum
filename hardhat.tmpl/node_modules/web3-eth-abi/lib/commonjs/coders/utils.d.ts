import { AbiParameter as ExternalAbiParameter } from 'abitype';
import { AbiInput, AbiParameter } from 'web3-types';
export declare const WORD_SIZE = 32;
export declare function alloc(size?: number): Uint8Array;
/**
 * Where possible returns a Uint8Array of the requested size that references
 * uninitialized memory. Only use if you are certain you will immediately
 * overwrite every value in the returned `Uint8Array`.
 */
export declare function allocUnsafe(size?: number): Uint8Array;
export declare function convertExternalAbiParameter(abiParam: ExternalAbiParameter): AbiParameter;
export declare function isAbiParameter(param: unknown): param is AbiParameter;
export declare function toAbiParams(abi: ReadonlyArray<AbiInput>): ReadonlyArray<AbiParameter>;
export declare function extractArrayType(param: AbiParameter): {
    size: number;
    param: AbiParameter;
};
/**
 * Param is dynamic if it's dynamic base type or if some of his children (components, array items)
 * is of dynamic type
 * @param param
 */
export declare function isDynamic(param: AbiParameter): boolean;
