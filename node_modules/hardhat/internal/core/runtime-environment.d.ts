import { Artifacts as IArtifacts, EnvironmentExtender, ExperimentalHardhatNetworkMessageTraceHook, HardhatArguments, HardhatConfig, HardhatRuntimeEnvironment, HardhatUserConfig, Network, ProviderExtender, RunTaskFunction, TasksMap, ScopesMap } from "../../types";
import { TaskProfile } from "./task-profiling";
export declare class Environment implements HardhatRuntimeEnvironment {
    readonly config: HardhatConfig;
    readonly hardhatArguments: HardhatArguments;
    readonly tasks: TasksMap;
    readonly scopes: ScopesMap;
    readonly userConfig: HardhatUserConfig;
    private static readonly _BLACKLISTED_PROPERTIES;
    network: Network;
    artifacts: IArtifacts;
    private readonly _environmentExtenders;
    entryTaskProfile?: TaskProfile;
    version: string;
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
    constructor(config: HardhatConfig, hardhatArguments: HardhatArguments, tasks: TasksMap, scopes: ScopesMap, environmentExtenders?: EnvironmentExtender[], experimentalHardhatNetworkMessageTraceHooks?: ExperimentalHardhatNetworkMessageTraceHook[], userConfig?: HardhatUserConfig, providerExtenders?: ProviderExtender[]);
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
    readonly run: RunTaskFunction;
    /**
     * Injects the properties of `this` (the Hardhat Runtime Environment) into the global scope.
     *
     * @param blacklist a list of property names that won't be injected.
     *
     * @returns a function that restores the previous environment.
     */
    injectToGlobal(blacklist?: string[]): () => void;
    /**
     * @param taskProfile Undefined if we aren't computing task profiles
     * @private
     */
    private _runTaskDefinition;
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
    private _resolveValidTaskArguments;
    /**
     * Resolves an argument according to a ParamDefinition rules.
     *
     * @param paramDefinition
     * @param argumentValue
     * @private
     */
    private _resolveArgument;
    /**
     * Checks if value is valid for the specified param definition.
     *
     * @param paramDefinition {ParamDefinition} - the param definition for validation
     * @param argumentValue - the value to be validated
     * @private
     * @throws HH301 if value is not valid for the param type
     */
    private _checkTypeValidation;
}
//# sourceMappingURL=runtime-environment.d.ts.map