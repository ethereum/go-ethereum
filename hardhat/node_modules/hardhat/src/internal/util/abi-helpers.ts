import * as abi from "@ethersproject/abi";

export class AbiHelpers {
  /**
   * Try to compute the selector for the function/event/error
   * with the given name and param types. Return undefined
   * if it cannot do it. This can happen if some ParamType is
   * not understood by @ethersproject/abi
   */
  public static computeSelector(
    name: string,
    inputs: any[]
  ): Uint8Array | undefined {
    try {
      const fragment = abi.FunctionFragment.from({
        type: "function",
        constant: true,
        name,
        inputs: inputs.map((i) => abi.ParamType.from(i)),
      });
      const selectorHex = abi.Interface.getSighash(fragment);

      return Buffer.from(selectorHex.slice(2), "hex");
    } catch {
      return;
    }
  }

  public static isValidCalldata(inputs: any[], calldata: Uint8Array): boolean {
    try {
      abi.defaultAbiCoder.decode(inputs, calldata);
      return true;
    } catch {
      return false;
    }
  }

  public static formatValues(values: any[]): string {
    return values.map((x) => AbiHelpers._formatValue(x)).join(", ");
  }

  private static _formatValue(value: any): string {
    // print nested values as [value1, value2, ...]
    if (Array.isArray(value)) {
      return `[${value.map((v) => AbiHelpers._formatValue(v)).join(", ")}]`;
    }

    // surround string values with quotes
    if (typeof value === "string") {
      return `"${value}"`;
    }

    return value.toString();
  }
}
