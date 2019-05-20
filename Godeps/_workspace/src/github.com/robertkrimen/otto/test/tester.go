package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"text/tabwriter"

	"github.com/robertkrimen/otto"
	"github.com/robertkrimen/otto/parser"
)

var flag_test *bool = flag.Bool("test", false, "")
var flag_report *bool = flag.Bool("report", false, "")

var match_ReferenceError_not_defined = regexp.MustCompile(`^ReferenceError: \S+ is not defined$`)
var match_lookahead = regexp.MustCompile(`Invalid regular expression: re2: Invalid \(\?[=!]\) <lookahead>`)
var match_backreference = regexp.MustCompile(`Invalid regular expression: re2: Invalid \\\d <backreference>`)
var match_TypeError_undefined = regexp.MustCompile(`^TypeError: Cannot access member '[^']+' of undefined$`)

var target = map[string]string{
	"test-angular-bindonce.js": "fail",  // (anonymous): Line 1:944 Unexpected token ( (and 40 more errors)
	"test-jsforce.js":          "fail",  // (anonymous): Line 9:28329 RuneError (and 5 more errors)
	"test-chaplin.js":          "parse", // Error: Chaplin requires Common.js or AMD modules
	"test-dropbox.js.js":       "parse", // Error: dropbox.js loaded in an unsupported JavaScript environment.
	"test-epitome.js":          "parse", // TypeError: undefined is not a function
	"test-portal.js":           "parse", // TypeError
	"test-reactive-coffee.js":  "parse", // Dependencies are not met for reactive: _ and $ not found
	"test-scriptaculous.js":    "parse", // script.aculo.us requires the Prototype JavaScript framework >= 1.6.0.3
	"test-waypoints.js":        "parse", // TypeError: undefined is not a function
	"test-webuploader.js":      "parse", // Error: `jQuery` is undefined
	"test-xuijs.js":            "parse", // TypeError: undefined is not a function
}

// http://cdnjs.com/
// http://api.cdnjs.com/libraries

func fetch(name, location string) error {
	response, err := http.Get(location)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}

	if !strings.HasSuffix(location, ".js") {
		return nil
	}

	filename := "test-" + name + ".js"
	fmt.Println(filename, len(body))
	return ioutil.WriteFile(filename, body, 0644)
}

func test(filename string) error {
	script, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}

	if !*flag_report {
		fmt.Fprintln(os.Stdout, filename, len(script))
	}

	parse := false
	option := target[filename]

	if option != "parse" {
		vm := otto.New()
		_, err = vm.Run(string(script))
		if err != nil {
			value := err.Error()
			switch {
			case match_ReferenceError_not_defined.MatchString(value):
			case match_TypeError_undefined.MatchString(value):
			case match_lookahead.MatchString(value):
			case match_backreference.MatchString(value):
			default:
				return err
			}
			parse = true
		}
	}

	if parse {
		_, err = parser.ParseFile(nil, filename, string(script), parser.IgnoreRegExpErrors)
		if err != nil {
			return err
		}
		target[filename] = "parse"
	}

	return nil
}

func main() {
	flag.Parse()

	filename := ""

	err := func() error {

		if flag.Arg(0) == "fetch" {
			response, err := http.Get("http://api.cdnjs.com/libraries")
			if err != nil {
				return err
			}
			defer response.Body.Close()
			body, err := ioutil.ReadAll(response.Body)
			if err != nil {
				return err
			}

			var tmp map[string]interface{}

			err = json.Unmarshal(body, &tmp)
			if err != nil {
				return err
			}

			var wg sync.WaitGroup

			for _, value := range tmp["results"].([]interface{}) {
				wg.Add(1)
				library := value.(map[string]interface{})
				go func() {
					defer wg.Done()
					fetch(library["name"].(string), library["latest"].(string))
				}()
			}

			wg.Wait()

			return nil
		}

		if *flag_report {
			files, err := ioutil.ReadDir(".")
			if err != nil {
				return err
			}
			writer := tabwriter.NewWriter(os.Stdout, 0, 8, 0, '\t', 0)
			fmt.Fprintln(writer, "", "\t| Status")
			fmt.Fprintln(writer, "---", "\t| ---")
			for _, file := range files {
				filename := file.Name()
				if !strings.HasPrefix(filename, "test-") {
					continue
				}
				err := test(filename)
				option := target[filename]
				name := strings.TrimPrefix(strings.TrimSuffix(filename, ".js"), "test-")
				if err == nil {
					switch option {
					case "":
						fmt.Fprintln(writer, name, "\t| pass")
					case "parse":
						fmt.Fprintln(writer, name, "\t| pass (parse)")
					case "re2":
						continue
						fmt.Fprintln(writer, name, "\t| unknown (re2)")
					}
				} else {
					fmt.Fprintln(writer, name, "\t| fail")
				}
			}
			writer.Flush()
			return nil
		}

		filename = flag.Arg(0)
		return test(filename)

	}()
	if err != nil {
		if filename != "" {
			if *flag_test && target[filename] == "fail" {
				goto exit
			}
			fmt.Fprintf(os.Stderr, "%s: %s\n", filename, err.Error())
		} else {
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(64)
	}
exit:
}
