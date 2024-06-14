"use strict";
var __createBinding = (this && this.__createBinding) || (Object.create ? (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    var desc = Object.getOwnPropertyDescriptor(m, k);
    if (!desc || ("get" in desc ? !m.__esModule : desc.writable || desc.configurable)) {
      desc = { enumerable: true, get: function() { return m[k]; } };
    }
    Object.defineProperty(o, k2, desc);
}) : (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    o[k2] = m[k];
}));
var __setModuleDefault = (this && this.__setModuleDefault) || (Object.create ? (function(o, v) {
    Object.defineProperty(o, "default", { enumerable: true, value: v });
}) : function(o, v) {
    o["default"] = v;
});
var __importStar = (this && this.__importStar) || function (mod) {
    if (mod && mod.__esModule) return mod;
    var result = {};
    if (mod != null) for (var k in mod) if (k !== "default" && Object.prototype.hasOwnProperty.call(mod, k)) __createBinding(result, mod, k);
    __setModuleDefault(result, mod);
    return result;
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.clearSnapshots = exports.loadFixture = void 0;
const errors_1 = require("./errors");
let snapshots = [];
/**
 * Useful in tests for setting up the desired state of the network.
 *
 * Executes the given function and takes a snapshot of the blockchain. Upon
 * subsequent calls to `loadFixture` with the same function, rather than
 * executing the function again, the blockchain will be restored to that
 * snapshot.
 *
 * _Warning_: don't use `loadFixture` with an anonymous function, otherwise the
 * function will be executed each time instead of using snapshots:
 *
 * - Correct usage: `loadFixture(deployTokens)`
 * - Incorrect usage: `loadFixture(async () => { ... })`
 */
async function loadFixture(fixture) {
    if (fixture.name === "") {
        throw new errors_1.FixtureAnonymousFunctionError();
    }
    const snapshot = snapshots.find((s) => s.fixture === fixture);
    const { takeSnapshot } = await Promise.resolve().then(() => __importStar(require("./helpers/takeSnapshot")));
    if (snapshot !== undefined) {
        try {
            await snapshot.restorer.restore();
            snapshots = snapshots.filter((s) => Number(s.restorer.snapshotId) <= Number(snapshot.restorer.snapshotId));
        }
        catch (e) {
            if (e instanceof errors_1.InvalidSnapshotError) {
                throw new errors_1.FixtureSnapshotError(e);
            }
            throw e;
        }
        return snapshot.data;
    }
    else {
        const data = await fixture();
        const restorer = await takeSnapshot();
        snapshots.push({
            restorer,
            fixture,
            data,
        });
        return data;
    }
}
exports.loadFixture = loadFixture;
/**
 * Clears every existing snapshot.
 */
async function clearSnapshots() {
    snapshots = [];
}
exports.clearSnapshots = clearSnapshots;
//# sourceMappingURL=loadFixture.js.map