// envdrift checks that dotenv files have the same contract.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/qvoo/Hello-World"
)

const usage = `envdrift 鈥?find configuration drift before it reaches production

Usage:
  envdrift [flags] [baseline.env target.env ...]

The first file is the contract. Every following file is checked against it.
With no paths, envdrift checks .env.example against .env.

Examples:
  envdrift .env.example .env.local .env.production
  envdrift --values --fail-on warning .env.example .env.ci
  envdrift --format json config/.env.example config/.env.staging

Exit codes:
  0  no findings at the selected failure level
  1  findings at the selected failure level, or an input error
`

type stringList []string

func (s *stringList) String() string { return strings.Join(*s, ",") }
func (s *stringList) Set(value string) error {
	*s = append(*s, value)
	return nil
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("envdrift", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.Usage = func() { fmt.Fprint(stderr, usage) }
	format := flags.String("format", "text", "output format: text or json")
	values := flags.Bool("values", false, "report values that differ (values are never printed)")
	failOn := flags.String("fail-on", "error", "failure threshold: error or warning")
	var ignored stringList
	flags.Var(&ignored, "ignore", "key to ignore; can be used more than once")
	help := flags.Bool("help", false, "show help")

	if err := flags.Parse(args); err != nil {
		return 1
	}
	if *help {
		flags.Usage()
		return 0
	}
	if *format != "text" && *format != "json" {
		fmt.Fprintln(stderr, "envdrift: --format must be text or json")
		return 1
	}
	if *failOn != "error" && *failOn != "warning" {
		fmt.Fprintln(stderr, "envdrift: --fail-on must be error or warning")
		return 1
	}

	paths := flags.Args()
	if len(paths) == 0 {
		paths = []string{".env.example", ".env"}
	}
	if len(paths) < 2 {
		fmt.Fprintln(stderr, "envdrift: provide a baseline and at least one target file")
		return 1
	}

	files := make([]envdrift.File, 0, len(paths))
	for _, path := range paths {
		file, err := os.Open(path)
		if err != nil {
			fmt.Fprintf(stderr, "envdrift: read %s: %v\n", filepath.Clean(path), err)
			return 1
		}
		parsed, err := envdrift.Parse(file, path)
		closeErr := file.Close()
		if err != nil {
			fmt.Fprintln(stderr, "envdrift:", err)
			return 1
		}
		if closeErr != nil {
			fmt.Fprintf(stderr, "envdrift: close %s: %v\n", filepath.Clean(path), closeErr)
			return 1
		}
		files = append(files, parsed)
	}

	report := envdrift.Compare(files[0], files[1:], envdrift.Options{
		CompareValues: *values,
		IgnoreKeys:    ignored,
	})
	var outputErr error
	if *format == "json" {
		outputErr = envdrift.WriteJSON(stdout, report)
	} else {
		outputErr = envdrift.WriteText(stdout, report)
	}
	if outputErr != nil {
		if !errors.Is(outputErr, os.ErrClosed) {
			fmt.Fprintln(stderr, "envdrift: write output:", outputErr)
		}
		return 1
	}
	if report.Fails(*failOn) {
		return 1
	}
	return 0
}

