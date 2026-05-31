package worldhelper

import (
	"testing"

	"github.com/kijimaD/ruins/internal/gamelog"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetGameLog(t *testing.T) {
	t.Parallel()

	t.Run("InitWorldで生成されたGameLogシングルトンを取得できる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		store := GetGameLog(world)
		require.NotNil(t, store)
	})

	t.Run("取得したストアにログを書き込める", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		store := GetGameLog(world)
		require.NotNil(t, store)

		gamelog.New(store).Append("テストメッセージ").Log()

		recent := store.GetRecent(1)
		require.Len(t, recent, 1)
		assert.Contains(t, recent[0], "テストメッセージ")
	})

	t.Run("複数回取得しても同じストアを返す", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		store1 := GetGameLog(world)
		store2 := GetGameLog(world)

		assert.Same(t, store1, store2)
	})

	t.Run("異なるWorldは独立したストアを持つ", func(t *testing.T) {
		t.Parallel()
		world1 := testutil.InitTestWorld(t)
		world2 := testutil.InitTestWorld(t)

		store1 := GetGameLog(world1)
		store2 := GetGameLog(world2)

		gamelog.New(store1).Append("world1のログ").Log()

		assert.Equal(t, 1, store1.Count())
		assert.Equal(t, 0, store2.Count())
	})
}
