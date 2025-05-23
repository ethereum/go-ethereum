import { Bytes, Numbers, Sha3Input, TypedObject, TypedObjectAbbreviated } from 'web3-types';
/**
 * A wrapper for ethereum-cryptography/keccak256 to allow hashing a `string` and a `bigint` in addition to `UInt8Array`
 * @param data - the input to hash
 * @returns - the Keccak-256 hash of the input
 *
 * @example
 * ```ts
 * console.log(web3.utils.keccak256Wrapper('web3.js'));
 * > 0x63667efb1961039c9bb0d6ea7a5abdd223a3aca7daa5044ad894226e1f83919a
 *
 * console.log(web3.utils.keccak256Wrapper(1));
 * > 0xc89efdaa54c0f20c7adf612882df0950f5a951637e0307cdcb4c672f298b8bc6
 *
 * console.log(web3.utils.keccak256Wrapper(0xaf12fd));
 * > 0x358640fd4719fa923525d74ab5ae80a594301aba5543e3492b052bf4598b794c
 * ```
 */
export declare const keccak256Wrapper: (data: Bytes | Numbers | string | ReadonlyArray<number>) => string;
export { keccak256Wrapper as keccak256 };
/**
 * computes the Keccak-256 hash of the input and returns a hexstring
 * @param data - the input to hash
 * @returns - the Keccak-256 hash of the input
 *
 * @example
 * ```ts
 * console.log(web3.utils.sha3('web3.js'));
 * > 0x63667efb1961039c9bb0d6ea7a5abdd223a3aca7daa5044ad894226e1f83919a
 *
 * console.log(web3.utils.sha3(''));
 * > undefined
 * ```
 */
export declare const sha3: (data: Bytes) => string | undefined;
/**
 * Will calculate the sha3 of the input but does return the hash value instead of null if for example a empty string is passed.
 * @param data - the input to hash
 * @returns - the Keccak-256 hash of the input
 *
 * @example
 * ```ts
 * conosle.log(web3.utils.sha3Raw('web3.js'));
 * > 0x63667efb1961039c9bb0d6ea7a5abdd223a3aca7daa5044ad894226e1f83919a
 *
 * console.log(web3.utils.sha3Raw(''));
 * > 0xc5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470
 * ```
 */
export declare const sha3Raw: (data: Bytes) => string;
/**
 * returns a string of the tightly packed value given based on the type
 * @param arg - the input to return the tightly packed value
 * @returns - the tightly packed value
 */
export declare const processSolidityEncodePackedArgs: (arg: Sha3Input) => string;
/**
 * Encode packed arguments to a hexstring
 */
export declare const encodePacked: (...values: Sha3Input[]) => string;
/**
 * Will tightly pack values given in the same way solidity would then hash.
 * returns a hash string, or null if input is empty
 * @param values - the input to return the tightly packed values
 * @returns - the keccack246 of the tightly packed values
 *
 * @example
 * ```ts
 * console.log(web3.utils.soliditySha3({ type: "string", value: "31323334" }));
 * > 0xf15f8da2ad27e486d632dc37d24912f634398918d6f9913a0a0ff84e388be62b
 * ```
 */
export declare const soliditySha3: (...values: Sha3Input[]) => string | undefined;
/**
 * Will tightly pack values given in the same way solidity would then hash.
 * returns a hash string, if input is empty will return `0xc5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470`
 * @param values - the input to return the tightly packed values
 * @returns - the keccack246 of the tightly packed values
 *
 * @example
 * ```ts
 * console.log(web3.utils.soliditySha3Raw({ type: "string", value: "helloworld" }))
 * > 0xfa26db7ca85ead399216e7c6316bc50ed24393c3122b582735e7f3b0f91b93f0
 * ```
 */
export declare const soliditySha3Raw: (...values: TypedObject[] | TypedObjectAbbreviated[]) => string;
/**
 * Get slot number for storage long string in contract. Basically for getStorage method
 * returns slotNumber where will data placed
 * @param mainSlotNumber - the slot number where will be stored hash of long string
 * @returns - the slot number where will be stored long string
 */
export declare const getStorageSlotNumForLongString: (mainSlotNumber: number | string) => string | undefined;
//# sourceMappingURL=hash.d.ts.map