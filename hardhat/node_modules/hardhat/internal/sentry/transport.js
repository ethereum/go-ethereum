"use strict";
var __createBinding = (this && this.__createBinding) || (Object.create ? (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    var desc = Object.getOwnPropertyDescriptor(m, k);
    if (!desc || ("get" in desc ? !m.__esModule : desc.writable || desc.configurable)) {
      desc = { enumerable: true, get: function() { return m[k]; } };
    }
    Object.defineProperty(o, k2, desc);
}) : (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    o[k2] = m[k];
}));
var __setModuleDefault = (this && this.__setModuleDefault) || (Object.create ? (function(o, v) {
    Object.defineProperty(o, "default", { enumerable: true, value: v });
}) : function(o, v) {
    o["default"] = v;
});
var __importStar = (this && this.__importStar) || function (mod) {
    if (mod && mod.__esModule) return mod;
    var result = {};
    if (mod != null) for (var k in mod) if (k !== "default" && Object.prototype.hasOwnProperty.call(mod, k)) __createBinding(result, mod, k);
    __setModuleDefault(result, mod);
    return result;
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.getSubprocessTransport = void 0;
const child_process_1 = require("child_process");
const path = __importStar(require("path"));
// This class is wrapped in a function to avoid having to
// import @sentry/node just for the BaseTransport base class
function getSubprocessTransport() {
    const { Status, Transports } = require("@sentry/node");
    class SubprocessTransport extends Transports.BaseTransport {
        async sendEvent(event) {
            const extra = event.extra ?? {};
            const { verbose = false, configPath } = extra;
            // don't send user's full config path for privacy reasons
            delete event.extra?.configPath;
            // we don't care about the verbose setting
            delete event.extra?.verbose;
            const serializedEvent = JSON.stringify(event);
            const env = {
                HARDHAT_SENTRY_EVENT: serializedEvent,
                HARDHAT_SENTRY_VERBOSE: verbose.toString(),
            };
            if (configPath !== undefined) {
                env.HARDHAT_SENTRY_CONFIG_PATH = configPath;
            }
            const subprocessPath = path.join(__dirname, "subprocess");
            const subprocess = (0, child_process_1.spawn)(process.execPath, [subprocessPath], {
                detached: true,
                env,
                stdio: (verbose ? "inherit" : "ignore"),
            });
            subprocess.unref();
            return {
                status: Status.Success,
            };
        }
    }
    return SubprocessTransport;
}
exports.getSubprocessTransport = getSubprocessTransport;
//# sourceMappingURL=transport.js.map