export = ObjectRecorder;
/**
 * Record objects that pass by in a stream.  If the same object is used more
 * than once, it can be value-shared using shared values.
 *
 * @see {@link http://cbor.schmorp.de/value-sharing}
 */
declare class ObjectRecorder {
    /**
     * Clear all of the objects that have been seen.  Revert to recording mode.
     */
    clear(): void;
    map: WeakMap<object, any>;
    count: number;
    recording: boolean;
    /**
     * Stop recording.
     */
    stop(): void;
    /**
     * Determine if wrapping a tag 28 or 29 around an object that has been
     * reused is appropriate.  This method stores state for which objects have
     * been seen.
     *
     * @param {object} obj Any object about to be serialized.
     * @returns {number} If recording: -1 for first use, index for second use.
     *   If not recording, -1 for never-duplicated, -2 for first use, index for
     *   subsequent uses.
     * @throws {Error} Recording does not match playback.
     */
    check(obj: object): number;
}
declare namespace ObjectRecorder {
    const NEVER: number;
    const FIRST: number;
}
