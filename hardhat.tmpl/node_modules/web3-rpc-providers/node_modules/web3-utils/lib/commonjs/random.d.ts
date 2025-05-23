/**
 * Returns a random byte array by the given bytes size
 * @param size - The size of the random byte array returned
 * @returns - random byte array
 *
 * @example
 * ```ts
 * console.log(web3.utils.randomBytes(32));
 * > Uint8Array(32) [
 *       93, 172, 226,  32,  33, 176, 156, 156,
 *       182,  30, 240,   2,  69,  96, 174, 197,
 *       33, 136, 194, 241, 197, 156, 110, 111,
 *       66,  87,  17,  88,  67,  48, 245, 183
 *    ]
 * ```
 */
export declare const randomBytes: (size: number) => Uint8Array;
/**
 * Returns a random hex string by the given bytes size
 * @param byteSize - The size of the random hex string returned
 * @returns - random hex string
 *
 * ```ts
 * console.log(web3.utils.randomHex(32));
 * > 0x139f5b88b72a25eab053d3b57fe1f8a9dbc62a526b1cb1774d0d7db1c3e7ce9e
 * ```
 */
export declare const randomHex: (byteSize: number) => string;
