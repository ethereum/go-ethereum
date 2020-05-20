package lightclient

import (
	"github.com/tendermint/go-amino"
	cryptoAmino "github.com/tendermint/tendermint/crypto/encoding/amino"
)

type Codec = amino.Codec

var Cdc *Codec

func init() {
	cdc := amino.NewCodec()
	cryptoAmino.RegisterAmino(cdc)
	Cdc = cdc.Seal()
}
