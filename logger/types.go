package logger

import (
	"time"
)

type utctime8601 struct{}

func (utctime8601) MarshalJSON() ([]byte, error) {
	// FIX This should be re-formated for proper ISO 8601
	return []byte(`"` + time.Now().UTC().Format(time.RFC3339Nano)[:26] + `Z"`), nil
}

type JsonLog interface {
	EventName() string
}

type LogEvent struct {
	Guid string      `json:"guid"`
	Ts   utctime8601 `json:"ts"`
	// Level string      `json:"level"`
}

type LogStarting struct {
	ClientString    string `json:"version_string"`
	Coinbase        string `json:"coinbase"`
	ProtocolVersion int    `json:"eth_version"`
	LogEvent
}

func (l *LogStarting) EventName() string {
	return "starting"
}

type P2PConnecting struct {
	RemoteId       string `json:"remote_id"`
	RemoteEndpoint string `json:"remote_endpoint"`
	NumConnections int    `json:"num_connections"`
	LogEvent
}

func (l *P2PConnecting) EventName() string {
	return "p2p.connecting"
}

type P2PConnected struct {
	RemoteId            string `json:"remote_id"`
	RemoteAddress       string `json:"remote_addr"`
	RemoteVersionString string `json:"remote_version_string"`
	NumConnections      int    `json:"num_connections"`
	LogEvent
}

func (l *P2PConnected) EventName() string {
	return "p2p.connected"
}

type P2PHandshaked struct {
	RemoteCapabilities []string `json:"remote_capabilities"`
	RemoteId           string   `json:"remote_id"`
	NumConnections     int      `json:"num_connections"`
	LogEvent
}

func (l *P2PHandshaked) EventName() string {
	return "p2p.handshaked"
}

type P2PDisconnected struct {
	NumConnections int    `json:"num_connections"`
	RemoteId       string `json:"remote_id"`
	LogEvent
}

func (l *P2PDisconnected) EventName() string {
	return "p2p.disconnected"
}

type P2PDisconnecting struct {
	Reason         string `json:"reason"`
	RemoteId       string `json:"remote_id"`
	NumConnections int    `json:"num_connections"`
	LogEvent
}

func (l *P2PDisconnecting) EventName() string {
	return "p2p.disconnecting"
}

type P2PDisconnectingBadHandshake struct {
	Reason         string `json:"reason"`
	RemoteId       string `json:"remote_id"`
	NumConnections int    `json:"num_connections"`
	LogEvent
}

func (l *P2PDisconnectingBadHandshake) EventName() string {
	return "p2p.disconnecting.bad_handshake"
}

type P2PDisconnectingBadProtocol struct {
	Reason         string `json:"reason"`
	RemoteId       string `json:"remote_id"`
	NumConnections int    `json:"num_connections"`
	LogEvent
}

func (l *P2PDisconnectingBadProtocol) EventName() string {
	return "p2p.disconnecting.bad_protocol"
}

type P2PDisconnectingReputation struct {
	Reason         string `json:"reason"`
	RemoteId       string `json:"remote_id"`
	NumConnections int    `json:"num_connections"`
	LogEvent
}

func (l *P2PDisconnectingReputation) EventName() string {
	return "p2p.disconnecting.reputation"
}

type P2PDisconnectingDHT struct {
	Reason         string `json:"reason"`
	RemoteId       string `json:"remote_id"`
	NumConnections int    `json:"num_connections"`
	LogEvent
}

func (l *P2PDisconnectingDHT) EventName() string {
	return "p2p.disconnecting.dht"
}

type P2PEthDisconnectingBadBlock struct {
	Reason         string `json:"reason"`
	RemoteId       string `json:"remote_id"`
	NumConnections int    `json:"num_connections"`
	LogEvent
}

func (l *P2PEthDisconnectingBadBlock) EventName() string {
	return "p2p.eth.disconnecting.bad_block"
}

type P2PEthDisconnectingBadTx struct {
	Reason         string `json:"reason"`
	RemoteId       string `json:"remote_id"`
	NumConnections int    `json:"num_connections"`
	LogEvent
}

func (l *P2PEthDisconnectingBadTx) EventName() string {
	return "p2p.eth.disconnecting.bad_tx"
}

type EthNewBlockMined struct {
	BlockNumber     int    `json:"block_number"`
	HeadHash        string `json:"head_hash"`
	BlockHash       string `json:"block_hash"`
	BlockHexRlp     string `json:"block_hexrlp"`
	BlockDifficulty int    `json:"block_difficulty"`
	BlockPrevHash   string `json:"block_prev_hash"`
	LogEvent
}

func (l *EthNewBlockMined) EventName() string {
	return "eth.newblock.mined"
}

type EthNewBlockBroadcasted struct {
	BlockNumber     int    `json:"block_number"`
	HeadHash        string `json:"head_hash"`
	BlockHash       string `json:"block_hash"`
	BlockDifficulty int    `json:"block_difficulty"`
	BlockPrevHash   string `json:"block_prev_hash"`
	LogEvent
}

func (l *EthNewBlockBroadcasted) EventName() string {
	return "eth.newblock.broadcasted"
}

type EthNewBlockReceived struct {
	BlockNumber     int    `json:"block_number"`
	HeadHash        string `json:"head_hash"`
	BlockHash       string `json:"block_hash"`
	BlockDifficulty int    `json:"block_difficulty"`
	BlockPrevHash   string `json:"block_prev_hash"`
	LogEvent
}

func (l *EthNewBlockReceived) EventName() string {
	return "eth.newblock.received"
}

type EthNewBlockIsKnown struct {
	BlockNumber     int    `json:"block_number"`
	HeadHash        string `json:"head_hash"`
	BlockHash       string `json:"block_hash"`
	BlockDifficulty int    `json:"block_difficulty"`
	BlockPrevHash   string `json:"block_prev_hash"`
	LogEvent
}

func (l *EthNewBlockIsKnown) EventName() string {
	return "eth.newblock.is_known"
}

type EthNewBlockIsNew struct {
	BlockNumber     int    `json:"block_number"`
	HeadHash        string `json:"head_hash"`
	BlockHash       string `json:"block_hash"`
	BlockDifficulty int    `json:"block_difficulty"`
	BlockPrevHash   string `json:"block_prev_hash"`
	LogEvent
}

func (l *EthNewBlockIsNew) EventName() string {
	return "eth.newblock.is_new"
}

type EthNewBlockMissingParent struct {
	BlockNumber     int    `json:"block_number"`
	HeadHash        string `json:"head_hash"`
	BlockHash       string `json:"block_hash"`
	BlockDifficulty int    `json:"block_difficulty"`
	BlockPrevHash   string `json:"block_prev_hash"`
	LogEvent
}

func (l *EthNewBlockMissingParent) EventName() string {
	return "eth.newblock.missing_parent"
}

type EthNewBlockIsInvalid struct {
	BlockNumber     int    `json:"block_number"`
	HeadHash        string `json:"head_hash"`
	BlockHash       string `json:"block_hash"`
	BlockDifficulty int    `json:"block_difficulty"`
	BlockPrevHash   string `json:"block_prev_hash"`
	LogEvent
}

func (l *EthNewBlockIsInvalid) EventName() string {
	return "eth.newblock.is_invalid"
}

type EthNewBlockChainIsOlder struct {
	BlockNumber     int    `json:"block_number"`
	HeadHash        string `json:"head_hash"`
	BlockHash       string `json:"block_hash"`
	BlockDifficulty int    `json:"block_difficulty"`
	BlockPrevHash   string `json:"block_prev_hash"`
	LogEvent
}

func (l *EthNewBlockChainIsOlder) EventName() string {
	return "eth.newblock.chain.is_older"
}

type EthNewBlockChainIsCanonical struct {
	BlockNumber     int    `json:"block_number"`
	HeadHash        string `json:"head_hash"`
	BlockHash       string `json:"block_hash"`
	BlockDifficulty int    `json:"block_difficulty"`
	BlockPrevHash   string `json:"block_prev_hash"`
	LogEvent
}

func (l *EthNewBlockChainIsCanonical) EventName() string {
	return "eth.newblock.chain.is_cannonical"
}

type EthNewBlockChainNotCanonical struct {
	BlockNumber     int    `json:"block_number"`
	HeadHash        string `json:"head_hash"`
	BlockHash       string `json:"block_hash"`
	BlockDifficulty int    `json:"block_difficulty"`
	BlockPrevHash   string `json:"block_prev_hash"`
	LogEvent
}

func (l *EthNewBlockChainNotCanonical) EventName() string {
	return "eth.newblock.chain.not_cannonical"
}

type EthNewBlockChainSwitched struct {
	BlockNumber     int    `json:"block_number"`
	HeadHash        string `json:"head_hash"`
	OldHeadHash     string `json:"old_head_hash"`
	BlockHash       string `json:"block_hash"`
	BlockDifficulty int    `json:"block_difficulty"`
	BlockPrevHash   string `json:"block_prev_hash"`
	LogEvent
}

func (l *EthNewBlockChainSwitched) EventName() string {
	return "eth.newblock.chain.switched"
}

type EthTxCreated struct {
	TxHash    string `json:"tx_hash"`
	TxSender  string `json:"tx_sender"`
	TxAddress string `json:"tx_address"`
	TxHexRLP  string `json:"tx_hexrlp"`
	TxNonce   int    `json:"tx_nonce"`
	LogEvent
}

func (l *EthTxCreated) EventName() string {
	return "eth.tx.created"
}

type EthTxReceived struct {
	TxHash    string `json:"tx_hash"`
	TxAddress string `json:"tx_address"`
	TxHexRLP  string `json:"tx_hexrlp"`
	RemoteId  string `json:"remote_id"`
	TxNonce   int    `json:"tx_nonce"`
	LogEvent
}

func (l *EthTxReceived) EventName() string {
	return "eth.tx.received"
}

type EthTxBroadcasted struct {
	TxHash    string `json:"tx_hash"`
	TxSender  string `json:"tx_sender"`
	TxAddress string `json:"tx_address"`
	TxNonce   int    `json:"tx_nonce"`
	LogEvent
}

func (l *EthTxBroadcasted) EventName() string {
	return "eth.tx.broadcasted"
}

type EthTxValidated struct {
	TxHash    string `json:"tx_hash"`
	TxSender  string `json:"tx_sender"`
	TxAddress string `json:"tx_address"`
	TxNonce   int    `json:"tx_nonce"`
	LogEvent
}

func (l *EthTxValidated) EventName() string {
	return "eth.tx.validated"
}

type EthTxIsInvalid struct {
	TxHash    string `json:"tx_hash"`
	TxSender  string `json:"tx_sender"`
	TxAddress string `json:"tx_address"`
	Reason    string `json:"reason"`
	TxNonce   int    `json:"tx_nonce"`
	LogEvent
}

func (l *EthTxIsInvalid) EventName() string {
	return "eth.tx.is_invalid"
}
