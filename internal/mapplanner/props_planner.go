// Package mapplanner のProps配置プランナー - 責務分離によりmapspawnerから移動
package mapplanner

import (
	w "github.com/kijimaD/ruins/internal/world"
)

// PropsSpec はProps配置仕様を表す
type PropsSpec struct {
	X    int    // X座標
	Y    int    // Y座標
	Name string // Prop名
}

// PropsPlanner はProps配置を担当するプランナー
type PropsPlanner struct {
	world       w.World
	plannerType PlannerType
}

// NewPropsPlanner はPropsプランナーを作成する
func NewPropsPlanner(world w.World, plannerType PlannerType) *PropsPlanner {
	return &PropsPlanner{
		world:       world,
		plannerType: plannerType,
	}
}

// PlanMeta はProps配置情報をMetaPlanに追加する
// MetaMapPlanner インターフェースを満たす
func (p *PropsPlanner) PlanMeta(_ *MetaPlan) error {
	// テンプレートベースのマップでテンプレート内でProps配置が完結するので使ってない
	return nil
}
