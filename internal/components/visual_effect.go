package components

import (
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
