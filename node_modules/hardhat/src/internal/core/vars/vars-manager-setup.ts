import debug from "debug";
import { VarsManager } from "./vars-manager";

const log = debug("hardhat:core:vars:varsManagerSetup");

/**
 * This class is ONLY used when collecting the required and optional vars that have to be filled by the user
 */
export class VarsManagerSetup extends VarsManager {
  private readonly _getVarsAlreadySet: Set<string>;
  private readonly _hasVarsAlreadySet: Set<string>;
  private readonly _getVarsWithDefaultValueAlreadySet: Set<string>;

  private readonly _getVarsToSet: Set<string>;
  private readonly _hasVarsToSet: Set<string>;
  private readonly _getVarsWithDefaultValueToSet: Set<string>;

  constructor(varsFilePath: string) {
    log("Creating a new instance of VarsManagerSetup");

    super(varsFilePath);

    this._getVarsAlreadySet = new Set();
    this._hasVarsAlreadySet = new Set();
    this._getVarsWithDefaultValueAlreadySet = new Set();

    this._getVarsToSet = new Set();
    this._hasVarsToSet = new Set();
    this._getVarsWithDefaultValueToSet = new Set();
  }

  // Checks if the key exists, and updates sets accordingly.
  // Ignore the parameter 'includeEnvs' defined in the parent class because during setup env vars are ignored.
  public has(key: string): boolean {
    log(`function 'has' called with key '${key}'`);

    const hasKey = super.has(key);

    if (hasKey) {
      this._hasVarsAlreadySet.add(key);
    } else {
      this._hasVarsToSet.add(key);
    }

    return hasKey;
  }

  // Gets the value for the provided key, and updates sets accordingly.
  // Ignore the parameter 'includeEnvs' defined in the parent class because during setup env vars are ignored.
  public get(key: string, defaultValue?: string): string {
    log(`function 'get' called with key '${key}'`);

    const varAlreadySet = super.has(key);

    if (varAlreadySet) {
      if (defaultValue !== undefined) {
        this._getVarsWithDefaultValueAlreadySet.add(key);
      } else {
        this._getVarsAlreadySet.add(key);
      }
    } else {
      if (defaultValue !== undefined) {
        this._getVarsWithDefaultValueToSet.add(key);
      } else {
        this._getVarsToSet.add(key);
      }
    }

    // Do not return undefined to avoid throwing an error
    return super.get(key, defaultValue) ?? "";
  }

  public getRequiredVarsAlreadySet(): string[] {
    return this._getRequired(this._getVarsAlreadySet, this._hasVarsAlreadySet);
  }

  public getOptionalVarsAlreadySet(): string[] {
    return this._getOptionals(
      this._getVarsAlreadySet,
      this._hasVarsAlreadySet,
      this._getVarsWithDefaultValueAlreadySet
    );
  }

  public getRequiredVarsToSet(): string[] {
    return this._getRequired(this._getVarsToSet, this._hasVarsToSet);
  }

  public getOptionalVarsToSet(): string[] {
    return this._getOptionals(
      this._getVarsToSet,
      this._hasVarsToSet,
      this._getVarsWithDefaultValueToSet
    );
  }

  // How to calculate required and optional variables:
  //
  // G = get function
  // H = has function
  // GD = get function with default value
  //
  // optional variables = H + (GD - G)
  // required variables = G - H
  private _getRequired(getVars: Set<string>, hasVars: Set<string>): string[] {
    return Array.from(getVars).filter((k) => !hasVars.has(k));
  }

  private _getOptionals(
    getVars: Set<string>,
    hasVars: Set<string>,
    getVarsWithDefault: Set<string>
  ): string[] {
    const result = new Set(hasVars);

    for (const k of getVarsWithDefault) {
      if (!getVars.has(k)) {
        result.add(k);
      }
    }

    return Array.from(result);
  }
}
