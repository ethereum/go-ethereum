package otto

import (
	"testing"
)

func TestScript(t *testing.T) {
	tt(t, func() {
		vm := New()

		script, err := vm.Compile("xyzzy", `var abc; if (!abc) abc = 0; abc += 2; abc;`)
		is(err, nil)

		str := script.String()
		is(str, "// xyzzy\nvar abc; if (!abc) abc = 0; abc += 2; abc;")

		value, err := vm.Run(script)
		is(err, nil)
		is(value, 2)

		if true {
			return
		}

		tmp, err := script.marshalBinary()
		is(err, nil)
		is(len(tmp), 1228)

		{
			script := &Script{}
			err = script.unmarshalBinary(tmp)
			is(err, nil)

			is(script.String(), str)

			value, err = vm.Run(script)
			is(err, nil)
			is(value, 4)

			tmp, err = script.marshalBinary()
			is(err, nil)
			is(len(tmp), 1228)
		}

		{
			script := &Script{}
			err = script.unmarshalBinary(tmp)
			is(err, nil)

			is(script.String(), str)

			value, err := vm.Run(script)
			is(err, nil)
			is(value, 6)

			tmp, err = script.marshalBinary()
			is(err, nil)
			is(len(tmp), 1228)
		}

		{
			version := scriptVersion
			scriptVersion = "bogus"

			script := &Script{}
			err = script.unmarshalBinary(tmp)
			is(err, "version mismatch")

			is(script.String(), "// \n")
			is(script.version, "")
			is(script.program == nil, true)
			is(script.filename, "")
			is(script.src, "")

			scriptVersion = version
		}
	})
}
