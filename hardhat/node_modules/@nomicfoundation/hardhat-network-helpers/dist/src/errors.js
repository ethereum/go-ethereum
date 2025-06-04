"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.OnlyHardhatNetworkError = exports.FixtureAnonymousFunctionError = exports.FixtureSnapshotError = exports.InvalidSnapshotError = exports.HardhatNetworkHelpersError = void 0;
const common_1 = require("hardhat/common");
class HardhatNetworkHelpersError extends common_1.CustomError {
    constructor(message) {
        super(message);
    }
}
exports.HardhatNetworkHelpersError = HardhatNetworkHelpersError;
class InvalidSnapshotError extends common_1.CustomError {
    constructor() {
        super("Trying to restore an invalid snapshot.");
    }
}
exports.InvalidSnapshotError = InvalidSnapshotError;
class FixtureSnapshotError extends common_1.CustomError {
    constructor(parent) {
        super(`There was an error reverting the snapshot of the fixture.

This might be caused by using hardhat_reset and loadFixture calls in a testcase.`, parent);
    }
}
exports.FixtureSnapshotError = FixtureSnapshotError;
class FixtureAnonymousFunctionError extends common_1.CustomError {
    constructor() {
        super(`Anonymous functions cannot be used as fixtures.

You probably did something like this:

    loadFixture(async () => { ... });

Instead, define a fixture function and refer to that same function in each call to loadFixture.

Learn more at https://hardhat.org/hardhat-network-helpers/docs/reference#fixtures`);
    }
}
exports.FixtureAnonymousFunctionError = FixtureAnonymousFunctionError;
class OnlyHardhatNetworkError extends common_1.CustomError {
    constructor(networkName, version) {
        let errorMessage = ``;
        if (version === undefined) {
            errorMessage = `This helper can only be used with Hardhat Network. You are connected to '${networkName}'.`;
        }
        else {
            errorMessage = `This helper can only be used with Hardhat Network. You are connected to '${networkName}', whose identifier is '${version}'`;
        }
        super(errorMessage);
    }
}
exports.OnlyHardhatNetworkError = OnlyHardhatNetworkError;
//# sourceMappingURL=errors.js.map