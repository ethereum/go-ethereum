"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.findDeployedContracts = void 0;
const execution_result_1 = require("../execution/types/execution-result");
const execution_state_1 = require("../execution/types/execution-state");
const assertions_1 = require("../utils/assertions");
function findDeployedContracts(deploymentState) {
    return Object.values(deploymentState.executionStates)
        .filter((exState) => exState.type === execution_state_1.ExecutionSateType.DEPLOYMENT_EXECUTION_STATE ||
        exState.type === execution_state_1.ExecutionSateType.CONTRACT_AT_EXECUTION_STATE)
        .filter((des) => des.status === execution_state_1.ExecutionStatus.SUCCESS)
        .map(_toDeployedContract)
        .reduce((acc, contract) => {
        acc[contract.id] = contract;
        return acc;
    }, {});
}
exports.findDeployedContracts = findDeployedContracts;
function _toDeployedContract(des) {
    switch (des.type) {
        case execution_state_1.ExecutionSateType.DEPLOYMENT_EXECUTION_STATE: {
            (0, assertions_1.assertIgnitionInvariant)(des.result !== undefined &&
                des.result.type === execution_result_1.ExecutionResultType.SUCCESS, `Deployment execution state ${des.id} should have a successful result to retrieve address`);
            return {
                id: des.id,
                contractName: des.contractName,
                address: des.result.address,
            };
        }
        case execution_state_1.ExecutionSateType.CONTRACT_AT_EXECUTION_STATE: {
            return {
                id: des.id,
                contractName: des.contractName,
                address: des.contractAddress,
            };
        }
    }
}
//# sourceMappingURL=find-deployed-contracts.js.map