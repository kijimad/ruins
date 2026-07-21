package overworld

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/mapplanner"
	"github.com/kijimaD/ruins/internal/save"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testChunkW consts.Tile  = 30
	testChunkH consts.Tile  = 20
	testK      consts.Chunk = 3
)

func TestSession_MaybeShift_東へ進むとシフトする(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	s := NewSession(mapplanner.PlannerTypeSmallRoom, &NewGameParams{RunSeed: 777, ChunkW: testChunkW, ChunkH: testChunkH, K: testK})
	require.NoError(t, s.Start(world))

	player, err := query.GetPlayerEntity(world)
	require.NoError(t, err)
	world.Components.GridElement.Get(player).X = 2 * testChunkW // 東チャンクへ踏み込む

	shifted, err := s.MaybeShift(world)
	require.NoError(t, err)
	assert.True(t, shifted, "東へ踏み込むとシフトする")
	assert.Equal(t, 1, int(s.EastIndex()), "東シフトで eastIndex が進む")
	assert.Equal(t, testChunkW, world.Components.GridElement.Get(player).X, "プレイヤーは中央へ戻る")
}

func TestSession_MaybeShift_複数チャンク跨ぎで連続シフト(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	s := NewSession(mapplanner.PlannerTypeSmallRoom, &NewGameParams{RunSeed: 777, ChunkW: testChunkW, ChunkH: testChunkH, K: testK})
	require.NoError(t, s.Start(world))

	player, err := query.GetPlayerEntity(world)
	require.NoError(t, err)
	world.Components.GridElement.Get(player).X = 100 // 2チャンク以上東（帯外）

	shifted, err := s.MaybeShift(world)
	require.NoError(t, err)
	assert.True(t, shifted)
	assert.Equal(t, 2, int(s.EastIndex()), "収まるまで連続シフトして eastIndex=2")
	px := world.Components.GridElement.Get(player).X
	assert.GreaterOrEqual(t, px, consts.Tile(testK/2)*testChunkW, "プレイヤーは中央チャンク内に収まる")
	assert.Less(t, px, consts.Tile(testK/2+1)*testChunkW, "プレイヤーは中央チャンク内に収まる")
}

// TestSession_MaybeShift_開始点より西へはシフトしない は eastIndex=0 で西へ移動しても
// eastIndex を負にしないことを固定する。
func TestSession_MaybeShift_開始点より西へはシフトしない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	s := NewSession(mapplanner.PlannerTypeSmallRoom, &NewGameParams{RunSeed: 777, ChunkW: testChunkW, ChunkH: testChunkH, K: testK})
	require.NoError(t, s.Start(world))
	require.Equal(t, 0, int(s.EastIndex()), "前提: 開始時 eastIndex=0")

	player, err := query.GetPlayerEntity(world)
	require.NoError(t, err)
	world.Components.GridElement.Get(player).X = 10 // 中央チャンクより西

	_, err = s.MaybeShift(world)
	require.NoError(t, err)
	assert.Equal(t, 0, int(s.EastIndex()), "開始点より西へはシフトしない（eastIndex は負にならない）")
}

func TestSession_MaybeShift_中央では動かない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	s := NewSession(mapplanner.PlannerTypeSmallRoom, &NewGameParams{RunSeed: 777, ChunkW: testChunkW, ChunkH: testChunkH, K: testK})
	require.NoError(t, s.Start(world))

	shifted, err := s.MaybeShift(world)
	require.NoError(t, err)
	assert.False(t, shifted, "中央チャンク内ではシフトしない")
	assert.Equal(t, 0, int(s.EastIndex()), "中央チャンク内では eastIndex 据え置き")
}

// TestSession_セーブ往復で帯状態が復元される は、SeamlessBand が serde に乗り、ロード後に
// セッションが同じ eastIndex で再構築できることを固定する。
func TestSession_セーブ往復で帯状態が復元される(t *testing.T) {
	t.Parallel()

	const chunkW, chunkH consts.Tile = 40, 20
	const k = 3

	world := testutil.InitTestWorld(t)
	s := NewSession(mapplanner.PlannerTypeOverworldField, &NewGameParams{RunSeed: 12345, ChunkW: chunkW, ChunkH: chunkH, K: k})
	require.NoError(t, s.Start(world))

	// 東へ1回シフトして eastIndex=1 にする
	player, err := query.GetPlayerEntity(world)
	require.NoError(t, err)
	world.Components.GridElement.Get(player).X = 2 * chunkW
	shifted, err := s.MaybeShift(world)
	require.NoError(t, err)
	require.True(t, shifted)
	require.Equal(t, 1, int(s.EastIndex()))
	require.Equal(t, 1, int(query.GetDungeon(world).SeamlessBand.EastIndex), "永続状態に同期される")

	// セーブ往復（メモリ内）
	sm, err := save.NewSerializationManager()
	require.NoError(t, err)
	jsonData, err := sm.GenerateWorldJSON(world)
	require.NoError(t, err)

	world2 := testutil.InitTestWorld(t)
	require.NoError(t, sm.RestoreWorldFromJSON(world2, jsonData))

	// SeamlessBand が復元されている
	sb := query.GetDungeon(world2).SeamlessBand
	assert.True(t, sb.Active, "Active が復元される")
	assert.Equal(t, 1, int(sb.EastIndex), "EastIndex が復元される")
	assert.Equal(t, uint64(12345), sb.RunSeed, "RunSeed が復元される")
	assert.Equal(t, chunkW, sb.ChunkW, "ChunkW が復元される")
	assert.Equal(t, k, int(sb.K), "K が復元される")

	// 寒波前線の config が復元される
	assert.True(t, sb.Front.Active, "FrontActive が復元される")
	assert.Equal(t, frontColdWidthChunks.Tiles(chunkW), sb.Front.ColdWidth, "FrontColdWidth が復元される")
	assert.Equal(t, frontAdvanceTurns, sb.Front.AdvanceTurns, "FrontAdvanceTurns が復元される")
	assert.Equal(t, frontStep, sb.Front.Step, "FrontStep が復元される")

	// 復元ワールドでロード用セッションを起動 → Band が eastIndex=1 で再構築される
	s2 := NewSession(mapplanner.PlannerTypeOverworldField, nil)
	require.NoError(t, s2.Start(world2))
	assert.Equal(t, 1, int(s2.EastIndex()), "ロード復元で Band が eastIndex=1 で再構築される")
	assert.Equal(t, chunkW*k, query.GetDungeon(world2).Level.TileWidth, "帯全幅の Level が保たれる")
	assert.True(t, s2.frontCfg.AdvanceTurns == frontAdvanceTurns && s2.frontCfg.Step == frontStep,
		"ロード復元で寒波前線 config も再構築される")

	// 復元ワールドに帯タイルが存在する（serde 復元）
	count := 0
	tileQuery := ecs.NewFilter1[gc.GridElement](world2.ECS).Query()
	for tileQuery.Next() {
		count++
	}
	assert.Positive(t, count, "帯タイルが serde で復元されている")
}

// TestSession_新規開始で街がオーバーワールドに配置される は、新規開始時に店・雇用・合成の会話NPCと
// 収納propが開始チャンクへ配置され、いずれもオーバーワールド帯へ束縛されることを固定する。
// これで街が専用ステージでなくオーバーワールドの地物として常在し、遺跡進入時に帯とともに退避される。
func TestSession_新規開始で街がオーバーワールドに配置される(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	s := NewSession(mapplanner.PlannerTypeOverworldField, &NewGameParams{RunSeed: 42, ChunkW: testChunkW, ChunkH: testChunkH, K: testK})
	require.NoError(t, s.Start(world))

	// 街の構成物を名前で探し、配置・帯束縛・相互作用の有無を確認する
	want := map[string]bool{"商人": false, "酒場の主人": false, "怪しい科学者": false, townStorageProp: false}
	q := ecs.NewFilter1[gc.Name](world.ECS).Query()
	for q.Next() {
		e := q.Entity()
		name := world.Components.Name.Get(e).Name
		if _, ok := want[name]; !ok {
			continue
		}
		want[name] = true
		require.True(t, world.Components.StageBound.Has(e), "%s はステージへ束縛される", name)
		assert.Equal(t, gc.NewOverworldStage(), world.Components.StageBound.Get(e).Key, "%s はオーバーワールド帯へ束縛される", name)
		assert.True(t, world.Components.Interactable.Has(e), "%s は相互作用を持つ", name)
	}
	for name, found := range want {
		assert.True(t, found, "街の構成物 %s が配置される", name)
	}
}

// TestSession_前線が総ターン数で前進する は、寒波前線の現在位置が GameTime.TotalTurns から
// 決定的に導出され、ターン経過で東へ進み、SeamlessBand.Front.EastAbsX に反映されることを固定する。
func TestSession_前線が総ターン数で前進する(t *testing.T) {
	t.Parallel()

	const chunkW, chunkH consts.Tile = 40, 20

	world := testutil.InitTestWorld(t)
	s := NewSession(mapplanner.PlannerTypeOverworldField, &NewGameParams{RunSeed: 1, ChunkW: chunkW, ChunkH: chunkH, K: 3})
	require.NoError(t, s.Start(world))

	d := query.GetDungeon(world)
	// 開始時（TotalTurns=0）は StartEast のまま。StartEast = bandOriginX(0) + chunkW = +chunkW
	d.GameTime.TotalTurns = 0
	s.UpdateFront(world)
	assert.Equal(t, consts.AbsTileX(chunkW), d.SeamlessBand.Front.EastAbsX, "0ターンは開始位置 +chunkW（西チャンク東端）")

	// frontAdvanceTurns ごとに frontStep 前進する。AdvanceTurns 未満は動かない
	d.GameTime.TotalTurns = frontAdvanceTurns - 1
	s.UpdateFront(world)
	assert.Equal(t, consts.AbsTileX(chunkW), d.SeamlessBand.Front.EastAbsX, "AdvanceTurns 未満は前進しない")

	d.GameTime.TotalTurns = frontAdvanceTurns
	s.UpdateFront(world)
	assert.Equal(t, consts.AbsTileX(chunkW)+consts.AbsTileX(frontStep), d.SeamlessBand.Front.EastAbsX, "AdvanceTurns で 1 段前進する")

	// 決定的: 同じターン数なら同じ位置
	before := d.SeamlessBand.Front.EastAbsX
	s.UpdateFront(world)
	assert.Equal(t, before, d.SeamlessBand.Front.EastAbsX, "冪等（導出値）")
}
