export interface Blake2bCTX {
    b: Uint8Array;
    h: Uint32Array;
    t: number;
    c: number;
    outlen: number;
}

/**
 * Creates a Blake2b hashing context
 * @param outlen between 1 and 64
 * @param key optional
 * @returns the hashing context
 */
export declare function blake2bInit(outlen?: number, key?: Uint8Array): Blake2bCTX;

/**
 * Updates a Blake2b streaming hash
 * @param ctx hashing context from blake2bInit()
 * @param input Byte array
 */
export declare function blake2bUpdate(ctx: Blake2bCTX, input: ArrayLike<number>): void;

/**
 * Completes a Blake2b streaming hash
 * @param ctx hashing context from blake2bInit()
 * @returns the final hash
 */
export declare function blake2bFinal(ctx: Blake2bCTX): Uint8Array;

/**
 *
 * @param input the input bytes, as a string, Buffer, or Uint8Array
 * @param key optional key Uint8Array, up to 64 bytes
 * @param outlen optional output length in bytes, defaults to 64
 * @returns an n-byte Uint8Array
 */
export declare function blake2b(input: string | Uint8Array, key?: Uint8Array, outlen?: number): Uint8Array;

/**
 * Computes the Blake2b hash of a string or byte array
 *
 * @param input the input bytes, as a string, Buffer, or Uint8Array
 * @param key optional key Uint8Array, up to 64 bytes
 * @param outlen outlen - optional output length in bytes, defaults to 64
 * @returns an n-byte hash in hex, all lowercase
 */
export declare function blake2bHex(input: string | Uint8Array, key?: Uint8Array, outlen?: number): string;

export interface Blake2sCTX {
    h: Uint32Array;
    b: Uint8Array;
    c: number;
    t: number;
    outlen: number;
}

/**
 * Creates a Blake2s hashing context
 * @param outlen between 1 and 32
 * @param key optional Uint8Array key
 * @returns the hashing context
 */
export declare function blake2sInit(outlen: number, key?: Uint8Array): Blake2sCTX;

/**
 * Updates a Blake2s streaming hash
 * @param ctx hashing context from blake2sinit()
 * @param input byte array
 */
export declare function blake2sUpdate(ctx: Blake2sCTX, input: ArrayLike<number>): void;

/**
 * Completes a Blake2s streaming hash
 * @param ctx hashing context from blake2sinit()
 * @returns Uint8Array containing the message digest
 */
export declare function blake2sFinal(ctx: Blake2sCTX): Uint8Array;

/**
 * Computes the Blake2s hash of a string or byte array, and returns a Uint8Array
 * @param input the input bytes, as a string, Buffer, or Uint8Array
 * @param key optional key Uint8Array, up to 32 bytes
 * @param outlen optional output length in bytes, defaults to 64
 * @returns an n-byte Uint8Array
 */
export declare function blake2s(input: string | Uint8Array, key?: Uint8Array, outlen?: number): Uint8Array;

/**
 *
 * @param input the input bytes, as a string, Buffer, or Uint8Array
 * @param key optional key Uint8Array, up to 32 bytes
 * @param outlen optional output length in bytes, defaults to 64
 * @returns an n-byte hash in hex, all lowercase
 */
export declare function blake2sHex(input: string | Uint8Array, key?: Uint8Array, outlen?: number): string;
