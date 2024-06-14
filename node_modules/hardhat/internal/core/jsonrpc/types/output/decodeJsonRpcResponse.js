"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.decodeJsonRpcResponse = void 0;
const PathReporter_1 = require("io-ts/lib/PathReporter");
const errors_1 = require("../../../providers/errors");
/* eslint-disable @nomicfoundation/hardhat-internal-rules/only-hardhat-error */
/**
 * This function decodes an RPC out type, throwing InvalidResponseError if it's not valid.
 */
function decodeJsonRpcResponse(value, codec) {
    const result = codec.decode(value);
    if (result.isLeft()) {
        throw new errors_1.InvalidResponseError(`Invalid JSON-RPC response's result.

Errors: ${PathReporter_1.PathReporter.report(result).join(", ")}`);
    }
    return result.value;
}
exports.decodeJsonRpcResponse = decodeJsonRpcResponse;
//# sourceMappingURL=decodeJsonRpcResponse.js.map