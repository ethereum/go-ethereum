import { BlockNumberOrTag, ContractInitOptions } from 'web3-types';
/**
 * @deprecated Will be removed in next release. Please use `web3-validator` package instead.
 */
export declare const isHexStrict: (hex: import("web3-validator").ValidInputTypes) => boolean;
/**
 * returns true if input is a hexstring, number or bigint
 *
 * @deprecated Will be removed in next release. Please use `web3-validator` package instead.
 */
export declare const isHex: (hex: import("web3-validator").ValidInputTypes) => boolean;
/**
 * Checks the checksum of a given address. Will also return false on non-checksum addresses.
 *
 * @deprecated Will be removed in next release. Please use `web3-validator` package instead.
 */
export declare const checkAddressCheckSum: (data: string) => boolean;
/**
 * Checks if a given string is a valid Ethereum address. It will also check the checksum, if the address has upper and lowercase letters.
 *
 * @deprecated Will be removed in next release. Please use `web3-validator` package instead.
 */
export declare const isAddress: (value: import("web3-validator").ValidInputTypes, checkChecksum?: boolean) => boolean;
/**
 * Returns true if the bloom is a valid bloom
 * https://github.com/joshstevens19/ethereum-bloom-filters/blob/fbeb47b70b46243c3963fe1c2988d7461ef17236/src/index.ts#L7
 *
 * @deprecated Will be removed in next release. Please use `web3-validator` package instead.
 */
export declare const isBloom: (bloom: import("web3-validator").ValidInputTypes) => boolean;
/**
 * Returns true if the value is part of the given bloom
 * note: false positives are possible.
 *
 * @deprecated Will be removed in next release. Please use `web3-validator` package instead.
 */
export declare const isInBloom: (bloom: string, value: string | Uint8Array) => boolean;
/**
 * Returns true if the ethereum users address is part of the given bloom note: false positives are possible.
 *
 * @deprecated Will be removed in next release. Please use `web3-validator` package instead.
 */
export declare const isUserEthereumAddressInBloom: (bloom: string, ethereumAddress: string) => boolean;
/**
 * Returns true if the contract address is part of the given bloom.
 * note: false positives are possible.
 *
 * @deprecated Will be removed in next release. Please use `web3-validator` package instead.
 */
export declare const isContractAddressInBloom: (bloom: string, contractAddress: string) => boolean;
/**
 * Checks if its a valid topic
 *
 * @deprecated Will be removed in next release. Please use `web3-validator` package instead.
 */
export declare const isTopic: (topic: string) => boolean;
/**
 * Returns true if the topic is part of the given bloom.
 * note: false positives are possible.
 *
 * @deprecated Will be removed in next release. Please use `web3-validator` package instead.
 */
export declare const isTopicInBloom: (bloom: string, topic: string) => boolean;
/**
 * Compares between block A and block B
 * @param blockA - Block number or string
 * @param blockB - Block number or string
 *
 * @returns - Returns -1 if a \< b, returns 1 if a \> b and returns 0 if a == b
 *
 * @example
 * ```ts
 * console.log(web3.utils.compareBlockNumbers('latest', 'pending'));
 * > -1
 *
 * console.log(web3.utils.compareBlockNumbers(12, 11));
 * > 1
 * ```
 */
export declare const compareBlockNumbers: (blockA: BlockNumberOrTag, blockB: BlockNumberOrTag) => 0 | 1 | -1;
export declare const isContractInitOptions: (options: unknown) => options is ContractInitOptions;
export declare const isNullish: (item: unknown) => item is undefined | null;
//# sourceMappingURL=validation.d.ts.map