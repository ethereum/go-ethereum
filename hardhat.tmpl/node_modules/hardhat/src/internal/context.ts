import {
  ConfigExtender,
  EnvironmentExtender,
  HardhatRuntimeEnvironment,
  ProviderExtender,
} from "../types";

import { assertHardhatInvariant, HardhatError } from "./core/errors";
import { ERRORS } from "./core/errors-list";
import { VarsManagerSetup } from "./core/vars/vars-manager-setup";
import { VarsManager } from "./core/vars/vars-manager";
import { TasksDSL } from "./core/tasks/dsl";
import { getVarsFilePath } from "./util/global-dir";
import { getRequireCachedFiles } from "./util/platform";

export type GlobalWithHardhatContext = typeof global & {
  __hardhatContext: HardhatContext;
};

export class HardhatContext {
  constructor() {
    this.varsManager = new VarsManager(getVarsFilePath());
  }

  public static isCreated(): boolean {
    const globalWithHardhatContext = global as GlobalWithHardhatContext;
    return globalWithHardhatContext.__hardhatContext !== undefined;
  }

  public static createHardhatContext(): HardhatContext {
    if (this.isCreated()) {
      throw new HardhatError(ERRORS.GENERAL.CONTEXT_ALREADY_CREATED);
    }
    const globalWithHardhatContext = global as GlobalWithHardhatContext;
    const ctx = new HardhatContext();
    globalWithHardhatContext.__hardhatContext = ctx;
    return ctx;
  }

  public static getHardhatContext(): HardhatContext {
    const globalWithHardhatContext = global as GlobalWithHardhatContext;
    const ctx = globalWithHardhatContext.__hardhatContext;
    if (ctx === undefined) {
      throw new HardhatError(ERRORS.GENERAL.CONTEXT_NOT_CREATED);
    }
    return ctx;
  }

  public static deleteHardhatContext() {
    const globalAsAny = global as any;
    globalAsAny.__hardhatContext = undefined;
  }

  public readonly tasksDSL = new TasksDSL();
  public readonly environmentExtenders: EnvironmentExtender[] = [];
  public environment?: HardhatRuntimeEnvironment;
  public readonly providerExtenders: ProviderExtender[] = [];
  public varsManager: VarsManager | VarsManagerSetup;

  public readonly configExtenders: ConfigExtender[] = [];

  private _filesLoadedBeforeConfig?: string[];
  private _filesLoadedAfterConfig?: string[];

  public setHardhatRuntimeEnvironment(env: HardhatRuntimeEnvironment) {
    if (this.environment !== undefined) {
      throw new HardhatError(ERRORS.GENERAL.CONTEXT_HRE_ALREADY_DEFINED);
    }
    this.environment = env;
  }

  public getHardhatRuntimeEnvironment(): HardhatRuntimeEnvironment {
    if (this.environment === undefined) {
      throw new HardhatError(ERRORS.GENERAL.CONTEXT_HRE_NOT_DEFINED);
    }
    return this.environment;
  }

  public setConfigLoadingAsStarted() {
    this._filesLoadedBeforeConfig = getRequireCachedFiles();
  }

  public setConfigLoadingAsFinished() {
    this._filesLoadedAfterConfig = getRequireCachedFiles();
  }

  public getFilesLoadedDuringConfig(): string[] {
    // No config was loaded
    if (this._filesLoadedBeforeConfig === undefined) {
      return [];
    }

    assertHardhatInvariant(
      this._filesLoadedAfterConfig !== undefined,
      "Config loading was set as started and not finished"
    );

    return arraysDifference(
      this._filesLoadedAfterConfig,
      this._filesLoadedBeforeConfig
    );
  }
}

function arraysDifference<T>(a: T[], b: T[]): T[] {
  return a.filter((e) => !b.includes(e));
}
