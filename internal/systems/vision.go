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

	// 色の最大値を保持
	var maxR, maxG, maxB float64

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

			// 光の強さ（1.0 = 中心、0.0 = 範囲端）
			lightStrength := 1.0 - normalizedDistance

			// 色は最大値を採用して元の光源より明るくならないようにする
			colorR := float64(lightSource.Color.R) * lightStrength
			colorG := float64(lightSource.Color.G) * lightStrength
			colorB := float64(lightSource.Color.B) * lightStrength

			if colorR > maxR {
				maxR = colorR
			}
			if colorG > maxG {
				maxG = colorG
			}
			if colorB > maxB {
				maxB = colorB
			}
		}
	}))

	// 色をクランプ（0-255）
	finalR := uint8(math.Min(255, maxR))
	finalG := uint8(math.Min(255, maxG))
	finalB := uint8(math.Min(255, maxB))

	return LightInfo{
		Darkness: minDarkness,
		Color:    color.RGBA{R: finalR, G: finalG, B: finalB, A: 255},
	}
}

// renderDistanceBasedDarkness は距離に応じた段階的暗闇を描画する
func renderDistanceBasedDarkness(world w.World, screen *ebiten.Image, visibilityData map[string]TileVisibility) {
	// カメラ位置とスケールを取得
	var cameraPos gc.Position
	cameraScale := 1.0 // デフォルトスケール

	// カメラのGridElementから位置を取得
	world.Manager.Join(
		world.Components.Camera,
		world.Components.GridElement,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		cameraGridElement := world.Components.GridElement.Get(entity).(*gc.GridElement)
		cameraPos = gc.Position{
			X: gc.Pixel(int(cameraGridElement.X)*int(consts.TileSize) + int(consts.TileSize)/2),
			Y: gc.Pixel(int(cameraGridElement.Y)*int(consts.TileSize) + int(consts.TileSize)/2),
		}
		camera := world.Components.Camera.Get(entity).(*gc.Camera)
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
	leftEdge := int(cameraPos.X) - actualScreenWidth/2
	rightEdge := int(cameraPos.X) + actualScreenWidth/2
	topEdge := int(cameraPos.Y) - actualScreenHeight/2
	bottomEdge := int(cameraPos.Y) + actualScreenHeight/2

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
					// 視界内: 光源のみで暗さを決定（視界の暗闇は無視）
					lightInfo = calculateLightSourceDarkness(world, tileX, tileY)
				} else {
					// 視界範囲内だが見えないタイル: 完全に暗い（壁で遮られている）
					lightInfo = LightInfo{Darkness: 1.0, Color: color.RGBA{R: 0, G: 0, B: 0, A: 255}}
				}
			} else {
				// 視界範囲外
				if explored := world.Resources.Dungeon.ExploredTiles[gridElement]; explored {
					// 探索済み視界外タイル: 完全に暗い（記憶として表示）
					lightInfo = LightInfo{Darkness: 1.0, Color: color.RGBA{R: 0, G: 0, B: 0, A: 255}}
				} else {
					// 未探索タイル: 描画しない（完全に隠れる）
					continue
				}
			}

			// 暗闇レベルが0より大きい場合のみ描画
			if lightInfo.Darkness > 0.0 {
				// タイルの画面座標を計算（スケールを考慮）
				worldX := float64(tileX * int(consts.TileSize))
				worldY := float64(tileY * int(consts.TileSize))
				screenX := (worldX-float64(cameraPos.X))*cameraScale + float64(screenWidth)/2
				screenY := (worldY-float64(cameraPos.Y))*cameraScale + float64(screenHeight)/2

				// 暗闇レベルに応じた画像を描画
				drawDarknessAtLevelWithColor(screen, screenX, screenY, lightInfo.Darkness, lightInfo.Color, cameraScale, int(consts.TileSize))
			}
		}
	}
}

// initializeDarknessCache は段階的暗闃用の画像キャッシュを初期化する
func initializeDarknessCache(tileSize int) {
	// tileSizeが0以下の場合は初期化しない
	if tileSize <= 0 {
		return
	}

	// より多くの暗闇レベルの画像を作成（10段階）
	darknessCacheImages = make([]*ebiten.Image, 11)

	// 各暗闇レベルの画像を作成
	darknessCacheImages[0] = nil // 0.0: 暗闇なし

	// 0.1から1.0まで0.1刻みで10段階作成
	for i := 1; i <= 10; i++ {
		darkness := float64(i) * 0.1
		alpha := uint8(darkness * 255) // 透明度を0-255に変換

		darknessCacheImages[i] = ebiten.NewImage(tileSize, tileSize)

		// 最外側（80%以上の暗さ）は真っ黒、それ以外は暖色系
		if darkness >= 0.8 {
			// 最外側は純黒
			darknessCacheImages[i].Fill(color.RGBA{0, 0, 0, alpha})
		} else {
			// 内側は暖色系の暗闇（微かな茶色がかった暗闇）
			r := uint8(float64(alpha) * 0.15) // 透明度の15%の赤成分
			g := uint8(float64(alpha) * 0.10) // 透明度の10%の緑成分
			b := uint8(float64(alpha) * 0.05) // 透明度の5%の青成分
			darknessCacheImages[i].Fill(color.RGBA{r, g, b, alpha})
		}
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

	// 暗闇レベルを0-10の範囲にクランプ
	darknessLevel := int(math.Max(0, math.Min(10, darkness*10+0.5)))

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
		// 暗闇の強さに応じた透明度
		alpha := uint8(darkness * 255)

		// 光源の色を使った暗闇を作る
		// 明るい部分（darknessが小さい）はランタンの色が強く、暗い部分は黒に近づく
		lightStrength := 1.0 - darkness // 0.0(暗い) ~ 1.0(明るい)
		darknessColor := color.RGBA{
			R: uint8(float64(lightColor.R) * lightStrength * 0.6), // 明るい部分でランタンの色
			G: uint8(float64(lightColor.G) * lightStrength * 0.5),
			B: uint8(float64(lightColor.B) * lightStrength * 0.3),
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
