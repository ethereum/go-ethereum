"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.open = void 0;
const child_process_1 = require("child_process");
const os_1 = __importDefault(require("os"));
function open(filePath) {
    let command;
    switch (os_1.default.platform()) {
        case "win32":
            command = "start";
            break;
        case "darwin":
            command = "open";
            break;
        default:
            command = "xdg-open";
    }
    try {
        (0, child_process_1.execSync)(`${command} ${filePath}`, { stdio: "ignore" });
    }
    catch {
        // do nothing
    }
}
exports.open = open;
//# sourceMappingURL=open.js.map