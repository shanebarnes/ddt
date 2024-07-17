package ddt

import (
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewReader(t *testing.T) {
	require.NotNil(t, NewReader(&os.File{}))
}

func TestNewWriter(t *testing.T) {
	require.NotNil(t, NewWriter(io.Discard))
}

func TestWriter_Write(t *testing.T) {
	t.Run("emptyBuffer", func(t *testing.T) {
		writer := NewWriter(io.Discard)
		n, err := writer.Write([]byte{})
		require.NoError(t, err)
		assert.Equal(t, 0, n)
	})

	t.Run("nilBuffer", func(t *testing.T) {
		writer := NewWriter(io.Discard)
		n, err := writer.Write(nil)
		require.NoError(t, err)
		assert.Equal(t, 0, n)
	})

	t.Run("nonEmptyBuffer", func(t *testing.T) {
		writer := NewWriter(io.Discard)
		n, err := writer.Write([]byte{1})
		require.NoError(t, err)
		assert.Equal(t, 1, n)
	})
}
