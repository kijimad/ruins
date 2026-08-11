package systems

import (
	"image/color"
	"math"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/geometry"
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// VisionSystem はタイルごとの視界を計算するUpdaterシステム。
// 計算結果の光源情報は VisionState シングルトンに書き込み、描画側はそこから参照する。
type VisionSystem struct {
	// プレイヤー位置キャッシュ（タイル移動ごとに更新）
	lastPlayer    consts.Coord[consts.WorldPixel]
	isInitialized bool
}

// NewVisionSystem はVisionSystemを初期化する
func NewVisionSystem() *VisionSystem {
	return &VisionSystem{}
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

	if query.GetDungeon(world) == nil {
		return nil
	}
	field := query.GetCurrentStageField(world)
	if field == nil {
		return nil
	}
	vs := query.GetVisionState(world)

	// 移動ごとの視界更新判定（移動ごとに更新）
	const updateThreshold = int(consts.TileSize)
	needsUpdate := !sys.isInitialized ||
		geometry.Abs(int(playerPos.X-sys.lastPlayer.X)) >= updateThreshold ||
		geometry.Abs(int(playerPos.Y-sys.lastPlayer.Y)) >= updateThreshold

	// 外部から要求された視界更新を消費する
	if vs.ConsumePendingUpdate() {
		needsUpdate = true
	}

	if !needsUpdate {
		return nil
	}

	// 視界遮断タイルのインデックスを構築する
	blockViewIndex := buildBlockViewIndex(world)

	// タイルの可視性マップを更新
	visionRadius := consts.WorldPixel(consts.VisionRadiusTiles) * consts.TileSize
	visibilityData := calculateTileVisibilityWithDistance(playerPos, visionRadius, blockViewIndex)

	// 光源情報を更新前にクリアする
	vs.LightSourceCache = make(map[gc.GridElement]gc.LightInfo)

	// 視界内タイルの光源情報を計算し、探索済みマークを行う。
	// マップ外座標はデータに含めない
	ambient := ambientLight(world)
	visibleTiles := make(map[gc.GridElement]bool)
	for _, tileData := range visibilityData {
		if !tileData.Visible {
			continue
		}
		gridElement := gc.GridElement{Coord: consts.Coord[consts.Tile]{X: consts.Tile(tileData.Col), Y: consts.Tile(tileData.Row)}}
		if !isInMapBounds(gridElement, field.Level) {
			continue
		}

		info := calculateLightSourceDarkness(world, consts.Coord[int]{X: tileData.Col, Y: tileData.Row}, blockViewIndex, ambient)
		// 明るさが閾値未満なら見えない。視界を光の届く範囲へ寄せる。
		// 見えないタイルは記憶側へ回るので、暗所の敵やアイテムは自然に隠れる
		if 1.0-float64(info.Darkness) < visibilityThreshold {
			continue
		}
		vs.LightSourceCache[gridElement] = info
		field.ExploredTiles[gridElement] = true
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
	field := query.GetCurrentStageField(world)
	vs := query.GetVisionState(world)

	// 現在見えているタイルを設定する。明るさは光源の加算結果 li.Darkness を使う。
	// ただし記憶タイルより暗くはしない。今見えているタイルが、ただ記憶しているだけの
	// タイルより暗く見えると、光の輪の縁だけが黒いリングになって不自然になるのを防ぐ
	for grid := range vs.VisibleTiles {
		visible := TileRenderVisible{Darkness: DarknessVisible}
		if li, ok := lights[grid]; ok {
			visible.Darkness = VisibleDarkness(math.Min(li.Darkness, float64(DarknessRemembered)))
			// 完全な暗黒には色を乗せない。描画側も無視するが、意図を明示する
			if li.Darkness < 1.0 {
				visible.LightColor = li.Color
			}
		}
		result[grid] = visible
	}

	// 視界外だが記憶済みのタイルを設定する
	if field != nil {
		for grid := range field.ExploredTiles {
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

// calculateTileVisibilityWithDistance はレイキャストでタイルごとの可視性と距離を計算する。
// 結果はキーで引かず順に走査するだけなので、マップでなくスライスで返す。座標は各要素の
// Row・Col が持つ。タイルごとの文字列キー生成を避け、視界内タイル数ぶんの alloc を無くす。
func calculateTileVisibilityWithDistance(playerPos consts.Coord[consts.WorldPixel], radius consts.WorldPixel, blockIndex map[gc.GridElement]bool) []TileVisibility {
	// プレイヤーの位置からタイル座標を計算
	playerTileX := int(playerPos.X) / int(consts.TileSize)
	playerTileY := int(playerPos.Y) / int(consts.TileSize)

	// 視界範囲を分割して段階的処理（視界範囲最適化）
	maxTileDistance := int(radius)/int(consts.TileSize) + 2

	// 走査する正方形のタイル数を上限に事前確保し、追加時の再確保を避ける
	side := 2*maxTileDistance + 1
	visibility := make([]TileVisibility, 0, side*side)

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

			// 視界範囲内のタイルのみ処理
			if distanceSquared <= radiusSquared {
				visible := isTileVisibleByRaycast(playerPos, tileCenter, blockIndex)

				visibility = append(visibility, TileVisibility{
					Row:     tileY,
					Col:     tileX,
					Visible: visible,
				})
			}
			// 視界外のタイルは処理しない（最適化）
		}
	}

	return visibility
}

// isTileVisibleByRaycast はタイルベース視界判定
func isTileVisibleByRaycast(player, target consts.Coord[consts.WorldPixel], blockIndex map[gc.GridElement]bool) bool {
	// タイル座標に変換
	playerTileX := int(player.X / consts.TileSize)
	playerTileY := int(player.Y / consts.TileSize)
	targetTileX := int(target.X / consts.TileSize)
	targetTileY := int(target.Y / consts.TileSize)

	// 同じタイルまたは隣接タイルは常に見える
	if geometry.Abs(targetTileX-playerTileX) <= 1 && geometry.Abs(targetTileY-playerTileY) <= 1 {
		return true
	}

	// ブレゼンハムのライン描画アルゴリズムでタイルベースの視線判定
	return bresenhamLineOfSight(playerTileX, playerTileY, targetTileX, targetTileY, blockIndex)
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

// ambientLight はフィールドの環境光の明るさを返す。屋内は微小、地上は時間帯の日照で決まる。
// 昼の屋外は全体が明るく松明が要らず、地下や深夜は松明の届く範囲だけが見える。
func ambientLight(world w.World) float64 {
	if query.IsOnOverworld(world) {
		return overworldDaylight(query.GetGameTime(world).GetTimeOfDay())
	}
	return dungeonAmbient
}

// overworldDaylight は地上の時間帯ごとの日照の明るさを返す。昼が最も明るく深夜が最も暗い。
func overworldDaylight(t gc.TimeOfDay) float64 {
	switch t {
	case gc.TimeDawn:
		return 0.40
	case gc.TimeMorning:
		return 0.72
	case gc.TimeDay:
		return 0.95
	case gc.TimeEvening:
		return 0.38
	case gc.TimeNight:
		return 0.14
	case gc.TimeMidnight:
		return 0.06
	default:
		return 0.95
	}
}

// calculateLightSourceDarkness はタイルの明るさを光源の加算合成で求め、暗さ=1-明るさで返す。
// 各光源は逆二乗ベースで減衰し、半径の外縁で滑らかに0へ落ちる。複数光源は加算し、
// 環境光 ambient を下駄として足す。壁で視線が遮られた光源は寄与しない。壁の裏へ光が漏れない。
// 色は各光源の寄与で加重平均する。
func calculateLightSourceDarkness(world w.World, tile consts.Coord[int], blockIndex map[gc.GridElement]bool, ambient float64) gc.LightInfo {
	brightness := ambient

	// 色は光源の寄与で加重平均する
	var totalR, totalG, totalB float64
	var totalWeight float64

	// 全ての光源をチェック。退避中ステージの光源は現ステージを照らさない
	lightQuery := query.ActiveFilter2[gc.LightSource, gc.GridElement](world).Query()
	for lightQuery.Next() {
		lightEntity := lightQuery.Entity()
		lightSource := world.Components.LightSource.Get(lightEntity)

		if !lightSource.Enabled {
			continue
		}

		lightGrid := world.Components.GridElement.Get(lightEntity)
		distance := geometry.Distance(float64(tile.X), float64(tile.Y), float64(lightGrid.X), float64(lightGrid.Y))
		if distance > float64(lightSource.Radius) {
			continue
		}
		// 光源からタイルへの視線が壁で遮られているなら光は届かない。壁の裏へ漏らさない。
		if !bresenhamLineOfSight(int(lightGrid.X), int(lightGrid.Y), tile.X, tile.Y, blockIndex) {
			continue
		}

		// 中心1タイルまでは最大の明るさにする
		if distance < 1.0 {
			distance = 1.0
		}
		nd := distance / float64(lightSource.Radius)

		// 逆二乗の減衰に、半径の外縁で0へ落ちる窓関数を掛ける。
		// window=1-nd^4 が縁を滑らかに閉じ、1/(1+K*nd^2) が逆二乗の効きを与える
		window := 1.0 - nd*nd*nd*nd
		if window < 0 {
			window = 0
		}
		atten := window * window / (1.0 + lightFalloffK*nd*nd)

		// 加算合成。重なるほど明るい
		brightness += atten

		// 色は寄与 atten で加重する
		totalR += float64(lightSource.Color.R) * atten
		totalG += float64(lightSource.Color.G) * atten
		totalB += float64(lightSource.Color.B) * atten
		totalWeight += atten
	}

	brightness = math.Max(0, math.Min(1, brightness))

	col := color.RGBA{A: 255}
	if totalWeight > 0 {
		col.R = uint8(math.Min(255, totalR/totalWeight))
		col.G = uint8(math.Min(255, totalG/totalWeight))
		col.B = uint8(math.Min(255, totalB/totalWeight))
	}

	return gc.LightInfo{
		Darkness: 1.0 - brightness,
		Color:    col,
	}
}

// 各タイル状態の暗闇の強さ
const (
	DarknessVisible    VisibleDarkness    = 0.15
	DarknessRemembered RememberedDarkness = 0.75
)

// ライティングの調整値
const (
	// dungeonAmbient は屋内の環境光。松明が無いと見えないくらい暗い
	dungeonAmbient = 0.06
	// visibilityThreshold はこの明るさ未満のタイルを見えないとみなす境界。視界を光の届く範囲へ寄せる
	visibilityThreshold = 0.10
	// lightFalloffK は逆二乗減衰の効き。大きいほど光源から急に暗くなる
	lightFalloffK = 3.0
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
