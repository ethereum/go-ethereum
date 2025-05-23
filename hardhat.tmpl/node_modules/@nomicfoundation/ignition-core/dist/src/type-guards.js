"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.isModuleParameterRuntimeValue = exports.isAccountRuntimeValue = exports.isRuntimeValue = exports.isRuntimeValueType = exports.isFutureThatSubmitsOnchainTransaction = exports.isDeploymentFuture = exports.isDeploymentType = exports.isArtifactContractAtFuture = exports.isNamedContractAtFuture = exports.isArtifactLibraryDeploymentFuture = exports.isNamedLibraryDeploymentFuture = exports.isArtifactContractDeploymentFuture = exports.isNamedContractDeploymentFuture = exports.isReadEventArgumentFuture = exports.isEncodeFunctionCallFuture = exports.isNamedStaticCallFuture = exports.isFunctionCallFuture = exports.isAddressResolvableFuture = exports.isCallableContractFuture = exports.isContractFuture = exports.isFuture = exports.isFutureType = exports.isArtifactType = void 0;
const module_1 = require("./types/module");
function isValidEnumValue(theEnum, value) {
    // Enums are objects that have entries that map:
    //   1) keys to values
    //   2) values to keys
    const key = theEnum[value];
    if (key === undefined) {
        return false;
    }
    return theEnum[key] === value;
}
/**
 * Returns true if potential is of type Artifact.
 *
 * @beta
 */
function isArtifactType(potential) {
    return (typeof potential === "object" &&
        potential !== null &&
        "contractName" in potential &&
        "bytecode" in potential &&
        "abi" in potential &&
        "linkReferences" in potential &&
        typeof potential.contractName === "string" &&
        typeof potential.bytecode === "string" &&
        Array.isArray(potential.abi) &&
        typeof potential.linkReferences === "object");
}
exports.isArtifactType = isArtifactType;
/**
 * Returns true if potential is of type FutureType.
 *
 * @beta
 */
function isFutureType(potential) {
    return (typeof potential === "string" && isValidEnumValue(module_1.FutureType, potential));
}
exports.isFutureType = isFutureType;
/**
 * Returns true if potential is of type Future.
 *
 * @beta
 */
function isFuture(potential) {
    return (typeof potential === "object" &&
        potential !== null &&
        "type" in potential &&
        isFutureType(potential.type));
}
exports.isFuture = isFuture;
/**
 * Returns true if future is of type ContractFuture<string>.
 *
 * @beta
 */
function isContractFuture(future) {
    switch (future.type) {
        case module_1.FutureType.NAMED_ARTIFACT_CONTRACT_DEPLOYMENT:
        case module_1.FutureType.CONTRACT_DEPLOYMENT:
        case module_1.FutureType.NAMED_ARTIFACT_LIBRARY_DEPLOYMENT:
        case module_1.FutureType.LIBRARY_DEPLOYMENT:
        case module_1.FutureType.NAMED_ARTIFACT_CONTRACT_AT:
        case module_1.FutureType.CONTRACT_AT:
            return true;
        default:
            return false;
    }
}
exports.isContractFuture = isContractFuture;
/**
 * Returns true if future is of type CallableContractFuture<string>.
 *
 * @beta
 */
function isCallableContractFuture(future) {
    switch (future.type) {
        case module_1.FutureType.NAMED_ARTIFACT_CONTRACT_DEPLOYMENT:
        case module_1.FutureType.CONTRACT_DEPLOYMENT:
        case module_1.FutureType.NAMED_ARTIFACT_CONTRACT_AT:
        case module_1.FutureType.CONTRACT_AT:
            return true;
        default:
            return false;
    }
}
exports.isCallableContractFuture = isCallableContractFuture;
/**
 * Returns true if future is of type AddressResolvable.
 *
 * @beta
 */
function isAddressResolvableFuture(future) {
    return (isContractFuture(future) ||
        future.type === module_1.FutureType.STATIC_CALL ||
        future.type === module_1.FutureType.READ_EVENT_ARGUMENT);
}
exports.isAddressResolvableFuture = isAddressResolvableFuture;
/**
 * Returns true if future is of type FunctionCallFuture\<string, string\>.
 *
 * @beta
 */
function isFunctionCallFuture(future) {
    return (future.type === module_1.FutureType.CONTRACT_CALL ||
        future.type === module_1.FutureType.STATIC_CALL);
}
exports.isFunctionCallFuture = isFunctionCallFuture;
/**
 * Returns true if future is of type NamedStaticCallFuture.
 *
 * @beta
 */
function isNamedStaticCallFuture(future) {
    return future.type === module_1.FutureType.STATIC_CALL;
}
exports.isNamedStaticCallFuture = isNamedStaticCallFuture;
/**
 * Returns true if future is of type EncodeFunctionCallFuture\<string, string\>.
 *
 * @beta
 */
function isEncodeFunctionCallFuture(potential) {
    return (isFuture(potential) && potential.type === module_1.FutureType.ENCODE_FUNCTION_CALL);
}
exports.isEncodeFunctionCallFuture = isEncodeFunctionCallFuture;
/**
 * Returns true if future is of type ReadEventArgumentFuture.
 *
 * @beta
 */
function isReadEventArgumentFuture(future) {
    return future.type === module_1.FutureType.READ_EVENT_ARGUMENT;
}
exports.isReadEventArgumentFuture = isReadEventArgumentFuture;
/**
 * Returns true if future is of type NamedContractDeploymentFuture.
 *
 * @beta
 */
function isNamedContractDeploymentFuture(future) {
    return future.type === module_1.FutureType.NAMED_ARTIFACT_CONTRACT_DEPLOYMENT;
}
exports.isNamedContractDeploymentFuture = isNamedContractDeploymentFuture;
/**
 * Returns true if future is of type ArtifactContractDeploymentFuture.
 *
 * @beta
 */
function isArtifactContractDeploymentFuture(future) {
    return future.type === module_1.FutureType.CONTRACT_DEPLOYMENT;
}
exports.isArtifactContractDeploymentFuture = isArtifactContractDeploymentFuture;
/**
 * Returns true if future is of type NamedLibraryDeploymentFuture.
 *
 * @beta
 */
function isNamedLibraryDeploymentFuture(future) {
    return future.type === module_1.FutureType.NAMED_ARTIFACT_LIBRARY_DEPLOYMENT;
}
exports.isNamedLibraryDeploymentFuture = isNamedLibraryDeploymentFuture;
/**
 * Returns true if future is of type ArtifactLibraryDeploymentFuture.
 *
 * @beta
 */
function isArtifactLibraryDeploymentFuture(future) {
    return future.type === module_1.FutureType.LIBRARY_DEPLOYMENT;
}
exports.isArtifactLibraryDeploymentFuture = isArtifactLibraryDeploymentFuture;
/**
 * Returns true if future is of type NamedContractAtFuture.
 *
 * @beta
 */
function isNamedContractAtFuture(future) {
    return future.type === module_1.FutureType.NAMED_ARTIFACT_CONTRACT_AT;
}
exports.isNamedContractAtFuture = isNamedContractAtFuture;
/**
 * Returns true if future is of type ArtifactContractAtFuture.
 *
 * @beta
 */
function isArtifactContractAtFuture(future) {
    return future.type === module_1.FutureType.CONTRACT_AT;
}
exports.isArtifactContractAtFuture = isArtifactContractAtFuture;
/**
 * Returns true if the type is of type DeploymentFuture<string>.
 *
 * @beta
 */
function isDeploymentType(potential) {
    const deploymentTypes = [
        module_1.FutureType.NAMED_ARTIFACT_CONTRACT_DEPLOYMENT,
        module_1.FutureType.CONTRACT_DEPLOYMENT,
        module_1.FutureType.NAMED_ARTIFACT_LIBRARY_DEPLOYMENT,
        module_1.FutureType.LIBRARY_DEPLOYMENT,
    ];
    return (typeof potential === "string" &&
        deploymentTypes.includes(potential));
}
exports.isDeploymentType = isDeploymentType;
/**
 * Returns true if future is of type DeploymentFuture<string>.
 *
 * @beta
 */
function isDeploymentFuture(future) {
    return isDeploymentType(future.type);
}
exports.isDeploymentFuture = isDeploymentFuture;
/**
 * Returns true if the future requires submitting a transaction on-chain
 *
 * @beta
 */
function isFutureThatSubmitsOnchainTransaction(f) {
    return (!isNamedStaticCallFuture(f) &&
        !isReadEventArgumentFuture(f) &&
        !isNamedContractAtFuture(f) &&
        !isArtifactContractAtFuture(f));
}
exports.isFutureThatSubmitsOnchainTransaction = isFutureThatSubmitsOnchainTransaction;
/**
 * Returns true if potential is of type RuntimeValueType.
 *
 * @beta
 */
function isRuntimeValueType(potential) {
    return (typeof potential === "string" &&
        isValidEnumValue(module_1.RuntimeValueType, potential));
}
exports.isRuntimeValueType = isRuntimeValueType;
/**
 * Returns true if potential is of type RuntimeValue.
 *
 * @beta
 */
function isRuntimeValue(potential) {
    return (typeof potential === "object" &&
        potential !== null &&
        "type" in potential &&
        isRuntimeValueType(potential.type));
}
exports.isRuntimeValue = isRuntimeValue;
/**
 * Return true if potential is an account runtime value.
 *
 * @beta
 */
function isAccountRuntimeValue(potential) {
    return (isRuntimeValue(potential) && potential.type === module_1.RuntimeValueType.ACCOUNT);
}
exports.isAccountRuntimeValue = isAccountRuntimeValue;
/**
 * Returns true if potential is of type ModuleParameterRuntimeValue<any>.
 *
 * @beta
 */
function isModuleParameterRuntimeValue(potential) {
    return (isRuntimeValue(potential) &&
        potential.type === module_1.RuntimeValueType.MODULE_PARAMETER);
}
exports.isModuleParameterRuntimeValue = isModuleParameterRuntimeValue;
//# sourceMappingURL=type-guards.js.map