import { BatchInitializeEvent, BeginNextBatchEvent, CallExecutionStateCompleteEvent, CallExecutionStateInitializeEvent, ContractAtExecutionStateInitializeEvent, DeploymentCompleteEvent, DeploymentExecutionStateCompleteEvent, DeploymentExecutionStateInitializeEvent, DeploymentInitializeEvent, DeploymentStartEvent, EncodeFunctionCallExecutionStateInitializeEvent, ExecutionEventListener, NetworkInteractionRequestEvent, OnchainInteractionBumpFeesEvent, OnchainInteractionDroppedEvent, OnchainInteractionReplacedByUserEvent, OnchainInteractionTimeoutEvent, ReadEventArgExecutionStateInitializeEvent, ReconciliationWarningsEvent, RunStartEvent, SendDataExecutionStateCompleteEvent, SendDataExecutionStateInitializeEvent, SetModuleIdEvent, SetStrategyEvent, StaticCallCompleteEvent, StaticCallExecutionStateCompleteEvent, StaticCallExecutionStateInitializeEvent, TransactionConfirmEvent, TransactionSendEvent, WipeApplyEvent } from "@nomicfoundation/ignition-core";
export declare class VerboseEventHandler implements ExecutionEventListener {
    deploymentInitialize(event: DeploymentInitializeEvent): void;
    wipeApply(event: WipeApplyEvent): void;
    deploymentExecutionStateInitialize(event: DeploymentExecutionStateInitializeEvent): void;
    deploymentExecutionStateComplete(event: DeploymentExecutionStateCompleteEvent): void;
    callExecutionStateInitialize(event: CallExecutionStateInitializeEvent): void;
    callExecutionStateComplete(event: CallExecutionStateCompleteEvent): void;
    staticCallExecutionStateInitialize(event: StaticCallExecutionStateInitializeEvent): void;
    staticCallExecutionStateComplete(event: StaticCallExecutionStateCompleteEvent): void;
    sendDataExecutionStateInitialize(event: SendDataExecutionStateInitializeEvent): void;
    sendDataExecutionStateComplete(event: SendDataExecutionStateCompleteEvent): void;
    contractAtExecutionStateInitialize(event: ContractAtExecutionStateInitializeEvent): void;
    readEventArgumentExecutionStateInitialize(event: ReadEventArgExecutionStateInitializeEvent): void;
    encodeFunctionCallExecutionStateInitialize(event: EncodeFunctionCallExecutionStateInitializeEvent): void;
    networkInteractionRequest(event: NetworkInteractionRequestEvent): void;
    transactionSend(event: TransactionSendEvent): void;
    transactionConfirm(event: TransactionConfirmEvent): void;
    staticCallComplete(event: StaticCallCompleteEvent): void;
    onchainInteractionBumpFees(event: OnchainInteractionBumpFeesEvent): void;
    onchainInteractionDropped(event: OnchainInteractionDroppedEvent): void;
    onchainInteractionReplacedByUser(event: OnchainInteractionReplacedByUserEvent): void;
    onchainInteractionTimeout(event: OnchainInteractionTimeoutEvent): void;
    batchInitialize(event: BatchInitializeEvent): void;
    deploymentStart(_event: DeploymentStartEvent): void;
    beginNextBatch(_event: BeginNextBatchEvent): void;
    deploymentComplete(_event: DeploymentCompleteEvent): void;
    reconciliationWarnings(event: ReconciliationWarningsEvent): void;
    setModuleId(event: SetModuleIdEvent): void;
    setStrategy(event: SetStrategyEvent): void;
    runStart(_event: RunStartEvent): void;
}
//# sourceMappingURL=verbose-event-handler.d.ts.map