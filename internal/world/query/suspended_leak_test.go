package query_test

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	leakStageA = gc.NewNamedDungeonStage("テスト遺跡", 1)
	leakStageB = gc.NewNamedDungeonStage("テスト遺跡", 2)
)

// addLeakChar は指定座標・ステージのキャラクターを作る。SoloAI で索引上キャラクター扱いになる
func addLeakChar(t *testing.T, world w.World, coord consts.Coord[consts.Tile], stage gc.StageKey, suspended bool) ecs.Entity {
	t.Helper()
	e := world.ECS.NewEntity()
	world.Components.GridElement.Add(e, &gc.GridElement{Coord: coord})
	world.Components.SoloAI.Add(e, &gc.SoloAI{})
	world.Components.StageBound.Add(e, &gc.StageBound{Key: stage})
	if suspended {
		world.Components.Suspended.Add(e, &gc.Suspended{})
	}
	return e
}

// TestSuspended_同座標でも現ステージだけが見える は共存方式の要である座標衝突の防止を検証する。
// 2階と3階が同じタイル(5,5)を持つ状況を作り、座標検索が現ステージのエンティティだけを返すことを確かめる。
func TestSuspended_同座標でも現ステージだけが見える(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	pos := consts.Coord[consts.Tile]{X: 5, Y: 5}

	active := addLeakChar(t, world, pos, leakStageA, false)
	suspended := addLeakChar(t, world, pos, leakStageB, true)

	// GetEntitiesAt: 同座標でも現ステージのエンティティだけを返す
	at := query.GetEntitiesAt(world, pos.X, pos.Y)
	assert.Contains(t, at, active, "現ステージのエンティティは返る")
	assert.NotContains(t, at, suspended, "退避中の同座標エンティティは漏れない")

	// SpatialIndex: 同座標の占有は現ステージのエンティティ。退避中に上書きされない
	query.InvalidateSpatialIndex(world)
	si := query.GetSpatialIndex(world)
	require.NotNil(t, si)
	got, ok := si.CharacterAt(pos)
	assert.True(t, ok, "同座標にキャラクターがいる")
	assert.Equal(t, active, got, "索引の占有は現ステージのエンティティ")

	// FindNearestEntity: 退避中エンティティは最近傍にならない
	from := &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 6, Y: 5}}
	nearest, _, _ := query.FindNearestEntity(world, ecs.Entity{}, from, func(e ecs.Entity) bool {
		return world.Components.SoloAI.Has(e)
	})
	require.NotNil(t, nearest, "最近傍が見つかる")
	assert.Equal(t, active, *nearest, "最近傍は現ステージのエンティティ")
}

// TestSuspended_APは退避中に回復しない は退避中ステージの敵がターン処理から凍結されることを検証する。
func TestSuspended_APは退避中に回復しない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	active := addLeakChar(t, world, consts.Coord[consts.Tile]{X: 5, Y: 5}, leakStageA, false)
	suspended := addLeakChar(t, world, consts.Coord[consts.Tile]{X: 6, Y: 6}, leakStageB, true)

	// 両者に空のTurnBasedを付け、APを0にしておく
	world.Components.TurnBased.Add(active, &gc.TurnBased{})
	world.Components.TurnBased.Add(suspended, &gc.TurnBased{})
	world.Components.Abilities.Add(active, &gc.Abilities{})
	world.Components.Abilities.Add(suspended, &gc.Abilities{})

	require.NoError(t, query.RestoreAllActionPoints(world))

	assert.Positive(t, world.Components.TurnBased.Get(active).AP.Max, "現ステージの敵はAPが計算される")
	assert.Zero(t, world.Components.TurnBased.Get(suspended).AP.Max, "退避中の敵はAP回復から凍結される")
}
