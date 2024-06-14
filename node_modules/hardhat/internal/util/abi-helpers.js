"use strict";
var __createBinding = (this && this.__createBinding) || (Object.create ? (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    var desc = Object.getOwnPropertyDescriptor(m, k);
    if (!desc || ("get" in desc ? !m.__esModule : desc.writable || desc.configurable)) {
      desc = { enumerable: true, get: function() { return m[k]; } };
    }
    Object.defineProperty(o, k2, desc);
}) : (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    o[k2] = m[k];
}));
var __setModuleDefault = (this && this.__setModuleDefault) || (Object.create ? (function(o, v) {
    Object.defineProperty(o, "default", { enumerable: true, value: v });
}) : function(o, v) {
    o["default"] = v;
});
var __importStar = (this && this.__importStar) || function (mod) {
    if (mod && mod.__esModule) return mod;
    var result = {};
    if (mod != null) for (var k in mod) if (k !== "default" && Object.prototype.hasOwnProperty.call(mod, k)) __createBinding(result, mod, k);
    __setModuleDefault(result, mod);
    return result;
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.AbiHelpers = void 0;
const abi = __importStar(require("@ethersproject/abi"));
class AbiHelpers {
    /**
     * Try to compute the selector for the function/event/error
     * with the given name and param types. Return undefined
     * if it cannot do it. This can happen if some ParamType is
     * not understood by @ethersproject/abi
     */
    static computeSelector(name, inputs) {
        try {
            const fragment = abi.FunctionFragment.from({
                type: "function",
                constant: true,
                name,
                inputs: inputs.map((i) => abi.ParamType.from(i)),
            });
            const selectorHex = abi.Interface.getSighash(fragment);
            return Buffer.from(selectorHex.slice(2), "hex");
        }
        catch {
            return;
        }
    }
    static isValidCalldata(inputs, calldata) {
        try {
            abi.defaultAbiCoder.decode(inputs, calldata);
            return true;
        }
        catch {
            return false;
        }
    }
    static formatValues(values) {
        return values.map((x) => AbiHelpers._formatValue(x)).join(", ");
    }
    static _formatValue(value) {
        // print nested values as [value1, value2, ...]
        if (Array.isArray(value)) {
            return `[${value.map((v) => AbiHelpers._formatValue(v)).join(", ")}]`;
        }
        // surround string values with quotes
        if (typeof value === "string") {
            return `"${value}"`;
        }
        return value.toString();
    }
}
exports.AbiHelpers = AbiHelpers;
//# sourceMappingURL=abi-helpers.js.map