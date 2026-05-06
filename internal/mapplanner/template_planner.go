package mapplanner

import (
	"fmt"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/maptemplate"
	"github.com/kijimaD/ruins/internal/raw"
)

// TemplatePlanner はテンプレートベースのマップ生成プランナー。
// 解決済みセル配列を使って地形・Props・NPCを配置する
type TemplatePlanner struct {
	Template    *maptemplate.ChunkTemplate
	ResolvedMap [][]maptemplate.MapCell
}

// NewTemplatePlanner はTemplatePlannerを生成する
func NewTemplatePlanner(template *maptemplate.ChunkTemplate, resolvedMap [][]maptemplate.MapCell) *TemplatePlanner {
	return &TemplatePlanner{
		Template:    template,
		ResolvedMap: resolvedMap,
	}
}

// PlanInitial はテンプレートから初期マップを生成する
func (p *TemplatePlanner) PlanInitial(metaPlan *MetaPlan) error {
	// マップサイズを設定
	width := p.Template.Size.W
	height := p.Template.Size.H
	metaPlan.Level.TileWidth = consts.Tile(width)
	metaPlan.Level.TileHeight = consts.Tile(height)

	// タイル配列を初期化
	totalTiles := width * height
	metaPlan.Tiles = make([]raw.TileRaw, totalTiles)

	// セル配列を走査して地形を配置する
	for y, row := range p.ResolvedMap {
		for x, cell := range row {
			idx := y*width + x
			if cell.Terrain == "" {
				return fmt.Errorf("セルの地形が未定義です (位置: %d, %d)", x, y)
			}
			tile := metaPlan.GetTile(cell.Terrain)
			metaPlan.Tiles[idx] = tile
		}
	}

	return nil
}

// PlanMeta はテンプレートからメタ情報を配置する
func (p *TemplatePlanner) PlanMeta(metaPlan *MetaPlan) error {
	// セル配列を走査してProps・NPCを配置する
	for y, row := range p.ResolvedMap {
		for x, cell := range row {
			if cell.Prop != "" {
				switch cell.Prop {
				case "warp_next":
					metaPlan.NextPortals = append(metaPlan.NextPortals, consts.Coord[int]{X: x, Y: y})
				case "warp_escape":
					metaPlan.EscapePortals = append(metaPlan.EscapePortals, consts.Coord[int]{X: x, Y: y})
				default:
					metaPlan.Props = append(metaPlan.Props, PropsSpec{
						Coord: consts.Coord[int]{X: x, Y: y},
						Name:  cell.Prop,
					})
				}
			}

			if cell.NPC != "" {
				metaPlan.NPCs = append(metaPlan.NPCs, NPCSpec{
					Coord: consts.Coord[int]{X: x, Y: y},
					Name:  cell.NPC,
				})
			}
		}
	}

	// スポーン地点をコピー
	metaPlan.SpawnPoints = append(metaPlan.SpawnPoints, p.Template.SpawnPoints...)

	return nil
}

// NewTemplatePlannerChain はテンプレートベースのPlannerChainを作成する
func NewTemplatePlannerChain(template *maptemplate.ChunkTemplate, resolvedMap [][]maptemplate.MapCell, seed uint64) (*PlannerChain, error) {
	width := consts.Tile(template.Size.W)
	height := consts.Tile(template.Size.H)

	chain := NewPlannerChain(width, height, seed)
	planner := NewTemplatePlanner(template, resolvedMap)
	chain.StartWith(planner)
	chain.With(planner) // PlanMeta用
	chain.With(EnvironmentPlanner{})

	return chain, nil
}
