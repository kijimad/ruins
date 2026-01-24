// Package mapplanner のProps配置プランナー - 責務分離によりmapspawnerから移動
package mapplanner

import (
	gc "github.com/kijimaD/ruins/internal/components"
	w "github.com/kijimaD/ruins/internal/world"
)

// PropsSpec はProps配置仕様を表す
type PropsSpec struct {
	X       int    // X座標
	Y       int    // Y座標
	PropKey string // Prop名
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
func (p *PropsPlanner) PlanMeta(planData *MetaPlan) error {
	// 町タイプの場合は固定Props配置を追加
	if p.plannerType.Name == PlannerTypeTown.Name {
		p.addTownProps(planData)
	}

	// ダンジョンタイプの場合は固定Props配置を追加（必要に応じて実装）
	// TODO: 必要に応じて他のタイプも実装
	return nil
}

// addTownProps は町用の固定Props配置をMetaPlanに追加する
func (p *PropsPlanner) addTownProps(planData *MetaPlan) {
	centerX := int(planData.Level.TileWidth) / 2
	centerY := int(planData.Level.TileHeight) / 2

	// 図書館のProps配置
	libraryProps := []struct {
		propKey string
		offsetX int
		offsetY int
	}{
		{"bookshelf", -8, -19}, // 北壁沿い
		{"bookshelf", -6, -19}, // 北壁沿い
		{"table", -7, -17},     // 閲覧机
		{"chair", -7, -16},     // 閲覧用椅子
		{"table", -4, -15},     // 学習机
	}

	for _, prop := range libraryProps {
		x := centerX + prop.offsetX
		y := centerY + prop.offsetY
		if p.isValidPropPosition(planData, gc.Tile(x), gc.Tile(y)) {
			planData.Props = append(planData.Props, PropsSpec{
				X:       x,
				Y:       y,
				PropKey: prop.propKey,
			})
		}
	}

	// 学校のProps配置
	schoolProps := []struct {
		propKey string
		offsetX int
		offsetY int
	}{
		{"bookshelf", 6, -19},  // 北壁
		{"bookshelf", 8, -19},  // 北壁
		{"bookshelf", 10, -19}, // 北壁
		{"bookshelf", 5, -16},  // 西壁
		{"bookshelf", 11, -16}, // 東壁
		{"table", 8, -15},      // 教卓
		{"chair", 8, -14},      // 教師用椅子
	}

	for _, prop := range schoolProps {
		x := centerX + prop.offsetX
		y := centerY + prop.offsetY
		if p.isValidPropPosition(planData, gc.Tile(x), gc.Tile(y)) {
			planData.Props = append(planData.Props, PropsSpec{
				X:       x,
				Y:       y,
				PropKey: prop.propKey,
			})
		}
	}

	// 住民の家1のProps配置
	house1Props := []struct {
		propKey string
		offsetX int
		offsetY int
	}{
		{"bed", 13, -7},   // 寝室
		{"table", 15, -5}, // 食事台
		{"chair", 15, -4}, // 食事用椅子
		{"chair", 16, -5}, // 食事用椅子
	}

	for _, prop := range house1Props {
		x := centerX + prop.offsetX
		y := centerY + prop.offsetY
		if p.isValidPropPosition(planData, gc.Tile(x), gc.Tile(y)) {
			planData.Props = append(planData.Props, PropsSpec{
				X:       x,
				Y:       y,
				PropKey: prop.propKey,
			})
		}
	}

	// 住民の家2のProps配置
	house2Props := []struct {
		propKey string
		offsetX int
		offsetY int
	}{
		{"bed", 14, 2},   // 寝室
		{"table", 16, 4}, // 食事台
		{"chair", 16, 5}, // 食事用椅子
	}

	for _, prop := range house2Props {
		x := centerX + prop.offsetX
		y := centerY + prop.offsetY
		if p.isValidPropPosition(planData, gc.Tile(x), gc.Tile(y)) {
			planData.Props = append(planData.Props, PropsSpec{
				X:       x,
				Y:       y,
				PropKey: prop.propKey,
			})
		}
	}

	// 公民館の座席配置
	hallProps := []struct {
		propKey string
		offsetX int
		offsetY int
	}{
		{"chair", -6, 12}, // 集会用座席
		{"chair", -4, 12}, // 集会用座席
		{"chair", 4, 12},  // 集会用座席
		{"chair", 6, 12},  // 集会用座席
	}

	for _, prop := range hallProps {
		x := centerX + prop.offsetX
		y := centerY + prop.offsetY
		if p.isValidPropPosition(planData, gc.Tile(x), gc.Tile(y)) {
			planData.Props = append(planData.Props, PropsSpec{
				X:       x,
				Y:       y,
				PropKey: prop.propKey,
			})
		}
	}

	// 事務所のProps配置
	officeProps := []struct {
		propKey string
		offsetX int
		offsetY int
	}{
		{"bed", 12, 13},       // 休憩用ベッド
		{"table", 14, 15},     // 事務机
		{"chair", 14, 16},     // 事務用椅子
		{"bookshelf", 18, 14}, // 書類棚
	}

	for _, prop := range officeProps {
		x := centerX + prop.offsetX
		y := centerY + prop.offsetY
		if p.isValidPropPosition(planData, gc.Tile(x), gc.Tile(y)) {
			planData.Props = append(planData.Props, PropsSpec{
				X:       x,
				Y:       y,
				PropKey: prop.propKey,
			})
		}
	}

	// 市場の露店（簡略化して一部のみ配置）
	marketProps := []struct {
		propKey string
		offsetX int
		offsetY int
	}{
		{"table", -12, 5}, // 露店1
		{"table", -9, 5},  // 露店2
		{"table", -6, 5},  // 露店3
	}

	for _, prop := range marketProps {
		x := centerX + prop.offsetX
		y := centerY + prop.offsetY
		if p.isValidPropPosition(planData, gc.Tile(x), gc.Tile(y)) {
			planData.Props = append(planData.Props, PropsSpec{
				X:       x,
				Y:       y,
				PropKey: prop.propKey,
			})
		}
	}
}

// isValidPropPosition はProp配置に適した位置かチェックする
func (p *PropsPlanner) isValidPropPosition(planData *MetaPlan, x, y gc.Tile) bool {
	// 範囲チェック
	if x < 0 || x >= planData.Level.TileWidth || y < 0 || y >= planData.Level.TileHeight {
		return false
	}

	tileIdx := planData.Level.XYTileIndex(x, y)
	if int(tileIdx) >= len(planData.Tiles) {
		return false
	}

	tile := planData.Tiles[tileIdx]
	// 床タイルにのみ配置可能
	return tile.Walkable
}
