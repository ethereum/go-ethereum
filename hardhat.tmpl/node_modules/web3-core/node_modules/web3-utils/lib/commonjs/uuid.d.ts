/**
 * Generate a version 4 (random) uuid
 * https://github.com/uuidjs/uuid/blob/main/src/v4.js#L5
 * @returns - A version 4 uuid of the form xxxxxxxx-xxxx-4xxx-xxxx-xxxxxxxxxxxx
 * @example
 * ```ts
 * console.log(web3.utils.uuidV4());
 * > "1b9d6bcd-bbfd-4b2d-9b5d-ab8dfbbd4bed"
 * ```
 */
export declare const uuidV4: () => string;
