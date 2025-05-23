"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.VerboseEventHandler = void 0;
const ignition_core_1 = require("@nomicfoundation/ignition-core");
class VerboseEventHandler {
    deploymentInitialize(event) {
        console.log(`Deployment initialized for chainId: ${event.chainId}`);
    }
    wipeApply(event) {
        console.log(`Removing the execution of future ${event.futureId}`);
    }
    deploymentExecutionStateInitialize(event) {
        console.log(`Starting to execute the deployment future ${event.futureId}`);
    }
    deploymentExecutionStateComplete(event) {
        switch (event.result.type) {
            case ignition_core_1.ExecutionEventResultType.SUCCESS: {
                return console.log(`Successfully completed the execution of deployment future ${event.futureId} with address ${event.result.result ?? "undefined"}`);
            }
            case ignition_core_1.ExecutionEventResultType.ERROR: {
                return console.log(`Execution of future ${event.futureId} failed with reason: ${event.result.error}`);
            }
            case ignition_core_1.ExecutionEventResultType.HELD: {
                return console.log(`Execution of future ${event.futureId}/${event.result.heldId} held with reason: ${event.result.reason}`);
            }
        }
    }
    callExecutionStateInitialize(event) {
        console.log(`Starting to execute the call future ${event.futureId}`);
    }
    callExecutionStateComplete(event) {
        switch (event.result.type) {
            case ignition_core_1.ExecutionEventResultType.SUCCESS: {
                return console.log(`Successfully completed the execution of call future ${event.futureId}`);
            }
            case ignition_core_1.ExecutionEventResultType.ERROR: {
                return console.log(`Execution of call future ${event.futureId} failed with reason: ${event.result.error}`);
            }
            case ignition_core_1.ExecutionEventResultType.HELD: {
                return console.log(`Execution of call future ${event.futureId}/${event.result.heldId} held with reason: ${event.result.reason}`);
            }
        }
    }
    staticCallExecutionStateInitialize(event) {
        console.log(`Starting to execute the static call future ${event.futureId}`);
    }
    staticCallExecutionStateComplete(event) {
        switch (event.result.type) {
            case ignition_core_1.ExecutionEventResultType.SUCCESS: {
                return console.log(`Successfully completed the execution of static call future ${event.futureId} with result ${event.result.result ?? "undefined"}`);
            }
            case ignition_core_1.ExecutionEventResultType.ERROR: {
                return console.log(`Execution of static call future ${event.futureId} failed with reason: ${event.result.error}`);
            }
            case ignition_core_1.ExecutionEventResultType.HELD: {
                return console.log(`Execution of static call future ${event.futureId}/${event.result.heldId} held with reason: ${event.result.reason}`);
            }
        }
    }
    sendDataExecutionStateInitialize(event) {
        console.log(`Started to execute the send data future ${event.futureId}`);
    }
    sendDataExecutionStateComplete(event) {
        switch (event.result.type) {
            case ignition_core_1.ExecutionEventResultType.SUCCESS: {
                return console.log(`Successfully completed the execution of send data future ${event.futureId} in tx ${event.result.result ?? "undefined"}`);
            }
            case ignition_core_1.ExecutionEventResultType.ERROR: {
                return console.log(`Execution of future ${event.futureId} failed with reason: ${event.result.error}`);
            }
            case ignition_core_1.ExecutionEventResultType.HELD: {
                return console.log(`Execution of send future ${event.futureId}/${event.result.heldId} held with reason: ${event.result.reason}`);
            }
        }
    }
    contractAtExecutionStateInitialize(event) {
        console.log(`Executed contract at future ${event.futureId}`);
    }
    readEventArgumentExecutionStateInitialize(event) {
        console.log(`Executed read event argument future ${event.futureId} with result ${event.result.result ?? "undefined"}`);
    }
    encodeFunctionCallExecutionStateInitialize(event) {
        console.log(`Executed encode function call future ${event.futureId} with result ${event.result.result ?? "undefined"}`);
    }
    networkInteractionRequest(event) {
        if (event.networkInteractionType ===
            ignition_core_1.ExecutionEventNetworkInteractionType.ONCHAIN_INTERACTION) {
            console.log(`New onchain interaction requested for future ${event.futureId}`);
        }
        else {
            console.log(`New static call requested for future ${event.futureId}`);
        }
    }
    transactionSend(event) {
        console.log(`Transaction ${event.hash} sent for onchain interaction of future ${event.futureId}`);
    }
    transactionConfirm(event) {
        console.log(`Transaction ${event.hash} confirmed`);
    }
    staticCallComplete(event) {
        console.log(`Static call completed for future ${event.futureId}`);
    }
    onchainInteractionBumpFees(event) {
        console.log(`A transaction with higher fees will be sent for onchain interaction of future ${event.futureId}`);
    }
    onchainInteractionDropped(event) {
        console.log(`Transactions for onchain interaction of future ${event.futureId} has been dropped and will be resent`);
    }
    onchainInteractionReplacedByUser(event) {
        console.log(`Transactions for onchain interaction of future ${event.futureId} has been replaced by the user and the onchain interaction exection will start again`);
    }
    onchainInteractionTimeout(event) {
        console.log(`Onchain interaction of future ${event.futureId} failed due to being resent too many times and not having confirmed`);
    }
    batchInitialize(event) {
        console.log(`Starting execution for batches: ${JSON.stringify(event.batches)}`);
    }
    deploymentStart(_event) {
        console.log(`Starting execution for new deployment`);
    }
    beginNextBatch(_event) {
        console.log(`Starting execution for next batch`);
    }
    deploymentComplete(_event) {
        console.log(`Deployment complete`);
    }
    reconciliationWarnings(event) {
        console.log(`Deployment produced reconciliation warnings:\n${event.warnings.join("  -")}`);
    }
    setModuleId(event) {
        console.log(`Starting validation for module: ${event.moduleName}`);
    }
    setStrategy(event) {
        console.log(`Starting execution with strategy: ${event.strategy}`);
    }
    runStart(_event) {
        console.log("Execution run starting");
    }
}
exports.VerboseEventHandler = VerboseEventHandler;
//# sourceMappingURL=verbose-event-handler.js.map