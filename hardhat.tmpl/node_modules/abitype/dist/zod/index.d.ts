import { z } from 'zod';
import { i as AbiParameter$1 } from '../abi-7aa1f183.js';
import '../config-edd78478.js';

declare const SolidityAddress: z.ZodLiteral<"address">;
declare const SolidityBool: z.ZodLiteral<"bool">;
declare const SolidityBytes: z.ZodString;
declare const SolidityFunction: z.ZodLiteral<"function">;
declare const SolidityString: z.ZodLiteral<"string">;
declare const SolidityTuple: z.ZodLiteral<"tuple">;
declare const SolidityInt: z.ZodString;
declare const SolidityArrayWithoutTuple: z.ZodString;
declare const SolidityArrayWithTuple: z.ZodString;
declare const SolidityArray: z.ZodUnion<[z.ZodString, z.ZodString]>;
declare const AbiParameter: z.ZodType<AbiParameter$1>;
declare const AbiStateMutability: z.ZodUnion<[z.ZodLiteral<"pure">, z.ZodLiteral<"view">, z.ZodLiteral<"nonpayable">, z.ZodLiteral<"payable">]>;
declare const AbiFunction: z.ZodEffects<z.ZodObject<{
    type: z.ZodLiteral<"function">;
    /**
     * @deprecated use `pure` or `view` from {@link AbiStateMutability} instead
     * https://github.com/ethereum/solidity/issues/992
     */
    constant: z.ZodOptional<z.ZodBoolean>;
    /**
     * @deprecated Vyper used to provide gas estimates
     * https://github.com/vyperlang/vyper/issues/2151
     */
    gas: z.ZodOptional<z.ZodNumber>;
    inputs: z.ZodArray<z.ZodType<AbiParameter$1, z.ZodTypeDef, AbiParameter$1>, "many">;
    name: z.ZodString;
    outputs: z.ZodArray<z.ZodType<AbiParameter$1, z.ZodTypeDef, AbiParameter$1>, "many">;
    /**
     * @deprecated use `payable` or `nonpayable` from {@link AbiStateMutability} instead
     * https://github.com/ethereum/solidity/issues/992
     */
    payable: z.ZodOptional<z.ZodBoolean>;
    stateMutability: z.ZodUnion<[z.ZodLiteral<"pure">, z.ZodLiteral<"view">, z.ZodLiteral<"nonpayable">, z.ZodLiteral<"payable">]>;
}, "strip", z.ZodTypeAny, {
    payable?: boolean | undefined;
    constant?: boolean | undefined;
    gas?: number | undefined;
    inputs: AbiParameter$1[];
    outputs: AbiParameter$1[];
    type: "function";
    name: string;
    stateMutability: "pure" | "view" | "nonpayable" | "payable";
}, {
    payable?: boolean | undefined;
    constant?: boolean | undefined;
    gas?: number | undefined;
    inputs: AbiParameter$1[];
    outputs: AbiParameter$1[];
    type: "function";
    name: string;
    stateMutability: "pure" | "view" | "nonpayable" | "payable";
}>, {
    payable?: boolean | undefined;
    constant?: boolean | undefined;
    gas?: number | undefined;
    inputs: AbiParameter$1[];
    outputs: AbiParameter$1[];
    type: "function";
    name: string;
    stateMutability: "pure" | "view" | "nonpayable" | "payable";
}, unknown>;
declare const AbiConstructor: z.ZodEffects<z.ZodObject<{
    type: z.ZodLiteral<"constructor">;
    /**
     * @deprecated use `pure` or `view` from {@link AbiStateMutability} instead
     * https://github.com/ethereum/solidity/issues/992
     */
    inputs: z.ZodArray<z.ZodType<AbiParameter$1, z.ZodTypeDef, AbiParameter$1>, "many">;
    /**
     * @deprecated use `payable` or `nonpayable` from {@link AbiStateMutability} instead
     * https://github.com/ethereum/solidity/issues/992
     */
    payable: z.ZodOptional<z.ZodBoolean>;
    stateMutability: z.ZodUnion<[z.ZodLiteral<"nonpayable">, z.ZodLiteral<"payable">]>;
}, "strip", z.ZodTypeAny, {
    payable?: boolean | undefined;
    inputs: AbiParameter$1[];
    type: "constructor";
    stateMutability: "nonpayable" | "payable";
}, {
    payable?: boolean | undefined;
    inputs: AbiParameter$1[];
    type: "constructor";
    stateMutability: "nonpayable" | "payable";
}>, {
    payable?: boolean | undefined;
    inputs: AbiParameter$1[];
    type: "constructor";
    stateMutability: "nonpayable" | "payable";
}, unknown>;
declare const AbiFallback: z.ZodEffects<z.ZodObject<{
    type: z.ZodLiteral<"fallback">;
    /**
     * @deprecated use `pure` or `view` from {@link AbiStateMutability} instead
     * https://github.com/ethereum/solidity/issues/992
     */
    inputs: z.ZodOptional<z.ZodTuple<[], null>>;
    /**
     * @deprecated use `payable` or `nonpayable` from {@link AbiStateMutability} instead
     * https://github.com/ethereum/solidity/issues/992
     */
    payable: z.ZodOptional<z.ZodBoolean>;
    stateMutability: z.ZodUnion<[z.ZodLiteral<"nonpayable">, z.ZodLiteral<"payable">]>;
}, "strip", z.ZodTypeAny, {
    payable?: boolean | undefined;
    inputs?: [] | undefined;
    type: "fallback";
    stateMutability: "nonpayable" | "payable";
}, {
    payable?: boolean | undefined;
    inputs?: [] | undefined;
    type: "fallback";
    stateMutability: "nonpayable" | "payable";
}>, {
    payable?: boolean | undefined;
    inputs?: [] | undefined;
    type: "fallback";
    stateMutability: "nonpayable" | "payable";
}, unknown>;
declare const AbiReceive: z.ZodObject<{
    type: z.ZodLiteral<"receive">;
    stateMutability: z.ZodLiteral<"payable">;
}, "strip", z.ZodTypeAny, {
    type: "receive";
    stateMutability: "payable";
}, {
    type: "receive";
    stateMutability: "payable";
}>;
declare const AbiEvent: z.ZodObject<{
    type: z.ZodLiteral<"event">;
    anonymous: z.ZodOptional<z.ZodBoolean>;
    inputs: z.ZodArray<z.ZodIntersection<z.ZodType<AbiParameter$1, z.ZodTypeDef, AbiParameter$1>, z.ZodObject<{
        indexed: z.ZodOptional<z.ZodBoolean>;
    }, "strip", z.ZodTypeAny, {
        indexed?: boolean | undefined;
    }, {
        indexed?: boolean | undefined;
    }>>, "many">;
    name: z.ZodString;
}, "strip", z.ZodTypeAny, {
    anonymous?: boolean | undefined;
    inputs: (AbiParameter$1 & {
        indexed?: boolean | undefined;
    })[];
    type: "event";
    name: string;
}, {
    anonymous?: boolean | undefined;
    inputs: (AbiParameter$1 & {
        indexed?: boolean | undefined;
    })[];
    type: "event";
    name: string;
}>;
declare const AbiError: z.ZodObject<{
    type: z.ZodLiteral<"error">;
    inputs: z.ZodArray<z.ZodType<AbiParameter$1, z.ZodTypeDef, AbiParameter$1>, "many">;
    name: z.ZodString;
}, "strip", z.ZodTypeAny, {
    inputs: AbiParameter$1[];
    type: "error";
    name: string;
}, {
    inputs: AbiParameter$1[];
    type: "error";
    name: string;
}>;
declare const AbiItemType: z.ZodUnion<[z.ZodLiteral<"constructor">, z.ZodLiteral<"event">, z.ZodLiteral<"error">, z.ZodLiteral<"fallback">, z.ZodLiteral<"function">, z.ZodLiteral<"receive">]>;
/**
 * Zod Schema for Contract [ABI Specification](https://docs.soliditylang.org/en/latest/abi-spec.html#json)
 *
 * @example
 * const parsedAbi = Abi.parse([…])
 */
declare const Abi: z.ZodArray<z.ZodUnion<[z.ZodObject<{
    type: z.ZodLiteral<"error">;
    inputs: z.ZodArray<z.ZodType<AbiParameter$1, z.ZodTypeDef, AbiParameter$1>, "many">;
    name: z.ZodString;
}, "strip", z.ZodTypeAny, {
    inputs: AbiParameter$1[];
    type: "error";
    name: string;
}, {
    inputs: AbiParameter$1[];
    type: "error";
    name: string;
}>, z.ZodObject<{
    type: z.ZodLiteral<"event">;
    anonymous: z.ZodOptional<z.ZodBoolean>;
    inputs: z.ZodArray<z.ZodIntersection<z.ZodType<AbiParameter$1, z.ZodTypeDef, AbiParameter$1>, z.ZodObject<{
        indexed: z.ZodOptional<z.ZodBoolean>;
    }, "strip", z.ZodTypeAny, {
        indexed?: boolean | undefined;
    }, {
        indexed?: boolean | undefined;
    }>>, "many">;
    name: z.ZodString;
}, "strip", z.ZodTypeAny, {
    anonymous?: boolean | undefined;
    inputs: (AbiParameter$1 & {
        indexed?: boolean | undefined;
    })[];
    type: "event";
    name: string;
}, {
    anonymous?: boolean | undefined;
    inputs: (AbiParameter$1 & {
        indexed?: boolean | undefined;
    })[];
    type: "event";
    name: string;
}>, z.ZodEffects<z.ZodIntersection<z.ZodObject<{
    /**
     * @deprecated use `pure` or `view` from {@link AbiStateMutability} instead
     * https://github.com/ethereum/solidity/issues/992
     */
    constant: z.ZodOptional<z.ZodBoolean>;
    /**
     * @deprecated Vyper used to provide gas estimates
     * https://github.com/vyperlang/vyper/issues/2151
     */
    gas: z.ZodOptional<z.ZodNumber>;
    /**
     * @deprecated use `payable` or `nonpayable` from {@link AbiStateMutability} instead
     * https://github.com/ethereum/solidity/issues/992
     */
    payable: z.ZodOptional<z.ZodBoolean>;
    stateMutability: z.ZodUnion<[z.ZodLiteral<"pure">, z.ZodLiteral<"view">, z.ZodLiteral<"nonpayable">, z.ZodLiteral<"payable">]>;
}, "strip", z.ZodTypeAny, {
    payable?: boolean | undefined;
    constant?: boolean | undefined;
    gas?: number | undefined;
    stateMutability: "pure" | "view" | "nonpayable" | "payable";
}, {
    payable?: boolean | undefined;
    constant?: boolean | undefined;
    gas?: number | undefined;
    stateMutability: "pure" | "view" | "nonpayable" | "payable";
}>, z.ZodDiscriminatedUnion<"type", [z.ZodObject<{
    type: z.ZodLiteral<"function">;
    inputs: z.ZodArray<z.ZodType<AbiParameter$1, z.ZodTypeDef, AbiParameter$1>, "many">;
    name: z.ZodString;
    outputs: z.ZodArray<z.ZodType<AbiParameter$1, z.ZodTypeDef, AbiParameter$1>, "many">;
}, "strip", z.ZodTypeAny, {
    inputs: AbiParameter$1[];
    outputs: AbiParameter$1[];
    type: "function";
    name: string;
}, {
    inputs: AbiParameter$1[];
    outputs: AbiParameter$1[];
    type: "function";
    name: string;
}>, z.ZodObject<{
    type: z.ZodLiteral<"constructor">;
    inputs: z.ZodArray<z.ZodType<AbiParameter$1, z.ZodTypeDef, AbiParameter$1>, "many">;
}, "strip", z.ZodTypeAny, {
    inputs: AbiParameter$1[];
    type: "constructor";
}, {
    inputs: AbiParameter$1[];
    type: "constructor";
}>, z.ZodObject<{
    type: z.ZodLiteral<"fallback">;
    inputs: z.ZodOptional<z.ZodTuple<[], null>>;
}, "strip", z.ZodTypeAny, {
    inputs?: [] | undefined;
    type: "fallback";
}, {
    inputs?: [] | undefined;
    type: "fallback";
}>, z.ZodObject<{
    type: z.ZodLiteral<"receive">;
    stateMutability: z.ZodLiteral<"payable">;
}, "strip", z.ZodTypeAny, {
    type: "receive";
    stateMutability: "payable";
}, {
    type: "receive";
    stateMutability: "payable";
}>]>>, {
    payable?: boolean | undefined;
    constant?: boolean | undefined;
    gas?: number | undefined;
    stateMutability: "pure" | "view" | "nonpayable" | "payable";
} & ({
    inputs: AbiParameter$1[];
    outputs: AbiParameter$1[];
    type: "function";
    name: string;
} | {
    inputs: AbiParameter$1[];
    type: "constructor";
} | {
    inputs?: [] | undefined;
    type: "fallback";
} | {
    type: "receive";
    stateMutability: "payable";
}), unknown>]>, "many">;

export { Abi, AbiConstructor, AbiError, AbiEvent, AbiFallback, AbiFunction, AbiItemType, AbiParameter, AbiReceive, AbiStateMutability, SolidityAddress, SolidityArray, SolidityArrayWithTuple, SolidityArrayWithoutTuple, SolidityBool, SolidityBytes, SolidityFunction, SolidityInt, SolidityString, SolidityTuple };
