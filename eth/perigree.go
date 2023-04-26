package eth

import (
	"fmt"
	"math"
	"math/big"
	"math/rand"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/log"
    "github.com/ethereum/go-ethereum/loggy"
)

const (
	MaxOldArrivals = 100000
	ArrivalReplace = 30000
	NanoTranslator = 1000000000.0
	Milli2Nano     = 1000000
	QuantNum       = 10
)

var (
	h              = &handler{}
	PeriConfig     = &ethconfig.PeriConfig{}
	oldArrivals    = make(map[common.Hash]int64)
	arrivals       = make(map[common.Hash]int64)
	arrivalPerPeer = make(map[common.Hash]map[string]int64)
	targetTx       = make(map[common.Hash]bool)

	tempUnregistration = make(map[string]bool)

	blocklist = make(map[string]bool)

	mapsMutex    = sync.Mutex{}
	tempMapMutex = sync.Mutex{}

	snapshot = make(map[string]string)

	MyAccount = make([]string, 0)
)

type txArrival struct {
	tx      common.Hash
	arrival int64
}

type idScore struct {
	id    string
	score float64
}

func init() {

}

// Start Peri (at the initialization of geth)
func StartPeri(pcfg *ethconfig.PeriConfig, hp *handler) {
	h = hp
	PeriConfig = pcfg
	ticker := time.NewTicker(time.Second * time.Duration(pcfg.Period))

	for {
		log.Warn("New Perigee period")
		<-ticker.C
		disconnectByScore()
	}
}

// Disconnect peers with highest latency score
func disconnectByScore() {
	mapsMutex.Lock()
	defer mapsMutex.Unlock()

	scores, excused := getScores()
	numScores := len(scores)

	// number of peers to drop
	numDrop := int(math.Round(float64(h.maxPeers)*PeriConfig.ReplaceRatio)) + numScores - h.maxPeers
	if numDrop < 0 {
		numDrop = 0
	}

	// Check if there is no tx recorded. If so, everyone is excused and skip dropping peers
	flagReplace := showStats(scores, excused, numDrop)
	if !flagReplace {
		log.Warn(fmt.Sprintf("No tx recorded. Perigee skipped. Current peerCount = %d", h.peers.len()))
		return
	}

	log.Warn(fmt.Sprintf("peerCount before dropping = %d", h.peers.len()))

	if !PeriConfig.Active { // Peri is inactive, drop randomly instead; Set ReplaceRatio=0 to disable dropping
		indices := make([]int, len(scores))
		for i := 0; i < len(indices); i++ {
			indices[i] = i
		}
		rand.Shuffle(len(indices), func(i, j int) {
			indices[i], indices[j] = indices[j], indices[i]
		})
		for i := 0; i < numDrop; i++ {
			id := scores[indices[i]].id
			h.removePeer(id)
			h.unregisterPeer(id)
			tempMapMutex.Lock()
			tempUnregistration[id] = true
			tempMapMutex.Unlock()
		}
	} else { // Drop nodes, and add them to the blocklist
		for i := 0; i < numDrop; i++ {
			id := scores[i].id
			if _, isExcused := excused[id]; isExcused {
				continue
			}
			blocklist[extractIPFromEnode(snapshot[id])] = true
			h.removePeer(id)
			h.unregisterPeer(id)
			tempMapMutex.Lock()
			tempUnregistration[id] = true
			tempMapMutex.Unlock()
		}
	}
	log.Warn(fmt.Sprintf("peerCount after dropping = %d", h.peers.len()))
	resetMaps()
}

// Lock is assumed to be held;
func resetMaps() {
	for tx, arrival := range arrivals {
		oldArrivals[tx] = arrival
	}

	// Clear old arrival states which are assumed not to be forwarded anymore
	if len(oldArrivals) > MaxOldArrivals {
		i := 0
		listArrivals := make([]txArrival, len(oldArrivals))
		for tx, arrival := range oldArrivals {
			listArrivals[i] = txArrival{tx, arrival}
			i++
		}

		// Sort arrival time by ascending order
		sort.Slice(listArrivals, func(i, j int) bool {
			return listArrivals[i].arrival < listArrivals[j].arrival
		})

		// Delete the earliest arrivals
		for i := 0; i < ArrivalReplace; i++ {
			delete(oldArrivals, listArrivals[i].tx)
		}
	}

	// Reset arrival states
	arrivals = make(map[common.Hash]int64)
	arrivalPerPeer = make(map[common.Hash]map[string]int64)
	targetTx = make(map[common.Hash]bool)
	snapshot = make(map[string]string)

	timer := time.NewTimer(time.Millisecond * 500)
	go func() {
		<-timer.C
		tempMapMutex.Lock()
		tempUnregistration = make(map[string]bool)
		tempMapMutex.Unlock()
		log.Warn(("temporary unregistration map cleared"))
	}()
}

// Generate a score report at the end of Peri period; Write them to log files
func showStats(scores []idScore, excused map[string]bool, numReplace int) bool {
	now := time.Now().String()
	absNow := mclock.Now()

	log.Warn(fmt.Sprintf("Perigee triggered at %s", now))

	numTx, numPeer := len(arrivals), len(scores)
	if PeriConfig.Targeted {
		numTx = len(targetTx)
	}

	// No tx recorded in current period
	if numTx == 0 {
		log.Warn(fmt.Sprintf("Perigee Summary:\n  # tx: \t%d\n"+
			"  # peers: \t%d\n",
			numTx, numPeer))

		if loggy.Config.FlagPerigee {
			s := fmt.Sprintf("\"time\": \"%s\", \"abstime\": %d, ", now, absNow)
			s += fmt.Sprintf("\"num_tx\": %d, \"num_peers\": %d, ", numTx, numPeer)
			go loggy.Log(" {"+s+"}", loggy.PerigeeMsg, loggy.Inbound)
			log.Warn("Loggy recorded perigee status")
		}
		return false
	}

	totalDeliveries := 0
	if PeriConfig.Targeted {
		for tx := range targetTx {
			totalDeliveries += len(arrivalPerPeer[tx])
		}
	} else {
		for _, dmap := range arrivalPerPeer {
			totalDeliveries += len(dmap)
		}
	}

	avgDeliveries := float64(totalDeliveries) / float64(numTx)

	// Compute overall average delay
	totalScores := 0.0
	for _, idScore := range scores {
		totalScores += idScore.score
	}
	avgDelayInSec := totalScores / float64(numPeer) / NanoTranslator

	// Compute quantiles of delay
	quantiles := getQuantiles(scores)

	// Display all delays
	delayStr := alignScores(scores, false)

	// Display all quantiles
	quantStr := alignQuantiles(quantiles, false)

	log.Warn(fmt.Sprintf("Perigee Summary:\n  # tx: \t%d\n"+
		"  # peers: \t%d\n  avg. tx delivered by: %.2f peers\n"+
		"  avg. delay: %.6f sec\n"+
		"  1/%d ~ %d/%d quantiles (sec):\n    %s\n"+
		"  all delays (sec):\n%s",
		numTx, numPeer, avgDeliveries, avgDelayInSec, QuantNum, QuantNum-1, QuantNum, quantStr, delayStr))

	if loggy.Config.FlagPerigee {
		s := fmt.Sprintf("\"time\": \"%s\", \"abstime\": %d, ", now, absNow)
		s += fmt.Sprintf("\"num_tx\": %d, \"num_peers\": %d, ", numTx, numPeer)
		s += fmt.Sprintf("\"avg_deliveries\": %.2f, ", avgDeliveries)
		s += fmt.Sprintf("\"avg_delay\": %.6f, ", avgDelayInSec)
		s += fmt.Sprintf("\"quantiles\": %s, ", alignQuantiles(quantiles, true))
		s += fmt.Sprintf("\"all_delays\": %s", alignScores(scores, true))

		list_evict := make([]string, 0)
		list_excused := make([]string, 0)
		for _, idScore := range scores[:numReplace] {
			if _, isExcused := excused[idScore.id]; isExcused {
				list_excused = append(list_excused, fmt.Sprintf("\"%s\": %.6f", snapshot[idScore.id], idScore.score/NanoTranslator))
			} else {
				list_evict = append(list_evict, fmt.Sprintf("\"%s\": %.6f", snapshot[idScore.id], idScore.score/NanoTranslator))
			}
		}

		s += ", \"peers_evicted\": {"
		s += strings.Join(list_evict, ", ")
		s += "}, \"peers_excused\": {"
		s += strings.Join(list_excused, ", ")
		s += "}, \"peers_kept\": {"

		list_kept := make([]string, 0)
		for _, idScore := range scores[numReplace:] {
			list_kept = append(list_kept, fmt.Sprintf("\"%s\": %.6f", snapshot[idScore.id], idScore.score/NanoTranslator))
		}

		s += strings.Join(list_kept, ", ")
		s += "}"

		go loggy.Log(" {"+s+"}", loggy.PerigeeMsg, loggy.Inbound)
		log.Warn("Loggy recorded perigee status")
	}
	return true
}

// Unit: ns
func getScores() ([]idScore, map[string]bool) {
	// scores := make(map[string]float64)
	scores := []idScore{}
	excused := make(map[string]bool)

	latestArrival := int64(0)
	for tx, firstArrival := range arrivals {
		if PeriConfig.Targeted {
			if _, isTarget := targetTx[tx]; !isTarget {
				continue
			}
		}
		if firstArrival > latestArrival {
			latestArrival = firstArrival
		}
	}

	// loop through the **currect** peers instead of recorded ones
	for id, peer := range h.peers.peers {
		snapshot[id] = h.peers.peers[id].Node().URLv4()
		birth := peer.Peer.Loggy_connectionStartTime.UnixNano()

		ntx, totalDelay, avgDelay := 0, int64(0), 0.0
		for tx, firstArrival := range arrivals {
			if PeriConfig.Targeted {
				if _, isTarget := targetTx[tx]; !isTarget {
					continue
				}
			}
			if firstArrival < birth {
				continue
			}

			arrival, forwarded := arrivalPerPeer[tx][id]
			delay := arrival - firstArrival
			if !forwarded || delay > int64(PeriConfig.MaxDelayPenalty*Milli2Nano) {
				delay = int64(PeriConfig.MaxDelayPenalty * Milli2Nano)
			} else if PeriConfig.Targeted {
				if !loggy.Config.FlagAllTx {
					if delay == 0 {
						loggy.ObserveAll(tx, peer.Node().URLv4(), firstArrival)
					}
				} else {
					loggy.ObserveAll(tx, peer.Node().URLv4(), arrival)
				}
			}

			ntx++
			totalDelay += delay
		}

		if ntx == 0 { // Check if the peer is connected too late (if so, excuse it temporarily)
			avgDelay = float64(int64(PeriConfig.MaxDelayPenalty * Milli2Nano))
			if birth > latestArrival-PeriConfig.MaxDeliveryTolerance*Milli2Nano {
				excused[id] = true
			}
		} else {
			avgDelay = float64(totalDelay) / float64(ntx)
		}

		scores = append(scores, idScore{id, avgDelay})
	}

	// Scores are sorted by descending order
	sort.Slice(scores, func(i, j int) bool {
		ndi, ndj := isNoDrop(scores[i].id), isNoDrop(scores[j].id)
		if ndi && !ndj {
			return false // give i lower priority when i cannot be dropped
		} else if ndj && !ndi {
			return true
		} else {
			return scores[i].score > scores[j].score
		}
	})

	return scores, excused
}

// Unit: ns
func getQuantiles(scores []idScore) []float64 {
	quantiles := make([]float64, QuantNum-1)
	maxpeers := float64(h.maxPeers)
	for i := 1; i < QuantNum; i++ {
		index := maxpeers * float64(i) / float64(QuantNum)
		index_int := int(index) // floor of index
		iscore := len(scores) - 1 - index_int
		w_lo, w_hi := float64(index_int)+1.0-index, index-float64(index_int)

		if index_int < len(scores)-1 {
			quantiles[i-1] = scores[iscore].score*w_lo + scores[iscore-1].score*w_hi
		} else if index_int == len(scores)-1 {
			quantiles[i-1] = scores[iscore].score*w_lo + float64(PeriConfig.MaxDelayPenalty*Milli2Nano)*w_hi
		} else {
			quantiles[i-1] = float64(PeriConfig.MaxDelayPenalty * Milli2Nano)
		}
	}

	return quantiles
}

func alignQuantiles(quants []float64, flagJson bool) string {
	quantEntry := make([]string, len(quants))
	for i := 0; i < len(quants); i++ {
		quantEntry[i] = fmt.Sprintf("%6.3f", quants[i]/NanoTranslator)
	}
	arrText := strings.Join(quantEntry, ", ")
	if flagJson {
		return "[" + arrText + "]"
	} else {
		return arrText
	}
}

func alignScores(scores []idScore, flagJson bool) string {
	if flagJson {
		var delayEntry []string
		for i := len(scores) - 1; i >= 0; i-- {
			delayEntry = append(delayEntry, fmt.Sprintf("%6.3f", scores[i].score/NanoTranslator))
		}
		return fmt.Sprintf("[%s]", strings.Join(delayEntry, ", "))
	} else {
		var delayText, delayLine []string
		j := 0
		for i := len(scores) - 1; i >= 0; i-- {
			delayLine = append(delayLine, fmt.Sprintf("%6.3f", scores[i].score/NanoTranslator))

			if j++; i == 0 || j >= 25 {
				delayText = append(delayText, "    "+strings.Join(delayLine, ", "))
				delayLine = []string{}
				j = 0
			}
		}
		return strings.Join(delayText, "\n")
	}
}

// Check the given signer address in the list provided by Peri config
func isVictimAccount(addr common.Address) bool {
	if isMyself(addr) {
		return false
	}

	for i := 0; i < len(PeriConfig.TargetAccountList); i++ {
		if strings.EqualFold(addr.Hex(), PeriConfig.TargetAccountList[i]) {
			return true
		}
	}
	return false
}

// Check if the given signer address is unlocked by self
func isMyself(addr common.Address) bool {
	for _, account := range MyAccount {
		if strings.EqualFold(addr.Hex(), account) {
			return true
		}
	}
	return false
}

func isSampledByHashDivision(txHash common.Hash) bool {
	if PeriConfig.ObservedTxRatio <= 0 {
		return false
	}
	if PeriConfig.ObservedTxRatio == 1 {
		return true
	}

	z := big.NewInt(0)
	return z.Mod(txHash.Big(), big.NewInt(int64(PeriConfig.ObservedTxRatio))).Cmp(big.NewInt(0)) == 0
}

// Check if a node is always undroppable (for instance, a bloXroute gateway)
func isNoDrop(id string) bool {
	enode := snapshot[id] // h.peers.peers[id].Node().URLv4()
	/*if strings.Contains(enode, ethconfig.SelfIP) {
		return true
	}*/
	for _, ip := range PeriConfig.NoDropList {
		if strings.Contains(enode, ip) {
			return true
		}
	}
	return false
}

func extractIPFromEnode(enode string) string {
	parts := strings.Split(enode, "@")
	parts = strings.Split(parts[len(parts)-1], ":")
	return parts[0]
}

func isBlocked(enode string) bool {
	_, blocked := blocklist[extractIPFromEnode(enode)]
	return blocked
}
