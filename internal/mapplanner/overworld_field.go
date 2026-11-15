package mapplanner

import (
	"errors"

	"github.com/kijimaD/ruins/internal/consts"
)

// ErrGroundNotSet は下地を使う計画を下地を渡さずに組んだときのエラー
var ErrGroundNotSet = errors.New("ground tile is not set")

// OverworldFieldPlanner はオーバーワールドの「開けた地形」チャンクの初期プランナー。
//
// 部屋を掘るダンジョンのプランナーと逆に、全面通行可能を既定にする。荒れ地チャンクは地形の壁を
// 一切置かない。壁が無いのでチャンクを東西に継いでも境界が詰まらず、東西通行は自明に保たれる。
type OverworldFieldPlanner struct{}

// PlanInitial は初期化を行う。開けた地形は部屋を持たないため何もしない。
func (OverworldFieldPlanner) PlanInitial(_ *MetaPlan) error { return nil }

// NewOverworldFieldPlanner は開けた地形のチェーンを作る。全面を下地で埋めるだけで障壁は置かない。
func NewOverworldFieldPlanner(width, height consts.Tile, seed uint64) (*PlannerChain, error) {
	chain := NewPlannerChain(width, height, seed)
	chain.PlanData.FixedPortals = true
	chain.StartWith(OverworldFieldPlanner{})
	chain.With(FillGround{})
	return chain, nil
}

// FillGround は全面を計画の下地 MetaPlan.Ground で埋める
type FillGround struct{}

// PlanMeta は全面を下地で埋める。下地が空ならエラーにする
func (FillGround) PlanMeta(planData *MetaPlan) error {
	if planData.Ground == "" {
		return ErrGroundNotSet
	}
	ground := planData.GetTile(planData.Ground)
	for i := range planData.Tiles {
		planData.Tiles[i] = ground
	}
	return nil
}
