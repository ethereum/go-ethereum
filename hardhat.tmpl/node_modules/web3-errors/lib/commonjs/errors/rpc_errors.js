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
exports.rpcErrorsMap = exports.LimitExceededError = exports.TransactionRejectedError = exports.VersionNotSupportedError = exports.ResourcesNotFoundError = exports.ResourceUnavailableError = exports.MethodNotSupported = exports.InvalidInputError = exports.InternalError = exports.InvalidParamsError = exports.MethodNotFoundError = exports.InvalidRequestError = exports.ParseError = exports.EIP1193ProviderRpcError = exports.RpcError = void 0;
const web3_error_base_js_1 = require("../web3_error_base.js");
const error_codes_js_1 = require("../error_codes.js");
const rpc_error_messages_js_1 = require("./rpc_error_messages.js");
class RpcError extends web3_error_base_js_1.BaseWeb3Error {
    constructor(rpcError, message) {
        super(message !== null && message !== void 0 ? message : rpc_error_messages_js_1.genericRpcErrorMessageTemplate.replace('*code*', rpcError.error.code.toString()));
        this.code = rpcError.error.code;
        this.id = rpcError.id;
        this.jsonrpc = rpcError.jsonrpc;
        this.jsonRpcError = rpcError.error;
    }
    toJSON() {
        return Object.assign(Object.assign({}, super.toJSON()), { error: this.jsonRpcError, id: this.id, jsonRpc: this.jsonrpc });
    }
}
exports.RpcError = RpcError;
class EIP1193ProviderRpcError extends web3_error_base_js_1.BaseWeb3Error {
    constructor(code, data) {
        var _a, _b, _c, _d;
        if (!code) {
            // this case should ideally not happen
            super();
        }
        else if ((_a = rpc_error_messages_js_1.RpcErrorMessages[code]) === null || _a === void 0 ? void 0 : _a.message) {
            super(rpc_error_messages_js_1.RpcErrorMessages[code].message);
        }
        else {
            // Retrieve the status code object for the given code from the table, by searching through the appropriate range
            const statusCodeRange = Object.keys(rpc_error_messages_js_1.RpcErrorMessages).find(statusCode => typeof statusCode === 'string' &&
                code >= parseInt(statusCode.split('-')[0], 10) &&
                code <= parseInt(statusCode.split('-')[1], 10));
            super((_c = (_b = rpc_error_messages_js_1.RpcErrorMessages[statusCodeRange !== null && statusCodeRange !== void 0 ? statusCodeRange : '']) === null || _b === void 0 ? void 0 : _b.message) !== null && _c !== void 0 ? _c : rpc_error_messages_js_1.genericRpcErrorMessageTemplate.replace('*code*', (_d = code === null || code === void 0 ? void 0 : code.toString()) !== null && _d !== void 0 ? _d : '""'));
        }
        this.code = code;
        this.data = data;
    }
}
exports.EIP1193ProviderRpcError = EIP1193ProviderRpcError;
class ParseError extends RpcError {
    constructor(rpcError) {
        super(rpcError, rpc_error_messages_js_1.RpcErrorMessages[error_codes_js_1.ERR_RPC_INVALID_JSON].message);
        this.code = error_codes_js_1.ERR_RPC_INVALID_JSON;
    }
}
exports.ParseError = ParseError;
class InvalidRequestError extends RpcError {
    constructor(rpcError) {
        super(rpcError, rpc_error_messages_js_1.RpcErrorMessages[error_codes_js_1.ERR_RPC_INVALID_REQUEST].message);
        this.code = error_codes_js_1.ERR_RPC_INVALID_REQUEST;
    }
}
exports.InvalidRequestError = InvalidRequestError;
class MethodNotFoundError extends RpcError {
    constructor(rpcError) {
        super(rpcError, rpc_error_messages_js_1.RpcErrorMessages[error_codes_js_1.ERR_RPC_INVALID_METHOD].message);
        this.code = error_codes_js_1.ERR_RPC_INVALID_METHOD;
    }
}
exports.MethodNotFoundError = MethodNotFoundError;
class InvalidParamsError extends RpcError {
    constructor(rpcError) {
        super(rpcError, rpc_error_messages_js_1.RpcErrorMessages[error_codes_js_1.ERR_RPC_INVALID_PARAMS].message);
        this.code = error_codes_js_1.ERR_RPC_INVALID_PARAMS;
    }
}
exports.InvalidParamsError = InvalidParamsError;
class InternalError extends RpcError {
    constructor(rpcError) {
        super(rpcError, rpc_error_messages_js_1.RpcErrorMessages[error_codes_js_1.ERR_RPC_INTERNAL_ERROR].message);
        this.code = error_codes_js_1.ERR_RPC_INTERNAL_ERROR;
    }
}
exports.InternalError = InternalError;
class InvalidInputError extends RpcError {
    constructor(rpcError) {
        super(rpcError, rpc_error_messages_js_1.RpcErrorMessages[error_codes_js_1.ERR_RPC_INVALID_INPUT].message);
        this.code = error_codes_js_1.ERR_RPC_INVALID_INPUT;
    }
}
exports.InvalidInputError = InvalidInputError;
class MethodNotSupported extends RpcError {
    constructor(rpcError) {
        super(rpcError, rpc_error_messages_js_1.RpcErrorMessages[error_codes_js_1.ERR_RPC_UNSUPPORTED_METHOD].message);
        this.code = error_codes_js_1.ERR_RPC_UNSUPPORTED_METHOD;
    }
}
exports.MethodNotSupported = MethodNotSupported;
class ResourceUnavailableError extends RpcError {
    constructor(rpcError) {
        super(rpcError, rpc_error_messages_js_1.RpcErrorMessages[error_codes_js_1.ERR_RPC_UNAVAILABLE_RESOURCE].message);
        this.code = error_codes_js_1.ERR_RPC_UNAVAILABLE_RESOURCE;
    }
}
exports.ResourceUnavailableError = ResourceUnavailableError;
class ResourcesNotFoundError extends RpcError {
    constructor(rpcError) {
        super(rpcError, rpc_error_messages_js_1.RpcErrorMessages[error_codes_js_1.ERR_RPC_MISSING_RESOURCE].message);
        this.code = error_codes_js_1.ERR_RPC_MISSING_RESOURCE;
    }
}
exports.ResourcesNotFoundError = ResourcesNotFoundError;
class VersionNotSupportedError extends RpcError {
    constructor(rpcError) {
        super(rpcError, rpc_error_messages_js_1.RpcErrorMessages[error_codes_js_1.ERR_RPC_NOT_SUPPORTED].message);
        this.code = error_codes_js_1.ERR_RPC_NOT_SUPPORTED;
    }
}
exports.VersionNotSupportedError = VersionNotSupportedError;
class TransactionRejectedError extends RpcError {
    constructor(rpcError) {
        super(rpcError, rpc_error_messages_js_1.RpcErrorMessages[error_codes_js_1.ERR_RPC_TRANSACTION_REJECTED].message);
        this.code = error_codes_js_1.ERR_RPC_TRANSACTION_REJECTED;
    }
}
exports.TransactionRejectedError = TransactionRejectedError;
class LimitExceededError extends RpcError {
    constructor(rpcError) {
        super(rpcError, rpc_error_messages_js_1.RpcErrorMessages[error_codes_js_1.ERR_RPC_LIMIT_EXCEEDED].message);
        this.code = error_codes_js_1.ERR_RPC_LIMIT_EXCEEDED;
    }
}
exports.LimitExceededError = LimitExceededError;
exports.rpcErrorsMap = new Map();
exports.rpcErrorsMap.set(error_codes_js_1.ERR_RPC_INVALID_JSON, { error: ParseError });
exports.rpcErrorsMap.set(error_codes_js_1.ERR_RPC_INVALID_REQUEST, {
    error: InvalidRequestError,
});
exports.rpcErrorsMap.set(error_codes_js_1.ERR_RPC_INVALID_METHOD, {
    error: MethodNotFoundError,
});
exports.rpcErrorsMap.set(error_codes_js_1.ERR_RPC_INVALID_PARAMS, { error: InvalidParamsError });
exports.rpcErrorsMap.set(error_codes_js_1.ERR_RPC_INTERNAL_ERROR, { error: InternalError });
exports.rpcErrorsMap.set(error_codes_js_1.ERR_RPC_INVALID_INPUT, { error: InvalidInputError });
exports.rpcErrorsMap.set(error_codes_js_1.ERR_RPC_UNSUPPORTED_METHOD, {
    error: MethodNotSupported,
});
exports.rpcErrorsMap.set(error_codes_js_1.ERR_RPC_UNAVAILABLE_RESOURCE, {
    error: ResourceUnavailableError,
});
exports.rpcErrorsMap.set(error_codes_js_1.ERR_RPC_TRANSACTION_REJECTED, {
    error: TransactionRejectedError,
});
exports.rpcErrorsMap.set(error_codes_js_1.ERR_RPC_MISSING_RESOURCE, {
    error: ResourcesNotFoundError,
});
exports.rpcErrorsMap.set(error_codes_js_1.ERR_RPC_NOT_SUPPORTED, {
    error: VersionNotSupportedError,
});
exports.rpcErrorsMap.set(error_codes_js_1.ERR_RPC_LIMIT_EXCEEDED, { error: LimitExceededError });
//# sourceMappingURL=rpc_errors.js.map