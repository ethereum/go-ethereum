import {
  ActionType,
  ConfigExtender,
  ConfigurableScopeDefinition,
  ConfigurableTaskDefinition,
  EnvironmentExtender,
  ProviderExtender,
  TaskArguments,
} from "../../../types";
import { HardhatContext } from "../../context";
import { HardhatError } from "../errors";
import { ERRORS } from "../errors-list";
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
export function task<TaskArgumentsT extends TaskArguments>(
  name: string,
  description?: string,
  action?: ActionType<TaskArgumentsT>
): ConfigurableTaskDefinition;

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
export function task<TaskArgumentsT extends TaskArguments>(
  name: string,
  action: ActionType<TaskArgumentsT>
): ConfigurableTaskDefinition;

export function task<TaskArgumentsT extends TaskArguments>(
  name: string,
  descriptionOrAction?: string | ActionType<TaskArgumentsT>,
  action?: ActionType<TaskArgumentsT>
): ConfigurableTaskDefinition {
  const ctx = HardhatContext.getHardhatContext();
  const dsl = ctx.tasksDSL;

  if (descriptionOrAction === undefined) {
    return dsl.task(name);
  }

  if (typeof descriptionOrAction !== "string") {
    return dsl.task(name, descriptionOrAction);
  }

  return dsl.task(name, descriptionOrAction, action);
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
export function subtask<TaskArgumentsT extends TaskArguments>(
  name: string,
  description?: string,
  action?: ActionType<TaskArgumentsT>
): ConfigurableTaskDefinition;

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
export function subtask<TaskArgumentsT extends TaskArguments>(
  name: string,
  action: ActionType<TaskArgumentsT>
): ConfigurableTaskDefinition;

export function subtask<TaskArgumentsT extends TaskArguments>(
  name: string,
  descriptionOrAction?: string | ActionType<TaskArgumentsT>,
  action?: ActionType<TaskArgumentsT>
): ConfigurableTaskDefinition {
  const ctx = HardhatContext.getHardhatContext();
  const dsl = ctx.tasksDSL;

  if (descriptionOrAction === undefined) {
    return dsl.subtask(name);
  }

  if (typeof descriptionOrAction !== "string") {
    return dsl.subtask(name, descriptionOrAction);
  }

  return dsl.subtask(name, descriptionOrAction, action);
}

// Backwards compatibility alias
export const internalTask = subtask;

export function scope(
  name: string,
  description?: string
): ConfigurableScopeDefinition {
  const ctx = HardhatContext.getHardhatContext();
  const dsl = ctx.tasksDSL;

  return dsl.scope(name, description);
}

export const types = argumentTypes;

/**
 * Register an environment extender what will be run after the
 * Hardhat Runtime Environment is initialized.
 *
 * @param extender A function that receives the Hardhat Runtime
 * Environment.
 */
export function extendEnvironment(extender: EnvironmentExtender) {
  const ctx = HardhatContext.getHardhatContext();
  ctx.environmentExtenders.push(extender);
}

/**
 * Register a config extender what will be run after the
 * Hardhat Runtime Environment is initialized.
 *
 * @param extender A function that receives the resolved config
 * to be modified and the config provided by the user
 */
export function extendConfig(extender: ConfigExtender) {
  const ctx = HardhatContext.getHardhatContext();
  ctx.configExtenders.push(extender);
}

/**
 * Register a provider extender what will be run after the
 * Hardhat Runtime Environment is initialized.
 *
 * @param extender A function that receives the current provider
 * and returns a new one.
 */
export function extendProvider(extender: ProviderExtender) {
  const ctx = HardhatContext.getHardhatContext();
  ctx.providerExtenders.push(extender);
}

/**
 * This object provides methods to interact with the configuration variables.
 */
export const vars = {
  has: hasVar,
  get: getVar,
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
function hasVar(varName: string): boolean {
  // varsManager will be an instance of VarsManager or VarsManagerSetup depending on the context (vars setup mode or not)
  return HardhatContext.getHardhatContext().varsManager.has(varName, true);
}

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
function getVar(varName: string, defaultValue?: string): string {
  // varsManager will be an instance of VarsManager or VarsManagerSetup depending on the context (vars setup mode or not)
  const value = HardhatContext.getHardhatContext().varsManager.get(
    varName,
    defaultValue,
    true
  );

  if (value !== undefined) return value;

  throw new HardhatError(ERRORS.VARS.VALUE_NOT_FOUND_FOR_VAR, {
    value: varName,
  });
}
