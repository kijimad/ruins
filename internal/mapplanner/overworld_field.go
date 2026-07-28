package mapplanner

import "github.com/kijimaD/ruins/internal/consts"

// OverworldFieldPlanner はオーバーワールドの「開けた地形」チャンクの初期プランナー。
//
// 部屋を掘るダンジョンのプランナーと逆に、全面通行可能を既定にする。荒れ地チャンクは地形の壁を
// 一切置かない。壁が無いのでチャンクを東西に継いでも境界が詰まらず、東西通行は自明に保たれる。
type OverworldFieldPlanner struct{}

// PlanInitial は初期化を行う。開けた地形は部屋を持たないため何もしない。
func (OverworldFieldPlanner) PlanInitial(_ *MetaPlan) error { return nil }

// NewOverworldFieldPlanner は開けた地形のチェーンを作る。全面を dirt で埋めるだけで障壁は置かない。
func NewOverworldFieldPlanner(width, height consts.Tile, seed uint64) (*PlannerChain, error) {
	chain := NewPlannerChain(width, height, seed)
	chain.StartWith(OverworldFieldPlanner{})
	chain.With(NewFillAll(consts.TileNameDirt))
	return chain, nil
}
