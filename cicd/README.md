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
    "pk": {{PRIVATE KEY}},
    "address": {{XDC wallet address}},
    "imageTag": {{Optional field to run different version of XDC}},
    "logLevel": {{Optional field to adjust the log level for the container}}
  },
  "xdc1": {...},
  "xdc{{NUMBER}}: {...}
}
```
2. Access to aws console, create a bucket with name `tf-devnet-bucket`:
  - You can choose any name, just make sure update the name in the s3 bucket name variable in `variables.tf`
  - And update the name of the terraform.backend.s3.bucket from `s3.tf`
3. Upload the file from step 1 into the above bucket with name `node-config.json`
4. In order to allow pipeline able to push and deploy via ECR and ECS, we require below environment variables to be injected into the CI pipeline:
  1. DOCKER_USERNAME
  2. DOCKER_PASSWORD
  3. AWS_ACCESS_KEY_ID
  4. AWS_SECRET_ACCESS_KEY
  
You are all set!

## How to run different version of XDC on selected nodes
1. Create a new image tag:
  - Check out the repo
  - Run docker build `docker build -t xdc-devnet -f cicd/devnet/Dockerfile .`
  - Run docker tag `docker tag xdc-devnet:latest xinfinorg/devnet:test-{{put your version number here}}`
  - Run docker push `docker push xinfinorg/devnet:test-{{Version number from step above}}`
2. Adjust node-config.json
  - Download the node-config.json from s3
  - Add/update the `imageTag` field with value of `test-{{version number you defined in step 1}}` for the selected number of nodes you want to test with
  - Optional: Adjust the log level by add/updating the field of `logLevel`
  - Save and upload to s3
3. Make a dummy PR and get merged. Wait it to be updated.