/**
 * Takes a bech32 encoded string and returns the human readable part ("prefix") and
 * a list of character positions in the bech32 alphabet ("words").
 *
 * @throws Throws on error
 */
export function decode(str: string, limit?: number): { prefix: string, words: number[] };

/**
 * Takes a bech32 encoded string and returns the human readable part ("prefix") and
 * a list of character positions in the bech32 alphabet ("words").
 *
 * @returns undefined when there was an error
 */
export function decodeUnsafe(str: string, limit?: number): ({ prefix: string, words: number[] }) | undefined;

/**
 * Takes a human readable part ("prefix") and a list of character positions in the
 * bech32 alphabet ("words") and returns a bech32 encoded string.
 */
export function encode(prefix: string, words: number[], limit?: number): string;

/**
 * Converts a list of character positions in the bech32 alphabet ("words")
 * to binary data.
 *
 * The returned data can be used to construct an Uint8Array or Buffer like this:
 *
 * ```ts
 * const a = new Uint8Array(fromWords(words));
 * const b = Buffer.from(fromWords(words));
 * ```
 *
 * @throws Throws on error
 */
export function fromWords(words: number[]): number[];

/**
 * Converts a list of character positions in the bech32 alphabet ("words")
 * to binary data.
 *
 * The returned data can be used to construct an Uint8Array or Buffer like this:
 *
 * ```ts
 * const a = new Uint8Array(fromWordsUnsafe(words));
 * const b = Buffer.from(fromWordsUnsafe(words));
 * ```
 *
 * @returns undefined when there was an error
 */
export function fromWordsUnsafe(words: number[]): number[] | undefined;

/**
 * Converts binary data to a list of character positions in the bech32 alphabet ("words").
 *
 * Uint8Arrays and Buffers can be passed as an argument directly:
 *
 * ```ts
 * const a = toWords(new Uint8Array([0x00, 0x11, 0x22]));
 * const b = toWords(Buffer.from("001122", "hex"));
 * ```
 *
 * @throws Throws on error
 */
export function toWords(bytes: ArrayLike<number>): number[];

/**
 * Converts binary data to a list of character positions in the bech32 alphabet ("words").
 *
 * Uint8Arrays and Buffers can be passed as an argument directly:
 *
 * ```ts
 * const a = toWordsUnsafe(new Uint8Array([0x00, 0x11, 0x22]));
 * const b = toWordsUnsafe(Buffer.from("001122", "hex"));
 * ```
 *
 * @returns undefined when there was an error
 */
export function toWordsUnsafe(bytes: ArrayLike<number>): number[] | undefined;
