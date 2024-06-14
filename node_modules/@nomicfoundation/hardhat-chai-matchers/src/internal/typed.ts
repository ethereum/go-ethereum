export function tryDereference(value: any, type: string) {
  const { Typed } = require("ethers") as typeof import("ethers");
  try {
    return Typed.dereference(value, type);
  } catch {
    return undefined;
  }
}
