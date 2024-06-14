"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.HardhatChaiMatchersNonChainableMatcherError = exports.HardhatChaiMatchersAssertionError = exports.HardhatChaiMatchersDecodingError = exports.HardhatChaiMatchersError = void 0;
const plugins_1 = require("hardhat/plugins");
class HardhatChaiMatchersError extends plugins_1.NomicLabsHardhatPluginError {
    constructor(message, parent) {
        super("@nomicfoundation/hardhat-chai-matchers", message, parent);
    }
}
exports.HardhatChaiMatchersError = HardhatChaiMatchersError;
class HardhatChaiMatchersDecodingError extends HardhatChaiMatchersError {
    constructor(encodedData, type, parent) {
        const message = `There was an error decoding '${encodedData}' as a ${type}`;
        super(message, parent);
    }
}
exports.HardhatChaiMatchersDecodingError = HardhatChaiMatchersDecodingError;
/**
 * This class is used to assert assumptions in our implementation. Chai's
 * AssertionError should be used for user assertions.
 */
class HardhatChaiMatchersAssertionError extends HardhatChaiMatchersError {
    constructor(message) {
        super(`Assertion error: ${message}`);
    }
}
exports.HardhatChaiMatchersAssertionError = HardhatChaiMatchersAssertionError;
class HardhatChaiMatchersNonChainableMatcherError extends HardhatChaiMatchersError {
    constructor(matcherName, previousMatcherName) {
        super(`The matcher '${matcherName}' cannot be chained after '${previousMatcherName}'. For more information, please refer to the documentation at: https://hardhat.org/chaining-async-matchers.`);
    }
}
exports.HardhatChaiMatchersNonChainableMatcherError = HardhatChaiMatchersNonChainableMatcherError;
//# sourceMappingURL=errors.js.map