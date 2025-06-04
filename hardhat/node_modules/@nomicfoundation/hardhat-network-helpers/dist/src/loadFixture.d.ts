type Fixture<T> = () => Promise<T>;
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
export declare function loadFixture<T>(fixture: Fixture<T>): Promise<T>;
/**
 * Clears every existing snapshot.
 */
export declare function clearSnapshots(): Promise<void>;
export {};
//# sourceMappingURL=loadFixture.d.ts.map