/**
 * Writes data from the VM step to the in-memory
 * coverage map constructed by the Instrumenter.
 */
class DataCollector {
  constructor(instrumentationData={}, viaIR){
    this.instrumentationData = instrumentationData;

    this.validOpcodes = this._getOpcodes(viaIR);
    this.lastHash = null;
    this.viaIR = viaIR;
    this.pcZeroCounter = 0;
    this.lastPcZeroCount = 0;
  }

  /**
   * VM step event handler. Detects instrumentation hashes when they are pushed to the
   * top of the stack. This runs millions of times - trying to keep it fast.
   * @param  {Object} info  vm step info
   */
  step(info){
    if (info.pc === 0) this.pcZeroCounter++;

    try {
      if (this.validOpcodes[info.opcode.name] && info.stack.length > 0){
        const idx = info.stack.length - 1;
        let hash = '0x' +  info.stack[idx].toString(16);
        this._registerHash(hash);
      }
    } catch (err) { /*Ignore*/ };
  }

  /**
   * Normalizes has string and marks hit.
   * @param  {String} hash bytes32 hash
   */
  _registerHash(hash){
    hash = this._normalizeHash(hash);

    if(this.instrumentationData[hash]){
      // abi.encode (used to circumvent viaIR) sometimes puts the hash on the stack twice
      // We should only skip duplicate hashes *within* a transaction (see issue #863)
      if (this.lastHash !== hash || this.lastPcZeroCount !== this.pcZeroCounter) {
        this.lastHash = hash;
        this.lastPcZeroCount = this.pcZeroCounter;
        this.instrumentationData[hash].hits++
      }
      return;
    }
  }

  /**
   * Left-pads zero prefixed bytes8 hashes to length 18. The '11' in the
   * comparison below is arbitrary. It provides a margin for recurring zeros
   * but prevents left-padding shorter irrelevant hashes
   *
   * @param  {String} hash  data hash from evm stack.
   * @return {String}       0x prefixed hash of length 18.
   */
  _normalizeHash(hash){
    // viaIR sometimes right-pads the hashes out to 32 bytes
    // but it doesn't preserve leading zeroes when it does this
    if (this.viaIR && hash.length >= 18) {
      hash = hash.slice(0,18);

      // Detect and recover from viaIR mangled hashes by left-padding single `0`
      if(!this.instrumentationData[hash]) {
        hash = hash.slice(2);
        hash = '0' + hash;
        hash = hash.slice(0,16);
        hash = '0x' + hash;
      }

    } else if (hash.length < 18 && hash.length > 11){
      hash = hash.slice(2);
      while(hash.length < 16) hash = '0' + hash;
      hash = '0x' + hash
    }
    return hash;
  }

  /**
   * Generates a list of all the opcodes to inspect for instrumentation hashes
   * When viaIR is true, it includes all DUPs and PUSHs, so things are a little slower.
   * @param {boolean} viaIR
   */
  _getOpcodes(viaIR) {
    let opcodes = {
      "PUSH1": true
    };

    if (!viaIR) return opcodes;

    for (let i = 2; i <= 32; i++) {
      const key = "PUSH" + i;
      opcodes[key] = viaIR;
    };

    for (let i = 1; i <= 16; i++ ) {
      const key = "DUP" + i;
      opcodes[key] = viaIR;
    }

    for (let i = 1; i <= 16; i++ ) {
      const key = "SWAP" + i;
      opcodes[key] = viaIR;
    }

    return opcodes;
  }

  /**
   * Unit test helper
   * @param {Object} data  Instrumenter.instrumentationData
   */
  _setInstrumentationData(data){
    this.instrumentationData = data;
  }
}

module.exports = DataCollector;
