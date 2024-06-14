"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.emitWithArgs = exports.supportEmit = exports.EMIT_CALLED = void 0;
const chai_1 = require("chai");
const util_1 = __importDefault(require("util"));
const utils_1 = require("../utils");
const constants_1 = require("./constants");
const errors_1 = require("./errors");
const utils_2 = require("./utils");
exports.EMIT_CALLED = "emitAssertionCalled";
async function waitForPendingTransaction(tx, provider) {
    let hash;
    if (tx instanceof Promise) {
        ({ hash } = await tx);
    }
    else if (typeof tx === "string") {
        hash = tx;
    }
    else {
        ({ hash } = tx);
    }
    if (hash === null) {
        throw new Error(`${JSON.stringify(tx)} is not a valid transaction`);
    }
    return provider.getTransactionReceipt(hash);
}
function supportEmit(Assertion, chaiUtils) {
    Assertion.addMethod(constants_1.EMIT_MATCHER, function (contract, eventName, ...args) {
        // capture negated flag before async code executes; see buildAssert's jsdoc
        const negated = this.__flags.negate;
        const tx = this._obj;
        (0, utils_2.preventAsyncMatcherChaining)(this, constants_1.EMIT_MATCHER, chaiUtils, true);
        const promise = this.then === undefined ? Promise.resolve() : this;
        const onSuccess = (receipt) => {
            // abort if the assertion chain was aborted, for example because
            // a `.not` was combined with a `.withArgs`
            if (chaiUtils.flag(this, constants_1.ASSERTION_ABORTED) === true) {
                return;
            }
            const assert = (0, utils_1.buildAssert)(negated, onSuccess);
            let eventFragment = null;
            try {
                eventFragment = contract.interface.getEvent(eventName);
            }
            catch (e) {
                // ignore error
            }
            if (eventFragment === null) {
                throw new chai_1.AssertionError(`Event "${eventName}" doesn't exist in the contract`);
            }
            const topic = eventFragment.topicHash;
            const contractAddress = contract.target;
            if (typeof contractAddress !== "string") {
                throw new errors_1.HardhatChaiMatchersAssertionError(`The contract target should be a string`);
            }
            if (args.length > 0) {
                throw new Error("`.emit` expects only two arguments: the contract and the event name. Arguments should be asserted with the `.withArgs` helper.");
            }
            this.logs = receipt.logs
                .filter((log) => log.topics.includes(topic))
                .filter((log) => log.address.toLowerCase() === contractAddress.toLowerCase());
            assert(this.logs.length > 0, `Expected event "${eventName}" to be emitted, but it wasn't`, `Expected event "${eventName}" NOT to be emitted, but it was`);
            chaiUtils.flag(this, "eventName", eventName);
            chaiUtils.flag(this, "contract", contract);
        };
        const derivedPromise = promise.then(() => {
            // abort if the assertion chain was aborted, for example because
            // a `.not` was combined with a `.withArgs`
            if (chaiUtils.flag(this, constants_1.ASSERTION_ABORTED) === true) {
                return;
            }
            if (contract.runner === null || contract.runner.provider === null) {
                throw new errors_1.HardhatChaiMatchersAssertionError("contract.runner.provider shouldn't be null");
            }
            return waitForPendingTransaction(tx, contract.runner.provider).then((receipt) => {
                (0, utils_2.assertIsNotNull)(receipt, "receipt");
                return onSuccess(receipt);
            });
        });
        chaiUtils.flag(this, exports.EMIT_CALLED, true);
        this.then = derivedPromise.then.bind(derivedPromise);
        this.catch = derivedPromise.catch.bind(derivedPromise);
        this.promise = derivedPromise;
        return this;
    });
}
exports.supportEmit = supportEmit;
async function emitWithArgs(context, Assertion, chaiUtils, expectedArgs, ssfi) {
    const negated = false; // .withArgs cannot be negated
    const assert = (0, utils_1.buildAssert)(negated, ssfi);
    tryAssertArgsArraysEqual(context, Assertion, chaiUtils, expectedArgs, context.logs, assert, ssfi);
}
exports.emitWithArgs = emitWithArgs;
const tryAssertArgsArraysEqual = (context, Assertion, chaiUtils, expectedArgs, logs, assert, ssfi) => {
    const eventName = chaiUtils.flag(context, "eventName");
    if (logs.length === 1) {
        const parsedLog = chaiUtils.flag(context, "contract").interface.parseLog(logs[0]);
        (0, utils_2.assertIsNotNull)(parsedLog, "parsedLog");
        return (0, utils_2.assertArgsArraysEqual)(Assertion, expectedArgs, parsedLog.args, `"${eventName}" event`, "event", assert, ssfi);
    }
    for (const index in logs) {
        if (index === undefined) {
            break;
        }
        else {
            try {
                const parsedLog = chaiUtils.flag(context, "contract").interface.parseLog(logs[index]);
                (0, utils_2.assertIsNotNull)(parsedLog, "parsedLog");
                (0, utils_2.assertArgsArraysEqual)(Assertion, expectedArgs, parsedLog.args, `"${eventName}" event`, "event", assert, ssfi);
                return;
            }
            catch { }
        }
    }
    assert(false, `The specified arguments (${util_1.default.inspect(expectedArgs)}) were not included in any of the ${context.logs.length} emitted "${eventName}" events`);
};
//# sourceMappingURL=emit.js.map