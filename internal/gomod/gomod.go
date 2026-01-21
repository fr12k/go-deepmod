// Package gomod provides utilities for parsing go.mod files.
package gomod

import (
	"fmt"
	"os"
	"strings"

	"github.com/fr12k/go-deepmod/internal/pathutil"
	"golang.org/x/mod/modfile"
	"golang.org/x/mod/semver"
)

// Replace represents a replace directive.
type Replace struct {
	Old, New   string // Module paths
	OldVer     string // Version constraint (optional)
	NewVer     string // Target version
	Source     string // Source module that defined this replace
}

// IsLocal returns true if this replace points to a local path.
func (r Replace) IsLocal() bool {
	return pathutil.IsLocalPath(r.New)
}

// ParseReplaces parses a go.mod file and returns its non-local replace directives.
func ParseReplaces(path string) ([]Replace, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	f, err := modfile.Parse(path, data, nil)
	if err != nil {
		return nil, err
	}

	var replaces []Replace
	for _, r := range f.Replace {
		rep := Replace{
			Old:    r.Old.Path,
			OldVer: r.Old.Version,
			New:    r.New.Path,
			NewVer: r.New.Version,
		}
		if !rep.IsLocal() {
			replaces = append(replaces, rep)
		}
	}
	return replaces, nil
}

// ParseReplacesWithSource parses a go.mod and returns replaces that have a "// from:" comment.
func ParseReplacesWithSource(path string) ([]Replace, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	f, err := modfile.Parse(path, data, nil)
	if err != nil {
		return nil, err
	}

	var replaces []Replace
	for _, r := range f.Replace {
		rep := Replace{
			Old:    r.Old.Path,
			OldVer: r.Old.Version,
			New:    r.New.Path,
			NewVer: r.New.Version,
		}
		// Extract source from comment
		if r.Syntax != nil {
			for _, c := range r.Syntax.Suffix {
				if text := c.Token; len(text) > 0 {
					if idx := indexOf(text, "from:"); idx >= 0 {
						rep.Source = strings.TrimSpace(text[idx+5:])
					}
				}
			}
		}
		if rep.Source != "" {
			replaces = append(replaces, rep)
		}
	}
	return replaces, nil
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// Deduplicate removes duplicate replaces, keeping the one with lowest version.
func Deduplicate(replaces []Replace) []Replace {
	byKey := make(map[string]Replace)
	for _, r := range replaces {
		key := r.Old
		if r.OldVer != "" {
			key = fmt.Sprintf("%s@%s", r.Old, r.OldVer)
		}
		if existing, ok := byKey[key]; !ok || semver.Compare(r.NewVer, existing.NewVer) < 0 {
			byKey[key] = r
		}
	}

	result := make([]Replace, 0, len(byKey))
	for _, r := range byKey {
		result = append(result, r)
	}
	return result
}
