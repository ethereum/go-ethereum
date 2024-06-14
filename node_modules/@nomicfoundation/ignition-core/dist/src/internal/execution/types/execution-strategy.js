"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.SIMULATION_SUCCESS_SIGNAL_TYPE = exports.OnchainInteractionResponseType = void 0;
/**
 * The different types of response that the execution engine can give when
 * asked to perform an onchain interaction.
 */
var OnchainInteractionResponseType;
(function (OnchainInteractionResponseType) {
    OnchainInteractionResponseType["SUCCESSFUL_TRANSACTION"] = "SUCCESSFUL_TRANSACTION";
    OnchainInteractionResponseType["SIMULATION_RESULT"] = "SIMULATION_RESULT";
})(OnchainInteractionResponseType || (exports.OnchainInteractionResponseType = OnchainInteractionResponseType = {}));
/**
 * The type of a SimulationSuccessSignal
 */
exports.SIMULATION_SUCCESS_SIGNAL_TYPE = "SIMULATION_SUCCESS_SIGNAL";
//# sourceMappingURL=execution-strategy.js.map