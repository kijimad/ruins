package systems

import (
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	w "github.com/kijimaD/ruins/internal/world"
	ecs "github.com/x-hgg-x/goecs/v2"
)

var (
	// 段階的な暗闇用の画像キャッシュ（透明度レベルを増加）
	darknessCacheImages []*ebiten.Image

	// プレイヤー位置キャッシュ（4px移動ごとに更新）
	playerPositionCache struct {
		lastPlayerX    gc.Pixel
		lastPlayerY    gc.Pixel
		visibilityData map[string]TileVisibility
		isInitialized  bool
	}

	// レイキャスト結果のキャッシュ
	raycastCache = make(map[raycastCacheKey]bool)

	// 光源色ごとの暗闇画像キャッシュ
	coloredDarknessCache = make(map[coloredDarknessCacheKey]*ebiten.Image)

	// 光源情報キャッシュ（タイル座標 -> 光源情報）
	lightSourceCache = make(map[gc.GridElement]LightInfo)
)

// raycastCacheKey はレイキャスト結果のキャッシュキー
type raycastCacheKey struct {
	PlayerX int
	PlayerY int
	TargetX int
	TargetY int
}

// coloredDarknessCacheKey は光源色ごとの暗闇画像のキャッシュキー
type coloredDarknessCacheKey struct {
	R             uint8
	G             uint8
	B             uint8
	DarknessLevel int
}

// abs は絶対値を返す
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// ClearVisionCaches は全ての視界関連キャッシュをクリアする（階移動時などに使用）
func ClearVisionCaches() {
	// プレイヤー位置キャッシュをクリア
	playerPositionCache.isInitialized = false
	playerPositionCache.visibilityData = nil

	// レイキャストキャッシュをクリア
	raycastCache = make(map[raycastCacheKey]bool)

	// 光源色キャッシュをクリア
	coloredDarknessCache = make(map[coloredDarknessCacheKey]*ebiten.Image)

	// 光源情報キャッシュをクリア
	lightSourceCache = make(map[gc.GridElement]LightInfo)
}

// VisionSystem はタイルごとの視界を管理する（暗闇描画はRenderSpriteSystemで行う）
type VisionSystem struct{}

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
		X: gc.Pixel(int(playerGridElement.X)*int(consts.TileSize) + int(consts.TileSize)/2),
		Y: gc.Pixel(int(playerGridElement.Y)*int(consts.TileSize) + int(consts.TileSize)/2),
	}

	// 移動ごとの視界更新判定（移動ごとに更新）
	const updateThreshold = int(consts.TileSize)
	needsUpdate := !playerPositionCache.isInitialized ||
		abs(int(playerPos.X-playerPositionCache.lastPlayerX)) >= updateThreshold ||
		abs(int(playerPos.Y-playerPositionCache.lastPlayerY)) >= updateThreshold

	// 外部から設定された視界更新フラグをチェックする(ドア開閉時など)
	if world.Resources.Dungeon != nil && world.Resources.Dungeon.NeedsForceUpdate {
		needsUpdate = true
		world.Resources.Dungeon.NeedsForceUpdate = false
	}

	if needsUpdate {
		// タイルの可視性マップを更新
		visionRadius := gc.Pixel(consts.VisionRadiusTiles * consts.TileSize)
		visibilityData := calculateTileVisibilityWithDistance(world, playerPos.X, playerPos.Y, visionRadius)

		// 光源情報キャッシュをクリア（更新前）
		lightSourceCache = make(map[gc.GridElement]LightInfo)

		// 視界内かつ光源があるタイルを探索済みとしてマーク
		for _, tileData := range visibilityData {
			if tileData.Visible {
				// 光源チェック
				lightInfo := calculateLightSourceDarkness(world, tileData.Col, tileData.Row)
				gridElement := gc.GridElement{X: gc.Tile(tileData.Col), Y: gc.Tile(tileData.Row)}

				// 光源情報をキャッシュに保存
				lightSourceCache[gridElement] = lightInfo

				// 光源範囲内（暗闇レベルが1.0未満）のみ探索済み
				if lightInfo.Darkness < 1.0 {
					world.Resources.Dungeon.ExploredTiles[gridElement] = true
				}
			}
		}

		// キャッシュ更新
		playerPositionCache.lastPlayerX = playerPos.X
		playerPositionCache.lastPlayerY = playerPos.Y
		playerPositionCache.visibilityData = visibilityData
		playerPositionCache.isInitialized = true
	}
	// 距離に応じた段階的暗闇の描画はRenderSpriteSystemで行う
	return nil
}

// TileVisibility はタイルの可視性を表す
type TileVisibility struct {
	Row      int
	Col      int
	Visible  bool
	Distance float64
	Darkness float64 // 0.0（明るい）から 1.0（真っ暗）
}

// calculateTileVisibilityWithDistance はレイキャストでタイルごとの可視性と距離を計算する
func calculateTileVisibilityWithDistance(world w.World, playerX, playerY, radius gc.Pixel) map[string]TileVisibility {
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
			if abs(dx) > maxTileDistance || abs(dy) > maxTileDistance {
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
				distanceToTile := math.Sqrt(distanceSquared)

				// Dark Days Ahead風の統一されたタイルベース視界判定
				visible := isTileVisibleByRaycast(world, float64(playerX), float64(playerY), tileCenterX, tileCenterY)

				// 距離に応じた暗闇の計算
				darkness := calculateDarknessByDistance(distanceToTile, float64(radius))

				visibilityMap[tileKey] = TileVisibility{
					Row:      tileY,
					Col:      tileX,
					Visible:  visible,
					Distance: distanceToTile,
					Darkness: darkness,
				}
			}
			// 視界外のタイルは処理しない（最適化）
		}
	}

	return visibilityMap
}

// isTileVisibleByRaycast はタイルベース視界判定
func isTileVisibleByRaycast(world w.World, playerX, playerY, targetX, targetY float64) bool {
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
	if result, exists := raycastCache[cacheKey]; exists {
		return result
	}

	// タイル座標に変換
	playerTileX := int(playerX / float64(consts.TileSize))
	playerTileY := int(playerY / float64(consts.TileSize))
	targetTileX := int(targetX / float64(consts.TileSize))
	targetTileY := int(targetY / float64(consts.TileSize))

	// 同じタイルまたは隣接タイルは常に見える
	if abs(targetTileX-playerTileX) <= 1 && abs(targetTileY-playerTileY) <= 1 {
		raycastCache[cacheKey] = true
		return true
	}

	// ブレゼンハムのライン描画アルゴリズムでタイルベースの視線判定
	result := bresenhamLineOfSight(world, playerTileX, playerTileY, targetTileX, targetTileY)

	// 結果をキャッシュ
	if len(raycastCache) < 15000 {
		raycastCache[cacheKey] = result
	}

	return result
}

// bresenhamLineOfSight はブレゼンハムアルゴリズムを使用したタイルベース視線判定
func bresenhamLineOfSight(world w.World, x0, y0, x1, y1 int) bool {
	dx := abs(x1 - x0)
	dy := abs(y1 - y0)

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
			if isBlockedByWall(world, gc.Pixel(tileCenterX), gc.Pixel(tileCenterY)) {
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

// calculateDarknessByDistance は距離に応じた暗闇レベルを計算する
func calculateDarknessByDistance(distance, maxRadius float64) float64 {
	if distance <= 0 {
		return 0.0 // プレイヤーの位置は完全に明るい
	}

	// 距離の正規化 (0.0-1.0)
	normalizedDistance := distance / maxRadius
	if normalizedDistance >= 1.0 {
		return 0.95 // 最遠距離でも完全に真っ暗にはしない
	}

	// 滑らかな二次カーブによる減衰（中心が明るく、外側に向かって滑らかに暗くなる）
	// 0.2までは完全に明るい（コア照明領域）
	if normalizedDistance <= 0.2 {
		return 0.0
	}

	// 0.2から1.0にかけて滑らかに暗くなる
	// 二次関数: y = ((x-0.2) / 0.8)^1.5 * 0.95
	adjustedDistance := (normalizedDistance - 0.2) / 0.8
	darkness := math.Pow(adjustedDistance, 1.5) * 0.95

	return darkness
}

// LightInfo は光源情報を保持する
type LightInfo struct {
	Darkness float64
	Color    color.RGBA
}

// calculateLightSourceDarkness は光源からの距離に応じた暗闇レベルと色を計算する
// getCachedLightInfo はキャッシュから光源情報を取得する
func getCachedLightInfo(world w.World, tileX, tileY int) LightInfo {
	gridElement := gc.GridElement{X: gc.Tile(tileX), Y: gc.Tile(tileY)}
	if info, exists := lightSourceCache[gridElement]; exists {
		return info
	}
	// キャッシュになければ計算
	return calculateLightSourceDarkness(world, tileX, tileY)
}

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
		dx := float64(tileX - int(lightGrid.X))
		dy := float64(tileY - int(lightGrid.Y))
		distance := math.Sqrt(dx*dx + dy*dy)

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

			// 光の強さを重みとして使用（近いほど強い）
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

// renderDistanceBasedDarkness は距離に応じた段階的暗闇を描画する
func renderDistanceBasedDarkness(world w.World, screen *ebiten.Image, visibilityData map[string]TileVisibility) {
	// カメラ位置とスケールを取得
	var cameraX, cameraY float64
	cameraScale := 1.0 // デフォルトスケール

	// カメラコンポーネントから位置を取得
	world.Manager.Join(
		world.Components.Camera,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		camera := world.Components.Camera.Get(entity).(*gc.Camera)
		cameraX = camera.X
		cameraY = camera.Y
		cameraScale = camera.Scale
	}))

	// 段階的暗闃用の画像を初期化（キャッシュ）
	if len(darknessCacheImages) == 0 {
		initializeDarknessCache(int(consts.TileSize))
	}

	// 画面上に表示されるタイル範囲を計算
	screenWidth := world.Resources.ScreenDimensions.Width
	screenHeight := world.Resources.ScreenDimensions.Height

	// スケールを考慮した実際の表示範囲を計算
	actualScreenWidth := int(float64(screenWidth) / cameraScale)
	actualScreenHeight := int(float64(screenHeight) / cameraScale)

	// カメラオフセットを考慮した画面範囲
	leftEdge := int(cameraX) - actualScreenWidth/2
	rightEdge := int(cameraX) + actualScreenWidth/2
	topEdge := int(cameraY) - actualScreenHeight/2
	bottomEdge := int(cameraY) + actualScreenHeight/2

	// タイル範囲に変換
	startTileX := leftEdge/int(consts.TileSize) - 1
	endTileX := rightEdge/int(consts.TileSize) + 1
	startTileY := topEdge/int(consts.TileSize) - 1
	endTileY := bottomEdge/int(consts.TileSize) + 1

	// 距離に応じた段階的暗闇を描画
	for tileX := startTileX; tileX <= endTileX; tileX++ {
		for tileY := startTileY; tileY <= endTileY; tileY++ {
			tileKey := fmt.Sprintf("%d,%d", tileX, tileY)
			gridElement := gc.GridElement{X: gc.Tile(tileX), Y: gc.Tile(tileY)}

			var lightInfo LightInfo

			// 視界データをチェック
			if tileData, exists := visibilityData[tileKey]; exists {
				if tileData.Visible {
					// 視界内: 光源の有無で効果を決定
					info := calculateLightSourceDarkness(world, tileX, tileY)
					if info.Darkness < 1.0 {
						// 光源範囲内: 光源の色味を薄く表示
						lightInfo = LightInfo{Darkness: 0.15, Color: info.Color}
					} else {
						// 光源範囲外: 薄い黒フィルター
						lightInfo = LightInfo{Darkness: 0.3, Color: color.RGBA{R: 0, G: 0, B: 0, A: 255}}
					}
				} else {
					// 視界範囲内だが見えないタイル（壁で遮蔽）: 完全に暗い
					lightInfo = LightInfo{Darkness: 1.0, Color: color.RGBA{R: 0, G: 0, B: 0, A: 255}}
				}
			} else {
				// 視界範囲外
				if !world.Resources.Dungeon.ExploredTiles[gridElement] {
					// 未探索タイル: 描画しない
					continue
				}
				// 探索済み視界外タイル: 完全に暗い
				lightInfo = LightInfo{Darkness: 1.0, Color: color.RGBA{R: 0, G: 0, B: 0, A: 255}}
			}

			// 暗闇レベルが0より大きい場合のみ描画
			if lightInfo.Darkness > 0.0 {
				worldX := float64(tileX * int(consts.TileSize))
				worldY := float64(tileY * int(consts.TileSize))
				screenX := (worldX-cameraX)*cameraScale + float64(screenWidth)/2
				screenY := (worldY-cameraY)*cameraScale + float64(screenHeight)/2

				drawDarknessAtLevelWithColor(screen, screenX, screenY, lightInfo.Darkness, lightInfo.Color, cameraScale, int(consts.TileSize))
			}
		}
	}
}

// DarknessLevels は暗闇の段階数を定義する
// 少ない段階数のほうが見た目が自然になる
const DarknessLevels = 4

// initializeDarknessCache は段階的暗闇用の画像キャッシュを初期化する
func initializeDarknessCache(tileSize int) {
	// tileSizeが0以下の場合は初期化しない
	if tileSize <= 0 {
		return
	}

	// 暗闇レベルの画像を作成（4段階: 0%, 33%, 66%, 100%）
	darknessCacheImages = make([]*ebiten.Image, DarknessLevels+1)

	// 各暗闇レベルの画像を作成
	darknessCacheImages[0] = nil // 0: 暗闇なし

	for i := 1; i <= DarknessLevels; i++ {
		darkness := float64(i) / float64(DarknessLevels)
		alpha := uint8(darkness * 255) // 透明度を0-255に変換

		darknessCacheImages[i] = ebiten.NewImage(tileSize, tileSize)
		darknessCacheImages[i].Fill(color.RGBA{0, 0, 0, alpha})
	}
}

// GetCurrentVisibilityData は現在の視界データを返す（レンダリング用）
func GetCurrentVisibilityData() map[string]TileVisibility {
	if playerPositionCache.isInitialized {
		return playerPositionCache.visibilityData
	}
	return nil
}

// isBlockedByWall は直接的な壁チェック
func isBlockedByWall(world w.World, x, y gc.Pixel) bool {
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

// drawDarknessAtLevelWithColor は光源の色を考慮した暗闇を描画する
func drawDarknessAtLevelWithColor(screen *ebiten.Image, x, y, darkness float64, lightColor color.RGBA, scale float64, tileSize int) {
	if darkness <= 0.0 {
		return // 暗闇なし
	}

	// 暗闇レベルを離散化（DarknessLevels段階）
	// 連続値を離散化することでグラデーションの段階を減らす
	darknessLevel := int(math.Ceil(darkness * float64(DarknessLevels)))
	if darknessLevel > DarknessLevels {
		darknessLevel = DarknessLevels
	}
	if darknessLevel < 1 {
		darknessLevel = 1
	}

	// 離散化した暗闇値を計算
	quantizedDarkness := float64(darknessLevel) / float64(DarknessLevels)

	// キャッシュキーを生成
	cacheKey := coloredDarknessCacheKey{
		R:             lightColor.R,
		G:             lightColor.G,
		B:             lightColor.B,
		DarknessLevel: darknessLevel,
	}

	// キャッシュから画像を取得、なければ生成
	darknessImg, exists := coloredDarknessCache[cacheKey]
	if !exists {
		// 離散化した暗闇の強さに応じた透明度
		alpha := uint8(quantizedDarkness * 255)

		// 光源の色味を控えめに反映した暗闇オーバーレイ
		colorStrength := 0.1
		darknessColor := color.RGBA{
			R: uint8(float64(lightColor.R) * colorStrength),
			G: uint8(float64(lightColor.G) * colorStrength),
			B: uint8(float64(lightColor.B) * colorStrength),
			A: alpha,
		}

		// 暗闇画像を生成してキャッシュ
		darknessImg = ebiten.NewImage(tileSize, tileSize)
		darknessImg.Fill(darknessColor)

		// キャッシュサイズ制限（メモリ節約）
		if len(coloredDarknessCache) < 1000 {
			coloredDarknessCache[cacheKey] = darknessImg
		}
	}

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(x, y)
	screen.DrawImage(darknessImg, op)
}
