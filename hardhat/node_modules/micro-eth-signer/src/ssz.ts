import { sha256 } from '@noble/hashes/sha2';
import * as P from 'micro-packed';
import { isBytes, isObject } from './utils.ts';
/*

Simple serialize (SSZ) is the serialization method used on the Beacon Chain.
SSZ is designed to be deterministic and also to Merkleize efficiently.
SSZ can be thought of as having two components:
a serialization scheme and a Merkleization scheme
that is designed to work efficiently with the serialized data structure.

- https://github.com/ethereum/consensus-specs/blob/f5277700e3b89c4d62bd4e88a559c2b938c6b0a5/ssz/simple-serialize.md
- https://github.com/ethereum/consensus-specs/blob/f5277700e3b89c4d62bd4e88a559c2b938c6b0a5/ssz/merkle-proofs.md
- https://www.ssz.dev/show

API difference:
- containers (vec/list) have arguments like (len, child).
  this is different from other SSZ library, but compatible with packed.
  there is good reason to do that: it allows create anonymous structures inside:
  `const t = SSZ.vector(10, SSZ.container({
    ...long multiline definition here...
  }))`
  if length is second argument it would look more complex and harder to read.
- bytes provided as bytes instead of hex strings (same as other libs)

*/
const BYTES_PER_CHUNK = 32; // Should be equal to digest size of hash
const EMPTY_CHUNK = new Uint8Array(BYTES_PER_CHUNK);

export const ForkSlots = {
  Phase0: 0,
  Altair: 2375680,
  Bellatrix: 4700013,
  Capella: 6209536,
  Deneb: 8626176,
} as const;

export type SSZCoder<T> = P.CoderType<T> & {
  default: T;
  info: { type: string };
  // merkleRoot calculated differently for composite types (even if they are fixed size)
  composite: boolean;
  chunkCount: number;
  // It is possible to create prover based on this, but we don't have one yet
  chunks: (value: T) => Uint8Array[];
  merkleRoot: (value: T) => Uint8Array;
  _isStableCompat: (other: SSZCoder<any>) => boolean;
};

// Utils for hashing
function chunks(data: Uint8Array): Uint8Array[] {
  const res = [];
  for (let i = 0; i < Math.ceil(data.length / BYTES_PER_CHUNK); i++) {
    const chunk = data.subarray(i * BYTES_PER_CHUNK, (i + 1) * BYTES_PER_CHUNK);
    if (chunk.length === BYTES_PER_CHUNK) res.push(chunk);
    else {
      const tmp = EMPTY_CHUNK.slice();
      tmp.set(chunk);
      res.push(tmp);
    }
  }
  return res;
}
const hash = (a: Uint8Array, b: Uint8Array): Uint8Array =>
  sha256.create().update(a).update(b).digest();
const mixInLength = (root: Uint8Array, length: number) =>
  hash(root, P.U256LE.encode(BigInt(length)));

// Will OOM without this, because tree padded to next power of two.
const zeroHashes = /* @__PURE__ */ (() => {
  const res = [EMPTY_CHUNK];
  for (let i = 0; i < 64; i++) res.push(hash(res[i], res[i]));
  return res;
})();

const merkleize = (chunks: Uint8Array[], limit?: number): Uint8Array => {
  let chunksLen = chunks.length;
  if (limit !== undefined) {
    if (limit < chunks.length) {
      throw new Error(
        `SSZ/merkleize: limit (${limit}) is less than the number of chunks (${chunks.length})`
      );
    }
    chunksLen = limit;
  }
  // log2(next power of two), we cannot use binary ops since it can be bigger than 2**32.
  const depth = Math.ceil(Math.log2(chunksLen));
  if (chunks.length == 0) return zeroHashes[depth];
  for (let l = 0; l < depth; l++) {
    const level = [];
    for (let i = 0; i < chunks.length; i += 2)
      level.push(hash(chunks[i], i + 1 < chunks.length ? chunks[i + 1] : zeroHashes[l]));
    chunks = level;
  }
  return chunks[0];
};

const checkSSZ = (o: any) => {
  if (
    typeof o !== 'object' ||
    o === null ||
    typeof o.encode !== 'function' ||
    typeof o.decode !== 'function' ||
    typeof o.merkleRoot !== 'function' ||
    typeof o.composite !== 'boolean' ||
    typeof o.chunkCount !== 'number'
  ) {
    throw new Error(`SSZ: wrong element: ${o} (${typeof o})`);
  }
};

// TODO: improve
const isStableCompat = <T>(a: SSZCoder<T>, b: SSZCoder<any>): boolean => {
  if (a === b) return true; // fast path
  const _a = a as any;
  const _b = b as any;
  if (_a.info && _b.info) {
    const aI = _a.info;
    const bI = _b.info;
    // Bitlist[N] / Bitvector[N] field types are compatible if they share the same capacity N.
    const bitTypes = ['bitList', 'bitVector'];
    if (bitTypes.includes(aI.type) && bitTypes.includes(bI.type) && aI.N === bI.N) return true;
    // List[T, N] / Vector[T, N] field types are compatible if T is compatible and if they also share the same capacity N.
    const listTypes = ['list', 'vector'];
    if (
      listTypes.includes(aI.type) &&
      listTypes.includes(bI.type) &&
      aI.N === bI.N &&
      aI.inner._isStableCompat(bI.inner)
    ) {
      return true;
    }
    // Container / StableContainer[N] field types are compatible if all inner field types are compatible,
    // if they also share the same field names in the same order, and for StableContainer[N] if they also
    // share the same capacity N.
    const contType = ['container', 'stableContainer'];
    if (contType.includes(aI.type) && contType.includes(bI.type)) {
      // both stable containers, but different capacity
      if (aI.N !== undefined && bI.N !== undefined && aI.N !== bI.N) return false;
      const kA = Object.keys(aI.fields);
      const kB = Object.keys(bI.fields);
      if (kA.length !== kB.length) return false;
      for (let i = 0; i < kA.length; i++) {
        const fA = kA[i];
        const fB = kB[i];
        if (fA !== fB) return false;
        if (!aI.fields[fA]._isStableCompat(bI.fields[fA])) return false;
      }
      return true;
    }
    // Profile[X] field types are compatible with StableContainer types compatible with X, and
    // are compatible with Profile[Y] where Y is compatible with X if also all inner field types
    // are compatible. Differences solely in optionality do not affect merkleization compatibility.
    if (aI.type === 'profile' || bI.type === 'profile') {
      //console.log('PROF PROF?', aI.type, bI.type, aI.container._isStableCompat(bI));
      if (aI.type === 'profile' && bI.type === 'stableContainer')
        return aI.container._isStableCompat(b);
      if (aI.type === 'stableContainer' && bI.type === 'profile')
        return a._isStableCompat(bI.container);
      if (aI.type === 'profile' && bI.type === 'profile')
        return aI.container._isStableCompat(bI.container);
    }
  }
  return false;
};

const basic = <T>(type: string, inner: P.CoderType<T>, def: T): SSZCoder<T> => ({
  ...inner,
  default: def,
  chunkCount: 1,
  composite: false,
  info: { type },
  _isStableCompat(other) {
    return isStableCompat(this, other);
  },
  chunks(value: T) {
    return [this.merkleRoot(value)];
  },
  merkleRoot: (value: T) => {
    const res = new Uint8Array(32);
    res.set(inner.encode(value));
    return res;
  },
});

const int = (len: number, small = true) =>
  P.apply(P.bigint(len, true), {
    encode: (from) => {
      if (!small) return from;
      if (BigInt(Number(from)) !== BigInt(from))
        throw new Error('ssz int: small integer is too big');
      return Number(from);
    },
    decode: (to: bigint | number) => {
      if (typeof to === 'bigint') return to;
      if (typeof to !== 'number' || !Number.isSafeInteger(to))
        throw new Error(`wrong type=${typeof to} expected number`);
      return BigInt(to);
    },
  });

const _0n = BigInt(0);
export const uint8 = basic('uint8', int(1), 0);
export const uint16 = basic('uint16', int(2), 0);
export const uint32 = basic('uint32', int(4), 0);
export const uint64 = basic('uint64', int(8, false), _0n);
export const uint128 = basic('uint128', int(16, false), _0n);
export const uint256 = basic('uint256', int(32, false), _0n);
export const boolean = basic('boolean', P.bool, false);

const array = <T>(len: P.Length, inner: SSZCoder<T>): P.CoderType<T[]> => {
  checkSSZ(inner);
  let arr = P.array(len, inner);
  // variable size arrays
  if (inner.size === undefined) {
    arr = P.wrap({
      encodeStream: P.array(len, P.pointer(P.U32LE, inner)).encodeStream,
      decodeStream: (r) => {
        const res: T[] = [];
        if (!r.leftBytes) return res;
        const first = P.U32LE.decodeStream(r);
        const len = (first - r.pos) / P.U32LE.size!;
        if (!Number.isSafeInteger(len)) throw r.err('SSZ/array: wrong fixed size length');
        const rest = P.array(len, P.U32LE).decodeStream(r);
        const offsets = [first, ...rest];
        // SSZ decoding requires very specific encoding and should throw on data constructed differently.
        // There is also ZST problem here (as in ETH ABI), but it is impossible to exploit since
        // definitions are hardcoded. Also, pointers very strict here.
        for (let i = 0; i < offsets.length; i++) {
          const pos = offsets[i];
          const next = i + 1 < offsets.length ? offsets[i + 1] : r.totalBytes;
          if (next < pos) throw r.err('SSZ/array: decreasing offset');
          const len = next - pos;
          if (r.pos !== pos) throw r.err('SSZ/array: wrong offset');
          res.push(inner.decode(r.bytes(len)));
        }
        return res;
      },
    });
  }
  return arr;
};

type VectorType<T> = SSZCoder<T[]> & { info: { type: 'vector'; N: number; inner: SSZCoder<T> } };
/**
 * Vector: fixed size ('len') array of elements 'inner'
 */
export const vector = <T>(len: number, inner: SSZCoder<T>): VectorType<T> => {
  if (!Number.isSafeInteger(len) || len <= 0)
    throw new Error(`SSZ/vector: wrong length=${len} (should be positive integer)`);
  return {
    ...array(len, inner),
    info: { type: 'vector', N: len, inner },
    _isStableCompat(other) {
      return isStableCompat(this, other);
    },
    default: new Array(len).fill(inner.default),
    composite: true,
    chunkCount: inner.composite ? Math.ceil((len * inner.size!) / 32) : len,
    chunks(value) {
      if (!inner.composite) return chunks(this.encode(value));
      return value.map((i) => inner.merkleRoot(i));
    },
    merkleRoot(value) {
      return merkleize(this.chunks(value));
    },
  };
};
type ListType<T> = SSZCoder<T[]> & { info: { type: 'list'; N: number; inner: SSZCoder<T> } };
/**
 * List: dynamic array of 'inner' elements with length limit maxLen
 */
export const list = <T>(maxLen: number, inner: SSZCoder<T>): ListType<T> => {
  checkSSZ(inner);
  const coder = P.validate(array(null, inner), (value) => {
    if (!Array.isArray(value) || value.length > maxLen)
      throw new Error(`SSZ/list: wrong value=${value} (len=${value.length} maxLen=${maxLen})`);
    return value;
  });
  return {
    ...coder,
    info: { type: 'list', N: maxLen, inner },
    _isStableCompat(other) {
      return isStableCompat(this, other);
    },
    composite: true,
    chunkCount: !inner.composite ? Math.ceil((maxLen * inner.size!) / BYTES_PER_CHUNK) : maxLen,
    default: [],
    chunks(value) {
      if (inner.composite) return value.map((i) => inner.merkleRoot(i));
      return chunks(this.encode(value));
    },
    merkleRoot(value) {
      return mixInLength(merkleize(this.chunks(value), this.chunkCount), value.length);
    },
  };
};

const wrapPointer = <T>(p: P.CoderType<T>) => (p.size === undefined ? P.pointer(P.U32LE, p) : p);
const wrapRawPointer = <T>(p: P.CoderType<T>) => (p.size === undefined ? P.U32LE : p);

// TODO: improve, unclear how
const fixOffsets = (
  r: P.Reader,
  fields: Record<string, P.CoderType<any>>,
  offsetFields: string[],
  obj: Record<string, any>,
  offset: number
) => {
  const offsets = [];
  for (const f of offsetFields) offsets.push(obj[f] + offset);
  for (let i = 0; i < offsets.length; i++) {
    // TODO: how to merge this with array?
    const name = offsetFields[i];
    const pos = offsets[i];
    const next = i + 1 < offsets.length ? offsets[i + 1] : r.totalBytes;
    if (next < pos) throw r.err('SSZ/container: decreasing offset');
    const len = next - pos;
    if (r.pos !== pos) throw r.err('SSZ/container: wrong offset');
    obj[name] = fields[name].decode(r.bytes(len));
  }
  return obj;
};

type ContainerCoder<T extends Record<string, SSZCoder<any>>> = SSZCoder<{
  [K in keyof T]: P.UnwrapCoder<T[K]>;
}> & { info: { type: 'container'; fields: T } };

/**
 * Container: Encodes object with multiple fields. P.struct for SSZ.
 */
export const container = <T extends Record<string, SSZCoder<any>>>(
  fields: T
): ContainerCoder<T> => {
  if (!Object.keys(fields).length) throw new Error('SSZ/container: no fields');
  const ptrCoder = P.struct(
    Object.fromEntries(Object.entries(fields).map(([k, v]) => [k, wrapPointer(v)]))
  ) as ContainerCoder<T>;
  const fixedCoder = P.struct(
    Object.fromEntries(Object.entries(fields).map(([k, v]) => [k, wrapRawPointer(v)]))
  );
  const offsetFields = Object.keys(fields).filter((i) => fields[i].size === undefined);
  const coder = P.wrap({
    encodeStream: ptrCoder.encodeStream,
    decodeStream: (r) => fixOffsets(r, fields, offsetFields, fixedCoder.decodeStream(r), 0) as any,
  }) as ContainerCoder<T>;
  return {
    ...coder,
    info: { type: 'container', fields },
    _isStableCompat(other) {
      return isStableCompat(this, other);
    },
    size: offsetFields.length ? undefined : fixedCoder.size, // structure is fixed size if all fields is fixed size
    default: Object.fromEntries(Object.entries(fields).map(([k, v]) => [k, v.default])) as {
      [K in keyof T]: P.UnwrapCoder<T[K]>;
    },
    composite: true,
    chunkCount: Object.keys(fields).length,
    chunks(value: T) {
      return Object.entries(fields).map(([k, v]) => v.merkleRoot(value[k]));
    },
    merkleRoot(value: T) {
      return merkleize(this.chunks(value as any));
    },
  };
};

// Like 'P.bits', but different direction
const bitsCoder = (len: number): P.Coder<Uint8Array, boolean[]> => ({
  encode: (data: Uint8Array): boolean[] => {
    const res: boolean[] = [];
    for (const byte of data) for (let i = 0; i < 8; i++) res.push(!!(byte & (1 << i)));
    for (let i = len; i < res.length; i++) {
      if (res[i]) throw new Error('SSZ/bitsCoder/encode: non-zero padding');
    }
    return res.slice(0, len);
  },
  decode: (data: boolean[]): Uint8Array => {
    const res = new Uint8Array(Math.ceil(len / 8));
    for (let i = 0; i < data.length; i++) if (data[i]) res[Math.floor(i / 8)] |= 1 << i % 8;
    return res;
  },
});
type BitVectorType = SSZCoder<boolean[]> & { info: { type: 'bitVector'; N: number } };
/**
 * BitVector: array of booleans with fixed size
 */
export const bitvector = (len: number): BitVectorType => {
  if (!Number.isSafeInteger(len) || len <= 0)
    throw new Error(`SSZ/bitVector: wrong length=${len} (should be positive integer)`);
  const bytesLen = Math.ceil(len / 8);
  const coder = P.apply(P.bytes(bytesLen), bitsCoder(len));
  return {
    ...coder,
    info: { type: 'bitVector', N: len },
    _isStableCompat(other) {
      return isStableCompat(this, other);
    },
    default: new Array(len).fill(false),
    composite: true,
    chunkCount: Math.ceil(len / 256),
    chunks(value) {
      return chunks(this.encode(value));
    },
    merkleRoot(value: boolean[]) {
      return merkleize(this.chunks(value), this.chunkCount);
    },
  };
};
type BitListType = SSZCoder<boolean[]> & { info: { type: 'bitList'; N: number } };
/**
 * BitList: array of booleans with dynamic size (but maxLen limit)
 */
export const bitlist = (maxLen: number): BitListType => {
  if (!Number.isSafeInteger(maxLen) || maxLen <= 0)
    throw new Error(`SSZ/bitList: wrong max length=${maxLen} (should be positive integer)`);
  let coder: P.CoderType<boolean[]> = P.wrap({
    encodeStream: (w, value) => {
      w.bytes(bitsCoder(value.length + 1).decode([...value, true])); // last true bit is terminator
    },
    decodeStream: (r) => {
      const bytes = r.bytes(r.leftBytes); // use everything
      if (!bytes.length || bytes[bytes.length - 1] === 0)
        throw new Error('SSZ/bitlist: empty trailing byte');
      const bits = bitsCoder(bytes.length * 8).encode(bytes);
      const terminator = bits.lastIndexOf(true);
      if (terminator === -1) throw new Error('SSZ/bitList: no terminator');
      return bits.slice(0, terminator);
    },
  });
  coder = P.validate(coder, (value) => {
    if (!Array.isArray(value) || value.length > maxLen)
      throw new Error(`SSZ/bitList/encode: wrong value=${value} (${typeof value})`);
    return value;
  });
  return {
    ...coder,
    info: { type: 'bitList', N: maxLen },
    _isStableCompat(other) {
      return isStableCompat(this, other);
    },
    size: undefined,
    default: [],
    chunkCount: Math.ceil(maxLen / 256),
    composite: true,
    chunks(value) {
      const data = value.length ? bitvector(value.length).encode(value) : EMPTY_CHUNK;
      return chunks(data);
    },
    merkleRoot(value: boolean[]) {
      return mixInLength(merkleize(this.chunks(value), this.chunkCount), value.length);
    },
  };
};

/**
 * Union type (None is null)
 * */
export const union = (
  ...types: (SSZCoder<any> | null)[]
): SSZCoder<{ selector: number; value: any }> => {
  if (types.length < 1 || types.length >= 128)
    throw Error('SSZ/union: should have [1...128) types');
  if (types[0] === null && types.length < 2)
    throw new Error('SSZ/union: should have at least 2 types if first is null');
  for (let i = 0; i < types.length; i++) {
    if (i > 0 && types[i] === null) throw new Error('SSZ/union: only first type can be null');
    if (types[i] !== null) checkSSZ(types[i]);
  }
  const coder = P.apply(
    P.tag(
      P.U8,
      Object.fromEntries(
        types.map((t, i) => [i, t === null ? P.magicBytes(P.EMPTY) : P.prefix(null, t)]) as any
      )
    ),
    {
      encode: ({ TAG, data }) => ({ selector: TAG, value: data }),
      decode: ({ selector, value }) => ({ TAG: selector, data: value }),
    }
  );
  return {
    ...(coder as any),
    size: undefined, // union is always variable size
    chunkCount: NaN,
    default: { selector: 0, value: types[0] === null ? null : types[0].default },
    _isStableCompat(other) {
      return isStableCompat(this, other);
    },
    composite: true,
    chunks({ selector, value }) {
      const type = types[selector];
      if (type === null) return EMPTY_CHUNK;
      return [types[selector]!.merkleRoot(value)];
    },
    merkleRoot: ({ selector, value }) => {
      const type = types[selector];
      if (type === null) return mixInLength(EMPTY_CHUNK, 0);
      return mixInLength(types[selector]!.merkleRoot(value), selector);
    },
  };
};
type ByteListType = SSZCoder<Uint8Array> & {
  info: { type: 'list'; N: number; inner: typeof byte };
};
/**
 * ByteList: same as List(len, SSZ.byte), but returns Uint8Array
 */
export const bytelist = (maxLen: number): ByteListType => {
  const coder = P.validate(P.bytes(null), (value) => {
    if (!isBytes(value) || value.length > maxLen)
      throw new Error(`SSZ/bytelist: wrong value=${value}`);
    return value;
  });
  return {
    ...coder,
    info: { type: 'list', N: maxLen, inner: byte },
    _isStableCompat(other) {
      return isStableCompat(this, other);
    },
    default: new Uint8Array([]),
    composite: true,
    chunkCount: Math.ceil(maxLen / 32),
    chunks(value) {
      return chunks(this.encode(value));
    },
    merkleRoot(value) {
      return mixInLength(merkleize(this.chunks(value), this.chunkCount), value.length);
    },
  };
};
type ByteVectorType = SSZCoder<Uint8Array> & {
  info: { type: 'vector'; N: number; inner: typeof byte };
};
/**
 * ByteVector: same as Vector(len, SSZ.byte), but returns Uint8Array
 */
export const bytevector = (len: number): ByteVectorType => {
  if (!Number.isSafeInteger(len) || len <= 0)
    throw new Error(`SSZ/vector: wrong length=${len} (should be positive integer)`);
  return {
    ...P.bytes(len),
    info: { type: 'vector', N: len, inner: byte },
    _isStableCompat(other) {
      return isStableCompat(this, other);
    },
    default: new Uint8Array(len),
    composite: true,
    chunkCount: Math.ceil(len / 32),
    chunks(value) {
      return chunks(this.encode(value));
    },
    merkleRoot(value) {
      return merkleize(this.chunks(value));
    },
  };
};

type StableContainerCoder<T extends Record<string, SSZCoder<any>>> = SSZCoder<{
  [K in keyof T]?: P.UnwrapCoder<T[K]>;
}> & { info: { type: 'stableContainer'; N: number; fields: T } };
/**
 * Same as container, but all values are optional using bitvector as prefix which indicates active fields
 */
export const stableContainer = <T extends Record<string, SSZCoder<any>>>(
  N: number,
  fields: T
): StableContainerCoder<T> => {
  const fieldsNames = Object.keys(fields) as (keyof T)[];
  const fieldsLen = fieldsNames.length;
  if (!fieldsLen) throw new Error('SSZ/stableContainer: no fields');
  if (fieldsLen > N) throw new Error('SSZ/stableContainer: more fields than N');
  const bv = bitvector(N);
  const coder = P.wrap({
    encodeStream: (w, value) => {
      const bsVal = new Array(N).fill(false);
      for (let i = 0; i < fieldsLen; i++) if (value[fieldsNames[i]] !== undefined) bsVal[i] = true;
      bv.encodeStream(w, bsVal);
      const activeFields = fieldsNames.filter((_, i) => bsVal[i]);
      const ptrCoder = P.struct(
        Object.fromEntries(activeFields.map((k) => [k, wrapPointer(fields[k])]))
      ) as StableContainerCoder<T>;
      w.bytes(ptrCoder.encode(value));
    },
    decodeStream: (r) => {
      const bsVal = bv.decodeStream(r);
      for (let i = fieldsLen; i < bsVal.length; i++) {
        if (bsVal[i] !== false) throw new Error('stableContainer: non-zero padding');
      }
      const activeFields = fieldsNames.filter((_, i) => bsVal[i]);
      const fixedCoder = P.struct(
        Object.fromEntries(activeFields.map((k) => [k, wrapRawPointer(fields[k])]))
      );
      const offsetFields = activeFields.filter((i) => fields[i].size === undefined);
      return fixOffsets(r, fields, offsetFields as any, fixedCoder.decodeStream(r), bv.size!);
    },
  }) as StableContainerCoder<T>;
  return {
    ...coder,
    info: { type: 'stableContainer', N, fields },
    size: undefined,
    default: Object.fromEntries(Object.entries(fields).map(([k, _v]) => [k, undefined])) as {
      [K in keyof T]: P.UnwrapCoder<T[K]>;
    },
    _isStableCompat(other) {
      return isStableCompat(this, other);
    },
    composite: true,
    chunkCount: N,
    chunks(value: P.UnwrapCoder<StableContainerCoder<T>>) {
      const res = Object.entries(fields).map(([k, v]) =>
        value[k] === undefined ? new Uint8Array(32) : v.merkleRoot(value[k])
      );
      while (res.length < N) res.push(new Uint8Array(32));
      return res;
    },
    merkleRoot(value: P.UnwrapCoder<StableContainerCoder<T>>) {
      const bsVal = new Array(N).fill(false);
      for (let i = 0; i < fieldsLen; i++) if (value[fieldsNames[i]] !== undefined) bsVal[i] = true;
      return hash(merkleize(this.chunks(value as any)), bv.merkleRoot(bsVal));
    },
  };
};

type ProfileCoder<
  T extends Record<string, SSZCoder<any>>,
  OptK extends keyof T & string,
  ReqK extends keyof T & string,
> = SSZCoder<{ [K in ReqK]: P.UnwrapCoder<T[K]> } & { [K in OptK]?: P.UnwrapCoder<T[K]> }> & {
  info: { type: 'profile'; container: StableContainerCoder<T> };
};

/**
 * Profile - fixed subset of stableContainer.
 * - fields and order of fields is exactly same as in underlying container
 * - some fields may be excluded or required in profile (all fields in stable container are always optional)
 * - adding new fields to underlying container won't change profile's constructed on top of it,
 *   because it is required to provide all list of optional fields.
 * - type of field can be changed inside profile (but we should be very explicit about this) to same shape type.
 *
 * @example
 * // class Shape(StableContainer[4]):
 * //     side: Optional[uint16]
 * //     color: Optional[uint8]
 * //     radius: Optional[uint16]
 *
 * // class Square(Profile[Shape]):
 * //     side: uint16
 * //     color: uint8
 *
 * // class Circle(Profile[Shape]):
 * //     color: uint8
 * //     radius: Optional[uint16]
 * // ->
 * const Shape = SSZ.stableContainer(4, {
 *   side: SSZ.uint16,
 *   color: SSZ.uint8,
 *   radius: SSZ.uint16,
 * });
 * const Square = profile(Shape, [], ['side', 'color']);
 * const Circle = profile(Shape, ['radius'], ['color']);
 * const Circle2 = profile(Shape, ['radius'], ['color'], { color: SSZ.byte });
 */
export const profile = <
  T extends Record<string, SSZCoder<any>>,
  OptK extends keyof T & string,
  ReqK extends keyof T & string,
>(
  c: StableContainerCoder<T>,
  optFields: OptK[],
  requiredFields: ReqK[] = [],
  replaceType: Record<string, any> = {}
): ProfileCoder<T, OptK, ReqK> => {
  checkSSZ(c);
  if (c.info.type !== 'stableContainer') throw new Error('profile: expected stableContainer');
  const containerFields: Set<string> = new Set(Object.keys(c.info.fields));
  if (!Array.isArray(optFields)) throw new Error('profile: optional fields should be array');
  const optFS: Set<string> = new Set(optFields);
  for (const f of optFS) {
    if (!containerFields.has(f)) throw new Error(`profile: unexpected optional field ${f}`);
  }
  if (!Array.isArray(requiredFields)) throw new Error('profile: required fields should be array');
  const reqFS: Set<string> = new Set(requiredFields);
  for (const f of reqFS) {
    if (!containerFields.has(f)) throw new Error(`profile: unexpected required field ${f}`);
    if (optFS.has(f as any as OptK))
      throw new Error(`profile: field ${f} is declared both as optional and required`);
  }
  if (!isObject(replaceType)) throw new Error('profile: replaceType should be object');
  for (const k in replaceType) {
    if (!containerFields.has(k)) throw new Error(`profile/replaceType: unexpected field ${k}`);
    if (!replaceType[k]._isStableCompat(c.info.fields[k]))
      throw new Error(`profile/replaceType: incompatible field ${k}`);
  }
  // Order should be same
  const allFields = Object.keys(c.info.fields).filter((i) => optFS.has(i) || reqFS.has(i));
  // bv is omitted if all fields are required!
  const fieldCoders = { ...c.info.fields, ...replaceType };
  let coder: ProfileCoder<T, OptK, ReqK>;
  if (optFS.size === 0) {
    // All fields are required, it is just container, possible with size
    coder = container(
      Object.fromEntries(allFields.map((k) => [k, fieldCoders[k]]))
    ) as any as ProfileCoder<T, OptK, ReqK>;
  } else {
    // NOTE: we cannot merge this with stable container,
    // because some fields are active and some is not (based on required/non-required)
    const bv = bitvector(optFS.size);
    const forFields = (fn: (f: string, optPos: number | undefined) => void) => {
      let optPos = 0;
      for (const f of allFields) {
        const isOpt = optFS.has(f);
        fn(f, isOpt ? optPos : undefined);
        if (isOpt) optPos++;
      }
    };
    coder = {
      ...P.wrap({
        encodeStream: (w, value) => {
          const bsVal = new Array(optFS.size).fill(false);
          const ptrCoder: any = {};
          forFields((f, optPos) => {
            const val = (value as any)[f];
            if (optPos !== undefined && val !== undefined) bsVal[optPos] = true;
            if (optPos === undefined && val === undefined)
              throw new Error(`profile.encode: empty required field ${f}`);
            if (val !== undefined) ptrCoder[f] = wrapPointer(fieldCoders[f]);
          });
          bv.encodeStream(w, bsVal);
          w.bytes(P.struct(ptrCoder).encode(value));
        },
        decodeStream: (r) => {
          let bsVal = bv.decodeStream(r);
          const fixedCoder: any = {};
          const offsetFields: string[] = [];
          forFields((f, optPos) => {
            if (optPos !== undefined && bsVal[optPos] === false) return;
            if (fieldCoders[f].size === undefined) offsetFields.push(f);
            fixedCoder[f] = wrapRawPointer(fieldCoders[f]);
          });
          return fixOffsets(
            r,
            fieldCoders,
            offsetFields,
            P.struct(fixedCoder).decodeStream(r),
            bv.size!
          ) as any;
        },
      }),
      size: undefined,
    } as ProfileCoder<T, OptK, ReqK>;
  }
  return {
    ...coder,
    info: { type: 'profile', container: c },
    default: Object.fromEntries(Array.from(reqFS).map((f) => [f, fieldCoders[f].default])) as {
      [K in ReqK]: P.UnwrapCoder<T[K]>;
    } & { [K in OptK]?: P.UnwrapCoder<T[K]> },
    _isStableCompat(other) {
      return isStableCompat(this, other);
    },
    composite: true,
    chunkCount: c.info.N,
    chunks(value: P.UnwrapCoder<ProfileCoder<T, OptK, ReqK>>) {
      return c.chunks(value);
    },
    merkleRoot(value: P.UnwrapCoder<ProfileCoder<T, OptK, ReqK>>) {
      return c.merkleRoot(value);
    },
  };
};

// Aliases
export const byte = uint8;
export const bit = boolean;
export const bool = boolean;
export const bytes = bytevector;

// TODO: this required for tests, but can be useful for other ETH related stuff.
// Also, blobs here. Since lib is pretty small (thanks to packed), why not?
// Deneb (last eth2 fork) types:
const MAX_VALIDATORS_PER_COMMITTEE = 2048;
const MAX_PROPOSER_SLASHINGS = 16;
const MAX_ATTESTER_SLASHINGS = 2;
const MAX_ATTESTATIONS = 128;
const MAX_DEPOSITS = 16;
const MAX_VOLUNTARY_EXITS = 16;
const MAX_TRANSACTIONS_PER_PAYLOAD = 1048576;
const BYTES_PER_LOGS_BLOOM = 256;
const MAX_EXTRA_DATA_BYTES = 32;
const DEPOSIT_CONTRACT_TREE_DEPTH = 2 ** 5;
const SYNC_COMMITTEE_SIZE = 512;
const MAX_BYTES_PER_TRANSACTION = 1073741824;
const MAX_BLS_TO_EXECUTION_CHANGES = 16;
const MAX_WITHDRAWALS_PER_PAYLOAD = 16;
const MAX_BLOB_COMMITMENTS_PER_BLOCK = 4096;
const SLOTS_PER_HISTORICAL_ROOT = 8192;
const HISTORICAL_ROOTS_LIMIT = 16777216;
const SLOTS_PER_EPOCH = 32;
const EPOCHS_PER_ETH1_VOTING_PERIOD = 64;
const VALIDATOR_REGISTRY_LIMIT = 1099511627776;
const EPOCHS_PER_HISTORICAL_VECTOR = 65536;
const EPOCHS_PER_SLASHINGS_VECTOR = 8192;
const JUSTIFICATION_BITS_LENGTH = 4;
const BYTES_PER_FIELD_ELEMENT = 32;
const FIELD_ELEMENTS_PER_BLOB = 4096;
const KZG_COMMITMENT_INCLUSION_PROOF_DEPTH = 17;
const SYNC_COMMITTEE_SUBNET_COUNT = 4;
const NEXT_SYNC_COMMITTEE_DEPTH = 5;
const BLOCK_BODY_EXECUTION_PAYLOAD_DEPTH = 4;
const FINALIZED_ROOT_DEPTH = 6;
// Electra
const MAX_COMMITTEES_PER_SLOT = 64;
const PENDING_PARTIAL_WITHDRAWALS_LIMIT = 134217728;
const PENDING_BALANCE_DEPOSITS_LIMIT = 134217728;
const PENDING_CONSOLIDATIONS_LIMIT = 262144;
const MAX_ATTESTER_SLASHINGS_ELECTRA = 1;
const MAX_ATTESTATIONS_ELECTRA = 8;
const MAX_DEPOSIT_REQUESTS_PER_PAYLOAD = 8192;
const MAX_WITHDRAWAL_REQUESTS_PER_PAYLOAD = 16;
const MAX_CONSOLIDATION_REQUESTS_PER_PAYLOAD = 1;

// We can reduce size if we inline these. But updates for new forks would be hard.
const Slot = uint64;
const Epoch = uint64;
const CommitteeIndex = uint64;
const ValidatorIndex = uint64;
const WithdrawalIndex = uint64;
const BlobIndex = uint64;
const Gwei = uint64;
const Root = bytevector(32);
const Hash32 = bytevector(32);
const Bytes32 = bytevector(32);
const Version = bytevector(4);
const DomainType = bytevector(4);
const ForkDigest = bytevector(4);
const Domain = bytevector(32);
const BLSPubkey = bytevector(48);
const KZGCommitment = bytevector(48);
const KZGProof = bytevector(48);
const BLSSignature = bytevector(96);
const Ether = uint64;
const ParticipationFlags = uint8;
const ExecutionAddress = bytevector(20);
const PayloadId = bytevector(8);
const Transaction = bytelist(MAX_BYTES_PER_TRANSACTION);
const Blob = bytevector(BYTES_PER_FIELD_ELEMENT * FIELD_ELEMENTS_PER_BLOB);

const Checkpoint = container({ epoch: Epoch, root: Root });
const AttestationData = container({
  slot: Slot,
  index: CommitteeIndex,
  beacon_block_root: Root,
  source: Checkpoint,
  target: Checkpoint,
});
const Attestation = container({
  aggregation_bits: bitlist(MAX_VALIDATORS_PER_COMMITTEE),
  data: AttestationData,
  signature: BLSSignature,
});
const AggregateAndProof = container({
  aggregator_index: ValidatorIndex,
  aggregate: Attestation,
  selection_proof: BLSSignature,
});
const IndexedAttestation = container({
  attesting_indices: list(MAX_VALIDATORS_PER_COMMITTEE, ValidatorIndex),
  data: AttestationData,
  signature: BLSSignature,
});
const AttesterSlashing = container({
  attestation_1: IndexedAttestation,
  attestation_2: IndexedAttestation,
});
const BLSToExecutionChange = container({
  validator_index: ValidatorIndex,
  from_bls_pubkey: BLSPubkey,
  to_execution_address: ExecutionAddress,
});
const Withdrawal = container({
  index: WithdrawalIndex,
  validator_index: ValidatorIndex,
  address: ExecutionAddress,
  amount: Gwei,
});
const ExecutionPayload = container({
  parent_hash: Hash32,
  fee_recipient: ExecutionAddress,
  state_root: Bytes32,
  receipts_root: Bytes32,
  logs_bloom: bytevector(BYTES_PER_LOGS_BLOOM),
  prev_randao: Bytes32,
  block_number: uint64,
  gas_limit: uint64,
  gas_used: uint64,
  timestamp: uint64,
  extra_data: bytelist(MAX_EXTRA_DATA_BYTES),
  base_fee_per_gas: uint256,
  block_hash: Hash32,
  transactions: list(MAX_TRANSACTIONS_PER_PAYLOAD, Transaction),
  withdrawals: list(MAX_WITHDRAWALS_PER_PAYLOAD, Withdrawal),
  blob_gas_used: uint64,
  excess_blob_gas: uint64,
});
MAX_WITHDRAWALS_PER_PAYLOAD;
const SigningData = container({ object_root: Root, domain: Domain });
const BeaconBlockHeader = container({
  slot: Slot,
  proposer_index: ValidatorIndex,
  parent_root: Root,
  state_root: Root,
  body_root: Root,
});
const SignedBeaconBlockHeader = container({ message: BeaconBlockHeader, signature: BLSSignature });
const ProposerSlashing = container({
  signed_header_1: SignedBeaconBlockHeader,
  signed_header_2: SignedBeaconBlockHeader,
});
const DepositData = container({
  pubkey: BLSPubkey,
  withdrawal_credentials: Bytes32,
  amount: Gwei,
  signature: BLSSignature,
});
const Deposit = container({
  proof: vector(DEPOSIT_CONTRACT_TREE_DEPTH + 1, Bytes32),
  data: DepositData,
});
const VoluntaryExit = container({ epoch: Epoch, validator_index: ValidatorIndex });
const SyncAggregate = container({
  sync_committee_bits: bitvector(SYNC_COMMITTEE_SIZE),
  sync_committee_signature: BLSSignature,
});
const Eth1Data = container({
  deposit_root: Root,
  deposit_count: uint64,
  block_hash: Hash32,
});
const SignedVoluntaryExit = container({ message: VoluntaryExit, signature: BLSSignature });
const SignedBLSToExecutionChange = container({
  message: BLSToExecutionChange,
  signature: BLSSignature,
});
const BeaconBlockBody = container({
  randao_reveal: BLSSignature,
  eth1_data: Eth1Data,
  graffiti: Bytes32,
  proposer_slashings: list(MAX_PROPOSER_SLASHINGS, ProposerSlashing),
  attester_slashings: list(MAX_ATTESTER_SLASHINGS, AttesterSlashing),
  attestations: list(MAX_ATTESTATIONS, Attestation),
  deposits: list(MAX_DEPOSITS, Deposit),
  voluntary_exits: list(MAX_VOLUNTARY_EXITS, SignedVoluntaryExit),
  sync_aggregate: SyncAggregate,
  execution_payload: ExecutionPayload,
  bls_to_execution_changes: list(MAX_BLS_TO_EXECUTION_CHANGES, SignedBLSToExecutionChange),
  blob_kzg_commitments: list(MAX_BLOB_COMMITMENTS_PER_BLOCK, KZGCommitment),
});
const BeaconBlock = container({
  slot: Slot,
  proposer_index: ValidatorIndex,
  parent_root: Root,
  state_root: Root,
  body: BeaconBlockBody,
});
const SyncCommittee = container({
  pubkeys: vector(SYNC_COMMITTEE_SIZE, BLSPubkey),
  aggregate_pubkey: BLSPubkey,
});
const Fork = container({
  previous_version: Version,
  current_version: Version,
  epoch: Epoch,
});
const Validator = container({
  pubkey: BLSPubkey,
  withdrawal_credentials: Bytes32,
  effective_balance: Gwei,
  slashed: boolean,
  activation_eligibility_epoch: Epoch,
  activation_epoch: Epoch,
  exit_epoch: Epoch,
  withdrawable_epoch: Epoch,
});
const ExecutionPayloadHeader = container({
  parent_hash: Hash32,
  fee_recipient: ExecutionAddress,
  state_root: Bytes32,
  receipts_root: Bytes32,
  logs_bloom: bytevector(BYTES_PER_LOGS_BLOOM),
  prev_randao: Bytes32,
  block_number: uint64,
  gas_limit: uint64,
  gas_used: uint64,
  timestamp: uint64,
  extra_data: bytelist(MAX_EXTRA_DATA_BYTES),
  base_fee_per_gas: uint256,
  block_hash: Hash32,
  transactions_root: Root,
  withdrawals_root: Root,
  blob_gas_used: uint64,
  excess_blob_gas: uint64,
});
const HistoricalSummary = container({
  block_summary_root: Root,
  state_summary_root: Root,
});
const BeaconState = container({
  genesis_time: uint64,
  genesis_validators_root: Root,
  slot: Slot,
  fork: Fork,
  latest_block_header: BeaconBlockHeader,
  block_roots: vector(SLOTS_PER_HISTORICAL_ROOT, Root),
  state_roots: vector(SLOTS_PER_HISTORICAL_ROOT, Root),
  historical_roots: list(HISTORICAL_ROOTS_LIMIT, Root),
  eth1_data: Eth1Data,
  eth1_data_votes: list(EPOCHS_PER_ETH1_VOTING_PERIOD * SLOTS_PER_EPOCH, Eth1Data),
  eth1_deposit_index: uint64,
  validators: list(VALIDATOR_REGISTRY_LIMIT, Validator),
  balances: list(VALIDATOR_REGISTRY_LIMIT, Gwei),
  randao_mixes: vector(EPOCHS_PER_HISTORICAL_VECTOR, Bytes32),
  slashings: vector(EPOCHS_PER_SLASHINGS_VECTOR, Gwei),
  previous_epoch_participation: list(VALIDATOR_REGISTRY_LIMIT, ParticipationFlags),
  current_epoch_participation: list(VALIDATOR_REGISTRY_LIMIT, ParticipationFlags),
  justification_bits: bitvector(JUSTIFICATION_BITS_LENGTH),
  previous_justified_checkpoint: Checkpoint,
  current_justified_checkpoint: Checkpoint,
  finalized_checkpoint: Checkpoint,
  inactivity_scores: list(VALIDATOR_REGISTRY_LIMIT, uint64),
  current_sync_committee: SyncCommittee,
  next_sync_committee: SyncCommittee,
  latest_execution_payload_header: ExecutionPayloadHeader,
  next_withdrawal_index: WithdrawalIndex,
  next_withdrawal_validator_index: ValidatorIndex,
  historical_summaries: list(HISTORICAL_ROOTS_LIMIT, HistoricalSummary),
});
const BlobIdentifier = container({
  block_root: Root,
  index: BlobIndex,
});
const BlobSidecar = container({
  index: BlobIndex,
  blob: Blob,
  kzg_commitment: KZGCommitment,
  kzg_proof: KZGProof,
  signed_block_header: SignedBeaconBlockHeader,
  kzg_commitment_inclusion_proof: vector(KZG_COMMITMENT_INCLUSION_PROOF_DEPTH, Bytes32),
});
const SyncCommitteeContribution = container({
  slot: Slot,
  beacon_block_root: Root,
  subcommittee_index: uint64,
  aggregation_bits: bitvector(SYNC_COMMITTEE_SIZE / SYNC_COMMITTEE_SUBNET_COUNT),
  signature: BLSSignature,
});
const ContributionAndProof = container({
  aggregator_index: ValidatorIndex,
  contribution: SyncCommitteeContribution,
  selection_proof: BLSSignature,
});
const DepositMessage = container({
  pubkey: BLSPubkey,
  withdrawal_credentials: Bytes32,
  amount: Gwei,
});
const Eth1Block = container({
  timestamp: uint64,
  deposit_root: Root,
  deposit_count: uint64,
});
const ForkData = container({ current_version: Version, genesis_validators_root: Root });
const HistoricalBatch = container({
  block_roots: vector(SLOTS_PER_HISTORICAL_ROOT, Root),
  state_roots: vector(SLOTS_PER_HISTORICAL_ROOT, Root),
});
const PendingAttestation = container({
  aggregation_bits: bitlist(MAX_VALIDATORS_PER_COMMITTEE),
  data: AttestationData,
  inclusion_delay: Slot,
  proposer_index: ValidatorIndex,
});
const PowBlock = container({
  block_hash: Hash32,
  parent_hash: Hash32,
  total_difficulty: uint256,
});
const SignedAggregateAndProof = container({ message: AggregateAndProof, signature: BLSSignature });
const SignedBeaconBlock = container({ message: BeaconBlock, signature: BLSSignature });
const SignedContributionAndProof = container({
  message: ContributionAndProof,
  signature: BLSSignature,
});
const SyncAggregatorSelectionData = container({ slot: Slot, subcommittee_index: uint64 });
const SyncCommitteeMessage = container({
  slot: Slot,
  beacon_block_root: Root,
  validator_index: ValidatorIndex,
  signature: BLSSignature,
});

const LightClientHeader = container({
  beacon: BeaconBlockHeader,
  execution: ExecutionPayloadHeader,
  execution_branch: vector(BLOCK_BODY_EXECUTION_PAYLOAD_DEPTH, Bytes32),
});
const LightClientBootstrap = container({
  header: LightClientHeader,
  current_sync_committee: SyncCommittee,
  current_sync_committee_branch: vector(NEXT_SYNC_COMMITTEE_DEPTH, Bytes32),
});
const LightClientUpdate = container({
  attested_header: LightClientHeader,
  next_sync_committee: SyncCommittee,
  next_sync_committee_branch: vector(NEXT_SYNC_COMMITTEE_DEPTH, Bytes32),
  finalized_header: LightClientHeader,
  finality_branch: vector(FINALIZED_ROOT_DEPTH, Bytes32),
  sync_aggregate: SyncAggregate,
  signature_slot: Slot,
});
const LightClientFinalityUpdate = container({
  attested_header: LightClientHeader,
  finalized_header: LightClientHeader,
  finality_branch: vector(FINALIZED_ROOT_DEPTH, Bytes32),
  sync_aggregate: SyncAggregate,
  signature_slot: Slot,
});
const LightClientOptimisticUpdate = container({
  attested_header: LightClientHeader,
  sync_aggregate: SyncAggregate,
  signature_slot: Slot,
});
// Electra
const DepositRequest = container({
  pubkey: BLSPubkey,
  withdrawal_credentials: Bytes32,
  amount: Gwei,
  signature: BLSSignature,
  index: uint64,
});
const WithdrawalRequest = container({
  source_address: ExecutionAddress,
  validator_pubkey: BLSPubkey,
  amount: Gwei,
});
const ConsolidationRequest = container({
  source_address: ExecutionAddress,
  source_pubkey: BLSPubkey,
  target_pubkey: BLSPubkey,
});
const PendingBalanceDeposit = container({
  index: ValidatorIndex,
  amount: Gwei,
});
const PendingPartialWithdrawal = container({
  index: ValidatorIndex,
  amount: Gwei,
  withdrawable_epoch: Epoch,
});
const PendingConsolidation = container({
  source_index: ValidatorIndex,
  target_index: ValidatorIndex,
});

export const ETH2_TYPES = {
  Slot,
  Epoch,
  CommitteeIndex,
  ValidatorIndex,
  WithdrawalIndex,
  Gwei,
  Root,
  Hash32,
  Bytes32,
  Version,
  DomainType,
  ForkDigest,
  Domain,
  BLSPubkey,
  BLSSignature,
  Ether,
  ParticipationFlags,
  ExecutionAddress,
  PayloadId,
  KZGCommitment,
  KZGProof,
  // Containters
  Checkpoint,
  AttestationData,
  Attestation,
  AggregateAndProof,
  IndexedAttestation,
  AttesterSlashing,
  BLSToExecutionChange,
  ExecutionPayload,
  SyncAggregate,
  VoluntaryExit,
  BeaconBlockHeader,
  SigningData,
  SignedBeaconBlockHeader,
  ProposerSlashing,
  DepositData,
  Deposit,
  SignedVoluntaryExit,
  Eth1Data,
  Withdrawal,
  BeaconBlockBody,
  BeaconBlock,
  SyncCommittee,
  Fork,
  Validator,
  ExecutionPayloadHeader,
  HistoricalSummary,
  BeaconState,
  BlobIdentifier,
  BlobSidecar,
  ContributionAndProof,
  DepositMessage,
  Eth1Block,
  ForkData,
  HistoricalBatch,
  PendingAttestation,
  PowBlock,
  Transaction,
  SignedAggregateAndProof,
  SignedBLSToExecutionChange,
  SignedBeaconBlock,
  SignedContributionAndProof,
  SyncAggregatorSelectionData,
  SyncCommitteeContribution,
  SyncCommitteeMessage,
  // Light client
  LightClientHeader,
  LightClientBootstrap,
  LightClientUpdate,
  LightClientOptimisticUpdate,
  LightClientFinalityUpdate,
  // Electra
  DepositRequest,
  WithdrawalRequest,
  ConsolidationRequest,
  PendingBalanceDeposit,
  PendingPartialWithdrawal,
  PendingConsolidation,
};

// EIP-7688
const MAX_ATTESTATION_FIELDS = 8;
const MAX_INDEXED_ATTESTATION_FIELDS = 8;
const MAX_EXECUTION_PAYLOAD_FIELDS = 64;
const MAX_BEACON_BLOCK_BODY_FIELDS = 64;
const MAX_BEACON_STATE_FIELDS = 128;
const MAX_EXECUTION_REQUESTS_FIELDS = 16;

const StableAttestation = stableContainer(MAX_ATTESTATION_FIELDS, {
  aggregation_bits: bitlist(MAX_VALIDATORS_PER_COMMITTEE * MAX_COMMITTEES_PER_SLOT),
  data: AttestationData,
  signature: BLSSignature,
  committee_bits: bitvector(MAX_COMMITTEES_PER_SLOT),
});
const StableIndexedAttestation = stableContainer(MAX_INDEXED_ATTESTATION_FIELDS, {
  attesting_indices: list(MAX_VALIDATORS_PER_COMMITTEE * MAX_COMMITTEES_PER_SLOT, ValidatorIndex),
  data: AttestationData,
  signature: BLSSignature,
});
const StableAttesterSlashing = container({
  attestation_1: StableIndexedAttestation,
  attestation_2: StableIndexedAttestation,
});
const StableExecutionRequests = stableContainer(MAX_EXECUTION_REQUESTS_FIELDS, {
  deposits: list(MAX_DEPOSIT_REQUESTS_PER_PAYLOAD, DepositRequest), // [New in Electra:EIP6110]
  withdrawals: list(MAX_WITHDRAWAL_REQUESTS_PER_PAYLOAD, WithdrawalRequest), // [New in Electra:EIP7002:EIP7251]
  consolidations: list(MAX_CONSOLIDATION_REQUESTS_PER_PAYLOAD, ConsolidationRequest), // [New in Electra:EIP7251]
});
const StableExecutionPayload = stableContainer(MAX_EXECUTION_PAYLOAD_FIELDS, {
  parent_hash: Hash32,
  fee_recipient: ExecutionAddress,
  state_root: Bytes32,
  receipts_root: Bytes32,
  logs_bloom: bytevector(BYTES_PER_LOGS_BLOOM),
  prev_randao: Bytes32,
  block_number: uint64,
  gas_limit: uint64,
  gas_used: uint64,
  timestamp: uint64,
  extra_data: bytelist(MAX_EXTRA_DATA_BYTES),
  base_fee_per_gas: uint256,
  block_hash: Hash32,
  transactions: list(MAX_TRANSACTIONS_PER_PAYLOAD, Transaction),
  withdrawals: list(MAX_WITHDRAWALS_PER_PAYLOAD, Withdrawal), // [New in Capella]
  blob_gas_used: uint64,
  excess_blob_gas: uint64,
  deposit_requests: list(MAX_DEPOSIT_REQUESTS_PER_PAYLOAD, DepositRequest), // [New in Electra:EIP6110]
  withdrawal_requests: list(MAX_WITHDRAWAL_REQUESTS_PER_PAYLOAD, WithdrawalRequest), // [New in Electra:EIP7002:EIP7251]
  consolidation_requests: list(MAX_CONSOLIDATION_REQUESTS_PER_PAYLOAD, ConsolidationRequest), // [New in Electra:EIP7251]
});
const StableExecutionPayloadHeader = stableContainer(MAX_EXECUTION_PAYLOAD_FIELDS, {
  parent_hash: Hash32,
  fee_recipient: ExecutionAddress,
  state_root: Bytes32,
  receipts_root: Bytes32,
  logs_bloom: bytevector(BYTES_PER_LOGS_BLOOM),
  prev_randao: Bytes32,
  block_number: uint64,
  gas_limit: uint64,
  gas_used: uint64,
  timestamp: uint64,
  extra_data: bytelist(MAX_EXTRA_DATA_BYTES),
  base_fee_per_gas: uint256,
  block_hash: Hash32,
  transactions_root: Root,
  withdrawals_root: Root, // [New in Capella]
  blob_gas_used: uint64, // [New in Deneb:EIP4844]
  excess_blob_gas: uint64, // [New in Deneb:EIP4844]
  deposit_requests_root: Root, // [New in Electra:EIP6110]
  withdrawal_requests_root: Root, // [New in Electra:EIP7002:EIP7251]
  consolidation_requests_root: Root, // [New in Electra:EIP7251]
});
const StableBeaconBlockBody = stableContainer(MAX_BEACON_BLOCK_BODY_FIELDS, {
  randao_reveal: BLSSignature,
  eth1_data: Eth1Data,
  graffiti: Bytes32,
  proposer_slashings: list(MAX_PROPOSER_SLASHINGS, ProposerSlashing),
  attester_slashings: list(MAX_ATTESTER_SLASHINGS_ELECTRA, StableAttesterSlashing), // [Modified in Electra:EIP7549]
  attestations: list(MAX_ATTESTATIONS_ELECTRA, StableAttestation), // [Modified in Electra:EIP7549]
  deposits: list(MAX_DEPOSITS, Deposit),
  voluntary_exits: list(MAX_VOLUNTARY_EXITS, SignedVoluntaryExit),
  sync_aggregate: SyncAggregate,
  execution_payload: StableExecutionPayload,
  bls_to_execution_changes: list(MAX_BLS_TO_EXECUTION_CHANGES, SignedBLSToExecutionChange),
  blob_kzg_commitments: list(MAX_BLOB_COMMITMENTS_PER_BLOCK, KZGCommitment),
  execution_requests: StableExecutionRequests,
});
const StableBeaconState = stableContainer(MAX_BEACON_STATE_FIELDS, {
  genesis_time: uint64,
  genesis_validators_root: Root,
  slot: Slot,
  fork: Fork,
  latest_block_header: BeaconBlockHeader,
  block_roots: vector(SLOTS_PER_HISTORICAL_ROOT, Root),
  state_roots: vector(SLOTS_PER_HISTORICAL_ROOT, Root),
  historical_roots: list(HISTORICAL_ROOTS_LIMIT, Root),
  eth1_data: Eth1Data,
  eth1_data_votes: list(EPOCHS_PER_ETH1_VOTING_PERIOD * SLOTS_PER_EPOCH, Eth1Data),
  eth1_deposit_index: uint64,
  validators: list(VALIDATOR_REGISTRY_LIMIT, Validator),
  balances: list(VALIDATOR_REGISTRY_LIMIT, Gwei),
  randao_mixes: vector(EPOCHS_PER_HISTORICAL_VECTOR, Bytes32),
  slashings: vector(EPOCHS_PER_SLASHINGS_VECTOR, Gwei),
  previous_epoch_participation: list(VALIDATOR_REGISTRY_LIMIT, ParticipationFlags),
  current_epoch_participation: list(VALIDATOR_REGISTRY_LIMIT, ParticipationFlags),
  justification_bits: bitvector(JUSTIFICATION_BITS_LENGTH),
  previous_justified_checkpoint: Checkpoint,
  current_justified_checkpoint: Checkpoint,
  finalized_checkpoint: Checkpoint,
  inactivity_scores: list(VALIDATOR_REGISTRY_LIMIT, uint64),
  current_sync_committee: SyncCommittee,
  next_sync_committee: SyncCommittee,
  latest_execution_payload_header: StableExecutionPayloadHeader,
  next_withdrawal_index: WithdrawalIndex,
  next_withdrawal_validator_index: ValidatorIndex,
  historical_summaries: list(HISTORICAL_ROOTS_LIMIT, HistoricalSummary),
  deposit_requests_start_index: uint64, // [New in Electra:EIP6110]
  deposit_balance_to_consume: Gwei, // [New in Electra:EIP7251]
  exit_balance_to_consume: Gwei, // [New in Electra:EIP7251]
  earliest_exit_epoch: Epoch, // [New in Electra:EIP7251]
  consolidation_balance_to_consume: Gwei, // [New in Electra:EIP7251]
  earliest_consolidation_epoch: Epoch, // [New in Electra:EIP7251]
  pending_balance_deposits: list(PENDING_BALANCE_DEPOSITS_LIMIT, PendingBalanceDeposit), // [New in Electra:EIP7251]
  pending_partial_withdrawals: list(PENDING_PARTIAL_WITHDRAWALS_LIMIT, PendingPartialWithdrawal), // [New in Electra:EIP7251]
  pending_consolidations: list(PENDING_CONSOLIDATIONS_LIMIT, PendingConsolidation), // [New in Electra:EIP7251]
});

export const ETH2_CONSENSUS = {
  StableAttestation,
  StableIndexedAttestation,
  StableAttesterSlashing,
  StableExecutionPayload,
  StableExecutionRequests,
  StableExecutionPayloadHeader,
  StableBeaconBlockBody,
  StableBeaconState,
};

// Tests (electra profiles): https://github.com/ethereum/consensus-specs/pull/3844#issuecomment-2239285376
// NOTE: these are different from EIP-7688 by some reasons, but since nothing is merged/completed in eth side, we just trying
// to pass these tests for now.
const IndexedAttestationElectra = profile(
  StableIndexedAttestation,
  [],
  ['attesting_indices', 'data', 'signature']
);
const AttesterSlashingElectra = container({
  attestation_1: IndexedAttestationElectra,
  attestation_2: IndexedAttestationElectra,
});
const ExecutionPayloadHeaderElectra = profile(
  StableExecutionPayloadHeader,
  [],
  [
    'parent_hash',
    'fee_recipient',
    'state_root',
    'receipts_root',
    'logs_bloom',
    'prev_randao',
    'block_number',
    'gas_limit',
    'gas_used',
    'timestamp',
    'extra_data',
    'base_fee_per_gas',
    'block_hash',
    'transactions_root',
    'withdrawals_root',
    'blob_gas_used',
    'excess_blob_gas',
  ]
);
const ExecutionRequests = profile(
  StableExecutionRequests,
  [],
  ['deposits', 'withdrawals', 'consolidations']
);
const AttestationElectra = profile(
  StableAttestation,
  [],
  ['aggregation_bits', 'data', 'signature', 'committee_bits']
);
const ExecutionPayloadElectra = profile(
  StableExecutionPayload,
  [],
  [
    'parent_hash',
    'fee_recipient',
    'state_root',
    'receipts_root',
    'logs_bloom',
    'prev_randao',
    'block_number',
    'gas_limit',
    'gas_used',
    'timestamp',
    'extra_data',
    'base_fee_per_gas',
    'block_hash',
    'transactions',
    'withdrawals',
    'blob_gas_used',
    'excess_blob_gas',
  ]
);
export const ETH2_PROFILES = {
  electra: {
    Attestation: AttestationElectra,
    AttesterSlashing: AttesterSlashingElectra,
    IndexedAttestation: IndexedAttestationElectra,
    ExecutionRequests,
    ExecutionPayloadHeader: ExecutionPayloadHeaderElectra,
    ExecutionPayload: ExecutionPayloadElectra,
    BeaconBlockBody: profile(
      StableBeaconBlockBody,
      [],
      [
        'randao_reveal',
        'eth1_data',
        'graffiti',
        'proposer_slashings',
        'attester_slashings',
        'attestations',
        'deposits',
        'voluntary_exits',
        'sync_aggregate',
        'execution_payload',
        'bls_to_execution_changes',
        'blob_kzg_commitments',
        'execution_requests',
      ],
      {
        attester_slashings: list(MAX_ATTESTER_SLASHINGS_ELECTRA, AttesterSlashingElectra),
        attestations: list(MAX_ATTESTATIONS_ELECTRA, AttestationElectra),
        execution_payload: ExecutionPayloadElectra,
        execution_requests: ExecutionRequests,
      }
    ),
    BeaconState: profile(
      StableBeaconState,
      [],
      [
        'genesis_time',
        'genesis_validators_root',
        'slot',
        'fork',
        'latest_block_header',
        'block_roots',
        'state_roots',
        'historical_roots',
        'eth1_data',
        'eth1_data_votes',
        'eth1_deposit_index',
        'validators',
        'balances',
        'randao_mixes',
        'slashings',
        'previous_epoch_participation',
        'current_epoch_participation',
        'justification_bits',
        'previous_justified_checkpoint',
        'current_justified_checkpoint',
        'finalized_checkpoint',
        'inactivity_scores',
        'current_sync_committee',
        'next_sync_committee',
        'latest_execution_payload_header',
        'next_withdrawal_index',
        'next_withdrawal_validator_index',
        'historical_summaries',
        'deposit_requests_start_index',
        'deposit_balance_to_consume',
        'exit_balance_to_consume',
        'earliest_exit_epoch',
        'consolidation_balance_to_consume',
        'earliest_consolidation_epoch',
        'pending_balance_deposits',
        'pending_partial_withdrawals',
        'pending_consolidations',
      ],
      {
        latest_execution_payload_header: ExecutionPayloadHeaderElectra,
      }
    ),
  },
};

/** Capella Types */
export const CapellaExecutionPayloadHeader = container({
  parent_hash: ETH2_TYPES.Hash32,
  fee_recipient: ETH2_TYPES.ExecutionAddress,
  state_root: ETH2_TYPES.Bytes32,
  receipts_root: ETH2_TYPES.Bytes32,
  logs_bloom: bytevector(BYTES_PER_LOGS_BLOOM),
  prev_randao: ETH2_TYPES.Bytes32,
  block_number: uint64,
  gas_limit: uint64,
  gas_used: uint64,
  timestamp: uint64,
  extra_data: bytelist(MAX_EXTRA_DATA_BYTES),
  base_fee_per_gas: uint256,
  block_hash: ETH2_TYPES.Hash32,
  transactions_root: ETH2_TYPES.Root,
  withdrawals_root: ETH2_TYPES.Root,
});

const CapellaBeaconBlockBody = container({
  randao_reveal: ETH2_TYPES.BLSSignature,
  eth1_data: ETH2_TYPES.Eth1Data,
  graffiti: ETH2_TYPES.Bytes32,
  proposer_slashings: list(MAX_PROPOSER_SLASHINGS, ETH2_TYPES.ProposerSlashing),
  attester_slashings: list(MAX_ATTESTER_SLASHINGS, ETH2_TYPES.AttesterSlashing),
  attestations: list(MAX_ATTESTATIONS, ETH2_TYPES.Attestation),
  deposits: list(MAX_DEPOSITS, ETH2_TYPES.Deposit),
  voluntary_exits: list(MAX_VOLUNTARY_EXITS, ETH2_TYPES.SignedVoluntaryExit),
  sync_aggregate: ETH2_TYPES.SyncAggregate,
  execution_payload: CapellaExecutionPayloadHeader,
  bls_to_execution_changes: list(
    MAX_BLS_TO_EXECUTION_CHANGES,
    ETH2_TYPES.SignedBLSToExecutionChange
  ),
});
export const CapellaBeaconBlock = container({
  slot: ETH2_TYPES.Slot,
  proposer_index: ETH2_TYPES.ValidatorIndex,
  parent_root: ETH2_TYPES.Root,
  state_root: ETH2_TYPES.Root,
  body: CapellaBeaconBlockBody,
});

export const CapellaSignedBeaconBlock = container({
  message: CapellaBeaconBlock,
  signature: ETH2_TYPES.BLSSignature,
});

export const CapellaBeaconState = container({
  genesis_time: uint64,
  genesis_validators_root: ETH2_TYPES.Root,
  slot: ETH2_TYPES.Slot,
  fork: ETH2_TYPES.Fork,
  latest_block_header: ETH2_TYPES.BeaconBlockHeader,
  block_roots: vector(SLOTS_PER_HISTORICAL_ROOT, ETH2_TYPES.Root),
  state_roots: vector(SLOTS_PER_HISTORICAL_ROOT, ETH2_TYPES.Root),
  historical_roots: list(HISTORICAL_ROOTS_LIMIT, ETH2_TYPES.Root),
  eth1_data: ETH2_TYPES.Eth1Data,
  eth1_data_votes: list(EPOCHS_PER_ETH1_VOTING_PERIOD * SLOTS_PER_EPOCH, ETH2_TYPES.Eth1Data),
  eth1_deposit_index: uint64,
  validators: list(VALIDATOR_REGISTRY_LIMIT, ETH2_TYPES.Validator),
  balances: list(VALIDATOR_REGISTRY_LIMIT, ETH2_TYPES.Gwei),
  randao_mixes: vector(EPOCHS_PER_HISTORICAL_VECTOR, ETH2_TYPES.Bytes32),
  slashings: vector(EPOCHS_PER_SLASHINGS_VECTOR, ETH2_TYPES.Gwei),
  previous_epoch_participation: list(VALIDATOR_REGISTRY_LIMIT, ETH2_TYPES.ParticipationFlags),
  current_epoch_participation: list(VALIDATOR_REGISTRY_LIMIT, ETH2_TYPES.ParticipationFlags),
  justification_bits: bitvector(JUSTIFICATION_BITS_LENGTH),
  previous_justified_checkpoint: ETH2_TYPES.Checkpoint,
  current_justified_checkpoint: ETH2_TYPES.Checkpoint,
  finalized_checkpoint: ETH2_TYPES.Checkpoint,
  inactivity_scores: list(VALIDATOR_REGISTRY_LIMIT, uint64),
  current_sync_committee: ETH2_TYPES.SyncCommittee,
  next_sync_committee: ETH2_TYPES.SyncCommittee,
  latest_execution_payload_header: CapellaExecutionPayloadHeader,
  next_withdrawal_index: uint64,
  next_withdrawal_validator_index: uint64,
  historical_summaries: list(HISTORICAL_ROOTS_LIMIT, ETH2_TYPES.HistoricalSummary),
});

/** Bellatrix Types */
export const BellatrixExecutionPayloadHeader = container({
  parent_hash: ETH2_TYPES.Hash32,
  fee_recipient: ETH2_TYPES.ExecutionAddress,
  state_root: ETH2_TYPES.Bytes32,
  receipts_root: ETH2_TYPES.Bytes32,
  logs_bloom: bytevector(BYTES_PER_LOGS_BLOOM),
  prev_randao: ETH2_TYPES.Bytes32,
  block_number: uint64,
  gas_limit: uint64,
  gas_used: uint64,
  timestamp: uint64,
  extra_data: bytelist(MAX_EXTRA_DATA_BYTES),
  base_fee_per_gas: uint256,
  block_hash: ETH2_TYPES.Hash32,
  transactions_root: ETH2_TYPES.Root,
});

const BellatrixBeaconBlockBody = container({
  randao_reveal: ETH2_TYPES.BLSSignature,
  eth1_data: ETH2_TYPES.Eth1Data,
  graffiti: ETH2_TYPES.Bytes32,
  proposer_slashings: list(MAX_PROPOSER_SLASHINGS, ETH2_TYPES.ProposerSlashing),
  attester_slashings: list(MAX_ATTESTER_SLASHINGS, ETH2_TYPES.AttesterSlashing),
  attestations: list(MAX_ATTESTATIONS, ETH2_TYPES.Attestation),
  deposits: list(MAX_DEPOSITS, ETH2_TYPES.Deposit),
  voluntary_exits: list(MAX_VOLUNTARY_EXITS, ETH2_TYPES.SignedVoluntaryExit),
  sync_aggregate: ETH2_TYPES.SyncAggregate,
  execution_payload: BellatrixExecutionPayloadHeader,
  bls_to_execution_changes: list(
    MAX_BLS_TO_EXECUTION_CHANGES,
    ETH2_TYPES.SignedBLSToExecutionChange
  ),
});
export const BellatrixBeaconBlock = container({
  slot: ETH2_TYPES.Slot,
  proposer_index: ETH2_TYPES.ValidatorIndex,
  parent_root: ETH2_TYPES.Root,
  state_root: ETH2_TYPES.Root,
  body: BellatrixBeaconBlockBody,
});

export const BellatrixSignedBeaconBlock = container({
  message: BellatrixBeaconBlock,
  signature: ETH2_TYPES.BLSSignature,
});

export const BellatrixBeaconState = container({
  genesis_time: uint64,
  genesis_validators_root: ETH2_TYPES.Root,
  slot: ETH2_TYPES.Slot,
  fork: ETH2_TYPES.Fork,
  latest_block_header: ETH2_TYPES.BeaconBlockHeader,
  block_roots: vector(SLOTS_PER_HISTORICAL_ROOT, ETH2_TYPES.Root),
  state_roots: vector(SLOTS_PER_HISTORICAL_ROOT, ETH2_TYPES.Root),
  historical_roots: list(HISTORICAL_ROOTS_LIMIT, ETH2_TYPES.Root),
  eth1_data: ETH2_TYPES.Eth1Data,
  eth1_data_votes: list(EPOCHS_PER_ETH1_VOTING_PERIOD * SLOTS_PER_EPOCH, ETH2_TYPES.Eth1Data),
  eth1_deposit_index: uint64,
  validators: list(VALIDATOR_REGISTRY_LIMIT, ETH2_TYPES.Validator),
  balances: list(VALIDATOR_REGISTRY_LIMIT, ETH2_TYPES.Gwei),
  randao_mixes: vector(EPOCHS_PER_HISTORICAL_VECTOR, ETH2_TYPES.Bytes32),
  slashings: vector(EPOCHS_PER_SLASHINGS_VECTOR, ETH2_TYPES.Gwei),
  previous_epoch_participation: list(VALIDATOR_REGISTRY_LIMIT, ETH2_TYPES.ParticipationFlags),
  current_epoch_participation: list(VALIDATOR_REGISTRY_LIMIT, ETH2_TYPES.ParticipationFlags),
  justification_bits: bitvector(JUSTIFICATION_BITS_LENGTH),
  previous_justified_checkpoint: ETH2_TYPES.Checkpoint,
  current_justified_checkpoint: ETH2_TYPES.Checkpoint,
  finalized_checkpoint: ETH2_TYPES.Checkpoint,
  inactivity_scores: list(VALIDATOR_REGISTRY_LIMIT, uint64),
  current_sync_committee: ETH2_TYPES.SyncCommittee,
  next_sync_committee: ETH2_TYPES.SyncCommittee,
  latest_execution_payload_header: BellatrixExecutionPayloadHeader,
});

/** Altair Types */
const AltairBeaconBlockBody = container({
  randao_reveal: ETH2_TYPES.BLSSignature,
  eth1_data: ETH2_TYPES.Eth1Data,
  graffiti: ETH2_TYPES.Bytes32,
  proposer_slashings: list(MAX_PROPOSER_SLASHINGS, ETH2_TYPES.ProposerSlashing),
  attester_slashings: list(MAX_ATTESTER_SLASHINGS, ETH2_TYPES.AttesterSlashing),
  attestations: list(MAX_ATTESTATIONS, ETH2_TYPES.Attestation),
  deposits: list(MAX_DEPOSITS, ETH2_TYPES.Deposit),
  voluntary_exits: list(MAX_VOLUNTARY_EXITS, ETH2_TYPES.SignedVoluntaryExit),
  sync_aggregate: ETH2_TYPES.SyncAggregate,
});
export const AltairBeaconBlock = container({
  slot: ETH2_TYPES.Slot,
  proposer_index: ETH2_TYPES.ValidatorIndex,
  parent_root: ETH2_TYPES.Root,
  state_root: ETH2_TYPES.Root,
  body: AltairBeaconBlockBody,
});

export const AltairSignedBeaconBlock = container({
  message: AltairBeaconBlock,
  signature: ETH2_TYPES.BLSSignature,
});

export const AltairBeaconState = container({
  genesis_time: uint64,
  genesis_validators_root: ETH2_TYPES.Root,
  slot: ETH2_TYPES.Slot,
  fork: ETH2_TYPES.Fork,
  latest_block_header: ETH2_TYPES.BeaconBlockHeader,
  block_roots: vector(SLOTS_PER_HISTORICAL_ROOT, ETH2_TYPES.Root),
  state_roots: vector(SLOTS_PER_HISTORICAL_ROOT, ETH2_TYPES.Root),
  historical_roots: list(HISTORICAL_ROOTS_LIMIT, ETH2_TYPES.Root),
  eth1_data: ETH2_TYPES.Eth1Data,
  eth1_data_votes: list(EPOCHS_PER_ETH1_VOTING_PERIOD * SLOTS_PER_EPOCH, ETH2_TYPES.Eth1Data),
  eth1_deposit_index: uint64,
  validators: list(VALIDATOR_REGISTRY_LIMIT, ETH2_TYPES.Validator),
  balances: list(VALIDATOR_REGISTRY_LIMIT, ETH2_TYPES.Gwei),
  randao_mixes: vector(EPOCHS_PER_HISTORICAL_VECTOR, ETH2_TYPES.Bytes32),
  slashings: vector(EPOCHS_PER_SLASHINGS_VECTOR, ETH2_TYPES.Gwei),
  previous_epoch_participation: list(VALIDATOR_REGISTRY_LIMIT, ETH2_TYPES.ParticipationFlags),
  current_epoch_participation: list(VALIDATOR_REGISTRY_LIMIT, ETH2_TYPES.ParticipationFlags),
  justification_bits: bitvector(JUSTIFICATION_BITS_LENGTH),
  previous_justified_checkpoint: ETH2_TYPES.Checkpoint,
  current_justified_checkpoint: ETH2_TYPES.Checkpoint,
  finalized_checkpoint: ETH2_TYPES.Checkpoint,
  inactivity_scores: list(VALIDATOR_REGISTRY_LIMIT, uint64),
  current_sync_committee: ETH2_TYPES.SyncCommittee,
  next_sync_committee: ETH2_TYPES.SyncCommittee,
});

/** Phase0 Types */
const Phase0BeaconBlockBody = container({
  randao_reveal: ETH2_TYPES.BLSSignature,
  eth1_data: ETH2_TYPES.Eth1Data,
  graffiti: ETH2_TYPES.Bytes32,
  proposer_slashings: list(MAX_PROPOSER_SLASHINGS, ETH2_TYPES.ProposerSlashing),
  attester_slashings: list(MAX_ATTESTER_SLASHINGS, ETH2_TYPES.AttesterSlashing),
  attestations: list(MAX_ATTESTATIONS, ETH2_TYPES.Attestation),
  deposits: list(MAX_DEPOSITS, ETH2_TYPES.Deposit),
  voluntary_exits: list(MAX_VOLUNTARY_EXITS, ETH2_TYPES.SignedVoluntaryExit),
});
export const Phase0BeaconBlock = container({
  slot: ETH2_TYPES.Slot,
  proposer_index: ETH2_TYPES.ValidatorIndex,
  parent_root: ETH2_TYPES.Root,
  state_root: ETH2_TYPES.Root,
  body: Phase0BeaconBlockBody,
});

export const Phase0SignedBeaconBlock = container({
  message: Phase0BeaconBlock,
  signature: ETH2_TYPES.BLSSignature,
});
export const Phase0BeaconState = container({
  genesis_time: uint64,
  genesis_validators_root: ETH2_TYPES.Root,
  slot: ETH2_TYPES.Slot,
  fork: ETH2_TYPES.Fork,
  latest_block_header: ETH2_TYPES.BeaconBlockHeader,
  block_roots: vector(SLOTS_PER_HISTORICAL_ROOT, ETH2_TYPES.Root),
  state_roots: vector(SLOTS_PER_HISTORICAL_ROOT, ETH2_TYPES.Root),
  historical_roots: list(HISTORICAL_ROOTS_LIMIT, ETH2_TYPES.Root),
  eth1_data: ETH2_TYPES.Eth1Data,
  eth1_data_votes: list(EPOCHS_PER_ETH1_VOTING_PERIOD * SLOTS_PER_EPOCH, ETH2_TYPES.Eth1Data),
  eth1_deposit_index: uint64,
  validators: list(VALIDATOR_REGISTRY_LIMIT, ETH2_TYPES.Validator),
  balances: list(VALIDATOR_REGISTRY_LIMIT, ETH2_TYPES.Gwei),
  randao_mixes: vector(EPOCHS_PER_HISTORICAL_VECTOR, ETH2_TYPES.Bytes32),
  slashings: vector(EPOCHS_PER_SLASHINGS_VECTOR, ETH2_TYPES.Gwei),
  previous_epoch_participation: list(VALIDATOR_REGISTRY_LIMIT, ETH2_TYPES.ParticipationFlags),
  current_epoch_participation: list(VALIDATOR_REGISTRY_LIMIT, ETH2_TYPES.ParticipationFlags),
  justification_bits: bitvector(JUSTIFICATION_BITS_LENGTH),
  previous_justified_checkpoint: ETH2_TYPES.Checkpoint,
  current_justified_checkpoint: ETH2_TYPES.Checkpoint,
  finalized_checkpoint: ETH2_TYPES.Checkpoint,
});
