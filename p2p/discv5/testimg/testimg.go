package main

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"image"
	"image/png"
	"math"
	"os"
	"sort"
	"strconv"

	"github.com/ethereum/go-ethereum/crypto"
)

var xs, ys, maxTime int
var minAbsTime, maxAbsTime int64

func set(pic *image.NRGBA, x, y, c, v int) {
	if v > 255 {
		v = 255
	}
	if x >= 0 && x < xs && y >= 0 && y < ys {
		pic.Pix[y*pic.Stride+x*4+c] = uint8(v)
	}
}

type nodeStats []struct{ wpSum, wpCnt, wpXcnt, regCnt, regXcnt uint64 }

type nodeInfo struct {
	maxMR  int
	topics map[string]struct{}
}

const (
	regStatDiv  = 60
	regStatYdiv = 30
)

type topicInfo struct {
	prefix    uint64
	nodes     uint64Slice
	nodeStats nodeStats
	nodeIdx   map[uint64]int
	pic, pic2 *image.NRGBA
	nodeRad   map[uint64]int
	regStats  []int
}

func main() {
	var nodes uint64Slice
	topics := make(map[string]*topicInfo)

	inputFile := "test.out"
	if len(os.Args) > 1 {
		inputFile = os.Args[1]
	}

	f, _ := os.Open(inputFile)
	scanner := bufio.NewScanner(f)
	scanner.Split(bufio.ScanWords)
	minAbsTime = math.MaxInt64
	for scanner.Scan() {
		w := scanner.Text()
		if w == "*N" {
			scanner.Scan()
			prefix, _ := strconv.ParseUint(scanner.Text(), 16, 64)
			nodes = append(nodes, prefix)
		}
		if w == "*R" {
			scanner.Scan()
			time, _ := strconv.ParseInt(scanner.Text(), 10, 64)
			if time > maxAbsTime {
				maxAbsTime = time
			}
			if time < minAbsTime {
				minAbsTime = time
			}
			scanner.Scan()
			topic := scanner.Text()
			if _, ok := topics[topic]; !ok {
				fmt.Println(topic)
				topicHash := crypto.Keccak256Hash([]byte(topic))
				topics[topic] = &topicInfo{prefix: binary.BigEndian.Uint64(topicHash[:8])}
			}
		}
	}
	f.Close()

	maxTime = int(maxAbsTime - minAbsTime)
	xs = maxTime / 10000
	ys = len(nodes)
	nodeIdx := make(map[uint64]int)
	for i, v := range nodes {
		nodeIdx[v] = i
	}
	nodeInfo := make([]nodeInfo, len(nodes))

	for _, t := range topics {
		t.nodes = make(uint64Slice, len(nodes))
		t.nodeStats = make(nodeStats, len(nodes))
		for i, v := range nodes {
			t.nodes[i] = v ^ t.prefix
		}
		sort.Sort(t.nodes)
		t.nodeIdx = make(map[uint64]int)
		for i, v := range t.nodes {
			t.nodeIdx[v^t.prefix] = i
		}

		t.pic = image.NewNRGBA(image.Rect(0, 0, xs, ys))
		for y := 0; y < ys; y++ {
			for x := 0; x < xs; x++ {
				set(t.pic, x, y, 3, 255)
			}
		}

		t.pic2 = image.NewNRGBA(image.Rect(0, 0, xs, ys))
		for y := 0; y < ys; y++ {
			for x := 0; x < xs; x++ {
				set(t.pic2, x, y, 3, 255)
			}
		}
		t.nodeRad = make(map[uint64]int)
		t.regStats = make([]int, xs/regStatDiv+1)
	}

	f, _ = os.Open(inputFile)
	scanner = bufio.NewScanner(f)
	scanner.Split(bufio.ScanWords)
	statBegin := int64(40000000)
	statEnd := int64(maxTime - 10000000)

	for scanner.Scan() {
		w := scanner.Text()
		if w == "*R" {
			scanner.Scan()
			time, _ := strconv.ParseInt(scanner.Text(), 10, 64)
			time -= minAbsTime
			scanner.Scan()
			t := topics[scanner.Text()]
			scanner.Scan()
			prefix, _ := strconv.ParseUint(scanner.Text(), 16, 64)
			scanner.Scan()
			rad, _ := strconv.ParseInt(scanner.Text(), 10, 64)
			if int(rad) != t.nodeRad[prefix] {
				t.nodeRad[prefix] = int(rad)
				radUint := uint64(rad) * ((^uint64(0)) / 1000000)
				x := int(time * int64(xs) / int64(maxTime))
				y := sort.Search(ys, func(i int) bool {
					return t.nodes[i] > radUint
				})
				set(t.pic, x, y, 1, 255)
			}
		}
		if w == "*MR" {
			scanner.Scan()
			time, _ := strconv.ParseInt(scanner.Text(), 10, 64)
			time -= minAbsTime
			scanner.Scan()
			topic := scanner.Text()
			t := topics[topic]
			scanner.Scan()
			prefix, _ := strconv.ParseUint(scanner.Text(), 16, 64)
			scanner.Scan()
			rad, _ := strconv.ParseInt(scanner.Text(), 10, 64)
			radUint := uint64(rad) * ((^uint64(0)) / 1000000)
			x := int(time * int64(xs) / int64(maxTime))
			y := sort.Search(ys, func(i int) bool {
				return t.nodes[i] > radUint
			})
			set(t.pic, x, y, 0, 255)
			ni := nodeInfo[nodeIdx[prefix]]
			if int(rad) > ni.maxMR {
				ni.maxMR = int(rad)
				if ni.topics == nil {
					ni.topics = make(map[string]struct{})
				}
				ni.topics[topic] = struct{}{}
			}
			nodeInfo[nodeIdx[prefix]] = ni
		}
		if w == "*W" {
			scanner.Scan()
			time, _ := strconv.ParseInt(scanner.Text(), 10, 64)
			time -= minAbsTime
			scanner.Scan()
			t := topics[scanner.Text()]
			scanner.Scan()
			prefix, _ := strconv.ParseUint(scanner.Text(), 16, 64)
			scanner.Scan()
			wp, _ := strconv.ParseInt(scanner.Text(), 10, 64)
			x := int(time * int64(xs) / int64(maxTime))
			y := t.nodeIdx[prefix]
			if time >= statBegin && time < statEnd {
				t.nodeStats[y].wpSum += uint64(wp)
				if wp >= 600000 {
					t.nodeStats[y].wpXcnt++
				}
				t.nodeStats[y].wpCnt++
			}
			/*set(t.pic2, x, y, 0, int(wp/100000))
			set(t.pic2, x, y, 1, int(wp/10000))
			set(t.pic2, x, y, 2, int(wp/1000))*/
			if wp >= 1800000 {
				set(t.pic2, x, y, 0, 255)
			}
			if wp >= 600000 {
				set(t.pic2, x, y, 1, 255)
			}
			if wp >= 60000 {
				set(t.pic2, x, y, 2, 255)
			}
		}
		if w == "*+" {
			scanner.Scan()
			time, _ := strconv.ParseInt(scanner.Text(), 10, 64)
			time -= minAbsTime
			scanner.Scan()
			t := topics[scanner.Text()]
			scanner.Scan()
			prefix, _ := strconv.ParseUint(scanner.Text(), 16, 64)
			x := int(time * int64(xs) / int64(maxTime))
			if x < xs {
				t.regStats[x/regStatDiv]++
			}
			y := t.nodeIdx[prefix]
			set(t.pic, x, y, 2, 255)
			scanner.Scan()
			prefix2, _ := strconv.ParseUint(scanner.Text(), 16, 64)
			y2 := t.nodeIdx[prefix2]
			if time >= statBegin && time < statEnd {
				t.nodeStats[y].regCnt++
				t.nodeStats[y2].regXcnt++
			}
		}
	}
	f.Close()

	for tt, t := range topics {
		f, _ = os.Create("test_" + tt + ".png")
		w := bufio.NewWriter(f)
		png.Encode(w, t.pic)
		w.Flush()
		f.Close()

		for x := 0; x < xs; x++ {
			yy := t.regStats[x/regStatDiv] / regStatYdiv
			if yy > ys {
				yy = ys
			}
			for y := 0; y < yy; y++ {
				set(t.pic2, x, ys-1-y, 1, 255)
			}
		}

		f, _ = os.Create("test2_" + tt + ".png")
		w = bufio.NewWriter(f)
		png.Encode(w, t.pic2)
		w.Flush()
		f.Close()

		if statEnd > statBegin {
			xxs := len(t.nodeStats)
			yys := 1000
			yyh := yys / 2
			pic3 := image.NewNRGBA(image.Rect(0, 0, xxs, yys))
			for y := 0; y < yys; y++ {
				for x := 0; x < xxs; x++ {
					set(pic3, x, y, 3, 255)
				}
			}
			for x := 0; x < xxs; x++ {
				wpy := 0
				if t.nodeStats[x].wpCnt > 0 {
					//					wpy = int(t.nodeStats[x].wpSum / t.nodeStats[x].wpCnt / 10000)
					wpy = int(uint64(yyh) * t.nodeStats[x].wpXcnt / t.nodeStats[x].wpCnt)
				}
				if wpy > yyh {
					wpy = yyh
				}
				for y := 0; y < wpy; y++ {
					set(pic3, x, yys-1-y, 1, 255)
				}
				regy := int(t.nodeStats[x].regCnt * 2400000 / uint64(statEnd-statBegin))
				if regy > yyh {
					regy = yyh
				}
				for y := 0; y < regy; y++ {
					set(pic3, x, yyh-1-y, 2, 255)
				}
				regy2 := int(t.nodeStats[x].regXcnt * 2400000 / uint64(statEnd-statBegin))
				if regy2 > yyh {
					regy2 = yyh
				}
				for y := 0; y < regy2; y++ {
					set(pic3, x, yyh-1-y, 0, 255)
				}
			}

			f, _ = os.Create("test3_" + tt + ".png")
			w = bufio.NewWriter(f)
			png.Encode(w, pic3)
			w.Flush()
			f.Close()
		}
	}

	for i, ni := range nodeInfo {
		fmt.Printf("%d %016x  maxMR = %d  ", i, nodes[i], ni.maxMR)
		for t, _ := range ni.topics {
			fmt.Printf(" %s", t)
		}
		fmt.Println()
	}
}

type uint64Slice []uint64

// Len is the number of elements in the collection.
func (s uint64Slice) Len() int {
	return len(s)
}

// Less reports whether the element with
// index i should sort before the element with index j.
func (s uint64Slice) Less(i, j int) bool {
	return s[i] < s[j]
}

// Swap swaps the elements with indexes i and j.
func (s uint64Slice) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
