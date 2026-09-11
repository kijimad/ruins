package systems

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/stretchr/testify/assert"
)

// TestShelterMsgid は遮蔽段階から表示 msgid を導く純関数を固定する。
// 未知の値は屋外へ落とすフォールバックも検証する。
//
// getFatigueBadgeColor と ambientTempDisplayColor の検証は
// hud_extractor_test.go 側に集約したため、ここでは重複を避けて shelterMsgid だけを扱う。
func TestShelterMsgid(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "Indoor", shelterMsgid(gc.ShelterFull))
	assert.Equal(t, "Semi-outdoor", shelterMsgid(gc.ShelterPartial))
	// ShelterNone は専用ケースを持たず、未知の値と同じ Outdoor フォールバックへ意図的に流れる
	assert.Equal(t, "Outdoor", shelterMsgid(gc.ShelterNone), "遮蔽なしは屋外")
	assert.Equal(t, "Outdoor", shelterMsgid(gc.ShelterType(99)), "未知の値は屋外へ落とす")
}
