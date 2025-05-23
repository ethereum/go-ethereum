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
exports.ExistingPluginNamespaceError = exports.AbiError = exports.OperationAbortError = exports.OperationTimeoutError = exports.MethodNotImplementedError = exports.FormatterError = exports.InvalidMethodParamsError = exports.InvalidNumberOfParamsError = void 0;
/* eslint-disable max-classes-per-file */
const error_codes_js_1 = require("../error_codes.js");
const web3_error_base_js_1 = require("../web3_error_base.js");
class InvalidNumberOfParamsError extends web3_error_base_js_1.BaseWeb3Error {
    constructor(got, expected, method) {
        super(`Invalid number of parameters for "${method}". Got "${got}" expected "${expected}"!`);
        this.got = got;
        this.expected = expected;
        this.method = method;
        this.code = error_codes_js_1.ERR_PARAM;
    }
    toJSON() {
        return Object.assign(Object.assign({}, super.toJSON()), { got: this.got, expected: this.expected, method: this.method });
    }
}
exports.InvalidNumberOfParamsError = InvalidNumberOfParamsError;
class InvalidMethodParamsError extends web3_error_base_js_1.BaseWeb3Error {
    constructor(hint) {
        super(`Invalid parameters passed. "${typeof hint !== 'undefined' ? hint : ''}"`);
        this.hint = hint;
        this.code = error_codes_js_1.ERR_INVALID_METHOD_PARAMS;
    }
    toJSON() {
        return Object.assign(Object.assign({}, super.toJSON()), { hint: this.hint });
    }
}
exports.InvalidMethodParamsError = InvalidMethodParamsError;
class FormatterError extends web3_error_base_js_1.BaseWeb3Error {
    constructor() {
        super(...arguments);
        this.code = error_codes_js_1.ERR_FORMATTERS;
    }
}
exports.FormatterError = FormatterError;
class MethodNotImplementedError extends web3_error_base_js_1.BaseWeb3Error {
    constructor() {
        super("The method you're trying to call is not implemented.");
        this.code = error_codes_js_1.ERR_METHOD_NOT_IMPLEMENTED;
    }
}
exports.MethodNotImplementedError = MethodNotImplementedError;
class OperationTimeoutError extends web3_error_base_js_1.BaseWeb3Error {
    constructor() {
        super(...arguments);
        this.code = error_codes_js_1.ERR_OPERATION_TIMEOUT;
    }
}
exports.OperationTimeoutError = OperationTimeoutError;
class OperationAbortError extends web3_error_base_js_1.BaseWeb3Error {
    constructor() {
        super(...arguments);
        this.code = error_codes_js_1.ERR_OPERATION_ABORT;
    }
}
exports.OperationAbortError = OperationAbortError;
class AbiError extends web3_error_base_js_1.BaseWeb3Error {
    constructor(message, props) {
        super(message);
        this.code = error_codes_js_1.ERR_ABI_ENCODING;
        this.props = props !== null && props !== void 0 ? props : {};
    }
}
exports.AbiError = AbiError;
class ExistingPluginNamespaceError extends web3_error_base_js_1.BaseWeb3Error {
    constructor(pluginNamespace) {
        super(`A plugin with the namespace: ${pluginNamespace} has already been registered.`);
        this.code = error_codes_js_1.ERR_EXISTING_PLUGIN_NAMESPACE;
    }
}
exports.ExistingPluginNamespaceError = ExistingPluginNamespaceError;
//# sourceMappingURL=generic_errors.js.map