import type { Event, Response } from "@sentry/node";
import { spawn } from "child_process";
import * as path from "path";

// This class is wrapped in a function to avoid having to
// import @sentry/node just for the BaseTransport base class
export function getSubprocessTransport(): any {
  const { Status, Transports } = require("@sentry/node");

  class SubprocessTransport extends Transports.BaseTransport {
    public async sendEvent(event: Event): Promise<Response> {
      const extra: { verbose?: boolean; configPath?: string } =
        event.extra ?? {};
      const { verbose = false, configPath } = extra;

      // don't send user's full config path for privacy reasons
      delete event.extra?.configPath;

      // we don't care about the verbose setting
      delete event.extra?.verbose;

      const serializedEvent = JSON.stringify(event);

      const env: Record<string, string> = {
        HARDHAT_SENTRY_EVENT: serializedEvent,
        HARDHAT_SENTRY_VERBOSE: verbose.toString(),
      };

      if (configPath !== undefined) {
        env.HARDHAT_SENTRY_CONFIG_PATH = configPath;
      }

      const subprocessPath = path.join(__dirname, "subprocess");

      const subprocess = spawn(process.execPath, [subprocessPath], {
        detached: true,
        env,
        stdio: (verbose ? "inherit" : "ignore") as any,
      });

      subprocess.unref();

      return {
        status: Status.Success,
      };
    }
  }

  return SubprocessTransport;
}
