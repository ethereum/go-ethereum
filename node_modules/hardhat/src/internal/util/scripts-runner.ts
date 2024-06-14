import debug from "debug";
import path from "path";

import { HardhatArguments } from "../../types";
import { isRunningHardhatCoreTests } from "../core/execution-mode";
import { getEnvVariablesMap } from "../core/params/env-variables";

const log = debug("hardhat:core:scripts-runner");

export async function runScript(
  scriptPath: string,
  scriptArgs: string[] = [],
  extraNodeArgs: string[] = [],
  extraEnvVars: { [name: string]: string } = {}
): Promise<number> {
  const { fork } = await import("child_process");

  return new Promise((resolve, reject) => {
    const processExecArgv = withFixedInspectArg(process.execArgv);

    const nodeArgs = [
      ...processExecArgv,
      ...getTsNodeArgsIfNeeded(
        scriptPath,
        extraEnvVars.HARDHAT_TYPECHECK === "true"
      ),
      ...extraNodeArgs,
    ];

    const envVars = { ...process.env, ...extraEnvVars };

    const childProcess = fork(scriptPath, scriptArgs, {
      stdio: "inherit",
      execArgv: nodeArgs,
      env: envVars,
    });

    childProcess.once("close", (status) => {
      log(`Script ${scriptPath} exited with status code ${status ?? "null"}`);

      resolve(status as number);
    });
    childProcess.once("error", reject);
  });
}

export async function runScriptWithHardhat(
  hardhatArguments: HardhatArguments,
  scriptPath: string,
  scriptArgs: string[] = [],
  extraNodeArgs: string[] = [],
  extraEnvVars: { [name: string]: string } = {}
): Promise<number> {
  log(`Creating Hardhat subprocess to run ${scriptPath}`);

  return runScript(
    scriptPath,
    scriptArgs,
    [
      ...extraNodeArgs,
      "--require",
      path.join(__dirname, "..", "..", "register"),
    ],
    {
      ...getEnvVariablesMap(hardhatArguments),
      ...extraEnvVars,
    }
  );
}

/**
 * Fix debugger "inspect" arg from process.argv, if present.
 *
 * When running this process with a debugger, a debugger port
 * is specified via the "--inspect-brk=" arg param in some IDEs/setups.
 *
 * This normally works, but if we do a fork afterwards, we'll get an error stating
 * that the port is already in use (since the fork would also use the same args,
 * therefore the same port number). To prevent this issue, we could replace the port number with
 * a different free one, or simply use the port-agnostic --inspect" flag, and leave the debugger
 * port selection to the Node process itself, which will pick an empty AND valid one.
 *
 * This way, we can properly use the debugger for this process AND for the executed
 * script itself - even if it's compiled using ts-node.
 */
function withFixedInspectArg(argv: string[]) {
  const fixIfInspectArg = (arg: string) => {
    if (arg.toLowerCase().includes("--inspect-brk=")) {
      return "--inspect";
    }
    return arg;
  };
  return argv.map(fixIfInspectArg);
}

function getTsNodeArgsIfNeeded(
  scriptPath: string,
  shouldTypecheck: boolean
): string[] {
  if (process.execArgv.includes("ts-node/register")) {
    return [];
  }

  // if we are running the tests we only want to transpile, or these tests
  // take forever
  if (isRunningHardhatCoreTests()) {
    return ["--require", "ts-node/register/transpile-only"];
  }

  // If the script we are going to run is .ts we need ts-node
  if (/\.tsx?$/i.test(scriptPath)) {
    return [
      "--require",
      `ts-node/register${shouldTypecheck ? "" : "/transpile-only"}`,
    ];
  }

  return [];
}
