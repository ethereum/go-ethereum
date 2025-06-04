'use strict'

/**
 * Record objects that pass by in a stream.  If the same object is used more
 * than once, it can be value-shared using shared values.
 *
 * @see {@link http://cbor.schmorp.de/value-sharing}
 */
class ObjectRecorder {
  constructor() {
    this.clear()
  }

  /**
   * Clear all of the objects that have been seen.  Revert to recording mode.
   */
  clear() {
    this.map = new WeakMap()
    this.count = 0
    this.recording = true
  }

  /**
   * Stop recording.
   */
  stop() {
    this.recording = false
  }

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
  check(obj) {
    const val = this.map.get(obj)
    if (val) {
      if (val.length > 1) {
        if (val[0] || this.recording) {
          return val[1]
        }

        val[0] = true
        return ObjectRecorder.FIRST
      }
      if (!this.recording) {
        return ObjectRecorder.NEVER
      }
      val.push(this.count++)
      // Second use while recording
      return val[1]
    }
    if (!this.recording) {
      throw new Error('New object detected when not recording')
    }
    this.map.set(obj, [false])
    // First use while recording
    return ObjectRecorder.NEVER
  }
}

ObjectRecorder.NEVER = -1
ObjectRecorder.FIRST = -2

module.exports = ObjectRecorder
