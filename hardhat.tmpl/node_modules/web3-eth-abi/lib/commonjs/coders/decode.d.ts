import { AbiInput, HexString } from 'web3-types';
export declare function decodeParameters(abis: AbiInput[] | ReadonlyArray<AbiInput>, bytes: HexString, _loose: boolean): {
    [key: string]: unknown;
    __length__: number;
};
