/**
 * Returns true if the bloom is a valid bloom
 * @param bloom The bloom
 */
export declare function isBloom(bloom: string): boolean;
/**
 * Returns true if the value is part of the given bloom
 * note: false positives are possible.
 * @param bloom encoded bloom
 * @param value The value
 */
export declare function isInBloom(bloom: string, value: string | Uint8Array): boolean;
/**
 * Returns true if the ethereum users address is part of the given bloom.
 * note: false positives are possible.
 * @param bloom encoded bloom
 * @param address the address to test
 */
export declare function isUserEthereumAddressInBloom(bloom: string, ethereumAddress: string): boolean;
/**
 * Returns true if the contract address is part of the given bloom.
 * note: false positives are possible.
 * @param bloom encoded bloom
 * @param contractAddress the contract address to test
 */
export declare function isContractAddressInBloom(bloom: string, contractAddress: string): boolean;
/**
 * Returns true if the topic is part of the given bloom.
 * note: false positives are possible.
 * @param bloom encoded bloom
 * @param topic the topic encoded hex
 */
export declare function isTopicInBloom(bloom: string, topic: string): boolean;
/**
 * Checks if its a valid topic
 * @param topic encoded hex topic
 */
export declare function isTopic(topic: string): boolean;
/**
 * Is valid address
 * @param address The address
 */
export declare function isAddress(address: string): boolean;
