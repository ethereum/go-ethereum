const _ = require("lodash");
const ethers = require("ethers");
const fs = require("fs");
const table = require("markdown-table");
const utils = require("./utils");
const util = require("util");

class CodeChecksReport {
  constructor(config) {
    this.config = config;
    this.increases = 0;
    this.decreases = 0;
    this.reportIsNew = true;
    this.success = true;

    this.previousData = config.previousData || { methods: {}, deployments: {} };
    this.newData = { methods: {}, deployments: {} };
  }

  /**
   * Generates a gas usage difference report for CodeCheck
   * @param  {Object} info   GasData instance with `methods` and `deployments` data
   */
  generate(info) {
    let highlightedDiff;
    let passFail;
    let alignment;
    const addedContracts = [];

    // ---------------------------------------------------------------------------------------------
    // Assemble section: Build Configuration
    // ---------------------------------------------------------------------------------------------
    let gwei = "-";
    let currency = "-";
    let rate = "-";

    const solc = utils.getSolcInfo(this.config.metadata);

    const token = this.config.token.toLowerCase();
    if (this.config.ethPrice && this.config.gasPrice) {
      gwei = `${parseInt(this.config.gasPrice)} gwei/gas`;
      currency = `${this.config.currency.toLowerCase()}`;
      rate = `${parseFloat(this.config.ethPrice).toFixed(
        2
      )} ${currency}/${token}`;
    }

    const configRows = [
      ["Option", "Settings"],
      ["solc: version", solc.version],
      ["solc: optimized", solc.optimizer],
      ["solc: runs", solc.runs],
      ["gas: block limit", ethers.utils.commify(info.blockLimit)],
      ["gas: price", gwei],
      [`gas: currency/${token} rate`, rate]
    ];

    const configTable = table(configRows);

    // ---------------------------------------------------------------------------------------------
    // Assemble section: methods
    // ---------------------------------------------------------------------------------------------

    const methodRows = [];
    const methodHeader = [
      " ",
      "Gas",
      " ",
      "Diff",
      "Diff %",
      "Calls",
      `${currency} avg`
    ];

    _.forEach(info.methods, (data, methodId) => {
      if (!data) return;

      let stats = {};

      if (data.gasData.length) {
        const total = data.gasData.reduce((acc, datum) => acc + datum, 0);
        stats.average = Math.round(total / data.gasData.length);

        stats.cost =
          this.config.ethPrice && this.config.gasPrice
            ? utils.gasToCost(
                stats.average,
                this.config.ethPrice,
                this.config.gasPrice
              )
            : "-";
      }

      stats.diff = this.getMethodDiff(methodId, stats.average);
      stats.percentDiff = this.getMethodPercentageDiff(methodId, stats.average);

      highlightedDiff = this.getHighlighting(stats.diff);
      passFail = this.getPassFail(stats.diff);

      if (data.numberOfCalls > 0) {
        // Contracts name row
        if (!addedContracts.includes(data.contract)) {
          addedContracts.push(data.contract);

          const titleSection = [
            this.entitle(data.contract),
            " ",
            " ",
            " ",
            " ",
            " ",
            " "
          ];
          titleSection.contractName = data.contract;
          titleSection.methodName = "0";
          methodRows.push(titleSection);
        }

        // Method row
        const methodSection = [
          this.indent(data.method),
          ethers.utils.commify(stats.average),
          passFail,
          highlightedDiff,
          stats.percentDiff,
          data.numberOfCalls.toString(),
          stats.cost.toString()
        ];
        methodSection.contractName = data.contract;
        methodSection.methodName = data.method;

        methodRows.push(methodSection);
        this.newData.methods[methodId] = stats.average;
      }
    });

    methodRows.sort((a, b) => {
      const contractName = a.contractName.localeCompare(b.contractName);
      const methodName = a.methodName.localeCompare(b.methodName);
      return contractName || methodName;
    });

    alignment = { align: ["l", "r", "c", "r", "r", "r", "r", "r"] };
    methodRows.unshift(methodHeader);
    const methodTable = table(methodRows, alignment);

    // ---------------------------------------------------------------------------------------------
    // Assemble section: deployments
    // ---------------------------------------------------------------------------------------------
    const deployRows = [];
    const deployHeader = [
      " ",
      "Gas",
      " ",
      "Diff",
      "Diff %",
      "Block %",
      `${currency} avg`
    ];

    // Alphabetize contract names
    info.deployments.sort((a, b) => a.name.localeCompare(b.name));

    info.deployments.forEach(contract => {
      let stats = {};
      if (!contract.gasData.length) return;

      const total = contract.gasData.reduce((acc, datum) => acc + datum, 0);
      stats.average = Math.round(total / contract.gasData.length);
      stats.percent = utils.gasToPercentOfLimit(stats.average, info.blockLimit);

      stats.cost =
        this.config.ethPrice && this.config.gasPrice
          ? utils.gasToCost(
              stats.average,
              this.config.ethPrice,
              this.config.gasPrice
            )
          : "-";

      stats.diff = this.getDeploymentDiff(contract.name, stats.average);
      stats.percentDiff = this.getDeploymentPercentageDiff(
        contract.name,
        stats.average
      );

      highlightedDiff = this.getHighlighting(stats.diff);
      passFail = this.getPassFail(stats.diff);

      const section = [
        this.entitle(contract.name),
        ethers.utils.commify(stats.average),
        passFail,
        highlightedDiff,
        stats.percentDiff,
        `${stats.percent} %`,
        stats.cost.toString()
      ];

      deployRows.push(section);
      this.newData.deployments[contract.name] = stats.average;
    });

    alignment = { align: ["l", "r", "c", "r", "r", "r", "r"] };
    deployRows.unshift(deployHeader);
    const deployTable = table(deployRows, alignment);

    // ---------------------------------------------------------------------------------------------
    // Final assembly
    // ---------------------------------------------------------------------------------------------

    const configTitle = "## Build Configuration\n";
    const methodTitle = "## Methods\n";
    const deployTitle = "## Deployments\n";

    const md =
      deployTitle +
      deployTable +
      `\n\n` +
      methodTitle +
      methodTable +
      `\n\n` +
      configTitle +
      configTable +
      `\n\n`;

    // ---------------------------------------------------------------------------------------------
    // Finish
    // ---------------------------------------------------------------------------------------------
    return md;
  }

  getDiff(previousVal, currentVal) {
    if (typeof previousVal === "number") {
      const diff = currentVal - previousVal;

      if (diff > 0) this.increases++;
      if (diff < 0) this.decreases++;

      this.reportIsNew = false;
      return diff;
    }
    return "-";
  }

  getPercentageDiff(previousVal, currentVal, maxThreshold) {
    let sign = "";

    if (typeof previousVal === "number") {
      const diff = Math.round(((currentVal - previousVal) / previousVal) * 100);

      if (diff > 0) {
        sign = "+";

        if (typeof maxThreshold === "number" && diff > maxThreshold) {
          this.success = false;
        }
      }

      return `${sign}${diff}%`;
    }
    return "-";
  }

  getMethodDiff(id, currentVal) {
    return this.getDiff(this.previousData.methods[id], currentVal);
  }

  getMethodPercentageDiff(id, currentVal) {
    return this.getPercentageDiff(
      this.previousData.methods[id],
      currentVal,
      this.config.maxMethodDiff
    );
  }

  getDeploymentDiff(id, currentVal) {
    return this.getDiff(this.previousData.deployments[id], currentVal);
  }

  getDeploymentPercentageDiff(id, currentVal) {
    return this.getPercentageDiff(
      this.previousData.deployments[id],
      currentVal,
      this.config.maxDeploymentDiff
    );
  }

  getPassFail(val) {
    const passed = `![passed](https://travis-ci.com/images/stroke-icons/icon-passed.png)`;
    const failed = `![failed](https://travis-ci.com/images/stroke-icons/icon-failed.png)`;

    if (val > 0) return failed;
    if (val < 0) return passed;
    return "";
  }

  getHighlighting(val) {
    if (val > 0) return `[**+${ethers.utils.commify(val)}**]()`;
    if (val < 0) return `[**${ethers.utils.commify(val)}**]()`;
    return val;
  }

  getShortDescription() {
    const increasesItem = this.increases === 1 ? "item" : "items";
    const decreasesItem = this.decreases === 1 ? "item" : "items";

    if (this.increases > 0 && this.decreases > 0) {
      return (
        `Gas usage increased for ${this.increases} ${increasesItem} and ` +
        `decreased for ${this.decreases} ${decreasesItem}`
      );
    } else if (this.increases > 0) {
      return `Gas usage increased for ${this.increases} ${increasesItem}`;
    } else if (this.decreases > 0) {
      return `Gas usage decreased for ${this.decreases} ${decreasesItem}`;
    } else if (this.reportIsNew) {
      return `New gas usage report!`;
    } else {
      return `Gas usage remained the same`;
    }
  }

  indent(val) {
    return `       *${val}*`;
  }

  entitle(val) {
    return `**${val}**`;
  }
}

module.exports = CodeChecksReport;
