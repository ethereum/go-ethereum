function number(n: number) {
  if (!Number.isSafeInteger(n) || n < 0) throw new Error(`positive integer expected, not ${n}`);
}

function bool(b: boolean) {
  if (typeof b !== 'boolean') throw new Error(`boolean expected, not ${b}`);
}

// copied from utils
export function isBytes(a: unknown): a is Uint8Array {
  return (
    a instanceof Uint8Array ||
    (a != null && typeof a === 'object' && a.constructor.name === 'Uint8Array')
  );
}

function bytes(b: Uint8Array | undefined, ...lengths: number[]) {
  if (!isBytes(b)) throw new Error('Uint8Array expected');
  if (lengths.length > 0 && !lengths.includes(b.length))
    throw new Error(`Uint8Array expected of length ${lengths}, not of length=${b.length}`);
}

type Hash = {
  (data: Uint8Array): Uint8Array;
  blockLen: number;
  outputLen: number;
  create: any;
};
function hash(h: Hash) {
  if (typeof h !== 'function' || typeof h.create !== 'function')
    throw new Error('Hash should be wrapped by utils.wrapConstructor');
  number(h.outputLen);
  number(h.blockLen);
}

function exists(instance: any, checkFinished = true) {
  if (instance.destroyed) throw new Error('Hash instance has been destroyed');
  if (checkFinished && instance.finished) throw new Error('Hash#digest() has already been called');
}
function output(out: any, instance: any) {
  bytes(out);
  const min = instance.outputLen;
  if (out.length < min) {
    throw new Error(`digestInto() expects output buffer of length at least ${min}`);
  }
}

export { number, bool, bytes, hash, exists, output };

const assert = { number, bool, bytes, hash, exists, output };
export default assert;
