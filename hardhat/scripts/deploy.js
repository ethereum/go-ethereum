async function main() {
  const [deployer] = await ethers.getSigners();
  console.log("Deploying contracts with the account:", deployer.address);

  const Token = await ethers.getContractFactory("Token");
  const token = await Token.deploy();

  // Log the token object to inspect its properties
  console.log("Token object:", token);

  if (token.deployTransaction) {
    console.log("Deploy transaction hash:", token.deployTransaction.hash);
  } else {
    console.error("Deploy transaction is undefined");
  }

  await token.deployed();

  console.log("Token deployed to:", token.address);
}

main()
  .then(() => process.exit(0))
  .catch((error) => {
    console.error(error);
    process.exit(1);
  });
