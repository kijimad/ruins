package systems

import (
	"image/color"
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
)

// TestGetFatigueBadgeColor は疲労段階からバッジ色を導く純関数を固定する。
// 過労だけ警告色、それ以外は注意色になる分岐を検証する。
func TestGetFatigueBadgeColor(t *testing.T) {
	t.Parallel()

	assert.Equal(t, color.RGBA{255, 50, 50, 255}, getFatigueBadgeColor(gc.FatigueExhausted), "過労は赤で警告する")
	for _, lv := range []gc.FatigueLevel{gc.FatigueRested, gc.FatigueNormal, gc.FatigueTired} {
		assert.Equal(t, color.RGBA{255, 200, 0, 255}, getFatigueBadgeColor(lv), "過労以外は黄")
	}
}

// TestShelterMsgid は遮蔽段階から表示 msgid を導く純関数を固定する。
// 未知の値は屋外へ落とすフォールバックも検証する。
func TestShelterMsgid(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "Indoor", shelterMsgid(gc.ShelterFull))
	assert.Equal(t, "Semi-outdoor", shelterMsgid(gc.ShelterPartial))
	assert.Equal(t, "Outdoor", shelterMsgid(gc.ShelterNone))
	assert.Equal(t, "Outdoor", shelterMsgid(gc.ShelterType(99)), "未知の値は屋外へ落とす")
}

// TestAmbientTempDisplayColor は環境温度から表示色を導く純関数を固定する。
// 快適域の内外で寒色・暖色・無彩色に分かれる境界を検証する。
func TestAmbientTempDisplayColor(t *testing.T) {
	t.Parallel()

	cold := color.RGBA{150, 190, 255, 255}
	warm := color.RGBA{255, 170, 120, 255}
	comfortable := color.RGBA{255, 255, 255, 255}

	assert.Equal(t, cold, ambientTempDisplayColor(query.ComfortableTempLower-1), "下限未満は寒色")
	assert.Equal(t, comfortable, ambientTempDisplayColor(query.ComfortableTempLower), "下限ちょうどは快適")
	assert.Equal(t, comfortable, ambientTempDisplayColor(query.ComfortableTempUpper), "上限ちょうどは快適")
	assert.Equal(t, warm, ambientTempDisplayColor(query.ComfortableTempUpper+1), "上限超は暖色")
}
