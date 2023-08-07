# Constants
This directory is used by dockerfile when builing the per environment constants.go
The benefit of this structure is to allow devnet, testnet and mainnet to have different constants configuration setup.

The default file under `common` directory is for the mainnet, whereas all files under this `constants` directory will override the `constants.go` when building image. For example, when building devnet images, we will do `ADD common/constants/constants.go.devnet /work/common/constants.go`