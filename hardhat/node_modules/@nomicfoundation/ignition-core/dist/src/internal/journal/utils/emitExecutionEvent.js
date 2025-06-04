"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.emitExecutionEvent = void 0;
const execution_events_1 = require("../../../types/execution-events");
const execution_result_1 = require("../../execution/types/execution-result");
const messages_1 = require("../../execution/types/messages");
const network_interaction_1 = require("../../execution/types/network-interaction");
const failedEvmExecutionResultToErrorDescription_1 = require("./failedEvmExecutionResultToErrorDescription");
function emitExecutionEvent(message, executionEventListener) {
    switch (message.type) {
        case messages_1.JournalMessageType.DEPLOYMENT_INITIALIZE: {
            executionEventListener.deploymentInitialize({
                type: execution_events_1.ExecutionEventType.DEPLOYMENT_INITIALIZE,
                chainId: message.chainId,
            });
            break;
        }
        case messages_1.JournalMessageType.WIPE_APPLY: {
            executionEventListener.wipeApply({
                type: execution_events_1.ExecutionEventType.WIPE_APPLY,
                futureId: message.futureId,
            });
            break;
        }
        case messages_1.JournalMessageType.DEPLOYMENT_EXECUTION_STATE_INITIALIZE: {
            executionEventListener.deploymentExecutionStateInitialize({
                type: execution_events_1.ExecutionEventType.DEPLOYMENT_EXECUTION_STATE_INITIALIZE,
                futureId: message.futureId,
            });
            break;
        }
        case messages_1.JournalMessageType.DEPLOYMENT_EXECUTION_STATE_COMPLETE: {
            executionEventListener.deploymentExecutionStateComplete({
                type: execution_events_1.ExecutionEventType.DEPLOYMENT_EXECUTION_STATE_COMPLETE,
                futureId: message.futureId,
                result: convertExecutionResultToEventResult(message.result),
            });
            break;
        }
        case messages_1.JournalMessageType.CALL_EXECUTION_STATE_INITIALIZE: {
            executionEventListener.callExecutionStateInitialize({
                type: execution_events_1.ExecutionEventType.CALL_EXECUTION_STATE_INITIALIZE,
                futureId: message.futureId,
            });
            break;
        }
        case messages_1.JournalMessageType.CALL_EXECUTION_STATE_COMPLETE: {
            executionEventListener.callExecutionStateComplete({
                type: execution_events_1.ExecutionEventType.CALL_EXECUTION_STATE_COMPLETE,
                futureId: message.futureId,
                result: convertExecutionResultToEventResult(message.result),
            });
            break;
        }
        case messages_1.JournalMessageType.STATIC_CALL_EXECUTION_STATE_INITIALIZE: {
            executionEventListener.staticCallExecutionStateInitialize({
                type: execution_events_1.ExecutionEventType.STATIC_CALL_EXECUTION_STATE_INITIALIZE,
                futureId: message.futureId,
            });
            break;
        }
        case messages_1.JournalMessageType.STATIC_CALL_EXECUTION_STATE_COMPLETE: {
            executionEventListener.staticCallExecutionStateComplete({
                type: execution_events_1.ExecutionEventType.STATIC_CALL_EXECUTION_STATE_COMPLETE,
                futureId: message.futureId,
                result: convertStaticCallResultToExecutionEventResult(message.result),
            });
            break;
        }
        case messages_1.JournalMessageType.ENCODE_FUNCTION_CALL_EXECUTION_STATE_INITIALIZE: {
            executionEventListener.encodeFunctionCallExecutionStateInitialize({
                type: execution_events_1.ExecutionEventType.ENCODE_FUNCTION_CALL_EXECUTION_STATE_INITIALIZE,
                futureId: message.futureId,
                result: {
                    type: execution_events_1.ExecutionEventResultType.SUCCESS,
                    result: message.result,
                },
            });
            break;
        }
        case messages_1.JournalMessageType.SEND_DATA_EXECUTION_STATE_INITIALIZE: {
            executionEventListener.sendDataExecutionStateInitialize({
                type: execution_events_1.ExecutionEventType.SEND_DATA_EXECUTION_STATE_INITIALIZE,
                futureId: message.futureId,
            });
            break;
        }
        case messages_1.JournalMessageType.SEND_DATA_EXECUTION_STATE_COMPLETE: {
            executionEventListener.sendDataExecutionStateComplete({
                type: execution_events_1.ExecutionEventType.SEND_DATA_EXECUTION_STATE_COMPLETE,
                futureId: message.futureId,
                result: convertExecutionResultToEventResult(message.result),
            });
            break;
        }
        case messages_1.JournalMessageType.CONTRACT_AT_EXECUTION_STATE_INITIALIZE: {
            executionEventListener.contractAtExecutionStateInitialize({
                type: execution_events_1.ExecutionEventType.CONTRACT_AT_EXECUTION_STATE_INITIALIZE,
                futureId: message.futureId,
            });
            break;
        }
        case messages_1.JournalMessageType.READ_EVENT_ARGUMENT_EXECUTION_STATE_INITIALIZE: {
            executionEventListener.readEventArgumentExecutionStateInitialize({
                type: execution_events_1.ExecutionEventType.READ_EVENT_ARGUMENT_EXECUTION_STATE_INITIALIZE,
                futureId: message.futureId,
                result: {
                    type: execution_events_1.ExecutionEventResultType.SUCCESS,
                    result: solidityParamToString(message.result),
                },
            });
            break;
        }
        case messages_1.JournalMessageType.NETWORK_INTERACTION_REQUEST: {
            executionEventListener.networkInteractionRequest({
                type: execution_events_1.ExecutionEventType.NETWORK_INTERACTION_REQUEST,
                networkInteractionType: message.networkInteraction.type ===
                    network_interaction_1.NetworkInteractionType.ONCHAIN_INTERACTION
                    ? execution_events_1.ExecutionEventNetworkInteractionType.ONCHAIN_INTERACTION
                    : execution_events_1.ExecutionEventNetworkInteractionType.STATIC_CALL,
                futureId: message.futureId,
            });
            break;
        }
        case messages_1.JournalMessageType.TRANSACTION_SEND: {
            executionEventListener.transactionSend({
                type: execution_events_1.ExecutionEventType.TRANSACTION_SEND,
                futureId: message.futureId,
                hash: message.transaction.hash,
            });
            break;
        }
        case messages_1.JournalMessageType.TRANSACTION_CONFIRM: {
            executionEventListener.transactionConfirm({
                type: execution_events_1.ExecutionEventType.TRANSACTION_CONFIRM,
                futureId: message.futureId,
                hash: message.hash,
            });
            break;
        }
        case messages_1.JournalMessageType.STATIC_CALL_COMPLETE: {
            executionEventListener.staticCallComplete({
                type: execution_events_1.ExecutionEventType.STATIC_CALL_COMPLETE,
                futureId: message.futureId,
            });
            break;
        }
        case messages_1.JournalMessageType.ONCHAIN_INTERACTION_BUMP_FEES: {
            executionEventListener.onchainInteractionBumpFees({
                type: execution_events_1.ExecutionEventType.ONCHAIN_INTERACTION_BUMP_FEES,
                futureId: message.futureId,
            });
            break;
        }
        case messages_1.JournalMessageType.ONCHAIN_INTERACTION_DROPPED: {
            executionEventListener.onchainInteractionDropped({
                type: execution_events_1.ExecutionEventType.ONCHAIN_INTERACTION_DROPPED,
                futureId: message.futureId,
            });
            break;
        }
        case messages_1.JournalMessageType.ONCHAIN_INTERACTION_REPLACED_BY_USER: {
            executionEventListener.onchainInteractionReplacedByUser({
                type: execution_events_1.ExecutionEventType.ONCHAIN_INTERACTION_REPLACED_BY_USER,
                futureId: message.futureId,
            });
            break;
        }
        case messages_1.JournalMessageType.ONCHAIN_INTERACTION_TIMEOUT: {
            executionEventListener.onchainInteractionTimeout({
                type: execution_events_1.ExecutionEventType.ONCHAIN_INTERACTION_TIMEOUT,
                futureId: message.futureId,
            });
            break;
        }
    }
}
exports.emitExecutionEvent = emitExecutionEvent;
function convertExecutionResultToEventResult(result) {
    switch (result.type) {
        case execution_result_1.ExecutionResultType.SUCCESS: {
            return {
                type: execution_events_1.ExecutionEventResultType.SUCCESS,
                result: "address" in result ? result.address : undefined,
            };
        }
        case execution_result_1.ExecutionResultType.STATIC_CALL_ERROR:
        case execution_result_1.ExecutionResultType.SIMULATION_ERROR: {
            return {
                type: execution_events_1.ExecutionEventResultType.ERROR,
                error: (0, failedEvmExecutionResultToErrorDescription_1.failedEvmExecutionResultToErrorDescription)(result.error),
            };
        }
        case execution_result_1.ExecutionResultType.STRATEGY_ERROR:
        case execution_result_1.ExecutionResultType.STRATEGY_SIMULATION_ERROR: {
            return {
                type: execution_events_1.ExecutionEventResultType.ERROR,
                error: result.error,
            };
        }
        case execution_result_1.ExecutionResultType.REVERTED_TRANSACTION: {
            return {
                type: execution_events_1.ExecutionEventResultType.ERROR,
                error: "Transaction reverted",
            };
        }
        case execution_result_1.ExecutionResultType.STRATEGY_HELD: {
            return {
                type: execution_events_1.ExecutionEventResultType.HELD,
                heldId: result.heldId,
                reason: result.reason,
            };
        }
    }
}
function convertStaticCallResultToExecutionEventResult(result) {
    switch (result.type) {
        case execution_result_1.ExecutionResultType.SUCCESS: {
            return {
                type: execution_events_1.ExecutionEventResultType.SUCCESS,
            };
        }
        case execution_result_1.ExecutionResultType.STATIC_CALL_ERROR: {
            return {
                type: execution_events_1.ExecutionEventResultType.ERROR,
                error: (0, failedEvmExecutionResultToErrorDescription_1.failedEvmExecutionResultToErrorDescription)(result.error),
            };
        }
        case execution_result_1.ExecutionResultType.STRATEGY_ERROR: {
            return {
                type: execution_events_1.ExecutionEventResultType.ERROR,
                error: result.error,
            };
        }
        case execution_result_1.ExecutionResultType.STRATEGY_HELD: {
            return {
                type: execution_events_1.ExecutionEventResultType.HELD,
                heldId: result.heldId,
                reason: result.reason,
            };
        }
    }
}
function solidityParamToString(param) {
    if (typeof param === "object") {
        return JSON.stringify(param);
    }
    if (typeof param === "string") {
        return param;
    }
    return param.toString();
}
//# sourceMappingURL=emitExecutionEvent.js.map