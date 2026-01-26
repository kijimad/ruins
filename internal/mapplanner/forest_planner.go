package mapplanner

import (
	"math"

	gc "github.com/kijimaD/ruins/internal/components"
)

// ForestPlanner は森風レイアウトを生成するビルダー
// 木々が点在し、自然な通路を持つ森を作成
type ForestPlanner struct{}

// PlanInitial は初期森マップをプランする
func (f ForestPlanner) PlanInitial(planData *MetaPlan) error {
	width := int(planData.Level.TileWidth)
	height := int(planData.Level.TileHeight)

	// 森の中に小さな空き地（部屋）をいくつか作成
	clearingCount := 3 + planData.RNG.IntN(4)

	for i := 0; i < clearingCount; i++ {
		// 空き地のサイズとランダムな位置
		clearingWidth := 4 + planData.RNG.IntN(6)
		clearingHeight := 4 + planData.RNG.IntN(6)

		// IntNの引数が正であることを保証
		maxXRange := max(1, width-clearingWidth-6)
		maxYRange := max(1, height-clearingHeight-6)
		x := 3 + planData.RNG.IntN(maxXRange)
		y := 3 + planData.RNG.IntN(maxYRange)

		// 円形に近い空き地を作成
		centerX := x + clearingWidth/2
		centerY := y + clearingHeight/2
		radius := math.Min(float64(clearingWidth), float64(clearingHeight)) / 2.0

		clearing := gc.Rect{
			X1: gc.Tile(centerX - int(radius)),
			Y1: gc.Tile(centerY - int(radius)),
			X2: gc.Tile(centerX + int(radius)),
			Y2: gc.Tile(centerY + int(radius)),
		}
		planData.Rooms = append(planData.Rooms, clearing)
	}
	return nil
}

// ForestTerrain は森の基本地形を生成する
type ForestTerrain struct{}

// PlanMeta は森の基本地形をタイルに描画する
func (f ForestTerrain) PlanMeta(planData *MetaPlan) error {
	// まず全体を床で埋める（森の地面）
	for i := range planData.Tiles {
		planData.Tiles[i] = planData.GetTile("floor")
	}

	// 空き地を床として確保
	for _, clearing := range planData.Rooms {
		f.createCircularClearing(planData, clearing)
	}
	return nil
}

// createCircularClearing は円形の空き地を作成する
func (f ForestTerrain) createCircularClearing(planData *MetaPlan, clearing gc.Rect) {
	centerX := float64(clearing.X1+clearing.X2) / 2.0
	centerY := float64(clearing.Y1+clearing.Y2) / 2.0
	radius := math.Min(float64(clearing.X2-clearing.X1), float64(clearing.Y2-clearing.Y1)) / 2.0

	for x := clearing.X1 - 1; x <= clearing.X2+1; x++ {
		for y := clearing.Y1 - 1; y <= clearing.Y2+1; y++ {
			dx := float64(x) - centerX
			dy := float64(y) - centerY
			distance := math.Sqrt(dx*dx + dy*dy)

			if distance <= radius {
				idx := planData.Level.XYTileIndex(x, y)
				planData.Tiles[idx] = planData.GetTile("floor")
			}
		}
	}
}

// ForestTrees は森に木々を配置する
type ForestTrees struct{}

// PlanMeta は森に木を配置する
func (f ForestTrees) PlanMeta(planData *MetaPlan) error {
	width := int(planData.Level.TileWidth)
	height := int(planData.Level.TileHeight)

	// 森全体に木を配置（60%の密度）
	for x := 1; x < width-1; x++ {
		for y := 1; y < height-1; y++ {
			idx := planData.Level.XYTileIndex(gc.Tile(x), gc.Tile(y))

			if planData.Tiles[idx].Walkable {
				// 空き地の近くでは木の密度を下げる
				treeDensity := f.calculateTreeDensity(planData, x, y)

				if planData.RNG.Float64() < treeDensity {
					// TODO: 木エンティティとして追加する
					planData.Tiles[idx] = planData.GetTile("wall")

					// 大きな木の場合、周囲にも追加の木を配置
					if planData.RNG.Float64() < 0.2 { // 20%の確率で大木
						f.placeLargeTree(planData, x, y)
					}
				}
			}
		}
	}
	return nil
}

// calculateTreeDensity は位置に基づいて木の密度を計算する
func (f ForestTrees) calculateTreeDensity(planData *MetaPlan, x, y int) float64 {
	baseDensity := 0.6 // 基本密度60%

	// 空き地からの距離に基づいて密度を調整
	minDistanceToClearing := 1000.0

	for _, clearing := range planData.Rooms {
		centerX := float64(clearing.X1+clearing.X2) / 2.0
		centerY := float64(clearing.Y1+clearing.Y2) / 2.0

		dx := float64(x) - centerX
		dy := float64(y) - centerY
		distance := math.Sqrt(dx*dx + dy*dy)

		if distance < minDistanceToClearing {
			minDistanceToClearing = distance
		}
	}

	// 空き地に近いほど木の密度を下げる
	if minDistanceToClearing < 5 {
		return baseDensity * 0.3 // 空き地の近くは30%
	} else if minDistanceToClearing < 10 {
		return baseDensity * 0.7 // 中距離は70%
	}
	return baseDensity // 遠くは基本密度
}

// placeLargeTree は大きな木を配置する
func (f ForestTrees) placeLargeTree(planData *MetaPlan, centerX, centerY int) {
	width := int(planData.Level.TileWidth)
	height := int(planData.Level.TileHeight)

	// 大木の周囲2x2または3x3エリアに追加の木を配置
	size := 1 + planData.RNG.IntN(2) // 1または2

	for dx := -size; dx <= size; dx++ {
		for dy := -size; dy <= size; dy++ {
			x, y := centerX+dx, centerY+dy

			if x >= 0 && x < width && y >= 0 && y < height {
				idx := planData.Level.XYTileIndex(gc.Tile(x), gc.Tile(y))

				if planData.Tiles[idx].Walkable && planData.RNG.Float64() < 0.7 {
					planData.Tiles[idx] = planData.GetTile("wall")
				}
			}
		}
	}
}

// ForestPaths は森の中に自然な通路を作成する
type ForestPaths struct{}

// PlanMeta は空き地間に自然な通路を作成する
func (f ForestPaths) PlanMeta(planData *MetaPlan) error {
	if len(planData.Rooms) < 2 {
		return nil
	}

	// 各空き地を他の空き地と繋ぐ
	for i := 0; i < len(planData.Rooms); i++ {
		for j := i + 1; j < len(planData.Rooms); j++ {
			// 距離が近い空き地のみ繋ぐ（50%の確率）
			if f.shouldCreatePath(planData, planData.Rooms[i], planData.Rooms[j]) {
				f.createNaturalPath(planData, planData.Rooms[i], planData.Rooms[j])
			}
		}
	}
	return nil
}

// shouldCreatePath は通路を作成するかどうかを判定する
func (f ForestPaths) shouldCreatePath(planData *MetaPlan, room1, room2 gc.Rect) bool {
	// 空き地間の距離を計算
	center1X := float64(room1.X1+room1.X2) / 2.0
	center1Y := float64(room1.Y1+room1.Y2) / 2.0
	center2X := float64(room2.X1+room2.X2) / 2.0
	center2Y := float64(room2.Y1+room2.Y2) / 2.0

	dx := center1X - center2X
	dy := center1Y - center2Y
	distance := math.Sqrt(dx*dx + dy*dy)

	// 近い空き地ほど繋がりやすい
	if distance < 15 {
		return planData.RNG.Float64() < 0.8
	} else if distance < 25 {
		return planData.RNG.Float64() < 0.4
	}
	return planData.RNG.Float64() < 0.1
}

// createNaturalPath は自然な曲線状の通路を作成する
func (f ForestPaths) createNaturalPath(planData *MetaPlan, room1, room2 gc.Rect) {
	width := int(planData.Level.TileWidth)
	height := int(planData.Level.TileHeight)

	center1X := int(room1.X1+room1.X2) / 2
	center1Y := int(room1.Y1+room1.Y2) / 2
	center2X := int(room2.X1+room2.X2) / 2
	center2Y := int(room2.Y1+room2.Y2) / 2

	// ベジェ曲線風の自然な通路を作成
	steps := 50
	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)

		// 制御点を追加してカーブを作る
		midX := (center1X + center2X) / 2
		midY := (center1Y + center2Y) / 2

		// ランダムな偏向を追加
		randomOffsetX := int(float64(planData.RNG.IntN(11)-5) * (1.0 - math.Abs(t-0.5)*2))
		randomOffsetY := int(float64(planData.RNG.IntN(11)-5) * (1.0 - math.Abs(t-0.5)*2))

		// 2次ベジェ曲線の近似
		x := int((1-t)*(1-t)*float64(center1X) + 2*(1-t)*t*float64(midX+randomOffsetX) + t*t*float64(center2X))
		y := int((1-t)*(1-t)*float64(center1Y) + 2*(1-t)*t*float64(midY+randomOffsetY) + t*t*float64(center2Y))

		// 通路を作成（少し幅を持たせる）
		for dx := -1; dx <= 1; dx++ {
			for dy := -1; dy <= 1; dy++ {
				nx, ny := x+dx, y+dy
				if nx >= 0 && nx < width && ny >= 0 && ny < height {
					idx := planData.Level.XYTileIndex(gc.Tile(nx), gc.Tile(ny))

					// 70%の確率で通路を作成（自然な感じに）
					if planData.RNG.Float64() < 0.7 {
						planData.Tiles[idx] = planData.GetTile("floor")
					}
				}
			}
		}
	}
}

// ForestWildlife は森に野生動物の痕跡（小さな空き地）を追加する
type ForestWildlife struct{}

// PlanMeta は森に小さな動物の痕跡を追加する
func (f ForestWildlife) PlanMeta(planData *MetaPlan) error {
	width := int(planData.Level.TileWidth)
	height := int(planData.Level.TileHeight)

	// 小さな動物の通り道や巣穴を作成
	wildlifeSpotCount := 2 + planData.RNG.IntN(4)

	for i := 0; i < wildlifeSpotCount; i++ {
		// IntNの引数が正であることを保証
		maxXRange := max(1, width-4)
		maxYRange := max(1, height-4)
		x := 2 + planData.RNG.IntN(maxXRange)
		y := 2 + planData.RNG.IntN(maxYRange)

		// 小さな円形の空き地を作成
		radius := 1 + planData.RNG.IntN(2)

		for dx := -radius; dx <= radius; dx++ {
			for dy := -radius; dy <= radius; dy++ {
				nx, ny := x+dx, y+dy
				if nx >= 0 && nx < width && ny >= 0 && ny < height {
					distance := math.Sqrt(float64(dx*dx + dy*dy))
					if distance <= float64(radius) {
						idx := planData.Level.XYTileIndex(gc.Tile(nx), gc.Tile(ny))
						planData.Tiles[idx] = planData.GetTile("floor")
					}
				}
			}
		}
	}
	return nil
}

// NewForestPlanner は森ビルダーを作成する
func NewForestPlanner(width gc.Tile, height gc.Tile, seed uint64) (*PlannerChain, error) {
	chain := NewPlannerChain(width, height, seed)
	chain.StartWith(ForestPlanner{})
	chain.With(ForestTerrain{})    // 基本地形を生成
	chain.With(ForestTrees{})      // 木を配置
	chain.With(ForestPaths{})      // 自然な通路を作成
	chain.With(ForestWildlife{})   // 野生動物の痕跡を追加
	chain.With(BridgeConnection{}) // 橋facilityとの接続のため最上列・最下列を床にする

	return chain, nil
}
