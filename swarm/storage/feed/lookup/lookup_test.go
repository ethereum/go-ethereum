// Copyright 2018 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package lookup_test

import (
	"context"
	"fmt"
	"math/rand"
	"testing"

	"github.com/ethereum/go-ethereum/swarm/log"
	"github.com/ethereum/go-ethereum/swarm/storage/feed/lookup"
)

type Data struct {
	Payload uint64
	Time    uint64
}

type Store map[lookup.EpochID]*Data

func write(store Store, epoch lookup.Epoch, value *Data) {
	log.Debug("Write: %d-%d, value='%d'\n", epoch.Base(), epoch.Level, value.Payload)
	store[epoch.ID()] = value
}

func update(store Store, last lookup.Epoch, now uint64, value *Data) lookup.Epoch {
	epoch := lookup.GetNextEpoch(last, now)

	write(store, epoch, value)

	return epoch
}

const Day = 60 * 60 * 24
const Year = Day * 365
const Month = Day * 30

func makeReadFunc(store Store, counter *int) lookup.ReadFunc {
	return func(ctx context.Context, epoch lookup.Epoch, now uint64) (interface{}, error) {
		*counter++
		data := store[epoch.ID()]
		var valueStr string
		if data != nil {
			valueStr = fmt.Sprintf("%d", data.Payload)
		}
		log.Debug("Read: %d-%d, value='%s'\n", epoch.Base(), epoch.Level, valueStr)
		if data != nil && data.Time <= now {
			return data, nil
		}
		return nil, nil
	}
}

func TestLookup(t *testing.T) {

	store := make(Store)
	readCount := 0
	readFunc := makeReadFunc(store, &readCount)

	// write an update every month for 12 months 3 years ago and then silence for two years
	now := uint64(1533799046)
	var epoch lookup.Epoch

	var lastData *Data
	for i := uint64(0); i < 12; i++ {
		t := uint64(now - Year*3 + i*Month)
		data := Data{
			Payload: t, //our "payload" will be the timestamp itself.
			Time:    t,
		}
		epoch = update(store, epoch, t, &data)
		lastData = &data
	}

	// try to get the last value

	value, err := lookup.Lookup(context.Background(), now, lookup.NoClue, readFunc)
	if err != nil {
		t.Fatal(err)
	}

	readCountWithoutHint := readCount

	if value != lastData {
		t.Fatalf("Expected lookup to return the last written value: %v. Got %v", lastData, value)
	}

	// reset the read count for the next test
	readCount = 0
	// Provide a hint to get a faster lookup. In particular, we give the exact location of the last update
	value, err = lookup.Lookup(context.Background(), now, epoch, readFunc)
	if err != nil {
		t.Fatal(err)
	}

	if value != lastData {
		t.Fatalf("Expected lookup to return the last written value: %v. Got %v", lastData, value)
	}

	if readCount > readCountWithoutHint {
		t.Fatalf("Expected lookup to complete with fewer or same reads than %d since we provided a hint. Did %d reads.", readCountWithoutHint, readCount)
	}

	// try to get an intermediate value
	// if we look for a value in now - Year*3 + 6*Month, we should get that value
	// Since the "payload" is the timestamp itself, we can check this.

	expectedTime := now - Year*3 + 6*Month

	value, err = lookup.Lookup(context.Background(), expectedTime, lookup.NoClue, readFunc)
	if err != nil {
		t.Fatal(err)
	}

	data, ok := value.(*Data)

	if !ok {
		t.Fatal("Expected value to contain data")
	}

	if data.Time != expectedTime {
		t.Fatalf("Expected value timestamp to be %d, got %d", data.Time, expectedTime)
	}

}

func TestOneUpdateAt0(t *testing.T) {

	store := make(Store)
	readCount := 0

	readFunc := makeReadFunc(store, &readCount)
	now := uint64(1533903729)

	var epoch lookup.Epoch
	data := Data{
		Payload: 79,
		Time:    0,
	}
	update(store, epoch, 0, &data)

	value, err := lookup.Lookup(context.Background(), now, lookup.NoClue, readFunc)
	if err != nil {
		t.Fatal(err)
	}
	if value != &data {
		t.Fatalf("Expected lookup to return the last written value: %v. Got %v", data, value)
	}
}

// Tests the update is found even when a bad hint is given
func TestBadHint(t *testing.T) {

	store := make(Store)
	readCount := 0

	readFunc := makeReadFunc(store, &readCount)
	now := uint64(1533903729)

	var epoch lookup.Epoch
	data := Data{
		Payload: 79,
		Time:    0,
	}

	// place an update for t=1200
	update(store, epoch, 1200, &data)

	// come up with some evil hint
	badHint := lookup.Epoch{
		Level: 18,
		Time:  1200000000,
	}

	value, err := lookup.Lookup(context.Background(), now, badHint, readFunc)
	if err != nil {
		t.Fatal(err)
	}
	if value != &data {
		t.Fatalf("Expected lookup to return the last written value: %v. Got %v", data, value)
	}
}

// Tests whether the update is found when the bad hint is exactly below the last update
func TestBadHintNextToUpdate(t *testing.T) {
	store := make(Store)
	readCount := 0

	readFunc := makeReadFunc(store, &readCount)
	now := uint64(1533903729)
	var last *Data

	/*  the following loop places updates in the following epochs:
	Update# Time       Base       Level
	0       1200000000 1174405120 25
	1       1200000001 1191182336 24
	2       1200000002 1199570944 23
	3       1200000003 1199570944 22
	4       1200000004 1199570944 21

	The situation we want to trigger is to give a bad hint exactly
	in T=1200000005, B=1199570944 and L=20, which is where the next
	update would have logically been.
	This affects only when the bad hint's base == previous update's base,
	in this case 1199570944

	*/
	var epoch lookup.Epoch
	for i := uint64(0); i < 5; i++ {
		data := Data{
			Payload: i,
			Time:    0,
		}
		last = &data
		epoch = update(store, epoch, 1200000000+i, &data)
	}

	// come up with some evil hint:
	// put it where the next update would have been
	badHint := lookup.Epoch{
		Level: 20,
		Time:  1200000005,
	}

	value, err := lookup.Lookup(context.Background(), now, badHint, readFunc)
	if err != nil {
		t.Fatal(err)
	}
	if value != last {
		t.Fatalf("Expected lookup to return the last written value: %v. Got %v", last, value)
	}
}

func TestContextCancellation(t *testing.T) {

	readFunc := func(ctx context.Context, epoch lookup.Epoch, now uint64) (interface{}, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	}

	ctx, cancel := context.WithCancel(context.Background())

	errc := make(chan error)

	go func() {
		_, err := lookup.Lookup(ctx, 1200000000, lookup.NoClue, readFunc)
		errc <- err
	}()

	cancel()

	if err := <-errc; err != context.Canceled {
		t.Fatalf("Expected lookup to return a context Cancelled error, got %v", err)
	}

	// text context cancellation during hint lookup:
	ctx, cancel = context.WithCancel(context.Background())
	errc = make(chan error)
	someHint := lookup.Epoch{
		Level: 25,
		Time:  300,
	}

	readFunc = func(ctx context.Context, epoch lookup.Epoch, now uint64) (interface{}, error) {
		if epoch == someHint {
			go cancel()
			<-ctx.Done()
			return nil, ctx.Err()
		}
		return nil, nil
	}

	go func() {
		_, err := lookup.Lookup(ctx, 301, someHint, readFunc)
		errc <- err
	}()

	if err := <-errc; err != context.Canceled {
		t.Fatalf("Expected lookup to return a context Cancelled error, got %v", err)
	}

}

func TestLookupFail(t *testing.T) {

	store := make(Store)
	readCount := 0

	readFunc := makeReadFunc(store, &readCount)
	now := uint64(1533903729)

	// don't write anything and try to look up.
	// we're testing we don't get stuck in a loop

	value, err := lookup.Lookup(context.Background(), now, lookup.NoClue, readFunc)
	if err != nil {
		t.Fatal(err)
	}
	if value != nil {
		t.Fatal("Expected value to be nil, since the update should've failed")
	}

	expectedReads := now/(1<<lookup.HighestLevel) + 1
	if uint64(readCount) != expectedReads {
		t.Fatalf("Expected lookup to fail after %d reads. Did %d reads.", expectedReads, readCount)
	}
}

func TestHighFreqUpdates(t *testing.T) {

	store := make(Store)
	readCount := 0

	readFunc := makeReadFunc(store, &readCount)
	now := uint64(1533903729)

	// write an update every second for the last 1000 seconds
	var epoch lookup.Epoch

	var lastData *Data
	for i := uint64(0); i <= 994; i++ {
		T := uint64(now - 1000 + i)
		data := Data{
			Payload: T, //our "payload" will be the timestamp itself.
			Time:    T,
		}
		epoch = update(store, epoch, T, &data)
		lastData = &data
	}

	value, err := lookup.Lookup(context.Background(), lastData.Time, lookup.NoClue, readFunc)
	if err != nil {
		t.Fatal(err)
	}

	if value != lastData {
		t.Fatalf("Expected lookup to return the last written value: %v. Got %v", lastData, value)
	}

	readCountWithoutHint := readCount
	// reset the read count for the next test
	readCount = 0
	// Provide a hint to get a faster lookup. In particular, we give the exact location of the last update
	value, err = lookup.Lookup(context.Background(), now, epoch, readFunc)
	if err != nil {
		t.Fatal(err)
	}

	if value != lastData {
		t.Fatalf("Expected lookup to return the last written value: %v. Got %v", lastData, value)
	}

	if readCount > readCountWithoutHint {
		t.Fatalf("Expected lookup to complete with fewer or equal reads than %d since we provided a hint. Did %d reads.", readCountWithoutHint, readCount)
	}

	for i := uint64(0); i <= 994; i++ {
		T := uint64(now - 1000 + i) // update every second for the last 1000 seconds
		value, err := lookup.Lookup(context.Background(), T, lookup.NoClue, readFunc)
		if err != nil {
			t.Fatal(err)
		}
		data, _ := value.(*Data)
		if data == nil {
			t.Fatalf("Expected lookup to return %d, got nil", T)
		}
		if data.Payload != T {
			t.Fatalf("Expected lookup to return %d, got %d", T, data.Time)
		}
	}
}

func TestSparseUpdates(t *testing.T) {

	store := make(Store)
	readCount := 0
	readFunc := makeReadFunc(store, &readCount)

	// write an update every 5 years 3 times starting in Jan 1st 1970 and then silence

	now := uint64(1533799046)
	var epoch lookup.Epoch

	var lastData *Data
	for i := uint64(0); i < 5; i++ {
		T := uint64(Year * 5 * i) // write an update every 5 years 3 times starting in Jan 1st 1970 and then silence
		data := Data{
			Payload: T, //our "payload" will be the timestamp itself.
			Time:    T,
		}
		epoch = update(store, epoch, T, &data)
		lastData = &data
	}

	// try to get the last value

	value, err := lookup.Lookup(context.Background(), now, lookup.NoClue, readFunc)
	if err != nil {
		t.Fatal(err)
	}

	readCountWithoutHint := readCount

	if value != lastData {
		t.Fatalf("Expected lookup to return the last written value: %v. Got %v", lastData, value)
	}

	// reset the read count for the next test
	readCount = 0
	// Provide a hint to get a faster lookup. In particular, we give the exact location of the last update
	value, err = lookup.Lookup(context.Background(), now, epoch, readFunc)
	if err != nil {
		t.Fatal(err)
	}

	if value != lastData {
		t.Fatalf("Expected lookup to return the last written value: %v. Got %v", lastData, value)
	}

	if readCount > readCountWithoutHint {
		t.Fatalf("Expected lookup to complete with fewer reads than %d since we provided a hint. Did %d reads.", readCountWithoutHint, readCount)
	}

}

// testG will hold precooked test results
// fields are abbreviated to reduce the size of the literal below
type testG struct {
	e lookup.Epoch // last
	n uint64       // next level
	x uint8        // expected result
}

// test cases
var testGetNextLevelCases = []testG{{e: lookup.Epoch{Time: 989875233, Level: 12}, n: 989875233, x: 11}, {e: lookup.Epoch{Time: 995807650, Level: 18}, n: 995598156, x: 19}, {e: lookup.Epoch{Time: 969167082, Level: 0}, n: 968990357, x: 18}, {e: lookup.Epoch{Time: 993087628, Level: 14}, n: 992987044, x: 20}, {e: lookup.Epoch{Time: 963364631, Level: 20}, n: 963364630, x: 19}, {e: lookup.Epoch{Time: 963497510, Level: 16}, n: 963370732, x: 18}, {e: lookup.Epoch{Time: 955421349, Level: 22}, n: 955421348, x: 21}, {e: lookup.Epoch{Time: 968220379, Level: 15}, n: 968220378, x: 14}, {e: lookup.Epoch{Time: 939129014, Level: 6}, n: 939128771, x: 11}, {e: lookup.Epoch{Time: 907847903, Level: 6}, n: 907791833, x: 18}, {e: lookup.Epoch{Time: 910835564, Level: 15}, n: 910835564, x: 14}, {e: lookup.Epoch{Time: 913578333, Level: 22}, n: 881808431, x: 25}, {e: lookup.Epoch{Time: 895818460, Level: 3}, n: 895818132, x: 9}, {e: lookup.Epoch{Time: 903843025, Level: 24}, n: 895609561, x: 23}, {e: lookup.Epoch{Time: 877889433, Level: 13}, n: 877877093, x: 15}, {e: lookup.Epoch{Time: 901450396, Level: 10}, n: 901450058, x: 9}, {e: lookup.Epoch{Time: 925179910, Level: 3}, n: 925168393, x: 16}, {e: lookup.Epoch{Time: 913485477, Level: 21}, n: 913485476, x: 20}, {e: lookup.Epoch{Time: 924462991, Level: 18}, n: 924462990, x: 17}, {e: lookup.Epoch{Time: 941175128, Level: 13}, n: 941175127, x: 12}, {e: lookup.Epoch{Time: 920126583, Level: 3}, n: 920100782, x: 19}, {e: lookup.Epoch{Time: 932403200, Level: 9}, n: 932279891, x: 17}, {e: lookup.Epoch{Time: 948284931, Level: 2}, n: 948284921, x: 9}, {e: lookup.Epoch{Time: 953540997, Level: 7}, n: 950547986, x: 22}, {e: lookup.Epoch{Time: 926639837, Level: 18}, n: 918608882, x: 24}, {e: lookup.Epoch{Time: 954637598, Level: 1}, n: 954578761, x: 17}, {e: lookup.Epoch{Time: 943482981, Level: 10}, n: 942924151, x: 19}, {e: lookup.Epoch{Time: 963580771, Level: 7}, n: 963580771, x: 6}, {e: lookup.Epoch{Time: 993744930, Level: 7}, n: 993690858, x: 16}, {e: lookup.Epoch{Time: 1018890213, Level: 12}, n: 1018890212, x: 11}, {e: lookup.Epoch{Time: 1030309411, Level: 2}, n: 1030309227, x: 9}, {e: lookup.Epoch{Time: 1063204997, Level: 20}, n: 1063204996, x: 19}, {e: lookup.Epoch{Time: 1094340832, Level: 6}, n: 1094340633, x: 7}, {e: lookup.Epoch{Time: 1077880597, Level: 10}, n: 1075914292, x: 20}, {e: lookup.Epoch{Time: 1051114957, Level: 18}, n: 1051114957, x: 17}, {e: lookup.Epoch{Time: 1045649701, Level: 22}, n: 1045649700, x: 21}, {e: lookup.Epoch{Time: 1066198885, Level: 14}, n: 1066198884, x: 13}, {e: lookup.Epoch{Time: 1053231952, Level: 1}, n: 1053210845, x: 16}, {e: lookup.Epoch{Time: 1068763404, Level: 14}, n: 1068675428, x: 18}, {e: lookup.Epoch{Time: 1039042173, Level: 15}, n: 1038973110, x: 17}, {e: lookup.Epoch{Time: 1050747636, Level: 6}, n: 1050747364, x: 9}, {e: lookup.Epoch{Time: 1030034434, Level: 23}, n: 1030034433, x: 22}, {e: lookup.Epoch{Time: 1003783425, Level: 18}, n: 1003783424, x: 17}, {e: lookup.Epoch{Time: 988163976, Level: 15}, n: 988084064, x: 17}, {e: lookup.Epoch{Time: 1007222377, Level: 15}, n: 1007222377, x: 14}, {e: lookup.Epoch{Time: 1001211375, Level: 13}, n: 1001208178, x: 14}, {e: lookup.Epoch{Time: 997623199, Level: 8}, n: 997623198, x: 7}, {e: lookup.Epoch{Time: 1026283830, Level: 10}, n: 1006681704, x: 24}, {e: lookup.Epoch{Time: 1019421907, Level: 20}, n: 1019421906, x: 19}, {e: lookup.Epoch{Time: 1043154306, Level: 16}, n: 1043108343, x: 16}, {e: lookup.Epoch{Time: 1075643767, Level: 17}, n: 1075325898, x: 18}, {e: lookup.Epoch{Time: 1043726309, Level: 20}, n: 1043726308, x: 19}, {e: lookup.Epoch{Time: 1056415324, Level: 17}, n: 1056415324, x: 16}, {e: lookup.Epoch{Time: 1088650219, Level: 13}, n: 1088650218, x: 12}, {e: lookup.Epoch{Time: 1088551662, Level: 7}, n: 1088543355, x: 13}, {e: lookup.Epoch{Time: 1069667265, Level: 6}, n: 1069667075, x: 7}, {e: lookup.Epoch{Time: 1079145970, Level: 18}, n: 1079145969, x: 17}, {e: lookup.Epoch{Time: 1083338876, Level: 7}, n: 1083338875, x: 6}, {e: lookup.Epoch{Time: 1051581086, Level: 4}, n: 1051568869, x: 14}, {e: lookup.Epoch{Time: 1028430882, Level: 4}, n: 1028430864, x: 5}, {e: lookup.Epoch{Time: 1057356462, Level: 1}, n: 1057356417, x: 5}, {e: lookup.Epoch{Time: 1033104266, Level: 0}, n: 1033097479, x: 13}, {e: lookup.Epoch{Time: 1031391367, Level: 11}, n: 1031387304, x: 14}, {e: lookup.Epoch{Time: 1049781164, Level: 15}, n: 1049781163, x: 14}, {e: lookup.Epoch{Time: 1027271628, Level: 12}, n: 1027271627, x: 11}, {e: lookup.Epoch{Time: 1057270560, Level: 23}, n: 1057270560, x: 22}, {e: lookup.Epoch{Time: 1047501317, Level: 15}, n: 1047501317, x: 14}, {e: lookup.Epoch{Time: 1058349035, Level: 11}, n: 1045175573, x: 24}, {e: lookup.Epoch{Time: 1057396147, Level: 20}, n: 1057396147, x: 19}, {e: lookup.Epoch{Time: 1048906375, Level: 18}, n: 1039616919, x: 25}, {e: lookup.Epoch{Time: 1074294831, Level: 20}, n: 1074294831, x: 19}, {e: lookup.Epoch{Time: 1088946052, Level: 1}, n: 1088917364, x: 14}, {e: lookup.Epoch{Time: 1112337595, Level: 17}, n: 1111008110, x: 22}, {e: lookup.Epoch{Time: 1099990284, Level: 5}, n: 1099968370, x: 15}, {e: lookup.Epoch{Time: 1087036441, Level: 16}, n: 1053967855, x: 25}, {e: lookup.Epoch{Time: 1069225185, Level: 8}, n: 1069224660, x: 10}, {e: lookup.Epoch{Time: 1057505479, Level: 9}, n: 1057505170, x: 14}, {e: lookup.Epoch{Time: 1072381377, Level: 12}, n: 1065950959, x: 22}, {e: lookup.Epoch{Time: 1093887139, Level: 8}, n: 1093863305, x: 14}, {e: lookup.Epoch{Time: 1082366510, Level: 24}, n: 1082366510, x: 23}, {e: lookup.Epoch{Time: 1103231132, Level: 14}, n: 1102292201, x: 22}, {e: lookup.Epoch{Time: 1094502355, Level: 3}, n: 1094324652, x: 18}, {e: lookup.Epoch{Time: 1068488344, Level: 12}, n: 1067577330, x: 19}, {e: lookup.Epoch{Time: 1050278233, Level: 12}, n: 1050278232, x: 11}, {e: lookup.Epoch{Time: 1047660768, Level: 5}, n: 1047652137, x: 17}, {e: lookup.Epoch{Time: 1060116167, Level: 11}, n: 1060114091, x: 12}, {e: lookup.Epoch{Time: 1068149392, Level: 21}, n: 1052074801, x: 24}, {e: lookup.Epoch{Time: 1081934120, Level: 6}, n: 1081933847, x: 8}, {e: lookup.Epoch{Time: 1107943693, Level: 16}, n: 1107096139, x: 25}, {e: lookup.Epoch{Time: 1131571649, Level: 9}, n: 1131570428, x: 11}, {e: lookup.Epoch{Time: 1123139367, Level: 0}, n: 1122912198, x: 20}, {e: lookup.Epoch{Time: 1121144423, Level: 6}, n: 1120568289, x: 20}, {e: lookup.Epoch{Time: 1089932411, Level: 17}, n: 1089932410, x: 16}, {e: lookup.Epoch{Time: 1104899012, Level: 22}, n: 1098978789, x: 22}, {e: lookup.Epoch{Time: 1094588059, Level: 21}, n: 1094588059, x: 20}, {e: lookup.Epoch{Time: 1114987438, Level: 24}, n: 1114987437, x: 23}, {e: lookup.Epoch{Time: 1084186305, Level: 7}, n: 1084186241, x: 6}, {e: lookup.Epoch{Time: 1058827111, Level: 8}, n: 1058826504, x: 9}, {e: lookup.Epoch{Time: 1090679810, Level: 12}, n: 1090616539, x: 17}, {e: lookup.Epoch{Time: 1084299475, Level: 23}, n: 1084299475, x: 22}}

func TestGetNextLevel(t *testing.T) {

	// First, test well-known cases
	last := lookup.Epoch{
		Time:  1533799046,
		Level: 5,
	}

	level := lookup.GetNextLevel(last, last.Time)
	expected := uint8(4)
	if level != expected {
		t.Fatalf("Expected GetNextLevel to return %d for same-time updates at a nonzero level, got %d", expected, level)
	}

	level = lookup.GetNextLevel(last, last.Time+(1<<lookup.HighestLevel)+3000)
	expected = lookup.HighestLevel
	if level != expected {
		t.Fatalf("Expected GetNextLevel to return %d for updates set 2^lookup.HighestLevel seconds away, got %d", expected, level)
	}

	level = lookup.GetNextLevel(last, last.Time+(1<<last.Level))
	expected = last.Level
	if level != expected {
		t.Fatalf("Expected GetNextLevel to return %d for updates set 2^last.Level seconds away, got %d", expected, level)
	}

	last.Level = 0
	level = lookup.GetNextLevel(last, last.Time)
	expected = 0
	if level != expected {
		t.Fatalf("Expected GetNextLevel to return %d for same-time updates at a zero level, got %d", expected, level)
	}

	// run a batch of 100 cooked tests
	for _, s := range testGetNextLevelCases {
		level := lookup.GetNextLevel(s.e, s.n)
		if level != s.x {
			t.Fatalf("Expected GetNextLevel to return %d for last=%s when now=%d, got %d", s.x, s.e.String(), s.n, level)
		}
	}

}

// cookGetNextLevelTests is used to generate a deterministic
// set of cases for TestGetNextLevel and thus "freeze" its current behavior
func CookGetNextLevelTests(t *testing.T) {
	st := ""
	var last lookup.Epoch
	last.Time = 1000000000
	var now uint64
	var expected uint8
	for i := 0; i < 100; i++ {
		last.Time += uint64(rand.Intn(1<<26)) - (1 << 25)
		last.Level = uint8(rand.Intn(25))
		v := last.Level + uint8(rand.Intn(lookup.HighestLevel))
		if v > lookup.HighestLevel {
			v = 0
		}
		now = last.Time + uint64(rand.Intn(1<<v+1)) - (1 << v)
		expected = lookup.GetNextLevel(last, now)
		st = fmt.Sprintf("%s,testG{e:lookup.Epoch{Time:%d, Level:%d}, n:%d, x:%d}", st, last.Time, last.Level, now, expected)
	}
	fmt.Println(st)
}
