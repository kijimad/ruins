package overworld

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/dungeon"
	"github.com/kijimaD/ruins/internal/mapplanner"
	"github.com/kijimaD/ruins/internal/save"
	"github.com/kijimaD/ruins/internal/testutil"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// countCubes は移動拠点キューブの数を返す。クエリは最後まで回す
func countCubes(world w.World) int {
	count := 0
	q := query.ActiveFilter1[gc.Drivable](world).Query()
	for q.Next() {
		count++
	}
	return count
}

const (
	testChunkW consts.Tile  = 30
	testChunkH consts.Tile  = 20
	testCols   consts.Chunk = 1 // 有界の回廊幅
	testRows   consts.Chunk = 3 // 北へ流す窓の行数
)

func TestDriver_Start_プレイヤー先在でもキューブをスポーンする(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	// 本番はキャラ作成で先にプレイヤーが湧く。その状況を再現する
	_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)

	s := NewDriver(mapplanner.PlannerTypeSmallRoom, dungeon.NewOverworldDefinition("オーバーワールド", 0, testChunkW, testChunkH, testCols, testRows), &NewGameParams{RunSeed: 777})
	require.NoError(t, s.Start(world))

	assert.Equal(t, 1, countCubes(world), "プレイヤーが先在してもキューブが1体スポーンする")
	assert.Equal(t, gc.TimeDawn, query.GetGameTime(world).GetTimeOfDay(), "新規開始は夜明けから始まる")
}

// TestDriver_Start_復帰経路ではキューブを生成しない は、帯が active な世界で Start を起動すると
// restoreFromSave の枝を通り startInitialBand が走らないので、ドライバがキューブを生成しないことを
// 固定する。実際の復帰ではキューブはプレイヤーと同じく serde で復元され、ドライバは生成しない。
func TestDriver_Start_復帰経路ではキューブを生成しない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	// 新規開始でキューブが1体湧き、帯が active になる
	s := NewDriver(mapplanner.PlannerTypeSmallRoom, dungeon.NewOverworldDefinition("オーバーワールド", 0, testChunkW, testChunkH, testCols, testRows), &NewGameParams{RunSeed: 777})
	require.NoError(t, s.Start(world))
	require.Equal(t, 1, countCubes(world), "前提: 新規開始でキューブが1体湧く")

	// 復帰を再現する。active な帯を持つ世界で params nil のドライバを起動すると restoreFromSave を通る
	s2 := NewDriver(mapplanner.PlannerTypeSmallRoom, dungeon.DungeonOverworld, nil)
	require.NoError(t, s2.Start(world))
	assert.Equal(t, 1, countCubes(world), "復帰経路ではドライバはキューブを生成しない")
}

func TestDriver_MaybeShift_北へ進むとシフトする(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	s := NewDriver(mapplanner.PlannerTypeSmallRoom, dungeon.NewOverworldDefinition("オーバーワールド", 0, testChunkW, testChunkH, testCols, testRows), &NewGameParams{RunSeed: 777})
	require.NoError(t, s.Start(world))

	player, err := query.GetPlayerEntity(world)
	require.NoError(t, err)
	// 北は -Y。中央行の北すなわち localY=0 の行へ踏み込む
	world.Components.GridElement.Get(player).Y = 0

	shifted, err := s.MaybeShift(world)
	require.NoError(t, err)
	assert.True(t, shifted, "北へ踏み込むとシフトする")
	assert.Equal(t, 1, int(s.NorthIndex()), "北シフトで northIndex が進む")
	assert.Equal(t, testChunkH, world.Components.GridElement.Get(player).Y, "プレイヤーは中央へ戻る")
}

func TestDriver_MaybeShift_複数チャンク跨ぎで連続シフト(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	s := NewDriver(mapplanner.PlannerTypeSmallRoom, dungeon.NewOverworldDefinition("オーバーワールド", 0, testChunkW, testChunkH, testCols, testRows), &NewGameParams{RunSeed: 777})
	require.NoError(t, s.Start(world))

	player, err := query.GetPlayerEntity(world)
	require.NoError(t, err)
	world.Components.GridElement.Get(player).Y = -10 // 2チャンク以上北（帯外）

	shifted, err := s.MaybeShift(world)
	require.NoError(t, err)
	assert.True(t, shifted)
	assert.Equal(t, 2, int(s.NorthIndex()), "収まるまで連続シフトして northIndex=2")
	py := world.Components.GridElement.Get(player).Y
	assert.GreaterOrEqual(t, py, consts.Tile(testRows/2)*testChunkH, "プレイヤーは中央チャンク内に収まる")
	assert.Less(t, py, consts.Tile(testRows/2+1)*testChunkH, "プレイヤーは中央チャンク内に収まる")
}

// TestDriver_MaybeShift_開始点より南へはシフトしない は northIndex=0 で南へ移動しても
// northIndex を負にしないことを固定する。
func TestDriver_MaybeShift_開始点より南へはシフトしない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	s := NewDriver(mapplanner.PlannerTypeSmallRoom, dungeon.NewOverworldDefinition("オーバーワールド", 0, testChunkW, testChunkH, testCols, testRows), &NewGameParams{RunSeed: 777})
	require.NoError(t, s.Start(world))
	require.Equal(t, 0, int(s.NorthIndex()), "前提: 開始時 northIndex=0")

	player, err := query.GetPlayerEntity(world)
	require.NoError(t, err)
	world.Components.GridElement.Get(player).Y = 2 * testChunkH // 中央チャンクより南

	_, err = s.MaybeShift(world)
	require.NoError(t, err)
	assert.Equal(t, 0, int(s.NorthIndex()), "開始点より南へはシフトしない（northIndex は負にならない）")
}

// TestDriver_MaybeShift_北進後は南へ戻らない は、北へシフトした後に南端より南へ移動しても
// 南シフトが起きず northIndex が戻らないことを固定する。帯は北へのみ進み破棄済み南チャンクを
// 再生成しないので、到達最南端より南へは戻れない。ShiftSouth 再導入への抑止線を兼ねる。
func TestDriver_MaybeShift_北進後は南へ戻らない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	s := NewDriver(mapplanner.PlannerTypeSmallRoom, dungeon.NewOverworldDefinition("オーバーワールド", 0, testChunkW, testChunkH, testCols, testRows), &NewGameParams{RunSeed: 777})
	require.NoError(t, s.Start(world))

	player, err := query.GetPlayerEntity(world)
	require.NoError(t, err)

	// 北チャンクへ踏み込んで1回シフトさせる
	world.Components.GridElement.Get(player).Y = 0
	shifted, err := s.MaybeShift(world)
	require.NoError(t, err)
	require.True(t, shifted, "前提: 北へ1回シフトする")
	require.Equal(t, 1, int(s.NorthIndex()), "前提: northIndex=1 になる")

	// 帯南端より南へ移動しても南シフトは起きず northIndex は戻らない
	world.Components.GridElement.Get(player).Y = testRows.Tiles(testChunkH) - 1
	shifted, err = s.MaybeShift(world)
	require.NoError(t, err)
	assert.False(t, shifted, "南へ移動してもシフトしない")
	assert.Equal(t, 1, int(s.NorthIndex()), "到達最南端より南へは戻れない（南チャンクを再生成しない）")
}

func TestDriver_MaybeShift_中央では動かない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	s := NewDriver(mapplanner.PlannerTypeSmallRoom, dungeon.NewOverworldDefinition("オーバーワールド", 0, testChunkW, testChunkH, testCols, testRows), &NewGameParams{RunSeed: 777})
	require.NoError(t, s.Start(world))

	shifted, err := s.MaybeShift(world)
	require.NoError(t, err)
	assert.False(t, shifted, "中央チャンク内ではシフトしない")
	assert.Equal(t, 0, int(s.NorthIndex()), "中央チャンク内では northIndex 据え置き")
}

// TestDriver_セーブ往復で帯状態が復元される は、SeamlessBand が serde に乗り、ロード後に
// ドライバが同じ northIndex で再構築できることを固定する。
func TestDriver_セーブ往復で帯状態が復元される(t *testing.T) {
	t.Parallel()

	const chunkW, chunkH consts.Tile = 40, 20
	const cols consts.Chunk = 1
	const rows consts.Chunk = 3

	world := testutil.InitTestWorld(t)
	s := NewDriver(mapplanner.PlannerTypeOverworldField, dungeon.NewOverworldDefinition("オーバーワールド", 0, chunkW, chunkH, cols, rows), &NewGameParams{RunSeed: 12345})
	require.NoError(t, s.Start(world))

	// 北へ1回シフトして northIndex=1 にする
	player, err := query.GetPlayerEntity(world)
	require.NoError(t, err)
	world.Components.GridElement.Get(player).Y = 0
	shifted, err := s.MaybeShift(world)
	require.NoError(t, err)
	require.True(t, shifted)
	require.Equal(t, 1, int(s.NorthIndex()))
	require.Equal(t, 1, int(query.GetSeamlessBand(world).NorthIndex), "永続状態に同期される")

	// セーブ往復（メモリ内）
	sm, err := save.NewSerializationManager()
	require.NoError(t, err)
	jsonData, err := sm.GenerateWorldJSON(world)
	require.NoError(t, err)

	world2 := testutil.InitTestWorld(t)
	require.NoError(t, sm.RestoreWorldFromJSON(world2, jsonData))

	// SeamlessBand が復元されている
	sb := *query.GetSeamlessBand(world2)
	assert.True(t, sb.Active, "Active が復元される")
	assert.Equal(t, 1, int(sb.NorthIndex), "NorthIndex が復元される")
	assert.Equal(t, uint64(12345), sb.RunSeed, "RunSeed が復元される")
	assert.Equal(t, chunkH, sb.ChunkH, "ChunkH が復元される")
	assert.Equal(t, rows, sb.Rows, "Rows が復元される")

	// 復元ワールドでロード用ドライバを起動 → Band が northIndex=1 で再構築される
	s2 := NewDriver(mapplanner.PlannerTypeOverworldField, dungeon.DungeonOverworld, nil)
	require.NoError(t, s2.Start(world2))
	assert.Equal(t, 1, int(s2.NorthIndex()), "ロード復元で Band が northIndex=1 で再構築される")
	assert.Equal(t, rows.Tiles(chunkH), query.GetCurrentStageField(world2).Level.TileHeight, "帯全高の Level が保たれる")

	// 復元ワールドに帯タイルが存在する（serde 復元）
	count := 0
	tileQuery := ecs.NewFilter1[gc.GridElement](world2.ECS).Query()
	for tileQuery.Next() {
		count++
	}
	assert.Positive(t, count, "帯タイルが serde で復元されている")
}

// TestNewChunkGen_集落は種別分類と一致し帯へ束縛される は、chunkTypeAt が集落と分類する
// チャンクに集落の会話NPCが実際に spawn され、オーバーワールド帯へ束縛され相互作用を持つことを
// 固定する。集落は Y 方向のリージョンに並ぶので、Y を走査して当選チャンクを探す。
func TestNewChunkGen_集落は種別分類と一致し帯へ束縛される(t *testing.T) {
	t.Parallel()

	const chunkW, chunkH consts.Tile = 30, 20
	const cols consts.Chunk = 1
	// chunkTypeAt が集落と分類するチャンクを探す。集落は Y リージョンなので Y を振る
	var seed uint64
	var c consts.Coord[consts.Chunk]
	found := false
	for s := uint64(1); s < 500 && !found; s++ {
		for y := range consts.Chunk(12) {
			if chunkTypeAt(s, consts.Coord[consts.Chunk]{Y: y}, cols) == chunkSettlement {
				seed, c, found = s, consts.Coord[consts.Chunk]{Y: y}, true
				break
			}
		}
	}
	require.True(t, found, "前提: 集落と分類されるチャンクが見つかる")

	world := testutil.InitTestWorld(t)
	gen := NewChunkGen(world, seed, chunkW, chunkH, cols, mapplanner.PlannerTypeSmallRoom)
	require.NoError(t, gen(c, 0, 0))

	// 商人が spawn され、オーバーワールド帯へ束縛され、相互作用を持つ
	merchantFound := false
	q := ecs.NewFilter1[gc.Name](world.ECS).Query()
	for q.Next() {
		e := q.Entity()
		if world.Components.Name.Get(e).Name != "Merchant" {
			continue
		}
		merchantFound = true
		require.True(t, world.Components.StageBound.Has(e), "商人はステージへ束縛される")
		assert.Equal(t, gc.NewOverworldStage(), world.Components.StageBound.Get(e).Key, "商人はオーバーワールド帯へ束縛される")
		assert.True(t, world.Components.Interactable.Has(e), "商人は相互作用を持つ")
	}
	assert.True(t, merchantFound, "集落と分類されるチャンクに商人が配置される")
}

// TestDriver_Rowsの書き込みと正規化 は、新規開始が Rows をセーブ対象へ書き込み、
// ゼロ値の Rows を復元時に 1 へ正規化することを固定する。
func TestDriver_Rowsの書き込みと正規化(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	s := NewDriver(mapplanner.PlannerTypeSmallRoom, dungeon.NewOverworldDefinition("オーバーワールド", 0, testChunkW, testChunkH, testCols, testRows), &NewGameParams{RunSeed: 777})
	require.NoError(t, s.Start(world))
	assert.Equal(t, testRows, query.GetSeamlessBand(world).Rows, "新規開始で Rows がセーブ対象へ書き込まれる")

	// Rows がゼロ値で復元される場合。1行帯へ正規化される
	oldWorld := testutil.InitTestWorld(t)
	drOld := NewDriver(mapplanner.PlannerTypeSmallRoom, nil, nil)
	sbOld := &gc.SeamlessBand{Active: true, RunSeed: 1, ChunkW: testChunkW, ChunkH: testChunkH, Cols: testCols}
	require.NoError(t, drOld.restoreFromSave(oldWorld, sbOld))
	assert.Equal(t, consts.Chunk(1), drOld.band.Rows(), "旧セーブのゼロ値 Rows は 1 へ正規化される")

	// Rows=3 のセーブはそのまま3行で復元される
	world3 := testutil.InitTestWorld(t)
	dr3 := NewDriver(mapplanner.PlannerTypeSmallRoom, nil, nil)
	sb3 := &gc.SeamlessBand{Active: true, RunSeed: 1, ChunkW: testChunkW, ChunkH: testChunkH, Cols: testCols, Rows: 3}
	require.NoError(t, dr3.restoreFromSave(world3, sb3))
	assert.Equal(t, consts.Chunk(3), dr3.band.Rows(), "Rows=3 のセーブは3行で復元される")
}

// TestDriver_3行帯の通し は rows=3 の帯で、新規開始の全行生成・行単位の北シフト・
// セーブ復元までが一貫して3行のまま保たれることを固定する。
func TestDriver_3行帯の通し(t *testing.T) {
	t.Parallel()

	const rows consts.Chunk = 3
	world := testutil.InitTestWorld(t)
	s := NewDriver(mapplanner.PlannerTypeOverworldField, dungeon.NewOverworldDefinition("オーバーワールド", 0, testChunkW, testChunkH, testCols, rows), &NewGameParams{RunSeed: 900})
	require.NoError(t, s.Start(world))

	countRows := func(wld w.World) []int {
		counts := make([]int, int(rows))
		q := ecs.NewFilter1[gc.GridElement](wld.ECS).Query()
		for q.Next() {
			y := wld.Components.GridElement.Get(q.Entity()).Y
			if y < 0 || y >= rows.Tiles(testChunkH) {
				continue
			}
			counts[int(y/testChunkH)]++
		}
		return counts
	}

	// Level は帯全域になり、全3行にタイルが生成されている
	field := query.GetCurrentStageField(world)
	assert.Equal(t, testCols.Tiles(testChunkW), field.Level.TileWidth, "幅は cols*chunkW")
	assert.Equal(t, rows.Tiles(testChunkH), field.Level.TileHeight, "高さは rows*chunkH")
	for r, c := range countRows(world) {
		assert.Positivef(t, c, "行%d にタイルが生成されている", r)
	}

	// プレイヤーは中央チャンク・中央行に立つ
	player, err := query.GetPlayerEntity(world)
	require.NoError(t, err)
	pg := world.Components.GridElement.Get(player)
	assert.Equal(t, (testCols/2).Tiles(testChunkW)+testChunkW/2, pg.X, "中央チャンクの中央 X")
	assert.Equal(t, (rows/2).Tiles(testChunkH)+testChunkH/2, pg.Y, "中央行の中央 Y")

	// 北シフトは行単位で全列を入れ替え、3行とも埋まったまま
	pg.Y = 0
	shifted, err := s.MaybeShift(world)
	require.NoError(t, err)
	require.True(t, shifted, "北へ踏み込むとシフトする")
	assert.Equal(t, 1, int(s.NorthIndex()), "northIndex が進む")
	for r, c := range countRows(world) {
		assert.Positivef(t, c, "シフト後も行%d にタイルがある", r)
	}

	// セーブ往復で Rows=3 が復元され、帯も3行で再構築される
	sm, err := save.NewSerializationManager()
	require.NoError(t, err)
	jsonData, err := sm.GenerateWorldJSON(world)
	require.NoError(t, err)
	world2 := testutil.InitTestWorld(t)
	require.NoError(t, sm.RestoreWorldFromJSON(world2, jsonData))
	assert.Equal(t, rows, query.GetSeamlessBand(world2).Rows, "Rows が復元される")

	s2 := NewDriver(mapplanner.PlannerTypeOverworldField, nil, nil)
	require.NoError(t, s2.Start(world2))
	assert.Equal(t, rows, s2.band.Rows(), "ロード復元で帯が3行で再構築される")
	assert.Equal(t, rows.Tiles(testChunkH), query.GetCurrentStageField(world2).Level.TileHeight, "帯全高の Level が保たれる")
	for r, c := range countRows(world2) {
		assert.Positivef(t, c, "復元後も行%d にタイルがある", r)
	}
}

// TestDriver_シフト後もタイルは座標ごとに1枚 は、市街地を含む帯を北へ複数回シフトしても
// タイルエンティティが座標ごとに1枚のままであることを固定する。置換や破棄の取りこぼしが
// あると同一座標に古いタイルが残留し、見えない壁の影などの怪奇現象になる。
func TestDriver_シフト後もタイルは座標ごとに1枚(t *testing.T) {
	t.Parallel()

	// 初期帯の視界内に市街地の断片が入る seed を選ぶ。市街地は Y リージョンに並ぶ。開始チャンク(行1)は避ける
	var seed uint64
	for s := uint64(1); s < 500; s++ {
		if urbanPlacement.At(s, consts.Coord[consts.Chunk]{Y: 2}, testCols) {
			seed = s
			break
		}
	}
	require.NotZero(t, seed, "前提: 市街地が行2に当たる seed がある")

	world := testutil.InitTestWorld(t)
	s := NewDriver(mapplanner.PlannerTypeOverworldField, dungeon.NewOverworldDefinition("オーバーワールド", 0, testChunkW, testChunkH, testCols, testRows), &NewGameParams{RunSeed: seed})
	require.NoError(t, s.Start(world))

	player, err := query.GetPlayerEntity(world)
	require.NoError(t, err)

	for range 3 {
		world.Components.GridElement.Get(player).Y = 0
		shifted, err := s.MaybeShift(world)
		require.NoError(t, err)
		require.True(t, shifted)
	}

	counts := map[gc.GridElement]int{}
	q := ecs.NewFilter2[gc.GridElement, gc.Tile](world.ECS).Query()
	for q.Next() {
		counts[*world.Components.GridElement.Get(q.Entity())]++
	}
	for g, c := range counts {
		assert.Equalf(t, 1, c, "座標 (%d,%d) のタイルは1枚", g.X, g.Y)
	}
}

// TestWalkableSpawnNear_壁に囲まれた中心でも歩行可能タイルを返す は、開始チャンクが建物や遺跡入口で
// 中心が壁でも、プレイヤーを壁の中へ湧かせず近傍の歩行可能タイルへ逃がすことを固定する。
func TestWalkableSpawnNear_壁に囲まれた中心でも歩行可能タイルを返す(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	center := consts.Coord[consts.Tile]{X: 10, Y: 10}

	// 中心とその 3×3 近傍を BlockPass の壁で埋める。raw に依存せず直接コンポーネントを付ける
	blockedCoords := map[consts.Coord[consts.Tile]]bool{}
	for dy := consts.Tile(-1); dy <= 1; dy++ {
		for dx := consts.Tile(-1); dx <= 1; dx++ {
			pos := consts.Coord[consts.Tile]{X: center.X + dx, Y: center.Y + dy}
			e := world.ECS.NewEntity()
			world.Components.GridElement.Add(e, &gc.GridElement{Coord: pos})
			world.Components.BlockPass.Add(e, &gc.BlockPass{})
			blockedCoords[pos] = true
		}
	}

	got := walkableSpawnNear(world, center)

	assert.Falsef(t, blockedCoords[got], "返り値 %v は壁でない", got)
	assert.NotEqual(t, center, got, "壁の中心そのものは返さず外へ逃げる")
}
