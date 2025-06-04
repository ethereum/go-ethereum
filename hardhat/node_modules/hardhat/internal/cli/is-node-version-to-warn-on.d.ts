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
export declare function isNodeVersionToWarnOn(nodeVersion: string): boolean;
//# sourceMappingURL=is-node-version-to-warn-on.d.ts.map