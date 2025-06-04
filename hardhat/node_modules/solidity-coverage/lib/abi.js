const ethersABI = require("@ethersproject/abi");
const difflib = require('difflib');

class AbiUtils {

  diff(orig={}, cur={}){
    let plus = 0;
    let minus = 0;

    const unifiedDiff = difflib.unifiedDiff(
      orig.humanReadableAbiList,
      cur.humanReadableAbiList,
      {
        fromfile: orig.contractName,
        tofile: cur.contractName,
        fromfiledate: `sha: ${orig.sha}`,
        tofiledate: `sha: ${cur.sha}`,
        lineterm: ''
      }
    );

    // Count changes (unified diff always has a plus & minus in header);
    if (unifiedDiff.length){
      plus = -1;
      minus = -1;
    }

    unifiedDiff.forEach(line => {
      if (line[0] === `+`) plus++;
      if (line[0] === `-`) minus++;
    })

    return {
      plus,
      minus,
      unifiedDiff
    }
  }

  toHumanReadableFunctions(contract){
    const human = [];
    const ethersOutput = new ethersABI.Interface(contract.abi).functions;
    const signatures = Object.keys(ethersOutput);

    for (const sig of signatures){
      const method = ethersOutput[sig];
      let returns = '';

      method.outputs.forEach(output => {
        (returns.length)
          ? returns += `, ${output.type}`
          : returns += output.type;
      });

      let readable = `${method.type} ${sig} ${method.stateMutability}`;

      if (returns.length){
        readable += ` returns (${returns})`
      }

      human.push(readable);
    }

    return human;
  }

  toHumanReadableEvents(contract){
    const human = [];
    const ethersOutput = new ethersABI.Interface(contract.abi).events;
    const signatures = Object.keys(ethersOutput);

    for (const sig of signatures){
      const method = ethersOutput[sig];
      const readable = `${ethersOutput[sig].type} ${sig}`;
      human.push(readable);
    }

    return human;
  }

  generateHumanReadableAbiList(_artifacts, sha){
    const list = [];
    if (_artifacts.length){
      for (const item of _artifacts){
        const fns = this.toHumanReadableFunctions(item);
        const evts = this.toHumanReadableEvents(item);
        const all = fns.concat(evts);
        list.push({
          contractName: item.contractName,
          sha: sha,
          humanReadableAbiList: all
        })
      }
    }
    return list;
  }
}

module.exports = AbiUtils;