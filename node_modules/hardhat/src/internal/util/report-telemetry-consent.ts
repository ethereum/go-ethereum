import { Analytics } from "../cli/analytics";

async function main() {
  // This default value shouldn't be necessary, but we add one to make it
  // easier to recognize if the telemetry consent value is not passed.
  const [telemetryConsent = "<undefined-telemetry-consent>"] =
    process.argv.slice(2);

  // we pass undefined as the telemetryConsent value because
  // this hit is done before the consent is saved
  const analytics = await Analytics.getInstance(undefined);

  const [_, consentHitPromise] = await analytics.sendTelemetryConsentHit(
    telemetryConsent as "yes" | "no"
  );
  await consentHitPromise;
}

main().catch(() => {});
