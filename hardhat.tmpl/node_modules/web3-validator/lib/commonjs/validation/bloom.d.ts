import { ValidInputTypes } from '../types.js';
/**
 * Returns true if the bloom is a valid bloom
 * https://github.com/joshstevens19/ethereum-bloom-filters/blob/fbeb47b70b46243c3963fe1c2988d7461ef17236/src/index.ts#L7
 */
export declare const isBloom: (bloom: ValidInputTypes) => boolean;
/**
 * Returns true if the value is part of the given bloom
 * note: false positives are possible.
 */
export declare const isInBloom: (bloom: string, value: string | Uint8Array) => boolean;
/**
 * Returns true if the ethereum users address is part of the given bloom note: false positives are possible.
 */
export declare const isUserEthereumAddressInBloom: (bloom: string, ethereumAddress: string) => boolean;
/**
 * Returns true if the contract address is part of the given bloom.
 * note: false positives are possible.
 */
export declare const isContractAddressInBloom: (bloom: string, contractAddress: string) => boolean;
