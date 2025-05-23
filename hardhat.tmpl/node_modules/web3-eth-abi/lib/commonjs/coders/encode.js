"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.encodeParameters = encodeParameters;
exports.inferTypesAndEncodeParameters = inferTypesAndEncodeParameters;
/*
This file is part of web3.js.

web3.js is free software: you can redistribute it and/or modify
it under the terms of the GNU Lesser General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

web3.js is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Lesser General Public License for more details.

You should have received a copy of the GNU Lesser General Public License
along with web3.js.  If not, see <http://www.gnu.org/licenses/>.
*/
const web3_errors_1 = require("web3-errors");
const web3_utils_1 = require("web3-utils");
const web3_validator_1 = require("web3-validator");
const index_js_1 = require("./base/index.js");
const utils_js_1 = require("./utils.js");
/**
 * @param params - The params to infer the ABI from
 * @returns The inferred ABI
 * @example
 * ```
 * inferParamsAbi([1, -1, 'hello', '0x1234', ])
 * ```
 * > [{ type: 'int256' }, { type: 'uint256' }, { type: 'string' }, { type: 'bytes' }]
 * ```
 */
function inferParamsAbi(params) {
    const abi = [];
    params.forEach(param => {
        if (Array.isArray(param)) {
            const inferredParams = inferParamsAbi(param);
            abi.push({
                type: 'tuple',
                components: inferredParams,
                name: '',
                // eslint-disable-next-line @typescript-eslint/no-unsafe-argument
            });
        }
        else {
            // eslint-disable-next-line @typescript-eslint/no-unsafe-argument
            abi.push({ type: (0, web3_utils_1.toHex)(param, true) });
        }
    });
    return abi;
}
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
function encodeParameters(abi, params) {
    if ((abi === null || abi === void 0 ? void 0 : abi.length) !== params.length) {
        throw new web3_errors_1.AbiError('Invalid number of values received for given ABI', {
            expected: abi === null || abi === void 0 ? void 0 : abi.length,
            received: params.length,
        });
    }
    const abiParams = (0, utils_js_1.toAbiParams)(abi);
    return web3_validator_1.utils.uint8ArrayToHexString((0, index_js_1.encodeTuple)({ type: 'tuple', name: '', components: abiParams }, params).encoded);
}
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
function inferTypesAndEncodeParameters(params) {
    try {
        const abiParams = inferParamsAbi(params);
        return web3_validator_1.utils.uint8ArrayToHexString((0, index_js_1.encodeTuple)({ type: 'tuple', name: '', components: abiParams }, params).encoded);
    }
    catch (e) {
        // throws If the inferred params type caused an error
        throw new web3_errors_1.AbiError('Could not infer types from given params', {
            params,
        });
    }
}
//# sourceMappingURL=encode.js.map