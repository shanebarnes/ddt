package ddt

import (
	"io"
	"os"
	"testing"

	"github.com/dustin/go-humanize"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func BenchmarkRandReader(b *testing.B) {
	reader := NewRandReader()
	buf := make([]byte, 2048)
	for i := 0; i < b.N; i++ {
		_, _ = reader.Read(buf)
	}
}

func BenchmarkZeroReader(b *testing.B) {
	reader := NewZeroReader()
	buf := make([]byte, 2048)
	for i := 0; i < b.N; i++ {
		_, _ = reader.Read(buf)
	}
}

func TestNewNullWriter(t *testing.T) {
	assert.NotNil(t, NewNullWriter())
}

func TestNullWriter_Write(t *testing.T) {
	t.Run("emptyBuffer", func(t *testing.T) {
		n, err := NewNullWriter().Write([]byte{})
		require.NoError(t, err)
		assert.Zero(t, n)
	})

	t.Run("nilBuffer", func(t *testing.T) {
		n, err := NewNullWriter().Write(nil)
		require.NoError(t, err)
		assert.Zero(t, n)
	})

	t.Run("nonEmptyBuffer", func(t *testing.T) {
		n, err := NewNullWriter().Write([]byte{1, 2, 3, 4})
		require.NoError(t, err)
		assert.Equal(t, 4, n)
	})
}

func TestNullWriter_WriteAt(t *testing.T) {
	t.Run("emptyBuffer", func(t *testing.T) {
		n, err := NewNullWriter().WriteAt([]byte{}, 0)
		require.NoError(t, err)
		assert.Zero(t, n)
	})

	t.Run("nilBuffer", func(t *testing.T) {
		n, err := NewNullWriter().WriteAt(nil, 0)
		require.NoError(t, err)
		assert.Zero(t, n)
	})

	t.Run("nonEmptyBuffer", func(t *testing.T) {
		n, err := NewNullWriter().WriteAt([]byte{1, 2, 3, 4}, humanize.MiByte)
		require.NoError(t, err)
		assert.Equal(t, 4, n)
	})
}

func TestNewRandReader(t *testing.T) {
	reader1 := NewRandReader()
	require.NotNil(t, reader1)

	reader2 := NewRandReader()
	require.NotNil(t, reader2)

	assert.NotEqualValues(t, reader1.seed, reader2.seed)
}

func TestRandReader_Read(t *testing.T) {
	t.Run("emptyBuffer", func(t *testing.T) {
		n, err := NewRandReader().Read([]byte{})
		require.NoError(t, err)
		assert.Zero(t, n)
	})

	t.Run("nilBuffer", func(t *testing.T) {
		n, err := NewRandReader().Read(nil)
		require.NoError(t, err)
		assert.Zero(t, n)
	})

	t.Run("nonEmptyBuffer", func(t *testing.T) {
		reader := NewRandReader()

		var buf1 = []byte{0, 0, 0, 0}
		n, err := reader.Read(buf1)
		require.NoError(t, err)
		assert.Equal(t, 4, n)
		assert.NotEqual(t, []byte{0, 0, 0, 0}, buf1)

		var buf2 = []byte{0, 0, 0, 0}
		n, err = reader.Read(buf2)
		require.NoError(t, err)
		assert.Equal(t, 4, n)
		assert.NotEqual(t, []byte{0, 0, 0, 0}, buf2)

		assert.NotEqual(t, buf1, buf2)
	})
}

func TestRandReader_ReadAt(t *testing.T) {
	t.Run("emptyBuffer", func(t *testing.T) {
		n, err := NewRandReader().ReadAt([]byte{}, 0)
		require.NoError(t, err)
		assert.Zero(t, n)
	})

	t.Run("nilBuffer", func(t *testing.T) {
		n, err := NewRandReader().ReadAt(nil, 0)
		require.NoError(t, err)
		assert.Zero(t, n)
	})

	t.Run("nonEmptyBuffer", func(t *testing.T) {
		reader := NewRandReader()

		var buf1 = []byte{0, 0, 0, 0}
		n, err := reader.ReadAt(buf1, humanize.MiByte)
		require.NoError(t, err)
		assert.Equal(t, 4, n)
		assert.NotEqual(t, []byte{0, 0, 0, 0}, buf1)

		var buf2 = []byte{0, 0, 0, 0}
		n, err = reader.ReadAt(buf2, humanize.MiByte)
		require.NoError(t, err)
		assert.Equal(t, 4, n)
		assert.NotEqual(t, []byte{0, 0, 0, 0}, buf2)

		assert.NotEqual(t, buf1, buf2)
	})
}

func TestNewReader(t *testing.T) {
	assert.NotNil(t, NewReader(&os.File{}))
}

func TestNewWriter(t *testing.T) {
	assert.NotNil(t, NewWriter(io.Discard))
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

func TestNewZeroReader(t *testing.T) {
	assert.NotNil(t, NewZeroReader())
}

func TestZeroReader_Read(t *testing.T) {
	t.Run("emptyBuffer", func(t *testing.T) {
		n, err := NewZeroReader().Read([]byte{})
		require.NoError(t, err)
		assert.Zero(t, n)
	})

	t.Run("nilBuffer", func(t *testing.T) {
		n, err := NewZeroReader().Read(nil)
		require.NoError(t, err)
		assert.Zero(t, n)
	})

	t.Run("nonEmptyBuffer", func(t *testing.T) {
		buf := []byte{1, 2, 3, 4}
		n, err := NewZeroReader().Read(buf)
		require.NoError(t, err)
		assert.Equal(t, 4, n)
		assert.Equal(t, []byte{0, 0, 0, 0}, buf)
	})
}

func TestZeroReader_ReadAt(t *testing.T) {
	t.Run("emptyBuffer", func(t *testing.T) {
		n, err := NewZeroReader().ReadAt([]byte{}, 0)
		require.NoError(t, err)
		assert.Zero(t, n)
	})

	t.Run("nilBuffer", func(t *testing.T) {
		n, err := NewZeroReader().ReadAt(nil, 0)
		require.NoError(t, err)
		assert.Zero(t, n)
	})

	t.Run("nonEmptyBuffer", func(t *testing.T) {
		buf := []byte{1, 2, 3, 4}
		n, err := NewZeroReader().ReadAt(buf, humanize.MiByte)
		require.NoError(t, err)
		assert.Equal(t, 4, n)
		assert.Equal(t, []byte{0, 0, 0, 0}, buf)
	})
}
