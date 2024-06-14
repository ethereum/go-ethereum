import * as Sentry from "@sentry/node";
import debug from "debug";

import { Anonymizer } from "./anonymizer";
import { SENTRY_DSN } from "./reporter";

const log = debug("hardhat:sentry:subprocess");

async function main() {
  const verbose = process.env.HARDHAT_SENTRY_VERBOSE === "true";

  if (verbose) {
    debug.enable("hardhat*");
  }

  log("starting subprocess");

  try {
    Sentry.init({
      dsn: SENTRY_DSN,
    });
  } catch (error) {
    log("Couldn't initialize Sentry: %O", error);
    process.exit(1);
  }

  const serializedEvent = process.env.HARDHAT_SENTRY_EVENT;
  if (serializedEvent === undefined) {
    log("HARDHAT_SENTRY_EVENT env variable is not set, exiting");
    Sentry.captureMessage(
      `There was an error parsing an event: HARDHAT_SENTRY_EVENT env variable is not set`
    );
    return;
  }

  let event: any;
  try {
    event = JSON.parse(serializedEvent);
  } catch {
    log(
      "HARDHAT_SENTRY_EVENT env variable doesn't have a valid JSON, exiting: %o",
      serializedEvent
    );
    Sentry.captureMessage(
      `There was an error parsing an event: HARDHAT_SENTRY_EVENT env variable doesn't have a valid JSON`
    );
    return;
  }

  try {
    const configPath = process.env.HARDHAT_SENTRY_CONFIG_PATH;

    const anonymizer = new Anonymizer(configPath);
    const anonymizedEvent = anonymizer.anonymize(event);

    if (anonymizedEvent.isRight()) {
      if (anonymizer.raisedByHardhat(anonymizedEvent.value)) {
        Sentry.captureEvent(anonymizedEvent.value);
      }
    } else {
      Sentry.captureMessage(
        `There was an error anonymizing an event: ${anonymizedEvent.value}`
      );
    }
  } catch (error: any) {
    log("Couldn't capture event %o, got error %O", event, error);
    Sentry.captureMessage(
      `There was an error capturing an event: ${error.message}`
    );
    return;
  }

  log("sentry event was sent");
}

main().catch(console.error);
