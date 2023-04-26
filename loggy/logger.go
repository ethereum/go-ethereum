// PERI_AND_LATENCY_RECORDER_CODE_PIECE
package loggy

import (
	"fmt"
	"os"
	"path"
	"strconv"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

type MessageType int
type MessageDirection int

const (
	// Protocol messages belonging to eth/62
	StatusMsg          MessageType = 0x00
	NewBlockHashesMsg  MessageType = 0x01
	TxMsg              MessageType = 0x02
	GetBlockHeadersMsg MessageType = 0x03
	BlockHeadersMsg    MessageType = 0x04
	GetBlockBodiesMsg  MessageType = 0x05
	BlockBodiesMsg     MessageType = 0x06
	NewBlockMsg        MessageType = 0x07

	// Protocol messages belonging to eth/63
	ReqNodeDataMsg MessageType = 0x0d
	GetNodeDataMsg MessageType = 0x0e
	ReqReceiptsMsg MessageType = 0x0f
	GetReceiptsMsg MessageType = 0x10

	BannedPeerMsg MessageType = 0xf8
	MyTxMsg       MessageType = 0xf9
	PeerMsg       MessageType = 0xfa
	ObserveTxMsg  MessageType = 0xfb
	VictimTxMsg   MessageType = 0xfc
	PerigeeMsg    MessageType = 0xfd
	RemovePeerMsg MessageType = 0xfe
	Other         MessageType = 0xff
)

const (
	Inbound  MessageDirection = 0
	Outbound MessageDirection = 1
)

const MAXMEM = 2000

var txmap map[common.Hash]int64
var oldTxmap map[common.Hash]int64

var lastEpochStart int64
var Loggymutex sync.Mutex
var PeerBanMutex sync.Mutex
var Config *LoggyConfig

func init() {
	txmap = make(map[common.Hash]int64)
	oldTxmap = make(map[common.Hash]int64)
}

// updates epoch if needed and returns true if updated
func changeEpochIfNeeded() bool {
	//new epoch every EPOCH_DURATION
	if time.Now().Unix() >= (lastEpochStart + Config.EPOCH_DURATION) {
		lastEpochStart = time.Now().Unix()
		return true
	}
	return false
}

// returns the path of file to log to and if the epoch just changed
func GET_LOG_FILE(msgtype MessageType, msgdir MessageDirection) string {
	changeEpochIfNeeded()
	epoch := strconv.FormatInt(lastEpochStart, 10)

	if msgtype == StatusMsg {
		if msgdir == Outbound {
			fname := path.Join(Config.LOGS_BASEPATH, fmt.Sprintf("StatusMsg_out_%s.jsonl", epoch))
			return fname
		} else if msgdir == Inbound {
			fname := path.Join(Config.LOGS_BASEPATH, fmt.Sprintf("StatusMsg_in_%s.jsonl", epoch))
			return fname
		}
	}

	if msgtype == NewBlockHashesMsg {
		if msgdir == Outbound {
			fname := path.Join(Config.LOGS_BASEPATH, fmt.Sprintf("NewBlockHashesMsg_out_%s.jsonl", epoch))
			return fname
		} else if msgdir == Inbound {
			fname := path.Join(Config.LOGS_BASEPATH, fmt.Sprintf("NewBlockHashesMsg_in_%s.jsonl", epoch))
			return fname
		}
	}

	if msgtype == TxMsg {
		if msgdir == Outbound {
			fname := path.Join(Config.LOGS_BASEPATH, fmt.Sprintf("TxMsg_out_%s.jsonl", epoch))
			return fname
		} else if msgdir == Inbound {
			fname := path.Join(Config.LOGS_BASEPATH, fmt.Sprintf("TxMsg_in_%s.jsonl", epoch))
			return fname
		}
	}

	if msgtype == GetBlockHeadersMsg {
		if msgdir == Outbound {
			fname := path.Join(Config.LOGS_BASEPATH, fmt.Sprintf("GetBlockHeadersMsg_out_%s.jsonl", epoch))
			return fname
		} else if msgdir == Inbound {
			fname := path.Join(Config.LOGS_BASEPATH, fmt.Sprintf("GetBlockHeadersMsg_in_%s.jsonl", epoch))
			return fname
		}
	}

	if msgtype == BlockHeadersMsg {
		if msgdir == Outbound {
			fname := path.Join(Config.LOGS_BASEPATH, fmt.Sprintf("BlockHeadersMsg_out_%s.jsonl", epoch))
			return fname
		} else if msgdir == Inbound {
			fname := path.Join(Config.LOGS_BASEPATH, fmt.Sprintf("BlockHeadersMsg_in_%s.jsonl", epoch))
			return fname
		}
	}

	if msgtype == GetBlockBodiesMsg {
		if msgdir == Outbound {
			fname := path.Join(Config.LOGS_BASEPATH, fmt.Sprintf("GetBlockBodiesMsg_out_%s.jsonl", epoch))
			return fname
		} else if msgdir == Inbound {
			fname := path.Join(Config.LOGS_BASEPATH, fmt.Sprintf("GetBlockBodiesMsg_in_%s.jsonl", epoch))
			return fname
		}
	}

	if msgtype == BlockBodiesMsg {
		if msgdir == Outbound {
			fname := path.Join(Config.LOGS_BASEPATH, fmt.Sprintf("BlockBodiesMsg_out_%s.jsonl", epoch))
			return fname
		} else if msgdir == Inbound {
			fname := path.Join(Config.LOGS_BASEPATH, fmt.Sprintf("BlockBodiesMsg_in_%s.jsonl", epoch))
			return fname
		}
	}

	if msgtype == NewBlockMsg {
		if msgdir == Outbound {
			fname := path.Join(Config.LOGS_BASEPATH, fmt.Sprintf("NewBlockMsg_out_%s.jsonl", epoch))
			return fname
		} else if msgdir == Inbound {
			fname := path.Join(Config.LOGS_BASEPATH, fmt.Sprintf("NewBlockMsg_in_%s.jsonl", epoch))
			return fname
		}
	}

	if msgtype == GetNodeDataMsg {
		if msgdir == Outbound {
			fname := path.Join(Config.LOGS_BASEPATH, fmt.Sprintf("GetNodeDataMsg_out_%s.jsonl", epoch))
			return fname
		} else if msgdir == Inbound {
			fname := path.Join(Config.LOGS_BASEPATH, fmt.Sprintf("GetNodeDataMsg_in_%s.jsonl", epoch))
			return fname
		}
	}

	if msgtype == ReqNodeDataMsg {
		if msgdir == Outbound {
			fname := path.Join(Config.LOGS_BASEPATH, fmt.Sprintf("NodeDataMsg_out_%s.jsonl", epoch))
			return fname
		} else if msgdir == Inbound {
			fname := path.Join(Config.LOGS_BASEPATH, fmt.Sprintf("NodeDataMsg_in_%s.jsonl", epoch))
			return fname
		}
	}

	if msgtype == ReqReceiptsMsg {
		if msgdir == Outbound {
			fname := path.Join(Config.LOGS_BASEPATH, fmt.Sprintf("GetReceiptsMsg_out_%s.jsonl", epoch))
			return fname
		} else if msgdir == Inbound {
			fname := path.Join(Config.LOGS_BASEPATH, fmt.Sprintf("GetReceiptsMsg_in_%s.jsonl", epoch))
			return fname
		}
	}

	if msgtype == GetReceiptsMsg {
		if msgdir == Outbound {
			fname := path.Join(Config.LOGS_BASEPATH, fmt.Sprintf("ReceiptsMsg_out_%s.jsonl", epoch))
			return fname
		} else if msgdir == Inbound {
			fname := path.Join(Config.LOGS_BASEPATH, fmt.Sprintf("ReceiptsMsg_in_%s.jsonl", epoch))
			return fname
		}
	}

	if msgtype == RemovePeerMsg {
		fname := path.Join(Config.LOGS_BASEPATH, fmt.Sprintf("RemovePeer_%s.jsonl", epoch))
		return fname
	}

	if msgtype == PerigeeMsg {
		fname := path.Join(Config.LOGS_BASEPATH, fmt.Sprintf("Perigee_%s.jsonl", epoch))
		return fname
	}

	if msgtype == MyTxMsg {
		fname := path.Join(Config.LOGS_BASEPATH, fmt.Sprintf("MyTx_%s.jsonl", epoch))
		return fname
	}

	if msgtype == VictimTxMsg {
		fname := path.Join(Config.LOGS_BASEPATH, fmt.Sprintf("VictimTx_%s.jsonl", epoch))
		return fname
	}

	if msgtype == ObserveTxMsg {
		fname := path.Join(Config.LOGS_BASEPATH, fmt.Sprintf("AllTx_%s.jsonl", epoch))
		return fname
	}

	if msgtype == PeerMsg {
		fname := path.Join(Config.LOGS_BASEPATH, fmt.Sprintf("Peer_%s.jsonl", epoch))
		return fname
	}

	if msgtype == BannedPeerMsg {
		fname := path.Join(Config.LOGS_BASEPATH, fmt.Sprintf("BannedNodes_%s.jsonl", epoch))
		return fname
	}

	return (path.Join(Config.LOGS_BASEPATH, fmt.Sprintf("other_%s.txt", epoch)))
}

// assumes valid json is being passed to function as string
func Log(jsonstr string, msgtype MessageType, msgdir MessageDirection) {
	Loggymutex.Lock()
	defer Loggymutex.Unlock()

	fname := GET_LOG_FILE(msgtype, msgdir)
	f, err := os.OpenFile(fname, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)

	if err != nil {
		panic("Cannot create file " + err.Error())
	}

	defer f.Close()
	f.WriteString(jsonstr)
	f.WriteString("\n")
}

func LogBan(peerID string, reason string) {
	timestamp := time.Now().String()

	PeerBanMutex.Lock()
	defer PeerBanMutex.Unlock()

	fname := GET_LOG_FILE(BannedPeerMsg, Inbound)
	f, err := os.OpenFile(fname, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic("Cannot create file " + err.Error())
	}

	defer f.Close()
	f.WriteString(fmt.Sprintf("{\"enode\": \"%s\", \"timestamp\": \"%s\", \"reason\": \"%s\"}", peerID, timestamp, reason))
	f.WriteString("\n")
}

func ObserveGeneral(txHash common.Hash, enode string, timestamp int64) {
	if !Config.FlagObserve {
		return
	}

	Loggymutex.Lock()
	defer Loggymutex.Unlock()

	if _, seen := oldTxmap[txHash]; seen {
		return
	}

	if t0, seen := txmap[txHash]; !seen || t0 > timestamp {
		txmap[txHash] = timestamp
	} else if !Config.FlagAllTx {
		return
	}

	if len(txmap) >= MAXMEM {
		oldTxmap = txmap
		txmap = make(map[common.Hash]int64)
	}

	go Log(fmt.Sprintf(`{"hash":"%s","enode":"%s","time":%d,"isotime":"%s"}`, txHash.Hex(), enode, int64(timestamp), time.Unix(0, timestamp).String()), ObserveTxMsg, Inbound)
}

func Observe(txHash common.Hash, timestamp int64) {
	ObserveGeneral(txHash, "", timestamp)
}

func ObserveAll(txHash common.Hash, enode string, timestamp int64) {
	ObserveGeneral(txHash, enode, timestamp)
}
