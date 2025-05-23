import { R as ResolvedConfig, b as Range, P as Prettify } from './config-edd78478.js';

type Address = ResolvedConfig['AddressType'];
type MBytes = '' | 1 | 2 | 3 | 4 | 5 | 6 | 7 | 8 | 9 | 10 | 11 | 12 | 13 | 14 | 15 | 16 | 17 | 18 | 19 | 20 | 21 | 22 | 23 | 24 | 25 | 26 | 27 | 28 | 29 | 30 | 31 | 32;
type MBits = '' | 8 | 16 | 24 | 32 | 40 | 48 | 56 | 64 | 72 | 80 | 88 | 96 | 104 | 112 | 120 | 128 | 136 | 144 | 152 | 160 | 168 | 176 | 184 | 192 | 200 | 208 | 216 | 224 | 232 | 240 | 248 | 256;
type SolidityAddress = 'address';
type SolidityBool = 'bool';
type SolidityBytes = `bytes${MBytes}`;
type SolidityFunction = 'function';
type SolidityString = 'string';
type SolidityTuple = 'tuple';
type SolidityInt = `${'u' | ''}int${MBits}`;
type SolidityFixedArrayRange = Range<ResolvedConfig['FixedArrayMinLength'], ResolvedConfig['FixedArrayMaxLength']>[number];
type SolidityFixedArraySizeLookup = {
    [Prop in SolidityFixedArrayRange as `${Prop}`]: Prop;
};
/**
 * Recursively build arrays up to maximum depth
 * or use a more broad type when maximum depth is switched "off"
 */
type _BuildArrayTypes<T extends string, Depth extends ReadonlyArray<number> = []> = ResolvedConfig['ArrayMaxDepth'] extends false ? `${T}[${string}]` : Depth['length'] extends ResolvedConfig['ArrayMaxDepth'] ? T : T extends `${any}[${SolidityFixedArrayRange | ''}]` ? _BuildArrayTypes<T | `${T}[${SolidityFixedArrayRange | ''}]`, [...Depth, 1]> : _BuildArrayTypes<`${T}[${SolidityFixedArrayRange | ''}]`, [...Depth, 1]>;
type SolidityArrayWithoutTuple = _BuildArrayTypes<SolidityAddress | SolidityBool | SolidityBytes | SolidityFunction | SolidityInt | SolidityString>;
type SolidityArrayWithTuple = _BuildArrayTypes<SolidityTuple>;
type SolidityArray = SolidityArrayWithoutTuple | SolidityArrayWithTuple;
type AbiType = SolidityArray | SolidityAddress | SolidityBool | SolidityBytes | SolidityFunction | SolidityInt | SolidityString | SolidityTuple;
type ResolvedAbiType = ResolvedConfig['StrictAbiType'] extends true ? AbiType : string;
type AbiInternalType = ResolvedAbiType | `address ${string}` | `contract ${string}` | `enum ${string}` | `struct ${string}`;
type AbiParameter = Prettify<{
    type: ResolvedAbiType;
    name?: string;
    /** Representation used by Solidity compiler */
    internalType?: AbiInternalType;
} & ({
    type: Exclude<ResolvedAbiType, SolidityTuple | SolidityArrayWithTuple>;
} | {
    type: SolidityTuple | SolidityArrayWithTuple;
    components: readonly AbiParameter[];
})>;
/**
 * State mutability for {@link AbiFunction}
 *
 * @see https://docs.soliditylang.org/en/latest/contracts.html#state-mutability
 */
type AbiStateMutability = 'pure' | 'view' | 'nonpayable' | 'payable';
/** Kind of {@link AbiParameter} */
type AbiParameterKind = 'inputs' | 'outputs';
/** ABI ["function"](https://docs.soliditylang.org/en/latest/abi-spec.html#json) type */
type AbiFunction = {
    type: 'function';
    /**
     * @deprecated use `pure` or `view` from {@link AbiStateMutability} instead
     * @see https://github.com/ethereum/solidity/issues/992
     */
    constant?: boolean;
    /**
     * @deprecated Vyper used to provide gas estimates
     * @see https://github.com/vyperlang/vyper/issues/2151
     */
    gas?: number;
    inputs: readonly AbiParameter[];
    name: string;
    outputs: readonly AbiParameter[];
    /**
     * @deprecated use `payable` or `nonpayable` from {@link AbiStateMutability} instead
     * @see https://github.com/ethereum/solidity/issues/992
     */
    payable?: boolean;
    stateMutability: AbiStateMutability;
};
/** ABI ["constructor"](https://docs.soliditylang.org/en/latest/abi-spec.html#json) type */
type AbiConstructor = {
    type: 'constructor';
    inputs: readonly AbiParameter[];
    /**
     * @deprecated use `payable` or `nonpayable` from {@link AbiStateMutability} instead
     * @see https://github.com/ethereum/solidity/issues/992
     */
    payable?: boolean;
    stateMutability: Extract<AbiStateMutability, 'payable' | 'nonpayable'>;
};
/** ABI ["fallback"](https://docs.soliditylang.org/en/latest/abi-spec.html#json) type */
type AbiFallback = {
    type: 'fallback';
    inputs?: readonly [];
    /**
     * @deprecated use `payable` or `nonpayable` from {@link AbiStateMutability} instead
     * @see https://github.com/ethereum/solidity/issues/992
     */
    payable?: boolean;
    stateMutability: Extract<AbiStateMutability, 'payable' | 'nonpayable'>;
};
/** ABI ["receive"](https://docs.soliditylang.org/en/latest/contracts.html#receive-ether-function) type */
type AbiReceive = {
    type: 'receive';
    stateMutability: Extract<AbiStateMutability, 'payable'>;
};
/** ABI ["event"](https://docs.soliditylang.org/en/latest/abi-spec.html#events) type */
type AbiEvent = {
    type: 'event';
    anonymous?: boolean;
    inputs: readonly (AbiParameter & {
        indexed?: boolean;
    })[];
    name: string;
};
/** ABI ["error"](https://docs.soliditylang.org/en/latest/abi-spec.html#errors) type */
type AbiError = {
    type: 'error';
    inputs: readonly AbiParameter[];
    name: string;
};
/** `"type"` name for {@link Abi} items. */
type AbiItemType = 'constructor' | 'error' | 'event' | 'fallback' | 'function' | 'receive';
/**
 * Contract [ABI Specification](https://docs.soliditylang.org/en/latest/abi-spec.html#json)
 */
type Abi = readonly (AbiConstructor | AbiError | AbiEvent | AbiFallback | AbiFunction | AbiReceive)[];
type TypedDataDomain = {
    chainId?: number;
    name?: string;
    salt?: ResolvedConfig['BytesType']['outputs'];
    verifyingContract?: Address;
    version?: string;
};
type TypedDataType = Exclude<AbiType, SolidityFunction | SolidityTuple | SolidityArrayWithTuple | 'int' | 'uint'>;
type TypedDataParameter = {
    name: string;
    type: TypedDataType | keyof TypedData | `${keyof TypedData}[${string | ''}]`;
};
/**
 * [EIP-712](https://eips.ethereum.org/EIPS/eip-712#definition-of-typed-structured-data-%F0%9D%95%8A) Typed Data Specification
 */
type TypedData = Prettify<{
    [key: string]: readonly TypedDataParameter[];
} & {
    [_ in TypedDataType]?: never;
}>;

export { AbiType as A, TypedDataDomain as B, MBits as M, SolidityAddress as S, TypedData as T, AbiParameterKind as a, SolidityBool as b, SolidityBytes as c, SolidityFunction as d, SolidityInt as e, SolidityString as f, SolidityTuple as g, SolidityArray as h, AbiParameter as i, SolidityFixedArrayRange as j, SolidityFixedArraySizeLookup as k, Abi as l, AbiStateMutability as m, TypedDataType as n, TypedDataParameter as o, AbiConstructor as p, AbiError as q, AbiEvent as r, AbiFallback as s, AbiFunction as t, AbiInternalType as u, AbiItemType as v, AbiReceive as w, Address as x, SolidityArrayWithoutTuple as y, SolidityArrayWithTuple as z };
