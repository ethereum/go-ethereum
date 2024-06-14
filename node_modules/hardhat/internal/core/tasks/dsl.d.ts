import { ActionType, ScopeDefinition, ScopesMap, TaskArguments, TaskDefinition, TasksMap } from "../../../types";
/**
 * This class defines the DSL used in Hardhat config files
 * for creating and overriding tasks.
 */
export declare class TasksDSL {
    readonly internalTask: {
        <TaskArgumentsT extends unknown>(name: string, description?: string, action?: ActionType<TaskArgumentsT> | undefined): TaskDefinition;
        <TaskArgumentsT_1 extends unknown>(name: string, action: ActionType<TaskArgumentsT_1>): TaskDefinition;
    };
    private readonly _tasks;
    private readonly _scopes;
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
    task<TaskArgumentsT extends TaskArguments>(name: string, description?: string, action?: ActionType<TaskArgumentsT>): TaskDefinition;
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
    task<TaskArgumentsT extends TaskArguments>(name: string, action: ActionType<TaskArgumentsT>): TaskDefinition;
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
    subtask<TaskArgumentsT extends TaskArguments>(name: string, description?: string, action?: ActionType<TaskArgumentsT>): TaskDefinition;
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
    subtask<TaskArgumentsT extends TaskArguments>(name: string, action: ActionType<TaskArgumentsT>): TaskDefinition;
    scope(name: string, description?: string): ScopeDefinition;
    /**
     * Retrieves the task definitions.
     *
     * @returns The tasks container.
     */
    getTaskDefinitions(): TasksMap;
    /**
     * Retrieves the scoped task definitions.
     *
     * @returns The scoped tasks container.
     */
    getScopesDefinitions(): ScopesMap;
    getTaskDefinition(scope: string | undefined, name: string): TaskDefinition | undefined;
    private _addTask;
}
//# sourceMappingURL=dsl.d.ts.map