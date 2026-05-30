// Command posfix normalizes file names to snake_case ASCII.
//
// Usage:
//
//	posfix [flags] [targets...]
//
// With no targets, posfix processes the files in the current directory. See the
// README for the full behavior and the roadmap.
package main

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/pflag"

	"github.com/brunobreis/posfix/internal/normalize"
	"github.com/brunobreis/posfix/internal/renamer"
)

// version is the build version. Override it at link time with:
//
//	go build -ldflags "-X main.version=v1.0.0" ./cmd/posfix
var version = "dev"

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run is the testable core of the command: it takes the raw arguments and the
// output streams explicitly, and returns the process exit code. Keeping main
// itself trivial lets the tests drive run directly with buffers.
func run(args []string, stdout, stderr io.Writer) int {
	fs := pflag.NewFlagSet("posfix", pflag.ContinueOnError)
	fs.SortFlags = false // show flags in definition order, not alphabetically
	fs.SetOutput(stderr)

	dryRun := fs.BoolP("dry-run", "n", false, "show what would happen, without renaming")
	recursive := fs.BoolP("recursive", "r", false, "descend into subdirectories")
	all := fs.BoolP("all", "a", false, "include hidden files (those starting with '.')")
	showHelp := fs.BoolP("help", "h", false, "show this help and exit")
	showVersion := fs.Bool("version", false, "show version and exit")

	printUsage := func(w io.Writer) {
		fmt.Fprint(w, "posfix - normalize file names to snake_case ASCII\n\n")
		fmt.Fprint(w, "Usage:\n  posfix [flags] [targets...]\n\n")
		fmt.Fprint(w, "With no targets, posfix processes files in the current directory.\n\n")
		fmt.Fprint(w, "Flags:\n")
		fmt.Fprint(w, fs.FlagUsages())
	}
	fs.Usage = func() { printUsage(stderr) }

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(stderr, "posfix: %v\n", err)
		printUsage(stderr)
		return 2
	}
	if *showHelp {
		printUsage(stdout)
		return 0
	}
	if *showVersion {
		fmt.Fprintf(stdout, "posfix %s\n", version)
		return 0
	}

	opts := renamer.Options{
		DryRun:    *dryRun,
		Recursive: *recursive,
		All:       *all,
		Style:     normalize.Snake{},
	}
	if renamer.Run(fs.Args(), opts, stdout, stderr) > 0 {
		return 1
	}
	return 0
}
