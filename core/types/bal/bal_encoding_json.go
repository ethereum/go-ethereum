package bal

import (
	"encoding/json"
	"fmt"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rlp"
)

func (c *ContractCode) MarshalJSON() ([]byte, error) {
	hexStr := fmt.Sprintf("%x", *c)
	return json.Marshal(hexStr)
}
func (e encodingBalanceChange) MarshalJSON() ([]byte, error) {
	type Alias encodingBalanceChange
	return json.Marshal(&struct {
		TxIdx string `json:"txIndex"`
		*Alias
	}{
		TxIdx: fmt.Sprintf("0x%x", e.TxIdx),
		Alias: (*Alias)(&e),
	})
}

func (e *encodingBalanceChange) UnmarshalJSON(data []byte) error {
	type Alias encodingBalanceChange
	aux := &struct {
		TxIdx string `json:"txIndex"`
		*Alias
	}{
		Alias: (*Alias)(e),
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
	type Alias encodingAccountNonce
	return json.Marshal(&struct {
		TxIdx string `json:"txIndex"`
		Nonce string `json:"nonce"`
		*Alias
	}{
		TxIdx: fmt.Sprintf("0x%x", e.TxIdx),
		Nonce: fmt.Sprintf("0x%x", e.Nonce),
		Alias: (*Alias)(&e),
	})
}

func (e *encodingAccountNonce) UnmarshalJSON(data []byte) error {
	type Alias encodingAccountNonce
	aux := &struct {
		TxIdx string `json:"txIndex"`
		Nonce string `json:"nonce"`
		*Alias
	}{
		Alias: (*Alias)(e),
	}
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
