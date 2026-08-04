// Package mapplanner のデバッグ用の全要素配置プランナー
package mapplanner

import (
	"log"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	w "github.com/kijimaD/ruins/internal/world"
)

// debugStorageBoxName はデバッグステージに置く収納箱の prop 名
const debugStorageBoxName = "木箱"

// DebugPopulatePlanner はデバッグステージへ街用NPCと収納箱を配置するプランナー。
// 中立faction、商人や酒場の主人など、の member をNPCとして、収納箱を1つ、部屋のグリッドに並べ、
// 街の会話・売買・収納を一度にテストできるようにする。
// 収納の中身は spawn 側が prop 定義から自動で決めるため、ここは名前と座標を積むだけでよい
type DebugPopulatePlanner struct{}

// PlanMeta は部屋内のグリッド座標へ街用NPCと収納箱を配置する
func (DebugPopulatePlanner) PlanMeta(planData *MetaPlan) error {
	if planData.RawMaster == nil {
		return nil
	}

	coords := debugGridCoords(planData)
	next := 0
	// place は次の配置可能な座標を1つ消費して add を呼ぶ。空きが無ければ false を返す
	place := func(add func(consts.Coord[consts.Tile])) bool {
		for next < len(coords) {
			c := coords[next]
			next++
			if planData.IsSpawnableTile(w.World{}, c.X, c.Y) {
				add(c)
				return true
			}
		}
		return false
	}

	dropped := 0

	// 街用NPC、中立faction、だけを配置する。敵やプレイヤーは置かない
	if planData.RawMaster.Members != nil {
		members := *planData.RawMaster.Members
		for i := range members {
			m := &members[i]
			if m.FactionType == nil || string(*m.FactionType) != gc.FactionNeutralName {
				continue
			}
			name := m.Name
			if !place(func(c consts.Coord[consts.Tile]) {
				planData.NPCs = append(planData.NPCs, NPCSpec{Coord: c, Name: name})
			}) {
				dropped++
			}
		}
	}

	// 収納箱を1つ置く。中身は spawn 時に loot が入る
	if !place(func(c consts.Coord[consts.Tile]) {
		planData.Props = append(planData.Props, PropsSpec{Coord: c, Name: debugStorageBoxName})
	}) {
		dropped++
	}

	if dropped > 0 {
		log.Printf("DebugPopulatePlanner: 配置枠が足りず %d 件を配置できませんでした", dropped)
	}
	return nil
}

// debugGridCoords は部屋の内側を2タイル間隔で走査した配置候補座標を返す。
// 間隔を空けて、配置物どうしや通行が詰まらないようにする
func debugGridCoords(planData *MetaPlan) []consts.Coord[consts.Tile] {
	var coords []consts.Coord[consts.Tile]
	for _, room := range planData.Rooms {
		for y := room.Min.Y + 1; y < room.Max.Y; y += 2 {
			for x := room.Min.X + 1; x < room.Max.X; x += 2 {
				coords = append(coords, consts.Coord[consts.Tile]{X: x, Y: y})
			}
		}
	}
	return coords
}
