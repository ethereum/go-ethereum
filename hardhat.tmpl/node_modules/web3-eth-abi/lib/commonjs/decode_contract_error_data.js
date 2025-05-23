"use strict";
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
Object.defineProperty(exports, "__esModule", { value: true });
exports.decodeContractErrorData = void 0;
const errors_api_js_1 = require("./api/errors_api.js");
const parameters_api_js_1 = require("./api/parameters_api.js");
const utils_js_1 = require("./utils.js");
const decodeContractErrorData = (errorsAbi, error) => {
    if (error === null || error === void 0 ? void 0 : error.data) {
        let errorName;
        let errorSignature;
        let errorArgs;
        try {
            const errorSha = error.data.slice(0, 10);
            const errorAbi = errorsAbi.find(abi => (0, errors_api_js_1.encodeErrorSignature)(abi).startsWith(errorSha));
            if (errorAbi === null || errorAbi === void 0 ? void 0 : errorAbi.inputs) {
                errorName = errorAbi.name;
                errorSignature = (0, utils_js_1.jsonInterfaceMethodToString)(errorAbi);
                // decode abi.inputs according to EIP-838
                errorArgs = (0, parameters_api_js_1.decodeParameters)([...errorAbi.inputs], error.data.substring(10));
            }
            else if (error.data.startsWith('0x08c379a0')) {
                // If ABI was not provided, check for the 2 famous errors: 'Error(string)' or 'Panic(uint256)'
                errorName = 'Error';
                errorSignature = 'Error(string)';
                // decode abi.inputs according to EIP-838
                errorArgs = (0, parameters_api_js_1.decodeParameters)([
                    {
                        name: 'message',
                        type: 'string',
                    },
                ], error.data.substring(10));
            }
            else if (error.data.startsWith('0x4e487b71')) {
                errorName = 'Panic';
                errorSignature = 'Panic(uint256)';
                // decode abi.inputs according to EIP-838
                errorArgs = (0, parameters_api_js_1.decodeParameters)([
                    {
                        name: 'code',
                        type: 'uint256',
                    },
                ], error.data.substring(10));
            }
            else {
                console.error('No matching error abi found for error data', error.data);
            }
        }
        catch (err) {
            console.error(err);
        }
        if (errorName) {
            error.setDecodedProperties(errorName, errorSignature, errorArgs);
        }
    }
};
exports.decodeContractErrorData = decodeContractErrorData;
//# sourceMappingURL=decode_contract_error_data.js.map