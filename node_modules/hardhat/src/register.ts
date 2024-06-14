import debug from "debug";

import { HardhatContext } from "./internal/context";
import { loadConfigAndTasks } from "./internal/core/config/config-loading";
import { getEnvHardhatArguments } from "./internal/core/params/env-variables";
import { HARDHAT_PARAM_DEFINITIONS } from "./internal/core/params/hardhat-params";
import { Environment } from "./internal/core/runtime-environment";
import {
  loadTsNode,
  willRunWithTypescript,
} from "./internal/core/typescript-support";
import {
  disableReplWriterShowProxy,
  isNodeCalledWithoutAScript,
} from "./internal/util/console";

if (!HardhatContext.isCreated()) {
  require("source-map-support/register");

  const ctx = HardhatContext.createHardhatContext();

  if (isNodeCalledWithoutAScript()) {
    disableReplWriterShowProxy();
  }

  const hardhatArguments = getEnvHardhatArguments(
    HARDHAT_PARAM_DEFINITIONS,
    process.env
  );

  if (hardhatArguments.verbose) {
    debug.enable("hardhat*");
  }

  if (willRunWithTypescript(hardhatArguments.config)) {
    loadTsNode(hardhatArguments.tsconfig, hardhatArguments.typecheck);
  }

  const { resolvedConfig, userConfig } = loadConfigAndTasks(hardhatArguments);

  const env = new Environment(
    resolvedConfig,
    hardhatArguments,
    ctx.tasksDSL.getTaskDefinitions(),
    ctx.tasksDSL.getScopesDefinitions(),
    ctx.environmentExtenders,
    ctx.experimentalHardhatNetworkMessageTraceHooks,
    userConfig,
    ctx.providerExtenders
  );

  ctx.setHardhatRuntimeEnvironment(env);

  env.injectToGlobal();
}
