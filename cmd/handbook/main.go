// Command handbook builds and serves the Carrybee onboarding handbook.
package main

import (
	"flag"
	"fmt"
	"os"

	"carrybee/handbook/internal/config"
)

const usage = `usage: handbook <build|serve|check> [flags]
  -config string    path to handbook.json (default "handbook.json")
  -apps-root string directory containing hive/, aladdin/, accounts/ (overrides config paths)
  -out string       output directory for build (default "dist")
  -port int         port for serve (default 4321)`

type options struct {
	configPath, appsRoot, out string
	port                      int
}

func parse(args []string) (string, options, error) {
	if len(args) == 0 {
		return "", options{}, fmt.Errorf("missing command")
	}
	fs := flag.NewFlagSet(args[0], flag.ContinueOnError)
	var o options
	fs.StringVar(&o.configPath, "config", "handbook.json", "")
	fs.StringVar(&o.appsRoot, "apps-root", "", "")
	fs.StringVar(&o.out, "out", "dist", "")
	fs.IntVar(&o.port, "port", 4321, "")
	if err := fs.Parse(args[1:]); err != nil {
		return "", o, err
	}
	return args[0], o, nil
}

func main() {
	cmd, o, err := parse(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(2)
	}
	cfg, err := config.Load(o.configPath, o.appsRoot)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := run(cmd, cfg, o); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(cmd string, cfg *config.Config, o options) error {
	switch cmd {
	case "build", "serve", "check":
		return fmt.Errorf("%s: not implemented yet", cmd)
	default:
		return fmt.Errorf("unknown command %q\n%s", cmd, usage)
	}
}
