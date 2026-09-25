package overworld

import (
	"testing"

	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestScatterZoneLootGroup_ゾーンごとに低価値groupを持つ は屋外散布 loot のゾーン別 group を固定する。
// 屋外には屑物だけを置く方針で、道沿いは紙屑、奥地は廃材・鉱片を引く。武器・防具・回復薬を含むテーブルは
// 使わない。lootGroup は raw.toml の scatterZones 行が単一出典。
func TestScatterZoneLootGroup_ゾーンごとに低価値groupを持つ(t *testing.T) {
	t.Parallel()

	raws := testutil.InitTestWorld(t).Resources.RawMaster

	road, ok := raw.GetScatterZone(raws, "roadside")
	require.True(t, ok, "roadside ゾーンが定義されている")
	assert.Equal(t, "scrap_of_paper", road.LootGroup, "道沿いは紙屑の group")

	wild, ok := raw.GetScatterZone(raws, "wild")
	require.True(t, ok, "wild ゾーンが定義されている")
	assert.Equal(t, "junk", wild.LootGroup, "奥地は廃材・くず鉄の group")
}
