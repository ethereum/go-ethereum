export = SharedValueEncoder;
/**
 * Implement value sharing.
 *
 * @see {@link cbor.schmorp.de/value-sharing}
 */
declare class SharedValueEncoder extends Encoder {
    constructor(opts: any);
    valueSharing: ObjectRecorder;
    /**
     * Between encoding runs, stop recording, and start outputing correct tags.
     */
    stopRecording(): void;
    /**
     * Remove the existing recording and start over.  Do this between encoding
     * pairs.
     */
    clearRecording(): void;
}
import Encoder = require("./encoder");
import ObjectRecorder = require("./objectRecorder");
