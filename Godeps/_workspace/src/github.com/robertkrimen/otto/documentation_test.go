package otto

import (
	"fmt"
)

func ExampleSynopsis() {

	vm := New()
	vm.Run(`
        abc = 2 + 2;
        console.log("The value of abc is " + abc); // 4
    `)

	value, _ := vm.Get("abc")
	{
		value, _ := value.ToInteger()
		fmt.Println(value)
	}

	vm.Set("def", 11)
	vm.Run(`
        console.log("The value of def is " + def);
    `)

	vm.Set("xyzzy", "Nothing happens.")
	vm.Run(`
        console.log(xyzzy.length);
    `)

	value, _ = vm.Run("xyzzy.length")
	{
		value, _ := value.ToInteger()
		fmt.Println(value)
	}

	value, err := vm.Run("abcdefghijlmnopqrstuvwxyz.length")
	fmt.Println(value)
	fmt.Println(err)

	vm.Set("sayHello", func(call FunctionCall) Value {
		fmt.Printf("Hello, %s.\n", call.Argument(0).String())
		return UndefinedValue()
	})

	vm.Set("twoPlus", func(call FunctionCall) Value {
		right, _ := call.Argument(0).ToInteger()
		result, _ := vm.ToValue(2 + right)
		return result
	})

	value, _ = vm.Run(`
        sayHello("Xyzzy");
        sayHello();

        result = twoPlus(2.0);
    `)
	fmt.Println(value)

	// Output:
	// The value of abc is 4
	// 4
	// The value of def is 11
	// 16
	// 16
	// undefined
	// ReferenceError: 'abcdefghijlmnopqrstuvwxyz' is not defined
	// Hello, Xyzzy.
	// Hello, undefined.
	// 4
}

func ExampleConsole() {

	vm := New()
	console := map[string]interface{}{
		"log": func(call FunctionCall) Value {
			fmt.Println("console.log:", formatForConsole(call.ArgumentList))
			return UndefinedValue()
		},
	}

	err := vm.Set("console", console)

	value, err := vm.Run(`
        console.log("Hello, World.");
    `)
	fmt.Println(value)
	fmt.Println(err)

	// Output:
	// console.log: Hello, World.
	// undefined
	// <nil>
}
