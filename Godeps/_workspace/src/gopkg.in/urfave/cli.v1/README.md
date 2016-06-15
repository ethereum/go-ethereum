[![Coverage](http://gocover.io/_badge/github.com/codegangsta/cli?0)](http://gocover.io/github.com/codegangsta/cli)
[![Build Status](https://travis-ci.org/codegangsta/cli.svg?branch=master)](https://travis-ci.org/codegangsta/cli)
[![GoDoc](https://godoc.org/github.com/codegangsta/cli?status.svg)](https://godoc.org/github.com/codegangsta/cli)
[![codebeat](https://codebeat.co/badges/0a8f30aa-f975-404b-b878-5fab3ae1cc5f)](https://codebeat.co/projects/github-com-codegangsta-cli)
[![Go Report Card](https://goreportcard.com/badge/codegangsta/cli)](https://goreportcard.com/report/codegangsta/cli)

# cli

cli is a simple, fast, and fun package for building command line apps in Go. The goal is to enable developers to write fast and distributable command line applications in an expressive way.

## Overview

Command line apps are usually so tiny that there is absolutely no reason why your code should *not* be self-documenting. Things like generating help text and parsing command flags/options should not hinder productivity when writing a command line app.

**This is where cli comes into play.** cli makes command line programming fun, organized, and expressive!

## Installation

Make sure you have a working Go environment (go 1.1+ is *required*). [See the install instructions](http://golang.org/doc/install.html).

To install cli, simply run:
```
$ go get github.com/codegangsta/cli
```

Make sure your `PATH` includes to the `$GOPATH/bin` directory so your commands can be easily used:
```
export PATH=$PATH:$GOPATH/bin
```

## Getting Started

One of the philosophies behind cli is that an API should be playful and full of discovery. So a cli app can be as little as one line of code in `main()`.

``` go
package main

import (
  "os"
  "github.com/codegangsta/cli"
)

func main() {
  cli.NewApp().Run(os.Args)
}
```

This app will run and show help text, but is not very useful. Let's give an action to execute and some help documentation:

<!-- {
  "output": "boom! I say!"
} -->
``` go
package main

import (
  "fmt"
  "os"

  "github.com/codegangsta/cli"
)

func main() {
  app := cli.NewApp()
  app.Name = "boom"
  app.Usage = "make an explosive entrance"
  app.Action = func(c *cli.Context) error {
    fmt.Println("boom! I say!")
    return nil
  }

  app.Run(os.Args)
}
```

Running this already gives you a ton of functionality, plus support for things like subcommands and flags, which are covered below.

## Example

Being a programmer can be a lonely job. Thankfully by the power of automation that is not the case! Let's create a greeter app to fend off our demons of loneliness!

Start by creating a directory named `greet`, and within it, add a file, `greet.go` with the following code in it:

<!-- {
  "output": "Hello friend!"
} -->
``` go
package main

import (
  "fmt"
  "os"

  "github.com/codegangsta/cli"
)

func main() {
  app := cli.NewApp()
  app.Name = "greet"
  app.Usage = "fight the loneliness!"
  app.Action = func(c *cli.Context) error {
    fmt.Println("Hello friend!")
    return nil
  }

  app.Run(os.Args)
}
```

Install our command to the `$GOPATH/bin` directory:

```
$ go install
```

Finally run our new command:

```
$ greet
Hello friend!
```

cli also generates neat help text:

```
$ greet help
NAME:
    greet - fight the loneliness!

USAGE:
    greet [global options] command [command options] [arguments...]

VERSION:
    0.0.0

COMMANDS:
    help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS
    --version	Shows version information
```

### Arguments

You can lookup arguments by calling the `Args` function on `cli.Context`.

``` go
...
app.Action = func(c *cli.Context) error {
  fmt.Println("Hello", c.Args()[0])
  return nil
}
...
```

### Flags

Setting and querying flags is simple.

``` go
...
app.Flags = []cli.Flag {
  cli.StringFlag{
    Name: "lang",
    Value: "english",
    Usage: "language for the greeting",
  },
}
app.Action = func(c *cli.Context) error {
  name := "someone"
  if c.NArg() > 0 {
    name = c.Args()[0]
  }
  if c.String("lang") == "spanish" {
    fmt.Println("Hola", name)
  } else {
    fmt.Println("Hello", name)
  }
  return nil
}
...
```

You can also set a destination variable for a flag, to which the content will be scanned.

``` go
...
var language string
app.Flags = []cli.Flag {
  cli.StringFlag{
    Name:        "lang",
    Value:       "english",
    Usage:       "language for the greeting",
    Destination: &language,
  },
}
app.Action = func(c *cli.Context) error {
  name := "someone"
  if c.NArg() > 0 {
    name = c.Args()[0]
  }
  if language == "spanish" {
    fmt.Println("Hola", name)
  } else {
    fmt.Println("Hello", name)
  }
  return nil
}
...
```

See full list of flags at http://godoc.org/github.com/codegangsta/cli

#### Placeholder Values

Sometimes it's useful to specify a flag's value within the usage string itself. Such placeholders are
indicated with back quotes.

For example this:

```go
cli.StringFlag{
  Name:  "config, c",
  Usage: "Load configuration from `FILE`",
}
```

Will result in help output like:

```
--config FILE, -c FILE   Load configuration from FILE
```

Note that only the first placeholder is used. Subsequent back-quoted words will be left as-is.

#### Alternate Names

You can set alternate (or short) names for flags by providing a comma-delimited list for the `Name`. e.g.

``` go
app.Flags = []cli.Flag {
  cli.StringFlag{
    Name: "lang, l",
    Value: "english",
    Usage: "language for the greeting",
  },
}
```

That flag can then be set with `--lang spanish` or `-l spanish`. Note that giving two different forms of the same flag in the same command invocation is an error.

#### Values from the Environment

You can also have the default value set from the environment via `EnvVar`.  e.g.

``` go
app.Flags = []cli.Flag {
  cli.StringFlag{
    Name: "lang, l",
    Value: "english",
    Usage: "language for the greeting",
    EnvVar: "APP_LANG",
  },
}
```

The `EnvVar` may also be given as a comma-delimited "cascade", where the first environment variable that resolves is used as the default.

``` go
app.Flags = []cli.Flag {
  cli.StringFlag{
    Name: "lang, l",
    Value: "english",
    Usage: "language for the greeting",
    EnvVar: "LEGACY_COMPAT_LANG,APP_LANG,LANG",
  },
}
```

#### Values from alternate input sources (YAML and others)

There is a separate package altsrc that adds support for getting flag values from other input sources like YAML.

In order to get values for a flag from an alternate input source the following code would be added to wrap an existing cli.Flag like below:

``` go
  altsrc.NewIntFlag(cli.IntFlag{Name: "test"})
```

Initialization must also occur for these flags. Below is an example initializing getting data from a yaml file below.

``` go
  command.Before = altsrc.InitInputSourceWithContext(command.Flags, NewYamlSourceFromFlagFunc("load"))
```

The code above will use the "load" string as a flag name to get the file name of a yaml file from the cli.Context.
It will then use that file name to initialize the yaml input source for any flags that are defined on that command.
As a note the "load" flag used would also have to be defined on the command flags in order for this code snipped to work.

Currently only YAML files are supported but developers can add support for other input sources by implementing the
altsrc.InputSourceContext for their given sources.

Here is a more complete sample of a command using YAML support:

``` go
  command := &cli.Command{
    Name:        "test-cmd",
    Aliases:     []string{"tc"},
    Usage:       "this is for testing",
    Description: "testing",
    Action: func(c *cli.Context) error {
      // Action to run
      return nil
    },
    Flags: []cli.Flag{
      NewIntFlag(cli.IntFlag{Name: "test"}),
      cli.StringFlag{Name: "load"}},
  }
  command.Before = InitInputSourceWithContext(command.Flags, NewYamlSourceFromFlagFunc("load"))
  err := command.Run(c)
```

### Subcommands

Subcommands can be defined for a more git-like command line app.

```go
...
app.Commands = []cli.Command{
  {
    Name:      "add",
    Aliases:     []string{"a"},
    Usage:     "add a task to the list",
    Action: func(c *cli.Context) error {
      fmt.Println("added task: ", c.Args().First())
      return nil
    },
  },
  {
    Name:      "complete",
    Aliases:     []string{"c"},
    Usage:     "complete a task on the list",
    Action: func(c *cli.Context) error {
      fmt.Println("completed task: ", c.Args().First())
      return nil
    },
  },
  {
    Name:      "template",
    Aliases:     []string{"r"},
    Usage:     "options for task templates",
    Subcommands: []cli.Command{
      {
        Name:  "add",
        Usage: "add a new template",
        Action: func(c *cli.Context) error {
          fmt.Println("new task template: ", c.Args().First())
          return nil
        },
      },
      {
        Name:  "remove",
        Usage: "remove an existing template",
        Action: func(c *cli.Context) error {
          fmt.Println("removed task template: ", c.Args().First())
          return nil
        },
      },
    },
  },
}
...
```

### Subcommands categories

For additional organization in apps that have many subcommands, you can
associate a category for each command to group them together in the help
output.

E.g.

```go
...
  app.Commands = []cli.Command{
    {
      Name: "noop",
    },
    {
      Name:     "add",
      Category: "template",
    },
    {
      Name:     "remove",
      Category: "template",
    },
  }
...
```

Will include:

```
...
COMMANDS:
    noop

  Template actions:
    add
    remove
...
```

### Exit code

Calling `App.Run` will not automatically call `os.Exit`, which means that by
default the exit code will "fall through" to being `0`.  An explicit exit code
may be set by returning a non-nil error that fulfills `cli.ExitCoder`, *or* a
`cli.MultiError` that includes an error that fulfills `cli.ExitCoder`, e.g.:

``` go
package main

import (
  "os"

  "github.com/codegangsta/cli"
)

func main() {
  app := cli.NewApp()
  app.Flags = []cli.Flag{
    cli.BoolTFlag{
      Name:  "ginger-crouton",
      Usage: "is it in the soup?",
    },
  }
  app.Action = func(ctx *cli.Context) error {
    if !ctx.Bool("ginger-crouton") {
      return cli.NewExitError("it is not in the soup", 86)
    }
    return nil
  }

  app.Run(os.Args)
}
```

### Bash Completion

You can enable completion commands by setting the `EnableBashCompletion`
flag on the `App` object.  By default, this setting will only auto-complete to
show an app's subcommands, but you can write your own completion methods for
the App or its subcommands.

```go
...
var tasks = []string{"cook", "clean", "laundry", "eat", "sleep", "code"}
app := cli.NewApp()
app.EnableBashCompletion = true
app.Commands = []cli.Command{
  {
    Name:  "complete",
    Aliases: []string{"c"},
    Usage: "complete a task on the list",
    Action: func(c *cli.Context) error {
       fmt.Println("completed task: ", c.Args().First())
       return nil
    },
    BashComplete: func(c *cli.Context) {
      // This will complete if no args are passed
      if c.NArg() > 0 {
        return
      }
      for _, t := range tasks {
        fmt.Println(t)
      }
    },
  }
}
...
```

#### To Enable

Source the `autocomplete/bash_autocomplete` file in your `.bashrc` file while
setting the `PROG` variable to the name of your program:

`PROG=myprogram source /.../cli/autocomplete/bash_autocomplete`

#### To Distribute

Copy `autocomplete/bash_autocomplete` into `/etc/bash_completion.d/` and rename
it to the name of the program you wish to add autocomplete support for (or
automatically install it there if you are distributing a package). Don't forget
to source the file to make it active in the current shell.

```
sudo cp src/bash_autocomplete /etc/bash_completion.d/<myprogram>
source /etc/bash_completion.d/<myprogram>
```

Alternatively, you can just document that users should source the generic
`autocomplete/bash_autocomplete` in their bash configuration with `$PROG` set
to the name of their program (as above).

### Generated Help Text Customization

All of the help text generation may be customized, and at multiple levels.  The
templates are exposed as variables `AppHelpTemplate`, `CommandHelpTemplate`, and
`SubcommandHelpTemplate` which may be reassigned or augmented, and full override
is possible by assigning a compatible func to the `cli.HelpPrinter` variable,
e.g.:

<!-- {
  "output": "Ha HA.  I pwnd the help!!1"
} -->
``` go
package main

import (
  "fmt"
  "io"
  "os"

  "github.com/codegangsta/cli"
)

func main() {
  // EXAMPLE: Append to an existing template
  cli.AppHelpTemplate = fmt.Sprintf(`%s

WEBSITE: http://awesometown.example.com

SUPPORT: support@awesometown.example.com

`, cli.AppHelpTemplate)

  // EXAMPLE: Override a template
  cli.AppHelpTemplate = `NAME:
   {{.Name}} - {{.Usage}}
USAGE:
   {{.HelpName}} {{if .VisibleFlags}}[global options]{{end}}{{if .Commands}} command
[command options]{{end}} {{if
.ArgsUsage}}{{.ArgsUsage}}{{else}}[arguments...]{{end}}
   {{if len .Authors}}
AUTHOR(S):
   {{range .Authors}}{{ . }}{{end}}
   {{end}}{{if .Commands}}
COMMANDS:
{{range .Commands}}{{if not .HideHelp}}   {{join .Names ", "}}{{ "\t"
}}{{.Usage}}{{ "\n" }}{{end}}{{end}}{{end}}{{if .VisibleFlags}}
GLOBAL OPTIONS:
   {{range .VisibleFlags}}{{.}}
   {{end}}{{end}}{{if .Copyright }}
COPYRIGHT:
   {{.Copyright}}
   {{end}}{{if .Version}}
VERSION:
   {{.Version}}
   {{end}}
`

  // EXAMPLE: Replace the `HelpPrinter` func
  cli.HelpPrinter = func(w io.Writer, templ string, data interface{}) {
    fmt.Println("Ha HA.  I pwnd the help!!1")
  }

  cli.NewApp().Run(os.Args)
}
```

## Contribution Guidelines

Feel free to put up a pull request to fix a bug or maybe add a feature. I will give it a code review and make sure that it does not break backwards compatibility. If I or any other collaborators agree that it is in line with the vision of the project, we will work with you to get the code into a mergeable state and merge it into the master branch.

If you have contributed something significant to the project, I will most likely add you as a collaborator. As a collaborator you are given the ability to merge others pull requests. It is very important that new code does not break existing code, so be careful about what code you do choose to merge. If you have any questions feel free to link @codegangsta to the issue in question and we can review it together.

If you feel like you have contributed to the project but have not yet been added as a collaborator, I probably forgot to add you. Hit @codegangsta up over email and we will get it figured out.
