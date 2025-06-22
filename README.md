# Structure

- ./docker-compose.yaml - builds Geth and Blockscout exporter
- ./Dockerfile.contracts - builds docker image with contracts
- ./.github/workflows:
  - ``ci-build.yaml - CI:Build tag trigger``
  - ``cd-deploy.yaml - CI:Deploy tag trigger``
- ./hardhat - hardhat application
- ./infra - GCP K8s cluster and manifests
- ./blockscout - resources for blocksout stack deployment