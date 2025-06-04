import { keccak_256 } from '@noble/hashes/sha3';
import { bytesToHex, concatBytes, hexToBytes } from '@noble/hashes/utils';
import * as P from 'micro-packed';
import {
  type IWeb3Provider,
  type Web3CallArgs,
  add0x,
  ethHex,
  omit,
  strip0x,
  zip,
} from '../utils.ts';

/*
There is NO network code in the file. However, a user can pass
NetProvider instance to createContract, and the method would do
network requests with the api.

There is some really crazy stuff going on here with Typescript types.
*/

function EPad<T>(p: P.CoderType<T>) {
  return P.padLeft(32, p, P.ZeroPad);
}

// Main difference between regular array: length stored outside and offsets calculated without length
function ethArray<T>(inner: P.CoderType<T>): P.CoderType<T[]> {
  return P.wrap({
    size: undefined,
    encodeStream: (w: P.Writer, value: T[]) => {
      U256BE_LEN.encodeStream(w, value.length);
      w.bytes(P.array(value.length, inner).encode(value));
    },
    decodeStream: (r: P.Reader): T[] =>
      P.array(U256BE_LEN.decodeStream(r), inner).decodeStream(r.offsetReader(r.pos)),
  });
}

const PTR = EPad(P.U32BE);
const ARRAY_RE = /(.+)(\[(\d+)?\])$/; // TODO: is this correct?
// Because u32 in eth is not real u32, just U256BE with limits...
const ethInt = (bits: number, signed = false) => {
  if (!Number.isSafeInteger(bits) || bits <= 0 || bits % 8 !== 0 || bits > 256)
    throw new Error('ethInt: invalid numeric type');
  const _bits = BigInt(bits);
  const inner = P.bigint(32, false, signed);
  return P.validate(
    P.wrap({
      size: inner.size,
      encodeStream: (w: P.Writer, value: bigint) => inner.encodeStream(w, value),
      decodeStream: (r: P.Reader): bigint => inner.decodeStream(r),
    }),
    (value) => {
      // TODO: validate useful for narrowing types, need to add support in types?
      if (typeof value === 'number') value = BigInt(value);
      P.utils.checkBounds(value, _bits, !!signed);
      return value;
    }
  );
};

// Ugly hack, because tuple of pointers considered "dynamic" without any reason.
function isDyn<T>(args: P.CoderType<T>[] | Record<string, P.CoderType<T>>) {
  let res = false;
  if (Array.isArray(args)) {
    for (let arg of args) if (arg.size === undefined) res = true;
  } else {
    for (let arg in args) if (args[arg].size === undefined) res = true;
  }
  return res;
}

// as const returns readonly stuff, remove readonly property
type Writable<T> = T extends {}
  ? {
      -readonly [P in keyof T]: Writable<T[P]>;
    }
  : T;
type ArrLike<T> = Array<T> | ReadonlyArray<T>;
export type IsEmptyArray<T> =
  T extends ReadonlyArray<any> ? (T['length'] extends 0 ? true : false) : true;

export type Component<T extends string> = {
  readonly name?: string;
  readonly type: T;
};
export type NamedComponent<T extends string> = Component<T> & { readonly name: string };
export type BaseComponent = Component<string>;
export type Tuple<TC extends ArrLike<Component<string>>> = {
  readonly name?: string;
  readonly type: 'tuple';
  readonly components: TC;
};

// Basic type support
// int<M>: two’s complement signed integer type of M bits, 0 < M <= 256, M % 8 == 0.
// prettier-ignore
type IntIdxType = ''    | '8'   | '16'  | '24'  | '32'  | '40'  | '48'  | '56'  |
  '64'  | '72'  | '80'  | '88'  | '96'  | '104' | '112' | '120' | '128' | '136' |
  '144' | '152' | '160' | '168' | '176' | '184' | '192' | '200' | '208' | '216' |
  '224' | '232' | '240' | '248' | '256';
type UintType = `uint${IntIdxType}`;
type IntType = `int${IntIdxType}`;
type NumberType = UintType | IntType;
// bytes<M>: binary type of M bytes, 0 < M <= 32.
// prettier-ignore
type ByteIdxType = '' | '1' | '2'  | '3'  | '4'  | '5'  | '6'  | '7'  | '8'  | '9'  |
  '10' | '11' | '12' | '13' | '14' | '15' | '16' | '17' | '18' | '19' | '20' | '21' |
  '22' | '23' | '24' | '25' | '26' | '27' | '28' | '29' | '30' | '31' | '32';
type ByteType = `bytes${ByteIdxType}`;

// [{name: 'a', type: 'string'}, {name: 'b', type: 'uint'}] -> {a: string, b: bigint};
export type MapTuple<T> =
  T extends ArrLike<Component<string> & { name: string }>
    ? {
        [K in T[number] as K['name']]: MapType<K>;
      }
    : T extends ArrLike<Component<string>>
      ? // [{name: 'a', type: 'string'}, {type: 'uint'}] -> [string, bigint];
        {
          [K in keyof T]: T[K] extends BaseComponent ? MapType<T[K]> : unknown;
        }
      : unknown;

// prettier-ignore
export type GetType<T extends string> =
  T extends `${infer Base}[]${infer Rest}` ? GetType<`${Base}${Rest}`>[] : // 'string[]' -> 'string'[]
  T extends `${infer Base}[${number}]${infer Rest}` ? GetType<`${Base}${Rest}`>[] : // 'string[3]' -> 'string'[]
  T extends 'address' ? string :
  T extends 'string' ? string :
  T extends 'bool' ? boolean :
  T extends NumberType ? bigint :
  T extends ByteType ? Uint8Array :
  unknown; // default

// prettier-ignore
export type MapType<T extends BaseComponent> =
  T extends Tuple<Array<Component<string>>> ? MapTuple<T['components']> :
  T extends Component<infer Type> ? GetType<Type> :
  unknown; // default

// Re-use ptr for len. u32 should be enough.
const U256BE_LEN = PTR;
export type UnmapType<T> = T extends MapType<infer U> ? U : never;
// NOTE: we need as const if we want to access string as values inside types :(
export function mapComponent<T extends BaseComponent>(c: T): P.CoderType<MapType<Writable<T>>> {
  // Arrays (should be first one, since recursive)
  let m;
  if ((m = ARRAY_RE.exec(c.type))) {
    const inner = mapComponent({ ...c, type: m[1] });
    if (inner.size === 0)
      throw new Error('mapComponent: arrays of zero-size elements disabled (possible DoS attack)');
    // Static array
    if (m[3] !== undefined) {
      const m3 = Number.parseInt(m[3]);
      if (!Number.isSafeInteger(m3)) throw new Error(`mapComponent: wrong array size=${m[3]}`);
      let out = P.array(m3, inner);
      // Static array of dynamic values should be behind pointer too, again without reason.
      if (inner.size === undefined) out = P.pointer(PTR, out);
      return out as any;
    } else {
      // Dynamic array
      return P.pointer(PTR, ethArray(inner)) as any;
    }
  }
  if (c.type === 'tuple') {
    const components: (Component<string> & { name?: string })[] = (c as any).components;
    let hasNames = true;
    const args: P.CoderType<any>[] = [];
    for (let comp of components) {
      if (!comp.name) hasNames = false;
      args.push(mapComponent(comp));
    }
    let out: any;
    // If there is names for all fields -- return struct, otherwise tuple
    if (hasNames) {
      const struct: Record<string, P.CoderType<unknown>> = {};
      for (const arg of components) {
        if (struct[arg.name!]) throw new Error(`mapType: same field name=${arg.name}`);
        struct[arg.name!] = mapComponent(arg);
      }
      out = P.struct(struct);
    } else out = P.tuple(args);
    // If tuple has dynamic elements it becomes dynamic too, without reason.
    if (isDyn(args)) out = P.pointer(PTR, out);
    return out;
  }
  if (c.type === 'string')
    return P.pointer(PTR, P.padRight(32, P.string(U256BE_LEN), P.ZeroPad)) as any;
  if (c.type === 'bytes')
    return P.pointer(PTR, P.padRight(32, P.bytes(U256BE_LEN), P.ZeroPad)) as any;
  if (c.type === 'address') return EPad(P.hex(20, { isLE: false, with0x: true })) as any;
  if (c.type === 'bool') return EPad(P.bool) as any;
  if ((m = /^(u?)int([0-9]+)?$/.exec(c.type)))
    return ethInt(m[2] ? +m[2] : 256, m[1] !== 'u') as any;
  if ((m = /^bytes([0-9]{1,2})$/.exec(c.type))) {
    const parsed = +m[1];
    if (!parsed || parsed > 32) throw new Error('wrong bytes<N> type');
    return P.padRight(32, P.bytes(parsed), P.ZeroPad) as any;
  }
  throw new Error(`mapComponent: unknown component=${c}`);
}

// If only one arg -- use as is, otherwise construct tuple by tuple rules
export type ArgsType<T extends ReadonlyArray<any> | undefined> =
  IsEmptyArray<T> extends true
    ? undefined // empty arr
    : T extends ReadonlyArray<any>
      ? T['length'] extends 1 // single elm
        ? MapType<T[0]>
        : MapTuple<T>
      : MapTuple<T>;

// Because args and output are not tuple
// TODO: try merge with mapComponent
export function mapArgs<T extends ArrLike<Component<string>>>(
  args: T
): P.CoderType<ArgsType<Writable<T>>> {
  // More ergonomic input/output
  if (args.length === 1) return mapComponent(args[0] as any) as any;
  let hasNames = true;
  for (const arg of args) if (!arg.name) hasNames = false;
  if (hasNames) {
    const out: Record<string, P.CoderType<unknown>> = {};
    for (const arg of args) {
      const name = (arg as any).name;
      if (out[name]) throw new Error(`mapArgs: same field name=${name}`);
      out[name] = mapComponent(arg as any) as any;
    }
    return P.struct(out) as any;
  } else return P.tuple(args.map(mapComponent)) as any;
}

export type FunctionType = Component<'function'> & {
  readonly inputs?: ReadonlyArray<Component<string>>;
  readonly outputs?: ReadonlyArray<Component<string>>;
};

type ContractMethodDecode<T extends FunctionType, O = ArgsType<T['outputs']>> =
  IsEmptyArray<T['outputs']> extends true
    ? {
        decodeOutput: (b: Uint8Array) => void;
      }
    : { decodeOutput: (b: Uint8Array) => O };

type ContractMethodEncode<T extends FunctionType, I = ArgsType<T['inputs']>> =
  IsEmptyArray<T['inputs']> extends true
    ? { encodeInput: () => Uint8Array }
    : { encodeInput: (v: I) => Uint8Array };

type ContractMethodGas<T extends FunctionType, I = ArgsType<T['inputs']>> =
  IsEmptyArray<T['inputs']> extends true
    ? { estimateGas: () => Promise<bigint> }
    : { estimateGas: (v: I) => Promise<bigint> };

type ContractMethodCall<
  T extends FunctionType,
  I = ArgsType<T['inputs']>,
  O = ArgsType<T['outputs']>,
> =
  IsEmptyArray<T['inputs']> extends true
    ? IsEmptyArray<T['outputs']> extends true
      ? {
          // no inputs, no outputs
          call: () => Promise<void>;
        }
      : {
          // no inputs, outputs
          call: () => Promise<O>;
        }
    : IsEmptyArray<T['outputs']> extends true
      ? {
          // inputs, no outputs
          call: (v: I) => Promise<void>;
        }
      : {
          // inputs, outputs
          call: (v: I) => Promise<O>;
        };

export type ContractMethod<T extends FunctionType> = ContractMethodEncode<T> &
  ContractMethodDecode<T>;

export type ContractMethodNet<T extends FunctionType> = ContractMethod<T> &
  ContractMethodGas<T> &
  ContractMethodCall<T>;

export type FnArg = {
  readonly type: string;
  readonly name?: string;
  readonly components?: ArrLike<FnArg>;
  readonly inputs?: ArrLike<FnArg>;
  readonly outputs?: ArrLike<FnArg>;
  readonly anonymous?: boolean;
  readonly indexed?: boolean;
};

export type ContractTypeFilter<T> = {
  [K in keyof T]: T[K] extends FunctionType & { name: string } ? T[K] : never;
};

export type ContractType<T extends Array<FnArg>, N, F = ContractTypeFilter<T>> =
  F extends ArrLike<FunctionType & { name: string }>
    ? {
        [K in F[number] as K['name']]: N extends IWeb3Provider
          ? ContractMethodNet<K>
          : ContractMethod<K>;
      }
    : never;

function fnSignature(o: FnArg): string {
  if (!o.type) throw new Error('ABI.fnSignature wrong argument');
  if (o.type === 'function' || o.type === 'event')
    return `${o.name || 'function'}(${(o.inputs || []).map((i) => fnSignature(i)).join(',')})`;
  if (o.type.startsWith('tuple')) {
    if (!o.components || !o.components.length) throw new Error('ABI.fnSignature wrong tuple');
    return `(${o.components.map((i) => fnSignature(i)).join(',')})${o.type.slice(5)}`;
  }
  return o.type;
}
// Function signature hash
export function evSigHash(o: FnArg): string {
  return bytesToHex(keccak_256(fnSignature(o)));
}
export function fnSigHash(o: FnArg): string {
  return evSigHash(o).slice(0, 8);
}

// High-level constructs for common ABI use-cases

/*
Call functions always takes two args: array/obj of input values and overrdides of tx params
output is array/obj too, but if there is single input or output, then they processed as-is without wrapping in array/obj.
if there is at least one named input/output (like (uin256 balance, address)) then it is processed as object, where unnamed elements
is refered by index position. Unfortunately it is impossible to do args/kwargs, since named arguments can be before unnamed one.
*/
export function createContract<T extends ArrLike<FnArg>>(
  abi: T,
  net: IWeb3Provider,
  contract?: string
): ContractType<Writable<T>, IWeb3Provider>;
export function createContract<T extends ArrLike<FnArg>>(
  abi: T,
  net?: undefined,
  contract?: string
): ContractType<Writable<T>, undefined>;
export function createContract<T extends ArrLike<FnArg>>(
  abi: T,
  net?: IWeb3Provider,
  contract?: string
): ContractType<Writable<T>, undefined> {
  // Find non-uniq function names so we can handle overloads
  let nameCnt: Record<string, number> = {};
  for (let fn of abi) {
    if (fn.type !== 'function') continue;
    const name = fn.name || 'function';
    if (!nameCnt[name]) nameCnt[name] = 1;
    else nameCnt[name]++;
  }
  const res: Record<string, any> = {};
  for (let fn of abi) {
    if (fn.type !== 'function') continue;
    let name = fn.name || 'function';
    if (nameCnt[name] > 1) name = fnSignature(fn);
    const sh = fnSigHash(fn);
    const inputs = fn.inputs && fn.inputs.length ? mapArgs(fn.inputs) : undefined;
    const outputs = fn.outputs ? mapArgs(fn.outputs) : undefined;
    const decodeOutput = (b: Uint8Array) => outputs && outputs.decode(b);
    const encodeInput = (v: unknown) =>
      concatBytes(hexToBytes(sh), inputs ? inputs.encode(v as any) : new Uint8Array());
    res[name] = { decodeOutput, encodeInput };

    // .call and .estimateGas call network, when net is available
    if (!net) continue;
    res[name].call = async (args: unknown, overrides: Web3CallArgs = {}) => {
      if (!contract && !overrides.to) throw new Error('No contract address');
      const data = add0x(bytesToHex(encodeInput(args)));
      const callArgs = Object.assign({ to: contract, data }, overrides);
      return decodeOutput(hexToBytes(strip0x(await net.ethCall(callArgs))));
    };
    res[name].estimateGas = async (args: unknown, overrides: Web3CallArgs = {}) => {
      if (!contract && !overrides.to) throw new Error('No contract address');
      const data = add0x(bytesToHex(encodeInput(args)));
      const callArgs = Object.assign({ to: contract, data }, overrides);
      return await net.estimateGas(callArgs);
    };
  }
  return res as any;
}

type GetCons<T extends ArrLike<FnArg>> = Extract<T[number], { type: 'constructor' }>;
type ConstructorType = Component<'constructor'> & {
  readonly inputs?: ReadonlyArray<Component<string>>;
};
type ConsArgs<T extends ConstructorType> =
  IsEmptyArray<T['inputs']> extends true ? undefined : ArgsType<T['inputs']>;

export function deployContract<T extends ArrLike<FnArg>>(
  abi: T,
  bytecodeHex: string,
  ...args: GetCons<T> extends never
    ? [args: unknown]
    : ConsArgs<GetCons<T>> extends undefined
      ? []
      : [args: ConsArgs<GetCons<T>>]
): string {
  const bytecode = ethHex.decode(bytecodeHex);
  let consCall;
  for (let fn of abi) {
    if (fn.type !== 'constructor') continue;
    const inputs = fn.inputs && fn.inputs.length ? mapArgs(fn.inputs) : undefined;
    if (inputs === undefined && args !== undefined && args.length)
      throw new Error('arguments to constructor without any');
    consCall = inputs ? inputs.encode(args[0] as any) : new Uint8Array();
  }
  if (!consCall) throw new Error('constructor not found');
  return ethHex.encode(concatBytes(bytecode, consCall));
}

export type EventType = NamedComponent<'event'> & {
  readonly inputs: ReadonlyArray<Component<string>>;
};

export type ContractEventTypeFilter<T> = { [K in keyof T]: T[K] extends EventType ? T[K] : never };

export type TopicsValue<T> = { [K in keyof T]: T[K] | null };

export type EventMethod<T extends EventType> = {
  decode: (topics: string[], data: string) => ArgsType<T['inputs']>;
  topics: (values: TopicsValue<ArgsType<T['inputs']>>) => (string | null)[];
};

export type ContractEventType<T extends Array<FnArg>, F = ContractEventTypeFilter<T>> =
  F extends ArrLike<EventType>
    ? {
        [K in F[number] as K['name']]: EventMethod<K>;
      }
    : never;

// TODO: try to simplify further
export function events<T extends ArrLike<FnArg>>(abi: T): ContractEventType<Writable<T>> {
  let res: Record<string, any> = {};
  for (let elm of abi) {
    // Only named events supported
    if (elm.type !== 'event' || !elm.name) continue;
    const inputs = elm.inputs || [];
    let hasNames = true;
    for (let i of inputs) if (!i.name) hasNames = false;
    const plainInp = inputs.filter((i) => !i.indexed);
    const indexedInp = inputs.filter((i) => i.indexed);
    const indexed = indexedInp.map((i) =>
      !['string', 'bytes', 'tuple'].includes(i.type) && !ARRAY_RE.exec(i.type)
        ? (mapArgs([i]) as any)
        : null
    );
    const parser = mapArgs(hasNames ? plainInp : plainInp.map((i) => omit(i, 'name'))) as any;
    const sigHash = evSigHash(elm);
    res[elm.name] = {
      decode(topics: string[], _data: string) {
        const data = hexToBytes(strip0x(_data));
        if (!elm.anonymous) {
          if (!topics[0]) throw new Error('No signature on non-anonymous event');
          if (strip0x(topics[0]).toLowerCase() !== sigHash) throw new Error('Wrong signature');
          topics = topics.slice(1);
        }
        if (topics.length !== indexed.length) throw new Error('Wrong topics length');
        let parsed = parser ? parser.decode(data) : hasNames ? {} : [];
        const indexedParsed = indexed.map((p, i) =>
          p ? p.decode(hexToBytes(strip0x(topics[i]))) : topics[i]
        );
        if (plainInp.length === 1) parsed = hasNames ? { [plainInp[0].name!]: parsed } : [parsed];
        if (hasNames) {
          let res = { ...parsed };
          for (let [a, p] of zip(indexedInp, indexedParsed)) res[a.name!] = p;
          return res;
        } else return inputs.map((i) => (!i.indexed ? parsed : indexedParsed).shift());
      },
      topics(values: any[] | Record<string, any>) {
        let res = [];
        if (!elm.anonymous) res.push(add0x(sigHash));
        // We require all keys to be set, even if they are null, to be sure nothing is accidentaly missed
        if ((hasNames ? Object.keys(values) : values).length !== inputs.length)
          throw new Error('Wrong topics args');
        for (let i = 0, ii = 0; i < inputs.length && ii < indexed.length; i++) {
          const [input, packer] = [inputs[i], indexed[ii]];
          if (!input.indexed) continue;
          const value = (values as any)[Array.isArray(values) ? i : inputs[i].name!];
          if (value === null) {
            res.push(null);
            continue;
          }
          let topic: string;
          if (packer) topic = bytesToHex(packer.encode(value));
          else if (['string', 'bytes'].includes(input.type)) topic = bytesToHex(keccak_256(value));
          else {
            let m: any, parts: Uint8Array[];
            if ((m = ARRAY_RE.exec(input.type)))
              parts = value.map((j: any) => mapComponent({ type: m[1] }).encode(j));
            else if (input.type === 'tuple' && input.components)
              parts = input.components.map((j) => (mapArgs([j]) as any).encode(value[j.name!]));
            else throw new Error('Unknown unsized type');
            topic = bytesToHex(keccak_256(concatBytes(...parts)));
          }
          res.push(add0x(topic));
          ii++;
        }
        return res;
      },
    };
  }
  return res as any;
}

// Same as 'Transaction Action' on Etherscan, provides human readable interpritation of decoded data
export type ContractABI = ReadonlyArray<FnArg & { readonly hint?: HintFn; readonly hook?: HookFn }>;
export type ContractInfo = {
  abi: 'ERC20' | 'ERC721' | 'ERC1155' | ContractABI;
  symbol?: string;
  decimals?: number;
  // For useful common contracts/exchanges
  name?: string;
  // Stable coin price against USD
  price?: number;
};
export type HintOpt = {
  contract?: string;
  amount?: bigint;
  contractInfo?: ContractInfo;
  contracts?: Record<string, ContractInfo>;
};
export type HintFn = (value: unknown, opt: HintOpt) => string;
export type HookFn = (
  decoder: Decoder,
  contract: string,
  info: SignatureInfo,
  opt: HintOpt
) => SignatureInfo;
type SignaturePacker = {
  name: string;
  signature: string;
  packer: P.CoderType<unknown>;
  hint?: HintFn;
  // Modifies decoder output. For multicall calls.
  hook?: HookFn;
};
type EventSignatureDecoder = {
  name: string;
  signature: string;
  decoder: (topics: string[], _data: string) => unknown;
  hint?: HintFn;
};

export type SignatureInfo = { name: string; signature: string; value: unknown; hint?: string };
export class Decoder {
  contracts: Record<string, Record<string, SignaturePacker>> = {};
  sighashes: Record<string, SignaturePacker[]> = {};
  evContracts: Record<string, Record<string, EventSignatureDecoder>> = {};
  evSighashes: Record<string, EventSignatureDecoder[]> = {};
  add(contract: string, abi: ContractABI): void {
    const ev: any = events(abi);
    contract = strip0x(contract).toLowerCase();
    if (!this.contracts[contract]) this.contracts[contract] = {};
    if (!this.evContracts[contract]) this.evContracts[contract] = {};
    for (let fn of abi) {
      if (fn.type === 'function') {
        const selector = fnSigHash(fn);
        const value = {
          name: fn.name || 'function',
          signature: fnSignature(fn),
          packer: fn.inputs && fn.inputs.length ? (mapArgs(fn.inputs) as any) : undefined,
          hint: fn.hint,
          hook: fn.hook,
        };
        this.contracts[contract][selector] = value;
        if (!this.sighashes[selector]) this.sighashes[selector] = [];
        this.sighashes[selector].push(value);
      } else if (fn.type === 'event') {
        if (fn.anonymous || !fn.name) continue;
        const selector = evSigHash(fn);
        const value = {
          name: fn.name,
          signature: fnSignature(fn),
          decoder: ev[fn.name]?.decode,
          hint: fn.hint,
        };
        this.evContracts[contract][selector] = value;
        if (!this.evSighashes[selector]) this.evSighashes[selector] = [];
        this.evSighashes[selector].push(value);
      }
    }
  }
  method(contract: string, data: Uint8Array): string | undefined {
    contract = strip0x(contract).toLowerCase();
    const sh = bytesToHex(data.slice(0, 4));
    if (!this.contracts[contract] || !this.contracts[contract][sh]) return;
    const { name } = this.contracts[contract][sh];
    return name;
  }
  // Returns: exact match, possible options of matches (array) or undefined.
  // Note that empty value possible if there is no arguments in call.
  decode(
    contract: string,
    _data: Uint8Array,
    opt: HintOpt
  ): SignatureInfo | SignatureInfo[] | undefined {
    contract = strip0x(contract).toLowerCase();
    const sh = bytesToHex(_data.slice(0, 4));
    const data = _data.slice(4);
    if (this.contracts[contract] && this.contracts[contract][sh]) {
      let { name, signature, packer, hint, hook } = this.contracts[contract][sh];
      const value = packer ? packer.decode(data) : undefined;
      let res: SignatureInfo = { name, signature, value };
      // NOTE: hint && hook fn is used only on exact match of contract!
      if (hook) res = hook(this, contract, res, opt);
      try {
        if (hint) res.hint = hint(value, Object.assign({ contract: add0x(contract) }, opt));
      } catch (e) {}
      return res;
    }
    if (!this.sighashes[sh] || !this.sighashes[sh].length) return;
    let res: SignatureInfo[] = [];
    for (let { name, signature, packer } of this.sighashes[sh]) {
      try {
        res.push({ name, signature, value: packer ? packer.decode(data) : undefined });
      } catch (err) {}
    }
    if (res.length) return res;
    return;
  }
  decodeEvent(
    contract: string,
    topics: string[],
    data: string,
    opt: HintOpt
  ): SignatureInfo | SignatureInfo[] | undefined {
    contract = strip0x(contract).toLowerCase();
    if (!topics.length) return;
    const sh = strip0x(topics[0]);
    const event = this.evContracts[contract];
    if (event && event[sh]) {
      let { name, signature, decoder, hint } = event[sh];
      const value = decoder(topics, data);
      let res: SignatureInfo = { name, signature, value };
      try {
        if (hint) res.hint = hint(value, Object.assign({ contract: add0x(contract) }, opt));
      } catch (e) {}
      return res;
    }
    if (!this.evSighashes[sh] || !this.evSighashes[sh].length) return;
    let res: SignatureInfo[] = [];
    for (let { name, signature, decoder } of this.evSighashes[sh]) {
      try {
        res.push({ name, signature, value: decoder(topics, data) });
      } catch (err) {}
    }
    if (res.length) return res;
    return;
  }
}
