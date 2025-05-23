import * as P from 'micro-packed';
import { type IWeb3Provider } from '../utils.ts';
type Writable<T> = T extends {} ? {
    -readonly [P in keyof T]: Writable<T[P]>;
} : T;
type ArrLike<T> = Array<T> | ReadonlyArray<T>;
export type IsEmptyArray<T> = T extends ReadonlyArray<any> ? (T['length'] extends 0 ? true : false) : true;
export type Component<T extends string> = {
    readonly name?: string;
    readonly type: T;
};
export type NamedComponent<T extends string> = Component<T> & {
    readonly name: string;
};
export type BaseComponent = Component<string>;
export type Tuple<TC extends ArrLike<Component<string>>> = {
    readonly name?: string;
    readonly type: 'tuple';
    readonly components: TC;
};
type IntIdxType = '' | '8' | '16' | '24' | '32' | '40' | '48' | '56' | '64' | '72' | '80' | '88' | '96' | '104' | '112' | '120' | '128' | '136' | '144' | '152' | '160' | '168' | '176' | '184' | '192' | '200' | '208' | '216' | '224' | '232' | '240' | '248' | '256';
type UintType = `uint${IntIdxType}`;
type IntType = `int${IntIdxType}`;
type NumberType = UintType | IntType;
type ByteIdxType = '' | '1' | '2' | '3' | '4' | '5' | '6' | '7' | '8' | '9' | '10' | '11' | '12' | '13' | '14' | '15' | '16' | '17' | '18' | '19' | '20' | '21' | '22' | '23' | '24' | '25' | '26' | '27' | '28' | '29' | '30' | '31' | '32';
type ByteType = `bytes${ByteIdxType}`;
export type MapTuple<T> = T extends ArrLike<Component<string> & {
    name: string;
}> ? {
    [K in T[number] as K['name']]: MapType<K>;
} : T extends ArrLike<Component<string>> ? {
    [K in keyof T]: T[K] extends BaseComponent ? MapType<T[K]> : unknown;
} : unknown;
export type GetType<T extends string> = T extends `${infer Base}[]${infer Rest}` ? GetType<`${Base}${Rest}`>[] : T extends `${infer Base}[${number}]${infer Rest}` ? GetType<`${Base}${Rest}`>[] : T extends 'address' ? string : T extends 'string' ? string : T extends 'bool' ? boolean : T extends NumberType ? bigint : T extends ByteType ? Uint8Array : unknown;
export type MapType<T extends BaseComponent> = T extends Tuple<Array<Component<string>>> ? MapTuple<T['components']> : T extends Component<infer Type> ? GetType<Type> : unknown;
export type UnmapType<T> = T extends MapType<infer U> ? U : never;
export declare function mapComponent<T extends BaseComponent>(c: T): P.CoderType<MapType<Writable<T>>>;
export type ArgsType<T extends ReadonlyArray<any> | undefined> = IsEmptyArray<T> extends true ? undefined : T extends ReadonlyArray<any> ? T['length'] extends 1 ? MapType<T[0]> : MapTuple<T> : MapTuple<T>;
export declare function mapArgs<T extends ArrLike<Component<string>>>(args: T): P.CoderType<ArgsType<Writable<T>>>;
export type FunctionType = Component<'function'> & {
    readonly inputs?: ReadonlyArray<Component<string>>;
    readonly outputs?: ReadonlyArray<Component<string>>;
};
type ContractMethodDecode<T extends FunctionType, O = ArgsType<T['outputs']>> = IsEmptyArray<T['outputs']> extends true ? {
    decodeOutput: (b: Uint8Array) => void;
} : {
    decodeOutput: (b: Uint8Array) => O;
};
type ContractMethodEncode<T extends FunctionType, I = ArgsType<T['inputs']>> = IsEmptyArray<T['inputs']> extends true ? {
    encodeInput: () => Uint8Array;
} : {
    encodeInput: (v: I) => Uint8Array;
};
type ContractMethodGas<T extends FunctionType, I = ArgsType<T['inputs']>> = IsEmptyArray<T['inputs']> extends true ? {
    estimateGas: () => Promise<bigint>;
} : {
    estimateGas: (v: I) => Promise<bigint>;
};
type ContractMethodCall<T extends FunctionType, I = ArgsType<T['inputs']>, O = ArgsType<T['outputs']>> = IsEmptyArray<T['inputs']> extends true ? IsEmptyArray<T['outputs']> extends true ? {
    call: () => Promise<void>;
} : {
    call: () => Promise<O>;
} : IsEmptyArray<T['outputs']> extends true ? {
    call: (v: I) => Promise<void>;
} : {
    call: (v: I) => Promise<O>;
};
export type ContractMethod<T extends FunctionType> = ContractMethodEncode<T> & ContractMethodDecode<T>;
export type ContractMethodNet<T extends FunctionType> = ContractMethod<T> & ContractMethodGas<T> & ContractMethodCall<T>;
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
    [K in keyof T]: T[K] extends FunctionType & {
        name: string;
    } ? T[K] : never;
};
export type ContractType<T extends Array<FnArg>, N, F = ContractTypeFilter<T>> = F extends ArrLike<FunctionType & {
    name: string;
}> ? {
    [K in F[number] as K['name']]: N extends IWeb3Provider ? ContractMethodNet<K> : ContractMethod<K>;
} : never;
export declare function evSigHash(o: FnArg): string;
export declare function fnSigHash(o: FnArg): string;
export declare function createContract<T extends ArrLike<FnArg>>(abi: T, net: IWeb3Provider, contract?: string): ContractType<Writable<T>, IWeb3Provider>;
export declare function createContract<T extends ArrLike<FnArg>>(abi: T, net?: undefined, contract?: string): ContractType<Writable<T>, undefined>;
type GetCons<T extends ArrLike<FnArg>> = Extract<T[number], {
    type: 'constructor';
}>;
type ConstructorType = Component<'constructor'> & {
    readonly inputs?: ReadonlyArray<Component<string>>;
};
type ConsArgs<T extends ConstructorType> = IsEmptyArray<T['inputs']> extends true ? undefined : ArgsType<T['inputs']>;
export declare function deployContract<T extends ArrLike<FnArg>>(abi: T, bytecodeHex: string, ...args: GetCons<T> extends never ? [args: unknown] : ConsArgs<GetCons<T>> extends undefined ? [] : [args: ConsArgs<GetCons<T>>]): string;
export type EventType = NamedComponent<'event'> & {
    readonly inputs: ReadonlyArray<Component<string>>;
};
export type ContractEventTypeFilter<T> = {
    [K in keyof T]: T[K] extends EventType ? T[K] : never;
};
export type TopicsValue<T> = {
    [K in keyof T]: T[K] | null;
};
export type EventMethod<T extends EventType> = {
    decode: (topics: string[], data: string) => ArgsType<T['inputs']>;
    topics: (values: TopicsValue<ArgsType<T['inputs']>>) => (string | null)[];
};
export type ContractEventType<T extends Array<FnArg>, F = ContractEventTypeFilter<T>> = F extends ArrLike<EventType> ? {
    [K in F[number] as K['name']]: EventMethod<K>;
} : never;
export declare function events<T extends ArrLike<FnArg>>(abi: T): ContractEventType<Writable<T>>;
export type ContractABI = ReadonlyArray<FnArg & {
    readonly hint?: HintFn;
    readonly hook?: HookFn;
}>;
export type ContractInfo = {
    abi: 'ERC20' | 'ERC721' | 'ERC1155' | ContractABI;
    symbol?: string;
    decimals?: number;
    name?: string;
    price?: number;
};
export type HintOpt = {
    contract?: string;
    amount?: bigint;
    contractInfo?: ContractInfo;
    contracts?: Record<string, ContractInfo>;
};
export type HintFn = (value: unknown, opt: HintOpt) => string;
export type HookFn = (decoder: Decoder, contract: string, info: SignatureInfo, opt: HintOpt) => SignatureInfo;
type SignaturePacker = {
    name: string;
    signature: string;
    packer: P.CoderType<unknown>;
    hint?: HintFn;
    hook?: HookFn;
};
type EventSignatureDecoder = {
    name: string;
    signature: string;
    decoder: (topics: string[], _data: string) => unknown;
    hint?: HintFn;
};
export type SignatureInfo = {
    name: string;
    signature: string;
    value: unknown;
    hint?: string;
};
export declare class Decoder {
    contracts: Record<string, Record<string, SignaturePacker>>;
    sighashes: Record<string, SignaturePacker[]>;
    evContracts: Record<string, Record<string, EventSignatureDecoder>>;
    evSighashes: Record<string, EventSignatureDecoder[]>;
    add(contract: string, abi: ContractABI): void;
    method(contract: string, data: Uint8Array): string | undefined;
    decode(contract: string, _data: Uint8Array, opt: HintOpt): SignatureInfo | SignatureInfo[] | undefined;
    decodeEvent(contract: string, topics: string[], data: string, opt: HintOpt): SignatureInfo | SignatureInfo[] | undefined;
}
export {};
//# sourceMappingURL=decoder.d.ts.map