package components

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/rawschema"
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
	e.Offset.Y += e.VelocityY
	return e.FadeAnimation.Update(deltaMs)
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
			Offset: consts.Coord[float64]{X: float64(screenW) / 2, Y: float64(screenH) * 2 / 5},
			Text:   textStr,
			Color:  color.RGBA{255, 255, 255, 255},
		},
		Face:      face,
		LineWidth: float64(screenW) * 0.7,
	}
}

// NewDamageEffect はダメージ数値表示エフェクトを作成する。数字は攻撃の属性の色で出し、
// 属性の倍率 elementMult で減ったら RESIST、増えたら WEAK を添える
func NewDamageEffect(damage int, element rawschema.Element, elementMult consts.Percent) *DamageTextEffect {
	text := fmt.Sprintf("%d", damage)
	switch {
	case elementMult < consts.PercentBase:
		text += " RESIST"
	case elementMult > consts.PercentBase:
		text += " WEAK"
	}
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
			Offset: consts.Coord[float64]{X: 0, Y: -8},
			Text:   text,
			Color:  elementDamageColor(element),
		},
		VelocityY: -0.5,
	}
}

// elementDamageColor は属性ごとのダメージの数字の色を返す。無属性は赤
func elementDamageColor(element rawschema.Element) color.RGBA {
	switch element {
	case rawschema.ElementNONE:
		return noElementDamageColor
	case rawschema.ElementFIRE:
		return color.RGBA{255, 150, 40, 255}
	case rawschema.ElementTHUNDER:
		return color.RGBA{255, 230, 60, 255}
	case rawschema.ElementCHILL:
		return color.RGBA{120, 210, 255, 255}
	case rawschema.ElementPHOTON:
		return color.RGBA{255, 255, 230, 255}
	}
	return noElementDamageColor
}

// noElementDamageColor は無属性のダメージの数字の色
var noElementDamageColor = color.RGBA{255, 80, 80, 255}

// NewConditionDamageEffect は失血や低体温など不調で削れた HP の数字を、攻撃のダメージと見分けられる紫で浮かべる
func NewConditionDamageEffect(damage int) *DamageTextEffect {
	effect := NewDamageEffect(damage, rawschema.ElementNONE, consts.PercentBase)
	effect.Color = color.RGBA{190, 110, 255, 255}
	return effect
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
			Offset: consts.Coord[float64]{X: 0, Y: -8},
			Text:   "MISS",
			Color:  color.RGBA{180, 180, 180, 255},
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
			Offset: consts.Coord[float64]{X: 0, Y: -8},
			Text:   fmt.Sprintf("+%d", amount),
			Color:  color.RGBA{80, 255, 80, 255},
		},
		VelocityY: -0.5,
	}
}
