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
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.any = exports.json = exports.inputFile = exports.float = exports.bigint = exports.int = exports.boolean = exports.string = void 0;
const fs = __importStar(require("fs"));
const fs_extra_1 = __importDefault(require("fs-extra"));
const errors_1 = require("../errors");
const errors_list_1 = require("../errors-list");
/**
 * String type.
 *
 * Accepts any kind of string.
 */
exports.string = {
    name: "string",
    parse: (argName, strValue) => strValue,
    /**
     * Check if argument value is of type "string"
     *
     * @param argName {string} argument's name - used for context in case of error.
     * @param value {any} argument's value to validate.
     *
     * @throws HH301 if value is not of type "string"
     */
    validate: (argName, value) => {
        const isString = typeof value === "string";
        if (!isString) {
            throw new errors_1.HardhatError(errors_list_1.ERRORS.ARGUMENTS.INVALID_VALUE_FOR_TYPE, {
                value,
                name: argName,
                type: exports.string.name,
            });
        }
    },
};
/**
 * Boolean type.
 *
 * Accepts only 'true' or 'false' (case-insensitive).
 * @throws HH301
 */
exports.boolean = {
    name: "boolean",
    parse: (argName, strValue) => {
        if (strValue.toLowerCase() === "true") {
            return true;
        }
        if (strValue.toLowerCase() === "false") {
            return false;
        }
        throw new errors_1.HardhatError(errors_list_1.ERRORS.ARGUMENTS.INVALID_VALUE_FOR_TYPE, {
            value: strValue,
            name: argName,
            type: "boolean",
        });
    },
    /**
     * Check if argument value is of type "boolean"
     *
     * @param argName {string} argument's name - used for context in case of error.
     * @param value {any} argument's value to validate.
     *
     * @throws HH301 if value is not of type "boolean"
     */
    validate: (argName, value) => {
        const isBoolean = typeof value === "boolean";
        if (!isBoolean) {
            throw new errors_1.HardhatError(errors_list_1.ERRORS.ARGUMENTS.INVALID_VALUE_FOR_TYPE, {
                value,
                name: argName,
                type: exports.boolean.name,
            });
        }
    },
};
/**
 * Int type.
 * Accepts either a decimal string integer or hexadecimal string integer.
 * @throws HH301
 */
exports.int = {
    name: "int",
    parse: (argName, strValue) => {
        const decimalPattern = /^\d+(?:[eE]\d+)?$/;
        const hexPattern = /^0[xX][\dABCDEabcde]+$/;
        if (strValue.match(decimalPattern) === null &&
            strValue.match(hexPattern) === null) {
            throw new errors_1.HardhatError(errors_list_1.ERRORS.ARGUMENTS.INVALID_VALUE_FOR_TYPE, {
                value: strValue,
                name: argName,
                type: exports.int.name,
            });
        }
        return Number(strValue);
    },
    /**
     * Check if argument value is of type "int"
     *
     * @param argName {string} argument's name - used for context in case of error.
     * @param value {any} argument's value to validate.
     *
     * @throws HH301 if value is not of type "int"
     */
    validate: (argName, value) => {
        const isInt = Number.isInteger(value);
        if (!isInt) {
            throw new errors_1.HardhatError(errors_list_1.ERRORS.ARGUMENTS.INVALID_VALUE_FOR_TYPE, {
                value,
                name: argName,
                type: exports.int.name,
            });
        }
    },
};
/**
 * BigInt type.
 * Accepts either a decimal string integer or hexadecimal string integer.
 * @throws HH301
 */
exports.bigint = {
    name: "bigint",
    parse: (argName, strValue) => {
        const decimalPattern = /^\d+(?:n)?$/;
        const hexPattern = /^0[xX][\dABCDEabcde]+$/;
        if (strValue.match(decimalPattern) === null &&
            strValue.match(hexPattern) === null) {
            throw new errors_1.HardhatError(errors_list_1.ERRORS.ARGUMENTS.INVALID_VALUE_FOR_TYPE, {
                value: strValue,
                name: argName,
                type: exports.bigint.name,
            });
        }
        return BigInt(strValue.replace("n", ""));
    },
    /**
     * Check if argument value is of type "bigint".
     *
     * @param argName {string} argument's name - used for context in case of error.
     * @param value {any} argument's value to validate.
     *
     * @throws HH301 if value is not of type "bigint"
     */
    validate: (argName, value) => {
        const isBigInt = typeof value === "bigint";
        if (!isBigInt) {
            throw new errors_1.HardhatError(errors_list_1.ERRORS.ARGUMENTS.INVALID_VALUE_FOR_TYPE, {
                value,
                name: argName,
                type: exports.bigint.name,
            });
        }
    },
};
/**
 * Float type.
 * Accepts either a decimal string number or hexadecimal string number.
 * @throws HH301
 */
exports.float = {
    name: "float",
    parse: (argName, strValue) => {
        const decimalPattern = /^(?:\d+(?:\.\d*)?|\.\d+)(?:[eE]\d+)?$/;
        const hexPattern = /^0[xX][\dABCDEabcde]+$/;
        if (strValue.match(decimalPattern) === null &&
            strValue.match(hexPattern) === null) {
            throw new errors_1.HardhatError(errors_list_1.ERRORS.ARGUMENTS.INVALID_VALUE_FOR_TYPE, {
                value: strValue,
                name: argName,
                type: exports.float.name,
            });
        }
        return Number(strValue);
    },
    /**
     * Check if argument value is of type "float".
     * Both decimal and integer number values are valid.
     *
     * @param argName {string} argument's name - used for context in case of error.
     * @param value {any} argument's value to validate.
     *
     * @throws HH301 if value is not of type "number"
     */
    validate: (argName, value) => {
        const isFloatOrInteger = typeof value === "number" && !isNaN(value);
        if (!isFloatOrInteger) {
            throw new errors_1.HardhatError(errors_list_1.ERRORS.ARGUMENTS.INVALID_VALUE_FOR_TYPE, {
                value,
                name: argName,
                type: exports.float.name,
            });
        }
    },
};
/**
 * Input file type.
 * Accepts a path to a readable file..
 * @throws HH302
 */
exports.inputFile = {
    name: "inputFile",
    parse(argName, strValue) {
        try {
            fs.accessSync(strValue, fs_extra_1.default.constants.R_OK);
            const stats = fs.lstatSync(strValue);
            if (stats.isDirectory()) {
                // This is caught and encapsulated in a hardhat error.
                // eslint-disable-next-line @nomicfoundation/hardhat-internal-rules/only-hardhat-error
                throw new Error(`${strValue} is a directory, not a file`);
            }
        }
        catch (error) {
            if (error instanceof Error) {
                throw new errors_1.HardhatError(errors_list_1.ERRORS.ARGUMENTS.INVALID_INPUT_FILE, {
                    name: argName,
                    value: strValue,
                }, error);
            }
            // eslint-disable-next-line @nomicfoundation/hardhat-internal-rules/only-hardhat-error
            throw error;
        }
        return strValue;
    },
    /**
     * Check if argument value is of type "inputFile"
     * File string validation succeeds if it can be parsed, ie. is a valid accessible file dir
     *
     * @param argName {string} argument's name - used for context in case of error.
     * @param value {any} argument's value to validate.
     *
     * @throws HH301 if value is not of type "inputFile"
     */
    validate: (argName, value) => {
        try {
            exports.inputFile.parse(argName, value);
        }
        catch (error) {
            // the input value is considered invalid, throw error.
            if (error instanceof Error) {
                throw new errors_1.HardhatError(errors_list_1.ERRORS.ARGUMENTS.INVALID_VALUE_FOR_TYPE, {
                    value,
                    name: argName,
                    type: exports.inputFile.name,
                }, error);
            }
            // eslint-disable-next-line @nomicfoundation/hardhat-internal-rules/only-hardhat-error
            throw error;
        }
    },
};
exports.json = {
    name: "json",
    parse(argName, strValue) {
        try {
            return JSON.parse(strValue);
        }
        catch (error) {
            if (error instanceof Error) {
                throw new errors_1.HardhatError(errors_list_1.ERRORS.ARGUMENTS.INVALID_JSON_ARGUMENT, {
                    param: argName,
                    error: error.message,
                }, error);
            }
            // eslint-disable-next-line @nomicfoundation/hardhat-internal-rules/only-hardhat-error
            throw error;
        }
    },
    /**
     * Check if argument value is of type "json". We consider everything except
     * undefined to be json.
     *
     * @param argName {string} argument's name - used for context in case of error.
     * @param value {any} argument's value to validate.
     *
     * @throws HH301 if value is not of type "json"
     */
    validate: (argName, value) => {
        if (value === undefined) {
            throw new errors_1.HardhatError(errors_list_1.ERRORS.ARGUMENTS.INVALID_VALUE_FOR_TYPE, {
                value,
                name: argName,
                type: exports.json.name,
            });
        }
    },
};
exports.any = {
    name: "any",
    validate(_argName, _argumentValue) { },
};
//# sourceMappingURL=argumentTypes.js.map