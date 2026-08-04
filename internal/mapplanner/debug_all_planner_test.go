package mapplanner

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDebugPopulatePlanner_街用NPCと収納箱だけを配置する(t *testing.T) {
	t.Parallel()

	chain, err := NewBigRoomPlanner(40, 40, 12345)
	require.NoError(t, err)

	player := oapi.IsPlayer(true)
	neutral := oapi.FactionMemberType(gc.FactionNeutralName)
	members := []oapi.Member{
		{Name: "商人", FactionType: &neutral},
		{Name: "酒場の主人", FactionType: &neutral},
		{Name: "火の玉"},                  // 敵、faction 無し
		{Name: "Ash", Player: &player}, // プレイヤー
	}
	rawMaster := CreateTestRawMaster()
	rawMaster.Members = &members
	chain.PlanData.RawMaster = rawMaster

	// 先に大部屋を描画してから配置する。配置は部屋内のスポーン可能タイルに限る
	require.NoError(t, chain.Plan())
	require.NoError(t, DebugPopulatePlanner{}.PlanMeta(&chain.PlanData))

	npcNames := make([]string, 0, len(chain.PlanData.NPCs))
	for _, n := range chain.PlanData.NPCs {
		npcNames = append(npcNames, n.Name)
	}
	assert.ElementsMatch(t, []string{"商人", "酒場の主人"}, npcNames, "中立factionの街用NPCだけが配置される")
	assert.NotContains(t, npcNames, "火の玉", "敵は配置しない")
	assert.NotContains(t, npcNames, "Ash", "プレイヤーキャラクターは配置しない")

	propNames := make([]string, 0, len(chain.PlanData.Props))
	for _, p := range chain.PlanData.Props {
		propNames = append(propNames, p.Name)
	}
	assert.Equal(t, []string{debugStorageBoxName}, propNames, "収納箱だけが配置される")
}

func TestDebugPopulatePlanner_RawMasterが無ければ何もしない(t *testing.T) {
	t.Parallel()

	chain, err := NewBigRoomPlanner(40, 40, 12345)
	require.NoError(t, err)
	chain.PlanData.RawMaster = CreateTestRawMaster()
	require.NoError(t, chain.Plan())

	chain.PlanData.RawMaster = nil
	require.NoError(t, DebugPopulatePlanner{}.PlanMeta(&chain.PlanData))
	assert.Empty(t, chain.PlanData.NPCs)
	assert.Empty(t, chain.PlanData.Props)
}

func TestPlannerTypeByName_デバッグ専用プランナーも解決できる(t *testing.T) {
	t.Parallel()

	pt, ok := PlannerTypeByName(PlannerTypeDebugAll.Name)
	require.True(t, ok, "デバッグ専用プランナーは名前で引ける")
	assert.Equal(t, PlannerTypeDebugAll.Name, pt.Name)
}
