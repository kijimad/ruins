package mapplanner

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlannerTypeDebugTown_テンプレが読めて街用NPCと収納箱を配置する(t *testing.T) {
	t.Parallel()
	chain, err := PlannerTypeDebugTown.PlannerFunc(0, 0, 12345)
	require.NoError(t, err)
	chain.PlanData.RawMaster = CreateTestRawMaster()
	require.NoError(t, chain.Plan())

	npcNames := make([]string, 0, len(chain.PlanData.NPCs))
	for _, n := range chain.PlanData.NPCs {
		npcNames = append(npcNames, n.Name)
	}
	assert.ElementsMatch(t, []string{"merchant", "old_soldier"}, npcNames, "街用NPC2種が配置される")

	propNames := make([]string, 0, len(chain.PlanData.Props))
	for _, p := range chain.PlanData.Props {
		propNames = append(propNames, p.Name)
	}
	assert.Contains(t, propNames, "wooden_crate", "収納箱が配置される")
	assert.Contains(t, propNames, "hearth", "焚き火の石組が配置される")

	assert.NotEmpty(t, chain.PlanData.SpawnPoints, "スポーン地点がある")
}
