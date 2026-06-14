package components

import (
	"image/color"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFadeAnimation_Update(t *testing.T) {
	t.Parallel()

	t.Run("フェードインフェーズでAlphaが増加する", func(t *testing.T) {
		t.Parallel()
		a := FadeAnimation{
			FadeInMs: 100, HoldMs: 100, FadeOutMs: 100,
			TotalMs: 300, RemainingMs: 300, Alpha: 0,
		}
		active := a.Update(50) // 50ms経過 → フェードイン中
		assert.True(t, active)
		assert.InDelta(t, 0.5, a.Alpha, 0.01)
	})

	t.Run("ホールドフェーズでAlphaが1.0になる", func(t *testing.T) {
		t.Parallel()
		a := FadeAnimation{
			FadeInMs: 100, HoldMs: 100, FadeOutMs: 100,
			TotalMs: 300, RemainingMs: 300, Alpha: 0,
		}
		a.Update(150) // 150ms経過 → ホールド中
		assert.InDelta(t, 1.0, a.Alpha, 0.01)
	})

	t.Run("フェードアウトフェーズでAlphaが減少する", func(t *testing.T) {
		t.Parallel()
		a := FadeAnimation{
			FadeInMs: 100, HoldMs: 100, FadeOutMs: 100,
			TotalMs: 300, RemainingMs: 300, Alpha: 0,
		}
		a.Update(250) // 250ms経過 → フェードアウト中
		assert.InDelta(t, 0.5, a.Alpha, 0.01)
	})

	t.Run("合計時間を超えるとfalseを返す", func(t *testing.T) {
		t.Parallel()
		a := FadeAnimation{
			FadeInMs: 100, HoldMs: 100, FadeOutMs: 100,
			TotalMs: 300, RemainingMs: 300, Alpha: 0,
		}
		active := a.Update(301)
		assert.False(t, active)
	})
}

func TestNewDamageEffect(t *testing.T) {
	t.Parallel()
	effect := NewDamageEffect(42)

	assert.Equal(t, "42", effect.Text)
	assert.Equal(t, color.RGBA{255, 80, 80, 255}, effect.Color)
	assert.Equal(t, -0.5, effect.VelocityY)
	assert.Equal(t, 0.0, effect.Alpha)
	assert.Equal(t, 800.0, effect.TotalMs)
	assert.Equal(t, effect.TotalMs, effect.RemainingMs)
	assert.Equal(t, -8.0, effect.OffsetY, "初期Y方向オフセット")
}

func TestNewMissEffect(t *testing.T) {
	t.Parallel()
	effect := NewMissEffect()

	assert.Equal(t, "MISS", effect.Text)
	assert.Equal(t, color.RGBA{180, 180, 180, 255}, effect.Color)
	assert.Equal(t, -0.3, effect.VelocityY)
	assert.Equal(t, 600.0, effect.TotalMs)
	assert.Equal(t, effect.TotalMs, effect.RemainingMs)
}

func TestNewHealEffect(t *testing.T) {
	t.Parallel()
	effect := NewHealEffect(25)

	assert.Equal(t, "+25", effect.Text)
	assert.Equal(t, color.RGBA{80, 255, 80, 255}, effect.Color)
	assert.Equal(t, -0.5, effect.VelocityY)
	assert.Equal(t, 800.0, effect.TotalMs)
}

func TestDamageTextEffect_Update_MovesY(t *testing.T) {
	t.Parallel()
	effect := NewDamageEffect(10)
	initialY := effect.OffsetY

	effect.Update(16.67) // 1フレーム分

	assert.Less(t, effect.OffsetY, initialY, "VelocityYが負なのでOffsetYは減少する")
	assert.Greater(t, effect.Alpha, 0.0, "フェードインが始まる")
}

func TestDamageTextEffect_Update_CompletionCycle(t *testing.T) {
	t.Parallel()
	effect := NewDamageEffect(100)

	// 全体の時間分だけ更新してエフェクトが完了することを確認
	active := true
	totalUpdates := 0
	for active {
		active = effect.Update(16.67)
		totalUpdates++
		require.Less(t, totalUpdates, 1000, "無限ループ防止")
	}

	assert.Greater(t, totalUpdates, 1, "複数フレームにわたってアニメーションする")
}

func TestNewSplashTextEffect(t *testing.T) {
	t.Parallel()
	effect := NewSplashTextEffect("テスト", nil, 800, 600)

	assert.Equal(t, "テスト", effect.Text)
	assert.Equal(t, 400.0, effect.OffsetX, "画面中央X")
	assert.Equal(t, 240.0, effect.OffsetY, "画面上部2/5")
	assert.Equal(t, 320.0, effect.LineWidth, "画面幅の40%")
	assert.Equal(t, 2800.0, effect.TotalMs)
	assert.Equal(t, color.RGBA{255, 255, 255, 255}, effect.Color)
}

func TestNewSpriteFadeoutEffect(t *testing.T) {
	t.Parallel()
	effect := NewSpriteFadeoutEffect("character", "slime_0")

	assert.Equal(t, "character", effect.SpriteSheetName)
	assert.Equal(t, "slime_0", effect.SpriteKey)
	assert.Equal(t, 1.0, effect.Alpha, "フェードアウトなので初期Alphaは1.0")
	assert.Equal(t, 400.0, effect.TotalMs)
	assert.Equal(t, 0.0, effect.FadeInMs, "フェードインなし")
}

func TestVisualEffects_MultipleEffects(t *testing.T) {
	t.Parallel()
	ve := &VisualEffects{
		Effects: []VisualEffect{
			NewDamageEffect(10),
			NewMissEffect(),
			NewHealEffect(5),
		},
	}

	assert.Len(t, ve.Effects, 3)

	// 全エフェクトを更新
	for _, effect := range ve.Effects {
		active := effect.Update(16.67)
		assert.True(t, active, "1フレーム目では全エフェクトがアクティブ")
	}
}
