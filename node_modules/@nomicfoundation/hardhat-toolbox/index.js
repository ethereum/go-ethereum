"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
require("@nomicfoundation/hardhat-chai-matchers");
require("@nomicfoundation/hardhat-ethers");
require("@nomicfoundation/hardhat-verify");
require("@nomicfoundation/hardhat-ignition-ethers");
require("@typechain/hardhat");
require("hardhat-gas-reporter");
require("solidity-coverage");
/**
 * If a new official plugin is added, make sure to update:
 *   - The tsconfig.json file
 *   - The hardhat-toolbox GitHub workflow
 *   - The parts of the documentation that install hardhat-toolbox with npm 6 or yarn
 *   - The list of dependencies that the sample projects install
 *   - The README
 */
const config_1 = require("hardhat/config");
(0, config_1.extendConfig)((config, userConfig) => {
    const configAsAny = config;
    // hardhat-gas-reporter doesn't use extendConfig, so
    // the values of config.gasReporter and userConfig.gasReporter
    // are the same. The userConfigVersion is frozen though, so we
    // shouldn't use it.
    const gasReporterConfig = configAsAny.gasReporter;
    configAsAny.gasReporter = gasReporterConfig ?? {};
    if (gasReporterConfig?.enabled === undefined) {
        // eslint-disable-next-line @typescript-eslint/strict-boolean-expressions
        configAsAny.gasReporter.enabled = process.env.REPORT_GAS ? true : false;
    }
    if (gasReporterConfig?.currency === undefined) {
        configAsAny.gasReporter.currency = "USD";
    }
    // We don't generate types for js projects
    if (userConfig?.typechain?.dontOverrideCompile === undefined) {
        config.typechain.dontOverrideCompile =
            config.paths.configFile.endsWith(".js") ||
                config.paths.configFile.endsWith(".cjs");
    }
});
//# sourceMappingURL=index.js.map