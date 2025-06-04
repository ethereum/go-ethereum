"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.reconcileFutureSpecificReconciliations = void 0;
const module_1 = require("../../types/module");
const reconcileArtifactContractAt_1 = require("./futures/reconcileArtifactContractAt");
const reconcileArtifactContractDeployment_1 = require("./futures/reconcileArtifactContractDeployment");
const reconcileArtifactLibraryDeployment_1 = require("./futures/reconcileArtifactLibraryDeployment");
const reconcileNamedContractAt_1 = require("./futures/reconcileNamedContractAt");
const reconcileNamedContractCall_1 = require("./futures/reconcileNamedContractCall");
const reconcileNamedContractDeployment_1 = require("./futures/reconcileNamedContractDeployment");
const reconcileNamedEncodeFunctionCall_1 = require("./futures/reconcileNamedEncodeFunctionCall");
const reconcileNamedLibraryDeployment_1 = require("./futures/reconcileNamedLibraryDeployment");
const reconcileNamedStaticCall_1 = require("./futures/reconcileNamedStaticCall");
const reconcileReadEventArgument_1 = require("./futures/reconcileReadEventArgument");
const reconcileSendData_1 = require("./futures/reconcileSendData");
async function reconcileFutureSpecificReconciliations(future, executionState, context) {
    switch (future.type) {
        case module_1.FutureType.NAMED_ARTIFACT_CONTRACT_DEPLOYMENT:
            return (0, reconcileNamedContractDeployment_1.reconcileNamedContractDeployment)(future, executionState, context);
        case module_1.FutureType.CONTRACT_DEPLOYMENT:
            return (0, reconcileArtifactContractDeployment_1.reconcileArtifactContractDeployment)(future, executionState, context);
        case module_1.FutureType.NAMED_ARTIFACT_LIBRARY_DEPLOYMENT:
            return (0, reconcileNamedLibraryDeployment_1.reconcileNamedLibraryDeployment)(future, executionState, context);
        case module_1.FutureType.LIBRARY_DEPLOYMENT:
            return (0, reconcileArtifactLibraryDeployment_1.reconcileArtifactLibraryDeployment)(future, executionState, context);
        case module_1.FutureType.CONTRACT_CALL:
            return (0, reconcileNamedContractCall_1.reconcileNamedContractCall)(future, executionState, context);
        case module_1.FutureType.STATIC_CALL:
            return (0, reconcileNamedStaticCall_1.reconcileNamedStaticCall)(future, executionState, context);
        case module_1.FutureType.ENCODE_FUNCTION_CALL:
            return (0, reconcileNamedEncodeFunctionCall_1.reconcileNamedEncodeFunctionCall)(future, executionState, context);
        case module_1.FutureType.NAMED_ARTIFACT_CONTRACT_AT:
            return (0, reconcileNamedContractAt_1.reconcileNamedContractAt)(future, executionState, context);
        case module_1.FutureType.CONTRACT_AT: {
            return (0, reconcileArtifactContractAt_1.reconcileArtifactContractAt)(future, executionState, context);
        }
        case module_1.FutureType.READ_EVENT_ARGUMENT: {
            return (0, reconcileReadEventArgument_1.reconcileReadEventArgument)(future, executionState, context);
        }
        case module_1.FutureType.SEND_DATA: {
            return (0, reconcileSendData_1.reconcileSendData)(future, executionState, context);
        }
    }
}
exports.reconcileFutureSpecificReconciliations = reconcileFutureSpecificReconciliations;
//# sourceMappingURL=reconcile-future-specific-reconciliations.js.map