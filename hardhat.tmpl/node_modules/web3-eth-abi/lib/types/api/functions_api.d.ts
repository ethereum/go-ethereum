import { AbiConstructorFragment, AbiFunctionFragment, DecodedParams, HexString } from 'web3-types';
/**
 * Encodes the function name to its ABI representation, which are the first 4 bytes of the sha3 of the function name including  types.
 * The JSON interface spec documentation https://docs.soliditylang.org/en/latest/abi-spec.html#json
 * @param functionName - The function name to encode or the `JSON interface` object of the function.
 * If the passed parameter is a string, it has to be in the form of `functionName(param1Type,param2Type,...)`. eg: myFunction(uint256,uint32[],bytes10,bytes)
 * @returns - The ABI signature of the function.
 * @example
 * ```ts
 * const signature = web3.eth.abi.encodeFunctionSignature({
 *   name: "myMethod",
 *   type: "function",
 *   inputs: [
 *     {
 *       type: "uint256",
 *       name: "myNumber",
 *     },
 *     {
 *       type: "string",
 *       name: "myString",
 *     },
 *   ],
 * });
 * console.log(signature);
 * > 0x24ee0097
 *
 * const signature = web3.eth.abi.encodeFunctionSignature('myMethod(uint256,string)')
 * console.log(signature);
 * > 0x24ee0097
 *
 * const signature = web3.eth.abi.encodeFunctionSignature('safeTransferFrom(address,address,uint256,bytes)');
 * console.log(signature);
 * > 0xb88d4fde
 * ```
 */
export declare const encodeFunctionSignature: (functionName: string | AbiFunctionFragment) => string;
/**
 * Encodes a function call using its `JSON interface` object and given parameters.
 * The JSON interface spec documentation https://docs.soliditylang.org/en/latest/abi-spec.html#json
 * @param jsonInterface - The `JSON interface` object of the function.
 * @param params - The parameters to encode
 * @returns - The ABI encoded function call, which, means the function signature and the parameters passed.
 * @example
 * ```ts
 * const sig = web3.eth.abi.encodeFunctionCall(
 *   {
 *     name: "myMethod",
 *     type: "function",
 *     inputs: [
 *       {
 *         type: "uint256",
 *         name: "myNumber",
 *       },
 *       {
 *         type: "string",
 *         name: "myString",
 *       },
 *     ],
 *   },
 *   ["2345675643", "Hello!%"]
 * );
 * console.log(sig);
 * > 0x24ee0097000000000000000000000000000000000000000000000000000000008bd02b7b0000000000000000000000000000000000000000000000000000000000000040000000000000000000000000000000000000000000000000000000000000000748656c6c6f212500000000000000000000000000000000000000000000000000
 *
 *
 *
 * const sig = web3.eth.abi.encodeFunctionCall(
 *   {
 *     inputs: [
 *       {
 *         name: "account",
 *         type: "address",
 *       },
 *     ],
 *     name: "balanceOf",
 *     outputs: [
 *       {
 *         name: "",
 *         type: "uint256",
 *       },
 *     ],
 *     stateMutability: "view",
 *     type: "function",
 *   },
 *   ["0x1234567890123456789012345678901234567890"]
 * );
 *
 * console.log(sig);
 * > 0x70a082310000000000000000000000001234567890123456789012345678901234567890
 * ```
 */
export declare const encodeFunctionCall: (jsonInterface: AbiFunctionFragment, params: unknown[]) => string;
/**
 * Decodes a function call data using its `JSON interface` object.
 * The JSON interface spec documentation https://docs.soliditylang.org/en/latest/abi-spec.html#json
 * @param functionsAbi - The `JSON interface` object of the function.
 * @param data - The data to decode
 * @param methodSignatureProvided - (Optional) if `false` do not remove the first 4 bytes that would rather contain the function signature.
 * @returns - The data decoded according to the passed ABI.
 * @example
 * ```ts
 * const data =
 * 	'0xa413686200000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000548656c6c6f0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000010416e6f74686572204772656574696e6700000000000000000000000000000000';
 * const params = decodeFunctionCall(
 * 	{
 * 		inputs: [
 * 			{ internalType: 'string', name: '_greeting', type: 'string' },
 * 			{ internalType: 'string', name: '_second_greeting', type: 'string' },
 * 		],
 * 		name: 'setGreeting',
 * 		outputs: [
 * 			{ internalType: 'bool', name: '', type: 'bool' },
 * 			{ internalType: 'string', name: '', type: 'string' },
 * 		],
 * 		stateMutability: 'nonpayable',
 * 		type: 'function',
 * 	},
 * 	data,
 * );

 * console.log(params);
 * > {
 * > 	'0': 'Hello',
 * > 	'1': 'Another Greeting',
 * > 	__length__: 2,
 * > 	__method__: 'setGreeting(string,string)',
 * > 	_greeting: 'Hello',
 * > 	_second_greeting: 'Another Greeting',
 * > }
 * ```
 */
export declare const decodeFunctionCall: (functionsAbi: AbiFunctionFragment | AbiConstructorFragment, data: HexString, methodSignatureProvided?: boolean) => DecodedParams & {
    __method__: string;
};
/**
 * Decodes a function call data using its `JSON interface` object.
 * The JSON interface spec documentation https://docs.soliditylang.org/en/latest/abi-spec.html#json
 * @returns - The ABI encoded function call, which, means the function signature and the parameters passed.
 * @param functionsAbi - The `JSON interface` object of the function.
 * @param returnValues - The data (the function-returned-values) to decoded
 * @returns - The function-returned-values decoded according to the passed ABI. If there are multiple values, it returns them as an object as the example below. But if it is a single value, it returns it only for simplicity.
 * @example
 * ```ts
 * // decode a multi-value data of a method
 * const data =
 * 	'0x00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000000548656c6c6f000000000000000000000000000000000000000000000000000000';
 * const decodedResult = decodeFunctionReturn(
 * 	{
 * 		inputs: [
 * 			{ internalType: 'string', name: '_greeting', type: 'string' }
 * 		],
 * 		name: 'setGreeting',
 * 		outputs: [
 * 			{ internalType: 'string', name: '', type: 'string' },
 * 			{ internalType: 'bool', name: '', type: 'bool' },
 * 		],
 * 		stateMutability: 'nonpayable',
 * 		type: 'function',
 * 	},
 * 	data,
 * );

 * console.log(decodedResult);
 * > { '0': 'Hello', '1': true, __length__: 2 }
 *
 *
 * // decode a single-value data of a method
 * const data =
 * 	'0x0000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000000548656c6c6f000000000000000000000000000000000000000000000000000000';
 * const decodedResult = decodeFunctionReturn(
 * 	{
 * 		inputs: [
 * 			{ internalType: 'string', name: '_greeting', type: 'string' }
 * 		],
 * 		name: 'setGreeting',
 * 		outputs: [{ internalType: 'string', name: '', type: 'string' }],
 * 		stateMutability: 'nonpayable',
 * 		type: 'function',
 * 	},
 * 	data,
 * );

 * console.log(decodedResult);
 * > 'Hello'
 * ```
 */
export declare const decodeFunctionReturn: (functionsAbi: AbiFunctionFragment, returnValues?: HexString) => unknown;
//# sourceMappingURL=functions_api.d.ts.map