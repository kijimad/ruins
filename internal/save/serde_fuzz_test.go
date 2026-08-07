package save

import (
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// FuzzDeserializeWorld は arkserde のデシリアライズ境界に任意バイト列を流し、
// 壊れたセーブデータでも panic せず error で返ることを保証する。
func FuzzDeserializeWorld(f *testing.F) {
	base := testutil.InitTestWorld(f)
	_, err := lifecycle.SpawnPlayer(base, consts.Coord[consts.Tile]{X: 5, Y: 5}, "Ash")
	require.NoError(f, err)
	_, err = lifecycle.SpawnEnemy(base, consts.Coord[consts.Tile]{X: 8, Y: 8}, "火の玉")
	require.NoError(f, err)
	valid, err := serializeWorld(base)
	require.NoError(f, err)

	f.Add(valid)
	f.Add([]byte(``))
	f.Add([]byte(`{}`))
	f.Add([]byte(`[]`))
	f.Add([]byte(`null`))
	f.Add([]byte(`{"components":null}`))

	f.Fuzz(func(t *testing.T, worldJSON []byte) {
		world := testutil.InitTestWorld(t)
		// arkserde の Deserialize はリセット済みワールドを要求する
		world.ECS.Reset()
		// 任意入力でも panic しないことだけを保証する。error 返却は許容
		assert.NotPanics(t, func() {
			_ = deserializeWorld(world, worldJSON)
		})
	})
}

// FuzzRestoreWorldFromJSON はロードの全経路（封筒パース→チェックサム→デシリアライズ→シングルトン再確立）
// に任意文字列を流し、壊れたセーブファイルでも panic しないことを保証する。
func FuzzRestoreWorldFromJSON(f *testing.F) {
	base := testutil.InitTestWorld(f)
	_, err := lifecycle.SpawnPlayer(base, consts.Coord[consts.Tile]{X: 5, Y: 5}, "Ash")
	require.NoError(f, err)
	sm, err := NewSerializationManager(WithSaveDir(f.TempDir()))
	require.NoError(f, err)
	validJSON, err := sm.GenerateWorldJSON(base)
	require.NoError(f, err)

	f.Add(validJSON)
	f.Add(``)
	f.Add(`not json`)
	f.Add(`{}`)
	f.Add(`{"version":"` + saveDataVersion + `","world":{}}`)

	f.Fuzz(func(t *testing.T, jsonData string) {
		world := testutil.InitTestWorld(t)
		sm, err := NewSerializationManager(WithSaveDir(t.TempDir()))
		require.NoError(t, err)
		// 壊れた入力でも panic せず error で返ること
		assert.NotPanics(t, func() {
			_ = sm.RestoreWorldFromJSON(world, jsonData)
		})
	})
}
