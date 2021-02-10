// +build darwin linux

package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsSpecialFile(t *testing.T) {
	assert.True(t, IsSpecialFile(os.DevNull))
	assert.True(t, IsSpecialFile("/dev/random"))
	assert.True(t, IsSpecialFile("/dev/urandom"))
	assert.True(t, IsSpecialFile("/dev/zero"))
	assert.True(t, IsSpecialFile("//dev//zero/"))
	assert.False(t, IsSpecialFile("/foo/bar"))
}