// Package mapplanner はマップ生成機能を提供する
// 参考: https://bfnightly.bracketproductions.com
package mapplanner

import (
	"fmt"
	"math/rand/v2"
	"reflect"
	"slices"
	"time"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/maptemplate"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/rawschema"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
)

// PropsSpec はProps配置仕様を表す
type PropsSpec struct {
	consts.Coord[consts.Tile]
	Name string // Prop名
}

// DecorSpec は地面の飾りの配置仕様を表す
type DecorSpec struct {
	consts.Coord[consts.Tile]
	Name string // 地面の飾りの id
}

// DoorSpec はドア配置仕様を表す
type DoorSpec struct {
	consts.Coord[consts.Tile]
	Orientation gc.DoorOrientation
}

// MetaPlan は階層のタイルを作る元になる概念の集合体
type MetaPlan struct {
	// Danger はこのプランに適用する危険度。敵やアイテムの湧きフィルタに使う。1以上であること。
	// 呼び出し側が世界の現在地から確定して渡す。
	Danger consts.Danger
	// Ground は計画の下地のタイル。地上の計画は全面を埋め、森は孤立した壁を、絵は下地を残す地形のマスをこれで置く
	Ground string
	// NeedsGround は間取りのタイルが下地 ground を指すか。立っていて Ground が空なら計画できない
	NeedsGround bool
	// FixedPortals はポータルを手続きで置かないか。絵がポータルを持つ計画と、ポータルを持たない地上の計画で立てる
	FixedPortals bool
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
	// ChargeRNG は見つかる物の残量を引く乱数。RNG と別の流れにし、物の定義を変えても配置の抽選をずらさない
	ChargeRNG *rand.Rand
	// 階層を構成するタイル群。長さはステージの大きさで決まる
	// 通行可能かを判定するための情報を保持している必要がある
	Tiles []rawschema.Tile
	// Environments は Tiles と同じ並びのタイルの環境。EnvironmentPlanner が計算する。計算前は nil
	Environments []gc.TileEnvironment
	// NextPortals は次の階へ進むポータルリスト
	NextPortals []consts.Coord[consts.Tile]
	// NPCs は配置予定のNPCリスト
	NPCs []NPCSpec
	// Items は配置予定のアイテムリスト
	Items []ItemSpec
	// Props は配置予定のPropsリスト
	Props []PropsSpec
	// Decors は配置予定の地面の飾りのリスト
	Decors []DecorSpec
	// Doors は配置予定のドアリスト
	Doors []DoorSpec
	// SpawnPoints はプレイヤーのスポーン地点リスト
	SpawnPoints []maptemplate.SpawnPoint
	// RawMaster はタイル生成に使用するマスターデータ
	RawMaster *rawschema.Raws
}

// IsSpawnableTile は指定タイル座標がスポーン可能かを返す
func (bm MetaPlan) IsSpawnableTile(_ w.World, tx consts.Tile, ty consts.Tile) bool {
	pos := consts.Coord[consts.Tile]{X: tx, Y: ty}
	idx := bm.Level.CoordToIndex(pos)
	tile := bm.Tiles[idx]
	// 通行不可なのでスポーン不可
	if tile.BlockPass {
		return false
	}

	// planning段階では、MetaPlan内の計画済みエンティティをチェック
	if bm.existPlannedEntityOnTile(pos) {
		return false
	}

	return true
}

// existPlannedEntityOnTile は指定座標に計画済みエンティティがあるかをチェック
func (bm MetaPlan) existPlannedEntityOnTile(pos consts.Coord[consts.Tile]) bool {
	return slices.Contains(bm.NextPortals, pos) ||
		slices.ContainsFunc(bm.NPCs, func(npc NPCSpec) bool { return npc.Coord == pos }) ||
		slices.ContainsFunc(bm.Items, func(item ItemSpec) bool { return item.Coord == pos }) ||
		slices.ContainsFunc(bm.Props, func(prop PropsSpec) bool { return prop.Coord == pos }) ||
		slices.ContainsFunc(bm.Doors, func(door DoorSpec) bool { return door.Coord == pos })
}

// outsideTile は地図の外を表すタイル。id を持たず、通れない
var outsideTile = rawschema.Tile{BlockPass: true}

// UpTile は上にあるタイルを調べる。地図の外は outsideTile を返す
func (bm MetaPlan) UpTile(idx gc.TileIdx) rawschema.Tile {
	targetIdx := gc.TileIdx(int(idx) - int(bm.Level.TileWidth))
	if targetIdx < 0 {
		return outsideTile
	}

	return bm.Tiles[targetIdx]
}

// DownTile は下にあるタイルを調べる。地図の外は outsideTile を返す
func (bm MetaPlan) DownTile(idx gc.TileIdx) rawschema.Tile {
	targetIdx := int(idx) + int(bm.Level.TileWidth)
	if targetIdx > len(bm.Tiles)-1 {
		return outsideTile
	}

	return bm.Tiles[targetIdx]
}

// LeftTile は左にあるタイルを調べる。地図の外は outsideTile を返す
func (bm MetaPlan) LeftTile(idx gc.TileIdx) rawschema.Tile {
	pos := bm.Level.IndexToCoord(idx)
	if pos.X == 0 {
		return outsideTile
	}

	// 左のタイルが同じ行であることを確認
	targetIdx := idx - 1
	if bm.Level.IndexToCoord(targetIdx).Y != pos.Y {
		return outsideTile
	}

	return bm.Tiles[targetIdx]
}

// RightTile は右にあるタイルを調べる。地図の外は outsideTile を返す
func (bm MetaPlan) RightTile(idx gc.TileIdx) rawschema.Tile {
	pos := bm.Level.IndexToCoord(idx)
	if pos.X == bm.Level.TileWidth-1 {
		return outsideTile
	}

	// 右のタイルが同じ行であることを確認
	targetIdx := idx + 1
	if bm.Level.IndexToCoord(targetIdx).Y != pos.Y {
		return outsideTile
	}

	return bm.Tiles[targetIdx]
}

// AdjacentAnyFloor は直交・斜めを含む近傍8タイルに床があるか判定する
func (bm MetaPlan) AdjacentAnyFloor(idx gc.TileIdx) bool {
	pos := bm.Level.IndexToCoord(idx)
	width := int(bm.Level.TileWidth)
	height := int(bm.Level.TileHeight)

	// 8方向の隣接タイル座標をチェック
	directions := [][2]int{
		{-1, -1}, {-1, 0}, {-1, 1}, // 上段
		{0, -1}, {0, 1}, // 中段（中心を除く）
		{1, -1}, {1, 0}, {1, 1}, // 下段
	}

	for _, dir := range directions {
		nx, ny := int(pos.X)+dir[0], int(pos.Y)+dir[1]

		// 境界チェック
		if nx < 0 || nx >= width || ny < 0 || ny >= height {
			continue
		}

		neighborIdx := bm.Level.CoordToIndex(consts.Coord[consts.Tile]{X: consts.Tile(nx), Y: consts.Tile(ny)})
		tile := bm.Tiles[neighborIdx]

		// 歩行可能
		if !tile.BlockPass {
			return true
		}
	}

	return false
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
	tiles := make([]rawschema.Tile, tileCount)

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
			Tiles:       tiles,
			Rooms:       []gc.Rect{},
			Corridors:   [][]gc.TileIdx{},
			RNG:         rand.New(rand.NewPCG(seed, seed+1)),
			ChargeRNG:   rand.New(rand.NewPCG(seed^lifecycle.LootChargeChannel, 0)),
			NextPortals: []consts.Coord[consts.Tile]{},
			NPCs:        []NPCSpec{},
			Items:       []ItemSpec{},
			Props:       []PropsSpec{},
			Decors:      []DecorSpec{},
			Doors:       []DoorSpec{},
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
	if b.PlanData.NeedsGround && b.PlanData.Ground == "" {
		return ErrGroundNotSet
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

// SpawnEntry はスポーン対象のエントリを表す。
type SpawnEntry struct {
	Name   string
	Weight float64
	Pack   consts.Dice // 1山あたりの個数
}

// PackSize はダイス表記からパックサイズを抽選する
func (e SpawnEntry) PackSize(rng *rand.Rand) int {
	return e.Pack.Roll(rng)
}

// PlannerType はマップ生成の設定を表す構造体
type PlannerType struct {
	// プランナー名
	Name string
	// 敵テーブル名。RawMasterから敵エントリを解決する際に使用する
	EnemyTableName string
	// アイテムテーブル名。RawMasterからアイテムエントリを解決する際に使用する
	ItemTableName string
	// フロアに置く敵の数
	EnemyCount rawschema.RandInt
	// フロアに置くアイテムの数
	ItemCount rawschema.RandInt
	// 危険度。敵やアイテムのフィルタリングに使用する。1以上であること。
	Danger consts.Danger
	// 地面を埋める下地のタイル。地上と森の間取りが使い、空ならエラーにする
	Ground string
	// プランナー関数。間取りの行は raws から引く
	PlannerFunc func(raws rawschema.Raws, width consts.Tile, height consts.Tile, seed uint64) (*PlannerChain, error)
}

var (
	// PlannerTypeRandom はランダム選択用のプランナータイプ
	PlannerTypeRandom = PlannerType{
		Name:        "Random",
		PlannerFunc: NewRandomPlanner,
	}

	// PlannerTypeOverworldField はシームレスワールドの開けた地形チャンクのプランナータイプ。
	// 通行可能がデフォルトで障壁は例外。チャンクを継いでも東西通行が保証される。
	// 下地は区分ごとに決まるので持たない。使う側が Ground を補わないと ErrGroundNotSet になる。
	PlannerTypeOverworldField = PlannerType{
		Name: "Overworld Field",
		PlannerFunc: func(_ rawschema.Raws, width, height consts.Tile, seed uint64) (*PlannerChain, error) {
			return NewOverworldFieldPlanner(width, height, seed)
		},
	}
)

// NewRandomPlanner は絵を使わない間取りの行からシード値で1つ選び、その間取りのチェーンを作成する
func NewRandomPlanner(raws rawschema.Raws, width consts.Tile, height consts.Tile, seed uint64) (*PlannerChain, error) {
	// シードが0の場合はランダムなシードを生成する。後続のビルダーに渡される
	if seed == 0 {
		seed = uint64(time.Now().UnixNano())
	}
	candidates := proceduralLayoutIDs(raws)
	if len(candidates) == 0 {
		return nil, fmt.Errorf("random planner selection error: %w", ErrNoLayout)
	}
	rng := rand.New(rand.NewPCG(seed, 0))
	chain, err := newLayoutChain(raws, candidates[rng.IntN(len(candidates))], width, height, seed)
	if err != nil {
		return nil, fmt.Errorf("random planner selection error: %w", err)
	}
	return chain, nil
}

// selectSpawnEntry はSpawnEntryリストから重み付き抽選で1つ選択する
func selectSpawnEntry(entries []SpawnEntry, rng *rand.Rand) (SpawnEntry, bool) {
	i, ok := consts.Pick(rng, entries, func(e SpawnEntry) float64 { return e.Weight })
	if !ok {
		return SpawnEntry{}, false
	}
	return entries[i], true
}

// selectRoom は部屋リストから面積で重み付けして1つ選択し、部屋とそのインデックスを返す
// 大きな部屋ほど選ばれやすくなり、配置可能タイル数に比例した自然な分布になる
func (bm *MetaPlan) selectRoom() (gc.Rect, int, bool) {
	i, ok := consts.Pick(bm.RNG, bm.Rooms, func(r gc.Rect) float64 {
		w, h := int(r.Width()), int(r.Height())
		if w <= 0 || h <= 0 {
			return 0
		}
		return float64(w * h)
	})
	if !ok {
		return gc.Rect{}, 0, false
	}
	return bm.Rooms[i], i, true
}

// randomPositionInRoom は指定した部屋内からスポーン可能なランダム座標を探す
// maxAttemptsを超えても見つからない場合はfalseを返す
func (bm *MetaPlan) randomPositionInRoom(room gc.Rect, world w.World, maxAttempts int) (consts.Tile, consts.Tile, bool) {
	rw, rh := int(room.Width()), int(room.Height())
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
		if tx < room.Min.X || tx > room.Max.X || ty < room.Min.Y || ty > room.Max.Y {
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
func (bm *MetaPlan) GetTile(name string) rawschema.Tile {
	if name == maptemplate.GroundTerrain {
		name = bm.Ground
	}
	if bm.RawMaster == nil {
		panic("RawMaster is not set; tile generation from TOML is required")
	}
	tile, err := raw.GetTile(*bm.RawMaster, name)
	if err != nil {
		panic(fmt.Sprintf("tile generation error: %v", err))
	}
	return tile
}

// isInAnyRoom は指定座標がいずれかの部屋内に含まれるかを判定する
func (bm *MetaPlan) isInAnyRoom(x, y consts.Tile) bool {
	for _, room := range bm.Rooms {
		if x >= room.Min.X && x <= room.Max.X && y >= room.Min.Y && y <= room.Max.Y {
			return true
		}
	}
	return false
}

// GetPlayerStartPosition はプレイヤーの開始位置を取得する
// ポータルへの到達性も確認し、到達可能な位置を返す
func (bm *MetaPlan) GetPlayerStartPosition() (consts.Coord[consts.Tile], error) {
	return NewPathFinder(bm).FindPlayerStartPosition()
}
