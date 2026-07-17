// Package worldstream は無限シームレスワールドの「アクティブ帯」を east 方向へ
// ストリーミングするための基盤操作を提供する。
//
// 帯シフト（プレイヤーが中央チャンクを東へ出たときに、東端チャンクを生成し
// 西端チャンクを破棄して座標をリベースする操作）は、次の2つの原子操作の合成で表現する。
// 詳細設計は docs/design/20260717_60.md を参照。
package worldstream

import (
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/mlange-42/ark/ecs"
)

// TranslateAllEntities は GridElement を持つ全エンティティの位置を (dx,dy) タイル平行移動する。
//
// 帯シフト時のリベース（プレイヤーを帯の中央へ戻し、帯ローカル座標を有界に保つ）の原子操作。
// バックパック/装備アイテムは GridElement を持たないため対象外（クエリで自然に除外される）。
// コンポーネント値の書き換えのみでアーキタイプは変えないため、クエリ反復中の更新で安全。
func TranslateAllEntities(world w.World, dx, dy consts.Tile) {
	query := ecs.NewFilter1[gc.GridElement](world.ECS).Query()
	for query.Next() {
		grid := world.Components.GridElement.Get(query.Entity())
		grid.X += dx
		grid.Y += dy
	}
}

// RemoveEntitiesInXRange は GridElement.X が [loX, hiX) にあるエンティティを削除する。
//
// 帯シフト時の西端チャンク破棄（前線に呑まれる領域の消去）の原子操作。keep が true を返す
// エンティティ（プレイヤー・隊員など残すべきもの）は削除しない。削除した数を返す。
// 反復中の削除を避けるため、対象を収集してから削除する。
func RemoveEntitiesInXRange(world w.World, loX, hiX consts.Tile, keep func(ecs.Entity) bool) int {
	var toRemove []ecs.Entity
	query := ecs.NewFilter1[gc.GridElement](world.ECS).Query()
	for query.Next() {
		entity := query.Entity()
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
