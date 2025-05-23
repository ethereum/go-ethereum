"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.selectHardfork = exports.hardforkGte = exports.getHardforkName = exports.HardforkName = void 0;
const constants_1 = require("../constants");
const errors_1 = require("../core/errors");
const errors_2 = require("../core/providers/errors");
/* eslint-disable @nomicfoundation/hardhat-internal-rules/only-hardhat-error */
var HardforkName;
(function (HardforkName) {
    HardforkName["FRONTIER"] = "chainstart";
    HardforkName["HOMESTEAD"] = "homestead";
    HardforkName["DAO"] = "dao";
    HardforkName["TANGERINE_WHISTLE"] = "tangerineWhistle";
    HardforkName["SPURIOUS_DRAGON"] = "spuriousDragon";
    HardforkName["BYZANTIUM"] = "byzantium";
    HardforkName["CONSTANTINOPLE"] = "constantinople";
    HardforkName["PETERSBURG"] = "petersburg";
    HardforkName["ISTANBUL"] = "istanbul";
    HardforkName["MUIR_GLACIER"] = "muirGlacier";
    HardforkName["BERLIN"] = "berlin";
    HardforkName["LONDON"] = "london";
    HardforkName["ARROW_GLACIER"] = "arrowGlacier";
    HardforkName["GRAY_GLACIER"] = "grayGlacier";
    HardforkName["MERGE"] = "merge";
    HardforkName["SHANGHAI"] = "shanghai";
    HardforkName["CANCUN"] = "cancun";
    HardforkName["PRAGUE"] = "prague";
})(HardforkName = exports.HardforkName || (exports.HardforkName = {}));
const HARDFORKS_ORDER = [
    HardforkName.FRONTIER,
    HardforkName.HOMESTEAD,
    HardforkName.DAO,
    HardforkName.TANGERINE_WHISTLE,
    HardforkName.SPURIOUS_DRAGON,
    HardforkName.BYZANTIUM,
    HardforkName.CONSTANTINOPLE,
    HardforkName.PETERSBURG,
    HardforkName.ISTANBUL,
    HardforkName.MUIR_GLACIER,
    HardforkName.BERLIN,
    HardforkName.LONDON,
    HardforkName.ARROW_GLACIER,
    HardforkName.GRAY_GLACIER,
    HardforkName.MERGE,
    HardforkName.SHANGHAI,
    HardforkName.CANCUN,
    HardforkName.PRAGUE,
];
function getHardforkName(name) {
    const hardforkName = Object.values(HardforkName)[Object.values(HardforkName).indexOf(name)];
    (0, errors_1.assertHardhatInvariant)(hardforkName !== undefined, `Invalid harfork name ${name}`);
    return hardforkName;
}
exports.getHardforkName = getHardforkName;
/**
 * Check if `hardforkA` is greater than or equal to `hardforkB`,
 * that is, if it includes all its changes.
 */
function hardforkGte(hardforkA, hardforkB) {
    // This function should not load any ethereumjs library, as it's used during
    // the Hardhat initialization, and that would make it too slow.
    const indexA = HARDFORKS_ORDER.lastIndexOf(hardforkA);
    const indexB = HARDFORKS_ORDER.lastIndexOf(hardforkB);
    return indexA >= indexB;
}
exports.hardforkGte = hardforkGte;
function selectHardfork(forkBlockNumber, currentHardfork, hardforkActivations, blockNumber) {
    if (forkBlockNumber === undefined || blockNumber > forkBlockNumber) {
        return currentHardfork;
    }
    if (hardforkActivations === undefined || hardforkActivations.size === 0) {
        throw new errors_2.InternalError(`No known hardfork for execution on historical block ${blockNumber.toString()} (relative to fork block number ${forkBlockNumber}). The node was not configured with a hardfork activation history.  See http://hardhat.org/custom-hardfork-history`);
    }
    /** search this._hardforkActivations for the highest block number that
     * isn't higher than blockNumber, and then return that found block number's
     * associated hardfork name. */
    const hardforkHistory = Array.from(hardforkActivations.entries());
    const [hardfork, activationBlock] = hardforkHistory.reduce(([highestHardfork, highestBlock], [thisHardfork, thisBlock]) => thisBlock > highestBlock && thisBlock <= blockNumber
        ? [thisHardfork, thisBlock]
        : [highestHardfork, highestBlock]);
    if (hardfork === undefined || blockNumber < activationBlock) {
        throw new errors_2.InternalError(`Could not find a hardfork to run for block ${blockNumber.toString()}, after having looked for one in the hardfork activation history, which was: ${JSON.stringify(hardforkHistory)}. For more information, see https://hardhat.org/hardhat-network/reference/#config`);
    }
    if (!constants_1.HARDHAT_NETWORK_SUPPORTED_HARDFORKS.includes(hardfork)) {
        throw new errors_2.InternalError(`Tried to run a call or transaction in the context of a block whose hardfork is "${hardfork}", but Hardhat Network only supports the following hardforks: ${constants_1.HARDHAT_NETWORK_SUPPORTED_HARDFORKS.join(", ")}`);
    }
    return hardfork;
}
exports.selectHardfork = selectHardfork;
//# sourceMappingURL=hardforks.js.map