"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.buildInitializeMessageFor = void 0;
const module_1 = require("../../../../types/module");
const messages_1 = require("../../types/messages");
const future_resolvers_1 = require("./future-resolvers");
async function buildInitializeMessageFor(future, deploymentState, strategy, deploymentParameters, deploymentLoader, accounts, defaultSender) {
    switch (future.type) {
        case module_1.FutureType.NAMED_ARTIFACT_CONTRACT_DEPLOYMENT:
        case module_1.FutureType.CONTRACT_DEPLOYMENT:
            const deploymentExecStateInit = _extendBaseInitWith(messages_1.JournalMessageType.DEPLOYMENT_EXECUTION_STATE_INITIALIZE, future, strategy.name, strategy.config, {
                futureType: future.type,
                artifactId: future.id,
                contractName: future.contractName,
                constructorArgs: (0, future_resolvers_1.resolveArgs)(future.constructorArgs, deploymentState, deploymentParameters, accounts),
                libraries: (0, future_resolvers_1.resolveLibraries)(future.libraries, deploymentState),
                value: (0, future_resolvers_1.resolveValue)(future.value, deploymentParameters, deploymentState, accounts),
                from: (0, future_resolvers_1.resolveFutureFrom)(future.from, accounts, defaultSender),
            });
            return deploymentExecStateInit;
        case module_1.FutureType.NAMED_ARTIFACT_LIBRARY_DEPLOYMENT:
        case module_1.FutureType.LIBRARY_DEPLOYMENT:
            const libraryDeploymentInit = _extendBaseInitWith(messages_1.JournalMessageType.DEPLOYMENT_EXECUTION_STATE_INITIALIZE, future, strategy.name, strategy.config, {
                futureType: future.type,
                artifactId: future.id,
                contractName: future.contractName,
                constructorArgs: [],
                libraries: (0, future_resolvers_1.resolveLibraries)(future.libraries, deploymentState),
                value: BigInt(0),
                from: (0, future_resolvers_1.resolveFutureFrom)(future.from, accounts, defaultSender),
            });
            return libraryDeploymentInit;
        case module_1.FutureType.CONTRACT_CALL: {
            const namedContractCallInit = _extendBaseInitWith(messages_1.JournalMessageType.CALL_EXECUTION_STATE_INITIALIZE, future, strategy.name, strategy.config, {
                args: (0, future_resolvers_1.resolveArgs)(future.args, deploymentState, deploymentParameters, accounts),
                functionName: future.functionName,
                contractAddress: (0, future_resolvers_1.resolveAddressForContractFuture)(future.contract, deploymentState),
                artifactId: future.contract.id,
                value: (0, future_resolvers_1.resolveValue)(future.value, deploymentParameters, deploymentState, accounts),
                from: (0, future_resolvers_1.resolveFutureFrom)(future.from, accounts, defaultSender),
            });
            return namedContractCallInit;
        }
        case module_1.FutureType.STATIC_CALL: {
            const namedStaticCallInit = _extendBaseInitWith(messages_1.JournalMessageType.STATIC_CALL_EXECUTION_STATE_INITIALIZE, future, strategy.name, strategy.config, {
                args: (0, future_resolvers_1.resolveArgs)(future.args, deploymentState, deploymentParameters, accounts),
                nameOrIndex: future.nameOrIndex,
                functionName: future.functionName,
                contractAddress: (0, future_resolvers_1.resolveAddressForContractFuture)(future.contract, deploymentState),
                artifactId: future.contract.id,
                from: (0, future_resolvers_1.resolveFutureFrom)(future.from, accounts, defaultSender),
            });
            return namedStaticCallInit;
        }
        case module_1.FutureType.ENCODE_FUNCTION_CALL: {
            const args = (0, future_resolvers_1.resolveArgs)(future.args, deploymentState, deploymentParameters, accounts);
            const result = await (0, future_resolvers_1.resolveEncodeFunctionCallResult)(future.contract.id, future.functionName, args, deploymentLoader);
            const encodeFunctionCallInit = _extendBaseInitWith(messages_1.JournalMessageType.ENCODE_FUNCTION_CALL_EXECUTION_STATE_INITIALIZE, future, strategy.name, strategy.config, {
                args,
                functionName: future.functionName,
                artifactId: future.contract.id,
                result,
            });
            return encodeFunctionCallInit;
        }
        case module_1.FutureType.NAMED_ARTIFACT_CONTRACT_AT:
        case module_1.FutureType.CONTRACT_AT: {
            const contractAtInit = _extendBaseInitWith(messages_1.JournalMessageType.CONTRACT_AT_EXECUTION_STATE_INITIALIZE, future, strategy.name, strategy.config, {
                futureType: future.type,
                contractName: future.contractName,
                contractAddress: (0, future_resolvers_1.resolveAddressLike)(future.address, deploymentState, deploymentParameters, accounts),
                artifactId: future.id,
            });
            return contractAtInit;
        }
        case module_1.FutureType.READ_EVENT_ARGUMENT: {
            const { txToReadFrom, emitterAddress, result } = await (0, future_resolvers_1.resolveReadEventArgumentResult)(future.futureToReadFrom, future.emitter, future.eventName, future.eventIndex, future.nameOrIndex, deploymentState, deploymentLoader);
            const readEventArgInit = _extendBaseInitWith(messages_1.JournalMessageType.READ_EVENT_ARGUMENT_EXECUTION_STATE_INITIALIZE, future, strategy.name, strategy.config, {
                artifactId: future.emitter.id,
                eventName: future.eventName,
                nameOrIndex: future.nameOrIndex,
                eventIndex: future.eventIndex,
                txToReadFrom,
                emitterAddress,
                result,
            });
            return readEventArgInit;
        }
        case module_1.FutureType.SEND_DATA:
            const sendDataInit = _extendBaseInitWith(messages_1.JournalMessageType.SEND_DATA_EXECUTION_STATE_INITIALIZE, future, strategy.name, strategy.config, {
                to: (0, future_resolvers_1.resolveSendToAddress)(future.to, deploymentState, deploymentParameters, accounts),
                value: (0, future_resolvers_1.resolveValue)(future.value, deploymentParameters, deploymentState, accounts),
                data: (0, future_resolvers_1.resolveFutureData)(future.data, deploymentState),
                from: (0, future_resolvers_1.resolveFutureFrom)(future.from, accounts, defaultSender),
            });
            return sendDataInit;
    }
}
exports.buildInitializeMessageFor = buildInitializeMessageFor;
function _extendBaseInitWith(messageType, future, strategy, strategyConfig, extension) {
    return {
        type: messageType,
        futureId: future.id,
        strategy,
        strategyConfig,
        dependencies: [...future.dependencies].map((f) => f.id),
        ...extension,
    };
}
//# sourceMappingURL=build-initialize-message-for.js.map