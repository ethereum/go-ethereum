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
exports.SimpleScopeDefinition = exports.OverriddenTaskDefinition = exports.SimpleTaskDefinition = void 0;
const errors_1 = require("../errors");
const errors_list_1 = require("../errors-list");
const types = __importStar(require("../params/argumentTypes"));
const hardhat_params_1 = require("../params/hardhat-params");
const util_1 = require("./util");
function isCLIArgumentType(type) {
    return "parse" in type;
}
/**
 * This class creates a task definition, which consists of:
 * * a name, that should be unique and will be used to call the task.
 * * a description. This is optional.
 * * the action that the task will execute.
 * * a set of parameters that can be used by the action.
 *
 */
class SimpleTaskDefinition {
    get name() {
        return this._task;
    }
    get scope() {
        return this._scope;
    }
    get description() {
        return this._description;
    }
    /**
     * Creates an empty task definition.
     *
     * This definition will have no params, and will throw a HH205 if executed.
     *
     * @param taskIdentifier The task's identifier.
     * @param isSubtask `true` if the task is a subtask, `false` otherwise.
     */
    constructor(taskIdentifier, isSubtask = false) {
        this.isSubtask = isSubtask;
        this.paramDefinitions = {};
        this.positionalParamDefinitions = [];
        this._positionalParamNames = new Set();
        this._hasVariadicParam = false;
        this._hasOptionalPositionalParam = false;
        const { scope, task } = (0, util_1.parseTaskIdentifier)(taskIdentifier);
        this._scope = scope;
        this._task = task;
        this.action = () => {
            throw new errors_1.HardhatError(errors_list_1.ERRORS.TASK_DEFINITIONS.ACTION_NOT_SET, {
                taskName: this._task,
            });
        };
    }
    /**
     * Sets the task's description.
     * @param description The description.
     */
    setDescription(description) {
        this._description = description;
        return this;
    }
    /**
     * Sets the task's action.
     * @param action The action.
     */
    setAction(action) {
        // TODO: There's probably something bad here. See types.ts for more info.
        this.action = action;
        return this;
    }
    /**
     * Adds a parameter to the task's definition.
     *
     * @remarks This will throw if the `name` is already used by this task or
     * by Hardhat's global parameters.
     *
     * @param name The parameter's name.
     * @param description The parameter's description.
     * @param defaultValue A default value. This must be `undefined` if `isOptional` is `true`.
     * @param type The param's `ArgumentType`. It will parse and validate the user's input.
     * @param isOptional `true` if the parameter is optional. It's default value is `true` if `defaultValue` is not `undefined`.
     */
    addParam(name, description, defaultValue, type, isOptional = defaultValue !== undefined) {
        if (type === undefined) {
            if (defaultValue === undefined) {
                return this.addParam(name, description, undefined, types.string, isOptional);
            }
            if (typeof defaultValue !== "string") {
                throw new errors_1.HardhatError(errors_list_1.ERRORS.TASK_DEFINITIONS.DEFAULT_VALUE_WRONG_TYPE, {
                    paramName: name,
                    taskName: this.name,
                });
            }
            return this.addParam(name, description, defaultValue, types.string, isOptional);
        }
        this._validateParamNameCasing(name);
        this._validateNameNotUsed(name);
        this._validateNoDefaultValueForMandatoryParam(defaultValue, isOptional, name);
        this._validateCLIArgumentTypesForExternalTasks(type);
        this.paramDefinitions[name] = {
            name,
            defaultValue,
            type,
            description,
            isOptional,
            isFlag: false,
            isVariadic: false,
        };
        return this;
    }
    /**
     * Adds an optional parameter to the task's definition.
     *
     * @see addParam.
     *
     * @param name the parameter's name.
     * @param description the parameter's description.
     * @param defaultValue a default value.
     * @param type param's type.
     */
    addOptionalParam(name, description, defaultValue, type) {
        return this.addParam(name, description, defaultValue, type, true);
    }
    /**
     * Adds a boolean parameter or flag to the task's definition.
     *
     * Flags are params with default value set to `false`, and that don't expect
     * values to be set in the CLI. A normal boolean param must be called with
     * `--param true`, while a flag is called with `--flag`.
     *
     * @param name the parameter's name.
     * @param description the parameter's description.
     */
    addFlag(name, description) {
        this._validateParamNameCasing(name);
        this._validateNameNotUsed(name);
        this.paramDefinitions[name] = {
            name,
            defaultValue: false,
            type: types.boolean,
            description,
            isFlag: true,
            isOptional: true,
            isVariadic: false,
        };
        return this;
    }
    /**
     * Adds a positional parameter to the task's definition.
     *
     * @remarks This will throw if the `name` is already used by this task or
     * by Hardhat's global parameters.
     * @remarks This will throw if `isOptional` is `false` and an optional positional
     * param was already set.
     * @remarks This will throw if a variadic positional param is already set.
     *
     * @param name The parameter's name.
     * @param description The parameter's description.
     * @param defaultValue A default value. This must be `undefined` if `isOptional` is `true`.
     * @param type The param's `ArgumentType`. It will parse and validate the user's input.
     * @param isOptional `true` if the parameter is optional. It's default value is `true` if `defaultValue` is not `undefined`.
     */
    addPositionalParam(name, description, defaultValue, type, isOptional = defaultValue !== undefined) {
        if (type === undefined) {
            if (defaultValue === undefined) {
                return this.addPositionalParam(name, description, undefined, types.string, isOptional);
            }
            if (typeof defaultValue !== "string") {
                throw new errors_1.HardhatError(errors_list_1.ERRORS.TASK_DEFINITIONS.DEFAULT_VALUE_WRONG_TYPE, {
                    paramName: name,
                    taskName: this.name,
                });
            }
            return this.addPositionalParam(name, description, defaultValue, types.string, isOptional);
        }
        this._validateParamNameCasing(name);
        this._validateNameNotUsed(name);
        this._validateNotAfterVariadicParam(name);
        this._validateNoMandatoryParamAfterOptionalOnes(name, isOptional);
        this._validateNoDefaultValueForMandatoryParam(defaultValue, isOptional, name);
        this._validateCLIArgumentTypesForExternalTasks(type);
        const definition = {
            name,
            defaultValue,
            type,
            description,
            isVariadic: false,
            isOptional,
            isFlag: false,
        };
        this._addPositionalParamDefinition(definition);
        return this;
    }
    /**
     * Adds an optional positional parameter to the task's definition.
     *
     * @see addPositionalParam.
     *
     * @param name the parameter's name.
     * @param description the parameter's description.
     * @param defaultValue a default value.
     * @param type param's type.
     */
    addOptionalPositionalParam(name, description, defaultValue, type) {
        return this.addPositionalParam(name, description, defaultValue, type, true);
    }
    /**
     * Adds a variadic positional parameter to the task's definition. Variadic
     * positional params act as `...rest` parameters in JavaScript.
     *
     * @param name The parameter's name.
     * @param description The parameter's description.
     * @param defaultValue A default value. This must be `undefined` if `isOptional` is `true`.
     * @param type The param's `ArgumentType`. It will parse and validate the user's input.
     * @param isOptional `true` if the parameter is optional. It's default value is `true` if `defaultValue` is not `undefined`.
     */
    addVariadicPositionalParam(name, description, defaultValue, type, isOptional = defaultValue !== undefined) {
        if (defaultValue !== undefined && !Array.isArray(defaultValue)) {
            defaultValue = [defaultValue];
        }
        if (type === undefined) {
            if (defaultValue === undefined) {
                return this.addVariadicPositionalParam(name, description, undefined, types.string, isOptional);
            }
            if (!this._isStringArray(defaultValue)) {
                throw new errors_1.HardhatError(errors_list_1.ERRORS.TASK_DEFINITIONS.DEFAULT_VALUE_WRONG_TYPE, {
                    paramName: name,
                    taskName: this.name,
                });
            }
            return this.addVariadicPositionalParam(name, description, defaultValue, types.string, isOptional);
        }
        this._validateParamNameCasing(name);
        this._validateNameNotUsed(name);
        this._validateNotAfterVariadicParam(name);
        this._validateNoMandatoryParamAfterOptionalOnes(name, isOptional);
        this._validateNoDefaultValueForMandatoryParam(defaultValue, isOptional, name);
        this._validateCLIArgumentTypesForExternalTasks(type);
        const definition = {
            name,
            defaultValue,
            type,
            description,
            isVariadic: true,
            isOptional,
            isFlag: false,
        };
        this._addPositionalParamDefinition(definition);
        return this;
    }
    /**
     * Adds a positional parameter to the task's definition.
     *
     * This will check if the `name` is already used and
     * if the parameter is being added after a varidic argument.
     *
     * @param name the parameter's name.
     * @param description the parameter's description.
     * @param defaultValue a default value.
     * @param type param's type.
     */
    addOptionalVariadicPositionalParam(name, description, defaultValue, type) {
        return this.addVariadicPositionalParam(name, description, defaultValue, type, true);
    }
    /**
     * Adds a positional parameter to the task's definition.
     *
     * @param definition the param's definition
     */
    _addPositionalParamDefinition(definition) {
        if (definition.isVariadic) {
            this._hasVariadicParam = true;
        }
        if (definition.isOptional) {
            this._hasOptionalPositionalParam = true;
        }
        this._positionalParamNames.add(definition.name);
        this.positionalParamDefinitions.push(definition);
    }
    /**
     * Validates if the given param's name is after a variadic parameter.
     * @param name the param's name.
     * @throws HH200
     */
    _validateNotAfterVariadicParam(name) {
        if (this._hasVariadicParam) {
            throw new errors_1.HardhatError(errors_list_1.ERRORS.TASK_DEFINITIONS.PARAM_AFTER_VARIADIC, {
                paramName: name,
                taskName: this.name,
            });
        }
    }
    /**
     * Validates if the param's name is already used.
     * @param name the param's name.
     *
     * @throws HH201 if `name` is already used as a param.
     * @throws HH202 if `name` is already used as a param by Hardhat
     */
    _validateNameNotUsed(name) {
        if (this._hasParamDefined(name)) {
            throw new errors_1.HardhatError(errors_list_1.ERRORS.TASK_DEFINITIONS.PARAM_ALREADY_DEFINED, {
                paramName: name,
                taskName: this.name,
            });
        }
        if (Object.keys(hardhat_params_1.HARDHAT_PARAM_DEFINITIONS).includes(name)) {
            throw new errors_1.HardhatError(errors_list_1.ERRORS.TASK_DEFINITIONS.PARAM_CLASHES_WITH_HARDHAT_PARAM, {
                paramName: name,
                taskName: this.name,
            });
        }
    }
    /**
     * Checks if the given name is already used.
     * @param name the param's name.
     */
    _hasParamDefined(name) {
        return (this.paramDefinitions[name] !== undefined ||
            this._positionalParamNames.has(name));
    }
    /**
     * Validates if a mandatory param is being added after optional params.
     *
     * @param name the param's name to be added.
     * @param isOptional true if the new param is optional, false otherwise.
     *
     * @throws HH203 if validation fail
     */
    _validateNoMandatoryParamAfterOptionalOnes(name, isOptional) {
        if (!isOptional && this._hasOptionalPositionalParam) {
            throw new errors_1.HardhatError(errors_list_1.ERRORS.TASK_DEFINITIONS.MANDATORY_PARAM_AFTER_OPTIONAL, {
                paramName: name,
                taskName: this.name,
            });
        }
    }
    _validateParamNameCasing(name) {
        const pattern = /^[a-z]+([a-zA-Z0-9])*$/;
        const match = name.match(pattern);
        if (match === null) {
            throw new errors_1.HardhatError(errors_list_1.ERRORS.TASK_DEFINITIONS.INVALID_PARAM_NAME_CASING, {
                paramName: name,
                taskName: this.name,
            });
        }
    }
    _validateNoDefaultValueForMandatoryParam(defaultValue, isOptional, name) {
        if (defaultValue !== undefined && !isOptional) {
            throw new errors_1.HardhatError(errors_list_1.ERRORS.TASK_DEFINITIONS.DEFAULT_IN_MANDATORY_PARAM, {
                paramName: name,
                taskName: this.name,
            });
        }
    }
    _isStringArray(values) {
        return Array.isArray(values) && values.every((v) => typeof v === "string");
    }
    _validateCLIArgumentTypesForExternalTasks(type) {
        if (this.isSubtask) {
            return;
        }
        if (!isCLIArgumentType(type)) {
            throw new errors_1.HardhatError(errors_list_1.ERRORS.TASK_DEFINITIONS.CLI_ARGUMENT_TYPE_REQUIRED, {
                task: this.name,
                type: type.name,
            });
        }
    }
}
exports.SimpleTaskDefinition = SimpleTaskDefinition;
/**
 * Allows you to override a previously defined task.
 *
 * When overriding a task you can:
 *  * flag it as a subtask
 *  * set a new description
 *  * set a new action
 *
 */
class OverriddenTaskDefinition {
    constructor(parentTaskDefinition, isSubtask = false) {
        this.parentTaskDefinition = parentTaskDefinition;
        this.isSubtask = isSubtask;
        this.isSubtask = isSubtask;
        this.parentTaskDefinition = parentTaskDefinition;
    }
    /**
     * Sets the task's description.
     * @param description The description.
     */
    setDescription(description) {
        this._description = description;
        return this;
    }
    /**
     * Overrides the parent task's action.
     * @param action the action.
     */
    setAction(action) {
        // TODO: There's probably something bad here. See types.ts for more info.
        this._action = action;
        return this;
    }
    /**
     * Retrieves the parent task's scope.
     */
    get scope() {
        return this.parentTaskDefinition.scope;
    }
    /**
     * Retrieves the parent task's name.
     */
    get name() {
        return this.parentTaskDefinition.name;
    }
    /**
     * Retrieves, if defined, the description of the overridden task,
     * otherwise retrieves the description of the parent task.
     */
    get description() {
        if (this._description !== undefined) {
            return this._description;
        }
        return this.parentTaskDefinition.description;
    }
    /**
     * Retrieves, if defined, the action of the overridden task,
     * otherwise retrieves the action of the parent task.
     */
    get action() {
        if (this._action !== undefined) {
            return this._action;
        }
        return this.parentTaskDefinition.action;
    }
    /**
     * Retrieves the parent task's param definitions.
     */
    get paramDefinitions() {
        return this.parentTaskDefinition.paramDefinitions;
    }
    /**
     * Retrieves the parent task's positional param definitions.
     */
    get positionalParamDefinitions() {
        return this.parentTaskDefinition.positionalParamDefinitions;
    }
    /**
     * Overriden tasks can't add new parameters.
     */
    addParam(name, description, defaultValue, type, isOptional) {
        if (isOptional === undefined || !isOptional) {
            return this._throwNoParamsOverrideError(errors_list_1.ERRORS.TASK_DEFINITIONS.OVERRIDE_NO_MANDATORY_PARAMS);
        }
        return this.addOptionalParam(name, description, defaultValue, type);
    }
    /**
     * Overriden tasks can't add new parameters.
     */
    addOptionalParam(name, description, defaultValue, type) {
        this.parentTaskDefinition.addOptionalParam(name, description, defaultValue, type);
        return this;
    }
    /**
     * Overriden tasks can't add new parameters.
     */
    addPositionalParam(_name, _description, _defaultValue, _type, _isOptional) {
        return this._throwNoParamsOverrideError(errors_list_1.ERRORS.TASK_DEFINITIONS.OVERRIDE_NO_POSITIONAL_PARAMS);
    }
    /**
     * Overriden tasks can't add new parameters.
     */
    addOptionalPositionalParam(_name, _description, _defaultValue, _type) {
        return this._throwNoParamsOverrideError(errors_list_1.ERRORS.TASK_DEFINITIONS.OVERRIDE_NO_POSITIONAL_PARAMS);
    }
    /**
     * Overriden tasks can't add new parameters.
     */
    addVariadicPositionalParam(_name, _description, _defaultValue, _type, _isOptional) {
        return this._throwNoParamsOverrideError(errors_list_1.ERRORS.TASK_DEFINITIONS.OVERRIDE_NO_VARIADIC_PARAMS);
    }
    /**
     * Overriden tasks can't add new parameters.
     */
    addOptionalVariadicPositionalParam(_name, _description, _defaultValue, _type) {
        return this._throwNoParamsOverrideError(errors_list_1.ERRORS.TASK_DEFINITIONS.OVERRIDE_NO_VARIADIC_PARAMS);
    }
    /**
     * Add a flag param to the overridden task.
     * @throws HH201 if param name was already defined in any parent task.
     * @throws HH209 if param name is not in camelCase.
     */
    addFlag(name, description) {
        this.parentTaskDefinition.addFlag(name, description);
        return this;
    }
    _throwNoParamsOverrideError(errorDescriptor) {
        throw new errors_1.HardhatError(errorDescriptor, {
            taskName: this.name,
        });
    }
}
exports.OverriddenTaskDefinition = OverriddenTaskDefinition;
class SimpleScopeDefinition {
    constructor(name, _description, _addTask, _addSubtask) {
        this.name = name;
        this._description = _description;
        this._addTask = _addTask;
        this._addSubtask = _addSubtask;
        this.tasks = {};
    }
    get description() {
        return this._description;
    }
    setDescription(description) {
        this._description = description;
        return this;
    }
    task(name, descriptionOrAction, action) {
        const task = this._addTask(name, descriptionOrAction, action);
        this.tasks[name] = task;
        return task;
    }
    subtask(name, descriptionOrAction, action) {
        const subtask = this._addSubtask(name, descriptionOrAction, action);
        this.tasks[name] = subtask;
        return subtask;
    }
}
exports.SimpleScopeDefinition = SimpleScopeDefinition;
//# sourceMappingURL=task-definitions.js.map