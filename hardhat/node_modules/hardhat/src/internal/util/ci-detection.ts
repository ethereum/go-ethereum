import type CiInfoT from "ci-info";

import os from "os";

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
export function isRunningOnCiServer(): boolean {
  const ci = require("ci-info") as typeof CiInfoT;
  return (
    ci.isCI ||
    isGithubActions() ||
    isLinuxWithoutDisplayServer() ||
    isNow() ||
    isAwsCodeBuild()
  );
}

function isGithubActions(): boolean {
  return process.env.GITHUB_ACTIONS !== undefined;
}

function isLinuxWithoutDisplayServer(): boolean {
  if (os.type() !== "Linux") {
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
  return (
    process.env.NOW !== undefined || process.env.DEPLOYMENT_ID !== undefined
  );
}

function isAwsCodeBuild() {
  return process.env.CODEBUILD_BUILD_NUMBER !== undefined;
}
