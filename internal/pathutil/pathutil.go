// Package pathutil provides utilities for detecting local filesystem paths.
package pathutil

import (
	"path/filepath"
	"strings"
)

// IsLocalPath returns true if the given path is a local filesystem path
// rather than a module path. Local paths include:
// - Relative paths starting with "./" or "../"
// - Absolute paths (starting with "/" on Unix or drive letter on Windows)
func IsLocalPath(path string) bool {
	if path == "" {
		return false
	}

	// Check for relative paths
	if strings.HasPrefix(path, "./") || strings.HasPrefix(path, "../") {
		return true
	}

	// Check for "." or ".." alone
	if path == "." || path == ".." {
		return true
	}

	// Check for absolute paths
	if filepath.IsAbs(path) {
		return true
	}

	// On Windows, check for paths starting with drive letter
	if len(path) >= 2 && path[1] == ':' {
		return true
	}

	return false
}
