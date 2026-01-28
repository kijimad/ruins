package mapplanner

import (
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/maptemplate"
	"github.com/kijimaD/ruins/internal/raw"
)

// BridgeFacilityWrapper は既存マップの上下に橋facilityを追加するプランナー
// 既存マップを拡張し、上部と下部に橋テンプレートを配置する
// TODO: 名前がわかりづらいのをどうにかする
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
		topTemplateName = "20x28_town_bridge_top"
		bottomTemplateName = "20x28_town_bridge_bottom"
	} else {
		// デフォルトは幅50用
		topTemplateName = "50x28_dungeon_bridge_top"
		bottomTemplateName = "50x28_dungeon_bridge_bottom"
	}

	// テンプレートとパレットを読み込む
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

	// 既存のエンティティ座標をシフト（テンプレート配置前に実行）
	bw.shiftExistingEntities(metaPlan, topHeight)

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

	// テンプレートから橋情報を収集
	bw.collectBridgesFromTemplate(metaPlan, topTemplate, 0)
	bw.collectBridgesFromTemplate(metaPlan, bottomTemplate, bottomStartY)

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
func (bw BridgeFacilityWrapper) shiftExistingEntities(metaPlan *MetaPlan, offsetY int) {
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
func (bw BridgeFacilityWrapper) collectBridgesFromTemplate(
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

	// ヒント情報を収集
	for _, hint := range template.HintPlacements {
		metaPlan.Hints = append(metaPlan.Hints, maptemplate.HintPlacement{
			HintType: hint.HintType,
			ExitID:   hint.ExitID,
			X:        hint.X,
			Y:        hint.Y + offsetY,
		})
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
	if err := loader.RegisterAllChunks([]string{"assets/levels/chunks"}); err != nil {
		return nil, err
	}

	return &BridgeFacilityWrapper{
		Loader: loader,
	}, nil
}
