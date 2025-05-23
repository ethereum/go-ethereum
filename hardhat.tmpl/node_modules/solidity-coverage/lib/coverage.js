/**
 * Converts instrumentation data accumulated a the vm steps to an instanbul spec coverage object.
 * @type {Coverage}
 */

const util = require('util');

class Coverage {

  constructor() {
    this.data = {};
    this.requireData = {};
  }

  /**
   * Initializes an entry in the coverage map for an instrumented contract. Tracks by
   * its canonical contract path, e.g. *not* by its location in the temp folder.
   * @param {Object} info          'info = instrumenter.instrument(contract, fileName, true)'
   * @param {String} contractPath   canonical path to contract file
   */

  addContract(info, contractPath) {
    this.data[contractPath] = {
      l: {},
      path: contractPath,
      s: {},
      b: {},
      f: {},
      fnMap: {},
      statementMap: {},
      branchMap: {},
    };
    this.requireData[contractPath] = { };

    info.runnableLines.forEach((item, idx) => {
      this.data[contractPath].l[info.runnableLines[idx]] = 0;
    });

    this.data[contractPath].fnMap = info.fnMap;
    for (let x = 1; x <= Object.keys(info.fnMap).length; x++) {
      this.data[contractPath].f[x] = 0;
    }

    this.data[contractPath].branchMap = info.branchMap;
    for (let x = 1; x <= Object.keys(info.branchMap).length; x++) {
      this.data[contractPath].b[x] = [0, 0];
      this.requireData[contractPath][x] = {
        preEvents: 0,
        postEvents: 0,
      };
    }

    this.data[contractPath].statementMap = info.statementMap;
    for (let x = 1; x <= Object.keys(info.statementMap).length; x++) {
      this.data[contractPath].s[x] = 0;
    }
  }

  /**
   * Populates an empty coverage map with values derived from a hash map of
   * data collected as the instrumented contracts are tested
   * @param  {Object} map of collected instrumentation data
   * @return {Object} coverage map.
   */
  generate(collectedData) {
    const hashes = Object.keys(collectedData);

    for (let hash of hashes){
      const data = collectedData[hash];
      const contractPath = collectedData[hash].contractPath;
      const id = collectedData[hash].id;

      // NB: Any branch using the injected fn which returns boolean will have artificially
      // doubled hits (because of something internal to Solidity about how the stack is managed)
      const hits = collectedData[hash].hits;

      switch(collectedData[hash].type){
        case 'line':       this.data[contractPath].l[id] = hits;                   break;
        case 'function':   this.data[contractPath].f[id] = hits;                   break;
        case 'statement':  this.data[contractPath].s[id] = hits;                   break;
        case 'branch':     this.data[contractPath].b[id][data.locationIdx] = hits; break;
        case 'and-true':   this.data[contractPath].b[id][data.locationIdx] = hits; break;
        case 'or-false':   this.data[contractPath].b[id][data.locationIdx] = hits; break;
        case 'requirePre':  this.requireData[contractPath][id].preEvents = hits;   break;
        case 'requirePost': this.requireData[contractPath][id].postEvents = hits;  break;
      }
    }

    // Finally, interpret the assert pre/post events
    const contractPaths = Object.keys(this.requireData);

    for (let contractPath of contractPaths){
      const contract = this.data[contractPath];

      for (let i = 1; i <= Object.keys(contract.b).length; i++) {
        const branch = this.requireData[contractPath][i];

        // Was it an assert branch?
        if (branch && branch.preEvents > 0){
          this.data[contractPath].b[i] = [
            branch.postEvents,
            branch.preEvents - branch.postEvents
          ]
        }
      }
    }

    return Object.assign({}, this.data);
  }
};

module.exports = Coverage;
