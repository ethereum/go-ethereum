export interface SnapshotRestorer {
    /**
     * Resets the state of the blockchain to the point in which the snapshot was
     * taken.
     */
    restore(): Promise<void>;
    snapshotId: string;
}
/**
 * Takes a snapshot of the state of the blockchain at the current block.
 *
 * Returns an object with a `restore` method that can be used to reset the
 * network to this state.
 */
export declare function takeSnapshot(): Promise<SnapshotRestorer>;
//# sourceMappingURL=takeSnapshot.d.ts.map