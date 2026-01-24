package mapplanner

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/maptemplate"
	"github.com/kijimaD/ruins/internal/raw"
)

// TemplatePlanner はテンプレートベースのマップ生成プランナー
type TemplatePlanner struct {
	Template *maptemplate.ChunkTemplate
	Palette  *maptemplate.Palette
}

// NewTemplatePlanner はTemplatePlannerを生成する
func NewTemplatePlanner(template *maptemplate.ChunkTemplate, palette *maptemplate.Palette) *TemplatePlanner {
	return &TemplatePlanner{
		Template: template,
		Palette:  palette,
	}
}

// PlanInitial はテンプレートから初期マップを生成する
func (p *TemplatePlanner) PlanInitial(metaPlan *MetaPlan) error {
	// マップサイズを設定
	width := p.Template.Size[0]
	height := p.Template.Size[1]
	metaPlan.Level.TileWidth = gc.Tile(width)
	metaPlan.Level.TileHeight = gc.Tile(height)

	// タイル配列を初期化
	totalTiles := width * height
	metaPlan.Tiles = make([]raw.TileRaw, totalTiles)

	// テンプレートマップを走査して地形を配置
	lines := p.Template.GetMapLines()
	for y, line := range lines {
		for x, char := range line {
			idx := y*width + x
			charStr := string(char)

			// パレットから地形を取得
			terrainName, ok := p.Palette.GetTerrain(charStr)
			if !ok {
				return fmt.Errorf("パレットに文字 '%s' の地形定義がありません (位置: %d, %d)", charStr, x, y)
			}
			tile := metaPlan.GetTile(terrainName)
			metaPlan.Tiles[idx] = tile
		}
	}

	return nil
}

// PlanMeta はテンプレートからメタ情報を配置する
func (p *TemplatePlanner) PlanMeta(metaPlan *MetaPlan) error {
	lines := p.Template.GetMapLines()

	// テンプレートマップを走査して家具を配置
	for y, line := range lines {
		for x, char := range line {
			charStr := string(char)

			if furnitureName, ok := p.Palette.GetFurniture(charStr); ok {
				// 配置予定リストに追加
				metaPlan.Props = append(metaPlan.Props, PropsSpec{
					X:       x,
					Y:       y,
					PropKey: furnitureName,
				})
			}
		}
	}

	return nil
}

// NewTemplatePlannerChain はテンプレートベースのPlannerChainを作成する
func NewTemplatePlannerChain(template *maptemplate.ChunkTemplate, palette *maptemplate.Palette, seed uint64) (*PlannerChain, error) {
	width := gc.Tile(template.Size[0])
	height := gc.Tile(template.Size[1])

	chain := NewPlannerChain(width, height, seed)
	planner := NewTemplatePlanner(template, palette)
	chain.StartWith(planner)
	chain.With(planner) // PlanMeta用

	return chain, nil
}
