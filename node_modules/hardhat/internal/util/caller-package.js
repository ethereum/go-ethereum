"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.getClosestCallerPackage = void 0;
const find_up_1 = __importDefault(require("find-up"));
const path_1 = __importDefault(require("path"));
function findClosestPackageJson(file) {
    return find_up_1.default.sync("package.json", { cwd: path_1.default.dirname(file) });
}
/**
 * Returns the name of the closest package in the callstack that isn't this.
 */
function getClosestCallerPackage() {
    const previousPrepareStackTrace = Error.prepareStackTrace;
    Error.prepareStackTrace = (e, s) => s;
    const error = new Error();
    const stack = error.stack;
    Error.prepareStackTrace = previousPrepareStackTrace;
    const currentPackage = findClosestPackageJson(__filename);
    for (const callSite of stack) {
        const fileName = callSite.getFileName();
        // fileName is string | null in @types/node <=18
        // and string | undefined in @types/node 20
        if (fileName !== null &&
            fileName !== undefined &&
            path_1.default.isAbsolute(fileName)) {
            const callerPackage = findClosestPackageJson(fileName);
            if (callerPackage === currentPackage) {
                continue;
            }
            if (callerPackage === null) {
                return undefined;
            }
            return require(callerPackage).name;
        }
    }
    return undefined;
}
exports.getClosestCallerPackage = getClosestCallerPackage;
//# sourceMappingURL=caller-package.js.map