const fs = require("fs");

task("deploy", "Deploys the contract")
  .addParam("unlockTime", "Timestamp after which the contract can be unlocked")
  .setAction(async (taskArgs, hre) => {
    const Contract = await hre.ethers.getContractFactory("Lock");
    const contract = await Contract.deploy(taskArgs.unlockTime);
    await contract.waitForDeployment();

    const address = contract.target;

    fs.writeFileSync("deployment-output.json", JSON.stringify({ address }));

    console.log("Deployed to:", address);
  });
