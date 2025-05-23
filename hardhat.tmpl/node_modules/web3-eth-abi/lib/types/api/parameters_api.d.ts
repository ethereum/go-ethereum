import { AbiInput, HexString } from 'web3-types';
export { encodeParameters, inferTypesAndEncodeParameters } from '../coders/encode.js';
/**
 * Encodes a parameter based on its type to its ABI representation.
 * @param abi -  The type of the parameter. See the [Solidity documentation](https://docs.soliditylang.org/en/develop/types.html) for a list of types.
 * @param param - The actual parameter to encode.
 * @returns -  The ABI encoded parameter
 * @example
 * ```ts
 *  const res = web3.eth.abi.encodeParameter("uint256", "2345675643");
 *  console.log(res);
 *  0x000000000000000000000000000000000000000000000000000000008bd02b7b
 *
 *  const res = web3.eth.abi.encodeParameter("uint", "2345675643");
 *
 *  console.log(res);
 *  >0x000000000000000000000000000000000000000000000000000000008bd02b7b
 *
 *    const res = web3.eth.abi.encodeParameter("bytes32", "0xdf3234");
 *
 *  console.log(res);
 *  >0xdf32340000000000000000000000000000000000000000000000000000000000
 *
 *   const res = web3.eth.abi.encodeParameter("bytes", "0xdf3234");
 *
 *  console.log(res);
 *  > 0x00000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000003df32340000000000000000000000000000000000000000000000000000000000
 *
 *   const res = web3.eth.abi.encodeParameter("bytes32[]", ["0xdf3234", "0xfdfd"]);
 *
 *  console.log(res);
 *  > 0x00000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000002df32340000000000000000000000000000000000000000000000000000000000fdfd000000000000000000000000000000000000000000000000000000000000
 *
 *  const res = web3.eth.abi.encodeParameter(
 *    {
 *      ParentStruct: {
 *        propertyOne: "uint256",
 *        propertyTwo: "uint256",
 *        childStruct: {
 *          propertyOne: "uint256",
 *          propertyTwo: "uint256",
 *        },
 *      },
 *    },
 *    {
 *      propertyOne: 42,
 *      propertyTwo: 56,
 *      childStruct: {
 *        propertyOne: 45,
 *        propertyTwo: 78,
 *      },
 *    }
 *  );
 *
 *  console.log(res);
 *  > 0x000000000000000000000000000000000000000000000000000000000000002a0000000000000000000000000000000000000000000000000000000000000038000000000000000000000000000000000000000000000000000000000000002d000000000000000000000000000000000000000000000000000000000000004e
 * ```
 */
export declare const encodeParameter: (abi: AbiInput, param: unknown) => string;
/**
 * Should be used to decode list of params
 */
export declare const decodeParametersWith: (abis: AbiInput[] | ReadonlyArray<AbiInput>, bytes: HexString, loose: boolean) => {
    [key: string]: unknown;
    __length__: number;
};
/**
 * Should be used to decode list of params
 */
/**
 * Decodes ABI encoded parameters to its JavaScript types.
 * @param abi -  An array of {@link AbiInput}. See the [Solidity documentation](https://docs.soliditylang.org/en/develop/types.html) for a list of types.
 * @param bytes - The ABI byte code to decode
 * @returns - The result object containing the decoded parameters.
 * @example
 * ```ts
 * let res = web3.eth.abi.decodeParameters(
 *    ["string", "uint256"],
 *    "0x000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000ea000000000000000000000000000000000000000000000000000000000000000848656c6c6f212521000000000000000000000000000000000000000000000000"
 *  );
 *  console.log(res);
 *  > { '0': 'Hello!%!', '1': 234n, __length__: 2 }
 *
 * let res = web3.eth.abi.decodeParameters(
 *    [
 *      {
 *        type: "string",
 *        name: "myString",
 *      },
 *      {
 *        type: "uint256",
 *        name: "myNumber",
 *      },
 *    ],
 *    "0x000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000ea000000000000000000000000000000000000000000000000000000000000000848656c6c6f212521000000000000000000000000000000000000000000000000"
 *  );
 * console.log(res);
 *  > {
 *  '0': 'Hello!%!',
 *  '1': 234n,
 *  __length__: 2,
 *  myString: 'Hello!%!',
 *  myNumber: 234n
 * }
 *
 * const res = web3.eth.abi.decodeParameters(
 *    [
 *      "uint8[]",
 *      {
 *        ParentStruct: {
 *          propertyOne: "uint256",
 *          propertyTwo: "uint256",
 *          childStruct: {
 *            propertyOne: "uint256",
 *            propertyTwo: "uint256",
 *          },
 *        },
 *      },
 *    ],
 *    "0x00000000000000000000000000000000000000000000000000000000000000a0000000000000000000000000000000000000000000000000000000000000002a0000000000000000000000000000000000000000000000000000000000000038000000000000000000000000000000000000000000000000000000000000002d000000000000000000000000000000000000000000000000000000000000004e0000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000000002a0000000000000000000000000000000000000000000000000000000000000018"
 *  );
 *  console.log(res);
 *  >
 *  '0': [ 42n, 24n ],
 *  '1': {
 *    '0': 42n,
 *    '1': 56n,
 *    '2': {
 *      '0': 45n,
 *      '1': 78n,
 *      __length__: 2,
 *      propertyOne: 45n,
 *      propertyTwo: 78n
 *    },
 *    __length__: 3,
 *    propertyOne: 42n,
 *    propertyTwo: 56n,
 *    childStruct: {
 *      '0': 45n,
 *      '1': 78n,
 *      __length__: 2,
 *      propertyOne: 45n,
 *      propertyTwo: 78n
 *    }
 *  },
 *  __length__: 2,
 *  ParentStruct: {
 *    '0': 42n,
 *    '1': 56n,
 *    '2': {
 *      '0': 45n,
 *      '1': 78n,
 *      __length__: 2,
 *      propertyOne: 45n,
 *      propertyTwo: 78n
 *    },
 *    __length__: 3,
 *    propertyOne: 42n,
 *    propertyTwo: 56n,
 *    childStruct: {
 *      '0': 45n,
 *      '1': 78n,
 *      __length__: 2,
 *      propertyOne: 45n,
 *      propertyTwo: 78n
 *    }
 *  }
 *}
 * ```
 */
export declare const decodeParameters: (abi: AbiInput[] | ReadonlyArray<AbiInput>, bytes: HexString) => {
    [key: string]: unknown;
    __length__: number;
};
/**
 * Should be used to decode bytes to plain param
 */
/**
 * Decodes an ABI encoded parameter to its JavaScript type.
 * @param abi -  The type of the parameter. See the [Solidity documentation](https://docs.soliditylang.org/en/develop/types.html) for a list of types.
 * @param bytes - The ABI byte code to decode
 * @returns - The decoded parameter
 * @example
 * ```ts
 *   const res = web3.eth.abi.decodeParameter(
 *    "uint256",
 *    "0x0000000000000000000000000000000000000000000000000000000000000010"
 *  );
 *  console.log(res);
 * > 16n
 *
 *  const res = web3.eth.abi.decodeParameter(
 *    "string",
 *    "0x0000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000000848656c6c6f212521000000000000000000000000000000000000000000000000"
 *  );
 *
 *  console.log(res);
 *  > Hello!%!
 *
 *  const res = web3.eth.abi.decodeParameter(
 *    {
 *      ParentStruct: {
 *        propertyOne: "uint256",
 *        propertyTwo: "uint256",
 *        childStruct: {
 *          propertyOne: "uint256",
 *          propertyTwo: "uint256",
 *        },
 *      },
 *    },
 *    "0x000000000000000000000000000000000000000000000000000000000000002a0000000000000000000000000000000000000000000000000000000000000038000000000000000000000000000000000000000000000000000000000000002d000000000000000000000000000000000000000000000000000000000000004e"
 *  );
 *
 *  console.log(res);
 *   {
 *  '0': 42n,
 *  '1': 56n,
 *  '2': {
 *    '0': 45n,
 *    '1': 78n,
 *    __length__: 2,
 *    propertyOne: 45n,
 *    propertyTwo: 78n
 *  },
 *  __length__: 3,
 *  propertyOne: 42n,
 *  propertyTwo: 56n,
 *  childStruct: {
 *    '0': 45n,
 *    '1': 78n,
 *    __length__: 2,
 *    propertyOne: 45n,
 *    propertyTwo: 78n
 *  }
 *}
 * ```
 */
export declare const decodeParameter: (abi: AbiInput, bytes: HexString) => unknown;
//# sourceMappingURL=parameters_api.d.ts.map