async function main() {
  const [deployer] = await ethers.getSigners();
  console.log("Deploying contracts with the account:", deployer.address);

  const Token = await ethers.getContractFactory("Token");
  const token = await Token.deploy();

  await token.deployed();  // Ensure the contract is deployed before accessing its address

  console.log("Token deployed to:", token.address);  // This should print the correct address
}

main()
  .then(() => process.exit(0))
  .catch((error) => {
    console.error(error);
    process.exit(1);
  });
