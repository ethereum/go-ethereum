import { ActionType, ConfigExtender, ConfigurableScopeDefinition, ConfigurableTaskDefinition, EnvironmentExtender, ExperimentalHardhatNetworkMessageTraceHook, ProviderExtender, TaskArguments } from "../../../types";
import * as argumentTypes from "../params/argumentTypes";
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
export declare function task<TaskArgumentsT extends TaskArguments>(name: string, description?: string, action?: ActionType<TaskArgumentsT>): ConfigurableTaskDefinition;
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
export declare function task<TaskArgumentsT extends TaskArguments>(name: string, action: ActionType<TaskArgumentsT>): ConfigurableTaskDefinition;
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
export declare function subtask<TaskArgumentsT extends TaskArguments>(name: string, description?: string, action?: ActionType<TaskArgumentsT>): ConfigurableTaskDefinition;
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
export declare function subtask<TaskArgumentsT extends TaskArguments>(name: string, action: ActionType<TaskArgumentsT>): ConfigurableTaskDefinition;
export declare const internalTask: typeof subtask;
export declare function scope(name: string, description?: string): ConfigurableScopeDefinition;
export declare const types: typeof argumentTypes;
/**
 * Register an environment extender what will be run after the
 * Hardhat Runtime Environment is initialized.
 *
 * @param extender A function that receives the Hardhat Runtime
 * Environment.
 */
export declare function extendEnvironment(extender: EnvironmentExtender): void;
/**
 * Register a config extender what will be run after the
 * Hardhat Runtime Environment is initialized.
 *
 * @param extender A function that receives the resolved config
 * to be modified and the config provided by the user
 */
export declare function extendConfig(extender: ConfigExtender): void;
/**
 * Register a provider extender what will be run after the
 * Hardhat Runtime Environment is initialized.
 *
 * @param extender A function that receives the current provider
 * and returns a new one.
 */
export declare function extendProvider(extender: ProviderExtender): void;
export declare function experimentalAddHardhatNetworkMessageTraceHook(hook: ExperimentalHardhatNetworkMessageTraceHook): void;
/**
 * This object provides methods to interact with the configuration variables.
 */
export declare const vars: {
    has: typeof hasVar;
    get: typeof getVar;
};
/**
 * Checks if a configuration variable exists.
 *
 * @remarks
 * This method, when used during setup (via `npx hardhat vars setup`), will mark the variable as optional.
 *
 * @param varName - The name of the variable to check.
 *
 * @returns `true` if the variable exists, `false` otherwise.
 */
declare function hasVar(varName: string): boolean;
/**
 * Gets the value of the given configuration variable.
 *
 * @remarks
 * This method, when used during setup (via `npx hardhat vars setup`), will mark the variable as required,
 * unless a default value is provided.
 *
 * @param varName - The name of the variable to retrieve.
 * @param [defaultValue] - An optional default value to return if the variable does not exist.
 *
 * @returns The value of the configuration variable if it exists, or the default value if provided.
 *
 * @throws HH1201 if the variable does not exist and no default value is set.
 */
declare function getVar(varName: string, defaultValue?: string): string;
export {};
//# sourceMappingURL=config-env.d.ts.map