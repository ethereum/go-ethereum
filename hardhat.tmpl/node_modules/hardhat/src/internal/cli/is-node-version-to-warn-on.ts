import semver from "semver";

import { assertHardhatInvariant } from "../core/errors";
import { SUPPORTED_NODE_VERSIONS } from "./constants";

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
export function isNodeVersionToWarnOn(nodeVersion: string): boolean {
  const supportedVersions = SUPPORTED_NODE_VERSIONS.join(" || ");

  // If the version is supported, no need to warn and short circuit
  if (semver.satisfies(nodeVersion, supportedVersions)) {
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

function _onOddNumberedVersion(nodeVersion: string) {
  return semver.major(nodeVersion) % 2 === 1;
}

function _lessThanMinimumSupportedVersion(
  nodeVersion: string,
  supportedVersions: string
) {
  const minSupportedVersion = semver.minVersion(supportedVersions);

  assertHardhatInvariant(
    minSupportedVersion !== null,
    "Unexpectedly failed to parse the minimum supported version of Node.js"
  );

  return semver.lt(nodeVersion, minSupportedVersion);
}
