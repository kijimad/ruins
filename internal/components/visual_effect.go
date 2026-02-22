package components

import (
	"fmt"
	"image/color"
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

// ScreenTextEffect は画面座標でテキストを表示するエフェクト
type ScreenTextEffect struct {
	OffsetX     float64    // X座標（画面座標）
	OffsetY     float64    // Y座標（画面座標）
	Text        string     // 表示テキスト
	Color       color.RGBA // 表示色
	FadeInMs    float64    // フェードイン時間（ミリ秒）
	HoldMs      float64    // 表示維持時間（ミリ秒）
	FadeOutMs   float64    // フェードアウト時間（ミリ秒）
	TotalMs     float64    // 合計時間
	RemainingMs float64    // 残り時間（ミリ秒）
	Alpha       float64    // 現在の透明度（0.0-1.0）
}

// Update はエフェクトを更新し、継続中ならtrueを返す
func (e *ScreenTextEffect) Update(deltaMs float64) bool {
	e.RemainingMs -= deltaMs
	elapsed := e.TotalMs - e.RemainingMs
	e.Alpha = calculateFadeAlpha(elapsed, e.FadeInMs, e.HoldMs, e.FadeOutMs)
	return e.RemainingMs > 0
}

// DamageTextEffect はエンティティ座標で浮遊テキストを表示するエフェクト
// ダメージ、回復、ミス、ヒットなどに使用する
type DamageTextEffect struct {
	OffsetX     float64    // X座標オフセット（ピクセル）
	OffsetY     float64    // Y座標オフセット（ピクセル）
	Text        string     // 表示テキスト
	Color       color.RGBA // 表示色
	VelocityY   float64    // Y方向の速度（上に浮かぶ効果用）
	FadeInMs    float64    // フェードイン時間（ミリ秒）
	HoldMs      float64    // 表示維持時間（ミリ秒）
	FadeOutMs   float64    // フェードアウト時間（ミリ秒）
	TotalMs     float64    // 合計時間
	RemainingMs float64    // 残り時間（ミリ秒）
	Alpha       float64    // 現在の透明度（0.0-1.0）
}

// Update はエフェクトを更新し、継続中ならtrueを返す
func (e *DamageTextEffect) Update(deltaMs float64) bool {
	e.RemainingMs -= deltaMs
	e.OffsetY += e.VelocityY
	elapsed := e.TotalMs - e.RemainingMs
	e.Alpha = calculateFadeAlpha(elapsed, e.FadeInMs, e.HoldMs, e.FadeOutMs)
	return e.RemainingMs > 0
}

// SpriteFadeoutEffect はスプライトをフェードアウト表示するエフェクト
// 敵撃破時などに使用する
type SpriteFadeoutEffect struct {
	SpriteSheetName string  // スプライトシート名
	SpriteKey       string  // スプライトキー
	FadeInMs        float64 // フェードイン時間（ミリ秒）
	HoldMs          float64 // 表示維持時間（ミリ秒）
	FadeOutMs       float64 // フェードアウト時間（ミリ秒）
	TotalMs         float64 // 合計時間
	RemainingMs     float64 // 残り時間（ミリ秒）
	Alpha           float64 // 現在の透明度（0.0-1.0）
}

// Update はエフェクトを更新し、継続中ならtrueを返す
func (e *SpriteFadeoutEffect) Update(deltaMs float64) bool {
	e.RemainingMs -= deltaMs
	elapsed := e.TotalMs - e.RemainingMs
	e.Alpha = calculateFadeAlpha(elapsed, e.FadeInMs, e.HoldMs, e.FadeOutMs)
	return e.RemainingMs > 0
}

// calculateFadeAlpha はフェードシーケンスのAlpha値を計算する
func calculateFadeAlpha(elapsed, fadeInMs, holdMs, fadeOutMs float64) float64 {
	if elapsed < fadeInMs {
		// フェードイン中
		if fadeInMs == 0 {
			return 1.0
		}
		return elapsed / fadeInMs
	}
	if elapsed < fadeInMs+holdMs {
		// ホールド中
		return 1.0
	}
	// フェードアウト中
	if fadeOutMs == 0 {
		return 0.0
	}
	fadeOutElapsed := elapsed - fadeInMs - holdMs
	return 1.0 - (fadeOutElapsed / fadeOutMs)
}

// NewScreenTextEffect は画面中央にフェード表示するテキストエフェクトを作成する
func NewScreenTextEffect(text string, screenW, screenH int) *ScreenTextEffect {
	fadeInMs := 500.0
	holdMs := 2000.0
	fadeOutMs := 500.0
	totalMs := fadeInMs + holdMs + fadeOutMs

	return &ScreenTextEffect{
		OffsetX:     float64(screenW) / 2,
		OffsetY:     float64(screenH) / 2,
		Text:        text,
		Color:       color.RGBA{255, 255, 255, 255},
		FadeInMs:    fadeInMs,
		HoldMs:      holdMs,
		FadeOutMs:   fadeOutMs,
		TotalMs:     totalMs,
		RemainingMs: totalMs,
		Alpha:       0.0,
	}
}

// NewDungeonTitleEffect はダンジョンタイトル表示エフェクトを作成する
func NewDungeonTitleEffect(dungeonName string, depth int, screenW, screenH int) *ScreenTextEffect {
	text := fmt.Sprintf("%s %dF", dungeonName, depth)
	return NewScreenTextEffect(text, screenW, screenH)
}

// NewDamageEffect はダメージ数値表示エフェクトを作成する
func NewDamageEffect(damage int) *DamageTextEffect {
	totalMs := 800.0
	return &DamageTextEffect{
		OffsetX:     0,
		OffsetY:     -8,
		Text:        fmt.Sprintf("%d", damage),
		Color:       color.RGBA{255, 80, 80, 255}, // 赤色
		VelocityY:   -0.5,                         // 上に浮かぶ
		FadeInMs:    100,
		HoldMs:      500,
		FadeOutMs:   200,
		TotalMs:     totalMs,
		RemainingMs: totalMs,
		Alpha:       0.0,
	}
}

// NewMissEffect はミス表示エフェクトを作成する
func NewMissEffect() *DamageTextEffect {
	totalMs := 600.0
	return &DamageTextEffect{
		OffsetX:     0,
		OffsetY:     -8,
		Text:        "MISS",
		Color:       color.RGBA{180, 180, 180, 255}, // グレー
		VelocityY:   -0.3,
		FadeInMs:    50,
		HoldMs:      400,
		FadeOutMs:   150,
		TotalMs:     totalMs,
		RemainingMs: totalMs,
		Alpha:       0.0,
	}
}

// NewHealEffect は回復数値表示エフェクトを作成する
func NewHealEffect(amount int) *DamageTextEffect {
	totalMs := 800.0
	return &DamageTextEffect{
		OffsetX:     0,
		OffsetY:     -8,
		Text:        fmt.Sprintf("+%d", amount),
		Color:       color.RGBA{80, 255, 80, 255}, // 緑色
		VelocityY:   -0.5,
		FadeInMs:    100,
		HoldMs:      500,
		FadeOutMs:   200,
		TotalMs:     totalMs,
		RemainingMs: totalMs,
		Alpha:       0.0,
	}
}

// NewSpriteFadeoutEffect はスプライトフェードアウトエフェクトを作成する
// 敵撃破時などにスプライトを表示してフェードアウトする
func NewSpriteFadeoutEffect(spriteSheetName, spriteKey string) *SpriteFadeoutEffect {
	totalMs := 400.0
	return &SpriteFadeoutEffect{
		SpriteSheetName: spriteSheetName,
		SpriteKey:       spriteKey,
		FadeInMs:        0,
		HoldMs:          100,
		FadeOutMs:       300,
		TotalMs:         totalMs,
		RemainingMs:     totalMs,
		Alpha:           1.0,
	}
}
