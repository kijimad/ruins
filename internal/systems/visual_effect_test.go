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
	titleEffect := gc.NewSplashTextEffect("テストダンジョン 1F", nil, 800, 600)
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
	effect, ok := ve.Effects[0].(*gc.SplashTextEffect)
	require.True(t, ok, "SplashTextEffectであるべき")

	assert.Equal(t, "テストダンジョン 1F", effect.Text)
	assert.Equal(t, 400.0, effect.OffsetX, "画面中央X")
	assert.Equal(t, 240.0, effect.OffsetY, "画面上部2/5")
	assert.Equal(t, 0.0, effect.Alpha, "初期Alpha")
	assert.Equal(t, 2800.0, effect.TotalMs, "合計時間")
	assert.Greater(t, effect.LineWidth, 0.0, "水平線が有効")

	// Update実行後のAlphaを確認
	sys := &VisualEffectSystem{}
	err := sys.Update(world)
	require.NoError(t, err)

	ve = world.Components.VisualEffect.Get(titleEntity).(*gc.VisualEffects)
	require.Len(t, ve.Effects, 1)
	effect, ok = ve.Effects[0].(*gc.SplashTextEffect)
	require.True(t, ok, "SplashTextEffectであるべき")

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
	titleEffect := gc.NewSplashTextEffect("テストダンジョン 1F", nil, 800, 600)
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

func TestVisualEffectSystem_DamageEffect(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	// ダメージエフェクトをGridElement付きで作成
	damageEffect := gc.NewDamageEffect(99)
	entity := world.Manager.NewEntity()
	entity.AddComponent(world.Components.GridElement, &gc.GridElement{X: 3, Y: 4})
	entity.AddComponent(world.Components.VisualEffect, &gc.VisualEffects{
		Effects: []gc.VisualEffect{damageEffect},
	})

	// 初期値を確認
	ve := world.Components.VisualEffect.Get(entity).(*gc.VisualEffects)
	require.Len(t, ve.Effects, 1)
	effect, ok := ve.Effects[0].(*gc.DamageTextEffect)
	require.True(t, ok)

	assert.Equal(t, "99", effect.Text)
	assert.Equal(t, -8.0, effect.OffsetY)
	initialY := effect.OffsetY

	// Update実行後にY座標が移動することを確認
	sys := &VisualEffectSystem{}
	err := sys.Update(world)
	require.NoError(t, err)

	ve = world.Components.VisualEffect.Get(entity).(*gc.VisualEffects)
	require.Len(t, ve.Effects, 1)
	effect = ve.Effects[0].(*gc.DamageTextEffect)
	assert.Less(t, effect.OffsetY, initialY, "VelocityYが負なのでY座標が減少する")
	assert.Greater(t, effect.Alpha, 0.0, "フェードインが始まる")
}

func TestVisualEffectSystem_MissEffect(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	missEffect := gc.NewMissEffect()
	entity := world.Manager.NewEntity()
	entity.AddComponent(world.Components.GridElement, &gc.GridElement{X: 5, Y: 5})
	entity.AddComponent(world.Components.VisualEffect, &gc.VisualEffects{
		Effects: []gc.VisualEffect{missEffect},
	})

	ve := world.Components.VisualEffect.Get(entity).(*gc.VisualEffects)
	require.Len(t, ve.Effects, 1)
	effect, ok := ve.Effects[0].(*gc.DamageTextEffect)
	require.True(t, ok, "MissEffectはDamageTextEffect型")
	assert.Equal(t, "MISS", effect.Text)

	// Update後もエフェクトが継続する
	sys := &VisualEffectSystem{}
	err := sys.Update(world)
	require.NoError(t, err)

	ve = world.Components.VisualEffect.Get(entity).(*gc.VisualEffects)
	assert.Len(t, ve.Effects, 1, "1フレーム目ではまだアクティブ")
}

func TestVisualEffectSystem_HealEffect(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	healEffect := gc.NewHealEffect(30)
	entity := world.Manager.NewEntity()
	entity.AddComponent(world.Components.GridElement, &gc.GridElement{X: 2, Y: 3})
	entity.AddComponent(world.Components.VisualEffect, &gc.VisualEffects{
		Effects: []gc.VisualEffect{healEffect},
	})

	ve := world.Components.VisualEffect.Get(entity).(*gc.VisualEffects)
	effect := ve.Effects[0].(*gc.DamageTextEffect)
	assert.Equal(t, "+30", effect.Text)

	sys := &VisualEffectSystem{}
	err := sys.Update(world)
	require.NoError(t, err)

	ve = world.Components.VisualEffect.Get(entity).(*gc.VisualEffects)
	assert.Len(t, ve.Effects, 1, "1フレーム目ではまだアクティブ")
}

func TestVisualEffectSystem_MultipleEffectsOnEntity(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	// 1つのエンティティに複数エフェクトを付与
	entity := world.Manager.NewEntity()
	entity.AddComponent(world.Components.GridElement, &gc.GridElement{X: 3, Y: 3})
	entity.AddComponent(world.Components.VisualEffect, &gc.VisualEffects{
		Effects: []gc.VisualEffect{
			gc.NewDamageEffect(50),
			gc.NewMissEffect(),
		},
	})

	ve := world.Components.VisualEffect.Get(entity).(*gc.VisualEffects)
	assert.Len(t, ve.Effects, 2)

	// Update後も両方アクティブ
	sys := &VisualEffectSystem{}
	err := sys.Update(world)
	require.NoError(t, err)

	ve = world.Components.VisualEffect.Get(entity).(*gc.VisualEffects)
	assert.Len(t, ve.Effects, 2, "両エフェクトがまだアクティブ")
}

func TestVisualEffectSystem_DamageEffectCompletion(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	// 残り時間が極小のダメージエフェクトを作成
	effect := &gc.DamageTextEffect{
		FadeAnimation: gc.FadeAnimation{
			FadeInMs: 100, HoldMs: 500, FadeOutMs: 200,
			TotalMs: 800, RemainingMs: 1,
		},
		TextProperties: gc.TextProperties{
			Text: "1", OffsetY: -8,
		},
		VelocityY: -0.5,
	}
	entity := world.Manager.NewEntity()
	entity.AddComponent(world.Components.GridElement, &gc.GridElement{X: 1, Y: 1})
	entity.AddComponent(world.Components.VisualEffect, &gc.VisualEffects{
		Effects: []gc.VisualEffect{effect},
	})

	sys := &VisualEffectSystem{}
	for i := 0; i < 5; i++ {
		err := sys.Update(world)
		require.NoError(t, err)
	}

	count := world.Manager.Join(world.Components.VisualEffect).Size()
	assert.Equal(t, 0, count, "完了したダメージエフェクトのエンティティは削除される")
}

func TestVisualEffectSystem_EffectCompletion(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	// 完了間近のエフェクトを作成
	effect := &gc.SplashTextEffect{
		FadeAnimation: gc.FadeAnimation{
			Alpha:       0.01,
			TotalMs:     100,
			RemainingMs: 1, // ほぼ完了（残り1ミリ秒）
		},
		TextProperties: gc.TextProperties{
			Text:    "テスト",
			OffsetX: 100,
			OffsetY: 100,
		},
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
