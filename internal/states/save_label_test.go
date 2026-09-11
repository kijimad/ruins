package states

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/save"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFormatSaveSlotLabel はセーブスロットの表示ラベルを固定する。
// セーブディレクトリは WithSaveDir で一時ディレクトリへ差し替え、t.Chdir を避ける。
func TestFormatSaveSlotLabel(t *testing.T) {
	t.Parallel()

	t.Run("セーブが無ければダッシュを返す", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		sm, err := save.NewSerializationManager(save.WithSaveDir(t.TempDir()))
		require.NoError(t, err)

		assert.Equal(t, "---", formatSaveSlotLabel(world, sm, "slot1"))
	})

	t.Run("セーブがあればプレイヤー名を含むラベルを返す", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		e := world.ECS.NewEntity()
		world.Components.Name.Add(e, &gc.Name{Name: "Ash"})
		world.Components.Player.Add(e, &gc.Player{})

		sm, err := save.NewSerializationManager(save.WithSaveDir(t.TempDir()))
		require.NoError(t, err)
		require.NoError(t, sm.SaveWorld(world, "slot1"))

		label := formatSaveSlotLabel(world, sm, "slot1")
		assert.Contains(t, label, "Ash", "プレイヤー名を含む")
		assert.NotEqual(t, "---", label, "セーブがあればダッシュではない")
		assert.NotEqual(t, "Has data", label, "名前と日時が取れればフォールバックしない")
	})
}

// TestBackChoice_Runは前の画面へ戻る は戻る選択肢が Pop 遷移を返すことを検証する。
func TestBackChoice_Runは前の画面へ戻る(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	ch := backChoice(world)
	require.NotEmpty(t, ch.Label, "ラベルを持つ")

	tr, err := ch.Run(world)
	require.NoError(t, err)
	assert.Equal(t, es.TransPop, tr.Type, "戻るは Pop 遷移")
}
