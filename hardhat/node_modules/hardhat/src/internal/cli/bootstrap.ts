#!/usr/bin/env node

import picocolors from "picocolors";

import { isNodeVersionToWarnOn } from "./is-node-version-to-warn-on";

if (isNodeVersionToWarnOn(process.version)) {
  console.warn(
    picocolors.yellow(picocolors.bold(`WARNING:`)),
    `You are currently using Node.js ${process.version}, which is not supported by Hardhat. This can lead to unexpected behavior. See https://hardhat.org/nodejs-versions`
  );
  console.log();
  console.log();
}

require("./cli");
