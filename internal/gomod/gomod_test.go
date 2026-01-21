package gomod

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseReplaces(t *testing.T) {
	dir := t.TempDir()
	gomod := filepath.Join(dir, "go.mod")

	content := `module example.com/test

go 1.21

replace github.com/foo/bar => github.com/fork/bar v1.0.0
replace github.com/local => ./local
replace github.com/baz => github.com/new/baz v2.0.0
`
	require.NoError(t, os.WriteFile(gomod, []byte(content), 0644))

	replaces, err := ParseReplaces(gomod)
	require.NoError(t, err)

	// Should only have non-local replaces
	assert.Len(t, replaces, 2)
	assert.Equal(t, "github.com/foo/bar", replaces[0].Old)
	assert.Equal(t, "github.com/fork/bar", replaces[0].New)
	assert.Equal(t, "v1.0.0", replaces[0].NewVer)
}

func TestDeduplicate(t *testing.T) {
	replaces := []Replace{
		{Old: "github.com/foo", New: "github.com/bar", NewVer: "v2.0.0", Source: "dep1"},
		{Old: "github.com/foo", New: "github.com/bar", NewVer: "v1.0.0", Source: "dep2"},
	}

	result := Deduplicate(replaces)

	assert.Len(t, result, 1)
	assert.Equal(t, "v1.0.0", result[0].NewVer) // Lowest version wins
}
