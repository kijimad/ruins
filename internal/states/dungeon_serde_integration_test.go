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
	st := &DungeonState{DefinitionName: dungeon.DungeonOverworld.Name}
	require.NoError(t, st.enterDungeon(world, dungeon.DungeonDebug.Name))

	ruinKey := gc.NewNamedDungeonStage(dungeon.DungeonDebug.Name, 1)
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
	st2 := &DungeonState{Depth: query.GetDungeon(newWorld).Depth}
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
	st := &DungeonState{DefinitionName: dungeon.DungeonOverworld.Name}
	require.NoError(t, st.enterDungeon(world, dungeon.DungeonDebug.Name))
	require.NoError(t, st.descend(world))

	// 遺跡の全フロアは定義名付きキーで一貫して識別される。enterDungeon が作る1階も
	// descend が作る深い階も同じ dungeonStageKey で揃うため、上り階段の結線が正しい階を指す。
	floor1Key := dungeonStageKey(dungeon.DungeonDebug.Name, 1)
	floor2Key := dungeonStageKey(dungeon.DungeonDebug.Name, 2)
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
	st2 := &DungeonState{Depth: query.GetDungeon(newWorld).Depth}
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
