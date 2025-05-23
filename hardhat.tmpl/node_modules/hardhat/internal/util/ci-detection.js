"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.isRunningOnCiServer = void 0;
const os_1 = __importDefault(require("os"));
// This has been tested in:
//   - Travis CI
//   - Circle CI
//   - GitHub Actions
//   - Azure Pipelines
//
// This should also work in this CI providers because they set process.env.CI:
//   - AppVeyor
//   - Bitbucket Pipelines
//   - GitLab CI
//
// This should also work:
//   - AWS CodeBuild -- Special case
//   - Jenkins -- Using process.env.BUILD_NUMBER
//   - ZEIT Now -- Special case
function isRunningOnCiServer() {
    const ci = require("ci-info");
    return (ci.isCI ||
        isGithubActions() ||
        isLinuxWithoutDisplayServer() ||
        isNow() ||
        isAwsCodeBuild());
}
exports.isRunningOnCiServer = isRunningOnCiServer;
function isGithubActions() {
    return process.env.GITHUB_ACTIONS !== undefined;
}
function isLinuxWithoutDisplayServer() {
    if (os_1.default.type() !== "Linux") {
        return false;
    }
    if (process.env.DISPLAY !== undefined) {
        return false;
    }
    if (process.env.WAYLAND_DISPLAY !== undefined) {
        return false;
    }
    return true;
}
function isNow() {
    return (process.env.NOW !== undefined || process.env.DEPLOYMENT_ID !== undefined);
}
function isAwsCodeBuild() {
    return process.env.CODEBUILD_BUILD_NUMBER !== undefined;
}
//# sourceMappingURL=ci-detection.js.map