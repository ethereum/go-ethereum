import { Abi, Artifact } from "./artifact";
/**
 * Base argument type that smart contracts can receive in their constructors
 * and functions.
 *
 * @beta
 */
export type BaseArgumentType = number | bigint | string | boolean | ContractFuture<string> | StaticCallFuture<string, string> | EncodeFunctionCallFuture<string, string> | ReadEventArgumentFuture | RuntimeValue;
/**
 * Argument type that smart contracts can receive in their constructors and functions.
 *
 * @beta
 */
export type ArgumentType = BaseArgumentType | ArgumentType[] | {
    [field: string]: ArgumentType;
};
/**
 * The different future types supported by Ignition.
 *
 * @beta
 */
export declare enum FutureType {
    NAMED_ARTIFACT_CONTRACT_DEPLOYMENT = "NAMED_ARTIFACT_CONTRACT_DEPLOYMENT",
    CONTRACT_DEPLOYMENT = "CONTRACT_DEPLOYMENT",
    NAMED_ARTIFACT_LIBRARY_DEPLOYMENT = "NAMED_ARTIFACT_LIBRARY_DEPLOYMENT",
    LIBRARY_DEPLOYMENT = "LIBRARY_DEPLOYMENT",
    CONTRACT_CALL = "CONTRACT_CALL",
    STATIC_CALL = "STATIC_CALL",
    ENCODE_FUNCTION_CALL = "ENCODE_FUNCTION_CALL",
    NAMED_ARTIFACT_CONTRACT_AT = "NAMED_ARTIFACT_CONTRACT_AT",
    CONTRACT_AT = "CONTRACT_AT",
    READ_EVENT_ARGUMENT = "READ_EVENT_ARGUMENT",
    SEND_DATA = "SEND_DATA"
}
/**
 * The unit of execution in an Ignition deploy.
 *
 * @beta
 */
export type Future = NamedArtifactContractDeploymentFuture<string> | ContractDeploymentFuture | NamedArtifactLibraryDeploymentFuture<string> | LibraryDeploymentFuture | ContractCallFuture<string, string> | StaticCallFuture<string, string> | EncodeFunctionCallFuture<string, string> | NamedArtifactContractAtFuture<string> | ContractAtFuture | ReadEventArgumentFuture | SendDataFuture;
/**
 * A future representing a contract. Either an existing one or one
 * that will be deployed.
 *
 * @beta
 */
export type ContractFuture<ContractNameT extends string> = NamedArtifactContractDeploymentFuture<ContractNameT> | ContractDeploymentFuture | NamedArtifactLibraryDeploymentFuture<ContractNameT> | LibraryDeploymentFuture | NamedArtifactContractAtFuture<ContractNameT> | ContractAtFuture;
/**
 * A future representing only contracts that can be called off-chain (i.e. not libraries).
 * Either an existing one or one that will be deployed.
 *
 * @beta
 */
export type CallableContractFuture<ContractNameT extends string> = NamedArtifactContractDeploymentFuture<ContractNameT> | ContractDeploymentFuture | NamedArtifactContractAtFuture<ContractNameT> | ContractAtFuture;
/**
 * A future representing a deployment.
 *
 * @beta
 */
export type DeploymentFuture<ContractNameT extends string> = NamedArtifactContractDeploymentFuture<ContractNameT> | ContractDeploymentFuture | NamedArtifactLibraryDeploymentFuture<ContractNameT> | LibraryDeploymentFuture;
/**
 * A future representing a call. Either a static one or one that modifies contract state
 *
 * @beta
 */
export type FunctionCallFuture<ContractNameT extends string, FunctionNameT extends string> = ContractCallFuture<ContractNameT, FunctionNameT> | StaticCallFuture<ContractNameT, FunctionNameT>;
/**
 * A future that can be resolved to a standard Ethereum address.
 *
 * @beta
 */
export type AddressResolvableFuture = ContractFuture<string> | StaticCallFuture<string, string> | ReadEventArgumentFuture;
/**
 * A future representing the deployment of a contract that belongs to this project.
 *
 * @beta
 */
export interface NamedArtifactContractDeploymentFuture<ContractNameT extends string> {
    type: FutureType.NAMED_ARTIFACT_CONTRACT_DEPLOYMENT;
    id: string;
    module: IgnitionModule;
    dependencies: Set<Future | IgnitionModule>;
    contractName: ContractNameT;
    constructorArgs: ArgumentType[];
    libraries: Record<string, ContractFuture<string>>;
    value: bigint | ModuleParameterRuntimeValue<bigint> | StaticCallFuture<string, string> | ReadEventArgumentFuture;
    from: string | AccountRuntimeValue | undefined;
}
/**
 * A future representing the deployment of a contract that we only know its artifact.
 * It may not belong to this project, and we may struggle to type.
 *
 * @beta
 */
export interface ContractDeploymentFuture<AbiT extends Abi = Abi> {
    type: FutureType.CONTRACT_DEPLOYMENT;
    id: string;
    module: IgnitionModule;
    dependencies: Set<Future | IgnitionModule>;
    contractName: string;
    artifact: Artifact<AbiT>;
    constructorArgs: ArgumentType[];
    libraries: Record<string, ContractFuture<string>>;
    value: bigint | ModuleParameterRuntimeValue<bigint> | StaticCallFuture<string, string> | ReadEventArgumentFuture;
    from: string | AccountRuntimeValue | undefined;
}
/**
 * A future representing the deployment of a library that belongs to this project
 *
 * @beta
 */
export interface NamedArtifactLibraryDeploymentFuture<LibraryNameT extends string> {
    type: FutureType.NAMED_ARTIFACT_LIBRARY_DEPLOYMENT;
    id: string;
    module: IgnitionModule;
    dependencies: Set<Future | IgnitionModule>;
    contractName: LibraryNameT;
    libraries: Record<string, ContractFuture<string>>;
    from: string | AccountRuntimeValue | undefined;
}
/**
 * A future representing the deployment of a library that we only know its artifact.
 * It may not belong to this project, and we may struggle to type.
 *
 * @beta
 */
export interface LibraryDeploymentFuture<AbiT extends Abi = Abi> {
    type: FutureType.LIBRARY_DEPLOYMENT;
    id: string;
    module: IgnitionModule;
    dependencies: Set<Future | IgnitionModule>;
    contractName: string;
    artifact: Artifact<AbiT>;
    libraries: Record<string, ContractFuture<string>>;
    from: string | AccountRuntimeValue | undefined;
}
/**
 * A future representing the calling of a contract function that modifies on-chain state
 *
 * @beta
 */
export interface ContractCallFuture<ContractNameT extends string, FunctionNameT extends string> {
    type: FutureType.CONTRACT_CALL;
    id: string;
    module: IgnitionModule;
    dependencies: Set<Future | IgnitionModule>;
    contract: ContractFuture<ContractNameT>;
    functionName: FunctionNameT;
    args: ArgumentType[];
    value: bigint | ModuleParameterRuntimeValue<bigint> | StaticCallFuture<string, string> | ReadEventArgumentFuture;
    from: string | AccountRuntimeValue | undefined;
}
/**
 * A future representing the static calling of a contract function that does not modify state
 *
 * @beta
 */
export interface StaticCallFuture<ContractNameT extends string, FunctionNameT extends string> {
    type: FutureType.STATIC_CALL;
    id: string;
    module: IgnitionModule;
    dependencies: Set<Future | IgnitionModule>;
    contract: ContractFuture<ContractNameT>;
    functionName: FunctionNameT;
    nameOrIndex: string | number;
    args: ArgumentType[];
    from: string | AccountRuntimeValue | undefined;
}
/**
 * A future representing the encoding of a contract function call
 *
 * @beta
 */
export interface EncodeFunctionCallFuture<ContractNameT extends string, FunctionNameT extends string> {
    type: FutureType.ENCODE_FUNCTION_CALL;
    id: string;
    module: IgnitionModule;
    dependencies: Set<Future | IgnitionModule>;
    contract: ContractFuture<ContractNameT>;
    functionName: FunctionNameT;
    args: ArgumentType[];
}
/**
 * A future representing a previously deployed contract at a known address that belongs to this project.
 *
 * @beta
 */
export interface NamedArtifactContractAtFuture<ContractNameT extends string> {
    type: FutureType.NAMED_ARTIFACT_CONTRACT_AT;
    id: string;
    module: IgnitionModule;
    dependencies: Set<Future | IgnitionModule>;
    contractName: ContractNameT;
    address: string | AddressResolvableFuture | ModuleParameterRuntimeValue<string>;
}
/**
 * A future representing a previously deployed contract at a known address with a given artifact.
 * It may not belong to this project, and we may struggle to type.
 *
 * @beta
 */
export interface ContractAtFuture<AbiT extends Abi = Abi> {
    type: FutureType.CONTRACT_AT;
    id: string;
    module: IgnitionModule;
    dependencies: Set<Future | IgnitionModule>;
    contractName: string;
    address: string | AddressResolvableFuture | ModuleParameterRuntimeValue<string>;
    artifact: Artifact<AbiT>;
}
/**
 * A future that represents reading an argument of an event emitted by the
 * transaction that executed another future.
 *
 * @beta
 */
export interface ReadEventArgumentFuture {
    type: FutureType.READ_EVENT_ARGUMENT;
    id: string;
    module: IgnitionModule;
    dependencies: Set<Future | IgnitionModule>;
    futureToReadFrom: NamedArtifactContractDeploymentFuture<string> | ContractDeploymentFuture | SendDataFuture | ContractCallFuture<string, string>;
    eventName: string;
    nameOrIndex: string | number;
    emitter: ContractFuture<string>;
    eventIndex: number;
}
/**
 * A future that represents sending arbitrary data to the EVM.
 *
 * @beta
 */
export interface SendDataFuture {
    type: FutureType.SEND_DATA;
    id: string;
    module: IgnitionModule;
    dependencies: Set<Future | IgnitionModule>;
    to: string | AddressResolvableFuture | ModuleParameterRuntimeValue<string> | AccountRuntimeValue;
    value: bigint | ModuleParameterRuntimeValue<bigint>;
    data: string | EncodeFunctionCallFuture<string, string> | undefined;
    from: string | AccountRuntimeValue | undefined;
}
/**
 * Base type of module parameters's values.
 *
 * @beta
 */
export type BaseSolidityParameterType = number | bigint | string | boolean;
/**
 * Types that can be passed across the Solidity ABI boundary.
 *
 * @beta
 */
export type SolidityParameterType = BaseSolidityParameterType | SolidityParameterType[] | {
    [field: string]: SolidityParameterType;
};
/**
 * Type of module parameters's values.
 *
 * @beta
 */
export type ModuleParameterType = SolidityParameterType | AccountRuntimeValue;
/**
 * The different runtime values supported by Ignition.
 *
 * @beta
 */
export declare enum RuntimeValueType {
    ACCOUNT = "ACCOUNT",
    MODULE_PARAMETER = "MODULE_PARAMETER"
}
/**
 * A value that's only available during deployment.
 *
 * @beta
 */
export type RuntimeValue = AccountRuntimeValue | ModuleParameterRuntimeValue<ModuleParameterType>;
/**
 * A local account.
 *
 * @beta
 */
export interface AccountRuntimeValue {
    type: RuntimeValueType.ACCOUNT;
    accountIndex: number;
}
/**
 * A module parameter.
 *
 * @beta
 */
export interface ModuleParameterRuntimeValue<ParamTypeT extends ModuleParameterType> {
    type: RuntimeValueType.MODULE_PARAMETER;
    moduleId: string;
    name: string;
    defaultValue: ParamTypeT | undefined;
}
/**
 * An object containing the parameters passed into the module.
 *
 * @beta
 */
export interface ModuleParameters {
    [parameterName: string]: SolidityParameterType;
}
/**
 * The results of deploying a module must be a dictionary of contract futures
 *
 * @beta
 */
export interface IgnitionModuleResult<ContractNameT extends string> {
    [name: string]: ContractFuture<ContractNameT>;
}
/**
 * A recipe for deploying and configuring contracts.
 *
 * @beta
 */
export interface IgnitionModule<ModuleIdT extends string = string, ContractNameT extends string = string, IgnitionModuleResultsT extends IgnitionModuleResult<ContractNameT> = IgnitionModuleResult<ContractNameT>> {
    id: ModuleIdT;
    futures: Set<Future>;
    submodules: Set<IgnitionModule>;
    results: IgnitionModuleResultsT;
}
//# sourceMappingURL=module.d.ts.map