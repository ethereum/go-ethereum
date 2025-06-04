"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.isNodeVersionToWarnOn = void 0;
const semver_1 = __importDefault(require("semver"));
const errors_1 = require("../core/errors");
const constants_1 = require("./constants");
/**
 * Determine if the node version should trigger an unsupported
 * warning.
 *
 * The current rule is that an unsupported warning will be shown if
 *
 * 1. An odd numbered version of Node.js is used - as this will never go to LTS
 * 2. The version is less than the minimum supported version
 *
 * We intentionally do not warn on newer **even** versions of Node.js.
 */
function isNodeVersionToWarnOn(nodeVersion) {
    const supportedVersions = constants_1.SUPPORTED_NODE_VERSIONS.join(" || ");
    // If the version is supported, no need to warn and short circuit
    if (semver_1.default.satisfies(nodeVersion, supportedVersions)) {
        return false;
    }
    if (_onOddNumberedVersion(nodeVersion)) {
        return true;
    }
    if (_lessThanMinimumSupportedVersion(nodeVersion, supportedVersions)) {
        return true;
    }
    // A newer version of Node.js that will go to LTS
    // we have opted not to warn.
    return false;
}
exports.isNodeVersionToWarnOn = isNodeVersionToWarnOn;
function _onOddNumberedVersion(nodeVersion) {
    return semver_1.default.major(nodeVersion) % 2 === 1;
}
function _lessThanMinimumSupportedVersion(nodeVersion, supportedVersions) {
    const minSupportedVersion = semver_1.default.minVersion(supportedVersions);
    (0, errors_1.assertHardhatInvariant)(minSupportedVersion !== null, "Unexpectedly failed to parse the minimum supported version of Node.js");
    return semver_1.default.lt(nodeVersion, minSupportedVersion);
}
//# sourceMappingURL=is-node-version-to-warn-on.js.map