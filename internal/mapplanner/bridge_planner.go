// Package mapplanner の橋配置プランナー
package mapplanner

import (
	w "github.com/kijimaD/ruins/internal/world"
)

// BridgeSpec は橋配置仕様を表す
type BridgeSpec struct {
	X        int    // X座標
	Y        int    // Y座標
	BridgeID string // 橋ID ("A", "B", "C", "D")
}

// BridgePlanner は橋配置を担当するプランナー
type BridgePlanner struct {
	world       w.World
	plannerType PlannerType
	depth       int    // 現在の階層深度
	gameSeed    uint64 // ゲームシード
}

// NewBridgePlanner は橋プランナーを作成する
func NewBridgePlanner(world w.World, plannerType PlannerType, depth int, gameSeed uint64) *BridgePlanner {
	return &BridgePlanner{
		world:       world,
		plannerType: plannerType,
		depth:       depth,
		gameSeed:    gameSeed,
	}
}

// PlanMeta は橋配置情報をMetaPlanに追加する
// MetaMapPlanner インターフェースを満たす
func (bp *BridgePlanner) PlanMeta(_ *MetaPlan) error {
	// 現時点では何もしない
	// 橋はテンプレートから配置されるため、ここでは処理不要
	return nil
}
