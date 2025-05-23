"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.resolveEncodeFunctionCallResult = exports.resolveReadEventArgumentResult = exports.resolveAddressLike = exports.resolveSendToAddress = exports.resolveAddressForContractFuture = exports.resolveLibraries = exports.resolveAccountRuntimeValue = exports.resolveFutureData = exports.resolveFutureFrom = exports.resolveArgs = exports.resolveValue = void 0;
const type_guards_1 = require("../../../../type-guards");
const assertions_1 = require("../../../utils/assertions");
const replace_within_arg_1 = require("../../../utils/replace-within-arg");
const resolve_module_parameter_1 = require("../../../utils/resolve-module-parameter");
const find_address_for_contract_future_by_id_1 = require("../../../views/find-address-for-contract-future-by-id");
const find_confirmed_transaction_by_future_id_1 = require("../../../views/find-confirmed-transaction-by-future-id");
const find_result_for_future_by_id_1 = require("../../../views/find-result-for-future-by-id");
const abi_1 = require("../../abi");
const convert_evm_tuple_to_solidity_param_1 = require("../../utils/convert-evm-tuple-to-solidity-param");
/**
 * Resolve a futures value to a bigint.
 *
 * @param givenValue - either a bigint or a module parameter runtime value
 * @param deploymentParameters - the user provided deployment parameters
 * @returns the resolved bigint
 */
function resolveValue(givenValue, deploymentParameters, deploymentState, accounts) {
    if (typeof givenValue === "bigint") {
        return givenValue;
    }
    let result;
    if ((0, type_guards_1.isFuture)(givenValue)) {
        result = (0, find_result_for_future_by_id_1.findResultForFutureById)(deploymentState, givenValue.id);
    }
    else {
        result = (0, resolve_module_parameter_1.resolveModuleParameter)(givenValue, {
            deploymentParameters,
            accounts,
        });
    }
    (0, assertions_1.assertIgnitionInvariant)(typeof result === "bigint", "Module parameter or future result used as value must be a bigint");
    return result;
}
exports.resolveValue = resolveValue;
/**
 * Recursively resolve an arguments array, replacing any runtime values
 * or futures with their resolved values.
 */
function resolveArgs(args, deploymentState, deploymentParameters, accounts) {
    const replace = (arg) => (0, replace_within_arg_1.replaceWithinArg)(arg, {
        bigint: (bi) => bi,
        future: (f) => {
            return (0, find_result_for_future_by_id_1.findResultForFutureById)(deploymentState, f.id);
        },
        accountRuntimeValue: (arv) => {
            return resolveAccountRuntimeValue(arv, accounts);
        },
        moduleParameterRuntimeValue: (mprv) => {
            return (0, resolve_module_parameter_1.resolveModuleParameter)(mprv, {
                deploymentParameters,
                accounts,
            });
        },
    });
    return args.map(replace);
}
exports.resolveArgs = resolveArgs;
/**
 * Resolve a future's from field to either undefined (meaning defer until execution)
 * or a string address.
 */
function resolveFutureFrom(from, accounts, defaultSender) {
    if (from === undefined) {
        return defaultSender;
    }
    if (typeof from === "string") {
        return from;
    }
    return resolveAccountRuntimeValue(from, accounts);
}
exports.resolveFutureFrom = resolveFutureFrom;
/**
 * Resolve a `send` future's data parameter to a string.
 */
function resolveFutureData(data, deploymentState) {
    if (data === undefined) {
        return "0x";
    }
    if (typeof data === "string") {
        return data;
    }
    const result = (0, find_result_for_future_by_id_1.findResultForFutureById)(deploymentState, data.id);
    (0, assertions_1.assertIgnitionInvariant)(typeof result === "string", "Expected future data to be a string");
    return result;
}
exports.resolveFutureData = resolveFutureData;
/**
 * Resolves an account runtime value to an address.
 */
function resolveAccountRuntimeValue(arv, accounts) {
    const address = accounts[arv.accountIndex];
    (0, assertions_1.assertIgnitionInvariant)(address !== undefined, `Account ${arv.accountIndex} not found`);
    return address;
}
exports.resolveAccountRuntimeValue = resolveAccountRuntimeValue;
/**
 * Resolve a futures dependent libraries to a map of library names to addresses.
 */
function resolveLibraries(libraries, deploymentState) {
    return Object.fromEntries(Object.entries(libraries).map(([key, lib]) => [
        key,
        (0, find_address_for_contract_future_by_id_1.findAddressForContractFuture)(deploymentState, lib.id),
    ]));
}
exports.resolveLibraries = resolveLibraries;
/**
 * Resolve a contract future down to the address it is deployed at.
 */
function resolveAddressForContractFuture(contract, deploymentState) {
    return (0, find_address_for_contract_future_by_id_1.findAddressForContractFuture)(deploymentState, contract.id);
}
exports.resolveAddressForContractFuture = resolveAddressForContractFuture;
/**
 * Resolve a SendDataFuture's "to" field to a valid ethereum address.
 */
function resolveSendToAddress(to, deploymentState, deploymentParameters, accounts) {
    if (typeof to === "string") {
        return to;
    }
    if ((0, type_guards_1.isAccountRuntimeValue)(to)) {
        return resolveAccountRuntimeValue(to, accounts);
    }
    return resolveAddressLike(to, deploymentState, deploymentParameters, accounts);
}
exports.resolveSendToAddress = resolveSendToAddress;
/**
 * Resolve the given address like to a valid ethereum address. Futures
 * will be resolved to their result then runtime checked to ensure
 * they are a valid address.
 */
function resolveAddressLike(addressLike, deploymentState, deploymentParameters, accounts) {
    if (typeof addressLike === "string") {
        return addressLike;
    }
    if ((0, type_guards_1.isModuleParameterRuntimeValue)(addressLike)) {
        const addressFromParam = (0, resolve_module_parameter_1.resolveModuleParameter)(addressLike, {
            deploymentParameters,
            accounts,
        });
        (0, assertions_1.assertIgnitionInvariant)(typeof addressFromParam === "string", "Module parameter used as address must be a string");
        return addressFromParam;
    }
    const result = (0, find_result_for_future_by_id_1.findResultForFutureById)(deploymentState, addressLike.id);
    const { isAddress } = require("ethers");
    (0, assertions_1.assertIgnitionInvariant)(typeof result === "string" && isAddress(result), `Future '${addressLike.id}' must be a valid address`);
    return result;
}
exports.resolveAddressLike = resolveAddressLike;
/**
 * Resolves a read event argument result to a SolidityParameterType.
 */
async function resolveReadEventArgumentResult(future, emitter, eventName, eventIndex, nameOrIndex, deploymentState, deploymentLoader) {
    const emitterAddress = resolveAddressForContractFuture(emitter, deploymentState);
    const emitterArtifact = await deploymentLoader.loadArtifact(emitter.id);
    const confirmedTx = (0, find_confirmed_transaction_by_future_id_1.findConfirmedTransactionByFutureId)(deploymentState, future.id);
    const evmValue = (0, abi_1.getEventArgumentFromReceipt)(confirmedTx.receipt, emitterArtifact, emitterAddress, eventName, eventIndex, nameOrIndex);
    return {
        result: (0, convert_evm_tuple_to_solidity_param_1.convertEvmValueToSolidityParam)(evmValue),
        emitterAddress,
        txToReadFrom: confirmedTx.hash,
    };
}
exports.resolveReadEventArgumentResult = resolveReadEventArgumentResult;
async function resolveEncodeFunctionCallResult(artifactId, functionName, args, deploymentLoader) {
    const artifact = await deploymentLoader.loadArtifact(artifactId);
    const { Interface } = require("ethers");
    const iface = new Interface(artifact.abi);
    return iface.encodeFunctionData(functionName, args);
}
exports.resolveEncodeFunctionCallResult = resolveEncodeFunctionCallResult;
//# sourceMappingURL=future-resolvers.js.map