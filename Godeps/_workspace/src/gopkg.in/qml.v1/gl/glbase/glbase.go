package glbase

// A Context represents an OpenGL context that may be rendered on by the
// version-specific APIs under this package.
type Context struct {
	// This is just a marker at the moment, as the GL.API functions will
	// initialize their GL context from the current context in the
	// renderer thread. The design supports proper expansion and fixes for
	// upstream changes that break that model, though.
	private struct{}
}

// Contexter is implemented by values that have an assigned OpenGL context.
type Contexter interface {
	GLContext() *Context
}

type (
	Bitfield uint32
	Enum     uint32
	Sync     uintptr
	Clampf   float32
	Clampd   float64

	Uniform      int32
	Attrib       int32
	Texture      uint32
	Program      uint32
	Shader       uint32
	Buffer       uint32
	Framebuffer  uint32
	Renderbuffer uint32
)
