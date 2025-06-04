import {
  ActionType,
  ScopeDefinition,
  ScopesMap,
  TaskArguments,
  TaskDefinition,
  TaskIdentifier,
  TasksMap,
} from "../../../types";
import { HardhatError, assertHardhatInvariant } from "../errors";
import { ERRORS } from "../errors-list";

import {
  OverriddenTaskDefinition,
  SimpleScopeDefinition,
  SimpleTaskDefinition,
} from "./task-definitions";
import { parseTaskIdentifier } from "./util";

/**
 * This class defines the DSL used in Hardhat config files
 * for creating and overriding tasks.
 */
export class TasksDSL {
  public readonly internalTask = this.subtask;

  private readonly _tasks: TasksMap = {};
  private readonly _scopes: ScopesMap = {};

  /**
   * Creates a task, overriding any previous task with the same name.
   *
   * @remarks The action must await every async call made within it.
   *
   * @param name The task's name.
   * @param description The task's description.
   * @param action The task's action.
   * @returns A task definition.
   */
  public task<TaskArgumentsT extends TaskArguments>(
    name: string,
    description?: string,
    action?: ActionType<TaskArgumentsT>
  ): TaskDefinition;

  /**
   * Creates a task without description, overriding any previous task
   * with the same name.
   *
   * @remarks The action must await every async call made within it.
   *
   * @param name The task's name.
   * @param action The task's action.
   *
   * @returns A task definition.
   */
  public task<TaskArgumentsT extends TaskArguments>(
    name: string,
    action: ActionType<TaskArgumentsT>
  ): TaskDefinition;

  public task<TaskArgumentsT extends TaskArguments>(
    name: string,
    descriptionOrAction?: string | ActionType<TaskArgumentsT>,
    action?: ActionType<TaskArgumentsT>
  ): TaskDefinition {
    // if this function is updated, update the corresponding callback
    // passed to `new SimpleScopeDefinition`
    return this._addTask(name, descriptionOrAction, action, false);
  }

  /**
   * Creates a subtask, overriding any previous task with the same name.
   *
   * @remarks The subtasks won't be displayed in the CLI help messages.
   * @remarks The action must await every async call made within it.
   *
   * @param name The task's name.
   * @param description The task's description.
   * @param action The task's action.
   * @returns A task definition.
   */
  public subtask<TaskArgumentsT extends TaskArguments>(
    name: string,
    description?: string,
    action?: ActionType<TaskArgumentsT>
  ): TaskDefinition;

  /**
   * Creates a subtask without description, overriding any previous
   * task with the same name.
   *
   * @remarks The subtasks won't be displayed in the CLI help messages.
   * @remarks The action must await every async call made within it.
   *
   * @param name The task's name.
   * @param action The task's action.
   * @returns A task definition.
   */
  public subtask<TaskArgumentsT extends TaskArguments>(
    name: string,
    action: ActionType<TaskArgumentsT>
  ): TaskDefinition;
  public subtask<TaskArgumentsT extends TaskArguments>(
    name: string,
    descriptionOrAction?: string | ActionType<TaskArgumentsT>,
    action?: ActionType<TaskArgumentsT>
  ): TaskDefinition {
    // if this function is updated, update the corresponding callback
    // passed to `new SimpleScopeDefinition`
    return this._addTask(name, descriptionOrAction, action, true);
  }

  public scope(name: string, description?: string): ScopeDefinition {
    if (this._tasks[name] !== undefined) {
      throw new HardhatError(ERRORS.TASK_DEFINITIONS.TASK_SCOPE_CLASH, {
        scopeName: name,
      });
    }

    const scopeDefinition = this._scopes[name];

    if (scopeDefinition !== undefined) {
      // if the scope already exists, the only thing we might
      // do is to update its description
      if (description !== undefined) {
        scopeDefinition.setDescription(description);
      }

      return scopeDefinition;
    }

    const scope = new SimpleScopeDefinition(
      name,
      description,
      (taskName, descriptionOrAction, action) =>
        // if this function is updated, update the dsl.task function too
        this._addTask(
          { scope: name, task: taskName },
          descriptionOrAction,
          action,
          false
        ),
      (subtaskName, descriptionOrAction, action) =>
        // if this function is updated, update the dsl.subtask function too
        this._addTask(
          { scope: name, task: subtaskName },
          descriptionOrAction,
          action,
          true
        )
    );

    this._scopes[name] = scope;

    return scope;
  }

  /**
   * Retrieves the task definitions.
   *
   * @returns The tasks container.
   */
  public getTaskDefinitions(): TasksMap {
    return this._tasks;
  }

  /**
   * Retrieves the scoped task definitions.
   *
   * @returns The scoped tasks container.
   */
  public getScopesDefinitions(): ScopesMap {
    return this._scopes;
  }

  public getTaskDefinition(
    scope: string | undefined,
    name: string
  ): TaskDefinition | undefined {
    if (scope === undefined) {
      return this._tasks[name];
    } else {
      return this._scopes[scope]?.tasks?.[name];
    }
  }

  private _addTask<TaskArgumentsT extends TaskArguments>(
    taskIdentifier: TaskIdentifier,
    descriptionOrAction?: string | ActionType<TaskArgumentsT>,
    action?: ActionType<TaskArgumentsT>,
    isSubtask?: boolean
  ) {
    const { scope, task } = parseTaskIdentifier(taskIdentifier);

    if (scope === undefined && this._scopes[task] !== undefined) {
      throw new HardhatError(ERRORS.TASK_DEFINITIONS.SCOPE_TASK_CLASH, {
        taskName: task,
      });
    }

    const parentTaskDefinition = this.getTaskDefinition(scope, task);

    let taskDefinition: TaskDefinition;

    if (parentTaskDefinition !== undefined) {
      taskDefinition = new OverriddenTaskDefinition(
        parentTaskDefinition,
        isSubtask
      );
    } else {
      taskDefinition = new SimpleTaskDefinition(taskIdentifier, isSubtask);
    }

    if (descriptionOrAction instanceof Function) {
      action = descriptionOrAction;
      descriptionOrAction = undefined;
    }

    if (descriptionOrAction !== undefined) {
      taskDefinition.setDescription(descriptionOrAction);
    }

    if (action !== undefined) {
      taskDefinition.setAction(action);
    }

    if (scope === undefined) {
      this._tasks[task] = taskDefinition;
    } else {
      const scopeDefinition = this._scopes[scope];
      assertHardhatInvariant(
        scopeDefinition !== undefined,
        "It shouldn't be possible to create a task in a scope that doesn't exist"
      );
      scopeDefinition.tasks[task] = taskDefinition;
    }

    return taskDefinition;
  }
}
