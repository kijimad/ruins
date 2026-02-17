// Package mapplanner はマップ生成機能を提供する
// 参考: https://bfnightly.bracketproductions.com
package mapplanner

import (
	"fmt"
	"math/rand/v2"
	"time"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/maptemplate"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/resources"
	w "github.com/kijimaD/ruins/internal/world"
)

// Coord はタイル座標を表す
type Coord struct {
	X int
	Y int
}

// PropsSpec はProps配置仕様を表す
type PropsSpec struct {
	Coord
	Name string // Prop名
}

// MetaPlan は階層のタイルを作る元になる概念の集合体
type MetaPlan struct {
	// 階層情報
	Level resources.Level
	// 部屋群。部屋は長方形の移動可能な空間のことをいう。
	// 部屋はタイルの集合体である
	Rooms []gc.Rect
	// 廊下群。廊下は部屋と部屋をつなぐ移動可能な空間のことをいう。
	// 廊下はタイルの集合体である
	Corridors [][]resources.TileIdx
	// 乱数生成器
	RNG *rand.Rand
	// 階層を構成するタイル群。長さはステージの大きさで決まる
	// 通行可能かを判定するための情報を保持している必要がある
	Tiles []raw.TileRaw
	// NextPortals は次の階へ進むポータルリスト
	NextPortals []Coord
	// EscapePortals は脱出用ポータルリスト
	EscapePortals []Coord
	// NPCs は配置予定のNPCリスト
	NPCs []NPCSpec
	// Items は配置予定のアイテムリスト
	Items []ItemSpec
	// Props は配置予定のPropsリスト
	Props []PropsSpec
	// SpawnPoints はプレイヤーのスポーン地点リスト
	SpawnPoints []maptemplate.SpawnPoint
	// RawMaster はタイル生成に使用するマスターデータ
	RawMaster *raw.Master
}

// IsSpawnableTile は指定タイル座標がスポーン可能かを返す
func (bm MetaPlan) IsSpawnableTile(_ w.World, tx gc.Tile, ty gc.Tile) bool {
	idx := bm.Level.XYTileIndex(tx, ty)
	tile := bm.Tiles[idx]
	// 通行不可なのでスポーン不可
	if tile.BlockPass {
		return false
	}

	// planning段階では、MetaPlan内の計画済みエンティティをチェック
	if bm.existPlannedEntityOnTile(int(tx), int(ty)) {
		return false
	}

	return true
}

// existPlannedEntityOnTile は指定座標に計画済みエンティティがあるかをチェック
func (bm MetaPlan) existPlannedEntityOnTile(x, y int) bool {
	for _, portal := range bm.NextPortals {
		if portal.X == x && portal.Y == y {
			return true
		}
	}

	for _, portal := range bm.EscapePortals {
		if portal.X == x && portal.Y == y {
			return true
		}
	}

	// NPCをチェック
	for _, npc := range bm.NPCs {
		if npc.X == x && npc.Y == y {
			return true
		}
	}

	// アイテムをチェック
	for _, item := range bm.Items {
		if item.X == x && item.Y == y {
			return true
		}
	}

	// Propsをチェック
	for _, prop := range bm.Props {
		if prop.X == x && prop.Y == y {
			return true
		}
	}

	return false
}

// UpTile は上にあるタイルを調べる
func (bm MetaPlan) UpTile(idx resources.TileIdx) raw.TileRaw {
	targetIdx := resources.TileIdx(int(idx) - int(bm.Level.TileWidth))
	if targetIdx < 0 {
		// 境界外（マップ外＝暗闇）として扱う
		return bm.GetTile(consts.TileNameVoid)
	}

	return bm.Tiles[targetIdx]
}

// DownTile は下にあるタイルを調べる
func (bm MetaPlan) DownTile(idx resources.TileIdx) raw.TileRaw {
	targetIdx := int(idx) + int(bm.Level.TileWidth)
	if targetIdx > len(bm.Tiles)-1 {
		// 境界外（マップ外＝暗闇）として扱う
		return bm.GetTile(consts.TileNameVoid)
	}

	return bm.Tiles[targetIdx]
}

// LeftTile は左にあるタイルを調べる
func (bm MetaPlan) LeftTile(idx resources.TileIdx) raw.TileRaw {
	x, y := bm.Level.XYTileCoord(idx)
	// 左端の場合は境界外（マップ外＝暗闇）
	if x == 0 {
		return bm.GetTile(consts.TileNameVoid)
	}

	// 左のタイルが同じ行であることを確認
	targetIdx := idx - 1
	_, targetY := bm.Level.XYTileCoord(targetIdx)
	if targetY != y {
		// 前の行にラップアラウンドしている（境界外）
		return bm.GetTile(consts.TileNameVoid)
	}

	return bm.Tiles[targetIdx]
}

// RightTile は右にあるタイルを調べる
func (bm MetaPlan) RightTile(idx resources.TileIdx) raw.TileRaw {
	x, y := bm.Level.XYTileCoord(idx)
	// 右端の場合は境界外（マップ外＝暗闇）
	if int(x) == int(bm.Level.TileWidth)-1 {
		return bm.GetTile(consts.TileNameVoid)
	}

	// 右のタイルが同じ行であることを確認
	targetIdx := idx + 1
	_, targetY := bm.Level.XYTileCoord(targetIdx)
	if targetY != y {
		// 次の行にラップアラウンドしている（境界外）
		return bm.GetTile(consts.TileNameVoid)
	}

	return bm.Tiles[targetIdx]
}

// AdjacentAnyFloor は直交・斜めを含む近傍8タイルに床があるか判定する
func (bm MetaPlan) AdjacentAnyFloor(idx resources.TileIdx) bool {
	x, y := bm.Level.XYTileCoord(idx)
	width := int(bm.Level.TileWidth)
	height := int(bm.Level.TileHeight)

	// 8方向の隣接タイル座標をチェック
	directions := [][2]int{
		{-1, -1}, {-1, 0}, {-1, 1}, // 上段
		{0, -1}, {0, 1}, // 中段（中心を除く）
		{1, -1}, {1, 0}, {1, 1}, // 下段
	}

	for _, dir := range directions {
		nx, ny := int(x)+dir[0], int(y)+dir[1]

		// 境界チェック
		if nx < 0 || nx >= width || ny < 0 || ny >= height {
			continue
		}

		neighborIdx := bm.Level.XYTileIndex(gc.Tile(nx), gc.Tile(ny))
		tile := bm.Tiles[neighborIdx]

		// 歩行可能
		if !tile.BlockPass {
			return true
		}
	}

	return false
}

// GetWallType は近傍パターンから適切な壁タイプを判定する
func (bm MetaPlan) GetWallType(idx resources.TileIdx) WallType {
	// 4方向の隣接タイルの床状況をチェック
	upFloor := bm.isFloorOrWarp(bm.UpTile(idx))
	downFloor := bm.isFloorOrWarp(bm.DownTile(idx))
	leftFloor := bm.isFloorOrWarp(bm.LeftTile(idx))
	rightFloor := bm.isFloorOrWarp(bm.RightTile(idx))

	// 単純なケース：一方向のみに床がある場合
	if singleWallType := bm.checkSingleDirectionWalls(upFloor, downFloor, leftFloor, rightFloor); singleWallType != WallTypeGeneric {
		return singleWallType
	}

	// 角のケース：2方向に床がある場合
	if cornerWallType := bm.checkCornerWalls(upFloor, downFloor, leftFloor, rightFloor); cornerWallType != WallTypeGeneric {
		return cornerWallType
	}

	// 複雑なパターンまたは判定不可の場合
	return WallTypeGeneric
}

// checkSingleDirectionWalls は単一方向に床がある場合の壁タイプを返す
func (bm MetaPlan) checkSingleDirectionWalls(upFloor, downFloor, leftFloor, rightFloor bool) WallType {
	if downFloor && !upFloor && !leftFloor && !rightFloor {
		return WallTypeTop // 下に床がある → 上壁
	}
	if upFloor && !downFloor && !leftFloor && !rightFloor {
		return WallTypeBottom // 上に床がある → 下壁
	}
	if rightFloor && !upFloor && !downFloor && !leftFloor {
		return WallTypeLeft // 右に床がある → 左壁
	}
	if leftFloor && !upFloor && !downFloor && !rightFloor {
		return WallTypeRight // 左に床がある → 右壁
	}
	return WallTypeGeneric
}

// checkCornerWalls は2方向に床がある場合の壁タイプを返す
func (bm MetaPlan) checkCornerWalls(upFloor, downFloor, leftFloor, rightFloor bool) WallType {
	if downFloor && rightFloor && !upFloor && !leftFloor {
		return WallTypeTopLeft // 下右に床 → 左上角
	}
	if downFloor && leftFloor && !upFloor && !rightFloor {
		return WallTypeTopRight // 下左に床 → 右上角
	}
	if upFloor && rightFloor && !downFloor && !leftFloor {
		return WallTypeBottomLeft // 上右に床 → 左下角
	}
	if upFloor && leftFloor && !downFloor && !rightFloor {
		return WallTypeBottomRight // 上左に床 → 右下角
	}
	return WallTypeGeneric
}

// isFloorOrWarp は移動可能タイルかを判定する（壁オートタイル用）
// BlockPassがtrueのタイルは床として扱わない
func (bm MetaPlan) isFloorOrWarp(tile raw.TileRaw) bool {
	return !tile.BlockPass
}

// PlannerChain は階層データMetaPlanに対して適用する生成ロジックを保持する構造体
type PlannerChain struct {
	Starter  *InitialMapPlanner
	Planners []MetaMapPlanner
	PlanData MetaPlan
}

// NewPlannerChain はシード値を指定してプランナーチェーンを作成する
// シードが0の場合はランダムなシードを生成する
func NewPlannerChain(width gc.Tile, height gc.Tile, seed uint64) *PlannerChain {
	tileCount := int(width) * int(height)
	tiles := make([]raw.TileRaw, tileCount)

	// シードが0の場合はランダムなシードを生成
	if seed == 0 {
		seed = uint64(time.Now().UnixNano())
	}

	return &PlannerChain{
		Starter:  nil,
		Planners: []MetaMapPlanner{},
		PlanData: MetaPlan{
			Level: resources.Level{
				TileWidth:  width,
				TileHeight: height,
			},
			Tiles:         tiles,
			Rooms:         []gc.Rect{},
			Corridors:     [][]resources.TileIdx{},
			RNG:           rand.New(rand.NewPCG(seed, seed+1)),
			NextPortals:   []Coord{},
			EscapePortals: []Coord{},
			NPCs:          []NPCSpec{},
			Items:         []ItemSpec{},
			Props:         []PropsSpec{},
		},
	}
}

// StartWith は初期プランナーを設定する
func (b *PlannerChain) StartWith(initialMapPlanner InitialMapPlanner) {
	b.Starter = &initialMapPlanner
}

// With はメタプランナーを追加する
func (b *PlannerChain) With(metaMapPlanner MetaMapPlanner) {
	b.Planners = append(b.Planners, metaMapPlanner)
}

// Plan はプランナーチェーンを実行してマップを生成する
func (b *PlannerChain) Plan() error {
	if b.Starter == nil {
		return fmt.Errorf("empty starter planner")
	}
	if err := (*b.Starter).PlanInitial(&b.PlanData); err != nil {
		return fmt.Errorf("PlanInitial failed: %w", err)
	}

	for _, meta := range b.Planners {
		if err := meta.PlanMeta(&b.PlanData); err != nil {
			return fmt.Errorf("PlanMeta failed: %w", err)
		}
	}
	return nil
}

// InitialMapPlanner は初期マップをプランするインターフェース
// タイルへの描画は行わず、構造体フィールドの値を初期化するだけ
type InitialMapPlanner interface {
	PlanInitial(*MetaPlan) error
}

// MetaMapPlanner はメタ情報をプランするインターフェース
type MetaMapPlanner interface {
	PlanMeta(*MetaPlan) error
}

// NewSmallRoomPlanner はシンプルな小部屋プランナーを作成する
func NewSmallRoomPlanner(width gc.Tile, height gc.Tile, seed uint64) (*PlannerChain, error) {
	chain := NewPlannerChain(width, height, seed)
	chain.StartWith(RectRoomPlanner{})
	chain.With(NewFillAll(consts.TileNameWall)) // 全体を壁で埋める
	chain.With(RoomDraw{})                      // 部屋を描画
	chain.With(LineCorridorPlanner{})           // 廊下を作成
	chain.With(ConvertIsolatedWalls{            // 床に隣接しない壁をvoidに変換
		ReplacementTile: consts.TileNameVoid,
	})

	return chain, nil
}

// NewBigRoomPlanner は大部屋プランナーを作成する
// ランダムにバリエーションを適用する統合版
func NewBigRoomPlanner(width gc.Tile, height gc.Tile, seed uint64) (*PlannerChain, error) {
	chain := NewPlannerChain(width, height, seed)
	chain.StartWith(BigRoomPlanner{})
	chain.With(NewFillAll(consts.TileNameWall)) // 全体を壁で埋める
	chain.With(BigRoomDraw{
		FloorTile: consts.TileNameFloor,
		WallTile:  consts.TileNameWall,
	}) // 大部屋を描画（バリエーション込み）
	chain.With(ConvertIsolatedWalls{ // 床に隣接しない壁をvoidに変換
		ReplacementTile: consts.TileNameVoid,
	})

	return chain, nil
}

// PlannerType はマップ生成の設定を表す構造体
type PlannerType struct {
	// プランナー名
	Name string
	// 敵をスポーンするか
	SpawnEnemies bool
	// アイテムをスポーンするか
	SpawnItems bool
	// ポータル位置を固定するか
	UseFixedPortalPos bool
	// アイテムテーブル名
	ItemTableName string
	// 敵テーブル名
	EnemyTableName string
	// プランナー関数
	PlannerFunc func(width gc.Tile, height gc.Tile, seed uint64) (*PlannerChain, error)
}

var (
	// PlannerTypeRandom はランダム選択用のプランナータイプ
	PlannerTypeRandom = PlannerType{
		Name:              "ランダム",
		SpawnEnemies:      true,
		SpawnItems:        true,
		UseFixedPortalPos: false,
		ItemTableName:     "通常",
		EnemyTableName:    "通常",
	}

	// PlannerTypeSmallRoom は小部屋ダンジョンのプランナータイプ
	PlannerTypeSmallRoom = PlannerType{
		Name:              "小部屋",
		SpawnEnemies:      true,
		SpawnItems:        true,
		UseFixedPortalPos: false,
		ItemTableName:     "通常",
		EnemyTableName:    "通常",
		PlannerFunc:       NewSmallRoomPlanner,
	}

	// PlannerTypeBigRoom は大部屋ダンジョンのプランナータイプ
	PlannerTypeBigRoom = PlannerType{
		Name:              "大部屋",
		SpawnEnemies:      true,
		SpawnItems:        true,
		UseFixedPortalPos: false,
		ItemTableName:     "通常",
		EnemyTableName:    "通常",
		PlannerFunc:       NewBigRoomPlanner,
	}

	// PlannerTypeCave は洞窟ダンジョンのプランナータイプ
	PlannerTypeCave = PlannerType{
		Name:              "洞窟",
		SpawnEnemies:      true,
		SpawnItems:        true,
		UseFixedPortalPos: false,
		ItemTableName:     "洞窟",
		EnemyTableName:    "洞窟",
		PlannerFunc:       NewCavePlanner,
	}

	// PlannerTypeRuins は廃墟ダンジョンのプランナータイプ
	PlannerTypeRuins = PlannerType{
		Name:              "廃墟",
		SpawnEnemies:      true,
		SpawnItems:        true,
		UseFixedPortalPos: false,
		ItemTableName:     "廃墟",
		EnemyTableName:    "廃墟",
		PlannerFunc:       NewRuinsPlanner,
	}

	// PlannerTypeForest は森ダンジョンのプランナータイプ
	PlannerTypeForest = PlannerType{
		Name:              "森",
		SpawnEnemies:      true,
		SpawnItems:        true,
		UseFixedPortalPos: false,
		ItemTableName:     "森",
		EnemyTableName:    "森",
		PlannerFunc:       NewForestPlanner,
	}

	// PlannerTypeTown は市街地のプランナータイプ
	PlannerTypeTown = PlannerType{
		Name:              "市街地",
		SpawnEnemies:      false, // 街では敵をスポーンしない
		SpawnItems:        false, // 街ではフィールドアイテムをスポーンしない
		UseFixedPortalPos: true,  // ポータル位置を固定
		ItemTableName:     "",    // 街ではアイテムをスポーンしないので空
		EnemyTableName:    "",    // 街では敵をスポーンしないので空
		PlannerFunc: func(_ gc.Tile, _ gc.Tile, seed uint64) (*PlannerChain, error) {
			return NewPlannerChainByTemplateType(TemplateTypeTownPlaza, seed)
		},
	}

	// PlannerTypeOfficeBuilding は事務所ビルのプランナータイプ
	PlannerTypeOfficeBuilding = PlannerType{
		Name:              "事務所ビル",
		SpawnEnemies:      false,
		SpawnItems:        false,
		UseFixedPortalPos: false,
		ItemTableName:     "",
		EnemyTableName:    "",
		PlannerFunc: func(_ gc.Tile, _ gc.Tile, seed uint64) (*PlannerChain, error) {
			return NewPlannerChainByTemplateType(TemplateTypeOfficeBuilding, seed)
		},
	}

	// PlannerTypeSmallTown は小さな町（複数の建物を配置）
	PlannerTypeSmallTown = PlannerType{
		Name:              "小さな町",
		SpawnEnemies:      false,
		SpawnItems:        false,
		UseFixedPortalPos: false,
		ItemTableName:     "",
		EnemyTableName:    "",
		PlannerFunc: func(_ gc.Tile, _ gc.Tile, seed uint64) (*PlannerChain, error) {
			return NewPlannerChainByTemplateType(TemplateTypeSmallTown, seed)
		},
	}

	// PlannerTypeTownPlaza は町の広場
	PlannerTypeTownPlaza = PlannerType{
		Name:              "広場",
		SpawnEnemies:      false,
		SpawnItems:        false,
		UseFixedPortalPos: true, // テンプレートでポータル位置が固定されている
		ItemTableName:     "",
		EnemyTableName:    "",
		PlannerFunc: func(_ gc.Tile, _ gc.Tile, seed uint64) (*PlannerChain, error) {
			return NewPlannerChainByTemplateType(TemplateTypeTownPlaza, seed)
		},
	}
)

// NewRandomPlanner はシード値を使用してランダムにプランナーを選択し作成する
func NewRandomPlanner(width gc.Tile, height gc.Tile, seed uint64) (*PlannerChain, error) {
	// シードが0の場合はランダムなシードを生成する。後続のビルダーに渡される
	if seed == 0 {
		seed = uint64(time.Now().UnixNano())
	}

	// シード値からランダムソースを作成（ビルダー選択用）
	rng := rand.New(rand.NewPCG(seed, 0))

	// ランダム選択対象のプランナータイプ（街は除外）
	candidateTypes := []PlannerType{
		PlannerTypeSmallRoom,
		PlannerTypeBigRoom,
		PlannerTypeCave,
		PlannerTypeRuins,
		PlannerTypeForest,
	}

	// ランダムに選択
	selectedType := candidateTypes[rng.IntN(len(candidateTypes))]

	chain, err := selectedType.PlannerFunc(width, height, seed)
	if err != nil {
		return nil, fmt.Errorf("ランダムプランナー選択エラー: %w", err)
	}
	return chain, nil
}

// GetTile はタイルを生成する
// TODO: エラーを潰しているだけなので直す
func (bm *MetaPlan) GetTile(name string) raw.TileRaw {
	if bm.RawMaster == nil {
		panic("RawMasterが設定されていない。TOMLからのタイル生成が必須である")
	}
	tile, err := bm.RawMaster.GetTile(name)
	if err != nil {
		panic(fmt.Sprintf("タイル生成エラー: %v", err))
	}
	return tile
}

// GetPlayerStartPosition はプレイヤーの開始位置を取得する
// ポータルへの到達性も確認し、到達可能な位置を返す
func (bm *MetaPlan) GetPlayerStartPosition() (Coord, error) {
	// SpawnPointsが設定されていればそれを使用（テンプレートマップ用）
	if len(bm.SpawnPoints) > 0 {
		return Coord{X: bm.SpawnPoints[0].X, Y: bm.SpawnPoints[0].Y}, nil
	}

	// プロシージャルマップ用: 自動的に歩行可能でポータルに到達可能な位置を探す
	width := int(bm.Level.TileWidth)
	height := int(bm.Level.TileHeight)

	// 到達性チェック用のPathFinderを作成
	pf := NewPathFinder(bm)

	// 候補位置を試す（中央から外側へ）
	attempts := []Coord{
		{width / 2, height / 2},         // 中央
		{width / 4, height / 4},         // 左上寄り
		{3 * width / 4, height / 4},     // 右上寄り
		{width / 4, 3 * height / 4},     // 左下寄り
		{3 * width / 4, 3 * height / 4}, // 右下寄り
	}

	for _, pos := range attempts {
		if bm.isValidSpawnPosition(pf, pos.X, pos.Y) {
			return pos, nil
		}
	}

	// 見つからない場合は全体をスキャン
	for _i, tile := range bm.Tiles {
		if !tile.BlockPass {
			i := resources.TileIdx(_i)
			x, y := bm.Level.XYTileCoord(i)
			if bm.isValidSpawnPosition(pf, int(x), int(y)) {
				return Coord{X: int(x), Y: int(y)}, nil
			}
		}
	}

	return Coord{}, fmt.Errorf("ポータルに到達可能な歩行可能タイルが見つかりません")
}

// isValidSpawnPosition は指定位置がスポーン可能かつポータルに到達可能かを判定する
func (bm *MetaPlan) isValidSpawnPosition(pf *PathFinder, x, y int) bool {
	idx := bm.Level.XYTileIndex(gc.Tile(x), gc.Tile(y))
	if int(idx) >= len(bm.Tiles) || bm.Tiles[idx].BlockPass {
		return false
	}

	// NextPortalsへの到達性をチェック
	for _, portal := range bm.NextPortals {
		if !pf.IsReachable(x, y, portal.X, portal.Y) {
			return false
		}
	}

	// EscapePortalsへの到達性をチェック
	for _, portal := range bm.EscapePortals {
		if !pf.IsReachable(x, y, portal.X, portal.Y) {
			return false
		}
	}

	return true
}
