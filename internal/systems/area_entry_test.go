package systems

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/dungeon"
	mapplanner "github.com/kijimaD/ruins/internal/mapplanner"
	"github.com/kijimaD/ruins/internal/overworld"
	"github.com/kijimaD/ruins/internal/testutil"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAreaEntrySystem_observe_別の区域に入ったときだけ入ったとみなす(t *testing.T) {
	t.Parallel()

	townA := overworld.Area{Key: "town:0,0", Name: "Town"}
	townB := overworld.Area{Key: "town:9,9", Name: "Town"}
	wasteland := overworld.Area{Name: "Wasteland"}
	shrine := overworld.Area{Name: "Shrine"}

	sys := &AreaEntrySystem{}
	seq := []overworld.Area{townA, townA, wasteland, townA, townB, wasteland, shrine, townB}
	got := make([]bool, 0, len(seq))
	for i, a := range seq {
		got = append(got, sys.observe(a, consts.Turn(100+i)))
	}
	assert.Equal(t, []bool{
		true,  // 最初に見たのが区域なら入ったとみなす
		false, // 同じ区域の中
		false, // 荒れ地に出た
		true,  // 荒れ地から町に戻った
		true,  // 同じ種類の別の区域に移った
		false, // 荒れ地に出た
		false, // 小さな場所に入った
		true,  // 小さな場所から区域に戻った
	}, got)
}

func TestAreaEntrySystem_observe_ターンが続いていなければ今いる区域に入ったとみなす(t *testing.T) {
	t.Parallel()

	townA := overworld.Area{Key: "town:0,0", Name: "Town"}
	wasteland := overworld.Area{Name: "Wasteland"}

	sys := &AreaEntrySystem{}
	got := []bool{
		sys.observe(townA, 100),
		sys.observe(townA, 100), // 開始時の観測の直後に、同じターンでもう一度見た
		sys.observe(townA, 500), // 同じ町のセーブを読み込んだ、あるいはダンジョンから同じ町に戻った
		sys.observe(townA, 501),
		sys.observe(wasteland, 900), // 荒れ地のセーブを読み込んだ
	}
	assert.Equal(t, []bool{true, false, true, false, false}, got)
}

func TestAreaLabel_地上でなければ空(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	assert.Empty(t, areaLabel(world))
}

func TestAreaLabel_地上では現在地の名前を出す(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	drv := overworld.NewDriver(mapplanner.PlannerTypeOverworldField, dungeon.NewOverworldDefinition("オーバーワールド", 0, 24, 24, 3, 1), &overworld.NewGameParams{RunSeed: 42})
	require.NoError(t, drv.Start(world))
	assert.Equal(t, "Wasteland", areaLabel(world), "開始地点は荒れ地")
}

func TestFormatAreaLabel(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	assert.Equal(t, []string{"Town / Clinic", "Shrine"}, []string{
		formatAreaLabel(world, overworld.Area{Key: "town:0,0", Name: "Town", Building: "Clinic"}),
		formatAreaLabel(world, overworld.Area{Name: "Shrine"}),
	})
}

func TestRunTurnEndSystems_登録したAreaEntrySystemが地上で回る(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t, testutil.WithUI())
	drv := overworld.NewDriver(mapplanner.PlannerTypeOverworldField, dungeon.NewOverworldDefinition("オーバーワールド", 0, 24, 24, 3, 1), &overworld.NewGameParams{RunSeed: 42})
	require.NoError(t, drv.Start(world))
	updaters, _ := InitializeSystems(world)
	name := AreaEntrySystem{}.String()
	sys, ok := updaters[name].(*AreaEntrySystem)
	require.True(t, ok, "InitializeSystems に登録する")
	// ほかのターン終わりの system はこの world では回らないので、登録したものだけを置く
	world.Updaters = map[string]w.Updater{name: sys}

	require.NoError(t, runTurnEndSystems(world))

	assert.True(t, sys.observed, "ターンの終わりにエリアを観測する")
}

func TestAreaEntrySystem_Restart_開始とロードのたびに今いる区域の名前を出す(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	drv := overworld.NewDriver(mapplanner.PlannerTypeOverworldField, dungeon.NewOverworldDefinition("オーバーワールド", 0, 24, 24, 7, 9), &overworld.NewGameParams{RunSeed: 12345})
	require.NoError(t, drv.Start(world))
	sb := query.GetSeamlessBand(world)

	// 帯の中から区域のチャンクを探し、プレイヤーをそこへ置く
	var target consts.Coord[consts.Tile]
	found := false
	for ly := consts.Chunk(0); ly < sb.Rows && !found; ly++ {
		for x := range sb.Cols {
			row := ly - sb.NorthIndex
			area := overworld.AreaAt(world.Resources.RawMaster, sb.RunSeed, consts.Coord[consts.Chunk]{X: x, Y: row}, sb.Cols, sb.SpawnChunkY())
			if area.Key != "" {
				target = consts.Coord[consts.Tile]{X: x.Tiles(sb.ChunkW) + 1, Y: ly.Tiles(sb.ChunkH) + 1}
				found = true
				break
			}
		}
	}
	require.True(t, found, "帯の中に区域がある")
	player, err := query.GetPlayerEntity(world)
	require.NoError(t, err)
	world.Components.GridElement.Get(player).Coord = target

	sys := &AreaEntrySystem{}
	// 開始してすぐセーブし、同じターンのままロードした
	require.NoError(t, sys.Restart(world))
	require.NoError(t, sys.Restart(world))

	assert.Len(t, splashTexts(world), 2, "ロードでも、続いて見える同じ区域の名前を出す")
}

// splashTexts はワールドに湧いているスプラッシュ表示エフェクトのテキストを列挙する
func splashTexts(world w.World) []string {
	var texts []string
	q := ecs.NewFilter1[gc.VisualEffects](world.ECS).Query()
	for q.Next() {
		for _, effect := range world.Components.VisualEffects.Get(q.Entity()).Effects {
			if splash, ok := effect.(*gc.SplashTextEffect); ok {
				texts = append(texts, splash.Text)
			}
		}
	}
	return texts
}
