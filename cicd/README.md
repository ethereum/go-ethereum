# CI/CD pipeline for XDC
This directory contains CI/CD scripts used for each of the XDC environments.

## How to deploy more nodes
Adjust the number of variable `num_of_nodes` under file `.env`. (**Maximum supported is 58**)

## Devnet
Each PR merged into `dev-upgrade` will trigger below actions:
- Tests
- Terraform to apply infrascture changes(if any)
- Docker build of XDC with devnet configurations with tag of `:latest`
- Docker push to docker hub. https://hub.docker.com/repository/docker/xinfinorg/devnet
- Deployment of the latest XDC image(from above) to devnet run by AWS ECS

### First time set up an new environment
1. Pre-generate a list of node private keys in below format
```
{
  "xdc0": {
    "pk": {{PRIVATE KEY}}
  },
  "xdc1": {...},
  "xdc{{NUMBER}}: {...}
}
```
2. Access to aws console, create a bucket with name `terraform-devnet-bucket`
3. Upload the file from step 1 into the above bucket with name `node-config.json`
4. In order to allow pipeline able to push and deploy via ECR and ECS, we require below environment variables to be injected into the CI pipeline:
  1. DOCKER_USERNAME
  2. DOCKER_PASSWORD
  3. AWS_ACCESS_KEY_ID
  4. AWS_SECRET_ACCESS_KEY
  
You are all set!

## Testnet
*** WIP ***
Testnet release build are triggered by cutting a "pre-release" tag which matches the name of `TESTNET-{{release-version}}` from dev-upgrade or master branch.
An example can be found here: https://github.com/XinFinOrg/XDPoSChain/releases/tag/Testnet-v2.0.0
For more information, refer to github documentation on the release: https://docs.github.com/en/repositories/releasing-projects-on-github/about-releases

## Mainnet
*** WIP ***
Mainnet release are triggered by making a normal release tag with name starting with `v` (stands for version) from the master branch.