"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.executionStateReducer = void 0;
const assertions_1 = require("../../utils/assertions");
const execution_state_1 = require("../types/execution-state");
const messages_1 = require("../types/messages");
const complete_execution_state_1 = require("./helpers/complete-execution-state");
const initializers_1 = require("./helpers/initializers");
const network_interaction_helpers_1 = require("./helpers/network-interaction-helpers");
const exStateTypesThatSupportOnchainInteractions = [
    execution_state_1.ExecutionStateType.DEPLOYMENT_EXECUTION_STATE,
    execution_state_1.ExecutionStateType.CALL_EXECUTION_STATE,
    execution_state_1.ExecutionStateType.SEND_DATA_EXECUTION_STATE,
];
const exStateTypesThatSupportOnchainInteractionsAndStaticCalls = [
    ...exStateTypesThatSupportOnchainInteractions,
    execution_state_1.ExecutionStateType.STATIC_CALL_EXECUTION_STATE,
];
function executionStateReducer(state, action) {
    switch (action.type) {
        case messages_1.JournalMessageType.DEPLOYMENT_EXECUTION_STATE_INITIALIZE:
            return (0, initializers_1.initialiseDeploymentExecutionStateFrom)(action);
        case messages_1.JournalMessageType.CALL_EXECUTION_STATE_INITIALIZE:
            return (0, initializers_1.initialiseCallExecutionStateFrom)(action);
        case messages_1.JournalMessageType.STATIC_CALL_EXECUTION_STATE_INITIALIZE:
            return (0, initializers_1.initialiseStaticCallExecutionStateFrom)(action);
        case messages_1.JournalMessageType.SEND_DATA_EXECUTION_STATE_INITIALIZE:
            return (0, initializers_1.initialiseSendDataExecutionStateFrom)(action);
        case messages_1.JournalMessageType.CONTRACT_AT_EXECUTION_STATE_INITIALIZE:
            return (0, initializers_1.initialiseContractAtExecutionStateFrom)(action);
        case messages_1.JournalMessageType.READ_EVENT_ARGUMENT_EXECUTION_STATE_INITIALIZE:
            return (0, initializers_1.initialiseReadEventArgumentExecutionStateFrom)(action);
        case messages_1.JournalMessageType.ENCODE_FUNCTION_CALL_EXECUTION_STATE_INITIALIZE:
            return (0, initializers_1.initialiseEncodeFunctionCallExecutionStateFrom)(action);
        case messages_1.JournalMessageType.DEPLOYMENT_EXECUTION_STATE_COMPLETE:
            return _ensureStateThen(state, action, [execution_state_1.ExecutionStateType.DEPLOYMENT_EXECUTION_STATE], complete_execution_state_1.completeExecutionState);
        case messages_1.JournalMessageType.CALL_EXECUTION_STATE_COMPLETE:
            return _ensureStateThen(state, action, [execution_state_1.ExecutionStateType.CALL_EXECUTION_STATE], complete_execution_state_1.completeExecutionState);
        case messages_1.JournalMessageType.STATIC_CALL_EXECUTION_STATE_COMPLETE:
            return _ensureStateThen(state, action, [execution_state_1.ExecutionStateType.STATIC_CALL_EXECUTION_STATE], complete_execution_state_1.completeExecutionState);
        case messages_1.JournalMessageType.SEND_DATA_EXECUTION_STATE_COMPLETE:
            return _ensureStateThen(state, action, [execution_state_1.ExecutionStateType.SEND_DATA_EXECUTION_STATE], complete_execution_state_1.completeExecutionState);
        case messages_1.JournalMessageType.NETWORK_INTERACTION_REQUEST:
            return _ensureStateThen(state, action, exStateTypesThatSupportOnchainInteractionsAndStaticCalls, network_interaction_helpers_1.appendNetworkInteraction);
        case messages_1.JournalMessageType.STATIC_CALL_COMPLETE:
            return _ensureStateThen(state, action, exStateTypesThatSupportOnchainInteractionsAndStaticCalls, network_interaction_helpers_1.completeStaticCall);
        case messages_1.JournalMessageType.TRANSACTION_PREPARE_SEND:
            return _ensureStateThen(state, action, exStateTypesThatSupportOnchainInteractions, network_interaction_helpers_1.applyNonceToOnchainInteraction);
        case messages_1.JournalMessageType.TRANSACTION_SEND:
            return _ensureStateThen(state, action, exStateTypesThatSupportOnchainInteractions, network_interaction_helpers_1.appendTransactionToOnchainInteraction);
        case messages_1.JournalMessageType.TRANSACTION_CONFIRM:
            return _ensureStateThen(state, action, exStateTypesThatSupportOnchainInteractions, network_interaction_helpers_1.confirmTransaction);
        case messages_1.JournalMessageType.ONCHAIN_INTERACTION_BUMP_FEES:
            return _ensureStateThen(state, action, exStateTypesThatSupportOnchainInteractions, network_interaction_helpers_1.bumpOnchainInteractionFees);
        case messages_1.JournalMessageType.ONCHAIN_INTERACTION_DROPPED:
            return _ensureStateThen(state, action, exStateTypesThatSupportOnchainInteractions, network_interaction_helpers_1.resendDroppedOnchainInteraction);
        case messages_1.JournalMessageType.ONCHAIN_INTERACTION_REPLACED_BY_USER:
            return _ensureStateThen(state, action, exStateTypesThatSupportOnchainInteractions, network_interaction_helpers_1.resetOnchainInteractionReplacedByUser);
        case messages_1.JournalMessageType.ONCHAIN_INTERACTION_TIMEOUT:
            return _ensureStateThen(state, action, exStateTypesThatSupportOnchainInteractions, network_interaction_helpers_1.onchainInteractionTimedOut);
    }
}
exports.executionStateReducer = executionStateReducer;
/**
 * Ensure the execution state is defined and of the correct type, then
 * run the given `then` function.
 *
 * @param state - the execution state
 * @param action - the message to reduce
 * @param allowedExStateTypes - the allowed execution states for the message
 * @param then - the reducer that will be passed the checked state and message
 * @returns a copy of the updated execution state
 */
function _ensureStateThen(state, action, allowedExStateTypes, then) {
    (0, assertions_1.assertIgnitionInvariant)(state !== undefined, `Execution state must be defined`);
    (0, assertions_1.assertIgnitionInvariant)(allowedExStateTypes.includes(state.type), `The execution state ${state.type} is not supported`);
    return then(state, action);
}
//# sourceMappingURL=execution-state-reducer.js.map