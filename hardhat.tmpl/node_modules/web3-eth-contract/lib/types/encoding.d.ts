import { AbiConstructorFragment, AbiEventFragment, AbiFunctionFragment, Filter, HexString, ContractOptions } from 'web3-types';
export { decodeEventABI } from 'web3-eth';
type Writeable<T> = {
    -readonly [P in keyof T]: T[P];
};
export declare const encodeEventABI: ({ address }: ContractOptions, event: AbiEventFragment & {
    signature: string;
}, options?: Filter) => Writeable<Filter>;
export declare const encodeMethodABI: (abi: AbiFunctionFragment | AbiConstructorFragment, args: unknown[], deployData?: HexString) => string;
/** @deprecated import `decodeFunctionCall` from ''web3-eth-abi' instead. */
export declare const decodeMethodParams: (functionsAbi: AbiFunctionFragment | AbiConstructorFragment, data: HexString, methodSignatureProvided?: boolean) => import("web3-types").DecodedParams & {
    __method__: string;
};
/** @deprecated import `decodeFunctionReturn` from ''web3-eth-abi' instead. */
export declare const decodeMethodReturn: (functionsAbi: AbiFunctionFragment, returnValues?: HexString) => unknown;
//# sourceMappingURL=encoding.d.ts.map