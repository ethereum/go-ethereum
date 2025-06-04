"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.validateParams = void 0;
const PathReporter_1 = require("io-ts/lib/PathReporter");
const errors_1 = require("../../../providers/errors");
/* eslint-disable @nomicfoundation/hardhat-internal-rules/only-hardhat-error */
/**
 * This function validates a list of params, throwing InvalidArgumentsError
 * if the validation fails, and returning their already-parsed types if
 * the validation succeeds.
 *
 * TODO: The type can probably be improved, removing the anys
 */
function validateParams(params, ...types) {
    if (types === undefined && params.length > 0) {
        throw new errors_1.InvalidArgumentsError(`No argument was expected and got ${params.length}`);
    }
    let optionalParams = 0;
    for (let i = types.length - 1; i >= 0; i--) {
        if (types[i].is(undefined)) {
            optionalParams += 1;
        }
        else {
            break;
        }
    }
    if (optionalParams === 0) {
        if (params.length !== types.length) {
            throw new errors_1.InvalidArgumentsError(`Expected exactly ${types.length} arguments and got ${params.length}`);
        }
    }
    else {
        if (params.length > types.length ||
            params.length < types.length - optionalParams) {
            throw new errors_1.InvalidArgumentsError(`Expected between ${types.length - optionalParams} and ${types.length} arguments and got ${params.length}`);
        }
    }
    const decoded = [];
    for (let i = 0; i < types.length; i++) {
        const result = types[i].decode(params[i]);
        if (result.isLeft()) {
            throw new errors_1.InvalidArgumentsError(`Errors encountered in param ${i}: ${PathReporter_1.PathReporter.report(result).join(", ")}`);
        }
        decoded.push(result.value);
    }
    return decoded;
}
exports.validateParams = validateParams;
//# sourceMappingURL=validation.js.map