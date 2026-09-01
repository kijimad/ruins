package gameaction

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContractIllness(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)

	contracted := ContractIllness(world, player, gc.ConditionLiverIllness)
	assert.True(t, contracted, "発症する")
	cond := world.Components.HealthStatus.Get(player).Parts[gc.BodyPartTorso].GetCondition(gc.ConditionLiverIllness)
	require.NotNil(t, cond, "胴に病気が付く")
	assert.True(t, cond.IsActive(), "発症済みで即座に効く")
	assert.True(t, world.Components.StatsChanged.Has(player), "StatsChanged を立てる")

	// 既に罹患していれば重ねて発症させない
	again := ContractIllness(world, player, gc.ConditionLiverIllness)
	assert.False(t, again, "二重発症しない")
}

func TestContractIllness_HealthStatusなしは何もしない(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	entity := world.ECS.NewEntity()
	assert.False(t, ContractIllness(world, entity, gc.ConditionLiverIllness))
	assert.False(t, world.Components.StatsChanged.Has(entity))
}
