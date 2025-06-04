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
exports.requestTelemetryConsent = exports.Analytics = void 0;
const debug_1 = __importDefault(require("debug"));
const node_os_1 = __importDefault(require("node:os"));
const node_path_1 = require("node:path");
const node_child_process_1 = require("node:child_process");
const execution_mode_1 = require("../core/execution-mode");
const ci_detection_1 = require("../util/ci-detection");
const global_dir_1 = require("../util/global-dir");
const packageInfo_1 = require("../util/packageInfo");
const prompt_1 = require("./prompt");
const log = (0, debug_1.default)("hardhat:core:analytics");
class Analytics {
    static async getInstance(telemetryConsent) {
        const analytics = new Analytics(await getClientId(), telemetryConsent, getUserType());
        return analytics;
    }
    constructor(clientId, telemetryConsent, userType) {
        this._analyticsUrl = "https://www.google-analytics.com/mp/collect";
        this._apiSecret = "fQ5joCsDRTOp55wX8a2cVw";
        this._measurementId = "G-8LQ007N2QJ";
        this._clientId = clientId;
        this._enabled =
            !(0, execution_mode_1.isLocalDev)() && !(0, ci_detection_1.isRunningOnCiServer)() && telemetryConsent === true;
        this._userType = userType;
        this._sessionId = Math.random().toString();
    }
    /**
     * Attempt to send a hit to Google Analytics using the Measurement Protocol.
     * This function returns immediately after starting the request, returning a function for aborting it.
     * The idea is that we don't want Hardhat tasks to be slowed down by a slow network request, so
     * Hardhat can abort the request if it takes too much time.
     *
     * Trying to abort a successfully completed request is a no-op, so it's always safe to call it.
     *
     * @returns The abort function
     */
    async sendTaskHit(scopeName, taskName) {
        if (!this._enabled) {
            return [() => { }, Promise.resolve()];
        }
        let eventParams = {};
        if ((scopeName === "ignition" && taskName === "deploy") ||
            (scopeName === undefined && taskName === "deploy")) {
            eventParams = {
                scope: scopeName,
                task: taskName,
            };
        }
        return this._sendHit(await this._buildTaskHitPayload(eventParams));
    }
    async sendTelemetryConsentHit(userConsent) {
        const telemetryConsentHitPayload = {
            client_id: "hardhat_telemetry_consent",
            user_id: "hardhat_telemetry_consent",
            user_properties: {},
            events: [
                {
                    name: "TelemetryConsentResponse",
                    params: {
                        userConsent,
                    },
                },
            ],
        };
        return this._sendHit(telemetryConsentHitPayload);
    }
    async _buildTaskHitPayload(eventParams = {}) {
        return {
            client_id: this._clientId,
            user_id: this._clientId,
            user_properties: {
                projectId: { value: "hardhat-project" },
                userType: { value: this._userType },
                hardhatVersion: { value: await getHardhatVersion() },
                operatingSystem: { value: node_os_1.default.platform() },
                nodeVersion: { value: process.version },
            },
            events: [
                {
                    name: "task",
                    params: {
                        // From the GA docs: amount of time someone spends with your web
                        // page in focus or app screen in the foreground
                        // The parameter has no use for our app, but it's required in order
                        // for user activity to display in standard reports like Realtime
                        engagement_time_msec: "10000",
                        session_id: this._sessionId,
                        ...eventParams,
                    },
                },
            ],
        };
    }
    _sendHit(payload) {
        const { request } = require("undici");
        const eventName = payload.events[0].name;
        log(`Sending hit for ${eventName}`);
        const controller = new AbortController();
        const abortAnalytics = () => {
            log(`Aborting hit for ${eventName}`);
            controller.abort();
        };
        log(`Hit payload: ${JSON.stringify(payload)}`);
        const hitPromise = request(this._analyticsUrl, {
            query: {
                api_secret: this._apiSecret,
                measurement_id: this._measurementId,
            },
            body: JSON.stringify(payload),
            method: "POST",
            signal: controller.signal,
        })
            .then(() => {
            log(`Hit for ${eventName} sent successfully`);
        })
            .catch(() => {
            log("Hit request failed");
        });
        return [abortAnalytics, hitPromise];
    }
}
exports.Analytics = Analytics;
async function getClientId() {
    let clientId = await (0, global_dir_1.readAnalyticsId)();
    if (clientId === undefined) {
        clientId =
            (await (0, global_dir_1.readSecondLegacyAnalyticsId)()) ??
                (await (0, global_dir_1.readFirstLegacyAnalyticsId)());
        if (clientId === undefined) {
            const { v4: uuid } = await Promise.resolve().then(() => __importStar(require("uuid")));
            log("Client Id not found, generating a new one");
            clientId = uuid();
        }
        await (0, global_dir_1.writeAnalyticsId)(clientId);
    }
    return clientId;
}
function getUserType() {
    return (0, ci_detection_1.isRunningOnCiServer)() ? "CI" : "Developer";
}
async function getHardhatVersion() {
    const { version } = await (0, packageInfo_1.getPackageJson)();
    return `Hardhat ${version}`;
}
async function requestTelemetryConsent() {
    const telemetryConsent = await (0, prompt_1.confirmTelemetryConsent)();
    if (telemetryConsent === undefined) {
        return;
    }
    (0, global_dir_1.writeTelemetryConsent)(telemetryConsent);
    const reportTelemetryConsentPath = (0, node_path_1.join)(__dirname, "..", "util", "report-telemetry-consent.js");
    const subprocess = (0, node_child_process_1.spawn)(process.execPath, [reportTelemetryConsentPath, telemetryConsent ? "yes" : "no"], {
        detached: true,
        stdio: "ignore",
    });
    subprocess.unref();
    return telemetryConsent;
}
exports.requestTelemetryConsent = requestTelemetryConsent;
//# sourceMappingURL=analytics.js.map