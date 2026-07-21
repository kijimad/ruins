package states

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/dungeon"
	"github.com/kijimaD/ruins/internal/mapplanner"
	"github.com/kijimaD/ruins/internal/overworld"
	"github.com/kijimaD/ruins/internal/save"
	"github.com/kijimaD/ruins/internal/testutil"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/kijimaD/ruins/internal/world/stage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// suspendedCount は指定ステージに束縛されたエンティティのうち退避中の数と総数を返す。
func suspendedCount(world w.World, key gc.StageKey) (suspended int, total int) {
	for _, e := range stage.BoundEntities(world, key) {
		total++
		if world.Components.Suspended.Has(e) {
			suspended++
		}
	}
	return suspended, total
}

// TestPhaseG_遺跡滞在中にセーブロードしても共存が復元され地上へ戻れる は Phase 8 の統合検証。
// 実オーバーワールドと街を生成し、遺跡へ入って共存状態(オーバーワールド退避+遺跡稼働)を作り、
// セーブして別 world へロードし、退避中のオーバーワールド+街と稼働中の遺跡が現物として復元され、
// ロード後も上り階段の結線をたどって地上へ戻れることを1本で確認する。
//
// これは共存方式の核心「継続は ECS に置きスタックに預けない」が serde 往復で成立することの
// エンドツーエンド検証で、開始点(街のオーバーワールド配置)・共存保持・serde 往復を束ねる。
func TestPhaseG_遺跡滞在中にセーブロードしても共存が復元され地上へ戻れる(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	// 実オーバーワールドと街を生成する。Start が現ステージをオーバーワールドに確定し、
	// プレイヤーと街の会話NPC・収納 prop を開始チャンクへ配置する。
	sess := overworld.NewSession(mapplanner.PlannerTypeOverworldField, &overworld.NewGameParams{RunSeed: 42, ChunkW: 30, ChunkH: 20, K: 3})
	require.NoError(t, sess.Start(world))

	player, err := query.GetPlayerEntity(world)
	require.NoError(t, err)
	entrancePos := world.Components.GridElement.Get(player).Coord

	// 街+帯がオーバーワールドへ束縛され、すべて稼働している
	owSuspended, owTotal := suspendedCount(world, gc.NewOverworldStage())
	require.Positive(t, owTotal, "街+帯がオーバーワールドへ束縛されている")
	require.Zero(t, owSuspended, "進入前はオーバーワールドが稼働している")

	// 遺跡へ入る。オーバーワールド State は BuilderType を持たず、遺跡定義で生成する。
	st := &DungeonState{DefinitionName: dungeon.DungeonOverworld.Name()}
	require.NoError(t, st.enterDungeon(world, dungeon.DungeonDebug.Name()))

	ruinKey := gc.NewNamedDungeonStage(dungeon.DungeonDebug.Name(), 1)
	require.Equal(t, ruinKey, query.GetDungeon(world).CurrentStage, "現ステージは遺跡1階")

	// セーブして別 world へロードする。共存状態がまるごと往復する。
	manager, err := save.NewSerializationManager(save.WithSaveDir(t.TempDir()))
	require.NoError(t, err)
	require.NoError(t, manager.SaveWorld(world, "phaseg"))

	newWorld := testutil.InitTestWorld(t)
	require.NoError(t, manager.LoadWorld(newWorld, "phaseg"))

	// 現ステージが遺跡のまま復元される
	assert.Equal(t, ruinKey, query.GetDungeon(newWorld).CurrentStage, "現ステージ=遺跡1階が復元される")

	// 退避中のオーバーワールド+街が現物として残り、すべて退避されたまま復元される
	owSuspended, owTotal = suspendedCount(newWorld, gc.NewOverworldStage())
	require.Positive(t, owTotal, "退避中のオーバーワールド+街が現物として復元される")
	assert.Equal(t, owTotal, owSuspended, "オーバーワールド+街は退避されたまま復元される")

	// 稼働中の遺跡1階が現物として残る
	require.NotEmpty(t, stage.BoundEntities(newWorld, ruinKey), "遺跡1階が現物として復元される")

	// ロード後に上り階段で地上へ戻る。結線が serde を跨いで保たれている
	st2 := &DungeonState{Depth: query.GetDungeon(newWorld).CurrentStage.Depth}
	handled, aerr := st2.ascend(newWorld)
	require.NoError(t, aerr)
	require.True(t, handled, "ロード後も上り階段の結線をたどって地上へ戻れる")
	assert.Equal(t, gc.NewOverworldStage(), query.GetDungeon(newWorld).CurrentStage, "地上へ戻る")

	// オーバーワールド+街が再稼働し、プレイヤーは入った入口へ戻る
	owSuspended, _ = suspendedCount(newWorld, gc.NewOverworldStage())
	assert.Zero(t, owSuspended, "地上へ戻るとオーバーワールド+街が再稼働する")

	newPlayer, err := query.GetPlayerEntity(newWorld)
	require.NoError(t, err)
	assert.Equal(t, entrancePos, newWorld.Components.GridElement.Get(newPlayer).Coord, "入った入口へ戻る")
}

// TestPhaseG_多層の共存がセーブロードを跨いで保持され順に地上まで戻れる は、退避ステージが
// 複数ある共存状態が serde 往復で保持されることを検証する。オーバーワールド+遺跡1階+遺跡2階の
// 3ステージを作り、上位2つを退避したままロードで復元し、上り階段を2回たどって遺跡2階から
// 遺跡1階、そして地上へ順に戻れることを確認する。単一退避ステージの往復では見えない、複数の
// 退避ステージが同時に現物として残る不変条件を固める。
func TestPhaseG_多層の共存がセーブロードを跨いで保持され順に地上まで戻れる(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	sess := overworld.NewSession(mapplanner.PlannerTypeOverworldField, &overworld.NewGameParams{RunSeed: 7, ChunkW: 30, ChunkH: 20, K: 3})
	require.NoError(t, sess.Start(world))

	// 遺跡へ入り、さらに1つ深い階へ降りる。3ステージが共存する。
	st := &DungeonState{DefinitionName: dungeon.DungeonOverworld.Name()}
	require.NoError(t, st.enterDungeon(world, dungeon.DungeonDebug.Name()))
	require.NoError(t, st.descend(world))

	// 遺跡の全フロアは定義名付きキーで一貫して識別される。enterDungeon が作る1階も
	// descend が作る深い階も同じ dungeonStageKey で揃うため、上り階段の結線が正しい階を指す。
	floor1Key := dungeonStageKey(dungeon.DungeonDebug.Name(), 1)
	floor2Key := dungeonStageKey(dungeon.DungeonDebug.Name(), 2)
	require.Equal(t, floor2Key, query.GetDungeon(world).CurrentStage, "現ステージは遺跡2階")

	// セーブして別 world へロードする。
	manager, err := save.NewSerializationManager(save.WithSaveDir(t.TempDir()))
	require.NoError(t, err)
	require.NoError(t, manager.SaveWorld(world, "phaseg_multi"))

	newWorld := testutil.InitTestWorld(t)
	require.NoError(t, manager.LoadWorld(newWorld, "phaseg_multi"))

	// 3ステージが現物として復元される。上位2つ(オーバーワールド・遺跡1階)は退避、遺跡2階は稼働。
	assert.Equal(t, floor2Key, query.GetDungeon(newWorld).CurrentStage, "現ステージ=遺跡2階が復元される")

	owSuspended, owTotal := suspendedCount(newWorld, gc.NewOverworldStage())
	require.Positive(t, owTotal, "オーバーワールド+街が現物として復元される")
	assert.Equal(t, owTotal, owSuspended, "オーバーワールド+街は退避されたまま復元される")

	f1Suspended, f1Total := suspendedCount(newWorld, floor1Key)
	require.Positive(t, f1Total, "遺跡1階が現物として復元される")
	assert.Equal(t, f1Total, f1Suspended, "遺跡1階は退避されたまま復元される")

	require.NotEmpty(t, stage.BoundEntities(newWorld, floor2Key), "遺跡2階が現物として復元される")

	// 上り階段を2回たどり、遺跡2階 → 遺跡1階 → 地上 と順に戻れる。
	st2 := &DungeonState{Depth: query.GetDungeon(newWorld).CurrentStage.Depth}
	handled, aerr := st2.ascend(newWorld)
	require.NoError(t, aerr)
	require.True(t, handled, "遺跡2階から1階へ上れる")
	assert.Equal(t, floor1Key, query.GetDungeon(newWorld).CurrentStage, "遺跡1階へ戻る")

	handled, aerr = st2.ascend(newWorld)
	require.NoError(t, aerr)
	require.True(t, handled, "遺跡1階から地上へ上れる")
	assert.Equal(t, gc.NewOverworldStage(), query.GetDungeon(newWorld).CurrentStage, "地上へ戻る")

	owSuspended, _ = suspendedCount(newWorld, gc.NewOverworldStage())
	assert.Zero(t, owSuspended, "地上へ戻るとオーバーワールド+街が再稼働する")
}

// TestPhaseG_遺跡から地上へ戻ると帯寸法と視界が復元され隊員も配置される は、遺跡から
// オーバーワールドへ戻る際の復帰処理を固定する。遺跡進入で Level が遺跡寸法に置き換わるため、
// 帰還時に帯寸法へ戻し視界を強制再計算しないと、プレイヤーが帯座標にいるのにマップが遺跡寸法の
// ままで真っ暗・ミニマップ No Data になり、隊員配置も範囲外で失敗する。この一連を1本で固める。
func TestPhaseG_遺跡から地上へ戻ると帯寸法と視界が復元され隊員も配置される(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	// 街中心はNPC・収納で密集する。実ゲームの帯パラメータで新規開始し隊員を1体連れる。
	sess := overworld.NewSession(mapplanner.PlannerTypeOverworldField, &overworld.NewGameParams{RunSeed: 1, ChunkW: 50, ChunkH: 50, K: 3})
	require.NoError(t, sess.Start(world))
	player, err := query.GetPlayerEntity(world)
	require.NoError(t, err)

	// 密集した街中心でも隊員を生成できる
	member, err := lifecycle.SpawnDefaultSquadMember(world, player)
	require.NoError(t, err)

	// 遺跡へ入る。Level が遺跡寸法に置き換わる。
	st := &DungeonState{DefinitionName: dungeon.DungeonOverworld.Name()}
	require.NoError(t, st.enterDungeon(world, dungeon.DungeonDebug.Name()))

	// 上り階段で密集した街へ戻る。帰還が失敗しないこと。
	st2 := &DungeonState{Depth: query.GetDungeon(world).CurrentStage.Depth}
	handled, aerr := st2.ascend(world)
	require.NoError(t, aerr, "密集した着地点でも帰還が失敗しない")
	require.True(t, handled, "上り階段の結線で地上へ戻れる")

	// 帯寸法の Level が復元され、視界の強制再計算が要求される。遺跡寸法のままだと真っ暗・No Data。
	sb := query.GetSeamlessBand(world)
	meta := query.GetCurrentStageMeta(world)
	assert.Equal(t, sb.K.Tiles(sb.ChunkW), meta.Level.TileWidth, "帯幅の Level が復元される")
	assert.Equal(t, sb.ChunkH, meta.Level.TileHeight, "帯高さの Level が復元される")
	assert.True(t, query.GetVisionState(world).NeedsForceUpdate, "視界の強制再計算が要求される")

	// 隊員は復元された帯寸法の範囲内に配置される
	si := query.GetSpatialIndex(world)
	require.NotNil(t, si)
	memberPos := world.Components.GridElement.Get(member).Coord
	assert.GreaterOrEqual(t, int(memberPos.X), 0)
	assert.GreaterOrEqual(t, int(memberPos.Y), 0)
	assert.Less(t, int(memberPos.X), int(si.MapWidth), "隊員はマップ範囲内に配置される")
	assert.Less(t, int(memberPos.Y), int(si.MapHeight), "隊員はマップ範囲内に配置される")
}

// TestPhaseG_遺跡内で保存しロード復元しても現ステージが遺跡のまま は、遺跡内で保存したセーブを
// ロード復元したとき、現ステージが遺跡のまま保たれ、オーバーワールドと誤判定されないことを固定する。
// 帯データはオーバーワールドのメタにしか無く遺跡進入で退避されるため、遺跡が現ステージのとき帯は
// 現ステージから外れる。これにより遺跡内で前線の霜が誤って描かれることが構造的に起きない。
// 復帰先も newResumeStateFactory が DungeonState を選ぶので、オーバーワールドの Start は呼ばれない。
func TestPhaseG_遺跡内で保存しロード復元しても現ステージが遺跡のまま(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	sess := overworld.NewSession(mapplanner.PlannerTypeOverworldField, &overworld.NewGameParams{RunSeed: 3, ChunkW: 30, ChunkH: 20, K: 3})
	require.NoError(t, sess.Start(world))

	// 遺跡へ入る。現ステージは遺跡、帯データはオーバーワールドのメタごと退避される。
	st := &DungeonState{DefinitionName: dungeon.DungeonOverworld.Name()}
	require.NoError(t, st.enterDungeon(world, dungeon.DungeonDebug.Name()))
	ruinKey := gc.NewNamedDungeonStage(dungeon.DungeonDebug.Name(), 1)
	require.Equal(t, ruinKey, query.GetDungeon(world).CurrentStage)

	// セーブして別 world へロードする。
	manager, err := save.NewSerializationManager(save.WithSaveDir(t.TempDir()))
	require.NoError(t, err)
	require.NoError(t, manager.SaveWorld(world, "ruinsave"))
	newWorld := testutil.InitTestWorld(t)
	require.NoError(t, manager.LoadWorld(newWorld, "ruinsave"))

	// 現ステージは帯データを持たない遺跡なので、オーバーワールドと誤判定しない。
	// これにより遺跡内で前線の霜が誤って描かれない。
	assert.False(t, query.IsOnOverworld(newWorld), "遺跡内なので帯データを持たずオーバーワールドと誤判定しない")
	assert.Nil(t, query.GetSeamlessBand(newWorld), "現ステージ(遺跡)は帯データを持たない")

	// 帯データを含む退避中のオーバーワールドは現物として残り、後で戻れる。
	require.NotEmpty(t, stage.BoundEntities(newWorld, gc.NewOverworldStage()), "退避中のオーバーワールドが現物として復元される")

	// 復帰先は DungeonState が選ばれる。オーバーワールドの Start は呼ばれない。
	rState, rErr := newResumeStateFactory(newWorld)()
	require.NoError(t, rErr)
	rf, ok := rState.(*DungeonState)
	require.True(t, ok, "遺跡内セーブの復帰先は DungeonState")
	assert.False(t, rf.isSeamless(), "遺跡モードで復帰する")

	// 復元後も現ステージは遺跡のまま。
	assert.Equal(t, ruinKey, query.GetDungeon(newWorld).CurrentStage, "遺跡内で保存したロードは現ステージを遺跡に保つ")
}
