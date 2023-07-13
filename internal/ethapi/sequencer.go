package ethapi

import (
	"context"
	"fmt"

	client "github.com/astriaorg/go-sequencer-client"
	sqproto "github.com/astriaorg/go-sequencer-client/proto"
	"github.com/ethereum/go-ethereum/log"
)

const rollupChainID = "ethereum"

func sendTransactionToSequencer(ctx context.Context, txBytes []byte) error {
	signer, err := client.GenerateSigner()
	if err != nil {
		return err
	}

	// default tendermint RPC endpoint
	c, err := client.NewClient("http://localhost:26657")
	if err != nil {
		return err
	}

	tx := &sqproto.UnsignedTransaction{
		Nonce: 1,
		Actions: []*sqproto.Action{
			{
				Value: &sqproto.Action_SequenceAction{
					SequenceAction: &sqproto.SequenceAction{
						ChainId: []byte(rollupChainID),
						Data:    txBytes,
					},
				},
			},
		},
	}

	signed, err := signer.SignTransaction(tx)
	if err != nil {
		return err
	}

	resp, err := c.BroadcastTxSync(ctx, signed)
	if err != nil {
		return err
	}

	if resp.Code != 0 {
		return fmt.Errorf("failed to broadcast tx (error code %d): %s", resp.Code, resp.Log)
	}

	log.Info("successfully broadcasted tx to sequencer", "hash", resp.Hash)
	return nil
}
