package mapplanner

import (
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/maptemplate"
	"github.com/kijimaD/ruins/internal/raw"
)

// BridgeAreaExtender は既存マップを拡張し、上部と下部に橋エリアテンプレートを配置する
type BridgeAreaExtender struct {
	Loader *maptemplate.TemplateLoader // テンプレートローダー
}

// Extend は既存マップを拡張して上部と下部に橋エリアを配置する
func (e BridgeAreaExtender) Extend(metaPlan *MetaPlan) error {
	oldWidth := int(metaPlan.Level.TileWidth)
	oldHeight := int(metaPlan.Level.TileHeight)

	// マップ幅に応じて適切なテンプレートを選択
	var topTemplateName, bottomTemplateName string
	if oldWidth == 20 {
		topTemplateName = "20x28_town_bridge_top"
		bottomTemplateName = "20x28_town_bridge_bottom"
	} else {
		// デフォルトは幅50用
		topTemplateName = "50x28_dungeon_bridge_top"
		bottomTemplateName = "50x28_dungeon_bridge_bottom"
	}

	// テンプレートとパレットを読み込む
	topTemplate, palette, err := e.Loader.LoadTemplateByName(topTemplateName, 0)
	if err != nil {
		return err
	}

	bottomTemplate, _, err := e.Loader.LoadTemplateByName(bottomTemplateName, 0)
	if err != nil {
		return err
	}

	// 上部と下部のテンプレートサイズを取得
	topHeight := topTemplate.Size.H
	bottomHeight := bottomTemplate.Size.H

	// 新しいマップサイズ: 上部橋エリア + 既存マップ + 下部橋エリア
	newHeight := topHeight + oldHeight + bottomHeight
	newWidth := oldWidth // 幅は変更しない（テンプレートは幅可変を想定）

	// 既存のエンティティ座標をシフト（テンプレート配置前に実行）
	e.shiftExistingEntities(metaPlan, topHeight)

	// 新しいタイル配列を作成
	newTiles := make([]raw.TileRaw, newWidth*newHeight)

	// 上部: 橋エリアテンプレートを配置
	e.placeTemplate(newTiles, topTemplate, palette, newWidth, 0, metaPlan)

	// 中央部分: 既存マップをコピー
	for y := 0; y < oldHeight; y++ {
		for x := 0; x < oldWidth; x++ {
			oldIdx := y*oldWidth + x
			newIdx := (y+topHeight)*newWidth + x
			newTiles[newIdx] = metaPlan.Tiles[oldIdx]
		}
	}

	// 下部: 橋エリアテンプレートを配置
	bottomStartY := topHeight + oldHeight
	e.placeTemplate(newTiles, bottomTemplate, palette, newWidth, bottomStartY, metaPlan)

	// MetaPlanを更新
	metaPlan.Tiles = newTiles
	metaPlan.Level.TileWidth = gc.Tile(newWidth)
	metaPlan.Level.TileHeight = gc.Tile(newHeight)

	// テンプレートから橋情報を収集
	e.collectBridgesFromTemplate(metaPlan, topTemplate, 0)
	e.collectBridgesFromTemplate(metaPlan, bottomTemplate, bottomStartY)

	return nil
}

// placeTemplate はテンプレートをタイル配列に配置する
func (e BridgeAreaExtender) placeTemplate(
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

			// パレットからPropsを取得
			if propName, ok := palette.GetProp(charStr); ok {
				metaPlan.Props = append(metaPlan.Props, PropsSpec{
					X:    x,
					Y:    y + offsetY,
					Name: propName,
				})
			}
		}
	}
}

// shiftExistingEntities は既存エンティティの座標をシフトする
func (e BridgeAreaExtender) shiftExistingEntities(metaPlan *MetaPlan, offsetY int) {
	for i := range metaPlan.Rooms {
		metaPlan.Rooms[i].Y1 += gc.Tile(offsetY)
		metaPlan.Rooms[i].Y2 += gc.Tile(offsetY)
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

// collectBridgesFromTemplate はテンプレートから出口とスポーン地点情報を収集してMetaPlanに記録する
// 実際のエンティティ配置はmapspawner.Spawn()で行われる
func (e BridgeAreaExtender) collectBridgesFromTemplate(
	metaPlan *MetaPlan,
	template *maptemplate.ChunkTemplate,
	offsetY int,
) {
	// 出口情報を収集
	for _, exitPlacement := range template.ExitPlacements {
		metaPlan.Exits = append(metaPlan.Exits, maptemplate.ExitPlacement{
			X:      exitPlacement.X,
			Y:      exitPlacement.Y + offsetY,
			ExitID: exitPlacement.ExitID,
		})
	}

	// スポーン地点情報を収集
	for _, spawnPoint := range template.SpawnPoints {
		metaPlan.SpawnPoints = append(metaPlan.SpawnPoints, maptemplate.SpawnPoint{
			X: spawnPoint.X,
			Y: spawnPoint.Y + offsetY,
		})
	}

	// 橋ヒント情報を収集
	for _, hint := range template.BridgeHintPlacements {
		metaPlan.BridgeHints = append(metaPlan.BridgeHints, maptemplate.BridgeHintPlacement{
			ExitID: hint.ExitID,
			X:      hint.X,
			Y:      hint.Y + offsetY,
		})
	}
}

// NewBridgeAreaExtender は橋エリア拡張器を作成する
func NewBridgeAreaExtender() (*BridgeAreaExtender, error) {
	loader := maptemplate.NewTemplateLoader()

	// パレットを登録
	if err := loader.RegisterAllPalettes([]string{"levels/palettes"}); err != nil {
		return nil, err
	}

	// チャンクテンプレートを登録
	if err := loader.RegisterAllChunks([]string{"levels/chunks"}); err != nil {
		return nil, err
	}

	return &BridgeAreaExtender{
		Loader: loader,
	}, nil
}
