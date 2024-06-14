import { ActionType, ArgumentType, ParamDefinition, ParamDefinitionsMap, ScopeDefinition, TaskArguments, TaskDefinition, TaskIdentifier, TasksMap } from "../../../types";
/**
 * This class creates a task definition, which consists of:
 * * a name, that should be unique and will be used to call the task.
 * * a description. This is optional.
 * * the action that the task will execute.
 * * a set of parameters that can be used by the action.
 *
 */
export declare class SimpleTaskDefinition implements TaskDefinition {
    readonly isSubtask: boolean;
    get name(): string;
    get scope(): string | undefined;
    get description(): string | undefined;
    readonly paramDefinitions: ParamDefinitionsMap;
    readonly positionalParamDefinitions: Array<ParamDefinition<any>>;
    action: ActionType<TaskArguments>;
    private _positionalParamNames;
    private _hasVariadicParam;
    private _hasOptionalPositionalParam;
    private _scope?;
    private _task;
    private _description?;
    /**
     * Creates an empty task definition.
     *
     * This definition will have no params, and will throw a HH205 if executed.
     *
     * @param taskIdentifier The task's identifier.
     * @param isSubtask `true` if the task is a subtask, `false` otherwise.
     */
    constructor(taskIdentifier: TaskIdentifier, isSubtask?: boolean);
    /**
     * Sets the task's description.
     * @param description The description.
     */
    setDescription(description: string): this;
    /**
     * Sets the task's action.
     * @param action The action.
     */
    setAction<TaskArgumentsT extends TaskArguments>(action: ActionType<TaskArgumentsT>): this;
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
    addParam<T>(name: string, description?: string, defaultValue?: T, type?: ArgumentType<T>, isOptional?: boolean): this;
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
    addOptionalParam<T>(name: string, description?: string, defaultValue?: T, type?: ArgumentType<T>): this;
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
    addFlag(name: string, description?: string): this;
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
    addPositionalParam<T>(name: string, description?: string, defaultValue?: T, type?: ArgumentType<T>, isOptional?: boolean): this;
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
    addOptionalPositionalParam<T>(name: string, description?: string, defaultValue?: T, type?: ArgumentType<T>): this;
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
    addVariadicPositionalParam<T>(name: string, description?: string, defaultValue?: T[] | T, type?: ArgumentType<T>, isOptional?: boolean): this;
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
    addOptionalVariadicPositionalParam<T>(name: string, description?: string, defaultValue?: T[] | T, type?: ArgumentType<T>): this;
    /**
     * Adds a positional parameter to the task's definition.
     *
     * @param definition the param's definition
     */
    private _addPositionalParamDefinition;
    /**
     * Validates if the given param's name is after a variadic parameter.
     * @param name the param's name.
     * @throws HH200
     */
    private _validateNotAfterVariadicParam;
    /**
     * Validates if the param's name is already used.
     * @param name the param's name.
     *
     * @throws HH201 if `name` is already used as a param.
     * @throws HH202 if `name` is already used as a param by Hardhat
     */
    private _validateNameNotUsed;
    /**
     * Checks if the given name is already used.
     * @param name the param's name.
     */
    private _hasParamDefined;
    /**
     * Validates if a mandatory param is being added after optional params.
     *
     * @param name the param's name to be added.
     * @param isOptional true if the new param is optional, false otherwise.
     *
     * @throws HH203 if validation fail
     */
    private _validateNoMandatoryParamAfterOptionalOnes;
    private _validateParamNameCasing;
    private _validateNoDefaultValueForMandatoryParam;
    private _isStringArray;
    private _validateCLIArgumentTypesForExternalTasks;
}
/**
 * Allows you to override a previously defined task.
 *
 * When overriding a task you can:
 *  * flag it as a subtask
 *  * set a new description
 *  * set a new action
 *
 */
export declare class OverriddenTaskDefinition implements TaskDefinition {
    readonly parentTaskDefinition: TaskDefinition;
    readonly isSubtask: boolean;
    private _description?;
    private _action?;
    constructor(parentTaskDefinition: TaskDefinition, isSubtask?: boolean);
    /**
     * Sets the task's description.
     * @param description The description.
     */
    setDescription(description: string): this;
    /**
     * Overrides the parent task's action.
     * @param action the action.
     */
    setAction<TaskArgumentsT extends TaskArguments>(action: ActionType<TaskArgumentsT>): this;
    /**
     * Retrieves the parent task's scope.
     */
    get scope(): string | undefined;
    /**
     * Retrieves the parent task's name.
     */
    get name(): string;
    /**
     * Retrieves, if defined, the description of the overridden task,
     * otherwise retrieves the description of the parent task.
     */
    get description(): string | undefined;
    /**
     * Retrieves, if defined, the action of the overridden task,
     * otherwise retrieves the action of the parent task.
     */
    get action(): ActionType<any>;
    /**
     * Retrieves the parent task's param definitions.
     */
    get paramDefinitions(): ParamDefinitionsMap;
    /**
     * Retrieves the parent task's positional param definitions.
     */
    get positionalParamDefinitions(): ParamDefinition<any>[];
    /**
     * Overriden tasks can't add new parameters.
     */
    addParam<T>(name: string, description?: string, defaultValue?: T, type?: ArgumentType<T>, isOptional?: boolean): this;
    /**
     * Overriden tasks can't add new parameters.
     */
    addOptionalParam<T>(name: string, description?: string, defaultValue?: T, type?: ArgumentType<T>): this;
    /**
     * Overriden tasks can't add new parameters.
     */
    addPositionalParam<T>(_name: string, _description?: string, _defaultValue?: T, _type?: ArgumentType<T>, _isOptional?: boolean): this;
    /**
     * Overriden tasks can't add new parameters.
     */
    addOptionalPositionalParam<T>(_name: string, _description?: string, _defaultValue?: T, _type?: ArgumentType<T>): this;
    /**
     * Overriden tasks can't add new parameters.
     */
    addVariadicPositionalParam<T>(_name: string, _description?: string, _defaultValue?: T[], _type?: ArgumentType<T>, _isOptional?: boolean): this;
    /**
     * Overriden tasks can't add new parameters.
     */
    addOptionalVariadicPositionalParam<T>(_name: string, _description?: string, _defaultValue?: T[], _type?: ArgumentType<T>): this;
    /**
     * Add a flag param to the overridden task.
     * @throws HH201 if param name was already defined in any parent task.
     * @throws HH209 if param name is not in camelCase.
     */
    addFlag(name: string, description?: string): this;
    private _throwNoParamsOverrideError;
}
type AddTaskFunction = <TaskArgumentsT extends TaskArguments>(name: string, descriptionOrAction?: string | ActionType<TaskArgumentsT>, action?: ActionType<TaskArgumentsT>) => TaskDefinition;
export declare class SimpleScopeDefinition implements ScopeDefinition {
    readonly name: string;
    private _description;
    private _addTask;
    private _addSubtask;
    tasks: TasksMap;
    constructor(name: string, _description: string | undefined, _addTask: AddTaskFunction, _addSubtask: AddTaskFunction);
    get description(): string | undefined;
    setDescription(description: string): this;
    task<TaskArgumentsT extends TaskArguments>(name: string, description?: string, action?: ActionType<TaskArgumentsT>): TaskDefinition;
    task<TaskArgumentsT extends TaskArguments>(name: string, action: ActionType<TaskArgumentsT>): TaskDefinition;
    subtask<TaskArgumentsT extends TaskArguments>(name: string, description?: string, action?: ActionType<TaskArgumentsT>): TaskDefinition;
    subtask<TaskArgumentsT extends TaskArguments>(name: string, action: ActionType<TaskArgumentsT>): TaskDefinition;
}
export {};
//# sourceMappingURL=task-definitions.d.ts.map