package simulation

import (
	"github.com/XinFinOrg/XDPoSChain/crypto"
	"math/big"
)

var (
	RpcEndpoint = "http://127.0.0.1:8501/"
	MinApply    = big.NewInt(0).Mul(big.NewInt(1000), big.NewInt(100000000000000000)) // 100 XDC
	Cap         = big.NewInt(0).Mul(big.NewInt(10000000000000), big.NewInt(10000000000000))
	Fee         = big.NewInt(100)

	MainKey, _ = crypto.HexToECDSA("65ec4d4dfbcac594a14c36baa462d6f73cd86134840f6cf7b80a1e1cd33473e2")
	MainAddr   = crypto.PubkeyToAddress(MainKey.PublicKey) //0x17F2beD710ba50Ed27aEa52fc4bD7Bda5ED4a037

	AirdropKey, _  = crypto.HexToECDSA("49a7b37aa6f6645917e7b807e9d1c00d4fa71f18343b0d4122a4d2df64dd6fee")
	AirdropAddr    = crypto.PubkeyToAddress(AirdropKey.PublicKey) // 0x0D3ab14BBaD3D99F4203bd7a11aCB94882050E7e
	AirDropAmount  = big.NewInt(10000000000)
	TransferAmount = big.NewInt(100000)

	ReceiverKey, _ = crypto.HexToECDSA("8a1f9a8f95be41cd7ccb6168179afb4504aefe388d1e14474d32c45c72ce7b7a")
	ReceiverAddr   = crypto.PubkeyToAddress(ReceiverKey.PublicKey) //0x703c4b2bD70c169f5717101CaeE543299Fc946C7
)
