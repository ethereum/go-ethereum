"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.resolveStrategy = void 0;
const errors_1 = require("../errors");
const errors_list_1 = require("../internal/errors-list");
const basic_strategy_1 = require("./basic-strategy");
const create2_strategy_1 = require("./create2-strategy");
function resolveStrategy(strategyName, strategyConfig) {
    if (strategyName === undefined) {
        return new basic_strategy_1.BasicStrategy();
    }
    switch (strategyName) {
        case "basic":
            return new basic_strategy_1.BasicStrategy();
        case "create2":
            if (strategyConfig === undefined) {
                throw new errors_1.IgnitionError(errors_list_1.ERRORS.STRATEGIES.MISSING_CONFIG, {
                    strategyName,
                });
            }
            if (typeof strategyConfig.salt !== "string") {
                throw new errors_1.IgnitionError(errors_list_1.ERRORS.STRATEGIES.MISSING_CONFIG_PARAM, {
                    strategyName,
                    requiredParam: "salt",
                });
            }
            if (hexStringLengthInBytes(strategyConfig.salt) !== 32) {
                throw new errors_1.IgnitionError(errors_list_1.ERRORS.STRATEGIES.INVALID_CONFIG_PARAM, {
                    strategyName,
                    paramName: "salt",
                    reason: "The salt must be 32 bytes in length",
                });
            }
            return new create2_strategy_1.Create2Strategy({ salt: strategyConfig.salt });
        default:
            throw new errors_1.IgnitionError(errors_list_1.ERRORS.STRATEGIES.UNKNOWN_STRATEGY, {
                strategyName,
            });
    }
}
exports.resolveStrategy = resolveStrategy;
function hexStringLengthInBytes(hexString) {
    const normalizedHexString = hexString.startsWith("0x")
        ? hexString.substring(2)
        : hexString;
    return normalizedHexString.length / 2;
}
//# sourceMappingURL=resolve-strategy.js.map