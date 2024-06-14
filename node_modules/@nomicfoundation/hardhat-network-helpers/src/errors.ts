import { CustomError } from "hardhat/common";

export class HardhatNetworkHelpersError extends CustomError {
  constructor(message: string) {
    super(message);
  }
}

export class InvalidSnapshotError extends CustomError {
  constructor() {
    super("Trying to restore an invalid snapshot.");
  }
}

export class FixtureSnapshotError extends CustomError {
  constructor(parent: InvalidSnapshotError) {
    super(
      `There was an error reverting the snapshot of the fixture.

This might be caused by using hardhat_reset and loadFixture calls in a testcase.`,
      parent
    );
  }
}

export class FixtureAnonymousFunctionError extends CustomError {
  constructor() {
    super(`Anonymous functions cannot be used as fixtures.

You probably did something like this:

    loadFixture(async () => { ... });

Instead, define a fixture function and refer to that same function in each call to loadFixture.

Learn more at https://hardhat.org/hardhat-network-helpers/docs/reference#fixtures`);
  }
}

export class OnlyHardhatNetworkError extends CustomError {
  constructor(networkName: string, version?: string) {
    let errorMessage: string = ``;
    if (version === undefined) {
      errorMessage = `This helper can only be used with Hardhat Network. You are connected to '${networkName}'.`;
    } else {
      errorMessage = `This helper can only be used with Hardhat Network. You are connected to '${networkName}', whose identifier is '${version}'`;
    }

    super(errorMessage);
  }
}
