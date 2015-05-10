// CookieJar - A contestant's algorithm toolbox
// Copyright (c) 2013 Peter Szilagyi. All rights reserved.
//
// CookieJar is dual licensed: you can redistribute it and/or modify it under
// the terms of the GNU General Public License as published by the Free Software
// Foundation, either version 3 of the License, or (at your option) any later
// version.
//
// The toolbox is distributed in the hope that it will be useful, but WITHOUT
// ANY WARRANTY; without even the implied warranty of MERCHANTABILITY or
// FITNESS FOR A PARTICULAR PURPOSE.  See the GNU General Public License for
// more details.
//
// Alternatively, the CookieJar toolbox may be used in accordance with the terms
// and conditions contained in a signed written agreement between you and the
// author(s).

package prque_test

import (
	"fmt"

	"gopkg.in/karalabe/cookiejar.v2/collections/prque"
)

// Insert some data into a priority queue and pop them out in prioritized order.
func Example_usage() {
	// Define some data to push into the priority queue
	prio := []float32{77.7, 22.2, 44.4, 55.5, 11.1, 88.8, 33.3, 99.9, 0.0, 66.6}
	data := []string{"zero", "one", "two", "three", "four", "five", "six", "seven", "eight", "nine"}

	// Create the priority queue and insert the prioritized data
	pq := prque.New()
	for i := 0; i < len(data); i++ {
		pq.Push(data[i], prio[i])
	}
	// Pop out the data and print them
	for !pq.Empty() {
		val, prio := pq.Pop()
		fmt.Printf("%.1f:%s ", prio, val)
	}
	// Output:
	// 99.9:seven 88.8:five 77.7:zero 66.6:nine 55.5:three 44.4:two 33.3:six 22.2:one 11.1:four 0.0:eight
}
