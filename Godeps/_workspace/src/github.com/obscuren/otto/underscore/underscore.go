/*
Package underscore contains the source for the JavaScript utility-belt library.

	import (
		_ "github.com/robertkrimen/otto/underscore"
	)
	// Every Otto runtime will now include underscore

http://underscorejs.org

https://github.com/documentcloud/underscore

By importing this package, you'll automatically load underscore every time you create a new Otto runtime.

To prevent this behavior, you can do the following:

	import (
		"github.com/robertkrimen/otto/underscore"
	)

	func init() {
		underscore.Disable()
	}

*/
package underscore

import (
	"github.com/robertkrimen/otto/registry"
)

var entry *registry.Entry = registry.Register(func() string {
	return Source()
})

// Enable underscore runtime inclusion.
func Enable() {
	entry.Enable()
}

// Disable underscore runtime inclusion.
func Disable() {
	entry.Disable()
}

// Source returns the underscore source.
func Source() string {
	return string(underscore())
}
