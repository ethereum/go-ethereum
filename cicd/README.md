# CI/CD pipeline for XDC
This directory contains CI/CD scripts used for each of the XDC environments.

### Devnet
Each PR merged into `dev-upgrade` will trigger below actions:
- Tests
- Terraform to apply infrascture changes(if any)
- Docker build of XDC with devnet configurations with tag of `:latest`
- Docker push to docker hub. https://hub.docker.com/repository/docker/xinfinorg/devnet
- Deployment of the latest XDC image(from above) to devnet run by AWS ECS

In order to allow pipeline able to push and deploy via ECR and ECS, we require below environment variables to be injected into the CI pipeline:
1. ECR_REPO_NAME
2. ECR_BASE_URI
3. AWS_ACCESS_KEY_ID
4. AWS_SECRET_ACCESS_KEY

#### How to spin up more nodes in devnet
NOTE: The terraform managed auto deployment is still under deployment. The current best way to spin up new nodes is done by below:
1. `docker pull xinfinorg/devnet:latest`
2. `docker run -it -e PRIVATE_KEYS={{Wallet-Private-key-Here}} xinfinorg/devnet:latest`


### Testnet
**WIP**

### Mainnet
**WIP**