package pathutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsLocalPath(t *testing.T) {
	// Local paths
	assert.True(t, IsLocalPath("./foo"))
	assert.True(t, IsLocalPath("../foo"))
	assert.True(t, IsLocalPath("."))
	assert.True(t, IsLocalPath(".."))
	assert.True(t, IsLocalPath("/absolute/path"))

	// Module paths
	assert.False(t, IsLocalPath(""))
	assert.False(t, IsLocalPath("github.com/foo/bar"))
	assert.False(t, IsLocalPath("golang.org/x/mod"))
}
