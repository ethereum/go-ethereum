"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.logJournalableMessage = void 0;
const execution_result_1 = require("../../execution/types/execution-result");
const messages_1 = require("../../execution/types/messages");
const network_interaction_1 = require("../../execution/types/network-interaction");
const formatters_1 = require("../../formatters");
function logJournalableMessage(message) {
    switch (message.type) {
        case messages_1.JournalMessageType.DEPLOYMENT_INITIALIZE:
            console.log(`Deployment started`);
            break;
        case messages_1.JournalMessageType.WIPE_APPLY: {
            console.log(`Removing the execution of future ${message.futureId} from the journal`);
        }
        case messages_1.JournalMessageType.DEPLOYMENT_EXECUTION_STATE_INITIALIZE:
            console.log(`Starting to execute the deployment future ${message.futureId}`);
            break;
        case messages_1.JournalMessageType.CALL_EXECUTION_STATE_INITIALIZE:
            console.log(`Starting to execute the call future ${message.futureId}`);
            break;
        case messages_1.JournalMessageType.STATIC_CALL_EXECUTION_STATE_INITIALIZE:
            console.log(`Starting to execute the static call future ${message.futureId}`);
            break;
        case messages_1.JournalMessageType.SEND_DATA_EXECUTION_STATE_INITIALIZE:
            console.log(`Started to execute the send data future ${message.futureId}`);
            break;
        case messages_1.JournalMessageType.STATIC_CALL_EXECUTION_STATE_COMPLETE:
            if (message.result.type === execution_result_1.ExecutionResultType.SUCCESS) {
                console.log(`Successfully completed the execution of static call future ${message.futureId} with result ${(0, formatters_1.formatSolidityParameter)(message.result.value)}`);
            }
            else {
                console.log(`Execution of future ${message.futureId} failed`);
            }
            break;
        case messages_1.JournalMessageType.DEPLOYMENT_EXECUTION_STATE_COMPLETE:
            if (message.result.type === execution_result_1.ExecutionResultType.SUCCESS) {
                console.log(`Successfully completed the execution of deployment future ${message.futureId} with result ${message.result.address}`);
            }
            else {
                console.log(`Execution of future ${message.futureId} failed`);
            }
            break;
        case messages_1.JournalMessageType.CALL_EXECUTION_STATE_COMPLETE:
            if (message.result.type === execution_result_1.ExecutionResultType.SUCCESS) {
                console.log(`Successfully completed the execution of call future ${message.futureId}`);
            }
            else {
                console.log(`Execution of future ${message.futureId} failed`);
            }
            break;
        case messages_1.JournalMessageType.SEND_DATA_EXECUTION_STATE_COMPLETE:
            if (message.result.type === execution_result_1.ExecutionResultType.SUCCESS) {
                console.log(`Successfully completed the execution of send data future ${message.futureId}`);
            }
            else {
                console.log(`Execution of future ${message.futureId} failed`);
            }
            break;
        case messages_1.JournalMessageType.CONTRACT_AT_EXECUTION_STATE_INITIALIZE:
            console.log(`Executed contract at future ${message.futureId}`);
            break;
        case messages_1.JournalMessageType.READ_EVENT_ARGUMENT_EXECUTION_STATE_INITIALIZE:
            console.log(`Executed read event argument future ${message.futureId} with result ${(0, formatters_1.formatSolidityParameter)(message.result)}`);
            break;
        case messages_1.JournalMessageType.ENCODE_FUNCTION_CALL_EXECUTION_STATE_INITIALIZE:
            console.log(`Executed encode function call future ${message.futureId} with result ${message.result}`);
            break;
        case messages_1.JournalMessageType.NETWORK_INTERACTION_REQUEST:
            if (message.networkInteraction.type ===
                network_interaction_1.NetworkInteractionType.ONCHAIN_INTERACTION) {
                console.log(`New onchain interaction ${message.networkInteraction.id} requested for future ${message.futureId}`);
            }
            else {
                console.log(`New static call ${message.networkInteraction.id} requested for future ${message.futureId}`);
            }
            break;
        case messages_1.JournalMessageType.TRANSACTION_SEND:
            console.log(`Transaction ${message.transaction.hash} sent for onchain interaction ${message.networkInteractionId} of future ${message.futureId}`);
            break;
        case messages_1.JournalMessageType.TRANSACTION_CONFIRM:
            console.log(`Transaction ${message.hash} confirmed`);
            break;
        case messages_1.JournalMessageType.STATIC_CALL_COMPLETE:
            console.log(`Static call ${message.networkInteractionId} completed for future ${message.futureId}`);
            break;
        case messages_1.JournalMessageType.ONCHAIN_INTERACTION_BUMP_FEES:
            console.log(`A transaction with higher fees will be sent for onchain interaction ${message.networkInteractionId} of future ${message.futureId}`);
            break;
        case messages_1.JournalMessageType.ONCHAIN_INTERACTION_DROPPED:
            console.log(`Transactions for onchain interaction ${message.networkInteractionId} of future ${message.futureId} has been dropped and will be resent`);
            break;
        case messages_1.JournalMessageType.ONCHAIN_INTERACTION_REPLACED_BY_USER:
            console.log(`Transactions for onchain interaction ${message.networkInteractionId} of future ${message.futureId} has been replaced by the user and the onchain interaction exection will start again`);
            break;
        case messages_1.JournalMessageType.ONCHAIN_INTERACTION_TIMEOUT:
            console.log(`Onchain interaction ${message.networkInteractionId} of future ${message.futureId} failed due to being resent too many times and not having confirmed`);
            break;
    }
}
exports.logJournalableMessage = logJournalableMessage;
//# sourceMappingURL=log.js.map