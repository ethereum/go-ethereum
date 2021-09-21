package main

import (
	"context"
	"fmt"
	"github.com/XinFinOrg/XDPoSChain/accounts/abi/bind"
	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/contracts/trc21issuer"
	"github.com/XinFinOrg/XDPoSChain/contracts/trc21issuer/simulation"
	"github.com/XinFinOrg/XDPoSChain/ethclient"
	"log"
	"math/big"
	"time"
)

func main() {
	fmt.Println("========================")
	fmt.Println("mainAddr", simulation.MainAddr.Hex())
	fmt.Println("airdropAddr", simulation.AirdropAddr.Hex())
	fmt.Println("receiverAddr", simulation.ReceiverAddr.Hex())
	fmt.Println("========================")
	client, err := ethclient.Dial(simulation.RpcEndpoint)
	if err != nil {
		fmt.Println(err, client)
	}
	nonce, _ := client.NonceAt(context.Background(), simulation.MainAddr, nil)

	// init trc21 issuer
	auth := bind.NewKeyedTransactor(simulation.MainKey)
	auth.Nonce = big.NewInt(int64(nonce))
	auth.Value = big.NewInt(0)      // in wei
	auth.GasLimit = uint64(4000000) // in units
	auth.GasPrice = big.NewInt(210000000000000)
	trc21IssuerAddr, trc21Issuer, err := trc21issuer.DeployTRC21Issuer(auth, client, simulation.MinApply)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("main address", simulation.MainAddr.Hex(), "nonce", nonce)
	fmt.Println("===> trc21 issuer address", trc21IssuerAddr.Hex())

	auth.Nonce = big.NewInt(int64(nonce + 1))

	// init trc20
	trc21TokenAddr, trc21Token, err := trc21issuer.DeployTRC21(auth, client, "TEST", "XDC", 18, simulation.Cap, simulation.Fee)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("===>  trc21 token address", trc21TokenAddr.Hex(), "cap", simulation.Cap)

	fmt.Println("wait 10s to execute init smart contract")
	time.Sleep(10 * time.Second)

	trc21Issuer.TransactOpts.Nonce = big.NewInt(int64(nonce + 2))
	trc21Issuer.TransactOpts.Value = simulation.MinApply
	trc21Issuer.TransactOpts.GasPrice = big.NewInt(common.DefaultMinGasPrice)
	trc21Token.TransactOpts.GasPrice = big.NewInt(common.DefaultMinGasPrice)
	trc21Token.TransactOpts.GasLimit = uint64(4000000)
	auth.GasPrice = big.NewInt(common.DefaultMinGasPrice)
	// get balance init trc21 smart contract
	balance, err := trc21Token.BalanceOf(simulation.MainAddr)
	if err != nil || balance.Cmp(simulation.Cap) != 0 {
		log.Fatal(err, "\tget\t", balance, "\twant\t", simulation.Cap)
	}
	fmt.Println("balance", balance, "mainAddr", simulation.MainAddr.Hex())

	// add trc20 list token trc21 issuer
	_, err = trc21Issuer.Apply(trc21TokenAddr)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("wait 10s to add token to list issuer")
	time.Sleep(10 * time.Second)

	//check trc21 SMC balance
	balance, err = client.BalanceAt(context.Background(), trc21IssuerAddr, nil)
	if err != nil || balance.Cmp(simulation.MinApply) != 0 {
		log.Fatal("can't get balance  in trc21Issuer SMC: ", err, "got", balance, "wanted", simulation.MinApply)
	}

	//check balance fee
	balanceIssuerFee, err := trc21Issuer.GetTokenCapacity(trc21TokenAddr)
	if err != nil || balanceIssuerFee.Cmp(simulation.MinApply) != 0 {
		log.Fatal("can't get balance token fee in  smart contract: ", err, "got", balanceIssuerFee, "wanted", simulation.MinApply)
	}
}
