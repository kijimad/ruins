package components

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// VisualEffect はビジュアルエフェクトのインターフェース
type VisualEffect interface {
	// Update はエフェクトを更新し、継続中ならtrueを返す
	Update(deltaMs float64) bool
}

// VisualEffects はエンティティに紐づくビジュアルエフェクトを管理する
type VisualEffects struct {
	Effects []VisualEffect
}

// FadeAnimation はフェードイン/ホールド/フェードアウトのアニメーション状態を管理する
type FadeAnimation struct {
	FadeInMs    float64 // フェードイン時間（ミリ秒）
	HoldMs      float64 // 表示維持時間（ミリ秒）
	FadeOutMs   float64 // フェードアウト時間（ミリ秒）
	TotalMs     float64 // 合計時間
	RemainingMs float64 // 残り時間（ミリ秒）
	Alpha       float64 // 現在の透明度（0.0-1.0）
}

// Update はフェードアニメーションを進め、継続中ならtrueを返す
func (a *FadeAnimation) Update(deltaMs float64) bool {
	a.RemainingMs -= deltaMs
	elapsed := a.TotalMs - a.RemainingMs
	a.Alpha = calculateFadeAlpha(elapsed, a.FadeInMs, a.HoldMs, a.FadeOutMs)
	return a.RemainingMs > 0
}

// calculateFadeAlpha はフェードシーケンスのAlpha値を計算する
func calculateFadeAlpha(elapsed, fadeInMs, holdMs, fadeOutMs float64) float64 {
	if elapsed < fadeInMs {
		if fadeInMs == 0 {
			return 1.0
		}
		return elapsed / fadeInMs
	}
	if elapsed < fadeInMs+holdMs {
		return 1.0
	}
	if fadeOutMs == 0 {
		return 0.0
	}
	fadeOutElapsed := elapsed - fadeInMs - holdMs
	return 1.0 - (fadeOutElapsed / fadeOutMs)
}

// TextProperties はテキスト表示に共通するプロパティをまとめる
type TextProperties struct {
	OffsetX float64    // X座標オフセット
	OffsetY float64    // Y座標オフセット
	Text    string     // 表示テキスト
	Color   color.RGBA // 表示色
}

// ScreenTextEffect は画面座標でテキストを表示するエフェクト
type ScreenTextEffect struct {
	FadeAnimation
	TextProperties
}

// SplashTextEffect はダンジョン開始時などに画面中央にスプラッシュ表示するエフェクト。
// 指定フォントで描画し、テキストの下に水平線を付ける
type SplashTextEffect struct {
	ScreenTextEffect
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
	holdMs := 1200.0
	fadeOutMs := 800.0
	totalMs := fadeInMs + holdMs + fadeOutMs

	return &SplashTextEffect{
		ScreenTextEffect: ScreenTextEffect{
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
		},
		Face:      face,
		LineWidth: float64(screenW) * 0.4,
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
