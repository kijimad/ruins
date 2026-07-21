package systems

import (
	"fmt"
	"image/color"
	"math"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/geometry"
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// raycastCacheKey はレイキャスト結果のキャッシュキー
type raycastCacheKey struct {
	Player consts.Coord[int]
	Target consts.Coord[int]
}

// VisionSystem はタイルごとの視界を計算するUpdaterシステム。
// 計算結果の光源情報は Dungeon シングルトンに書き込み、描画側はそこから参照する。
// レイキャストキャッシュなどの内部メモは本システムが保持し、フロア変化時に自身で破棄する
type VisionSystem struct {
	// プレイヤー位置キャッシュ（タイル移動ごとに更新）
	lastPlayer    consts.Coord[consts.WorldPixel]
	isInitialized bool

	// レイキャスト結果のキャッシュ
	raycastCache map[raycastCacheKey]bool

	// キャッシュが対象とするフロアの識別情報。変化を検知して内部キャッシュを破棄する
	lastDepth          int
	lastDefinitionName string
}

// NewVisionSystem はVisionSystemを初期化する
func NewVisionSystem() *VisionSystem {
	return &VisionSystem{
		raycastCache: make(map[raycastCacheKey]bool),
	}
}

// invalidateOnFloorChange はフロアが切り替わっていれば内部キャッシュを破棄する。
// レイキャスト結果は壁配置に依存するため、フロアをまたいで再利用すると誤った視界になる
func (sys *VisionSystem) invalidateOnFloorChange(dungeon *gc.Dungeon, vs *gc.VisionState) {
	if dungeon.CurrentStage.Depth == sys.lastDepth && dungeon.DefinitionName == sys.lastDefinitionName {
		return
	}
	sys.lastDepth = dungeon.CurrentStage.Depth
	sys.lastDefinitionName = dungeon.DefinitionName
	sys.isInitialized = false
	sys.raycastCache = make(map[raycastCacheKey]bool)
	vs.LightSourceCache = make(map[gc.GridElement]gc.LightInfo)
}

// String はシステム名を返す
// w.Updater interfaceを実装
func (sys VisionSystem) String() string {
	return "VisionSystem"
}

// Update は視界計算を行う
// w.Updater interfaceを実装
func (sys *VisionSystem) Update(world w.World) error {
	// プレイヤー位置を取得
	var playerGridElement *gc.GridElement
	playerQuery := ecs.NewFilter2[gc.GridElement, gc.Player](world.ECS).Query()
	for playerQuery.Next() {
		entity := playerQuery.Entity()
		playerGridElement = world.Components.GridElement.Get(entity)
	}

	if playerGridElement == nil {
		return nil
	}

	// タイル座標をワールドピクセル座標に変換
	playerPos := consts.TileCenterToWorld(playerGridElement.Coord)

	dungeon := query.GetDungeon(world)
	if dungeon == nil {
		return nil
	}
	meta := query.GetCurrentStageMeta(world)
	if meta == nil {
		return nil
	}
	vs := query.GetVisionState(world)

	// フロアが切り替わっていれば壁配置依存の内部キャッシュを破棄する
	sys.invalidateOnFloorChange(dungeon, vs)

	// 移動ごとの視界更新判定（移動ごとに更新）
	const updateThreshold = int(consts.TileSize)
	needsUpdate := !sys.isInitialized ||
		geometry.Abs(int(playerPos.X-sys.lastPlayer.X)) >= updateThreshold ||
		geometry.Abs(int(playerPos.Y-sys.lastPlayer.Y)) >= updateThreshold

	// 外部から設定された視界更新フラグをチェックする(扉開閉時など)
	if vs.NeedsForceUpdate {
		needsUpdate = true
		vs.NeedsForceUpdate = false
	}

	if !needsUpdate {
		return nil
	}

	// 視界遮断タイルのインデックスを構築する
	blockViewIndex := buildBlockViewIndex(world)

	// タイルの可視性マップを更新
	visionRadius := consts.WorldPixel(consts.VisionRadiusTiles) * consts.TileSize
	visibilityData := calculateTileVisibilityWithDistance(playerPos, visionRadius, sys.raycastCache, blockViewIndex)

	// 光源情報を更新前にクリアする
	vs.LightSourceCache = make(map[gc.GridElement]gc.LightInfo)

	// 視界内タイルの光源情報を計算し、探索済みマークを行う。
	// マップ外座標はデータに含めない
	visibleTiles := make(map[gc.GridElement]bool)
	for _, tileData := range visibilityData {
		if !tileData.Visible {
			continue
		}
		gridElement := gc.GridElement{Coord: consts.Coord[consts.Tile]{X: consts.Tile(tileData.Col), Y: consts.Tile(tileData.Row)}}
		if !isInMapBounds(gridElement, meta.Level) {
			continue
		}

		vs.LightSourceCache[gridElement] = calculateLightSourceDarkness(world, consts.Coord[int]{X: tileData.Col, Y: tileData.Row})
		meta.ExploredTiles[gridElement] = true
		visibleTiles[gridElement] = true
	}
	vs.VisibleTiles = visibleTiles

	sys.lastPlayer = playerPos
	sys.isInitialized = true

	return nil
}

// TileVisibility はレイキャストによるタイルの可視性判定結果を表す
type TileVisibility struct {
	Row     int
	Col     int
	Visible bool
}

type (
	// VisibleDarkness は視界内タイルの暗闇の強さを表す
	VisibleDarkness float64
	// RememberedDarkness は記憶済みタイルの暗闇の強さを表す
	RememberedDarkness float64
)

// TileRenderInfo はタイルごとの描画情報を表す
type TileRenderInfo interface {
	tileRenderInfo()
}

// TileRenderVisible は視界内の状態
type TileRenderVisible struct {
	Darkness   VisibleDarkness // 暗闘の強さ。0.0で完全に明るく、1.0で完全に暗い
	LightColor color.RGBA      // 光源がある場合の色
}

func (TileRenderVisible) tileRenderInfo() {}

// TileRenderRemembered は視界外だが記憶済みの状態。床のみうっすら描画する
type TileRenderRemembered struct {
	Darkness RememberedDarkness
}

func (TileRenderRemembered) tileRenderInfo() {}

// computeTileRenderMap はタイルごとの描画情報を一括計算する。
// VisibleTiles・ExploredTiles・光源情報を統合して、
// 各描画関数が参照するだけで済む描画情報マップを返す
func computeTileRenderMap(world w.World, lights map[gc.GridElement]gc.LightInfo) map[gc.GridElement]TileRenderInfo {
	result := make(map[gc.GridElement]TileRenderInfo)
	meta := query.GetCurrentStageMeta(world)
	vs := query.GetVisionState(world)

	// 現在見えているタイルを設定する
	for grid := range vs.VisibleTiles {
		visible := TileRenderVisible{Darkness: DarknessVisible}
		if li, ok := lights[grid]; ok && li.Darkness < 1.0 {
			visible.LightColor = li.Color
		}
		result[grid] = visible
	}

	// 視界外だが記憶済みのタイルを設定する
	if meta != nil {
		for grid := range meta.ExploredTiles {
			if _, exists := result[grid]; !exists {
				result[grid] = TileRenderRemembered{Darkness: DarknessRemembered}
			}
		}
	}

	return result
}

// isInMapBounds は座標がマップの有効範囲内かを判定する
func isInMapBounds(grid gc.GridElement, level gc.Level) bool {
	return grid.X >= 0 && grid.X < level.TileWidth && grid.Y >= 0 && grid.Y < level.TileHeight
}

// calculateTileVisibilityWithDistance はレイキャストでタイルごとの可視性と距離を計算する
func calculateTileVisibilityWithDistance(playerPos consts.Coord[consts.WorldPixel], radius consts.WorldPixel, rcCache map[raycastCacheKey]bool, blockIndex map[gc.GridElement]bool) map[string]TileVisibility {
	visibilityMap := make(map[string]TileVisibility)

	// プレイヤーの位置からタイル座標を計算
	playerTileX := int(playerPos.X) / int(consts.TileSize)
	playerTileY := int(playerPos.Y) / int(consts.TileSize)

	// 視界範囲を分割して段階的処理（視界範囲最適化）
	maxTileDistance := int(radius)/int(consts.TileSize) + 2

	// タイルベース視界判定（Dark Days Ahead風）

	for dx := -maxTileDistance; dx <= maxTileDistance; dx++ {
		for dy := -maxTileDistance; dy <= maxTileDistance; dy++ {
			tileX := playerTileX + dx
			tileY := playerTileY + dy

			// タイルの中心座標を計算
			tileCenter := consts.TileCenterToWorld(consts.Coord[consts.Tile]{X: consts.Tile(tileX), Y: consts.Tile(tileY)})

			// プレイヤーからタイル中心への距離をチェック（平方根計算の最適化）
			dxF := float64(tileCenter.X - playerPos.X)
			dyF := float64(tileCenter.Y - playerPos.Y)
			distanceSquared := dxF*dxF + dyF*dyF
			radiusSquared := float64(radius) * float64(radius)

			tileKey := fmt.Sprintf("%d,%d", tileX, tileY)

			// 視界範囲内のタイルのみ処理
			if distanceSquared <= radiusSquared {
				visible := isTileVisibleByRaycast(playerPos, tileCenter, rcCache, blockIndex)

				visibilityMap[tileKey] = TileVisibility{
					Row:     tileY,
					Col:     tileX,
					Visible: visible,
				}
			}
			// 視界外のタイルは処理しない（最適化）
		}
	}

	return visibilityMap
}

// isTileVisibleByRaycast はタイルベース視界判定
func isTileVisibleByRaycast(player, target consts.Coord[consts.WorldPixel], rcCache map[raycastCacheKey]bool, blockIndex map[gc.GridElement]bool) bool {
	// キャッシュキーを生成（4刻みに丸めて近い視線を共有する）
	cacheKey := raycastCacheKey{
		Player: consts.Coord[int]{X: int(player.X/4) * 4, Y: int(player.Y/4) * 4},
		Target: consts.Coord[int]{X: int(target.X/4) * 4, Y: int(target.Y/4) * 4},
	}

	// キャッシュから結果をチェック
	if result, exists := rcCache[cacheKey]; exists {
		return result
	}

	// タイル座標に変換
	playerTileX := int(player.X / consts.TileSize)
	playerTileY := int(player.Y / consts.TileSize)
	targetTileX := int(target.X / consts.TileSize)
	targetTileY := int(target.Y / consts.TileSize)

	// 同じタイルまたは隣接タイルは常に見える
	if geometry.Abs(targetTileX-playerTileX) <= 1 && geometry.Abs(targetTileY-playerTileY) <= 1 {
		rcCache[cacheKey] = true
		return true
	}

	// ブレゼンハムのライン描画アルゴリズムでタイルベースの視線判定
	result := bresenhamLineOfSight(playerTileX, playerTileY, targetTileX, targetTileY, blockIndex)

	// 結果をキャッシュ
	if len(rcCache) < 15000 {
		rcCache[cacheKey] = result
	}

	return result
}

// bresenhamLineOfSight はブレゼンハムアルゴリズムを使用したタイルベース視線判定
func bresenhamLineOfSight(x0, y0, x1, y1 int, blockIndex map[gc.GridElement]bool) bool {
	dx := geometry.Abs(x1 - x0)
	dy := geometry.Abs(y1 - y0)

	var sx, sy int
	if x0 < x1 {
		sx = 1
	} else {
		sx = -1
	}
	if y0 < y1 {
		sy = 1
	} else {
		sy = -1
	}

	err := dx - dy
	x, y := x0, y0

	for {
		// ターゲットに到達したら見える
		if x == x1 && y == y1 {
			return true
		}

		// 現在のタイルが視界を遮るかチェック
		if blockIndex[gc.GridElement{Coord: consts.Coord[consts.Tile]{X: consts.Tile(x), Y: consts.Tile(y)}}] {
			return false
		}

		e2 := 2 * err
		if e2 > -dy {
			err -= dy
			x += sx
		}
		if e2 < dx {
			err += dx
			y += sy
		}
	}
}

// calculateLightSourceDarkness は光源からの距離に応じた暗闇レベルと色を計算する
func calculateLightSourceDarkness(world w.World, tile consts.Coord[int]) gc.LightInfo {
	minDarkness := 1.0 // 完全に暗い状態からスタート

	// 加重平均用の累積値
	var totalR, totalG, totalB float64
	var totalWeight float64

	// 全ての光源をチェック。退避中ステージの光源は現ステージを照らさない
	lightQuery := query.ActiveFilter2[gc.LightSource, gc.GridElement](world).Query()
	for lightQuery.Next() {
		lightEntity := lightQuery.Entity()
		lightSource := world.Components.LightSource.Get(lightEntity)

		// 無効な光源はスキップ
		if !lightSource.Enabled {
			continue
		}

		lightGrid := world.Components.GridElement.Get(lightEntity)

		// 距離計算（タイル単位）
		distance := geometry.Distance(float64(tile.X), float64(tile.Y), float64(lightGrid.X), float64(lightGrid.Y))

		// 光源範囲内かチェック
		if distance <= float64(lightSource.Radius) {
			// 光源中心（距離0-1タイル）も周囲と同じ明るさにする
			if distance < 1.0 {
				distance = 1.0
			}

			// 距離の正規化
			normalizedDistance := distance / float64(lightSource.Radius)

			// 光源中心から滑らかに暗くなる
			// 中心付近も少し暗くする（最小暗闇レベル0.3）
			darkness := math.Pow(normalizedDistance, 1.5)*0.6 + 0.3

			// 暗闇レベルは最も明るい光源を採用
			if darkness < minDarkness {
				minDarkness = darkness
			}

			// 光の強さを重みとして使用する。近いほど強い
			weight := 1.0 - normalizedDistance

			// 加重平均のための累積
			totalR += float64(lightSource.Color.R) * weight
			totalG += float64(lightSource.Color.G) * weight
			totalB += float64(lightSource.Color.B) * weight
			totalWeight += weight
		}
	}

	// 加重平均を計算
	var finalR, finalG, finalB uint8
	if totalWeight > 0 {
		finalR = uint8(math.Min(255, totalR/totalWeight))
		finalG = uint8(math.Min(255, totalG/totalWeight))
		finalB = uint8(math.Min(255, totalB/totalWeight))
	}

	return gc.LightInfo{
		Darkness: minDarkness,
		Color:    color.RGBA{R: finalR, G: finalG, B: finalB, A: 255},
	}
}

// 各タイル状態の暗闇の強さ
const (
	DarknessVisible    VisibleDarkness    = 0.15
	DarknessRemembered RememberedDarkness = 0.75
)

// buildBlockViewIndex は全BlockViewエンティティのタイル座標をインデックス化する
func buildBlockViewIndex(world w.World) map[gc.GridElement]bool {
	index := make(map[gc.GridElement]bool)
	// 退避中ステージの遮蔽物は現ステージの視界を遮らない
	blockViewQuery := query.ActiveFilter2[gc.GridElement, gc.BlockView](world).Query()
	for blockViewQuery.Next() {
		entity := blockViewQuery.Entity()
		grid := world.Components.GridElement.Get(entity)
		index[*grid] = true
	}
	return index
}
