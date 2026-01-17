package mapspawner

import (
	"testing"

	"github.com/kijimaD/ruins/internal/raw"
	"github.com/stretchr/testify/require"
)

// createTestRawMaster はテスト用のRawMasterを作成する
// raw.tomlファイルから読み込む
func createTestRawMaster(t *testing.T) *raw.Master {
	t.Helper()
	rawMaster, err := raw.LoadFromFile("metadata/entities/raw/raw.toml")
	require.NoError(t, err, "raw.tomlの読み込みに失敗")
	return &rawMaster
}
