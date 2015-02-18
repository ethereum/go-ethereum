package otto

import (
	"fmt"
	"os"
	"strings"
)

func formatForConsole(argumentList []Value) string {
	output := []string{}
	for _, argument := range argumentList {
		output = append(output, fmt.Sprintf("%v", argument))
	}
	return strings.Join(output, " ")
}

func builtinConsole_log(call FunctionCall) Value {
	fmt.Fprintln(os.Stdout, formatForConsole(call.ArgumentList))
	return UndefinedValue()
}

func builtinConsole_error(call FunctionCall) Value {
	fmt.Fprintln(os.Stdout, formatForConsole(call.ArgumentList))
	return UndefinedValue()
}

// Nothing happens.
func builtinConsole_dir(call FunctionCall) Value {
	return UndefinedValue()
}

func builtinConsole_time(call FunctionCall) Value {
	return UndefinedValue()
}

func builtinConsole_timeEnd(call FunctionCall) Value {
	return UndefinedValue()
}

func builtinConsole_trace(call FunctionCall) Value {
	return UndefinedValue()
}

func builtinConsole_assert(call FunctionCall) Value {
	return UndefinedValue()
}

func (runtime *_runtime) newConsole() *_object {

	return newConsoleObject(runtime)
}
