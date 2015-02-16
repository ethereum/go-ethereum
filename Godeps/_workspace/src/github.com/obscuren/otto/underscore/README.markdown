# underscore
--
    import "github.com/robertkrimen/otto/underscore"

Package underscore contains the source for the JavaScript utility-belt library.

    import (
    	_ "github.com/robertkrimen/otto/underscore"
    )
    // Every Otto runtime will now include underscore

http://underscorejs.org

https://github.com/documentcloud/underscore

By importing this package, you'll automatically load underscore every time you
create a new Otto runtime.

To prevent this behavior, you can do the following:

    import (
    	"github.com/robertkrimen/otto/underscore"
    )

    func init() {
    	underscore.Disable()
    }

## Usage

#### func  Disable

```go
func Disable()
```
Disable underscore runtime inclusion.

#### func  Enable

```go
func Enable()
```
Enable underscore runtime inclusion.

#### func  Source

```go
func Source() string
```
Source returns the underscore source.

--
**godocdown** http://github.com/robertkrimen/godocdown
