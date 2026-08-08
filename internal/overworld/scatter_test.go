package overworld

import (
	"sort"
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// scatterTestSeed はテストで使う固定の RunSeed。散布は純関数なので同じ seed で必ず同じ世界になる。
const scatterTestSeed uint64 = 424242

// scatterTestRows はテスト帯の行数。ゾーンや経路の当選行抽選に使う。
const scatterTestRows consts.Chunk = 3

// scatterTestChunk は本番と同じ 24 タイル四方のテストチャンク寸法。散布の密度と位相を本番相当で見る。
const scatterTestChunk consts.Tile = 24

// newDirtChunk は scatterTestChunk 四方の土タイルだけを band ローカル座標 0.. へ敷き、その chunkGeom を
// 返す。散布を他フィーチャから切り離し、置ける先が土だけの開けたチャンクで検証するための土台。
func newDirtChunk(t *testing.T, world w.World) chunkGeom {
	t.Helper()
	for y := range scatterTestChunk {
		for x := range scatterTestChunk {
			_, err := lifecycle.SpawnTile(world, consts.TileNameDirt, x, y, new(0))
			require.NoError(t, err)
		}
	}
	return chunkGeom{
		offsetX: 0, offsetY: 0, chunkW: scatterTestChunk, chunkH: scatterTestChunk,
		tiles: &tileIndex{world: world, loX: 0, hiX: scatterTestChunk},
	}
}

// firstWastelandChunk は scatterTestSeed のもとで最初に見つかる wasteland チャンク座標を返す。
func firstWastelandChunk(t *testing.T) consts.Coord[consts.Chunk] {
	t.Helper()
	for cx := range consts.Chunk(200) {
		c := consts.Coord[consts.Chunk]{X: cx, Y: 0}
		if chunkTypeAt(scatterTestSeed, c, scatterTestRows) == chunkWasteland {
			return c
		}
	}
	require.FailNow(t, "wasteland チャンクが見つからない")
	return consts.Coord[consts.Chunk]{}
}

// firstNonWastelandChunk は scatterTestSeed のもとで最初に見つかる wasteland 以外のチャンク座標を返す。
func firstNonWastelandChunk(t *testing.T) consts.Coord[consts.Chunk] {
	t.Helper()
	for cx := range consts.Chunk(200) {
		c := consts.Coord[consts.Chunk]{X: cx, Y: 0}
		if chunkTypeAt(scatterTestSeed, c, scatterTestRows) != chunkWasteland {
			return c
		}
	}
	require.FailNow(t, "wasteland 以外のチャンクが見つからない")
	return consts.Coord[consts.Chunk]{}
}

// propAt は配置済み prop の座標と名前。散布結果の比較に使う。
type propAt struct {
	pos  consts.Coord[consts.Tile]
	id   string
	name string
}

// collectProps は Fixed を持つ全 prop を座標順で集める。草・岩は BlockPass を持たないので、散布物の
// 網羅は固定物マーカーの Fixed で引く。SpawnProp は全 prop へ Fixed を付ける。タイルは Fixed を持たない。
func collectProps(world w.World) []propAt {
	var out []propAt
	q := query.ActiveFilter2[gc.GridElement, gc.Fixed](world).Query()
	for q.Next() {
		e := q.Entity()
		ge := *world.Components.GridElement.Get(e)
		name := ""
		if world.Components.Name.Has(e) {
			name = world.Components.Name.Get(e).Name
		}
		id := ""
		if world.Components.RawID.Has(e) {
			id = world.Components.RawID.Get(e).ID
		}
		out = append(out, propAt{pos: ge.Coord, id: id, name: name})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].pos.Y != out[j].pos.Y {
			return out[i].pos.Y < out[j].pos.Y
		}
		if out[i].pos.X != out[j].pos.X {
			return out[i].pos.X < out[j].pos.X
		}
		return out[i].name < out[j].name
	})
	return out
}

// collectBlockingTiles は BlockPass を持つエンティティの占有タイルを集める。通行性の flood 判定で
// 壁として扱うのはこれだけで、歩行可能な草地や plant・moving_stone は含まない。
func collectBlockingTiles(world w.World) map[consts.Coord[consts.Tile]]bool {
	blocked := make(map[consts.Coord[consts.Tile]]bool)
	q := query.ActiveFilter2[gc.GridElement, gc.BlockPass](world).Query()
	for q.Next() {
		blocked[world.Components.GridElement.Get(q.Entity()).Coord] = true
	}
	return blocked
}

// TestScatterFeature_同じseedは同じ配置 は散布が決定的な純関数であることを固定する。同じ seed と
// チャンク座標なら別の world でも同じ prop が同じ座標に並ぶ。再訪一致と serde 安全の土台。
func TestScatterFeature_同じseedは同じ配置(t *testing.T) {
	t.Parallel()
	c := firstWastelandChunk(t)

	run := func() []propAt {
		world := testutil.InitTestWorld(t)
		g := newDirtChunk(t, world)
		require.NoError(t, openTerrainFeature{}.place(world, scatterTestSeed, c, scatterTestRows, g))
		return collectProps(world)
	}

	a := run()
	b := run()
	assert.NotEmpty(t, a, "wasteland チャンクには何か散布される")
	assert.Equal(t, a, b, "同じ seed と座標なら同じ配置になる")
}

// TestScatterFeature_土系の空きにだけ置く は、散布した prop が必ず土系の地面の上にあり、かつ2つの
// prop が同じタイルへ重ならないことを固定する。占有回避と非地面回避の回帰。座標の一意性を全 prop で
// 見るので、anchor 同士だけでなく衛星と anchor、衛星同士の重複もここで捕捉する。
func TestScatterFeature_土系の空きにだけ置く(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	c := firstWastelandChunk(t)
	g := newDirtChunk(t, world)
	require.NoError(t, openTerrainFeature{}.place(world, scatterTestSeed, c, scatterTestRows, g))

	tiles := g.tiles.get()
	seen := make(map[gc.GridElement]bool)
	props := collectProps(world)
	require.NotEmpty(t, props)
	for _, p := range props {
		assert.True(t, isEarthTile(world, tiles, p.pos), "散布は土系の地面の上にだけ置く: %v", p)
		key := gc.GridElement{Coord: p.pos}
		assert.False(t, seen[key], "2つの prop が同じタイルへ重ならない: %v", p)
		seen[key] = true
	}
}

// TestScatterFeature_草を撒く は、散布が地面へ草・雑草の prop を撒いて野原の質感を出すことを固定する。
// 草は透明背景の prop でタイルのオートタイルを経由しないため、縁や暗いフィルが出ない。歩行可能。
func TestScatterFeature_草を撒く(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	c := firstWastelandChunk(t)
	g := newDirtChunk(t, world)
	require.NoError(t, openTerrainFeature{}.place(world, scatterTestSeed, c, scatterTestRows, g))

	grass := 0
	for _, p := range collectProps(world) {
		if p.id == scatterGrassProp || p.id == scatterWeedProp {
			grass++
		}
	}
	assert.Positive(t, grass, "地面へ草の prop が撒かれる")
}

// TestScatterFeature_wasteland以外は置かない は、散布のスコープが wasteland 限定であることを固定する。
// 集落や市街のチャンクでは散布が何も置かず、建物との占有衝突を構造的に避ける。
func TestScatterFeature_wasteland以外は置かない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	c := firstNonWastelandChunk(t)
	g := newDirtChunk(t, world)
	require.NoError(t, openTerrainFeature{}.place(world, scatterTestSeed, c, scatterTestRows, g))

	assert.Empty(t, collectProps(world), "wasteland 以外では散布しない")
}

// TestScatterFeature_外周は空けて横断できる は、散布後も外周1タイルの環が空いたままで、隊商が
// チャンクを横断できることを固定する。密度の希釈と経路マスクによる通行性の構造保証の回帰。
func TestScatterFeature_外周は空けて横断できる(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	c := firstWastelandChunk(t)
	g := newDirtChunk(t, world)
	require.NoError(t, openTerrainFeature{}.place(world, scatterTestSeed, c, scatterTestRows, g))

	// 壁として扱うのは BlockPass を持つ prop だけ。草地・plant・moving_stone は歩行可能
	blocked := collectBlockingTiles(world)

	// 外周1タイルの環は候補から外れるので常に空く。西端から東端、北端から南端の双方へ到達できる
	for y := range scatterTestChunk {
		assert.False(t, blocked[consts.Coord[consts.Tile]{X: 0, Y: y}], "西端の列は空く")
		assert.False(t, blocked[consts.Coord[consts.Tile]{X: scatterTestChunk - 1, Y: y}], "東端の列は空く")
	}
	for x := range scatterTestChunk {
		assert.False(t, blocked[consts.Coord[consts.Tile]{X: x, Y: 0}], "北端の行は空く")
		assert.False(t, blocked[consts.Coord[consts.Tile]{X: x, Y: scatterTestChunk - 1}], "南端の行は空く")
	}
	assert.True(t, floodCrosses(blocked, scatterTestChunk, scatterTestChunk, true), "西端から東端へ到達できる")
	assert.True(t, floodCrosses(blocked, scatterTestChunk, scatterTestChunk, false), "北端から南端へ到達できる")
}

// floodCrosses は blocked を壁として、horizontal なら西端の列から東端へ、そうでなければ北端の行から
// 南端へ、4近傍 flood-fill で到達できるかを返す。隊商が縦横どちらにも横断できることの検証に使う。
func floodCrosses(blocked map[consts.Coord[consts.Tile]]bool, cw, ch consts.Tile, horizontal bool) bool {
	visited := make(map[consts.Coord[consts.Tile]]bool)
	var stack []consts.Coord[consts.Tile]
	push := func(p consts.Coord[consts.Tile]) {
		if !blocked[p] && !visited[p] {
			visited[p] = true
			stack = append(stack, p)
		}
	}
	if horizontal {
		for y := range ch {
			push(consts.Coord[consts.Tile]{X: 0, Y: y})
		}
	} else {
		for x := range cw {
			push(consts.Coord[consts.Tile]{X: x, Y: 0})
		}
	}
	dirs := []consts.Coord[consts.Tile]{{X: 1}, {X: -1}, {Y: 1}, {Y: -1}}
	for len(stack) > 0 {
		cur := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if (horizontal && cur.X == cw-1) || (!horizontal && cur.Y == ch-1) {
			return true
		}
		for _, d := range dirs {
			n := consts.Coord[consts.Tile]{X: cur.X + d.X, Y: cur.Y + d.Y}
			if n.X < 0 || n.X >= cw || n.Y < 0 || n.Y >= ch {
				continue
			}
			push(n)
		}
	}
	return false
}

// TestPickScatterEntry_決定的に重みで引く は、同じハッシュ値なら同じ entry を返し、重みに比例して
// 選ばれることを固定する。Go の map を抽選に使わず slice を順に走るので決定的。
func TestPickScatterEntry_決定的に重みで引く(t *testing.T) {
	t.Parallel()
	entries := []scatterEntry{
		{Ref: "", Weight: 1},
		{Ref: "a", Weight: 2},
		{Ref: "b", Weight: 1},
	}
	// total=4。h%4 が 0→null, 1..2→a, 3→b
	assert.Empty(t, pickScatterEntry(entries, true, 0).Ref)
	assert.Equal(t, "a", pickScatterEntry(entries, true, 1).Ref)
	assert.Equal(t, "a", pickScatterEntry(entries, true, 2).Ref)
	assert.Equal(t, "b", pickScatterEntry(entries, true, 3).Ref)
	assert.Equal(t, "a", pickScatterEntry(entries, true, 5).Ref, "h は total で剰余を取る")
}

// TestPickScatterEntry_bigAllowed偽はBigを外す は、位相や経路で大物が許されないタイルでは Big を
// 候補から除いて引くことを固定する。実効的に大物を1位相かつ経路外へ寄せる仕組み。
func TestPickScatterEntry_bigAllowed偽はBigを外す(t *testing.T) {
	t.Parallel()
	entries := []scatterEntry{
		{Ref: "small", Weight: 1},
		{Ref: "huge", Weight: 100, Big: true},
	}
	// bigAllowed 偽なら huge は候補から外れ、残る small だけが総重量になる
	for h := range uint64(10) {
		assert.Equal(t, "small", pickScatterEntry(entries, false, h).Ref, "Big を外すと small だけ引く")
	}
	// total=101 で先頭 small が h%101==0 を占めるので、Big を引くには h>=1
	assert.Equal(t, "huge", pickScatterEntry(entries, true, 1).Ref, "bigAllowed 真なら Big も引ける")
}

// TestOnBigPhase_位相0のタイルだけ真 は、大物を乗せる位相が P で1つに絞られ実効密度が 1/P に
// 希釈されることを固定する。
func TestOnBigPhase_位相0のタイルだけ真(t *testing.T) {
	t.Parallel()
	// (x + 2y) mod 3 == 0 のタイルだけ真
	assert.True(t, onBigPhase(consts.Coord[consts.Tile]{X: 0, Y: 0}))
	assert.True(t, onBigPhase(consts.Coord[consts.Tile]{X: 3, Y: 0}))
	assert.True(t, onBigPhase(consts.Coord[consts.Tile]{X: 1, Y: 1}), "1+2=3 は位相0")
	assert.False(t, onBigPhase(consts.Coord[consts.Tile]{X: 1, Y: 0}))
	assert.False(t, onBigPhase(consts.Coord[consts.Tile]{X: 2, Y: 0}))
	// 負の座標でも周期が連続する
	assert.True(t, onBigPhase(consts.Coord[consts.Tile]{X: -3, Y: 0}))
	assert.False(t, onBigPhase(consts.Coord[consts.Tile]{X: -1, Y: 0}))
}

// TestOutdoorZoneAt_集落近傍は道沿い は、集落の当選チャンクが道沿いゾーンになり、奥地の
// wasteland が別ゾーンになることを固定する。
func TestOutdoorZoneAt_集落近傍は道沿い(t *testing.T) {
	t.Parallel()
	// 集落の当選チャンクそのものは距離0で必ず道沿い
	winner := settlementPlacement.WinnerOf(scatterTestSeed, 2, scatterTestRows)
	assert.Equal(t, zoneRoadside, outdoorZoneAt(scatterTestSeed, winner, scatterTestRows), "集落の当選チャンクは道沿い")

	// 奥地ゾーンのチャンクが少なくとも1つ存在し、関数がゾーンを識別できる
	foundWild := false
	for cx := range consts.Chunk(100) {
		if outdoorZoneAt(scatterTestSeed, consts.Coord[consts.Chunk]{X: cx, Y: 0}, scatterTestRows) == zoneWild {
			foundWild = true
			break
		}
	}
	assert.True(t, foundWild, "奥地ゾーンのチャンクが存在する")
}

// TestChebToSeg_線分までのチェビシェフ距離 は、経路マスクが使う水平・垂直線分への距離が正しいことを
// 固定する。線分の内側は直交距離、端点の外側は角までの距離になる。
func TestChebToSeg_線分までのチェビシェフ距離(t *testing.T) {
	t.Parallel()
	// 水平線分 y=0, x in [0,10]
	assert.Equal(t, consts.Tile(0), chebToHSeg(consts.Coord[consts.Tile]{X: 5, Y: 0}, 0, 0, 10), "線分上は0")
	assert.Equal(t, consts.Tile(5), chebToHSeg(consts.Coord[consts.Tile]{X: 5, Y: 5}, 0, 0, 10), "内側は直交距離")
	assert.Equal(t, consts.Tile(5), chebToHSeg(consts.Coord[consts.Tile]{X: 15, Y: 0}, 0, 0, 10), "端の外は水平はみ出し")
	// 垂直線分 x=0, y in [0,10]
	assert.Equal(t, consts.Tile(0), chebToVSeg(consts.Coord[consts.Tile]{X: 0, Y: 5}, 0, 0, 10), "線分上は0")
	assert.Equal(t, consts.Tile(3), chebToVSeg(consts.Coord[consts.Tile]{X: 3, Y: 5}, 0, 0, 10), "内側は直交距離")
}
