package components

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// SplashTextEffect はダンジョン開始時などに画面中央にスプラッシュ表示するエフェクト。
// 指定フォントで描画し、テキストの下に水平線を付ける
type SplashTextEffect struct {
	FadeAnimation
	TextProperties
	Face      text.Face // 描画に使用するフォント
	LineWidth float64   // テキスト下の水平線の幅
}

// DamageTextEffect はエンティティ座標で浮遊テキストを表示するエフェクト。
// ダメージ、回復、ミス、ヒットなどに使用する
type DamageTextEffect struct {
	FadeAnimation
	TextProperties
	VelocityY float64 // Y方向の速度（上に浮かぶ効果用）
}

// Update はフェードアニメーションを進めつつY座標を移動させる
func (e *DamageTextEffect) Update(deltaMs float64) bool {
	e.OffsetY += e.VelocityY
	return e.FadeAnimation.Update(deltaMs)
}

// SpriteFadeoutEffect はスプライトをフェードアウト表示するエフェクト。
// 敵撃破時などに使用する
type SpriteFadeoutEffect struct {
	FadeAnimation
	SpriteSheetName string // スプライトシート名
	SpriteKey       string // スプライトキー
}

// NewSplashTextEffect はダンジョン開始時などのスプラッシュエフェクトを作成する
func NewSplashTextEffect(textStr string, face text.Face, screenW, screenH int) *SplashTextEffect {
	fadeInMs := 800.0
	holdMs := 1600.0
	fadeOutMs := 800.0
	totalMs := fadeInMs + holdMs + fadeOutMs

	return &SplashTextEffect{
		FadeAnimation: FadeAnimation{
			FadeInMs:    fadeInMs,
			HoldMs:      holdMs,
			FadeOutMs:   fadeOutMs,
			TotalMs:     totalMs,
			RemainingMs: totalMs,
			Alpha:       0.0,
		},
		TextProperties: TextProperties{
			OffsetX: float64(screenW) / 2,
			OffsetY: float64(screenH) * 2 / 5,
			Text:    textStr,
			Color:   color.RGBA{255, 255, 255, 255},
		},
		Face:      face,
		LineWidth: float64(screenW) * 0.7,
	}
}

// NewDamageEffect はダメージ数値表示エフェクトを作成する
func NewDamageEffect(damage int) *DamageTextEffect {
	totalMs := 800.0
	return &DamageTextEffect{
		FadeAnimation: FadeAnimation{
			FadeInMs:    100,
			HoldMs:      500,
			FadeOutMs:   200,
			TotalMs:     totalMs,
			RemainingMs: totalMs,
			Alpha:       0.0,
		},
		TextProperties: TextProperties{
			OffsetX: 0,
			OffsetY: -8,
			Text:    fmt.Sprintf("%d", damage),
			Color:   color.RGBA{255, 80, 80, 255},
		},
		VelocityY: -0.5,
	}
}

// NewMissEffect はミス表示エフェクトを作成する
func NewMissEffect() *DamageTextEffect {
	totalMs := 600.0
	return &DamageTextEffect{
		FadeAnimation: FadeAnimation{
			FadeInMs:    50,
			HoldMs:      400,
			FadeOutMs:   150,
			TotalMs:     totalMs,
			RemainingMs: totalMs,
			Alpha:       0.0,
		},
		TextProperties: TextProperties{
			OffsetX: 0,
			OffsetY: -8,
			Text:    "MISS",
			Color:   color.RGBA{180, 180, 180, 255},
		},
		VelocityY: -0.3,
	}
}

// NewHealEffect は回復数値表示エフェクトを作成する
func NewHealEffect(amount int) *DamageTextEffect {
	totalMs := 800.0
	return &DamageTextEffect{
		FadeAnimation: FadeAnimation{
			FadeInMs:    100,
			HoldMs:      500,
			FadeOutMs:   200,
			TotalMs:     totalMs,
			RemainingMs: totalMs,
			Alpha:       0.0,
		},
		TextProperties: TextProperties{
			OffsetX: 0,
			OffsetY: -8,
			Text:    fmt.Sprintf("+%d", amount),
			Color:   color.RGBA{80, 255, 80, 255},
		},
		VelocityY: -0.5,
	}
}

// NewSpriteFadeoutEffect はスプライトフェードアウトエフェクトを作成する
func NewSpriteFadeoutEffect(spriteSheetName, spriteKey string) *SpriteFadeoutEffect {
	totalMs := 400.0
	return &SpriteFadeoutEffect{
		FadeAnimation: FadeAnimation{
			FadeInMs:    0,
			HoldMs:      100,
			FadeOutMs:   300,
			TotalMs:     totalMs,
			RemainingMs: totalMs,
			Alpha:       1.0,
		},
		SpriteSheetName: spriteSheetName,
		SpriteKey:       spriteKey,
	}
}
