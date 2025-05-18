package main

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/holiman/uint256"
	"golang.org/x/sync/errgroup"
)

const (
	GREEN_DOT  = "\033[32m.\033[0m" // success
	RED_X      = "\033[31mx\033[0m" // expected failure case
	YELLOW_S   = "\033[33ms\033[0m" // unexpected failure, skip
	basePort   = 30000
	numWorkers = 10
	numTasks   = 20
)

type UnexpectedCodeError struct {
	Message string
}

func (e *UnexpectedCodeError) Error() string {
	return e.Message
}

func TestLag(t *testing.T) {
	t.Parallel()
	tasks := make(chan int, numTasks)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	errG, _ := errgroup.WithContext(ctx)
	for w := 0; w < numWorkers; w++ {
		workerID := w
		errG.Go(func() error {
			for taskID := range tasks {
				chErr := make(chan error, 1)
				task := &task{
					workerID: workerID,
					taskID:   taskID,
				}
				go func() {
					chErr <- task.run(t)
				}()

				select {
				case <-ctx.Done():
					t.Logf("Worker %d cancel task %d", workerID, taskID)
					task.cleanup(t)
					return nil

				case err := <-chErr:
					if err != nil {
						switch err.(type) {
						case *UnexpectedCodeError:
							fmt.Print(RED_X)
							cancel()
							t.Errorf("Worker %d failed on task %d with: %s", workerID, taskID, err.Error())
						default:
							fmt.Print(YELLOW_S, err)
						}
					} else {
						fmt.Print(GREEN_DOT)
					}
					return err
				}
			}
			return nil
		})
	}
	for i := range numTasks {
		tasks <- i
	}
	close(tasks)
	errG.Wait()
}

type task struct {
	workerID    int
	taskID      int
	testGethRef atomic.Pointer[testgeth]
}

func (task *task) cleanup(t *testing.T) {
	if g := task.testGethRef.Load(); g != nil && task.testGethRef.CompareAndSwap(g, nil) {
		t.Logf("Worker %d cleanup", task.workerID)
		g.Kill()
		g.WaitExit()
	}
}

func (task *task) run(t *testing.T) (returnErr error) {
	datadir := t.TempDir()
	port := basePort + task.workerID*100 + task.taskID
	args := []string{
		"--datadir",
		datadir, // passing in a `datadir` is required to reproduce
		"--dev",
		"--dev.period",
		"3",
		"--ws",
		"--ws.port",
		fmt.Sprintf("%d", port),
		"--ws.api",
		"admin,eth,web3,debug",
		"--ipcpath",
		filepath.Join(datadir, "geth.ipc"),
	}
	g := runGeth(t, args...)

	keydir := filepath.Join(datadir, "keystore")
	keyFiles, err := getFilesInKeystore(keydir)
	if err != nil {
		return err
	}
	if len(keyFiles) == 0 {
		return errors.New("no key files found")
	}
	keyJson, err := os.ReadFile(keyFiles[0])
	if err != nil {
		return err
	}
	ks := keystore.NewKeyStore(keydir, keystore.LightScryptN, keystore.LightScryptP)
	keyfileAcct := ks.Accounts()[0]
	password := ""
	key, err := keystore.DecryptKey(keyJson, password)
	if err != nil {
		return err
	}

	if !task.testGethRef.CompareAndSwap(nil, g) {
		// task cancelled
		go g.Kill()
		return nil
	}

	defer func() {
		task.cleanup(t)
	}()

	client, err := waitForClient(port, 20*time.Second)
	if err != nil {
		return err
	}
	defer client.Close()

	// fund new acct
	ctx := context.Background()
	value := big.NewInt(0).Exp(big.NewInt(10), big.NewInt(22), nil)
	gasLimit := uint64(21000)
	gasPrice, err := client.SuggestGasPrice(ctx)
	if err != nil {
		return err
	}

	privateKey, err := crypto.GenerateKey()
	if err != nil {
		return err
	}
	newAcct := crypto.PubkeyToAddress(privateKey.PublicKey)
	nonce, err := client.PendingNonceAt(ctx, keyfileAcct.Address)
	if err != nil {
		return err
	}
	tx := types.NewTransaction(nonce, newAcct, value, gasLimit, gasPrice, nil)
	chainID, err := client.ChainID(ctx)
	if err != nil {
		return err
	}
	if err := sendSignedTransaction(ctx, client, tx, key.PrivateKey, chainID); err != nil {
		return err
	}

	balance, err := client.BalanceAt(ctx, newAcct, nil)
	if err != nil {
		return err
	}

	if balance.Cmp(big.NewInt(0)) == 0 {
		return errors.New("no funding yet")
	}

	// send 7702 tx
	nonce, err = client.PendingNonceAt(ctx, newAcct)
	if err != nil {
		return err
	}

	signed, err := types.SignSetCode(privateKey, types.SetCodeAuthorization{
		Address: common.HexToAddress("0xdeadbeef00000000000000000000000000000000"),
		ChainID: *uint256.NewInt(chainID.Uint64()),
		Nonce:   nonce + 1,
	})
	if err != nil {
		return err
	}

	codeTx := &types.SetCodeTx{
		ChainID:    uint256.NewInt(chainID.Uint64()),
		Nonce:      nonce,
		Gas:        200000,
		To:         newAcct,
		Value:      uint256.NewInt(0),
		Data:       []byte("0x"),
		GasTipCap:  uint256.NewInt(10e11),
		GasFeeCap:  uint256.NewInt(10e11),
		AccessList: []types.AccessTuple{},
		AuthList:   []types.SetCodeAuthorization{signed},
	}
	tx = types.NewTx(codeTx)
	if err := sendSignedTransaction(ctx, client, tx, privateKey, chainID); err != nil {
		return err
	}

	// the default block id is "latest"
	code, err := client.CodeAt(ctx, newAcct, nil)
	if err != nil {
		return err
	}

	if !strings.HasPrefix(common.Bytes2Hex(code), "ef0100deadbeef") {
		return &UnexpectedCodeError{Message: "Code was not set!"}
	}

	codeTx.Nonce = nonce + 2
	signed, err = types.SignSetCode(privateKey, types.SetCodeAuthorization{
		Address: common.HexToAddress("0x0000000000000000000000000000000000000000"),
		ChainID: *uint256.NewInt(chainID.Uint64()),
		Nonce:   nonce + 3,
	})
	if err != nil {
		return err
	}

	codeTx.AuthList = []types.SetCodeAuthorization{signed}
	// clear code
	clearTx := types.NewTx(codeTx)
	if err := sendSignedTransaction(ctx, client, clearTx, privateKey, chainID); err != nil {
		return err
	}
	clearedCode, err := client.CodeAt(ctx, newAcct, nil)
	if err != nil {
		return err
	}
	if len(clearedCode) != 0 {
		return &UnexpectedCodeError{Message: "Code was not cleared!"}
	}
	return nil
}

func getFilesInKeystore(keyFolder string) ([]string, error) {
	var entries []os.DirEntry
	var err error

	maxRetries := 5
	retryDelay := 500 * time.Millisecond

	for i := 0; i < maxRetries; i++ {
		entries, err = os.ReadDir(keyFolder)
		if err == nil {
			break
		}
		time.Sleep(retryDelay)
	}

	if err != nil {
		return nil, err
	}

	var filePaths []string
	for _, entry := range entries {
		if !entry.IsDir() {
			filePaths = append(filePaths, filepath.Join(keyFolder, entry.Name()))
		}
	}

	return filePaths, nil
}

func waitForClient(port int, timeout time.Duration) (*ethclient.Client, error) {
	endpoint := fmt.Sprintf("ws://127.0.0.1:%d", port)
	start := time.Now()

	for time.Since(start) < timeout {
		client, err := ethclient.Dial(endpoint)
		if err == nil {
			return client, nil
		}
		time.Sleep(100 * time.Millisecond)
	}

	return nil, fmt.Errorf("failed to connect to client at %s within %s", endpoint, timeout)
}

func sendSignedTransaction(
	ctx context.Context,
	client *ethclient.Client,
	tx *types.Transaction,
	privateKey *ecdsa.PrivateKey,
	chainID *big.Int,
) error {
	signedTx, err := types.SignTx(tx, types.LatestSignerForChainID(chainID), privateKey)
	if err != nil {
		return err
	}

	if err := client.SendTransaction(ctx, signedTx); err != nil {
		return err
	}

	for {
		_, err := client.TransactionReceipt(ctx, signedTx.Hash())
		if err == nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	return nil
}
