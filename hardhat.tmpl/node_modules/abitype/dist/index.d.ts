import { A as AbiType, a as AbiParameterKind, S as SolidityAddress, b as SolidityBool, c as SolidityBytes, d as SolidityFunction, e as SolidityInt, f as SolidityString, g as SolidityTuple, h as SolidityArray, i as AbiParameter, j as SolidityFixedArrayRange, k as SolidityFixedArraySizeLookup, l as Abi, m as AbiStateMutability, T as TypedData, n as TypedDataType, o as TypedDataParameter, M as MBits } from './abi-7aa1f183.js';
export { l as Abi, p as AbiConstructor, q as AbiError, r as AbiEvent, s as AbiFallback, t as AbiFunction, u as AbiInternalType, v as AbiItemType, i as AbiParameter, a as AbiParameterKind, w as AbiReceive, m as AbiStateMutability, A as AbiType, x as Address, S as SolidityAddress, h as SolidityArray, z as SolidityArrayWithTuple, y as SolidityArrayWithoutTuple, b as SolidityBool, c as SolidityBytes, j as SolidityFixedArrayRange, k as SolidityFixedArraySizeLookup, d as SolidityFunction, e as SolidityInt, f as SolidityString, g as SolidityTuple, T as TypedData, B as TypedDataDomain, o as TypedDataParameter, n as TypedDataType } from './abi-7aa1f183.js';
import { R as ResolvedConfig, E as Error$1, T as Tuple, M as Merge, a as Trim, I as IsUnknown, P as Prettify, F as Filter } from './config-edd78478.js';
export { C as Config, D as DefaultConfig, R as ResolvedConfig } from './config-edd78478.js';

type BaseErrorArgs = {
    docsPath?: string;
    metaMessages?: string[];
} & ({
    cause?: never;
    details?: string;
} | {
    cause: BaseError | Error;
    details?: never;
});
declare class BaseError extends Error {
    details: string;
    docsPath?: string;
    metaMessages?: string[];
    shortMessage: string;
    name: string;
    constructor(shortMessage: string, args?: BaseErrorArgs);
}

/**
 * Infers embedded primitive type of any type
 *
 * @param T - Type to infer
 * @returns Embedded type of {@link TType}
 *
 * @example
 * type Result = Narrow<['foo', 'bar', 1]>
 */
type Narrow<TType> = (TType extends Function ? TType : never) | (TType extends string | number | boolean | bigint ? TType : never) | (TType extends [] ? [] : never) | {
    [K in keyof TType]: Narrow<TType[K]>;
};
/**
 * Infers embedded primitive type of any type
 * Same as `as const` but without setting the object as readonly and without needing the user to use it.
 *
 * @param value - Value to infer
 * @returns Value with embedded type inferred
 *
 * @example
 * const result = narrow(['foo', 'bar', 1])
 */
declare function narrow<TType>(value: Narrow<TType>): Narrow<TType>;

/**
 * Converts {@link AbiType} to corresponding TypeScript primitive type.
 *
 * Does not include full array or tuple conversion. Use {@link AbiParameterToPrimitiveType} to fully convert arrays and tuples.
 *
 * @param TAbiType - {@link AbiType} to convert to TypeScript representation
 * @param TAbiParameterKind - Optional {@link AbiParameterKind} to narrow by parameter type
 * @returns TypeScript primitive type
 */
type AbiTypeToPrimitiveType<TAbiType extends AbiType, TAbiParameterKind extends AbiParameterKind = AbiParameterKind> = PrimitiveTypeLookup<TAbiType, TAbiParameterKind>[TAbiType];
type PrimitiveTypeLookup<TAbiType extends AbiType, TAbiParameterKind extends AbiParameterKind = AbiParameterKind> = {
    [_ in SolidityAddress]: ResolvedConfig['AddressType'];
} & {
    [_ in SolidityBool]: boolean;
} & {
    [_ in SolidityBytes]: ResolvedConfig['BytesType'][TAbiParameterKind];
} & {
    [_ in SolidityFunction]: `${ResolvedConfig['AddressType']}${string}`;
} & {
    [_ in SolidityInt]: TAbiType extends `${'u' | ''}int${infer TBits}` ? TBits extends keyof BitsTypeLookup ? BitsTypeLookup[TBits] : Error$1<'Unknown bits value.'> : Error$1<`Unknown 'SolidityInt' format.`>;
} & {
    [_ in SolidityString]: string;
} & {
    [_ in SolidityTuple]: Record<string, unknown>;
} & {
    [_ in SolidityArray]: readonly unknown[];
};
type GreaterThan48Bits = Exclude<MBits, 8 | 16 | 24 | 32 | 40 | 48 | ''>;
type LessThanOrEqualTo48Bits = Exclude<MBits, GreaterThan48Bits | ''>;
type NoBits = Exclude<MBits, GreaterThan48Bits | LessThanOrEqualTo48Bits>;
type BitsTypeLookup = {
    [_ in `${LessThanOrEqualTo48Bits}`]: ResolvedConfig['IntType'];
} & {
    [_ in `${GreaterThan48Bits}`]: ResolvedConfig['BigIntType'];
} & {
    [_ in NoBits]: ResolvedConfig['BigIntType'];
};
/**
 * Converts {@link AbiParameter} to corresponding TypeScript primitive type.
 *
 * @param TAbiParameter - {@link AbiParameter} to convert to TypeScript representation
 * @param TAbiParameterKind - Optional {@link AbiParameterKind} to narrow by parameter type
 * @returns TypeScript primitive type
 */
type AbiParameterToPrimitiveType<TAbiParameter extends AbiParameter | {
    name: string;
    type: unknown;
}, TAbiParameterKind extends AbiParameterKind = AbiParameterKind> = TAbiParameter['type'] extends Exclude<AbiType, SolidityTuple | SolidityArray> ? AbiTypeToPrimitiveType<TAbiParameter['type'], TAbiParameterKind> : TAbiParameter extends {
    type: SolidityTuple;
    components: infer TComponents extends readonly AbiParameter[];
} ? TComponents extends readonly [] ? [] : _HasUnnamedAbiParameter<TComponents> extends true ? readonly [
    ...{
        [K in keyof TComponents]: AbiParameterToPrimitiveType<TComponents[K], TAbiParameterKind>;
    }
] : {
    [Component in TComponents[number] as Component extends {
        name: string;
    } ? Component['name'] : never]: AbiParameterToPrimitiveType<Component, TAbiParameterKind>;
} : 
/**
 * First, infer `Head` against a known size type (either fixed-length array value or `""`).
 *
 * | Input           | Head         |
 * | --------------- | ------------ |
 * | `string[]`      | `string`     |
 * | `string[][][3]` | `string[][]` |
 */
TAbiParameter['type'] extends `${infer Head}[${'' | `${SolidityFixedArrayRange}`}]` ? TAbiParameter['type'] extends `${Head}[${infer Size}]` ? Size extends keyof SolidityFixedArraySizeLookup ? Tuple<AbiParameterToPrimitiveType<Merge<TAbiParameter, {
    type: Head;
}>, TAbiParameterKind>, SolidityFixedArraySizeLookup[Size]> : readonly AbiParameterToPrimitiveType<Merge<TAbiParameter, {
    type: Head;
}>, TAbiParameterKind>[] : never : ResolvedConfig['StrictAbiType'] extends true ? TAbiParameter['type'] extends infer TAbiType extends string ? Error$1<`Unknown type '${TAbiType}'.`> : never : unknown;
type _HasUnnamedAbiParameter<TAbiParameters extends readonly AbiParameter[]> = TAbiParameters extends readonly [
    infer Head extends AbiParameter,
    ...infer Tail extends readonly AbiParameter[]
] ? Head extends {
    name: string;
} ? Head['name'] extends '' ? true : _HasUnnamedAbiParameter<Tail> : true : false;
/**
 * Converts array of {@link AbiParameter} to corresponding TypeScript primitive types.
 *
 * @param TAbiParameters - Array of {@link AbiParameter} to convert to TypeScript representations
 * @param TAbiParameterKind - Optional {@link AbiParameterKind} to narrow by parameter type
 * @returns Array of TypeScript primitive types
 */
type AbiParametersToPrimitiveTypes<TAbiParameters extends readonly AbiParameter[], TAbiParameterKind extends AbiParameterKind = AbiParameterKind> = {
    [K in keyof TAbiParameters]: AbiParameterToPrimitiveType<TAbiParameters[K], TAbiParameterKind>;
};
/**
 * Checks if type is {@link Abi}.
 *
 * @param TAbi - {@link Abi} to check
 * @returns Boolean for whether {@link TAbi} is {@link Abi}
 */
type IsAbi<TAbi> = TAbi extends Abi ? true : false;
/**
 * Extracts all {@link AbiFunction} types from {@link Abi}.
 *
 * @param TAbi - {@link Abi} to extract functions from
 * @param TAbiStateMutibility - {@link AbiStateMutability} to filter by
 * @returns All {@link AbiFunction} types from {@link Abi}
 */
type ExtractAbiFunctions<TAbi extends Abi, TAbiStateMutibility extends AbiStateMutability = AbiStateMutability> = Extract<TAbi[number], {
    type: 'function';
    stateMutability: TAbiStateMutibility;
}>;
/**
 * Extracts all {@link AbiFunction} names from {@link Abi}.
 *
 * @param TAbi - {@link Abi} to extract function names from
 * @param TAbiStateMutibility - {@link AbiStateMutability} to filter by
 * @returns Union of function names
 */
type ExtractAbiFunctionNames<TAbi extends Abi, TAbiStateMutibility extends AbiStateMutability = AbiStateMutability> = ExtractAbiFunctions<TAbi, TAbiStateMutibility>['name'];
/**
 * Extracts {@link AbiFunction} with name from {@link Abi}.
 *
 * @param TAbi - {@link Abi} to extract {@link AbiFunction} from
 * @param TFunctionName - String name of function to extract from {@link Abi}
 * @returns Matching {@link AbiFunction}
 */
type ExtractAbiFunction<TAbi extends Abi, TFunctionName extends ExtractAbiFunctionNames<TAbi>> = Extract<ExtractAbiFunctions<TAbi>, {
    name: TFunctionName;
}>;
/**
 * Extracts all {@link AbiEvent} types from {@link Abi}.
 *
 * @param TAbi - {@link Abi} to extract events from
 * @returns All {@link AbiEvent} types from {@link Abi}
 */
type ExtractAbiEvents<TAbi extends Abi> = Extract<TAbi[number], {
    type: 'event';
}>;
/**
 * Extracts all {@link AbiEvent} names from {@link Abi}.
 *
 * @param TAbi - {@link Abi} to extract event names from
 * @returns Union of event names
 */
type ExtractAbiEventNames<TAbi extends Abi> = ExtractAbiEvents<TAbi>['name'];
/**
 * Extracts {@link AbiEvent} with name from {@link Abi}.
 *
 * @param TAbi - {@link Abi} to extract {@link AbiEvent} from
 * @param TEventName - String name of event to extract from {@link Abi}
 * @returns Matching {@link AbiEvent}
 */
type ExtractAbiEvent<TAbi extends Abi, TEventName extends ExtractAbiEventNames<TAbi>> = Extract<ExtractAbiEvents<TAbi>, {
    name: TEventName;
}>;
/**
 * Extracts all {@link AbiError} types from {@link Abi}.
 *
 * @param TAbi - {@link Abi} to extract errors from
 * @returns All {@link AbiError} types from {@link Abi}
 */
type ExtractAbiErrors<TAbi extends Abi> = Extract<TAbi[number], {
    type: 'error';
}>;
/**
 * Extracts all {@link AbiError} names from {@link Abi}.
 *
 * @param TAbi - {@link Abi} to extract error names from
 * @returns Union of error names
 */
type ExtractAbiErrorNames<TAbi extends Abi> = ExtractAbiErrors<TAbi>['name'];
/**
 * Extracts {@link AbiError} with name from {@link Abi}.
 *
 * @param TAbi - {@link Abi} to extract {@link AbiError} from
 * @param TErrorName - String name of error to extract from {@link Abi}
 * @returns Matching {@link AbiError}
 */
type ExtractAbiError<TAbi extends Abi, TErrorName extends ExtractAbiErrorNames<TAbi>> = Extract<ExtractAbiErrors<TAbi>, {
    name: TErrorName;
}>;
/**
 * Checks if type is {@link TypedData}.
 *
 * @param TTypedData - {@link TypedData} to check
 * @returns Boolean for whether {@link TTypedData} is {@link TypedData}
 */
type IsTypedData<TTypedData> = TTypedData extends TypedData ? {
    [K in keyof TTypedData]: {
        [K2 in TTypedData[K][number] as K2['type'] extends keyof TTypedData ? never : K2['type'] extends `${keyof TTypedData & string}[${string}]` ? never : K2['type'] extends TypedDataType ? never : K2['name']]: false;
    };
} extends {
    [K in keyof TTypedData]: Record<string, never>;
} ? true : false : false;
/**
 * Converts {@link TTypedData} to corresponding TypeScript primitive types.
 *
 * @param TTypedData - {@link TypedData} to convert
 * @param TAbiParameterKind - Optional {@link AbiParameterKind} to narrow by parameter type
 * @returns Union of TypeScript primitive types
 */
type TypedDataToPrimitiveTypes<TTypedData extends TypedData, TAbiParameterKind extends AbiParameterKind = AbiParameterKind, TKeyReferences extends {
    [_: string]: unknown;
} | unknown = unknown> = {
    [K in keyof TTypedData]: {
        [K2 in TTypedData[K][number] as K2['name']]: K2['type'] extends K ? Error$1<`Cannot convert self-referencing struct '${K2['type']}' to primitive type.`> : K2['type'] extends keyof TTypedData ? K2['type'] extends keyof TKeyReferences ? Error$1<`Circular reference detected. '${K2['type']}' is a circular reference.`> : TypedDataToPrimitiveTypes<Exclude<TTypedData, K>, TAbiParameterKind, TKeyReferences & {
            [_ in K2['type']]: true;
        }>[K2['type']] : K2['type'] extends `${infer TType extends keyof TTypedData & string}[${infer Tail}]` ? AbiParameterToPrimitiveType<Merge<K2, {
            type: `tuple[${Tail}]`;
            components: _TypedDataParametersToAbiParameters<TTypedData[TType], TTypedData>;
        }>, TAbiParameterKind> : K2['type'] extends TypedDataType ? AbiParameterToPrimitiveType<K2, TAbiParameterKind> : Error$1<`Cannot convert unknown type '${K2['type']}' to primitive type.`>;
    };
};
type _TypedDataParametersToAbiParameters<TTypedDataParameters extends readonly TypedDataParameter[], TTypedData extends TypedData> = {
    [K in keyof TTypedDataParameters]: TTypedDataParameters[K] extends infer TTypedDataParameter extends {
        name: string;
        type: unknown;
    } ? TTypedDataParameter['type'] extends keyof TTypedData ? Merge<TTypedDataParameter, {
        type: 'tuple';
        components: _TypedDataParametersToAbiParameters<TTypedData[TTypedDataParameter['type']], TTypedData>;
    }> : TTypedDataParameter['type'] extends `${infer TType extends keyof TTypedData & string}[${infer Tail}]` ? Merge<TTypedDataParameter, {
        type: `tuple[${Tail}]`;
        components: _TypedDataParametersToAbiParameters<TTypedData[TType], TTypedData>;
    }> : TTypedDataParameter : never;
};

type ErrorSignature<TName extends string = string, TParameters extends string = string> = `error ${TName}(${TParameters})`;
type IsErrorSignature<T extends string> = T extends ErrorSignature<infer Name> ? IsName<Name> : false;
type EventSignature<TName extends string = string, TParameters extends string = string> = `event ${TName}(${TParameters})`;
type IsEventSignature<T extends string> = T extends EventSignature<infer Name> ? IsName<Name> : false;
type FunctionSignature<TName extends string = string, TTail extends string = string> = `function ${TName}(${TTail}`;
type IsFunctionSignature<T> = T extends FunctionSignature<infer Name> ? IsName<Name> extends true ? T extends ValidFunctionSignatures ? true : T extends `function ${string}(${infer Parameters})` ? Parameters extends InvalidFunctionParameters ? false : true : false : false : false;
type Scope = 'public' | 'external';
type Returns = `returns (${string})`;
type ValidFunctionSignatures = `function ${string}()` | `function ${string}() ${Returns}` | `function ${string}() ${AbiStateMutability}` | `function ${string}() ${Scope}` | `function ${string}() ${AbiStateMutability} ${Returns}` | `function ${string}() ${Scope} ${Returns}` | `function ${string}() ${Scope} ${AbiStateMutability}` | `function ${string}() ${Scope} ${AbiStateMutability} ${Returns}` | `function ${string}(${string}) ${Returns}` | `function ${string}(${string}) ${AbiStateMutability}` | `function ${string}(${string}) ${Scope}` | `function ${string}(${string}) ${AbiStateMutability} ${Returns}` | `function ${string}(${string}) ${Scope} ${Returns}` | `function ${string}(${string}) ${Scope} ${AbiStateMutability}` | `function ${string}(${string}) ${Scope} ${AbiStateMutability} ${Returns}`;
type StructSignature<TName extends string = string, TProperties extends string = string> = `struct ${TName} {${TProperties}}`;
type IsStructSignature<T extends string> = T extends StructSignature<infer Name> ? IsName<Name> : false;
type ConstructorSignature<TTail extends string = string> = `constructor(${TTail}`;
type IsConstructorSignature<T> = T extends ConstructorSignature ? T extends ValidConstructorSignatures ? true : false : false;
type ValidConstructorSignatures = `constructor(${string})` | `constructor(${string}) payable`;
type FallbackSignature<TAbiStateMutability extends '' | ' payable' = ''> = `fallback() external${TAbiStateMutability}`;
type ReceiveSignature = 'receive() external payable';
type IsSignature<T extends string> = (IsErrorSignature<T> extends true ? true : never) | (IsEventSignature<T> extends true ? true : never) | (IsFunctionSignature<T> extends true ? true : never) | (IsStructSignature<T> extends true ? true : never) | (IsConstructorSignature<T> extends true ? true : never) | (T extends FallbackSignature ? true : never) | (T extends ReceiveSignature ? true : never) extends infer Condition ? [Condition] extends [never] ? false : true : false;
type Signature<T extends string, K extends string | unknown = unknown> = IsSignature<T> extends true ? T : Error$1<`Signature "${T}" is invalid${K extends string ? ` at position ${K}` : ''}.`>;
type Signatures<T extends readonly string[]> = {
    [K in keyof T]: Signature<T[K], K>;
};
type Modifier = 'calldata' | 'indexed' | 'memory' | 'storage';
type FunctionModifier = Extract<Modifier, 'calldata' | 'memory' | 'storage'>;
type EventModifier = Extract<Modifier, 'indexed'>;
type IsName<TName extends string> = TName extends '' ? false : ValidateName<TName> extends TName ? true : false;
type ValidateName<TName extends string, CheckCharacters extends boolean = false> = TName extends `${string}${' '}${string}` ? Error$1<`Name "${TName}" cannot contain whitespace.`> : IsSolidityKeyword<TName> extends true ? Error$1<`"${TName}" is a protected Solidity keyword.`> : CheckCharacters extends true ? IsValidCharacter<TName> extends true ? TName : Error$1<`"${TName}" contains invalid character.`> : TName;
type IsSolidityKeyword<T extends string> = T extends SolidityKeywords ? true : false;
type SolidityKeywords = 'after' | 'alias' | 'anonymous' | 'apply' | 'auto' | 'byte' | 'calldata' | 'case' | 'catch' | 'constant' | 'copyof' | 'default' | 'defined' | 'error' | 'event' | 'external' | 'false' | 'final' | 'function' | 'immutable' | 'implements' | 'in' | 'indexed' | 'inline' | 'internal' | 'let' | 'mapping' | 'match' | 'memory' | 'mutable' | 'null' | 'of' | 'override' | 'partial' | 'private' | 'promise' | 'public' | 'pure' | 'reference' | 'relocatable' | 'return' | 'returns' | 'sizeof' | 'static' | 'storage' | 'struct' | 'super' | 'supports' | 'switch' | 'this' | 'true' | 'try' | 'typedef' | 'typeof' | 'var' | 'view' | 'virtual';
type IsValidCharacter<T extends string> = T extends `${ValidCharacters}${infer Tail}` ? Tail extends '' ? true : IsValidCharacter<Tail> : false;
type ValidCharacters = 'A' | 'B' | 'C' | 'D' | 'E' | 'F' | 'G' | 'H' | 'I' | 'J' | 'K' | 'L' | 'M' | 'N' | 'O' | 'P' | 'Q' | 'R' | 'S' | 'T' | 'U' | 'V' | 'W' | 'X' | 'Y' | 'Z' | 'a' | 'b' | 'c' | 'd' | 'e' | 'f' | 'g' | 'h' | 'i' | 'j' | 'k' | 'l' | 'm' | 'n' | 'o' | 'p' | 'q' | 'r' | 's' | 't' | 'u' | 'v' | 'w' | 'x' | 'y' | 'z' | '0' | '1' | '2' | '3' | '4' | '5' | '6' | '7' | '8' | '9' | '_';
type InvalidFunctionParameters = `${string}${MangledReturns} (${string}` | `${string}) ${MangledReturns}${string}` | `${string})${string}${MangledReturns}${string}(${string}`;
type MangledReturns = `r${string}eturns` | `re${string}turns` | `ret${string}urns` | `retu${string}rns` | `retur${string}ns` | `return${string}s` | `r${string}e${string}turns` | `r${string}et${string}urns` | `r${string}etu${string}rns` | `r${string}etur${string}ns` | `r${string}eturn${string}s` | `re${string}t${string}urns` | `re${string}tu${string}rns` | `re${string}tur${string}ns` | `re${string}turn${string}s` | `ret${string}u${string}rns` | `ret${string}ur${string}ns` | `ret${string}urn${string}s` | `retu${string}r${string}ns` | `retu${string}rn${string}s` | `retur${string}n${string}s` | `r${string}e${string}t${string}urns` | `r${string}e${string}tu${string}rns` | `r${string}e${string}tur${string}ns` | `r${string}e${string}turn${string}s` | `re${string}t${string}u${string}rns` | `re${string}t${string}ur${string}ns` | `re${string}t${string}urn${string}s` | `ret${string}u${string}r${string}ns` | `ret${string}u${string}rn${string}s` | `retu${string}r${string}n${string}s` | `r${string}e${string}t${string}u${string}rns` | `r${string}e${string}t${string}ur${string}ns` | `r${string}e${string}t${string}urn${string}s` | `re${string}t${string}u${string}r${string}ns` | `re${string}t${string}u${string}rn${string}s` | `ret${string}u${string}r${string}n${string}s` | `r${string}e${string}t${string}u${string}r${string}ns` | `r${string}e${string}t${string}u${string}rn${string}s` | `re${string}t${string}u${string}r${string}n${string}s` | `r${string}e${string}t${string}u${string}r${string}n${string}s`;

type ParseSignature<TSignature extends string, TStructs extends StructLookup | unknown = unknown> = (IsErrorSignature<TSignature> extends true ? TSignature extends ErrorSignature<infer Name, infer Parameters> ? {
    readonly name: Name;
    readonly type: 'error';
    readonly inputs: ParseAbiParameters$1<SplitParameters<Parameters>, {
        Structs: TStructs;
    }>;
} : never : never) | (IsEventSignature<TSignature> extends true ? TSignature extends EventSignature<infer Name, infer Parameters> ? {
    readonly name: Name;
    readonly type: 'event';
    readonly inputs: ParseAbiParameters$1<SplitParameters<Parameters>, {
        Modifier: EventModifier;
        Structs: TStructs;
    }>;
} : never : never) | (IsFunctionSignature<TSignature> extends true ? TSignature extends FunctionSignature<infer Name, infer Tail> ? {
    readonly name: Name;
    readonly type: 'function';
    readonly stateMutability: _ParseFunctionParametersAndStateMutability<TSignature>['StateMutability'];
    readonly inputs: ParseAbiParameters$1<SplitParameters<_ParseFunctionParametersAndStateMutability<TSignature>['Inputs']>, {
        Modifier: FunctionModifier;
        Structs: TStructs;
    }>;
    readonly outputs: Tail extends `${string}returns (${infer Returns})` ? ParseAbiParameters$1<SplitParameters<Returns>, {
        Modifier: FunctionModifier;
        Structs: TStructs;
    }> : readonly [];
} : never : never) | (IsConstructorSignature<TSignature> extends true ? {
    readonly type: 'constructor';
    readonly stateMutability: _ParseConstructorParametersAndStateMutability<TSignature>['StateMutability'];
    readonly inputs: ParseAbiParameters$1<SplitParameters<_ParseConstructorParametersAndStateMutability<TSignature>['Inputs']>, {
        Structs: TStructs;
    }>;
} : never) | (TSignature extends FallbackSignature<infer StateMutability> ? {
    readonly type: 'fallback';
    readonly stateMutability: StateMutability extends `${string}payable` ? 'payable' : 'nonpayable';
} : never) | (TSignature extends ReceiveSignature ? {
    readonly type: 'receive';
    readonly stateMutability: 'payable';
} : never);
type ParseOptions = {
    Modifier?: Modifier;
    Structs?: StructLookup | unknown;
};
type DefaultParseOptions = object;
type ParseAbiParameters$1<T extends readonly string[], Options extends ParseOptions = DefaultParseOptions> = T extends [''] ? readonly [] : readonly [
    ...{
        [K in keyof T]: ParseAbiParameter$1<T[K], Options>;
    }
];
type ParseAbiParameter$1<T extends string, Options extends ParseOptions = DefaultParseOptions> = (T extends `(${string})${string}` ? _ParseTuple<T, Options> : T extends `${infer Type} ${infer Tail}` ? Trim<Tail> extends infer Trimmed extends string ? // TODO: data location modifiers only allowed for struct/array types
{
    readonly type: Trim<Type>;
} & _SplitNameOrModifier<Trimmed, Options> : never : {
    readonly type: T;
}) extends infer ShallowParameter extends AbiParameter & {
    type: string;
    indexed?: boolean;
} ? (ShallowParameter['type'] extends keyof Options['Structs'] ? {
    readonly type: 'tuple';
    readonly components: Options['Structs'][ShallowParameter['type']];
} & (IsUnknown<ShallowParameter['name']> extends false ? {
    readonly name: ShallowParameter['name'];
} : object) & (ShallowParameter['indexed'] extends true ? {
    readonly indexed: true;
} : object) : ShallowParameter['type'] extends `${infer Type extends string & keyof Options['Structs']}[${infer Tail}]` ? {
    readonly type: `tuple[${Tail}]`;
    readonly components: Options['Structs'][Type];
} & (IsUnknown<ShallowParameter['name']> extends false ? {
    readonly name: ShallowParameter['name'];
} : object) & (ShallowParameter['indexed'] extends true ? {
    readonly indexed: true;
} : object) : ShallowParameter) extends infer Parameter extends AbiParameter & {
    type: string;
    indexed?: boolean;
} ? Prettify<_ValidateAbiParameter<Parameter>> : never : never;
type SplitParameters<T extends string, Result extends unknown[] = [], Current extends string = '', Depth extends ReadonlyArray<number> = []> = T extends '' ? Current extends '' ? [...Result] : Depth['length'] extends 0 ? [...Result, Trim<Current>] : Error$1<`Unbalanced parentheses. "${Current}" has too many opening parentheses.`> : T extends `${infer Char}${infer Tail}` ? Char extends ',' ? Depth['length'] extends 0 ? SplitParameters<Tail, [...Result, Trim<Current>], ''> : SplitParameters<Tail, Result, `${Current}${Char}`, Depth> : Char extends '(' ? SplitParameters<Tail, Result, `${Current}${Char}`, [...Depth, 1]> : Char extends ')' ? Depth['length'] extends 0 ? Error$1<`Unbalanced parentheses. "${Current}" has too many closing parentheses.`> : SplitParameters<Tail, Result, `${Current}${Char}`, Pop<Depth>> : SplitParameters<Tail, Result, `${Current}${Char}`, Depth> : [];
type Pop<T extends ReadonlyArray<number>> = T extends [...infer R, any] ? R : [];
type _ValidateAbiParameter<TAbiParameter extends AbiParameter> = (TAbiParameter extends {
    name: string;
} ? ValidateName<TAbiParameter['name']> extends infer Name ? Name extends TAbiParameter['name'] ? TAbiParameter : Merge<TAbiParameter, {
    readonly name: Name;
}> : never : TAbiParameter) extends infer Parameter ? (ResolvedConfig['StrictAbiType'] extends true ? Parameter extends {
    type: AbiType;
} ? Parameter : Merge<Parameter, {
    readonly type: Error$1<`Type "${Parameter extends {
        type: string;
    } ? Parameter['type'] : string}" is not a valid ABI type.`>;
}> : Parameter) extends infer Parameter2 extends {
    type: unknown;
} ? Parameter2['type'] extends `${infer Prefix extends 'u' | ''}int${infer Suffix extends `[${string}]` | ''}` ? Merge<Parameter2, {
    readonly type: `${Prefix}int256${Suffix}`;
}> : Parameter2 : never : never;
type _ParseFunctionParametersAndStateMutability<TSignature extends string> = TSignature extends `${infer Head}returns (${string})` ? _ParseFunctionParametersAndStateMutability<Trim<Head>> : TSignature extends `function ${string}(${infer Parameters})` ? {
    Inputs: Parameters;
    StateMutability: 'nonpayable';
} : TSignature extends `function ${string}(${infer Parameters}) ${infer ScopeOrStateMutability extends Scope | AbiStateMutability | `${Scope} ${AbiStateMutability}`}` ? {
    Inputs: Parameters;
    StateMutability: ScopeOrStateMutability extends `${Scope} ${infer StateMutability extends AbiStateMutability}` ? StateMutability : ScopeOrStateMutability extends AbiStateMutability ? ScopeOrStateMutability : 'nonpayable';
} : never;
type _ParseConstructorParametersAndStateMutability<TSignature extends string> = TSignature extends `constructor(${infer Parameters}) payable` ? {
    Inputs: Parameters;
    StateMutability: 'payable';
} : TSignature extends `constructor(${infer Parameters})` ? {
    Inputs: Parameters;
    StateMutability: 'nonpayable';
} : never;
type _ParseTuple<T extends `(${string})${string}`, Options extends ParseOptions = DefaultParseOptions> = T extends `(${infer Parameters})` ? {
    readonly type: 'tuple';
    readonly components: ParseAbiParameters$1<SplitParameters<Parameters>, Omit<Options, 'Modifier'>>;
} : T extends `(${infer Head})[${'' | `${SolidityFixedArrayRange}`}]` ? T extends `(${Head})[${infer Size}]` ? {
    readonly type: `tuple[${Size}]`;
    readonly components: ParseAbiParameters$1<SplitParameters<Head>, Omit<Options, 'Modifier'>>;
} : never : T extends `(${infer Parameters})[${'' | `${SolidityFixedArrayRange}`}] ${infer NameOrModifier}` ? T extends `(${Parameters})[${infer Size}] ${NameOrModifier}` ? NameOrModifier extends `${string}) ${string}` ? _UnwrapNameOrModifier<NameOrModifier> extends infer Parts extends {
    NameOrModifier: string;
    End: string;
} ? {
    readonly type: 'tuple';
    readonly components: ParseAbiParameters$1<SplitParameters<`${Parameters})[${Size}] ${Parts['End']}`>, Omit<Options, 'Modifier'>>;
} & _SplitNameOrModifier<Parts['NameOrModifier'], Options> : never : {
    readonly type: `tuple[${Size}]`;
    readonly components: ParseAbiParameters$1<SplitParameters<Parameters>, Omit<Options, 'Modifier'>>;
} & _SplitNameOrModifier<NameOrModifier, Options> : never : T extends `(${infer Parameters}) ${infer NameOrModifier}` ? NameOrModifier extends `${string}) ${string}` ? _UnwrapNameOrModifier<NameOrModifier> extends infer Parts extends {
    NameOrModifier: string;
    End: string;
} ? {
    readonly type: 'tuple';
    readonly components: ParseAbiParameters$1<SplitParameters<`${Parameters}) ${Parts['End']}`>, Omit<Options, 'Modifier'>>;
} & _SplitNameOrModifier<Parts['NameOrModifier'], Options> : never : {
    readonly type: 'tuple';
    readonly components: ParseAbiParameters$1<SplitParameters<Parameters>, Omit<Options, 'Modifier'>>;
} & _SplitNameOrModifier<NameOrModifier, Options> : never;
type _SplitNameOrModifier<T extends string, Options extends ParseOptions = DefaultParseOptions> = Trim<T> extends infer Trimmed ? Options extends {
    Modifier: Modifier;
} ? Trimmed extends `${infer Mod extends Options['Modifier']} ${infer Name}` ? {
    readonly name: Trim<Name>;
} & (Mod extends 'indexed' ? {
    readonly indexed: true;
} : object) : Trimmed extends Options['Modifier'] ? Trimmed extends 'indexed' ? {
    readonly indexed: true;
} : object : {
    readonly name: Trimmed;
} : {
    readonly name: Trimmed;
} : never;
type _UnwrapNameOrModifier<T extends string, Current extends string = ''> = T extends `${infer Head}) ${infer Tail}` ? _UnwrapNameOrModifier<Tail, `${Current}${Current extends '' ? '' : ') '}${Head}`> : {
    End: Trim<Current>;
    NameOrModifier: Trim<T>;
};

type StructLookup = Record<string, readonly AbiParameter[]>;
type ParseStructs<TSignatures extends readonly string[]> = {
    [Signature in TSignatures[number] as ParseStruct<Signature> extends infer Struct extends {
        name: string;
    } ? Struct['name'] : never]: ParseStruct<Signature>['components'];
} extends infer Structs extends Record<string, readonly (AbiParameter & {
    type: string;
})[]> ? {
    [StructName in keyof Structs]: ResolveStructs<Structs[StructName], Structs>;
} : never;
type ParseStruct<TSignature extends string, TStructs extends StructLookup | unknown = unknown> = TSignature extends StructSignature<infer Name, infer Properties> ? {
    readonly name: Trim<Name>;
    readonly components: ParseStructProperties<Properties, TStructs>;
} : never;
type ResolveStructs<TAbiParameters extends readonly (AbiParameter & {
    type: string;
})[], TStructs extends Record<string, readonly (AbiParameter & {
    type: string;
})[]>, TKeyReferences extends {
    [_: string]: unknown;
} | unknown = unknown> = readonly [
    ...{
        [K in keyof TAbiParameters]: TAbiParameters[K]['type'] extends `${infer Head extends string & keyof TStructs}[${infer Tail}]` ? Head extends keyof TKeyReferences ? Error$1<`Circular reference detected. Struct "${TAbiParameters[K]['type']}" is a circular reference.`> : {
            readonly name: TAbiParameters[K]['name'];
            readonly type: `tuple[${Tail}]`;
            readonly components: ResolveStructs<TStructs[Head], TStructs, TKeyReferences & {
                [_ in Head]: true;
            }>;
        } : TAbiParameters[K]['type'] extends keyof TStructs ? TAbiParameters[K]['type'] extends keyof TKeyReferences ? Error$1<`Circular reference detected. Struct "${TAbiParameters[K]['type']}" is a circular reference.`> : {
            readonly name: TAbiParameters[K]['name'];
            readonly type: 'tuple';
            readonly components: ResolveStructs<TStructs[TAbiParameters[K]['type']], TStructs, TKeyReferences & {
                [_ in TAbiParameters[K]['type']]: true;
            }>;
        } : TAbiParameters[K];
    }
];
type ParseStructProperties<T extends string, TStructs extends StructLookup | unknown = unknown, Result extends any[] = []> = Trim<T> extends `${infer Head};${infer Tail}` ? ParseStructProperties<Tail, TStructs, [
    ...Result,
    ParseAbiParameter$1<Head, {
        Structs: TStructs;
    }>
]> : Result;

/**
 * Parses human-readable ABI into JSON {@link Abi}
 *
 * @param TSignatures - Human-readable ABI
 * @returns Parsed {@link Abi}
 *
 * @example
 * type Result = ParseAbi<
 *   // ^? type Result = readonly [{ name: "balanceOf"; type: "function"; stateMutability:...
 *   [
 *     'function balanceOf(address owner) view returns (uint256)',
 *     'event Transfer(address indexed from, address indexed to, uint256 amount)',
 *   ]
 * >
 */
type ParseAbi<TSignatures extends readonly string[] | readonly unknown[]> = string[] extends TSignatures ? Abi : TSignatures extends readonly string[] ? TSignatures extends Signatures<TSignatures> ? ParseStructs<TSignatures> extends infer Structs ? {
    [K in keyof TSignatures]: TSignatures[K] extends string ? ParseSignature<TSignatures[K], Structs> : never;
} extends infer Mapped extends readonly unknown[] ? Filter<Mapped, never> extends infer Result ? Result extends readonly [] ? never : Result : never : never : never : never : never;
/**
 * Parses human-readable ABI into JSON {@link Abi}
 *
 * @param signatures - Human-Readable ABI
 * @returns Parsed {@link Abi}
 *
 * @example
 * const abi = parseAbi([
 *   //  ^? const abi: readonly [{ name: "balanceOf"; type: "function"; stateMutability:...
 *   'function balanceOf(address owner) view returns (uint256)',
 *   'event Transfer(address indexed from, address indexed to, uint256 amount)',
 * ])
 */
declare function parseAbi<TSignatures extends readonly string[] | readonly unknown[]>(signatures: Narrow<TSignatures> & (TSignatures extends readonly string[] ? TSignatures extends readonly [] ? Error$1<'At least one signature required.'> : string[] extends TSignatures ? unknown : Signatures<TSignatures> : never)): ParseAbi<TSignatures>;

/**
 * Parses human-readable ABI item (e.g. error, event, function) into {@link Abi} item
 *
 * @param TSignature - Human-readable ABI item
 * @returns Parsed {@link Abi} item
 *
 * @example
 * type Result = ParseAbiItem<'function balanceOf(address owner) view returns (uint256)'>
 * //   ^? type Result = { name: "balanceOf"; type: "function"; stateMutability: "view";...
 *
 * @example
 * type Result = ParseAbiItem<
 *   // ^? type Result = { name: "foo"; type: "function"; stateMutability: "view"; inputs:...
 *   ['function foo(Baz bar) view returns (string)', 'struct Baz { string name; }']
 * >
 */
type ParseAbiItem<TSignature extends string | readonly string[] | readonly unknown[]> = (TSignature extends string ? string extends TSignature ? Abi[number] : TSignature extends Signature<TSignature> ? ParseSignature<TSignature> : never : never) | (TSignature extends readonly string[] ? string[] extends TSignature ? Abi[number] : TSignature extends Signatures<TSignature> ? ParseStructs<TSignature> extends infer Structs ? {
    [K in keyof TSignature]: ParseSignature<TSignature[K] extends string ? TSignature[K] : never, Structs>;
} extends infer Mapped extends readonly unknown[] ? Filter<Mapped, never>[0] extends infer Result ? Result extends undefined ? never : Result : never : never : never : never : never);
/**
 * Parses human-readable ABI item (e.g. error, event, function) into {@link Abi} item
 *
 * @param signature - Human-readable ABI item
 * @returns Parsed {@link Abi} item
 *
 * @example
 * const abiItem = parseAbiItem('function balanceOf(address owner) view returns (uint256)')
 * //    ^? const abiItem: { name: "balanceOf"; type: "function"; stateMutability: "view";...
 *
 * @example
 * const abiItem = parseAbiItem([
 *   //  ^? const abiItem: { name: "foo"; type: "function"; stateMutability: "view"; inputs:...
 *   'function foo(Baz bar) view returns (string)',
 *   'struct Baz { string name; }',
 * ])
 */
declare function parseAbiItem<TSignature extends string | readonly string[] | readonly unknown[]>(signature: Narrow<TSignature> & ((TSignature extends string ? string extends TSignature ? unknown : Signature<TSignature> : never) | (TSignature extends readonly string[] ? TSignature extends readonly [] ? Error$1<'At least one signature required.'> : string[] extends TSignature ? unknown : Signatures<TSignature> : never))): ParseAbiItem<TSignature>;

/**
 * Parses human-readable ABI parameter into {@link AbiParameter}
 *
 * @param TParam - Human-readable ABI parameter
 * @returns Parsed {@link AbiParameter}
 *
 * @example
 * type Result = ParseAbiParameter<'address from'>
 * //   ^? type Result = { type: "address"; name: "from"; }
 *
 * @example
 * type Result = ParseAbiParameter<
 *   // ^? type Result = { type: "tuple"; components: [{ type: "string"; name:...
 *   ['Baz bar', 'struct Baz { string name; }']
 * >
 */
type ParseAbiParameter<TParam extends string | readonly string[] | readonly unknown[]> = (TParam extends string ? TParam extends '' ? never : string extends TParam ? AbiParameter : ParseAbiParameter$1<TParam, {
    Modifier: Modifier;
}> : never) | (TParam extends readonly string[] ? string[] extends TParam ? AbiParameter : ParseStructs<TParam> extends infer Structs ? {
    [K in keyof TParam]: TParam[K] extends string ? IsStructSignature<TParam[K]> extends true ? never : ParseAbiParameter$1<TParam[K], {
        Modifier: Modifier;
        Structs: Structs;
    }> : never;
} extends infer Mapped extends readonly unknown[] ? Filter<Mapped, never>[0] extends infer Result ? Result extends undefined ? never : Result : never : never : never : never);
/**
 * Parses human-readable ABI parameter into {@link AbiParameter}
 *
 * @param param - Human-readable ABI parameter
 * @returns Parsed {@link AbiParameter}
 *
 * @example
 * const abiParameter = parseAbiParameter('address from')
 * //    ^? const abiParameter: { type: "address"; name: "from"; }
 *
 * @example
 * const abiParameter = parseAbiParameter([
 *   //  ^? const abiParameter: { type: "tuple"; components: [{ type: "string"; name:...
 *   'Baz bar',
 *   'struct Baz { string name; }',
 * ])
 */
declare function parseAbiParameter<TParam extends string | readonly string[] | readonly unknown[]>(param: Narrow<TParam> & ((TParam extends string ? TParam extends '' ? Error$1<'Empty string is not allowed.'> : unknown : never) | (TParam extends readonly string[] ? TParam extends readonly [] ? Error$1<'At least one parameter required.'> : string[] extends TParam ? unknown : unknown : never))): ParseAbiParameter<TParam>;

/**
 * Parses human-readable ABI parameters into {@link AbiParameter}s
 *
 * @param TParams - Human-readable ABI parameters
 * @returns Parsed {@link AbiParameter}s
 *
 * @example
 * type Result = ParseAbiParameters('address from, address to, uint256 amount')
 * //   ^? type Result: [{ type: "address"; name: "from"; }, { type: "address";...
 *
 * @example
 * type Result = ParseAbiParameters<
 *   // ^? type Result: [{ type: "tuple"; components: [{ type: "string"; name:...
 *   ['Baz bar', 'struct Baz { string name; }']
 * >
 */
type ParseAbiParameters<TParams extends string | readonly string[] | readonly unknown[]> = (TParams extends string ? TParams extends '' ? never : string extends TParams ? readonly AbiParameter[] : ParseAbiParameters$1<SplitParameters<TParams>, {
    Modifier: Modifier;
}> : never) | (TParams extends readonly string[] ? string[] extends TParams ? AbiParameter : ParseStructs<TParams> extends infer Structs ? {
    [K in keyof TParams]: TParams[K] extends string ? IsStructSignature<TParams[K]> extends true ? never : ParseAbiParameters$1<SplitParameters<TParams[K]>, {
        Modifier: Modifier;
        Structs: Structs;
    }> : never;
} extends infer Mapped extends readonly unknown[] ? Filter<Mapped, never>[0] extends infer Result ? Result extends undefined ? never : Result : never : never : never : never);
/**
 * Parses human-readable ABI parameters into {@link AbiParameter}s
 *
 * @param params - Human-readable ABI parameters
 * @returns Parsed {@link AbiParameter}s
 *
 * @example
 * const abiParameters = parseAbiParameters('address from, address to, uint256 amount')
 * //    ^? const abiParameters: [{ type: "address"; name: "from"; }, { type: "address";...
 *
 * @example
 * const abiParameters = parseAbiParameters([
 *   //  ^? const abiParameters: [{ type: "tuple"; components: [{ type: "string"; name:...
 *   'Baz bar',
 *   'struct Baz { string name; }',
 * ])
 */
declare function parseAbiParameters<TParams extends string | readonly string[] | readonly unknown[]>(params: Narrow<TParams> & ((TParams extends string ? TParams extends '' ? Error$1<'Empty string is not allowed.'> : unknown : never) | (TParams extends readonly string[] ? TParams extends readonly [] ? Error$1<'At least one parameter required.'> : string[] extends TParams ? unknown : unknown : never))): ParseAbiParameters<TParams>;

export { AbiParameterToPrimitiveType, AbiParametersToPrimitiveTypes, AbiTypeToPrimitiveType, BaseError, ExtractAbiError, ExtractAbiErrorNames, ExtractAbiErrors, ExtractAbiEvent, ExtractAbiEventNames, ExtractAbiEvents, ExtractAbiFunction, ExtractAbiFunctionNames, ExtractAbiFunctions, IsAbi, IsTypedData, Narrow, ParseAbi, ParseAbiItem, ParseAbiParameter, ParseAbiParameters, TypedDataToPrimitiveTypes, narrow, parseAbi, parseAbiItem, parseAbiParameter, parseAbiParameters };
