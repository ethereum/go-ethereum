"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.buildAssert = void 0;
const chai_1 = require("chai");
/**
 * This function is used by the matchers to obtain an `assert` function, which
 * should be used instead of `this.assert`.
 *
 * The first parameter is the value of the `negated` flag. Keep in mind that
 * this value should be captured at the beginning of the matcher's
 * implementation, before any async code is executed. Otherwise things like
 * `.to.emit().and.not.to.emit()` won't work, because by the time the async part
 * of the first emit is executd, the `.not` (executed synchronously) has already
 * modified the flag.
 *
 * The second parameter is what Chai calls the "start stack function indicator",
 * a function that is used to build the stack trace. It's unclear to us what's
 * the best way to use this value, so this needs some trial-and-error. Use the
 * existing matchers for a reference of something that works well enough.
 */
function buildAssert(negated, ssfi) {
    return function (condition, messageFalse, messageTrue) {
        if (!negated && !condition) {
            if (messageFalse === undefined) {
                throw new Error("Assertion doesn't have an error message. Please open an issue to report this.");
            }
            const message = typeof messageFalse === "function" ? messageFalse() : messageFalse;
            throw new chai_1.AssertionError(message, undefined, ssfi);
        }
        if (negated && condition) {
            if (messageTrue === undefined) {
                throw new Error("Assertion doesn't have an error message. Please open an issue to report this.");
            }
            const message = typeof messageTrue === "function" ? messageTrue() : messageTrue;
            throw new chai_1.AssertionError(message, undefined, ssfi);
        }
    };
}
exports.buildAssert = buildAssert;
//# sourceMappingURL=utils.js.map