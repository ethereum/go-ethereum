import { AbiInput } from 'web3-types';
/**
 * Encodes a parameter based on its type to its ABI representation.
 * @param abi - An array of {@link AbiInput}. See [Solidity's documentation](https://solidity.readthedocs.io/en/v0.5.3/abi-spec.html#json) for more details.
 * @param params - The actual parameters to encode.
 * @returns - The ABI encoded parameters
 * @example
 * ```ts
 * const res = web3.eth.abi.encodeParameters(
 *    ["uint256", "string"],
 *    ["2345675643", "Hello!%"]
 *  );
 *
 *  console.log(res);
 *  > 0x000000000000000000000000000000000000000000000000000000008bd02b7b0000000000000000000000000000000000000000000000000000000000000040000000000000000000000000000000000000000000000000000000000000000748656c6c6f212500000000000000000000000000000000000000000000000000
 * ```
 */
export declare function encodeParameters(abi: ReadonlyArray<AbiInput>, params: unknown[]): string;
/**
 * Infer a smart contract method parameter type and then encode this parameter.
 * @param params - The parameters to encode.
 * @returns - The ABI encoded parameters
 *
 * @remarks
 * This method is useful when you don't know the type of the parameters you want to encode. It will infer the type of the parameters and then encode them.
 * However, it is not recommended to use this method when you know the type of the parameters you want to encode. In this case, use the {@link encodeParameters} method instead.
 * The type inference is not perfect and can lead to unexpected results. Especially when you want to encode an array, uint that is not uint256 or bytes....
 * @example
 * ```ts
 * const res = web3.eth.abi.encodeParameters(
 *    ["2345675643", "Hello!%"]
 *  );
 *
 *  console.log(res);
 *  > 0x000000000000000000000000000000000000000000000000000000008bd02b7b0000000000000000000000000000000000000000000000000000000000000040000000000000000000000000000000000000000000000000000000000000000748656c6c6f212500000000000000000000000000000000000000000000000000
 * ```
 */
export declare function inferTypesAndEncodeParameters(params: unknown[]): string;
