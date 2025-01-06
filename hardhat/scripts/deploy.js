async function main() {
  const [deployer] = await ethers.getSigners();
  console.log("Deploying contracts with the account:", deployer.address);

  const Token = await ethers.getContractFactory("Token");
  const token = await Token.deploy();

  await token.deployTransaction.wait();  // Wait for the transaction to be mined

  console.log("Token deployed to:", token.address);  // This should print the correct address
}

main()
  .then(() => process.exit(0))
  .catch((error) => {
    console.error(error);
    process.exit(1);
  });
