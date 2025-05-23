import { encodeParameters } from 'web3-eth-abi';
/**
 * The object for `web3.abi`
 */
declare const _default: {
    encodeEventSignature: (functionName: string | import("web3-types").AbiEventFragment) => string;
    encodeFunctionCall: (jsonInterface: import("web3-types").AbiFunctionFragment, params: unknown[]) => string;
    encodeFunctionSignature: (functionName: string | import("web3-types").AbiFunctionFragment) => string;
    encodeParameter: (abi: import("web3-types").AbiInput, param: unknown) => string;
    encodeParameters: typeof encodeParameters;
    decodeParameter: (abi: import("web3-types").AbiInput, bytes: import("web3-types").HexString) => unknown;
    decodeParameters: (abi: import("web3-types").AbiInput[] | ReadonlyArray<import("web3-types").AbiInput>, bytes: import("web3-types").HexString) => {
        [key: string]: unknown;
        __length__: number;
    };
    decodeLog: <ReturnType extends import("web3-types").DecodedParams>(inputs: Array<import("web3-types").AbiParameter> | ReadonlyArray<import("web3-types").AbiParameter>, data: import("web3-types").HexString, topics: string | string[]) => ReturnType;
};
export default _default;
