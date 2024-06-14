import { bytesToBigInt } from "@nomicfoundation/ethereumjs-util";
import { assertHardhatInvariant } from "../../core/errors";

const { rawDecode } = require("ethereumjs-abi");

// selector of Error(string)
const ERROR_SELECTOR = "08c379a0";
// selector of Panic(uint256)
const PANIC_SELECTOR = "4e487b71";

/**
 * Represents the returnData of a transaction, whose contents are unknown.
 */
export class ReturnData {
  private _selector: string | undefined;

  constructor(public value: Uint8Array) {
    if (value.length >= 4) {
      this._selector = Buffer.from(value.slice(0, 4)).toString("hex");
    }
  }

  public isEmpty(): boolean {
    return this.value.length === 0;
  }

  public matchesSelector(selector: Uint8Array): boolean {
    if (this._selector === undefined) {
      return false;
    }

    return this._selector === Buffer.from(selector).toString("hex");
  }

  public isErrorReturnData(): boolean {
    return this._selector === ERROR_SELECTOR;
  }

  public isPanicReturnData(): boolean {
    return this._selector === PANIC_SELECTOR;
  }

  public decodeError(): string {
    if (this.isEmpty()) {
      return "";
    }

    assertHardhatInvariant(
      this._selector === ERROR_SELECTOR,
      "Expected return data to be a Error(string)"
    );

    const [decodedError] = rawDecode(["string"], this.value.slice(4)) as [
      string
    ];

    return decodedError;
  }

  public decodePanic(): bigint {
    assertHardhatInvariant(
      this._selector === PANIC_SELECTOR,
      "Expected return data to be a Panic(uint256)"
    );

    // we are assuming that panic codes are smaller than Number.MAX_SAFE_INTEGER
    const errorCode = bytesToBigInt(this.value.slice(4));

    return errorCode;
  }

  public getSelector(): string | undefined {
    return this._selector;
  }
}
