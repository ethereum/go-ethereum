// Copyright 2020 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

// package flags

// import (
//     "fmt"
//     "os"
//     "regexp"
//     "sort"
//     "strings"

//     "github.com/ethereum/go-ethereum/internal/version"
//     "github.com/ethereum/go-ethereum/log"
//     "github.com/ethereum/go-ethereum/params"
//     "github.com/mattn/go-isatty"
//     "github.com/urfave/cli/v2"
// )

// // usecolor defines whether the CLI help should use colored output or normal dumb
// // colorless terminal formatting.
// var usecolor = (isatty.IsTerminal(os.Stdout.Fd()) || isatty.IsCygwinTerminal(os.Stdout.Fd())) && os.Getenv("TERM") != "dumb"

// // NewApp creates an app with sane defaults.
// func NewApp(usage string) *cli.App {
//     git, _ := version.VCS()
//     app := cli.NewApp()
//     app.EnableBashCompletion = true
//     app.Version = params.VersionWithCommit(git.Commit, git.Date)
//     app.Usage = usage
//     app.Copyright = "Copyright 2013-2024 The go-ethereum Authors"
//     app.Before = func(ctx *cli.Context) error {
//         MigrateGlobalFlags(ctx)
//         return nil
//     }
//     return app
// }

// // Merge merges the given flag slices.
// func Merge(groups ...[]cli.Flag) []cli.Flag {
//     var ret []cli.Flag
//     for _, group := range groups {
//         ret = append(ret, group...)
//     }
//     return ret
// }

// var migrationApplied = map[*cli.Command]struct{}{}

// // MigrateGlobalFlags makes all global flag values available in the
// // context. This should be called as early as possible in app.Before.
// func MigrateGlobalFlags(ctx *cli.Context) {
//     var iterate func(cs []*cli.Command, fn func(*cli.Command))
//     iterate = func(cs []*cli.Command, fn func(*cli.Command)) {
//         for _, cmd := range cs {
//             if _, ok := migrationApplied[cmd]; ok {
//                 continue
//             }
//             migrationApplied[cmd] = struct{}{}
//             fn(cmd)
//             iterate(cmd.Subcommands, fn)
//         }
//     }

//     // This iterates over all commands and wraps their action function.
//     iterate(ctx.App.Commands, func(cmd *cli.Command) {
//         if cmd.Action == nil {
//             return
//         }

//         action := cmd.Action
//         cmd.Action = func(ctx *cli.Context) error {
//             doMigrateFlags(ctx)
//             return action(ctx)
//         }
//     })
// }

// func doMigrateFlags(ctx *cli.Context) {
//     // Figure out if there are any aliases of commands. If there are, we want
//     // to ignore them when iterating over the flags.
//     aliases := make(map[string]bool)
//     for _, fl := range ctx.Command.Flags {
//         for _, alias := range fl.Names()[1:] {
//             aliases[alias] = true
//         }
//     }
//     for _, name := range ctx.FlagNames() {
//         for _, parent := range ctx.Lineage()[1:] {
//             if parent.IsSet(name) {
//                 if _, isAlias := aliases[name]; isAlias {
//                     continue
//                 }
//                 if result := parent.StringSlice(name); len(result) > 0 {
//                     ctx.Set(name, strings.Join(result, ","))
//                 } else {
//                     ctx.Set(name, parent.String(name))
//                 }
//                 break
//             }
//         }
//     }
// }

// func init() {
//     if usecolor {
//         cli.AppHelpTemplate = regexp.MustCompile("[A-Z ]+:").ReplaceAllString(cli.AppHelpTemplate, "\u001B[33m$0\u001B[0m")
//         cli.AppHelpTemplate = strings.ReplaceAll(cli.AppHelpTemplate, "{{template \"visibleFlagCategoryTemplate\" .}}", "{{range .VisibleFlagCategories}}\n   {{if .Name}}\u001B[33m{{.Name}}\u001B[0m\n\n   {{end}}{{$flglen := len .Flags}}{{range $i, $e := .Flags}}{{if eq (subtract $flglen $i) 1}}{{$e}}\n{{else}}{{$e}}\n   {{end}}{{end}}{{end}}")
//     }
//     cli.FlagStringer = FlagString
// }

// // FlagString prints a single flag in help.
// func FlagString(f cli.Flag) string {
//     df, ok := f.(cli.DocGenerationFlag)
//     if !ok {
//         return ""
//     }
//     needsPlaceholder := df.TakesValue()
//     placeholder := ""
//     if needsPlaceholder {
//         placeholder = "value"
//     }

//     namesText := cli.FlagNamePrefixer(df.Names(), placeholder)

//     defaultValueString := ""
//     if s := df.GetDefaultText(); s != "" {
//         default
