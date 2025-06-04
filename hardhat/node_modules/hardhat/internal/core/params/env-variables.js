"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.getEnvHardhatArguments = exports.getEnvVariablesMap = exports.paramNameToEnvVariable = void 0;
const ArgumentsParser_1 = require("../../cli/ArgumentsParser");
const unsafe_1 = require("../../util/unsafe");
const errors_1 = require("../errors");
const errors_list_1 = require("../errors-list");
const HARDHAT_ENV_ARGUMENT_PREFIX = "HARDHAT_";
function paramNameToEnvVariable(paramName) {
    // We create it starting from the result of ArgumentsParser.paramNameToCLA
    // so it's easier to explain and understand their equivalences.
    return ArgumentsParser_1.ArgumentsParser.paramNameToCLA(paramName)
        .replace(ArgumentsParser_1.ArgumentsParser.PARAM_PREFIX, HARDHAT_ENV_ARGUMENT_PREFIX)
        .replace(/-/g, "_")
        .toUpperCase();
}
exports.paramNameToEnvVariable = paramNameToEnvVariable;
function getEnvVariablesMap(hardhatArguments) {
    const values = {};
    for (const [name, value] of Object.entries(hardhatArguments)) {
        if (value === undefined) {
            continue;
        }
        values[paramNameToEnvVariable(name)] = value.toString();
    }
    return values;
}
exports.getEnvVariablesMap = getEnvVariablesMap;
function getEnvHardhatArguments(paramDefinitions, envVariables) {
    const envArgs = {};
    for (const paramName of (0, unsafe_1.unsafeObjectKeys)(paramDefinitions)) {
        const definition = paramDefinitions[paramName];
        const envVarName = paramNameToEnvVariable(paramName);
        const rawValue = envVariables[envVarName];
        if (rawValue !== undefined) {
            try {
                envArgs[paramName] = definition.type.parse(paramName, rawValue);
            }
            catch (error) {
                if (error instanceof Error) {
                    throw new errors_1.HardhatError(errors_list_1.ERRORS.ARGUMENTS.INVALID_ENV_VAR_VALUE, {
                        varName: envVarName,
                        value: rawValue,
                    }, error);
                }
                // eslint-disable-next-line @nomicfoundation/hardhat-internal-rules/only-hardhat-error
                throw error;
            }
        }
        else {
            envArgs[paramName] = definition.defaultValue;
        }
    }
    // TODO: This is a little type-unsafe, but we know we have all the needed arguments
    return envArgs;
}
exports.getEnvHardhatArguments = getEnvHardhatArguments;
//# sourceMappingURL=env-variables.js.map