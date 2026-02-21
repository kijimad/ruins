package components

import (
	"fmt"
	"image/color"
)

// VisualEffect はエンティティに紐づくビジュアルエフェクトを管理する
// GridElementを持つエンティティの場合はワールド座標に追従して描画される
// GridElementを持たないエンティティの場合は画面座標で描画される
type VisualEffect struct {
	Effects []EffectInstance
}

// EffectInstance は個々のエフェクトインスタンス
type EffectInstance struct {
	Type        EffectType // エフェクトの種類
	RemainingMs float64    // 残り時間（ミリ秒）
	OffsetX     float64    // X座標オフセット（ピクセル）
	OffsetY     float64    // Y座標オフセット（ピクセル）
	Text        string     // テキスト表示用
	Color       color.RGBA // 表示色
	VelocityY   float64    // Y方向の速度（上に浮かぶ効果用）

	// フェードアニメーション用
	FadeInMs  float64 // フェードイン時間（ミリ秒）
	HoldMs    float64 // 表示維持時間（ミリ秒）
	FadeOutMs float64 // フェードアウト時間（ミリ秒）
	TotalMs   float64 // 合計時間（初期化時に計算）
	Alpha     float64 // 現在の透明度（0.0-1.0）
}

// EffectType はエフェクトの種類
type EffectType int

const (
	// EffectTypeDamage はダメージ数値表示エフェクト
	EffectTypeDamage EffectType = iota
	// EffectTypeHeal は回復数値表示エフェクト
	EffectTypeHeal
	// EffectTypeHit は被弾エフェクト
	EffectTypeHit
	// EffectTypeMiss はミス表示エフェクト
	EffectTypeMiss
	// EffectTypeScreenText は画面テキスト表示エフェクト
	EffectTypeScreenText
)

// NewScreenTextEffect は画面中央にフェード表示するテキストエフェクトを作成する
func NewScreenTextEffect(text string, screenW, screenH int) EffectInstance {
	fadeInMs := 500.0
	holdMs := 2000.0
	fadeOutMs := 500.0
	totalMs := fadeInMs + holdMs + fadeOutMs

	return EffectInstance{
		Type:        EffectTypeScreenText,
		RemainingMs: totalMs,
		OffsetX:     float64(screenW) / 2,
		OffsetY:     float64(screenH) / 2,
		Text:        text,
		Color:       color.RGBA{255, 255, 255, 255},
		VelocityY:   0,
		FadeInMs:    fadeInMs,
		HoldMs:      holdMs,
		FadeOutMs:   fadeOutMs,
		TotalMs:     totalMs,
		Alpha:       0.0,
	}
}

// NewDungeonTitleEffect はダンジョンタイトル表示エフェクトを作成する
func NewDungeonTitleEffect(dungeonName string, depth int, screenW, screenH int) EffectInstance {
	text := fmt.Sprintf("%s %dF", dungeonName, depth)
	return NewScreenTextEffect(text, screenW, screenH)
}

// NewDamageEffect はダメージ数値表示エフェクトを作成する
func NewDamageEffect(damage int) EffectInstance {
	return EffectInstance{
		Type:        EffectTypeDamage,
		RemainingMs: 800,
		OffsetX:     0,
		OffsetY:     -8,
		Text:        fmt.Sprintf("%d", damage),
		Color:       color.RGBA{255, 80, 80, 255}, // 赤色
		VelocityY:   -0.5,                         // 上に浮かぶ
		FadeInMs:    100,
		HoldMs:      500,
		FadeOutMs:   200,
		TotalMs:     800,
		Alpha:       0.0,
	}
}

// NewMissEffect はミス表示エフェクトを作成する
func NewMissEffect() EffectInstance {
	return EffectInstance{
		Type:        EffectTypeMiss,
		RemainingMs: 600,
		OffsetX:     0,
		OffsetY:     -8,
		Text:        "MISS",
		Color:       color.RGBA{180, 180, 180, 255}, // グレー
		VelocityY:   -0.3,
		FadeInMs:    50,
		HoldMs:      400,
		FadeOutMs:   150,
		TotalMs:     600,
		Alpha:       0.0,
	}
}

// NewHealEffect は回復数値表示エフェクトを作成する
func NewHealEffect(amount int) EffectInstance {
	return EffectInstance{
		Type:        EffectTypeHeal,
		RemainingMs: 800,
		OffsetX:     0,
		OffsetY:     -8,
		Text:        fmt.Sprintf("+%d", amount),
		Color:       color.RGBA{80, 255, 80, 255}, // 緑色
		VelocityY:   -0.5,
		FadeInMs:    100,
		HoldMs:      500,
		FadeOutMs:   200,
		TotalMs:     800,
		Alpha:       0.0,
	}
}
