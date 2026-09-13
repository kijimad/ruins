package uicore_test

import (
	"math"
	"testing"

	"github.com/kijimaD/ruins/internal/widgets/uicore"
	"github.com/stretchr/testify/assert"
)

func TestResolveText_既定は左上基準で回転なし(t *testing.T) {
	t.Parallel()

	p := uicore.ResolveText()
	assert.Equal(t, uicore.AnchorTopLeft, p.Anchor)
	assert.Zero(t, p.Angle)
}

func TestResolveText_Centeredは中央基準にする(t *testing.T) {
	t.Parallel()

	p := uicore.ResolveText(uicore.Centered())
	assert.Equal(t, uicore.AnchorCenter, p.Anchor)
	assert.Zero(t, p.Angle)
}

func TestResolveText_Rotatedは角度を持ち中央基準を伴う(t *testing.T) {
	t.Parallel()

	p := uicore.ResolveText(uicore.Rotated(math.Pi / 3))
	assert.Equal(t, uicore.AnchorCenter, p.Anchor, "回転は中央を軸にする")
	assert.InDelta(t, math.Pi/3, p.Angle, 1e-9)
}

func TestResolveText_角度があれば順序によらず中央基準へ揃う(t *testing.T) {
	t.Parallel()

	// Rotated の後に Centered を重ねても、角度と中央基準の不変条件は保たれる
	p := uicore.ResolveText(uicore.Rotated(math.Pi/2), uicore.Centered())
	assert.Equal(t, uicore.AnchorCenter, p.Anchor)
	assert.InDelta(t, math.Pi/2, p.Angle, 1e-9)
}

func TestResolveText_Rotated0は回転なしの中央揃えになる(t *testing.T) {
	t.Parallel()

	// Rotated(0) は Angle=0 なので正規化は働かないが、Rotated が中央基準を立てる。
	// 結果は回転なし・中央揃えで Centered() と等価になる
	p := uicore.ResolveText(uicore.Rotated(0))
	assert.Equal(t, uicore.AnchorCenter, p.Anchor)
	assert.Zero(t, p.Angle)
	assert.Equal(t, uicore.ResolveText(uicore.Centered()), p, "Rotated(0) は Centered() と等価")
}
