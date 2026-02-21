package systems

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVisualEffectSystem_DungeonTitle(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	// 画面サイズを設定
	world.Resources.SetScreenDimensions(800, 600)

	// ダンジョンタイトルエフェクトを作成
	titleEffect := gc.NewDungeonTitleEffect("テストダンジョン", 1, 800, 600)
	titleEntity := world.Manager.NewEntity()
	titleEntity.AddComponent(world.Components.VisualEffect, &gc.VisualEffect{
		Effects: []gc.EffectInstance{titleEffect},
	})

	// エフェクトが作成されたことを確認
	count := world.Manager.Join(world.Components.VisualEffect).Size()
	assert.Equal(t, 1, count, "VisualEffectエンティティが1つ存在するべき")

	// エフェクトの初期値を確認
	ve := world.Components.VisualEffect.Get(titleEntity).(*gc.VisualEffect)
	require.Len(t, ve.Effects, 1)
	effect := ve.Effects[0]

	assert.Equal(t, gc.EffectTypeScreenText, effect.Type)
	assert.Equal(t, "テストダンジョン 1F", effect.Text)
	assert.Equal(t, 400.0, effect.OffsetX, "画面中央X")
	assert.Equal(t, 300.0, effect.OffsetY, "画面中央Y")
	assert.Equal(t, 0.0, effect.Alpha, "初期Alpha")
	assert.Equal(t, 3000.0, effect.TotalMs, "合計時間")

	// Update実行後のAlphaを確認
	sys := &VisualEffectSystem{}
	err := sys.Update(world)
	require.NoError(t, err)

	ve = world.Components.VisualEffect.Get(titleEntity).(*gc.VisualEffect)
	require.Len(t, ve.Effects, 1)
	effect = ve.Effects[0]

	assert.Greater(t, effect.Alpha, 0.0, "Update後はAlphaが0より大きいべき")
	t.Logf("Alpha after update: %f", effect.Alpha)
}
