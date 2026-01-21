// sync-replaces syncs replace directives from direct dependencies.
//
//	//go:generate sync-replaces
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"

	"github.com/fr12k/go-deepmod/internal/sync"
)

func main() {
	var opts sync.Options
	flag.BoolVar(&opts.DryRun, "n", false, "dry run")
	flag.BoolVar(&opts.DryRun, "dry-run", false, "dry run")
	flag.BoolVar(&opts.Verbose, "v", false, "verbose")
	flag.BoolVar(&opts.SkipTidy, "skip-tidy", false, "skip go mod tidy")
	flag.StringVar(&opts.Dir, "dir", "", "module directory")
	flag.Parse()

	opts.Output = os.Stdout

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	if err := sync.Run(ctx, opts); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
