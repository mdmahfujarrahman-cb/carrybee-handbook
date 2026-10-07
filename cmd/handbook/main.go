// Command handbook builds and serves the Carrybee onboarding handbook.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"time"

	"carrybee/handbook/internal/config"
	"carrybee/handbook/internal/server"
	"carrybee/handbook/internal/site"
)

const usage = `usage: handbook <build|serve|check> [flags]
  -config string    path to handbook.json (default "handbook.json")
  -apps-root string directory containing hive/, aladdin/, accounts/ (overrides config paths)
  -out string       output directory for build (default "dist")
  -port int         port for serve (default 4321)
  -allow-snapshot   check: accept apps loaded from snapshot/scan.json when their repo is missing`

type options struct {
	configPath, appsRoot, out string
	port                      int
	allowSnapshot             bool
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
	fs.BoolVar(&o.allowSnapshot, "allow-snapshot", false, "")
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
	case "build":
		res, err := site.Build(cfg, o.out)
		if err != nil {
			return err
		}
		printResult(res)
		return nil
	case "check":
		return check(cfg, o.allowSnapshot)
	case "serve":
		abs, err := filepath.Abs(o.out)
		if err != nil {
			return err
		}
		return server.New(cfg, abs, site.Build).ListenAndServe(o.port)
	default:
		return fmt.Errorf("unknown command %q\n%s", cmd, usage)
	}
}

// checkProblems lists what makes check fail: warnings, missing descriptions and, unless allowed,
// apps that were loaded from the snapshot instead of being scanned live.
func checkProblems(res *site.Result, allowSnapshot bool) []string {
	problems := append(append([]string{}, res.Warnings...), res.Missing...)
	if allowSnapshot {
		return problems
	}
	return append(problems, snapshotProblems(res)...)
}

// snapshotProblems names each app that was loaded from the snapshot instead of scanned live.
func snapshotProblems(res *site.Result) []string {
	out := make([]string, 0, len(res.FromSnapshot))
	for _, id := range res.FromSnapshot {
		out = append(out, id+": not scanned live (loaded from "+config.SnapshotRel+"; pass -allow-snapshot to accept)")
	}
	return out
}

func check(cfg *config.Config, allowSnapshot bool) error {
	tmp, err := os.MkdirTemp("", "handbook-check-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	res, err := site.Build(cfg, tmp)
	if err != nil {
		return err
	}
	printResult(res)
	problems := checkProblems(res, allowSnapshot)
	if !allowSnapshot {
		for _, p := range snapshotProblems(res) {
			fmt.Println("error:", p)
		}
	}
	for _, b := range site.CheckLinks(tmp) {
		fmt.Println("broken link:", b)
		problems = append(problems, b)
	}
	if len(problems) > 0 {
		return fmt.Errorf("check failed: %d problem(s)", len(problems))
	}
	fmt.Println("check passed")
	return nil
}

func printResult(res *site.Result) {
	for _, id := range sortedIDs(res.Scans) {
		s := res.Scans[id]
		status := "ok"
		switch {
		case slices.Contains(res.FromSnapshot, id):
			status = "from snapshot built " + res.SnapshotBuiltAt.Format(time.RFC3339)
		case s.Error != "":
			status = "NOT SCANNED: " + s.Error
		case len(s.Skipped) > 0:
			status = fmt.Sprintf("ok, %d unreadable file(s) skipped", len(s.Skipped))
		}
		fmt.Printf("%-9s %3d modules %3d pages %2d cycles  %s  %s\n", id, len(s.Modules), len(s.Routes), len(s.Cycles), s.Commit, status)
	}
	fmt.Printf("pages rendered: %d\n", res.Pages)
	for _, w := range res.Warnings {
		fmt.Println("warning:", w)
	}
	for _, m := range res.Missing {
		fmt.Println("needs description:", m)
	}
}

func sortedIDs[V any](m map[string]V) []string {
	ids := make([]string, 0, len(m))
	for id := range m {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}
