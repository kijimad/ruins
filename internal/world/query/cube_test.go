package query_test

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
)

// addWeightEntity は指定ステージのフィールド上に束縛した重量エンティティを作る。suspended で退避中にする
func addWeightEntity(t *testing.T, world w.World, mg consts.Milligram, stage gc.StageKey, suspended bool) {
	t.Helper()
	e := world.ECS.NewEntity()
	world.Components.Weight.Add(e, &gc.Weight{Milligram: mg})
	world.Components.LocationOnField.Add(e, &gc.LocationOnField{})
	world.Components.StageBound.Add(e, &gc.StageBound{Key: stage})
	if suspended {
		world.Components.Suspended.Add(e, &gc.Suspended{})
	}
}

func TestPushCost(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		total consts.Milligram
		want  int
	}{
		{"空のキューブは基準APだけかかる", 0, consts.PushCostBase},
		{"総重量3kgで基準に3kgぶん加算される", consts.Milligram(3 * consts.MilligramPerKg), consts.PushCostBase + 3*consts.PushCostPerKg},
		{"総重量10kgで基準に10kgぶん加算される", consts.Milligram(10 * consts.MilligramPerKg), consts.PushCostBase + 10*consts.PushCostPerKg},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, query.PushCost(tt.total))
		})
	}
}

func TestCubeWeight_内部ステージ束縛の重量を合算し他ステージを除く(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	interior := gc.NewDungeonStage("キューブ内部", 1)
	other := gc.NewDungeonStage("別の内部", 1)

	addWeightEntity(t, world, consts.Milligram(2*consts.MilligramPerKg), interior, false)
	addWeightEntity(t, world, consts.Milligram(3*consts.MilligramPerKg), interior, false)
	addWeightEntity(t, world, consts.Milligram(5*consts.MilligramPerKg), other, false)

	assert.Equal(t, consts.Milligram(5*consts.MilligramPerKg), query.CubeWeight(world, interior))
}

func TestCubeWeight_退避中の内部エンティティも集計する(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	interior := gc.NewDungeonStage("キューブ内部", 1)

	// 外にいる間、内部は Suspended になる。それでも総重量は保持したい
	addWeightEntity(t, world, consts.Milligram(4*consts.MilligramPerKg), interior, true)

	assert.Equal(t, consts.Milligram(4*consts.MilligramPerKg), query.CubeWeight(world, interior))
}

func TestCubeWeight_床に無い物は数えない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	interior := gc.NewDungeonStage("キューブ内部", 1)

	// 床にある物は数える
	addWeightEntity(t, world, consts.Milligram(2*consts.MilligramPerKg), interior, false)

	// 内部で拾って背包へ移した物。LocationOnField は外れるが StageBound は残る。総重量から抜ける
	carried := world.ECS.NewEntity()
	world.Components.Weight.Add(carried, &gc.Weight{Milligram: consts.Milligram(9 * consts.MilligramPerKg)})
	world.Components.StageBound.Add(carried, &gc.StageBound{Key: interior})

	assert.Equal(t, consts.Milligram(2*consts.MilligramPerKg), query.CubeWeight(world, interior), "床にある物だけを数え、持ち去った物は除く")
}

func TestPartyPushPower_PlayerとSquadMemberのAPを合算し他を除く(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	player := world.ECS.NewEntity()
	world.Components.Player.Add(player, &gc.Player{})
	world.Components.TurnBased.Add(player, &gc.TurnBased{AP: gc.IntPool{Max: 40, Current: 40}})

	member := world.ECS.NewEntity()
	world.Components.SquadMember.Add(member, &gc.SquadMember{})
	world.Components.TurnBased.Add(member, &gc.TurnBased{AP: gc.IntPool{Max: 30, Current: 30}})

	// 敵はパーティAPに数えない
	enemy := world.ECS.NewEntity()
	world.Components.SoloAI.Add(enemy, &gc.SoloAI{})
	world.Components.TurnBased.Add(enemy, &gc.TurnBased{AP: gc.IntPool{Max: 100, Current: 100}})

	assert.Equal(t, 70, query.PartyPushPower(world))
}
