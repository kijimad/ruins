package hud

import (
	"testing"

	"github.com/kijimaD/ruins/internal/widgets/theme"
	"github.com/stretchr/testify/assert"
)

// TestBodyTempFillColor_片方向_平熱は無彩色で冷えるほど青 は、体温ゲージの塗り色が片方向設計に
// 従うことを固定する。0が平熱かつ上限なので ratio=1 は無彩色の平熱、下がるほど寒色へ寄る。
// 旧・双方向設計では ratio=1 が火照りの赤になっていた回帰を防ぐ。
func TestBodyTempFillColor_片方向_平熱は無彩色で冷えるほど青(t *testing.T) {
	t.Parallel()

	assert.Equal(t, theme.HUDTempNeutral, bodyTempFillColor(1.0), "満タンの平熱は無彩色。火照りの赤にしない")
	assert.Equal(t, theme.HUDTempCold, bodyTempFillColor(0.0), "最も冷えた状態は寒色")
	// ratio が下がるほど赤みが減り寒色へ寄る
	assert.Less(t, bodyTempFillColor(0.2).R, bodyTempFillColor(0.8).R, "冷えるほど赤が減る")
}
