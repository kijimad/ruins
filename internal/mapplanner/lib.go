// Package mapplanner はマップ生成機能を提供する
// 参考: https://bfnightly.bracketproductions.com
package mapplanner

import (
	"fmt"
	"math/rand/v2"
	"reflect"
	"time"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/maptemplate"
	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/kijimaD/ruins/internal/raw"
	w "github.com/kijimaD/ruins/internal/world"
)

// PropsSpec はProps配置仕様を表す
type PropsSpec struct {
	consts.Coord[int]
	Name string // Prop名
}

// DoorSpec はドア配置仕様を表す
type DoorSpec struct {
	consts.Coord[int]
	Orientation gc.DoorOrientation
}

// MetaPlan は階層のタイルを作る元になる概念の集合体
type MetaPlan struct {
	// 階層情報
	Level gc.Level
	// 部屋群。部屋は長方形の移動可能な空間のことをいう。
	// 部屋はタイルの集合体である
	Rooms []gc.Rect
	// 廊下群。廊下は部屋と部屋をつなぐ移動可能な空間のことをいう。
	// 廊下はタイルの集合体である
	Corridors [][]gc.TileIdx
	// 乱数生成器
	RNG *rand.Rand
	// 階層を構成するタイル群。長さはステージの大きさで決まる
	// 通行可能かを判定するための情報を保持している必要がある
	Tiles []oapi.Tile
	// NextPortals は次の階へ進むポータルリスト
	NextPortals []consts.Coord[int]
	// EscapePortals は脱出用ポータルリスト
	EscapePortals []consts.Coord[int]
	// NPCs は配置予定のNPCリスト
	NPCs []NPCSpec
	// Items は配置予定のアイテムリスト
	Items []ItemSpec
	// Props は配置予定のPropsリスト
	Props []PropsSpec
	// Doors は配置予定のドアリスト
	Doors []DoorSpec
	// SpawnPoints はプレイヤーのスポーン地点リスト
	SpawnPoints []maptemplate.SpawnPoint
	// RawMaster はタイル生成に使用するマスターデータ
	RawMaster *oapi.Raws
}

// IsSpawnableTile は指定タイル座標がスポーン可能かを返す
func (bm MetaPlan) IsSpawnableTile(_ w.World, tx consts.Tile, ty consts.Tile) bool {
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

	// ドアをチェック
	for _, door := range bm.Doors {
		if door.X == x && door.Y == y {
			return true
		}
	}

	return false
}

// UpTile は上にあるタイルを調べる
func (bm MetaPlan) UpTile(idx gc.TileIdx) oapi.Tile {
	targetIdx := gc.TileIdx(int(idx) - int(bm.Level.TileWidth))
	if targetIdx < 0 {
		// 境界外（マップ外＝暗闇）として扱う
		return bm.GetTile(consts.TileNameVoid)
	}

	return bm.Tiles[targetIdx]
}

// DownTile は下にあるタイルを調べる
func (bm MetaPlan) DownTile(idx gc.TileIdx) oapi.Tile {
	targetIdx := int(idx) + int(bm.Level.TileWidth)
	if targetIdx > len(bm.Tiles)-1 {
		// 境界外（マップ外＝暗闇）として扱う
		return bm.GetTile(consts.TileNameVoid)
	}

	return bm.Tiles[targetIdx]
}

// LeftTile は左にあるタイルを調べる
func (bm MetaPlan) LeftTile(idx gc.TileIdx) oapi.Tile {
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
func (bm MetaPlan) RightTile(idx gc.TileIdx) oapi.Tile {
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
func (bm MetaPlan) AdjacentAnyFloor(idx gc.TileIdx) bool {
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

		neighborIdx := bm.Level.XYTileIndex(consts.Tile(nx), consts.Tile(ny))
		tile := bm.Tiles[neighborIdx]

		// 歩行可能
		if !tile.BlockPass {
			return true
		}
	}

	return false
}

// GetWallType は近傍パターンから適切な壁タイプを判定する
func (bm MetaPlan) GetWallType(idx gc.TileIdx) WallType {
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
func (bm MetaPlan) isFloorOrWarp(tile oapi.Tile) bool {
	return !tile.BlockPass
}

// PlannerChain は階層データMetaPlanに対して適用する生成ロジックを保持する構造体
type PlannerChain struct {
	Starter   *InitialMapPlanner
	Planners  []MetaMapPlanner
	PlanData  MetaPlan
	Snapshots []Snapshot // フェーズごとのスナップショット
	Recording bool       // trueの場合、各フェーズ完了時にスナップショットを記録する
}

// NewPlannerChain はシード値を指定してプランナーチェーンを作成する
// シードが0の場合はランダムなシードを生成する
func NewPlannerChain(width consts.Tile, height consts.Tile, seed uint64) *PlannerChain {
	tileCount := int(width) * int(height)
	tiles := make([]oapi.Tile, tileCount)

	// シードが0の場合はランダムなシードを生成
	if seed == 0 {
		seed = uint64(time.Now().UnixNano())
	}

	return &PlannerChain{
		Starter:  nil,
		Planners: []MetaMapPlanner{},
		PlanData: MetaPlan{
			Level: gc.Level{
				TileWidth:  width,
				TileHeight: height,
			},
			Tiles:         tiles,
			Rooms:         []gc.Rect{},
			Corridors:     [][]gc.TileIdx{},
			RNG:           rand.New(rand.NewPCG(seed, seed+1)),
			NextPortals:   []consts.Coord[int]{},
			EscapePortals: []consts.Coord[int]{},
			NPCs:          []NPCSpec{},
			Items:         []ItemSpec{},
			Props:         []PropsSpec{},
			Doors:         []DoorSpec{},
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
	b.takeSnapshot("Initial")

	for _, meta := range b.Planners {
		if err := meta.PlanMeta(&b.PlanData); err != nil {
			return fmt.Errorf("PlanMeta failed: %w", err)
		}
		b.takeSnapshot(plannerName(meta))
	}
	return nil
}

// plannerName はMetaMapPlannerの型名を返す
func plannerName(p MetaMapPlanner) string {
	t := reflect.TypeOf(p)
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return t.Name()
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
func NewSmallRoomPlanner(width consts.Tile, height consts.Tile, seed uint64) (*PlannerChain, error) {
	chain := NewPlannerChain(width, height, seed)
	chain.StartWith(RectRoomPlanner{})
	chain.With(NewFillAll(consts.TileNameWall)) // 全体を壁で埋める
	chain.With(RoomDraw{})                      // 部屋を描画
	chain.With(LineCorridorPlanner{})           // 廊下を作成
	chain.With(DoorPlanner{DoorChance: 0.8})    // 入口にランダムにドアを配置
	chain.With(ConvertIsolatedWalls{            // 床に隣接しない壁をvoidに変換
		ReplacementTile: consts.TileNameVoid,
	})
	chain.With(EnvironmentPlanner{})

	return chain, nil
}

// NewBigRoomPlanner は大部屋プランナーを作成する
// ランダムにバリエーションを適用する統合版
func NewBigRoomPlanner(width consts.Tile, height consts.Tile, seed uint64) (*PlannerChain, error) {
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
	chain.With(EnvironmentPlanner{})

	return chain, nil
}

// SpawnEntry はスポーン対象のエントリを表す。
type SpawnEntry struct {
	Name    string
	Weight  float64
	PackMin int // パックの最小数。1以上
	PackMax int // パックの最大数。PackMin以上
}

// PackSize はPackMin〜PackMaxの範囲でランダムなパックサイズを返す
func (e SpawnEntry) PackSize(rng *rand.Rand) int {
	if e.PackMin == e.PackMax {
		return e.PackMin
	}
	return e.PackMin + rng.IntN(e.PackMax-e.PackMin+1)
}

// ItemGroupSubtype はアイテムグループの選択方式
type ItemGroupSubtype string

const (
	// ItemGroupDistribution はエントリ群から重み比率に基づいて1つだけ選ぶ。weightは相対比率として扱う
	ItemGroupDistribution ItemGroupSubtype = "distribution"
	// ItemGroupCollection は各エントリを独立に確率判定する。weightは0-100の出現確率(%)として扱う。両方出ることも、どちらも出ないこともある
	ItemGroupCollection ItemGroupSubtype = "collection"
)

// ItemSource はアイテム配置の元になるデータ
// テーブルエントリから解決済みの状態で保持する
type ItemSource struct {
	Weight  float64          // テーブルレベルの重み
	Subtype ItemGroupSubtype // グループの選択方式
	Entries []SpawnEntry     // グループ内のエントリ
}

// PlannerType はマップ生成の設定を表す構造体
type PlannerType struct {
	// プランナー名
	Name string
	// ポータル位置を固定するか
	UseFixedPortalPos bool
	// 敵テーブル名。RawMasterから敵エントリを解決する際に使用する
	EnemyTableName string
	// アイテムテーブル名。RawMasterからアイテムエントリを解決する際に使用する
	ItemTableName string
	// 階層の深度。敵やアイテムのフィルタリングに使用する
	Depth int
	// プランナー関数
	PlannerFunc func(width consts.Tile, height consts.Tile, seed uint64) (*PlannerChain, error)
}

var (
	// PlannerTypeRandom はランダム選択用のプランナータイプ
	PlannerTypeRandom = PlannerType{
		Name: "ランダム",
	}

	// PlannerTypeSmallRoom は小部屋ダンジョンのプランナータイプ
	PlannerTypeSmallRoom = PlannerType{
		Name:        "小部屋",
		PlannerFunc: NewSmallRoomPlanner,
	}

	// PlannerTypeBigRoom は大部屋ダンジョンのプランナータイプ
	PlannerTypeBigRoom = PlannerType{
		Name:        "大部屋",
		PlannerFunc: NewBigRoomPlanner,
	}

	// PlannerTypeCave は洞窟ダンジョンのプランナータイプ
	PlannerTypeCave = PlannerType{
		Name:        "洞窟",
		PlannerFunc: NewCavePlanner,
	}

	// PlannerTypeRuins は廃墟ダンジョンのプランナータイプ
	PlannerTypeRuins = PlannerType{
		Name:        "廃墟",
		PlannerFunc: NewRuinsPlanner,
	}

	// PlannerTypeForest は森ダンジョンのプランナータイプ
	PlannerTypeForest = PlannerType{
		Name:        "森",
		PlannerFunc: NewForestPlanner,
	}

	// PlannerTypeOverworldField はシームレスワールドの開けた地形チャンクのプランナータイプ。
	// 通行可能がデフォルトで障壁は例外。チャンクを継いでも東西通行が保証される。
	// UseFixedPortalPos=true はフロア降り/帰還ポータルを持たないため、
	// 手続き的なポータル配置をスキップさせる意味で使う。
	PlannerTypeOverworldField = PlannerType{
		Name:              "原野",
		UseFixedPortalPos: true,
		PlannerFunc:       NewOverworldFieldPlanner,
	}

	// PlannerTypeTown は市街地のプランナータイプ
	PlannerTypeTown = PlannerType{
		Name:              "市街地",
		UseFixedPortalPos: true,
		PlannerFunc: func(_ consts.Tile, _ consts.Tile, seed uint64) (*PlannerChain, error) {
			return NewPlannerChainByTemplateType(TemplateTypeTownPlaza, seed)
		},
	}

	// PlannerTypeOfficeBuilding は事務所ビルのプランナータイプ
	PlannerTypeOfficeBuilding = PlannerType{
		Name:              "事務所ビル",
		UseFixedPortalPos: true,
		PlannerFunc: func(_ consts.Tile, _ consts.Tile, seed uint64) (*PlannerChain, error) {
			return NewPlannerChainByTemplateType(TemplateTypeOfficeBuilding, seed)
		},
	}

	// PlannerTypeSmallTown は小さな町（複数の建物を配置）
	PlannerTypeSmallTown = PlannerType{
		Name:              "小さな町",
		UseFixedPortalPos: true,
		PlannerFunc: func(_ consts.Tile, _ consts.Tile, seed uint64) (*PlannerChain, error) {
			return NewPlannerChainByTemplateType(TemplateTypeSmallTown, seed)
		},
	}

	// PlannerTypeTownPlaza は町の広場
	PlannerTypeTownPlaza = PlannerType{
		Name:              "広場",
		UseFixedPortalPos: true,
		PlannerFunc: func(_ consts.Tile, _ consts.Tile, seed uint64) (*PlannerChain, error) {
			return NewPlannerChainByTemplateType(TemplateTypeTownPlaza, seed)
		},
	}

	// PlannerTypeBossFloor はボスフロアのプランナータイプ
	PlannerTypeBossFloor = PlannerType{
		Name:              "ボスフロア",
		UseFixedPortalPos: true,
		PlannerFunc: func(_ consts.Tile, _ consts.Tile, seed uint64) (*PlannerChain, error) {
			return NewPlannerChainByTemplateType(TemplateTypeBossFloor, seed)
		},
	}

	// AllPlannerTypes はPlannerFuncを持つ全PlannerTypeの一覧。
	// ランダム選択用のPlannerTypeRandomは含まない
	AllPlannerTypes = []PlannerType{
		PlannerTypeSmallRoom,
		PlannerTypeBigRoom,
		PlannerTypeCave,
		PlannerTypeRuins,
		PlannerTypeForest,
		PlannerTypeOverworldField,
		PlannerTypeTown,
		PlannerTypeOfficeBuilding,
		PlannerTypeSmallTown,
		PlannerTypeTownPlaza,
		PlannerTypeBossFloor,
	}
)

// NewRandomPlanner はシード値を使用してランダムにプランナーを選択し作成する
func NewRandomPlanner(width consts.Tile, height consts.Tile, seed uint64) (*PlannerChain, error) {
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

// selectSpawnEntry はSpawnEntryリストから重み付き抽選で1つ選択する
func selectSpawnEntry(entries []SpawnEntry, rng *rand.Rand) (SpawnEntry, error) {
	return raw.SelectByWeightFunc(
		entries,
		func(e SpawnEntry) float64 { return e.Weight },
		func(e SpawnEntry) SpawnEntry { return e },
		rng,
	)
}

// selectRoom は部屋リストから面積で重み付けして1つ選択し、部屋とそのインデックスを返す
// 大きな部屋ほど選ばれやすくなり、配置可能タイル数に比例した自然な分布になる
func (bm *MetaPlan) selectRoom() (gc.Rect, int, bool) {
	if len(bm.Rooms) == 0 {
		return gc.Rect{}, 0, false
	}
	totalArea := 0
	for _, r := range bm.Rooms {
		w := int(r.Max.X - r.Min.X)
		h := int(r.Max.Y - r.Min.Y)
		if w > 0 && h > 0 {
			totalArea += w * h
		}
	}
	if totalArea == 0 {
		idx := bm.RNG.IntN(len(bm.Rooms))
		return bm.Rooms[idx], idx, true
	}
	roll := bm.RNG.IntN(totalArea)
	cumulative := 0
	for i, r := range bm.Rooms {
		w := int(r.Max.X - r.Min.X)
		h := int(r.Max.Y - r.Min.Y)
		if w > 0 && h > 0 {
			cumulative += w * h
		}
		if roll < cumulative {
			return bm.Rooms[i], i, true
		}
	}
	idx := len(bm.Rooms) - 1
	return bm.Rooms[idx], idx, true
}

// randomPositionInRoom は指定した部屋内からスポーン可能なランダム座標を探す
// maxAttemptsを超えても見つからない場合はfalseを返す
func (bm *MetaPlan) randomPositionInRoom(room gc.Rect, world w.World, maxAttempts int) (consts.Tile, consts.Tile, bool) {
	rw := int(room.Max.X - room.Min.X)
	rh := int(room.Max.Y - room.Min.Y)
	if rw <= 0 || rh <= 0 {
		return 0, 0, false
	}
	for range maxAttempts {
		tx := consts.Tile(int(room.Min.X) + bm.RNG.IntN(rw))
		ty := consts.Tile(int(room.Min.Y) + bm.RNG.IntN(rh))
		if bm.IsSpawnableTile(world, tx, ty) {
			return tx, ty, true
		}
	}
	return 0, 0, false
}

// randomPositionNear は指定座標の近く、かつ部屋内からスポーン可能なランダム座標を探す。
// 大部屋でクラスタメンバーを密集させるために使用する
func (bm *MetaPlan) randomPositionNear(centerX, centerY consts.Tile, radius int, room gc.Rect, world w.World, maxAttempts int) (consts.Tile, consts.Tile, bool) {
	for range maxAttempts {
		dx := bm.RNG.IntN(radius*2+1) - radius
		dy := bm.RNG.IntN(radius*2+1) - radius
		tx := consts.Tile(int(centerX) + dx)
		ty := consts.Tile(int(centerY) + dy)
		if tx < room.Min.X || tx >= room.Max.X || ty < room.Min.Y || ty >= room.Max.Y {
			continue
		}
		if bm.IsSpawnableTile(world, tx, ty) {
			return tx, ty, true
		}
	}
	return 0, 0, false
}

// GetTile はタイルを生成する
// TODO: エラーを潰しているだけなので直す
func (bm *MetaPlan) GetTile(name string) oapi.Tile {
	if bm.RawMaster == nil {
		panic("RawMasterが設定されていない。TOMLからのタイル生成が必須である")
	}
	tile, err := raw.GetTile(*bm.RawMaster, name)
	if err != nil {
		panic(fmt.Sprintf("タイル生成エラー: %v", err))
	}
	return tile
}

// isInAnyRoom は指定座標がいずれかの部屋内に含まれるかを判定する
func (bm *MetaPlan) isInAnyRoom(x, y consts.Tile) bool {
	for _, room := range bm.Rooms {
		if x >= room.Min.X && x < room.Max.X && y >= room.Min.Y && y < room.Max.Y {
			return true
		}
	}
	return false
}

// GetPlayerStartPosition はプレイヤーの開始位置を取得する
// ポータルへの到達性も確認し、到達可能な位置を返す
func (bm *MetaPlan) GetPlayerStartPosition() (consts.Coord[int], error) {
	return NewPathFinder(bm).FindPlayerStartPosition()
}
