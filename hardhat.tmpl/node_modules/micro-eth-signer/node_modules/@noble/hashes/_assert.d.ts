/**
 * Internal assertion helpers.
 * @module
 */
/** Asserts something is positive integer. */
declare function anumber(n: number): void;
/** Asserts something is Uint8Array. */
declare function abytes(b: Uint8Array | undefined, ...lengths: number[]): void;
/** Hash interface. */
export type Hash = {
    (data: Uint8Array): Uint8Array;
    blockLen: number;
    outputLen: number;
    create: any;
};
/** Asserts something is hash */
declare function ahash(h: Hash): void;
/** Asserts a hash instance has not been destroyed / finished */
declare function aexists(instance: any, checkFinished?: boolean): void;
/** Asserts output is properly-sized byte array */
declare function aoutput(out: any, instance: any): void;
export { anumber, abytes, ahash, aexists, aoutput };
//# sourceMappingURL=_assert.d.ts.map