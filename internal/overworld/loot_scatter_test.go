package overworld

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestOutdoorLootTableFor_ゾーンごとにテーブルを返す は屋外散布 loot のゾーン別テーブル写像を固定する。
// 道沿いは廃墟の残り物、奥地は野外の loot を引く。ゾーンを足して case を忘れると exhaustive lint が止める。
func TestOutdoorLootTableFor_ゾーンごとにテーブルを返す(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "ruins_area", outdoorLootTableFor(zoneRoadside), "道沿いは廃墟テーブル")
	assert.Equal(t, "forest", outdoorLootTableFor(zoneWild), "奥地は森テーブル")
}

// TestOutdoorLootTableFor_未知ゾーンはpanic は不変条件違反を早期に落とすことを固定する。
func TestOutdoorLootTableFor_未知ゾーンはpanic(t *testing.T) {
	t.Parallel()

	assert.Panics(t, func() { outdoorLootTableFor(outdoorZone("unknown")) }, "未知ゾーンは panic する")
}
