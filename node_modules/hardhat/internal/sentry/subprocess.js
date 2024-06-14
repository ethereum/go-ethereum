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
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const Sentry = __importStar(require("@sentry/node"));
const debug_1 = __importDefault(require("debug"));
const anonymizer_1 = require("./anonymizer");
const reporter_1 = require("./reporter");
const log = (0, debug_1.default)("hardhat:sentry:subprocess");
async function main() {
    const verbose = process.env.HARDHAT_SENTRY_VERBOSE === "true";
    if (verbose) {
        debug_1.default.enable("hardhat*");
    }
    log("starting subprocess");
    try {
        Sentry.init({
            dsn: reporter_1.SENTRY_DSN,
        });
    }
    catch (error) {
        log("Couldn't initialize Sentry: %O", error);
        process.exit(1);
    }
    const serializedEvent = process.env.HARDHAT_SENTRY_EVENT;
    if (serializedEvent === undefined) {
        log("HARDHAT_SENTRY_EVENT env variable is not set, exiting");
        Sentry.captureMessage(`There was an error parsing an event: HARDHAT_SENTRY_EVENT env variable is not set`);
        return;
    }
    let event;
    try {
        event = JSON.parse(serializedEvent);
    }
    catch {
        log("HARDHAT_SENTRY_EVENT env variable doesn't have a valid JSON, exiting: %o", serializedEvent);
        Sentry.captureMessage(`There was an error parsing an event: HARDHAT_SENTRY_EVENT env variable doesn't have a valid JSON`);
        return;
    }
    try {
        const configPath = process.env.HARDHAT_SENTRY_CONFIG_PATH;
        const anonymizer = new anonymizer_1.Anonymizer(configPath);
        const anonymizedEvent = anonymizer.anonymize(event);
        if (anonymizedEvent.isRight()) {
            if (anonymizer.raisedByHardhat(anonymizedEvent.value)) {
                Sentry.captureEvent(anonymizedEvent.value);
            }
        }
        else {
            Sentry.captureMessage(`There was an error anonymizing an event: ${anonymizedEvent.value}`);
        }
    }
    catch (error) {
        log("Couldn't capture event %o, got error %O", event, error);
        Sentry.captureMessage(`There was an error capturing an event: ${error.message}`);
        return;
    }
    log("sentry event was sent");
}
main().catch(console.error);
//# sourceMappingURL=subprocess.js.map