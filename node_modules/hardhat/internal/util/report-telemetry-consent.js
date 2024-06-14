"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
const analytics_1 = require("../cli/analytics");
async function main() {
    // This default value shouldn't be necessary, but we add one to make it
    // easier to recognize if the telemetry consent value is not passed.
    const [telemetryConsent = "<undefined-telemetry-consent>"] = process.argv.slice(2);
    // we pass undefined as the telemetryConsent value because
    // this hit is done before the consent is saved
    const analytics = await analytics_1.Analytics.getInstance(undefined);
    const [_, consentHitPromise] = await analytics.sendTelemetryConsentHit(telemetryConsent);
    await consentHitPromise;
}
main().catch(() => { });
//# sourceMappingURL=report-telemetry-consent.js.map