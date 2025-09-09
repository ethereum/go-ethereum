// Copyright 2025 the libevm authors.
//
// The libevm additions to go-ethereum are free software: you can redistribute
// them and/or modify them under the terms of the GNU Lesser General Public License
// as published by the Free Software Foundation, either version 3 of the License,
// or (at your option) any later version.
//
// The libevm additions are distributed in the hope that they will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU Lesser
// General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see
// <http://www.gnu.org/licenses/>.

package options_test

import (
	"fmt"

	"github.com/ava-labs/libevm/libevm/options"
)

// config is an arbitrary type to be configured with [options.Option] values.
// Although it can be exported, there is typically no need.
type config struct {
	num  int
	flag bool
}

// An Option configures an arbitrary type. Using a type alias (=) instead of a
// completely new type is recommended as it maintains compatibility with helpers
// such as [options.Func].
type Option = options.Option[config]

func WithNum(n int) Option {
	return options.Func[config](func(c *config) {
		c.num = n
	})
}

func WithFlag(b bool) Option {
	return options.Func[config](func(c *config) {
		c.flag = b
	})
}

func Example() {
	num42 := WithNum(42)
	flagOn := WithFlag(true)

	// Some IDEs and linters might complain about the redundant type parameter
	// as it can be inferred, but it makes for more readable code.
	fromZero := options.As[config](num42, flagOn)
	fmt.Printf("From zero value: %T(%+[1]v)\n", fromZero)

	fromDefault := options.ApplyTo(&config{
		num: 100,
	}, flagOn)
	fmt.Printf("Applied to default value: %T(%+[1]v)\n", fromDefault)

	// Output:
	// From zero value: *options_test.config(&{num:42 flag:true})
	// Applied to default value: *options_test.config(&{num:100 flag:true})
}
