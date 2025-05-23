"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.HelpPrinter = void 0;
const errors_1 = require("../core/errors");
const errors_list_1 = require("../core/errors-list");
const ArgumentsParser_1 = require("./ArgumentsParser");
class HelpPrinter {
    constructor(_programName, _executableName, _version, _hardhatParamDefinitions, _tasks, _scopes) {
        this._programName = _programName;
        this._executableName = _executableName;
        this._version = _version;
        this._hardhatParamDefinitions = _hardhatParamDefinitions;
        this._tasks = _tasks;
        this._scopes = _scopes;
    }
    printGlobalHelp(includeSubtasks = false) {
        console.log(`${this._programName} version ${this._version}\n`);
        console.log(`Usage: ${this._executableName} [GLOBAL OPTIONS] [SCOPE] <TASK> [TASK OPTIONS]\n`);
        console.log("GLOBAL OPTIONS:\n");
        let length = this._printParamDetails(this._hardhatParamDefinitions);
        console.log("\n\nAVAILABLE TASKS:\n");
        length = this._printTasks(this._tasks, includeSubtasks, length);
        if (Object.keys(this._scopes).length > 0) {
            console.log("\n\nAVAILABLE TASK SCOPES:\n");
            this._printScopes(this._scopes, length);
        }
        console.log("");
        console.log(`To get help for a specific task run: npx ${this._executableName} help [SCOPE] <TASK>\n`);
    }
    printScopeHelp(scopeDefinition, includeSubtasks = false) {
        const name = scopeDefinition.name;
        const description = scopeDefinition.description ?? "";
        console.log(`${this._programName} version ${this._version}`);
        console.log(`\nUsage: hardhat [GLOBAL OPTIONS] ${name} <TASK> [TASK OPTIONS]`);
        console.log(`\nAVAILABLE TASKS:\n`);
        if (this._scopes[name] === undefined) {
            throw new errors_1.HardhatError(errors_list_1.ERRORS.ARGUMENTS.UNRECOGNIZED_SCOPE, {
                scope: name,
            });
        }
        this._printTasks(this._scopes[name].tasks, includeSubtasks);
        console.log(`\n${name}: ${description}`);
        console.log(`\nFor global options help run: ${this._executableName} help\n`);
    }
    printTaskHelp(taskDefinition) {
        const { description = "", name, paramDefinitions, positionalParamDefinitions, } = taskDefinition;
        console.log(`${this._programName} version ${this._version}\n`);
        const paramsList = this._getParamsList(paramDefinitions);
        const positionalParamsList = this._getPositionalParamsList(positionalParamDefinitions);
        const scope = taskDefinition.scope !== undefined ? `${taskDefinition.scope} ` : "";
        console.log(`Usage: ${this._executableName} [GLOBAL OPTIONS] ${scope}${name}${paramsList}${positionalParamsList}\n`);
        if (Object.keys(paramDefinitions).length > 0) {
            console.log("OPTIONS:\n");
            this._printParamDetails(paramDefinitions);
            console.log("");
        }
        if (positionalParamDefinitions.length > 0) {
            console.log("POSITIONAL ARGUMENTS:\n");
            this._printPositionalParamDetails(positionalParamDefinitions);
            console.log("");
        }
        console.log(`${name}: ${description}\n`);
        console.log(`For global options help run: ${this._executableName} help\n`);
    }
    _printTasks(tasksMap, includeSubtasks, length = 0) {
        const taskNameList = Object.entries(tasksMap)
            .filter(([, taskDefinition]) => includeSubtasks || !taskDefinition.isSubtask)
            .map(([taskName]) => taskName)
            .sort();
        const nameLength = taskNameList
            .map((n) => n.length)
            .reduce((a, b) => Math.max(a, b), length);
        for (const name of taskNameList) {
            const { description = "" } = tasksMap[name];
            console.log(`  ${name.padEnd(nameLength)}\t${description}`);
        }
        return nameLength;
    }
    _printScopes(scopesMap, length) {
        const scopeNamesList = Object.entries(scopesMap)
            .map(([scopeName]) => scopeName)
            .sort();
        const nameLength = scopeNamesList
            .map((n) => n.length)
            .reduce((a, b) => Math.max(a, b), length);
        for (const name of scopeNamesList) {
            const { description = "" } = scopesMap[name];
            console.log(`  ${name.padEnd(nameLength)}\t${description}`);
        }
        return nameLength;
    }
    _getParamValueDescription(paramDefinition) {
        return `<${paramDefinition.type.name.toUpperCase()}>`;
    }
    _getParamsList(paramDefinitions) {
        let paramsList = "";
        for (const name of Object.keys(paramDefinitions).sort()) {
            const definition = paramDefinitions[name];
            const { isFlag, isOptional } = definition;
            paramsList += " ";
            if (isOptional) {
                paramsList += "[";
            }
            paramsList += `${ArgumentsParser_1.ArgumentsParser.paramNameToCLA(name)}`;
            if (!isFlag) {
                paramsList += ` ${this._getParamValueDescription(definition)}`;
            }
            if (isOptional) {
                paramsList += "]";
            }
        }
        return paramsList;
    }
    _getPositionalParamsList(positionalParamDefinitions) {
        let paramsList = "";
        for (const definition of positionalParamDefinitions) {
            const { isOptional, isVariadic, name } = definition;
            paramsList += " ";
            if (isOptional) {
                paramsList += "[";
            }
            if (isVariadic) {
                paramsList += "...";
            }
            paramsList += name;
            if (isOptional) {
                paramsList += "]";
            }
        }
        return paramsList;
    }
    _printParamDetails(paramDefinitions) {
        const paramsNameLength = Object.keys(paramDefinitions)
            .map((n) => ArgumentsParser_1.ArgumentsParser.paramNameToCLA(n).length)
            .reduce((a, b) => Math.max(a, b), 0);
        for (const name of Object.keys(paramDefinitions).sort()) {
            const { description, defaultValue, isOptional, isFlag } = paramDefinitions[name];
            let msg = `  ${ArgumentsParser_1.ArgumentsParser.paramNameToCLA(name).padEnd(paramsNameLength)}\t`;
            if (description !== undefined) {
                msg += `${description} `;
            }
            if (isOptional && defaultValue !== undefined && !isFlag) {
                msg += `(default: ${JSON.stringify(defaultValue)})`;
            }
            console.log(msg);
        }
        return paramsNameLength;
    }
    _printPositionalParamDetails(positionalParamDefinitions) {
        const paramsNameLength = positionalParamDefinitions
            .map((d) => d.name.length)
            .reduce((a, b) => Math.max(a, b), 0);
        for (const definition of positionalParamDefinitions) {
            const { name, description, isOptional, defaultValue } = definition;
            let msg = `  ${name.padEnd(paramsNameLength)}\t`;
            if (description !== undefined) {
                msg += `${description} `;
            }
            if (isOptional && defaultValue !== undefined) {
                msg += `(default: ${JSON.stringify(defaultValue)})`;
            }
            console.log(msg);
        }
    }
}
exports.HelpPrinter = HelpPrinter;
//# sourceMappingURL=HelpPrinter.js.map