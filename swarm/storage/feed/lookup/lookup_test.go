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
	"time"

	"github.com/ethereum/go-ethereum/swarm/storage/feed/lookup"
)

type AlgorithmInfo struct {
	Lookup lookup.Algorithm
	Name   string
}

var algorithms = []AlgorithmInfo{
	{lookup.FluzCapacitorAlgorithm, "FluzCapacitor"},
	{lookup.LongEarthAlgorithm, "LongEarth"},
}

const enablePrintMetrics = false // set to true to display algorithm benchmarking stats

func printMetric(metric string, store *Store, elapsed time.Duration) {
	if enablePrintMetrics {
		fmt.Printf("metric=%s, readcount=%d (successful=%d, failed=%d), cached=%d, canceled=%d, maxSimult=%d, elapsed=%s\n", metric,
			store.reads, store.successful, store.failed, store.cacheHits, store.canceled, store.maxSimultaneous, elapsed)
	}
}

const Day = 60 * 60 * 24
const Year = Day * 365
const Month = Day * 30

// DefaultStoreConfig indicates the time the different read
// operations will take in the simulation
// This allows to measure an algorithm performance relative
// to other
var DefaultStoreConfig = &StoreConfig{
	CacheReadTime:      50 * time.Millisecond,
	FailedReadTime:     1000 * time.Millisecond,
	SuccessfulReadTime: 500 * time.Millisecond,
}

// TestLookup verifies if the last update and intermediates are
// found and if that same last update is found faster if a hint is given
func TestLookup(t *testing.T) {
	// ### 1.- Initialize stopwatch time sim
	stopwatch := NewStopwatch(50 * time.Millisecond)
	lookup.TimeAfter = stopwatch.TimeAfter()
	defer stopwatch.Stop()

	// ### 2.- Setup mock storage and generate updates
	store := NewStore(DefaultStoreConfig)
	readFunc := store.MakeReadFunc()

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
		epoch = store.Update(epoch, t, &data)
		lastData = &data
	}

	// ### 3.- Test all algorithms
	for _, algo := range algorithms {
		t.Run(algo.Name, func(t *testing.T) {

			store.Reset() // reset the store read counters

			// ### 3.1.- Test how long it takes to find the last update without a hint:
			timeElapsedWithoutHint := stopwatch.Measure(func() {

				// try to get the last value
				value, err := algo.Lookup(context.Background(), now, lookup.NoClue, readFunc)
				if err != nil {
					t.Fatal(err)
				}
				if value != lastData {
					t.Fatalf("Expected lookup to return the last written value: %v. Got %v", lastData, value)
				}

			})
			printMetric("SIMPLE READ", store, timeElapsedWithoutHint)

			store.Reset() // reset the read counters for the next test

			// ### 3.2.- Test how long it takes to find the last update *with* a hint.
			// it should take less time!
			timeElapsed := stopwatch.Measure(func() {
				// Provide a hint to get a faster lookup. In particular, we give the exact location of the last update
				value, err := algo.Lookup(context.Background(), now, epoch, readFunc)
				if err != nil {
					t.Fatal(err)
				}
				if value != lastData {
					t.Fatalf("Expected lookup to return the last written value: %v. Got %v", lastData, value)
				}
			})
			printMetric("WITH HINT", store, stopwatch.Elapsed())

			if timeElapsed > timeElapsedWithoutHint {
				t.Fatalf("Expected lookup to complete faster than %s since we provided a hint. Took %s", timeElapsedWithoutHint, timeElapsed)
			}

			store.Reset() // reset the read counters for the next test

			// ### 3.3.- try to get an intermediate value
			// if we look for a value in, e.g., now - Year*3 + 6*Month, we should get that value
			// Since the "payload" is the timestamp itself, we can check this.
			expectedTime := now - Year*3 + 6*Month
			timeElapsed = stopwatch.Measure(func() {
				value, err := algo.Lookup(context.Background(), expectedTime, lookup.NoClue, readFunc)
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
			})
			printMetric("INTERMEDIATE READ", store, timeElapsed)
		})
	}
}

// TestOneUpdateAt0 checks if the lookup algorithm can return an update that
// is precisely set at t=0
func TestOneUpdateAt0(t *testing.T) {
	// ### 1.- Initialize stopwatch time sim
	stopwatch := NewStopwatch(50 * time.Millisecond)
	lookup.TimeAfter = stopwatch.TimeAfter()
	defer stopwatch.Stop()

	// ### 2.- Setup mock storage and generate updates
	store := NewStore(DefaultStoreConfig)
	readFunc := store.MakeReadFunc()

	now := uint64(1533903729)

	var epoch lookup.Epoch
	data := Data{
		Payload: 79,
		Time:    0,
	}
	store.Update(epoch, 0, &data) //place 1 update in t=0

	// ### 3.- Test all algorithms
	for _, algo := range algorithms {
		t.Run(algo.Name, func(t *testing.T) {
			store.Reset() // reset the read counters for the next test
			timeElapsed := stopwatch.Measure(func() {
				value, err := algo.Lookup(context.Background(), now, lookup.NoClue, readFunc)
				if err != nil {
					t.Fatal(err)
				}
				if value != &data {
					t.Fatalf("Expected lookup to return the last written value: %v. Got %v", data, value)
				}
			})
			printMetric("SIMPLE", store, timeElapsed)
		})
	}
}

// TestBadHint tests if the update is found even when a bad hint is given
func TestBadHint(t *testing.T) {
	// ### 1.- Initialize stopwatch time sim
	stopwatch := NewStopwatch(50 * time.Millisecond)
	lookup.TimeAfter = stopwatch.TimeAfter()
	defer stopwatch.Stop()

	// ### 2.- Setup mock storage and generate updates
	store := NewStore(DefaultStoreConfig)
	readFunc := store.MakeReadFunc()

	now := uint64(1533903729)

	var epoch lookup.Epoch
	data := Data{
		Payload: 79,
		Time:    0,
	}

	// place an update for t=1200
	store.Update(epoch, 1200, &data)

	// come up with some evil hint
	badHint := lookup.Epoch{
		Level: 18,
		Time:  1200000000,
	}

	// ### 3.- Test all algorithms
	for _, algo := range algorithms {
		t.Run(algo.Name, func(t *testing.T) {
			store.Reset()
			timeElapsed := stopwatch.Measure(func() {
				value, err := algo.Lookup(context.Background(), now, badHint, readFunc)
				if err != nil {
					t.Fatal(err)
				}
				if value != &data {
					t.Fatalf("Expected lookup to return the last written value: %v. Got %v", data, value)
				}
			})
			printMetric("SIMPLE", store, timeElapsed)
		})
	}
}

// TestBadHintNextToUpdate checks whether the update is found when the bad hint is exactly below the last update
func TestBadHintNextToUpdate(t *testing.T) {
	// ### 1.- Initialize stopwatch time sim
	stopwatch := NewStopwatch(50 * time.Millisecond)
	lookup.TimeAfter = stopwatch.TimeAfter()
	defer stopwatch.Stop()

	// ### 2.- Setup mock storage and generate updates
	store := NewStore(DefaultStoreConfig)
	readFunc := store.MakeReadFunc()

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
		epoch = store.Update(epoch, 1200000000+i, &data)
	}

	// come up with some evil hint:
	// put it where the next update would have been
	badHint := lookup.Epoch{
		Level: 20,
		Time:  1200000005,
	}

	// ### 3.- Test all algorithms
	for _, algo := range algorithms {
		t.Run(algo.Name, func(t *testing.T) {
			store.Reset() // reset read counters for next test

			timeElapsed := stopwatch.Measure(func() {
				value, err := algo.Lookup(context.Background(), now, badHint, readFunc)
				if err != nil {
					t.Fatal(err)
				}
				if value != last {
					t.Fatalf("Expected lookup to return the last written value: %v. Got %v", last, value)
				}
			})
			printMetric("SIMPLE", store, timeElapsed)
		})
	}
}

// TestContextCancellation checks whether a lookup can be canceled
func TestContextCancellation(t *testing.T) {

	// ### 1.- Test all algorithms
	for _, algo := range algorithms {
		t.Run(algo.Name, func(t *testing.T) {

			// ### 2.1.- Test a simple cancel of an always blocking read function
			readFunc := func(ctx context.Context, epoch lookup.Epoch, now uint64) (interface{}, error) {
				<-ctx.Done()
				return nil, ctx.Err()
			}

			ctx, cancel := context.WithCancel(context.Background())
			errc := make(chan error)

			go func() {
				_, err := algo.Lookup(ctx, 1200000000, lookup.NoClue, readFunc)
				errc <- err
			}()

			cancel() //actually cancel the lookup

			if err := <-errc; err != context.Canceled {
				t.Fatalf("Expected lookup to return a context canceled error, got %v", err)
			}

			// ### 2.2.- Test context cancellation during hint lookup:
			ctx, cancel = context.WithCancel(context.Background())
			errc = make(chan error)
			someHint := lookup.Epoch{
				Level: 25,
				Time:  300,
			}
			// put up a read function that gets canceled only on hint lookup
			readFunc = func(ctx context.Context, epoch lookup.Epoch, now uint64) (interface{}, error) {
				if epoch == someHint {
					go cancel()
					<-ctx.Done()
					return nil, ctx.Err()
				}
				return nil, nil
			}

			go func() {
				_, err := algo.Lookup(ctx, 301, someHint, readFunc)
				errc <- err
			}()

			if err := <-errc; err != context.Canceled {
				t.Fatalf("Expected lookup to return a context canceled error, got %v", err)
			}
		})
	}

}

// TestLookupFail makes sure the lookup function fails on a timely manner
// when there are no updates at all
func TestLookupFail(t *testing.T) {
	// ### 1.- Initialize stopwatch time sim
	stopwatch := NewStopwatch(50 * time.Millisecond)
	lookup.TimeAfter = stopwatch.TimeAfter()
	defer stopwatch.Stop()

	// ### 2.- Setup mock storage, without adding updates
	// don't write anything and try to look up.
	// we're testing we don't get stuck in a loop and that the lookup
	// function converges in a timely fashion
	store := NewStore(DefaultStoreConfig)
	readFunc := store.MakeReadFunc()

	now := uint64(1533903729)

	// ### 3.- Test all algorithms
	for _, algo := range algorithms {
		t.Run(algo.Name, func(t *testing.T) {
			store.Reset()

			stopwatch.Measure(func() {
				value, err := algo.Lookup(context.Background(), now, lookup.NoClue, readFunc)
				if err != nil {
					t.Fatal(err)
				}
				if value != nil {
					t.Fatal("Expected value to be nil, since the update should've failed")
				}
			})

			printMetric("SIMPLE", store, stopwatch.Elapsed())
		})
	}
}

func TestHighFreqUpdates(t *testing.T) {
	// ### 1.- Initialize stopwatch time sim
	stopwatch := NewStopwatch(50 * time.Millisecond)
	lookup.TimeAfter = stopwatch.TimeAfter()
	defer stopwatch.Stop()

	// ### 2.- Setup mock storage and add one update per second
	// for the last 1000 seconds:
	store := NewStore(DefaultStoreConfig)
	readFunc := store.MakeReadFunc()

	now := uint64(1533903729)

	var epoch lookup.Epoch

	var lastData *Data
	for i := uint64(0); i <= 994; i++ {
		T := uint64(now - 1000 + i)
		data := Data{
			Payload: T, //our "payload" will be the timestamp itself.
			Time:    T,
		}
		epoch = store.Update(epoch, T, &data)
		lastData = &data
	}

	// ### 3.- Test all algorithms
	for _, algo := range algorithms {
		t.Run(algo.Name, func(t *testing.T) {
			store.Reset() // reset read counters for next test

			// ### 3.1.- Test how long it takes to find the last update without a hint:
			timeElapsedWithoutHint := stopwatch.Measure(func() {
				value, err := algo.Lookup(context.Background(), lastData.Time, lookup.NoClue, readFunc)
				stopwatch.Stop()
				if err != nil {
					t.Fatal(err)
				}

				if value != lastData {
					t.Fatalf("Expected lookup to return the last written value: %v. Got %v", lastData, value)
				}
			})
			printMetric("SIMPLE", store, timeElapsedWithoutHint)

			// reset the read count for the next test
			store.Reset()

			// ### 3.2.- Now test how long it takes to find the last update *with* a hint,
			// it should take less time!
			timeElapsed := stopwatch.Measure(func() {
				// Provide a hint to get a faster lookup. In particular, we give the exact location of the last update
				value, err := algo.Lookup(context.Background(), now, epoch, readFunc)
				stopwatch.Stop()
				if err != nil {
					t.Fatal(err)
				}

				if value != lastData {
					t.Fatalf("Expected lookup to return the last written value: %v. Got %v", lastData, value)
				}

			})
			if timeElapsed > timeElapsedWithoutHint {
				t.Fatalf("Expected lookup to complete faster than %s since we provided a hint. Took %s", timeElapsedWithoutHint, timeElapsed)
			}
			printMetric("WITH HINT", store, timeElapsed)

			store.Reset() // reset read counters

			// ### 3.3.- Test multiple lookups at different intervals
			timeElapsed = stopwatch.Measure(func() {
				for i := uint64(0); i <= 10; i++ {
					T := uint64(now - 1000 + i)
					value, err := algo.Lookup(context.Background(), T, lookup.NoClue, readFunc)
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
			})
			printMetric("MULTIPLE", store, timeElapsed)
		})
	}
}

// TestSparseUpdates checks the lookup algorithm when
// updates come sparsely and in bursts
func TestSparseUpdates(t *testing.T) {
	// ### 1.- Initialize stopwatch time sim
	stopwatch := NewStopwatch(50 * time.Millisecond)
	lookup.TimeAfter = stopwatch.TimeAfter()
	defer stopwatch.Stop()

	// ### 2.- Setup mock storage and write an updates sparsely in bursts,
	// every 5 years 3 times starting in Jan 1st 1970 and then silence
	store := NewStore(DefaultStoreConfig)
	readFunc := store.MakeReadFunc()

	now := uint64(633799046)
	var epoch lookup.Epoch

	var lastData *Data
	for i := uint64(0); i < 3; i++ {
		for j := uint64(0); j < 10; j++ {
			T := uint64(Year*5*i + j) // write a burst of 10 updates every 5 years 3 times starting in Jan 1st 1970 and then silence
			data := Data{
				Payload: T, //our "payload" will be the timestamp itself.
				Time:    T,
			}
			epoch = store.Update(epoch, T, &data)
			lastData = &data
		}
	}

	// ### 3.- Test all algorithms
	for _, algo := range algorithms {
		t.Run(algo.Name, func(t *testing.T) {
			store.Reset() // reset read counters for next test

			// ### 3.1.- Test how long it takes to find the last update without a hint:
			timeElapsedWithoutHint := stopwatch.Measure(func() {
				value, err := algo.Lookup(context.Background(), now, lookup.NoClue, readFunc)
				stopwatch.Stop()
				if err != nil {
					t.Fatal(err)
				}

				if value != lastData {
					t.Fatalf("Expected lookup to return the last written value: %v. Got %v", lastData, value)
				}
			})
			printMetric("SIMPLE", store, timeElapsedWithoutHint)

			// reset the read count for the next test
			store.Reset()

			// ### 3.2.- Now test how long it takes to find the last update *with* a hint,
			// it should take less time!
			timeElapsed := stopwatch.Measure(func() {
				value, err := algo.Lookup(context.Background(), now, epoch, readFunc)
				if err != nil {
					t.Fatal(err)
				}

				if value != lastData {
					t.Fatalf("Expected lookup to return the last written value: %v. Got %v", lastData, value)
				}
			})
			if timeElapsed > timeElapsedWithoutHint {
				t.Fatalf("Expected lookup to complete faster than %s since we provided a hint. Took %s", timeElapsedWithoutHint, timeElapsed)
			}

			printMetric("WITH HINT", store, stopwatch.Elapsed())

		})
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
var testGetNextLevelCases = []testG{testG{e: lookup.Epoch{Time: 989875233, Level: 12}, n: 989807323, x: 24}, testG{e: lookup.Epoch{Time: 995807650, Level: 18}, n: 995807649, x: 17}, testG{e: lookup.Epoch{Time: 969167082, Level: 0}, n: 969111431, x: 18}, testG{e: lookup.Epoch{Time: 993087628, Level: 14}, n: 993087627, x: 13}, testG{e: lookup.Epoch{Time: 963364631, Level: 20}, n: 962941578, x: 19}, testG{e: lookup.Epoch{Time: 963497510, Level: 16}, n: 963497509, x: 15}, testG{e: lookup.Epoch{Time: 955421349, Level: 22}, n: 929292183, x: 27}, testG{e: lookup.Epoch{Time: 968220379, Level: 15}, n: 968220378, x: 14}, testG{e: lookup.Epoch{Time: 939129014, Level: 6}, n: 939126953, x: 11}, testG{e: lookup.Epoch{Time: 907847903, Level: 6}, n: 907846146, x: 11}, testG{e: lookup.Epoch{Time: 910835564, Level: 15}, n: 703619757, x: 28}, testG{e: lookup.Epoch{Time: 913578333, Level: 22}, n: 913578332, x: 21}, testG{e: lookup.Epoch{Time: 895818460, Level: 3}, n: 895818132, x: 9}, testG{e: lookup.Epoch{Time: 903843025, Level: 24}, n: 903843025, x: 23}, testG{e: lookup.Epoch{Time: 877889433, Level: 13}, n: 149120378, x: 29}, testG{e: lookup.Epoch{Time: 901450396, Level: 10}, n: 858997793, x: 26}, testG{e: lookup.Epoch{Time: 925179910, Level: 3}, n: 925177237, x: 13}, testG{e: lookup.Epoch{Time: 913485477, Level: 21}, n: 907146511, x: 22}, testG{e: lookup.Epoch{Time: 924462991, Level: 18}, n: 924462990, x: 17}, testG{e: lookup.Epoch{Time: 941175128, Level: 13}, n: 941168924, x: 13}, testG{e: lookup.Epoch{Time: 920126583, Level: 3}, n: 538054817, x: 28}, testG{e: lookup.Epoch{Time: 891721312, Level: 18}, n: 890975671, x: 21}, testG{e: lookup.Epoch{Time: 920397342, Level: 11}, n: 920396960, x: 10}, testG{e: lookup.Epoch{Time: 953406530, Level: 3}, n: 953406530, x: 2}, testG{e: lookup.Epoch{Time: 920024527, Level: 23}, n: 920024527, x: 22}, testG{e: lookup.Epoch{Time: 927050922, Level: 7}, n: 927049632, x: 11}, testG{e: lookup.Epoch{Time: 894599900, Level: 10}, n: 890021707, x: 22}, testG{e: lookup.Epoch{Time: 883010150, Level: 3}, n: 882969902, x: 15}, testG{e: lookup.Epoch{Time: 855561102, Level: 22}, n: 855561102, x: 21}, testG{e: lookup.Epoch{Time: 828245477, Level: 19}, n: 825245571, x: 22}, testG{e: lookup.Epoch{Time: 851095026, Level: 4}, n: 851083702, x: 13}, testG{e: lookup.Epoch{Time: 879209039, Level: 11}, n: 879209039, x: 10}, testG{e: lookup.Epoch{Time: 859265651, Level: 0}, n: 840582083, x: 24}, testG{e: lookup.Epoch{Time: 827349870, Level: 24}, n: 827349869, x: 23}, testG{e: lookup.Epoch{Time: 819602318, Level: 3}, n: 18446744073490860182, x: 31}, testG{e: lookup.Epoch{Time: 849708538, Level: 7}, n: 849708538, x: 6}, testG{e: lookup.Epoch{Time: 873885094, Level: 11}, n: 873881798, x: 11}, testG{e: lookup.Epoch{Time: 852169070, Level: 1}, n: 852049399, x: 17}, testG{e: lookup.Epoch{Time: 852885343, Level: 8}, n: 852875652, x: 13}, testG{e: lookup.Epoch{Time: 830957057, Level: 8}, n: 830955867, x: 10}, testG{e: lookup.Epoch{Time: 807353611, Level: 4}, n: 807325211, x: 16}, testG{e: lookup.Epoch{Time: 803198793, Level: 8}, n: 696477575, x: 26}, testG{e: lookup.Epoch{Time: 791356887, Level: 10}, n: 791356003, x: 10}, testG{e: lookup.Epoch{Time: 817771215, Level: 12}, n: 817708431, x: 17}, testG{e: lookup.Epoch{Time: 846211146, Level: 14}, n: 846211146, x: 13}, testG{e: lookup.Epoch{Time: 821849822, Level: 9}, n: 821849229, x: 9}, testG{e: lookup.Epoch{Time: 789508756, Level: 9}, n: 789508755, x: 8}, testG{e: lookup.Epoch{Time: 814088521, Level: 12}, n: 814088512, x: 11}, testG{e: lookup.Epoch{Time: 813665673, Level: 6}, n: 813548257, x: 17}, testG{e: lookup.Epoch{Time: 791472209, Level: 6}, n: 720857845, x: 26}, testG{e: lookup.Epoch{Time: 805687744, Level: 2}, n: 805687720, x: 6}, testG{e: lookup.Epoch{Time: 783153927, Level: 12}, n: 783134053, x: 14}, testG{e: lookup.Epoch{Time: 815033655, Level: 11}, n: 815033654, x: 10}, testG{e: lookup.Epoch{Time: 821184581, Level: 6}, n: 821184464, x: 11}, testG{e: lookup.Epoch{Time: 841908114, Level: 2}, n: 841636025, x: 18}, testG{e: lookup.Epoch{Time: 862969167, Level: 20}, n: 862919955, x: 19}, testG{e: lookup.Epoch{Time: 887604565, Level: 21}, n: 887604564, x: 20}, testG{e: lookup.Epoch{Time: 863723789, Level: 10}, n: 858274530, x: 22}, testG{e: lookup.Epoch{Time: 851533290, Level: 10}, n: 851531385, x: 11}, testG{e: lookup.Epoch{Time: 826032484, Level: 14}, n: 826032484, x: 13}, testG{e: lookup.Epoch{Time: 819401505, Level: 7}, n: 818943526, x: 18}, testG{e: lookup.Epoch{Time: 800886832, Level: 12}, n: 800563106, x: 19}, testG{e: lookup.Epoch{Time: 780767476, Level: 10}, n: 694450997, x: 26}, testG{e: lookup.Epoch{Time: 789209418, Level: 15}, n: 789209417, x: 14}, testG{e: lookup.Epoch{Time: 816086666, Level: 9}, n: 816034646, x: 18}, testG{e: lookup.Epoch{Time: 835407077, Level: 21}, n: 835407076, x: 20}, testG{e: lookup.Epoch{Time: 846527322, Level: 20}, n: 846527321, x: 19}, testG{e: lookup.Epoch{Time: 850131130, Level: 19}, n: 18446744073670013406, x: 31}, testG{e: lookup.Epoch{Time: 842248607, Level: 24}, n: 783963834, x: 28}, testG{e: lookup.Epoch{Time: 816181999, Level: 2}, n: 816124867, x: 15}, testG{e: lookup.Epoch{Time: 806627026, Level: 17}, n: 756013427, x: 28}, testG{e: lookup.Epoch{Time: 826223084, Level: 4}, n: 826169865, x: 16}, testG{e: lookup.Epoch{Time: 835380147, Level: 21}, n: 835380147, x: 20}, testG{e: lookup.Epoch{Time: 860137874, Level: 3}, n: 860137782, x: 7}, testG{e: lookup.Epoch{Time: 860623757, Level: 8}, n: 860621582, x: 12}, testG{e: lookup.Epoch{Time: 875464114, Level: 24}, n: 875464114, x: 23}, testG{e: lookup.Epoch{Time: 853804052, Level: 6}, n: 853804051, x: 5}, testG{e: lookup.Epoch{Time: 864150903, Level: 14}, n: 854360673, x: 24}, testG{e: lookup.Epoch{Time: 850104561, Level: 23}, n: 850104561, x: 22}, testG{e: lookup.Epoch{Time: 878020186, Level: 24}, n: 878020186, x: 23}, testG{e: lookup.Epoch{Time: 900150940, Level: 8}, n: 899224760, x: 21}, testG{e: lookup.Epoch{Time: 869566202, Level: 2}, n: 869566199, x: 3}, testG{e: lookup.Epoch{Time: 851878045, Level: 5}, n: 851878045, x: 4}, testG{e: lookup.Epoch{Time: 824469671, Level: 12}, n: 824466504, x: 13}, testG{e: lookup.Epoch{Time: 819830223, Level: 9}, n: 816550241, x: 22}, testG{e: lookup.Epoch{Time: 813720249, Level: 20}, n: 801351581, x: 28}, testG{e: lookup.Epoch{Time: 831200185, Level: 20}, n: 830760165, x: 19}, testG{e: lookup.Epoch{Time: 838915973, Level: 9}, n: 838915972, x: 8}, testG{e: lookup.Epoch{Time: 812902644, Level: 5}, n: 812902644, x: 4}, testG{e: lookup.Epoch{Time: 812755887, Level: 3}, n: 812755887, x: 2}, testG{e: lookup.Epoch{Time: 822497779, Level: 8}, n: 822486000, x: 14}, testG{e: lookup.Epoch{Time: 832407585, Level: 9}, n: 579450238, x: 28}, testG{e: lookup.Epoch{Time: 799645403, Level: 23}, n: 799645403, x: 22}, testG{e: lookup.Epoch{Time: 827279665, Level: 2}, n: 826723872, x: 19}, testG{e: lookup.Epoch{Time: 846062554, Level: 6}, n: 765881119, x: 28}, testG{e: lookup.Epoch{Time: 855122998, Level: 6}, n: 855122978, x: 5}, testG{e: lookup.Epoch{Time: 841905104, Level: 4}, n: 751401236, x: 28}, testG{e: lookup.Epoch{Time: 857737438, Level: 12}, n: 325468127, x: 29}, testG{e: lookup.Epoch{Time: 838103691, Level: 18}, n: 779030823, x: 28}, testG{e: lookup.Epoch{Time: 841581240, Level: 22}, n: 841581239, x: 21}}

// TestGetNextLevel tests the lookup.GetNextLevel function
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

// CookGetNextLevelTests is used to generate a deterministic
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
