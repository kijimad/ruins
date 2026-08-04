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
// 中立faction、商人や酒場の主人など、の member をNPCとして、収納箱を1つ、狭い部屋へ詰めて置き、
// プレイヤーのすぐ近くで街の会話・売買・収納をまとめてテストできるようにする。
// 収納の中身は spawn 側が prop 定義から自動で決めるため、ここは名前と座標を積むだけでよい
type DebugPopulatePlanner struct{}

// PlanMeta は部屋の内側を上から順に走査し、街用NPCと収納箱を置けるタイルへ詰めて配置する
func (DebugPopulatePlanner) PlanMeta(planData *MetaPlan) error {
	if planData.RawMaster == nil {
		return nil
	}

	// 配置物を順に並べる。街用NPC、中立faction、を先に、収納箱を最後に置く。敵やプレイヤーは置かない
	var toPlace []func(consts.Coord[consts.Tile])
	if planData.RawMaster.Members != nil {
		members := *planData.RawMaster.Members
		for i := range members {
			m := &members[i]
			if m.FactionType == nil || string(*m.FactionType) != gc.FactionNeutralName {
				continue
			}
			name := m.Name
			toPlace = append(toPlace, func(c consts.Coord[consts.Tile]) {
				planData.NPCs = append(planData.NPCs, NPCSpec{Coord: c, Name: name})
			})
		}
	}
	toPlace = append(toPlace, func(c consts.Coord[consts.Tile]) {
		planData.Props = append(planData.Props, PropsSpec{Coord: c, Name: debugStorageBoxName})
	})

	// 部屋の内側を詰めて配置する。狭い部屋なのでプレイヤーのすぐ近くにまとまる
	idx := 0
	for _, room := range planData.Rooms {
		for y := room.Min.Y + 1; y < room.Max.Y && idx < len(toPlace); y++ {
			for x := room.Min.X + 1; x < room.Max.X && idx < len(toPlace); x++ {
				if !planData.IsSpawnableTile(w.World{}, x, y) {
					continue
				}
				toPlace[idx](consts.Coord[consts.Tile]{X: x, Y: y})
				idx++
			}
		}
	}
	if idx < len(toPlace) {
		log.Printf("DebugPopulatePlanner: 配置枠が足りず %d 件を配置できませんでした", len(toPlace)-idx)
	}
	return nil
}
