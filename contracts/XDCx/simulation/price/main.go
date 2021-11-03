package main

import (
	"context"
	"fmt"
	"github.com/XinFinOrg/XDPoSChain/XDCxlending/lendingstate"
	"github.com/XinFinOrg/XDPoSChain/core/state"
	"math/big"
	"os"
	"time"

	"github.com/XinFinOrg/XDPoSChain/accounts/abi/bind"
	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/contracts/XDCx"
	"github.com/XinFinOrg/XDPoSChain/crypto"
	"github.com/XinFinOrg/XDPoSChain/ethclient"
)

func main() {
	client, err := ethclient.Dial("http://127.0.0.1:8501/")
	if err != nil {
		fmt.Println(err, client)
	}
	MainKey, _ := crypto.HexToECDSA(os.Getenv("OWNER_KEY"))
	MainAddr := crypto.PubkeyToAddress(MainKey.PublicKey)

	nonce, _ := client.NonceAt(context.Background(), MainAddr, nil)
	auth := bind.NewKeyedTransactor(MainKey)
	auth.Value = big.NewInt(0)      // in wei
	auth.GasLimit = uint64(4000000) // in units
	auth.GasPrice = big.NewInt(250000000000000)

	// init trc21 issuer
	auth.Nonce = big.NewInt(int64(nonce))

	price := new(big.Int)
	price.SetString(os.Getenv("PRICE"), 10)

	lendContract, _ := XDCx.NewLendingRelayerRegistration(auth, common.HexToAddress(os.Getenv("LENDING_ADDRESS")), client)

	token := common.HexToAddress(os.Getenv("TOKEN_ADDRESS"))
	lendingToken := common.HexToAddress(os.Getenv("LENDING_TOKEN_ADDRESS"))

	tx, err := lendContract.SetCollateralPrice(token, lendingToken, price)
	if err != nil {
		fmt.Println("Set price failed!", err)
	}

	time.Sleep(5 * time.Second)
	r, err := client.TransactionReceipt(context.Background(), tx.Hash())
	if err != nil {
		fmt.Println("Get receipt failed", err)
	}
	fmt.Println("Done receipt status", r.Status)

	collateralState := state.GetLocMappingAtKey(token.Hash(), lendingstate.CollateralMapSlot)
	locMapPrices := collateralState.Add(collateralState, lendingstate.CollateralStructSlots["price"])
	locLendingTokenPriceByte := crypto.Keccak256(lendingToken.Hash().Bytes(), common.BigToHash(locMapPrices).Bytes())

	locCollateralPrice := common.BigToHash(new(big.Int).Add(new(big.Int).SetBytes(locLendingTokenPriceByte), lendingstate.PriceStructSlots["price"]))
	locBlockNumber := common.BigToHash(new(big.Int).Add(new(big.Int).SetBytes(locLendingTokenPriceByte), lendingstate.PriceStructSlots["blockNumber"]))

	priceByte, err := client.StorageAt(context.Background(), common.HexToAddress(os.Getenv("LENDING_ADDRESS")), locCollateralPrice, nil)
	fmt.Println(new(big.Int).SetBytes(priceByte), err)
	blockNumberByte, err := client.StorageAt(context.Background(), common.HexToAddress(os.Getenv("LENDING_ADDRESS")), locBlockNumber, nil)
	fmt.Println(new(big.Int).SetBytes(blockNumberByte), err)
}
