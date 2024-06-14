"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.DEFAULT_AUTOMINE_REQUIRED_CONFIRMATIONS = exports.defaultConfig = void 0;
/**
 * Ignitions default deployment configuration values.
 */
exports.defaultConfig = {
    blockPollingInterval: 1_000,
    timeBeforeBumpingFees: 3 * 60 * 1_000,
    maxFeeBumps: 4,
    requiredConfirmations: 5,
};
/**
 * The default number of confirmations to wait for when automining.
 */
exports.DEFAULT_AUTOMINE_REQUIRED_CONFIRMATIONS = 1;
//# sourceMappingURL=defaultConfig.js.map