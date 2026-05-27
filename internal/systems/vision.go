package systems

import (
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/geometry"
	w "github.com/kijimaD/ruins/internal/world"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// raycastCacheKey はレイキャスト結果のキャッシュキー
type raycastCacheKey struct {
	PlayerX int
	PlayerY int
	TargetX int
	TargetY int
}

// VisionSystem はタイルごとの視界を管理する（暗闘描画はRenderSpriteSystemで行う）
type VisionSystem struct {
	// プレイヤー位置キャッシュ（タイル移動ごとに更新）
	lastPlayerX    consts.Pixel
	lastPlayerY    consts.Pixel
	visibilityData map[string]TileVisibility
	isInitialized  bool

	// レイキャスト結果のキャッシュ
	raycastCache map[raycastCacheKey]bool

	// 光源情報キャッシュ（タイル座標 -> 光源情報）。RenderSpriteSystemからも参照される
	lightSourceCache map[gc.GridElement]LightInfo
}

// NewVisionSystem はVisionSystemを初期化する
func NewVisionSystem() *VisionSystem {
	return &VisionSystem{
		raycastCache:     make(map[raycastCacheKey]bool),
		lightSourceCache: make(map[gc.GridElement]LightInfo),
	}
}

// ClearCaches は全ての視界関連キャッシュをクリアする（階移動時などに使用）
func (sys *VisionSystem) ClearCaches() {
	sys.isInitialized = false
	sys.visibilityData = nil
	sys.raycastCache = make(map[raycastCacheKey]bool)
	sys.lightSourceCache = make(map[gc.GridElement]LightInfo)
}

// String はシステム名を返す
// w.Renderer interfaceを実装
func (sys VisionSystem) String() string {
	return "VisionSystem"
}

// Draw は視界計算を行う
// w.Renderer interfaceを実装
func (sys *VisionSystem) Draw(world w.World, _ *ebiten.Image) error {
	// プレイヤー位置を取得
	var playerGridElement *gc.GridElement
	world.Manager.Join(
		world.Components.GridElement,
		world.Components.Player,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		playerGridElement = world.Components.GridElement.Get(entity).(*gc.GridElement)
	}))

	if playerGridElement == nil {
		return nil
	}

	// タイル座標をピクセル座標に変換
	playerPos := &gc.Position{
		X: consts.Pixel(int(playerGridElement.X)*int(consts.TileSize) + int(consts.TileSize)/2),
		Y: consts.Pixel(int(playerGridElement.Y)*int(consts.TileSize) + int(consts.TileSize)/2),
	}

	// 移動ごとの視界更新判定（移動ごとに更新）
	const updateThreshold = int(consts.TileSize)
	needsUpdate := !sys.isInitialized ||
		geometry.Abs(int(playerPos.X-sys.lastPlayerX)) >= updateThreshold ||
		geometry.Abs(int(playerPos.Y-sys.lastPlayerY)) >= updateThreshold

	// 外部から設定された視界更新フラグをチェックする(扉開閉時など)
	if world.Resources.Dungeon != nil && world.Resources.Dungeon.NeedsForceUpdate {
		needsUpdate = true
		world.Resources.Dungeon.NeedsForceUpdate = false
	}

	if needsUpdate {
		// タイルの可視性マップを更新
		visionRadius := consts.Pixel(consts.VisionRadiusTiles * consts.TileSize)
		visibilityData := calculateTileVisibilityWithDistance(world, playerPos.X, playerPos.Y, visionRadius, sys.raycastCache)

		// 光源情報キャッシュをクリア（更新前）
		sys.lightSourceCache = make(map[gc.GridElement]LightInfo)

		isDark := world.Resources.Dungeon.Dark

		// 視界内タイルの光源情報を計算し、探索済みマークを行う
		for _, tileData := range visibilityData {
			if tileData.Visible {
				lightInfo := calculateLightSourceDarkness(world, tileData.Col, tileData.Row)
				gridElement := gc.GridElement{X: consts.Tile(tileData.Col), Y: consts.Tile(tileData.Row)}

				// 光源情報をキャッシュに保存
				sys.lightSourceCache[gridElement] = lightInfo

				// 暗闇フロアでは光源範囲内のみ探索済み、明るいフロアでは視界内すべて探索済み
				if !isDark || lightInfo.Darkness < 1.0 {
					world.Resources.Dungeon.ExploredTiles[gridElement] = true
				}
			}
		}

		// 現在フレームで見えているタイルをリソースに反映する
		visibleTiles := make(map[gc.GridElement]bool)
		for _, tileData := range visibilityData {
			if tileData.Visible {
				gridElement := gc.GridElement{X: consts.Tile(tileData.Col), Y: consts.Tile(tileData.Row)}
				if isDark {
					// 暗闇フロアでは光源範囲内のみ可視
					if li, ok := sys.lightSourceCache[gridElement]; ok && li.Darkness < 1.0 {
						visibleTiles[gridElement] = true
					}
				} else {
					// 明るいフロアでは視界内すべて可視
					visibleTiles[gridElement] = true
				}
			}
		}
		world.Resources.Dungeon.VisibleTiles = visibleTiles

		// キャッシュ更新
		sys.lastPlayerX = playerPos.X
		sys.lastPlayerY = playerPos.Y
		sys.visibilityData = visibilityData
		sys.isInitialized = true
	}
	// 距離に応じた段階的暗闇の描画はRenderSpriteSystemで行う
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
	// DarkDarkness は暗闇フロアで光源外タイルの暗闇の強さを表す
	DarkDarkness float64
	// RememberedDarkness は記憶済みタイルの暗闇の強さを表す
	RememberedDarkness float64
)

// TileRenderInfo はタイルごとの描画情報を表す
type TileRenderInfo interface {
	tileRenderInfo()
}

// TileRenderVisible は視界内の状態
type TileRenderVisible struct {
	Darkness   VisibleDarkness // 暗闇の強さ。0.0で完全に明るく、1.0で完全に暗い
	LightColor color.RGBA      // 光源がある場合の色
}

func (TileRenderVisible) tileRenderInfo() {}

// TileRenderDark は視界内だが暗くて見えない状態。暗闇フロアで光源外のタイル
type TileRenderDark struct {
	Darkness DarkDarkness
}

func (TileRenderDark) tileRenderInfo() {}

// TileRenderRemembered は視界外だが記憶済みの状態。床のみうっすら描画する
type TileRenderRemembered struct {
	Darkness RememberedDarkness
}

func (TileRenderRemembered) tileRenderInfo() {}

// computeTileRenderMap はタイルごとの描画情報を一括計算する。
// VisibleTiles・ExploredTiles・光源情報を統合して、
// 各描画関数が参照するだけで済む描画情報マップを返す
func computeTileRenderMap(world w.World, lights map[gc.GridElement]LightInfo) map[gc.GridElement]TileRenderInfo {
	result := make(map[gc.GridElement]TileRenderInfo)

	// 現在見えているタイルを設定する
	for grid := range world.Resources.Dungeon.VisibleTiles {
		visible := TileRenderVisible{Darkness: DarknessVisible}
		if li, ok := lights[grid]; ok && li.Darkness < 1.0 {
			visible.LightColor = li.Color
		}
		result[grid] = visible
	}

	// 暗闇フロアで視界内だが光源外のタイルを設定する
	if world.Resources.Dungeon.Dark {
		for grid, li := range lights {
			if _, exists := result[grid]; !exists && li.Darkness >= 1.0 {
				result[grid] = TileRenderDark{Darkness: DarknessDark}
			}
		}
	}

	// 視界外だが記憶済みのタイルを設定する
	for grid := range world.Resources.Dungeon.ExploredTiles {
		if _, exists := result[grid]; !exists {
			result[grid] = TileRenderRemembered{Darkness: DarknessRemembered}
		}
	}

	return result
}

// calculateTileVisibilityWithDistance はレイキャストでタイルごとの可視性と距離を計算する
func calculateTileVisibilityWithDistance(world w.World, playerX, playerY, radius consts.Pixel, rcCache map[raycastCacheKey]bool) map[string]TileVisibility {
	visibilityMap := make(map[string]TileVisibility)

	// プレイヤーの位置からタイル座標を計算
	playerTileX := int(playerX) / int(consts.TileSize)
	playerTileY := int(playerY) / int(consts.TileSize)

	// 視界範囲を分割して段階的処理（視界範囲最適化）
	maxTileDistance := int(radius)/int(consts.TileSize) + 2

	// タイルベース視界判定（Dark Days Ahead風）

	for dx := -maxTileDistance; dx <= maxTileDistance; dx++ {
		for dy := -maxTileDistance; dy <= maxTileDistance; dy++ {
			tileX := playerTileX + dx
			tileY := playerTileY + dy

			// 早期距離チェック（枝払い）
			if geometry.Abs(dx) > maxTileDistance || geometry.Abs(dy) > maxTileDistance {
				continue
			}

			// タイルの中心座標を計算
			tileCenterX := float64(tileX*int(consts.TileSize) + int(consts.TileSize)/2)
			tileCenterY := float64(tileY*int(consts.TileSize) + int(consts.TileSize)/2)

			// プレイヤーからタイル中心への距離をチェック（平方根計算の最適化）
			dxF := tileCenterX - float64(playerX)
			dyF := tileCenterY - float64(playerY)
			distanceSquared := dxF*dxF + dyF*dyF
			radiusSquared := float64(radius) * float64(radius)

			tileKey := fmt.Sprintf("%d,%d", tileX, tileY)

			// 視界範囲内のタイルのみ処理
			if distanceSquared <= radiusSquared {
				visible := isTileVisibleByRaycast(world, float64(playerX), float64(playerY), tileCenterX, tileCenterY, rcCache)

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
func isTileVisibleByRaycast(world w.World, playerX, playerY, targetX, targetY float64, rcCache map[raycastCacheKey]bool) bool {
	// キャッシュキーを生成
	px := int(playerX/4) * 4
	py := int(playerY/4) * 4
	tx := int(targetX/4) * 4
	ty := int(targetY/4) * 4
	cacheKey := raycastCacheKey{
		PlayerX: px,
		PlayerY: py,
		TargetX: tx,
		TargetY: ty,
	}

	// キャッシュから結果をチェック
	if result, exists := rcCache[cacheKey]; exists {
		return result
	}

	// タイル座標に変換
	playerTileX := int(playerX / float64(consts.TileSize))
	playerTileY := int(playerY / float64(consts.TileSize))
	targetTileX := int(targetX / float64(consts.TileSize))
	targetTileY := int(targetY / float64(consts.TileSize))

	// 同じタイルまたは隣接タイルは常に見える
	if geometry.Abs(targetTileX-playerTileX) <= 1 && geometry.Abs(targetTileY-playerTileY) <= 1 {
		rcCache[cacheKey] = true
		return true
	}

	// ブレゼンハムのライン描画アルゴリズムでタイルベースの視線判定
	result := bresenhamLineOfSight(world, playerTileX, playerTileY, targetTileX, targetTileY)

	// 結果をキャッシュ
	if len(rcCache) < 15000 {
		rcCache[cacheKey] = result
	}

	return result
}

// bresenhamLineOfSight はブレゼンハムアルゴリズムを使用したタイルベース視線判定
func bresenhamLineOfSight(world w.World, x0, y0, x1, y1 int) bool {
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

		// 現在のタイルが壁かチェック（ターゲット以外）
		if x != x1 || y != y1 {
			tileCenterX := float64(x*int(consts.TileSize) + int(consts.TileSize)/2)
			tileCenterY := float64(y*int(consts.TileSize) + int(consts.TileSize)/2)
			if isBlockedByWall(world, consts.Pixel(tileCenterX), consts.Pixel(tileCenterY)) {
				return false // 壁に遮られている
			}
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

// LightInfo は光源情報を保持する
type LightInfo struct {
	Darkness float64
	Color    color.RGBA
}

// calculateLightSourceDarkness は光源からの距離に応じた暗闇レベルと色を計算する
func calculateLightSourceDarkness(world w.World, tileX, tileY int) LightInfo {
	minDarkness := 1.0 // 完全に暗い状態からスタート

	// 加重平均用の累積値
	var totalR, totalG, totalB float64
	var totalWeight float64

	// 全ての光源をチェック
	world.Manager.Join(
		world.Components.LightSource,
		world.Components.GridElement,
	).Visit(ecs.Visit(func(lightEntity ecs.Entity) {
		lightSource := world.Components.LightSource.Get(lightEntity).(*gc.LightSource)

		// 無効な光源はスキップ
		if !lightSource.Enabled {
			return
		}

		lightGrid := world.Components.GridElement.Get(lightEntity).(*gc.GridElement)

		// 距離計算（タイル単位）
		distance := geometry.Distance(float64(tileX), float64(tileY), float64(lightGrid.X), float64(lightGrid.Y))

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
	}))

	// 加重平均を計算
	var finalR, finalG, finalB uint8
	if totalWeight > 0 {
		finalR = uint8(math.Min(255, totalR/totalWeight))
		finalG = uint8(math.Min(255, totalG/totalWeight))
		finalB = uint8(math.Min(255, totalB/totalWeight))
	}

	return LightInfo{
		Darkness: minDarkness,
		Color:    color.RGBA{R: finalR, G: finalG, B: finalB, A: 255},
	}
}

// 各タイル状態の暗闇の強さ
const (
	DarknessVisible    VisibleDarkness    = 0.15
	DarknessDark       DarkDarkness       = 0.9
	DarknessRemembered RememberedDarkness = 0.75
)

// isBlockedByWall は直接的な壁チェック
func isBlockedByWall(world w.World, x, y consts.Pixel) bool {
	fx, fy := float64(x), float64(y)

	// GridElement + BlockView のチェック（32x32タイル）
	tileX := int(fx / float64(consts.TileSize))
	tileY := int(fy / float64(consts.TileSize))

	// タイル座標でのチェック（1回のスキャンで済ませる）
	blocked := false
	world.Manager.Join(
		world.Components.GridElement,
		world.Components.BlockView,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		grid := world.Components.GridElement.Get(entity).(*gc.GridElement)
		if int(grid.X) == tileX && int(grid.Y) == tileY {
			blocked = true
		}
	}))

	return blocked
}
