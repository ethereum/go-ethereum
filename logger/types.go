package logger

import (
	"time"
)

type utctime8601 struct{}

func (utctime8601) MarshalJSON() ([]byte, error) {
	// FIX This should be re-formated for proper ISO 8601
	return []byte(`"` + time.Now().UTC().Format(time.RFC3339Nano)[:26] + `Z"`), nil
}

//"starting"
type LogStarting struct {
	ClientString    string      `json:"version_string"`
	Guid            string      `json:"guid"`
	Coinbase        string      `json:"coinbase"`
	ProtocolVersion int         `json:"eth_version"`
	Ts              utctime8601 `json:"ts"`
}

//"p2p.connecting"
type P2PConnecting struct {
	RemoteId       string      `json:"remote_id"`
	RemoteEndpoint string      `json:"remote_endpoint"`
	Guid           string      `json:"guid"`
	NumConnections int         `json:"num_connections"`
	Ts             utctime8601 `json:"ts"`
}

//"p2p.connected"
type P2PConnected struct {
	Guid           string      `json:"guid"`
	NumConnections int         `json:"num_connections"`
	RemoteId       string      `json:"remote_id"`
	Ts             utctime8601 `json:"ts"`
}

//"p2p.handshaked"
type P2PHandshaked struct {
	RemoteCapabilities []string `json:"remote_capabilities"`
	RemoteId           string   `json:"remote_id"`
	Guid               string   `json:"guid"`
	NumConnections     int      `json:"num_connections"`
	Ts                 string   `json:"ts"`
}

//"p2p.disconnected"
type P2PDisconnected struct {
	Guid           string      `json:"guid"`
	NumConnections int         `json:"num_connections"`
	RemoteId       string      `json:"remote_id"`
	Ts             utctime8601 `json:"ts"`
}

//"p2p.disconnecting"
type P2PDisconnecting struct {
	Reason         string      `json:"reason"`
	RemoteId       string      `json:"remote_id"`
	Guid           string      `json:"guid"`
	NumConnections int         `json:"num_connections"`
	Ts             utctime8601 `json:"ts"`
}

//"p2p.disconnecting.bad_handshake"
type P2PDisconnectingBadHandshake struct {
	Reason         string      `json:"reason"`
	RemoteId       string      `json:"remote_id"`
	Guid           string      `json:"guid"`
	NumConnections int         `json:"num_connections"`
	Ts             utctime8601 `json:"ts"`
}

//"p2p.disconnecting.bad_protocol"
type P2PDisconnectingBadProtocol struct {
	Reason         string      `json:"reason"`
	RemoteId       string      `json:"remote_id"`
	Guid           string      `json:"guid"`
	NumConnections int         `json:"num_connections"`
	Ts             utctime8601 `json:"ts"`
}

//"p2p.disconnecting.reputation"
type P2PDisconnectingReputation struct {
	Reason         string      `json:"reason"`
	RemoteId       string      `json:"remote_id"`
	Guid           string      `json:"guid"`
	NumConnections int         `json:"num_connections"`
	Ts             utctime8601 `json:"ts"`
}

//"p2p.disconnecting.dht"
type P2PDisconnectingDHT struct {
	Reason         string      `json:"reason"`
	RemoteId       string      `json:"remote_id"`
	Guid           string      `json:"guid"`
	NumConnections int         `json:"num_connections"`
	Ts             utctime8601 `json:"ts"`
}

//"p2p.eth.disconnecting.bad_block"
type P2PEthDisconnectingBadBlock struct {
	Reason         string      `json:"reason"`
	RemoteId       string      `json:"remote_id"`
	Guid           string      `json:"guid"`
	NumConnections int         `json:"num_connections"`
	Ts             utctime8601 `json:"ts"`
}

//"p2p.eth.disconnecting.bad_tx"
type P2PEthDisconnectingBadTx struct {
	Reason         string      `json:"reason"`
	RemoteId       string      `json:"remote_id"`
	Guid           string      `json:"guid"`
	NumConnections int         `json:"num_connections"`
	Ts             utctime8601 `json:"ts"`
}

//"eth.newblock.mined"
type EthNewBlockMined struct {
	BlockNumber     int         `json:"block_number"`
	HeadHash        string      `json:"head_hash"`
	BlockHash       string      `json:"block_hash"`
	BlockHexRlp     string      `json:"block_hexrlp"`
	BlockDifficulty int         `json:"block_difficulty"`
	Guid            string      `json:"guid"`
	BlockPrevHash   string      `json:"block_prev_hash"`
	Ts              utctime8601 `json:"ts"`
}

//"eth.newblock.broadcasted"
type EthNewBlockBroadcasted struct {
	BlockNumber     int         `json:"block_number"`
	HeadHash        string      `json:"head_hash"`
	BlockHash       string      `json:"block_hash"`
	BlockDifficulty int         `json:"block_difficulty"`
	Guid            string      `json:"guid"`
	BlockPrevHash   string      `json:"block_prev_hash"`
	Ts              utctime8601 `json:"ts"`
}

//"eth.newblock.received"
type EthNewBlockReceived struct {
	BlockNumber     int         `json:"block_number"`
	HeadHash        string      `json:"head_hash"`
	BlockHash       string      `json:"block_hash"`
	BlockDifficulty int         `json:"block_difficulty"`
	Guid            string      `json:"guid"`
	BlockPrevHash   string      `json:"block_prev_hash"`
	Ts              utctime8601 `json:"ts"`
}

//"eth.newblock.is_known"
type EthNewBlockIsKnown struct {
	BlockNumber     int         `json:"block_number"`
	HeadHash        string      `json:"head_hash"`
	BlockHash       string      `json:"block_hash"`
	BlockDifficulty int         `json:"block_difficulty"`
	Guid            string      `json:"guid"`
	BlockPrevHash   string      `json:"block_prev_hash"`
	Ts              utctime8601 `json:"ts"`
}

//"eth.newblock.is_new"
type EthNewBlockIsNew struct {
	BlockNumber     int         `json:"block_number"`
	HeadHash        string      `json:"head_hash"`
	BlockHash       string      `json:"block_hash"`
	BlockDifficulty int         `json:"block_difficulty"`
	Guid            string      `json:"guid"`
	BlockPrevHash   string      `json:"block_prev_hash"`
	Ts              utctime8601 `json:"ts"`
}

//"eth.newblock.missing_parent"
type EthNewBlockMissingParent struct {
	BlockNumber     int         `json:"block_number"`
	HeadHash        string      `json:"head_hash"`
	BlockHash       string      `json:"block_hash"`
	BlockDifficulty int         `json:"block_difficulty"`
	Guid            string      `json:"guid"`
	BlockPrevHash   string      `json:"block_prev_hash"`
	Ts              utctime8601 `json:"ts"`
}

//"eth.newblock.is_invalid"
type EthNewBlockIsInvalid struct {
	BlockNumber     int         `json:"block_number"`
	HeadHash        string      `json:"head_hash"`
	BlockHash       string      `json:"block_hash"`
	BlockDifficulty int         `json:"block_difficulty"`
	Guid            string      `json:"guid"`
	BlockPrevHash   string      `json:"block_prev_hash"`
	Ts              utctime8601 `json:"ts"`
}

//"eth.newblock.chain.is_older"
type EthNewBlockChainIsOlder struct {
	BlockNumber     int         `json:"block_number"`
	HeadHash        string      `json:"head_hash"`
	BlockHash       string      `json:"block_hash"`
	BlockDifficulty int         `json:"block_difficulty"`
	Guid            string      `json:"guid"`
	BlockPrevHash   string      `json:"block_prev_hash"`
	Ts              utctime8601 `json:"ts"`
}

//"eth.newblock.chain.is_cannonical"
type EthNewBlockChainIsCanonical struct {
	BlockNumber     int         `json:"block_number"`
	HeadHash        string      `json:"head_hash"`
	BlockHash       string      `json:"block_hash"`
	BlockDifficulty int         `json:"block_difficulty"`
	Guid            string      `json:"guid"`
	BlockPrevHash   string      `json:"block_prev_hash"`
	Ts              utctime8601 `json:"ts"`
}

//"eth.newblock.chain.not_cannonical"
type EthNewBlockChainNotCanonical struct {
	BlockNumber     int         `json:"block_number"`
	HeadHash        string      `json:"head_hash"`
	BlockHash       string      `json:"block_hash"`
	BlockDifficulty int         `json:"block_difficulty"`
	Guid            string      `json:"guid"`
	BlockPrevHash   string      `json:"block_prev_hash"`
	Ts              utctime8601 `json:"ts"`
}

//"eth.newblock.chain.switched"
type EthNewBlockChainSwitched struct {
	BlockNumber     int         `json:"block_number"`
	HeadHash        string      `json:"head_hash"`
	OldHeadHash     string      `json:"old_head_hash"`
	BlockHash       string      `json:"block_hash"`
	BlockDifficulty int         `json:"block_difficulty"`
	Guid            string      `json:"guid"`
	BlockPrevHash   string      `json:"block_prev_hash"`
	Ts              utctime8601 `json:"ts"`
}

//"eth.tx.created"
type EthTxCreated struct {
	TxHash    string      `json:"tx_hash"`
	TxSender  string      `json:"tx_sender"`
	TxAddress string      `json:"tx_address"`
	TxHexRLP  string      `json:"tx_hexrlp"`
	TxNonce   int         `json:"tx_nonce"`
	Guid      string      `json:"guid"`
	Ts        utctime8601 `json:"ts"`
}

//"eth.tx.received"
type EthTxReceived struct {
	TxHash    string      `json:"tx_hash"`
	TxAddress string      `json:"tx_address"`
	TxHexRLP  string      `json:"tx_hexrlp"`
	RemoteId  string      `json:"remote_id"`
	TxNonce   int         `json:"tx_nonce"`
	Guid      string      `json:"guid"`
	Ts        utctime8601 `json:"ts"`
}

//"eth.tx.broadcasted"
type EthTxBroadcasted struct {
	TxHash    string      `json:"tx_hash"`
	TxSender  string      `json:"tx_sender"`
	TxAddress string      `json:"tx_address"`
	TxNonce   int         `json:"tx_nonce"`
	Guid      string      `json:"guid"`
	Ts        utctime8601 `json:"ts"`
}

//"eth.tx.validated"
type EthTxValidated struct {
	TxHash    string      `json:"tx_hash"`
	TxSender  string      `json:"tx_sender"`
	TxAddress string      `json:"tx_address"`
	TxNonce   int         `json:"tx_nonce"`
	Guid      string      `json:"guid"`
	Ts        utctime8601 `json:"ts"`
}

//"eth.tx.is_invalid"
type EthTxIsInvalid struct {
	TxHash    string      `json:"tx_hash"`
	TxSender  string      `json:"tx_sender"`
	TxAddress string      `json:"tx_address"`
	Reason    string      `json:"reason"`
	TxNonce   int         `json:"tx_nonce"`
	Guid      string      `json:"guid"`
	Ts        utctime8601 `json:"ts"`
}
