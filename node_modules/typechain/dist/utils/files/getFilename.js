"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.getFilename = void 0;
const path_1 = require("path");
function getFilename(path) {
    return (0, path_1.parse)(path).name;
}
exports.getFilename = getFilename;
//# sourceMappingURL=getFilename.js.map