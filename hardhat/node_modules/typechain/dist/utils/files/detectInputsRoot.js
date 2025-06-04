"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.detectInputsRoot = void 0;
const path_1 = require("path");
const lowestCommonPath_1 = require("./lowestCommonPath");
const shortenFullJsonFilePath_1 = require("./shortenFullJsonFilePath");
function detectInputsRoot(allFiles) {
    return allFiles.length === 1 ? (0, path_1.dirname)((0, shortenFullJsonFilePath_1.shortenFullJsonFilePath)(allFiles[0], allFiles)) : (0, lowestCommonPath_1.lowestCommonPath)(allFiles);
}
exports.detectInputsRoot = detectInputsRoot;
//# sourceMappingURL=detectInputsRoot.js.map