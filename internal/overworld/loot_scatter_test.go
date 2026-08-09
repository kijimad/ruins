package overworld

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestOutdoorLootGroupFor_ゾーンごとに低価値groupを返す は屋外散布 loot のゾーン別 group 写像を固定する。
// 屋外には屑物だけを置く方針で、道沿いは紙屑、奥地は廃材・鉱片を引く。武器・防具・回復薬を含むテーブルは
// 使わない。ゾーンを足して case を忘れると exhaustive lint が止める。
func TestOutdoorLootGroupFor_ゾーンごとに低価値groupを返す(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "scrap_of_paper", outdoorLootGroupFor(zoneRoadside), "道沿いは紙屑の group")
	assert.Equal(t, "materials", outdoorLootGroupFor(zoneWild), "奥地は廃材・鉱片の group")
}

// TestOutdoorLootGroupFor_未知ゾーンはpanic は不変条件違反を早期に落とすことを固定する。
func TestOutdoorLootGroupFor_未知ゾーンはpanic(t *testing.T) {
	t.Parallel()

	assert.Panics(t, func() { outdoorLootGroupFor(outdoorZone("unknown")) }, "未知ゾーンは panic する")
}
