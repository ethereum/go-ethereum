import {
  ActionType,
  ArgumentType,
  CLIArgumentType,
  ParamDefinition,
  ParamDefinitionsMap,
  ScopeDefinition,
  TaskArguments,
  TaskDefinition,
  TaskIdentifier,
  TasksMap,
} from "../../../types";
import { HardhatError } from "../errors";
import { ErrorDescriptor, ERRORS } from "../errors-list";
import * as types from "../params/argumentTypes";
import { HARDHAT_PARAM_DEFINITIONS } from "../params/hardhat-params";
import { parseTaskIdentifier } from "./util";

function isCLIArgumentType(
  type: ArgumentType<any>
): type is CLIArgumentType<any> {
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
export class SimpleTaskDefinition implements TaskDefinition {
  public get name() {
    return this._task;
  }
  public get scope() {
    return this._scope;
  }
  public get description() {
    return this._description;
  }
  public readonly paramDefinitions: ParamDefinitionsMap = {};
  public readonly positionalParamDefinitions: Array<ParamDefinition<any>> = [];
  public action: ActionType<TaskArguments>;

  private _positionalParamNames: Set<string>;
  private _hasVariadicParam: boolean;
  private _hasOptionalPositionalParam: boolean;
  private _scope?: string;
  private _task: string;
  private _description?: string;

  /**
   * Creates an empty task definition.
   *
   * This definition will have no params, and will throw a HH205 if executed.
   *
   * @param taskIdentifier The task's identifier.
   * @param isSubtask `true` if the task is a subtask, `false` otherwise.
   */
  constructor(
    taskIdentifier: TaskIdentifier,
    public readonly isSubtask: boolean = false
  ) {
    this._positionalParamNames = new Set();
    this._hasVariadicParam = false;
    this._hasOptionalPositionalParam = false;
    const { scope, task } = parseTaskIdentifier(taskIdentifier);
    this._scope = scope;
    this._task = task;
    this.action = () => {
      throw new HardhatError(ERRORS.TASK_DEFINITIONS.ACTION_NOT_SET, {
        taskName: this._task,
      });
    };
  }

  /**
   * Sets the task's description.
   * @param description The description.
   */
  public setDescription(description: string) {
    this._description = description;
    return this;
  }

  /**
   * Sets the task's action.
   * @param action The action.
   */
  public setAction<TaskArgumentsT extends TaskArguments>(
    action: ActionType<TaskArgumentsT>
  ) {
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
  public addParam<T>(
    name: string,
    description?: string,
    defaultValue?: T,
    type?: ArgumentType<T>,
    isOptional: boolean = defaultValue !== undefined
  ): this {
    if (type === undefined) {
      if (defaultValue === undefined) {
        return this.addParam(
          name,
          description,
          undefined,
          types.string,
          isOptional
        );
      }

      if (typeof defaultValue !== "string") {
        throw new HardhatError(
          ERRORS.TASK_DEFINITIONS.DEFAULT_VALUE_WRONG_TYPE,
          {
            paramName: name,
            taskName: this.name,
          }
        );
      }

      return this.addParam(
        name,
        description,
        defaultValue,
        types.string,
        isOptional
      );
    }

    this._validateParamNameCasing(name);
    this._validateNameNotUsed(name);
    this._validateNoDefaultValueForMandatoryParam(
      defaultValue,
      isOptional,
      name
    );
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
  public addOptionalParam<T>(
    name: string,
    description?: string,
    defaultValue?: T,
    type?: ArgumentType<T>
  ): this {
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
  public addFlag(name: string, description?: string) {
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
  public addPositionalParam<T>(
    name: string,
    description?: string,
    defaultValue?: T,
    type?: ArgumentType<T>,
    isOptional = defaultValue !== undefined
  ): this {
    if (type === undefined) {
      if (defaultValue === undefined) {
        return this.addPositionalParam(
          name,
          description,
          undefined,
          types.string,
          isOptional
        );
      }

      if (typeof defaultValue !== "string") {
        throw new HardhatError(
          ERRORS.TASK_DEFINITIONS.DEFAULT_VALUE_WRONG_TYPE,
          {
            paramName: name,
            taskName: this.name,
          }
        );
      }

      return this.addPositionalParam(
        name,
        description,
        defaultValue,
        types.string,
        isOptional
      );
    }

    this._validateParamNameCasing(name);
    this._validateNameNotUsed(name);
    this._validateNotAfterVariadicParam(name);
    this._validateNoMandatoryParamAfterOptionalOnes(name, isOptional);
    this._validateNoDefaultValueForMandatoryParam(
      defaultValue,
      isOptional,
      name
    );
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
  public addOptionalPositionalParam<T>(
    name: string,
    description?: string,
    defaultValue?: T,
    type?: ArgumentType<T>
  ): this {
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
  public addVariadicPositionalParam<T>(
    name: string,
    description?: string,
    defaultValue?: T[] | T,
    type?: ArgumentType<T>,
    isOptional = defaultValue !== undefined
  ): this {
    if (defaultValue !== undefined && !Array.isArray(defaultValue)) {
      defaultValue = [defaultValue];
    }

    if (type === undefined) {
      if (defaultValue === undefined) {
        return this.addVariadicPositionalParam(
          name,
          description,
          undefined,
          types.string,
          isOptional
        );
      }

      if (!this._isStringArray(defaultValue)) {
        throw new HardhatError(
          ERRORS.TASK_DEFINITIONS.DEFAULT_VALUE_WRONG_TYPE,
          {
            paramName: name,
            taskName: this.name,
          }
        );
      }

      return this.addVariadicPositionalParam(
        name,
        description,
        defaultValue,
        types.string,
        isOptional
      );
    }

    this._validateParamNameCasing(name);
    this._validateNameNotUsed(name);
    this._validateNotAfterVariadicParam(name);
    this._validateNoMandatoryParamAfterOptionalOnes(name, isOptional);
    this._validateNoDefaultValueForMandatoryParam(
      defaultValue,
      isOptional,
      name
    );
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
  public addOptionalVariadicPositionalParam<T>(
    name: string,
    description?: string,
    defaultValue?: T[] | T,
    type?: ArgumentType<T>
  ): this {
    return this.addVariadicPositionalParam(
      name,
      description,
      defaultValue,
      type,
      true
    );
  }

  /**
   * Adds a positional parameter to the task's definition.
   *
   * @param definition the param's definition
   */
  private _addPositionalParamDefinition(definition: ParamDefinition<any>) {
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
  private _validateNotAfterVariadicParam(name: string) {
    if (this._hasVariadicParam) {
      throw new HardhatError(ERRORS.TASK_DEFINITIONS.PARAM_AFTER_VARIADIC, {
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
  private _validateNameNotUsed(name: string) {
    if (this._hasParamDefined(name)) {
      throw new HardhatError(ERRORS.TASK_DEFINITIONS.PARAM_ALREADY_DEFINED, {
        paramName: name,
        taskName: this.name,
      });
    }

    if (Object.keys(HARDHAT_PARAM_DEFINITIONS).includes(name)) {
      throw new HardhatError(
        ERRORS.TASK_DEFINITIONS.PARAM_CLASHES_WITH_HARDHAT_PARAM,
        {
          paramName: name,
          taskName: this.name,
        }
      );
    }
  }

  /**
   * Checks if the given name is already used.
   * @param name the param's name.
   */
  private _hasParamDefined(name: string) {
    return (
      this.paramDefinitions[name] !== undefined ||
      this._positionalParamNames.has(name)
    );
  }

  /**
   * Validates if a mandatory param is being added after optional params.
   *
   * @param name the param's name to be added.
   * @param isOptional true if the new param is optional, false otherwise.
   *
   * @throws HH203 if validation fail
   */
  private _validateNoMandatoryParamAfterOptionalOnes(
    name: string,
    isOptional: boolean
  ) {
    if (!isOptional && this._hasOptionalPositionalParam) {
      throw new HardhatError(
        ERRORS.TASK_DEFINITIONS.MANDATORY_PARAM_AFTER_OPTIONAL,
        {
          paramName: name,
          taskName: this.name,
        }
      );
    }
  }

  private _validateParamNameCasing(name: string) {
    const pattern = /^[a-z]+([a-zA-Z0-9])*$/;
    const match = name.match(pattern);
    if (match === null) {
      throw new HardhatError(
        ERRORS.TASK_DEFINITIONS.INVALID_PARAM_NAME_CASING,
        {
          paramName: name,
          taskName: this.name,
        }
      );
    }
  }

  private _validateNoDefaultValueForMandatoryParam(
    defaultValue: any | undefined,
    isOptional: boolean,
    name: string
  ) {
    if (defaultValue !== undefined && !isOptional) {
      throw new HardhatError(
        ERRORS.TASK_DEFINITIONS.DEFAULT_IN_MANDATORY_PARAM,
        {
          paramName: name,
          taskName: this.name,
        }
      );
    }
  }

  private _isStringArray(values: any): values is string[] {
    return Array.isArray(values) && values.every((v) => typeof v === "string");
  }

  private _validateCLIArgumentTypesForExternalTasks(type: ArgumentType<any>) {
    if (this.isSubtask) {
      return;
    }

    if (!isCLIArgumentType(type)) {
      throw new HardhatError(
        ERRORS.TASK_DEFINITIONS.CLI_ARGUMENT_TYPE_REQUIRED,
        {
          task: this.name,
          type: type.name,
        }
      );
    }
  }
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
export class OverriddenTaskDefinition implements TaskDefinition {
  private _description?: string;
  private _action?: ActionType<TaskArguments>;

  constructor(
    public readonly parentTaskDefinition: TaskDefinition,
    public readonly isSubtask: boolean = false
  ) {
    this.isSubtask = isSubtask;
    this.parentTaskDefinition = parentTaskDefinition;
  }

  /**
   * Sets the task's description.
   * @param description The description.
   */
  public setDescription(description: string) {
    this._description = description;
    return this;
  }

  /**
   * Overrides the parent task's action.
   * @param action the action.
   */
  public setAction<TaskArgumentsT extends TaskArguments>(
    action: ActionType<TaskArgumentsT>
  ) {
    // TODO: There's probably something bad here. See types.ts for more info.
    this._action = action;
    return this;
  }

  /**
   * Retrieves the parent task's scope.
   */
  public get scope() {
    return this.parentTaskDefinition.scope;
  }

  /**
   * Retrieves the parent task's name.
   */
  public get name() {
    return this.parentTaskDefinition.name;
  }

  /**
   * Retrieves, if defined, the description of the overridden task,
   * otherwise retrieves the description of the parent task.
   */
  public get description() {
    if (this._description !== undefined) {
      return this._description;
    }

    return this.parentTaskDefinition.description;
  }

  /**
   * Retrieves, if defined, the action of the overridden task,
   * otherwise retrieves the action of the parent task.
   */
  public get action() {
    if (this._action !== undefined) {
      return this._action;
    }

    return this.parentTaskDefinition.action;
  }

  /**
   * Retrieves the parent task's param definitions.
   */
  public get paramDefinitions() {
    return this.parentTaskDefinition.paramDefinitions;
  }

  /**
   * Retrieves the parent task's positional param definitions.
   */
  public get positionalParamDefinitions() {
    return this.parentTaskDefinition.positionalParamDefinitions;
  }

  /**
   * Overriden tasks can't add new parameters.
   */
  public addParam<T>(
    name: string,
    description?: string,
    defaultValue?: T,
    type?: ArgumentType<T>,
    isOptional?: boolean
  ): this {
    if (isOptional === undefined || !isOptional) {
      return this._throwNoParamsOverrideError(
        ERRORS.TASK_DEFINITIONS.OVERRIDE_NO_MANDATORY_PARAMS
      );
    }
    return this.addOptionalParam(name, description, defaultValue, type);
  }

  /**
   * Overriden tasks can't add new parameters.
   */
  public addOptionalParam<T>(
    name: string,
    description?: string,
    defaultValue?: T,
    type?: ArgumentType<T>
  ): this {
    this.parentTaskDefinition.addOptionalParam(
      name,
      description,
      defaultValue,
      type
    );
    return this;
  }

  /**
   * Overriden tasks can't add new parameters.
   */
  public addPositionalParam<T>(
    _name: string,
    _description?: string,
    _defaultValue?: T,
    _type?: ArgumentType<T>,
    _isOptional?: boolean
  ): this {
    return this._throwNoParamsOverrideError(
      ERRORS.TASK_DEFINITIONS.OVERRIDE_NO_POSITIONAL_PARAMS
    );
  }

  /**
   * Overriden tasks can't add new parameters.
   */
  public addOptionalPositionalParam<T>(
    _name: string,
    _description?: string,
    _defaultValue?: T,
    _type?: ArgumentType<T>
  ): this {
    return this._throwNoParamsOverrideError(
      ERRORS.TASK_DEFINITIONS.OVERRIDE_NO_POSITIONAL_PARAMS
    );
  }

  /**
   * Overriden tasks can't add new parameters.
   */
  public addVariadicPositionalParam<T>(
    _name: string,
    _description?: string,
    _defaultValue?: T[],
    _type?: ArgumentType<T>,
    _isOptional?: boolean
  ): this {
    return this._throwNoParamsOverrideError(
      ERRORS.TASK_DEFINITIONS.OVERRIDE_NO_VARIADIC_PARAMS
    );
  }

  /**
   * Overriden tasks can't add new parameters.
   */
  public addOptionalVariadicPositionalParam<T>(
    _name: string,
    _description?: string,
    _defaultValue?: T[],
    _type?: ArgumentType<T>
  ): this {
    return this._throwNoParamsOverrideError(
      ERRORS.TASK_DEFINITIONS.OVERRIDE_NO_VARIADIC_PARAMS
    );
  }

  /**
   * Add a flag param to the overridden task.
   * @throws HH201 if param name was already defined in any parent task.
   * @throws HH209 if param name is not in camelCase.
   */
  public addFlag(name: string, description?: string): this {
    this.parentTaskDefinition.addFlag(name, description);
    return this;
  }

  private _throwNoParamsOverrideError(errorDescriptor: ErrorDescriptor): never {
    throw new HardhatError(errorDescriptor, {
      taskName: this.name,
    });
  }
}

type AddTaskFunction = <TaskArgumentsT extends TaskArguments>(
  name: string,
  descriptionOrAction?: string | ActionType<TaskArgumentsT>,
  action?: ActionType<TaskArgumentsT>
) => TaskDefinition;

export class SimpleScopeDefinition implements ScopeDefinition {
  public tasks: TasksMap = {};

  constructor(
    public readonly name: string,
    private _description: string | undefined,
    private _addTask: AddTaskFunction,
    private _addSubtask: AddTaskFunction
  ) {}

  public get description() {
    return this._description;
  }

  public setDescription(description: string): this {
    this._description = description;
    return this;
  }

  public task<TaskArgumentsT extends TaskArguments>(
    name: string,
    description?: string,
    action?: ActionType<TaskArgumentsT>
  ): TaskDefinition;
  public task<TaskArgumentsT extends TaskArguments>(
    name: string,
    action: ActionType<TaskArgumentsT>
  ): TaskDefinition;
  public task<TaskArgumentsT extends TaskArguments>(
    name: string,
    descriptionOrAction?: string | ActionType<TaskArgumentsT>,
    action?: ActionType<TaskArgumentsT>
  ) {
    const task = this._addTask(name, descriptionOrAction, action);

    this.tasks[name] = task;

    return task;
  }

  public subtask<TaskArgumentsT extends TaskArguments>(
    name: string,
    description?: string,
    action?: ActionType<TaskArgumentsT>
  ): TaskDefinition;
  public subtask<TaskArgumentsT extends TaskArguments>(
    name: string,
    action: ActionType<TaskArgumentsT>
  ): TaskDefinition;
  public subtask<TaskArgumentsT extends TaskArguments>(
    name: string,
    descriptionOrAction?: string | ActionType<TaskArgumentsT>,
    action?: ActionType<TaskArgumentsT>
  ) {
    const subtask = this._addSubtask(name, descriptionOrAction, action);

    this.tasks[name] = subtask;

    return subtask;
  }
}
