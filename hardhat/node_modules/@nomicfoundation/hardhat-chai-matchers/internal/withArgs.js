"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.supportWithArgs = exports.anyUint = exports.anyValue = void 0;
const chai_1 = require("chai");
const common_1 = require("hardhat/common");
const constants_1 = require("./constants");
const emit_1 = require("./emit");
const revertedWithCustomError_1 = require("./reverted/revertedWithCustomError");
/**
 * A predicate for use with .withArgs(...), to induce chai to accept any value
 * as a positive match with the argument.
 *
 * Example: expect(contract.emitInt()).to.emit(contract, "Int").withArgs(anyValue)
 */
function anyValue() {
    return true;
}
exports.anyValue = anyValue;
/**
 * A predicate for use with .withArgs(...), to induce chai to accept any
 * unsigned integer as a positive match with the argument.
 *
 * Example: expect(contract.emitUint()).to.emit(contract, "Uint").withArgs(anyUint)
 */
function anyUint(i) {
    if (typeof i === "number") {
        if (i < 0) {
            throw new chai_1.AssertionError(`anyUint expected its argument to be an unsigned integer, but it was negative, with value ${i}`);
        }
        return true;
    }
    else if ((0, common_1.isBigNumber)(i)) {
        const bigInt = (0, common_1.normalizeToBigInt)(i);
        if (bigInt < 0) {
            throw new chai_1.AssertionError(`anyUint expected its argument to be an unsigned integer, but it was negative, with value ${bigInt}`);
        }
        return true;
    }
    throw new chai_1.AssertionError(`anyUint expected its argument to be an integer, but its type was '${typeof i}'`);
}
exports.anyUint = anyUint;
function supportWithArgs(Assertion, chaiUtils) {
    Assertion.addMethod("withArgs", function (...expectedArgs) {
        const { emitCalled } = validateInput.call(this, chaiUtils);
        const { isAddressable } = require("ethers");
        // Resolve arguments to their canonical form:
        // - Addressable → address
        const resolveArgument = (arg) => isAddressable(arg) ? arg.getAddress() : arg;
        const onSuccess = (resolvedExpectedArgs) => {
            if (emitCalled) {
                return (0, emit_1.emitWithArgs)(this, Assertion, chaiUtils, resolvedExpectedArgs, onSuccess);
            }
            else {
                return (0, revertedWithCustomError_1.revertedWithCustomErrorWithArgs)(this, Assertion, chaiUtils, resolvedExpectedArgs, onSuccess);
            }
        };
        const promise = (this.then === undefined ? Promise.resolve() : this)
            .then(() => Promise.all(expectedArgs.map(resolveArgument)))
            .then(onSuccess);
        this.then = promise.then.bind(promise);
        this.catch = promise.catch.bind(promise);
        return this;
    });
}
exports.supportWithArgs = supportWithArgs;
function validateInput(chaiUtils) {
    try {
        if (Boolean(this.__flags.negate)) {
            throw new Error("Do not combine .not. with .withArgs()");
        }
        const emitCalled = chaiUtils.flag(this, emit_1.EMIT_CALLED) === true;
        const revertedWithCustomErrorCalled = chaiUtils.flag(this, revertedWithCustomError_1.REVERTED_WITH_CUSTOM_ERROR_CALLED) === true;
        if (!emitCalled && !revertedWithCustomErrorCalled) {
            throw new Error("withArgs can only be used in combination with a previous .emit or .revertedWithCustomError assertion");
        }
        if (emitCalled && revertedWithCustomErrorCalled) {
            throw new Error("withArgs called with both .emit and .revertedWithCustomError, but these assertions cannot be combined");
        }
        return { emitCalled };
    }
    catch (e) {
        // signal that validation failed to allow the matchers to finish early
        chaiUtils.flag(this, constants_1.ASSERTION_ABORTED, true);
        // discard subject since it could potentially be a rejected promise
        Promise.resolve(this._obj).catch(() => { });
        throw e;
    }
}
//# sourceMappingURL=withArgs.js.map