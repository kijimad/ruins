package save

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGzipBytes_往復で元のJSONに戻る(t *testing.T) {
	t.Parallel()
	original := []byte(`{"version":"1","world":{"entities":[1,2,3,1,2,3,1,2,3]}}`)

	compressed, err := gzipBytes(original)
	require.NoError(t, err)
	assert.Equal(t, []byte{0x1f, 0x8b}, compressed[:2], "gzip の magic で始まる")

	restored, err := gunzipBytes(compressed)
	require.NoError(t, err)
	assert.Equal(t, original, restored)
}

func TestGunzipBytes_gzipでないデータはエラー(t *testing.T) {
	t.Parallel()
	_, err := gunzipBytes([]byte("not gzip data"))
	require.Error(t, err)
}
