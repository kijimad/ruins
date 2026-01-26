package mapplanner

// BridgeConnection は橋facilityとの接続のため、最上列・最下列を床にするPlanner
// BridgeFacilityWrapperと組み合わせて使用する
type BridgeConnection struct{}

// PlanMeta はマップの最上列と最下列を床タイルに変更する
// 橋facilityとの接続をスムーズにするため、境界付近の行を床にする
func (b BridgeConnection) PlanMeta(planData *MetaPlan) error {
	width := int(planData.Level.TileWidth)
	height := int(planData.Level.TileHeight)

	floorTile := planData.GetTile("floor")

	// マップサイズに応じて床にする行数を決定
	// 小さいマップ（高さ15以下）では1行、それ以外は2行
	rowsToFloor := 1
	if height > 15 {
		rowsToFloor = 2
	}

	// 最上N行を床にする（橋facilityとダンジョンの境界壁を除去）
	for y := 0; y < rowsToFloor && y < height; y++ {
		for x := 0; x < width; x++ {
			planData.Tiles[y*width+x] = floorTile
		}
	}

	// 最下N行を床にする（橋facilityとダンジョンの境界壁を除去）
	for y := height - rowsToFloor; y < height && y >= 0; y++ {
		for x := 0; x < width; x++ {
			planData.Tiles[y*width+x] = floorTile
		}
	}

	return nil
}
