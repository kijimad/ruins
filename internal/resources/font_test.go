package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFont_存在するフォントを読み込める(t *testing.T) {
	t.Parallel()

	f, err := NewFont("file/fonts/dougenzaka/KH-Dot-Dougenzaka-16.ttf")

	require.NoError(t, err)
	assert.NotNil(t, f.FaceSource)
	assert.NotNil(t, f.Font)
}

func TestNewFont_存在しないパスはエラー(t *testing.T) {
	t.Parallel()

	_, err := NewFont("file/fonts/not-exist.ttf")

	require.Error(t, err)
	assert.ErrorContains(t, err, "file/fonts/not-exist.ttf")
}
