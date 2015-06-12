/*
Package cl provides a binding to the OpenCL api. It's mostly a low-level
wrapper that avoids adding functionality while still making the interface
a little more friendly and easy to use.

Resource life-cycle management:

For any CL object that gets created (buffer, queue, kernel, etc..) you should
call object.Release() when finished with it to free the CL resources. This
explicitely calls the related clXXXRelease method for the type. However,
as a fallback there is a finalizer set for every resource item that takes
care of it (eventually) if Release isn't called. In this way you can have
better control over the life cycle of resources while having a fall back
to avoid leaks. This is similar to how file handles and such are handled
in the Go standard packages.
*/
package cl

// #include "headers/1.2/opencl.h"
// #cgo CFLAGS: -Iheaders/1.2
// #cgo darwin LDFLAGS: -framework OpenCL
// #cgo linux LDFLAGS: -lOpenCL
import "C"
import "errors"

var ErrUnsupported = errors.New("cl: unsupported")
