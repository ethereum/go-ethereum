"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.findAddressForContractFuture = void 0;
const execution_result_1 = require("../execution/types/execution-result");
const execution_state_1 = require("../execution/types/execution-state");
const assertions_1 = require("../utils/assertions");
/**
 * Find the address for the future by its id. Only works for ContractAt, NamedLibrary,
 * NamedContract, ArtifactLibrary, ArtifactContract as only they result in an
 * address on completion.
 *
 * Assumes that the future has been completed.
 *
 * @param deploymentState
 * @param futureId
 * @returns
 */
function findAddressForContractFuture(deploymentState, futureId) {
    const exState = deploymentState.executionStates[futureId];
    (0, assertions_1.assertIgnitionInvariant)(exState !== undefined, `Expected execution state for ${futureId} to exist, but it did not`);
    (0, assertions_1.assertIgnitionInvariant)(exState.type === execution_state_1.ExecutionSateType.DEPLOYMENT_EXECUTION_STATE ||
        exState.type === execution_state_1.ExecutionSateType.CONTRACT_AT_EXECUTION_STATE, `Can only resolve an address for a ContractAt, NamedLibrary, NamedContract, ArtifactLibrary, ArtifactContract`);
    if (exState.type === execution_state_1.ExecutionSateType.CONTRACT_AT_EXECUTION_STATE) {
        return exState.contractAddress;
    }
    (0, assertions_1.assertIgnitionInvariant)(exState.result !== undefined, `Expected execution state for ${futureId} to have a result, but it did not`);
    (0, assertions_1.assertIgnitionInvariant)(exState.result.type === execution_result_1.ExecutionResultType.SUCCESS, `Cannot access the result of ${futureId}, it was not a deployment success`);
    return exState.result.address;
}
exports.findAddressForContractFuture = findAddressForContractFuture;
//# sourceMappingURL=find-address-for-contract-future-by-id.js.map