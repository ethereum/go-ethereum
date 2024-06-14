"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.calculateStartingMessage = void 0;
const chalk_1 = __importDefault(require("chalk"));
/**
 * Display the temporary starting message. Note this does not print a newline.
 *
 * @param state - the UI state
 */
function calculateStartingMessage({ moduleName, deploymentDir, }) {
    const warningMessage = chalk_1.default.yellow(chalk_1.default.bold(`You are running Hardhat Ignition against an in-process instance of Hardhat Network.
This will execute the deployment, but the results will be lost.
You can use --network <network-name> to deploy to a different network.`));
    const startingMessage = `Hardhat Ignition starting for [ ${moduleName ?? "unknown"} ]...`;
    return deploymentDir === undefined
        ? `${warningMessage}\n\n${startingMessage}`
        : startingMessage;
}
exports.calculateStartingMessage = calculateStartingMessage;
//# sourceMappingURL=calculate-starting-message.js.map