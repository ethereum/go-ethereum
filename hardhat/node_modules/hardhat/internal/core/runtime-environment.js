"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.Environment = void 0;
const debug_1 = __importDefault(require("debug"));
const artifacts_1 = require("../artifacts");
const packageInfo_1 = require("../util/packageInfo");
const config_loading_1 = require("./config/config-loading");
const errors_1 = require("./errors");
const errors_list_1 = require("./errors-list");
const construction_1 = require("./providers/construction");
const lazy_initialization_1 = require("./providers/lazy-initialization");
const task_definitions_1 = require("./tasks/task-definitions");
const task_profiling_1 = require("./task-profiling");
const util_1 = require("./tasks/util");
const log = (0, debug_1.default)("hardhat:core:hre");
class Environment {
    /**
     * Initializes the Hardhat Runtime Environment and the given
     * extender functions.
     *
     * @remarks The extenders' execution order is given by the order
     * of the requires in the hardhat's config file and its plugins.
     *
     * @param config The hardhat's config object.
     * @param hardhatArguments The parsed hardhat's arguments.
     * @param tasks A map of tasks.
     * @param scopes A map of scopes.
     * @param environmentExtenders A list of environment extenders.
     * @param providerExtenders A list of provider extenders.
     */
    constructor(config, hardhatArguments, tasks, scopes, environmentExtenders = [], userConfig = {}, providerExtenders = []) {
        this.config = config;
        this.hardhatArguments = hardhatArguments;
        this.tasks = tasks;
        this.scopes = scopes;
        this.userConfig = userConfig;
        this.version = (0, packageInfo_1.getHardhatVersion)();
        /**
         * Executes the task with the given name.
         *
         * @param taskIdentifier The task or scoped task to be executed.
         * @param taskArguments A map of task's arguments.
         * @param subtaskArguments A map of subtasks to their arguments.
         *
         * @throws a HH303 if there aren't any defined tasks with the given name.
         * @returns a promise with the task's execution result.
         */
        this.run = async (taskIdentifier, taskArguments = {}, subtaskArguments = {}, callerTaskProfile) => {
            const { scope, task } = (0, util_1.parseTaskIdentifier)(taskIdentifier);
            let taskDefinition;
            if (scope === undefined) {
                taskDefinition = this.tasks[task];
                log("Running task %s", task);
            }
            else {
                const scopeDefinition = this.scopes[scope];
                if (scopeDefinition === undefined) {
                    throw new errors_1.HardhatError(errors_list_1.ERRORS.ARGUMENTS.UNRECOGNIZED_SCOPE, {
                        scope,
                    });
                }
                taskDefinition = scopeDefinition.tasks?.[task];
                log("Running scoped task %s %s", scope, task);
            }
            if (taskDefinition === undefined) {
                if (scope !== undefined) {
                    throw new errors_1.HardhatError(errors_list_1.ERRORS.ARGUMENTS.UNRECOGNIZED_SCOPED_TASK, {
                        scope,
                        task,
                    });
                }
                throw new errors_1.HardhatError(errors_list_1.ERRORS.ARGUMENTS.UNRECOGNIZED_TASK, {
                    task,
                });
            }
            const resolvedTaskArguments = this._resolveValidTaskArguments(taskDefinition, taskArguments, subtaskArguments);
            let taskProfile;
            if (this.hardhatArguments.flamegraph === true) {
                taskProfile = (0, task_profiling_1.createTaskProfile)(task);
                if (callerTaskProfile !== undefined) {
                    callerTaskProfile.children.push(taskProfile);
                }
                else {
                    this.entryTaskProfile = taskProfile;
                }
            }
            try {
                return await this._runTaskDefinition(taskDefinition, resolvedTaskArguments, subtaskArguments, taskProfile);
            }
            catch (e) {
                (0, config_loading_1.analyzeModuleNotFoundError)(e, this.config.paths.configFile);
                // eslint-disable-next-line @nomicfoundation/hardhat-internal-rules/only-hardhat-error
                throw e;
            }
            finally {
                if (taskProfile !== undefined) {
                    (0, task_profiling_1.completeTaskProfile)(taskProfile);
                }
            }
        };
        log("Creating HardhatRuntimeEnvironment");
        const networkName = hardhatArguments.network !== undefined
            ? hardhatArguments.network
            : config.defaultNetwork;
        const networkConfig = config.networks[networkName];
        if (networkConfig === undefined) {
            throw new errors_1.HardhatError(errors_list_1.ERRORS.NETWORK.CONFIG_NOT_FOUND, {
                network: networkName,
            });
        }
        this.artifacts = new artifacts_1.Artifacts(config.paths.artifacts);
        const provider = new lazy_initialization_1.LazyInitializationProviderAdapter(async () => {
            log(`Creating provider for network ${networkName}`);
            return (0, construction_1.createProvider)(config, networkName, this.artifacts, providerExtenders);
        });
        this.network = {
            name: networkName,
            config: networkConfig,
            provider,
        };
        this._environmentExtenders = environmentExtenders;
        environmentExtenders.forEach((extender) => extender(this));
    }
    /**
     * Injects the properties of `this` (the Hardhat Runtime Environment) into the global scope.
     *
     * @param blacklist a list of property names that won't be injected.
     *
     * @returns a function that restores the previous environment.
     */
    injectToGlobal(blacklist = Environment._BLACKLISTED_PROPERTIES) {
        const globalAsAny = global;
        const previousValues = {};
        const previousHre = globalAsAny.hre;
        globalAsAny.hre = this;
        for (const [key, value] of Object.entries(this)) {
            if (blacklist.includes(key)) {
                continue;
            }
            previousValues[key] = globalAsAny[key];
            globalAsAny[key] = value;
        }
        return () => {
            for (const [key, _] of Object.entries(this)) {
                if (blacklist.includes(key)) {
                    continue;
                }
                globalAsAny.hre = previousHre;
                globalAsAny[key] = previousValues[key];
            }
        };
    }
    /**
     * @param taskProfile Undefined if we aren't computing task profiles
     * @private
     */
    async _runTaskDefinition(taskDefinition, taskArguments, subtaskArguments, taskProfile) {
        let runSuperFunction;
        if (taskDefinition instanceof task_definitions_1.OverriddenTaskDefinition) {
            runSuperFunction = async (_taskArguments = taskArguments, _subtaskArguments = subtaskArguments) => {
                log("Running %s's super", taskDefinition.name);
                if (taskProfile === undefined) {
                    return this._runTaskDefinition(taskDefinition.parentTaskDefinition, _taskArguments, _subtaskArguments);
                }
                const parentTaskProfile = (0, task_profiling_1.createParentTaskProfile)(taskProfile);
                taskProfile.children.push(parentTaskProfile);
                try {
                    return await this._runTaskDefinition(taskDefinition.parentTaskDefinition, _taskArguments, _subtaskArguments, parentTaskProfile);
                }
                finally {
                    (0, task_profiling_1.completeTaskProfile)(parentTaskProfile);
                }
            };
            runSuperFunction.isDefined = true;
        }
        else {
            runSuperFunction = async () => {
                throw new errors_1.HardhatError(errors_list_1.ERRORS.TASK_DEFINITIONS.RUNSUPER_NOT_AVAILABLE, {
                    taskName: taskDefinition.name,
                });
            };
            runSuperFunction.isDefined = false;
        }
        const runSuper = runSuperFunction;
        const globalAsAny = global;
        const previousRunSuper = globalAsAny.runSuper;
        globalAsAny.runSuper = runSuper;
        // We create a proxied version of `this`, as we want to keep track of the
        // `subtaskArguments` and `taskProfile` through `run` invocations. This
        // way we keep track of callers's data, even when tasks are run in parallel.
        const proxiedHre = new Proxy(this, {
            get(target, p, receiver) {
                if (p === "run") {
                    return (_name, _taskArguments, _subtaskArguments) => target.run(_name, _taskArguments, { ..._subtaskArguments, ...subtaskArguments }, // parent subtask args take precedence
                    taskProfile);
                }
                return Reflect.get(target, p, receiver);
            },
        });
        if (this.hardhatArguments.flamegraph === true) {
            // We modify the `this` again to add  a few utility methods.
            proxiedHre.adhocProfile = async (_name, f) => {
                const adhocProfile = (0, task_profiling_1.createTaskProfile)(_name);
                taskProfile.children.push(adhocProfile);
                try {
                    return await f();
                }
                finally {
                    (0, task_profiling_1.completeTaskProfile)(adhocProfile);
                }
            };
            proxiedHre.adhocProfileSync = (_name, f) => {
                const adhocProfile = (0, task_profiling_1.createTaskProfile)(_name);
                taskProfile.children.push(adhocProfile);
                try {
                    return f();
                }
                finally {
                    (0, task_profiling_1.completeTaskProfile)(adhocProfile);
                }
            };
        }
        const uninjectFromGlobal = proxiedHre.injectToGlobal();
        try {
            return await taskDefinition.action(taskArguments, proxiedHre, runSuper);
        }
        finally {
            uninjectFromGlobal();
            globalAsAny.runSuper = previousRunSuper;
        }
    }
    /**
     * Check that task arguments are within TaskDefinition defined params constraints.
     * Also, populate missing, non-mandatory arguments with default param values (if any).
     *
     * @private
     * @throws HardhatError if any of the following are true:
     *  > a required argument is missing
     *  > an argument's value's type doesn't match the defined param type
     *
     * @param taskDefinition
     * @param taskArguments
     * @returns resolvedTaskArguments
     */
    _resolveValidTaskArguments(taskDefinition, taskArguments, subtaskArguments) {
        const { name: taskName, paramDefinitions, positionalParamDefinitions, } = taskDefinition;
        const nonPositionalParamDefinitions = Object.values(paramDefinitions);
        // gather all task param definitions
        const allTaskParamDefinitions = [
            ...nonPositionalParamDefinitions,
            ...positionalParamDefinitions,
        ];
        const resolvedArguments = {};
        for (const paramDefinition of allTaskParamDefinitions) {
            const paramName = paramDefinition.name;
            const argumentValue = subtaskArguments[taskName]?.[paramName] ?? taskArguments[paramName];
            const resolvedArgumentValue = this._resolveArgument(paramDefinition, argumentValue, taskDefinition.name);
            if (resolvedArgumentValue !== undefined) {
                resolvedArguments[paramName] = resolvedArgumentValue;
            }
        }
        // We keep the args in taskArguments that were not resolved
        return { ...taskArguments, ...resolvedArguments };
    }
    /**
     * Resolves an argument according to a ParamDefinition rules.
     *
     * @param paramDefinition
     * @param argumentValue
     * @private
     */
    _resolveArgument(paramDefinition, argumentValue, taskName) {
        const { name, isOptional, defaultValue } = paramDefinition;
        if (argumentValue === undefined) {
            if (isOptional) {
                // undefined & optional argument -> return defaultValue
                return defaultValue;
            }
            // undefined & mandatory argument -> error
            throw new errors_1.HardhatError(errors_list_1.ERRORS.ARGUMENTS.MISSING_TASK_ARGUMENT, {
                param: name,
                task: taskName,
            });
        }
        // arg was present -> validate type, if applicable
        this._checkTypeValidation(paramDefinition, argumentValue);
        return argumentValue;
    }
    /**
     * Checks if value is valid for the specified param definition.
     *
     * @param paramDefinition {ParamDefinition} - the param definition for validation
     * @param argumentValue - the value to be validated
     * @private
     * @throws HH301 if value is not valid for the param type
     */
    _checkTypeValidation(paramDefinition, argumentValue) {
        const { name: paramName, type, isVariadic } = paramDefinition;
        // in case of variadic param, argValue is an array and the type validation must pass for all values.
        // otherwise, it's a single value that is to be validated
        const argumentValueContainer = isVariadic ? argumentValue : [argumentValue];
        for (const value of argumentValueContainer) {
            type.validate(paramName, value);
        }
    }
}
Environment._BLACKLISTED_PROPERTIES = [
    "injectToGlobal",
    "entryTaskProfile",
    "_runTaskDefinition",
    "_extenders",
];
exports.Environment = Environment;
//# sourceMappingURL=runtime-environment.js.map