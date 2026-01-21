// Package sync provides functionality for syncing replace directives.
package sync

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/fr12k/go-deepmod/internal/gomod"
)

// Options configures the sync operation.
type Options struct {
	Output   io.Writer
	Dir      string
	DryRun   bool
	Verbose  bool
	SkipTidy bool
}

// moduleInfo from go list -m -json
type moduleInfo struct {
	Path     string `json:"Path"`
	Version  string `json:"Version"`
	Dir      string `json:"Dir"`
	Main     bool   `json:"Main"`
	Indirect bool   `json:"Indirect"`
}

// Run syncs replace directives from all direct dependencies.
func Run(ctx context.Context, opts Options) error {
	if opts.Output == nil {
		opts.Output = os.Stdout
	}
	if opts.Dir == "" {
		var err error
		opts.Dir, err = os.Getwd()
		if err != nil {
			return err
		}
	}

	log := func(format string, args ...any) {
		fmt.Fprintf(opts.Output, format+"\n", args...)
	}
	verbose := func(format string, args ...any) {
		if opts.Verbose {
			log(format, args...)
		}
	}

	// List direct dependencies
	deps, err := listDirectDeps(ctx, opts.Dir)
	if err != nil {
		return fmt.Errorf("listing deps: %w", err)
	}
	verbose("Found %d direct dependencies", len(deps))

	// Collect replaces from all deps
	var allReplaces []gomod.Replace
	for _, dep := range deps {
		verbose("Processing %s@%s", dep.Path, dep.Version)

		if dep.Dir == "" {
			verbose("  skipping: no local dir")
			continue
		}

		replaces, err := gomod.ParseReplaces(filepath.Join(dep.Dir, "go.mod"))
		if err != nil {
			verbose("  skipping: %v", err)
			continue
		}

		for i := range replaces {
			replaces[i].Source = dep.Path
		}
		allReplaces = append(allReplaces, replaces...)
	}

	// Deduplicate
	replaces := gomod.Deduplicate(allReplaces)

	// Find stale replaces (in main go.mod with "// from:" but no longer in deps)
	stale, err := findStaleReplaces(opts.Dir, replaces)
	if err != nil {
		verbose("warning: could not check for stale replaces: %v", err)
	}

	if len(replaces) == 0 && len(stale) == 0 {
		log("No replace directives to sync")
		return nil
	}

	if opts.DryRun {
		if len(replaces) > 0 {
			log("Would apply %d replace directives:", len(replaces))
			for _, r := range replaces {
				log("  %s => %s %s // from: %s", r.Old, r.New, r.NewVer, r.Source)
			}
		}
		if len(stale) > 0 {
			log("Would remove %d stale replace directives:", len(stale))
			for _, r := range stale {
				log("  %s (was from: %s)", r.Old, r.Source)
			}
		}
		return nil
	}

	// Apply replaces
	for _, r := range replaces {
		arg := fmt.Sprintf("%s=%s@%s", r.Old, r.New, r.NewVer)
		if err := runCmd(ctx, opts.Dir, "go", "mod", "edit", "-replace="+arg); err != nil {
			return fmt.Errorf("applying replace %s: %w", r.Old, err)
		}
		if err := addSourceComment(opts.Dir, r); err != nil {
			verbose("warning: could not add source comment: %v", err)
		}
		log("Applied: %s => %s %s // from: %s", r.Old, r.New, r.NewVer, r.Source)
	}

	// Remove stale replaces
	for _, r := range stale {
		arg := r.Old
		if r.OldVer != "" {
			arg = fmt.Sprintf("%s@%s", r.Old, r.OldVer)
		}
		if err := runCmd(ctx, opts.Dir, "go", "mod", "edit", "-dropreplace="+arg); err != nil {
			return fmt.Errorf("removing stale replace %s: %w", r.Old, err)
		}
		log("Removed stale: %s (was from: %s)", r.Old, r.Source)
	}

	if !opts.SkipTidy {
		verbose("Running go mod tidy...")
		if err := runCmd(ctx, opts.Dir, "go", "mod", "tidy"); err != nil {
			return fmt.Errorf("go mod tidy: %w", err)
		}
	}

	return nil
}

func listDirectDeps(ctx context.Context, dir string) ([]moduleInfo, error) {
	cmd := exec.CommandContext(ctx, "go", "list", "-m", "-json", "all")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var deps []moduleInfo
	dec := json.NewDecoder(bytes.NewReader(out))
	for dec.More() {
		var m moduleInfo
		if err := dec.Decode(&m); err != nil {
			return nil, err
		}
		if !m.Main && !m.Indirect {
			deps = append(deps, m)
		}
	}
	return deps, nil
}

func runCmd(ctx context.Context, dir, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// findStaleReplaces finds replaces in go.mod that have "// from:" comments
// but are no longer provided by any dependency.
func findStaleReplaces(dir string, current []gomod.Replace) ([]gomod.Replace, error) {
	goModPath := filepath.Join(dir, "go.mod")
	existing, err := gomod.ParseReplacesWithSource(goModPath)
	if err != nil {
		return nil, err
	}

	// Build set of current replace keys
	currentKeys := make(map[string]bool)
	for _, r := range current {
		currentKeys[r.Old] = true
	}

	// Find stale ones
	var stale []gomod.Replace
	for _, r := range existing {
		if !currentKeys[r.Old] {
			stale = append(stale, r)
		}
	}
	return stale, nil
}

func addSourceComment(dir string, r gomod.Replace) error {
	goModPath := filepath.Join(dir, "go.mod")
	data, err := os.ReadFile(goModPath)
	if err != nil {
		return err
	}

	// Build pattern to find
	pattern := fmt.Sprintf("replace %s => %s %s", r.Old, r.New, r.NewVer)
	comment := fmt.Sprintf(" // from: %s", r.Source)

	content := string(data)
	if strings.Contains(content, pattern) && !strings.Contains(content, pattern+comment) {
		content = strings.Replace(content, pattern, pattern+comment, 1)
		return os.WriteFile(goModPath, []byte(content), 0644)
	}
	return nil
}
