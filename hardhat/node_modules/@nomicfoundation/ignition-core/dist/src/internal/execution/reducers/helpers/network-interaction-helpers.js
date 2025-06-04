"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.onchainInteractionTimedOut = exports.resetOnchainInteractionReplacedByUser = exports.resendDroppedOnchainInteraction = exports.bumpOnchainInteractionFees = exports.completeStaticCall = exports.confirmTransaction = exports.applyNonceToOnchainInteraction = exports.appendTransactionToOnchainInteraction = exports.appendNetworkInteraction = void 0;
const immer_1 = require("immer");
const assertions_1 = require("../../../utils/assertions");
const find_onchain_interaction_by_1 = require("../../../views/execution-state/find-onchain-interaction-by");
const find_static_call_by_1 = require("../../../views/execution-state/find-static-call-by");
const find_transaction_by_1 = require("../../../views/execution-state/find-transaction-by");
const execution_state_1 = require("../../types/execution-state");
const network_interaction_1 = require("../../types/network-interaction");
/**
 * Add a new network interaction to the execution state.
 *
 * @param state - the execution state that will be added to
 * @param action - the request message that contains the network interaction
 * @returns a copy of the execution state with the addition network interaction
 */
function appendNetworkInteraction(state, action) {
    return (0, immer_1.produce)(state, (draft) => {
        if (draft.type === execution_state_1.ExecutionStateType.STATIC_CALL_EXECUTION_STATE) {
            (0, assertions_1.assertIgnitionInvariant)(action.networkInteraction.type === network_interaction_1.NetworkInteractionType.STATIC_CALL, `Static call execution states like ${draft.id} cannot have onchain interactions`);
            draft.networkInteractions.push(action.networkInteraction);
            return;
        }
        draft.networkInteractions.push(action.networkInteraction.type ===
            network_interaction_1.NetworkInteractionType.ONCHAIN_INTERACTION
            ? {
                ...action.networkInteraction,
                transactions: [],
                nonce: undefined,
                shouldBeResent: false,
            }
            : action.networkInteraction);
    });
}
exports.appendNetworkInteraction = appendNetworkInteraction;
/**
 * Add a transaction to an onchain interaction within an execution state.
 *
 * If the onchain interaction didn't have a nonce yet, it will be set to
 * the nonce of the transaction.
 *
 * This function also sets the onchain interaction's `shouldBeResent` flag
 * to `false`.
 *
 * @param state - the execution state that will be added to
 * @param action - the request message that contains the transaction
 * @returns a copy of the execution state with the additional transaction
 */
function appendTransactionToOnchainInteraction(state, action) {
    return (0, immer_1.produce)(state, (draft) => {
        const onchainInteraction = (0, find_onchain_interaction_by_1.findOnchainInteractionBy)(draft, action.networkInteractionId);
        if (onchainInteraction.nonce === undefined) {
            onchainInteraction.nonce = action.nonce;
        }
        else {
            (0, assertions_1.assertIgnitionInvariant)(onchainInteraction.nonce === action.nonce, `New transaction sent for ${state.id}/${onchainInteraction.id} with nonce ${action.nonce} but expected ${onchainInteraction.nonce}`);
        }
        onchainInteraction.shouldBeResent = false;
        onchainInteraction.transactions.push(action.transaction);
    });
}
exports.appendTransactionToOnchainInteraction = appendTransactionToOnchainInteraction;
/**
 * Sets the nonce of the onchain interaction within an execution state.
 *
 * @param state - the execution state that will be added to
 * @param action - the request message that contains the transaction prepare message
 * @returns a copy of the execution state with the nonce set
 */
function applyNonceToOnchainInteraction(state, action) {
    return (0, immer_1.produce)(state, (draft) => {
        const onchainInteraction = (0, find_onchain_interaction_by_1.findOnchainInteractionBy)(draft, action.networkInteractionId);
        if (onchainInteraction.nonce === undefined) {
            onchainInteraction.nonce = action.nonce;
        }
        else {
            (0, assertions_1.assertIgnitionInvariant)(onchainInteraction.nonce === action.nonce, `New transaction sent for ${state.id}/${onchainInteraction.id} with nonce ${action.nonce} but expected ${onchainInteraction.nonce}`);
        }
    });
}
exports.applyNonceToOnchainInteraction = applyNonceToOnchainInteraction;
/**
 * Confirm a transaction for an onchain interaction within an execution state.
 *
 * @param state - the execution state that will be updated within
 * @param action - the request message that contains the transaction details
 * @returns a copy of the execution state with transaction confirmed
 */
function confirmTransaction(state, action) {
    return (0, immer_1.produce)(state, (draft) => {
        const onchainInteraction = (0, find_onchain_interaction_by_1.findOnchainInteractionBy)(draft, action.networkInteractionId);
        const transaction = (0, find_transaction_by_1.findTransactionBy)(draft, action.networkInteractionId, action.hash);
        transaction.receipt = action.receipt;
        // we intentionally clear other transactions
        onchainInteraction.transactions = [transaction];
    });
}
exports.confirmTransaction = confirmTransaction;
/**
 * Complete the static call network interaction within an execution state.
 *
 * @param state - the execution state that will be updated
 * @param action - the request message that contains the static call result details
 * @returns a copy of the execution state with the static call confirmed
 */
function completeStaticCall(state, action) {
    return (0, immer_1.produce)(state, (draft) => {
        const onchainInteraction = (0, find_static_call_by_1.findStaticCallBy)(draft, action.networkInteractionId);
        onchainInteraction.result = action.result;
    });
}
exports.completeStaticCall = completeStaticCall;
/**
 * Sets the state `shouldBeResent` of an OnchainInteraction to `true`
 * so that a new transaction with higher fees is sent.
 *
 * @param state - the execution state that will be updated within
 * @param action - the request message that contains the onchain interaction details
 * @returns a copy of the execution state with transaction confirmed
 */
function bumpOnchainInteractionFees(state, action) {
    return (0, immer_1.produce)(state, (draft) => {
        const onchainInteraction = (0, find_onchain_interaction_by_1.findOnchainInteractionBy)(draft, action.networkInteractionId);
        onchainInteraction.shouldBeResent = true;
    });
}
exports.bumpOnchainInteractionFees = bumpOnchainInteractionFees;
/**
 * Sets the state `shouldBeResent` of a dropped OnchainInteraction to `true`
 * so that a new transaction is sent.
 *
 * @param state - the execution state that will be updated within
 * @param action - the request message that contains the onchain interaction details
 * @returns a copy of the execution state with transaction confirmed
 */
function resendDroppedOnchainInteraction(state, action) {
    return (0, immer_1.produce)(state, (draft) => {
        const onchainInteraction = (0, find_onchain_interaction_by_1.findOnchainInteractionBy)(draft, action.networkInteractionId);
        onchainInteraction.shouldBeResent = true;
    });
}
exports.resendDroppedOnchainInteraction = resendDroppedOnchainInteraction;
/**
 * Resets an OnchainInteraction's nonce, transactions and shouldBeResent
 * due to the user having invalidated the nonce that has been used.
 *
 * @param state - the execution state that will be updated within
 * @param action - the request message that contains the onchain interaction details
 * @returns a copy of the execution state with transaction confirmed
 */
function resetOnchainInteractionReplacedByUser(state, action) {
    return (0, immer_1.produce)(state, (draft) => {
        const onchainInteraction = (0, find_onchain_interaction_by_1.findOnchainInteractionBy)(draft, action.networkInteractionId);
        onchainInteraction.transactions = [];
        onchainInteraction.nonce = undefined;
        onchainInteraction.shouldBeResent = false;
    });
}
exports.resetOnchainInteractionReplacedByUser = resetOnchainInteractionReplacedByUser;
/**
 * Sets an execution state to `TIMEOUT` due to an onchain interaction
 * not being confirmed within the timeout period.
 */
function onchainInteractionTimedOut(state, _action) {
    return (0, immer_1.produce)(state, (draft) => {
        draft.status = execution_state_1.ExecutionStatus.TIMEOUT;
    });
}
exports.onchainInteractionTimedOut = onchainInteractionTimedOut;
//# sourceMappingURL=network-interaction-helpers.js.map