"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.revertedWithCustomErrorWithArgs = exports.supportRevertedWithCustomError = exports.REVERTED_WITH_CUSTOM_ERROR_CALLED = void 0;
const constants_1 = require("../constants");
const utils_1 = require("../utils");
const utils_2 = require("../../utils");
const utils_3 = require("./utils");
exports.REVERTED_WITH_CUSTOM_ERROR_CALLED = "customErrorAssertionCalled";
function supportRevertedWithCustomError(Assertion, chaiUtils) {
    Assertion.addMethod(constants_1.REVERTED_WITH_CUSTOM_ERROR_MATCHER, function (contract, expectedCustomErrorName, ...args) {
        // capture negated flag before async code executes; see buildAssert's jsdoc
        const negated = this.__flags.negate;
        const { iface, expectedCustomError } = validateInput(this._obj, contract, expectedCustomErrorName, args);
        (0, utils_1.preventAsyncMatcherChaining)(this, constants_1.REVERTED_WITH_CUSTOM_ERROR_MATCHER, chaiUtils);
        const onSuccess = () => {
            if (chaiUtils.flag(this, constants_1.ASSERTION_ABORTED) === true) {
                return;
            }
            const assert = (0, utils_2.buildAssert)(negated, onSuccess);
            assert(false, `Expected transaction to be reverted with custom error '${expectedCustomErrorName}', but it didn't revert`);
        };
        const onError = (error) => {
            if (chaiUtils.flag(this, constants_1.ASSERTION_ABORTED) === true) {
                return;
            }
            const { toBeHex } = require("ethers");
            const assert = (0, utils_2.buildAssert)(negated, onError);
            const returnData = (0, utils_3.getReturnDataFromError)(error);
            const decodedReturnData = (0, utils_3.decodeReturnData)(returnData);
            if (decodedReturnData.kind === "Empty") {
                assert(false, `Expected transaction to be reverted with custom error '${expectedCustomErrorName}', but it reverted without a reason`);
            }
            else if (decodedReturnData.kind === "Error") {
                assert(false, `Expected transaction to be reverted with custom error '${expectedCustomErrorName}', but it reverted with reason '${decodedReturnData.reason}'`);
            }
            else if (decodedReturnData.kind === "Panic") {
                assert(false, `Expected transaction to be reverted with custom error '${expectedCustomErrorName}', but it reverted with panic code ${toBeHex(decodedReturnData.code)} (${decodedReturnData.description})`);
            }
            else if (decodedReturnData.kind === "Custom") {
                if (decodedReturnData.id === expectedCustomError.selector) {
                    // add flag with the data needed for .withArgs
                    const customErrorAssertionData = {
                        contractInterface: iface,
                        customError: expectedCustomError,
                        returnData,
                    };
                    this.customErrorData = customErrorAssertionData;
                    assert(true, undefined, `Expected transaction NOT to be reverted with custom error '${expectedCustomErrorName}', but it was`);
                }
                else {
                    // try to decode the actual custom error
                    // this will only work when the error comes from the given contract
                    const actualCustomError = iface.getError(decodedReturnData.id);
                    if (actualCustomError === null) {
                        assert(false, `Expected transaction to be reverted with custom error '${expectedCustomErrorName}', but it reverted with a different custom error`);
                    }
                    else {
                        assert(false, `Expected transaction to be reverted with custom error '${expectedCustomErrorName}', but it reverted with custom error '${actualCustomError.name}'`);
                    }
                }
            }
            else {
                const _exhaustiveCheck = decodedReturnData;
            }
        };
        const derivedPromise = Promise.resolve(this._obj).then(onSuccess, onError);
        // needed for .withArgs
        chaiUtils.flag(this, exports.REVERTED_WITH_CUSTOM_ERROR_CALLED, true);
        this.promise = derivedPromise;
        this.then = derivedPromise.then.bind(derivedPromise);
        this.catch = derivedPromise.catch.bind(derivedPromise);
        return this;
    });
}
exports.supportRevertedWithCustomError = supportRevertedWithCustomError;
function validateInput(obj, contract, expectedCustomErrorName, args) {
    try {
        // check the case where users forget to pass the contract as the first
        // argument
        if (typeof contract === "string" || contract?.interface === undefined) {
            // discard subject since it could potentially be a rejected promise
            throw new TypeError("The first argument of .revertedWithCustomError must be the contract that defines the custom error");
        }
        // validate custom error name
        if (typeof expectedCustomErrorName !== "string") {
            throw new TypeError("Expected the custom error name to be a string");
        }
        const iface = contract.interface;
        const expectedCustomError = iface.getError(expectedCustomErrorName);
        // check that interface contains the given custom error
        if (expectedCustomError === null) {
            throw new Error(`The given contract doesn't have a custom error named '${expectedCustomErrorName}'`);
        }
        if (args.length > 0) {
            throw new Error("`.revertedWithCustomError` expects only two arguments: the contract and the error name. Arguments should be asserted with the `.withArgs` helper.");
        }
        return { iface, expectedCustomError };
    }
    catch (e) {
        // if the input validation fails, we discard the subject since it could
        // potentially be a rejected promise
        Promise.resolve(obj).catch(() => { });
        throw e;
    }
}
async function revertedWithCustomErrorWithArgs(context, Assertion, chaiUtils, expectedArgs, ssfi) {
    const negated = false; // .withArgs cannot be negated
    const assert = (0, utils_2.buildAssert)(negated, ssfi);
    const customErrorAssertionData = context.customErrorData;
    if (customErrorAssertionData === undefined) {
        throw new Error("[.withArgs] should never happen, please submit an issue to the Hardhat repository");
    }
    const { contractInterface, customError, returnData } = customErrorAssertionData;
    const errorFragment = contractInterface.getError(customError.name);
    (0, utils_1.assertIsNotNull)(errorFragment, "errorFragment");
    // We transform ether's Array-like object into an actual array as it's safer
    const actualArgs = (0, utils_3.resultToArray)(contractInterface.decodeErrorResult(errorFragment, returnData));
    (0, utils_1.assertArgsArraysEqual)(Assertion, expectedArgs, actualArgs, `"${customError.name}" custom error`, "error", assert, ssfi);
}
exports.revertedWithCustomErrorWithArgs = revertedWithCustomErrorWithArgs;
//# sourceMappingURL=revertedWithCustomError.js.map