import type { SnapshotRestorer } from "./helpers/takeSnapshot";

import {
  FixtureAnonymousFunctionError,
  FixtureSnapshotError,
  InvalidSnapshotError,
} from "./errors";

type Fixture<T> = () => Promise<T>;

interface Snapshot<T> {
  restorer: SnapshotRestorer;
  fixture: Fixture<T>;
  data: T;
}

let snapshots: Array<Snapshot<any>> = [];

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
export async function loadFixture<T>(fixture: Fixture<T>): Promise<T> {
  if (fixture.name === "") {
    throw new FixtureAnonymousFunctionError();
  }

  const snapshot = snapshots.find((s) => s.fixture === fixture);

  const { takeSnapshot } = await import("./helpers/takeSnapshot");

  if (snapshot !== undefined) {
    try {
      await snapshot.restorer.restore();
      snapshots = snapshots.filter(
        (s) =>
          Number(s.restorer.snapshotId) <= Number(snapshot.restorer.snapshotId)
      );
    } catch (e) {
      if (e instanceof InvalidSnapshotError) {
        throw new FixtureSnapshotError(e);
      }

      throw e;
    }

    return snapshot.data;
  } else {
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

/**
 * Clears every existing snapshot.
 */
export async function clearSnapshots() {
  snapshots = [];
}
