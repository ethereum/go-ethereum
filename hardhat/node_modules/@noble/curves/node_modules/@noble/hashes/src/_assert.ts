/**
 * Internal assertion helpers.
 * @module
 */

/** Asserts something is positive integer. */
function anumber(n: number): void {
  if (!Number.isSafeInteger(n) || n < 0) throw new Error('positive integer expected, got ' + n);
}

/** Is number an Uint8Array? Copied from utils for perf. */
function isBytes(a: unknown): a is Uint8Array {
  return a instanceof Uint8Array || (ArrayBuffer.isView(a) && a.constructor.name === 'Uint8Array');
}

/** Asserts something is Uint8Array. */
function abytes(b: Uint8Array | undefined, ...lengths: number[]): void {
  if (!isBytes(b)) throw new Error('Uint8Array expected');
  if (lengths.length > 0 && !lengths.includes(b.length))
    throw new Error('Uint8Array expected of length ' + lengths + ', got length=' + b.length);
}

/** Hash interface. */
export type Hash = {
  (data: Uint8Array): Uint8Array;
  blockLen: number;
  outputLen: number;
  create: any;
};

/** Asserts something is hash */
function ahash(h: Hash): void {
  if (typeof h !== 'function' || typeof h.create !== 'function')
    throw new Error('Hash should be wrapped by utils.wrapConstructor');
  anumber(h.outputLen);
  anumber(h.blockLen);
}

/** Asserts a hash instance has not been destroyed / finished */
function aexists(instance: any, checkFinished = true): void {
  if (instance.destroyed) throw new Error('Hash instance has been destroyed');
  if (checkFinished && instance.finished) throw new Error('Hash#digest() has already been called');
}

/** Asserts output is properly-sized byte array */
function aoutput(out: any, instance: any): void {
  abytes(out);
  const min = instance.outputLen;
  if (out.length < min) {
    throw new Error('digestInto() expects output buffer of length at least ' + min);
  }
}

export { anumber, abytes, ahash, aexists, aoutput };
