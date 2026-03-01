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
	titleEntity.AddComponent(world.Components.VisualEffect, &gc.VisualEffects{
		Effects: []gc.VisualEffect{titleEffect},
	})

	// エフェクトが作成されたことを確認
	count := world.Manager.Join(world.Components.VisualEffect).Size()
	assert.Equal(t, 1, count, "VisualEffectエンティティが1つ存在するべき")

	// エフェクトの初期値を確認
	ve := world.Components.VisualEffect.Get(titleEntity).(*gc.VisualEffects)
	require.Len(t, ve.Effects, 1)
	effect, ok := ve.Effects[0].(*gc.ScreenTextEffect)
	require.True(t, ok, "ScreenTextEffectであるべき")

	assert.Equal(t, "テストダンジョン 1F", effect.Text)
	assert.Equal(t, 400.0, effect.OffsetX, "画面中央X")
	assert.Equal(t, 200.0, effect.OffsetY, "画面上部1/3")
	assert.Equal(t, 0.0, effect.Alpha, "初期Alpha")
	assert.Equal(t, 3000.0, effect.TotalMs, "合計時間")

	// Update実行後のAlphaを確認
	sys := &VisualEffectSystem{}
	err := sys.Update(world)
	require.NoError(t, err)

	ve = world.Components.VisualEffect.Get(titleEntity).(*gc.VisualEffects)
	require.Len(t, ve.Effects, 1)
	effect, ok = ve.Effects[0].(*gc.ScreenTextEffect)
	require.True(t, ok, "ScreenTextEffectであるべき")

	assert.Greater(t, effect.Alpha, 0.0, "Update後はAlphaが0より大きいべき")
	t.Logf("Alpha after update: %f", effect.Alpha)
}

func TestVisualEffectSystem_SpriteFadeout(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	// シルエットエフェクトを作成
	spriteFadeoutEffect := gc.NewSpriteFadeoutEffect("character", "slime_0")
	effectEntity := world.Manager.NewEntity()
	effectEntity.AddComponent(world.Components.GridElement, &gc.GridElement{X: 5, Y: 5})
	effectEntity.AddComponent(world.Components.VisualEffect, &gc.VisualEffects{
		Effects: []gc.VisualEffect{spriteFadeoutEffect},
	})

	// エフェクトが作成されたことを確認
	count := world.Manager.Join(world.Components.VisualEffect).Size()
	assert.Equal(t, 1, count)

	// エフェクトの初期値を確認
	ve := world.Components.VisualEffect.Get(effectEntity).(*gc.VisualEffects)
	require.Len(t, ve.Effects, 1)
	effect, ok := ve.Effects[0].(*gc.SpriteFadeoutEffect)
	require.True(t, ok, "SpriteFadeoutEffectであるべき")

	assert.Equal(t, "character", effect.SpriteSheetName)
	assert.Equal(t, "slime_0", effect.SpriteKey)
	assert.Equal(t, 1.0, effect.Alpha, "初期Alphaは1.0")
	assert.Equal(t, 400.0, effect.TotalMs)

	// Update実行後のAlphaを確認（フェードアウトが進む）
	sys := &VisualEffectSystem{}
	err := sys.Update(world)
	require.NoError(t, err)

	ve = world.Components.VisualEffect.Get(effectEntity).(*gc.VisualEffects)
	require.Len(t, ve.Effects, 1)
	effect, ok = ve.Effects[0].(*gc.SpriteFadeoutEffect)
	require.True(t, ok, "SpriteFadeoutEffectであるべき")

	// ホールド期間中なのでまだ1.0のはず
	assert.Equal(t, 1.0, effect.Alpha, "ホールド期間中はAlphaが1.0")
}

func TestVisualEffectSystem_DisableAnimation(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	// アニメーションを無効化
	world.Config.DisableAnimation = true

	// エフェクトを作成
	titleEffect := gc.NewDungeonTitleEffect("テストダンジョン", 1, 800, 600)
	titleEntity := world.Manager.NewEntity()
	titleEntity.AddComponent(world.Components.VisualEffect, &gc.VisualEffects{
		Effects: []gc.VisualEffect{titleEffect},
	})

	// エフェクトが存在することを確認
	count := world.Manager.Join(world.Components.VisualEffect).Size()
	assert.Equal(t, 1, count)

	// Update実行（アニメーション無効時は即座に削除される）
	sys := &VisualEffectSystem{}
	err := sys.Update(world)
	require.NoError(t, err)

	// エフェクトエンティティが削除されたことを確認
	count = world.Manager.Join(world.Components.VisualEffect).Size()
	assert.Equal(t, 0, count, "アニメーション無効時はエフェクトが即座に削除される")
}

func TestVisualEffectSystem_EffectCompletion(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	// 完了間近のエフェクトを作成
	effect := &gc.ScreenTextEffect{
		Text:        "テスト",
		OffsetX:     100,
		OffsetY:     100,
		Alpha:       0.01,
		TotalMs:     100,
		RemainingMs: 1, // ほぼ完了（残り1ミリ秒）
	}
	effectEntity := world.Manager.NewEntity()
	effectEntity.AddComponent(world.Components.VisualEffect, &gc.VisualEffects{
		Effects: []gc.VisualEffect{effect},
	})

	// Update実行（エフェクトが完了して削除される）
	sys := &VisualEffectSystem{}

	// 複数回更新してエフェクトを完了させる
	for i := 0; i < 10; i++ {
		err := sys.Update(world)
		require.NoError(t, err)
	}

	// エフェクトエンティティが削除されたことを確認
	count := world.Manager.Join(world.Components.VisualEffect).Size()
	assert.Equal(t, 0, count, "完了したエフェクトのエンティティは削除される")
}
