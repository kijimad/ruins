package states

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/save"
	"github.com/kijimaD/ruins/internal/testutil"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// saveWithPlayer はプレイヤー1体を保存した world と、その SerializationManager を返す。
func saveWithPlayer(t *testing.T, slot string) (w.World, *save.SerializationManager) {
	t.Helper()
	world := testutil.InitTestWorld(t)
	e := world.ECS.NewEntity()
	world.Components.Name.Add(e, &gc.Name{Name: "Ash"})
	world.Components.Player.Add(e, &gc.Player{})

	sm, err := save.NewSerializationManager(save.WithSaveDir(t.TempDir()))
	require.NoError(t, err)
	require.NoError(t, sm.SaveWorld(world, slot))
	return world, sm
}

// TestInteractionChoices_プレイヤー不在なら空の選択肢 を検証する。
func TestInteractionChoices_プレイヤー不在なら空の選択肢(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	header, choices := interactionChoices(world)
	assert.Empty(t, header)
	assert.Empty(t, choices)
}

// TestSameTileActionChoices_プレイヤー不在なら空の選択肢 を検証する。
func TestSameTileActionChoices_プレイヤー不在なら空の選択肢(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	header, choices := sameTileActionChoices(world)
	assert.Empty(t, header)
	assert.Empty(t, choices)
}

// TestLoadSlotChoice はセーブの有無でスロット選択肢が変わることを検証する。
func TestLoadSlotChoice(t *testing.T) {
	t.Parallel()

	t.Run("セーブが無ければ空スロットの選択肢", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		sm, err := save.NewSerializationManager(save.WithSaveDir(t.TempDir()))
		require.NoError(t, err)

		ch := loadSlotChoice(world, sm, "slot1")
		assert.Equal(t, "---", ch.Label, "空スロットはダッシュ")
		assert.True(t, ch.Header)
	})

	t.Run("セーブがあれば名前付きの選択肢を組みRunでロードする", func(t *testing.T) {
		t.Parallel()
		world, sm := saveWithPlayer(t, "slot1")

		ch := loadSlotChoice(world, sm, "slot1")
		assert.NotEqual(t, "---", ch.Label, "セーブありはダッシュでない")
		assert.Contains(t, ch.Label, "Ash", "プレイヤー名を含む")
		assert.Equal(t, 1, ch.Indent)

		// Run はセーブをロードして再開状態へ置き換える
		tr, err := ch.Run(world)
		require.NoError(t, err)
		assert.Equal(t, es.TransReplace, tr.Type, "ロード成功は Replace 遷移")
	})
}
