"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.replaceLastLine = exports.printLine = void 0;
const ansi_escapes_1 = __importDefault(require("ansi-escapes"));
function printLine(line) {
    console.log(line);
}
exports.printLine = printLine;
function replaceLastLine(newLine) {
    if (process.stdout.isTTY === true) {
        process.stdout.write(
        // eslint-disable-next-line prefer-template
        ansi_escapes_1.default.cursorHide +
            ansi_escapes_1.default.cursorPrevLine +
            newLine +
            ansi_escapes_1.default.eraseEndLine +
            "\n" +
            ansi_escapes_1.default.cursorShow);
    }
    else {
        process.stdout.write(`${newLine}\n`);
    }
}
exports.replaceLastLine = replaceLastLine;
//# sourceMappingURL=logger.js.map