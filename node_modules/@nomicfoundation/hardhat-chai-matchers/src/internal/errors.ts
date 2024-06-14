import { NomicLabsHardhatPluginError } from "hardhat/plugins";

export class HardhatChaiMatchersError extends NomicLabsHardhatPluginError {
  constructor(message: string, parent?: Error) {
    super("@nomicfoundation/hardhat-chai-matchers", message, parent);
  }
}

export class HardhatChaiMatchersDecodingError extends HardhatChaiMatchersError {
  constructor(encodedData: string, type: string, parent: Error) {
    const message = `There was an error decoding '${encodedData}' as a ${type}`;

    super(message, parent);
  }
}

/**
 * This class is used to assert assumptions in our implementation. Chai's
 * AssertionError should be used for user assertions.
 */
export class HardhatChaiMatchersAssertionError extends HardhatChaiMatchersError {
  constructor(message: string) {
    super(`Assertion error: ${message}`);
  }
}

export class HardhatChaiMatchersNonChainableMatcherError extends HardhatChaiMatchersError {
  constructor(matcherName: string, previousMatcherName: string) {
    super(
      `The matcher '${matcherName}' cannot be chained after '${previousMatcherName}'. For more information, please refer to the documentation at: https://hardhat.org/chaining-async-matchers.`
    );
  }
}
