"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.getFileExtension = void 0;
const path_1 = require("path");
function getFileExtension(path) {
    return (0, path_1.parse)(path).ext;
}
exports.getFileExtension = getFileExtension;
//# sourceMappingURL=getFileExtension.js.map