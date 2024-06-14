/* eslint-disable @typescript-eslint/prefer-function-type */

interface ErrorConstructor<T extends any[]> {
  new (...args: T): Error;
}

export function ensure<T extends any[]>(
  condition: boolean,
  ErrorToThrow: ErrorConstructor<T>,
  ...errorArgs: T
): asserts condition {
  if (!condition) {
    throw new ErrorToThrow(...errorArgs);
  }
}
