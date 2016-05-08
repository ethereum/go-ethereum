package cli

import (
	"fmt"
	"io/ioutil"
	"strings"
)

// Command is a subcommand for a cli.App.
type Command struct {
	// The name of the command
	Name string
	// short name of the command. Typically one character (deprecated, use `Aliases`)
	ShortName string
	// A list of aliases for the command
	Aliases []string
	// A short description of the usage of this command
	Usage string
	// Custom text to show on USAGE section of help
	UsageText string
	// A longer explanation of how the command works
	Description string
	// A short description of the arguments of this command
	ArgsUsage string
	// The function to call when checking for bash command completions
	BashComplete func(context *Context)
	// An action to execute before any sub-subcommands are run, but after the context is ready
	// If a non-nil error is returned, no sub-subcommands are run
	Before func(context *Context) error
	// An action to execute after any subcommands are run, but after the subcommand has finished
	// It is run even if Action() panics
	After func(context *Context) error
	// The function to call when this command is invoked
	Action func(context *Context)
	// Execute this function, if an usage error occurs. This is useful for displaying customized usage error messages.
	// This function is able to replace the original error messages.
	// If this function is not set, the "Incorrect usage" is displayed and the execution is interrupted.
	OnUsageError func(context *Context, err error) error
	// List of child commands
	Subcommands []Command
	// List of flags to parse
	Flags []Flag
	// Treat all flags as normal arguments if true
	SkipFlagParsing bool
	// Boolean to hide built-in help command
	HideHelp bool

	// Full name of command for help, defaults to full command name, including parent commands.
	HelpName        string
	commandNamePath []string
}

// Returns the full name of the command.
// For subcommands this ensures that parent commands are part of the command path
func (c Command) FullName() string {
	if c.commandNamePath == nil {
		return c.Name
	}
	return strings.Join(c.commandNamePath, " ")
}

// Invokes the command given the context, parses ctx.Args() to generate command-specific flags
func (c Command) Run(ctx *Context) (err error) {
	if len(c.Subcommands) > 0 {
		return c.startApp(ctx)
	}

	if !c.HideHelp && (HelpFlag != BoolFlag{}) {
		// append help to flags
		c.Flags = append(
			c.Flags,
			HelpFlag,
		)
	}

	if ctx.App.EnableBashCompletion {
		c.Flags = append(c.Flags, BashCompletionFlag)
	}

	set := flagSet(c.Name, c.Flags)
	set.SetOutput(ioutil.Discard)

	if !c.SkipFlagParsing {
		firstFlagIndex := -1
		terminatorIndex := -1
		for index, arg := range ctx.Args() {
			if arg == "--" {
				terminatorIndex = index
				break
			} else if arg == "-" {
				// Do nothing. A dash alone is not really a flag.
				continue
			} else if strings.HasPrefix(arg, "-") && firstFlagIndex == -1 {
				firstFlagIndex = index
			}
		}

		if firstFlagIndex > -1 {
			args := ctx.Args()
			regularArgs := make([]string, len(args[1:firstFlagIndex]))
			copy(regularArgs, args[1:firstFlagIndex])

			var flagArgs []string
			if terminatorIndex > -1 {
				flagArgs = args[firstFlagIndex:terminatorIndex]
				regularArgs = append(regularArgs, args[terminatorIndex:]...)
			} else {
				flagArgs = args[firstFlagIndex:]
			}

			err = set.Parse(append(flagArgs, regularArgs...))
		} else {
			err = set.Parse(ctx.Args().Tail())
		}
	} else {
		if c.SkipFlagParsing {
			err = set.Parse(append([]string{"--"}, ctx.Args().Tail()...))
		}
	}

	if err != nil {
		if c.OnUsageError != nil {
			err := c.OnUsageError(ctx, err)
			return err
		} else {
			fmt.Fprintln(ctx.App.Writer, "Incorrect Usage.")
			fmt.Fprintln(ctx.App.Writer)
			ShowCommandHelp(ctx, c.Name)
			return err
		}
	}

	nerr := normalizeFlags(c.Flags, set)
	if nerr != nil {
		fmt.Fprintln(ctx.App.Writer, nerr)
		fmt.Fprintln(ctx.App.Writer)
		ShowCommandHelp(ctx, c.Name)
		return nerr
	}
	context := NewContext(ctx.App, set, ctx)

	if checkCommandCompletions(context, c.Name) {
		return nil
	}

	if checkCommandHelp(context, c.Name) {
		return nil
	}

	if c.After != nil {
		defer func() {
			afterErr := c.After(context)
			if afterErr != nil {
				if err != nil {
					err = NewMultiError(err, afterErr)
				} else {
					err = afterErr
				}
			}
		}()
	}

	if c.Before != nil {
		err := c.Before(context)
		if err != nil {
			fmt.Fprintln(ctx.App.Writer, err)
			fmt.Fprintln(ctx.App.Writer)
			ShowCommandHelp(ctx, c.Name)
			return err
		}
	}

	context.Command = c
	c.Action(context)
	return nil
}

func (c Command) Names() []string {
	names := []string{c.Name}

	if c.ShortName != "" {
		names = append(names, c.ShortName)
	}

	return append(names, c.Aliases...)
}

// Returns true if Command.Name or Command.ShortName matches given name
func (c Command) HasName(name string) bool {
	for _, n := range c.Names() {
		if n == name {
			return true
		}
	}
	return false
}

func (c Command) startApp(ctx *Context) error {
	app := NewApp()

	// set the name and usage
	app.Name = fmt.Sprintf("%s %s", ctx.App.Name, c.Name)
	if c.HelpName == "" {
		app.HelpName = c.HelpName
	} else {
		app.HelpName = app.Name
	}

	if c.Description != "" {
		app.Usage = c.Description
	} else {
		app.Usage = c.Usage
	}

	// set CommandNotFound
	app.CommandNotFound = ctx.App.CommandNotFound

	// set the flags and commands
	app.Commands = c.Subcommands
	app.Flags = c.Flags
	app.HideHelp = c.HideHelp

	app.Version = ctx.App.Version
	app.HideVersion = ctx.App.HideVersion
	app.Compiled = ctx.App.Compiled
	app.Author = ctx.App.Author
	app.Email = ctx.App.Email
	app.Writer = ctx.App.Writer

	// bash completion
	app.EnableBashCompletion = ctx.App.EnableBashCompletion
	if c.BashComplete != nil {
		app.BashComplete = c.BashComplete
	}

	// set the actions
	app.Before = c.Before
	app.After = c.After
	if c.Action != nil {
		app.Action = c.Action
	} else {
		app.Action = helpSubcommand.Action
	}

	for index, cc := range app.Commands {
		app.Commands[index].commandNamePath = []string{c.Name, cc.Name}
	}

	return app.RunAsSubcommand(ctx)
}
