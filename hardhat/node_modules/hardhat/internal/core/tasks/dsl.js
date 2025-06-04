"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.TasksDSL = void 0;
const errors_1 = require("../errors");
const errors_list_1 = require("../errors-list");
const task_definitions_1 = require("./task-definitions");
const util_1 = require("./util");
/**
 * This class defines the DSL used in Hardhat config files
 * for creating and overriding tasks.
 */
class TasksDSL {
    constructor() {
        this.internalTask = this.subtask;
        this._tasks = {};
        this._scopes = {};
    }
    task(name, descriptionOrAction, action) {
        // if this function is updated, update the corresponding callback
        // passed to `new SimpleScopeDefinition`
        return this._addTask(name, descriptionOrAction, action, false);
    }
    subtask(name, descriptionOrAction, action) {
        // if this function is updated, update the corresponding callback
        // passed to `new SimpleScopeDefinition`
        return this._addTask(name, descriptionOrAction, action, true);
    }
    scope(name, description) {
        if (this._tasks[name] !== undefined) {
            throw new errors_1.HardhatError(errors_list_1.ERRORS.TASK_DEFINITIONS.TASK_SCOPE_CLASH, {
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
        const scope = new task_definitions_1.SimpleScopeDefinition(name, description, (taskName, descriptionOrAction, action) => 
        // if this function is updated, update the dsl.task function too
        this._addTask({ scope: name, task: taskName }, descriptionOrAction, action, false), (subtaskName, descriptionOrAction, action) => 
        // if this function is updated, update the dsl.subtask function too
        this._addTask({ scope: name, task: subtaskName }, descriptionOrAction, action, true));
        this._scopes[name] = scope;
        return scope;
    }
    /**
     * Retrieves the task definitions.
     *
     * @returns The tasks container.
     */
    getTaskDefinitions() {
        return this._tasks;
    }
    /**
     * Retrieves the scoped task definitions.
     *
     * @returns The scoped tasks container.
     */
    getScopesDefinitions() {
        return this._scopes;
    }
    getTaskDefinition(scope, name) {
        if (scope === undefined) {
            return this._tasks[name];
        }
        else {
            return this._scopes[scope]?.tasks?.[name];
        }
    }
    _addTask(taskIdentifier, descriptionOrAction, action, isSubtask) {
        const { scope, task } = (0, util_1.parseTaskIdentifier)(taskIdentifier);
        if (scope === undefined && this._scopes[task] !== undefined) {
            throw new errors_1.HardhatError(errors_list_1.ERRORS.TASK_DEFINITIONS.SCOPE_TASK_CLASH, {
                taskName: task,
            });
        }
        const parentTaskDefinition = this.getTaskDefinition(scope, task);
        let taskDefinition;
        if (parentTaskDefinition !== undefined) {
            taskDefinition = new task_definitions_1.OverriddenTaskDefinition(parentTaskDefinition, isSubtask);
        }
        else {
            taskDefinition = new task_definitions_1.SimpleTaskDefinition(taskIdentifier, isSubtask);
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
        }
        else {
            const scopeDefinition = this._scopes[scope];
            (0, errors_1.assertHardhatInvariant)(scopeDefinition !== undefined, "It shouldn't be possible to create a task in a scope that doesn't exist");
            scopeDefinition.tasks[task] = taskDefinition;
        }
        return taskDefinition;
    }
}
exports.TasksDSL = TasksDSL;
//# sourceMappingURL=dsl.js.map