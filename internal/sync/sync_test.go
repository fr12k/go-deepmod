package sync

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRun_DryRun(t *testing.T) {
	dir := setupTestModuleWithReplaces(t)

	var out bytes.Buffer
	err := Run(context.Background(), Options{
		Dir:    dir,
		DryRun: true,
		Output: &out,
	})
	require.NoError(t, err)

	output := out.String()
	assert.Contains(t, output, "Would apply")
	assert.Contains(t, output, "github.com/old/module")

	// Verify go.mod unchanged (no replace added)
	data, _ := os.ReadFile(filepath.Join(dir, "go.mod"))
	assert.NotContains(t, string(data), "replace github.com/old/module")
}

func TestRun_Apply(t *testing.T) {
	dir := setupTestModuleWithReplaces(t)

	var out bytes.Buffer
	err := Run(context.Background(), Options{
		Dir:      dir,
		SkipTidy: true,
		Output:   &out,
	})
	require.NoError(t, err)

	// Verify replace was applied with source comment
	data, _ := os.ReadFile(filepath.Join(dir, "go.mod"))
	content := string(data)
	assert.Contains(t, content, "replace github.com/old/module")
	assert.Contains(t, content, "// from: example.com/dep")
}

func TestRun_NoReplaces(t *testing.T) {
	dir := t.TempDir()

	gomod := `module example.com/test
go 1.21
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte(gomod), 0644))

	var out bytes.Buffer
	err := Run(context.Background(), Options{
		Dir:    dir,
		Output: &out,
	})
	require.NoError(t, err)
	assert.Contains(t, out.String(), "No replace directives")
}

func TestRun_RemoveStaleReplace(t *testing.T) {
	dir := t.TempDir()

	// Create the "dependency" module WITHOUT the replace directive
	depDir := filepath.Join(dir, "dep")
	require.NoError(t, os.MkdirAll(depDir, 0755))

	depGomod := `module example.com/dep

go 1.21
`
	require.NoError(t, os.WriteFile(filepath.Join(depDir, "go.mod"), []byte(depGomod), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(depDir, "dep.go"), []byte("package dep\n"), 0644))

	// Create main module with a stale replace (has "// from:" comment but dep no longer has it)
	gomod := `module example.com/test

go 1.21

require example.com/dep v0.0.0

replace example.com/dep => ./dep
replace github.com/stale/module => github.com/other/module v1.0.0 // from: example.com/dep
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte(gomod), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n\nimport _ \"example.com/dep\"\n\nfunc main() {}\n"), 0644))

	var out bytes.Buffer
	err := Run(context.Background(), Options{
		Dir:      dir,
		SkipTidy: true,
		Output:   &out,
	})
	require.NoError(t, err)

	// Verify stale replace was removed
	data, _ := os.ReadFile(filepath.Join(dir, "go.mod"))
	content := string(data)
	assert.NotContains(t, content, "github.com/stale/module")
	assert.Contains(t, out.String(), "Removed stale")
}

func TestRun_RemoveStaleReplace_DryRun(t *testing.T) {
	dir := t.TempDir()

	depDir := filepath.Join(dir, "dep")
	require.NoError(t, os.MkdirAll(depDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(depDir, "go.mod"), []byte("module example.com/dep\n\ngo 1.21\n"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(depDir, "dep.go"), []byte("package dep\n"), 0644))

	gomod := `module example.com/test

go 1.21

require example.com/dep v0.0.0

replace example.com/dep => ./dep
replace github.com/stale/module => github.com/other/module v1.0.0 // from: example.com/dep
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte(gomod), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n\nimport _ \"example.com/dep\"\n\nfunc main() {}\n"), 0644))

	var out bytes.Buffer
	err := Run(context.Background(), Options{
		Dir:    dir,
		DryRun: true,
		Output: &out,
	})
	require.NoError(t, err)

	// Verify dry run output
	output := out.String()
	assert.Contains(t, output, "Would remove")
	assert.Contains(t, output, "github.com/stale/module")

	// Verify go.mod unchanged
	data, _ := os.ReadFile(filepath.Join(dir, "go.mod"))
	assert.Contains(t, string(data), "github.com/stale/module")
}

// setupTestModuleWithReplaces creates a test module that depends on a local module with replaces.
func setupTestModuleWithReplaces(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	// Create the "dependency" module with replace directives
	depDir := filepath.Join(dir, "dep")
	require.NoError(t, os.MkdirAll(depDir, 0755))

	depGomod := `module example.com/dep

go 1.21

replace github.com/old/module => github.com/new/module v1.0.0
replace github.com/local/thing => ./local
`
	require.NoError(t, os.WriteFile(filepath.Join(depDir, "go.mod"), []byte(depGomod), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(depDir, "dep.go"), []byte("package dep\n"), 0644))

	// Create main module that depends on the local dep
	gomod := `module example.com/test

go 1.21

require example.com/dep v0.0.0

replace example.com/dep => ./dep
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte(gomod), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n\nimport _ \"example.com/dep\"\n\nfunc main() {}\n"), 0644))

	return dir
}
