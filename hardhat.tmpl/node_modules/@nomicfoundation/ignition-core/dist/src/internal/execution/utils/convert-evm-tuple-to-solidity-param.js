"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.convertEvmTupleToSolidityParam = exports.convertEvmValueToSolidityParam = void 0;
const isArray_1 = __importDefault(require("lodash/isArray"));
const assertions_1 = require("../../utils/assertions");
function convertEvmValueToSolidityParam(evmValue) {
    if ((0, isArray_1.default)(evmValue)) {
        return evmValue.map(convertEvmValueToSolidityParam);
    }
    if (typeof evmValue === "object") {
        return evmValue.positional.map(convertEvmValueToSolidityParam);
    }
    return evmValue;
}
exports.convertEvmValueToSolidityParam = convertEvmValueToSolidityParam;
function convertEvmTupleToSolidityParam(evmTuple) {
    const converted = convertEvmValueToSolidityParam(evmTuple);
    (0, assertions_1.assertIgnitionInvariant)(Array.isArray(converted), "Failed to convert an EvmTuple to SolidityParameterType[]");
    return converted;
}
exports.convertEvmTupleToSolidityParam = convertEvmTupleToSolidityParam;
//# sourceMappingURL=convert-evm-tuple-to-solidity-param.js.map