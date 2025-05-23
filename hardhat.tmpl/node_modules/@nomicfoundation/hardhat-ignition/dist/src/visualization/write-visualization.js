"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.writeVisualization = void 0;
const fs_extra_1 = require("fs-extra");
const plugins_1 = require("hardhat/plugins");
const path_1 = __importDefault(require("path"));
async function writeVisualization(visualizationPayload, { cacheDir }) {
    const templateDir = path_1.default.join(require.resolve("@nomicfoundation/ignition-ui/package.json"), "../dist");
    const templateDirExists = await (0, fs_extra_1.pathExists)(templateDir);
    if (!templateDirExists) {
        throw new plugins_1.NomicLabsHardhatPluginError("@nomicfouncation/hardhat-ignition", `Unable to find template directory: ${templateDir}`);
    }
    const visualizationDir = path_1.default.join(cacheDir, "visualization");
    await (0, fs_extra_1.ensureDir)(visualizationDir);
    const indexHtml = await (0, fs_extra_1.readFile)(path_1.default.join(templateDir, "index.html"));
    const updatedHtml = indexHtml
        .toString()
        .replace('{ "unloaded": true }', JSON.stringify(visualizationPayload));
    await (0, fs_extra_1.writeFile)(path_1.default.join(visualizationDir, "index.html"), updatedHtml);
}
exports.writeVisualization = writeVisualization;
//# sourceMappingURL=write-visualization.js.map