package worldstream

import (
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/mlange-42/ark/ecs"
)

// TranslateAllEntities は GridElement を持つ全エンティティの位置を dx,dy タイル平行移動する。
//
// 帯シフト時のリベースの原子操作。リベースはプレイヤーを帯の中央へ戻し、帯ローカル座標を有界に保つ。
// バックパック/装備アイテムは GridElement を持たないため対象外で、クエリで自然に除外される。
// コンポーネント値の書き換えのみでアーキタイプは変えないため、クエリ反復中の更新で安全。
func TranslateAllEntities(world w.World, dx, dy consts.Tile) {
	q := ecs.NewFilter1[gc.GridElement](world.ECS).Query()
	for q.Next() {
		grid := world.Components.GridElement.Get(q.Entity())
		grid.X += dx
		grid.Y += dy
	}
}

// RemoveEntitiesInXRange は GridElement.X が [loX, hiX) にあるエンティティを削除する。
//
// 帯シフト時の西端チャンク破棄の原子操作で、前線に呑まれる領域を消去する。keep が true を返す
// エンティティは削除しない。プレイヤー・隊員など残すべきものが該当する。削除した数を返す。
// 反復中の削除を避けるため、対象を収集してから削除する。
func RemoveEntitiesInXRange(world w.World, loX, hiX consts.Tile, keep func(ecs.Entity) bool) int {
	var toRemove []ecs.Entity
	q := ecs.NewFilter1[gc.GridElement](world.ECS).Query()
	for q.Next() {
		entity := q.Entity()
		grid := world.Components.GridElement.Get(entity)
		if grid.X < loX || grid.X >= hiX {
			continue
		}
		if keep != nil && keep(entity) {
			continue
		}
		toRemove = append(toRemove, entity)
	}
	for _, entity := range toRemove {
		if world.ECS.Alive(entity) {
			world.ECS.RemoveEntity(entity)
		}
	}
	return len(toRemove)
}

// KeepPlayerAndSquad は「プレイヤーと隊員は残す」keep 述語を返す。
// 西端破棄でリーダー・隊員を巻き込まないための定型。
func KeepPlayerAndSquad(world w.World) func(ecs.Entity) bool {
	return func(entity ecs.Entity) bool {
		return world.Components.Player.Has(entity) || world.Components.SquadMember.Has(entity)
	}
}
