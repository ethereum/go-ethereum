"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.assertArgsArraysEqual = exports.preventAsyncMatcherChaining = exports.assertIsNotNull = void 0;
const constants_1 = require("./constants");
const errors_1 = require("./errors");
function assertIsNotNull(value, valueName) {
    if (value === null) {
        throw new errors_1.HardhatChaiMatchersAssertionError(`${valueName} should not be null`);
    }
}
exports.assertIsNotNull = assertIsNotNull;
function preventAsyncMatcherChaining(context, matcherName, chaiUtils, allowSelfChaining = false) {
    const previousMatcherName = chaiUtils.flag(context, constants_1.PREVIOUS_MATCHER_NAME);
    if (previousMatcherName === undefined) {
        chaiUtils.flag(context, constants_1.PREVIOUS_MATCHER_NAME, matcherName);
        return;
    }
    if (previousMatcherName === matcherName && allowSelfChaining) {
        return;
    }
    throw new errors_1.HardhatChaiMatchersNonChainableMatcherError(matcherName, previousMatcherName);
}
exports.preventAsyncMatcherChaining = preventAsyncMatcherChaining;
function assertArgsArraysEqual(Assertion, expectedArgs, actualArgs, tag, assertionType, assert, ssfi) {
    try {
        innerAssertArgsArraysEqual(Assertion, expectedArgs, actualArgs, assertionType, assert, ssfi);
    }
    catch (err) {
        err.message = `Error in ${tag}: ${err.message}`;
        throw err;
    }
}
exports.assertArgsArraysEqual = assertArgsArraysEqual;
function innerAssertArgsArraysEqual(Assertion, expectedArgs, actualArgs, assertionType, assert, ssfi) {
    assert(actualArgs.length === expectedArgs.length, `Expected arguments array to have length ${expectedArgs.length}, but it has ${actualArgs.length}`);
    for (const [index, expectedArg] of expectedArgs.entries()) {
        try {
            innerAssertArgEqual(Assertion, expectedArg, actualArgs[index], assertionType, assert, ssfi);
        }
        catch (err) {
            const ordinal = require("ordinal");
            err.message = `Error in the ${ordinal(index + 1)} argument assertion: ${err.message}`;
            throw err;
        }
    }
}
function innerAssertArgEqual(Assertion, expectedArg, actualArg, assertionType, assert, ssfi) {
    const ethers = require("ethers");
    if (typeof expectedArg === "function") {
        try {
            if (expectedArg(actualArg) === true)
                return;
        }
        catch (e) {
            assert(false, `The predicate threw when called: ${e.message}`
            // no need for a negated message, since we disallow mixing .not. with
            // .withArgs
            );
        }
        assert(false, `The predicate did not return true`
        // no need for a negated message, since we disallow mixing .not. with
        // .withArgs
        );
    }
    else if (expectedArg instanceof Uint8Array) {
        new Assertion(actualArg, undefined, ssfi, true).equal(ethers.hexlify(expectedArg));
    }
    else if (expectedArg?.length !== undefined &&
        typeof expectedArg !== "string") {
        innerAssertArgsArraysEqual(Assertion, expectedArg, actualArg, assertionType, assert, ssfi);
    }
    else {
        if (actualArg.hash !== undefined && actualArg._isIndexed === true) {
            if (assertionType !== "event") {
                throw new Error("Should not get an indexed event when the assertion type is not event. Please open an issue about this.");
            }
            new Assertion(actualArg.hash, undefined, ssfi, true).to.not.equal(expectedArg, "The actual value was an indexed and hashed value of the event argument. The expected value provided to the assertion should be the actual event argument (the pre-image of the hash). You provided the hash itself. Please supply the actual event argument (the pre-image of the hash) instead.");
            const expectedArgBytes = ethers.isHexString(expectedArg)
                ? ethers.getBytes(expectedArg)
                : ethers.toUtf8Bytes(expectedArg);
            const expectedHash = ethers.keccak256(expectedArgBytes);
            new Assertion(actualArg.hash, undefined, ssfi, true).to.equal(expectedHash, `The actual value was an indexed and hashed value of the event argument. The expected value provided to the assertion was hashed to produce ${expectedHash}. The actual hash and the expected hash ${actualArg.hash} did not match`);
        }
        else {
            new Assertion(actualArg, undefined, ssfi, true).equal(expectedArg);
        }
    }
}
//# sourceMappingURL=utils.js.map