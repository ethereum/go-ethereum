import findup from "find-up";
import path from "path";

function findClosestPackageJson(file: string): string | null {
  return findup.sync("package.json", { cwd: path.dirname(file) });
}

/**
 * Returns the name of the closest package in the callstack that isn't this.
 */
export function getClosestCallerPackage(): string | undefined {
  const previousPrepareStackTrace = Error.prepareStackTrace;

  Error.prepareStackTrace = (e, s) => s;

  const error = new Error();
  const stack: NodeJS.CallSite[] = error.stack as any;

  Error.prepareStackTrace = previousPrepareStackTrace;

  const currentPackage = findClosestPackageJson(__filename)!;

  for (const callSite of stack) {
    const fileName = callSite.getFileName();
    // fileName is string | null in @types/node <=18
    // and string | undefined in @types/node 20
    if (
      fileName !== null &&
      fileName !== undefined &&
      path.isAbsolute(fileName)
    ) {
      const callerPackage = findClosestPackageJson(fileName);

      if (callerPackage === currentPackage) {
        continue;
      }

      if (callerPackage === null) {
        return undefined;
      }

      return require(callerPackage).name;
    }
  }

  return undefined;
}
