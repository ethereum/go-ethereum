"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.pathFromCwd = void 0;
const path_1 = __importDefault(require("path"));
const process_1 = __importDefault(require("process"));
function pathFromCwd(thePath) {
    const cwd = process_1.default.cwd();
    if (thePath.startsWith(cwd)) {
        return `.${path_1.default.sep}${path_1.default.relative(process_1.default.cwd(), thePath)}`;
    }
    return thePath;
}
exports.pathFromCwd = pathFromCwd;
//# sourceMappingURL=cwd-relative-path.js.map