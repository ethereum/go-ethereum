import fs from "fs-extra";
import debug from "debug";
import { HardhatError } from "../errors";
import { ERRORS } from "../errors-list";

interface Var {
  value: string;
}

interface VarsFile {
  _format: string; // Version of the json vars file
  vars: Record<string, Var>;
}

const log = debug("hardhat:core:vars:varsManager");

export class VarsManager {
  private readonly _VERSION = "hh-vars-1";
  private readonly _ENV_VAR_PREFIX = "HARDHAT_VAR_";
  private readonly _storageCache: VarsFile;
  private readonly _envCache: Record<string, string>;

  constructor(private readonly _varsFilePath: string) {
    log("Creating a new instance of VarsManager");

    this._initializeVarsFile();
    this._storageCache = fs.readJSONSync(this._varsFilePath);

    this._envCache = {};
    this._loadVarsFromEnv();
  }

  public getStoragePath(): string {
    return this._varsFilePath;
  }

  public set(key: string, value: string) {
    this.validateKey(key);

    if (value === "") {
      throw new HardhatError(ERRORS.VARS.INVALID_EMPTY_VALUE);
    }

    const vars = this._storageCache.vars;

    vars[key] = { value };
    this._writeStoredVars(vars);
  }

  public has(key: string, includeEnvs: boolean = false): boolean {
    if (includeEnvs && key in this._envCache) {
      return true;
    }

    return key in this._storageCache.vars;
  }

  public get(
    key: string,
    defaultValue?: string,
    includeEnvs: boolean = false
  ): string | undefined {
    if (includeEnvs && key in this._envCache) {
      return this._envCache[key];
    }

    return this._storageCache.vars[key]?.value ?? defaultValue;
  }

  public getEnvVars(): string[] {
    return Object.keys(this._envCache).map(
      (k) => `${this._ENV_VAR_PREFIX}${k}`
    );
  }

  public list(): string[] {
    return Object.keys(this._storageCache.vars);
  }

  public delete(key: string): boolean {
    const vars = this._storageCache.vars;

    if (vars[key] === undefined) return false;

    delete vars[key];
    this._writeStoredVars(vars);

    return true;
  }

  public validateKey(key: string) {
    const KEY_REGEX = /^[a-zA-Z_]+[a-zA-Z0-9_]*$/;

    if (!KEY_REGEX.test(key)) {
      throw new HardhatError(ERRORS.VARS.INVALID_CONFIG_VAR_NAME, {
        value: key,
      });
    }
  }

  private _initializeVarsFile() {
    if (!fs.pathExistsSync(this._varsFilePath)) {
      // Initialize the vars file if it does not exist
      log(
        `Vars file do not exist. Creating a new one at '${this._varsFilePath}' with version '${this._VERSION}'`
      );

      fs.writeJSONSync(this._varsFilePath, this._getVarsFileStructure(), {
        spaces: 2,
      });
    }
  }

  private _getVarsFileStructure(): VarsFile {
    return {
      _format: this._VERSION,
      vars: {},
    };
  }

  private _loadVarsFromEnv() {
    log("Loading ENV variables if any");

    for (const key in process.env) {
      if (key.startsWith(this._ENV_VAR_PREFIX)) {
        const envVar = process.env[key];

        if (
          envVar === undefined ||
          envVar.replace(/[\s\t]/g, "").length === 0
        ) {
          throw new HardhatError(ERRORS.ARGUMENTS.INVALID_ENV_VAR_VALUE, {
            varName: key,
            value: envVar!,
          });
        }

        const envKey = key.replace(this._ENV_VAR_PREFIX, "");
        this.validateKey(envKey);

        // Store only in cache, not in a file, as the vars are sourced from environment variables
        this._envCache[envKey] = envVar;
      }
    }
  }

  private _writeStoredVars(vars: Record<string, Var>) {
    // ENV variables are not stored in the file
    this._storageCache.vars = vars;
    fs.writeJSONSync(this._varsFilePath, this._storageCache, { spaces: 2 });
  }
}
