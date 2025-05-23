"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.ArgumentsParser = void 0;
const task_names_1 = require("../../builtin-tasks/task-names");
const errors_1 = require("../core/errors");
const errors_list_1 = require("../core/errors-list");
class ArgumentsParser {
    static paramNameToCLA(paramName) {
        return (ArgumentsParser.PARAM_PREFIX +
            paramName
                .split(/(?=[A-Z])/g)
                .map((s) => s.toLowerCase())
                .join("-"));
    }
    static cLAToParamName(cLA) {
        if (cLA.toLowerCase() !== cLA) {
            throw new errors_1.HardhatError(errors_list_1.ERRORS.ARGUMENTS.PARAM_NAME_INVALID_CASING, {
                param: cLA,
            });
        }
        const parts = cLA
            .slice(ArgumentsParser.PARAM_PREFIX.length)
            .split("-")
            .filter((x) => x.length > 0);
        return (parts[0] +
            parts
                .slice(1)
                .map((s) => s[0].toUpperCase() + s.slice(1))
                .join(""));
    }
    parseHardhatArguments(hardhatParamDefinitions, envVariableArguments, rawCLAs) {
        const hardhatArguments = {};
        let scopeOrTaskName;
        const allUnparsedCLAs = [];
        for (let i = 0; i < rawCLAs.length; i++) {
            const arg = rawCLAs[i];
            if (scopeOrTaskName === undefined) {
                if (!this._hasCLAParamNameFormat(arg)) {
                    scopeOrTaskName = arg;
                    allUnparsedCLAs.push(arg);
                    continue;
                }
                if (!this._isCLAParamName(arg, hardhatParamDefinitions)) {
                    throw new errors_1.HardhatError(errors_list_1.ERRORS.ARGUMENTS.UNRECOGNIZED_COMMAND_LINE_ARG, { argument: arg });
                }
                i = this._parseArgumentAt(rawCLAs, i, hardhatParamDefinitions, hardhatArguments, scopeOrTaskName);
            }
            else {
                if (!this._isCLAParamName(arg, hardhatParamDefinitions)) {
                    allUnparsedCLAs.push(arg);
                    continue;
                }
                i = this._parseArgumentAt(rawCLAs, i, hardhatParamDefinitions, hardhatArguments, scopeOrTaskName);
            }
        }
        return {
            hardhatArguments: this._addHardhatDefaultArguments(hardhatParamDefinitions, envVariableArguments, hardhatArguments),
            scopeOrTaskName,
            allUnparsedCLAs,
        };
    }
    parseScopeAndTaskNames(allUnparsedCLAs, taskDefinitions, scopeDefinitions) {
        const [firstCLA, secondCLA] = allUnparsedCLAs;
        if (allUnparsedCLAs.length === 0) {
            return {
                taskName: task_names_1.TASK_HELP,
                unparsedCLAs: [],
            };
        }
        else if (allUnparsedCLAs.length === 1) {
            if (scopeDefinitions[firstCLA] !== undefined) {
                // this is a bit of a hack, but it's the easiest way to print
                // the help of a scope when no task is specified
                return {
                    taskName: task_names_1.TASK_HELP,
                    unparsedCLAs: [firstCLA],
                };
            }
            else if (taskDefinitions[firstCLA] !== undefined) {
                return {
                    taskName: firstCLA,
                    unparsedCLAs: allUnparsedCLAs.slice(1),
                };
            }
            else {
                throw new errors_1.HardhatError(errors_list_1.ERRORS.ARGUMENTS.UNRECOGNIZED_TASK, {
                    task: firstCLA,
                });
            }
        }
        else {
            const scopeDefinition = scopeDefinitions[firstCLA];
            if (scopeDefinition !== undefined) {
                if (scopeDefinition.tasks[secondCLA] !== undefined) {
                    return {
                        scopeName: firstCLA,
                        taskName: secondCLA,
                        unparsedCLAs: allUnparsedCLAs.slice(2),
                    };
                }
                else {
                    throw new errors_1.HardhatError(errors_list_1.ERRORS.ARGUMENTS.UNRECOGNIZED_SCOPED_TASK, {
                        scope: firstCLA,
                        task: secondCLA,
                    });
                }
            }
            else if (taskDefinitions[firstCLA] !== undefined) {
                return {
                    taskName: firstCLA,
                    unparsedCLAs: allUnparsedCLAs.slice(1),
                };
            }
            else {
                throw new errors_1.HardhatError(errors_list_1.ERRORS.ARGUMENTS.UNRECOGNIZED_TASK, {
                    task: firstCLA,
                });
            }
        }
    }
    parseTaskArguments(taskDefinition, rawCLAs) {
        const { paramArguments, rawPositionalArguments } = this._parseTaskParamArguments(taskDefinition, rawCLAs);
        const positionalArguments = this._parsePositionalParamArgs(rawPositionalArguments, taskDefinition.positionalParamDefinitions);
        return { ...paramArguments, ...positionalArguments };
    }
    _parseTaskParamArguments(taskDefinition, rawCLAs) {
        const paramArguments = {};
        const rawPositionalArguments = [];
        for (let i = 0; i < rawCLAs.length; i++) {
            const arg = rawCLAs[i];
            if (!this._hasCLAParamNameFormat(arg)) {
                rawPositionalArguments.push(arg);
                continue;
            }
            if (!this._isCLAParamName(arg, taskDefinition.paramDefinitions)) {
                throw new errors_1.HardhatError(errors_list_1.ERRORS.ARGUMENTS.UNRECOGNIZED_PARAM_NAME, {
                    param: arg,
                });
            }
            i = this._parseArgumentAt(rawCLAs, i, taskDefinition.paramDefinitions, paramArguments, taskDefinition.name);
        }
        this._addTaskDefaultArguments(taskDefinition, paramArguments);
        return { paramArguments, rawPositionalArguments };
    }
    _addHardhatDefaultArguments(hardhatParamDefinitions, envVariableArguments, hardhatArguments) {
        return {
            ...envVariableArguments,
            ...hardhatArguments,
        };
    }
    _addTaskDefaultArguments(taskDefinition, taskArguments) {
        for (const paramName of Object.keys(taskDefinition.paramDefinitions)) {
            const definition = taskDefinition.paramDefinitions[paramName];
            if (taskArguments[paramName] !== undefined) {
                continue;
            }
            if (!definition.isOptional) {
                throw new errors_1.HardhatError(errors_list_1.ERRORS.ARGUMENTS.MISSING_TASK_ARGUMENT, {
                    param: ArgumentsParser.paramNameToCLA(paramName),
                    task: taskDefinition.name,
                });
            }
            taskArguments[paramName] = definition.defaultValue;
        }
    }
    _isCLAParamName(str, paramDefinitions) {
        if (!this._hasCLAParamNameFormat(str)) {
            return false;
        }
        const name = ArgumentsParser.cLAToParamName(str);
        return paramDefinitions[name] !== undefined;
    }
    _hasCLAParamNameFormat(str) {
        return str.startsWith(ArgumentsParser.PARAM_PREFIX);
    }
    _parseArgumentAt(rawCLAs, index, paramDefinitions, parsedArguments, scopeOrTaskName) {
        const claArg = rawCLAs[index];
        const paramName = ArgumentsParser.cLAToParamName(claArg);
        const definition = paramDefinitions[paramName];
        if (parsedArguments[paramName] !== undefined) {
            throw new errors_1.HardhatError(errors_list_1.ERRORS.ARGUMENTS.REPEATED_PARAM, {
                param: claArg,
            });
        }
        if (definition.isFlag) {
            parsedArguments[paramName] = true;
        }
        else {
            index++;
            const value = rawCLAs[index];
            if (value === undefined) {
                throw new errors_1.HardhatError(errors_list_1.ERRORS.ARGUMENTS.MISSING_TASK_ARGUMENT, {
                    param: ArgumentsParser.paramNameToCLA(paramName),
                    task: scopeOrTaskName ?? "help",
                });
            }
            // We only parse the arguments of non-subtasks, and those only
            // accept CLIArgumentTypes.
            const type = definition.type;
            parsedArguments[paramName] = type.parse(paramName, value);
        }
        return index;
    }
    _parsePositionalParamArgs(rawPositionalParamArgs, positionalParamDefinitions) {
        const args = {};
        for (let i = 0; i < positionalParamDefinitions.length; i++) {
            const definition = positionalParamDefinitions[i];
            // We only parse the arguments of non-subtasks, and those only
            // accept CLIArgumentTypes.
            const type = definition.type;
            const rawArg = rawPositionalParamArgs[i];
            if (rawArg === undefined) {
                if (!definition.isOptional) {
                    throw new errors_1.HardhatError(errors_list_1.ERRORS.ARGUMENTS.MISSING_POSITIONAL_ARG, {
                        param: definition.name,
                    });
                }
                args[definition.name] = definition.defaultValue;
            }
            else if (!definition.isVariadic) {
                args[definition.name] = type.parse(definition.name, rawArg);
            }
            else {
                args[definition.name] = rawPositionalParamArgs
                    .slice(i)
                    .map((raw) => type.parse(definition.name, raw));
            }
        }
        const lastDefinition = positionalParamDefinitions[positionalParamDefinitions.length - 1];
        const hasVariadicParam = lastDefinition !== undefined && lastDefinition.isVariadic;
        if (!hasVariadicParam &&
            rawPositionalParamArgs.length > positionalParamDefinitions.length) {
            throw new errors_1.HardhatError(errors_list_1.ERRORS.ARGUMENTS.UNRECOGNIZED_POSITIONAL_ARG, {
                argument: rawPositionalParamArgs[positionalParamDefinitions.length],
            });
        }
        return args;
    }
}
ArgumentsParser.PARAM_PREFIX = "--";
exports.ArgumentsParser = ArgumentsParser;
//# sourceMappingURL=ArgumentsParser.js.map