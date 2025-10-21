package bal

import (
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/holiman/uint256"
)

func (c *ContractCode) MarshalJSON() ([]byte, error) {
	hexStr := fmt.Sprintf("%x", *c)
	return json.Marshal(hexStr)
}
func (e encodingBalanceChange) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		TxIdx   string `json:"txIndex"`
		Balance *uint256.Int
	}{
		TxIdx:   fmt.Sprintf("0x%x", e.TxIdx),
		Balance: e.Balance,
	})
}

func (e *encodingBalanceChange) UnmarshalJSON(data []byte) error {
	aux := &struct {
		TxIdx   string `json:"txIndex"`
		Balance *uint256.Int
	}{
		Balance: e.Balance,
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if len(aux.TxIdx) >= 2 && aux.TxIdx[:2] == "0x" {
		if _, err := fmt.Sscanf(aux.TxIdx, "0x%x", &e.TxIdx); err != nil {
			return err
		}
	}
	return nil
}
func (e encodingAccountNonce) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		TxIdx string `json:"txIndex"`
		Nonce string `json:"nonce"`
	}{
		TxIdx: fmt.Sprintf("0x%x", e.TxIdx),
		Nonce: fmt.Sprintf("0x%x", e.Nonce),
	})
}

func (e *encodingAccountNonce) UnmarshalJSON(data []byte) error {
	aux := &struct {
		TxIdx string `json:"txIndex"`
		Nonce string `json:"nonce"`
	}{}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if len(aux.TxIdx) >= 2 && aux.TxIdx[:2] == "0x" {
		if _, err := fmt.Sscanf(aux.TxIdx, "0x%x", &e.TxIdx); err != nil {
			return err
		}
	}
	if len(aux.Nonce) >= 2 && aux.Nonce[:2] == "0x" {
		if _, err := fmt.Sscanf(aux.Nonce, "0x%x", &e.Nonce); err != nil {
			return err
		}
	}
	return nil
}

// UnmarshalJSON implements json.Unmarshaler to decode from RLP hex bytes
func (b *BlockAccessList) UnmarshalJSON(input []byte) error {
	// Handle both hex string and object formats
	var hexBytes hexutil.Bytes
	if err := json.Unmarshal(input, &hexBytes); err == nil {
		// It's a hex string, decode from RLP
		return rlp.DecodeBytes(hexBytes, b)
	}

	// Otherwise try to unmarshal as structured JSON
	var tmp []AccountAccess
	if err := json.Unmarshal(input, &tmp); err != nil {
		return err
	}
	*b = BlockAccessList(tmp)
	return nil
}

// MarshalJSON implements json.Marshaler to encode as RLP hex bytes
func (b BlockAccessList) MarshalJSON() ([]byte, error) {
	// Encode to RLP then to hex
	rlpBytes, err := rlp.EncodeToBytes(b)
	if err != nil {
		return nil, err
	}
	return json.Marshal(hexutil.Bytes(rlpBytes))
}

func (b BlockAccessList) String() string {
	aux := []AccountAccess{}
	for _, access := range b {
		aux = append(aux, access)
	}

	res, err := json.MarshalIndent(aux, "", "    ")
	if err != nil {
		panic(err)
	}

	return string(res)
}
