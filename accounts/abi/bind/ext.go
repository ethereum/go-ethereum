package bind

import (
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
)

type DeployOptions struct {
	ReceiptQueryInterval    time.Duration
	DeployRetryInterval     time.Duration
	ConfirmationInterval    time.Duration
	MaxReceiptQueryAttempts int
	MaxDeployAttempts       int
}

func DefaultDeployOptions() *DeployOptions {
	return &DeployOptions{
		ReceiptQueryInterval:    10000000000, // 10 sec
		DeployRetryInterval:     5000000000,  //  5 sec
		ConfirmationInterval:    60000000000, // 60 sec
		MaxReceiptQueryAttempts: 5,
		MaxDeployAttempts:       100,
	}
}

// implemented by eth.APIBackend
type Backend interface {
	GetTxReceipt(txhash common.Hash) (map[string]interface{}, error)
	BalanceAt(address common.Address) (*big.Int, error)
	CodeAt(address common.Address) (string, error)
	ContractBackend
}

func Deploy(deployF func(*TransactOpts, ContractBackend) (*types.Transaction, error), contractCode string, deployTransactor *TransactOpts, opt *DeployOptions, backend Backend) (contractAddr common.Address, err error) {

	deployRetryTimer := time.NewTimer(0).C
	var receiptQueryTimer <-chan time.Time
	var deployRetries, receiptQueries int
	var txhash common.Hash

DEPLOY:
	for {
		select {
		case <-deployRetryTimer:
			deployRetries++
			if deployRetries == opt.MaxDeployAttempts {
				return common.Address{}, fmt.Errorf("deployment failed...giving up after %v attempts", opt.MaxDeployAttempts)
			}
			tx, err := deployF(deployTransactor, backend)
			if err != nil {
				glog.V(logger.Warn).Infof("deployment failed: %v (attempt %v)", err, deployRetries)
				deployRetryTimer = time.NewTimer(opt.DeployRetryInterval).C
				continue DEPLOY
			}

			txhash = tx.Hash()
			deployRetryTimer = nil
			receiptQueryTimer = time.NewTimer(0).C

		case <-receiptQueryTimer:
			receipt, _ := backend.GetTxReceipt(txhash)
			receiptQueries++
			if receipt == nil {
				if receiptQueries == opt.MaxReceiptQueryAttempts {
					glog.V(logger.Warn).Infof("attempt %s deployment failed. Given up after %v attempts", opt.MaxReceiptQueryAttempts)

					deployRetryTimer = time.NewTimer(opt.DeployRetryInterval).C
					receiptQueryTimer = nil
					continue DEPLOY

				}
				glog.V(logger.Detail).Infof("new checkbook contract (txhash: %v) not yet mined... checking in %v", txhash.Hex(), opt.ReceiptQueryInterval)
				receiptQueryTimer = time.NewTimer(opt.ReceiptQueryInterval).C
				continue DEPLOY
			}

			contractAddr = receipt["contractAddress"].(common.Address)
			glog.V(logger.Detail).Infof("new chequebook contract mined at %v (owner: %v)", contractAddr.Hex(), deployTransactor.From.Hex())
			<-time.NewTimer(opt.ConfirmationInterval).C
			err = Validate(contractAddr, contractCode, backend)
			if err != nil {
				glog.V(logger.Warn).Infof("invalid contract at %v after %v: %v", contractAddr.Hex(), opt.ConfirmationInterval, err)
				deployRetryTimer = time.NewTimer(opt.DeployRetryInterval).C
				receiptQueryTimer = nil
				continue DEPLOY
			}
			break DEPLOY

		} // select
	} // for
	return contractAddr, nil
}

func Validate(contractAddr common.Address, expCode string, backend Backend) (err error) {
	if (contractAddr == common.Address{}) {
		return fmt.Errorf("zero address")
	}
	code, err := backend.CodeAt(contractAddr)
	if err != nil {
		return err
	}
	if len(expCode) > 0 && code != expCode {
		return fmt.Errorf("incorrect code %v:\n%v\n%v", contractAddr.Hex(), code, expCode)
	}
	return nil
}
