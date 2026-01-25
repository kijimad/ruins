package mapplanner

import (
	"fmt"
	"log"
	"strings"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/maptemplate"
	"github.com/kijimaD/ruins/internal/raw"
)

// BridgeFacilityWrapper は既存マップの上下に橋facilityを追加するプランナー
// 既存マップを拡張し、上部と下部に橋テンプレートを配置する
type BridgeFacilityWrapper struct {
	Loader *maptemplate.TemplateLoader // テンプレートローダー
}

// PlanMeta は既存マップを拡張して上部と下部に橋facilityを配置する
func (bw BridgeFacilityWrapper) PlanMeta(metaPlan *MetaPlan) error {
	oldWidth := int(metaPlan.Level.TileWidth)
	oldHeight := int(metaPlan.Level.TileHeight)

	// マップ幅に応じて適切なテンプレートを選択
	var topTemplateName, bottomTemplateName string
	if oldWidth == 20 {
		topTemplateName = "20x15_town_bridge_top"
		bottomTemplateName = "20x20_town_bridge_bottom"
	} else {
		// デフォルトは幅50用
		topTemplateName = "50x15_dungeon_bridge_top"
		bottomTemplateName = "50x20_dungeon_bridge_bottom"
	}

	// テンプレートを読み込む
	topTemplate, palette, err := bw.Loader.LoadTemplateByName(topTemplateName, 0)
	if err != nil {
		return err
	}

	bottomTemplate, _, err := bw.Loader.LoadTemplateByName(bottomTemplateName, 0)
	if err != nil {
		return err
	}

	// 上部と下部のテンプレートサイズを取得
	topHeight := topTemplate.Size.H
	bottomHeight := bottomTemplate.Size.H

	// 新しいマップサイズ: 上部facility + 既存マップ + 下部facility
	newHeight := topHeight + oldHeight + bottomHeight
	newWidth := oldWidth // 幅は変更しない（テンプレートは幅可変を想定）

	// 新しいタイル配列を作成
	newTiles := make([]raw.TileRaw, newWidth*newHeight)

	// 上部: 橋facilityテンプレートを配置
	bw.placeTemplate(newTiles, topTemplate, palette, newWidth, 0, metaPlan)

	// 中央部分: 既存マップをコピー
	for y := 0; y < oldHeight; y++ {
		for x := 0; x < oldWidth; x++ {
			oldIdx := y*oldWidth + x
			newIdx := (y+topHeight)*newWidth + x
			newTiles[newIdx] = metaPlan.Tiles[oldIdx]
		}
	}

	// 下部: 橋facilityテンプレートを配置
	bottomStartY := topHeight + oldHeight
	bw.placeTemplate(newTiles, bottomTemplate, palette, newWidth, bottomStartY, metaPlan)

	// MetaPlanを更新
	metaPlan.Tiles = newTiles
	metaPlan.Level.TileWidth = gc.Tile(newWidth)
	metaPlan.Level.TileHeight = gc.Tile(newHeight)

	// マップの文字列表現をログ出力
	var mapLines []string
	for y := 0; y < newHeight; y++ {
		line := ""
		for x := 0; x < newWidth; x++ {
			idx := y*newWidth + x
			tile := newTiles[idx]
			switch tile.Name {
			case "bridge_a":
				line += "A"
			case "bridge_b":
				line += "B"
			case "bridge_c":
				line += "C"
			case "bridge_d":
				line += "D"
			case "floor":
				line += "."
			case "wall":
				line += "#"
			case "void":
				line += "-"
			default:
				line += "?"
			}
		}
		mapLines = append(mapLines, fmt.Sprintf("%3d: %s", y, line))
	}
	log.Printf("BridgeFacilityWrapper: Final map (width=%d, height=%d):\n%s", newWidth, newHeight, strings.Join(mapLines, "\n"))

	// 既存のエンティティ座標をシフト
	bw.shiftExistingEntities(metaPlan, topHeight)

	// テンプレートから橋とプレイヤー位置を配置
	bw.placeBridgesFromTemplate(metaPlan, topTemplate, 0)
	bw.placeBridgesFromTemplate(metaPlan, bottomTemplate, bottomStartY)
	bw.setPlayerStartFromTemplate(metaPlan, bottomTemplate, bottomStartY)

	return nil
}

// placeTemplate はテンプレートをタイル配列に配置する
func (bw BridgeFacilityWrapper) placeTemplate(
	tiles []raw.TileRaw,
	template *maptemplate.ChunkTemplate,
	palette *maptemplate.Palette,
	mapWidth, offsetY int,
	metaPlan *MetaPlan,
) {
	lines := template.GetMapLines()
	for y, line := range lines {
		for x, char := range line {
			if x >= mapWidth {
				continue // マップ幅を超える部分は無視
			}
			idx := (y+offsetY)*mapWidth + x
			charStr := string(char)

			// パレットから地形を取得
			if terrainName, ok := palette.GetTerrain(charStr); ok {
				tile := metaPlan.GetTile(terrainName)
				tiles[idx] = tile
			}
		}
	}
}

// shiftExistingEntities は既存エンティティの座標をシフトする
func (bw BridgeFacilityWrapper) shiftExistingEntities(metaPlan *MetaPlan, offsetY int) {
	for i := range metaPlan.Rooms {
		metaPlan.Rooms[i].Y1 += gc.Tile(offsetY)
		metaPlan.Rooms[i].Y2 += gc.Tile(offsetY)
	}

	for i := range metaPlan.WarpPortals {
		metaPlan.WarpPortals[i].Y += offsetY
	}

	for i := range metaPlan.NPCs {
		metaPlan.NPCs[i].Y += offsetY
	}

	for i := range metaPlan.Items {
		metaPlan.Items[i].Y += offsetY
	}

	for i := range metaPlan.Props {
		metaPlan.Props[i].Y += offsetY
	}
}

// placeBridgesFromTemplate はテンプレートから橋を配置する
func (bw BridgeFacilityWrapper) placeBridgesFromTemplate(
	metaPlan *MetaPlan,
	template *maptemplate.ChunkTemplate,
	offsetY int,
) {
	for _, bridge := range template.Bridges {
		metaPlan.Bridges = append(metaPlan.Bridges, BridgeSpec{
			X:        bridge.X,
			Y:        bridge.Y + offsetY,
			BridgeID: bridge.BridgeID,
		})
	}
}

// setPlayerStartFromTemplate はテンプレートからプレイヤー開始位置を設定する
func (bw BridgeFacilityWrapper) setPlayerStartFromTemplate(
	metaPlan *MetaPlan,
	template *maptemplate.ChunkTemplate,
	offsetY int,
) {
	lines := template.GetMapLines()
	for y, line := range lines {
		for x, char := range line {
			if char == '@' {
				metaPlan.PlayerStartPosition = &struct{ X, Y int }{
					X: x,
					Y: y + offsetY,
				}
				return
			}
		}
	}
}

// NewBridgeFacilityWrapper は橋facilityラッパーを作成する
func NewBridgeFacilityWrapper() (*BridgeFacilityWrapper, error) {
	loader := maptemplate.NewTemplateLoader()

	// パレットを登録
	if err := loader.RegisterAllPalettes([]string{"assets/levels/palettes"}); err != nil {
		return nil, err
	}

	// チャンクテンプレートを登録
	if err := loader.RegisterAllChunks([]string{"assets/levels/layouts"}); err != nil {
		return nil, err
	}

	return &BridgeFacilityWrapper{
		Loader: loader,
	}, nil
}
