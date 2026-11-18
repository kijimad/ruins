package lifecycle

import (
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/raw"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// SpawnRemains は跡の行 id から跡を組み、coord へ置く。プレイ中に置くので、置いた直後に今のステージへ束縛し、
// 今のステージのクエリにすぐ乗るようにする
func SpawnRemains(world w.World, id string, coord consts.Coord[consts.Tile]) (ecs.Entity, error) {
	entitySpec, err := raw.NewRemainsSpec(world.Resources.RawMaster, id)
	if err != nil {
		return gc.InvalidEntity, err
	}
	entitySpec.GridElement = &gc.GridElement{Coord: coord}
	if d := query.GetDungeon(world); d != nil {
		entitySpec.StageBound = &gc.StageBound{Key: d.CurrentStage}
	}
	return world.Components.AddEntity(world.ECS, &entitySpec), nil
}
